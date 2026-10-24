-- Push live-event notifications from Postgres itself, so the SSE `/live`
-- endpoint can LISTEN and forward new events instantly instead of polling the
-- event table on a 1-second loop. An AFTER INSERT trigger on `event` fires
-- pg_notify on channel 'laforge_event' with the build id, so a listener knows
-- exactly which build got a new event and queries only then.

-- +goose Up

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION laforge_notify_event() RETURNS trigger AS $$
BEGIN
  PERFORM pg_notify('laforge_event', NEW.build_id::text);
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER event_notify AFTER INSERT ON event
  FOR EACH ROW EXECUTE FUNCTION laforge_notify_event();

-- +goose Down

DROP TRIGGER IF EXISTS event_notify ON event;
DROP FUNCTION IF EXISTS laforge_notify_event();
