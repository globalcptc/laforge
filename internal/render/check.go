package render

import (
	"fmt"
	"github.com/globalcptc/laforge/internal/compose"
	"sort"

	"github.com/globalcptc/laforge/internal/loader"
)

// RenderError is one rendering failure, kept separate from
// schema.FieldError (which is about the config itself, not what it renders
// to) but shaped the same way for consistent CLI output.
type RenderError struct {
	Environment string
	Team        int
	As          string
	Message     string
}

// CheckAll renders every script against every host copy, in every team, in
// every environment, exactly as `laforge check` is described:
// "renders everything... alongside the renders it checks
// schema conformance, name collisions, depends_on targets that exist,
// public ports being a subset of the host's ports, and validators asking
// for the wrong platform." The loader (see internal/loader) already
// produces that second half; CheckAll adds the rendering pass on top and
// reuses the same Content rather than reloading.
func CheckAll(repoRoot string, c *loader.Content) []RenderError {
	var errs []RenderError

	envNames := make([]string, len(c.Environments))
	for i, e := range c.Environments {
		envNames[i] = e.Name
	}
	sort.Strings(envNames)

	for _, envName := range envNames {
		env := findEnvironment(c, envName)
		for team := 1; team <= env.Teams; team++ {
			for _, as := range placementNames(env) {
				ctx, err := Resolve(c, envName, as, team)
				if err != nil {
					errs = append(errs, RenderError{Environment: envName, Team: team, As: as, Message: err.Error()})
					continue
				}
				errs = append(errs, checkOneHost(repoRoot, c, ctx)...)
			}
		}
	}
	return errs
}

func checkOneHost(repoRoot string, c *loader.Content, ctx *Context) []RenderError {
	var errs []RenderError
	add := func(err error) {
		if err != nil {
			errs = append(errs, RenderError{Environment: ctx.EnvironmentName, Team: ctx.Team, As: ctx.As, Message: err.Error()})
		}
	}

	// Load a compose container's project exactly as deploying it would, so a
	// missing file, a service with no image, or a bind mount that won't be
	// shipped is caught here, not on the day.
	if ctx.Compose != "" {
		_, err := compose.Load(repoRoot, ctx.Compose, nil)
		add(err)
	}

	for _, step := range ctx.Steps {
		for _, err := range RenderStepFields(ctx.As, step, ctx, c) {
			add(err)
		}
		if scriptName, ok := step["script"].(string); ok {
			checkScriptRun(repoRoot, c, ctx, scriptName, add)
		}
	}
	// Schedule is a separate list from Steps now, not nested inside it
	// (see loader.Host.Schedule's own comment) -- same template/script
	// checks apply, since a schedule entry is the same action shape.
	for _, entry := range ctx.Schedule {
		for _, err := range RenderStepFields(ctx.As, entry, ctx, c) {
			add(err)
		}
		if scriptName, ok := entry["script"].(string); ok {
			checkScriptRun(repoRoot, c, ctx, scriptName, add)
		}
	}
	return errs
}

func checkScriptRun(repoRoot string, c *loader.Content, ctx *Context, scriptName string, add func(error)) {
	script := findScript(c, scriptName)
	if script == nil {
		// Already caught by loader.checkScriptReferences; don't double
		// report if it happens to slip through here on partially-invalid
		// content used outside a real check (see Resolve's own comment).
		return
	}
	if _, err := RenderScript(repoRoot, script, ctx, c); err != nil {
		add(err)
	}
	for _, v := range script.Validate {
		for _, err := range RenderStepFields(ctx.As, loader.Step(v), ctx, c) {
			add(err)
		}
	}
}

// placementNames returns every `as` name in an environment's topology, in
// deterministic order.
func placementNames(env *loader.Environment) []string {
	var names []string
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
				names = append(names, cp.As)
			}
		}
	}
	return names
}

func (e RenderError) String() string {
	return fmt.Sprintf("%s team=%d host=%s: %s", e.Environment, e.Team, e.As, e.Message)
}
