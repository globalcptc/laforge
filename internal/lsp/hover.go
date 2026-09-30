package lsp

import (
	"fmt"
	"regexp"
	"strings"

	"go.lsp.dev/protocol"
)

var varTokenRE = regexp.MustCompile(`vars\.([A-Za-z0-9_]+)`)
var fieldTokenRE = regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_]*):`)

// Hover is "What a var holds and which file it came from in the
// environment/network/host cascade" for a script file's `vars.<key>` reference, or the
// schema's own field documentation for a config file's key -- the same
// two real data sources Completion uses, just surfaced on hover instead
// of as completion candidates.
func Hover(w *Workspace, relPath, text string, pos protocol.Position) *protocol.Hover {
	content, _ := w.Snapshot()
	if content == nil {
		return nil
	}
	line := lineAt(text, pos.Line)

	if script := scriptForSourceFile(content, relPath); script != nil {
		key, ok := tokenAt(line, pos.Character, varTokenRE)
		if !ok {
			return nil
		}
		vars, _ := scriptScope(content, script.Name)
		for _, v := range vars {
			if v.Key != key {
				continue
			}
			md := fmt.Sprintf("**vars.%s**\n\n`%s`\n\nFrom: %s", v.Key, v.ExampleValue, v.Source)
			if v.SourceFile != "" {
				md += fmt.Sprintf(" (`%s`)", v.SourceFile)
			}
			if v.SeenIn < v.Total {
				md += fmt.Sprintf("\n\nAvailable on %d/%d hosts running this script.", v.SeenIn, v.Total)
			}
			return &protocol.Hover{Contents: &protocol.MarkupContent{Kind: protocol.MarkupKindMarkdown, Value: md}}
		}
		return nil
	}

	if strings.HasSuffix(relPath, ".yaml") || strings.HasSuffix(relPath, ".yml") {
		// documentAt: same reasoning as configCompletion's own -- a file
		// can hold several `---`-separated objects, and the cursor's own
		// document is the one whose schema actually applies.
		doc, _ := documentAt(text, pos.Line)
		kind, ok := headerKind(doc)
		if !ok {
			return nil
		}
		m := fieldTokenRE.FindStringSubmatch(strings.TrimLeft(line, " \t"))
		if m == nil {
			return nil
		}
		fields, err := schemaFieldsForKind(kind)
		if err != nil {
			return nil
		}
		for _, f := range fields {
			if f.Name != m[1] {
				continue
			}
			md := fmt.Sprintf("**%s**", f.Name)
			if f.Type != "" {
				md += fmt.Sprintf(" `%s`", f.Type)
			}
			if len(f.Enum) > 0 {
				md += fmt.Sprintf("\n\none of: %s", strings.Join(f.Enum, ", "))
			}
			if f.Description != "" {
				md += "\n\n" + f.Description
			}
			return &protocol.Hover{Contents: &protocol.MarkupContent{Kind: protocol.MarkupKindMarkdown, Value: md}}
		}
	}

	return nil
}

// tokenAt finds re's first match on line whose span contains the given
// rune offset, returning its first capture group -- or the whole match,
// if re has no capture group of its own. re must have exactly zero or
// one capture group; this never panics regardless of which.
func tokenAt(line string, character uint32, re *regexp.Regexp) (string, bool) {
	r := []rune(line)
	pos := int(character)
	for _, loc := range re.FindAllStringSubmatchIndex(line, -1) {
		start, end := byteToRune(line, loc[0]), byteToRune(line, loc[1])
		if pos < start || pos > end {
			continue
		}
		groupStart, groupEnd := loc[0], loc[1]
		if len(loc) >= 4 && loc[2] >= 0 && loc[3] >= 0 {
			groupStart, groupEnd = loc[2], loc[3]
		}
		return string(r[byteToRune(line, groupStart):byteToRune(line, groupEnd)]), true
	}
	return "", false
}

func byteToRune(s string, byteIdx int) int {
	return len([]rune(s[:byteIdx]))
}
