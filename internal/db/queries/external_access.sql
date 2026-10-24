-- name: UpsertExternalAccess :exec
-- Record (or refresh) the external endpoint a builder assigned for one host's
-- public port. The builder allocates against its own hoster (Incus network
-- forwards, an AWS public IP, ...), so this table is purely the surfaced result
-- the UI/CLI read -- never the allocator. Idempotent per (object, protocol,
-- internal_port).
INSERT INTO external_access (deployed_object_id, protocol, internal_port, public_address, external_port)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (deployed_object_id, protocol, internal_port)
DO UPDATE SET public_address = EXCLUDED.public_address, external_port = EXCLUDED.external_port, updated_at = now();

-- name: ListExternalAccessByObject :many
SELECT * FROM external_access WHERE deployed_object_id = $1 ORDER BY protocol, internal_port;

-- name: ListExternalAccessByBuild :many
-- Every external endpoint in a build, with the team/host it belongs to -- the
-- build-wide "how do I reach each team's hosts" view (one shared uplink IP with
-- per-team ports on Incus/MicroCloud, a public IP per host on AWS).
SELECT ea.deployed_object_id, ea.protocol, ea.internal_port, ea.public_address, ea.external_port,
       t.team_number, o.object_name, o.as_name, o.kind
FROM external_access ea
JOIN deployed_object o ON o.id = ea.deployed_object_id
JOIN team t ON t.id = o.team_id
WHERE t.build_id = $1
ORDER BY t.team_number, o.object_name, ea.protocol, ea.internal_port;

-- name: DeleteExternalAccessForObject :exec
-- Clear an object's recorded endpoints before re-realizing them (or when it is
-- torn down for redeploy). A hard object delete cascades these away anyway.
DELETE FROM external_access WHERE deployed_object_id = $1;
