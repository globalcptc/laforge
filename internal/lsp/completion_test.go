package lsp

import (
	"testing"

	"go.lsp.dev/protocol"
)

func labels(items []protocol.CompletionItem) map[string]protocol.CompletionItem {
	m := make(map[string]protocol.CompletionItem, len(items))
	for _, it := range items {
		m[it.Label] = it
	}
	return m
}

// TestConfigCompletionNestedFields is "everything a host has lives
// inside the host: object" made concrete: completion one level under
// the header line (where every real field actually lives now -- see
// host.schema.json's own $comment) offers the type's own fields.
func TestConfigCompletionNestedFields(t *testing.T) {
	text := "host:\n  name: webserver\n  \n"
	items := configCompletion(text, protocol.Position{Line: 2, Character: 2})
	got := labels(items)
	for _, want := range []string{"os", "size", "disk", "ports", "steps", "depends_on"} {
		if _, ok := got[want]; !ok {
			t.Errorf("nested host completion missing %q; got %v", want, got)
		}
	}
	// "options I'm missing, not options I already have": name is
	// already set on line 2, so it shouldn't be re-offered.
	if _, ok := got["name"]; ok {
		t.Errorf("nested host completion offered %q again, want it excluded -- it's already set", "name")
	}
}

// TestConfigCompletionTopLevelOffersNothing is the flip side: at the
// document's own top level (indent 0), the only real key is the type
// header itself, which is already there by definition -- there's
// nothing left to usefully complete at that depth.
func TestConfigCompletionTopLevelOffersNothing(t *testing.T) {
	text := "host:\n  name: webserver\n\n"
	items := configCompletion(text, protocol.Position{Line: 2, Character: 0})
	if len(items) != 0 {
		t.Errorf("top-level completion = %v, want none (the type key is already there)", items)
	}
}

// TestConfigCompletionSecondDocumentInMultiDocFile is the real fix for
// "a file with several `---`-separated objects gets Kind-detected once,
// from the first document only" -- a network
// document sits after a host document in the same file, and completion
// inside the SECOND one must offer network's own fields (cidr,
// visible_from, ...), not host's.
func TestConfigCompletionSecondDocumentInMultiDocFile(t *testing.T) {
	text := "host:\n  name: web01\n  os: ubuntu22\n---\nnetwork:\n  name: extra\n  \n"
	items := configCompletion(text, protocol.Position{Line: 6, Character: 2})
	got := labels(items)
	for _, want := range []string{"cidr", "visible_from"} {
		if _, ok := got[want]; !ok {
			t.Errorf("second-document completion missing network field %q; got %v", want, got)
		}
	}
	for _, notWant := range []string{"os", "size", "disk"} {
		if _, ok := got[notWant]; ok {
			t.Errorf("second-document completion offered host field %q, want only network's own fields", notWant)
		}
	}
}

// TestConfigCompletionPortsFlowStyle is the real fix for "let's go ahead
// and fix the steps and ports" -- `ports: { }` flow-style is how every
// real example in this repo actually writes it, and completion inside
// it used to offer nothing at all.
func TestConfigCompletionPortsFlowStyle(t *testing.T) {
	text := "host:\n  name: webserver\n  ports: { \n"
	items := configCompletion(text, protocol.Position{Line: 2, Character: uint32(len("  ports: { "))})
	got := labels(items)
	for _, want := range []string{"tcp", "udp"} {
		if _, ok := got[want]; !ok {
			t.Errorf("ports flow completion missing %q; got %v", want, got)
		}
	}
}

// TestConfigCompletionPortsFlowStyleExcludesPresent proves the same
// "what am I missing" rule applies inside a flow block: tcp is already
// there, so only udp should be offered.
func TestConfigCompletionPortsFlowStyleExcludesPresent(t *testing.T) {
	text := "host:\n  name: webserver\n  ports: { tcp: [\"80\"], \n"
	items := configCompletion(text, protocol.Position{Line: 2, Character: uint32(len("  ports: { tcp: [\"80\"], "))})
	got := labels(items)
	if _, ok := got["tcp"]; ok {
		t.Errorf("ports flow completion re-offered \"tcp\", already present; got %v", got)
	}
	if _, ok := got["udp"]; !ok {
		t.Errorf("ports flow completion missing \"udp\"; got %v", got)
	}
}

// TestConfigCompletionPortsBlockStyle is the same fields, block style --
// `ports:` with `tcp:`/`udp:` nested underneath rather than `{ }`.
func TestConfigCompletionPortsBlockStyle(t *testing.T) {
	text := "host:\n  name: webserver\n  ports:\n    \n"
	items := configCompletion(text, protocol.Position{Line: 3, Character: 4})
	got := labels(items)
	for _, want := range []string{"tcp", "udp"} {
		if _, ok := got[want]; !ok {
			t.Errorf("ports block completion missing %q; got %v", want, got)
		}
	}
}

// TestConfigCompletionStepActionFlowStyle is a step's own sub-fields --
// write_file's path/content/mode -- inside its real, flow-style shape
// (`- write_file: { path: ..., content: ... }`, exactly what every real
// step in this repo's own examples uses).
func TestConfigCompletionStepActionFlowStyle(t *testing.T) {
	text := "host:\n  name: webserver\n  steps:\n    - write_file: { \n"
	items := configCompletion(text, protocol.Position{Line: 3, Character: uint32(len("    - write_file: { "))})
	got := labels(items)
	for _, want := range []string{"path", "content", "mode"} {
		if _, ok := got[want]; !ok {
			t.Errorf("write_file flow completion missing %q; got %v", want, got)
		}
	}
}

// TestConfigCompletionStepActionSubFieldInsertsFullExample is a real
// live report: accepting "path" inside `write_file: { }` produced only
// the bare word "path", still leaving the real shape as something to
// look up. Every sub-field of every step action (common.schema.json's
// stepItem.properties.*.properties) now has its own "example", the same
// convention as the top-level fields, so accepting one inserts a real
// starter value.
func TestConfigCompletionStepActionSubFieldInsertsFullExample(t *testing.T) {
	text := "host:\n  name: webserver\n  steps:\n    - write_file: { pa\n"
	items := configCompletion(text, protocol.Position{Line: 3, Character: uint32(len("    - write_file: { pa"))})
	got := labels(items)
	insert, ok := got["path"].InsertText.Get()
	if !ok || insert != "path: /etc/motd" {
		t.Errorf("InsertText for \"path\" = %q, want \"path: /etc/motd\"", insert)
	}
}

// TestConfigCompletionPortsSubFieldInsertsFullExample is the same fix
// applied to ports: { } -- accepting "tcp" now inserts a real example
// array, not just the bare key.
func TestConfigCompletionPortsSubFieldInsertsFullExample(t *testing.T) {
	text := "host:\n  name: webserver\n  ports: { tc\n"
	items := configCompletion(text, protocol.Position{Line: 2, Character: uint32(len("  ports: { tc"))})
	got := labels(items)
	insert, ok := got["tcp"].InsertText.Get()
	want := `tcp: ["8080"]`
	if !ok || insert != want {
		t.Errorf("InsertText for \"tcp\" = %q, want %q", insert, want)
	}
}

// TestConfigCompletionStepActionFlowStyleMidValueOffersNothing confirms
// the same "don't guess mid-value" rule applies inside a step action's
// own flow block, not just at the type's own top level.
func TestConfigCompletionStepActionFlowStyleMidValueOffersNothing(t *testing.T) {
	text := "host:\n  name: webserver\n  steps:\n    - write_file: { path: \"/etc/mot\n"
	items := configCompletion(text, protocol.Position{Line: 3, Character: uint32(len("    - write_file: { path: \"/etc/mot"))})
	if len(items) != 0 {
		t.Errorf("mid-value flow completion = %v, want none", items)
	}
}

// TestConfigCompletionNewStepListItem is the other real ask: starting a
// brand new `steps:` list item (a blank line right after Enter, exactly
// what the auto-trigger in extension.ts fires on) offers the real step
// action keys, not the host's own top-level fields.
func TestConfigCompletionNewStepListItem(t *testing.T) {
	text := "host:\n  name: webserver\n  steps:\n    - script: base\n    \n"
	items := configCompletion(text, protocol.Position{Line: 4, Character: 4})
	got := labels(items)
	for _, want := range []string{"script", "download", "write_file", "reboot"} {
		if _, ok := got[want]; !ok {
			t.Errorf("new steps list item completion missing %q; got %v", want, got)
		}
	}
	if _, ok := got["schedule"]; ok {
		t.Errorf("new steps list item completion offered \"schedule\" -- it's no longer a step kind, it's the top-level schedule: field now; got %v", got)
	}
	// A real, live report: getting back "- download: " with nothing else
	// meant still having to look up download's own shape by hand.
	// Accepting a step action now inserts that action's own real example
	// (common.schema.json's stepItem.properties.download), not just the
	// bare key.
	insert, ok := got["download"].InsertText.Get()
	if !ok || insert != "- download: { from: https://files.cp.tc/site.tar.gz, to: /tmp/site.tar.gz }" {
		t.Errorf("new steps list item InsertText for \"download\" = %q, want the real download example, dash-prefixed", insert)
	}
}

// TestConfigCompletionNewStepListItemAfterDash is the same position, but
// the user already typed the leading "- " -- InsertText must not add a
// second one.
func TestConfigCompletionNewStepListItemAfterDash(t *testing.T) {
	text := "host:\n  name: webserver\n  steps:\n    - \n"
	items := configCompletion(text, protocol.Position{Line: 3, Character: 6})
	got := labels(items)
	insert, ok := got["script"].InsertText.Get()
	if !ok || insert != "script: base" {
		t.Errorf("new steps list item (dash already typed) InsertText for \"script\" = %q, want \"script: base\" (script's own real example, no extra dash)", insert)
	}
}

// TestConfigCompletionPartialPrefixFiltersTopLevelFields is the real
// fix for a live report: typing "po" on a container offered nothing
// from this provider at all (the old "don't guess" rule bailed on any
// non-blank prefix) -- so whatever the editor actually showed for it
// wasn't from here, and this is what closes that gap: a bare partial
// identifier is now a live filter, not an instant decline.
func TestConfigCompletionPartialPrefixFiltersTopLevelFields(t *testing.T) {
	text := "container:\n  name: scoreboard\n  po\n"
	items := configCompletion(text, protocol.Position{Line: 2, Character: 4})
	got := labels(items)
	if _, ok := got["ports"]; !ok {
		t.Errorf("prefix \"po\" missing \"ports\"; got %v", got)
	}
	for label := range got {
		if label != "ports" {
			t.Errorf("prefix \"po\" should only match names starting with \"po\", got extra %q", label)
		}
	}
}

// TestConfigCompletionPartialPrefixExcludesNonMatches is the same fix's
// other half, against the same live report: "visi" on a network should
// filter down to the one real field (visible_from), not offer nothing
// and not offer anything else.
func TestConfigCompletionPartialPrefixExcludesNonMatches(t *testing.T) {
	text := "network:\n  name: vdi\n  cidr: 10.0.254.0/24\n  visi\n"
	items := configCompletion(text, protocol.Position{Line: 3, Character: 6})
	got := labels(items)
	if _, ok := got["visible_from"]; !ok {
		t.Errorf("prefix \"visi\" missing \"visible_from\"; got %v", got)
	}
	if len(got) != 1 {
		t.Errorf("prefix \"visi\" should match exactly one field, got %v", got)
	}
}

// TestConfigCompletionAcceptingPortsInsertsWorkingSnippet is the second
// half of the same live report: filtering to the right field name
// wasn't enough on its own -- accepting "ports" produced only the bare
// word "ports", still leaving the real shape (`{ tcp: [...] }`) as
// something to look up and type by hand. A field with a real schema
// "example" (internal/schema/schemas/host.schema.json's own "ports"
// entry) now becomes the InsertText, so accepting the completion
// produces an immediately valid, edit-in-place block.
func TestConfigCompletionAcceptingPortsInsertsWorkingSnippet(t *testing.T) {
	text := "container:\n  name: scoreboard\n  po\n"
	items := configCompletion(text, protocol.Position{Line: 2, Character: 4})
	got := labels(items)
	insert, ok := got["ports"].InsertText.Get()
	if !ok {
		t.Fatalf("\"ports\" has no InsertText -- still just the bare label")
	}
	if insert != `ports: { tcp: ["8080"] }` {
		t.Errorf("InsertText for \"ports\" = %q, want a real working snippet", insert)
	}
}

// TestConfigCompletionMultilineExampleIndentsToMatchFile is "steps"'s
// own version of the same fix: its example spans several lines, and
// each continuation line must pick up the current line's own
// indentation (2 existing + 2 more for a list item = 4, matching every
// real steps: list in this repo) rather than landing flush against the
// left margin, which a plain multi-line insert would otherwise do.
func TestConfigCompletionMultilineExampleIndentsToMatchFile(t *testing.T) {
	text := "host:\n  name: webserver\n  st\n"
	items := configCompletion(text, protocol.Position{Line: 2, Character: 4})
	got := labels(items)
	insert, ok := got["steps"].InsertText.Get()
	if !ok {
		t.Fatalf("\"steps\" has no InsertText")
	}
	want := "steps:\n    - script: base\n    - write_file: { path: /etc/motd, content: \"Property of Allports\" }\n    - service: { name: nginx, action: restart }"
	if insert != want {
		t.Errorf("InsertText for \"steps\" =\n%s\nwant\n%s", insert, want)
	}
}

// TestConfigCompletionFlowPartialPrefixFilters is the same filtering
// applied inside a flow block: "ports: { tc" should narrow to tcp, not
// also offer udp.
func TestConfigCompletionFlowPartialPrefixFilters(t *testing.T) {
	text := "host:\n  name: webserver\n  ports: { tc\n"
	items := configCompletion(text, protocol.Position{Line: 2, Character: uint32(len("  ports: { tc"))})
	got := labels(items)
	if _, ok := got["tcp"]; !ok {
		t.Errorf("prefix \"tc\" missing \"tcp\"; got %v", got)
	}
	if _, ok := got["udp"]; ok {
		t.Errorf("prefix \"tc\" shouldn't match \"udp\"; got %v", got)
	}
}

// TestConfigCompletionStepActionPartialPrefixAfterDash is the same
// filtering for a step's own action key, once the dash is already typed.
func TestConfigCompletionStepActionPartialPrefixAfterDash(t *testing.T) {
	text := "host:\n  name: webserver\n  steps:\n    - wr\n"
	items := configCompletion(text, protocol.Position{Line: 3, Character: uint32(len("    - wr"))})
	got := labels(items)
	insert, ok := got["write_file"].InsertText.Get()
	want := `write_file: { path: /etc/motd, content: "Property of Allports" }`
	if !ok || insert != want {
		t.Errorf("InsertText for \"write_file\" after \"- wr\" = %q, want %q (its own real example, dash already typed)", insert, want)
	}
	if _, ok := got["download"]; ok {
		t.Errorf("prefix \"wr\" shouldn't match \"download\"; got %v", got)
	}
}

func TestConfigCompletionEnumValue(t *testing.T) {
	text := "script:\n  name: base\n  language: \n"
	items := configCompletion(text, protocol.Position{Line: 2, Character: uint32(len("  language: "))})
	got := labels(items)
	for _, want := range []string{"bash", "powershell", "batch"} {
		if _, ok := got[want]; !ok {
			t.Errorf("language enum completion missing %q; got %v", want, got)
		}
	}
	if len(items) != 3 {
		t.Errorf("len(items) = %d, want exactly 3 (only the real enum values)", len(items))
	}
}

func TestConfigCompletionMidValueOffersNothing(t *testing.T) {
	text := "host:\n  name: web\n"
	// Cursor is mid-value on the name line itself, not on an empty line
	// or a "key: " line -- nothing to sensibly complete.
	items := configCompletion(text, protocol.Position{Line: 1, Character: 9})
	if len(items) != 0 {
		t.Errorf("mid-value completion = %v, want none", items)
	}
}

// TestScriptCompletionOnlyInsideTemplateExpr is the plan's own rule made
// concrete: completion in a script only means something inside a real
// `{{ }}` expression, against real scope data from examples/lm-test's
// "base" script.
func TestScriptCompletionOnlyInsideTemplateExpr(t *testing.T) {
	w := newTestWorkspace(t)

	outside := Completion(w, "scripts/base.sh", "#!/usr/bin/env bash\necho hello\n", protocol.Position{Line: 1, Character: 11})
	if len(outside) != 0 {
		t.Errorf("completion outside any {{ }} = %v, want none", outside)
	}

	text := "#!/usr/bin/env bash\necho {{ .vars.\n"
	pos := protocol.Position{Line: 1, Character: uint32(len("echo {{ .vars."))}
	inside := Completion(w, "scripts/base.sh", text, pos)
	got := labels(inside)
	if _, ok := got["vars.company"]; !ok {
		t.Errorf("completion inside {{ }} for base.sh missing vars.company (an environment-level var visible everywhere); got %v", got)
	}
	if _, ok := got["host.hostname"]; !ok {
		t.Errorf("completion inside {{ }} missing host.hostname; got %v", got)
	}
	if _, ok := got[`people "employees"`]; !ok {
		t.Errorf("completion inside {{ }} missing the real people source \"employees\"; got %v", got)
	}
}

// TestScriptCompletionFlagsPartialAvailabilityInDetail proves the
// user-visible half of the "available on N of M hosts" rule -- not just
// scriptScope's own numbers (already covered by
// TestScriptScopeUnionsVarsAndFlagsPartialAvailability), but that the
// completion item actually surfaces it in Detail, where an editor shows
// it.
func TestScriptCompletionFlagsPartialAvailabilityInDetail(t *testing.T) {
	w := newTestWorkspace(t)
	text := "#!/usr/bin/env bash\necho {{ .vars.\n"
	pos := protocol.Position{Line: 1, Character: uint32(len("echo {{ .vars."))}
	items := Completion(w, "scripts/base.sh", text, pos)
	got := labels(items)
	dbPW, ok := got["vars.db_admin_pw"]
	if !ok {
		t.Fatalf("completion missing vars.db_admin_pw; got %v", got)
	}
	detail, _ := dbPW.Detail.Get()
	if detail == "" || detail[:9] != "available" {
		t.Fatalf("db_admin_pw.Detail = %q, want it to start with \"available on N/M hosts...\" (it's set only on the database host)", detail)
	}
}

// TestConfigCompletionValidateBlock offers the validator check kinds on a new
// `- ` item inside a step's validate: block, at its real nested depth.
func TestConfigCompletionValidateBlock(t *testing.T) {
	text := "host:\n  name: web\n  steps:\n    - script: harden\n      validate:\n        - \n"
	// Line 5 is "        - " (8 spaces + "- "); cursor right after the dash+space.
	items := configCompletion(text, protocol.Position{Line: 5, Character: 10})
	got := labels(items)
	for _, want := range []string{"service_running", "port_listening", "file_exists", "user_exists", "process_running"} {
		if _, ok := got[want]; !ok {
			t.Errorf("validate-block completion missing %q; got %v", want, got)
		}
	}
	// It must not offer step action keys or type fields here.
	if _, ok := got["script"]; ok {
		t.Errorf("validate-block completion wrongly offered a step action key %q", "script")
	}
}
