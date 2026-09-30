package lsp

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/globalcptc/laforge/internal/schema"
)

// schemaField is one property of a content type's own JSON Schema,
// exactly what "Completion in config files: driven by the JSON Schema,
// so every field and enum is offered with its documentation"
// needs. Description is often empty -- the
// schemas under internal/schema/schemas don't widely use JSON Schema's
// own "description" keyword yet;
// completion still offers the real field name, type, and enum values
// either way.
type schemaField struct {
	Name        string              `json:"name"`
	Type        string              `json:"type"`
	Enum        []string            `json:"enum,omitempty"`
	Description string              `json:"description,omitempty"`
	Example     string              `json:"example,omitempty"`
	Options     []schemaFieldOption `json:"options,omitempty"`
}

// schemaFieldOption is one member of a field whose real values are each
// their own small shape with their own meaning -- steps' 15 action
// kinds, concretely -- so a bare name (what a real JSON Schema "enum"
// would give, and can't even be used here: steps' actual value is an
// array of objects, not a string) isn't enough to explain what each one
// actually is. Populated from the same per-action description/example
// already on common.schema.json's stepItem.properties -- one source of
// truth, not a second copy of the same text.
type schemaFieldOption struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Example     string `json:"example,omitempty"`
}

// propDoc is one property's own raw JSON Schema shape, shared by every
// object-schema decode in this file (a full type's own schema, and the
// smaller inline object schemas nested completion resolves -- ports,
// and each step's own action object). "example" isn't a JSON Schema
// keyword with defined semantics (the real "examples" keyword holds
// instance data, not prose) -- it's this project's own convention for a
// short, real YAML snippet worth showing in the field reference panel,
// added deliberately only where a bare field/type/enum listing is hard
// to picture (steps, findings, tags, vars).
type propDoc struct {
	Type        json.RawMessage    `json:"type"`
	Description string             `json:"description"`
	Example     string             `json:"example"`
	Enum        []string           `json:"enum"`
	Properties  map[string]propDoc `json:"properties"`
}

// fieldsFromProperties turns a decoded properties map into []schemaField,
// sorted by name -- the one place that shape conversion happens, so
// schemaFieldsForKind and every nested resolver below agree on it.
func fieldsFromProperties(properties map[string]propDoc) []schemaField {
	fields := make([]schemaField, 0, len(properties))
	for name, p := range properties {
		var typ string
		json.Unmarshal(p.Type, &typ) // a $ref'd property has no inline "type"; typ stays "" and that's fine, still a real field name
		fields = append(fields, schemaField{Name: name, Type: typ, Enum: p.Enum, Description: p.Description, Example: p.Example})
	}
	sort.Slice(fields, func(i, j int) bool { return fields[i].Name < fields[j].Name })
	return fields
}

// kindDescription is the type's own description -- "A host: a full VM
// the builder deploys..." -- not any one field's. Lives on the same
// nested object schemaFieldsForKind reads fields from (see that
// function's own doc comment for why the fields sit one level down),
// so this is a second, small parse of the same raw schema rather than
// a change to schemaFieldsForKind's signature, which completion.go and
// hover.go both call and have no use for this.
func kindDescription(kind schema.Kind) (string, error) {
	raw, err := schema.RawSchema(kind)
	if err != nil {
		return "", err
	}
	var doc struct {
		Properties map[string]propDoc `json:"properties"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return "", err
	}
	return doc.Properties[string(kind)].Description, nil
}

// schemaFieldsForKind parses kind's own raw embedded schema (not the
// compiled jsonschema.Schema internal/schema.Compile produces --
// completion wants the document's own shape, not a validator) and
// returns its real fields, sorted by name. Every field lives one level
// down from the document's own top level now -- the document's only
// real top-level property is the type key itself (`host`, `network`,
// ...), whose own nested object is where `name`, `os`, `size`, and
// everything else actually live (see host.schema.json's own $comment).
func schemaFieldsForKind(kind schema.Kind) ([]schemaField, error) {
	raw, err := schema.RawSchema(kind)
	if err != nil {
		return nil, err
	}
	var doc struct {
		Properties map[string]propDoc `json:"properties"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}
	inner, ok := doc.Properties[string(kind)]
	if !ok {
		return nil, nil
	}
	fields := fieldsFromProperties(inner.Properties)
	if err := attachActionOptions(fields); err != nil {
		return nil, err
	}
	return fields, nil
}

// attachActionOptions fills in the "steps" and "schedule" fields'
// Options in place, when this kind has either, with every real action's
// own description and example -- "a full listing of every potential
// step type" (a real ask, and the same real ask for schedule entries)
// made of the same per-action data stepListItemCompletion/
// scheduleListItemCompletion's own snippets already come from
// (common.schema.json's stepItem/scheduleItem properties), not a
// second, separately-maintained copy. validate and when are excluded:
// modifiers/metadata a step or schedule entry carries, never an action
// kind on their own.
func attachActionOptions(fields []schemaField) error {
	for i := range fields {
		var actions []schemaField
		var err error
		switch fields[i].Name {
		case "steps":
			actions, err = stepActionKeys()
		case "schedule":
			actions, err = scheduleActionKeys()
		default:
			continue
		}
		if err != nil {
			return err
		}
		opts := make([]schemaFieldOption, 0, len(actions))
		for _, a := range actions {
			if a.Name == "validate" || a.Name == "when" {
				continue
			}
			opts = append(opts, schemaFieldOption{Name: a.Name, Description: a.Description, Example: a.Example})
		}
		fields[i].Options = opts
	}
	return nil
}

// commonDef returns common.schema.json's own $defs.<name>, decoded --
// what every nested resolver below reads from, since ports and every
// step action object are defined there, not in any one type's own
// schema (host.schema.json etc. only `$ref` them).
func commonDef(name string) (propDoc, error) {
	raw, err := schema.RawCommonSchema()
	if err != nil {
		return propDoc{}, err
	}
	var doc struct {
		Defs map[string]propDoc `json:"$defs"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return propDoc{}, err
	}
	def, ok := doc.Defs[name]
	if !ok {
		return propDoc{}, fmt.Errorf("no $defs.%s in common.json", name)
	}
	return def, nil
}

// portsFields is `ports:`'s own two fields (tcp, udp) -- what nested
// completion offers inside a host or container's `ports:` block,
// flow-style (`ports: { | }`) or block-style alike.
func portsFields() ([]schemaField, error) {
	def, err := commonDef("portsBlock")
	if err != nil {
		return nil, err
	}
	return fieldsFromProperties(def.Properties), nil
}

// stepActionKeys is every step's own action key (script, run, download,
// ..., schedule) plus validate -- what completion offers when starting a
// brand new `steps:` list item. stepItem's own top-level properties ARE
// exactly this set (see common.schema.json's own $comment on stepItem:
// "exactly one action key required").
func stepActionKeys() ([]schemaField, error) {
	def, err := commonDef("stepItem")
	if err != nil {
		return nil, err
	}
	return fieldsFromProperties(def.Properties), nil
}

// scheduleActionKeys is scheduleItem's own action keys -- the same set
// stepActionKeys returns, minus `schedule` itself (never a schedule
// kind) and plus `when` (not an action, filtered out by callers the
// same way they already filter `validate`).
func scheduleActionKeys() ([]schemaField, error) {
	def, err := commonDef("scheduleItem")
	if err != nil {
		return nil, err
	}
	return fieldsFromProperties(def.Properties), nil
}

// validateCheckKeys is every validator's own check kind (file_exists,
// user_exists, service_running, port_listening, ...) -- what completion
// offers when starting a new `- ` item inside a `validate:` block.
// validatorItem's properties ARE exactly that set.
func validateCheckKeys() ([]schemaField, error) {
	def, err := commonDef("validatorItem")
	if err != nil {
		return nil, err
	}
	return fieldsFromProperties(def.Properties), nil
}

// stepActionFields is one action's own sub-fields -- write_file's
// path/content/mode, download's from/to, and so on -- what completion
// offers inside that action's own `{ ... }`. Scalar actions (script,
// run, reboot's own optional delay aside) have no properties of their
// own to offer, which this returns as "ok, zero fields" rather than an
// error -- a real, valid answer (there's genuinely nothing to complete
// inside `script: |`, since script's value is the script name itself,
// not an object), not a failure to resolve one.
func stepActionFields(action string) ([]schemaField, bool, error) {
	def, err := commonDef("stepItem")
	if err != nil {
		return nil, false, err
	}
	actionDef, ok := def.Properties[action]
	if !ok {
		return nil, false, nil
	}
	return fieldsFromProperties(actionDef.Properties), true, nil
}

// scheduleActionFields is scheduleActionKeys' own per-action sub-fields,
// the schedule-entry analog of stepActionFields.
func scheduleActionFields(action string) ([]schemaField, bool, error) {
	def, err := commonDef("scheduleItem")
	if err != nil {
		return nil, false, err
	}
	actionDef, ok := def.Properties[action]
	if !ok {
		return nil, false, nil
	}
	return fieldsFromProperties(actionDef.Properties), true, nil
}

// headerKind reads a content file's own required first line ("Every
// document starts with a header line, `<type>:`, which is what declares
// what the object is -- its name lives one level in now, alongside
// every other field, see host.schema.json's own $comment) to know which
// schema governs it, without a full YAML decode -- completion needs this
// before the buffer necessarily parses cleanly (that's exactly when
// completion is most useful). `>=`, not `>`: the header line is now
// exactly "host:" with nothing after it (a block follows on later
// lines), not "host: <name>" on one line -- a real behavior change from
// when the name lived inline here.
func headerKind(text string) (schema.Kind, bool) {
	for _, line := range splitLines(text) {
		trimmed := trimLeadingSpace(line)
		if trimmed == "" || trimmed[0] == '#' {
			continue
		}
		for _, k := range schema.Kinds() {
			prefix := string(k) + ":"
			if len(trimmed) >= len(prefix) && trimmed[:len(prefix)] == prefix {
				return k, true
			}
		}
		return "", false // first non-blank, non-comment line isn't a header -- give up rather than guess
	}
	return "", false
}

// documentAt is the real fix for "a file with several `---`-separated
// objects gets Kind-detected once, from the first document only"
// -- returns the sub-text of the document that
// actually contains line, plus that document's own first line number
// within the whole buffer, so a caller can run headerKind (and anything
// position-relative, like configCompletion's indent check) against just
// that document instead of always the file's first one. Every content
// file is exactly one `---`-separated document today, so this is a
// no-op for the common case: one document, `[0, len(lines))`, start 0.
func documentAt(text string, line uint32) (doc string, startLine int) {
	lines := splitLines(text)
	type span struct{ start, end int }
	var segs []span
	start := 0
	for i, l := range lines {
		if strings.TrimRight(l, "\r") == "---" {
			segs = append(segs, span{start, i})
			start = i + 1
		}
	}
	segs = append(segs, span{start, len(lines)})

	for _, seg := range segs {
		if int(line) >= seg.start && int(line) < seg.end {
			return strings.Join(lines[seg.start:seg.end], "\n"), seg.start
		}
	}
	// Past the end of every segment (cursor on a blank trailing line at
	// true EOF, same out-of-bounds case lineAt itself tolerates) -- the
	// last segment is the one to extend.
	last := segs[len(segs)-1]
	return strings.Join(lines[last.start:last.end], "\n"), last.start
}

// ancestorKeyAt is the real "position-tracking," scoped to what
// completion actually needs (`steps:`, `ports:`)
// rather than a full YAML AST: walking upward from localLine (within a
// single document, indentation-based, matching this repo's own uniform
// 2-space convention), it returns the key name of the nearest line whose
// indentation is exactly wantIndent -- the key that OWNS whatever block
// localLine's own line sits inside, at that depth. Returns "" once
// indentation drops below wantIndent before such a line is found (we've
// left any block that could be at that depth) or if the line found at
// wantIndent isn't a plain `key:` (a list item's own dash is at
// wantIndent+2 in this repo's convention, so list items are transparent
// here, not mistaken for the owning key).
func ancestorKeyAt(doc string, localLine uint32, wantIndent int) string {
	lines := splitLines(doc)
	for i := int(localLine) - 1; i >= 0; i-- {
		l := lines[i]
		if strings.TrimSpace(l) == "" {
			continue
		}
		ind := leadingSpaceCount(l)
		if ind < wantIndent {
			return ""
		}
		if ind == wantIndent {
			trimmed := strings.TrimSpace(l)
			if m := presentKeyRE.FindStringSubmatch(trimmed); m != nil {
				return m[1]
			}
			return ""
		}
		// ind > wantIndent: still inside a deeper block above us
		// (a sibling steps: item, or content nested even further) --
		// keep walking up past it.
	}
	return ""
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	lines = append(lines, s[start:])
	return lines
}

func trimLeadingSpace(s string) string {
	i := 0
	for i < len(s) && (s[i] == ' ' || s[i] == '\t' || s[i] == '\r') {
		i++
	}
	return s[i:]
}
