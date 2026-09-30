-- Persist a host/container's authored tags onto its runtime deployed_object
-- rows so ad-hoc and scheduled tasks can TARGET by tag, not just team/kind/name
-- ("the whole point of tags is to search AND
-- operate on them"). Tags were already on the content host/container rows and
-- exposed for the Hosts page's search; this carries them onto the deployed
-- objects an operator actually acts on. The reconciler sets them every pass, so
-- an edited tag propagates on the next converge. Never part of the build
-- fingerprint (tags never reach a script), so this drives no redeploy.

-- +goose Up

ALTER TABLE deployed_object ADD COLUMN tags jsonb NOT NULL DEFAULT '{}'::jsonb;

-- +goose Down

ALTER TABLE deployed_object DROP COLUMN tags;
