# Incus builder

The Incus builder (`kind: incus`) deploys LaForge builds onto one or more **independent
Incus servers**. It is a pool: each host has its own connection, storage pool and
network mode. The hosts share nothing, and every team is placed on exactly one of them
(team *N* goes to host `(N-1) % hosts`), so a team's networks, hosts and containers
always live together.

Use it for standalone Incus servers. A clustered MicroCloud/LXD deployment uses the
[MicroCloud builder](microcloud.md) instead — they are separate builders.

- [What it creates](#what-it-creates)
- [Preparing a host](#preparing-a-host)
- [Permissions](#permissions)
- [Connecting it to LaForge](#connecting-it-to-laforge)
- [Builder settings](#builder-settings)
- [The docker base image](#the-docker-base-image)
- [Limits and troubleshooting](#limits-and-troubleshooting)

---

## What it creates

Everything is created in the server's **`default` project**, named `lf-…` with a short
hash so builds never collide.

| Content | On Incus |
| --- | --- |
| `network:` | A native managed bridge with host routing, DHCP and IPv4 NAT, one per deployment subnet. Legacy OVN mode creates logical networks through an OVN uplink. IPv4 only. |
| `host:` | A system container, or a VM when the builder image is marked VM (Windows always is), with a NIC on the team's network at its `last_octet` address. The agent arrives through cloud-init (Linux) or a NoCloud config-drive ISO imported as a custom storage volume (Windows). |
| `container:` with `image:` | A native OCI application container pulled straight from the registry, with the LaForge agent pushed in as its entrypoint. |
| `container:` with `compose:` | A nesting system container booted from the builder's [docker base image](#the-docker-base-image), with the agent delivered by cloud-init. |
| `visible_from` / `ports:` | Per-instance NIC ACLs allow declared ports from the same subnet and `visible_from` subnets. OVN mode also creates network peers. |
| `public:` | A NAT proxy device on each running instance, listening on the assigned host external IP at a per-team port, plus a NIC ACL allowing its public ports. Use native bridge networking for this. |

Every instance carries `user.laforge_team`, which is how access windows find a team's
instances.

## Preparing a host

Each Incus server in the pool needs:

- **Incus 6.3 or newer** (native OCI containers). Compose containers and the docker base
  image are tested on 7.5.
- **The HTTPS API listening** — `incus config set core.https_address=:8443` — and
  reachable from the LaForge runner, orchestrator and API.
- **Native bridge NIC ACL support and the nftables firewall** for bridge mode.
  LaForge detects `network_bridge_acl_devices` in the server's API extensions and
  `environment.firewall=nftables`; this is available on the tested Incus 7.5.1 hosts.
  No OVN services or uplink network are needed. Incus creates the subnet bridges,
  and traffic uses the host's existing default route.
- For **legacy OVN mode only**, OVN services and a managed physical/bridge uplink
  (usually `UPLINK`) with enough `ipv4.ovn.ranges` addresses for all logical networks.
- **A storage pool** for instance disks. Keep it off the root filesystem: a loop-file
  pool on `/` fills it.
- **Outbound internet** (or mirrors) for the image server, registries, and — if any
  content uses Docker Compose — the docker base image build.
- **Nothing else running Docker on the host itself.** Docker on an Incus host rewrites
  the firewall's forwarding policy and breaks Incus bridges.

Deployed instances also need to reach LaForge: the agent gateway (`GATEWAY_PUBLIC_ADDR`)
and the API (`API_PUBLIC_URL`, for the agent download). See
[Addressing](../../USAGE.md#addressing-the-urls-explained).

## Permissions

LaForge talks to Incus only through the REST API, as a **trusted TLS client** whose
certificate it generates itself when you connect the builder. It never uses SSH, the
Unix socket, or root on the host.

### What LaForge does with the API

This is every call the builder makes, from `internal/builder/incus`. Everything except
the first two rows is in the `default` project.

| Area | Calls | Why |
| --- | --- | --- |
| Server | `GET /1.0` | Version and extension checks; confirming trust after enrollment. |
| Trust | `POST /1.0/certificates` (trust token) | Once, when the builder is connected. Never again. |
| Storage pools, networks | `GET /1.0/storage-pools`, `GET /1.0/networks` | Listing what to pick from when connecting (read-only); connecting fails without it. |
| Storage volumes | `POST`/`DELETE /1.0/storage-pools/{pool}/volumes/custom/…` | Windows config-drive ISOs only. |
| Instances | create (from an image, or as a copy of a snapshot — `source.type: copy`), `GET` snapshots, `GET`, `PUT`/`PATCH` (config and devices), delete; `PUT …/state` (start, stop, restart); `POST …/exec`; `POST …/files` | Deploying and powering instances; access windows (closing one detaches the instance's NIC and stores it in the instance's config; opening restores it); external-access proxy devices; pushing the agent into OCI containers; the docker base image build. |
| Images | `POST /1.0/images` (pull, publish), `GET /1.0/images`, `GET`/`DELETE /1.0/images/aliases/…` | Pulling builder images from simplestreams/OCI registries; publishing the docker base image. |
| Networks | `POST`/`GET`/`PATCH`/`DELETE /1.0/networks…`, `POST /1.0/networks/{n}/peers` | Team bridges or OVN networks and attaching ACLs. Peers and the uplink are used only in OVN mode. |
| Network ACLs | `POST`/`PUT`/`DELETE /1.0/network-acls…` | `visible_from` and `ports:` enforcement. |
| Operations | `GET /1.0/operations?recursion=1`, `GET /1.0/operations/{id}/wait` | Finding unfinished instance creation after a retry/restart, and waiting for operations to finish. |

It does **not** touch server configuration, other projects, profiles, cluster
membership, other certificates, or the uplink network's own configuration.

### Option 1: a full trusted client (what the wizard does today)

On the host:

```bash
incus config trust add laforge
```

Paste the printed token into **Admin → Infrastructure → New Builder**. LaForge becomes an
unrestricted trusted client: it can do anything on that server. This is the simplest
option and the one LaForge is tested with. Use it on servers dedicated to LaForge.

### Option 2: a restricted client (least privilege)

Incus can restrict a client certificate to named projects. LaForge works entirely in
`default`, so:

```bash
incus config trust add laforge --restricted --projects default
```

Restricted to `default`, LaForge can still manage every instance, network, ACL, image
and volume in that project — which is everything it needs — but can no longer change
server configuration, cluster membership, storage pools, other projects, or the trust
store.

> **Not yet verified against a live server.** The call list above is exact; whether
> every call is allowed for a restricted client depends on Incus's authorization rules
> (notably listing storage pools — connecting the builder fails if LaForge can't, since
> the wizard's pool and uplink choices come from that list — and creating OVN networks
> on the uplink). Before relying
> on it, connect a builder with a restricted token and run one build that has a deployment
> network, a Linux host, a Windows host, a compose container and a `public:` port. Any
> denied call fails that build with Incus's own "not authorized" error.

Because LaForge always uses `default`, restricting it to a dedicated project isn't
possible today; if the server also hosts other workloads in `default`, LaForge can see
and change them.

### If the project is itself restricted

If you set `restricted=true` on the `default` project (unusual), also allow:

- `restricted.containers.nesting=allow` — compose containers and the docker base image
  run Docker inside a nesting container.
- `restricted.networks.uplinks=<your uplink>` — only for OVN mode. Native managed
  bridge creation requires permission to create host networks; a restricted project
  may refuse it.
- `restricted.devices.proxy=allow` — `public:` ports are proxy devices.
- `restricted.virtual-machines.lowlevel=allow` — VMs are created with
  `security.secureboot=false`.

## Connecting it to LaForge

1. On each host, print a trust token (see [Permissions](#permissions)).
2. In **Admin → Infrastructure → New Builder**, choose **Incus**, add one host per token,
   and paste each one. LaForge checks the token's server fingerprint, enrolls its own
   client certificate, and reads the host's storage pools, networks and images for you
   to pick from.
3. Pick each host's **storage pool** and **Deployment networking**. Choose **Native
   bridges** on supported hosts; no uplink value is needed. Existing bridges and their
   subnets are shown to help avoid overlaps. OVN mode shows managed physical/bridge
   uplink suggestions, with the network type and address beside each name.
4. Enable **One team per configured host**, map the **images** and **sizes**, and save.
   A build with more teams than distinct configured hosts fails before deployment.
   Fewer teams than hosts is allowed; hosts are assigned in their configured order.

A token is single-use and expires; if enrollment fails, generate a new one.

## Builder settings

| Setting | Per | What it is |
| --- | --- | --- |
| Hosts | builder | One entry per Incus server: its connection (from enrollment), storage pool, `network_type` (`bridge` or `ovn`), optional OVN uplink, and operation timeout. |
| One team per configured host | builder | `incus_one_team_per_host`: at most one team per host **within a build**. Rejects insufficient capacity and duplicate endpoint/server identities; does not reserve hosts across separate builds. |
| Images | builder | Content `os:` names (and `compose-host`, optionally) mapped to an image (alias + simplestreams server, or a fingerprint already on every host) or to an instance snapshot to copy (see [Images or snapshots](#images-or-snapshots)); mark VM images (Windows) as VM. |
| Sizes | builder | Content `size:` names mapped to `limits.cpu` / `limits.memory`. |
| External access IP and port range | builder | For `public:` ports: the address proxy devices listen on, and the window of external ports (default 40000–50000, 100 per team). |

A build fails validation before deploying anything if content uses an `os:` with no
image mapped, or has a compose container and the builder has neither a built docker base
image nor a `compose-host` image.

## Bridge placement and migration

Team content repeats the same subnets. Native bridges share the host routing table,
so each team needs a different server even if the one-team checkbox is unchecked.
LaForge rejects a layout that would reuse a bridge host, and checks each requested
subnet against existing managed bridges before creating it. Another active build
using the same CIDRs must be torn down or assigned different servers. Keep the host
list and its order unchanged while a build is deployed.

Saved configurations with an OVN uplink and no `network_type` keep using OVN.
To migrate, tear down the old deployment, select native bridges for each host, and
rebuild. Changing the selection does not convert existing logical networks. Bridge
mode does not attach team VMs to `incusbr0`; it creates a bridge for each content
subnet. The default profile's `incusbr0` can still provide internet to image-build
containers, and its subnet must not overlap deployment subnets.

## Public port mappings

Set **External IPs** to addresses owned by/reachable through the selected Incus hosts.
For one team per host, list one address per host in the **same order** as the host
list. For two hosts, team 1 uses the first address on host 1 and team 2 uses the
second address on host 2. The field does not route an address from one host to another.
Allow the configured port range through any upstream firewall.

Forwarding waits for the public target set to finish deploying and for Incus to
report each target `Running`. Waiting for startup does not consume failure retries.
New or replaced public VMs trigger reconciliation. Public-port NIC rules are merged
with private access rules, so the default-deny firewall permits forwarded clients.

Recorded mappings are available at `GET /builds/{build_id}/external-access`:
`team_number`, `object_name`, `protocol`, `internal_port`, `external_port`, and
`public_address`. `GET /builds/{build_id}/external-access/export` also exports login
credentials; both routes require repository read access.

The bridge design follows Incus's [NAT proxy requirements](https://linuxcontainers.org/incus/docs/main/reference/devices_proxy/#nat-mode)
and [NIC ACL behavior](https://linuxcontainers.org/incus/docs/main/howto/network_acls/).

## Images or snapshots

Each name content uses (`os: win2019`, `os: ubuntu22`) is mapped, in the builder's
**Images** step, to one of two sources:

- **An image** in the host's image store, as before.
- **A copy of an instance snapshot** — a stopped *template* instance you keep, with a
  snapshot of it. On storage like Ceph a copy from a snapshot is a cheap clone, so this is
  usually much faster to deploy than unpacking an image, especially for large VMs.

The Images step lists the snapshots it finds in `default` on each host next to the images; tick one and
give it the name content uses. A template in a different project can be added by typing
its project, instance and snapshot. Saving checks that every snapshot exists on every host in the pool (a team can land on any host).

**What LaForge does with a snapshot:** it creates each instance as a copy of
`<instance>/<snapshot>` (without the template's other snapshots), on the builder's storage
pool, then — before the copy first boots — **replaces its devices with exactly LaForge's
own** (the team NIC at its address, the root disk, the Windows config drive) and its
profiles with just `default`. A template's other devices — a NIC on a management network,
an attached data volume, a GPU — are never carried into team instances, so one team's
copy can't share a network or disk with another's.

**Preparing a template:**

1. Build the instance the way every copy should start, and leave cloud-init installed
   (cloudbase-init on Windows): the agent is delivered through it on each copy's first
   boot.
2. Reset first-boot state so each copy runs it again: on Linux `cloud-init clean --logs`;
   on Windows, generalize it as you would for any cloned image (sysprep with
   cloudbase-init's unattend). Each copy gets a new instance identity, which is what makes
   cloud-init treat it as a new machine.
3. Stop it, then snapshot it, e.g. `incus snapshot create win-tmpl golden`.
4. Keep the template on the builder's storage pool. A copy to a different pool is a full
   copy, not a clone, and loses most of the speed.
5. Mark it **VM** in the builder if the template is a VM (the wizard does this for listed
   snapshots); it has to match.

Re-snapshotting a template under the same name changes what new copies get, but not
existing instances — redeploy them to pick it up. The `compose-host` override can be a
snapshot too.

> Snapshot sources are tested against a simulated Incus API, not yet a live
> Incus server. Before an event, deploy one Linux and one Windows host from a snapshot and
> confirm the agent checks in.

## The docker base image

Containers that run a Docker Compose project boot from a Docker-ready image LaForge
builds itself. Single-image containers don't need it, so it is built on demand: **Admin →
Infrastructure → the builder's Base image → Rebuild**.

The job runs on **every host in the pool** (they share no image store):

1. Creates `laforge-docker-base-build`, a nesting container from
   `images:ubuntu/24.04/cloud` on the host's storage pool.
2. Installs Docker Engine and the compose plugin from Docker's apt repository
   (`download.docker.com`), and proves Docker can run a container under nesting.
3. Cleans it up and publishes it as the image alias `laforge-docker-base`.

It takes a few minutes per host and needs outbound access to the image server, Ubuntu's
archive, `download.docker.com` and Docker Hub. Rebuild after changing hosts, or to pick
up a newer Docker. The published image is about 300 MB in the host's image store, which
lives on the root filesystem.

An image named `compose-host` in the builder's images overrides the base image: compose
containers are then deployed from it like a host.

## Limits and troubleshooting

- **On a shared cluster** (other tenants' instances, networks, pools and projects),
  Discovery avoids asking the server for every object in full. Native bridge creation
  does read managed network configuration to check for subnet overlaps. LXD and Incus work out
  everything that uses each one for that, which can take longer than a request is
  allowed. It lists names, which is cheap. It reads details only for what it needs: a
  bounded number of networks (likely uplinks first), each storage pool, the chosen
  project, the instances that have snapshots, and its own `lf-…` instances. It keeps to
  a few requests at a time. In a project of LaForge's own, its instances are read with
  one listing. If something still can't be read while connecting, the wizard shows a
  warning and takes a typed name instead.
- **Everything is in the `default` project.** See [Permissions](#permissions).
- **OVN network creation can stall** while the host is busy creating or deleting other
  networks; it is retried by the runner.
- **Slow instance creation:** a VM may show `STOPPED` while its `Creating instance`
  operation is still running (for example, while converting a large image). LaForge
  continues polling that same operation across operation-timeout windows until it
  finishes or the runner's context is cancelled. An expired polling window does not
  trigger another create or replace the instance's config drive. After a runner
  restart or a deployment retry, LaForge checks the project's active operations
  before adopting an instance, replacing its config drive, or deleting it. Real
  creation failures and connection errors still surface as errors. Use
  `incus operation list` and `incus operation show ID` to check progress.
- **LVM thin clones fall back to full image conversion when the requested VM
  disk is smaller than the cached image volume.** A pool with `volume.size=100GiB`
  creates 100 GiB image volumes; a host with `disk_gb: 60` requests only 60 GB.
  Incus cannot shrink the cloned block volume and unpacks the image again. Check
  `incus storage show POOL` and `lvs -o lv_name,origin,lv_size`: clones have an
  image origin, while independent conversions do not. Keep the image-cache size
  no larger than the smallest requested disk that uses it, or request larger
  disks. Changing the pool's default can regenerate image caches on subsequent
  creates, so plan that change outside an active build.
- **"no image configured for os …"** — map that `os:` name in the builder's images.
- **"no docker base image yet"** — build it (above), or add a `compose-host` image.
- **Agents never check in** — the instance can't reach `GATEWAY_PUBLIC_ADDR` or
  `API_PUBLIC_URL`. Native bridges NAT through the host's default route. OVN
  networks NAT out through the uplink; the Incus host itself usually
  can't reach instance addresses on OVN networks, so test from inside an instance.
