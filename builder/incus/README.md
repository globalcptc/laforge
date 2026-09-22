# Incus builder

Deploys LaForge environments to one or more standalone
[Incus](https://linuxcontainers.org/incus/) hosts. It is meant for test
builds and small events. For a clustered deployment with shared storage and
OVN, use the [MicroCloud builder](../microcloud/README.md).

Status: implemented and unit tested. **Not yet validated against a live Incus
host.** See [Unvalidated assumptions](#unvalidated-assumptions).

## How it works

- Each team is placed on **one** host and never spans hosts. With several
  hosts configured, team `N` goes to `hosts[(N-1) % len(hosts)]` and is
  pinned in `Team.Vars["incus_host"]`. Host order in the config is
  therefore significant.
- Each team gets its own Incus project (`features.networks=false`, so the
  team can reach the shared transit bridge) and one **gateway** container.
- Each LaForge network becomes an isolated Incus bridge with no host address,
  DHCP or NAT. Every team reuses the same CIDRs, and because nothing on the
  host has an address in them they never collide in the host routing table.
- The gateway is a plain container created from `gateway_image`. It has one
  NIC on each team bridge (at `.254`), and one NIC on the host's NAT'd
  `transit_network`. It provides, in place of OVN:
  - **DHCP** (dnsmasq, static reservations only). Each VM NIC gets a
    deterministic MAC, and the gateway reserves the LaForge `SubnetIP` for
    it, so Linux and Windows are treated identically.
  - **Egress** (nftables masquerade out of the transit NIC), so agents can
    download and call back to the LaForge server.
  - **Routing** between the team's own networks (IP forwarding).
  - **Ingress** (see below).
- The gateway is reconciled from the ent rows on every network and host
  deploy or teardown (`gateway.go`), under a per-team lock.

## Ingress on a shared public IP

Each host has one public IP shared by all its teams, so ports are offset per
team. For a host with `ingress_port_base` B and `ingress_port_stride` S, team
`T`'s Nth exposed port (TCP first, then UDP, each sorted) is published on
`ingress_listen_address` port `B + T*S + N`, forwarded by a proxy device on
the team gateway to the ingress host.

Only single port numbers are supported in `exposed_tcp_ports` and
`exposed_udp_ports` (no ranges), one ingress host per team
(`vars = { public_ingress = "true" }`), and an ingress host may expose at most
`ingress_port_stride` ports. The mapping is recorded in
`Team.Vars["ingress_ports"]` as JSON, e.g. `{"tcp/443": 10301}`, and the host
IP in `Team.Vars["ingress_address"]` and the host's `PublicIP` var. Users must
connect to the mapped port, not the standard one.

## Host preparation

On each Incus host:

```shell
incus config set core.https_address :8443
# client cert generated on the LaForge box, see docs/session-handoff.md
incus config trust add-certificate client.crt --name laforge

# Gateway image (container, with cloud-init). Packages are installed at
# first boot, so the transit bridge needs outbound internet.
incus image copy images:debian/12/cloud local: --alias lf-gateway

# VM images, keyed in "images" by the LaForge host `os` value
incus image copy images:ubuntu/22.04/cloud local: --alias ubuntu22 --vm
```

`transit_network` must be an existing managed `bridge` network with NAT
(usually `incusbr0`). Copy the server certificate
(`/var/lib/incus/server.crt`) to the LaForge box; `base_url` must match its
SAN.

Register the builder in `conf.json` / `conf.prod.json` under a friendly name
with `"builder": "incus"` (`conf.prod.json.example` already has an `incus`
entry), and copy one of the example configs:

- [`configs/incus.json.example`](../../configs/incus.json.example): one host
  with ingress, plus a Windows image entry using `image_config`.
- [`configs/incus-multihost.json.example`](../../configs/incus-multihost.json.example):
  two hosts, teams spread round-robin.
- [`example.laforge`](example.laforge): a sample environment using
  `builder = "incus"` with two networks and a public ingress host. Check it
  with `go run ./cmd/laforge-test builder/incus/example.laforge`.

## Windows

Windows is not blocked by the builder, but the image contract is
unvalidated. The builder sends PowerShell user data (`<powershell>`) in
`cloud-init.user-data`, so the image needs cloudbase-init (or equivalent) that
consumes it, the Incus agent, and virtio drivers. Use `image_config` to set
per-OS instance options such as `security.secureboot=false`, and size Windows
hosts generously in `instance_sizes`.

## Unvalidated assumptions

These need a hands-on check on a real host before relying on the builder:

1. Non-NAT proxy devices connect from inside the gateway's network namespace,
   so they can reach team bridge addresses.
2. The gateway image accepts the cloud-init user data (dnsmasq, nftables,
   systemd) and hot-plugged NICs appear under the requested interface names.
3. Windows: user-data consumption, Administrator password handling and
   secure boot behaviour for the chosen image.
4. The `incus/v6` client (v6.10.1) against the server versions in use.

## Tests

```shell
go test ./builder/incus/
```
