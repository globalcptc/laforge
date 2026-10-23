-- +goose Up
-- A user-chosen display color for a builder, shown as a colored pill wherever the
-- builder's name appears (builder list, info page, home usage, build rows). Empty
-- string = no color chosen (the UI falls back to a neutral pill). A hex string
-- like "#4c63b0"; the UI validates/picks it.
ALTER TABLE builder_config ADD COLUMN color text NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE builder_config DROP COLUMN color;
