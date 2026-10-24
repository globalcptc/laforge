// Package hclconvert is the "HCL to YAML converter: reads the
// existing content repo and emits the new format, including hosts,
// networks, scripts, identities, and each `envs/*` environment. Scripts
// carry over untouched; only the surrounding definitions change. Flags
// anything it cannot translate rather than guessing."
//
// This file is the generic half: walk every .laforge file with a real HCL
// parser (hashicorp/hcl/v2 -- the same library the old server itself uses)
// into untyped blocks, without assuming a schema up front. convert.go does
// the actual old-shape-to-new-shape mapping on top of that.
package hclconvert

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
)

// Block is one untyped HCL block: `<Type> "label1" "label2" { ... }`.
// Everything the old system's real content actually uses turns out to
// need at most one label,
// but this doesn't assume that.
type Block struct {
	Type       string
	Labels     []string
	Attrs      map[string]interface{}
	Blocks     []*Block
	SourceFile string // relative to the repo root
}

// Note is one thing the converter could not translate automatically, or
// translated with a judgment call worth a human's eyes -- "flags anything
// it cannot translate rather than guessing."
type Note struct {
	File     string
	Severity string // "info" (a deliberate, documented simplification) or "warn" (needs review)
	Message  string
}

// ParseRepo walks every .laforge file under root and returns every
// top-level block found, plus parse-level notes (a file that fails to
// parse at all is a Note, not a fatal error -- one bad file shouldn't stop
// the other 2,800). `include { path = ... }` blocks are the old system's
// own file-inclusion mechanism; since this walks the whole tree directly
// rather than resolving includes, they carry no information for a
// converter and are dropped silently rather than flagged.
func ParseRepo(root string) ([]*Block, []Note, error) {
	var blocks []*Block
	var notes []Note

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".laforge") {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		fileBlocks, fileNotes := parseFile(path, rel)
		blocks = append(blocks, fileBlocks...)
		notes = append(notes, fileNotes...)
		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	sort.Slice(blocks, func(i, j int) bool {
		if blocks[i].SourceFile != blocks[j].SourceFile {
			return blocks[i].SourceFile < blocks[j].SourceFile
		}
		return blocks[i].Type < blocks[j].Type
	})
	return blocks, notes, nil
}

func parseFile(path, rel string) ([]*Block, []Note) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, []Note{{File: rel, Severity: "warn", Message: fmt.Sprintf("could not read file: %v", err)}}
	}

	parser := hclparse.NewParser()
	f, diags := parser.ParseHCL(src, path)
	if diags.HasErrors() {
		return nil, []Note{{File: rel, Severity: "warn", Message: fmt.Sprintf("HCL parse error, file skipped entirely: %s", diags.Error())}}
	}

	body, ok := f.Body.(*hclsyntax.Body)
	if !ok {
		return nil, []Note{{File: rel, Severity: "warn", Message: "file body is not hclsyntax (unexpected), skipped"}}
	}

	var blocks []*Block
	var notes []Note
	for _, b := range body.Blocks {
		if b.Type == "include" {
			continue // see ParseRepo's doc comment
		}
		blk, blkNotes := walkBlock(b, rel)
		blocks = append(blocks, blk)
		notes = append(notes, blkNotes...)
	}
	return blocks, notes
}

func walkBlock(b *hclsyntax.Block, sourceFile string) (*Block, []Note) {
	blk := &Block{
		Type:       b.Type,
		Labels:     append([]string{}, b.Labels...),
		Attrs:      make(map[string]interface{}),
		SourceFile: sourceFile,
	}
	var notes []Note

	for name, attr := range b.Body.Attributes {
		v, diags := attr.Expr.Value(nil)
		if diags.HasErrors() {
			notes = append(notes, Note{File: sourceFile, Severity: "warn",
				Message: fmt.Sprintf("%s %v: could not evaluate attribute %q (%s), left out of conversion", b.Type, b.Labels, name, diags.Error())})
			continue
		}
		blk.Attrs[name] = ctyToGo(v)
	}

	for _, nested := range b.Body.Blocks {
		nb, nn := walkBlock(nested, sourceFile)
		blk.Blocks = append(blk.Blocks, nb)
		notes = append(notes, nn...)
	}

	return blk, notes
}

// ctyToGo converts an already-evaluated cty.Value into plain Go types
// (string, bool, float64, []interface{}, map[string]interface{}, nil) --
// everything yaml.Marshal and this package's own conversion logic already
// know how to work with, so nothing downstream needs to know cty exists.
func ctyToGo(v cty.Value) interface{} {
	if v.IsNull() {
		return nil
	}
	t := v.Type()
	switch {
	case t == cty.String:
		return v.AsString()
	case t == cty.Bool:
		return v.True()
	case t == cty.Number:
		f, _ := v.AsBigFloat().Float64()
		return f
	case t.IsTupleType() || t.IsListType() || t.IsSetType():
		var out []interface{}
		for it := v.ElementIterator(); it.Next(); {
			_, ev := it.Element()
			out = append(out, ctyToGo(ev))
		}
		return out
	case t.IsObjectType() || t.IsMapType():
		out := make(map[string]interface{})
		for it := v.ElementIterator(); it.Next(); {
			k, ev := it.Element()
			out[k.AsString()] = ctyToGo(ev)
		}
		return out
	default:
		return fmt.Sprintf("%v", v)
	}
}

// --- small typed-accessor helpers used throughout convert.go ---

func (b *Block) str(name string) string {
	if v, ok := b.Attrs[name].(string); ok {
		return v
	}
	return ""
}

func (b *Block) boolean(name string) bool {
	v, _ := b.Attrs[name].(bool)
	return v
}

func (b *Block) number(name string) (float64, bool) {
	v, ok := b.Attrs[name].(float64)
	return v, ok
}

func (b *Block) strList(name string) []string {
	raw, _ := b.Attrs[name].([]interface{})
	var out []string
	for _, v := range raw {
		if s, ok := v.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func (b *Block) strMap(name string) map[string]string {
	raw, _ := b.Attrs[name].(map[string]interface{})
	if len(raw) == 0 {
		return nil
	}
	out := make(map[string]string, len(raw))
	for k, v := range raw {
		out[k] = fmt.Sprintf("%v", v)
	}
	return out
}

func (b *Block) childrenOfType(typ string) []*Block {
	var out []*Block
	for _, c := range b.Blocks {
		if c.Type == typ {
			out = append(out, c)
		}
	}
	return out
}

func (b *Block) label() string {
	if len(b.Labels) > 0 {
		return b.Labels[0]
	}
	return ""
}
