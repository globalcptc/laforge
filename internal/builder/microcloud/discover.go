package microcloud

import (
	"context"
	"encoding/json"

	"github.com/globalcptc/laforge/internal/builder"
)

// This file is the live-discovery half of "we have an API, why are we
// asking for images/storage pools/OVN uplinks as strings" -- a real
// Incus/MicroCloud server already knows its own storage pools, networks,
// and cached images; a builder config's own admin UI should read them
// live and let an operator pick, not type a name blind and find out it
// was wrong on the first real deploy. Every type here is a small
// projection of Incus's real API response, the same discipline client.go
// itself already uses (see its own doc comment on apiResponse/APIError
// being verified against a live daemon, not just documentation).

// StoragePoolInfo is one entry of GET /1.0/storage-pools?recursion=1.
type StoragePoolInfo = builder.StoragePoolInfo

// ListStoragePools lists every storage pool this server knows about --
// real candidates for Config.StoragePool (a host's root disk device).
func (c *Client) ListStoragePools(ctx context.Context) ([]StoragePoolInfo, error) {
	raw, err := c.get(ctx, "/1.0/storage-pools?recursion=1")
	if err != nil {
		return nil, err
	}
	var out []StoragePoolInfo
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// NetworkInfo is one entry of GET /1.0/networks?recursion=1.
type NetworkInfo = builder.NetworkInfo

// ListNetworks lists every network this server knows about. An OVN
// uplink is always an existing "physical" (or occasionally unmanaged
// "bridge") network the cluster operator already created during
// bring-up -- see Config.OVNUplinkNetwork's own doc comment -- so a
// caller building an uplink picker filters this list to Type=="physical"
// itself; returning the full list here (not pre-filtered) keeps this
// method a plain mirror of the real API, same as ListStoragePools.
func (c *Client) ListNetworks(ctx context.Context) ([]NetworkInfo, error) {
	raw, err := c.get(ctx, "/1.0/networks?recursion=1")
	if err != nil {
		return nil, err
	}
	var out []NetworkInfo
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ImageInfo is a trimmed projection of GET /1.0/images?recursion=1 --
// just enough to let an operator recognize and pick a cached image
// (alias, OS/release, architecture, container vs. VM), not Incus's full
// metadata blob.
type ImageInfo = builder.ImageInfo

// rawImage mirrors the real API shape (aliases is a list of {name,
// description} objects, not plain strings) -- unmarshaled into this
// first, then flattened into ImageInfo.Aliases for a caller that just
// wants names.
type rawImage struct {
	Fingerprint  string `json:"fingerprint"`
	Architecture string `json:"architecture"`
	Type         string `json:"type"`
	Properties   struct {
		OS          string `json:"os"`
		Release     string `json:"release"`
		Description string `json:"description"`
	} `json:"properties"`
	Aliases []struct {
		Name string `json:"name"`
	} `json:"aliases"`
}

// ListImages lists every image already cached locally on this server --
// real candidates for Config.Images (mapping a content os/image name to
// an Incus alias). Images pulled from a remote simplestreams server
// (images.linuxcontainers.org, etc.) that have never been used here
// won't appear until they're actually cached -- a real, named limitation
// of "what can we discover," not a bug: the alternative (querying a
// remote image server's own catalog) is a separate, much larger fetch
// this method deliberately doesn't attempt.
func (c *Client) ListImages(ctx context.Context) ([]ImageInfo, error) {
	raw, err := c.get(ctx, "/1.0/images?recursion=1")
	if err != nil {
		return nil, err
	}
	var rows []rawImage
	if err := json.Unmarshal(raw, &rows); err != nil {
		return nil, err
	}
	out := make([]ImageInfo, 0, len(rows))
	for _, row := range rows {
		img := ImageInfo{
			Fingerprint:  row.Fingerprint,
			Architecture: row.Architecture,
			Type:         row.Type,
			// Never nil: an unaliased image must still serialize as [],
			// not null -- the UI indexes it.
			Aliases: []string{},
		}
		img.Properties.OS = row.Properties.OS
		img.Properties.Release = row.Properties.Release
		img.Properties.Description = row.Properties.Description
		for _, a := range row.Aliases {
			img.Aliases = append(img.Aliases, a.Name)
		}
		out = append(out, img)
	}
	return out, nil
}

// ProjectInfo is one entry of GET /1.0/projects?recursion=1.
type ProjectInfo = builder.ProjectInfo

// ListProjects lists the projects this client can see -- every project for a
// fully trusted client, only its allowed ones for a restricted one.
func (c *Client) ListProjects(ctx context.Context) ([]ProjectInfo, error) {
	raw, err := c.get(ctx, "/1.0/projects?recursion=1")
	if err != nil {
		return nil, err
	}
	var rows []struct {
		Name        string            `json:"name"`
		Description string            `json:"description"`
		Config      map[string]string `json:"config"`
	}
	if err := json.Unmarshal(raw, &rows); err != nil {
		return nil, err
	}
	out := make([]ProjectInfo, 0, len(rows))
	for _, r := range rows {
		on := func(key string) bool { return r.Config[key] == "true" }
		out = append(out, ProjectInfo{
			Name: r.Name, Description: r.Description,
			Networks: on("features.networks"), Images: on("features.images"),
			Profiles: on("features.profiles"), StorageVolumes: on("features.storage.volumes"),
			Restricted: on("restricted"),
		})
	}
	return out, nil
}
