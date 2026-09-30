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
	Size        string
	DiskGB      int
	TCPPorts    []string
	UDPPorts    []string
	// CloudInitUserData, when set, is delivered to the instance at
	// create time (Incus: the cloud-init.user-data config key). Agents
	// aren't baked into images -- this is how a fresh host installs and
	// starts its own agent on first boot. Empty means no agent delivery
	// (a build still deploys).
	CloudInitUserData string
	CloudInitViaISO   bool
}

type ContainerSpec struct {
	ExternalName       string
	DisplayName        string // see HostSpec.DisplayName
	Team               string
	Network            string
	NetworkDisplayName string // see HostSpec.NetworkDisplayName
	Address            string
	// Image is the OCI image ref the container runs (e.g. "nginx:alpine").
	// It is NOT an entry in a builder's LXD image map -- a container always
	// boots the builder's docker base image and `docker run`s this ref.
	Image string
	Size  string
	// Env and Command are the container's environment and command override.
	Env               map[string]string
	Command           []string
	TCPPorts          []string
	UDPPorts          []string
	CloudInitUserData string
	CloudInitViaISO   bool
	// AgentBinary is the object's patched LaForge agent (a static musl-linux
	// binary), delivered INTO a native OCI container so it runs the agent as the
	// container's entrypoint and supervises the image's real command -- making a
	// container check in and run steps/validators exactly like a host. A native
	// builder (Incus) pushes and runs it; a nesting builder (MicroCloud) ignores
	// it and uses CloudInitUserData instead. Empty when agent delivery is off.
	AgentBinary []byte
	// AgentDownloadURL is the same agent as AgentBinary but fetched at start via
	// its one-time-token URL, for a container platform that can't take a pushed
	// binary (AWS Fargate, OpenStack Zun): an init step downloads it, then the
	// app container runs it in supervisor mode. Empty when agent delivery is off.
	AgentDownloadURL string
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
