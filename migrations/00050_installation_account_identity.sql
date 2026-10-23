-- +goose Up
-- A GitHub App is installed at most once per account (org or user), so the
-- ACCOUNT -- not the numeric installation_id -- is the stable identity of
-- "this org's installation of LaForge". Keying UpsertGithubInstallation on
-- installation_id meant a reinstall (which gets a brand-new installation_id)
-- INSERTED a second row instead of updating the existing one, stranding every
-- repository still pointing at the old row's uuid and breaking token minting
-- (GitHub 404s the dead installation id). A unique constraint on account_login
-- lets the upsert converge on the existing row and refresh its installation_id
-- in place, keeping the uuid -- and therefore every repository FK -- stable.

-- First collapse any duplicate rows for the same account that the old,
-- installation_id-keyed upsert may already have produced: keep the most
-- recently updated row, move its siblings' repositories and
-- installation_repository rows onto it, then drop the siblings. Without this
-- the UNIQUE constraint below would fail to apply on an affected database.
-- +goose StatementBegin
DO $$
DECLARE
    acct text;
    keep uuid;
BEGIN
    FOR acct IN
        SELECT account_login FROM github_installation GROUP BY account_login HAVING count(*) > 1
    LOOP
        SELECT id INTO keep FROM github_installation
            WHERE account_login = acct
            ORDER BY updated_at DESC, created_at DESC
            LIMIT 1;
        UPDATE repository SET installation_id = keep
            WHERE installation_id IN (
                SELECT id FROM github_installation WHERE account_login = acct AND id <> keep);
        DELETE FROM installation_repository
            WHERE installation_id IN (
                SELECT id FROM github_installation WHERE account_login = acct AND id <> keep);
        DELETE FROM github_installation WHERE account_login = acct AND id <> keep;
    END LOOP;
END $$;
-- +goose StatementEnd

ALTER TABLE github_installation ADD CONSTRAINT github_installation_account_login_key UNIQUE (account_login);

-- +goose Down
ALTER TABLE github_installation DROP CONSTRAINT github_installation_account_login_key;
