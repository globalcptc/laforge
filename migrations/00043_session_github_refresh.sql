-- Keep a browser session's GitHub token refreshed. A LaForge session lasts a
-- week (sessionTTL), but the GitHub App's user access token expires in ~8h and
-- comes with a refresh token (~6 months). We were discarding the refresh token
-- and storing no expiry, so once the access token lapsed, every call that used
-- it (repo access checks, collaborator lookups) failed oddly while the person
-- still appeared logged in. These columns let the api refresh the access token
-- on use. All nullable / default-empty so sessions created before this keep
-- working (just un-refreshable, as before).

-- +goose Up
ALTER TABLE session ADD COLUMN github_token_expires_at timestamptz;
ALTER TABLE session ADD COLUMN github_refresh_token text NOT NULL DEFAULT '';
ALTER TABLE session ADD COLUMN github_refresh_expires_at timestamptz;

-- +goose Down
ALTER TABLE session DROP COLUMN github_refresh_expires_at;
ALTER TABLE session DROP COLUMN github_refresh_token;
ALTER TABLE session DROP COLUMN github_token_expires_at;
