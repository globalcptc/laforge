package api

import "testing"

// TestShellReachable pins which deploy states can serve a shell: a box that is
// up (running/building/finished), never one still deploying, failed, or gone.
func TestShellReachable(t *testing.T) {
	reachable := []string{"running", "building", "finished"}
	for _, s := range reachable {
		if !shellReachable(s) {
			t.Errorf("shellReachable(%q) = false, want true", s)
		}
	}
	unreachable := []string{"pending", "deploying", "deploy_failed", "build_failed", "invalid", "destroying", "destroyed", ""}
	for _, s := range unreachable {
		if shellReachable(s) {
			t.Errorf("shellReachable(%q) = true, want false", s)
		}
	}
}
