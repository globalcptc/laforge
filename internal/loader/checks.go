package loader

import (
	"fmt"
	"strings"

	"github.com/globalcptc/laforge/internal/schedule"
)

// crossCheck runs everything a single document's JSON Schema can't express
// on its own, because it needs to see the whole loaded repository at once.
// Per `laforge check`: "name collisions, depends_on
// targets that exist, public ports being a subset of the host's ports, and
// validators asking for the wrong platform."
//
// `extends:` bases are already folded into their children by resolveExtends
// (see extends.go) before this runs, so every check below sees merged
// objects. checkExtends still validates the reference itself (same-type
// target, no self- or multi-node cycle), and checkResolvedRequired confirms
// that after merging an extending object actually has the fields the schema
// let it omit (os/size/disk, image/size, cidr, teams).
func (c *Content) crossCheck() {
	c.checkNameCollisions()
	c.checkDependsOn()
	c.checkEnvironmentTopology()
	c.checkPublicPorts()
	c.checkPeopleReferences()
	c.checkNetworkVisibility()
	c.checkScriptReferences()
	c.checkScheduleExpressions()
	c.checkValidatorPlatforms()
	c.checkExtends()
	c.checkResolvedRequired()
}

// objectNames returns every host+container name (they share one namespace)
// mapped to its source file, for existence and collision checks.
func (c *Content) objectNames() map[string]string {
	m := make(map[string]string)
	for _, h := range c.Hosts {
		m[h.Name] = h.SourceFile
	}
	for _, ct := range c.Containers {
		m[ct.Name] = ct.SourceFile
	}
	return m
}

func (c *Content) networkNames() map[string]string {
	m := make(map[string]string)
	for _, n := range c.Networks {
		m[n.Name] = n.SourceFile
	}
	return m
}

func (c *Content) checkNameCollisions() {
	// Hosts and containers share one namespace -- "Names are unique per
	// type. Hosts and containers share one namespace, because an
	// environment refers to either by name."
	seen := make(map[string]string)
	check := func(name, file string) {
		if first, dup := seen[name]; dup {
			c.addf(file, 0, "host/container name %q collides with %s -- hosts and containers share one namespace", name, first)
		}
		seen[name] = file
	}
	for _, h := range c.Hosts {
		check(h.Name, h.SourceFile)
	}
	for _, ct := range c.Containers {
		check(ct.Name, ct.SourceFile)
	}

	seenNet := make(map[string]string)
	for _, n := range c.Networks {
		if first, dup := seenNet[n.Name]; dup {
			c.addf(n.SourceFile, 0, "network name %q collides with %s", n.Name, first)
		}
		seenNet[n.Name] = n.SourceFile
	}

	seenScript := make(map[string]string)
	for _, s := range c.Scripts {
		if first, dup := seenScript[s.Name]; dup {
			c.addf(s.SourceFile, 0, "script name %q collides with %s", s.Name, first)
		}
		seenScript[s.Name] = s.SourceFile
	}

	seenEnv := make(map[string]string)
	for _, e := range c.Environments {
		if first, dup := seenEnv[e.Name]; dup {
			c.addf(e.SourceFile, 0, "environment name %q collides with %s", e.Name, first)
		}
		seenEnv[e.Name] = e.SourceFile
	}
}

// checkDependsOn resolves every host/container's depends_on entries against
// the combined host/container/network namespace. "A dependency naming
// something not in the environment fails validation" -- here "the
// environment" is read as "the repository," since depends_on names an
// object, not a placement, and objects are repository-wide.
func (c *Content) checkDependsOn() {
	objects := c.objectNames()
	networks := c.networkNames()

	resolve := func(file, name string) {
		if name == "" {
			return
		}
		if _, ok := objects[name]; ok {
			return
		}
		if _, ok := networks[name]; ok {
			return
		}
		c.addf(file, 0, "depends_on references %q, which is not a host, container, or network in this repository", name)
	}
	for _, h := range c.Hosts {
		for _, d := range h.DependsOn {
			resolve(h.SourceFile, d)
		}
	}
	for _, ct := range c.Containers {
		for _, d := range ct.DependsOn {
			resolve(ct.SourceFile, d)
		}
	}
	c.checkDependsOnCycles()
}

// checkDependsOnCycles rejects a depends_on cycle among hosts/containers.
// Ordering is enforced at deploy time by holding an object's deploy task until
// its dependencies are up (see reconcile), so a cycle would otherwise wedge the
// whole build with every object stuck pending, waiting on the other. Networks
// are ignored here -- they have no outgoing depends_on, so they can't be on a
// cycle. Reports each object that sits on a cycle.
func (c *Content) checkDependsOnCycles() {
	// adjacency + source file, over the shared host/container namespace.
	deps := make(map[string][]string)
	file := make(map[string]string)
	objects := c.objectNames()
	add := func(name, src string, dependsOn []string) {
		file[name] = src
		for _, d := range dependsOn {
			if _, ok := objects[d]; ok { // object dep only (skip networks/unknowns)
				deps[name] = append(deps[name], d)
			}
		}
	}
	for _, h := range c.Hosts {
		add(h.Name, h.SourceFile, h.DependsOn)
	}
	for _, ct := range c.Containers {
		add(ct.Name, ct.SourceFile, ct.DependsOn)
	}

	const (
		white = 0 // unvisited
		gray  = 1 // on the current DFS stack
		black = 2 // done
	)
	color := make(map[string]int)
	var dfs func(name string) bool // returns true if a cycle was found through name
	dfs = func(name string) bool {
		color[name] = gray
		for _, d := range deps[name] {
			if color[d] == gray {
				c.addf(file[name], 0, "host/container %q is part of a depends_on cycle (via %q)", name, d)
				return true
			}
			if color[d] == white && dfs(d) {
				return true
			}
		}
		color[name] = black
		return false
	}
	for name := range deps {
		if color[name] == white {
			if dfs(name) {
				return // one clear cycle error is enough
			}
		}
	}
}

// checkEnvironmentTopology resolves every environment's `networks:` block:
// network names must exist, object names must exist, and `as` names must be
// unique within the environment (they become hostnames).
func (c *Content) checkEnvironmentTopology() {
	networks := c.networkNames()
	objects := c.objectNames()

	for _, e := range c.Environments {
		asNames := make(map[string]bool)
		for netName, objMap := range e.Networks {
			if _, ok := networks[netName]; !ok {
				c.addf(e.SourceFile, 0, "environment %q uses network %q, which does not exist in this repository", e.Name, netName)
			}
			for objName, copies := range objMap {
				if _, ok := objects[objName]; !ok {
					c.addf(e.SourceFile, 0, "environment %q places %q on network %q, but no host or container named %q exists", e.Name, objName, netName, objName)
					continue
				}
				for _, cp := range copies {
					if cp.As != "" {
						if asNames[cp.As] {
							c.addf(e.SourceFile, 0, "environment %q has two copies named %q -- `as` becomes the hostname and must be unique within the environment", e.Name, cp.As)
						}
						asNames[cp.As] = true
					}
				}
			}
		}
	}
}

func portSet(p Ports) map[string]bool {
	s := make(map[string]bool, len(p.TCP)+len(p.UDP))
	for _, t := range p.TCP {
		s["tcp/"+t] = true
	}
	for _, u := range p.UDP {
		s["udp/"+u] = true
	}
	return s
}

// checkPublicPorts validates that every host/container's `public:` ports are a
// subset of its own declared `ports` -- you can only expose a port the host
// actually serves. `public:` is a property of the host (it lives in the host
// file), so this is checked per definition, not per environment placement.
func (c *Content) checkPublicPorts() {
	check := func(file, name string, declared Ports, public *Ports) {
		if public == nil {
			return
		}
		set := portSet(declared)
		for _, t := range public.TCP {
			if !set["tcp/"+t] {
				c.addf(file, 0, "%q publishes tcp port %s in `public`, which is not in its own `ports`", name, t)
			}
		}
		for _, u := range public.UDP {
			if !set["udp/"+u] {
				c.addf(file, 0, "%q publishes udp port %s in `public`, which is not in its own `ports`", name, u)
			}
		}
	}
	for _, h := range c.Hosts {
		check(h.SourceFile, h.Name, h.Ports, h.Public)
	}
	for _, ct := range c.Containers {
		check(ct.SourceFile, ct.Name, ct.Ports, ct.Public)
	}
}

// checkPeopleReferences resolves every `people: [...]` list (on hosts,
// containers, and scripts) against every loaded person's username, which
// must be unique across all people sources for an unqualified reference
// like `people: [jdoe]` to be unambiguous.
func (c *Content) checkPeopleReferences() {
	// A `people:` entry names a whole people/*.csv file (a source), optionally
	// filtered to a subset of that file's usernames. So validation is
	// file-scoped: the file must exist, and every filter username must exist in
	// THAT file. (Usernames no longer need to be globally unique across files --
	// a ref says which file it means, so the old cross-source uniqueness rule is
	// gone.)
	sourceUsers := make(map[string]map[string]bool) // source name -> set of usernames
	for _, src := range c.People {
		users := make(map[string]bool, len(src.People))
		for _, p := range src.People {
			users[p.Username] = true
		}
		sourceUsers[src.Name] = users
	}

	resolve := func(file string, refs []PeopleRef) {
		for _, ref := range refs {
			users, ok := sourceUsers[ref.File]
			if !ok {
				c.addf(file, 0, "people file %q does not match any people/*.csv (loaded: %s)", ref.File, peopleSourceList(c))
				continue
			}
			for _, u := range ref.Filter {
				if !users[u] {
					c.addf(file, 0, "people filter %q is not a username in people/%s.csv", u, ref.File)
				}
			}
		}
	}
	for _, h := range c.Hosts {
		resolve(h.SourceFile, h.People)
	}
	for _, ct := range c.Containers {
		resolve(ct.SourceFile, ct.People)
	}
	for _, s := range c.Scripts {
		resolve(s.SourceFile, s.People)
	}
}

// checkNetworkVisibility resolves every network's `visible_from:` allowlist:
// each entry must name a defined network (the builder turns it into an ACL
// rule allowing that network to reach this one), and a network can't list
// itself (it always reaches itself; a self-entry is a mistake).
func (c *Content) checkNetworkVisibility() {
	names := make(map[string]bool, len(c.Networks))
	for _, n := range c.Networks {
		names[n.Name] = true
	}
	for _, n := range c.Networks {
		for _, from := range n.VisibleFrom {
			if from == n.Name {
				c.addf(n.SourceFile, 0, "network %q lists itself in visible_from -- a network always reaches itself", n.Name)
				continue
			}
			if !names[from] {
				c.addf(n.SourceFile, 0, "network %q visible_from references %q, which is not a defined network", n.Name, from)
			}
		}
	}
}

// peopleSourceList is a comma-joined list of loaded people source names, for a
// "file does not match" error that tells the author what IS available.
func peopleSourceList(c *Content) string {
	if len(c.People) == 0 {
		return "(no people/*.csv files)"
	}
	names := make([]string, 0, len(c.People))
	for _, s := range c.People {
		names = append(names, s.Name)
	}
	return strings.Join(names, ", ")
}

// checkScriptReferences resolves every step's `script:` action and every
// schedule entry's `script:` action against the loaded Scripts. schedule
// is its own top-level list now, not nested inside steps, so it gets its
// own loop rather than a nested lookup inside the steps one.
func (c *Content) checkScriptReferences() {
	scripts := make(map[string]bool)
	for _, s := range c.Scripts {
		scripts[s.Name] = true
	}

	checkSteps := func(file string, steps []Step) {
		for _, st := range steps {
			if name, ok := st["script"].(string); ok && !scripts[name] {
				c.addf(file, 0, "step references script %q, which does not exist", name)
			}
		}
	}
	checkSchedule := func(file string, entries []Step) {
		for _, e := range entries {
			if name, ok := e["script"].(string); ok && !scripts[name] {
				c.addf(file, 0, "scheduled entry references script %q, which does not exist", name)
			}
		}
	}
	for _, h := range c.Hosts {
		checkSteps(h.SourceFile, h.Steps)
		checkSchedule(h.SourceFile, h.Schedule)
	}
	for _, ct := range c.Containers {
		checkSteps(ct.SourceFile, ct.Steps)
		checkSchedule(ct.SourceFile, ct.Schedule)
	}
}

// checkScheduleExpressions validates every schedule entry's `when:`
// against internal/schedule's grammar -- a phrasing outside the grammar
// fails validation at commit time, naming the file, the same as any
// other content mistake this pass catches.
func (c *Content) checkScheduleExpressions() {
	check := func(file string, entries []Step) {
		for _, e := range entries {
			when, _ := e["when"].(string)
			if _, err := schedule.Parse(when); err != nil {
				c.addf(file, 0, "%s", err.Error())
			}
		}
	}
	for _, h := range c.Hosts {
		check(h.SourceFile, h.Schedule)
	}
	for _, ct := range c.Containers {
		check(ct.SourceFile, ct.Schedule)
	}
}

// checkValidatorPlatforms catches the one validator the plan calls out by
// name as platform-specific: "registry" only makes sense on Windows.
// "Wrong platform is a validation error, not a runtime surprise... before
// the build starts." This checks validate blocks authored directly on a
// step; a script's own validate block propagating to whatever host runs it
// is a many-to-many check not yet implemented here (see crossCheck's
// doc comment).
// isWindowsOS recognizes an os name as Windows for the registry-validator
// platform check. Matches both the descriptive form ("windows-server-2022")
// and the short form a builder image map often uses ("win2019", "win2022").
func isWindowsOS(os string) bool {
	lower := strings.ToLower(os)
	return strings.Contains(lower, "windows") || strings.HasPrefix(lower, "win")
}

func (c *Content) checkValidatorPlatforms() {
	check := func(file, os string, steps []Step) {
		windows := isWindowsOS(os)
		for _, st := range steps {
			validate, _ := st["validate"].([]interface{})
			for _, v := range validate {
				vm, ok := v.(Step)
				if !ok {
					continue
				}
				if _, ok := vm["registry"]; ok && !windows {
					c.addf(file, 0, "step uses the `registry` validator on a host whose os is %q -- registry is Windows-only", os)
				}
			}
		}
	}
	for _, h := range c.Hosts {
		check(h.SourceFile, h.OS, h.Steps)
	}
	// Containers don't have a meaningful Windows/Linux distinction the same
	// way (image-based, not os-based) -- skipped deliberately, not an
	// oversight.
}

// checkExtends validates every `extends:` reference: the target must be
// another object of the SAME type (a host extends a host, not a container),
// and the chain must not loop (self-reference or a longer A->B->A cycle). The
// actual field merge is done in extends.go before this runs; a bad reference
// here means the merge was skipped, so checkResolvedRequired will also flag any
// field the object was then left missing.
func (c *Content) checkExtends() {
	// Per-type name sets: hosts and containers share one namespace for
	// collisions, but extends is type-specific (a host can only extend a host).
	hostNames := make(map[string]bool, len(c.Hosts))
	for _, h := range c.Hosts {
		hostNames[h.Name] = true
	}
	containerNames := make(map[string]bool, len(c.Containers))
	for _, ct := range c.Containers {
		containerNames[ct.Name] = true
	}
	netNames := c.networkNames()
	envNames := make(map[string]bool, len(c.Environments))
	for _, e := range c.Environments {
		envNames[e.Name] = true
	}

	for _, n := range c.Networks {
		c.checkExtendsRef(n.SourceFile, "network", n.Name, n.Extends, func(s string) bool { _, ok := netNames[s]; return ok },
			func(name string) string { return c.networkExtendsTarget(name) })
	}
	for _, h := range c.Hosts {
		c.checkExtendsRef(h.SourceFile, "host", h.Name, h.Extends, func(s string) bool { return hostNames[s] },
			func(name string) string { return c.hostExtendsTarget(name) })
	}
	for _, ct := range c.Containers {
		c.checkExtendsRef(ct.SourceFile, "container", ct.Name, ct.Extends, func(s string) bool { return containerNames[s] },
			func(name string) string { return c.containerExtendsTarget(name) })
	}
	for _, e := range c.Environments {
		c.checkExtendsRef(e.SourceFile, "environment", e.Name, e.Extends, func(s string) bool { return envNames[s] },
			func(name string) string { return c.environmentExtendsTarget(name) })
	}
}

// checkExtendsRef validates one object's extends: exists (same type), isn't a
// self-reference, and isn't part of a multi-node cycle. next walks one hop
// along the chain (returns "" at the end or on a broken link).
func (c *Content) checkExtendsRef(file, kind, name, extends string, exists func(string) bool, next func(string) string) {
	if extends == "" {
		return
	}
	if extends == name {
		c.addf(file, 0, "%s %q extends itself", kind, name)
		return
	}
	if !exists(extends) {
		c.addf(file, 0, "%s %q extends %q, which is not a defined %s", kind, name, extends, kind)
		return
	}
	// Walk the chain from this node; if we return to it, it's a cycle.
	seen := map[string]bool{name: true}
	for cur := extends; cur != ""; cur = next(cur) {
		if seen[cur] {
			c.addf(file, 0, "%s %q is part of an extends cycle (via %q)", kind, name, cur)
			return
		}
		seen[cur] = true
	}
}

func (c *Content) hostExtendsTarget(name string) string {
	for _, h := range c.Hosts {
		if h.Name == name {
			return h.Extends
		}
	}
	return ""
}

func (c *Content) containerExtendsTarget(name string) string {
	for _, ct := range c.Containers {
		if ct.Name == name {
			return ct.Extends
		}
	}
	return ""
}

func (c *Content) networkExtendsTarget(name string) string {
	for _, n := range c.Networks {
		if n.Name == name {
			return n.Extends
		}
	}
	return ""
}

func (c *Content) environmentExtendsTarget(name string) string {
	for _, e := range c.Environments {
		if e.Name == name {
			return e.Extends
		}
	}
	return ""
}

// checkResolvedRequired confirms that after extends-merging, an object that
// used `extends:` to skip a schema-required field actually ended up with one
// (from its base chain). Without this, `host: {name: web, extends: base}` where
// base never supplied os would validate against the schema (which relaxes
// os/size/disk when extends is present) and then resolve to a broken, empty
// object. Only objects that used extends can reach here missing a field --
// non-extending ones are already caught by the per-document schema.
func (c *Content) checkResolvedRequired() {
	for _, h := range c.Hosts {
		if h.Extends == "" {
			continue
		}
		for _, m := range missingHostFields(h) {
			c.addf(h.SourceFile, 0, "host %q extends %q but still has no %s (its extends chain provides none)", h.Name, h.Extends, m)
		}
	}
	for _, ct := range c.Containers {
		if ct.Extends == "" {
			continue
		}
		if ct.Image == "" {
			c.addf(ct.SourceFile, 0, "container %q extends %q but still has no image (its extends chain provides none)", ct.Name, ct.Extends)
		}
		if ct.Size == "" {
			c.addf(ct.SourceFile, 0, "container %q extends %q but still has no size (its extends chain provides none)", ct.Name, ct.Extends)
		}
	}
	for _, n := range c.Networks {
		if n.Extends == "" {
			continue
		}
		if n.CIDR == "" {
			c.addf(n.SourceFile, 0, "network %q extends %q but still has no cidr (its extends chain provides none)", n.Name, n.Extends)
		}
	}
	for _, e := range c.Environments {
		if e.Extends == "" {
			continue
		}
		if e.Teams == 0 {
			c.addf(e.SourceFile, 0, "environment %q extends %q but still has no teams (its extends chain provides none)", e.Name, e.Extends)
		}
	}
}

// missingHostFields lists the schema-required host fields (os/size/disk) a host
// still lacks after extends-merging.
func missingHostFields(h Host) []string {
	var out []string
	if h.OS == "" {
		out = append(out, "os")
	}
	if h.Size == "" {
		out = append(out, "size")
	}
	if h.Disk == 0 {
		out = append(out, "disk")
	}
	return out
}

// Summary is a quick human-readable count, used by cmd/laforge's minimal
// validate command (see cmd/laforge/main.go) ahead of the real `laforge
// check`.
func (c *Content) Summary() string {
	return fmt.Sprintf(
		"%d environment(s), %d network(s), %d host(s), %d container(s), %d script(s), %d people source(s), %d error(s)",
		len(c.Environments), len(c.Networks), len(c.Hosts), len(c.Containers), len(c.Scripts), len(c.People), len(c.Errors),
	)
}
