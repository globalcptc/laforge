package lsp

import (
	"testing"

	"github.com/globalcptc/laforge/internal/schema"
)

func TestSchemaFieldsForKindHost(t *testing.T) {
	fields, err := schemaFieldsForKind(schema.KindHost)
	if err != nil {
		t.Fatal(err)
	}
	byName := make(map[string]schemaField, len(fields))
	for _, f := range fields {
		byName[f.Name] = f
	}
	for _, want := range []string{"name", "os", "size", "disk", "ports", "steps", "vars", "tags", "findings", "depends_on"} {
		if _, ok := byName[want]; !ok {
			t.Errorf("schema fields for host missing %q; got %+v", want, fields)
		}
	}
	if byName["disk"].Type != "integer" {
		t.Errorf("disk.Type = %q, want integer", byName["disk"].Type)
	}
}

// stepActionOptionNames is every real step/schedule action kind, shared
// by the two tests below since steps and schedule entries offer the
// same 14 actions (schedule itself excluded -- scheduling a schedule
// makes no sense, which is exactly why it isn't a step kind either
// anymore; see internal/schedule for where scheduling actually moved).
var stepActionOptionNames = []string{
	"script", "run", "download", "upload", "write_file", "append_file",
	"extract", "delete", "change_perms", "create_user", "set_password",
	"add_to_group", "service", "reboot",
}

// TestSchemaFieldsForKindHostStepsListsEveryActionKind is a real ask:
// "add to the host field reference a full listing of every potential
// step type," each with its own description and example, not just a
// bare name -- steps is an array of objects, not a string, so a real
// JSON Schema "enum" can't express this at all (it would make the
// compiled validator reject every real steps list). attachActionOptions
// fills schemaField.Options from the same per-action data
// stepListItemCompletion's own snippets already come from
// (common.schema.json's stepItem properties) -- one source, not a
// second copy. validate is deliberately excluded: a modifier a step
// carries alongside a real action, never a step kind on its own. schedule
// no longer belongs here at all -- it's its own top-level field now
// (see TestSchemaFieldsForKindHostScheduleListsEveryActionKind).
func TestSchemaFieldsForKindHostStepsListsEveryActionKind(t *testing.T) {
	fields, err := schemaFieldsForKind(schema.KindHost)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range fields {
		if f.Name != "steps" {
			continue
		}
		byName := make(map[string]schemaFieldOption, len(f.Options))
		for _, o := range f.Options {
			byName[o.Name] = o
		}
		for _, w := range stepActionOptionNames {
			opt, ok := byName[w]
			if !ok {
				t.Errorf("steps options missing %q; got %v", w, f.Options)
				continue
			}
			if opt.Description == "" {
				t.Errorf("steps option %q has no description", w)
			}
			if opt.Example == "" {
				t.Errorf("steps option %q has no example", w)
			}
		}
		if _, ok := byName["validate"]; ok {
			t.Errorf("steps options should exclude \"validate\" (a modifier, not a step kind); got %v", f.Options)
		}
		if _, ok := byName["schedule"]; ok {
			t.Errorf("steps options should no longer include \"schedule\" -- it's a top-level field now, not a step kind; got %v", f.Options)
		}
		return
	}
	t.Fatal("no \"steps\" field found in host schema")
}

// TestSchemaFieldsForKindHostScheduleListsEveryActionKind is the new
// top-level `schedule:` field's own version of the test above -- same
// 14 actions, each with a real description and example, but excluding
// both `validate` (a modifier) and `when` (metadata, not an action kind)
// from the options list shown in the field reference panel.
func TestSchemaFieldsForKindHostScheduleListsEveryActionKind(t *testing.T) {
	fields, err := schemaFieldsForKind(schema.KindHost)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range fields {
		if f.Name != "schedule" {
			continue
		}
		byName := make(map[string]schemaFieldOption, len(f.Options))
		for _, o := range f.Options {
			byName[o.Name] = o
		}
		for _, w := range stepActionOptionNames {
			opt, ok := byName[w]
			if !ok {
				t.Errorf("schedule options missing %q; got %v", w, f.Options)
				continue
			}
			if opt.Description == "" {
				t.Errorf("schedule option %q has no description", w)
			}
			if opt.Example == "" {
				t.Errorf("schedule option %q has no example", w)
			}
		}
		if _, ok := byName["validate"]; ok {
			t.Errorf("schedule options should exclude \"validate\"; got %v", f.Options)
		}
		if _, ok := byName["when"]; ok {
			t.Errorf("schedule options should exclude \"when\" (metadata, not an action kind); got %v", f.Options)
		}
		return
	}
	t.Fatal("no \"schedule\" field found in host schema")
}

func TestSchemaFieldsForKindScriptHasLanguageEnum(t *testing.T) {
	fields, err := schemaFieldsForKind(schema.KindScript)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range fields {
		if f.Name == "language" {
			want := map[string]bool{"bash": true, "powershell": true, "batch": true}
			if len(f.Enum) != len(want) {
				t.Fatalf("language enum = %v, want exactly %v", f.Enum, want)
			}
			for _, e := range f.Enum {
				if !want[e] {
					t.Fatalf("language enum contains unexpected value %q", e)
				}
			}
			return
		}
	}
	t.Fatal("no \"language\" field found in script schema")
}

func TestHeaderKind(t *testing.T) {
	cases := []struct {
		text string
		want schema.Kind
		ok   bool
	}{
		{"host:\n  name: webserver\n  os: ubuntu22\n", schema.KindHost, true},
		{"# a comment first\nenvironment:\n  name: lm-test\n", schema.KindEnvironment, true},
		{"script:\n  name: base\n  language: bash\n", schema.KindScript, true},
		{"not_a_header_at_all\n", "", false},
		{"", "", false},
	}
	for _, c := range cases {
		got, ok := headerKind(c.text)
		if ok != c.ok || (ok && got != c.want) {
			t.Errorf("headerKind(%q) = (%q, %v), want (%q, %v)", c.text, got, ok, c.want, c.ok)
		}
	}
}
