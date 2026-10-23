// Build preflight validates image mappings and the configured team placement
// before reconciliation creates teams, deployed objects, or deployment tasks.
package orchestrator

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/globalcptc/laforge/internal/builder"
	"github.com/globalcptc/laforge/internal/builderconfig"
	"github.com/globalcptc/laforge/internal/db"
	"github.com/globalcptc/laforge/internal/loader"
)

// imageCatalog is satisfied by any resolved builder that exposes its own
// os/image name set -- today internal/builder/incus.Builder and
// microcloud.Builder (via Config.Images) and internal/builder/incuspool.Pool
// (its pool-wide Images), through a small ImageNames() method rather than a
// growing type switch, so a future builder kind picks this check up for free.
type imageCatalog interface {
	ImageNames() map[string]bool
}

// resolveBuilderImages resolves build's configured builder (via its
// builder_config, internal/builderconfig.Resolve) and returns the set of
// os/image names it has configured, for a builder kind that exposes one, plus
// the builder's name. A build with no configured_build at all (the ad-hoc/test
// path) returns nil images and an empty name -- nothing to check.
func resolveBuilderImages(ctx context.Context, q *db.Queries, pool *pgxpool.Pool, build db.Build, teamCount int) (images map[string]bool, builderName string, kind string, b builder.Builder, err error) {
	if !build.ConfiguredBuildID.Valid {
		return nil, "", "", nil, nil
	}
	cb, err := q.GetConfiguredBuild(ctx, build.ConfiguredBuildID)
	if err != nil {
		return nil, "", "", nil, fmt.Errorf("loading configured build: %w", err)
	}
	row, err := q.GetBuilderConfigByName(ctx, cb.BuilderConfigName)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, cb.BuilderConfigName, "", nil, fmt.Errorf("no such builder config %q", cb.BuilderConfigName)
		}
		return nil, cb.BuilderConfigName, "", nil, fmt.Errorf("loading builder config %q: %w", cb.BuilderConfigName, err)
	}
	b, err = builderconfig.Resolve(pool, row)
	if err != nil {
		return nil, cb.BuilderConfigName, row.Kind, nil, fmt.Errorf("resolving builder %q: %w", cb.BuilderConfigName, err)
	}
	if placement, ok := b.(interface{ ValidateTeamCount(int) error }); ok {
		if err := placement.ValidateTeamCount(teamCount); err != nil {
			return nil, cb.BuilderConfigName, row.Kind, b, err
		}
	}
	if ic, ok := b.(imageCatalog); ok {
		images = ic.ImageNames()
	}
	// A compose container boots from the builder's docker base image once it
	// has been built, so that satisfies the compose-host requirement too.
	if images != nil && row.DockerBaseFingerprint != "" && !images[builder.ComposeHostImage] {
		withBase := make(map[string]bool, len(images)+1)
		for name := range images {
			withBase[name] = true
		}
		withBase[builder.ComposeHostImage] = true
		images = withBase
	}
	return images, cb.BuilderConfigName, row.Kind, b, nil
}

// compatibilityRecheck is how long a passed compatibility check holds for one
// (build, content revision). The check asks the hoster about every source
// image (SourceDiskMinBytes reads each image's snapshot), which on a loaded
// MicroCloud cluster takes 10+ seconds per image -- run on every 2-second
// pass per team, as it was, it ate the whole pass budget before a single
// host was reconciled. The answer only changes when the content revision or
// the builder's images change, so it's kept and re-asked hourly.
const compatibilityRecheck = time.Hour

var (
	compatMu      sync.Mutex
	compatPassed  = map[string]time.Time{}    // "build\x00revision\x00builder" -> when it last passed
	diskMinCache  = map[string]diskMinEntry{} // "builder\x00os" -> the source's minimum
	diskMinExpiry = time.Hour
)

type diskMinEntry struct {
	minBytes int64
	source   string
	ok       bool
	at       time.Time
}

// checkBuilderCompatibility validates image mappings, team placement, and that
// each host's requested disk fits its mapped source (a copied block volume can't
// be shrunk below the template it comes from). A pass is remembered per
// (build, revision, builder) for compatibilityRecheck; a failure is not, so
// the next pass asks again and the build's error clears as soon as it's fixed.
func checkBuilderCompatibility(ctx context.Context, q *db.Queries, pool *pgxpool.Pool, build db.Build, c *loader.Content, env *loader.Environment, containerNames map[string]bool) error {
	images, builderName, kind, b, err := resolveBuilderImages(ctx, q, pool, build, env.Teams)
	if err != nil {
		return err
	}
	if builderName == "" {
		return nil // ad-hoc/test build, nothing configured to check against
	}
	cacheKey := build.ID.String() + "\x00" + build.ContentRevisionID.String() + "\x00" + builderName
	compatMu.Lock()
	passedAt, passed := compatPassed[cacheKey]
	compatMu.Unlock()
	if passed && time.Since(passedAt) < compatibilityRecheck {
		return nil
	}
	if err := checkBuilderCompatibilityNow(ctx, b, builderName, kind, c, env, containerNames, images); err != nil {
		return err
	}
	compatMu.Lock()
	compatPassed[cacheKey] = time.Now()
	compatMu.Unlock()
	return nil
}

func checkBuilderCompatibilityNow(ctx context.Context, b builder.Builder, builderName, kind string, c *loader.Content, env *loader.Environment, containerNames map[string]bool, images map[string]bool) error {
	if err := checkEnvironmentImages(env, c, containerNames, images); err != nil {
		// The compose-host remedy differs by builder kind (LXD builds a base image,
		// cloud builders map one), so format that one here where the kind is known.
		if errors.Is(err, errComposeHostMissing) {
			return fmt.Errorf("environment %q incompatible with builder %q: %s", env.Name, builderName, composeHostHint(kind))
		}
		return fmt.Errorf("environment %q incompatible with builder %q: %w", env.Name, builderName, err)
	}
	if err := checkHostDiskCapacity(ctx, b, builderName, c, env, containerNames); err != nil {
		return fmt.Errorf("environment %q incompatible with builder %q: %w", env.Name, builderName, err)
	}
	return nil
}

// diskCapacitySource is satisfied by a builder (today MicroCloud) that copies
// hosts from a source whose root block volume can't be shrunk, exposing the
// source's minimum root capacity so preflight can reject an undersized host disk
// before anything is provisioned.
type diskCapacitySource interface {
	SourceDiskMinBytes(ctx context.Context, os string) (minBytes int64, source string, ok bool, err error)
}

// checkHostDiskCapacity rejects any placed host whose requested `disk:` is
// smaller than the minimum capacity of the source its os maps to. Only hosts
// with an explicit disk size are checked -- with none, the copy keeps the
// source's own size and can't shrink. A source whose minimum can't be determined
// is left to the deploy-time shrink guard. Reads are deduplicated by (os,disk).
func checkHostDiskCapacity(ctx context.Context, b builder.Builder, builderName string, c *loader.Content, env *loader.Environment, containerNames map[string]bool) error {
	dc, ok := b.(diskCapacitySource)
	if !ok {
		return nil
	}
	// A source's minimum is a property of the builder's image, not of this
	// build: remembered per (builder, os) for diskMinExpiry so a dozen builds
	// on one cluster read each snapshot once an hour, not each per pass.
	sourceMin := func(os string) (int64, string, bool, error) {
		key := builderName + "\x00" + os
		compatMu.Lock()
		e, hit := diskMinCache[key]
		compatMu.Unlock()
		if hit && time.Since(e.at) < diskMinExpiry {
			return e.minBytes, e.source, e.ok, nil
		}
		started := time.Now()
		minBytes, source, ok, err := dc.SourceDiskMinBytes(ctx, os)
		if err != nil {
			log.Printf("orchestrator: builder %q: reading the source size for os %q failed after %s (the deploy-time guard still applies): %v", builderName, os, time.Since(started).Round(time.Millisecond), err)
			return 0, source, false, err
		}
		log.Printf("orchestrator: builder %q: source for os %q is %s, minimum %d bytes (read in %s; cached for %s)", builderName, os, source, minBytes, time.Since(started).Round(time.Millisecond), diskMinExpiry)
		compatMu.Lock()
		diskMinCache[key] = diskMinEntry{minBytes: minBytes, source: source, ok: ok, at: time.Now()}
		compatMu.Unlock()
		return minBytes, source, ok, nil
	}
	checked := map[string]bool{}
	for _, objs := range env.Networks {
		for objectName, copies := range objs {
			if len(copies) == 0 || containerNames[objectName] {
				continue
			}
			h := findHost(c, objectName)
			if h == nil || h.Disk <= 0 {
				continue
			}
			key := fmt.Sprintf("%s|%d", h.OS, h.Disk)
			if checked[key] {
				continue
			}
			checked[key] = true
			minBytes, source, ok, err := sourceMin(h.OS)
			if err != nil || !ok {
				continue // transient read error, or minimum undeterminable -- deploy guards it
			}
			requested := int64(h.Disk) * 1_000_000_000 // host.disk is GB (decimal), matching how the builder renders it
			if requested < minBytes {
				needGB := (minBytes + 999_999_999) / 1_000_000_000 // ceil to whole GB
				return fmt.Errorf("host %q requests %d GB, but os %q source %s requires at least %s (%d bytes); increase host.disk to at least %d GB",
					h.Name, h.Disk, h.OS, source, humanBytes(minBytes), minBytes, needGB)
			}
		}
	}
	return nil
}

// humanBytes renders a capacity the way the source expressed it: whole GiB when
// it divides evenly (the common template case, e.g. "64 GiB"), else whole GB,
// else a fractional GiB.
func humanBytes(n int64) string {
	switch {
	case n%(1<<30) == 0:
		return fmt.Sprintf("%d GiB", n/(1<<30))
	case n%1_000_000_000 == 0:
		return fmt.Sprintf("%d GB", n/1_000_000_000)
	default:
		return fmt.Sprintf("%.1f GiB", float64(n)/(1<<30))
	}
}

// checkEnvironmentImages verifies every host's os has an image mapped in the
// builder config (the exact "windows-server-2022 not listed" example the plan
// itself gives). A container's `image` is an OCI ref, not an os-map entry, so
// it's deliberately not checked here. images is nil when the resolved builder
// exposes no inspectable image map -- that one check is skipped, not failed
// closed.
func checkEnvironmentImages(env *loader.Environment, c *loader.Content, containerNames map[string]bool, images map[string]bool) error {
	if images == nil {
		return nil
	}
	imageNames := map[string]bool{}
	for _, objs := range env.Networks {
		for objectName, copies := range objs {
			if len(copies) == 0 {
				continue
			}
			if containerNames[objectName] {
				// A compose container runs on a machine booted from the
				// builder's compose-host image.
				if ct := findContainer(c, objectName); ct != nil && ct.Compose != "" {
					imageNames[builder.ComposeHostImage] = true
				}
				continue
			}
			if h := findHost(c, objectName); h != nil {
				imageNames[h.OS] = true
			}
		}
	}
	for name := range imageNames {
		if !images[name] && name == builder.ComposeHostImage {
			// Formatted kind-aware by the caller: LXD builders build a docker base
			// image, cloud builders map a Docker-capable image instead (no build).
			return errComposeHostMissing
		}
		if !images[name] {
			return fmt.Errorf("uses os/image %q, which the builder has no image configured for", name)
		}
	}
	return nil
}

// errComposeHostMissing signals that the environment has a compose container but
// the builder has no compose-host image. checkBuilderCompatibility turns it into
// a kind-aware message (see composeHostHint) -- the remedy differs by builder.
var errComposeHostMissing = errors.New("missing compose-host image")

// composeHostHint explains how to satisfy the compose-host requirement for a
// builder of the given kind. LXD builders (incus/microcloud) build a docker base
// image from the UI; cloud builders have no base image to build -- you map an
// existing Docker-capable image under the name "compose-host" instead (the
// "Infrastructure → Base image" button the LXD message mentions does not exist
// for them).
func composeHostHint(kind string) string {
	switch kind {
	case "incus", "microcloud":
		return fmt.Sprintf("has a compose container, which needs the builder's docker base image -- build it first (Infrastructure → Base image), or add a builder image named %q", builder.ComposeHostImage)
	case "aws":
		return fmt.Sprintf("has a compose container, which needs a builder image named %q mapped to a Docker-capable AMI -- add it in the builder's Images (there is no base image to build on AWS)", builder.ComposeHostImage)
	default:
		return fmt.Sprintf("has a compose container, which needs a builder image named %q mapped to a Docker-capable image -- add it in the builder's Images", builder.ComposeHostImage)
	}
}
