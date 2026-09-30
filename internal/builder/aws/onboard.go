package aws

import (
	"context"
	"fmt"

	"github.com/globalcptc/laforge/internal/builder"
)

// AWS onboarding is env-credential based, not trust-token based: an AWS builder
// authenticates from the standard AWS_* environment chain on the runner (see
// resolveAWS), so there is no per-hoster secret to enroll and store, and no
// interactive resource discovery here yet -- images (AMIs) and sizes (instance
// types) are entered in the wizard as identifiers. This onboarder makes that
// explicit and keeps the per-type contract complete, rather than the connect
// endpoint silently having no answer for kind "aws".
func init() { builder.RegisterOnboarder("aws", onboarder{}) }

type onboarder struct{}

func (onboarder) Onboard(ctx context.Context, req builder.OnboardRequest) (*builder.Onboarding, error) {
	if req.Token != "" {
		return nil, fmt.Errorf("%w: aws builders authenticate from environment credentials (AWS_*), not a trust token", builder.ErrInvalidOnboardRequest)
	}
	// Nothing to store (env credentials) and no interactive discovery yet.
	return &builder.Onboarding{}, nil
}

func (onboarder) Rediscover(ctx context.Context, conn builder.Connection) (*builder.Discovery, error) {
	return &builder.Discovery{}, nil
}
