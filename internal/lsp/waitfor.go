package lsp

import (
	"regexp"
	"sort"
	"strings"

	"go.lsp.dev/protocol"

	"github.com/globalcptc/laforge/internal/loader"
)

// waitForValueRE matches a cursor sitting in the value of one of wait_for's
// own keys -- block style (`host: bff-d|`), a list entry (`- host: |`), or
// flow style (`wait_for: { host: dc, milestone: |`).
var waitForValueRE = regexp.MustCompile(`(?:^|[{,\s])(host|milestone|step):\s*([A-Za-z0-9_-]*)$`)

var waitForHostRE = regexp.MustCompile(`(?:^|[{,\s])host:\s*([A-Za-z0-9_-]+)`)

// waitForValueCompletion offers real values inside a step's `wait_for:`:
// every host/container name for `host:`, and -- once `host:` is set beside
// it -- that host's declared milestones for `milestone:` or its script steps
// for `step:`. These are exactly the values the loader's wait_for check will
// accept, so a volunteer never has to open the target host's file to find
// the milestone name. handled is false when the cursor isn't in such a value.
func waitForValueCompletion(content *loader.Content, text string, pos protocol.Position) ([]protocol.CompletionItem, bool) {
	doc, docStart := documentAt(text, pos.Line)
	localLine := int(pos.Line) - docStart
	lines := splitLines(doc)
	if localLine < 0 || localLine >= len(lines) {
		return nil, false
	}
	line := lines[localLine]
	prefix := runePrefix(line, pos.Character)
	m := waitForValueRE.FindStringSubmatch(prefix)
	if m == nil {
		return nil, false
	}
	key, typed := m[1], m[2]

	var host string
	if i := strings.Index(prefix, "wait_for:"); i >= 0 {
		// Flow style: everything is on this line.
		if hm := waitForHostRE.FindStringSubmatch(line[i:]); hm != nil {
			host = hm[1]
		}
	} else {
		keyCol := strings.Index(line, key+":")
		if keyCol < 0 || !underWaitFor(lines, localLine, keyCol) {
			return nil, false
		}
		host = blockSiblingHost(lines, localLine, keyCol)
	}

	var values []string
	switch key {
	case "host":
		for _, h := range content.Hosts {
			values = append(values, h.Name)
		}
		for _, c := range content.Containers {
			values = append(values, c.Name)
		}
	case "milestone", "step":
		steps := targetSteps(content, host)
		seen := map[string]bool{}
		for _, st := range steps {
			v := st.Milestone()
			if key == "step" {
				v, _ = st["script"].(string)
			}
			if v != "" && !seen[v] {
				seen[v] = true
				values = append(values, v)
			}
		}
	}
	sort.Strings(values)
	items := make([]protocol.CompletionItem, 0, len(values))
	for _, v := range values {
		if typed != "" && !strings.HasPrefix(v, typed) {
			continue
		}
		detail := "host or container"
		if key != "host" {
			detail = key + " on " + host
		}
		items = append(items, protocol.CompletionItem{Label: v, Kind: protocol.CompletionItemKindValue, Detail: protocol.NewOptional(detail)})
	}
	return items, true
}

// underWaitFor reports whether the key at keyCol on line idx belongs to a
// `wait_for:` mapping, directly or as one entry of a wait_for list.
func underWaitFor(lines []string, idx, keyCol int) bool {
	col := keyCol
	for i := idx - 1; i >= 0; i-- {
		l := lines[i]
		if strings.TrimSpace(l) == "" {
			continue
		}
		ind := leadingSpaceCount(l)
		trimmed := strings.TrimSpace(l)
		if ind >= col {
			continue // a sibling key of the same mapping, or deeper
		}
		if strings.HasPrefix(trimmed, "- ") && ind+2 == col {
			// The `- ` that opens this list entry: wait_for sits above it.
			col = ind
			continue
		}
		return strings.HasPrefix(trimmed, "wait_for:")
	}
	return false
}

// blockSiblingHost finds the `host:` value in the same block-style mapping as
// the key at keyCol on line idx (above or below it), or "".
func blockSiblingHost(lines []string, idx, keyCol int) string {
	check := func(l string) (string, bool) {
		if hm := waitForHostRE.FindStringSubmatch(l); hm != nil && strings.Index(l, "host:") == keyCol {
			return hm[1], true
		}
		return "", false
	}
	for i := idx - 1; i >= 0; i-- {
		if h, ok := check(lines[i]); ok {
			return h
		}
		if strings.TrimSpace(lines[i]) != "" && leadingSpaceCount(lines[i]) < keyCol {
			break
		}
	}
	for i := idx + 1; i < len(lines); i++ {
		l := lines[i]
		if strings.TrimSpace(l) == "" {
			continue
		}
		if leadingSpaceCount(l) < keyCol || strings.HasPrefix(strings.TrimSpace(l), "- ") {
			break
		}
		if h, ok := check(l); ok {
			return h
		}
	}
	return ""
}

func targetSteps(content *loader.Content, name string) []loader.Step {
	for _, h := range content.Hosts {
		if h.Name == name {
			return h.Steps
		}
	}
	for _, c := range content.Containers {
		if c.Name == name {
			return c.Steps
		}
	}
	return nil
}
