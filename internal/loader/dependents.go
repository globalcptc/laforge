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
// Edges are taken only among host/container objects -- the same namespace
// checkDependsOnCycles walks -- since networks have no outgoing depends_on and
// are foundational (a network rebuild is its own, separate concern). The
// returned names are content object names; scoping the result to one team's
// deployed_object rows is the caller's job (depends_on is declared at the
// object level, and every team is an identical copy).
func (c *Content) Dependents(roots []string) []string {
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
	}
	for _, ct := range c.Containers {
		addEdges(ct.Name, ct.DependsOn)
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
