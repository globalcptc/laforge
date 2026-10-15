package loader

import "sort"

// Dependents returns every host/container object name that transitively depends
// on any of roots (via depends_on), INCLUDING the roots themselves. It is the
// blast radius of rebuilding those objects: because depends_on points from a
// dependent to its prerequisite ("A depends_on B" = A needs B up first),
// everything built on top of a rebuilt object must be rebuilt too, or it would
// be left running against infrastructure that was torn down and recreated
// underneath it.
//
// A step-level wait_for counts as an edge too, unless it sets `rebuild: false`
// -- the author's way of saying the waiting host stays correct when its target
// is recreated underneath it.
//
// Edges are taken only among host/container objects -- the same namespace
// checkDependsOnCycles walks -- since networks have no outgoing depends_on and
// are foundational (a network rebuild is its own, separate concern). The
// returned names are content object names; scoping the result to one team's
// deployed_object rows is the caller's job (depends_on is declared at the
// object level, and every team is an identical copy).
func (c *Content) Dependents(roots []string) []string {
	return c.dependents(roots, true)
}

// dependents is Dependents' walk. rebuildOnly drops the wait_for edges that set
// `rebuild: false` -- the rebuild cascade's view; DependentCounts keeps them.
func (c *Content) dependents(roots []string, rebuildOnly bool) []string {
	objects := c.objectNames()

	// Reverse adjacency: reverse[d] lists the objects that declare depends_on d,
	// so a walk from a root reaches everything that (transitively) depends on it.
	reverse := make(map[string][]string)
	addEdges := func(name string, dependsOn []string) {
		for _, d := range dependsOn {
			if _, ok := objects[d]; ok { // object dep only (skip networks/unknowns)
				reverse[d] = append(reverse[d], name)
			}
		}
	}
	for _, h := range c.Hosts {
		addEdges(h.Name, h.DependsOn)
		addEdges(h.Name, waitTargets(h.Name, h.Steps, rebuildOnly))
	}
	for _, ct := range c.Containers {
		addEdges(ct.Name, ct.DependsOn)
		addEdges(ct.Name, waitTargets(ct.Name, ct.Steps, rebuildOnly))
	}

	seen := make(map[string]bool)
	var stack []string
	for _, r := range roots {
		if !seen[r] { // roots are always included, even one not present in content
			seen[r] = true
			stack = append(stack, r)
		}
	}
	for len(stack) > 0 {
		n := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		for _, dependent := range reverse[n] {
			if !seen[dependent] {
				seen[dependent] = true
				stack = append(stack, dependent)
			}
		}
	}

	out := make([]string, 0, len(seen))
	for n := range seen {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

// DependencyDepth maps each host/container object name to its depth in the
// depends_on graph: 0 for an object with no object-level dependencies (a
// root), otherwise 1 + the greatest depth among its dependencies. It is used
// to order deploys roots-first, so that a dependency's box (and then its
// configuration) gets underway before the dependents that wait on it --
// "prioritize image deployments based on the tree." A step-level wait_for
// counts the same as depends_on here (whatever its `rebuild:`), so a host that
// others wait on deploys ahead of them. Only object edges count
// (networks have no depends_on and are foundational, exactly as Dependents and
// checkDependsOnCycles treat them); a dependency naming something not placed
// as an object resolves to depth 0. checkDependsOnCycles guarantees a DAG, so
// the memoized recursion always terminates; a cycle that somehow slipped
// through is broken defensively by treating an in-progress node as depth 0.
func (c *Content) DependencyDepth() map[string]int {
	objects := c.objectNames()
	dependsOn := make(map[string][]string, len(objects))
	record := func(name string, deps []string) {
		var kept []string
		for _, d := range deps {
			if _, ok := objects[d]; ok { // object dep only (skip networks/unknowns)
				kept = append(kept, d)
			}
		}
		dependsOn[name] = kept
	}
	for _, h := range c.Hosts {
		record(h.Name, append(append([]string{}, h.DependsOn...), waitTargets(h.Name, h.Steps, false)...))
	}
	for _, ct := range c.Containers {
		record(ct.Name, append(append([]string{}, ct.DependsOn...), waitTargets(ct.Name, ct.Steps, false)...))
	}

	depth := make(map[string]int, len(objects))
	inProgress := make(map[string]bool, len(objects))
	var compute func(name string) int
	compute = func(name string) int {
		if d, ok := depth[name]; ok {
			return d
		}
		if inProgress[name] {
			return 0 // defensive: a cycle the loader should already have rejected
		}
		inProgress[name] = true
		max := 0
		for _, d := range dependsOn[name] {
			if dd := compute(d) + 1; dd > max {
				max = dd
			}
		}
		inProgress[name] = false
		depth[name] = max
		return max
	}
	for name := range objects {
		compute(name)
	}
	return depth
}

// DependentCounts maps each host/container object name to how many OTHER
// objects transitively depend on it -- its weight on the critical path. A
// domain controller every Windows host depends_on scores high; a leaf
// workstation scores 0. Deploys are prioritized by it (see
// internal/orchestrator's deploy task payload): DependencyDepth alone ranks
// every object with no dependencies of its own as an equal "root", so a
// standalone workstation tied with the DC everything else waits on. Every
// step-level wait_for counts here, `rebuild: false` included: a host that
// others wait on is on their critical path whether or not a rebuild of it
// cascades to them.
func (c *Content) DependentCounts() map[string]int {
	objects := c.objectNames()
	out := make(map[string]int, len(objects))
	for name := range objects {
		out[name] = len(c.dependents([]string{name}, false)) - 1 // Dependents includes the root itself
	}
	return out
}

// waitTargets lists the distinct hosts/containers that self's steps wait on
// through `wait_for:`, in first-seen order, never self (a self-wait is its own
// content error, not a cycle). rebuildOnly keeps only the waits that join the
// target's rebuild cascade (`rebuild:` unset or true).
func waitTargets(self string, steps []Step, rebuildOnly bool) []string {
	var out []string
	seen := map[string]bool{}
	for _, st := range steps {
		for _, w := range st.WaitFor() {
			if w.Host == self || seen[w.Host] || (rebuildOnly && !w.RebuildsWithTarget()) {
				continue
			}
			seen[w.Host] = true
			out = append(out, w.Host)
		}
	}
	return out
}
