package lsp

import (
	"fmt"
	"regexp"
	"strings"

	"go.lsp.dev/protocol"

	"github.com/globalcptc/laforge/internal/loader"
)

var keyLineRE = regexp.MustCompile(`^(\s*)([A-Za-z_][A-Za-z0-9_]*):\s*$`)

// identifierOnlyRE is "this trimmed prefix is still just a bare, partial
// key name" (e.g. "po", "vdi_") -- the one shape every field-name
// completion path below now treats as a live filter rather than
// declining outright. Anything else (a colon, a brace, a quote, other
// punctuation) stays a real "don't guess" case.
var identifierOnlyRE = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// Completion is "Completion in config files: driven by the JSON Schema...
// Completion in scripts: offers the real template context",
// dispatched purely on whether
// relPath is a content YAML file or a script's own source file --
// deliberately two different, real completion sources rather than one
// generic mechanism pretending to cover both.
func Completion(w *Workspace, relPath, text string, pos protocol.Position) []protocol.CompletionItem {
	content, _ := w.Snapshot()
	if content == nil {
		return nil
	}
	if script := scriptForSourceFile(content, relPath); script != nil {
		return scriptCompletion(content, script, text, pos)
	}
	if strings.HasSuffix(relPath, ".yaml") || strings.HasSuffix(relPath, ".yml") {
		return configCompletion(text, pos)
	}
	return nil
}

// nestedFieldIndent is how deep a type's own fields sit under its header
// line now that every field lives inside the type key's own object
// (`host:\n  name: ...\n  os: ...`, not `host: <name>` with os/size/...
// as top-level siblings -- see host.schema.json's own $comment). Content
// in this repo is uniformly 2-space indented, matching every real
// example file.
const nestedFieldIndent = 2

func configCompletion(text string, pos protocol.Position) []protocol.CompletionItem {
	// documentAt: a file can hold several `---`-separated objects,
	// and the cursor's own document is the one
	// whose header/fields/indentation actually apply -- not necessarily
	// the file's first one.
	doc, docStart := documentAt(text, pos.Line)
	kind, ok := headerKind(doc)
	if !ok {
		return nil
	}
	localLine := pos.Line - uint32(docStart)
	line := lineAt(doc, localLine)
	prefix := runePrefix(line, pos.Character)

	// Flow-style, one level past the type's own fields: `ports: { | }`
	// or `write_file: { path: "x", | }` -- resolved and handled first,
	// since this repo's real content overwhelmingly writes both ports
	// and step actions this way.
	if items, handled := flowFieldCompletion(doc, prefix); handled {
		return items
	}

	// A brand new `steps:` list item -- offer the step's own action
	// keys (script, download, write_file, ...), not the type's own
	// top-level fields.
	if items, handled := stepListItemCompletion(doc, localLine, line, prefix); handled {
		return items
	}

	// Same for a brand new `schedule:` list item -- the same action
	// keys, plus `when`, against a `schedule` ancestor instead of `steps`.
	if items, handled := scheduleListItemCompletion(doc, localLine, line, prefix); handled {
		return items
	}

	// A new `- ` item inside a `validate:` block -- offer the check kinds
	// (service_running, port_listening, ...), which can nest under a step,
	// a schedule entry, or a script definition at any depth.
	if items, handled := validateListItemCompletion(doc, localLine, line, prefix); handled {
		return items
	}

	fields, err := schemaFieldsForKind(kind)
	if err != nil {
		return nil
	}

	if m := keyLineRE.FindStringSubmatch(prefix); m != nil {
		key := m[2]
		for _, f := range fields {
			if f.Name == key && len(f.Enum) > 0 {
				items := make([]protocol.CompletionItem, len(f.Enum))
				for i, e := range f.Enum {
					items[i] = protocol.CompletionItem{Label: e, Kind: protocol.CompletionItemKindEnumMember}
				}
				return items
			}
		}
		if key == "when" && ancestorKeyAt(doc, localLine, leadingSpaceCount(line)-2) == "schedule" {
			return whenStarterCompletion()
		}
		return nil // a known key with no enum -- nothing to complete for its value
	}

	typed := strings.TrimSpace(prefix)
	if typed != "" && !identifierOnlyRE.MatchString(typed) {
		return nil // mid-value or otherwise-unrecognized on this line -- don't guess
	}

	indent := leadingSpaceCount(line)
	indentStr := line[:indent]
	if indent == nestedFieldIndent {
		// The type's own fields, directly under its header line.
		present := presentKeysInBlock(doc, nestedFieldIndent)
		return fieldItems(fields, present, typed, indentStr)
	}
	if indent == nestedFieldIndent+2 && ancestorKeyAt(doc, localLine, nestedFieldIndent) == "ports" {
		// Block-style ports:, the other real shape it's written in --
		// same fields flowFieldCompletion resolves for the `{ }` form.
		portsFieldsList, err := portsFields()
		if err != nil {
			return nil
		}
		present := presentKeysInBlock(doc, indent)
		return fieldItems(portsFieldsList, present, typed, indentStr)
	}
	// Any other depth -- genuinely nested content past what steps/ports
	// resolve (e.g. inside a `validate:` block, or a step's own action
	// object written block-style, which no real content here does) --
	// offer nothing rather than the wrong list.
	return nil
}

// fieldItems turns []schemaField into real completion items, excluding
// whatever's in present and, when typed is non-empty, anything that
// doesn't start with it -- the one place that shape is built, shared by
// every nested resolver in this file so a "what am I missing, not what
// do I already have" field list, detail, and documentation always look
// the same regardless of which context produced it.
//
// A field with a real schema "example" (steps, ports, findings, tags,
// vars -- see internal/schema/schemas/*.schema.json) gets that example
// as its InsertText, indented to fit where the cursor actually is, not
// just its bare name: accepting "ports" produces a working
// `ports: { tcp: ["8080"] }`, not a name you then still have to know
// the shape of. indent is the current line's own leading whitespace --
// prepended to every line of a multi-line example after its first,
// since a plain-text completion insert doesn't auto-indent continuation
// lines the way typing does. Fields with no example (still most of
// them) fall back to the plain label, same as before.
func fieldItems(fields []schemaField, present map[string]bool, typed string, indent string) []protocol.CompletionItem {
	items := make([]protocol.CompletionItem, 0, len(fields))
	for _, f := range fields {
		if present[f.Name] {
			continue
		}
		if typed != "" && !strings.HasPrefix(f.Name, typed) {
			continue
		}
		detail := f.Type
		if len(f.Enum) > 0 {
			detail = "enum: " + strings.Join(f.Enum, " | ")
		}
		item := protocol.CompletionItem{
			Label: f.Name,
			Kind:  protocol.CompletionItemKindField,
		}
		if f.Example != "" {
			item.InsertText = protocol.NewOptional(indentContinuationLines(f.Example, indent))
		}
		if detail != "" {
			item.Detail = protocol.NewOptional(detail)
		}
		if f.Description != "" {
			item.Documentation = &protocol.MarkupContent{Kind: protocol.MarkupKindPlainText, Value: f.Description}
		}
		items = append(items, item)
	}
	return items
}

// indentContinuationLines prepends indent to every line of example after
// its first -- the first line lands wherever the cursor already is (past
// whatever indentation is already on the line), but each subsequent
// newline resets to column 0 in a plain-text insert, so without this a
// multi-line example would land flush against the left margin instead
// of nested under the key it belongs to.
func indentContinuationLines(example, indent string) string {
	if indent == "" || !strings.Contains(example, "\n") {
		return example
	}
	lines := strings.Split(example, "\n")
	for i := 1; i < len(lines); i++ {
		lines[i] = indent + lines[i]
	}
	return strings.Join(lines, "\n")
}

// flowOpenRE matches "<key>: {" followed by whatever's been typed inside
// the flow mapping so far, up to the cursor, with no further `{`/`}` in
// between -- single-level flow nesting, which is the only kind any real
// content in this repo uses (`ports: { tcp: [...] }`, `write_file: {
// path: ..., content: ... }`). flowReadyRE is "the flow content so far
// ends right at a place a new key could start" -- immediately after the
// opening `{`, or after a `,`, optionally with whitespace -- not
// mid-identifier or mid-value, the same "don't guess" rule
// configCompletion's own top-level check already applies.
var flowOpenRE = regexp.MustCompile(`([A-Za-z_][A-Za-z0-9_]*):\s*\{([^{}]*)$`)
var flowKeyRE = regexp.MustCompile(`(?:^|,)\s*([A-Za-z_][A-Za-z0-9_]*)\s*:`)

// flowReadyRE is "the flow content so far ends right where a key could
// start, optionally with a partial key name already typed" -- right
// after the opening `{` or a `,`, optional whitespace, then an optional
// bare identifier fragment (captured as the filter prefix). Still not
// mid-value: a value in progress (a quote, a `[`, digits after a `:`)
// fails to match at all, which stays "don't guess."
var flowReadyRE = regexp.MustCompile(`(?:^|,)\s*([A-Za-z_][A-Za-z0-9_]*)?$`)

// flowFieldCompletion resolves `ports: { | }` and any step action's own
// `key: { | }` (write_file, download, ...) to that key's real sub-fields.
// The bool return is "this position was recognized as flow-style
// content at all" (even if not ready for a new key, or the owning key
// isn't one this resolves) -- true suppresses configCompletion's other
// branches from guessing at a position that's genuinely inside a flow
// mapping this function just doesn't have a field list for.
func flowFieldCompletion(doc, prefix string) ([]protocol.CompletionItem, bool) {
	m := flowOpenRE.FindStringSubmatch(prefix)
	if m == nil {
		return nil, false
	}
	key, flowSoFar := m[1], m[2]
	readyMatch := flowReadyRE.FindStringSubmatch(flowSoFar)
	if readyMatch == nil {
		return nil, true // inside a real flow block, but mid-value -- don't guess
	}
	typed := readyMatch[1]
	var fields []schemaField
	var err error
	switch key {
	case "ports":
		fields, err = portsFields()
	default:
		var ok bool
		fields, ok, err = stepActionFields(key)
		if err == nil && !ok {
			return nil, false // not ports, and not a recognized step action either -- not our position to handle
		}
	}
	if err != nil {
		return nil, false
	}
	present := map[string]bool{}
	for _, fm := range flowKeyRE.FindAllStringSubmatch(flowSoFar, -1) {
		present[fm[1]] = true
	}
	return fieldItems(fields, present, typed, ""), true
}

// stepListItemCompletion offers step action keys (script, download,
// write_file, ...) when the cursor is starting a brand new item in a
// `steps:` list -- an empty line, a bare `-`, or either of those plus a
// partially-typed action name (`- wr`, `wr`), whose nearest ancestor key
// (indentation-wise) is `steps`. Returns each item's InsertText including
// the leading "- " when the user hasn't typed the dash yet, so accepting
// a completion produces a real, valid step (`- script: `) in one action
// rather than two.
func stepListItemCompletion(doc string, localLine uint32, line, prefix string) ([]protocol.CompletionItem, bool) {
	trimmed := strings.TrimSpace(prefix)
	needsDash := true
	typed := trimmed
	if strings.HasPrefix(trimmed, "-") {
		needsDash = false
		typed = strings.TrimSpace(strings.TrimPrefix(trimmed, "-"))
	}
	if typed != "" && !identifierOnlyRE.MatchString(typed) {
		return nil, false
	}
	indent := leadingSpaceCount(line)
	if indent < 2 || ancestorKeyAt(doc, localLine, indent-2) != "steps" {
		return nil, false
	}
	fields, err := stepActionKeys()
	if err != nil {
		return nil, false
	}
	// "- " (when needed) adds 2 columns in front of the example's own
	// first line, so a continuation line (validate's own multi-line
	// example) needs indent plus those same 2 columns to land under the
	// action key, not under the dash.
	indentStr := line[:indent]
	if needsDash {
		indentStr += "  "
	}
	items := fieldItems(fields, nil, typed, indentStr)
	for i := range items {
		// fieldItems already set InsertText from the action's own real
		// example (see common.schema.json's stepItem.properties) when it
		// has one -- every real action does now. Only a genuinely
		// exampleless field would fall back to the bare "label: ".
		insert, ok := items[i].InsertText.Get()
		if !ok || insert == "" {
			insert = items[i].Label + ": "
		}
		if needsDash {
			insert = "- " + insert
		}
		items[i].InsertText = protocol.NewOptional(insert)
	}
	return items, true
}

// scheduleListItemCompletion is stepListItemCompletion's own twin for a
// `schedule:` list -- same new-item detection (blank line, bare `-`, or
// either plus a partial name), same InsertText-from-real-example
// behavior, but offering scheduleActionKeys (the 13 actions valid in a
// schedule entry, plus `when` itself -- a real entry needs both, in
// either order) against an ancestor key of `schedule` instead of `steps`.
func scheduleListItemCompletion(doc string, localLine uint32, line, prefix string) ([]protocol.CompletionItem, bool) {
	trimmed := strings.TrimSpace(prefix)
	needsDash := true
	typed := trimmed
	if strings.HasPrefix(trimmed, "-") {
		needsDash = false
		typed = strings.TrimSpace(strings.TrimPrefix(trimmed, "-"))
	}
	if typed != "" && !identifierOnlyRE.MatchString(typed) {
		return nil, false
	}
	indent := leadingSpaceCount(line)
	if indent < 2 || ancestorKeyAt(doc, localLine, indent-2) != "schedule" {
		return nil, false
	}
	fields, err := scheduleActionKeys()
	if err != nil {
		return nil, false
	}
	indentStr := line[:indent]
	if needsDash {
		indentStr += "  "
	}
	items := fieldItems(fields, nil, typed, indentStr)
	for i := range items {
		insert, ok := items[i].InsertText.Get()
		if !ok || insert == "" {
			insert = items[i].Label + ": "
		}
		if needsDash {
			insert = "- " + insert
		}
		items[i].InsertText = protocol.NewOptional(insert)
	}
	return items, true
}

// validateListItemCompletion offers the validator check kinds (file_exists,
// user_exists, service_running, port_listening, ...) when the cursor starts a
// new `- ` item inside a `validate:` block -- which can appear at any depth (a
// step's own validate, a schedule entry's, a script's), so this keys off the
// `validate` ancestor two columns shallower, not an absolute indent. Mirrors
// stepListItemCompletion.
func validateListItemCompletion(doc string, localLine uint32, line, prefix string) ([]protocol.CompletionItem, bool) {
	trimmed := strings.TrimSpace(prefix)
	needsDash := true
	typed := trimmed
	if strings.HasPrefix(trimmed, "-") {
		needsDash = false
		typed = strings.TrimSpace(strings.TrimPrefix(trimmed, "-"))
	}
	if typed != "" && !identifierOnlyRE.MatchString(typed) {
		return nil, false
	}
	indent := leadingSpaceCount(line)
	if indent < 2 || ancestorKeyAt(doc, localLine, indent-2) != "validate" {
		return nil, false
	}
	fields, err := validateCheckKeys()
	if err != nil {
		return nil, false
	}
	indentStr := line[:indent]
	if needsDash {
		indentStr += "  "
	}
	items := fieldItems(fields, nil, typed, indentStr)
	for i := range items {
		insert, ok := items[i].InsertText.Get()
		if !ok || insert == "" {
			insert = items[i].Label + ": "
		}
		if needsDash {
			insert = "- " + insert
		}
		items[i].InsertText = protocol.NewOptional(insert)
	}
	return items, true
}

// leadingSpaceCount is indentation depth in spaces -- tabs never appear
// in this repo's real content (every example is 2-space indented), so
// this doesn't need to reason about tab width.
func leadingSpaceCount(line string) int {
	n := 0
	for _, r := range line {
		if r != ' ' {
			break
		}
		n++
	}
	return n
}

// presentKeysInBlock is "what's already set" for configCompletion's own
// "options I'm missing, not options I already have" rule -- every key at
// exactly indent spaces anywhere in the document. Not position-aware
// about which *block* those keys belong to (a document only ever has one
// type key at the top, so there's only one indent-2 block to begin
// with -- this only needs to stop mattering once nested completion goes
// deeper than one level).
var presentKeyRE = regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_]*):`)

func presentKeysInBlock(text string, indent int) map[string]bool {
	present := map[string]bool{}
	prefix := strings.Repeat(" ", indent)
	for _, line := range splitLines(text) {
		if !strings.HasPrefix(line, prefix) || leadingSpaceCount(line) != indent {
			continue
		}
		if m := presentKeyRE.FindStringSubmatch(line[indent:]); m != nil {
			present[m[1]] = true
		}
	}
	return present
}

// scriptCompletion only offers anything inside an unclosed `{{` on the
// current line -- "in scope where?" only means
// something inside an actual template expression; the rest of a shell
// script is just shell script.
func scriptCompletion(content *loader.Content, script *loader.Script, text string, pos protocol.Position) []protocol.CompletionItem {
	line := lineAt(text, pos.Line)
	prefix := runePrefix(line, pos.Character)
	if !insideTemplateExpr(prefix) {
		return nil
	}

	vars, peopleSources := scriptScope(content, script.Name)

	var items []protocol.CompletionItem
	for _, v := range vars {
		detail := fmt.Sprintf("%s var, example: %s", v.Source, v.ExampleValue)
		if v.SeenIn < v.Total {
			detail = fmt.Sprintf("available on %d/%d hosts running this script -- %s", v.SeenIn, v.Total, detail)
		}
		items = append(items, protocol.CompletionItem{
			Label:  "vars." + v.Key,
			Kind:   protocol.CompletionItemKindVariable,
			Detail: protocol.NewOptional(detail),
		})
	}

	for _, f := range []string{"host.hostname", "host.address", "host.os", "host.image", "host.size", "host.kind"} {
		items = append(items, protocol.CompletionItem{Label: f, Kind: protocol.CompletionItemKindField, Detail: protocol.NewOptional("always available")})
	}
	for _, f := range []string{"network.name", "network.cidr"} {
		items = append(items, protocol.CompletionItem{Label: f, Kind: protocol.CompletionItemKindField, Detail: protocol.NewOptional("always available")})
	}
	for _, f := range []string{"build.environment", "build.team", "build.teams", "build.start", "build.stop"} {
		items = append(items, protocol.CompletionItem{Label: f, Kind: protocol.CompletionItemKindField, Detail: protocol.NewOptional("always available")})
	}
	for _, ps := range peopleSources {
		cols := strings.Join(peopleColumns(ps), ", ")
		items = append(items, protocol.CompletionItem{
			Label:      fmt.Sprintf(`people "%s"`, ps.Name),
			InsertText: protocol.NewOptional(fmt.Sprintf(`people "%s"`, ps.Name)),
			Kind:       protocol.CompletionItemKindFunction,
			Detail:     protocol.NewOptional("columns: " + cols),
		})
	}
	return items
}

// peopleColumns is "username" plus every attribute key from a source's
// first row -- loader.PeopleSource carries no column list of its own
// (only render.PeopleSummary does, and that requires a resolved Context
// this completion path doesn't have), and every real CSV has the same
// columns for every row, so the first row is enough.
func peopleColumns(ps loader.PeopleSource) []string {
	cols := []string{"username"}
	if len(ps.People) == 0 {
		return cols
	}
	for k := range ps.People[0].Attributes {
		cols = append(cols, k)
	}
	return cols
}

// insideTemplateExpr reports whether prefix (the current line up to the
// cursor) has an unclosed `{{` -- the last `{{` comes after the last
// `}}`, or there's a `{{` with no `}}` at all.
func insideTemplateExpr(prefix string) bool {
	open := strings.LastIndex(prefix, "{{")
	if open < 0 {
		return false
	}
	lastClose := strings.LastIndex(prefix, "}}")
	return lastClose < open
}

func lineAt(text string, line uint32) string {
	lines := splitLines(text)
	if int(line) >= len(lines) {
		return ""
	}
	return lines[line]
}

// runePrefix slices s up to the given LSP character offset. LSP
// positions are UTF-16 code units by default; this treats Character as
// a rune offset instead, which matches UTF-16 exactly for the ASCII
// content every real .laforge YAML/script file in this repo is written
// in -- a real, documented simplification,
// not a silent bug for the content this actually needs to handle.
func runePrefix(s string, character uint32) string {
	r := []rune(s)
	if int(character) > len(r) {
		character = uint32(len(r))
	}
	return string(r[:character])
}

// whenStarterCompletion offers a handful of literal example phrases for
// a schedule entry's `when:` value, spanning all three grammar clause
// kinds (interval, daily-at, anchored) and all four anchors -- a
// starting point to edit, not a grammar-aware assistant that builds a
// phrase for you (out of scope). Deliberately NOT driven by a real JSON
// Schema "enum" on `when` (internal/schema/schemas/common.schema.json):
// a real enum would wrongly restrict validation to only these literal
// strings, defeating the entire point of internal/schedule's grammar
// accepting real variation in phrasing. This is completion-only.
func whenStarterCompletion() []protocol.CompletionItem {
	phrases := []string{
		"Every hour",
		"Every 30 minutes",
		"Every day at 10:00am and 2:00pm",
		"Every 30 minutes after 2:00pm",
		"45 minutes after competition start",
		"2 hours before competition end",
		"1 hour after access opens",
		"30 minutes before access closes",
	}
	items := make([]protocol.CompletionItem, len(phrases))
	for i, p := range phrases {
		items[i] = protocol.CompletionItem{Label: p, Kind: protocol.CompletionItemKindEnumMember}
	}
	return items
}
