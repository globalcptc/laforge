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
