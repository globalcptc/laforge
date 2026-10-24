package hclconvert

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/globalcptc/laforge/internal/loader"
)

// oldTemplateSyntaxRe matches the old agent's template context, which used
// a capitalized, dot-chained shape ({{ $.Host.OverridePassword }},
// {{ $.Environment.Name }}, {{ index $.Network.Vars "..." }}, a
// {{ $.Identities }} range, the old-only .HCLID field, and a custom `Base`
// function) that has no equivalent in the new lowercase context
// (vars.x, .host.x, ...) the render engine built. "Scripts carry over untouched"
// is deliberate, so this never rewrites a script -- it only flags one, per
// "flags anything it cannot translate rather than guessing." A real script
// that happens to run against every environment in this repo would have
// already been caught by `laforge check`'s render pass; this catches the
// rest too, since most of the 472 scripts here aren't wired into either of
// the two environments that exist today and so are otherwise unchecked.
var oldTemplateSyntaxRe = regexp.MustCompile(`\$\.[A-Z][A-Za-z]*|\$[A-Za-z_][A-Za-z0-9_]*\.HCLID\b|\bBase\s*\(|\bBase\s+\$`)

// Write serializes a Result to a real content repository directory tree,
// matching internal/loader's expected layout exactly (see
// "How files are identified"). Script source files
// are copied byte-for-byte from the original repo -- "scripts carry over
// untouched."
func Write(r *Result, outDir string) error {
	for _, dir := range []string{"networks", "hosts", "scripts", "people"} {
		if err := os.MkdirAll(filepath.Join(outDir, dir), 0o755); err != nil {
			return err
		}
	}

	for _, n := range r.Networks {
		if err := writeYAML(filepath.Join(outDir, "networks", n.Name+".yaml"), map[string]loader.Network{"network": n}); err != nil {
			return err
		}
	}
	for _, h := range r.Hosts {
		if err := writeYAML(filepath.Join(outDir, "hosts", h.Name+".yaml"), map[string]loader.Host{"host": h}); err != nil {
			return err
		}
	}
	for _, s := range r.Scripts {
		clean := strings.TrimPrefix(s.Source, "./")
		s.Source = filepath.Base(clean)
		if err := writeYAML(filepath.Join(outDir, "scripts", s.Name+".yaml"), map[string]loader.Script{"script": s}); err != nil {
			return err
		}
		if src, ok := r.scriptSourceOnDisk[s.Name]; ok {
			content, err := os.ReadFile(src)
			if err != nil {
				r.note(s.Name, "warn", fmt.Sprintf("could not copy real source file %s: %v", src, err))
			} else {
				if err := os.WriteFile(filepath.Join(outDir, "scripts", s.Source), content, 0o644); err != nil {
					return err
				}
				if m := oldTemplateSyntaxRe.FindString(string(content)); m != "" {
					r.note(s.Name, "warn", fmt.Sprintf("script body still uses old-model template syntax (e.g. %q) -- carried over untouched by design, but it needs porting to the new context (vars.x, .host.x, ...) before it will render; many old scripts are single-year or reference-only and may never need this", m))
				}
			}
		}
	}
	for _, e := range r.Environments {
		if err := writeYAML(filepath.Join(outDir, e.Name+".yaml"), map[string]loader.Environment{"environment": e}); err != nil {
			return err
		}
	}
	for group, people := range r.People {
		if err := writePeopleCSV(filepath.Join(outDir, "people", group+".csv"), people); err != nil {
			return err
		}
	}
	return nil
}

// writeYAML's callers wrap v in a one-key map (`map[string]loader.Host{"host": h}`,
// etc.) before calling this -- every field a content object has lives
// inside its own type key's object now (`host:\n  name: ...\n  os: ...`,
// not `host: <name>` with os/size/... as top-level siblings, see
// host.schema.json's own $comment), so marshaling h directly would put
// its fields at the wrong level. loader.Host/Network/Script/Environment's
// own yaml tags (name/os/size/...) are unchanged by this -- only where
// this function nests them.
func writeYAML(path string, v interface{}) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := yaml.NewEncoder(f)
	enc.SetIndent(2)
	defer enc.Close()
	return enc.Encode(v)
}

// writePeopleCSV rebuilds a CSV from converted rows. Column order is
// deterministic (username first, then every attribute key seen across the
// group, alphabetically) so repeated conversion runs produce a stable diff.
func writePeopleCSV(path string, people []loader.Person) error {
	colSet := make(map[string]bool)
	for _, p := range people {
		for k := range p.Attributes {
			colSet[k] = true
		}
	}
	var cols []string
	for k := range colSet {
		cols = append(cols, k)
	}
	sort.Strings(cols)

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := csv.NewWriter(f)
	defer w.Flush()

	header := append([]string{"username"}, cols...)
	if err := w.Write(header); err != nil {
		return err
	}
	sort.Slice(people, func(i, j int) bool { return people[i].Username < people[j].Username })
	for _, p := range people {
		row := make([]string, len(header))
		row[0] = p.Username
		for i, c := range cols {
			row[i+1] = p.Attributes[c]
		}
		if err := w.Write(row); err != nil {
			return err
		}
	}
	return nil
}
