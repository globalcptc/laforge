# Writing a LaForge Builder

This guide is for a human or an agent implementing a new **builder** — the
hoster-specific layer that turns LaForge's abstract, builder-agnostic content into
real infrastructure on some platform (Incus, MicroCloud/LXD, AWS, OpenStack, or
something new). It documents the two interfaces a builder implements, the
non-negotiable requirements, how to wire a new one in, and — most valuably — the
lessons learned building the first four, so you don't rediscover them the hard way.

Read this alongside the code: `internal/builder/builder.go` (the `Builder`
contract), `internal/builder/onboard.go` (the `Onboarder` contract), and the
`internal/builder/incus` package (the reference implementation — the most complete
and the only one verified live with real packets end to end).

---

## 1. Philosophy: a builder is deliberately dumb

A builder takes a **fully resolved spec** (a deterministic external name the caller
already decided, plus abstract fields) and does **exactly one thing per call, safe
to repeat**. It:

- **Talks to the hoster API directly.** No Terraform/OpenTofu/Ansible layer.
- **Knows nothing about Postgres, scheduling, teams, or builds.** Those live in
  `internal/orchestrator` and `internal/runner`. A builder is only ever the thing
  on the other end of one call.
- **Is builder-agnostic at the content layer.** The same environment file deploys
  on any builder without change. There is **no capabilities opt-out**: every
  builder MUST implement the whole contract for real. If a platform lacks a
  primitive (a cloud with no container runtime), the builder uses whatever
  workaround the platform offers (Fargate, Zun, native OCI, nested Docker). If it
  genuinely cannot, it is simply not a valid builder — it fails at deploy time with
  a clear error, never a flag that silently varies the environment per builder.

### The two hard rules

1. **Every op is `ensure` (create-or-adopt), keyed by a deterministic name/tag.**
   A retry after partial success must converge, not duplicate. `Deploy*` returns
   the same `externalRef` whether it created the resource or adopted an existing
   one. `Destroy*` on an already-gone resource is success, not an error.

2. **The caller owns the name; the builder just uses it.** `ExternalName` is
   assigned by `internal/orchestrator`, decided once, handed to every retry
   unchanged. This is what makes "ensure" work at all — see §3.

---

## 2. The `Builder` interface

Every method, with its real contract (`internal/builder/builder.go`):

```go
type Builder interface {
    DeployNetwork(ctx, NetworkSpec)   (externalRef string, err error)
    DeployHost(ctx, HostSpec)         (externalRef string, err error)
    DeployContainer(ctx, ContainerSpec) (externalRef string, err error)

    DestroyNetwork(ctx, team, externalRef string) error
    DestroyHost(ctx, team, externalRef string)     error
    DestroyContainer(ctx, team, externalRef string) error

    Inspect(ctx) ([]Resource, error)

    OpenAccess(ctx, team string)  error
    CloseAccess(ctx, team string) error

    PowerAction(ctx, team, externalRef, action string, force bool) error

    ConfigureNetworkAccess(ctx, team string, networks []NetworkAccess) error
}
```

### Deploy\* — create or adopt, return the ref

- **Adopt first:** look for a resource already tagged/named with `spec.ExternalName`
  and return its ref if found, before creating anything.
- **Return the deterministic ref even on failure** where the resource may have been
  partially created (see the config-drive lesson in §6) — the runner records it via
  `SetDeployedObjectExternalRef`, so teardown can still destroy an orphan.
- `DeployHost` attaches the instance to `spec.Network` (the team's network) — never
  the hoster's default network, or you get zero team isolation (a real bug found
  live; see §6).
- `DeployContainer`: use the platform's container primitive. Native OCI where it
  exists (Incus), nested Docker where it doesn't (MicroCloud/LXD), Fargate (AWS),
  Zun (OpenStack). The agent is delivered *into* the container (`AgentBinary` for a
  pushed binary, `AgentDownloadURL` for fetch-at-start) so it checks in and runs
  steps/validators exactly like a host.

### Destroy\* — idempotent, takes `team`

`team` is passed because a *pool* builder (`incuspool`: many independent hosts, no
shared endpoint) needs it to find which host holds `externalRef`. Single-endpoint
builders ignore it. A destroy of a not-found resource returns nil (success).

### Inspect — list what actually exists

Returns `[]Resource{ExternalRef, Kind, State}` for adoption and drift detection.
`State` is a **normalized** power state (`PowerStateRunning`/`Stopped`/`Other`, or
`""` for a network or a builder that can't report it) — map the hoster's own
vocabulary onto these. This is the load-bearing input for the power-state poll, so
get instance liveness right even if you defer network drift.

### OpenAccess / CloseAccess — behavioural, team-level

The contract: **closing a team must block all external ingress AND terminate
established connections, not merely stop new ones.** This is the one place a
"stateful firewall that only blocks new flows" is not good enough.

- Incus/MicroCloud: remove the instance NIC (`type: none`), which drops in-flight
  connections. This is the gold standard — it actually severs established flows.
- AWS/OpenStack (DRAFT): security-group revoke blocks new flows but SGs are
  **stateful**, so established flows linger. Fully honoring the contract needs a
  stateless NACL deny / ENI bounce (AWS) or conntrack flush on the compute host
  (OpenStack). Documented as a DRAFT gap, not silently under-delivered.

### PowerAction — infrastructure-level, agent-independent

`start` / `stop` / `reboot`, addressed by `externalRef`, working even when the
in-guest agent is dead. `force` selects a hard power operation (immediate
power-off/reset) over a graceful ACPI one; meaningless for `start`.

### ConfigureNetworkAccess — the network firewall (team-level)

Enforces two things together:

1. **`visible_from`** — which sibling networks may reach a network (default-deny).
2. **`ports:`** — the per-host ingress firewall: a host is reachable only on its
   declared TCP/UDP ports; a host that declared none is reachable on nothing
   ("allow open ports and deny the rest").

It is **team-level, not per-network**, because the policy is inherently
cross-network (a network's rule references its siblings) and must never leak across
teams (CIDRs are identical across teams — isolation comes from a per-team routing
domain, never from CIDR rules alone).

Input is `[]NetworkAccess`, each carrying the network's `CIDR`, its `VisibleFrom`
sibling external-names, and its `Hosts` (`[]HostAccess{Address, ExternalRef,
TCPPorts, UDPPorts}`). See §5 for the OVN vs security-group mechanics and the
hard-won lessons.

### The spec types grew fields on purpose

`Team`, `Network`, `Address`, `NetworkDisplayName`, ports, and the agent fields were
each **added when a real build proved them missing**, not designed up front. The
pattern: a builder that doesn't need a field ignores it (the `fake` builder ignores
almost all of them). Follow that shape if you add one — make it advisory where you
can, and never break the builders that don't consume it.

---

## 3. Deterministic naming — why it matters

`ExternalName` is a hash-derived, caller-assigned identity. Builders turn it into a
hoster resource name and also derive a readable name from `DisplayName` where the
hoster allows (see `incus.instanceName` / `incus.networkName`). Two payoffs:

- **Ensure/idempotency.** The name is decided once, so a retry finds and adopts
  what a partial previous attempt created.
- **No dependency-ordering system needed.** A host's `Network` field is the *same*
  deterministic name `DeployNetwork` used, computed by the caller regardless of
  whether the network's own deploy has run yet. If the network isn't there,
  attaching fails with a real error, which the runner's task retry/backoff resolves
  once the network task completes. Don't build cross-resource waiting into a
  builder — let deterministic names + retry do it.

Tag/label **every** resource you create with the external name, the team, and a
"managed by LaForge" marker. That triple is how `Inspect` and adopt find things.

---

## 4. The `Onboarder` interface

Onboarding is the *other half* of the contract, kept separate from `Builder` on
purpose (`internal/builder/onboard.go`). `Builder` is an already-configured, running
hoster. `Onboarder` is what happens **before** that: connecting a brand-new hoster
and reading what it has.

```go
type Onboarder interface {
    Onboard(ctx, OnboardRequest)  (*Onboarding, error) // connect + validate + discover
    Rediscover(ctx, Connection)   (*Discovery, error)  // re-read from stored creds
}
```

- It differs **fundamentally by type**, which is why it can't be a `Builder` method:
  Incus/MicroCloud onboard by **enrolling a trust token** (a certificate exchange,
  producing stored client/server certs in `Connection`); AWS/OpenStack "onboard" by
  **validating environment credentials** and store no per-hoster secret
  (`Connection.HasCredential()` is false).
- `Onboard` returns `Connection` (what to persist in `builder_credential`) plus
  `Discovery` (the pickable `StoragePools`, `Networks`, `Images` an operator chooses
  from). A cloud with no interactive discovery fills what it can (an AMI is an
  `ImageInfo`) or leaves slices empty (never nil — the wire must be a JSON array).
- Register from an `init()` with `builder.RegisterOnboarder(kind, o)`. Double
  registration panics at startup (a programming error, not last-wins).

Every builder package has an `onboard.go` — read `incus/onboard.go` (trust-token
flow) and `aws/onboard.go` (env-credential flow) for the two shapes.

---

## 5. Network access mechanics (the deep end)

This is where builders differ most and where the live-verification lessons are
concentrated. The goal is identical everywhere: **from an allowed source, a host is
reachable only on its declared ports; everything else is dropped; same-network and
cross-network alike.**

### Incus / MicroCloud (OVN) — verified live with real packets

The working design (proven live with a packet-level reachability probe, not
just "the daemon accepted the config"):

1. **Peering** for routing: full-mesh peer the team's OVN networks so they route.
   Peering only within a team is the isolation boundary.
2. **Network-level default-deny** ingress: set
   `security.acls.default.ingress.action: drop` (egress `allow`) on each network.
   This is what makes **same-switch** traffic default-denied — a critical, subtle
   point (see the lesson below).
3. **Per-host NIC ACL** for the allows: attach a `security.acls` ACL to each
   instance's `eth0` device (→ OVN `to-lport`), whose ingress rules allow the host's
   declared ports from its own network CIDR (same-net siblings) + its `visible_from`
   CIDRs (cross-net). No `destination` in the rule — the port itself is the
   destination. A host with no ports gets no allow rules → reachable on nothing.

Clean up per-host ACLs when the instance is destroyed (they're orphaned otherwise).

### AWS / OpenStack (security groups) — DRAFT, never run against an account

Per-host security group whose ingress rules allow the declared ports from
own-network + `visible_from` CIDRs, made the instance's authoritative group. Cloud
SGs are stateful and per-ENI/per-port, so — unlike OVN network ACLs — they filter
**same-subnet** traffic for free. Cross-network still needs routing (VPC peering or
a per-team router/subnet model), which the DRAFT does not yet build. See the file
headers in `aws/aws.go` and `openstack/openstack.go` for the documented gaps.

---

## 6. Lessons learned (read before you start)

These cost real debugging time on the four existing builders. Most are OVN/LXD, but
the meta-lessons apply to any platform.

### OVN / LXD ACLs

- **Default action goes at the NETWORK level; allow rules go on the NIC.** Setting
  `security.acls.default.ingress.action: drop` on the *NIC device* silently dropped
  **everything** — the allow rules were never consulted. Setting the default on the
  *network* and attaching the allow ACL to the NIC is what works. This one wasn't
  derivable from docs; it took a packet probe to find. If you touch ACLs, **verify
  with real packets**, not just "the API accepted it."
- **Network-level ACLs do NOT filter same-switch (intra-subnet) traffic**; only
  `to-lport` ACLs at the instance port do. That's why the port firewall lives on the
  NIC, not the network. (`ovn-nbctl acl-add ls to-lport ... 'outport == "vm-port"
  ...'` is the raw primitive; LXD's NIC `security.acls` generates it.)
- **A NIC (`to-lport`) allow rule must not carry a `destination`** — the port is
  already the destination; including one made it silently not match.
- **OVN allow rules are unioned over the default-deny.** To scope by both source and
  port you must combine them in one rule; a separate source-only rule re-opens all
  ports.
- **IPv6 uplink trap:** OVN auto-assigns an IPv6 subnet; if the uplink has no IPv6
  range, attaching an ACL (a full network re-validation) fails with
  `"volatile.network.ipv6.address cannot be empty..."`. LaForge is IPv4-only, so
  `DeployNetwork` and any network PATCH set `ipv6.address: none`.
- **Clustered LXD returns 500, not 409, when you re-POST an existing OVN network**
  ("Network is not in pending state"). An adopt-on-retry that keys off 409 breaks on
  a real cluster — do an explicit existence GET before create (found on MicroCloud).
- **MicroCloud is LXD, not Incus.** It reads `X-LXD-*` headers, not `X-Incus-*`;
  send both if you share wire code. MicroCloud and Incus are **separate builders**
  implementing one interface — never conflate them.

### Instances / lifecycle

- **Attach instances to the team network explicitly.** Falling back to the hoster
  default network meant every team landed on the same bridge — zero isolation,
  found live.
- **Record `externalRef` even on a failed create.** A config-drive import that fails
  *after* the instance is created leaves an orphan nothing can destroy unless the
  deterministic name/ref was recorded. Return the name even on error.
- **Config-drive is replace-on-retry, not adopt.** A one-time agent token is baked
  into the drive each attempt, so a stale drive from a timed-out attempt must be
  deleted and re-imported, never adopted.
- **Slow storage needs a longer operation timeout.** win2019 on Ceph exceeded a
  120s default; the create rode a 900s timeout to completion. Make the operation
  timeout configurable per builder.
- **Secure Boot off for lab VM images** — Ubuntu cloud images loop on "prohibited by
  secure boot policy" otherwise; these are throwaway hosts, so it buys nothing.

### Access / firewalls

- **Cloud security groups are stateful.** Revoking ingress blocks new flows but not
  established ones; `CloseAccess`'s "terminate established" needs a stateless NACL
  deny / ENI bounce (AWS) or conntrack flush (OpenStack). NIC removal (Incus) severs
  them for real.
- **Cloud SGs filter same-subnet; OVN network ACLs don't.** Don't assume one
  platform's ACL scoping matches another's — verify where the enforcement point is.

### Meta

- **DRAFT means fail loudly, never under-enforce silently.** An unfinished builder
  method returns a clear error rather than leaving a competition network open. A
  security control you can't verify is worse than one that refuses to run.
- **ACL/firewall work is not done until it's verified with real traffic.** Every
  OVN lesson above came from a packet probe contradicting what the API accepted.

---

## 7. Wiring a new builder in

1. Create `internal/builder/<kind>/` implementing `builder.Builder`. Add a
   compile-time assertion: `var _ builder.Builder = (*Builder)(nil)`.
2. Implement `builder.Onboarder` in `<kind>/onboard.go` and
   `builder.RegisterOnboarder("<kind>", ...)` from an `init()`.
3. Add the kind to `internal/builderconfig/resolve.go`'s `Resolve` switch (and
   `verify.go` / any kind-specific config plumbing). The DB stores builder config in
   shared `builder_config` columns; reuse them or add as needed.
4. Add the kind to the wizard UI (`ui/src/components/builder-wizard/StepType.tsx`),
   marked **Draft** with a warning until it's verified against a real account.
5. Keep content builder-agnostic — do **not** add a capability flag.

---

## 8. Testing requirements

- **Unit-test the pure logic** offline (rule generation, name derivation, spec
  mapping) — see `incus/ingressrules_test.go`. No daemon needed; runs in CI.
- **Live smoke test** the full `Builder` surface against real hardware/an account,
  gated behind env vars so it **skips cleanly** when they're unset (see
  `incus/builder_test.go`'s `liveClient` / `LAFORGE_INCUS_TEST_*`). Cover: network
  create/adopt/destroy, host up→running→adopt→destroy, Open/CloseAccess actually
  severing reachability, and container check-in.
- **For network access, probe real packets.** `incus/netaccess_probe_test.go`
  deploys instances on peered networks and measures reachability by timing (a
  dropped port hits the connect timeout; an allowed-but-closed port RSTs instantly),
  asserting declared-port-allowed / undeclared-port-dropped, cross- AND same-network.
  This is the only kind of test that would have caught the NIC-vs-network default
  lesson.
- `go build ./... && go vet ./... && go test ./...` must be clean. A new builder's
  live tests skipping (no hardware in CI) is expected and fine.

### Definition of done for a new builder

- All `Builder` methods implemented for real (no capability opt-out).
- `Onboarder` implemented and registered; connect + rediscover work.
- Ensure/idempotency holds (double-deploy adopts, double-destroy is nil).
- Team isolation is real (instances on the team network, not the default).
- `ConfigureNetworkAccess` enforces `visible_from` + `ports:`, **verified with a
  packet probe**, same-network included.
- `CloseAccess` terminates established connections (or the gap is documented as
  DRAFT with the specific fix named).
- Unit tests for pure logic; live smoke + packet probe that skip without env.
- Wizard entry (Draft until account-verified); `resolve.go` wired.
