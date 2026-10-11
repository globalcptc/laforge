package builder

import "errors"

// ErrExternalAccessNotReady defers forwarding without consuming a failure attempt.
// The runner must retry after the hoster reports that the instance is running.
var ErrExternalAccessNotReady = errors.New("instance is not running yet")

func ExternalAccessDeployReady(status string) bool {
	switch status {
	case "running", "building", "finished", "deployed", "build_failed", "invalid":
		return true
	default:
		return false
	}
}
