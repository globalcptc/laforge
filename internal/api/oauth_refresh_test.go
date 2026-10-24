package api

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/globalcptc/laforge/internal/db"
)

// TestRefreshGithubTokenRenewsExpired proves the core of "keep their GitHub
// session refreshed": a session whose GitHub access token has expired but whose
// refresh token is still good gets a fresh access token (rotated refresh token
// too), persisted, so callers keep working instead of hitting odd 401s.
func TestRefreshGithubTokenRenewsExpired(t *testing.T) {
	env := setupAPITest(t)
	ctx := context.Background()

	// Unique identity + token per run: this suite shares one Postgres DB across
	// packages/runs, and the session token_hash is unique.
	uniq := time.Now().UnixNano()
	acct, err := env.q.UpsertAccount(ctx, db.UpsertAccountParams{GithubID: uniq, GithubLogin: fmt.Sprintf("refresh-user-%d", uniq), AvatarUrl: db.StrPtr("")})
	if err != nil {
		t.Fatalf("UpsertAccount: %v", err)
	}
	raw := fmt.Sprintf("refresh-raw-%d", uniq)
	t.Cleanup(func() {
		env.server.Pool.Exec(context.Background(), "DELETE FROM session WHERE token_hash = $1", hashToken(raw))
		env.server.Pool.Exec(context.Background(), "DELETE FROM account WHERE id = $1", acct.ID)
	})
	_, err = env.q.CreateSession(ctx, db.CreateSessionParams{
		AccountID:              acct.ID,
		TokenHash:              hashToken(raw),
		GithubToken:            "stale-access",
		ExpiresAt:              pgtype.Timestamptz{Time: time.Now().Add(7 * 24 * time.Hour), Valid: true}, // LaForge session still valid
		GithubTokenExpiresAt:   pgtype.Timestamptz{Time: time.Now().Add(-time.Hour), Valid: true},         // GitHub token expired
		GithubRefreshToken:     "rt-123",
		GithubRefreshExpiresAt: pgtype.Timestamptz{Time: time.Now().Add(30 * 24 * time.Hour), Valid: true},
	})
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	row, err := env.q.GetSessionByTokenHash(ctx, hashToken(raw))
	if err != nil {
		t.Fatalf("GetSessionByTokenHash: %v", err)
	}

	refreshed, err := env.server.refreshGithubToken(ctx, row)
	if err != nil {
		t.Fatalf("refreshGithubToken: %v", err)
	}
	if refreshed.GithubToken != "refreshed-rt-123" {
		t.Fatalf("token = %q, want refreshed-rt-123 (the fake rotates on refresh_token grant)", refreshed.GithubToken)
	}

	// Persisted, and no longer expired, so the next request won't refresh again.
	after, err := env.q.GetSessionByTokenHash(ctx, hashToken(raw))
	if err != nil {
		t.Fatalf("GetSessionByTokenHash (after): %v", err)
	}
	if after.GithubToken != "refreshed-rt-123" {
		t.Fatalf("persisted token = %q, want refreshed-rt-123", after.GithubToken)
	}
	if after.GithubRefreshToken != "rotated-rt-123" {
		t.Fatalf("persisted refresh token = %q, want rotated-rt-123 (rotation stored)", after.GithubRefreshToken)
	}
	if !after.GithubTokenExpiresAt.Valid || time.Now().After(after.GithubTokenExpiresAt.Time) {
		t.Fatalf("new token expiry = %v, want a future time", after.GithubTokenExpiresAt)
	}
}

// TestRefreshGithubTokenPassesThroughNonExpiring: a session with no recorded
// token expiry (the App doesn't expire user tokens, or it predates this
// feature) is returned untouched -- no refresh attempted.
func TestRefreshGithubTokenPassesThroughNonExpiring(t *testing.T) {
	env := setupAPITest(t)
	ctx := context.Background()
	row := db.GetSessionByTokenHashRow{GithubToken: "static-token"} // no GithubTokenExpiresAt
	got, err := env.server.refreshGithubToken(ctx, row)
	if err != nil {
		t.Fatalf("refreshGithubToken: %v", err)
	}
	if got.GithubToken != "static-token" {
		t.Fatalf("token = %q, want untouched static-token", got.GithubToken)
	}
}
