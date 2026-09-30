// Build -> Topology: "what the environment resolved into: teams, networks
// with CIDRs, hosts and containers with addresses and dependencies.
// Readable as a tree". The tree is the real deployed
// copies (deployed_object rows, per team) grouped under their network,
// enriched with the content facts a copy doesn't itself carry -- a
// network's CIDR, a host's OS, a container's image, each object's size and
// its depends_on -- looked up from the build's own content revision the
// same cached-by-name way resolveFindings does (internal/api/findings.go).
// Addresses aren't tracked in this rewrite (the builder assigns them and we
// don't record them yet), so they're deliberately absent rather than shown
// blank.
package api

import (
	"encoding/json"
	"net/http"
	"sort"

	"github.com/globalcptc/laforge/internal/db"
)

type topologyMember struct {
	ObjectID   string   `json:"object_id"`
	Kind       string   `json:"kind"` // host | container
	ObjectName string   `json:"object_name"`
	AsName     string   `json:"as_name,omitempty"`
	Status     string   `json:"status"`
	PowerState string   `json:"power_state,omitempty"`
	OS         string            `json:"os,omitempty"`    // host only
	Image      string            `json:"image,omitempty"` // container only
	Size       string            `json:"size,omitempty"`
	DependsOn  []string          `json:"depends_on,omitempty"`
	Tags       map[string]string `json:"tags,omitempty"` // content tags -- for search/filter/operate on the Hosts page
}

type topologyNetwork struct {
	ObjectID string           `json:"object_id,omitempty"` // the network's own deployed_object id, if it has one
	Name     string           `json:"name"`
	Cidr     string           `json:"cidr,omitempty"`
	Status   string           `json:"status,omitempty"`
	Members  []topologyMember `json:"members"`
}

type topologyTeam struct {
	TeamNumber int32             `json:"team_number"`
	Networks   []topologyNetwork `json:"networks"`
}

type topologyResponse struct {
	Teams []topologyTeam `json:"teams"`
}

// noNetworkGroup is the bucket for a host/container the content places on no
// network -- it still belongs in the tree, just not under a network node.
const noNetworkGroup = ""

func decodeStringList(raw []byte) []string {
	if len(raw) == 0 {
		return nil
	}
	var out []string
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil
	}
	return out
}

func decodeTags(raw []byte) map[string]string {
	if len(raw) == 0 {
		return nil
	}
	var out map[string]string
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil
	}
	return out
}

func (s *Server) handleGetTopology(w http.ResponseWriter, r *http.Request) {
	build, repo, err := s.buildAndOwningRepository(r)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	if _, err := s.requireLevel(r.Context(), r, repo, levelRead); err != nil {
		writeAuthError(w, err)
		return
	}

	teams, err := s.Queries.ListTeamsByBuild(r.Context(), build.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	teamNumberByID := make(map[string]int32, len(teams))
	for _, tm := range teams {
		teamNumberByID[tm.ID.String()] = tm.TeamNumber
	}

	objs, err := s.Queries.ListDeployedObjectsByBuild(r.Context(), build.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	rev := build.ContentRevisionID
	// Cache content lookups by name across teams -- every team resolves the
	// same definitions, so this is a handful of queries, not one per copy.
	netCidr := map[string]string{}
	getNetCidr := func(name string) string {
		if v, ok := netCidr[name]; ok {
			return v
		}
		v := ""
		if n, err := s.Queries.GetNetworkByRevisionAndName(r.Context(), db.GetNetworkByRevisionAndNameParams{ContentRevisionID: rev, Name: name}); err == nil {
			v = n.Cidr
		}
		netCidr[name] = v
		return v
	}
	type hostMeta struct {
		os, size string
		deps     []string
		tags     map[string]string
	}
	hostCache := map[string]hostMeta{}
	getHost := func(name string) hostMeta {
		if v, ok := hostCache[name]; ok {
			return v
		}
		var m hostMeta
		if h, err := s.Queries.GetHostByRevisionAndName(r.Context(), db.GetHostByRevisionAndNameParams{ContentRevisionID: rev, Name: name}); err == nil {
			m = hostMeta{os: db.StrOrEmpty(h.Os), size: db.StrOrEmpty(h.Size), deps: decodeStringList(h.DependsOn), tags: decodeTags(h.Tags)}
		}
		hostCache[name] = m
		return m
	}
	type ctrMeta struct {
		image, size string
		deps        []string
		tags        map[string]string
	}
	ctrCache := map[string]ctrMeta{}
	getContainer := func(name string) ctrMeta {
		if v, ok := ctrCache[name]; ok {
			return v
		}
		var m ctrMeta
		if c, err := s.Queries.GetContainerByRevisionAndName(r.Context(), db.GetContainerByRevisionAndNameParams{ContentRevisionID: rev, Name: name}); err == nil {
			m = ctrMeta{image: db.StrOrEmpty(c.Image), size: db.StrOrEmpty(c.Size), deps: decodeStringList(c.DependsOn), tags: decodeTags(c.Tags)}
		}
		ctrCache[name] = m
		return m
	}

	// Per team, group members under their network, tracking the network's
	// own deployed_object row (for its status/CIDR and a drill-in id).
	type teamBuild struct {
		order    []string // stable network order: content order isn't recorded, so first-seen
		networks map[string]*topologyNetwork
	}
	byTeam := map[int32]*teamBuild{}
	ensureTeam := func(tn int32) *teamBuild {
		if byTeam[tn] == nil {
			byTeam[tn] = &teamBuild{networks: map[string]*topologyNetwork{}}
		}
		return byTeam[tn]
	}
	ensureNet := func(tb *teamBuild, name string) *topologyNetwork {
		if tb.networks[name] == nil {
			tb.networks[name] = &topologyNetwork{Name: name, Cidr: getNetCidr(name), Members: []topologyMember{}}
			tb.order = append(tb.order, name)
		}
		return tb.networks[name]
	}

	for _, o := range objs {
		tn, ok := teamNumberByID[o.TeamID.String()]
		if !ok {
			continue
		}
		tb := ensureTeam(tn)
		switch o.Kind {
		case "network":
			n := ensureNet(tb, o.ObjectName)
			n.ObjectID = o.ID.String()
			n.Status = o.Status
			if n.Cidr == "" {
				n.Cidr = getNetCidr(o.ObjectName)
			}
		case "host", "container":
			netName := db.StrOrEmpty(o.NetworkName)
			n := ensureNet(tb, netName)
			m := topologyMember{
				ObjectID: o.ID.String(), Kind: o.Kind, ObjectName: o.ObjectName,
				AsName: db.StrOrEmpty(o.AsName), Status: o.Status, PowerState: o.PowerState,
			}
			if o.Kind == "host" {
				h := getHost(o.ObjectName)
				m.OS, m.Size, m.DependsOn, m.Tags = h.os, h.size, h.deps, h.tags
			} else {
				c := getContainer(o.ObjectName)
				m.Image, m.Size, m.DependsOn, m.Tags = c.image, c.size, c.deps, c.tags
			}
			n.Members = append(n.Members, m)
		}
	}

	teamNums := make([]int32, 0, len(byTeam))
	for tn := range byTeam {
		teamNums = append(teamNums, tn)
	}
	sort.Slice(teamNums, func(i, j int) bool { return teamNums[i] < teamNums[j] })

	resp := topologyResponse{Teams: make([]topologyTeam, 0, len(teamNums))}
	for _, tn := range teamNums {
		tb := byTeam[tn]
		// A real network node first, then the no-network bucket last.
		sort.SliceStable(tb.order, func(i, j int) bool {
			if (tb.order[i] == noNetworkGroup) != (tb.order[j] == noNetworkGroup) {
				return tb.order[j] == noNetworkGroup
			}
			return tb.order[i] < tb.order[j]
		})
		nets := make([]topologyNetwork, 0, len(tb.order))
		for _, name := range tb.order {
			n := tb.networks[name]
			sort.SliceStable(n.Members, func(i, j int) bool {
				a, b := n.Members[i], n.Members[j]
				if a.AsName != b.AsName {
					return a.AsName < b.AsName
				}
				return a.ObjectName < b.ObjectName
			})
			nets = append(nets, *n)
		}
		resp.Teams = append(resp.Teams, topologyTeam{TeamNumber: tn, Networks: nets})
	}
	writeJSON(w, http.StatusOK, resp)
}
