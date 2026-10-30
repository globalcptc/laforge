-- +goose Up
-- The content revision each object's instance was deployed from. Reconcile
-- recreates an up object for a fingerprint change only when the build has since
-- moved to a different revision (new content was applied): under the same
-- revision a differing fingerprint can't be a content change, so it's reported
-- and the instance is left alone rather than destroyed.
ALTER TABLE deployed_object ADD COLUMN content_revision_id uuid REFERENCES content_revision(id) ON DELETE SET NULL;

-- What's up now was deployed from its build's current revision.
UPDATE deployed_object o SET content_revision_id = b.content_revision_id
FROM team t JOIN build b ON b.id = t.build_id
WHERE t.id = o.team_id AND o.status NOT IN ('pending', 'destroyed');

-- +goose Down
ALTER TABLE deployed_object DROP COLUMN content_revision_id;
