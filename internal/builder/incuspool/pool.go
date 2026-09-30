// Package incuspool is the real "multiple independent Incus hosts"
// builder -- kind="incus" in builder_config (migration 00014), distinct
// from kind="microcloud" (internal/builder/microcloud.Builder, its own
// LXD-based builder, pointed at one real cluster endpoint directly -- no pool
// needed since any cluster member answers for the whole thing).
//
// Confirmed directly: "we may have a situation where we have multiple
// incus boxes and we'll need to spread teams intelligently across them as
// it won't necessarily be in a cluster. We'll need to keep teams on
// individual hosts and use a round-robin style approach to spread the
// load amongst those hosts." Pool is that: N genuinely independent
// internal/builder/incus.Builder instances (each its own daemon, its own
// credentials, no shared storage or OVN control plane between them), with
// every call routed to exactly one host by team number -- deterministic
// modulo assignment (team N always resolves to host (N-1)%len(Hosts)),
// not a persisted assignment table. This matters for the same reason
// every other "assigned once, up front" decision in this codebase is
// recomputed rather than stored (see builder.NetworkSpec's own doc
// comment on ExternalName): a retried task, or a runner restarted between
// calls, must resolve the exact same host every time with nothing to
// read back first, and a team's own size (how many objects it has) never
// changes hostFor's answer, so "spread the load" only has to hold at
// content-authoring time (how many teams an environment has), not be
// rebalanced live.
package incuspool

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/globalcptc/laforge/internal/builder"
	"github.com/globalcptc/laforge/internal/builder/incus"
)

// Pool routes every Builder call to one of Hosts by team, and only
// aggregates across all of them for Inspect (which isn't team-scoped) --
// see hostForTeam's own doc comment for the routing rule.
type Pool struct {
	Hosts []*incus.Builder
	// Images is the pool-wide os/image catalog (see migration 00014's own
	// doc comment: content's os/image/size names are pool-wide, not
	// per-host) -- exposed via ImageNames so
	// internal/orchestrator.checkBuilderCompatibility's own per-image
	// validation works identically to a single incus.Builder's
	// Config.Images.
	Images map[string]incus.ImageRef
}

// New constructs a Pool from hosts (already-built single-endpoint
// builders, one per independent Incus host) and the pool-wide image
// catalog. Every host in hosts already carries its own Config (Sizes,
// OVNUplinkNetwork, StoragePool) -- Images is passed separately only
// because ImageNames needs it at the Pool level, not because any
// individual host's own Config.Images differs; internal/builderconfig's
// own resolveIncusPool sets every host's Config.Images to the same map
// this field holds.
func New(hosts []*incus.Builder, images map[string]incus.ImageRef) *Pool {
	return &Pool{Hosts: hosts, Images: images}
}

// ImageNames satisfies the same small, informal interface
// internal/orchestrator.checkBuilderCompatibility already uses for a
// single incus.Builder's own Config.Images.
func (p *Pool) ImageNames() map[string]bool {
	names := make(map[string]bool, len(p.Images))
	for name := range p.Images {
		names[name] = true
	}
	return names
}

// hostForTeam is the whole routing rule: "round-robin style... spread the
// load" made deterministic. team must parse as a positive integer -- the
// same team-number string every other builder call already receives
// (internal/runner formats it via strconv.FormatInt), so a non-numeric
// team here means something upstream is broken, not a real scenario to
// handle gracefully.
func (p *Pool) hostForTeam(team string) (*incus.Builder, error) {
	if len(p.Hosts) == 0 {
		return nil, errors.New("incus pool has no hosts configured")
	}
	n, err := strconv.Atoi(team)
	if err != nil || n < 1 {
		return nil, fmt.Errorf("incus pool: team %q is not a valid team number", team)
	}
	return p.Hosts[(n-1)%len(p.Hosts)], nil
}

func (p *Pool) DeployNetwork(ctx context.Context, spec builder.NetworkSpec) (string, error) {
	h, err := p.hostForTeam(spec.Team)
	if err != nil {
		return "", err
	}
	return h.DeployNetwork(ctx, spec)
}

func (p *Pool) DeployHost(ctx context.Context, spec builder.HostSpec) (string, error) {
	h, err := p.hostForTeam(spec.Team)
	if err != nil {
		return "", err
	}
	return h.DeployHost(ctx, spec)
}

func (p *Pool) DeployContainer(ctx context.Context, spec builder.ContainerSpec) (string, error) {
	h, err := p.hostForTeam(spec.Team)
	if err != nil {
		return "", err
	}
	return h.DeployContainer(ctx, spec)
}

func (p *Pool) DestroyNetwork(ctx context.Context, team, externalRef string) error {
	h, err := p.hostForTeam(team)
	if err != nil {
		return err
	}
	return h.DestroyNetwork(ctx, team, externalRef)
}

func (p *Pool) DestroyHost(ctx context.Context, team, externalRef string) error {
	h, err := p.hostForTeam(team)
	if err != nil {
		return err
	}
	return h.DestroyHost(ctx, team, externalRef)
}

func (p *Pool) DestroyContainer(ctx context.Context, team, externalRef string) error {
	h, err := p.hostForTeam(team)
	if err != nil {
		return err
	}
	return h.DestroyContainer(ctx, team, externalRef)
}

func (p *Pool) PowerAction(ctx context.Context, team, externalRef, action string, force bool) error {
	h, err := p.hostForTeam(team)
	if err != nil {
		return err
	}
	return h.PowerAction(ctx, team, externalRef, action, force)
}

// Inspect aggregates every host's own Inspect -- not team-scoped (Inspect
// lists everything that exists, for adoption/drift detection), and every
// ExternalRef is already globally unique (internal/runner's
// deterministicExternalName embeds build/team/kind/identity), so a plain
// concatenation is correct: no host can report a ref another host also
// has.
func (p *Pool) Inspect(ctx context.Context) ([]builder.Resource, error) {
	var all []builder.Resource
	for i, h := range p.Hosts {
		res, err := h.Inspect(ctx)
		if err != nil {
			return nil, fmt.Errorf("inspecting pool host %d: %w", i, err)
		}
		all = append(all, res...)
	}
	return all, nil
}

func (p *Pool) OpenAccess(ctx context.Context, team string) error {
	h, err := p.hostForTeam(team)
	if err != nil {
		return err
	}
	return h.OpenAccess(ctx, team)
}

func (p *Pool) CloseAccess(ctx context.Context, team string) error {
	h, err := p.hostForTeam(team)
	if err != nil {
		return err
	}
	return h.CloseAccess(ctx, team)
}

// ConfigureNetworkAccess dispatches to the team's own host: all of a team's
// networks live on the one host hostForTeam picks (deterministic per team), so
// the whole visible_from mesh + ACLs are configured there in one place.
func (p *Pool) ConfigureNetworkAccess(ctx context.Context, team string, networks []builder.NetworkAccess) error {
	h, err := p.hostForTeam(team)
	if err != nil {
		return err
	}
	return h.ConfigureNetworkAccess(ctx, team, networks)
}
