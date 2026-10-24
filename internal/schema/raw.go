package schema

import "fmt"

// kindToFile is kindToID's own mapping (schema.go), inverted the other
// way -- from Kind to the embedded file that actually defines it, for
// callers that want the raw JSON Schema document itself (tooling:
// internal/lsp's completion, not validation, which goes through Compile/
// Validate instead).
var kindToFile = map[Kind]string{
	KindEnvironment: "schemas/environment.schema.json",
	KindNetwork:     "schemas/network.schema.json",
	KindHost:        "schemas/host.schema.json",
	KindContainer:   "schemas/container.schema.json",
	KindScript:      "schemas/script.schema.json",
}

// RawSchema returns kind's own embedded JSON Schema document, unparsed.
// Exists for tooling that wants the schema's own shape (field names,
// enums, descriptions) rather than a compiled validator -- see
// internal/lsp/schemadoc.go, which is the one real caller.
func RawSchema(kind Kind) ([]byte, error) {
	path, ok := kindToFile[kind]
	if !ok {
		return nil, fmt.Errorf("no schema registered for kind %q", kind)
	}
	return schemaFS.ReadFile(path)
}

// RawCommonSchema returns common.schema.json, the shared $defs every
// other schema's $refs resolve against (tagsMap, findingsList, and so
// on) -- needed alongside RawSchema by anything that wants to follow a
// "$ref": "common.json#/$defs/..." the way internal/lsp's completion
// does.
func RawCommonSchema() ([]byte, error) {
	return schemaFS.ReadFile("schemas/common.schema.json")
}

// Kinds lists every content Kind with a schema, in a stable order.
func Kinds() []Kind {
	return []Kind{KindEnvironment, KindNetwork, KindHost, KindContainer, KindScript}
}
