-- +goose Up
-- AWS credentials are entered in the builder wizard and stored like an Incus
-- client credential: in builder_credential, referenced from builder_config by
-- id, and never returned to the UI after enrollment (the summary row omits the
-- secret). builder_credential was cert-shaped for Incus; these two columns hold
-- an AWS access key instead (the cert columns stay '' for an AWS row).
ALTER TABLE builder_credential ADD COLUMN aws_access_key_id text NOT NULL DEFAULT '';
ALTER TABLE builder_credential ADD COLUMN aws_secret_access_key text NOT NULL DEFAULT '';

-- The AWS builder's enrolled credential, mirroring incus_credential_id.
ALTER TABLE builder_config ADD COLUMN aws_credential_id uuid REFERENCES builder_credential(id);

-- +goose Down
ALTER TABLE builder_config DROP COLUMN aws_credential_id;
ALTER TABLE builder_credential DROP COLUMN aws_secret_access_key;
ALTER TABLE builder_credential DROP COLUMN aws_access_key_id;
