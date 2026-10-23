-- +goose Up
-- A rebuild asked for while the object already has a task in flight (a deploy
-- being created right now, an earlier destroy) can't queue its destroy yet
-- (one open task per object). It used to reset the object anyway, losing the
-- hoster reference, and the request was then forgotten: the in-flight deploy
-- finished and the host the operator wanted rebuilt simply stayed. The
-- request is now recorded here and carried out by the reconciler as soon as
-- the object has no open task; it's cleared when the destroy is queued.
ALTER TABLE deployed_object ADD COLUMN rebuild_requested_at timestamptz;

-- +goose Down
ALTER TABLE deployed_object DROP COLUMN rebuild_requested_at;
