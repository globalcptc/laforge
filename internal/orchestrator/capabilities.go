// This file is Reconcile's pre-build validation of the one thing that is NOT a
// capability opt-out: every os/image a host references must have a concrete
// image mapped in the builder config, so "uses windows-server-2022, which the
// builder has no image for" fails before the build starts rather than mid-deploy.
// There is deliberately no
// capability check -- content is builder-agnostic; a builder either implements
// the whole contract or it isn't a valid builder (see internal/builder's own doc
// comment). Runs once per Reconcile pass, before any team/deployed_object work.
package orchestrator

import (
	"context"
	"errors"
	"fmt"

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
func resolveBuilderImages(ctx context.Context, q *db.Queries, pool *pgxpool.Pool, build db.Build) (images map[string]bool, builderName string, err error) {
	if !build.ConfiguredBuildID.Valid {
		return nil, "", nil
	}
	cb, err := q.GetConfiguredBuild(ctx, build.ConfiguredBuildID)
	if err != nil {
		return nil, "", fmt.Errorf("loading configured build: %w", err)
	}
	row, err := q.GetBuilderConfigByName(ctx, cb.BuilderConfigName)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, cb.BuilderConfigName, fmt.Errorf("no such builder config %q", cb.BuilderConfigName)
		}
		return nil, cb.BuilderConfigName, fmt.Errorf("loading builder config %q: %w", cb.BuilderConfigName, err)
	}
	b, err := builderconfig.Resolve(pool, row)
	if err != nil {
		return nil, cb.BuilderConfigName, fmt.Errorf("resolving builder %q: %w", cb.BuilderConfigName, err)
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
	return images, cb.BuilderConfigName, nil
}

// checkBuilderCompatibility validates that every os/image the environment
// references is mapped in the resolved builder's config. There is no capability
// gating -- content never varies by builder -- so this is purely the
// image-completeness check.
func checkBuilderCompatibility(ctx context.Context, q *db.Queries, pool *pgxpool.Pool, build db.Build, c *loader.Content, env *loader.Environment, containerNames map[string]bool) error {
	images, builderName, err := resolveBuilderImages(ctx, q, pool, build)
	if err != nil {
		return err
	}
	if builderName == "" {
		return nil // ad-hoc/test build, nothing configured to check against
	}
	if err := checkEnvironmentImages(env, c, containerNames, images); err != nil {
		return fmt.Errorf("environment %q incompatible with builder %q: %w", env.Name, builderName, err)
	}
	return nil
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
			return fmt.Errorf("has a compose container, which needs the builder's docker base image -- build it first (Infrastructure → Base image), or add a builder image named %q", name)
		}
		if !images[name] {
			return fmt.Errorf("uses os/image %q, which the builder has no image configured for", name)
		}
	}
	return nil
}
