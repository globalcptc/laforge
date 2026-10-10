package microcloud

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const placementCacheTTL = 30 * time.Second

var errPlacementCapacity = errors.New("no eligible MicroCloud member has usable host stats and sufficient free RAM")

// Builder instances are short-lived; share sampling and reservations across them.
// Reservations coordinate this runner process, not other LaForge processes.
var placements sync.Map // endpoint + project -> *placementState

type placementState struct {
	mu           sync.Mutex
	refreshing   chan struct{}
	sampled      time.Time
	members      []placementMember
	fallback     string
	reservations map[string]*placementReservation
}

type placementReservation struct {
	member  string
	cpu     float64
	memory  uint64
	users   int
	expires time.Time
}

type placementMember struct {
	Name         string            `json:"server_name"`
	Status       string            `json:"status"`
	Architecture string            `json:"architecture"`
	Config       map[string]string `json:"config"`
	Groups       []string          `json:"groups"`
	stats        *memberSysinfo
}

type memberSysinfo struct {
	LoadAverages []float64 `json:"load_averages"`
	LogicalCPUs  uint64    `json:"logical_cpus"`
	TotalRAM     uint64    `json:"total_ram"`
	FreeRAM      uint64    `json:"free_ram"`
}

func (s memberSysinfo) valid() bool {
	if s.LogicalCPUs == 0 || s.TotalRAM == 0 || s.FreeRAM > s.TotalRAM || len(s.LoadAverages) != 3 {
		return false
	}
	for _, load := range s.LoadAverages {
		if load < 0 || math.IsNaN(load) || math.IsInf(load, 0) {
			return false
		}
	}
	return true
}

// Only explicit counts and absolute RAM limits can be reserved accurately.
// CPU pin sets, percentages and inherited limits remain LXD's responsibility.
func placementResources(size SizeSpec) (float64, uint64, bool) {
	cpu, err := strconv.ParseUint(size.CPU, 10, 32)
	if err != nil || cpu == 0 {
		return 0, 0, false
	}
	value := size.Memory
	multiplier := float64(1)
	for _, unit := range []struct {
		suffix string
		factor float64
	}{{"TiB", 1 << 40}, {"GiB", 1 << 30}, {"MiB", 1 << 20}, {"KiB", 1 << 10}, {"TB", 1e12}, {"GB", 1e9}, {"MB", 1e6}, {"kB", 1e3}, {"B", 1}} {
		if strings.HasSuffix(value, unit.suffix) {
			value = strings.TrimSuffix(value, unit.suffix)
			multiplier = unit.factor
			break
		}
	}
	memory, err := strconv.ParseFloat(value, 64)
	memory = math.Ceil(memory * multiplier)
	if err != nil || math.IsNaN(memory) || math.IsInf(memory, 0) || memory < 1 || memory >= math.Exp2(64) {
		return 0, 0, false
	}
	return float64(cpu), uint64(memory), true
}

// selectInstanceTarget never relocates an existing instance. A target is chosen
// once and retained across all create-operation retries, including uncertain ones.
func (b *Builder) selectInstanceTarget(ctx context.Context, name string, size SizeSpec) (string, func(), error) {
	noop := func() {}
	_, err := b.Client.get(ctx, "/1.0/instances/"+url.PathEscape(name))
	if err == nil {
		return "", noop, nil
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.HTTPStatus != http.StatusNotFound {
		return "", noop, err
	}
	cpu, memory, ok := placementResources(size)
	if !ok {
		slog.InfoContext(ctx, "MicroCloud delegating placement to LXD", "instance", name, "reason", "size needs explicit CPU count and absolute RAM")
		return "", noop, nil
	}
	key := strings.TrimRight(b.Client.BaseURL, "/") + "\x00" + b.Client.Project
	value, _ := placements.LoadOrStore(key, &placementState{reservations: make(map[string]*placementReservation)})
	state := value.(*placementState)
	if err := state.refresh(ctx, b); err != nil {
		return "", noop, err
	}
	state.mu.Lock()
	defer state.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return "", noop, err
	}
	if state.fallback != "" {
		slog.WarnContext(ctx, "MicroCloud delegating placement to LXD", "instance", name, "reason", state.fallback)
		return "", noop, nil
	}
	now := time.Now()
	for key, reservation := range state.reservations {
		if reservation.users == 0 && !now.Before(reservation.expires) {
			delete(state.reservations, key)
		}
	}
	reservation := state.reservations[name]
	if reservation == nil {
		member, score := choosePlacement(state.members, state.reservations, cpu, memory)
		if member == "" {
			// Retry with a new sample, never route an observed unhealthy cluster
			// through LXD's fallback as though monitoring were unsupported.
			state.sampled = time.Time{}
			return "", noop, errPlacementCapacity
		}
		reservation = &placementReservation{member: member, cpu: cpu, memory: memory}
		state.reservations[name] = reservation
		slog.InfoContext(ctx, "MicroCloud selected member using host stats", "instance", name, "project", b.Client.Project, "member", member, "projected_pressure", score, "requested_cpus", cpu, "requested_ram_bytes", memory)
	}
	reservation.users++
	var once sync.Once
	return reservation.member, func() {
		once.Do(func() {
			state.mu.Lock()
			defer state.mu.Unlock()
			reservation.users--
			// Boot load and RAM use take time to reach host stats. Keep a short
			// cooldown even on failure, since creation may still be in progress.
			reservation.expires = time.Now().Add(time.Minute)
		})
	}, nil
}

func choosePlacement(members []placementMember, reservations map[string]*placementReservation, cpu float64, memory uint64) (string, float64) {
	best, bestScore := "", math.Inf(1)
	for _, member := range members {
		if !strings.EqualFold(member.Status, "online") || member.stats == nil {
			continue
		}
		stats := member.stats
		free, load := stats.FreeRAM, math.Max(stats.LoadAverages[0], stats.LoadAverages[1])
		for _, reservation := range reservations {
			if reservation.member == member.Name {
				load += reservation.cpu
				free -= min(free, reservation.memory)
			}
		}
		if free < memory || cpu > float64(stats.LogicalCPUs) {
			continue
		}
		// Compare normalized pressures so a 104-core/1-TB host and a
		// 128-core/2-TB host are judged against their actual capacities.
		pressure := math.Max((load+cpu)/float64(stats.LogicalCPUs), 1-float64(free-memory)/float64(stats.TotalRAM))
		if pressure < bestScore || (pressure == bestScore && member.Name < best) {
			best, bestScore = member.Name, pressure
		}
	}
	return best, bestScore
}

func (s *placementState) refresh(ctx context.Context, b *Builder) error {
	for {
		s.mu.Lock()
		if !s.sampled.IsZero() && time.Since(s.sampled) < placementCacheTTL {
			s.mu.Unlock()
			return nil
		}
		if pending := s.refreshing; pending != nil {
			s.mu.Unlock()
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-pending:
				continue
			}
		}
		s.refreshing = make(chan struct{})
		s.mu.Unlock()
		members, fallback, err := b.samplePlacement(ctx)
		s.mu.Lock()
		if err == nil {
			s.members, s.fallback, s.sampled = members, fallback, time.Now()
		}
		close(s.refreshing)
		s.refreshing = nil
		s.mu.Unlock()
		return err
	}
}

func placementUnsupported(err error) bool {
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		return false
	}
	return apiErr.HTTPStatus == 403 || apiErr.HTTPStatus == 404 || apiErr.HTTPStatus == 501 ||
		(apiErr.HTTPStatus == 400 && strings.Contains(strings.ToLower(apiErr.Message), "not clustered"))
}

// Same endpoints and sysinfo fields as TNMC-STATUS.py. Limit fanout to two
// requests and cap the whole sampling pass so monitoring cannot stall a build.
func (b *Builder) samplePlacement(ctx context.Context) ([]placementMember, string, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	raw, err := b.Client.get(ctx, "/1.0/cluster/members?recursion=1")
	if placementUnsupported(err) {
		return nil, "cluster member API unavailable to this client", nil
	}
	if err != nil {
		return nil, "", err
	}
	var members []placementMember
	if err := json.Unmarshal(raw, &members); err != nil {
		return nil, "", fmt.Errorf("decoding cluster members: %w", err)
	}
	if len(members) == 0 {
		return nil, "", errPlacementCapacity
	}
	// Without resolving a remote image's architecture, explicitly targeting a
	// mixed-architecture cluster would bypass LXD's compatibility selection.
	architecture := members[0].Architecture
	for _, member := range members {
		if architecture == "" || member.Architecture != architecture {
			return nil, "cluster architectures need LXD image compatibility selection", nil
		}
	}
	var allowedGroups []string
	if b.Client.Project != "" && b.Client.Project != "default" {
		raw, err := b.Client.get(ctx, "/1.0/projects/"+url.PathEscape(b.Client.Project))
		if placementUnsupported(err) {
			return nil, "project placement policy unavailable to this client", nil
		}
		if err != nil {
			return nil, "", err
		}
		var project struct {
			Config map[string]string `json:"config"`
		}
		if err := json.Unmarshal(raw, &project); err != nil {
			return nil, "", err
		}
		if project.Config["restricted"] == "true" {
			if project.Config["restricted.cluster.target"] != "allow" {
				return nil, "project restricts explicit cluster targets", nil
			}
			for _, group := range strings.Split(project.Config["restricted.cluster.groups"], ",") {
				if group = strings.TrimSpace(group); group != "" {
					allowedGroups = append(allowedGroups, group)
				}
			}
		}
	}
	sort.Slice(members, func(i, j int) bool { return members[i].Name < members[j].Name })
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 2)
	for i := range members {
		member := &members[i]
		mode := member.Config["scheduler.instance"]
		if member.Name == "" || !strings.EqualFold(member.Status, "online") || (mode != "" && mode != "all") || !placementGroupAllowed(member.Groups, allowedGroups) {
			continue
		}
		select {
		case semaphore <- struct{}{}:
		case <-ctx.Done():
			wg.Wait()
			return nil, "", ctx.Err()
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { <-semaphore }()
			requestCtx, cancel := context.WithTimeout(ctx, time.Minute)
			defer cancel()
			raw, err := b.Client.get(requestCtx, "/1.0/cluster/members/"+url.PathEscape(member.Name)+"/state")
			var state struct {
				Sysinfo memberSysinfo `json:"sysinfo"`
			}
			if err == nil {
				err = json.Unmarshal(raw, &state)
			}
			if err == nil && !state.Sysinfo.valid() {
				err = errors.New("invalid member sysinfo")
			}
			if err != nil {
				slog.WarnContext(ctx, "MicroCloud excluding member with unavailable host stats", "member", member.Name, "error", err)
				return
			}
			member.stats = &state.Sysinfo
		}()
	}
	wg.Wait()
	if err := ctx.Err(); err != nil {
		return nil, "", err
	}
	return members, "", nil
}

func placementGroupAllowed(groups, allowed []string) bool {
	if len(allowed) == 0 {
		return true
	}
	for _, group := range groups {
		for _, permitted := range allowed {
			if group == permitted {
				return true
			}
		}
	}
	return false
}
