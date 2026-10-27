// Package builder is the contract every hoster-specific implementation
// satisfies -- the Builder contract, made
// concrete in Go. A builder is deliberately dumb: it takes a fully
// resolved spec (a deterministic external name the caller already
// decided, plus whatever abstract fields the spec needs) and does exactly
// one thing per call, safe to repeat. Nothing here talks to Postgres,
// makes scheduling decisions, or knows what a "team" or a "build" is --
// that's internal/orchestrator and internal/runner. This package is only
// ever the thing on the other end of one call.
//
// "Builders call hoster APIs directly. No Terraform or OpenTofu layer" --
// this interface is deliberately that thin.
package builder

import "context"

// There is deliberately no "capabilities" opt-out: every builder MUST implement
// the whole contract for real. Content is builder-agnostic -- the same
// environment file deploys on any builder without change -- so a builder that
// cannot do part of the spec (a cloud that has no container primitive, say)
// uses whatever workaround its platform offers (Fargate, Zun, native OCI,
// nested Docker), and if it genuinely cannot, it is simply not a valid builder.
// An incomplete builder fails at deploy time with a clear error, not by a
// capability flag that would let the environment silently vary per builder.

// NetworkSpec, HostSpec, and ContainerSpec are what a Deploy* call needs.
// ExternalName is assigned by the caller (internal/orchestrator), not the
// builder -- "idempotent builders: every builder op is `ensure`
// (create-or-adopt), using deterministic resource names/tags, so a retry
// after partial success converges instead of duplicating" only works if
// the name is decided once, up front, by whoever owns desired state, and
// handed to every retry unchanged.
//
// Team was added while building the real
// Incus builder: OpenAccess/CloseAccess are required to act on "every
// resource belonging to this team," but nothing before this carried team
// identity through the Builder interface at all -- every resource that
// exists is here because a Deploy* call put it there, so team identity
// has to be attached at that point or a builder has no way to answer
// "which of these are this team's" later. A plain string (the team
// number, formatted by the caller -- see internal/runner), not a richer
// type: builders that don't need it (the fake builder) simply ignore it,
// same as ExternalName's own "the caller decided, the builder just uses
// it" shape.
// Power actions for Builder.PowerAction -- the three infrastructure-level
// operations a host page exposes, mapped to whatever each builder's hoster
// calls them (Incus: start / stop / restart).
const (
	PowerStart  = "start"
	PowerStop   = "stop"
	PowerReboot = "reboot"
)

type NetworkSpec struct {
	ExternalName string
	DisplayName  string // "t<team><network>", for a readable hoster name -- see incus.networkName
	Team         string
	CIDR         string
	// VPCCIDR is the environment's overall address space (Environment.CIDR):
	// the supernet every network fits inside. A cloud builder that hosts a whole
	// team in one VPC (AWS) creates that VPC with exactly this CIDR and carves
	// each network's subnet from it. Empty when the environment declares none --
	// the on-prem builders (Incus/MicroCloud) ignore it regardless, and AWS then
	// falls back to inferring a containing /16 from CIDR.
	VPCCIDR string
}

// Network, like Team, was found missing while proving a real two-team
// build end to end: nothing wired a deployed host or container to the
// per-team network DeployNetwork actually created, so every instance
// fell back to whatever the hoster's own default network is -- verified
// live against Incus, where that meant every team's hosts landed on the
// same shared incusbr0, no isolation between teams at all despite
// DeployNetwork creating a genuinely separate bridge per team. Network is
// the same deterministic ExternalName DeployNetwork was called with for
// this host/container's network, computed the same way by the caller
// (internal/runner) regardless of whether that network's own deploy task
// has actually run yet -- deterministic naming means a builder can
// resolve it without waiting, and if the underlying network genuinely
// isn't there yet, a real "attach to a nonexistent network" error is
// exactly the right outcome: it fails this deploy, which the existing
// task retry/backoff resolves once the network task completes, with no
// separate dependency-ordering system needed.
// Address is the copy's own address -- content's CIDR + last_octet,
// resolved by the caller the same way internal/render resolves it for
// script templating (render.Address), so a script and the host it runs on
// never disagree about the host's own IP. Optional: empty means the
// builder picks however it normally would (DHCP, if it has any) -- the
// fake builder and any future builder without per-instance addressing
// simply ignore it, same shape as Team and Network before it.
type HostSpec struct {
	ExternalName string
	// DisplayName is a human-readable label for the instance at the hoster
	// -- "<branch>-t<team>-<host>" -- so an operator looking at the hoster
	// directly (not through LaForge) can tell what a VM is, instead of the
	// opaque hash ExternalName reduces to. Advisory only: a builder that
	// can produce a readable-but-still-deterministic name from it should
	// (see internal/builder/incus.instanceName), and one that can't just
	// ignores it and keeps using ExternalName, same as Address/Network.
	DisplayName string
	Team        string
	Network     string
	// NetworkDisplayName is the same readable hint for the network this
	// instance attaches to, so the NIC references the network by the exact
	// readable name DeployNetwork gave it (see incus.networkName) rather
	// than a hash. Advisory, like DisplayName.
	NetworkDisplayName string
	Address            string
	OS                 string
	Size               string
	DiskGB             int
	TCPPorts           []string
	UDPPorts           []string
	// AgentDownloadURL is the one-time-token URL the fresh instance fetches its
	// own agent binary from on first boot (agentdelivery.Delivery.DownloadURL).
	// Agents aren't baked into images -- this is how a host installs and starts
	// its agent. Empty means no agent delivery (a build still deploys). The
	// BUILDER renders this into its image's user-data format using the renderers
	// in internal/agentdelivery (LinuxUserData / WindowsCloudbaseUserData /
	// WindowsEC2LaunchUserData), choosing by agentdelivery.PlatformFor(OS) -- the
	// format is a property of the image/hoster, so the builder owns it.
	AgentDownloadURL string
	// AgentBinary is the per-host patched agent binary (agentdelivery.Delivery.
	// Binary). When set, a builder that can attach a NoCloud config drive writes
	// it onto that drive so the first-boot user-data copies it locally instead of
	// downloading (embed-first; AgentDownloadURL stays as the fallback). Empty
	// means download-only. A builder with no config drive (AWS) ignores it and
	// relies on AgentDownloadURL.
	AgentBinary []byte
	// TempAdminPassword is a throwaway admin password (agentdelivery.
	// TempAdminPassword) the first-boot user-data sets so a host is reachable
	// before its agent runs the real set_password step. Empty means don't set
	// one. It is deliberately not the environment's real password -- see that
	// function's doc.
	TempAdminPassword string
	// Priority is this object's place on the build's critical path: how many
	// objects (transitively) depends_on it, so a domain controller the rest of a
	// team waits on outranks a leaf workstation. 0 = nothing waits on it. Set by
	// the orchestrator, carried on the deploy task. A builder that admits
	// creates through a bounded queue (Incus's per-host cap) serves higher
	// priorities first; the rest ignore it.
	Priority int
}

// DockerBaseAlias is the image alias a builder's "docker base image" job
// publishes on the hoster: an Ubuntu system container image with Docker Engine
// and the compose plugin installed from Docker's own repository. MicroCloud boots every `container:` from it (LXD
// has no OCI runtime); both MicroCloud and Incus boot a compose container's
// machine from it.
const DockerBaseAlias = "laforge-docker-base"

// OldKernelStorageDriver is Docker's built-in, dependency-free storage driver
// for hosts whose kernel can't do overlay2. overlay2 (Docker's default) needs a
// kernel new enough to mount overlayfs as an unprivileged/nested user, which old
// MicroCloud nodes don't have, so Docker there fails to start or to run a
// container. "vfs" is the naive copy-per-layer driver built into Docker with no
// kernel feature or extra package required -- it works on any kernel, at the
// cost of disk space and layer-copy speed. We only force it where overlay2 is
// unavailable (MicroCloud); builders on modern kernels (Incus) keep overlay2.
const OldKernelStorageDriver = "vfs"

// DockerBaseInstallScript is what the docker-base job runs inside its build
// container: Docker Engine and the compose plugin from Docker's own apt
// repository (download.docker.com) rather than the distribution's packages,
// which trail it by a long way. It then checks that the daemon came up under
// nesting and can actually run a container, and prints DOCKER_OK on success.
//
// storageDriver, when non-empty, is pinned in /etc/docker/daemon.json BEFORE the
// daemon first starts (so docker-ce's own postinst start already uses it), for
// hosts that can't do overlay2 -- see OldKernelStorageDriver. Empty leaves
// Docker's default (overlay2).
func DockerBaseInstallScript(storageDriver string) string {
	daemonJSON := ""
	if storageDriver != "" {
		daemonJSON = `install -m 0755 -d /etc/docker
printf '%s' '{"storage-driver":"` + storageDriver + `"}' > /etc/docker/daemon.json
`
	}
	return `export DEBIAN_FRONTEND=noninteractive
apt-get update && apt-get install -y ca-certificates curl || exit 1
install -m 0755 -d /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg -o /etc/apt/keyrings/docker.asc || exit 1
chmod a+r /etc/apt/keyrings/docker.asc
. /etc/os-release
echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/ubuntu ${UBUNTU_CODENAME:-$VERSION_CODENAME} stable" > /etc/apt/sources.list.d/docker.list
` + daemonJSON + `apt-get update && apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin || exit 1
systemctl enable --now docker
sleep 3
docker --version
docker info >/dev/null 2>&1 && docker compose version && docker run --rm hello-world >/dev/null 2>&1 && docker image rm hello-world >/dev/null 2>&1 && echo DOCKER_OK || echo DOCKER_BAD`
}

// DockerBaseCleanupScript trims the build container before it is published:
// apt caches, and cloud-init's record of having run, so an instance created
// from the image runs its own user-data (the agent install) on first boot.
const DockerBaseCleanupScript = `apt-get clean; rm -rf /var/lib/apt/lists/*; cloud-init clean --logs >/dev/null 2>&1; true`

// ComposeHostSpec is the machine a compose container runs on, as the host a
// builder deploys for it (see ContainerSpec.ComposeHost). It runs at the
// container's own configured size, resolved through the builder's size map like
// any host.
func (s ContainerSpec) ComposeHostSpec() HostSpec {
	return HostSpec{
		ExternalName: s.ExternalName, DisplayName: s.DisplayName, Team: s.Team,
		Network: s.Network, NetworkDisplayName: s.NetworkDisplayName, Address: s.Address,
		OS: ComposeHostImage, Size: s.Size, DiskGB: s.DiskGB,
		TCPPorts: s.TCPPorts, UDPPorts: s.UDPPorts,
		AgentDownloadURL: s.AgentDownloadURL,
		Priority:         s.Priority,
	}
}

// ComposeHostImage is the name, in a builder's image map, of the image a
// compose container's machine boots from: any Linux image, ideally one with
// Docker and the compose plugin already installed (it is installed on first
// boot otherwise). Configured per builder, like a host's `os` images; content
// never names it.
const ComposeHostImage = "compose-host"

// DefaultComposeDiskGB is a compose container's root disk when it sets none.
const DefaultComposeDiskGB = 40

type ContainerSpec struct {
	ExternalName       string
	DisplayName        string // see HostSpec.DisplayName
	Team               string
	Network            string
	NetworkDisplayName string // see HostSpec.NetworkDisplayName
	Address            string
	// ComposeHost marks a container that runs a Docker Compose project rather
	// than one image. The builder gives it a Docker-capable Linux machine --
	// booted from its ComposeHostImage, with DiskGB of disk and the agent
	// delivered via AgentDownloadURL like any host -- and nothing else: the
	// project is unpacked and started by the agent, as the container's first
	// commands. Image, Env, Command and the registry fields are empty.
	ComposeHost bool
	DiskGB      int
	// Image is the OCI image ref the container runs (e.g. "nginx:alpine").
	// It is NOT an entry in a builder's LXD image map -- a container always
	// boots the builder's docker base image and `docker run`s this ref.
	Image string
	Size  string
	// Env and Command are the container's environment and command override.
	Env      map[string]string
	Command  []string
	TCPPorts []string
	UDPPorts []string
	// AgentBinary is the object's patched LaForge agent (a static musl-linux
	// binary). It is planted so the agent is the application container's own
	// entrypoint, supervising the image's real command -- making a container
	// check in and run its steps/validators from INSIDE the app, exactly like a
	// host, on every builder. A native-OCI builder (Incus) pushes it in and sets
	// oci.entrypoint; a nesting builder (MicroCloud) pushes it into the Docker
	// host and bind-mounts it as the nested container's `--entrypoint`. Either
	// way the agent runs inside the app container and never sees the container
	// runtime, so content (`docker ps`, `process_running: dockerd`) behaves the
	// same regardless of builder. Empty when agent delivery is off (the image
	// then runs its own entrypoint, unmanaged).
	AgentBinary []byte
	// AgentDownloadURL is the same agent as AgentBinary but fetched at start via
	// its one-time-token URL, for a container platform that can't take a pushed
	// binary (AWS Fargate, OpenStack Zun): an init step downloads it, then the
	// app container runs it in supervisor mode. Empty when agent delivery is off.
	AgentDownloadURL string
	// RegistryHost/RegistryUser/RegistrySecret are the credential for the image's
	// registry, resolved by the caller (internal/runner) from stored registry
	// credentials. A nesting builder (MicroCloud) `docker login`s with them
	// before pulling a private image; empty means an unauthenticated/public pull.
	// A native-OCI builder that pulls through its own daemon's configured
	// registry auth ignores them, same "each builder takes what it needs" shape
	// as the agent fields above.
	RegistryHost   string
	RegistryUser   string
	RegistrySecret string
	// LogDriver/LogOptions are the environment's container_logs config (a log
	// driver name and its driver-specific options). When LogDriver is non-empty
	// the builder forwards this container's console output using its native
	// mechanism -- a docker `--log-driver`/`--log-opt` on the run, an ECS task
	// logConfiguration, etc. Empty means no forwarding. A ComposeHost ignores
	// these: a compose project's logging is set as the Docker daemon default on
	// its host before `docker compose up` (see internal/gateway/compose.go), so
	// every service inherits it.
	LogDriver  string
	LogOptions map[string]string
	// Priority is the same critical-path rank as HostSpec.Priority.
	Priority int
}

// Resource is one thing Inspect finds already existing at the hoster,
// identified the same way a Deploy* call would name it -- what adoption
// and drift detection key off.
type Resource struct {
	ExternalRef string `json:"external_ref"`
	Kind        string `json:"kind"`  // network | host | container
	State       string `json:"state"` // live power state of an instance -- see PowerState* below; "" for a network or a builder that can't report it
}

// Normalized instance power states an Inspect result carries in
// Resource.State, so "is the instance actually up" is builder truth
// tracked independently of the LaForge agent's own health. A builder maps
// its hoster's own vocabulary onto these (Incus "Running"/"Stopped"/...);
// "" means not applicable (a network) or not reported.
const (
	PowerStateRunning = "running"
	PowerStateStopped = "stopped"
	PowerStateOther   = "other" // exists but neither cleanly running nor stopped (frozen, error, starting...)
)

// InstanceStater is an optional builder ability: read ONE instance's power
// state straight from the hoster. The power-state poll uses it to confirm an
// instance its full listing left out before calling it missing -- a clustered
// hoster's listing can come back without instances a busy member didn't answer
// for in time, and one incomplete listing must not mark running hosts missing.
// exists is false only when the hoster says the instance isn't there; any
// other failure is an error, and the caller leaves the state as it was.
type InstanceStater interface {
	InstanceState(ctx context.Context, externalRef string) (state string, exists bool, err error)
}

// Metric is one self-describing stat a builder publishes about its own backend
// (the hoster/stack, NOT the deployed guests). The UI renders generically from
// these, so a builder can add a metric and have it appear with no UI change:
// Kind "timeseries" draws a chart from stored history, "gauge" draws a
// usage-vs-limit bar. See docs/plans/builder-monitoring.md.
type Metric struct {
	Key   string   `json:"key"`   // stable id: "cpu", "mem", "disk_ops", "net", "api_latency", "quota:vpcs"
	Label string   `json:"label"` // human label for display
	Node  string   `json:"node"`  // "" = builder-wide; a server/member name for per-node (Incus)
	Kind  string   `json:"kind"`  // MetricTimeseries | MetricGauge
	Unit  string   `json:"unit"`  // "%", "ops/s", "ms", "count", "bytes/s", ...
	Value float64  `json:"value"`
	Max   *float64 `json:"max,omitempty"` // set only for a gauge (the quota limit); nil = unbounded series
}

const (
	MetricTimeseries = "timeseries"
	MetricGauge      = "gauge"
	// MetricCounter is a monotonically-increasing total (e.g. cumulative network
	// bytes). Value carries the raw counter; the API turns consecutive samples
	// into a per-second rate for the chart, so the builder never has to compute a
	// rate itself (and a single live sample simply yields no rate yet). Unit is
	// the DISPLAY unit of that rate, e.g. "bytes/s".
	MetricCounter = "counter"
)

// Stats is a builder's current reading of its own backend health -- a flat list
// of self-describing Metrics plus any best-effort warnings (a node that couldn't
// be read, a quota near its limit). Sampled on an interval by the collector and
// stored for the time-series history; gauges are shown at their latest value.
type Stats struct {
	Metrics  []Metric `json:"metrics"`
	Warnings []string `json:"warnings,omitempty"`
}

// MetricProvider is the OPTIONAL stats contract: a builder that can report its
// stack's health implements it, and is probed for it (b.(MetricProvider)) the
// same way the other optional builder capabilities are. A builder that doesn't
// (OpenStack, the fake builder) simply has no stats UI.
type MetricProvider interface {
	Stats(ctx context.Context) (Stats, error)
}

// ConsoleLogReader is the OPTIONAL instance-console contract: a builder that can
// read a deployed instance's console output (boot messages, cloud-init or
// cloudbase-init progress) implements it, and is probed for it
// (b.(ConsoleLogReader)) like MetricProvider. The orchestrator captures it when
// a host's agent never checks in, since the console is the one view of a guest
// that doesn't depend on the agent. ConsoleLog returns at most the last maxBytes
// of the log.
type ConsoleLogReader interface {
	ConsoleLog(ctx context.Context, team, externalRef string, maxBytes int) (string, error)
}

// TailBytes keeps the last max bytes of a console log (max <= 0 keeps it all).
func TailBytes(b []byte, max int) string {
	if max > 0 && len(b) > max {
		b = b[len(b)-max:]
	}
	return string(b)
}

// Demand is what an environment will need from its builder, estimated across all
// teams BEFORE any hoster call, so a quota wall is caught up front instead of
// mid-deploy (the sibling of the AWS IAM pre-flight check). Builder-agnostic: it
// carries LaForge concepts (teams, public hosts, size names), and the builder
// translates them into its own units -- on AWS, Teams -> VPCs, PublicHosts ->
// Elastic IPs, and each size -> an instance type whose vCPUs count toward the
// region vCPU quota. See docs/plans/builder-monitoring.md.
type Demand struct {
	Teams          int            // one isolated network per team (one VPC on AWS)
	PublicHosts    int            // total across teams -- each needs an external address (an EIP on AWS)
	InstanceCounts map[string]int // size name -> number of host instances across all teams
}

// Readiness is a ReadinessChecker's verdict: human-readable warnings about limits
// the build would exceed. Empty Warnings means clear to deploy. It's advisory --
// the check never blocks a deploy, it surfaces what a quota would.
type Readiness struct {
	Warnings []string `json:"warnings,omitempty"`
}

// ReadinessChecker is the OPTIONAL pre-deploy capacity contract: a builder whose
// backend enforces account limits implements it and is probed for it
// (b.(ReadinessChecker)), the same way MetricProvider is. Only AWS implements it
// today -- it's the one hoster where a quota silently breaks a deploy. A builder
// that doesn't simply has no pre-deploy readiness warnings.
type ReadinessChecker interface {
	CheckReadiness(ctx context.Context, d Demand) (Readiness, error)
}

// Builder is the contract. Every Deploy*/Destroy* call must be an
// "ensure": safe to call more than once for the same ExternalName /
// externalRef without creating a second resource or erroring on a
// not-found second destroy.
type Builder interface {
	DeployNetwork(ctx context.Context, spec NetworkSpec) (externalRef string, err error)
	DeployHost(ctx context.Context, spec HostSpec) (externalRef string, err error)
	DeployContainer(ctx context.Context, spec ContainerSpec) (externalRef string, err error)

	// Destroy* takes team for the same reason Deploy* does -- added
	// (2026-09-25) while building internal/builder/incuspool, the real
	// "multiple independent Incus hosts" builder: with no single shared
	// endpoint, a pool has no way to know which underlying host holds
	// externalRef without it. A builder that doesn't need it (fake, and
	// incus.Builder itself, which only ever has one endpoint) simply
	// ignores it, same as every other field this interface has grown this
	// way (see NetworkSpec's own doc comment on Team/Network).
	DestroyNetwork(ctx context.Context, team, externalRef string) error
	DestroyHost(ctx context.Context, team, externalRef string) error
	DestroyContainer(ctx context.Context, team, externalRef string) error

	// Inspect lists what actually exists at the hoster right now -- "list
	// what exists, for adoption and drift."
	Inspect(ctx context.Context) ([]Resource, error)

	// OpenAccess and CloseAccess are required, behavioural operations:
	// "closing a team must block all external ingress and terminate
	// established connections, not merely stop new ones." Team identity
	// is a plain string here (an external team name/tag a builder
	// assigns meaning to); nothing above this package needs to know how.
	OpenAccess(ctx context.Context, team string) error
	CloseAccess(ctx context.Context, team string) error

	// PowerAction performs an infrastructure-level power operation on one
	// already-deployed instance at the hoster, addressed by its externalRef
	// -- independent of the in-guest agent, so it works even when the agent
	// is dead or the OS is wedged. action is one of PowerStart, PowerStop,
	// PowerReboot. force selects a hard power operation (an immediate
	// power-off / reset) over a graceful, OS-level one (an ACPI
	// shutdown/reboot the guest can react to and flush to disk); force is
	// meaningless for PowerStart and ignored there.
	PowerAction(ctx context.Context, team, externalRef, action string, force bool) error

	// ConfigureExternalAccess realizes content's `public:` ports -- making the
	// given hosts reachable from OUTSIDE the environment on those ports (RDP and
	// the like) -- and returns the real endpoints assigned, so an operator or
	// competitor knows the address:port to connect to. How is builder-specific
	// and the whole reason this is per-builder: a public IP per host (AWS,
	// OpenStack), or a port-NAT from a shared uplink IP with allocated external
	// ports (Incus, MicroCloud, which share one external IP across every team).
	// Idempotent: re-asserting the same hosts returns the same endpoints without
	// duplicating forwards. The caller (internal/runner) records what comes back;
	// the builder, not the caller, allocates, by inspecting its own hoster's
	// live forwards -- so there is no cross-build port collision to coordinate in
	// our own state. A builder that genuinely cannot do external ingress returns
	// an error rather than a capability opt-out, same contract as the rest here.
	ConfigureExternalAccess(ctx context.Context, team string, hosts []ExternalHost) ([]ExternalEndpoint, error)

	// ConfigureNetworkAccess enforces a team's inter-network reachability
	// (content's `visible_from:`). It is a
	// TEAM-level operation, not per-network, because the policy is inherently
	// cross-network (a network's rule references its siblings). Given the full
	// set of a team's networks, it must ensure, idempotently:
	//   - a network is reachable from a sibling network IFF it lists that
	//     sibling in VisibleFrom (default-deny),
	//   - no reachability ever crosses to another team (CIDRs are identical
	//     across teams, so isolation must come from a per-team routing domain,
	//     never from the CIDR-based rules alone),
	//   - same-network traffic and egress (the internet, the LaForge gateway)
	//     stay open.
	// How is the builder's business: OVN peering + ACLs (Incus/MicroCloud), a
	// per-team VPC + security groups (AWS), a per-team router + security groups
	// (OpenStack). A builder that cannot enforce this returns an error rather
	// than silently under-enforcing -- there is no capability opt-out.
	ConfigureNetworkAccess(ctx context.Context, team string, networks []NetworkAccess) error
}

// ExternalHost is one deployed host/container to make reachable from outside the
// environment, on its listed public ports. ExternalRef/Address are the deployed
// instance's own (what DeployHost returned, and its lab IP); TCPPorts/UDPPorts
// are the subset of its `ports` that content marked `public:`. Empty port lists
// mean nothing to expose.
type ExternalHost struct {
	ExternalRef string
	Address     string
	TCPPorts    []string
	UDPPorts    []string
}

// ExternalEndpoint is one realized external mapping: the public address:port a
// client connects to for a host's internal port. On a public-IP builder
// ExternalPort equals InternalPort (no remap) and PublicAddress is that host's
// own public IP; on a shared-IP builder ExternalPort is the allocated port and
// PublicAddress is <shared-ip>:<ExternalPort>. The runner records these in
// external_access for the UI/CLI to show.
type ExternalEndpoint struct {
	ExternalRef   string
	Protocol      string // "tcp" | "udp"
	InternalPort  string
	ExternalPort  string
	PublicAddress string // "ip:port"
}

// NetworkAccess is one network's place in its team's reachability policy, passed
// to ConfigureNetworkAccess. VisibleFrom holds the ExternalNames of the sibling
// networks allowed to reach this one (empty = none may). ExternalName/DisplayName
// are the same deterministic names DeployNetwork was called with, so a builder
// can address the already-deployed network; CIDR is the network's subnet, which
// a builder turns into the concrete source/destination of an allow rule.
//
// Hosts carries each host/container on this network with the ports it declared,
// so the builder can enforce `ports:` as an ingress firewall: from an allowed
// sibling network, a host is reachable ONLY on its declared TCP/UDP ports, and a
// host that declared no ports is reachable on none (deny-all). A builder that
// enforces only VisibleFrom (source) without Hosts (port) under-enforces.
type NetworkAccess struct {
	ExternalName string
	DisplayName  string
	CIDR         string
	VisibleFrom  []string     // ExternalNames of the sibling networks allowed to reach this one
	Hosts        []HostAccess // the hosts/containers on this network and their allowed ports
}

// HostAccess is one host/container's ingress firewall on its network: only the
// listed TCP/UDP ports are reachable, and a host with empty lists is reachable
// on nothing (deny-all). Address is the host's IP on this network (the
// destination a builder scopes its per-host port rules to). ExternalRef is the
// deployed instance's ref (its hoster name), so a builder can attach the rule at
// the instance's own port -- what makes same-network (same-switch) traffic
// subject to the firewall too, not just routed cross-network traffic.
type HostAccess struct {
	Address     string
	ExternalRef string
	TCPPorts    []string
	UDPPorts    []string
}
