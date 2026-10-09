# MicroCloud builder

The MicroCloud builder (`kind: microcloud`) deploys LaForge builds onto a **MicroCloud
cluster** — Canonical's LXD with MicroOVN and (usually) MicroCeph — through one cluster
endpoint. Any member answers for the whole cluster, and LXD decides which member runs
each instance.

It speaks LXD's API, not Incus's. Standalone Incus servers use the
[Incus builder](incus.md) instead — they are separate builders with separate code.

- [What it creates](#what-it-creates)
- [Preparing the cluster](#preparing-the-cluster)
- [Permissions](#permissions)
- [Connecting it to LaForge](#connecting-it-to-laforge)
- [Builder settings](#builder-settings)
- [The docker base image](#the-docker-base-image)
- [Limits and troubleshooting](#limits-and-troubleshooting)

---

## What it creates

Everything is created in **one LXD project**, chosen when the builder is set up
(`default` unless you pick another — see [Using a project other than
`default`](#using-a-project-other-than-default)), named `lf-…` with a short hash so
builds never collide.

| Content | On MicroCloud |
| --- | --- |
| `network:` | An OVN network (`type: ovn`) routed out through the cluster's uplink network, one per team per network. IPv4 only. |
| `host:` | A system container, or a VM when the builder image is marked VM (Windows always is), with a NIC on the team's OVN network at its `last_octet` address. The agent arrives through cloud-init (Linux) or a NoCloud config-drive ISO imported as a custom storage volume (Windows). |
| `container:` with `image:` | LXD has no OCI runtime, so a nesting system container booted from the [docker base image](#the-docker-base-image) runs the image with Docker. The agent is pushed in and bind-mounted as the application container's entrypoint, so it runs inside the application, never beside it. |
| `container:` with `compose:` | A nesting system container booted from the docker base image, with the agent delivered by cloud-init; the agent starts the project with `docker compose`. |
| `visible_from` / `ports:` | OVN network peers between a team's own networks, and a network ACL per network allowing only the declared sources and ports. |
| `public:` | A NAT proxy device on the instance, listening on the builder's external IP at a per-team port. |

Every instance carries `user.laforge_team`, which is how access windows find a team's
instances.

## Preparing the cluster

- **MicroCloud with MicroOVN**, and an uplink network for OVN networks to route through
  (MicroCloud names it `UPLINK` by convention), with a range for OVN router addresses
  sized for *teams × networks* (each OVN network takes one).
- **A storage pool every member can use.** On MicroCeph it is usually called `remote`.
- **The cluster API reachable** from the LaForge runner, orchestrator and API, on the
  address in the trust token or one you give when connecting.
- **Outbound internet** (or mirrors) for the image server, registries, and the docker
  base image build.

Deployed instances also need to reach LaForge: the agent gateway (`GATEWAY_PUBLIC_ADDR`)
and the API (`API_PUBLIC_URL`). See
[Addressing](../../USAGE.md#addressing-the-urls-explained).

### Using a project other than `default`

A hosting provider may want LaForge kept out of `default`. Create a project for it and
pick that project in the builder wizard's **Placement** step. Everything LaForge creates —
instances, team networks, network ACLs, images (including the docker base image),
config-drive volumes — then goes in that project. Storage pools are cluster-wide, and the
uplink network stays where it is (in `default`), so those choices don't change.

Create the project with its own networks, or team networks and ACLs would still be
created in `default`:

```bash
lxc project create laforge -c features.networks=true -c features.images=true -c features.profiles=true
```

- **`features.networks=true`** — required in practice: team OVN networks and their ACLs
  live in the project. The wizard warns if it's off.
- **`features.images=true`** — optional; the project keeps its own images (and its own
  docker base image) instead of sharing `default`'s. The wizard's image list shows the
  chosen project's images either way.
- **`features.profiles=true`** — optional. LaForge gives every instance its own root disk
  on the builder's storage pool and its own NIC on the team network, so it doesn't rely on
  the project's default profile — **except** the docker base image build, whose temporary
  container uses the default profile's network to reach the internet. Give that profile a
  NIC, or the build fails saying so:

  ```bash
  lxc profile device add default eth0 nic network=<an OVN or bridge network with internet> --project laforge
  ```

If the project is `restricted=true`, also see
[If the project is restricted](#if-the-project-is-restricted).

## Permissions

LaForge talks to LXD only through the REST API, as a **trusted TLS client** whose
certificate it generates itself when you connect the builder. It never uses SSH, the
Unix socket, or root on any member.

### What LaForge does with the API

This is every call the builder makes, from `internal/builder/microcloud`. Everything
except the server, trust, storage-pool and network listing rows is in the builder's
project.

| Area | Calls | Why |
| --- | --- | --- |
| Server | `GET /1.0` | Version and extension checks; confirming trust after enrollment. |
| Trust | `POST /1.0/certificates`, or `POST /1.0/auth/identities/tls` for an identity token | Once, when the builder is connected. Never again. |
| Storage pools, networks | `GET /1.0/storage-pools`, `GET /1.0/networks` | Listing what to pick from when connecting (read-only); connecting fails without it. |
| Storage volumes | `POST`/`DELETE /1.0/storage-pools/{pool}/volumes/custom/…` | Windows config-drive ISOs only. |
| Instances | create (from an image, or as a copy of a snapshot — `source.type: copy`), `GET` snapshots, `GET`, `PUT`/`PATCH` (config and devices), delete; `PUT …/state` (start, stop, restart); `POST …/exec`; `POST …/files` | Deploying and powering instances; access windows (closing one detaches the instance's NIC and stores it in the instance's config; opening restores it); external-access proxy devices; pushing the agent and running Docker in a container's machine; the docker base image build. |
| Images | `POST /1.0/images` (pull, publish), `GET /1.0/images`, `GET`/`DELETE /1.0/images/aliases/…` | Pulling builder images from simplestreams servers; publishing the docker base image. |
| Networks | `POST`/`GET`/`PATCH`/`DELETE /1.0/networks…`, `POST /1.0/networks/{n}/peers` | Team OVN networks, peering, attaching ACLs. Creating an OVN network uses the uplink network. |
| Network ACLs | `POST`/`PUT`/`DELETE /1.0/network-acls…` | `visible_from` and `ports:` enforcement. |
| Operations | `GET /1.0/operations/{id}/wait` | Waiting for the above to finish. |

It does **not** touch cluster or server configuration, cluster members, other projects,
profiles, the uplink network's own configuration, or other identities.

### Option 1: a full trusted client (what the wizard does today)

On any cluster member:

```bash
lxc config trust add --name laforge
```

Paste the printed token into **Admin → Infrastructure → New Builder**. LaForge becomes an
unrestricted trusted client of the cluster: it can do anything on it. This is the
simplest option and the one LaForge is tested with.

### Option 2: fine-grained permissions (LXD 5.21 and newer)

LXD's built-in authorization lets you give LaForge exactly the project it uses. Create a
group, grant it LaForge's project (`laforge` here; `default` if you didn't create one),
and issue an identity token in that group:

```bash
lxc auth group create laforge
lxc auth group permission add laforge project laforge operator
lxc auth group permission add laforge server viewer
lxc auth identity create tls/laforge --group laforge
```

- `project laforge operator` — create, view, edit and delete everything in that project
  (instances, images, networks, network ACLs, storage volumes), but not the project's
  own configuration. Nothing in any other project.
- `server viewer` — read-only access to server-level objects. LaForge needs it to list
  storage pools and networks when the builder is connected: connecting fails if it can't,
  and the wizard's pool and uplink choices come from that list. It also lets LaForge
  *see* (not change) other projects.

Paste the token `lxc auth identity create` prints into the wizard; LaForge recognizes an
identity token and redeems it through the identities API.

> **Not yet verified against a live cluster.** The call list above is exact; whether
> every call is allowed under these entitlements depends on LXD's authorization rules
> (notably creating OVN networks on the uplink, and pulling images from a remote
> server). Before relying on it, connect a builder with this token and run one build that
> has an OVN network, a Linux host, a Windows host, a single-image container, a compose
> container and a `public:` port. Any denied call fails that build with LXD's own
> "not authorized" error.

With a dedicated project, this keeps LaForge away from every other workload on the
cluster: it can change only its own project, and only read the rest.

Templates kept in a separate project (see [Images or snapshots](#images-or-snapshots))
only need to be readable: LaForge copies from them and never changes them. With
`server viewer` that's already the case; without it, grant
`lxc auth group permission add laforge project <templates> viewer`. This, too, is
unverified against a live cluster.

A full trusted client (option 1) still works with a dedicated project — LaForge uses only
the project it's configured with — but nothing stops it reaching others.

### If the project is restricted

If LaForge's project has `restricted=true`, also allow:

- `restricted.containers.nesting=allow` — every LaForge container runs Docker inside a
  nesting container.
- `restricted.networks.uplinks=<your uplink>` — OVN networks route through it.
- `restricted.devices.proxy=allow` — `public:` ports are proxy devices.
- `restricted.virtual-machines.lowlevel=allow` — VMs are created with
  `security.secureboot=false`.

## Connecting it to LaForge

1. On any cluster member, print a token (see [Permissions](#permissions)).
2. In **Admin → Infrastructure → New Builder**, choose **MicroCloud** and paste it.
   LaForge checks the token's server fingerprint, enrolls its own client certificate,
   and reads the cluster's storage pools, networks and images for you to pick from. If
   the cluster is reachable on a different address than the token lists (through NAT,
   say), give that address too.
3. Pick the **project** (see [Using a project other than
   `default`](#using-a-project-other-than-default)), the **storage pool** and the **OVN
   uplink network**, map the **images** and **sizes** content uses, and save. If LaForge's
   identity can't list projects, type the project's name instead. Saving starts the [docker base image](#the-docker-base-image)
   build.

A token is single-use and expires; if enrollment fails, generate a new one.

## Builder settings

| Setting | What it is |
| --- | --- |
| Connection | The cluster endpoint and LaForge's client certificate, from enrollment. |
| Project | The LXD project everything is created in; `default` if left unset. |
| Storage pool | Where instance disks, config-drive volumes and the base image build go. |
| OVN uplink network | The network team OVN networks route through. |
| Images | Content `os:` names (and `compose-host`, optionally) mapped to an image (alias + simplestreams server (Canonical's `https://cloud-images.ubuntu.com/releases` works with LXD), or a fingerprint) or to an instance snapshot to copy (see [Images or snapshots](#images-or-snapshots)); mark VM images (Windows) as VM. |
| Sizes | Content `size:` names mapped to `limits.cpu` / `limits.memory`. |
| External access IP and port range | For `public:` ports: the address proxy devices listen on, and the window of external ports (default 40000–50000, 100 per team). |

A build fails validation before deploying anything if content uses an `os:` with no
image mapped, or has a compose container and the docker base image hasn't been built
(and there's no `compose-host` image).

## Images or snapshots

Each name content uses (`os: win2019`, `os: ubuntu22`) is mapped, in the builder's
**Images** step, to one of two sources:

- **An image** in the cluster's image store, as before.
- **A copy of an instance snapshot** — a stopped *template* instance you keep, with a
  snapshot of it. On storage like Ceph a copy from a snapshot is a cheap clone, so this is
  usually much faster to deploy than unpacking an image, especially for large VMs.

The Images step lists the snapshots it finds in the builder's project next to the images; tick one and
give it the name content uses. A template in a different project can be added by typing
its project, instance and snapshot. Saving checks that every snapshot exists.

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
3. Stop it, then snapshot it, e.g. `lxc snapshot win-tmpl golden`.
4. Keep the template on the builder's storage pool. A copy to a different pool is a full
   copy, not a clone, and loses most of the speed.
5. Mark it **VM** in the builder if the template is a VM (the wizard does this for listed
   snapshots); it has to match.

Re-snapshotting a template under the same name changes what new copies get, but not
existing instances — redeploy them to pick it up. The `compose-host` override can be a
snapshot too.

> Snapshot sources are tested against a simulated LXD API, not yet a live
> MicroCloud cluster. Before an event, deploy one Linux and one Windows host from a snapshot and
> confirm the agent checks in.

## The docker base image

Every container on MicroCloud boots from a Docker-ready image LaForge builds on the
cluster. It is built when the builder is created, and again whenever you choose **Admin →
Infrastructure → the builder's Base image → Rebuild**:

1. Imports the Ubuntu container base from the builder's base image server (Canonical's
   image server, `22.04`, by default) and creates `laforge-docker-base-build`, a nesting
   container, from it.
2. Installs Docker Engine and the compose plugin from Docker's apt repository
   (`download.docker.com`), and proves Docker can run a container under nesting.
3. Cleans it up and publishes it as the image alias `laforge-docker-base`, recorded on
   the builder by fingerprint (the cluster shares one image store).

It needs outbound access to the image server, Ubuntu's archive, `download.docker.com`
and Docker Hub. **Rebuild once after upgrading LaForge**: builds made before this
release installed Ubuntu's `docker.io` package without the compose plugin. Single-image
containers keep working on an older image; compose containers would install the plugin
at deploy time instead.

An image named `compose-host` in the builder's images overrides the base image for
compose containers only: they are then deployed from it like a host.

## Limits and troubleshooting

- **Changing an existing builder's project** doesn't move anything: builds already
  deployed stay in the old project and can't be managed from the new one. Tear them down
  first, then change the project, and rebuild the base image if the new project keeps its
  own images.
- **"the build container has no network device"** — the base image build ran in a
  project whose default profile has no NIC; add one (above).
- **"Network is not in pending state"** on a retried network deploy is a known LXD
  cluster race; the builder checks for and adopts the existing network.
- **"no image configured for os …"** — map that `os:` name in the builder's images.
- **"no docker base image yet"** — the base image build hasn't finished, or failed: open
  the builder's Base image console for its log.
- **Agents never check in** — the instance can't reach `GATEWAY_PUBLIC_ADDR` or
  `API_PUBLIC_URL`. Test from inside an instance; cluster members usually can't reach
  instance addresses on OVN networks directly.
