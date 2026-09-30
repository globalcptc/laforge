// Package schema compiles and applies the JSON Schemas in ./schemas against
// decoded YAML content, a "validator
// with line-numbered errors".
package schema

import (
	"bytes"
	"embed"
	"fmt"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

//go:embed schemas/*.json
var schemaFS embed.FS

// Kind is one of the six type-discriminator header keys a content file can
// start with. "people" isn't here — people sources are CSV, validated
// separately (see internal/loader).
type Kind string

const (
	KindEnvironment Kind = "environment"
	KindNetwork     Kind = "network"
	KindHost        Kind = "host"
	KindContainer   Kind = "container"
	KindScript      Kind = "script"
)

// Every schema file's own $id, matching what's written in the .schema.json
// files. $refs like "common.json#/$defs/tagsMap" resolve *relative to the
// referencing document's own $id*, not relative to a filename — so schemas
// must be registered and compiled under these exact URLs, not bare
// filenames, or cross-file $refs silently fail to resolve.
const (
	idCommon      = "https://laforge.dev/schema/common.json"
	idEnvironment = "https://laforge.dev/schema/environment.json"
	idNetwork     = "https://laforge.dev/schema/network.json"
	idHost        = "https://laforge.dev/schema/host.json"
	idContainer   = "https://laforge.dev/schema/container.json"
	idScript      = "https://laforge.dev/schema/script.json"
)

var schemaFiles = map[string]string{
	"schemas/common.schema.json":      idCommon,
	"schemas/environment.schema.json": idEnvironment,
	"schemas/network.schema.json":     idNetwork,
	"schemas/host.schema.json":        idHost,
	"schemas/container.schema.json":   idContainer,
	"schemas/script.schema.json":      idScript,
}

var kindToID = map[Kind]string{
	KindEnvironment: idEnvironment,
	KindNetwork:     idNetwork,
	KindHost:        idHost,
	KindContainer:   idContainer,
	KindScript:      idScript,
}

type Compiled struct {
	schemas map[Kind]*jsonschema.Schema
}

// Compile loads every schema file from the embedded FS, registers each
// under its own $id (see the const block above for why), and resolves all
// cross-file $refs into common.json. Called once at process start.
func Compile() (*Compiled, error) {
	c := jsonschema.NewCompiler()
	c.Draft = jsonschema.Draft2020

	for path, id := range schemaFiles {
		b, err := schemaFS.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", path, err)
		}
		if err := c.AddResource(id, bytes.NewReader(b)); err != nil {
			return nil, fmt.Errorf("add resource %s: %w", id, err)
		}
	}

	compiled := &Compiled{schemas: make(map[Kind]*jsonschema.Schema)}
	for kind, id := range kindToID {
		s, err := c.Compile(id)
		if err != nil {
			return nil, fmt.Errorf("compile %s: %w", id, err)
		}
		compiled.schemas[kind] = s
	}
	return compiled, nil
}

// Validate runs the schema for kind against an already-decoded value (the
// output of yaml.Node.Decode(&v) — see internal/loader), returning the raw
// library error for the caller to translate into line-numbered FieldErrors.
func (c *Compiled) Validate(kind Kind, v interface{}) error {
	s, ok := c.schemas[kind]
	if !ok {
		return fmt.Errorf("no schema registered for kind %q", kind)
	}
	return s.Validate(v)
}
