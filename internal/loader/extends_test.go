package loader_test

import (
	"testing"

	"github.com/globalcptc/laforge/internal/loader"
)

func scriptDoc(name string) string {
	return "script:\n  name: " + name + "\n  language: bash\n  source: " + name + ".sh\n"
}

func simpleHost(name string) string {
	return "host:\n  name: " + name + "\n  os: ubuntu22\n  size: small\n  disk: 20\n"
}

// findHost returns the loaded host by name, failing if it's absent.
func findHost(t *testing.T, c *loader.Content, name string) loader.Host {
	t.Helper()
	for _, h := range c.Hosts {
		if h.Name == name {
			return h
		}
	}
	t.Fatalf("host %q not found among %d hosts", name, len(c.Hosts))
	return loader.Host{}
}

func TestExtendsDeepMergeHost(t *testing.T) {
	root := writeRepo(t, map[string]string{
		"hosts/base.yaml": `host:
  name: base
  os: ubuntu22
  size: small
  disk: 20
  depends_on: [dc]
  vars:
    role: base
    keep: yes
  tags:
    tier: web
  steps:
    - script: base-setup
`,
		"hosts/web.yaml": `host:
  name: web
  extends: base
  disk: 40
  depends_on: [dc, db]
  vars:
    role: web
  tags:
    env: prod
  steps:
    - script: web-setup
`,
		"scripts/base-setup.yaml": scriptDoc("base-setup"),
		"scripts/web-setup.yaml":  scriptDoc("web-setup"),
		"hosts/dc.yaml":           simpleHost("dc"),
		"hosts/db.yaml":           simpleHost("db"),
	})
	c, _ := loader.Load(root)
	if msgs := errorMessages(c); len(msgs) != 0 {
		t.Fatalf("unexpected errors: %v", msgs)
	}
	web := findHost(t, c, "web")

	// Scalars: inherited when unset (os/size), child wins when set (disk).
	if web.OS != "ubuntu22" || web.Size != "small" {
		t.Errorf("expected inherited os/size, got os=%q size=%q", web.OS, web.Size)
	}
	if web.Disk != 40 {
		t.Errorf("expected child disk 40 to win, got %d", web.Disk)
	}
	// Maps: merged, child wins a clash, base-only keys survive.
	if web.Vars["role"] != "web" || web.Vars["keep"] != "yes" {
		t.Errorf("vars merge wrong: %v", web.Vars)
	}
	if web.Tags["tier"] != "web" || web.Tags["env"] != "prod" {
		t.Errorf("tags merge wrong: %v", web.Tags)
	}
	// Lists: base first then child (steps), set-deduped (depends_on).
	if len(web.Steps) != 2 || web.Steps[0].ActionKey() != "script" {
		t.Fatalf("expected 2 steps base-then-child, got %v", web.Steps)
	}
	if web.Steps[0]["script"] != "base-setup" || web.Steps[1]["script"] != "web-setup" {
		t.Errorf("step order wrong: %v", web.Steps)
	}
	if len(web.DependsOn) != 2 { // [dc] + [dc, db] deduped -> [dc, db]
		t.Errorf("depends_on dedup wrong: %v", web.DependsOn)
	}
	// The base itself is untouched.
	base := findHost(t, c, "base")
	if base.Disk != 20 || len(base.Steps) != 1 {
		t.Errorf("base object mutated by merge: disk=%d steps=%v", base.Disk, base.Steps)
	}
}

func TestExtendsTransitiveChain(t *testing.T) {
	root := writeRepo(t, map[string]string{
		"hosts/a.yaml": `host:
  name: a
  os: ubuntu22
  size: small
  disk: 20
`,
		"hosts/b.yaml": `host:
  name: b
  extends: a
  size: medium
`,
		"hosts/c.yaml": `host:
  name: c
  extends: b
  disk: 80
`,
	})
	c, _ := loader.Load(root)
	if msgs := errorMessages(c); len(msgs) != 0 {
		t.Fatalf("unexpected errors: %v", msgs)
	}
	got := findHost(t, c, "c")
	if got.OS != "ubuntu22" || got.Size != "medium" || got.Disk != 80 {
		t.Errorf("transitive merge wrong: os=%q size=%q disk=%d", got.OS, got.Size, got.Disk)
	}
}

func TestExtendsCycleIsCaught(t *testing.T) {
	root := writeRepo(t, map[string]string{
		"hosts/a.yaml": `host:
  name: a
  extends: b
`,
		"hosts/b.yaml": `host:
  name: b
  extends: a
`,
	})
	c, _ := loader.Load(root)
	if !containsSubstring(errorMessages(c), "extends cycle") {
		t.Fatalf("expected an extends-cycle error, got: %v", errorMessages(c))
	}
}

func TestExtendsCrossTypeIsRejected(t *testing.T) {
	root := writeRepo(t, map[string]string{
		"containers/app.yaml": `container:
  name: app
  image: nginx:alpine
  size: small
`,
		"hosts/web.yaml": `host:
  name: web
  extends: app
`,
	})
	c, _ := loader.Load(root)
	if !containsSubstring(errorMessages(c), "is not a defined host") {
		t.Fatalf("expected a cross-type extends error, got: %v", errorMessages(c))
	}
}

func TestExtendsChainMissingRequiredFieldIsCaught(t *testing.T) {
	// base has no disk, and neither does the child -- the merged host is still
	// missing a schema-required field, which must be reported.
	root := writeRepo(t, map[string]string{
		"hosts/base.yaml": `host:
  name: base
  os: ubuntu22
  size: small
  extends: root
`,
		"hosts/root.yaml": `host:
  name: root
  os: ubuntu22
  size: small
  extends: base2
`,
		"hosts/base2.yaml": `host:
  name: base2
  os: ubuntu22
  size: small
  disk: 0
  extends: nothing-here
`,
	})
	c, _ := loader.Load(root)
	if !containsSubstring(errorMessages(c), "has no disk") {
		t.Fatalf("expected a missing-required-field error, got: %v", errorMessages(c))
	}
}

func TestExtendsContainerInheritsImage(t *testing.T) {
	root := writeRepo(t, map[string]string{
		"containers/base.yaml": `container:
  name: cbase
  image: nginx:alpine
  size: small
  env:
    A: "1"
`,
		"containers/app.yaml": `container:
  name: app
  extends: cbase
  env:
    B: "2"
`,
	})
	c, _ := loader.Load(root)
	if msgs := errorMessages(c); len(msgs) != 0 {
		t.Fatalf("unexpected errors: %v", msgs)
	}
	var app loader.Container
	for _, ct := range c.Containers {
		if ct.Name == "app" {
			app = ct
		}
	}
	if app.Image != "nginx:alpine" || app.Size != "small" {
		t.Errorf("container inheritance wrong: image=%q size=%q", app.Image, app.Size)
	}
	if app.Env["A"] != "1" || app.Env["B"] != "2" {
		t.Errorf("container env merge wrong: %v", app.Env)
	}
}

func TestDependsOnCycleIsCaught(t *testing.T) {
	root := writeRepo(t, map[string]string{
		"hosts/a.yaml": "host:\n  name: a\n  os: ubuntu22\n  size: small\n  disk: 20\n  depends_on: [b]\n",
		"hosts/b.yaml": "host:\n  name: b\n  os: ubuntu22\n  size: small\n  disk: 20\n  depends_on: [a]\n",
	})
	c, _ := loader.Load(root)
	if !containsSubstring(errorMessages(c), "depends_on cycle") {
		t.Fatalf("expected a depends_on cycle error, got: %v", errorMessages(c))
	}
}
