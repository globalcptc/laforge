-- name: CreateBuilderConfig :one
INSERT INTO builder_config (
    name, kind, incus_api_url, incus_client_cert_path, incus_client_key_path,
    incus_server_cert_pem, incus_ovn_uplink_network, incus_storage_pool,
    incus_operation_timeout_seconds, incus_images, incus_sizes, incus_hosts,
    incus_credential_id, external_access_ip, external_port_min, external_port_max,
    incus_project, microcloud_public_access
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18
) RETURNING *;

-- name: GetBuilderConfigByName :one
SELECT * FROM builder_config WHERE name = $1;

-- name: ListBuilderConfigs :many
SELECT * FROM builder_config ORDER BY name;

-- name: UpdateBuilderConfig :one
UPDATE builder_config SET
    kind = $2, incus_api_url = $3, incus_client_cert_path = $4, incus_client_key_path = $5,
    incus_server_cert_pem = $6, incus_ovn_uplink_network = $7, incus_storage_pool = $8,
    incus_operation_timeout_seconds = $9, incus_images = $10, incus_sizes = $11, incus_hosts = $12,
    incus_credential_id = $13, external_access_ip = $14, external_port_min = $15, external_port_max = $16,
    incus_project = $17, microcloud_public_access = $18, updated_at = now()
WHERE name = $1
RETURNING *;

-- name: DeleteBuilderConfig :exec
DELETE FROM builder_config WHERE name = $1;

-- name: CreateBuilderCredential :one
INSERT INTO builder_credential (
    api_url, server_name, server_fingerprint, server_cert_pem, client_cert_pem, client_key_pem
) VALUES (
    $1, $2, $3, $4, $5, $6
) RETURNING id, api_url, server_name, server_fingerprint, created_at;

-- name: GetBuilderCredential :one
SELECT * FROM builder_credential WHERE id = $1;

-- name: GetBuilderCredentialSummary :one
-- Everything about a credential except key material -- what the admin UI
-- shows for an already-connected host.
SELECT id, api_url, server_name, server_fingerprint, created_at FROM builder_credential WHERE id = $1;
