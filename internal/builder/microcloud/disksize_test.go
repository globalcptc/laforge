package microcloud

import (
	"context"
	"errors"
	"testing"
)

func TestParseSizeBytes(t *testing.T) {
	cases := []struct {
		in   string
		want int64
		bad  bool
	}{
		{"64GiB", 64 << 30, false},   // 68,719,476,736
		{"60GB", 60_000_000_000, false},
		{"68719476736", 68719476736, false}, // bare bytes
		{"100MiB", 100 << 20, false},
		{"1.5GiB", int64(1.5 * float64(1<<30)), false},
		{"", 0, true},
		{"GiB", 0, true},
		{"12ZB", 0, true}, // unknown unit
	}
	for _, c := range cases {
		got, err := parseSizeBytes(c.in)
		if c.bad {
			if err == nil {
				t.Errorf("parseSizeBytes(%q) = %d, want error", c.in, got)
			}
			continue
		}
		if err != nil || got != c.want {
			t.Errorf("parseSizeBytes(%q) = %d, %v; want %d", c.in, got, err, c.want)
		}
	}
}

func TestIsShrinkError(t *testing.T) {
	err := errors.New(`lxd api: http=200 code=500: Create instance from copy: Block volumes cannot be shrunk: Cannot be shrunk`)
	if !isShrinkError(err) {
		t.Error("want the real shrink error detected")
	}
	if isShrinkError(errors.New("some other failure")) {
		t.Error("unrelated error misdetected as a shrink error")
	}
	if isShrinkError(nil) {
		t.Error("nil should not be a shrink error")
	}
}

func snapshotBuilder(client *Client, img ImageRef) *Builder {
	return New(client, Config{Images: map[string]ImageRef{"win": img}, StoragePool: "remote"})
}

func TestSourceDiskMinBytes(t *testing.T) {
	snap := ImageRef{Source: SourceSnapshot, Instance: "windows-2016-base", Snapshot: "sysprepped", SourceProject: "CPTC-Templates", VM: true}

	t.Run("explicit root device size", func(t *testing.T) {
		_, client := newProjectRecorder(t, map[string]interface{}{
			"/1.0/instances/windows-2016-base/snapshots/sysprepped": map[string]interface{}{
				"expanded_devices": map[string]interface{}{
					"root": map[string]string{"type": "disk", "path": "/", "pool": "remote", "size": "64GiB"},
				},
			},
		})
		b := snapshotBuilder(client, snap)
		min, src, ok, err := b.SourceDiskMinBytes(context.Background(), "win")
		if err != nil || !ok {
			t.Fatalf("SourceDiskMinBytes = (%d,%q,%v,%v), want ok", min, src, ok, err)
		}
		if min != 64<<30 {
			t.Errorf("min = %d, want %d (64 GiB)", min, int64(64)<<30)
		}
		if src != "CPTC-Templates/windows-2016-base/sysprepped" {
			t.Errorf("source = %q", src)
		}
	})

	t.Run("omitted device size falls back to the volume's configured size", func(t *testing.T) {
		img := ImageRef{Source: SourceSnapshot, Instance: "win2019-base", Snapshot: "sysprepped", SourceProject: "CPTC-Templates", VM: true}
		_, client := newProjectRecorder(t, map[string]interface{}{
			"/1.0/instances/win2019-base/snapshots/sysprepped": map[string]interface{}{
				"expanded_devices": map[string]interface{}{
					"root": map[string]string{"type": "disk", "path": "/", "pool": "remote"}, // no size
				},
			},
			"/1.0/storage-pools/remote/volumes/virtual-machine/win2019-base/snapshots/sysprepped": map[string]interface{}{
				"config": map[string]string{"size": "100GiB"},
			},
		})
		b := snapshotBuilder(client, img)
		min, _, ok, err := b.SourceDiskMinBytes(context.Background(), "win")
		if err != nil || !ok || min != 100<<30 {
			t.Fatalf("SourceDiskMinBytes = (%d,%v,%v), want 100 GiB", min, ok, err)
		}
	})

	t.Run("image source has no shrink constraint", func(t *testing.T) {
		_, client := newProjectRecorder(t, map[string]interface{}{})
		b := snapshotBuilder(client, ImageRef{Fingerprint: "abc123"})
		if _, _, ok, err := b.SourceDiskMinBytes(context.Background(), "win"); ok || err != nil {
			t.Errorf("image source should return ok=false, got ok=%v err=%v", ok, err)
		}
	})

	t.Run("unknown os", func(t *testing.T) {
		_, client := newProjectRecorder(t, map[string]interface{}{})
		b := snapshotBuilder(client, snap)
		if _, _, ok, _ := b.SourceDiskMinBytes(context.Background(), "nope"); ok {
			t.Error("unknown os should return ok=false")
		}
	})

	t.Run("undeterminable: no device size and no volume size", func(t *testing.T) {
		img := ImageRef{Source: SourceSnapshot, Instance: "mystery", Snapshot: "s", SourceProject: "p", VM: true}
		_, client := newProjectRecorder(t, map[string]interface{}{
			"/1.0/instances/mystery/snapshots/s": map[string]interface{}{
				"expanded_devices": map[string]interface{}{"root": map[string]string{"type": "disk", "pool": "remote"}},
			},
			// volume routes return empty config -> size undeterminable
		})
		b := snapshotBuilder(client, img)
		if _, _, ok, err := b.SourceDiskMinBytes(context.Background(), "win"); ok || err != nil {
			t.Errorf("want ok=false,err=nil when size is undeterminable, got ok=%v err=%v", ok, err)
		}
	})
}
