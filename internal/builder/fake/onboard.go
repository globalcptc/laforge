package fake

import (
	"context"

	"github.com/globalcptc/laforge/internal/builder"
)

// The fake builder onboards trivially: it simulates its hoster in Postgres, so
// there's nothing to connect to and nothing to discover. Registered for
// contract completeness (and so tests that exercise the onboarding registry have
// a no-op kind), even though the wizard doesn't offer fake as a choice.
func init() { builder.RegisterOnboarder("fake", onboarder{}) }

type onboarder struct{}

func (onboarder) Onboard(ctx context.Context, req builder.OnboardRequest) (*builder.Onboarding, error) {
	return &builder.Onboarding{}, nil
}

func (onboarder) Rediscover(ctx context.Context, conn builder.Connection) (*builder.Discovery, error) {
	return &builder.Discovery{}, nil
}
