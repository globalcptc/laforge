package builder

import "errors"

// ErrPermanentConfig marks a failure that retrying cannot fix because it's a
// property of the configuration itself (e.g. a project name so long that no
// valid instance name fits the hoster's socket-path limit). The runner fails a
// task carrying this immediately instead of burning its retry budget on a
// deterministic failure. Wrap it with %w and a specific, actionable message.
var ErrPermanentConfig = errors.New("permanent configuration error")
