// Disk-size validation for snapshot-copy sources (ported from the MicroCloud
// builder). LXD/Incus can grow a copied block volume but NEVER shrink it ("Block
// volumes cannot be shrunk"), so a host whose `disk:` is smaller than the template
// snapshot it copies from passes instance creation's early checks and then fails
// deterministically at the copy. This resolves a snapshot source's minimum root
// capacity so preflight can reject the mismatch before anything is provisioned.
package incus

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// sizeMultipliers maps LXD size-string units to bytes. Decimal (GB = 10^9) and
// binary (GiB = 2^30) are distinct -- exactly the 60GB vs 64GiB mismatch the bug
// turned on -- so both normalize to bytes here.
var sizeMultipliers = map[string]int64{
	"": 1, "B": 1,
	"kB": 1e3, "KB": 1e3, "MB": 1e6, "GB": 1e9, "TB": 1e12, "PB": 1e15,
	"KiB": 1 << 10, "MiB": 1 << 20, "GiB": 1 << 30, "TiB": 1 << 40, "PiB": 1 << 50,
}

// parseSizeBytes turns an LXD size string ("64GiB", "60GB", "68719476736") into
// bytes. A bare number is bytes; an unknown unit errors so a surprising format is
// never silently read as zero.
func parseSizeBytes(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty size")
	}
	i := 0
	for i < len(s) && (s[i] == '.' || (s[i] >= '0' && s[i] <= '9')) {
		i++
	}
	numStr, unit := s[:i], strings.TrimSpace(s[i:])
	if numStr == "" {
		return 0, fmt.Errorf("no number in size %q", s)
	}
	num, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return 0, fmt.Errorf("parsing size %q: %w", s, err)
	}
	mult, ok := sizeMultipliers[unit]
	if !ok {
		return 0, fmt.Errorf("unknown size unit %q in %q", unit, s)
	}
	return int64(num * float64(mult)), nil
}

// isShrinkError reports whether an LXD create error is the block-volume shrink
// rejection ("Block volumes cannot be shrunk") -- a permanent size mismatch.
func isShrinkError(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "cannot be shrunk")
}

// SourceDiskMinBytes resolves the minimum root-disk capacity a copy from the
// snapshot mapped to os must have (its block volume can't be shrunk below the
// source), a human description of that source, and whether it could be
// determined. Only snapshot-copy sources carry this constraint; non-snapshot os
// names return ok=false. Reads the snapshot's root-device size first, then falls
// back to the backing volume's configured size when the device omits one. ok=false
// with no error means the minimum genuinely couldn't be determined.
func (b *Builder) SourceDiskMinBytes(ctx context.Context, os string) (minBytes int64, source string, ok bool, err error) {
	img, found := b.Config.Images[os]
	if !found || !img.IsSnapshot() {
		return 0, "", false, nil
	}
	source = img.Instance + "/" + img.Snapshot
	if img.SourceProject != "" {
		source = img.SourceProject + "/" + source
	}
	scoped := *b.Client
	if img.SourceProject != "" {
		scoped.Project = img.SourceProject
	}

	raw, err := scoped.get(ctx, "/1.0/instances/"+url.PathEscape(img.Instance)+"/snapshots/"+url.PathEscape(img.Snapshot))
	if err != nil {
		return 0, source, false, fmt.Errorf("reading source snapshot %s: %w", source, err)
	}
	var snap struct {
		Devices         map[string]map[string]string `json:"devices"`
		ExpandedDevices map[string]map[string]string `json:"expanded_devices"`
	}
	if err := json.Unmarshal(raw, &snap); err != nil {
		return 0, source, false, fmt.Errorf("decoding source snapshot %s: %w", source, err)
	}

	if sz := rootDeviceField(snap.ExpandedDevices, snap.Devices, "size"); sz != "" {
		if n, perr := parseSizeBytes(sz); perr == nil && n > 0 {
			return n, source, true, nil
		}
	}
	pool := rootDeviceField(snap.ExpandedDevices, snap.Devices, "pool")
	if pool != "" {
		if n, found := b.volumeSizeBytes(ctx, &scoped, pool, img.Instance, img.Snapshot); found {
			return n, source, true, nil
		}
	}
	return 0, source, false, nil
}

// rootDeviceField returns a field of the "root" disk device, preferring the
// expanded devices (profiles merged in) over the instance's own.
func rootDeviceField(expanded, own map[string]map[string]string, field string) string {
	for _, set := range []map[string]map[string]string{expanded, own} {
		if root, ok := set["root"]; ok {
			if v := root[field]; v != "" {
				return v
			}
		}
	}
	return ""
}

// volumeSizeBytes reads the configured size of a VM's root block volume (or its
// snapshot) from the storage pool, for sources whose root device omits a size.
func (b *Builder) volumeSizeBytes(ctx context.Context, scoped *Client, pool, instance, snapshot string) (int64, bool) {
	base := "/1.0/storage-pools/" + url.PathEscape(pool) + "/volumes/virtual-machine/" + url.PathEscape(instance)
	for _, path := range []string{base + "/snapshots/" + url.PathEscape(snapshot), base} {
		raw, err := scoped.get(ctx, path)
		if err != nil {
			continue
		}
		var vol struct {
			Config map[string]string `json:"config"`
		}
		if json.Unmarshal(raw, &vol) != nil {
			continue
		}
		if sz := vol.Config["size"]; sz != "" {
			if n, perr := parseSizeBytes(sz); perr == nil && n > 0 {
				return n, true
			}
		}
	}
	return 0, false
}
