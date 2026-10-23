package db

import (
	"context"
	"crypto/rand"

	"github.com/jackc/pgx/v5/pgtype"
)

// RenderSecretForBuild returns the build's password-generator secret (see
// migrations/00069), creating a fresh random one on first use. Every caller
// that renders content for a real build passes it to render.Context.Secret,
// so generated passwords are identical across those renders.
func RenderSecretForBuild(ctx context.Context, q *Queries, buildID pgtype.UUID) ([]byte, error) {
	fresh := make([]byte, 32)
	if _, err := rand.Read(fresh); err != nil {
		return nil, err
	}
	return q.EnsureBuildRenderSecret(ctx, EnsureBuildRenderSecretParams{BuildID: buildID, Secret: fresh})
}
