package incus

import (
	"context"
	"fmt"

	"github.com/globalcptc/laforge/internal/builder"
)

// The Incus builder's onboarding: enrolling a trust token (`incus config trust
// add laforge` on the host) into a stored client credential, then reading the
// host's real storage pools, networks, and images. Registered under kind
// "incus" so internal/api's connect endpoint dispatches here by kind rather than
// hard-coding this family's flow. See internal/builder.Onboarder for the
// contract; the MicroCloud builder implements the same contract independently
// (both speak the LXD/Incus REST enrollment protocol, but they are separate
// packages, not one wearing two names).
func init() { builder.RegisterOnboarder("incus", onboarder{}) }

type onboarder struct{}

func (onboarder) Onboard(ctx context.Context, req builder.OnboardRequest) (*builder.Onboarding, error) {
	if _, err := ParseTrustToken(req.Token); err != nil {
		return nil, fmt.Errorf("%w: %v", builder.ErrInvalidOnboardRequest, err)
	}
	enr, err := Enroll(ctx, req.Token, "", req.Address)
	if err != nil {
		return nil, err
	}
	client, err := NewClient(enr.APIURL, enr.ClientCertPEM, enr.ClientKeyPEM, enr.ServerCertPEM, "")
	if err != nil {
		return nil, err
	}
	disc, err := discoverResources(ctx, client)
	if err != nil {
		return nil, err
	}
	return &builder.Onboarding{
		Connection: builder.Connection{
			APIURL: enr.APIURL, ServerName: enr.ServerName, ServerFingerprint: enr.ServerFingerprint,
			ServerCertPEM: enr.ServerCertPEM, ClientCertPEM: enr.ClientCertPEM, ClientKeyPEM: enr.ClientKeyPEM,
		},
		Discovery: disc,
	}, nil
}

func (onboarder) Rediscover(ctx context.Context, conn builder.Connection) (*builder.Discovery, error) {
	client, err := NewClient(conn.APIURL, conn.ClientCertPEM, conn.ClientKeyPEM, conn.ServerCertPEM, "")
	if err != nil {
		return nil, err
	}
	disc, err := discoverResources(ctx, client)
	if err != nil {
		return nil, err
	}
	return &disc, nil
}

// discoverResources reads a host's pickable resources, never returning nil
// slices so the API always ships JSON arrays.
func discoverResources(ctx context.Context, c *Client) (builder.Discovery, error) {
	pools, err := c.ListStoragePools(ctx)
	if err != nil {
		return builder.Discovery{}, fmt.Errorf("listing storage pools: %w", err)
	}
	networks, err := c.ListNetworks(ctx)
	if err != nil {
		return builder.Discovery{}, fmt.Errorf("listing networks: %w", err)
	}
	images, err := c.ListImages(ctx)
	if err != nil {
		return builder.Discovery{}, fmt.Errorf("listing images: %w", err)
	}
	if pools == nil {
		pools = []builder.StoragePoolInfo{}
	}
	if networks == nil {
		networks = []builder.NetworkInfo{}
	}
	if images == nil {
		images = []builder.ImageInfo{}
	}
	return builder.Discovery{StoragePools: pools, Networks: networks, Images: images}, nil
}
