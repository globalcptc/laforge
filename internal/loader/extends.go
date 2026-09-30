package loader

// extends.go implements `extends:` inheritance as a deep, child-wins merge,
// folding a base object's fields into every object that extends it. It runs
// once, right after loading and before the cross-file checks, so every
// downstream consumer (validation, render, ingest, fingerprint) sees objects
// that already carry their inherited fields -- there is no separate "resolve
// extends" step anywhere else.
//
// Merge semantics (confirmed design, deep merge):
//   - scalars (os/size/disk/cidr/teams/...): the child's value wins when it
//     set one; otherwise it inherits the base's. This is what lets an
//     extending host omit os/size/disk -- the schema makes them optional
//     exactly when `extends:` is present, and the base supplies them here.
//   - maps (vars/tags/env): merged key by key, the child winning a clash.
//   - lists (steps/schedule/people/findings/ports/depends_on/...): the base's
//     come first, then the child's are appended -- a base can supply common
//     setup and the child adds to it. Set-like lists (depends_on, ports,
//     visible_from) are de-duplicated.
//
// Chains resolve transitively (a base may itself extend another), base-first,
// and are cycle-safe: an in-progress node short-circuits re-entry so a
// mistaken A->B->A never loops (checkExtends reports the cycle as an error
// separately). A missing base is likewise skipped here and reported there.

// resolveExtends folds every object's `extends:` base into it, in place.
func (c *Content) resolveExtends() {
	c.resolveNetworkExtends()
	c.resolveHostExtends()
	c.resolveContainerExtends()
	c.resolveEnvironmentExtends()
}

// extendState tracks a node during transitive resolution: unvisited, in
// progress (on the current chain -- re-entering it means a cycle), or done.
type extendState int8

const (
	extendUnvisited extendState = iota
	extendInProgress
	extendDone
)

func (c *Content) resolveHostExtends() {
	idx := make(map[string]int, len(c.Hosts))
	for i := range c.Hosts {
		idx[c.Hosts[i].Name] = i
	}
	state := make([]extendState, len(c.Hosts))
	var resolve func(i int)
	resolve = func(i int) {
		if state[i] != extendUnvisited { // done, or in-progress (cycle): stop
			return
		}
		state[i] = extendInProgress
		h := &c.Hosts[i]
		if bi, ok := idx[h.Extends]; ok && h.Extends != "" && h.Extends != h.Name {
			resolve(bi)
			*h = mergeHost(*h, c.Hosts[bi])
		}
		state[i] = extendDone
	}
	for i := range c.Hosts {
		resolve(i)
	}
}

func (c *Content) resolveContainerExtends() {
	idx := make(map[string]int, len(c.Containers))
	for i := range c.Containers {
		idx[c.Containers[i].Name] = i
	}
	state := make([]extendState, len(c.Containers))
	var resolve func(i int)
	resolve = func(i int) {
		if state[i] != extendUnvisited {
			return
		}
		state[i] = extendInProgress
		ct := &c.Containers[i]
		if bi, ok := idx[ct.Extends]; ok && ct.Extends != "" && ct.Extends != ct.Name {
			resolve(bi)
			*ct = mergeContainer(*ct, c.Containers[bi])
		}
		state[i] = extendDone
	}
	for i := range c.Containers {
		resolve(i)
	}
}

func (c *Content) resolveNetworkExtends() {
	idx := make(map[string]int, len(c.Networks))
	for i := range c.Networks {
		idx[c.Networks[i].Name] = i
	}
	state := make([]extendState, len(c.Networks))
	var resolve func(i int)
	resolve = func(i int) {
		if state[i] != extendUnvisited {
			return
		}
		state[i] = extendInProgress
		n := &c.Networks[i]
		if bi, ok := idx[n.Extends]; ok && n.Extends != "" && n.Extends != n.Name {
			resolve(bi)
			*n = mergeNetwork(*n, c.Networks[bi])
		}
		state[i] = extendDone
	}
	for i := range c.Networks {
		resolve(i)
	}
}

func (c *Content) resolveEnvironmentExtends() {
	idx := make(map[string]int, len(c.Environments))
	for i := range c.Environments {
		idx[c.Environments[i].Name] = i
	}
	state := make([]extendState, len(c.Environments))
	var resolve func(i int)
	resolve = func(i int) {
		if state[i] != extendUnvisited {
			return
		}
		state[i] = extendInProgress
		e := &c.Environments[i]
		if bi, ok := idx[e.Extends]; ok && e.Extends != "" && e.Extends != e.Name {
			resolve(bi)
			*e = mergeEnvironment(*e, c.Environments[bi])
		}
		state[i] = extendDone
	}
	for i := range c.Environments {
		resolve(i)
	}
}

// mergeHost returns child with base's fields folded in (child wins).
func mergeHost(child, base Host) Host {
	child.OS = orString(child.OS, base.OS)
	child.Size = orString(child.Size, base.Size)
	child.Disk = orInt(child.Disk, base.Disk)
	child.Ports = mergePorts(base.Ports, child.Ports)
	child.DependsOn = dedupAppend(base.DependsOn, child.DependsOn)
	child.Steps = appendSteps(base.Steps, child.Steps)
	child.Schedule = appendSteps(base.Schedule, child.Schedule)
	child.Vars = mergeStringMap(base.Vars, child.Vars)
	child.Tags = mergeStringMap(base.Tags, child.Tags)
	child.Findings = appendFindings(base.Findings, child.Findings)
	child.People = appendPeople(base.People, child.People)
	return child
}

func mergeContainer(child, base Container) Container {
	child.Image = orString(child.Image, base.Image)
	child.Size = orString(child.Size, base.Size)
	child.Env = mergeStringMap(base.Env, child.Env)
	child.Command = appendStrings(base.Command, child.Command)
	child.Ports = mergePorts(base.Ports, child.Ports)
	child.DependsOn = dedupAppend(base.DependsOn, child.DependsOn)
	child.Steps = appendSteps(base.Steps, child.Steps)
	child.Schedule = appendSteps(base.Schedule, child.Schedule)
	child.Vars = mergeStringMap(base.Vars, child.Vars)
	child.Tags = mergeStringMap(base.Tags, child.Tags)
	child.Findings = appendFindings(base.Findings, child.Findings)
	child.People = appendPeople(base.People, child.People)
	return child
}

func mergeNetwork(child, base Network) Network {
	child.CIDR = orString(child.CIDR, base.CIDR)
	child.VisibleFrom = dedupAppend(base.VisibleFrom, child.VisibleFrom)
	child.Vars = mergeStringMap(base.Vars, child.Vars)
	child.Tags = mergeStringMap(base.Tags, child.Tags)
	child.Findings = appendFindings(base.Findings, child.Findings)
	return child
}

func mergeEnvironment(child, base Environment) Environment {
	child.Schema = orInt(child.Schema, base.Schema)
	child.Description = orString(child.Description, base.Description)
	child.Teams = orInt(child.Teams, base.Teams)
	child.RootPass = orString(child.RootPass, base.RootPass)
	child.Start = orString(child.Start, base.Start)
	child.Stop = orString(child.Stop, base.Stop)
	if child.DNS == nil {
		child.DNS = base.DNS
	}
	child.Access = append(append([]AccessWindow{}, base.Access...), child.Access...)
	child.Vars = mergeStringMap(base.Vars, child.Vars)
	child.Tags = mergeStringMap(base.Tags, child.Tags)
	child.Findings = appendFindings(base.Findings, child.Findings)
	child.Networks = mergeNetworksTopology(base.Networks, child.Networks)
	return child
}

// mergeNetworksTopology merges an environment's `networks:` placement map. It
// is a union at network and object granularity; when both base and child place
// the same object on the same network, the child's copy list replaces the
// base's (an override, not a concat -- appending would place the object twice).
func mergeNetworksTopology(base, child map[string]map[string][]Copy) map[string]map[string][]Copy {
	if base == nil && child == nil {
		return nil
	}
	out := make(map[string]map[string][]Copy)
	for net, objs := range base {
		out[net] = make(map[string][]Copy, len(objs))
		for obj, copies := range objs {
			out[net][obj] = copies
		}
	}
	for net, objs := range child {
		if out[net] == nil {
			out[net] = make(map[string][]Copy, len(objs))
		}
		for obj, copies := range objs {
			out[net][obj] = copies // child wins for this object on this network
		}
	}
	return out
}

// --- small merge helpers ---

func orString(child, base string) string {
	if child != "" {
		return child
	}
	return base
}

func orInt(child, base int) int {
	if child != 0 {
		return child
	}
	return base
}

// mergeStringMap overlays child onto base (child wins), returning a new map so
// no object shares its base's backing map. Returns nil when both are empty, to
// keep round-trips (and fingerprints) identical to the un-extended case.
func mergeStringMap(base, child map[string]string) map[string]string {
	if len(base) == 0 && len(child) == 0 {
		return nil
	}
	out := make(map[string]string, len(base)+len(child))
	for k, v := range base {
		out[k] = v
	}
	for k, v := range child {
		out[k] = v
	}
	return out
}

func mergePorts(base, child Ports) Ports {
	return Ports{
		TCP: dedupAppend(base.TCP, child.TCP),
		UDP: dedupAppend(base.UDP, child.UDP),
	}
}

// dedupAppend returns base's items followed by child's, dropping duplicates and
// preserving first-seen order. For set-like lists (depends_on, ports, ...).
func dedupAppend(base, child []string) []string {
	if len(base) == 0 && len(child) == 0 {
		return nil
	}
	seen := make(map[string]bool, len(base)+len(child))
	out := make([]string, 0, len(base)+len(child))
	for _, s := range append(append([]string{}, base...), child...) {
		if seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}

// appendStrings returns base's items followed by child's, keeping duplicates
// and order (for ordered lists like a container command).
func appendStrings(base, child []string) []string {
	if len(base) == 0 && len(child) == 0 {
		return nil
	}
	return append(append([]string{}, base...), child...)
}

func appendSteps(base, child []Step) []Step {
	if len(base) == 0 && len(child) == 0 {
		return nil
	}
	return append(append([]Step{}, base...), child...)
}

func appendFindings(base, child []Finding) []Finding {
	if len(base) == 0 && len(child) == 0 {
		return nil
	}
	return append(append([]Finding{}, base...), child...)
}

func appendPeople(base, child []PeopleRef) []PeopleRef {
	if len(base) == 0 && len(child) == 0 {
		return nil
	}
	return append(append([]PeopleRef{}, base...), child...)
}
