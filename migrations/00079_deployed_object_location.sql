-- +goose Up
-- Where the hoster placed each object: the server (an Incus pool host) or
-- cluster member its instance runs on, as the hoster names it. Read by the
-- power-state poll; '' until a poll has reported it, and cleared when the
-- instance is redeployed or destroyed, since a new instance may land elsewhere.
ALTER TABLE deployed_object ADD COLUMN instance_location text NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE deployed_object DROP COLUMN instance_location;
