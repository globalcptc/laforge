package aws

import (
	"errors"
	"testing"

	"github.com/aws/smithy-go"
)

// apiErr builds an AWS-style API error carrying a specific error code, the shape
// the classifiers below match on (via smithy.APIError / errors.As).
func apiErr(code string) error {
	return &smithy.GenericAPIError{Code: code, Message: "test"}
}

// TestDestroyErrorClassifiers pins the "already gone / retry later" rules the
// no-op-safe teardown relies on: a malformed id or not-found means the resource
// never existed (or is already deleted), a missing ECS cluster means there's
// nothing to stop, and a dependency violation is a transient "retry" -- none of
// which should wedge a teardown.
func TestDestroyErrorClassifiers(t *testing.T) {
	cases := []struct {
		name                                         string
		err                                          error
		malformed, notFound, clusterGone, dependency bool
	}{
		{"instance malformed", apiErr("InvalidInstanceID.Malformed"), true, false, false, false},
		{"subnet malformed", apiErr("InvalidSubnetId.Malformed"), true, false, false, false},
		{"instance not found", apiErr("InvalidInstanceID.NotFound"), false, true, false, false},
		{"group not found", apiErr("InvalidGroup.NotFound"), false, true, false, false},
		{"cluster not found", apiErr("ClusterNotFoundException"), false, false, true, false},
		{"dependency violation", apiErr("DependencyViolation"), false, false, false, true},
		{"unrelated error", errors.New("boom"), false, false, false, false},
		{"throttling is none of these", apiErr("Throttling"), false, false, false, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := isMalformed(c.err); got != c.malformed {
				t.Errorf("isMalformed = %v, want %v", got, c.malformed)
			}
			if got := isNotFound(c.err); got != c.notFound {
				t.Errorf("isNotFound = %v, want %v", got, c.notFound)
			}
			if got := isECSClusterNotFound(c.err); got != c.clusterGone {
				t.Errorf("isECSClusterNotFound = %v, want %v", got, c.clusterGone)
			}
			if got := isDependencyViolation(c.err); got != c.dependency {
				t.Errorf("isDependencyViolation = %v, want %v", got, c.dependency)
			}
		})
	}
}
