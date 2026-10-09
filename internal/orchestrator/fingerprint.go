package orchestrator

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/globalcptc/laforge/internal/compose"
	"github.com/globalcptc/laforge/internal/loader"
	"github.com/globalcptc/laforge/internal/render"
)

// Fingerprint is "the rendered output is the fingerprint":
// "Everything that produces a host is
// hashed together... If a change would alter a single byte the agent
// receives, the host is rebuilt; if it would not, it cannot matter to the
// host." A mismatch between a freshly computed Fingerprint and a
// deployed_object's stored one is what Reconcile treats as "changed,"
// triggering destroy-then-redeploy.
//
// What's hashed, for one specific team+copy (not just the bare
// definition, since two copies of the same host can resolve to different
// addresses and therefore different rendered script output):
//
//   - The definition's own non-templated fields: os/image, size, disk,
//     ports, depends_on -- "hostname, os or image, size, disk, ports,
//     [or] address" per the plan's own table of what counts as changed.
//   - The raw (unrendered) step list as JSON -- catches a step being
//     added, removed, reordered, or having any of its own fields edited,
//     including inline `run:`/`write_file:` template text.
//   - Every var visible to this copy, key and resolved value -- "vars the
//     host uses" -- which is what makes an inline step referencing
//     `{{ vars.x }}` correctly rebuild when x's *value* changes even
//     though the step's own unrendered text didn't.
//   - The actual rendered content of every script a step references --
//     "script contents" -- so editing a script body, or anything that
//     body's own template references, rebuilds every host that runs it.
//
// Deliberately not modeled yet: a host/container's own
// `schedule:` list (its own top-level field now, not nested inside
// steps -- see loader.Host.Schedule's own comment) and a script's
// `validate:` block (findings/tags never reach a script either way,
// matching "tags and findings never reach a script, so they never cause
// work" -- they're correctly excluded, not missed).
func Fingerprint(repoRoot string, c *loader.Content, ctx *render.Context, disk int, ports loader.Ports, dependsOn []string) (string, error) {
	h := sha256.New()

	fmt.Fprintf(h, "os=%s\nimage=%s\nsize=%s\ndisk=%d\n", ctx.OS, ctx.Image, ctx.Size, disk)

	// A compose container is its project: editing the compose file, or any file
	// shipped with it, rebuilds it.
	if ctx.Compose != "" {
		b, err := compose.Load(repoRoot, ctx.Compose, nil)
		if err != nil {
			return "", fmt.Errorf("loading compose project for fingerprint: %w", err)
		}
		fmt.Fprintf(h, "compose=%s:%x\n", ctx.Compose, sha256.Sum256(b.Archive))
	}

	tcp := append([]string(nil), ports.TCP...)
	udp := append([]string(nil), ports.UDP...)
	sort.Strings(tcp)
	sort.Strings(udp)
	fmt.Fprintf(h, "tcp=%v\nudp=%v\n", tcp, udp)

	deps := append([]string(nil), dependsOn...)
	sort.Strings(deps)
	fmt.Fprintf(h, "depends_on=%v\n", deps)

	stepsJSON, err := json.Marshal(ctx.Steps)
	if err != nil {
		return "", fmt.Errorf("marshaling steps: %w", err)
	}
	h.Write(stepsJSON)
	h.Write([]byte{'\n'})

	varKeys := make([]string, 0, len(ctx.Vars))
	varVals := make(map[string]string, len(ctx.Vars))
	for _, v := range ctx.Vars {
		varKeys = append(varKeys, v.Key)
		varVals[v.Key] = v.Value
	}
	sort.Strings(varKeys)
	for _, k := range varKeys {
		fmt.Fprintf(h, "var:%s=%s\n", k, varVals[k])
	}

	for _, step := range ctx.Steps {
		scriptName, ok := step["script"].(string)
		if !ok {
			continue
		}
		script := findScript(c, scriptName)
		if script == nil {
			// An unresolvable script reference is a validation error
			// caught elsewhere (internal/render.CheckAll); Reconcile never
			// runs against content that failed that check, so this path
			// only matters for a caller using Fingerprint standalone.
			fmt.Fprintf(h, "script:%s=<unresolved>\n", scriptName)
			continue
		}
		rendered, err := render.RenderScript(repoRoot, script, ctx, c)
		if err != nil {
			return "", fmt.Errorf("rendering script %q for fingerprint: %w", scriptName, err)
		}
		fmt.Fprintf(h, "script:%s:\n%s\n", scriptName, rendered)
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// networkFingerprint is Fingerprint's much simpler counterpart for a
// network itself: networks never run scripts or resolve per-team (every
// team's copy of a network is byte-identical by design -- "every team
// gets exactly the same network"), so there's nothing to render, only the
// network's own definition to hash.
func networkFingerprint(n *loader.Network) string {
	h := sha256.New()
	fmt.Fprintf(h, "cidr=%s\nvisible_from=%v\n", n.CIDR, n.VisibleFrom)
	varKeys := make([]string, 0, len(n.Vars))
	for k := range n.Vars {
		varKeys = append(varKeys, k)
	}
	sort.Strings(varKeys)
	for _, k := range varKeys {
		fmt.Fprintf(h, "var:%s=%s\n", k, n.Vars[k])
	}
	return hex.EncodeToString(h.Sum(nil))
}

func findScript(c *loader.Content, name string) *loader.Script {
	for i := range c.Scripts {
		if c.Scripts[i].Name == name {
			return &c.Scripts[i]
		}
	}
	return nil
}
