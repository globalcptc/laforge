-- The LXD project a MicroCloud builder creates everything in -- instances,
-- team networks, network ACLs, images, config-drive volumes. NULL or '' is the
-- `default` project, which is what every builder used before this existed. A
-- hosting provider may not let LaForge use `default`.

-- +goose Up

ALTER TABLE builder_config ADD COLUMN incus_project text;

-- +goose Down

ALTER TABLE builder_config DROP COLUMN incus_project;
