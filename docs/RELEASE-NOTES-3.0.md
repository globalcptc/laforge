# LaForge 3.0

**A ground-up rewrite.** LaForge 3.0 replaces the previous HCL/`ent` codebase with
a new system built around **plain-language YAML content in Git**, a small set of
single-purpose Go services, a **Rust on-host agent**, and a modern React console.
The guiding idea: describe a competition environment the way you'd describe
application config, commit it, and get real, identically-replicated infrastructure
on whatever platform you have — without the content ever naming a hoster, an image
ID, or an IP pool.

This note covers the whole 3.0 line.

---

## Highlights

- **Infrastructure-as-content in YAML.** Networks, hosts, containers, the software on
  them, and who can reach what — authored as YAML, validated on every push, and the
  single source of truth for a deploy.
- **Write once, deploy anywhere.** Content is builder-agnostic. One Go `Builder`
  interface maps abstract concepts (`os: ubuntu22`, `size: small`) to a platform, with
  builders for **Incus**, **MicroCloud/LXD**, **AWS**, and **OpenStack** (plus a `fake`
  builder for tests and an Incus host-pool variant).
- **Every team an identical, isolated copy.** You write the topology once; LaForge
  replicates it per team and keeps teams off each other.
- **Hosts configure themselves.** A small, embedded Rust agent pulls each host's ordered
  setup steps, runs them, validates the result, and heartbeats — so "the environment is
  ready" is an observable state, not a hope.
- **Access is a schedule.** Windows open and close on their own; per-network
  reachability and per-host firewalls are declared in content and enforced for you.
- **A real operator console and CLI**, an **mTLS** trust model end to end, and an
  authoring experience with a **VS Code extension** and language server.

---

## Content & Authoring

- **YAML content model.** Environments, networks, hosts, containers, scripts, and people
  are authored as YAML and committed to a Git repo. Content is deliberately
  platform-neutral — no hoster, image, or IP details leak in.
- **Validate before you deploy.** `laforge check` runs the exact same schema +
  cross-reference validation the server runs on every push: every file against its
  schema, every reference (scripts, hosts, networks, `depends_on`) resolved, and every
  script rendered against every host in every team.
- **Rich authored surface**: per-host/container `steps:` (install software, create
  users, write files, download/extract, run scripts) with per-step `validate:` checks
  and `ignore_errors`; a top-level `schedule:` array with **natural-language timing**
  ("every 30 minutes", "45 minutes after competition start"); `depends_on` ordering;
  `tags:` that cascade from environment → network → script → object; `ports:` firewalls;
  a `public:` block for external access; and a `people:` field backed by CSV rosters.
- **Docker Compose projects as containers.** A `container:` can name a compose file
  (`compose: app/compose.yaml`, relative to its own file) instead of an `image:`. The
  compose file's directory is shipped whole and run with real `docker compose` on one
  machine with one address; images are pulled, never built. Supported on the Incus and
  MicroCloud builders, which boot that machine from the builder's docker base image —
  now built for Incus builders too, and now installing Docker Engine and the compose
  plugin from Docker's own apt repository instead of Ubuntu's `docker.io` package.
- **`.laforgeignore`.** A content repo can list paths that aren't LaForge content (a
  compose project, another tool's YAML) in a `.gitignore`-style file at its root;
  `laforge check`, the server, and the editor all skip them.
- **One-shot migration from 2.x.** `laforge convert <old-hcl-repo> <new-repo>` translates
  an existing HCL content repository into the new YAML format, printing every judgment
  call and anything that needs manual review.

## Authoring Tooling (VS Code + LSP)

- A **VS Code extension** (`editors/vscode/`) and a standalone **`laforge-lsp`** language
  server give schema-aware completion, hover docs, a full **field-reference panel**, a
  **live render preview** of a script against an example host, and `laforge check`
  diagnostics as you type — the logic lives once in Go, so any LSP editor benefits.

## Builders & Platforms

- **One interface, many platforms.** Each builder implements a single Go contract
  (deploy/destroy networks, hosts, containers; open/close access; configure network
  reachability and external access). No capability opt-outs — a builder implements the
  whole contract or returns a clear error.
- Builders ship for **Incus**, **MicroCloud/LXD** (note: these are *separate* builders),
  **AWS**, and **OpenStack**, plus an Incus **host-pool** placement variant.
- **OS → image mapping** is a builder's job: content says `os: ubuntu22`, the builder
  resolves it to a real image, and the CLI/API can list a builder's mappings and the
  images its hoster actually holds — so "no image for X" is diagnosable, not a mystery.
- **Private registry credentials** and a **builder image-build** path are supported for
  container workloads.

## Deploy & Lifecycle

- **Build and deploy are separate verbs.** A `planned` build resolves the whole
  environment (real teams, real fingerprints, fully inspectable) while touching no
  hoster; deploying applies it.
- **A reconcile loop, not a script.** The orchestrator compares each live build's desired
  state to reality on a timer and enqueues tasks; the runner leases and executes them in
  **parallel across a worker pool**. Deterministic resource names make every task safe to
  retry (create-or-adopt), and **"rebuild means recreate"** — a changed fingerprint tears
  the object down and redeploys it.
- **`depends_on` waits for configuration, not just boot.** The box deploys ahead of time
  (roots first, for efficiency), but a dependent's **steps** don't run until its
  dependencies have fully *finished* configuring — a domain controller must be a working
  DC, not just a booted Windows box.
- **Force-rebuild** a host and all its downstream dependents from the UI, and **clean
  teardown** at the end of an event.

## The Agent

- A single, fully-embedded **Rust binary** with no external config: it holds its pinned
  identity, opens an **mTLS** connection to the gateway, pulls its ordered steps, runs
  them, and reports structured results — including **per-check validator results**.
- **Cross-platform**: static musl Linux builds and Windows builds (cross-compiled), with
  real step actions (scripts, users, services, file write, **direct** download, extract).
- **Host metrics on every heartbeat** (CPU, memory, disk, network) and **container
  console-log forwarding** to a generic JSONL sink (Splunk/Vector/Loki/Elastic-friendly).
- Runs as PID 1 inside application containers on every builder, so containers check in and
  run steps exactly like hosts.
- Basic **anti-tamper / anti-debug self-checks** and per-build chaff.
- **Silent on the box by default**: the agent writes nothing to stdout, stderr, or disk —
  only what it reports to the servers over mTLS ever leaves it, so a captured box yields no
  local agent logs. An environment can set `agent-debug: true` to get a local log file next
  to the binary for debugging; the flag is baked into the binary, so it can't be switched on
  by editing a box's launcher.

## Access & Scheduling

- **Scheduled access windows** open and close competition access automatically; an
  operator can **extend or reduce** a team's end time (including as a penalty), and the
  schedule is enforced, not babysat.
- **Per-network reachability** (`visible_from`) and **per-host port firewalls** (`ports:`)
  are declared in content and enforced at the hoster — on Incus/MicroCloud via **OVN NIC
  ACLs** (verified live), with cloud security-group support in progress.
- **External access (NAT-in)** for specific ports (e.g. RDP): a host's `public:` ports are
  realized as reachable endpoints per builder — a shared uplink IP with per-team ports on
  Incus/MicroCloud, a public IP per host on cloud — surfaced in the UI and exportable from
  the CLI (`laforge access`, with `--csv`/`--json` including rendered logins).
- **Scheduled in-guest work** (the `schedule:` array) dispatches real agent commands at
  natural-language times and recurrences, including once-per-window anchors.

## Interactive Remote Shell (New)

- Open a **fully interactive root/admin shell** on any deployed host or container, from
  **both the UI** (a per-host "Open terminal") **and the CLI** (`laforge shell`). It's a
  real PTY — vim, colors, Ctrl-C, tab-completion, window resize — **encrypted end to end**
  (WebSocket/TLS to the API, internal mTLS to the gateway, mTLS to the agent), gated on
  manage access like `laforge run`, audited per session, and capped to one or two at a
  time. Linux (openpty) and Windows (ConPTY).

## Operations & Observability

- **A React operator console**: active builds grouped by repository on a home dashboard;
  per-build hosts/containers/networks with live status, infra power state, and agent
  health; live **logs over SSE**; **findings**, **topology**, **access** and **external
  access** views; the **schedule**; **artifacts**; and per-host **heartbeat history with
  metrics**.
- **An append-only event journal** records every lifecycle, step, access, and shell event
  for live troubleshooting and after-action review; live updates stream over SSE via
  Postgres `LISTEN/NOTIFY`.
- **"Needs attention"** on the home page is scoped to the viewer's own builds and is
  per-user dismissible, so a busy instance stays legible.
- **Ad-hoc and scheduled commands** across targeted hosts (`laforge run` / `laforge
  schedule`) with team/tag/kind/id/search/network selectors and a dry-run.
- **A CA/cert expiry banner** and a `cert-status` endpoint/CLI so the agent trust anchor
  never lapses silently.
- **Version self-check**: the CLI and the language server compare their build to the
  latest GitHub release and nudge you to update (`LAFORGE_NO_UPDATE_CHECK` opts out).

## Security & Access Control

- **GitHub App + OAuth.** Repositories connect via a GitHub App; operators sign in with
  GitHub. Access is limited to people with access to the connected repositories, with
  per-repository levels (read / build / manage / admin) and instance admins.
- **mTLS everywhere it matters.** A self-signed CA anchors the gateway; the runner mints a
  per-host agent client certificate at deploy time; the agent pins the gateway. The new
  shell relay reuses the same CA with its own internal listener and an API client cert.
- A hardened agent-gateway wire protocol (length-prefixed JSON inside TLS, fuzz-tested
  framing) that **speaks only the agent protocol** to the outside.

## CLI

`laforge` covers the authoring and operating surface:

- **Local, no server:** `context`, `render`, `check`, `convert`.
- **Server-backed:** `login`, `repo`, `build` (configure/list/status/follow/lock),
  `builders`, `images`, `registries`, `run`, `schedule`, `access` (+`--csv`/`--json`),
  `cert-status`, and `shell`.

## Platform & Deployment

- Runs as a small **docker-compose** stack: `api`, `orchestrator`, `runner`, `gateway`,
  `ui`, Postgres, and a `migrate` step (goose). An agent-factory image cross-compiles the
  Rust agent for the deploy targets.
- An **OpenAPI 3.1 spec** (`docs/openapi.yaml`) documents the HTTP API.

---

## Upgrading from 2.x

LaForge 3.0 is a **new system**, not an in-place upgrade:

- **Content format changed** from HCL to YAML — run `laforge convert` on your old content
  repo and review what it flags.
- **Fresh database** on the new schema (goose migrations). Follow the migration-deploy
  sequence in `docs/` / `USAGE.md`; a constraint-tightening migration wants a `migrate`
  rebuild and writers stopped first.
- **Set up certificates** with `scripts/gen-certs.sh` (now also issuing the API client
  cert for interactive shells — pass the API-facing gateway name as a second argument when
  you want that feature).

See **[README.md](../README.md)**, **[CONFIGURATION.md](../CONFIGURATION.md)**, and
**[USAGE.md](../USAGE.md)** to get started.
