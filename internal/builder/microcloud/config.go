package microcloud

// Config is a builder config's Incus-specific half: "each builder config
// maps abstract names to concrete images/sizes/container templates."
// Never exposed to content authors, never taking vars -- server-side only,
// same as every other builder config in the plan.
type Config struct {
	// Images maps a content Host's `os` or Container's `image` field to
	// where Incus should actually pull it from. A missing entry is a
	// validation failure before a build starts ("an environment using
	// [an unlisted os] fails validation"), not a runtime surprise.
	Images map[string]ImageRef
	// Sizes maps a content `size` field to concrete resource limits.
	Sizes map[string]SizeSpec
	// OVNUplinkNetwork is the name of the physical/unmanaged network on
	// the Incus cluster that OVN routes external traffic through --
	// Canonical's own MicroCloud/OVN setup docs call this "UPLINK" by
	// convention, but it's whatever the cluster operator named it when
	// they ran `incus network create <name> --type=physical` (or bridge)
	// during cluster bring-up, so it's a per-cluster builder config value,
	// not a constant. Required: every DeployNetwork call creates an OVN
	// logical network with this as its `network` (uplink) key. See
	// Builder's own doc comment for why OVN, not a plain bridge, is what
	// DeployNetwork uses.
	OVNUplinkNetwork string
	// StoragePool is the Incus storage pool a host's root disk device is
	// created on -- defaults to "default" (this package's own single-node
	// test daemon's real pool name, and Incus's own out-of-the-box
	// default) when left unset, so every existing builder config and test
	// keeps working unchanged. A real MicroCloud cluster's own storage
	// pool is very often NOT named "default" -- found live against a real
	// cluster: a
	// MicroCeph-backed MicroCloud names its pool "remote" by convention,
	// and deployInstance's root device previously hardcoded "default"
	// unconditionally, failing every host with a real disk size against
	// that cluster with "Failed loading storage pool: Storage pool not
	// found." Per-cluster, like OVNUplinkNetwork above, not a constant.
	StoragePool string
	// DockerBaseFingerprint is the fingerprint of this builder's published
	// docker-ready base image (built by the per-builder image-build job,
	// migration 00024). A LaForge `container:` is a Docker container, run
	// inside a thin, nesting-enabled LXD system container booted from THIS
	// image (which has Docker preinstalled) -- see DeployContainer. Empty
	// until the base image has been built, in which case a container deploy
	// fails with a clear "build the base image first" message.
	DockerBaseFingerprint string
}

// storagePoolOrDefault is deployInstance's own resolution of
// Config.StoragePool -- see that field's own doc comment for why "default"
// is the fallback, not an empty string (an empty pool name would fail
// instance creation outright, and every builder config written before
// this field existed implicitly meant "default").
func (c Config) storagePoolOrDefault() string {
	if c.StoragePool != "" {
		return c.StoragePool
	}
	return "default"
}

// HostConfig is one pool member's full connection info -- everything
// NewClient needs, plus the same per-cluster Config fields a single-host
// builder_config already carries (OVNUplinkNetwork, StoragePool,
// OperationTimeoutSeconds), since two genuinely independent hosts can
// each need their own of these just as easily as their own credentials.
// This is internal/builder/incuspool's own wire format for a
// builder_config row's incus_hosts JSONB array (migration 00014) -- one
// element per independent, non-clustered Incus host in the pool, as
// opposed to kind="microcloud"'s single set of incus_* columns for one
// real (clustered) endpoint.
type HostConfig struct {
	// CredentialID names a builder_credential row (migration 00016) made by
	// enrolling with an Incus trust token -- when set, it supplies the
	// connection and APIURL/ClientCertPath/ClientKeyPath/ServerCertPEM are
	// left empty. Those older fields remain for hosts configured before
	// enrollment existed.
	CredentialID            string `json:"credential_id,omitempty"`
	APIURL                  string `json:"api_url"`
	ClientCertPath          string `json:"client_cert_path"`
	ClientKeyPath           string `json:"client_key_path"`
	ServerCertPEM           string `json:"server_cert_pem"`
	OVNUplinkNetwork        string `json:"ovn_uplink_network"`
	StoragePool             string `json:"storage_pool"`
	OperationTimeoutSeconds int    `json:"operation_timeout_seconds,omitempty"`
}

// ImageRef and SizeSpec carry JSON tags so a builder_config row's
// incus_images/incus_sizes JSONB columns (internal/builderconfig.Resolve)
// round-trip directly into Config.Images/Config.Sizes with no separate,
// duplicate wire-format struct needed.
type ImageRef struct {
	// Fingerprint, when set, names one exact image already on the Incus
	// host(s) -- the same image content has the same fingerprint on every
	// host it was imported to, so a pool deploys the identical image
	// everywhere, and an image with no alias is still usable. Alias/Server/
	// Protocol are the fallback for configs that name an image by alias
	// (optionally on a remote simplestreams server).
	Fingerprint string `json:"fingerprint,omitempty"`
	Alias       string `json:"alias"`    // e.g. "ubuntu/22.04", "windows-server-2022"
	Server      string `json:"server"`   // e.g. "https://images.linuxcontainers.org"
	Protocol    string `json:"protocol"` // "simplestreams"
	// VM marks an abstract os name as needing a real virtual machine
	// instance (kernel isolation from the host) rather than an LXC
	// container -- required for Windows, since a Windows guest can't
	// share a Linux host's kernel the way an LXC container does. Not
	// everything on the VM path could be verified in this session's test
	// environment (no /dev/kvm, no licensed Windows image available).
	VM bool `json:"vm"`
}

type SizeSpec struct {
	CPU    string `json:"cpu"`    // Incus limits.cpu, e.g. "2"
	Memory string `json:"memory"` // Incus limits.memory, e.g. "4GiB"
	// Type is the cloud instance type / flavor this size maps to on the
	// non-Incus builders (an EC2 instance type like "t3.medium", an OpenStack
	// flavor id/name). Ignored by the Incus builders, which use CPU/Memory.
	// The image/size maps are shared columns across every builder kind, so
	// this rides alongside CPU/Memory rather than in a separate column.
	Type string `json:"type,omitempty"`
}
