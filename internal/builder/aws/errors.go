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

// isDuplicate is the "this ingress rule already exists" case, so a repeated
// OpenAccess converges instead of erroring.
func isDuplicate(err error) bool {
	return apiErrorCode(err) == "InvalidPermission.Duplicate"
}
