-- DNS is no longer a builder-deployed object. LaForge doesn't run DNS; the
-- resolved records are exposed in the rendered template data for a script on the
-- Domain Controller / Bind host to consume (render.Context.TemplateData). The
-- "dns" deployed_object kind, the builder DeployDNSRecords method, and
-- reconcileDNS are all removed. Delete any existing "dns" rows (and their
-- tasks/events) so the reconciler doesn't try to destroy them through a builder
-- path that no longer exists. The kind CHECK constraint still permits 'dns'
-- (harmless -- nothing creates one anymore); not tightened here to avoid failing
-- on any in-flight row.

-- +goose Up

DELETE FROM event WHERE task_id IN (
    SELECT id FROM task WHERE deployed_object_id IN (
        SELECT id FROM deployed_object WHERE kind = 'dns'
    )
);
DELETE FROM task WHERE deployed_object_id IN (
    SELECT id FROM deployed_object WHERE kind = 'dns'
);
DELETE FROM deployed_object WHERE kind = 'dns';

-- +goose Down

-- No-op: dns deployed_objects are intentionally not recreated (DNS left the
-- builder path entirely).
SELECT 1;
