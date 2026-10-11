package aws

import (
	"errors"
	"strings"

	"github.com/aws/smithy-go"
)

// EC2 signals "the thing you're deleting/revoking is already gone" and
// "you're adding a rule that already exists" through API error codes, not
// HTTP status alone. These keep the ensure-semantics (a second destroy, or a
// repeated OpenAccess, is a success) without matching on message text.

func apiErrorCode(err error) string {
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		return apiErr.ErrorCode()
	}
	return ""
}

// isNotFound covers the family of "...NotFound" codes (InvalidVpcID.NotFound,
// InvalidInstanceID.NotFound, InvalidGroup.NotFound, InvalidPermission.NotFound,
// ...) plus the malformed variant AWS sometimes uses for an already-deleted id.
func isNotFound(err error) bool {
	code := apiErrorCode(err)
	return strings.HasSuffix(code, ".NotFound") ||
		strings.HasSuffix(code, "NotFound") ||
		code == "InvalidPermission.NotFound"
}

// isDuplicate covers the "this already exists" codes so a repeated ensure
// converges instead of erroring: a duplicate ingress rule, an IGW already
// attached, a route already present, a NACL entry already there, a VPC CIDR
// already associated.
func isDuplicate(err error) bool {
	switch apiErrorCode(err) {
	case "InvalidPermission.Duplicate",
		"Resource.AlreadyAssociated",
		"RouteAlreadyExists",
		"NetworkAclEntryAlreadyExists",
		"InvalidVpcCidrBlock.DuplicateCidr",
		"CidrConflict",
		"Gateway.AlreadyAttached",
		"Resource.AlreadyExists":
		return true
	}
	return false
}

// alreadyAssociated is the "subnet is already associated with this route table"
// case, so re-associating on a retry is a no-op success.
func alreadyAssociated(err error) bool {
	return apiErrorCode(err) == "Resource.AlreadyAssociated"
}

// isDependencyViolation is AWS's "can't delete this yet, something still
// references it" -- a security group still attached to a terminating instance's
// (or stopping Fargate task's) ENI, or a VPC still holding resources. It's
// transient during teardown: the referencing resource is on its way out, so a
// caller treats it as "retry later," not a hard failure, and the task's existing
// retry/backoff finishes the teardown once the reference is gone.
func isDependencyViolation(err error) bool {
	return apiErrorCode(err) == "DependencyViolation"
}

// isMalformed is AWS's "that isn't even a valid id" (InvalidInstanceID.Malformed,
// InvalidSubnetId.Malformed, ...). During teardown it means we were handed a
// deterministic ExternalName instead of a real AWS id -- because the object's
// deploy failed before it ever recorded one -- so there is nothing to delete and
// the destroy treats it as already gone.
func isMalformed(err error) bool {
	return strings.HasSuffix(apiErrorCode(err), ".Malformed")
}

// isECSClusterNotFound is ECS's "no such cluster": when the configured cluster is
// wrong or gone, a StopTask has nothing to stop, so teardown treats it as done
// rather than wedging the build forever on a config mistake.
func isECSClusterNotFound(err error) bool {
	return apiErrorCode(err) == "ClusterNotFoundException"
}
