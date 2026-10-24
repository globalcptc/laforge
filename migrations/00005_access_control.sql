-- The access model: "GitHub is the only identity and
-- permission system... No local accounts, no internal permission tables"
-- for WHO EXISTS (that's GitHub's job, checked live against the GitHub
-- API the same way internal/api/auth.go's requirePush already does for
-- the CLI/webhook path) -- but a browser UI needs a real signed-in
-- session, and "LaForge decides what they can do to a running
-- competition" (four levels: read/build/manage/admin, per repository)
-- does need a table, since it's LaForge's own authorization decision, not
-- GitHub's.
--
-- Named `account`, not `person`: the `person` table already
-- exists and means something entirely different (a row from a content
-- repo's people CSV -- a game character, part of an event's cast). This
-- is the signed-in LaForge operator's own identity; conflating the two
-- names would have been a real, confusing bug waiting to happen.
--
-- account is deliberately thin: just enough of a GitHub identity to show
-- "signed in as @lucas" and to key sessions/access grants on, not a user
-- profile system. Re-fetched from GitHub on each login rather than kept
-- in sync some other way.

-- +goose Up

CREATE TABLE account (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    github_id    bigint NOT NULL UNIQUE,
    github_login text NOT NULL,
    avatar_url   text,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now()
);

-- A signed-in browser session. token is a random opaque value; only its
-- SHA-256 hash is stored (same discipline as not storing GitHub tokens in
-- the clear anywhere -- see auth.go), so a leaked database dump alone
-- can't be replayed as a live cookie. github_token is stored (not
-- hashed): it's needed on every subsequent request to act on the user's
-- own GitHub permissions exactly like the CLI's bearer-token path does,
-- so this session IS the credential, not a reference to one kept
-- elsewhere. Sessions are bounded-lived and re-issued by signing in
-- again, not refreshed silently.
CREATE TABLE session (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id    uuid NOT NULL REFERENCES account(id) ON DELETE CASCADE,
    token_hash    text NOT NULL UNIQUE,
    github_token  text NOT NULL,
    created_at    timestamptz NOT NULL DEFAULT now(),
    expires_at    timestamptz NOT NULL
);

CREATE INDEX session_expires_idx ON session (expires_at);

-- "Four per-repository access levels: read, build, manage, admin. GitHub
-- decides who exists; LaForge decides what they can do to a running
-- competition." An account with no row here but real GitHub push access
-- still gets `build` implicitly (see internal/api's authorization code --
-- this table is for granting MORE than GitHub push implies, e.g. `manage`
-- or `admin`, and for granting access to people who can read a private
-- repo's results without being able to push). levels are not a strict
-- hierarchy in storage (an account could theoretically be granted
-- `manage` without `build`), but internal/api's authorization enforces
-- the plan's intended cascade ("each including the ones before it").
CREATE TABLE repository_access (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    repository_id  uuid NOT NULL REFERENCES repository(id) ON DELETE CASCADE,
    account_id     uuid NOT NULL REFERENCES account(id) ON DELETE CASCADE,
    level          text NOT NULL CHECK (level IN ('read', 'build', 'manage', 'admin')),
    granted_by     uuid REFERENCES account(id),
    created_at     timestamptz NOT NULL DEFAULT now(),
    UNIQUE (repository_id, account_id)
);

-- +goose Down

DROP TABLE repository_access;
DROP TABLE session;
DROP TABLE account;
