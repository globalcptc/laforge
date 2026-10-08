package loader_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/globalcptc/laforge/internal/loader"
)

// writeRepo materializes a small file tree under t.TempDir() from a
// path->content map, so each test's fixture is visible right next to its
// assertions instead of living in a separate examples/ directory.
func writeRepo(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for rel, content := range files {
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func errorMessages(c *loader.Content) []string {
	var out []string
	for _, e := range c.Errors {
		out = append(out, e.Message)
	}
	return out
}

func containsSubstring(msgs []string, substr string) bool {
	for _, m := range msgs {
		if strings.Contains(m, substr) {
			return true
		}
	}
	return false
}

func TestRealExampleRepoLoadsClean(t *testing.T) {
	c, err := loader.Load("../../examples/lm-test")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(c.Errors) != 0 {
		t.Fatalf("expected the real example repo to load clean, got %d errors:\n%s", len(c.Errors), strings.Join(errorMessages(c), "\n"))
	}
	if len(c.Environments) != 1 || len(c.Hosts) != 7 || len(c.Containers) != 1 || len(c.Scripts) != 11 || len(c.People) != 1 {
		t.Fatalf("unexpected counts: %s", c.Summary())
	}
}

func TestHostContainerNameCollisionIsCaught(t *testing.T) {
	root := writeRepo(t, map[string]string{
		"hosts/db.yaml": "host:\n  name: db\n  os: ubuntu22\n  size: small\n  disk: 20\n",
		"hosts/db2.yaml": `container:
  name: db
  image: x
  size: small
`,
	})
	c, _ := loader.Load(root)
	if !containsSubstring(errorMessages(c), "share one namespace") {
		t.Fatalf("expected a namespace-collision error, got: %v", errorMessages(c))
	}
}

func TestDependsOnMissingTargetIsCaught(t *testing.T) {
	root := writeRepo(t, map[string]string{
		"hosts/web.yaml": `host:
  name: web
  os: ubuntu22
  size: small
  disk: 20
  depends_on: [nonexistent-thing]
`,
	})
	c, _ := loader.Load(root)
	if !containsSubstring(errorMessages(c), `"nonexistent-thing"`) {
		t.Fatalf("expected a depends_on error naming the missing target, got: %v", errorMessages(c))
	}
}

func TestPublicPortMustBeSubsetOfHostPorts(t *testing.T) {
	root := writeRepo(t, map[string]string{
		"hosts/wg.yaml": `host:
  name: wireguard
  os: ubuntu22
  size: small
  disk: 20
  ports: { udp: ["51820"] }
  public: { tcp: ["51820"] }
`,
		"networks/vpn.yaml": "network:\n  name: vpn\n  cidr: 10.0.1.0/24\n",
		"env.yaml": `environment:
  name: e
  teams: 1
  networks:
    vpn:
      wireguard:
        - as: wg01
          last_octet: 5
`,
	})
	c, _ := loader.Load(root)
	if !containsSubstring(errorMessages(c), "not in") {
		t.Fatalf("expected a public-port-not-subset error, got: %v", errorMessages(c))
	}
}

func TestPublicPortThatIsDeclaredPasses(t *testing.T) {
	root := writeRepo(t, map[string]string{
		"hosts/wg.yaml": `host:
  name: wireguard
  os: ubuntu22
  size: small
  disk: 20
  ports: { udp: ["51820"] }
  public: { udp: ["51820"] }
`,
		"networks/vpn.yaml": "network:\n  name: vpn\n  cidr: 10.0.1.0/24\n",
		"env.yaml": `environment:
  name: e
  teams: 1
  networks:
    vpn:
      wireguard:
        - as: wg01
          last_octet: 5
`,
	})
	c, _ := loader.Load(root)
	if len(c.Errors) != 0 {
		t.Fatalf("expected no errors when the public port is declared, got: %v", errorMessages(c))
	}
}

// agent-debug on the environment parses into Environment.AgentDebug (default
// false when omitted). It rides to the DB and is baked into each agent binary.
func TestEnvironmentAgentDebugParses(t *testing.T) {
	root := writeRepo(t, map[string]string{
		"networks/lan.yaml": "network:\n  name: lan\n  cidr: 10.0.1.0/24\n",
		"env-on.yaml": `environment:
  name: on
  teams: 1
  agent-debug: true
  networks:
    lan: {}
`,
		"env-off.yaml": `environment:
  name: off
  teams: 1
  networks:
    lan: {}
`,
	})
	c, _ := loader.Load(root)
	if len(c.Errors) != 0 {
		t.Fatalf("unexpected load errors: %v", errorMessages(c))
	}
	got := map[string]bool{}
	for _, e := range c.Environments {
		got[e.Name] = e.AgentDebug
	}
	if !got["on"] {
		t.Error("agent-debug: true did not parse into AgentDebug")
	}
	if got["off"] {
		t.Error("an environment without agent-debug should default to false")
	}
}

// A single public port that falls inside a declared RANGE is a subset -- the
// check compares intervals, not literal token strings. Regression for a bug
// where `ports: ["1-65535"]` + `public: ["3389"]` was wrongly rejected because
// the literal "3389" did not string-match the token "1-65535".
func TestPublicPortWithinDeclaredRangePasses(t *testing.T) {
	root := writeRepo(t, map[string]string{
		"hosts/rdp.yaml": `host:
  name: jump
  os: windows-server-2022
  size: small
  disk: 20
  ports: { tcp: ["1-65535"] }
  public: { tcp: ["3389"] }
`,
		"networks/lan.yaml": "network:\n  name: lan\n  cidr: 10.0.1.0/24\n",
		"env.yaml": `environment:
  name: e
  teams: 1
  networks:
    lan:
      jump:
        - as: j01
          last_octet: 5
`,
	})
	c, _ := loader.Load(root)
	if len(c.Errors) != 0 {
		t.Fatalf("expected no errors for a public port inside a declared range, got: %v", errorMessages(c))
	}
}

// A public port OUTSIDE every declared range is still caught, including when the
// declared ports are themselves ranges.
func TestPublicPortOutsideDeclaredRangeIsCaught(t *testing.T) {
	root := writeRepo(t, map[string]string{
		"hosts/rdp.yaml": `host:
  name: jump
  os: windows-server-2022
  size: small
  disk: 20
  ports: { tcp: ["1-3388", "3390-65535"] }
  public: { tcp: ["3389"] }
`,
		"networks/lan.yaml": "network:\n  name: lan\n  cidr: 10.0.1.0/24\n",
		"env.yaml": `environment:
  name: e
  teams: 1
  networks:
    lan:
      jump:
        - as: j01
          last_octet: 5
`,
	})
	c, _ := loader.Load(root)
	if !containsSubstring(errorMessages(c), "not in") {
		t.Fatalf("expected a public-port-not-subset error for a gap between ranges, got: %v", errorMessages(c))
	}
}

func TestEnvironmentTopologyUnknownObjectIsCaught(t *testing.T) {
	root := writeRepo(t, map[string]string{
		"networks/prod.yaml": "network:\n  name: prod\n  cidr: 10.0.1.0/24\n",
		"env.yaml": `environment:
  name: e
  teams: 1
  networks:
    prod:
      ghost-host:
        - as: g01
          last_octet: 5
`,
	})
	c, _ := loader.Load(root)
	if !containsSubstring(errorMessages(c), `"ghost-host"`) {
		t.Fatalf("expected an error naming the undefined host/container, got: %v", errorMessages(c))
	}
}

func TestDuplicateAsNameWithinEnvironmentIsCaught(t *testing.T) {
	root := writeRepo(t, map[string]string{
		"hosts/kali.yaml":   "host:\n  name: kali\n  os: kali\n  size: large\n  disk: 100\n",
		"networks/vdi.yaml": "network:\n  name: vdi\n  cidr: 10.0.1.0/24\n",
		"env.yaml": `environment:
  name: e
  teams: 1
  networks:
    vdi:
      kali:
        - as: kali01
          last_octet: 1
        - as: kali01
          last_octet: 2
`,
	})
	c, _ := loader.Load(root)
	if !containsSubstring(errorMessages(c), "must be unique within the environment") {
		t.Fatalf("expected a duplicate-as-name error, got: %v", errorMessages(c))
	}
}

func TestPeopleFilterMustResolve(t *testing.T) {
	root := writeRepo(t, map[string]string{
		"people/employees.csv": "username,email\njdoe,jdoe@x.example\n",
		"hosts/db.yaml": `host:
  name: db
  os: ubuntu22
  size: small
  disk: 20
  people:
    - file: employees
      filter: [nosuchuser]
`,
	})
	c, _ := loader.Load(root)
	if !containsSubstring(errorMessages(c), `"nosuchuser"`) {
		t.Fatalf("expected a people-filter error, got: %v", errorMessages(c))
	}
}

func TestPeopleFileMustExist(t *testing.T) {
	root := writeRepo(t, map[string]string{
		"hosts/db.yaml": `host:
  name: db
  os: ubuntu22
  size: small
  disk: 20
  people:
    - file: nosuchfile
`,
	})
	c, _ := loader.Load(root)
	if !containsSubstring(errorMessages(c), `"nosuchfile"`) {
		t.Fatalf("expected a people-file error, got: %v", errorMessages(c))
	}
}

func TestVisibleFromMustReferenceDefinedNetworks(t *testing.T) {
	root := writeRepo(t, map[string]string{
		"networks/vdi.yaml":    "network:\n  name: vdi\n  cidr: 10.0.254.0/24\n",
		"networks/client.yaml": "network:\n  name: client\n  cidr: 10.0.1.0/24\n  visible_from: [vdi, nosuchnet]\n",
	})
	c, _ := loader.Load(root)
	if !containsSubstring(errorMessages(c), `"nosuchnet"`) {
		t.Fatalf("expected an unknown-network error for visible_from, got: %v", errorMessages(c))
	}
	// The valid entry (vdi) must NOT be flagged.
	for _, m := range errorMessages(c) {
		if strings.Contains(m, `"vdi"`) {
			t.Fatalf("valid visible_from entry vdi was wrongly flagged: %v", errorMessages(c))
		}
	}
}

func TestVisibleFromRejectsSelfReference(t *testing.T) {
	root := writeRepo(t, map[string]string{
		"networks/client.yaml": "network:\n  name: client\n  cidr: 10.0.1.0/24\n  visible_from: [client]\n",
	})
	c, _ := loader.Load(root)
	if !containsSubstring(errorMessages(c), "lists itself") {
		t.Fatalf("expected a self-reference error, got: %v", errorMessages(c))
	}
}

func TestDuplicateUsernameAcrossPeopleSourcesIsAllowed(t *testing.T) {
	// A username may now appear in more than one people/*.csv: references are by
	// file (`people: [{file: a}]`), so there's no cross-file ambiguity to guard
	// against, and the old uniqueness rule is gone.
	root := writeRepo(t, map[string]string{
		"people/a.csv": "username,email\njdoe,jdoe@a.example\n",
		"people/b.csv": "username,email\njdoe,jdoe@b.example\n",
	})
	c, _ := loader.Load(root)
	if containsSubstring(errorMessages(c), "also appears in") {
		t.Fatalf("cross-file duplicate usernames should be allowed now, got: %v", errorMessages(c))
	}
}

func TestScriptReferenceMustResolve(t *testing.T) {
	root := writeRepo(t, map[string]string{
		"hosts/web.yaml": `host:
  name: web
  os: ubuntu22
  size: small
  disk: 20
  steps:
    - script: nonexistent-script
`,
	})
	c, _ := loader.Load(root)
	if !containsSubstring(errorMessages(c), `"nonexistent-script"`) {
		t.Fatalf("expected a script-reference error, got: %v", errorMessages(c))
	}
}

func TestExtendsMissingTargetIsCaught(t *testing.T) {
	root := writeRepo(t, map[string]string{
		"hosts/x.yaml": `host:
  name: x
  extends: nonexistent-base
`,
	})
	c, _ := loader.Load(root)
	if !containsSubstring(errorMessages(c), `extends "nonexistent-base"`) {
		t.Fatalf("expected an extends-target error, got: %v", errorMessages(c))
	}
}

func TestExtendsSelfReferenceIsCaught(t *testing.T) {
	root := writeRepo(t, map[string]string{
		"hosts/x.yaml": `host:
  name: x
  extends: x
`,
	})
	c, _ := loader.Load(root)
	if !containsSubstring(errorMessages(c), "extends itself") {
		t.Fatalf("expected a self-extends error, got: %v", errorMessages(c))
	}
}

func TestRegistryValidatorOnLinuxHostIsCaught(t *testing.T) {
	root := writeRepo(t, map[string]string{
		"hosts/web.yaml": `host:
  name: web
  os: ubuntu22
  size: small
  disk: 20
  steps:
    - script: base
      validate:
        - registry: { key: "HKLM\\x", value: "y", expected: "z" }
`,
		"scripts/base.yaml": "script:\n  name: base\n  language: bash\n  source: base.sh\n",
		"scripts/base.sh":   "#!/usr/bin/env bash\necho hi\n",
	})
	c, _ := loader.Load(root)
	if !containsSubstring(errorMessages(c), "Windows-only") {
		t.Fatalf("expected a wrong-platform validator error, got: %v", errorMessages(c))
	}
}

func TestPeopleCSVWithoutUsernameColumnIsRejected(t *testing.T) {
	root := writeRepo(t, map[string]string{
		"people/a.csv": "name,email\njdoe,jdoe@example.com\n",
	})
	c, _ := loader.Load(root)
	if !containsSubstring(errorMessages(c), "no 'username' column") {
		t.Fatalf("expected a missing-username-column error, got: %v", errorMessages(c))
	}
}

func TestTwoDiscriminatorKeysOnOneDocumentIsRejected(t *testing.T) {
	root := writeRepo(t, map[string]string{
		"weird.yaml": "host: a\nnetwork: b\nos: ubuntu22\nsize: small\ndisk: 20\ncidr: 10.0.0.0/24\n",
	})
	c, _ := loader.Load(root)
	if !containsSubstring(errorMessages(c), "more than one header key") {
		t.Fatalf("expected a multiple-discriminator error, got: %v", errorMessages(c))
	}
}

func TestNoDiscriminatorKeyIsRejected(t *testing.T) {
	root := writeRepo(t, map[string]string{
		"weird.yaml": "foo: bar\n",
	})
	c, _ := loader.Load(root)
	if !containsSubstring(errorMessages(c), "no type header found") {
		t.Fatalf("expected a no-type-header error, got: %v", errorMessages(c))
	}
}

// TestScheduleIsASeparateListNotAStepKind is the real re-architecture:
// "steps run once, in order; schedule entries fire independently, on
// their own clock" -- schedule was moved out of steps: entirely into
// its own top-level field, and this proves a real host loads it as
// real, typed data (Host.Schedule), not silently dropped.
func TestScheduleIsASeparateListNotAStepKind(t *testing.T) {
	root := writeRepo(t, map[string]string{
		"hosts/a.yaml": `host:
  name: a
  os: ubuntu22
  size: small
  disk: 20
  schedule:
    - when: Every 30 minutes
      script: reboot
`,
		"scripts/reboot.yaml": "script:\n  name: reboot\n  language: bash\n  source: reboot.sh\n",
		"scripts/reboot.sh":   "#!/usr/bin/env bash\nreboot\n",
	})
	c, err := loader.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Errors) != 0 {
		t.Fatalf("expected no errors, got: %+v", c.Errors)
	}
	if len(c.Hosts) != 1 || len(c.Hosts[0].Schedule) != 1 {
		t.Fatalf("expected exactly one host with one schedule entry, got: %+v", c.Hosts)
	}
	if c.Hosts[0].Schedule[0]["when"] != "Every 30 minutes" {
		t.Errorf("Schedule[0][\"when\"] = %v, want \"Every 30 minutes\"", c.Hosts[0].Schedule[0]["when"])
	}
}

// TestScheduleActionAsStepIsRejected proves the removal side of the
// same re-architecture: `schedule:` is no longer a valid step kind
// inside `steps:` at all.
func TestScheduleActionAsStepIsRejected(t *testing.T) {
	root := writeRepo(t, map[string]string{
		"hosts/a.yaml": `host:
  name: a
  os: ubuntu22
  size: small
  disk: 20
  steps:
    - schedule: { cron: "*/30 * * * *", script: reboot }
`,
	})
	c, _ := loader.Load(root)
	if !containsSubstring(errorMessages(c), "schedule") {
		t.Fatalf("expected a schema error rejecting \"schedule\" as a step, got: %v", errorMessages(c))
	}
}

// TestScheduleInvalidWhenExpressionIsRejected is checkScheduleExpressions
// exercised through a real load, not called directly.
func TestScheduleInvalidWhenExpressionIsRejected(t *testing.T) {
	root := writeRepo(t, map[string]string{
		"hosts/a.yaml": `host:
  name: a
  os: ubuntu22
  size: small
  disk: 20
  schedule:
    - when: every purple minutes
      script: reboot
`,
		"scripts/reboot.yaml": "script:\n  name: reboot\n  language: bash\n  source: reboot.sh\n",
		"scripts/reboot.sh":   "#!/usr/bin/env bash\nreboot\n",
	})
	c, _ := loader.Load(root)
	if !containsSubstring(errorMessages(c), "unrecognized schedule expression") {
		t.Fatalf("expected an unrecognized-schedule-expression error, got: %v", errorMessages(c))
	}
}

// TestScheduleScriptReferenceIsChecked is checkScriptReferences' own new
// loop over Schedule, exercised through a real load.
func TestScheduleScriptReferenceIsChecked(t *testing.T) {
	root := writeRepo(t, map[string]string{
		"hosts/a.yaml": `host:
  name: a
  os: ubuntu22
  size: small
  disk: 20
  schedule:
    - when: Every hour
      script: does-not-exist
`,
	})
	c, _ := loader.Load(root)
	if !containsSubstring(errorMessages(c), `scheduled entry references script "does-not-exist"`) {
		t.Fatalf("expected a missing-script error for the schedule entry, got: %v", errorMessages(c))
	}
}

// TestLaforgeIgnoreKeepsOtherYAMLOutOfContent: paths listed in .laforgeignore
// are not read at all -- which is what lets a Docker Compose project, or any
// other tool's YAML, live in a content repo.
func TestLaforgeIgnoreKeepsOtherYAMLOutOfContent(t *testing.T) {
	files := map[string]string{
		"hosts/web.yaml":                "host:\n  name: web\n  os: ubuntu22\n  size: small\n  disk: 20\n",
		"containers/app/compose.yaml":   "services:\n  web:\n    image: nginx:alpine\n",
		"containers/app/conf/extra.yml": "anything: goes\n",
		"vendor/chart/values.yaml":      "replicas: 2\n",
		"notes/scratch.yaml":            "todo: later\n",
		"deep/down/skipme.yaml":         "x: 1\n",
		"hosts/ignored-host.yaml":       "host:\n  name: ghost\n  os: ubuntu22\n  size: small\n  disk: 20\n",
	}
	c, _ := loader.Load(writeRepo(t, files))
	if n := len(c.Errors); n != 5 {
		t.Fatalf("without an ignore file every stray YAML is an error; got %d: %v", n, errorMessages(c))
	}

	files[".laforgeignore"] = "# compose projects and vendored config\ncontainers/app/\n/vendor\nnotes/*.yaml\nskipme.yaml\n**/ignored-*.yaml\n"
	c, _ = loader.Load(writeRepo(t, files))
	if len(c.Errors) != 0 {
		t.Fatalf("ignored paths should not be read; got: %v", c.Errors)
	}
	if len(c.Hosts) != 1 || c.Hosts[0].Name != "web" {
		t.Errorf("hosts = %+v, want only web (ignored-host.yaml is ignored)", c.Hosts)
	}

	files[".laforgeignore"] = "!hosts/web.yaml\n[bad\n"
	c, _ = loader.Load(writeRepo(t, files))
	msgs := errorMessages(c)
	if !containsSubstring(msgs, "negated patterns") || !containsSubstring(msgs, "not a valid pattern") {
		t.Errorf("expected errors for the unsupported and malformed patterns, got: %v", msgs)
	}
}

// TestComposeContainer covers the content rules for a container that runs a
// Compose project: a path relative to its own file, one of image/compose, and a
// pointer to .laforgeignore when the project's YAML gets read as content.
func TestComposeContainer(t *testing.T) {
	good := map[string]string{
		".laforgeignore":                "containers/flaky/\n",
		"containers/flaky.yaml":         "container:\n  name: flaky\n  compose: flaky/compose.yaml\n  size: small\n  disk: 60\n",
		"containers/flaky/compose.yaml": "services:\n  web:\n    image: nginx:alpine\n",
		"containers/child.yaml":         "container:\n  name: child\n  extends: flaky\n",
		"containers/plain.yaml":         "container:\n  name: plain\n  extends: flaky\n  image: nginx:alpine\n",
	}
	c, _ := loader.Load(writeRepo(t, good))
	if len(c.Errors) != 0 {
		t.Fatalf("should load clean, got: %v", c.Errors)
	}
	byName := map[string]loader.Container{}
	for _, ct := range c.Containers {
		byName[ct.Name] = ct
	}
	if got := byName["flaky"].ComposeFile; got != "containers/flaky/compose.yaml" {
		t.Errorf("ComposeFile = %q, want it resolved from the container's own file", got)
	}
	if ch := byName["child"]; ch.ComposeFile != "containers/flaky/compose.yaml" || ch.Disk != 60 {
		t.Errorf("child should inherit the project and disk, got %+v", ch)
	}
	if p := byName["plain"]; p.Image != "nginx:alpine" || p.Compose != "" {
		t.Errorf("a child that sets image replaces the base's compose, got image=%q compose=%q", p.Image, p.Compose)
	}

	for name, tc := range map[string]struct{ container, want string }{
		"both":          {"container:\n  name: x\n  image: a\n  compose: flaky/compose.yaml\n  size: small\n", ""},
		"neither":       {"container:\n  name: x\n  size: small\n", ""},
		"env":           {"container:\n  name: x\n  compose: flaky/compose.yaml\n  size: small\n  env: { A: b }\n", "put them in the compose file"},
		"escapes repo":  {"container:\n  name: x\n  compose: ../../elsewhere/compose.yaml\n  size: small\n", "must be a path inside the content repo"},
		"disk on image": {"container:\n  name: x\n  extends: base\n  disk: 20\n---\ncontainer:\n  name: base\n  image: a\n  size: small\n", "only applies with compose"},
		"no directory":  {"container:\n  name: x\n  compose: compose.yaml\n  size: small\n", "must be in its own directory"},
	} {
		c, _ := loader.Load(writeRepo(t, map[string]string{".laforgeignore": "containers/flaky/\n", "containers/x.yaml": tc.container}))
		if len(c.Errors) == 0 || !containsSubstring(errorMessages(c), tc.want) {
			t.Errorf("%s: expected an error mentioning %q, got: %v", name, tc.want, errorMessages(c))
		}
	}

	delete(good, ".laforgeignore")
	c, _ = loader.Load(writeRepo(t, good))
	if !containsSubstring(errorMessages(c), `add "containers/flaky/" to .laforgeignore`) {
		t.Errorf("expected a pointer to .laforgeignore, got: %v", errorMessages(c))
	}
}
