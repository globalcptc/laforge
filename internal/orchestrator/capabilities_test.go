package orchestrator

import (
	"context"
	"strings"
	"testing"

	"github.com/globalcptc/laforge/internal/builder"
	"github.com/globalcptc/laforge/internal/loader"
)

// fakeDiskBuilder is a builder.Builder (via the embedded nil interface -- only
// SourceDiskMinBytes is ever called by the disk check) that reports a fixed
// source minimum, so checkHostDiskCapacity can be tested without a live hoster.
type fakeDiskBuilder struct {
	builder.Builder
	min  int64
	src  string
	ok   bool
	err  error
	seen map[string]bool
}

func (f *fakeDiskBuilder) SourceDiskMinBytes(_ context.Context, os string) (int64, string, bool, error) {
	if f.seen == nil {
		f.seen = map[string]bool{}
	}
	f.seen[os] = true
	return f.min, f.src, f.ok, f.err
}

func envWithHost(hostName, os string, disk int) (*loader.Content, *loader.Environment) {
	c := &loader.Content{Hosts: []loader.Host{{Name: hostName, OS: os, Disk: disk}}}
	env := &loader.Environment{
		Name:  "e",
		Teams: 1,
		Networks: map[string]map[string][]loader.Copy{
			"lan": {hostName: {{As: hostName}}},
		},
	}
	return c, env
}

func TestCheckHostDiskCapacity(t *testing.T) {
	const giB64 = int64(64) << 30 // 68,719,476,736 -- the win2016 template minimum

	t.Run("below minimum is rejected with an actionable message", func(t *testing.T) {
		c, env := envWithHost("bff-dc-01", "win2016", 60) // 60 GB = 60e9 < 64 GiB
		b := &fakeDiskBuilder{min: giB64, src: "CPTC-Templates/windows-2016-base/sysprepped", ok: true}
		err := checkHostDiskCapacity(context.Background(), b, "disk-test-1", c, env, nil)
		if err == nil {
			t.Fatal("want rejection for a 60 GB host against a 64 GiB source")
		}
		for _, want := range []string{"bff-dc-01", "60 GB", "win2016", "64 GiB", "68719476736", "69 GB"} {
			if !strings.Contains(err.Error(), want) {
				t.Errorf("error %q missing %q", err.Error(), want)
			}
		}
	})

	t.Run("equal is accepted", func(t *testing.T) {
		// 69 GB = 69e9 >= 68,719,476,736.
		c, env := envWithHost("h", "win2016", 69)
		b := &fakeDiskBuilder{min: giB64, ok: true}
		if err := checkHostDiskCapacity(context.Background(), b, "disk-test-2", c, env, nil); err != nil {
			t.Errorf("69 GB should satisfy a 64 GiB source: %v", err)
		}
	})

	t.Run("above minimum is accepted", func(t *testing.T) {
		c, env := envWithHost("h", "win2016", 120)
		b := &fakeDiskBuilder{min: giB64, ok: true}
		if err := checkHostDiskCapacity(context.Background(), b, "disk-test-3", c, env, nil); err != nil {
			t.Errorf("120 GB should satisfy a 64 GiB source: %v", err)
		}
	})

	t.Run("disk 64 is still too small for a 64 GiB source (GB vs GiB)", func(t *testing.T) {
		c, env := envWithHost("h", "win2016", 64) // 64 GB = 64e9 < 64 GiB = 68.7e9
		b := &fakeDiskBuilder{min: giB64, ok: true}
		if err := checkHostDiskCapacity(context.Background(), b, "disk-test-4", c, env, nil); err == nil {
			t.Error("64 GB (decimal) must not satisfy a 64 GiB (binary) source")
		}
	})

	t.Run("no explicit disk is skipped (copy keeps the source size)", func(t *testing.T) {
		c, env := envWithHost("h", "win2016", 0)
		b := &fakeDiskBuilder{min: giB64, ok: true}
		if err := checkHostDiskCapacity(context.Background(), b, "disk-test-5", c, env, nil); err != nil {
			t.Errorf("a host with no disk: should be skipped: %v", err)
		}
		if b.seen["win2016"] {
			t.Error("should not even query the source for a host with no disk")
		}
	})

	t.Run("undeterminable source minimum is skipped (deploy guards it)", func(t *testing.T) {
		c, env := envWithHost("h", "win2019", 60)
		b := &fakeDiskBuilder{ok: false}
		if err := checkHostDiskCapacity(context.Background(), b, "disk-test-6", c, env, nil); err != nil {
			t.Errorf("an undeterminable minimum must not fail preflight: %v", err)
		}
	})
}

func TestHumanBytes(t *testing.T) {
	cases := map[int64]string{
		64 << 30:        "64 GiB",
		60_000_000_000:  "60 GB",
		(64 << 30) + 1:  "64.0 GiB",
	}
	for n, want := range cases {
		if got := humanBytes(n); got != want {
			t.Errorf("humanBytes(%d) = %q, want %q", n, got, want)
		}
	}
}
