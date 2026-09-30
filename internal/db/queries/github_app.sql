-- name: UpsertGithubInstallation :one
-- Fired by the `installation` webhook event ("created"/"unsuspend") --
-- an ON CONFLICT update so a re-delivered event (GitHub retries) or a
-- reinstall after a prior uninstall converges instead of erroring.
INSERT INTO github_installation (installation_id, account_login, account_type)
VALUES ($1, $2, $3)
ON CONFLICT (installation_id) DO UPDATE SET
    account_login = EXCLUDED.account_login,
    account_type = EXCLUDED.account_type,
    suspended = false,
    updated_at = now()
RETURNING *;

-- name: SetGithubInstallationSuspended :one
UPDATE github_installation SET suspended = $2, updated_at = now() WHERE installation_id = $1 RETURNING *;

-- name: GetGithubInstallationByInstallationID :one
SELECT * FROM github_installation WHERE installation_id = $1;

-- name: ListGithubInstallations :many
-- Every installation this instance knows about, approved repos and all --
-- found missing by direct product feedback: the old "Installations"
-- screen only ever listed repositories with nothing approved yet, so an
-- installation with everything already approved (the common case once a
-- repo's been set up) simply vanished from the one screen meant to be
-- "everywhere we have the app installed... allow us to manage it."
SELECT * FROM github_installation ORDER BY account_login;

-- name: DeleteGithubInstallation :exec
-- Fired by `installation` "deleted" -- cascades to installation_repository
-- (ON DELETE CASCADE), so a full uninstall cleans up both tables in one
-- statement. Any `repository` row that was approved from this
-- installation keeps its own row (it's LaForge's own tracked content,
-- not GitHub's to delete) but its installation_id now points at nothing
-- -- see ListRepositoriesWithLostInstallation.
DELETE FROM github_installation WHERE installation_id = $1;

-- name: UpsertInstallationRepository :one
INSERT INTO installation_repository (installation_id, github_owner, github_repo, github_repo_id)
VALUES ($1, $2, $3, $4)
ON CONFLICT (installation_id, github_repo_id) DO UPDATE SET
    github_owner = EXCLUDED.github_owner, github_repo = EXCLUDED.github_repo
RETURNING *;

-- name: DeleteInstallationRepository :exec
DELETE FROM installation_repository WHERE installation_id = $1 AND github_repo_id = $2;

-- name: ListInstallationRepositoriesByInstallation :many
SELECT * FROM installation_repository WHERE installation_id = $1 ORDER BY github_owner, github_repo;

-- name: ListAllInstallationRepositories :many
-- The other half of ListGithubInstallations's own fix: every
-- installation_repository row, approved or not, so the Installations
-- screen can show a real per-repo approved/pending state instead of a
-- repo disappearing from the list the moment it's approved.
SELECT ir.installation_id, ir.github_owner, ir.github_repo, ir.github_repo_id,
       (r.id IS NOT NULL)::boolean AS approved, r.id AS repository_id
FROM installation_repository ir
LEFT JOIN repository r ON r.github_owner = ir.github_owner AND r.github_repo = ir.github_repo
ORDER BY ir.github_owner, ir.github_repo;

-- name: ListUnapprovedInstallationRepositories :many
-- "Installed but not yet approved": every installation_repository row
-- with no matching repository row (matched by owner/repo, the same pair
-- `repository`'s own unique constraint already keys on) -- exactly the
-- admin screen's own list, and the one query behind "install on GitHub,
-- approve in LaForge."
SELECT ir.*, gi.account_login, gi.account_type
FROM installation_repository ir
JOIN github_installation gi ON gi.id = ir.installation_id
LEFT JOIN repository r ON r.github_owner = ir.github_owner AND r.github_repo = ir.github_repo
WHERE r.id IS NULL
ORDER BY ir.github_owner, ir.github_repo;

-- name: ApproveInstalledRepository :one
-- The other half of ListUnapprovedInstallationRepositories: creates the
-- real repository row, tagged with which installation it was approved
-- from so its content can later be fetched through that installation's
-- own scoped token rather than a broader one.
INSERT INTO repository (github_owner, github_repo, installation_id)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetRepositoryInstallation :one
-- Resolves a repository's own installation row (for minting a scoped
-- installation token in internal/api/webhook.go's reconcile) --
-- pgx.ErrNoRows either means installation_id is null (never approved
-- through an installation) or the installation was since deleted; both
-- cases fall back to GITHUB_SERVICE_TOKEN the same way.
SELECT gi.* FROM github_installation gi
JOIN repository r ON r.installation_id = gi.id
WHERE r.id = $1;

-- name: GetGithubInstallation :one
SELECT * FROM github_installation WHERE id = $1;
