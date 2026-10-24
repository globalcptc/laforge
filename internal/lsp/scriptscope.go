package lsp

import (
	"path/filepath"
	"sort"

	"github.com/globalcptc/laforge/internal/loader"
	"github.com/globalcptc/laforge/internal/render"
)

// scriptForSourceFile finds the loader.Script whose real source file (the
// .sh/.ps1/.bat next to its .yaml definition) is rel, for completion in a
// script's own source file -- editing base.sh needs to know it's the
// script named "base" to find every host that runs it.
func scriptForSourceFile(c *loader.Content, rel string) *loader.Script {
	for i := range c.Scripts {
		s := &c.Scripts[i]
		if filepath.ToSlash(filepath.Join(filepath.Dir(s.SourceFile), s.Source)) == filepath.ToSlash(rel) {
			return s
		}
	}
	return nil
}

// varUsage is one var's real usage across every context a script runs
// in -- "Completion offers the union of every context the script runs
// in, marking anything that is not available everywhere: `db_name` shows
// as available on 2 of the 5 hosts that run this script".
type varUsage struct {
	Key          string
	ExampleValue string
	Source       string // "environment" | "network" | "host/container", from whichever context set it first
	// SourceFile is that same first context's real file -- "Hover: what
	// a var holds and which file it came from in the environment/
	// network/host cascade", resolved from
	// Source plus that context's own environment/network/object name.
	SourceFile string
	SeenIn     int
	Total      int
}

// scriptScope resolves every real (environment, team, as) triple that
// runs scriptName -- walking the environment topology and every host/
// container's own Steps, matching Reconcile's own traversal
// (internal/orchestrator/reconcile.go) -- and unions each one's real
// render.Context. Returns the var usage union plus the fields that are
// unconditionally present in every context (host/network/build), and
// every known people source, all real completion candidates for a
// script file's template expressions.
func scriptScope(c *loader.Content, scriptName string) (vars []varUsage, peopleSources []loader.PeopleSource) {
	containerNames := make(map[string]bool, len(c.Containers))
	for _, ct := range c.Containers {
		containerNames[ct.Name] = true
	}

	type triple struct {
		env  string
		team int
		as   string
	}
	var triples []triple

	for _, env := range c.Environments {
		for _, objs := range env.Networks {
			for objectName, copies := range objs {
				steps := stepsForObject(c, objectName, containerNames[objectName])
				if !stepsReferenceScript(steps, scriptName) {
					continue
				}
				for _, cp := range copies {
					for team := 1; team <= env.Teams; team++ {
						triples = append(triples, triple{env: env.Name, team: team, as: cp.As})
					}
				}
			}
		}
	}

	seen := make(map[string]*varUsage)
	var order []string
	for _, tr := range triples {
		ctx, err := render.Resolve(c, tr.env, tr.as, tr.team)
		if err != nil {
			continue // partially-invalid content mid-edit -- skip, don't fail the whole completion
		}
		for _, v := range ctx.Vars {
			u, ok := seen[v.Key]
			if !ok {
				u = &varUsage{Key: v.Key, ExampleValue: v.Value, Source: v.Source, SourceFile: sourceFileFor(c, ctx, v.Source)}
				seen[v.Key] = u
				order = append(order, v.Key)
			}
			u.SeenIn++
		}
	}
	sort.Strings(order)
	for _, k := range order {
		u := seen[k]
		u.Total = len(triples)
		vars = append(vars, *u)
	}

	return vars, c.People
}

// sourceFileFor resolves a var's cascade level ("environment" | "network"
// | "host/container", exactly as render.mergeVars sets VarEntry.Source)
// against ctx's own environment/network/object name to find the real
// file that var was written in.
func sourceFileFor(c *loader.Content, ctx *render.Context, level string) string {
	switch level {
	case "environment":
		for i := range c.Environments {
			if c.Environments[i].Name == ctx.EnvironmentName {
				return c.Environments[i].SourceFile
			}
		}
	case "network":
		for i := range c.Networks {
			if c.Networks[i].Name == ctx.NetworkName {
				return c.Networks[i].SourceFile
			}
		}
	case "host/container":
		for i := range c.Hosts {
			if c.Hosts[i].Name == ctx.ObjectName {
				return c.Hosts[i].SourceFile
			}
		}
		for i := range c.Containers {
			if c.Containers[i].Name == ctx.ObjectName {
				return c.Containers[i].SourceFile
			}
		}
	}
	return ""
}

// stepsForObject returns the object's Steps AND its Schedule, combined --
// schedule is a separate top-level list now (see loader.Host.Schedule's
// own comment), not nested inside steps, but both are the same Step
// shape and both can reference a script by name, which is all
// stepsReferenceScript below actually checks.
func stepsForObject(c *loader.Content, objectName string, isContainer bool) []loader.Step {
	if isContainer {
		for i := range c.Containers {
			if c.Containers[i].Name == objectName {
				return append(append([]loader.Step{}, c.Containers[i].Steps...), c.Containers[i].Schedule...)
			}
		}
		return nil
	}
	for i := range c.Hosts {
		if c.Hosts[i].Name == objectName {
			return append(append([]loader.Step{}, c.Hosts[i].Steps...), c.Hosts[i].Schedule...)
		}
	}
	return nil
}

func stepsReferenceScript(steps []loader.Step, scriptName string) bool {
	for _, step := range steps {
		if name, ok := step["script"].(string); ok && name == scriptName {
			return true
		}
	}
	return false
}
