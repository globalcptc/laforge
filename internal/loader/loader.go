package loader

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/globalcptc/laforge/internal/schema"
)

// discriminators lists the header keys checked for, in a fixed order so
// "which key is present" errors are deterministic rather than depending on
// Go's randomized map iteration.
var discriminators = []struct {
	key  string
	kind schema.Kind
}{
	{"environment", schema.KindEnvironment},
	{"network", schema.KindNetwork},
	{"host", schema.KindHost},
	{"container", schema.KindContainer},
	{"script", schema.KindScript},
}

// Content is everything a repository resolved to: every object of every
// type, every people source, and every error hit along the way. Loading
// never stops at the first error -- `laforge check`:
// "renders everything... reports pass/fail," which only works if a broken
// file doesn't hide problems in every other file.
type Content struct {
	Environments []Environment
	Networks     []Network
	Hosts        []Host
	Containers   []Container
	Scripts      []Script
	People       []PeopleSource

	Errors []schema.FieldError
}

func (c *Content) addf(file string, line int, format string, args ...interface{}) {
	c.Errors = append(c.Errors, schema.FieldError{
		File:    file,
		Line:    line,
		Message: fmt.Sprintf(format, args...),
	})
}

// Load walks root, validates and decodes every .yaml/.yml file, loads every
// people/*.csv, and runs the cross-file checks a single document's JSON
// Schema can't express. It always returns a Content, even when Errors is
// non-empty -- partial results are useful for tooling (the language server,
// the UI) that wants to show what *did* parse alongside what didn't.
func Load(root string) (*Content, error) {
	compiled, err := schema.Compile()
	if err != nil {
		return nil, fmt.Errorf("compile schemas: %w", err)
	}

	c := &Content{}
	ignore, ignoreErrs := loadIgnore(root)
	c.Errors = append(c.Errors, ignoreErrs...)
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		if rel != "." && ignore.match(filepath.ToSlash(rel), d.IsDir()) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		switch {
		case strings.HasSuffix(path, ".yaml"), strings.HasSuffix(path, ".yml"):
			c.loadYAMLFile(compiled, rel, path)
		case strings.HasSuffix(path, ".csv") && strings.Contains(filepath.ToSlash(rel), "people/"):
			c.loadPeopleCSV(rel, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Fold `extends:` bases into their children before the cross-file checks,
	// so validation (and every downstream consumer) sees objects that already
	// carry their inherited fields -- e.g. the public-ports-subset check must
	// see a host's inherited ports, and required-field presence is judged on
	// the merged object.
	c.resolveExtends()
	c.crossCheck()
	sortErrors(c.Errors)
	return c, nil
}

func (c *Content) loadYAMLFile(compiled *schema.Compiled, rel, path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		c.addf(rel, 0, "read file: %v", err)
		return
	}

	dec := yaml.NewDecoder(strings.NewReader(string(data)))
	for {
		var root yaml.Node
		if err := dec.Decode(&root); err != nil {
			if err == io.EOF {
				break
			}
			c.addf(rel, 0, "parse YAML: %v", err)
			return
		}
		c.loadDocument(compiled, rel, &root)
	}
}

func (c *Content) loadDocument(compiled *schema.Compiled, rel string, root *yaml.Node) {
	var v interface{}
	if err := root.Decode(&v); err != nil {
		c.addf(rel, root.Line, "decode YAML: %v", err)
		return
	}
	// yaml.v3 auto-resolves unquoted ISO8601-looking scalars (start,
	// stop, access.open/close) into time.Time, which the JSON Schema
	// library can't handle (it expects only JSON-native types). Forcing
	// authors to quote every date would reintroduce exactly the YAML
	// footgun the schema-validation approach exists to avoid, so this
	// converts them back to plain RFC3339 strings instead.
	v = sanitizeForSchema(v)
	if v == nil {
		// A stray trailing "---" with nothing after it decodes to an empty
		// document. Not an error -- just nothing to do.
		return
	}
	m, ok := v.(map[string]interface{})
	if !ok {
		c.addf(rel, root.Line, "expected a mapping at the top level of this document, got something else")
		return
	}

	var present []struct {
		key  string
		kind schema.Kind
	}
	for _, d := range discriminators {
		if _, ok := m[d.key]; ok {
			present = append(present, d)
		}
	}
	switch len(present) {
	case 0:
		var keys []string
		for k := range m {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		c.addf(rel, root.Line, "no type header found (expected one of environment/network/host/container/script as a top-level key; found: %s)", strings.Join(keys, ", "))
		return
	case 1:
		// fall through
	default:
		var found []string
		for _, p := range present {
			found = append(found, p.key)
		}
		c.addf(rel, root.Line, "a document may only be one type, found more than one header key: %s", strings.Join(found, ", "))
		return
	}
	kind := present[0].kind

	verr := compiled.Validate(kind, v)
	for _, fe := range schema.TranslateError(rel, verr, root) {
		c.Errors = append(c.Errors, fe)
	}
	if verr != nil {
		// Don't attempt to decode into a typed struct over known-invalid
		// data -- the schema errors above are the useful signal here.
		return
	}

	// Every real field lives inside the type key's own object now (`host:
	// {name: ..., os: ...}`, not `host: <name>` with os/size/... as
	// top-level siblings) -- decoding straight into e.g. Host would try to
	// read `os`/`size`/... off the DOCUMENT's own top level, where they no
	// longer are. A one-field wrapper per kind gets the real object out
	// (root.Decode still runs against the whole document node, the same
	// call as before) before anything downstream ever sees it -- Content's
	// own Environments/Networks/Hosts/... stay exactly the unwrapped types
	// they always were.
	switch kind {
	case schema.KindEnvironment:
		var wrapper struct {
			Environment Environment `yaml:"environment"`
		}
		root.Decode(&wrapper)
		wrapper.Environment.SourceFile = rel
		c.Environments = append(c.Environments, wrapper.Environment)
	case schema.KindNetwork:
		var wrapper struct {
			Network Network `yaml:"network"`
		}
		root.Decode(&wrapper)
		wrapper.Network.SourceFile = rel
		c.Networks = append(c.Networks, wrapper.Network)
	case schema.KindHost:
		var wrapper struct {
			Host Host `yaml:"host"`
		}
		root.Decode(&wrapper)
		wrapper.Host.SourceFile = rel
		c.Hosts = append(c.Hosts, wrapper.Host)
	case schema.KindContainer:
		var wrapper struct {
			Container Container `yaml:"container"`
		}
		root.Decode(&wrapper)
		wrapper.Container.SourceFile = rel
		if wrapper.Container.Compose != "" {
			wrapper.Container.ComposeFile = path.Join(path.Dir(filepath.ToSlash(rel)), filepath.ToSlash(wrapper.Container.Compose))
		}
		c.Containers = append(c.Containers, wrapper.Container)
	case schema.KindScript:
		var wrapper struct {
			Script Script `yaml:"script"`
		}
		root.Decode(&wrapper)
		wrapper.Script.SourceFile = rel
		c.Scripts = append(c.Scripts, wrapper.Script)
	}
}

// sanitizeForSchema recursively converts any time.Time yaml.v3 produced
// during decoding back into an RFC3339 string, leaving everything else
// untouched. See the comment where this is called for why.
func sanitizeForSchema(v interface{}) interface{} {
	switch x := v.(type) {
	case time.Time:
		return x.Format(time.RFC3339)
	case map[string]interface{}:
		for k, val := range x {
			x[k] = sanitizeForSchema(val)
		}
		return x
	case []interface{}:
		for i, val := range x {
			x[i] = sanitizeForSchema(val)
		}
		return x
	default:
		return v
	}
}

func sortErrors(errs []schema.FieldError) {
	sort.Slice(errs, func(i, j int) bool {
		if errs[i].File != errs[j].File {
			return errs[i].File < errs[j].File
		}
		return errs[i].Line < errs[j].Line
	})
}
