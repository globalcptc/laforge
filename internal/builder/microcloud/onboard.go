package microcloud

import (
	"context"
	"fmt"

	"github.com/globalcptc/laforge/internal/builder"
)

// The MicroCloud builder's onboarding: enrolling an LXD trust token into a
// stored client credential, then reading the cluster's real storage pools,
// networks, and images. Registered under kind "microcloud". This implements the
// same builder.Onboarder contract the Incus builder does, but independently --
// MicroCloud (LXD) and Incus are separate packages sharing only the interface,
// not one implementation.
func init() { builder.RegisterOnboarder("microcloud", onboarder{}) }

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
	disc, err := discoverResources(ctx, client, "")
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
	disc, err := discoverResources(ctx, client, conn.Project)
	if err != nil {
		return nil, err
	}
	return &disc, nil
}

// discoverResources reads a cluster's pickable resources, never returning nil
// slices so the API always ships JSON arrays. c is unscoped: storage pools are
// server-wide and uplink networks always live in `default`. Images are read in
// project, which has its own when its features.images is on. The project list
// is best effort -- an identity allowed only some projects may not be able to
// list them, and that mustn't stop it being connected.
func discoverResources(ctx context.Context, c *Client, project string) (builder.Discovery, error) {
	// Every listing is best effort: on a shared cluster any of them can be slow
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
	scoped := *c
	scoped.Project = project
	images, err := scoped.ListImages(ctx)
	if err != nil {
		warn(fmt.Sprintf("images in project %q", projectOrDefault(project)), err)
	}
	projects, err := c.ListProjects(ctx, project, "default")
	if err != nil {
		// A restricted identity may not be allowed to list projects at all;
		// the project is then typed, and that's expected rather than a warning.
		projects = nil
	}
	snapshots, err := scoped.ListSnapshots(ctx)
	if err != nil {
		warn(fmt.Sprintf("instance snapshots in project %q", projectOrDefault(project)), err)
	}
	return builder.Discovery{
		StoragePools: orEmpty(pools), Networks: orEmpty(networks), NetworkNames: orEmpty(networkNames),
		Images: orEmpty(images), Projects: orEmpty(projects), Snapshots: orEmpty(snapshots), Warnings: orEmpty(warnings),
	}, nil
}

// orEmpty keeps the API's lists JSON arrays, never null.
func orEmpty[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}

func projectOrDefault(p string) string {
	if p == "" {
		return "default"
	}
	return p
}
