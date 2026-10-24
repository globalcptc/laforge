package openstack

import (
	"context"
	"fmt"

	"github.com/globalcptc/laforge/internal/builder"
)

// OpenStack onboarding is env-credential based, not trust-token based: an
// OpenStack builder authenticates from the standard OS_* environment chain on
// the runner (see resolveOpenStack), so there is no per-hoster secret to enroll
// and store, and no interactive resource discovery here yet -- images (Glance)
// and sizes (flavors) are entered in the wizard as identifiers. This onboarder
// makes that explicit and keeps the per-type contract complete.
func init() { builder.RegisterOnboarder("openstack", onboarder{}) }

type onboarder struct{}

func (onboarder) Onboard(ctx context.Context, req builder.OnboardRequest) (*builder.Onboarding, error) {
	if req.Token != "" {
		return nil, fmt.Errorf("%w: openstack builders authenticate from environment credentials (OS_*), not a trust token", builder.ErrInvalidOnboardRequest)
	}
	return &builder.Onboarding{}, nil
}

func (onboarder) Rediscover(ctx context.Context, conn builder.Connection) (*builder.Discovery, error) {
	return &builder.Discovery{}, nil
}
