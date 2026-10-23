-- Gate the once-per-resolve `build.resolved` timeline event. The orchestrator
-- loads + resolves + renders a build's whole content every reconcile pass (every
-- 2s), but that work was invisible on the build's event log -- a freshly
-- triggered build "just sat there" for however long the (cold) checkout + resolve
-- took, with nothing to show what was happening. resolved_revision_id records the
-- content revision the orchestrator last emitted a build.resolved event for, so
-- that event fires once on the first resolve, and again only when the content
-- revision changes (a rebuild to a new commit) -- never every tick.

-- +goose Up
ALTER TABLE build ADD COLUMN resolved_revision_id uuid;

-- +goose Down
ALTER TABLE build DROP COLUMN resolved_revision_id;
