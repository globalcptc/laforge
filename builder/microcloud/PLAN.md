# LaForge MicroCloud Builder — Development Plan

> Status: implementation in progress. Phases 0–5 are implemented;
> live-cluster validation and Windows image validation remain.
> Target: Canonical MicroCloud as the production LaForge backend.

## 1. What we are actually building

**MicroCloud is not a provisioning API.** It is an installer that drives
three other snaps — LXD, MicroCeph, and MicroOVN — and wires them into a
working cluster. After `microcloud init`, MicroCloud steps out of the
way; per Canonical's docs, _"MicroCloud does not manage the services
that it deploys. After the deployment process, the individual services
are operating independently."_

So this builder targets the **LXD API** on any cluster member. The
MicroCloud API (microcluster-based, `/1.0/services`) only handles
cluster lifecycle — adding and removing members — which is an operator
task LaForge should never touch.

The **OpenStack builder is the template to copy**, not vSphere/NSX-T.
LXD has a direct analogue for nearly every OpenStack object
`builder/openstack/openstack.go` touches, and the port-range parsing,
`vdi`/`vdi_visible` special-casing, and cloud-init agent bootstrap carry
over almost verbatim.

## 2. Concept mapping

| LaForge | OpenStack builder | MicroCloud/LXD builder |
| --- | --- | --- |
| `Team` | Neutron router | **Project** (pinned to one cluster member) + peerings + one ingress forward |
| `ProvisionedNetwork` | network + subnet + router port | **OVN network** (`--type=ovn`, `network=UPLINK`) |
| `ProvisionedHost` | Nova server + security group | **Instance** *(no ACL — see §3.6)* |
| `Host.InstanceSize` | flavor UUID | `limits.cpu` / `limits.memory` |
| `Host.Disk.Size` | boot volume size | `devices.root.size` on the `remote` pool |
| `Host.OS` | image UUID | image alias |
| `Host.LastOctet` | `FixedIP` | NIC device `ipv4.address` |
| `Host.ExposedTCPPorts` | secgroup rules | ingress forward port list (team ingress host only) |
| Team ingress | floating IP | **Network forward** on the team's VDI network |
| AZ balancing | `getOptimalAvailabilityZone` | *(delete it — LXD schedules)* |

The planner's call order (`DeployTeam` → `DeployNetwork` → `DeployHost`,
see `planner/build.go`) maps onto project → OVN network → instance, so
the `Vars` hand-off pattern the OpenStack builder already uses works
unchanged.

### Vars written back to the DB

| Entity | Key | Value |
| --- | --- | --- |
| `Team` | `lxd_project` | project name |
| `Team` | `lxd_cluster_group` | one-member group the project is pinned to |
| `Team` | `lxd_networks` | comma-separated, drives the peerings |
| `Team` | `ingress_address` | allocated forward listen address |
| `ProvisionedNetwork` | `lxd_network` | OVN network name |
| `ProvisionedHost` | `lxd_instance` | instance name |
| `ProvisionedHost` | `PublicIP` | forward listen address, if exposed |

`PublicIP` keeps that exact spelling because other parts of LaForge
already read the key.

## 3. Networking design

This is the part that needed real thought, so it gets its own section.

### 3.1 Why OVN is mandatory

`planner/plan.go` gives every team's `ProvisionedNetwork` the *same*
CIDR as the environment's `Network` definition:

```405:413:planner/plan.go
	entProvisionedNetwork, err := client.ProvisionedNetwork.Create().
		SetName(entNetwork.Name).
		SetCidr(entNetwork.Cidr).
```

Ten teams means ten identical `10.0.x.0/24` subnets. Linux bridges
cannot do that. OVN can, and the upstream docs name this as its reason
to exist: _"useful for labs and multi-tenant environments where the same
logical subnets are used in multiple discrete networks."_

MicroCloud already deploys and wires OVN via MicroOVN during
`microcloud init`, including creating the `UPLINK` physical network from
the gateway and range supplied at the prompts (`ovn.ipv4_gateway` /
`ovn.ipv4_range` in a preseed). On a bare LXD or Incus host this is the
single fiddliest piece of setup; here it is free.

### 3.2 Two different uplink address pools — don't conflate them

The `UPLINK` network carries two separate config keys, and they do
different jobs. This tripped me up initially and is worth being precise
about:

| Key | Purpose | Consumption |
| --- | --- | --- |
| `ipv4.ovn.ranges` | Each OVN network's virtual router gets one address here for outbound SNAT. Allocated by LXD automatically at network creation. | 1 per provisioned network |
| `ipv4.routes` | The set of addresses permitted as **network forward listen addresses**. | 1 per team (see §3.4) |

**Neither of these is a public IP.** `UPLINK` is a private L2 segment
(the docs use `192.0.2.0/24`). Public addressing happens at the site
border, outside LaForge entirely.

At the agreed target scale — **10 teams × 2 networks** — that is 20
addresses for `ipv4.ovn.ranges` and 10 for `ipv4.routes`, so roughly 30
total with headroom. A `/24` uplink is ample. The only real risk is that
`microcloud init` sized the range without anyone thinking about it.

**Action before the first real build: check `UPLINK`'s
`ipv4.ovn.ranges`, and set `ipv4.routes`** (MicroCloud's init does not
prompt for the latter at all).

### 3.3 Intra-team routing needs OVN peerings

OpenStack gives a team one router with every team network hanging off
it, so hosts on different networks within a team route to each other for
free. LXD OVN gives **every network its own logical router**.

**Without peering, intra-team cross-subnet traffic is dropped — not
degraded, dropped.** `lr-vdi`'s entire routing table is *connected
10.0.2.0/24* and *default via the uplink gateway*. A packet for
`10.0.1.5` has nothing to match but the default route, so it leaves via
the gateway port and lands on the physical uplink router, which has no
route to `10.0.1.0/24` either — and could never be given one, because
every team uses that same CIDR. The packet dies there.

The SNAT that happens on the way out is incidental to that outcome. It
would only matter in a hypothetical where someone hand-added uplink
routes for the internal subnets, at which point the traffic would work
but arrive with a rewritten source address and hairpin through the
border router. That hypothetical is unreachable anyway because of the
CIDR collision, so treat the failure mode as "no connectivity", full
stop.

The fix is OVN **network peering** (`CreateNetworkPeer`). LXD installs a
static route on each logical router for the other's subnet, pointing at
a direct router-to-router port. Traffic then matches that more-specific
route instead of the default, egresses the peer port rather than the
gateway port, and so never hits the SNAT rule — which in OVN is bound to
the gateway port. Source addresses survive and the traffic never leaves
OVN.

A team with _n_ networks needs _n(n-1)/2_ pairs. The agreed topology is
**two networks per team — VDI plus competition hosts — so exactly one
bidirectional peering per team.** Calling it a "mesh" oversells it. The
code should still handle the general case, since adding a third network
later is a plausible change, but nothing here is complex at this scale.
Upstream has an open feature request for multiple subnets on one logical
router, which would remove even this.

**Two traps here, both of which change the code:**

1. **Peerings must be mutual.** Creating one side leaves the
   relationship `pending`; it only activates when the mirror exists.
2. **Wrong names fail silently.** Per the LXD docs, if the target
   project or network does not exist, the API returns **no error** — the
   peering just stays `pending` forever. This is deliberate (it stops
   callers probing for networks in other projects) but it is a real
   footgun for generated names. After creating both sides the builder
   must read the peer back and assert `status == "created"` rather than
   trusting a nil error.

Since networks arrive one at a time, build them incrementally in
`DeployNetwork`: peer the new network against every sibling already
listed in the team's `lxd_networks` var. That avoids needing a
post-pass, which the `Builder` interface has no hook for anyway.

#### Alternatives considered

**One large OVN network per team, segmented by ACLs.** If a team's
subnets sit inside a common supernet, create one `/16` OVN network and
treat the LaForge networks as address ranges within it. One router,
everything routes, no peering. Rejected: it is a single flat L2, so
there is no real segmentation to discover or pivot across, ARP scans see
the whole team, and TTLs do not decrement between "subnets". For a
pentest competition that is a meaningful loss of realism, and it also
breaks the per-network `.254` gateway convention.

**A per-team router/firewall VM** (pfSense/VyOS/Linux). Team networks
set `ipv4.nat=false` and DHCP hands out the VM as gateway via
`ipv4.dhcp.gateway`. This is the most realistic option and it composes
with §3.4 — the same VM could be both the team edge firewall and the VDI
ingress, making it a scoreable artifact rather than invisible plumbing.
Costs an instance per team, multi-NIC ordering complexity in the
builder, and hairpins all intra-team traffic through one VM.
**Decided against for now** — peerings are cheaper and need no
instances. Revisit only if we later want teams graded on an edge
firewall.

### 3.4 External access: one public IP, one forward per team

Given that each team has exactly one externally reachable service (the
VDI proxy — Guacamole, a Node proxy, WireGuard, whatever) and no other
public exposure, the design collapses nicely.

A network forward belongs to one OVN network and its targets must be
addresses **within that network's subnet**, so a single forward cannot
span teams. But one forward can expose many ports to many targets inside
its own network. That gives us:

- **One forward per team**, on that team's VDI network, targeting the
  team's ingress host.
- **One listen address per team**, allocated out of `UPLINK`'s
  `ipv4.routes`. Private, not public.
- **One public IP total**, at the site border, port-mapped to the
  per-team listen addresses. This is exactly the one-IP/many-ports model
  already used with WireGuard, and it lives in border config, not in
  LaForge.

LaForge's whole job here is to create the forward and record the listen
address in `Team.Vars["ingress_address"]`, so border and Guacamole
config can be generated from it.

**LXD allocates the listen address for us** — pass `"0.0.0.0"` as
`ListenAddress` and it picks a free one from `ipv4.routes`. This is
gated on the `network_allocate_external_ips` API extension, which I
confirmed is present in LXD 5.21.3. So there is no custom IPAM to write;
guard it with `HasExtension()` and read the assigned address back.

The builder should stay agnostic about *what* the ingress service is.
Declare it as an ordinary LaForge `Host` in the VDI network carrying
`vars = { public_ingress = "true" }`, and forward the ports already
listed in its `exposed_tcp_ports`. Swapping Guacamole for WireGuard then
becomes an environment-config change with no builder work.

### 3.5 Cross-team isolation is free

Teams must not reach each other, and that is the default rather than
something we configure:

- Peerings are explicit and named. Two networks route to each other only
  because we asked for it, so there is simply no path between projects.
- Even the uplink path is a dead end. Every team uses identical CIDRs,
  so the uplink router cannot hold a route to `10.0.1.0/24` for one team
  without ambiguity — there is nothing to route to.
- LXD additionally installs anti-spoofing router policies that drop
  traffic arriving on the uplink port bearing a peer network's source
  addresses, so a team cannot forge its way into a peering it is not
  part of.

Instances SNAT outbound onto `UPLINK`, so they can address the uplink
subnet — including other teams' forward listen addresses. **This is
accepted and needs no mitigation.** Those ingress services are
internet-reachable by design and are protected by per-team credentials,
so a team reaching another team's proxy is no different from any
internet user doing so. Do not add ACLs for this.

### 3.5.1 Egress is just plain internet

The LaForge server lives on a public internet address and is not
attached to the MicroCloud cluster at all. Same for Splunk. So there is
no split-horizon routing to design and no special-case paths: **every
instance needs ordinary internet-routable egress, exactly like any other
host on the network.**

That falls out of the design with no extra work. Each OVN network has
`ipv4.nat=true`, so instances SNAT to their router's uplink address; the
uplink gateway routes to the internet. Nothing else to configure.

Two requirements follow:

- The uplink gateway must have a working default route to the internet.
- `dns.nameservers` on each OVN network must resolve public names, so
  the agent can find the LaForge and Splunk hostnames. Either point
  `entEnvironment.Config["master_dns_server"]` at a resolver that
  forwards externally, or fall back to a public resolver.

#### The server addresses must be correct *before* the build runs

The gRPC server address is compiled into each agent binary rather than
read from config at runtime:

```624:624:planner/plan.go
		err = grpc.BuildAgent(logger, fmt.Sprint(entProvisionedHost.ID), laforgeConfig.Agent.GrpcServerUri, binaryName, isWindowsHost, laforgeConfig.AgentDebug)
```

**This is deliberate and we are keeping it.** The agent binary is the
only LaForge artifact on a competition host, so there is no config file
for a student to edit and repoint. Tampering then requires patching the
binary, and an agent that stops checking in is visible. Hosts live for a
weekend and the server hostname is stable, so there is no operational
need for runtime configurability.

The consequence is purely one of ordering: the value has to be right
before agents are rendered. Three settings carry the LaForge address and
**all three default to `localhost` in this repo**:

| Setting | File | Consequence if left as localhost |
| --- | --- | --- |
| `agent.grpc_server_uri` | `conf.json` | Every agent binary compiled pointing at localhost. Nothing checks in, and fixing it means re-rendering and rebuilding all agents. |
| `agent.api_download_url` | `conf.json` | Provisioning-step downloads (scripts, files, ansible) all fail. |
| `laforge_server_url` | `configs/microcloud.json` | cloud-init bootstrap URL is wrong, so the agent never installs. |

None of this is MicroCloud-specific — every builder has the same
requirement today and none of them validate it, so this builder should
not grow a special case either. It is noted here only because it is the
most likely cause of a "host boots fine, agent never appears" failure at
Phase 4, and that is worth knowing before spending an afternoon on it.

### 3.6 No network ACLs

East-west filtering between a team's own subnets is handled by host-based
firewalls, not by the fabric. That removes the entire ACL layer from the
builder:

- No per-host `NetworkACL` object, and no `security.acls` on the NIC.
- `Host.ExposedTCPPorts` / `ExposedUDPPorts` feed **only** the team
  ingress forward's port list (§3.4), not security-group rules. The
  port-range parsing from `openstack.go` still ports over, but it is
  called once per team instead of once per host.
- Agents reach the LaForge server with no special allowance. Outbound
  works by default because OVN SNATs through the uplink
  (`ipv4.nat=true`); the only deployment requirement is that the uplink
  can route to the LaForge server, for both the cloud-init agent
  download and the gRPC check-in.

**This also dodges a trap.** The moment `security.acls` is attached to a
NIC, the default action for *unmatched* traffic becomes `reject` in
**both** directions. An ACL added later for one narrow purpose would
silently cut off every agent check-in on that network. If we ever do
attach ACLs — for the §3.5 caveat, say — the very first rule has to be
an explicit allow-all egress.

## 4. Other MicroCloud-specific differences

### 4.1 Storage is Ceph

MicroCloud creates a `remote` pool backed by MicroCeph alongside
`local`. Root disks belong on `remote`: it is the only option that lets
LXD move an instance between members, and it is what makes cluster
healing meaningful.

### 4.2 Placement: one team per cluster member

The OpenStack builder hand-rolls placement — `getOptimalAvailabilityZone`
lists hypervisors, counts running VMs per AZ, and picks the emptiest
(~70 lines). **Drop all of it.** LXD schedules to the least-loaded
member by default, and we want deliberate placement anyway.

**Decision (confirmed): pin each team to a single cluster member**,
spreading teams across members, using cluster groups. Two reasons:

1. **Locality.** Peered traffic uses the distributed router-to-router
   port rather than a gateway port, so its pipeline runs on the source
   chassis. Co-locate a team's instances and intra-team east-west never
   leaves that host's OVS — no Geneve, no physical NIC. Peering is what
   makes this reachable; placement alone does nothing (see the note
   below).
2. **Blast radius.** A member failure takes one team fully offline
   rather than degrading all of them. That is the accepted tradeoff.

Implement it **declaratively**, not by threading `UseTarget()` through
every call: create a one-member cluster group per team and restrict the
team's *project* to it. LXD then enforces placement for every instance
in the project, including any created outside the builder. Both
`clustering_groups` and `projects_restricted_cluster_target` are present
in 5.21.3, and the client has full `ClusterGroup` CRUD.

> **Note: placement does not affect routing, and gateways stay
> scattered.** OVN routes on logical tables, not locality, so
> co-location never removes the need for peering. Worse, LXD assigns
> each network's gateway chassis a *stable-random* priority derived from
> the chassis group name and node ID (`addChassisGroupEntry` in
> `driver_ovn.go`), with no config key to override it. A co-located
> team's network gateways will still land on random members, so any
> traffic that takes a gateway port tunnels off-host. Only peered
> traffic gets the locality benefit.

Still call `GetClusterMembers()` once at builder construction, to log
topology and fail fast if any member is not `Online`.

### 4.3 Authentication

TLS client-certificate trust:

```shell
lxc config trust add laforge.crt --name laforge
```

Same model as the existing NSX-T principal-identity client, so the
config carries cert/key/CA paths in a familiar shape. LXD 5.21 also
supports OIDC and fine-grained authorization groups, which is
interesting for scoping team access to their own project, but that is a
competition-operations feature, not a builder feature.

## 5. Dependency and toolchain

The hardware runs **MicroCloud 2.x LTS**, which ships **LXD 5.21 LTS**,
so pin the client to that series — `lxd-5.21.3`, pseudo-version
`v0.0.0-20250122082529-1d9ac8759979`.

**This forces a Go toolchain bump.** There is no LXD tag consumable at
Go 1.21:

| LXD tag | `go` directive |
| --- | --- |
| lxd-5.0.3 | 1.19 |
| lxd-5.20 | 1.20 |
| lxd-5.21.2 | 1.22.4 |
| lxd-5.21.3 | 1.23.3 |
| lxd-5.21.4 | 1.24.5 |
| main | 1.26.7 |

I confirmed the impact by running the `go get` in a scratch `go 1.21`
module: it silently rewrote the directive to `go 1.23.3`. LaForge pins
1.21 in three places — `go.mod`, `.go-version`, and the `Dockerfile`
(`golang:1.21-bullseye`) — all three move together. Do this as its own
change, separate from the builder, since the blast radius includes ent
0.12.5 and gqlgen 0.17.41.

Two more dependency notes:

- LXD publishes **no semver module tags** — `proxy.golang.org`'s version
  list for `github.com/canonical/lxd` is empty. Pin a pseudo-version:
  `go get github.com/canonical/lxd/client@lxd-5.21.3` resolves to
  `v0.0.0-20250122082529-1d9ac8759979`. Leave a comment in `go.mod` so
  nobody "fixes" it later.
- Otherwise it is a clean dependency. Verified: pure Go, no cgo,
  `CGO_ENABLED=0 GOOS=linux GOARCH=amd64` cross-compiles fine, and the
  actually-linked module set is small (`canonical/lxd`, `pongo2`,
  `go-jose/v4`, the `zitadel/oidc` trio, `otel`, `go-logr`, plus
  `golang.org/x/*` we already have). The alarming entries in the module
  graph — `go-lxc`, `go-libvirt`, `criu`, `libovsdb`, `gobgp` — are
  daemon-side and pruned at build time.

## 6. Package layout

```
builder/microcloud/
  microcloud.go         # MicroCloudBuilder + the 6 interface methods
  client.go             # connection, project scoping, operation waiting
  naming.go             # generateProjectName / NetworkName / InstanceName
  peering.go            # intra-team peerings
  placement.go          # per-team cluster group + project restriction
  userdata.go           # cloud-init agent bootstrap
  README.md             # MicroCloud + border setup guide
  examples/
    deploy-team/  deploy-network/  deploy-host/
    teardown-team/ teardown-network/ teardown-host/
```

No `ipam.go` — §3.4 hands address allocation to LXD.

Plus three integration points, all trivial:

- `builder/builder.go`: a `case "microcloud":` and `NewMicroCloudBuilder()`.
- `configs/microcloud.json.example`.
- `docs/builders.md`: list the `microcloud` slug.

The `examples/*/main.go` harnesses are the primary dev loop — they load
the real config, pull the newest build out of Postgres, and invoke one
builder method against live hardware. Copy them from
`builder/openstack/examples/`.

## 7. Config file shape

`configs/microcloud.json.example`:

```json
{
  "laforge_server_url": "https://laforge.example.com",
  "max_build_workers": 8,
  "max_teardown_workers": 8,
  "base_url": "https://microcloud-01.example.com:8443",
  "client_cert_path": "/etc/laforge/microcloud/client.crt",
  "client_key_path": "/etc/laforge/microcloud/client.key",
  "server_cert_path": "/etc/laforge/microcloud/server.crt",
  "storage_pool": "remote",
  "uplink_network": "UPLINK",
  "instance_type": "virtual-machine",
  "instance_sizes": {
    "nano": { "cpu": "1", "memory": "1GiB" },
    "small": { "cpu": "2", "memory": "4GiB" },
    "medium": { "cpu": "4", "memory": "8GiB" },
    "large": { "cpu": "8", "memory": "16GiB" }
  },
  "images": {
    "ubuntu22": "ubuntu22",
    "w2k19": "w2k19",
    "w2k22": "w2k22"
  }
}
```

Images: pre-import one per `Host.OS` string, aliased to match. Keeping
the mapping explicit mirrors the OpenStack `images` UUID map and avoids
a simplestreams round trip per host.

```shell
lxc image copy ubuntu:22.04 local: --alias ubuntu22 --vm
```

## 8. Phases

Each phase ends where the matching `examples/` harness runs green
against the cluster.

### Phase 0 — Toolchain bump

`go.mod`, `.go-version`, `Dockerfile` to 1.23.x. Standalone change,
merged and tested before any builder work.

### Phase 1 — Plumbing

Add the dependency, the `MicroCloudBuilder` struct with six stubbed
methods, the config struct, and the `builder.go` switch case. Confirm an
environment with `builder = "microcloud"` resolves.

### Phase 2 — Connection and teams

```go
server, err := lxd.ConnectLXD(cfg.BaseUrl, &lxd.ConnectionArgs{
    TLSClientCert: clientCertPEM,
    TLSClientKey:  clientKeyPEM,
    TLSServerCert: serverCertPEM,
})
```

`DeployTeam` creates the project; everything downstream runs through
`server.UseProject(name)`:

```go
err = server.CreateProject(api.ProjectsPost{
    Name: builder.generateProjectName(entEnvironment, entTeam, entBuild),
    ProjectPut: api.ProjectPut{
        Config: map[string]string{
            "features.networks": "true",
            "features.profiles": "true",
            "features.images":   "false",
        },
    },
})
```

It also creates the team's one-member cluster group and pins the project
to it, per §4.2 — member chosen round-robin by `TeamNumber` over
`GetClusterMembers()`.

### Phase 3 — Networks and peerings

One OVN network per provisioned network, gateway pinned to `.254` to
match the convention the OpenStack builder and the LaForge templates
already assume:

```go
err = projectServer.CreateNetwork(api.NetworksPost{
    Name: networkName,
    Type: "ovn",
    NetworkPut: api.NetworkPut{Config: map[string]string{
        "network":         cfg.UplinkNetwork,   // "UPLINK"
        "ipv4.address":    routerAddress + "/24", // x.x.x.254/24
        "ipv4.nat":        "true",
        "ipv6.address":    "none",
        "dns.nameservers": dnsServer, // entEnvironment.Config["master_dns_server"]
    }},
})
```

Then peer against the team's existing networks, both directions, and
verify each pair actually came up (§3.3):

```go
for _, sibling := range siblings {
    err = projectServer.CreateNetworkPeer(networkName, api.NetworkPeersPost{
        Name:          sibling,
        TargetProject: projectName,
        TargetNetwork: sibling,
    })
    // ...and the mirror call on `sibling` targeting `networkName`

    peer, _, err := projectServer.GetNetworkPeer(networkName, sibling)
    if err != nil || peer.Status != "created" {
        return fmt.Errorf("peering %s<->%s did not activate (status %q)",
            networkName, sibling, peer.Status)
    }
}
```

Deliverable, and the one that proves the whole design: two teams' worth
of networks with identical CIDRs coexisting; a host on a team's VDI
network reaching a host on a sibling competition network **with its real
source address intact**; and no path at all to the other team.

### Phase 4 — Hosts (Linux)

Port the OpenStack `DeployHost` body, minus the whole security-group
section (§3.6):

```go
op, err := projectServer.CreateInstance(api.InstancesPost{
    Name:  instanceName,
    Type:  api.InstanceTypeVM,
    Start: true,
    Source: api.InstanceSource{Type: "image", Alias: cfg.Images[entHost.OS]},
    InstancePut: api.InstancePut{
        Config: map[string]string{
            "limits.cpu":           size.CPU,
            "limits.memory":        size.Memory,
            "cloud-init.user-data": userData,
        },
        Devices: map[string]map[string]string{
            "eth0": {
                "type": "nic", "network": networkName,
                "ipv4.address": hostAddress,
            },
            "root": {
                "type": "disk", "pool": cfg.StoragePool, "path": "/",
                "size": fmt.Sprintf("%dGiB", entDisk.Size),
            },
        },
    },
})
err = op.Wait()
```

`userData` is byte-for-byte the bootstrap script already in
`openstack.go`; `cloud-init.user-data` replaces Nova's `UserData`.

Deliverable: a Linux host boots, takes its static `LastOctet` address,
and the agent checks in over gRPC.

### Phase 5 — Team ingress

The forward from §3.4, for the host carrying
`vars = { public_ingress = "true" }`:

```go
err = projectServer.CreateNetworkForward(vdiNetworkName,
    api.NetworkForwardsPost{
        ListenAddress: "0.0.0.0", // LXD allocates from UPLINK ipv4.routes
        NetworkForwardPut: api.NetworkForwardPut{
            Config: map[string]string{"target_address": ingressHostAddress},
            Ports:  portsFromExposedTCP,
        },
    })
```

Read the assigned address back and store it in
`Team.Vars["ingress_address"]`. Deliverable: RDP reaches a VDI host
through the team's proxy, from off-cluster, via the single border IP.

### Phase 6 — Windows

Deferred deliberately. LXD runs Windows VMs fine but there is no
cloud-init, so the agent has to arrive another way. In order of
preference: bake it into the image and self-update (effectively what the
vSphere builder relies on); cloudbase-init consuming the same NoCloud
seed; or `CreateInstanceFile` + `ExecInstance` once the LXD agent is up
in the guest. Do not block Phases 2–5 on this.

### Phase 7 — Hardening

Retry/backoff around operations, `waitForObject`-style polling helpers
ported from `openstack.go`, teardown that tolerates already-deleted
objects, worker-pool tuning.

## 9. Open questions to resolve on the hardware

### Settled

| Question | Decision |
| --- | --- |
| Placement | One team per cluster member, via one-member cluster groups + project restriction (§4.2) |
| Team edge | OVN peerings only; no router VM (§3.3) |
| Instance type | VMs everywhere (§7) |
| MicroCloud channel | 2.x LTS → LXD 5.21 → pin `lxd-5.21.3` (§5) |
| Scale | 10 teams × 2 networks → ~30 uplink addresses, 1 peering per team (§3.2, §3.3) |
| Team↔team ingress access | Accepted; per-team credentials handle it (§3.5) |
| LaForge/Splunk reachability | Both public internet; instances get plain internet egress, no special routing (§3.5.1) |

### Still open — needs the hardware

1. **Do OVN peers work between two networks in the same project?** The
   API takes a `TargetProject`, which suggests yes, and the entire
   intra-team design rests on it. Prove this first. Capture on the
   destination and confirm the source address is the real internal one,
   not the router's uplink address.
2. **Is `UPLINK` actually configured for this?** Check `ipv4.ovn.ranges`
   has room for ~30, and set `ipv4.routes` — `microcloud init` never
   prompts for it, so it is almost certainly unset (§3.2).
3. **Does an instance get working internet egress and public DNS?**
   (§3.5.1) Boot one host, resolve and reach the LaForge and Splunk
   addresses. If it fails, check the uplink default route and
   `dns.nameservers` before anything else.
4. **Does MicroCeph `remote` storage honour the root-disk `size`
   override** the same way local pools do? Ceph RBD sizing behaves
   differently from ZFS/dir, and this is the kind of thing that only
   fails at scale.
5. **What does `Vars`-based idempotency look like on rebuild?** The
   OpenStack builder leans on `delete(newVars, ...)` during teardown;
   confirm that holds when an LXD operation fails halfway.

Items 1–3 are all answerable during Phase 3 and should be, since each
can invalidate work done later.

## 10. Sequencing

Phase 3 is the one that can invalidate the design, and Phases 0–2 are
mostly mechanical. Get to Phase 3 fast and spend real time there before
writing host code. If overlapping-subnet isolation or the peerings do
not behave, the fallback is per-team CIDR remapping in
`planner/plan.go` — a much larger and more invasive change that would
affect every existing builder, and we want to know that early.
