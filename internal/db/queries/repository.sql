-- name: CreateRepository :one
INSERT INTO repository (github_owner, github_repo)
VALUES ($1, $2)
RETURNING *;

-- name: GetRepository :one
SELECT * FROM repository WHERE id = $1;

-- name: GetRepositoryByOwnerRepo :one
SELECT * FROM repository WHERE github_owner = $1 AND github_repo = $2;

-- name: ListRepositories :many
SELECT * FROM repository ORDER BY github_owner, github_repo;


-- name: ListRepositoriesByInstallation :many
SELECT * FROM repository WHERE installation_id = $1 ORDER BY github_owner, github_repo;
