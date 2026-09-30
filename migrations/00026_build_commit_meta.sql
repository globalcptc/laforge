-- Commit metadata for the builds tables, and an auto-built marker.
--
-- The configured-builds page shows a build's tracked commit, but only the SHA
-- was ever stored. These add the commit's subject and committer time (read
-- from the fetched checkout at ingest), so a build row can show "what commit
-- is this" -- hash, message, and time -- rather than an opaque SHA. All three
-- are nullable: pre-existing revisions have none, and a revision that couldn't
-- be read still records its SHA.
--
-- build.auto_built records whether a build was created automatically off a
-- CI-passing push (repository auto-deploy) or by a deliberate "Build Now"
-- click, so the UI can flag the automatic ones. Defaults false (a manual
-- build), which is also the safe assumption for every build that predates this.

-- +goose Up

ALTER TABLE content_revision ADD COLUMN commit_message text;
ALTER TABLE content_revision ADD COLUMN commit_author text;
ALTER TABLE content_revision ADD COLUMN committed_at timestamptz;
ALTER TABLE build ADD COLUMN auto_built boolean NOT NULL DEFAULT false;

-- +goose Down

ALTER TABLE build DROP COLUMN auto_built;
ALTER TABLE content_revision DROP COLUMN committed_at;
ALTER TABLE content_revision DROP COLUMN commit_author;
ALTER TABLE content_revision DROP COLUMN commit_message;
