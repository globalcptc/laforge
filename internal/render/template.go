package render

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/globalcptc/laforge/internal/loader"
)

// TemplateData is the root "." value a script or step field renders
// against. The spec's own examples use two different (both valid)
// Go template access styles and this supports both rather than picking one:
//
//   - dot-context field access: {{ .host.hostname }}, {{ .vars.company }}
//   - bare niladic-function chains: {{ vars.db_pw }}, {{ range people "x" }}
//
// The two don't collide: a leading "." resolves against this map, a bare
// identifier resolves against the FuncMap in funcMap() below -- Go's
// template grammar treats them as separate lookups.
func (ctx *Context) TemplateData() map[string]interface{} {
	vars := ctx.EffectiveVars()
	varsIface := make(map[string]interface{}, len(vars))
	for k, v := range vars {
		varsIface[k] = v
	}

	var peers []map[string]interface{}
	for _, p := range ctx.Peers {
		peers = append(peers, map[string]interface{}{
			"as": p.As, "kind": p.Kind, "address": p.Address,
		})
	}

	// dns is the full resolved record set, for a script that configures DNS on
	// the Domain Controller / Bind host (LaForge doesn't run DNS itself).
	var dns []map[string]interface{}
	for _, r := range ctx.DNS {
		dns = append(dns, map[string]interface{}{
			"name": r.Name, "type": r.Type, "value": r.Value, "priority": r.Priority,
		})
	}

	// people is THIS object's resolved `people:` set (see Context.ObjectPeople) --
	// so `{{ range .people }}{{ .username }}{{ end }}` targets exactly the people
	// assigned to this host. Each row is columns + username, same row shape the
	// bare `people "file"` function returns for a whole file.
	people := make([]map[string]interface{}, 0, len(ctx.ObjectPeople))
	for _, row := range ctx.ObjectPeople {
		m := make(map[string]interface{}, len(row))
		for k, v := range row {
			m[k] = v
		}
		people = append(people, m)
	}

	return map[string]interface{}{
		"host": map[string]interface{}{
			"hostname": ctx.As,
			"address":  ctx.Address,
			"os":       ctx.OS,
			"image":    ctx.Image,
			"size":     ctx.Size,
			"kind":     ctx.ObjectKind,
		},
		"network": map[string]interface{}{
			"name":  ctx.NetworkName,
			"cidr":  ctx.NetworkCIDR,
			"hosts": peers,
		},
		"build": map[string]interface{}{
			"environment": ctx.EnvironmentName,
			"team":        ctx.Team,
			"teams":       ctx.TeamsTotal,
			"start":       ctx.Start,
			"stop":        ctx.Stop,
		},
		"vars":   varsIface,
		"dns":    dns,
		"people": people,
	}
}

// funcMap provides the bare-identifier forms: `vars.key` (a niladic
// function whose result is then field/key-chained -- valid Go template
// grammar, verified empirically before relying on it) and
// `range people "name"`, which looks up a people source by name and
// returns its rows as plain string maps so `.username`, `.first_name`,
// etc. resolve as ordinary map keys inside the range.
func funcMap(ctx *Context, content *loader.Content) template.FuncMap {
	return template.FuncMap{
		"vars": func() map[string]string {
			return ctx.EffectiveVars()
		},
		"people": func(sourceName string) ([]map[string]string, error) {
			for _, src := range content.People {
				if src.Name != sourceName {
					continue
				}
				var rows []map[string]string
				for _, p := range src.People {
					row := make(map[string]string, len(p.Attributes)+1)
					for k, v := range p.Attributes {
						row[k] = v
					}
					row["username"] = p.Username
					rows = append(rows, row)
				}
				return rows, nil
			}
			return nil, fmt.Errorf("no people source named %q (loaded sources: %s)", sourceName, peopleSourceNames(content))
		},
	}
}

func peopleSourceNames(c *loader.Content) string {
	var names []string
	for _, s := range c.People {
		names = append(names, s.Name)
	}
	if len(names) == 0 {
		return "(none)"
	}
	out := names[0]
	for _, n := range names[1:] {
		out += ", " + n
	}
	return out
}

// RenderString renders one template string. name is used as the template's
// own name, so a failure's error message names the real source file and
// line -- "an error when the commit is validated, naming the file and
// line" -- for free from Go's own template error formatting, not
// reimplemented here.
//
// Strict mode: missingkey=error means an unknown map key (`.vars.hostnme`,
// a typo'd people column, etc.) is a hard error rather than silently
// rendering as "<no value>". This is what makes "{{ .hostnme }} is an
// error when the commit is validated" true.
func RenderString(name, text string, ctx *Context, content *loader.Content) (string, error) {
	tmpl, err := template.New(name).Option("missingkey=error").Funcs(funcMap(ctx, content)).Parse(text)
	if err != nil {
		return "", fmt.Errorf("%s: %w", name, err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, ctx.TemplateData()); err != nil {
		return "", fmt.Errorf("%s: %w", name, err)
	}
	return buf.String(), nil
}
