// Package render resolves
// the template context for one host copy in one team ("laforge context"),
// rendering a script or step-field string against it ("laforge render"),
// and doing that for every host in every team ("laforge check").
package render

import (
	"fmt"
	"sort"

	"github.com/globalcptc/laforge/internal/loader"
)

// VarEntry is one effective variable, with where it came from and what it
// shadowed -- "every var with where it came from in the cascade."
type VarEntry struct {
	Key      string
	Value    string
	Source   string // "environment", "network", or "host"/"container"
	Shadowed []ShadowedValue
}

type ShadowedValue struct {
	Source string
	Value  string
}

type NetworkPeer struct {
	As      string
	Kind    string // "host" or "container"
	Address string
}

type PeopleSummary struct {
	Name    string
	Rows    int
	Columns []string
}

// Context is everything resolved for one host-or-container copy in one
// team: the presentation-friendly view (for `laforge context`) and,
// via TemplateData, the flat form a template actually executes against.
type Context struct {
	EnvironmentName string
	Team            int
	TeamsTotal      int
	Start, Stop     string

	As          string // the copy's `as` name -- the hostname
	ObjectName  string // the underlying host/container definition's name
	ObjectKind  string // "host" or "container"
	OS          string // set when ObjectKind == "host"
	Image       string // set when ObjectKind == "container"
	Size        string
	Address     string
	NetworkName string
	NetworkCIDR string
	Peers       []NetworkPeer // other copies on the same network, same team

	Vars   []VarEntry
	People []PeopleSummary

	// ObjectPeople is THIS object's own resolved `people:` set -- every row from
	// each people/*.csv file its `people:` list names (narrowed by any `filter:`),
	// as flat attribute maps (columns + username). Exposed to scripts as
	// `{{ .people }}`, so an account-creation script iterates exactly the people
	// assigned to this host, not a globally-named source. Distinct from People
	// above, which is the whole-repo source summary the environment view shows.
	ObjectPeople []map[string]string

	// DNS is the environment's full resolved record set (an A record per host
	// from its address, plus the environment's own dns.records), exposed in the
	// template data. LaForge does not run DNS -- it's configured by a script on
	// the Domain Controller / Bind host, which renders these into a zone file /
	// hosts file / PowerShell. Same set for every team (the topology is
	// identical per team), so it doesn't depend on ctx.Team.
	DNS []DNSRecord

	// Steps is the resolved object's own step list, used by check.go to
	// know what to render. Not part of the template data.
	Steps []loader.Step

	// Schedule is the resolved object's own schedule list -- a separate
	// field from Steps, not nested inside it (see loader.Host.Schedule's
	// own comment for why). Also not part of the template data; check.go
	// uses it the same way it uses Steps, to validate each entry's own
	// script: reference.
	Schedule []loader.Step
}

// Resolve finds the copy named `as` in environment `envName`'s topology,
// for `team`, and builds its full Context. team is 1-indexed, matching how
// the plan's own CLI examples write it ("--team 3").
func Resolve(c *loader.Content, envName, as string, team int) (*Context, error) {
	env := findEnvironment(c, envName)
	if env == nil {
		return nil, fmt.Errorf("no environment named %q in this repository", envName)
	}
	if team < 1 || team > env.Teams {
		return nil, fmt.Errorf("team %d is out of range: environment %q has %d teams", team, envName, env.Teams)
	}

	netName, objName, copy, found := findCopy(env, as)
	if !found {
		return nil, fmt.Errorf("no placement named %q in environment %q", as, envName)
	}

	network := findNetwork(c, netName)
	if network == nil {
		// Already caught by loader.checkEnvironmentTopology on a real
		// repo, but Resolve is also usable standalone (e.g. by a future
		// language server) against partially-invalid content.
		return nil, fmt.Errorf("environment %q references network %q, which does not exist", envName, netName)
	}

	ctx := &Context{
		EnvironmentName: envName,
		Team:            team,
		TeamsTotal:      env.Teams,
		Start:           env.Start,
		Stop:            env.Stop,
		As:              as,
		ObjectName:      objName,
		NetworkName:     netName,
		NetworkCIDR:     network.CIDR,
	}

	addr, err := Address(network.CIDR, copy.LastOctet)
	if err != nil {
		return nil, fmt.Errorf("computing address for %q: %w", as, err)
	}
	ctx.Address = addr

	var objVars map[string]string
	var objPeople []loader.PeopleRef
	if h := findHost(c, objName); h != nil {
		ctx.ObjectKind = "host"
		ctx.OS = h.OS
		ctx.Size = h.Size
		ctx.Steps = h.Steps
		ctx.Schedule = h.Schedule
		objVars = h.Vars
		objPeople = h.People
	} else if ct := findContainer(c, objName); ct != nil {
		ctx.ObjectKind = "container"
		ctx.Image = ct.Image
		ctx.Size = ct.Size
		ctx.Steps = ct.Steps
		ctx.Schedule = ct.Schedule
		objVars = ct.Vars
		objPeople = ct.People
	} else {
		return nil, fmt.Errorf("environment %q places %q, which is not a defined host or container", envName, objName)
	}

	ctx.Peers = findPeers(c, env, netName, as)
	ctx.Vars = mergeVars(env.Vars, network.Vars, objVars)
	ctx.People = peopleSummaries(c)
	ctx.ObjectPeople = resolveObjectPeople(c, objPeople)
	// Best-effort: a records-resolution problem (already caught by the loader on
	// a real repo) leaves DNS empty rather than failing to render an object that
	// doesn't use it.
	ctx.DNS, _ = DNSRecords(c, env)

	return ctx, nil
}

func findEnvironment(c *loader.Content, name string) *loader.Environment {
	for i := range c.Environments {
		if c.Environments[i].Name == name {
			return &c.Environments[i]
		}
	}
	return nil
}

func findNetwork(c *loader.Content, name string) *loader.Network {
	for i := range c.Networks {
		if c.Networks[i].Name == name {
			return &c.Networks[i]
		}
	}
	return nil
}

func findHost(c *loader.Content, name string) *loader.Host {
	for i := range c.Hosts {
		if c.Hosts[i].Name == name {
			return &c.Hosts[i]
		}
	}
	return nil
}

func findContainer(c *loader.Content, name string) *loader.Container {
	for i := range c.Containers {
		if c.Containers[i].Name == name {
			return &c.Containers[i]
		}
	}
	return nil
}

// findCopy searches every network/object in the environment's topology for
// a copy with this `as` name. Unambiguous because the loader's
// checkEnvironmentTopology already enforces `as` uniqueness within an
// environment -- see internal/loader/checks.go.
func findCopy(env *loader.Environment, as string) (netName, objName string, copy loader.Copy, found bool) {
	// Sorted iteration so results are deterministic even though this can
	// only ever match once -- makes any future ambiguity bug reproducible
	// rather than order-dependent.
	var nets []string
	for n := range env.Networks {
		nets = append(nets, n)
	}
	sort.Strings(nets)
	for _, n := range nets {
		var objs []string
		for o := range env.Networks[n] {
			objs = append(objs, o)
		}
		sort.Strings(objs)
		for _, o := range objs {
			for _, cp := range env.Networks[n][o] {
				if cp.As == as {
					return n, o, cp, true
				}
			}
		}
	}
	return "", "", loader.Copy{}, false
}

func findPeers(c *loader.Content, env *loader.Environment, netName, excludeAs string) []NetworkPeer {
	var peers []NetworkPeer
	objs := env.Networks[netName]
	var names []string
	for o := range objs {
		names = append(names, o)
	}
	sort.Strings(names)
	for _, objName := range names {
		kind := "host"
		if findHost(c, objName) == nil {
			kind = "container"
		}
		for _, cp := range objs[objName] {
			if cp.As == excludeAs {
				continue
			}
			peers = append(peers, NetworkPeer{As: cp.As, Kind: kind})
		}
	}
	return peers
}

// mergeVars implements "environment < network < host/container" --
// precedence when the same key appears twice -- and records every level
// that set each key, not just the winner.
func mergeVars(levels ...map[string]string) []VarEntry {
	sourceNames := []string{"environment", "network", "host/container"}
	byKey := make(map[string][]ShadowedValue) // in precedence order
	var order []string
	for i, lvl := range levels {
		var keys []string
		for k := range lvl {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			if _, seen := byKey[k]; !seen {
				order = append(order, k)
			}
			byKey[k] = append(byKey[k], ShadowedValue{Source: sourceNames[i], Value: lvl[k]})
		}
	}
	sort.Strings(order)
	var out []VarEntry
	for _, k := range order {
		vals := byKey[k]
		winner := vals[len(vals)-1]
		e := VarEntry{Key: k, Value: winner.Value, Source: winner.Source}
		if len(vals) > 1 {
			e.Shadowed = vals[:len(vals)-1]
		}
		out = append(out, e)
	}
	return out
}

func peopleSummaries(c *loader.Content) []PeopleSummary {
	var out []PeopleSummary
	for _, src := range c.People {
		colSet := make(map[string]bool)
		for _, p := range src.People {
			for col := range p.Attributes {
				colSet[col] = true
			}
		}
		var cols []string
		for col := range colSet {
			cols = append(cols, col)
		}
		sort.Strings(cols)
		out = append(out, PeopleSummary{Name: src.Name, Rows: len(src.People), Columns: cols})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// mergePeopleRows appends extra rows to base, dropping any whose `username`
// already appears in base -- so a script's own `people:` adds to the host's
// assigned people without double-listing anyone already on the host.
func mergePeopleRows(base, extra []map[string]string) []map[string]string {
	if len(extra) == 0 {
		return base
	}
	seen := make(map[string]bool, len(base))
	for _, r := range base {
		seen[r["username"]] = true
	}
	out := append([]map[string]string{}, base...)
	for _, r := range extra {
		if seen[r["username"]] {
			continue
		}
		seen[r["username"]] = true
		out = append(out, r)
	}
	return out
}

// resolveObjectPeople turns an object's `people:` refs into the flat row list a
// script sees as `{{ .people }}`: every row of each named people/*.csv file (or
// just the filtered usernames), as attribute maps with `username` included. A
// ref naming a file that isn't loaded, or a filter naming a username not in it,
// contributes nothing here -- checkPeopleReferences already reports those as real
// validation errors, so rendering doesn't need to fail a second time. Order is
// stable: refs in list order, rows in CSV order; a username already added by an
// earlier ref is not duplicated by a later one.
func resolveObjectPeople(c *loader.Content, refs []loader.PeopleRef) []map[string]string {
	if len(refs) == 0 {
		return nil
	}
	sources := make(map[string][]loader.Person, len(c.People))
	for _, src := range c.People {
		sources[src.Name] = src.People
	}
	var out []map[string]string
	seen := make(map[string]bool)
	for _, ref := range refs {
		people, ok := sources[ref.File]
		if !ok {
			continue // unknown file -- already a loader error
		}
		want := make(map[string]bool, len(ref.Filter))
		for _, u := range ref.Filter {
			want[u] = true
		}
		for _, p := range people {
			if len(ref.Filter) > 0 && !want[p.Username] {
				continue
			}
			if seen[p.Username] {
				continue
			}
			seen[p.Username] = true
			row := make(map[string]string, len(p.Attributes)+1)
			for k, v := range p.Attributes {
				row[k] = v
			}
			row["username"] = p.Username
			out = append(out, row)
		}
	}
	return out
}

// EffectiveVars is the flat key->value map after cascade resolution, the
// form a template actually needs (see template.go).
func (ctx *Context) EffectiveVars() map[string]string {
	m := make(map[string]string, len(ctx.Vars))
	for _, v := range ctx.Vars {
		m[v.Key] = v.Value
	}
	return m
}
