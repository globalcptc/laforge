package runner

import (
	"context"

	"github.com/globalcptc/laforge/internal/compose"
	"github.com/globalcptc/laforge/internal/db"
)

// registryHost extracts the registry host from an OCI image ref, or "" for a
// Docker Hub ref (see compose.RegistryHost, shared with `compose:` steps).
func registryHost(image string) string {
	return compose.RegistryHost(image)
}

// registryCredFor returns the stored credential for an image's registry, or
// nil when there is none (a public image, or an unauthenticated registry). A
// Docker Hub ref looks up the special "docker.io" host, so a private Docker
// Hub account can still be configured. The runner passes the result to the
// builder on the ContainerSpec; the builder (MicroCloud) `docker login`s with
// it before pulling, since the image pull is the builder's job, not an agent
// step -- a container's agent runs inside the app and has no Docker.
func (r *Runner) registryCredFor(ctx context.Context, q *db.Queries, image string) *db.RegistryCredential {
	host := registryHost(image)
	if host == "" {
		host = "docker.io"
	}
	cred, err := q.GetRegistryCredentialByHost(ctx, host)
	if err != nil {
		return nil
	}
	return &cred
}
