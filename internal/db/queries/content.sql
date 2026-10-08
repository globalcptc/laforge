-- name: CreateContentRevision :one
INSERT INTO content_revision (repository_id, commit_sha, ref, commit_message, commit_author, committed_at)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetContentRevisionByRepoAndSHA :one
SELECT * FROM content_revision WHERE repository_id = $1 AND commit_sha = $2;

-- name: GetContentRevision :one
SELECT * FROM content_revision WHERE id = $1;

-- name: GetLatestContentRevisionByRepository :one
-- "The user database from this repo's CSVs" (People) is
-- scoped to a repository, not any one build -- this is what "latest" means
-- absent an explicit ?revision= override: the most recently ingested
-- commit, valid or not (an invalid commit's people/findings are still
-- real content worth looking at while someone's fixing the validation
-- error, same reasoning as showing a failed build's rendered scripts).
SELECT * FROM content_revision WHERE repository_id = $1 ORDER BY created_at DESC LIMIT 1;

-- name: SetContentRevisionValidation :one
UPDATE content_revision
SET valid = $2, validation_errors = $3, validated_at = now()
WHERE id = $1
RETURNING *;

-- name: CreateEnvironment :one
INSERT INTO environment (
  content_revision_id, path, name, schema_version, description, teams,
  root_password, start_at, stop_at, dns, access,
  vars, tags, findings, extends, agent_debug, container_logs
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17
) RETURNING *;

-- name: GetEnvironmentContainerLogsForObject :one
-- The container_logs config (driver + options, as jsonb) for the environment a
-- deployed object belongs to -- object -> team -> build -> environment, same
-- walk as GetEnvironmentAgentDebugForObject. The runner reads it to apply the
-- builder's native log driver to a container. NULL when the environment sets no
-- container_logs.
SELECT e.container_logs
FROM deployed_object o
JOIN team t ON t.id = o.team_id
JOIN build b ON b.id = t.build_id
JOIN environment e ON e.content_revision_id = b.content_revision_id AND e.name = b.environment_name
WHERE o.id = $1;

-- name: GetEnvironmentAgentDebugForObject :one
-- The agent-debug flag for the environment a deployed object belongs to, found
-- by walking object -> team -> build -> environment (build keys the environment
-- by content_revision_id + name). Used by the runner when it plants an agent,
-- so the flag is baked into that host's binary. Defaults are such that any gap
-- in the chain simply yields no row (treated as debug off).
SELECT e.agent_debug
FROM deployed_object o
JOIN team t ON t.id = o.team_id
JOIN build b ON b.id = t.build_id
JOIN environment e ON e.content_revision_id = b.content_revision_id AND e.name = b.environment_name
WHERE o.id = $1;

-- name: ListEnvironmentsByRevision :many
SELECT * FROM environment WHERE content_revision_id = $1 ORDER BY name;

-- name: GetEnvironmentByRevisionAndName :one
SELECT * FROM environment WHERE content_revision_id = $1 AND name = $2;

-- name: GetEnvironmentByRevisionAndPath :one
-- configured_build.environment_path is a file path ("lm-test.yaml"), not
-- an environment name -- this is how handleTriggerBuild resolves one to
-- the other to fill in build.environment_name, mirroring how
-- orchestrator.findEnvironment and every DB-side lookup key on name, not
-- path.
SELECT * FROM environment WHERE content_revision_id = $1 AND path = $2;

-- name: CreateNetwork :one
INSERT INTO network (content_revision_id, path, name, cidr, visible_from, vars, tags, findings, extends)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;

-- name: GetNetworkByRevisionAndName :one
SELECT * FROM network WHERE content_revision_id = $1 AND name = $2;

-- name: CreateHost :one
INSERT INTO host (content_revision_id, path, name, os, size, disk_gb, ports, depends_on, steps, schedule, vars, tags, findings, people, extends)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
RETURNING *;

-- name: GetHostByRevisionAndName :one
SELECT * FROM host WHERE content_revision_id = $1 AND name = $2;

-- name: CreateContainer :one
INSERT INTO container (content_revision_id, path, name, image, size, ports, depends_on, steps, schedule, vars, tags, findings, people, extends)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
RETURNING *;

-- name: GetContainerByRevisionAndName :one
SELECT * FROM container WHERE content_revision_id = $1 AND name = $2;

-- name: CreateScript :one
INSERT INTO script (content_revision_id, path, name, description, language, source_path, timeout_seconds, args, ignore_errors, tags, findings, people, validate)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
RETURNING *;

-- name: GetScriptByRevisionAndName :one
SELECT * FROM script WHERE content_revision_id = $1 AND name = $2;

-- name: CreatePeopleSource :one
INSERT INTO people_source (content_revision_id, path, name)
VALUES ($1, $2, $3)
RETURNING *;

-- name: ListPeopleSourcesByRevision :many
SELECT * FROM people_source WHERE content_revision_id = $1 ORDER BY name;

-- name: CreatePerson :one
INSERT INTO person (people_source_id, username, attributes)
VALUES ($1, $2, $3)
RETURNING *;

-- name: ListPersonByRevision :many
-- Every person from every people CSV in one content revision -- "the user
-- database from this repo's CSVs," searchable (the People
-- screen). people_source_name lets the UI show which CSV a row came from
-- without a second round trip.
SELECT person.*, people_source.name AS people_source_name
FROM person
JOIN people_source ON people_source.id = person.people_source_id
WHERE people_source.content_revision_id = $1
ORDER BY people_source.name, person.username;

-- name: CreatePlacement :one
INSERT INTO placement (environment_id, network_name, object_kind, object_name, as_name, last_octet, public)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;
