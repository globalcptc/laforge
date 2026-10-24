package schema_test

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/globalcptc/laforge/internal/schema"
)

// validateHost is a small test helper: parses src as YAML (keeping the
// yaml.Node tree for line resolution), decodes it, validates against the
// host schema, and returns the translated line-numbered errors.
func validateHost(t *testing.T, src string) []schema.FieldError {
	t.Helper()
	c, err := schema.Compile()
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	var root yaml.Node
	if err := yaml.Unmarshal([]byte(src), &root); err != nil {
		t.Fatalf("yaml.Unmarshal: %v", err)
	}
	var v interface{}
	if err := root.Decode(&v); err != nil {
		t.Fatalf("root.Decode: %v", err)
	}
	verr := c.Validate(schema.KindHost, v)
	return schema.TranslateError("host.yaml", verr, &root)
}

func TestValidHostPassesClean(t *testing.T) {
	errs := validateHost(t, `host:
  name: web01
  os: ubuntu22
  size: small
  disk: 40
  ports:
    tcp: ["80", "443"]
  steps:
    - script: base
`)
	if len(errs) != 0 {
		t.Fatalf("expected zero errors on valid content, got %+v", errs)
	}
}

func TestStepWithNoActionKeyIsCollapsedToOneMessage(t *testing.T) {
	errs := validateHost(t, `host:
  name: web01
  os: ubuntu22
  size: small
  disk: 40
  steps:
    - script: base
    - validate:
        - user_exists: bob
`)
	if len(errs) != 1 {
		t.Fatalf("expected exactly one collapsed error (15 raw oneOf branches should become 1), got %d: %+v", len(errs), errs)
	}
	e := errs[0]
	if e.Line != 8 {
		t.Errorf("expected the error on line 8 (the '- validate:' step), got line %d", e.Line)
	}
	if !strings.Contains(e.Message, "must specify exactly one of") {
		t.Errorf("expected a collapsed 'must specify exactly one of' message, got %q", e.Message)
	}
	if !strings.Contains(e.Message, "script") || !strings.Contains(e.Message, "reboot") {
		t.Errorf("expected the collapsed message to list valid action keys, got %q", e.Message)
	}
}

func TestStepWithTwoActionKeysIsRejected(t *testing.T) {
	errs := validateHost(t, `host:
  name: web01
  os: ubuntu22
  size: small
  disk: 40
  steps:
    - script: base
      run: echo hi
`)
	if len(errs) != 1 {
		t.Fatalf("expected exactly one error, got %d: %+v", len(errs), errs)
	}
	if errs[0].Line != 7 {
		t.Errorf("expected the error on line 7 (where the step starts), got line %d", errs[0].Line)
	}
}

func TestFindingSeverityOutOfRangePointsAtTheValue(t *testing.T) {
	errs := validateHost(t, `host:
  name: web01
  os: ubuntu22
  size: small
  disk: 40
  findings:
    - severity: 9
      difficulty: 2
      description: bad severity
`)
	if len(errs) != 1 {
		t.Fatalf("expected exactly one error, got %d: %+v", len(errs), errs)
	}
	e := errs[0]
	if e.Line != 7 {
		t.Errorf("expected the error on line 7 (the 'severity: 9' line), got line %d", e.Line)
	}
	if !strings.Contains(e.Message, "<= 5") {
		t.Errorf("expected a range message, got %q", e.Message)
	}
}

func TestUnknownFieldIsCaughtNearTheTypo(t *testing.T) {
	errs := validateHost(t, `host:
  name: web01
  os: ubuntu22
  size: small
  disk: 40
  tagz:
    role: db
`)
	if len(errs) != 1 {
		t.Fatalf("expected exactly one error, got %d: %+v", len(errs), errs)
	}
	e := errs[0]
	// Resolves to the typo'd key's value (line 7), one line below the key
	// itself (line 6) -- see the comment on adjustPointerForMessage for why
	// that's the current tradeoff rather than line 1 (the whole document).
	if e.Line != 7 {
		t.Errorf("expected the error near line 6-7 (the 'tagz:' typo), got line %d", e.Line)
	}
	if !strings.Contains(e.Message, "tagz") {
		t.Errorf("expected the message to name the bad field, got %q", e.Message)
	}
}

func TestMissingRequiredFieldIsRejected(t *testing.T) {
	errs := validateHost(t, `host:
  name: web01
  os: ubuntu22
  size: small
`)
	if len(errs) == 0 {
		t.Fatal("expected an error for missing required 'disk' field, got none")
	}
	found := false
	for _, e := range errs {
		if strings.Contains(e.Message, "disk") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected an error mentioning 'disk', got %+v", errs)
	}
}

func TestContainerRejectsDiskField(t *testing.T) {
	c, err := schema.Compile()
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	var root yaml.Node
	src := `container:
  name: scoreboard
  image: laforge/scoreboard:latest
  size: small
  disk: 40
`
	if err := yaml.Unmarshal([]byte(src), &root); err != nil {
		t.Fatalf("yaml.Unmarshal: %v", err)
	}
	var v interface{}
	root.Decode(&v)
	verr := c.Validate(schema.KindContainer, v)
	if verr == nil {
		t.Fatal("expected an error: containers don't have a disk field")
	}
}

func TestRegistryValidatorRequiresAllThreeFields(t *testing.T) {
	errs := validateHost(t, `host:
  name: dc01
  os: windows-server-2022
  size: medium
  disk: 100
  steps:
    - script: harden
      validate:
        - registry: { key: "HKLM\\Foo", value: "Bar" }
`)
	if len(errs) == 0 {
		t.Fatal("expected an error: registry validator is missing 'expected'")
	}
}
