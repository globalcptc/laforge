package schema

import (
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v5"
	"gopkg.in/yaml.v3"
)

// FieldError is one problem with one file, resolved back to a source line —
// this is the whole point of the "validator with line-numbered
// errors": an author sees exactly where to look, not a JSON pointer.
type FieldError struct {
	File    string
	Line    int
	Column  int
	Pointer string // the raw JSON pointer, kept for anyone who wants it
	Message string
}

// TranslateError walks a *jsonschema.ValidationError tree (or passes through
// any other error unchanged) into flat, line-numbered FieldErrors. root must
// be the yaml.Node this validation ran against — decoded from the same
// bytes, not re-parsed, so line numbers actually line up.
func TranslateError(file string, err error, root *yaml.Node) []FieldError {
	if err == nil {
		return nil
	}
	ve, ok := err.(*jsonschema.ValidationError)
	if !ok {
		return []FieldError{{File: file, Message: err.Error()}}
	}

	var out []FieldError
	for _, leaf := range collectLeaves(ve) {
		node := resolvePointer(root, adjustPointerForMessage(leaf.instanceLocation, leaf.message))
		fe := FieldError{
			File:    file,
			Pointer: leaf.instanceLocation,
			Message: leaf.message,
		}
		if node != nil {
			fe.Line = node.Line
			fe.Column = node.Column
		}
		out = append(out, fe)
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].Line != out[j].Line {
			return out[i].Line < out[j].Line
		}
		return out[i].Pointer < out[j].Pointer
	})
	return dedupe(out)
}

type leaf struct {
	instanceLocation string
	message          string
}

var missingPropRe = regexp.MustCompile(`^missing properties: (.+)$`)
var quotedRe = regexp.MustCompile(`'([^']+)'`)
var additionalPropRe = regexp.MustCompile(`^additionalProperties '([^']+)' not allowed$`)

// adjustPointerForMessage handles the one case where the library's
// InstanceLocation points at the containing object rather than the actual
// problem: "additionalProperties 'x' not allowed" names the disallowed key
// only in the message text, not the pointer. Appending it gets the line
// resolver from "the top of this object" to "right where the typo is"
// (technically the typo'd key's *value*, since resolvePointer returns
// Content[i+1] for a mapping segment — one line off from the key itself
// when the value is a nested block, close enough to find it immediately,
// worth tightening to point at the key token itself later).
func adjustPointerForMessage(pointer, message string) string {
	if m := additionalPropRe.FindStringSubmatch(message); m != nil {
		return pointer + "/" + m[1]
	}
	return pointer
}

// collectLeaves walks the error tree and returns the useful, specific
// messages. Two things happen along the way:
//
//   - A oneOf failure where NO branch matched produces one cause per branch,
//     each just "missing properties: 'x'" — 15 near-identical lines for a
//     step with no action key at all. Those get collapsed into one message
//     naming every valid key, since that's what's actually useful to read.
//   - A oneOf failure where MORE THAN ONE branch matched (e.g. a step with
//     two action keys) already comes back from the library as a single
//     clean leaf ("valid against schemas at indexes 0 and 1") — nothing to
//     do there, it falls through the ordinary leaf case below.
func collectLeaves(err *jsonschema.ValidationError) []leaf {
	if err.Message == "oneOf failed" && allUnmatchedRequired(err.Causes) {
		var keys []string
		for _, c := range err.Causes {
			m := missingPropRe.FindStringSubmatch(c.Message)
			if m == nil {
				continue
			}
			for _, q := range quotedRe.FindAllStringSubmatch(m[1], -1) {
				keys = append(keys, q[1])
			}
		}
		sort.Strings(keys)
		return []leaf{{
			instanceLocation: err.InstanceLocation,
			message:          "must specify exactly one of: " + strings.Join(keys, ", ") + " (found none)",
		}}
	}

	if len(err.Causes) == 0 {
		return []leaf{{instanceLocation: err.InstanceLocation, message: err.Message}}
	}

	var out []leaf
	for _, c := range err.Causes {
		out = append(out, collectLeaves(c)...)
	}
	return out
}

func allUnmatchedRequired(causes []*jsonschema.ValidationError) bool {
	if len(causes) < 2 {
		return false
	}
	for _, c := range causes {
		if len(c.Causes) != 0 || !missingPropRe.MatchString(c.Message) {
			return false
		}
	}
	return true
}

func dedupe(errs []FieldError) []FieldError {
	seen := make(map[string]bool)
	var out []FieldError
	for _, e := range errs {
		key := e.File + "|" + e.Pointer + "|" + e.Message
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, e)
	}
	return out
}

// resolvePointer walks an RFC 6901 JSON pointer (e.g. "/steps/0/change_perms")
// against a yaml.Node tree decoded from the same document a jsonschema
// error's InstanceLocation refers to, returning the node at that path — or
// the deepest node it could reach, which is exactly right for a "missing
// required property" error: the pointer names the object that's missing the
// field, not the (nonexistent) field itself.
func resolvePointer(root *yaml.Node, pointer string) *yaml.Node {
	if root == nil {
		return nil
	}
	current := root
	if current.Kind == yaml.DocumentNode && len(current.Content) > 0 {
		current = current.Content[0]
	}
	if pointer == "" {
		return current
	}
	segments := strings.Split(strings.TrimPrefix(pointer, "/"), "/")
	for _, seg := range segments {
		seg = unescapePointerSegment(seg)
		switch current.Kind {
		case yaml.MappingNode:
			found := false
			for i := 0; i+1 < len(current.Content); i += 2 {
				if current.Content[i].Value == seg {
					current = current.Content[i+1]
					found = true
					break
				}
			}
			if !found {
				return current // stop at the deepest resolvable ancestor
			}
		case yaml.SequenceNode:
			idx, err := strconv.Atoi(seg)
			if err != nil || idx < 0 || idx >= len(current.Content) {
				return current
			}
			current = current.Content[idx]
		default:
			return current
		}
	}
	return current
}

func unescapePointerSegment(s string) string {
	s = strings.ReplaceAll(s, "~1", "/")
	s = strings.ReplaceAll(s, "~0", "~")
	return s
}
