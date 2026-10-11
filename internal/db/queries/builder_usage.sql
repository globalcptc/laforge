-- Builder usage summary for the builder page: the active builds running on a
-- builder, the resources they have deployed (by kind/status), and the public
-- endpoints they expose. A build belongs to a builder via its configured_build.
-- "Active" = live states (resources exist on the hoster). See
-- docs/plans/builder-monitoring.md.

-- name: ListActiveBuildsByBuilder :many
SELECT b.id, b.status, b.environment_name, b.created_at
FROM build b
JOIN configured_build cb ON cb.id = b.configured_build_id
WHERE cb.builder_config_name = $1
  AND b.status IN ('deploying', 'building', 'finished', 'tearing_down')
ORDER BY b.created_at DESC;

-- name: CountDeployedObjectsByBuilder :many
SELECT o.kind, o.status, count(*)::bigint AS n
FROM deployed_object o
JOIN team t ON t.id = o.team_id
JOIN build b ON b.id = t.build_id
JOIN configured_build cb ON cb.id = b.configured_build_id
WHERE cb.builder_config_name = $1
  AND b.status IN ('deploying', 'building', 'finished', 'tearing_down')
GROUP BY o.kind, o.status;

-- name: ListPublicEndpointsByBuilder :many
SELECT ea.public_address, ea.protocol, ea.external_port,
       o.object_name, o.as_name, t.team_number, b.id AS build_id
FROM external_access ea
JOIN deployed_object o ON o.id = ea.deployed_object_id
JOIN team t ON t.id = o.team_id
JOIN build b ON b.id = t.build_id
JOIN configured_build cb ON cb.id = b.configured_build_id
WHERE cb.builder_config_name = $1
  AND b.status IN ('deploying', 'building', 'finished')
ORDER BY t.team_number, o.object_name, ea.external_port;
