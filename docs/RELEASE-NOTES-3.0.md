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
  (each with an optional `delay:` to let a service settle before it's checked)
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
- **Generated passwords.** `set_password` and `create_user` take `generate: { type, size,
  label }` in place of `password:`, and templates have the same generators as functions:
  `passphrase` (`Patch-Session-Query-4484`), `wordchain` (`fork-Brick-Spoon-Snail-test`)
  and `randompassword`. Values are stable for the build (derived from a per-build secret),
  so a script can use `{{ passphrase 3 }}` to write out the password a step set; a `team:`
  or `build:` label shares one value across a team or every team.
- **Credentials are tracked per host.** Every password LaForge sets — the automatic admin
  step, `set_password`, `create_user` — is recorded per account and shown on the host's
  Info tab (with whether its step has run yet), listed by `laforge passwords`, and used
  as the login in external-access exports.
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
  Template functions (including the password generators) complete and hover inside
  `{{ }}` in scripts and in templated YAML values, and nested step objects such as
  `generate: { type: … }` complete their keys and values.

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
- **Step-level waits: `wait_for` and `milestone`.** A step can wait on just the point
  another host needs to reach, e.g. a workstation's `win-join-domain` waits for the DC's
  `domain-ready` milestone. The steps before the wait run straight away, and the DC's
  logging tail no longer holds anyone up. Every reference is checked at push time, a
  failed target fails the waiting host with a clear reason, and `rebuild: false` keeps a
  waiter out of its target's rebuild cascade. See
  [CONFIGURATION.md](../CONFIGURATION.md#waiting-on-another-host-wait_for-and-milestone).
- **Force-rebuild** a host and all its downstream dependents from the UI, and **clean
  teardown** at the end of an event.
- **Restart from step.** A host whose build stopped — a failed step, a validator that
  didn't pass — can be restarted from any authored step instead of rebuilt: that step and
  every one after it run again in full (downloads fetch again, scripts write and run
  again, validators re-check) on the box as it is, nothing destroyed, no hoster call.
  Picked once for a host definition and applied to the chosen teams' copies; hosts that
  were failed for depending on it resume. From the host's Steps panel, the hosts page,
  or `laforge restart-steps --build <id> --from <step>`. Steps should be safe to run
  twice (CONFIGURATION.md → Steps).
- **Hold hosts on step-only changes** (per configured build, off by default): a push
  that changes only a host's steps or scripts marks the host *steps changed* instead of
  recreating it, so the fix is applied with a restart from the changed step. Changes to
  the image, size, disk, ports or dependencies still recreate the host; turning the
  switch off recreates any held hosts.

## The Agent

- A single, fully-embedded **Rust binary** with no external config: it holds its pinned
  identity, opens an **mTLS** connection to the gateway, pulls its ordered steps, runs
  them, and reports structured results — including **per-check validator results**.
- **Cross-platform**: static musl Linux builds and Windows builds (cross-compiled), with
  real step actions (scripts, users, services, file write, **direct** download, extract).
- **Host metrics on every heartbeat** (CPU, memory, disk, network) and **container
  log forwarding** to an external collector (Splunk, …), configured per environment with
  `container_logs` in content. It's **native-first** — each builder uses its own log
  mechanism (a docker `--log-driver`, a Fargate `logConfiguration`, the Docker daemon
  default for a compose project) so a large build's log volume never funnels through one
  LaForge process; Incus (native-OCI) and Zun fall back to the agent → gateway.
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
  the CLI (`laforge access`, with `--csv`/`--json` including each host's current login).
- **Export for Scoring.** The External Access page's wizard (and `laforge access
  --scoring`) writes the scoring app's Team Connections import JSON: pick endpoints
  (leaving your own hosts, like OutOfScope, out of every team), set the scoring season,
  competition and team-number offset, and download.
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

- **Everything LaForge does is logged, twice**: as a line in the service's own log and
  as an event on the build's Logs tab. The orchestrator records every decision with its
  reason (deploy, recreate, remove, rebuild, teardown, held, failed for a network or a
  dependency, each status move, and what it's waiting on); the runner every task
  attempt and its duration; the api every operator action with who made it; the
  builders every hoster request, operation and destroy phase; the gateway every step.
  Long log lines open in place on the Logs tab, which loads the newest 5,000 events and
  fetches earlier pages on demand ("Load earlier"), so a long or stuck build's journal
  always opens; the live stream starts at the newest loaded event rather than replaying
  the journal from the beginning, and rows are ordered by time whatever request they
  arrived in. "Still waiting" entries are written to the journal at most twice an hour
  per host (the orchestrator's own log has them every minute) and pruned after a day.
- **Concurrency limits hold.** A builder's limit (*Max concurrency* in the wizard) now
  applies to the next task leased, not after a pool has drained, and a task is only
  leased once a slot is free (before, one task per pool sat leased and unheartbeated
  while waiting). The runner logs each pool's limit, every lease's slot, and "at its
  limit, N waiting" every 30 s while saturated. A failed destroy — a rebuild's or a
  content change's — is queued again (with its hoster reference) before anything is
  redeployed in its place, and a host whose old instance can't be destroyed is stopped
  with the error instead of sitting pending.
- **Reconcile passes finish.** The build/builder compatibility check read every source
  image's snapshot from the hoster on every 2-second pass (10+ seconds each on a loaded
  MicroCloud cluster), so a pass timed out inside team 1 and later teams were never
  reconciled. A passed check is now remembered per build and revision for an hour, and
  each source's size per builder; a pass that does time out records why on the build
  (`reconcile.error`, naming the phase) and `reconcile.recovered` when passes complete
  again. Builder metrics probes are bounded to 30 s.
- **Nothing waits on a hung hoster.** The orchestrator's periodic jobs (power-state
  poll, access and schedule dispatch, housekeeping, metrics) run off its main loop, each
  bounded and never two at once — a tick that finds the previous run still going is
  skipped and logged with how long it has been running. Power states are inspected per
  builder in parallel, 60 s each; the check-in watchdog's hoster calls are bounded; the
  runner fails and retries a task that runs past `RUNNER_TASK_TIMEOUT` (45 m) so a hung
  builder call can't hold a concurrency slot forever; the api has header/idle timeouts
  and bounds its live drift scan.
- **Saving a builder no longer waits on the hoster** unless its images, hosts or
  connection changed; a concurrency or size change saves at once. When images are
  re-checked, the snapshot reads run in parallel, and if the check still fails or times
  out the wizard offers *Save without verifying images*.
- **No more stalls between deploys, destroys, rebuilds and teardowns.**
  - A deploy that finds its instance already exists — an earlier attempt the cluster
    accepted before the runner went away — waits for that creation to finish and adopts
    it when it's this deployment's (every instance now records which deployment made it),
    instead of failing; someone else's is refused, permanently. Both the MicroCloud and
    Incus builders do this; instances created before deployment names were recorded
    are matched by team. A destroy likewise waits out an unfinished creation instead of
    racing it.
  - Restarting or redeploying the runner hands its in-flight tasks back (pending, the
    lease not counted as an attempt) instead of leaving them leased until they expire and
    count as failures. The runner gets 30 s to stop.
  - A rebuild of a host with a deploy queued withdraws that deploy; with one running, the
    rebuild is recorded and carried out the moment it ends — it used to reset the host,
    losing its hoster reference, and the request was then forgotten.
  - A teardown withdraws deploys that haven't started rather than waiting for them to
    create instances only to destroy them; while something is still running, it says
    what.
  - A host whose deploy failed is retried after a backoff (30 s doubling to 10 m) rather
    than on every pass, and not after five failures in a row or a permanent failure — a
    rebuild resets that.
- **MicroCloud deploys no longer queue behind ACL rewrites.** Every external-access pass
  re-wrote every public host's ACL, unchanged, holding the builder's networking lock
  across the whole team — and on a loaded cluster an ACL write takes minutes, while
  Windows/public-NIC deploys need that lock for their own ACL. An ACL that already
  matches is now left alone (read first, no lock), the lock is taken only around a real
  write, and waiting on it is logged every 30 s, as is any hoster request still
  unanswered after 30 s. Opening/closing a team's access and configuring its network
  access no longer hold the lock across the whole team either — each network change and
  each ACL write takes it for itself — and a host's NIC isn't updated when its port ACL
  is already attached. Access and network configuration tasks run in their own lane,
  capped at the builder's limit (at least 4), beside its deploys.
- **Lost agents fail the host instead of stalling it.** A host whose agent checked in and
  then went silent for `AGENT_LOST_AFTER` (30 m) with steps still to run, or whose agent
  never connected within `AGENT_NEVER_CHECKED_IN_AFTER` (60 m) of the instance coming
  up, goes `build_failed` with the reason and time — visible, and rebuildable or
  restartable — rather than sitting "building" with 31 pending steps for the rest of the
  event.
- **A builder's slots are shared fairly between builds.** The next task leased comes
  from the build with the fewest deploys already running, so one build's backlog of a
  hundred deploys can't starve another build's two.
- **Destroys lease ahead of deploys.** A rebuild's or teardown's destroy no longer waits
  behind a queue of deploys for one of the builder's slots: it frees capacity and is what
  the operator asked for, so it goes first.
- **The concurrency limit counts deploys and destroys.** A builder's *Max concurrency*
  is how many instances are created or destroyed at once; access and network
  configuration tasks (seconds each) run alongside instead of queueing behind a hundred
  deploys. The runner's log names the slot per lease and what's waiting.
- **Leaner MicroCloud destroys.** A destroy reads the instance once, stops and deletes
  it, and then removes only what that instance actually had — its public NIC's ACL, its
  per-host port ACL, its Windows config-drive volume — concurrently, instead of a blind
  sequence of every cleanup plus a confirmation read. A plain Linux host is three hoster
  requests rather than seven.

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
  `builders`, `images`, `registries`, `run`, `schedule`, `access`
  (+`--csv`/`--json`/`--scoring`), `passwords`, `cert-status`, and `shell`.

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
