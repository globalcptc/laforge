# LaForge 3.0 — UI and UX specification

Companion to [the rewrite plan](let-s-start-to-plan-lively-matsumoto.md). That document
decides what the system does; this one decides what a person sees and does.

## Who uses this, and when

Three very different modes, and the interface has to serve all of them without becoming
three products.

| Mode | Who | What it feels like |
| --- | --- | --- |
| **Authoring** | Volunteers writing hosts, scripts, and people. The largest group, least interested in LaForge itself. | Calm, iterative, days or weeks before an event. Mostly in an editor, occasionally here to check a build. |
| **Operating** | Whoever runs the competition. A handful of people. | Live, tense, hours. Something is wrong and twenty teams are waiting. |
| **Administering** | Repo owners and the people who grant access. | Rare, deliberate, consequential. |

The operating mode sets the bar. **Anything that is hard to find at 2am is broken**, and
that principle decides most of the arguments below.

**These are usually the same people.** The volunteer who wrote the environment is often
the one running it during the event. So there is one path, not two: everything lives
under the repository they already know. Builds are not a separate destination; they are
something a repository has.

## Principles

1. **Show the blast radius before the action.** Every destructive or wide-reaching
   action states what it will touch, with a count, before it runs. "Reboot 20 hosts
   across 20 teams" is a sentence the operator reads before confirming, not a thing they
   discover afterwards.
2. **Errors live in the interface.** Never the browser console. A failure appears where
   the work is, says what failed, and links to the evidence.
3. **Colour is never alone.** Every state pairs a colour with an icon and a label.
4. **One hop to the cause.** A red cell reaches the failing step, its rendered script,
   the source file, and the commit without a search.
5. **The same selection model everywhere.** Filters, search, and bulk actions behave
   identically in every view that lists things.
6. **Nothing important is transient.** Toasts are a courtesy; the message centre is the
   record.
7. **Scale is a design input, not an optimisation.** Four thousand hosts is the normal
   case, not the stress case.

## Information architecture

```
Sign in (GitHub)
│
├── Home                     your active builds grouped by repo, with hosting metrics
│
├── Repositories
│   └── <repo>
│       ├── Overview         branches, recent commits, validation status
│       ├── Environments     environment files at a ref, with a resolved preview
│       ├── Builds           every build of this repo
│       │   └── <build>
│       │       ├── Overview     state, commit, builder, counts, storage
│       │       ├── Hosts        matrix ⇄ table, selection, bulk actions
│       │       │   └── <host>   state, steps, logs, rendered scripts, actions
│       │       ├── Topology     what the environment resolved into
│       │       ├── Logs         hierarchical, filterable, live
│       │       ├── Access       windows, per-team state, countdown
│       │       ├── Schedule     injects, upcoming and past
│       │       ├── Findings     instances in this build, export
│       │       └── Artifacts    storage by class, purge, delete
│       ├── People           the user database from this repo's CSVs
│       ├── Findings         defined in this repo's content
│       └── Settings         access, configured builds, auto-build/deploy  (admin)
│
└── Admin
    ├── People & access      who can do what, per repository
    ├── Builder configs      hosters, credentials, image and size maps
    └── Repositories         add, remove, defaults
```

**Everything flows through the repository.** A build is always reached as
`repository → Builds → <build>`, never as a separate top-level list, because the people
using this think in terms of the competition they own, not a flat pool of builds across
everything. Home is the one exception, and it is a shortcut into this tree rather than a
parallel structure: it gathers the builds you care about so you do not have to walk into
each repo to find them.

## Global shell

### Top bar

```
[ LaForge ]   [ repo / environment ▾ ]        [ ⚠ 3 ]  [ 🔔 12 ]  [ avatar ▾ ]
```

- **Context switcher** — current repository and, when inside one, the build. Keyboard
  reachable, searchable, most-recent first. This is how someone running three
  simultaneous events moves between them.
- **Attention badge (⚠)** — unresolved build or deploy failures. Distinct from messages
  because it means *work is stopped*. Zero state: hidden entirely.
- **Message centre (🔔)** — see below.
- **User menu** — account, API keys, theme, sign out.

During an event the top bar also carries the **access countdown** for the current build,
because it is the single most asked question and it should never require navigation.

### Navigation

Left sidebar, collapsible. Sections appear according to access level, so a read-only
user does not see Admin at all rather than seeing it disabled. Modules register
themselves, so a new feature is additive.

### Message centre

A count, a list, and a brief toast on arrival that settles into the list.

| Kind | Toast | Placement | Persists |
| --- | --- | --- | --- |
| Build or deploy failure | No | **Banner on the build**, and the ⚠ badge | Until acknowledged |
| Step failure, agent missing, validator failure | Yes | Message list | Yes |
| Completion, state change, access opened/closed | Only if the user triggered it | Message list | Yes |
| Someone else's action on a build you watch | No | Message list | Yes |

Server-side, per-user read state, survives refresh. A build that fails at 3am is waiting
in the morning. Every message carries its object and clicks through pre-filtered.

Filters: unread, failures only, this build, this team.

## Screens

### Sign in

Single GitHub button. No local accounts, no password field, nothing else on the page.

**After sign-in with no access:** an explicit empty state — "You are signed in as
@lucas. No repositories have been shared with you." — plus who to ask. This is a real
state for a new volunteer and it should not look like a broken page.

### Home

The landing page after sign-in. Its job is to answer "what is live right now, and is it
healthy" without walking into each repository. **Active builds, grouped by repository**,
with the hosting picture for each.

Layout: one section per repository the user has access to that has an active build, most
recently active first. A repository with nothing deployed does not appear here — it is
still one click away under Repositories.

Within each repository's section:

- **Each active build** (deployed or deploying) as a card: environment, state, commit,
  builder, and a health strip — team count, agents healthy / late / missing, steps
  outstanding, failures.
- **Hosting metrics for that build's environment**: how much is deployed against it
  (hosts, containers, networks), and whatever the builder reports about the target —
  capacity or quota used where the hoster exposes it, public endpoints in use, the
  runner activity currently touching it. Enough to see at a glance whether the hosting
  environment itself is the problem, before digging into individual hosts.
- **The access countdown** if the build has access windows and one is active.

Above the per-repo sections, a thin **attention strip** surfaces anything failed across
all of them, because a stuck build is the one thing that should jump the grouping.

For a volunteer with one competition this is that competition's status board. For
someone running three at once it is all three, side by side, still organised the way
they think about them.

Every card and metric clicks straight through to the relevant screen inside that repo's
build, pre-filtered.

### Repository → Overview

- Branches, with each one's latest commit and **validation status** (the same output as
  `laforge check`).
- A failed validation shows file, line, and message inline. This is where an author
  lands after a push that did not pass.
- Configured builds for this repo, and their current state.

### Repository → Environments

The environment files at a chosen ref, and for each one a **resolved preview**: teams,
networks, hosts and containers, addresses, public endpoints. Computed without touching
a hoster, so it is free and instant.

This is the answer to "what would this build actually create", available before anyone
commits to anything. It is the same resolution the build step performs, so what is shown
is what would be built.

Actions: **Build** (resolve at this commit), and from a build, **Deploy**.

### Repository → Builds

Every build of this repository, filterable by environment, state, and builder. Columns:
environment, state, commit and branch, builder config, teams, created, who, storage.

Scoped to the one repository, because that is where builds live in this structure. There
is no cross-repository build list; Home is where builds from several repos come
together, and it shows only the active ones on purpose.

State is the vocabulary from "Status", with `planned` visually distinct from `deployed`
because one exists only in LaForge and the other is costing a sponsor money.

### Build → Overview

This is the build's dashboard. It answers "is this build healthy" before anything else,
so the top of the screen is a **health band** — metrics and charts — and the detail
follows beneath.

The header carries state, environment, commit (linked to GitHub), builder config, and
team count.

**Health band (top of the screen).** A row of stat tiles and a few charts, all fed by
server-side aggregates rather than counted in the browser, all using the `StatusBadge`
colour-and-icon vocabulary so a glance reads the same as everywhere else.

Stat tiles, the numbers that matter in one look:

| Tile | Shows |
| --- | --- |
| **Provisioning** | Hosts provisioned of total, as a big number and a percent |
| **Agents healthy** | Count and share checked in within the window |
| **Agents late / missing** | Broken out, because missing is a different problem from late |
| **Failures** | Failed steps or builder ops right now, red when non-zero |
| **Queued work** | Tasks no agent has collected — the early warning before failures |
| **Access** | Open, closed, or mixed, with the countdown when a window is active |

Charts, for shape and trend rather than a single value:

- **Provisioning progress** — a stacked bar across the whole build: pending → running →
  provisioned, with failed called out. The single clearest "how far along is this" view,
  and during a deploy it fills in live.
- **Agent check-ins over time** — a line chart, so a cliff (a network dropping, a team
  going dark) is visible the instant it happens rather than inferred from a count.
- **State by team** — a small multiple or stacked bar per team, which surfaces the
  "one team is wrong" shape here on the dashboard before anyone opens the host matrix.
- **Failures grouped by cause** — a short ranked bar: "40 hosts: `install-mysql` exit 1"
  as one entry, not forty. This is the thing that turns a wall of red into a single
  fixable problem.

Every tile and chart is a link: click Failures and land in the matrix filtered to failed
hosts; click a team's bar and land filtered to that team; click check-ins and open the
live log.

**Below the band:**

- **Health detail** — failures grouped by cause, expanded, each expandable to the hosts
  it covers and one hop from there to the rendered script and commit.
- **Access** — current state, countdown, per-team exceptions.
- **Upcoming changes** — if a newer commit has built, what deploying it would alter, at
  environment, team, network, and host level.
- **Storage** — by class, with what each cleanup step would free.
- **Actions** — deploy, rebuild, destroy, purge, delete, mark competition started.

### Build → Hosts (matrix)

The primary operating view. Teams down, hosts across, grouped by network, one cell per
host, coloured and iconed by state.

```
            ┌─ prod ──────────────┐ ┌─ vdi ────┐ ┌ vpn ┐
            db01  web01  dc01  ws01  kali01 kali02  wg
  Team 1     ●     ●      ●     ●      ●      ●     ●
  Team 2     ●     ●      ▲     ●      ●      ●     ●
  Team 3     ●     ✕      ●     ●      ●      ●     ●
  Team 4     ✕     ✕      ✕     ✕      ✕      ✕     ✕     ← whole team
  …
             ↑ one host, every team
```

- **A bad row is one team.** Their network, their build, something local.
- **A bad column is one host everywhere.** A broken script or image — content, not
  infrastructure.

That distinction is the first question asked when something breaks, and the shape of the
view answers it without a query.

**Cell states.** Each cell is a filled shape carrying one **primary state** — colour,
icon, and, in the accessible/high-contrast mode, a distinct glyph — so it never relies
on colour alone. This is the authoritative vocabulary that `StatusBadge` renders
everywhere else; the matrix is just its densest use. Icons are Lucide names.

| State | Colour | Icon | Meaning |
| --- | --- | --- | --- |
| **Pending** | Slate / muted | `circle-dashed` | Declared, nothing created yet |
| **Deploying** | Blue | `loader` (spin) | Builder is creating it at the hoster |
| **Booting** | Blue | `power` | Created, waiting for the first agent check-in |
| **Provisioning** | Amber | `loader-2` (spin) | Agent is running steps now |
| **Healthy** | Green | `circle-check` | Provisioned, agent checked in within the window |
| **Idle-queued** | Teal | `clock` | Healthy, but tasks are waiting the agent has not collected |
| **Agent late** | Amber | `clock-alert` | Was healthy, now overdue for a check-in |
| **Agent missing** | Orange | `plug-zap` | Long overdue; the host is probably gone |
| **Failed** | Red | `circle-x` | A step or builder operation failed |
| **Rebuilding** | Purple | `refresh-cw` (spin) | Being recreated from a new commit or by hand |
| **Destroying** | Slate | `loader` (spin) | Teardown in progress |
| **Torn down** | Muted / outline | `minus-circle` | Removed deliberately |

**Secondary conditions ride as a small corner badge**, so a cell can be Healthy *and*
carry a marker: `bell` for a pending scheduled inject, `lock` when its team's access is
closed, `alert-triangle` when its last step logged a validator warning under
`ignore_errors`. Primary state is the fill; secondary is the corner. Two channels, no
overloading one colour with two meanings.

A cell whose team's access is **closed** is dimmed as a whole, so a locked-out team reads
as a visibly quieter block without losing its per-host states underneath.

**Charts alongside the matrix.** The grid is the picture, but three compact charts frame
it and, crucially, **respond to the current filter and selection** so they describe what
you are actually looking at:

- **State distribution bar** — one horizontal stacked bar above the grid, segmented by
  the states above, that doubles as the **legend** and the count. It reflects the active
  filter: narrow to `prod` and it recounts. This is where "12 failed, 40 late" lives
  without reading four thousand cells.
- **Per-column health** — a thin bar under each column header showing, for that one host
  across every team, the share healthy vs not. A column that is half red is the
  bad-column signal made quantitative, right where the eye already is.
- **Per-row health** — a compact indicator at the end of each team's row, the same idea
  rotated: how much of this team is up. A row that is all red ends in a red cap.

When a **selection** is active, the state-distribution bar switches to summarise the
selection ("512 selected: 470 healthy, 30 late, 12 failed"), so an operator sees the
composition of what they are about to act on before the action, matching the
show-the-blast-radius principle.

Interactions: hover for a summary card; click to select; shift-click and drag for
ranges; click a row or column header to select it entirely; zoom from whole environment
to one team; click through to host detail.

### Build → Hosts (table)

The same data and the same filters, laid out like a cloud provider's instance list.
Columns: name, object, team, network, address, image, size, state, agent last check-in,
steps done/total, public endpoint. Sortable, with column visibility under the user's
control and virtualised rows.

**Selection is shared with the matrix.** Filter and select in one, switch, and the
selection is intact. The expected rhythm is: spot the bad column in the matrix, switch
to the table, read the detail, act.

### Host detail

- **Identity** — object, team, network, address, public endpoint, image, size.
- **Agent** — last check-in, version, queued tasks not yet collected.
- **Steps** — ordered, each with state, duration, validator results, and its **rendered
  script exactly as delivered**.
- **Log** — this host's slice of the event journal.
- **Actions** — run a command, reboot, restart a service, re-run a step, rebuild,
  destroy. Rebuild warns that it recreates the host and loses its state.

### Build → Topology

What the environment resolved into: teams, networks with CIDRs, hosts and containers
with addresses and dependencies. Readable as a tree, with the dependency graph available
for a network or a team when ordering is the question.

### Build → Logs

The event journal as a hierarchy: build → team → host → step → command. Live over SSE
while anything is running, browsable afterwards.

Filters: team, host, network, object, state, time range, free text. A saved default of
**failures only**, because that is the common case.

Each entry shows command, stdout, stderr, exit code, duration, validator results, and
links to the rendered script, the source file, and the commit.

### Build → Access

- Current state per team: open, closed, or overridden, with the countdown.
- The schedule from the environment file, and any manual overrides on top.
- Actions: open or close a team now, extend by a duration, open or close everything.
- Audit trail of every change, with who and when.

Extensions are the common case during an event — a team lost an hour to a technical
problem — so "extend team 4 by 30 minutes" is one control, not a form.

### Build → Schedule

Injects, upcoming and past. Each shows its cron, next fire, which teams, and history.
Manage access can cancel or reschedule, per team or across the environment. A missed
inject that ran late is flagged with how late it was.

### Build → Findings

Findings instantiated in this build, grouped by severity, with the object each came from
and whether it was direct or inherited from a script. Export is an API pull; this screen
offers CSV from the same endpoint.

### Build → Artifacts

Storage by class — agent binaries, rendered scripts, uploaded files, step output — with
sizes and a total. Actions: purge artifacts, delete build, pin against retention
policies. Each states exactly what it removes and what survives.

### People

The user database from this repo's CSVs. Searchable by name, username, department,
title. Shows passwords, because during an event that is the point.

Sourced per CSV, so "who are the domain admins" is a filter rather than a question.

### Admin → People & access

Per repository, who holds read / build / manage / admin. Adding someone means finding
their GitHub identity and choosing a level. The screen states plainly that GitHub
controls who can push and LaForge controls what they can do to a running competition.

### Admin → Builder configs

Hosters: connection, credentials (write-only, never displayed), image map, size map,
public address pool. A **test connection** action, because a bad builder config should
be caught here rather than during a build.

### Admin → Repositories

Add, remove, and set defaults: auto-build (on by default), auto-deploy (follows the
environment's state by default), and the default builder config.

## Core components

| Component | Behaviour |
| --- | --- |
| `StatusBadge` | The state vocabulary from "Cell states" — colour + Lucide icon + label, one definition rendered identically everywhere. |
| `HostCell` | One matrix cell: primary-state fill, secondary corner badge, dimming for closed access, selection state. |
| `StateDistributionBar` | Stacked bar that is both legend and count, recomputed for the current filter or selection. |
| `HealthBand` | The dashboard's top row: stat tiles + charts from server-side aggregates, every element deep-linking into a filtered view. |
| `StatTile` | One big number with label and state colour; the atom of the health band and reusable on Home. |
| `ProgressBar` | Stacked build/host progress (pending → running → provisioned, failed called out), fills live during a deploy. |
| `HostMatrix` | Virtualised grid, team × object, network grouping, zoom, selection. |
| `HostTable` | Virtualised table, sortable, configurable columns, shared selection. |
| `TargetPicker` | Compose filters: object, tag, network, team, search. Used by ad-hoc tasks, teardown, and both host views. |
| `SelectionBar` | Appears when anything is selected. Shows the count, lists available actions, never lets an action run without the count visible. |
| `ImpactDialog` | Confirmation with the full list of what will be touched. Used for destroy, rebuild, bulk commands, and access changes. |
| `LogViewer` | Hierarchical, streaming, filterable, deep-linkable to a single entry. |
| `DiffView` | What a commit would change, at environment/team/network/host level. |
| `RenderedScript` | A script as delivered, with links to source and commit. |
| `AccessCountdown` | Time until the next transition, per-team exceptions visible. |
| `MessageCentre` | Toasts, list, unread state, filters. |
| `StorageBreakdown` | Bytes by class, with cleanup estimates. |
| `EmptyState` | Every list has a real one that explains what would populate it and how. |

## Key flows

### A volunteer's first day

1. Signs in with GitHub. No repositories → empty state naming who to ask.
2. An admin grants **build** on the content repo.
3. They see the repo, its branches, and the environments.
4. They push a branch. Validation runs; the branch shows green or shows file and line.
5. They open the environment, see the resolved preview, press **Build**.
6. The build reaches `planned`. They inspect rendered scripts for a host.
7. Press **Deploy**. Watch the matrix fill in.

Nothing in that path requires reading documentation first, which is the test.

### Authoring loop

Edit in the editor with the language server → `laforge check` locally → push → validation
in the UI → build → inspect the rendered script for the host in question → deploy.

The UI's job here is small and specific: show validation failures with file and line,
and show rendered output. Everything else happens in the editor.

### Operating: something is wrong

1. Dashboard shows failures grouped by cause: "40 hosts: `install-mysql` exit 1".
2. Click through → matrix, pre-filtered to those hosts. It is a column, so it is content.
3. Open one host → the failing step → stderr → the rendered script → the source line.
4. Fix in the repo, push. Validation passes, a new build appears.
5. **Upcoming changes** shows it would rebuild those 40 hosts and nothing else.
6. Deploy. Watch the column go green.

### Operating: end of day one

1. The countdown in the top bar reads 00:12:00.
2. Team 4 reports they lost an hour. Access → extend team 4 by 60 minutes.
3. The countdown now shows the default plus team 4's exception.
4. At the close time, every other team drops. Sessions terminate, not just new
   connections.
5. Overnight, pushes still build. The competition is marked started, so nothing deploys
   on its own.

### Admin: onboarding an event

1. Add the repository, set auto-build and auto-deploy defaults, choose a builder config.
2. Grant levels to the volunteers.
3. Configure a build: branch, environment file, builder config.
4. Deploy when ready; mark the competition started when it begins.

## States and edge cases

Every list, table, and panel specifies four states, and they are part of the component
rather than an afterthought:

- **Loading** — skeletons that match the shape of the content, not spinners.
- **Empty** — what would populate this, and the action that would.
- **Error** — what failed, what to try, and a link to the detail. Never a bare message.
- **Forbidden** — what level this needs and who can grant it. Sections the user cannot
  use are hidden, but a deep link they lack access to explains itself rather than 404ing.

Specific cases worth designing rather than discovering:

| Case | Behaviour |
| --- | --- |
| Build with 0 hosts deployed yet | Matrix shows the shape with every cell pending, not an empty page. |
| Agent never checks in | Distinct from "late": after a threshold it reads *missing*, and the host offers rebuild. |
| Gateway unreachable from the browser | Live updates degrade to polling with a visible banner. Nothing silently stops updating. |
| Two people acting on one build | Actions appear in each other's message centre. Conflicting destructive actions are serialised server-side, and the second sees what the first did. |
| A build whose repo access was revoked | Disappears from listings. A deep link explains rather than erroring. |
| Purged artifacts | Rendered scripts and logs are gone; the screen says so and when it happened, rather than showing an empty viewer. |

## Performance

- **Virtualise everything that can be large**: matrix cells, table rows, log entries.
- **Stream, do not poll, for live state.** SSE from the event journal, with polling as a
  visible fallback.
- **Aggregate server-side.** "Failures grouped by cause" is a query, not four thousand
  rows filtered in the browser.
- **The matrix is the stress case**: 4000 cells, state changing continuously during a
  deploy. Target is smooth interaction while a build runs, and it is worth building
  early against synthetic data rather than discovering the ceiling later.
- **Selection of thousands must stay cheap** — hold predicates, not arrays of IDs, so
  "every database host in every team" is a filter rather than 4000 identifiers.

## Accessibility

- Colour plus icon plus label for every state, always.
- Full keyboard path through selection, filtering, and confirmation.
- The matrix has a table equivalent carrying the same information, which is also why the
  table view is a peer rather than a secondary feature.
- Focus is visible and never trapped; dialogs return focus where they took it.

## Open questions

1. **Matrix ordering.** Hosts across the top need a stable order — definition order,
   alphabetical, or by network then address. Definition order is probably right, since
   it matches how authors think about the file.
2. **Density at 40 teams.** Twenty teams by two hundred hosts is comfortable. Double it
   and the matrix needs either horizontal virtualisation with a minimap or a collapse
   to network-level summary cells.
3. **How much editing happens in the UI.** The plan generates forms from the JSON
   Schema, which implies editing content in the browser. That competes with the
   git-native model and needs a decision: read-only with links to GitHub, or true
   editing that commits.
4. **Watching.** Whether people subscribe to specific builds for notifications, or
   receive everything their access covers.
5. **Mobile.** An operator with a phone at 2am is plausible. Probably: dashboard,
   messages, and access controls work small; the matrix does not.
