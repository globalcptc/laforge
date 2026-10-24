package microcloud

import (
	"errors"
	"strings"
	"testing"

	"github.com/globalcptc/laforge/internal/builder"
)

// instanceName and sanitizeName are pure -- no daemon needed, so these run
// everywhere (unlike builder_test.go, which skips without a live Incus).
func TestInstanceNameReadableAndBounded(t *testing.T) {
	const ext = "build-11111111-2222-3333-4444-555555555555-team-3-host-webserver"

	t.Run("readable from a display hint", func(t *testing.T) {
		got := instanceName("main-t3-webserver", ext, "default")
		if !strings.HasPrefix(got, "lf-main-t3-webserver-") {
			t.Fatalf("instanceName = %q, want it to start with the readable lf-main-t3-webserver-", got)
		}
		// A 6-hex disambiguator keeps it unique across builds of the same branch.
		if suffix := got[len("lf-main-t3-webserver-"):]; len(suffix) != 6 {
			t.Fatalf("suffix = %q, want 6 hex chars", suffix)
		}
	})

	t.Run("deterministic", func(t *testing.T) {
		if a, b := instanceName("main-t3-webserver", ext, "default"), instanceName("main-t3-webserver", ext, "default"); a != b {
			t.Fatalf("not deterministic: %q vs %q", a, b)
		}
	})

	t.Run("same host, different builds don't collide", func(t *testing.T) {
		a := instanceName("main-t3-webserver", ext+"-A", "default")
		b := instanceName("main-t3-webserver", ext+"-B", "default")
		if a == b {
			t.Fatalf("both builds got %q -- the suffix must disambiguate them", a)
		}
	})

	t.Run("empty hint falls back to the hash", func(t *testing.T) {
		if got, want := instanceName("", ext, "default"), shortName(ext); got != want {
			t.Fatalf("instanceName(\"\", ...) = %q, want the shortName hash %q", got, want)
		}
	})

	t.Run("network name is readable, lf-prefixed, and within 15 chars", func(t *testing.T) {
		const netExt = "build-11111111-2222-3333-4444-555555555555-network-vdi"
		got := networkName("t1vdi", netExt)
		if len(got) > 15 {
			t.Fatalf("network name %q is %d chars, over Incus's 15-char cap", got, len(got))
		}
		if !strings.HasPrefix(got, "lf-t1vdi") {
			t.Fatalf("network name = %q, want it to start with lf-t1vdi (Inspect keys off the lf- prefix)", got)
		}
		if strings.Contains(got, "-t1") && strings.Count(got, "-") != 1 {
			t.Fatalf("network name %q should have exactly the one lf- hyphen", got)
		}
	})

	t.Run("network deploy and instance NIC agree on the name", func(t *testing.T) {
		// The network's own deploy and any instance referencing it must
		// compute the identical name from the identical inputs.
		const netExt = "build-abc-network-corpdmz"
		deployName := networkName("t3corpdmz", netExt)
		nicName := networkName("t3corpdmz", netExt)
		if deployName != nicName {
			t.Fatalf("network deploy named it %q but the NIC referenced %q", deployName, nicName)
		}
	})

	t.Run("long network label truncates but still fits", func(t *testing.T) {
		got := networkName("t10corporatedmz", "build-x-network-corporatedmz")
		if len(got) > 15 {
			t.Fatalf("network name %q is %d chars, over the 15-char cap", got, len(got))
		}
	})

	t.Run("sanitizes and stays within Incus's limits", func(t *testing.T) {
		// A messy branch with slashes, spaces, and uppercase.
		got := instanceName("Feature/Big Rewrite-t10-DB Server!!", ext, "default")
		for _, r := range got {
			ok := r == '-' || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
			if !ok {
				t.Fatalf("instanceName produced an invalid char %q in %q", r, got)
			}
		}
		if strings.Contains(got, "--") || strings.HasSuffix(got, "-") {
			t.Fatalf("instanceName has doubled/trailing hyphens: %q", got)
		}
		// Must leave room for the "-cidata" volume suffix under the 63-char cap.
		if len(got)+len("-cidata") > 63 {
			t.Fatalf("instanceName %q (%d) + -cidata exceeds Incus's 63-char limit", got, len(got))
		}
	})
}

// socketPath mirrors the path LXD opens for a VM's qemu monitor, the length that
// actually gates a VM start (issue #63).
func socketPath(project, name string) string {
	return lxdSnapLogsDir + project + "_" + name + qemuMonitorSuffix
}

func TestInstanceNameFitsSocketPath(t *testing.T) {
	// The exact failing case from issue #63: a 21-char project and a long
	// placement that produced a 108-byte qemu.monitor path.
	const project = "CPTC-Comp-Environment"
	const ext = "build-f8417f6d-team-1-triplecorkscrew"

	t.Run("long project + placement stays within the socket limit", func(t *testing.T) {
		name := instanceName("lf-cptc12-forrest-t1-triplecorkscrew", ext, project)
		if p := socketPath(project, name); len(p) > unixSocketPathMax {
			t.Fatalf("socket path is %d bytes (name %q), over the %d limit", len(p), name, unixSocketPathMax)
		}
		// Still deterministic and suffixed for uniqueness.
		if name != instanceName("lf-cptc12-forrest-t1-triplecorkscrew", ext, project) {
			t.Fatal("not deterministic under the socket cap")
		}
	})

	t.Run("boundary: a name at the 107-byte path fits, 108 would not", func(t *testing.T) {
		// Craft a name exactly at the limit and one over, and confirm the budget
		// function agrees with the real path length.
		maxName := maxInstanceNameLen(project)
		atLimit := strings.Repeat("a", maxName)
		if p := len(socketPath(project, atLimit)); p != unixSocketPathMax {
			t.Fatalf("name at maxInstanceNameLen yields a %d-byte path, want exactly %d", p, unixSocketPathMax)
		}
		over := strings.Repeat("a", maxName+1)
		if p := len(socketPath(project, over)); p <= unixSocketPathMax {
			t.Fatalf("name one over the budget yields %d bytes, expected > %d", p, unixSocketPathMax)
		}
	})

	t.Run("default project keeps the familiar 63-char cap, names unchanged", func(t *testing.T) {
		// A short project must not shrink names below the old behavior.
		if got := maxInstanceNameLen("default"); got != 63-len("-cidata") {
			t.Fatalf("maxInstanceNameLen(default) = %d, want %d (cidata cap)", got, 63-len("-cidata"))
		}
	})

	t.Run("project too long is a permanent error, not a retry loop", func(t *testing.T) {
		if err := validateInstanceNameBudget("default"); err != nil {
			t.Fatalf("default project should validate: %v", err)
		}
		huge := strings.Repeat("x", 90)
		err := validateInstanceNameBudget(huge)
		if err == nil {
			t.Fatal("a 90-char project should be rejected")
		}
		if !errors.Is(err, builder.ErrPermanentConfig) {
			t.Fatalf("want ErrPermanentConfig, got %v", err)
		}
	})
}
