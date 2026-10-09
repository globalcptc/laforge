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
	pools, err := c.ListStoragePools(ctx)
	if err != nil {
		return builder.Discovery{}, fmt.Errorf("listing storage pools: %w", err)
	}
	// Networks only feed the uplink picker, which also takes a typed name, so
	// failing to read them is a warning, not a failed connection.
	var warnings []string
	networks, networkNames, err := c.ListNetworks(ctx)
	if err != nil {
		warnings = append(warnings, fmt.Sprintf("couldn't list the server's networks (%v); type the uplink network's name instead", err))
	}
	scoped := *c
	scoped.Project = project
	images, err := scoped.ListImages(ctx)
	if err != nil {
		return builder.Discovery{}, fmt.Errorf("listing images in project %q: %w", projectOrDefault(project), err)
	}
	projects, err := c.ListProjects(ctx)
	if err != nil || projects == nil {
		projects = []builder.ProjectInfo{}
	}
	// Snapshots are optional (most builders use images), so not being able to
	// list them mustn't stop a builder connecting.
	snapshots, err := scoped.ListSnapshots(ctx)
	if err != nil || snapshots == nil {
		snapshots = []builder.SnapshotInfo{}
	}
	if pools == nil {
		pools = []builder.StoragePoolInfo{}
	}
	if networks == nil {
		networks = []builder.NetworkInfo{}
	}
	if networkNames == nil {
		networkNames = []string{}
	}
	if warnings == nil {
		warnings = []string{}
	}
	if images == nil {
		images = []builder.ImageInfo{}
	}
	return builder.Discovery{StoragePools: pools, Networks: networks, Images: images, Projects: projects, Snapshots: snapshots, NetworkNames: networkNames, Warnings: warnings}, nil
}

func projectOrDefault(p string) string {
	if p == "" {
		return "default"
	}
	return p
}
