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
	// Every listing is best effort: on a shared server any of them can be slow
	// or refused, and each only feeds a picker that also takes a typed name.
	// What couldn't be read becomes a warning, never a failed connection.
	var warnings []string
	warn := func(what string, err error) {
		warnings = append(warnings, fmt.Sprintf("couldn't list the %s (%v); type the name instead", what, err))
	}
	pools, err := c.ListStoragePools(ctx)
	if err != nil {
		warn("storage pools", err)
	}
	networks, networkNames, err := c.ListNetworks(ctx)
	if err != nil {
		warn("networks", err)
	}
	images, err := c.ListImages(ctx)
	if err != nil {
		warn("images", err)
	}
	snapshots, err := c.ListSnapshots(ctx)
	if err != nil {
		warn("instance snapshots", err)
	}
	return builder.Discovery{
		StoragePools: orEmpty(pools), Networks: orEmpty(networks), NetworkNames: orEmpty(networkNames),
		Images: orEmpty(images), Projects: []builder.ProjectInfo{}, Snapshots: orEmpty(snapshots), Warnings: orEmpty(warnings),
	}, nil
}

// orEmpty keeps the API's lists JSON arrays, never null.
func orEmpty[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}
