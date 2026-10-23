# LaForge Content Configuration

This is the complete reference for the YAML that describes a competition environment.
It's written for the people who author content — developers and volunteers — not just
for engineers. Every field is explained in plain language, with examples that build up
from the simplest possible environment to a complete multi-network game.

If you use the **VS Code extension** (`editors/vscode/`), you get all of this as
autocomplete and hover documentation while you type, plus live validation.

- [How content is organized](#how-content-is-organized)
- [Your first environment (the simplest thing that works)](#your-first-environment)
- [Environment](#environment) · [Network](#network) · [Host](#host) · [Container](#container) · [Script](#script) · [People (CSV)](#people-csv)
- [Steps: the action kinds](#steps-the-action-kinds)
- [Waiting on another host: `wait_for` and `milestone`](#waiting-on-another-host-wait_for-and-milestone)
- [Validators](#validators)
- [Schedules: natural-language timing](#schedules)
- [Templating: what's in scope in a script](#templating)
- [`extends`: reusing a base object](#extends)
- [Checking your content](#checking-your-content)

---

## How content is organized

A content repository is a Git repo with one directory per object type:

```
your-content-repo/
  game.yaml            # the environment (one per event); can be any name, lives at the root
  networks/
    corp.yaml
    dmz.yaml
  hosts/
    webserver.yaml
    domain-controller.yaml
  containers/
    scoreboard.yaml
    flakyframes.yaml
    flakyframes/       # a Docker Compose project a container runs
      compose.yaml
      .env
  scripts/
    install-nginx.yaml
    install-nginx.sh   # a script's source sits next to its .yaml definition
  people/
    employees.csv
  .laforgeignore       # optional: paths that aren't LaForge content
```

Each YAML file starts with a **type header** — the first key names what it is:

```yaml
host:            # this file is a host
  name: webserver
  ...
```

Files can hold several `---`-separated objects, but one object per file is the norm.

**Every `.yaml`/`.yml` file in the repo is read as LaForge content.** If the repo also
holds YAML that isn't — a Compose project, another tool's configuration — list those
paths in a `.laforgeignore` file at the root. It uses the familiar subset of `.gitignore`
syntax, and `laforge check`, the server, and the editor all honor it:

```
# .laforgeignore -- one pattern per line; comments get their own line

# a directory (trailing slash)
containers/flakyframes/
# anchored to the repo root
/vendor
# * ? [..] match within one path segment
notes/*.yaml
# ** spans directories
**/generated-*.yml
# a bare name matches at any depth
scratch.yaml
```

Negation (`!pattern`) is not supported.

**The pieces are reusable; the environment connects them.** A host or a script doesn't
know which environment or network it belongs to. The **environment's `networks:`
topology** is the single place where hosts and containers are placed onto networks,
copied per team, and given addresses. This lets you reuse the same `webserver.yaml`
across many games.

---

## Your first environment

The smallest useful environment: two teams, one network, one host running one command.

`game.yaml`:

```yaml
environment:
  name: first-game
  teams: 2
  networks:
    corp:
      webserver:
        - as: web01
          last_octet: 10
```

`networks/corp.yaml`:

```yaml
network:
  name: corp
  cidr: 10.0.1.0/24
```

`hosts/webserver.yaml`:

```yaml
host:
  name: webserver
  os: ubuntu22
  size: small
  disk: 20
  ports:
    tcp: ["80"]
  steps:
    - run: apt-get update && apt-get install -y nginx
```

That's a complete, valid environment. Each of the 2 teams gets an identical `corp`
network (`10.0.1.0/24`) with a host `web01` at `10.0.1.10` that installs nginx and
exposes port 80.

Everything below is the detail for going from here to a full game.

---

## Environment

The environment is the file a build starts from — one per event. It declares the teams,
the schedule, and the topology. It never mentions a builder or an IP pool (those are a
deployment-time choice, not content).

```yaml
environment:
  name: cptc-quals
  description: Qualifier environment for Allports Inc.
  teams: 10
  root_password: P@ssword123
  start: 2026-10-01T09:00:00Z
  stop: 2026-10-03T17:00:00Z
  vars:
    company: Allports
  tags:
    season: fall
  access:
    - open: 2026-10-01T09:00:00Z
      close: 2026-10-01T18:00:00Z
    - open: 2026-10-02T09:00:00Z
      close: 2026-10-02T18:00:00Z
  dns:
    type: bind
    root_domain: allports.local
  networks:
    corp:
      domain-controller:
        - as: dc01
          last_octet: 5
      workstation:
        - as: ws01
          last_octet: 20
        - as: ws02
          last_octet: 21
    dmz:
      webserver:
        - as: web01
          last_octet: 10
```

(`web01` exposes a port externally — but that's declared with `public:` on the
webserver **host definition**, not here in the topology; see [Host](#host).)

| Field | What it is |
| --- | --- |
| `name` | Unique name for this environment. |
| `schema` | Schema/format version of this environment file. |
| `description` | Human-readable description of the game. |
| `teams` | Number of teams. Every team gets an identical copy of the topology — nothing is ever templated per team. |
| `root_password` | A password stored in the clear **on purpose** — content credentials exist to be found during the competition, not to be real secrets. |
| `agent-debug` | (Optional, default `false`) Turn on the agent's local debug log — a file next to the agent binary on every host/container in this environment. Off by default, the agent writes **nothing** locally (no stdout/stderr) and only reports to the LaForge servers. The flag is baked into each agent binary at deploy time, so it can't be switched on by editing a box's launcher. Leave off in a real competition (the boxes are in a hostile network); turn on only to debug the agent itself. |
| `container_logs` | (Optional) Forward every **container's** console output to an external collector (Splunk, …) using each builder's native logging — see [Container logs](#container-log-forwarding). Omit to disable. Container-only; hosts are VMs with their own logging. |
| `start` / `stop` | When the event starts and ends. Used by schedules ("45 minutes after competition start"). |
| `dns` | DNS settings (see [DNS](#dns)). A records for every host/container are generated automatically; this adds anything else. LaForge doesn't run DNS itself — the records are handed to a host (a domain controller or Bind server) to serve. |
| `access` | The planned access schedule — e.g. closing overnight on a multi-day event. Per-team overrides during a live event happen from the UI/API, not here. Each entry has an `open:` and `close:` time. |
| `vars` | Key/value data cascaded down to every network, host, and container. Available in scripts as `{{ vars.company }}`. |
| `tags` | Labels for searching, grouping, and ad-hoc targeting. |
| `findings` | Findings that belong to the environment as a whole (see [Findings](#findings)). |
| `networks` | **The topology.** For each network, which hosts/containers are placed on it, how many copies, and their addresses. The only place hosts, containers, and networks are connected. |
| `extends` | Inherit from another environment (see [`extends`](#extends)). |

### The topology (`networks:`)

The topology maps `network name → object name → list of copies`. Each copy is one
running instance of that host/container:

| Copy field | What it is |
| --- | --- |
| `as` | The hostname this copy gets. Also what `depends_on` and generated DNS records resolve it by. |
| `last_octet` | The last octet of this copy's address on the network's CIDR (0–255). With `cidr: 10.0.1.0/24` and `last_octet: 10`, the address is `10.0.1.10`. |

Placing the same object twice (two `as` entries) gives a team two copies of it.

### DNS

```yaml
dns:
  type: bind
  root_domain: allports.local
  dns_servers: ["10.0.1.5"]
  ntp_servers: ["pool.ntp.org"]
  records:
    - name: mail
      type: A
      target: 10.0.1.25
```

| Field | What it is |
| --- | --- |
| `type` | DNS backend, e.g. `bind`. |
| `root_domain` | Root domain every generated hostname is placed under. |
| `dns_servers` | Resolvers handed to hosts. |
| `ntp_servers` | NTP servers handed to hosts. |
| `records` | Custom records on top of the automatic A record per host. Each has `name`, `type`, `target`, and optional `priority` (for MX). |

The full resolved record set is available to a script as `{{ .dns }}`, so a script on
the DC/Bind host can render a zone file from it.

---

## Network

A network is one CIDR, shared **identically** by every team — never templated per team.
Isolation between teams comes from a per-team routing domain, not from the addresses.

```yaml
network:
  name: dmz
  cidr: 10.0.2.0/24
  visible_from: [corp]
  vars:
    zone: dmz
```

| Field | What it is |
| --- | --- |
| `name` | Unique name, referenced from the topology and from other objects' `depends_on`. |
| `cidr` | The CIDR (e.g. `10.0.1.0/24`), identical for every team. Combined with each copy's `last_octet` to produce that copy's address. |
| `visible_from` | The other networks (by name) allowed to reach this one. **Default-deny:** with no `visible_from`, no other network can reach this one. A `client` network with `visible_from: [vdi]` is reachable from the VDI network and nothing else. Each entry must be a defined network. |
| `vars` | Key/value data cascaded down to every host and container on this network. |
| `tags` | Labels for searching, grouping, and ad-hoc targeting. |
| `findings` | Findings that belong to this network specifically. |
| `extends` | Inherit from another network (see [`extends`](#extends)). |

`visible_from` controls **which networks** can reach this one; a host's `ports:`
controls **which ports** on that host are reachable. Together they are the firewall.

---

## Host

A host is a full VM. `os` and `size` are **abstract names** — a builder config maps them
to a concrete image and CPU/memory for its platform, so the same host runs anywhere.

Simple:

```yaml
host:
  name: webserver
  os: ubuntu22
  size: small
  disk: 20
  ports:
    tcp: ["80", "443"]
```

Complete (with dependencies, setup steps, a recurring reboot, and a finding):

```yaml
host:
  name: workstation
  os: windows-server-2022
  size: medium
  disk: 60
  ports:
    tcp: ["3389"]
  public:
    tcp: ["3389"]
  depends_on: [domain-controller]
  vars:
    role: workstation
  tags:
    team_facing: "true"
  people:
    - file: employees
  steps:
    - script: join-domain
    - create_user:
        name: helpdesk
        password: "{{ vars.helpdesk_password }}"
        groups: ["Administrators"]
      validate:
        - user_exists: helpdesk
  schedule:
    - when: every day at 3:00am
      reboot: {}
  findings:
    - severity: 4
      difficulty: 2
      description: Weak local admin password on the helpdesk account.
```

| Field | What it is |
| --- | --- |
| `name` | Unique name, referenced from the topology and `depends_on`. **Not** a hostname — the topology's `as` field sets the actual hostname. |
| `os` | Abstract OS name (`ubuntu22`, `windows-server-2022`, `kali`, …). Mapped to a concrete image per builder. |
| `size` | Abstract size name (`small`, `medium`, `large`). Mapped to concrete CPU/memory per builder. |
| `disk` | Disk size in GB. |
| `ports` | TCP/UDP ports this host listens on — the ingress firewall allowlist (see [Ports](#ports)). |
| `public` | (Optional) The subset of `ports` to expose **outside** the competition network (e.g. RDP for direct access). A property of the host, so it lives here — not on the topology placement. Every port listed must also appear in `ports`. Omit, or set to `false`, to keep the host private. The builder realizes it (a public IP, or a port-NAT on a shared external IP) and the assigned address:port shows in the build's external access view (`laforge access`). |
| `depends_on` | Other hosts, containers, or networks that must be **finished** before **any** of this host's steps run. The box still deploys right away, and its steps show as *Dependency Blocked* until every dependency has completed all of its own steps. Names the **object**, not one copy: every copy of that object in the same team is waited on. This host's own network is always waited on implicitly. To wait on just part of another host, or only from part-way through this host's steps, use a step's [`wait_for`](#waiting-on-another-host-wait_for-and-milestone) instead. |
| `steps` | The ordered setup steps, run once after provisioning (see [Steps](#steps-the-action-kinds)). |
| `schedule` | Recurring or anchor-relative triggers, independent of `steps:` order — each fires on its own clock (see [Schedules](#schedules)). |
| `vars` | Key/value data for this host's scripts, on top of what cascades from its environment and network. |
| `tags` | Labels for searching, grouping, and ad-hoc targeting. |
| `findings` | Findings that belong to this host specifically. |
| `people` | People (from a `people/*.csv`) this host concerns — e.g. accounts to create (see [People](#people-csv)). |
| `extends` | Inherit from another host (see [`extends`](#extends)). |

---

## Container

The same idea as a host, deployed as a container instead of a VM. Every builder supports
containers — natively where the platform has a container primitive, as a nested Docker
runtime where it doesn't. A container runs either a single `image`, or a whole Docker
Compose project (`compose`, below).

```yaml
container:
  name: scoreboard
  image: registry.internal/team/scoreboard:latest
  size: small
  env:
    DB_HOST: db01
  command: ["--port", "8080"]
  ports:
    tcp: ["8080"]
```

| Field | What it is |
| --- | --- |
| `name` | Unique name, referenced from the topology and `depends_on`. Not a hostname — the topology's `as` sets that. |
| `image` | The OCI image ref, e.g. `nginx:alpine` or `registry.internal/app:1.2`. Pulled from Docker Hub, or a configured private registry when the ref names one. |
| `compose` | Instead of `image`: the path to a Docker Compose file, relative to this file. Runs the whole project (see [Compose projects](#compose-projects)). |
| `disk` | With `compose` only: root disk in GB for the machine the project runs on (default 40). |
| `size` | Abstract size name, mapped to concrete CPU/memory per builder. |
| `env` | Environment variables passed to the container (`docker -e`). Not with `compose`. |
| `command` | Overrides the image's default command arguments. Not with `compose`. |
| `ports` | TCP/UDP ports this container listens on. |
| `public` | (Optional) The subset of `ports` to expose outside the competition network — same meaning and rules as a host's (see [Host](#host)). |
| `depends_on` | Same as a host's. |
| `steps` / `schedule` | Same as a host's — a container runs the LaForge agent as its entrypoint, so it configures and reports exactly like a host. Steps and validators run **inside** the container, so they describe the application (`process_running: nginx`, `port_listening: 80`), never the container runtime underneath it (`process_running: dockerd` is wrong, and fails on every builder). |
| `vars` / `tags` / `findings` / `people` / `extends` | Same as a host's. |

### Compose projects

A container can run a Docker Compose project instead of a single image. Point `compose:`
at the compose file, relative to the container's own file — the same way a script points
at its `source:`:

```yaml
# containers/flakyframes.yaml
container:
  name: flakyframes
  compose: flakyframes/compose.yaml
  size: medium
  ports: { tcp: ["80", "443"] }    # the ports the compose file publishes
  steps:
    - run: curl -fsSk https://localhost/healthz
      validate:
        - port_listening: { port: 443 }
```

```
containers/
  flakyframes.yaml
  flakyframes/
    compose.yaml
    .env
    nginx/
      nginx.conf
      certs/
```

```
# .laforgeignore
containers/flakyframes/
```

The compose file's **directory** is the project. LaForge gives the container one machine
with one address, copies the whole directory to it, logs in to any registry it has a
stored credential for (Admin → Infrastructure), pulls the images, and starts the project
with real `docker compose up --wait` — so it counts as started only once every service is
running and its healthcheck, if it has one, passes. The container's own `steps:` run
after that, on that machine.

What to know:

- **One address.** The project is reached on the container's address, through the ports
  the compose file publishes. List the same ports in `ports:`; they are its firewall.
  Networks inside the compose file stay private to the project.
- **Images are pulled, never built.** A service can keep `build:` for local development,
  but must also name an `image:` that exists in a registry. Pin a version tag.
- **The compose file is templated; supporting files opt in with `.tmpl`.** The compose
  file gets LaForge's [template pass](#templating) at deploy — the same `{{ vars.x }}`,
  `{{ .Team }}`, `{{ .People }}` context a script or `run:` gets — so it can vary per team
  or object. Quote template values so the file stays valid YAML
  (`image: "{{ vars.app_image }}"`). A supporting file is templated by naming it with a
  `.tmpl` suffix: `nginx.conf.tmpl` is rendered and shipped as `nginx.conf` (the suffix is
  stripped) — so mount the stripped name (`./nginx.conf`). Files without `.tmpl` travel
  verbatim. Compose's own `${VAR}`/`$host` interpolation is left untouched either way and
  still resolves at `docker compose` runtime. The directory is limited to 1 MiB compressed
  — configuration, not application code. Bind mounts must be relative and inside it
  (`./nginx/nginx.conf`).
- **Add the directory to `.laforgeignore`**, so its YAML isn't read as LaForge content.
  `laforge check` tells you when you've forgotten.
- **Volumes last as long as the container.** They survive a restart and are gone on a
  rebuild. Editing any file in the directory rebuilds the container.
- **Steps see the machine, not a service.** `docker compose` is available to them;
  `process_running: nginx` is not true there, `port_listening: 443` is.
- **Builders.** The Incus and MicroCloud builders run compose containers, each on a
  machine booted from the builder's own Docker-ready base image, which an admin builds
  once (see [USAGE.md](USAGE.md#compose-containers-builder-setup)). Other builders reject
  a compose container.

`laforge check` loads every project and reports a missing compose file, a service
without an `image:`, or a bind mount that won't be shipped.

---

## Script

A script is a template rendered **per host, per team** before it runs, with the
environment's vars, people data, and host/network facts all in scope. The `.yaml`
defines it; the actual source (`.sh`, `.ps1`, …) sits next to it.

`scripts/create-users.yaml`:

```yaml
script:
  name: create-users
  description: Creates every employee's local account.
  language: bash
  source: create-users.sh
  timeout: 120
  people:
    - file: employees
  findings:
    - severity: 3
      difficulty: 2
      description: Employee accounts share a default password.
  validate:
    - user_exists: jdoe
```

`scripts/create-users.sh`:

```bash
#!/bin/bash
{{ range .people }}
useradd -m {{ .username }}
echo "{{ .username }}:{{ .password }}" | chpasswd
{{ end }}
```

A host runs it with a `script:` step: `- script: create-users`.

| Field | What it is |
| --- | --- |
| `name` | Unique name, referenced from a `script:` step. |
| `description` | Human-readable description of what the script does. |
| `language` | Which interpreter runs the rendered source (e.g. `bash`, `powershell`). |
| `source` | Path to the source file, alongside this definition (e.g. `install-mysql.sh`). |
| `timeout` | Seconds before the step running this script is killed as failed. A step that hits its timeout is not retried. |
| `args` | Extra arguments passed to the script when it runs. |
| `ignore_errors` | If true, a non-zero exit warns instead of failing the whole build. |
| `retries` | How many times a failed run is retried before the step fails (default 2). Set `retries: 0` for a long, deterministic script such as a wait loop, which would only fail the same way again. |
| `tags` | Labels; also flow to every host/container running this script. |
| `findings` | Findings this script introduces — they travel with it to every host that runs it. |
| `people` | People this script concerns, e.g. accounts it creates. Folded into `{{ .people }}` when the script renders. |
| `validate` | Checks run after every host/container's execution of this script, on top of any validate on the step itself. |

---

## People (CSV)

A `people/*.csv` file is a roster — the users, accounts, or personas a host or script
works with. The first row is a header; a `username` column is required.

`people/employees.csv`:

```csv
username,first_name,last_name,email,password
jdoe,Jane,Doe,jdoe@allports.local,Autumn2026!
awhite,Alan,White,awhite@allports.local,Autumn2026!
```

You reference a file (not individual people) from a host or script's `people:` list:

```yaml
people:
  - file: employees            # everyone in people/employees.csv
  - file: contractors
    filter: [bsmith, klee]     # just these two from that file
```

| Field | What it is |
| --- | --- |
| `file` | A `people/*.csv` file, by name without the extension. Unfiltered, every row is included. |
| `filter` | Usernames from that file to include instead of everyone — each must be a `username` value in that CSV. |

The resolved rows are available in the object's scripts as `{{ range .people }}` (each
row's columns are fields: `.username`, `.email`, …). A script can also pull a whole file
directly with the `people "employees"` function regardless of assignment.

---

## Steps: the action kinds

`steps:` is an ordered list. Each entry is **exactly one** action kind, plus an optional
`validate:` block. Steps run once, in order, after the host is provisioned. (For
recurring work, use [`schedule:`](#schedules) instead.)

Write steps so they're **safe to run twice**: install if missing, write the whole file,
idempotent commands. A rebuild runs them all again on a fresh box, and an operator can
**restart a host's build from any step** on the box as it is — that step and every one
after it run again in full (downloads fetch again, scripts write and run again,
validators re-check), possibly on a box where they half-happened the first time.

| Action | What it does |
| --- | --- |
| `script` | Runs a script definition (`scripts/<name>.yaml`) by name — the most common step. `- script: install-nginx` |
| `run` | An inline one-line shell/PowerShell command, for something too small for its own script. `- run: systemctl enable nginx` |
| `download` | Downloads a file from a URL onto the host (the agent fetches it directly). `- download: { from: "https://…/tool.zip", to: /opt/tool.zip }` |
| `upload` | Sends a file from the host back to LaForge (logs, generated credentials, evidence). |
| `write_file` | Writes a file with the given content, replacing anything there. `- write_file: { path: /etc/motd, content: "…" }` |
| `append_file` | Appends content to a file, creating it if needed. |
| `extract` | Extracts an archive (zip/tar/tgz) into a destination folder. `- extract: { src: /opt/tool.zip, dest: /opt/tool }` |
| `delete` | Deletes a file or directory. |
| `change_perms` | Changes a file/directory's permissions and/or owner. |
| `create_user` | Creates a user. `- create_user: { name: bob, password: "…", groups: [sudo] }`, or `generate:` instead of `password:` (see [Generated passwords](#generated-passwords)). |
| `set_password` | Rotates an existing user's password (separate from `create_user` so a later step can change it). Give `password:` or `generate:`; LaForge records the result and shows it on the host's Info tab under Credentials. |
| `add_to_group` | Adds an existing user to a group. |
| `service` | Starts, stops, or manages a systemd (Linux) or Windows service. |
| `reboot` | Reboots the host. The agent reports success first, then re-registers once it's back. Takes an optional `delay:` (a duration string like `"30s"`) to wait before rebooting; `reboot: {}` reboots immediately. |

Any step can carry a `validate:` block that runs after it (see below). A step can also
carry a [`wait_for:`](#waiting-on-another-host-wait_for-and-milestone), which holds it until
another host is ready, and a [`milestone:`](#waiting-on-another-host-wait_for-and-milestone)
name that other hosts can wait on.

---

## Waiting on another host: `wait_for` and `milestone`

`depends_on` is all-or-nothing: **none** of a host's steps run until the dependency has
finished **all** of its steps. That is often far more waiting than you need. A workstation
can install software, set its hostname and reboot without a domain controller. It only
needs the DC when it joins the domain. Even then, it needs a DC whose domain and DNS are
up, not a finished DC. That point usually comes well before the DC finishes setting up
logging and monitoring.

Two step fields let you say exactly that:

- **`milestone:`** names a point in a host's steps that other hosts can wait on.
- **`wait_for:`** holds a step, and every step after it, until another host reaches a
  milestone, a step, or the end of its steps.

### Example: a workstation that joins the domain

The domain controller marks the point where the domain is ready to join:

```yaml
host:
  name: bff-dc-01
  os: win2019
  steps:
    - script: winbase
    - script: win-domain-create
    - reboot: {}
    - script: win-dns-server-configure
    - script: win-domain-populate
      milestone: domain-ready       # domain, DNS and the user roster are in place
    - script: win-sysmon-install    # logging and monitoring: nobody waits on these
    - script: win-splunk-forwarder
```

The workstation runs everything it can on its own, then waits just before it joins:

```yaml
host:
  name: bff-ws-2019
  os: win2019
  # no depends_on: the wait_for below says exactly where the DC is needed
  steps:
    - script: win-time-configure    # these run as soon as the box is up
    - script: winbase
    - script: prepare-windows
    - script: win-reboot-record
    - reboot: {}
    - script: win-reboot-check
    - script: win-join-domain       # this step and every step after it wait
      wait_for:
        host: bff-dc-01
        milestone: domain-ready
    - script: win-reboot-record
    - reboot: {}
    - script: win-domain-member-check
```

Here is how the two hosts proceed:

1. Both boxes deploy at the same time.
2. While the DC is still building its domain, the workstation runs `win-time-configure`
   through `win-reboot-check`.
3. When the workstation reaches `win-join-domain`, it stops. In the UI's step list, that
   step and every step after it show as **Dependency Blocked**, under a note that reads
   *Waits for bff-dc-01 · domain-ready*. The host's own status changes from *Running
   Steps* to **Dependency Blocked** once it has nothing left to run before the wait.
4. As soon as `win-domain-populate` and its `validate:` checks complete on **the same
   team's** DC, the workstation's steps are released and it joins the domain.
5. Meanwhile the DC carries on with Sysmon and Splunk. Nobody waits on those.

### Two ways to wait

**Work up to a step, then wait.** Put `wait_for` on the first step that needs the other host.
Every step before it runs straight away. The wait only begins when the host reaches that
step. The workstation above works this way.

**Hold a host until another host is finished.** Put `wait_for` on the step you want to hold,
and give only the `host`. That step waits until the other host has finished **all** of its
steps. This is like `depends_on`, except that the steps before it still run straight away:

```yaml
steps:
  - script: install-tools           # runs straight away
  - script: configure-dns-client    # waits until every step on bff-dc-01 is done
    wait_for:
      host: bff-dc-01
```

To hold **all** of a host's steps until another host finishes, use `depends_on`, or put
`wait_for` on the first step.

### `wait_for` fields

```yaml
wait_for:
  host: bff-dc-01           # required
  milestone: domain-ready   # optional; use milestone OR step, never both
  rebuild: true             # optional, default true
```

| Field | What it is |
| --- | --- |
| `host` | The host or container to wait on, by its object `name` (the same name `depends_on` uses). It always means the copy in **the same team**. If the environment places that object more than once per team, every copy must reach the point. |
| `milestone` | Wait until that host completes the step marked with this `milestone:` name. This is the recommended form. |
| `step` | Wait until that host has run the `script:` step with this script name, e.g. `step: win-domain-populate`. The script must appear exactly once in that host's steps. If it appears more than once, mark the one you mean with a `milestone:` and use that instead. |
| `rebuild` | Whether rebuilding the host you wait on (with *include dependents*) also rebuilds **this** host. Defaults to `true`. Set it to `false` when this host is still fine after the target is rebuilt, e.g. it only copied a file or checked DNS once. See [Rebuilds and `wait_for`](#rebuilds-and-wait_for). |

If you give neither `milestone` nor `step`, the wait lasts until that host has finished
**all** of its steps.

A step can wait on several hosts at once. Give `wait_for` a list. The step is released only
when **every** entry is satisfied:

```yaml
- script: app-configure
  wait_for:
    - host: db-01
      milestone: schema-loaded
    - host: bff-dc-01
      milestone: domain-ready
```

`wait_for` works on hosts and containers, in any combination. A container can wait on a
host, and a host can wait on a container. It goes on `steps:` entries only. A
[`schedule:`](#schedules) entry runs on its own clock, so it can't wait.

### `milestone` fields

`milestone:` is a short name you can put on any step. It may contain letters, digits, `-`
and `_`. The milestone is reached when that step and its `validate:` checks complete. If
the step is an `ignore_errors` script that fails, the milestone still counts as reached,
because the host moves on past it.

Milestone names must be unique within a host. Different hosts may use the same name (for
example, `ready`). The `host` in `wait_for` says which host you mean.

Prefer `milestone` over `step`. A milestone says what the point **means** ("domain-ready").
The DC's author can then reorder, rename or split scripts without breaking every host that
waits on it, because the milestone moves with whichever step finishes the job.

### When the host you wait on fails

Sometimes the target can never reach the point you are waiting for:

- the step you wait on fails for good (after its retries, and without `ignore_errors`), or
- the host you wait on fails to build.

When that happens, the waiting host is marked **failed** with a reason such as `waiting on
bff-dc-01 · domain-ready, which failed`. This is the same way a failed `depends_on`
dependency is reported. Steps that already ran stay done. Fix and rebuild the target, then
rebuild the waiting host.

### Rebuilds and `wait_for`

When you rebuild a host with **include dependents**, LaForge also rebuilds every host that
depends on it. By default, that includes every host that has a `wait_for` on it. Set
`rebuild: false` on a `wait_for` to leave the waiting host alone.

Waiting hosts handle a target rebuild like this:

- A host that is still **blocked** on the target keeps waiting. It now waits for the
  rebuilt target to reach the milestone again.
- A host that has already passed its wait is not re-run unless it is rebuilt too.

### Checks LaForge runs on your content

`laforge check`, the VS Code extension and every push report these as errors:

- `host` names an object that doesn't exist.
- `milestone` isn't declared on any step of that host.
- `step` isn't a `script:` step of that host, or it appears more than once.
- Both `milestone` and `step` are set.
- A host waits on itself.
- Two steps on one host declare the same milestone.
- Waits form a loop (for example, A waits on B while B waits on A), whether through
  `wait_for`, `depends_on` or a mix of the two.
- `wait_for` appears on a `schedule:` entry.

If the environment doesn't place the target in the team at all, LaForge skips that
`wait_for` at build time and notes it in the build's events, the same as `depends_on`.

### `wait_for` vs `depends_on`

| | `depends_on` | `wait_for` |
| --- | --- | --- |
| Where it goes | On the host | On one step |
| What waits | All of this host's steps | That step and every step after it |
| Waits until | The other host finishes all its steps | A milestone, a step, or all its steps |
| Box deploy | Not delayed. Dependencies are deployed first. | Not delayed. Targets are deployed first. |
| Rebuild cascade | Always | Unless `rebuild: false` |

You can use both on one host. Note that a `depends_on` already holds every step of the
host, so a `wait_for` on the same target adds nothing.

---

## Validators

A `validate:` block is a list of checks that run after a step (or after every run of a
script). Every check runs even if an earlier one fails, so one run reports every problem.

```yaml
steps:
  - script: install-mysql
    validate:
      - service_running: mysql
      - port_listening: { port: 3306 }
        delay: "10s"          # wait 10s before this check, to let mysql finish starting
      - user_exists: dbadmin
```

| Check | What it asserts |
| --- | --- |
| `file_exists` | A path that must exist on the host after the step runs. |
| `file_absent` | A path that must NOT exist. |
| `file_hash` | A file whose contents must match an exact sha256 hash. |
| `file_contains` | A file that must contain specific text or match a regular expression. |
| `user_exists` | A username that must exist on the host. |
| `group_exists` | A group that must exist. |
| `user_in_group` | A user that must belong to a group. |
| `service_running` | A service that must be running (systemd on Linux, Windows Services on Windows). |
| `port_listening` | A port that must be listening. |
| `process_running` | A process name that must be running. |
| `registry` | A Windows registry value that must match exactly (Windows hosts only). |

### Delaying a check

Any check can carry an optional **`delay`** — a duration string the agent waits *before*
running that one check, so a service or port has time to settle after the step. It's a
sibling of the check key:

```yaml
validate:
  - service_running: nginx
    delay: "5s"
  - port_listening: { port: 443 }
    delay: "15s"
```

The value is a duration string like `"5s"`, `"10s"`, `"500ms"`, or `"1m"` — the same
format as `reboot`'s `delay:`. The delay applies only to the check it's on; checks without
a `delay` run immediately. Because the agent runs tasks on a background worker, a delay
never stops the host from checking in or from accepting an interactive shell while it
waits.

---

## Schedules

`schedule:` is a sibling of `steps:` for recurring or time-anchored work. Each entry is
one action kind (the same set as steps, minus `schedule` itself) plus a required
`when:` — a **natural-language** time expression.

```yaml
schedule:
  - when: every 30 minutes
    script: heartbeat-check
  - when: 45 minutes after competition start
    script: inject-phishing
  - when: every day at 2:00am
    reboot: {}
  - when: 30 minutes before access closes
    script: snapshot-evidence
```

`when:` has three shapes:

- **Interval** — `every [N] <minute|hour|day>(s) [before|after <time or anchor>]`:
  `every hour`, `every 30 minutes`, `every 15 minutes after 2:00pm`.
- **Daily at** — `every day at <time>[, <time>]…`: `every day at 10:00am and 2:00pm`.
- **Anchored** — `[N <unit>(s)] before|after <anchor>`: `45 minutes after competition start`.

Anchors: `competition start`, `competition end` (each fires once), and `access opens`,
`access closes` (fire once per access window — every day on a multi-day event). Times are
12-hour (`2:00pm`) or 24-hour (`14:00`).

---

## Templating

A script's source — and a `run:` command, and a [compose file](#compose-projects) — is
rendered with Go templates before it's used, per host per team. (In a compose file, quote
template values so it stays valid YAML, and note compose's own `${VAR}` syntax is left for
`docker compose` to resolve at runtime.) What's in scope:

| Reference | Value |
| --- | --- |
| `{{ .host.hostname }}` | This copy's hostname (its `as`). Also `.host.address`, `.host.os`, `.host.image`, `.host.size`, `.host.kind`. |
| `{{ .network.name }}` | This host's network. Also `.network.cidr`, and `.network.hosts` (its siblings on the same network). |
| `{{ .build.team }}` | This team's number. Also `.build.environment`, `.build.teams` (total), `.build.start`, `.build.stop`. |
| `{{ vars.company }}` | Any variable, cascaded environment → network → host (most specific wins). |
| `{{ range .people }}` | This object's assigned people rows; each row's columns are fields (`.username`, `.email`, …). |
| `{{ range people "employees" }}` | Every row of a named people file, regardless of assignment. |
| `{{ .dns }}` | The full resolved DNS record set, for a script that configures DNS. |
| `{{ passphrase 3 }}` | A [generated password](#generated-passwords). Also `wordchain`, `randompassword`. |

Example — a PowerShell script that creates every assigned employee:

```powershell
{{ range .people }}
New-LocalUser -Name "{{ .username }}" -Password (ConvertTo-SecureString "{{ .password }}" -AsPlainText -Force)
{{ end }}
Write-Host "Configured {{ .host.hostname }} for team {{ .build.team }}"
```

### Generated passwords

Three generators make passwords for you, available as template functions and as the
`generate:` option on `set_password` and `create_user`:

| Generator | Size means | Example |
| --- | --- | --- |
| `passphrase` | Words (1–12): capitalized 4–7 letter words, then a 4-digit number | `Patch-Session-Query-4484` |
| `wordchain` | Words (1–12): 3–5 letter words, each randomly capitalized | `fork-Brick-Spoon-Snail-test` |
| `random` (template: `randompassword`) | Characters (8–128), at least one upper, lower, digit and symbol | `p7K-x!Qe2+Wm_rTz` |

```yaml
steps:
  - set_password: { name: Administrator, generate: { type: passphrase, size: 3 } }
  - create_user: { name: svc_backup, generate: { type: random, size: 20, label: backup }, groups: [Administrators] }
  - script: write-admin-note   # can use {{ passphrase 3 }} -- the same Administrator password
```

Generated passwords are **stable for the build**: they're derived from a secret LaForge
keeps per build, so every render of the same host gives the same value, and a host
rebuilt within the build gets its passwords back. A new build gets new ones. Each value
is keyed by host, generator, size and label, so:

- `generate: { type: passphrase, size: 3 }` and `{{ passphrase 3 }}` on the same host are
  the **same password** — a script can write out the password a step set.
- A `label` gives a host a different password of the same kind:
  `{{ passphrase 3 "svc" }}`, `generate: { type: passphrase, size: 3, label: svc }`.
- A label starting `team:` shares one password across the team's hosts (a domain account
  every workstation uses); `build:` shares it across every team.

`laforge check`, `laforge render` and the editor preview have no build, so they render
stand-in values with the same shape.

---

## `extends`

Any environment, network, host, or container can `extends:` another object of the same
type to reuse a base. The base is **deep-merged** in:

- **Scalars** the child sets win; otherwise they're inherited (so a child can omit
  `os`/`size`/`disk` and take the base's).
- **Maps** (`vars`, `tags`, `env`) merge key by key, child winning a clash.
- **Lists** (`steps`, `schedule`, `people`, `ports`) are the base's followed by the
  child's.

```yaml
# hosts/base-linux.yaml
host:
  name: base-linux
  os: ubuntu22
  size: small
  disk: 20
  steps:
    - run: apt-get update
---
# hosts/webserver.yaml — inherits os/size/disk/steps, adds its own
host:
  name: webserver
  extends: base-linux
  ports:
    tcp: ["80"]
  steps:
    - run: apt-get install -y nginx     # runs after the base's apt-get update
```

---

## Ports

`ports:` is a host/container's ingress firewall: from an allowed network, the host is
reachable **only** on the listed ports; a host that declares no ports is reachable on
none.

```yaml
ports:
  tcp: ["80", "443", "8000-8100"]     # single ports or ranges
  udp: ["53"]
```

Combined with a network's [`visible_from`](#network) (which *networks* may reach it),
this is the complete reachability model.

---

## Container log forwarding

Set `container_logs` on the **environment** to forward every container's console output to
an external collector. It's one block for the whole environment — you don't annotate
individual containers, and it covers both single-image and compose containers identically:

```yaml
environment:
  name: allports
  container_logs:
    driver: splunk
    options:
      splunk-url: https://splunk.example:8088
      splunk-token: 00000000-0000-0000-0000-000000000000
      splunk-index: cptc
```

- `driver` is the log driver name as the platform knows it (`splunk`, `fluentd`, `awslogs`, …).
- `options` are that driver's own keys, passed straight through (the Splunk ingestion token
  lives here — it's a write-only HEC token, and content isn't shared).

**How it's delivered (native-first).** LaForge doesn't ship the bytes itself where the
platform can — each builder uses its own native logging, so a large number of containers
never funnels through LaForge:

| Where the container runs | How `container_logs` is applied |
| --- | --- |
| MicroCloud (nested docker) | `docker run --log-driver …` |
| AWS Fargate | the task's `logConfiguration` |
| A compose project (any builder) | the Docker **daemon default** on its host, before `docker compose up`, so every service inherits it |
| Incus (native OCI) / OpenStack Zun | **fallback:** the agent forwards to the gateway, which ships to the collector |

The fallback (Incus/Zun, which have no native driver) currently supports the **`splunk`**
driver only; other drivers on those two builders aren't forwarded. A compose service that
declares its own `logging:` block still overrides the daemon default.

This is container-only — hosts are full VMs with their own logging. Omit `container_logs`
to forward nothing.

---

## Findings

A finding records a vulnerability or weakness present in the environment, for scoring and
reporting. Findings can live on an environment, network, host, container, or script; a
script's findings travel to every host that runs it.

```yaml
findings:
  - severity: 4        # 1 (minor) … 5 (critical)
    difficulty: 2      # 1 (trivial) … 5 (very difficult)
    description: SQL injection in the login form.
```

---

## Checking your content

Validate a content repository before committing:

```bash
laforge check .
```

It parses every file against the schema, runs the cross-file checks (name collisions,
`depends_on` and `visible_from` targets exist, script/people references resolve, schedule
expressions parse, validators match the host OS), and renders every script for every host
in every team to catch template errors — without touching any hoster. A clean run means
the content will build. The VS Code extension runs the same checks live as you type.

For a container that runs a [Compose project](#compose-projects), it also loads the
project: the compose file must exist and parse, every service must name an `image:`, and
every relative bind mount must be inside the project's directory.

Every `.yaml`/`.yml` file is checked unless [`.laforgeignore`](#how-content-is-organized)
lists it. A file that isn't LaForge content and isn't ignored fails with "no type header
found".
