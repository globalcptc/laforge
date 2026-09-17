# Canonical MicroCloud builder

This builder deploys LaForge teams as isolated LXD projects on a
Canonical MicroCloud cluster.

Each team is restricted to one cluster member. Each LaForge network is
an OVN network inside the team project; sibling networks are mutually
peered so traffic routes directly inside OVN without SNAT. Projects are
never peered with one another.

## MicroCloud prerequisites

- MicroCloud 2.x LTS with LXD 5.21 LTS.
- A healthy LXD cluster with every member online.
- A MicroCeph-backed storage pool (normally `remote`).
- A MicroOVN physical uplink network (normally `UPLINK`).
- Working internet egress and public DNS through the uplink.
- Enough addresses in `UPLINK`'s `ipv4.ovn.ranges` for one address per
  LaForge provisioned network.
- An `ipv4.routes` CIDR on `UPLINK` for automatically allocated team
  ingress forwards.

For ten teams with two networks each, reserve at least 20 router
addresses and 10 forward addresses:

```shell
lxc network set UPLINK ipv4.ovn.ranges=192.0.2.100-192.0.2.149
lxc network set UPLINK ipv4.routes=192.0.2.192/27
```

Use ranges appropriate for the actual uplink subnet. They must not
overlap addresses assigned elsewhere.

## Authentication

Enable the LXD HTTPS listener and trust a dedicated LaForge client
certificate:

```shell
lxc config set core.https_address :8443
lxc config trust add client.crt --name laforge
```

Copy the client certificate, private key, and LXD server certificate to
the LaForge server and reference them from the builder config.

## Images

Import each VM image into the default LXD project and give it the alias
used by the LaForge host's `os` field:

```shell
lxc image copy ubuntu:22.04 local: --alias ubuntu22 --vm
```

Windows deployment is currently rejected. It will be enabled only after
the image contract for the LXD VM agent, Cloudbase-Init, administrator
password handling, and agent bootstrap has been validated.

## LaForge configuration

Copy `configs/microcloud.json.example`, fill in the cluster endpoint,
certificates, image aliases, and instance sizes, then register it in
`conf.json`:

```json
{
  "builders": {
    "microcloud-prod": {
      "builder": "microcloud",
      "config": "configs/microcloud.json"
    }
  }
}
```

Use the friendly name in the environment:

```hcl
environment "/envs/example" {
  builder = "microcloud-prod"
}
```

Mark exactly one host per team as the externally reachable VDI proxy:

```hcl
vars = {
  public_ingress = "true"
}
```

Its `exposed_tcp_ports` and `exposed_udp_ports` become the LXD network
forward's allowed ports. The allocated uplink address is saved as
`Team.Vars["ingress_address"]` and
`ProvisionedHost.Vars["PublicIP"]`.
