-- +goose Up
-- The environment's overall address space (Environment.CIDR in the YAML): the
-- supernet every network in the environment must fit inside. Builder-agnostic
-- and optional -- the on-prem builders (Incus/MicroCloud) ignore it; a cloud
-- builder that hosts a whole team in one VPC (AWS) uses it as the VPC CIDR.
-- Mirrors the YAML field into the environment row like every other scalar;
-- NULL for an environment that declares none.
ALTER TABLE environment ADD COLUMN cidr text;

-- +goose Down
ALTER TABLE environment DROP COLUMN cidr;
