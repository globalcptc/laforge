package aws

import (
	"context"
	"fmt"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sts"

	"github.com/globalcptc/laforge/internal/builder"
)

// AWS onboarding enrolls an access key entered in the builder wizard: it
// validates the key with an STS GetCallerIdentity call, then returns it as the
// Connection to store (in builder_credential, like an Incus client credential --
// the wizard only ever sees the resulting credential id, never the secret
// again). If no key is supplied it falls back to the ambient environment chain
// (a runner instance-profile role), validating that instead, so a connect from
// an older UI or an instance-role setup still works. Images (AMIs) and sizes
// (instance types) are entered in the wizard as identifiers; there is no
// interactive resource discovery here yet.
func init() { builder.RegisterOnboarder("aws", onboarder{}) }

type onboarder struct{}

func (onboarder) Onboard(ctx context.Context, req builder.OnboardRequest) (*builder.Onboarding, error) {
	if req.Token != "" {
		return nil, fmt.Errorf("%w: aws builders authenticate with an access key or an instance role, not a trust token", builder.ErrInvalidOnboardRequest)
	}
	// Only one of the pair supplied is a user error worth catching early.
	if (req.AWSAccessKeyID == "") != (req.AWSSecretAccessKey == "") {
		return nil, fmt.Errorf("%w: provide both an AWS access key id and secret access key, or neither (to use the runner's instance role)", builder.ErrInvalidOnboardRequest)
	}
	// STS (like every v2 client) needs a region to resolve its endpoint, even
	// though GetCallerIdentity is global. The wizard picks the real region in a
	// later step, so for this credential check just use AWS_REGION if the runner
	// has one, else a default -- the identity answer is the same from any region.
	opts := []func(*awsconfig.LoadOptions) error{awsconfig.WithRegion(envOr("AWS_REGION", "us-east-1"))}
	if req.AWSAccessKeyID != "" {
		opts = append(opts, awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(req.AWSAccessKeyID, req.AWSSecretAccessKey, "")))
	}
	cfg, err := awsconfig.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("%w: loading AWS credentials: %v", builder.ErrInvalidOnboardRequest, err)
	}
	if _, err := sts.NewFromConfig(cfg).GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{}); err != nil {
		return nil, fmt.Errorf("%w: those AWS credentials are not usable: %v", builder.ErrInvalidOnboardRequest, err)
	}
	// An explicit key is stored; an instance-role connect stores nothing.
	conn := builder.Connection{AWSAccessKeyID: req.AWSAccessKeyID, AWSSecretAccessKey: req.AWSSecretAccessKey}
	return &builder.Onboarding{Connection: conn}, nil
}

// Rediscover is implemented in discover.go.
