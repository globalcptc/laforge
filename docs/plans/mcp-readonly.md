# Plan: Read-Only MCP Server for LaForge

**Status:** design approved 2026-10-31. Phases 1–3 plus the power action (§5.6) implemented, uncommitted: API keys + page (migration `00080`), `internal/mcp` at `/mcp` with the `laforge_mcp` role and `mcp_audit` (migration `00081`), and the full tool set (53 tools, §7) through direct queries and the read bridge (§5.5). Recent Activity on both API Keys pages is built (`GET /auth/api-keys/activity`, `GET /admin/api-keys/activity`). Not started: prod rollout (phase 4), OAuth. **Written:** 2026-10-31, against `main` at `4b7cc9f8`.

Volunteers debugging a live competition (for example build `9531b5b4`, cptc12-central) today get build
state through the CLI, the API and `psql` on the prod box: agent heartbeats, gaps grouped by cluster
member (`deployed_object.instance_location`), ad-hoc task output and object events. This plan gives
their AI agents (Claude Code, Claude Desktop, Cursor and others) that same view over the
[Model Context Protocol](https://modelcontextprotocol.io). Agents can **look and cannot change anything**,
and they never see a credential.

It also adds a **self-service API Keys page**, linked from the top-right user menu. Any LaForge user
can mint, see and revoke their own keys there.

## Contents

1. [Summary of Decisions](#1-summary-of-decisions)
2. [Transport, SDK and Hosting](#2-transport-sdk-and-hosting)
3. [Auth: Personal API Keys](#3-auth-personal-api-keys)
4. [API Keys Page (UI)](#4-api-keys-page-ui)
5. [Read-Only Enforcement](#5-read-only-enforcement)
6. [Secrets](#6-secrets)
7. [Tool Set (As Built)](#7-tool-set-as-built)
8. [Rate Limiting and Cost](#8-rate-limiting-and-cost)
9. [Audit Logging](#9-audit-logging)
10. [Database Changes](#10-database-changes)
11. [Phases](#11-phases)
12. [Existing Issues Found While Surveying](#12-existing-issues-found-while-surveying)
13. [Open Decisions](#13-open-decisions)

## 1. Summary of Decisions

| Area | Recommendation |
|---|---|
| SDK | Official `github.com/modelcontextprotocol/go-sdk` (pinned at v1.8.0, released 2026-09-04) |
| Transport | Streamable HTTP, **stateless** mode, JSON responses (no long-lived SSE) |
| Hosting | Inside `laforge-api`, at `https://api.laforge.cp.tc/mcp`, as its own package `internal/mcp` with its own database pool |
| Auth | Personal API keys (`lfk_…`), created on a self-service page, read-only scope, mandatory expiry, revocable |
| Actions | **One: power** (start/stop/reboot, force or not) on hosts and containers, for keys whose owner has **manage** on the repository -- the same rule as the UI. No new key scope. Rebuild, deploy, teardown and commands stay UI/CLI-only (§5.6) |
| Read-only guarantee | Two paths, both structural: direct queries run as a Postgres role `laforge_mcp` with SELECT-only, column-level grants; everything else goes through a read bridge that holds only the API's read pages and caps the caller at read level (§5.5) |
| Secrets | **Decided 2026-10-31:** MCP shows everything a read-level user sees in the UI: host passwords and credentials, rendered scripts, cloud-init, vars, step and task payloads, people. It still never shows what the API shows no one (builder and registry credentials, sessions and GitHub tokens, agent download tokens, the password-generator secret) |
| Cost | Per-key token bucket, a global concurrency cap, `statement_timeout` on the role, hard row and byte caps on every tool |
| Audit | `mcp_audit` table, one row per tool call, shown on the API Keys page |

## 2. Transport, SDK and Hosting

### SDK

There are two maintained Go SDKs:

- **`github.com/modelcontextprotocol/go-sdk`**: the official SDK, maintained by the MCP organisation
  together with Google. v1.7.0 was released 2026-07-28 with full support for the 2026-07-28 protocol revision.
  It ships an `auth` package (`auth.RequireBearerToken` with a `TokenVerifier`, `TokenInfo` reaching tool
  handlers through `req.Extra.TokenInfo`, and RFC 9728 protected-resource metadata), plus
  `mcp.NewStreamableHTTPHandler` with `StreamableHTTPOptions{Stateless: true}`.
- `github.com/mark3labs/mcp-go`: a community SDK that came earlier and is widely used.

**Recommendation: the official go-sdk.** It tracks the spec first, has the auth middleware we need, and
its typed tool handlers (`mcp.AddTool` with Go input/output structs) generate the JSON schemas for us.
Pin an exact version in `go.mod`. Before building, confirm the `TokenVerifier` signature against that
version, because it changed between v0.3 and v1.1.

### Transport

Use streamable HTTP in **stateless** mode, with tool results returned as plain JSON responses:

- Every tool here is a single request and response. Nothing needs server-to-client streaming, sampling
  or a long-lived session.
- Stateless mode is required for 2026-07-28 clients anyway. It also leaves no per-session state on a 2-CPU box,
  and Cloudflare's ~100 s proxy timeout never comes into play.
- `Mcp-Method` and `Mcp-Name` headers, which the SDK now copies from the body, make the tool name
  visible in Caddy logs without parsing bodies.

stdio is not offered. A volunteer's agent connects over the network with a key. A local stdio
wrapper adds nothing over `claude mcp add --transport http`.

### Hosting: in `laforge-api`, not a new binary

| | In `laforge-api` at `/mcp` (recommended) | Separate `cmd/laforge-mcp` |
|---|---|---|
| Deploy | Nothing new: Caddy and Cloudflare already route `api.laforge.cp.tc` → `api:8080` | New compose service, Caddy route, image |
| Shared pieces | Reuses `authCache`, level resolution, `isInstanceAdmin`, the GitHub App installation token | Must re-implement level resolution and needs the App private key |
| Isolation | Process isolation is lost, but the guarantee in §5 lives in Postgres, not in the process | The process has no write credentials at all |
| CPU | Shares the API's 2 CPUs, bounded by §8 | Same box, same CPUs; no real gain |

The structural guarantee (§5) comes from the database role and from the package boundary, not from the
process boundary. So the extra service buys little. `internal/mcp` is written so it *could* run alone:
it takes its dependencies as interfaces (§5.2). If we ever want a separate process, the split is a new
`main`, not a rewrite.

Mounting: `ServeHTTP` sends `/mcp` to the MCP handler **before** the REST mux, so REST logging,
`forgetAccess` and `forgetBuild` never run for it, and REST handlers can never be reached through it.

Cloudflare: add a WAF/Bot-Fight skip rule for `api.laforge.cp.tc/mcp*`. Cloudflare already blocks
Python urllib's default User-Agent (error 1010). MCP clients send their own User-Agents, and some
will be caught the same way. The bearer key is the real gate on this path.

## 3. Auth: Personal API Keys

### Why not GitHub tokens

The API accepts a GitHub session cookie or a bearer GitHub token (`authSessionForRequest`, `auth.go`).
Handing an agent a GitHub token is the wrong shape, for three reasons:

- The OAuth scope is `repo read:user`, so the token can read and **write** every repository the volunteer can.
- It expires in about 8 hours and refreshes through the CLI's credential file.
- It is not scoped to LaForge, nor to reading.

### The key

- Format: `lfk_` + 32 random bytes, base62 encoded. The prefix lets secret scanners spot a leaked key
  (we can register it with GitHub secret scanning later), and lets the API refuse keys on REST routes (§5.3).
- Storage: only `sha256(key)` is stored, as the `session` table already does (`hashToken`, `oauth.go`).
  The plaintext is shown **once**, at creation.
- Fields: owner account, a name ("Claude on my laptop"), the display prefix (`lfk_ab12…`), scope, an
  optional repository allow-list, `expires_at` (**required**: 7, 30 or 90 days, never "no expiry"),
  `last_used_at`, `revoked_at`.
- Scope in v1: only `read`. The column exists so a future scope cannot be confused with `read`. See
  §13 on whether keys should ever work for the CLI.

### What a key can see

A key's level on a repository is **`min(read, owner's effective level)`**, computed live, and only on
the repositories in its allow-list if it has one:

1. Instance admin (`isInstanceAdmin` on the owner's login) → read on every repository.
2. Otherwise, a `repository_access` row for the owner wins (raise, lower or `none`), as in `effectiveLevel`.
3. Otherwise, the owner's GitHub permission on the repository. **This is the one real difference:**
   `effectiveLevel` asks GitHub using the caller's *own* GitHub token (`sess.GithubToken`), and a key
   has none. Instead, ask with the repository's **App installation token**:
   `GET /repos/{owner}/{repo}/collaborators/{login}/permission`. This needs a small
   `ghclient.GetCollaboratorPermission`; `ListCollaborators` already exists. If GitHub can't answer,
   the key gets `none` for that repository, not a cached guess.

Answers are cached in the existing `authCache` levels map (15 s TTL). Losing GitHub access, losing a
`repository_access` grant or being removed as instance admin therefore takes effect within 15 seconds,
with no key-side action needed.

### Verifying a request

`auth.RequireBearerToken` wraps the MCP handler with a `TokenVerifier`:

1. Hash the bearer value and look up `api_key` by `token_hash` (cached about 15 s, like sessions).
2. Reject if not found, revoked, expired, or the owner account is gone. The SDK writes a 401 with a
   `WWW-Authenticate` header.
3. Put `{key id, account id, login, scope, repository allow-list}` in `TokenInfo.Extra`, and the
   account id in `TokenInfo.UserID`.
4. Bump `last_used_at` at most once a minute per key, to avoid a write per call.

### Revocation

- The owner revokes on the API Keys page: `revoked_at` is set, and the verifier cache for that hash is cleared.
- An instance admin can list and revoke anyone's keys on `/admin/api-keys`. This is the incident lever.
- Expiry is enforced on every request.
- No delete. Revoked and expired rows stay, so audit rows keep pointing at something. They are hidden
  from the page by default.

### Later: OAuth

Claude.ai custom connectors and some hosted clients expect MCP's OAuth 2.1 flow, with dynamic client
registration, instead of a pasted header. The SDK's protected-resource metadata handler makes it
possible to add later: LaForge acts as the authorization server, the GitHub sign-in is the login step,
and the result is an `api_key` row. Out of scope for v1. Claude Code, Claude Desktop (through
`mcp-remote`) and Cursor all accept a bearer header today.

## 4. API Keys Page (UI)

**Link:** a new "API Keys" item (with the `KeyRound` icon) in `UserMenu` (`ui/src/components/UserMenu.tsx`),
between the timezone picker and Sign Out. It is visible to every signed-in user.

**Route:** `/settings/api-keys` in `ui/src/router.tsx`.

**Page:**

- Title **API Keys**, with a one-line explanation: keys let an AI agent read your builds through
  LaForge's MCP server, with no access to passwords and no ability to change anything.
- **Create API Key** button (primary, accent blue). It opens a modal with:
  - Name (required).
  - Expires In: 7 / 30 / 90 days, default 30.
  - Limit To Repositories: optional multi-select from the repositories the user can see.
- After creation, a **Copy Your New Key** modal shows the key once, with copy buttons for:
  ```bash
  claude mcp add --transport http laforge https://api.laforge.cp.tc/mcp --header "Authorization: Bearer lfk_…"
  ```
  and the equivalent JSON block for `.mcp.json` / Claude Desktop. A warning says the key will not be shown again.
- Table **Your Keys** with columns Name, Key (`lfk_ab12…`), Repositories, Created, Expires, Last Used, and
  a **Revoke** button (danger red, with a confirm). Expired and revoked keys are behind a "Show Inactive" toggle.
- Section **Recent Activity**: the last 50 `mcp_audit` rows for the user's keys (time, key, tool, build, status, duration).

**Admin:** `/admin/api-keys` (linked with the other admin pages) has every user's active keys, with Owner
and Last Used columns and Revoke.

**REST endpoints.** These are session-cookie-only: an API key can never create, list or revoke keys.

| Method | Path | Who |
|---|---|---|
| GET | `/auth/api-keys` | The signed-in user's own keys |
| POST | `/auth/api-keys` | Create; returns the plaintext once |
| DELETE | `/auth/api-keys/{id}` | Revoke your own key |
| GET | `/auth/api-keys/activity` | Your own recent `mcp_audit` rows (`?limit`, default 50, max 200) |
| GET | `/admin/api-keys/activity` | Instance admin: everyone's recent `mcp_audit` rows |
| GET | `/admin/api-keys` | Instance admin: all keys |
| DELETE | `/admin/api-keys/{id}` | Instance admin: revoke any key |

## 5. Read-Only Enforcement

"We only registered read tools" is not enough. The survey found GET handlers that **write**:
`handleRenderObject` and `handleObjectCloudInit` call `db.RenderSecretForBuild`, which runs
`EnsureBuildRenderSecret` (an INSERT), and `authSessionForRequest` upserts `account` rows. A tool built
on "reuse the GET handler" would inherit those writes. The guarantee therefore has four layers, and the
first is the one that holds even if the Go code is wrong.

### 5.1 A Postgres role that cannot write (the load-bearing layer)

This follows the `laforge_gateway` precedent (`migrations/00004_agent_tables.sql`): a role whose grant
surface *is* the security boundary, with a test that connects as that role and asserts Postgres
rejects forbidden queries.

- `CREATE ROLE laforge_mcp LOGIN`. The password comes from `.env` in prod, like the gateway's.
- `GRANT SELECT (<explicit column list>)` on the whitelisted tables only (§10). These are
  **column-level grants**, so secret columns (`environment.root_password`, `agent_task.payload`, …) are
  not granted at all. A `SELECT *` or a stray join on them fails with `permission denied`.
- The only writes granted: `INSERT` on `mcp_audit`, and `UPDATE (last_used_at)` on `api_key`.
- `ALTER ROLE laforge_mcp SET statement_timeout = '5s'` and a low connection limit (§8).
- `internal/mcp` gets its **own** `pgxpool`, opened with `MCP_DATABASE_URL` as `laforge_mcp`. It is never
  handed `Server.Pool` or the main `*db.Queries`.

Because `SELECT *` fails under column grants, MCP queries live in their own sqlc file,
`internal/db/queries/mcp.sql`, with explicit column lists and `Mcp…` names. That is also a review
forcing function: every column an agent can see is listed in one file.

### 5.2 Package boundary

- `internal/mcp` does not import `internal/api`, `internal/runner`, `internal/builder/*` or
  `internal/gateway`. So there are no builder clients (no live `Inspect` or `Stats` calls), no gateway
  connection and no GitHub write client. A dependency test (`go list -deps`) fails the build if this changes.
- The two things it needs from the API are passed in as narrow interfaces:
  - `LevelResolver.ReadableRepos(ctx, accountID) ([]uuid, error)`, which wraps §3's rule and `authCache`.
  - `Redactor.For(ctx, buildID) (func(string) string, error)`, see §6.2.

### 5.3 Keys can't reach REST, and REST credentials can't reach MCP

- `bearerToken`/`authSessionForRequest` refuse any token starting with `lfk_` with a clear 401 ("API
  keys only work with /mcp"). It is never sent to GitHub.
- The MCP verifier accepts only `lfk_` keys. Session cookies and GitHub tokens are refused, so a
  browser session can't be ridden into `/mcp` either. CORS is not enabled on `/mcp`.

### 5.4 Tool hygiene

- Every tool is registered with `ToolAnnotations{ReadOnlyHint: true, OpenWorldHint: false}`.
- Every tool resolves `build_id` → repository and checks it against `ReadableRepos` before querying.
  A shared helper does this, so no tool can skip it, and a test walks every registered tool with a key
  that has no access and asserts each one refuses.
- No tool takes free-form SQL, a URL, or a path.

### 5.5 The read bridge (as built)

Most tools return an API read page instead of re-implementing it. `api.MCPAccess.Get` runs a GET
in process through **`readMux`**, a router that holds only the read routes listed in
`internal/api/mcp_bridge.go`, never the REST router. The request carries the key owner as its
identity through a context value that no HTTP client can set, and:

- `sessionFromRequest` returns that identity, with no GitHub token;
- `effectiveLevel` resolves the owner's level the way §3 describes (grant, then GitHub collaborator permission through the installation token);
- `requireLevel` refuses anything above `levelRead`, so a page gated at build, manage or admin answers 403 even for a repository admin;
- only GET is ever issued, and `readMux` has nothing else registered.

Instance-admin read pages (builder details, image build logs, installations) work for an
instance admin's key, as they do for that admin's session. Before calling a page, the MCP tool
checks the build or repository against the key's own readable set, which is narrower than the
owner's when the key is limited to some repositories. `TestMCPBridgeReadsOnlyAndCapsAtRead`
covers all of this: actions, live calls and streams are unreachable, a repository admin is capped
at read, and the build is unchanged afterwards.

These pages run on the API's normal database pool, not the `laforge_mcp` role. Their read-only
guarantee is the route allowlist and the level cap, not Postgres grants. One listed page does
write: `render`/`cloud-init` create the build's password-generator secret on first use, exactly
as the UI's own visit would.

### 5.6 The one action: power (as built)

**Decided 2026-10-31:** keys can power hosts, tied to the owner's existing permissions. No new
scope is needed: if they can manage the repository, they can power its hosts, exactly as with the
UI's power buttons. **Rebuilds are not offered**; they stay in the UI and CLI, as do deploy,
teardown, ad-hoc commands, restart-from-step and revalidate.

- `power` (build, action `start`/`stop`/`reboot`, `force`, and targets: `objects` by name or id,
  or `team`/`kind`/`search`/`network`/`tags`, or `all=true` for the whole build, which must be
  given explicitly) calls `api.MCPAccess.Act`.
- `Act` runs `POST /builds/{id}/power` through **`actionMux`**, a router holding that single route,
  as the key's owner at their **real** level (the identity's `action` flag lifts the read cap).
  `handlePowerAction`'s own `requireLevel(levelManage)` decides. It also does the same
  after-change bookkeeping as `ServeHTTP` (clearing the access and shared-read caches) and logs
  the action.
- Networks and DNS records can't be powered; LaForge's power targeting already skips them, and
  the tool refuses them up front. The action runs in the background, exactly as from the UI:
  per-host `power.*` events, with the build log naming the caller as `<login> (via MCP)`.
- The tool is annotated destructive (not read-only), so clients like Claude Code ask before each
  call. It has its own limit on top of the general one: a burst of 3, then one call every 10 seconds per owner.
- The database role is unchanged and still can't write. Power runs on the API's normal pool,
  like the UI's.

`TestMCPActionBridgePowerOnly` covers it: read and build levels get 403, manage gets 202, and
rebuild, deploy, teardown, tasks, restart-steps, revalidate and team access are unreachable through
the action bridge, even for an instance admin.

## 6. Secrets

**Decided 2026-10-31 (supersedes the first draft's "no secrets tier"):** API keys are for the same
people who can use the API, so MCP shows exactly what a read-level person sees in the UI. That
includes host admin passwords and every account LaForge set (`list_credentials`, `get_object_infra`,
`export_external_access`), rendered scripts (`render_object`), cloud-init with its temporary login,
authored vars and env (`get_object_config`), step, ad-hoc and scheduled-task payloads, and people CSVs.
No redactor runs, since there is nothing to hide from these readers.

What stays out is what the API never shows anyone. The role is not granted it, and no bridged page returns it:

| Data | Where it lives |
|---|---|
| Builder credentials (Incus/LXD client keys, the AWS secret key) | `builder_credential` |
| Registry passwords | `registry_credential.secret` (`list_registry_credentials` shows host and username) |
| Sessions and the GitHub tokens in them | `session` |
| Agent download tokens and binaries | `agent_artifact` |
| The build's password-generator secret | `build_render_secret` |
| Runner task payloads (builder specs, which carry agent tokens) | `task.payload` |

## 7. Tool Set (As Built)

54 tools: 53 read-only, plus `power` (§5.6), marked destructive. Builds are named by full id or a unique prefix (`9531b5b4`).
Hosts and containers are named by id, id prefix or name, with `team` when several teams have one.

**Direct queries (role `laforge_mcp`, `internal/db/queries/mcp.sql`).** These answer questions no
single API page does:

| Tool | What it answers |
|---|---|
| `whoami`, `list_repos` | The key, its owner and expiry; the readable repositories |
| `list_builds`, `get_build`, `list_objects` | Builds; a one-build summary with counts by status, failure kind, agent health and **cluster location**; objects with filters (team, kind, status, agent status, location, name) |
| `heartbeat_gaps` | Silences between check-ins in a window, longest first, rolled up per cluster member, plus agents silent right now |
| `search_agent_tasks` | Steps, ad-hoc commands and scheduled runs across a whole build, by status, ad-hoc, command, team and time, with payload, error and output tail |
| `get_step_output` | One task's full output, paged |
| `list_runner_tasks` | The runner's deploy/destroy/power queue, including stuck or retrying work |
| `list_shell_sessions` | Who opened a terminal on which host, and when |
| `list_content_revisions` | Ingested commits with their validation errors |

**API read pages (through the read bridge, §5.5).** Each tool is one page, so it returns the same
JSON the UI shows:

| Area | Tools (API route) |
|---|---|
| Overview | `home` (`/home`) |
| Repositories | `list_configured_builds`, `get_configured_build`, `list_revision_environments`, `list_people`, `list_repo_access` (instance admins), `my_repo_access` |
| Build | `get_build_detail`, `get_dashboard`, `get_topology`, `list_build_events` (logs), `events_summary`, `list_findings`, `list_external_access`, `export_external_access`, `list_credentials`, `list_validation_reports`, `get_validation_report`, `list_scheduled_tasks`, `list_scheduled_task_runs`, `get_artifacts`, `get_upcoming` |
| Host / container | `list_steps`, `list_object_events`, `list_object_heartbeats`, `get_object_infra`, `get_object_config`, `render_object`, `get_cloud_init`, `list_object_validation_runs` |
| Builders | `list_builders`, `list_builder_configs`, `get_builder_config`, `get_builder_stats` (window up to 24h), `get_builder_usage`, `list_image_builds`, `get_image_build_log` (admins) |
| Instance | `list_instance_admins`, `list_installations`, `list_unapproved_repositories`, `list_registry_credentials` (admins), `get_cert_status` |

Every page tool takes `max_field_chars` (default 4000, at most 200 000). Longer strings, such as a
rendered script or a step's output, are cut with a note saying how to ask for more.

**Not reachable, on purpose:**
- Live builder calls: `/drift`, `/readiness` and available images. They walk the hypervisor and need build or manage level anyway.
- Live registry and GitHub browsing (registry images, branches, environment files).
- Streams (`/live`, terminals).
- Every action.

## 8. Rate Limiting and Cost

The prod box has 2 CPUs. `agent_heartbeat` holds millions of rows (72 h retention, `cmd/laforge-orchestrator`
`-heartbeat-retention`), and the API has **no rate limiting today**. Agents loop, so the MCP path has to be
bounded on its own.

- **Per key:** a `golang.org/x/time/rate` token bucket, 2 requests/s with a burst of 10. Over the limit,
  the call returns a tool error ("rate limited, retry in Ns"), not an HTTP 429, so the agent sees and backs off.
- **Per account:** the same bucket shared across a user's keys, so ten keys don't mean ten times the load.
- **Global:** a semaphore of 3 concurrent MCP tool calls. Excess calls wait up to 2 s, then fail. The UI's
  own requests never queue behind agents.
- **Database:** the `laforge_mcp` pool has max 3 connections. The role has `statement_timeout = 5s`.
  Postgres `max_connections` has headroom (it was raised for runner concurrency).
- **Query shape:**
  - Every list query has a hard `LIMIT`.
  - Heartbeat tools require a build or object and a window of at most 6 h, always on
    `agent_heartbeat_object_time_idx`. Never the dashboard's unbounded `ListAgentHeartbeatsByBuild`.
  - Event tools use `event_build_idx` / `event_deployed_object_id_idx`.
  - Each new query in `mcp.sql` gets an `EXPLAIN` against `laforge_test`, with a prod-sized heartbeat
    table, before merge.
- **Response size:** results are capped at 64 KB of JSON. Longer results are truncated with
  `"truncated": true` and a cursor. This also keeps agents' context usage sane.
- **Caching:** a 2 s per-(tool, args, build) cache like `sharedread.go`, so several agents watching one
  build share the work.

## 9. Audit Logging

- **`mcp_audit`**, one row per tool call:
  - `api_key_id`, `account_id`, `tool`, `args` (JSON, capped at 2 KB), `build_id` (when the tool has one)
  - `status` (`ok` / `denied` / `error` / `rate_limited`), `error`, `duration_ms`, `result_bytes`
  - `remote_addr` (Cloudflare's `CF-Connecting-IP`), `user_agent`, `created_at`
- Written asynchronously through a small buffered channel, so a slow insert never delays a tool call.
  If the buffer is full the row is dropped and counted, never blocking.
- **Retention:** 30 days, pruned by the orchestrator next to the heartbeat cleanup.
- **Visible on:**
  - The API Keys page: your own activity (§4).
  - `/admin/api-keys`: all keys.
  - The API log: a line per call, in the same format `ServeHTTP` uses for writes:
    `mcp: <tool> build=<id> -> ok in 41ms (by <login> key lfk_ab12… from <ip>)`.
- Denied calls are audited too. A key probing builds it can't see should be visible.

## 10. Database Changes

One migration (next free number; currently `00080`):

```sql
CREATE TABLE api_key (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  account_id uuid NOT NULL REFERENCES account(id) ON DELETE CASCADE,
  name text NOT NULL,
  prefix text NOT NULL,                 -- 'lfk_ab12' for display
  token_hash text NOT NULL UNIQUE,      -- sha256 hex
  scope text NOT NULL DEFAULT 'read' CHECK (scope IN ('read')),
  repository_ids uuid[],                -- NULL = every repo the owner can read
  created_at timestamptz NOT NULL DEFAULT now(),
  expires_at timestamptz NOT NULL,
  last_used_at timestamptz,
  revoked_at timestamptz
);
CREATE INDEX api_key_account_idx ON api_key (account_id);

CREATE TABLE mcp_audit ( ... per §9 ... );
CREATE INDEX mcp_audit_key_time_idx ON mcp_audit (api_key_id, created_at);
CREATE INDEX mcp_audit_time_idx ON mcp_audit (created_at);

CREATE ROLE laforge_mcp LOGIN PASSWORD '<dev-only>';
ALTER ROLE laforge_mcp SET statement_timeout = '5s';
ALTER ROLE laforge_mcp CONNECTION LIMIT 5;
```

Grants (column lists abbreviated; the migration spells out every column):

| Table | Grant |
|---|---|
| `repository`, `configured_build`, `build`, `team`, `content_revision` | SELECT on non-secret columns |
| `deployed_object` | SELECT on all columns (none is secret) |
| `agent_session` | SELECT, without `cert_fingerprint` |
| `agent_heartbeat` | SELECT, without `cert_fingerprint` |
| `agent_task` | SELECT, **without `payload`** |
| `validator_result`, `validation_run` | SELECT |
| `event` | SELECT |
| `external_access` | SELECT |
| `scheduled_task` | SELECT, **without `payload`** |
| `environment` | SELECT (`id`, `name`, `content_revision_id`, access-window columns), **without `root_password`** |
| `builder_config` | SELECT (`name`, `kind`, `color`) only |
| `builder_metric_sample` | SELECT |
| `account` (`id`, `github_login`) | SELECT, to name a key's owner. **As built:** level resolution runs API-side (`api.Server.MCPAccess`, on the privileged pool), so the role is *not* granted `repository_access` or `instance_admin` |
| `api_key` | SELECT (only the hash is stored, so no column is secret), UPDATE (`last_used_at`) |
| `mcp_audit` | INSERT |
| Everything else | **Nothing**: `session`, `builder_credential`, `registry_credential`, `build_render_secret`, `agent_artifact`, `person`, `people_source`, `task`, `shell_session`, `github_installation`, … |

**Deploy note:** nothing manual. Compose sets `MCP_DATABASE_URL` inline, like the gateway's connection, and
the `migrate` service creates the role on `up`. Its password is exposed exactly as much as `laforge`'s and
`laforge_gateway`'s are.

## 11. Phases

1. **Keys and page.** Migration, `api_key` CRUD endpoints, `lfk_` rejection on REST, API Keys page +
   `UserMenu` link, admin list. Useful on its own and testable before MCP exists.
2. **MCP skeleton.** `internal/mcp` with the go-sdk, a stateless handler at `/mcp`, the verifier,
   the `laforge_mcp` pool, rate limits, audit, plus:
   - `whoami`, `list_repos`, `list_builds`, `get_build`, `list_objects`.
   - The role test (connect as `laforge_mcp`, assert INSERT/UPDATE/DELETE and `root_password` /
     `payload` reads are rejected).
   - The dependency test and the every-tool-refuses-without-access test.
3. **Debugging tools.** `agent_health`, `heartbeat_gaps`, events, steps + `get_step_output` with the
   redactor, `list_adhoc_tasks`, validation, external access, builder stats and usage.
4. **Prod.** Cloudflare skip rule, `.env` role password, a volunteer trial on a live build, and tuning of
   limits from `mcp_audit` timings.
5. **Later, if wanted.** OAuth for hosted connectors, a `debug_build` prompt, findings, image build logs.

## 12. Existing Issues Found While Surveying

These are not part of this plan, but MCP design surfaced them. Each is worth its own change:

- **REST reads with no access check.** `handleGetRepo`, `handleListConfiguredBuilds` and
  `handleGetConfiguredBuild` check nothing. `handleListRepos` returns *every* repository to any caller
  without a session cookie, including anonymous ones, by design ("owner/name isn't sensitive").
  Configured builds expose branch, environment and builder names to the internet.
- **Passwords at `levelRead`.** `/external-access/export`, `/credentials`, `/objects/{id}/infra` and
  `/people` hand every host password, plus the people CSVs, to anyone with GitHub *read* on the content
  repo. For a private content repo that is a small group. It is still the lowest level, and these
  arguably belong at `levelBuild`.
- **Payloads on scheduled-task reads.** `/scheduled-tasks` and `/scheduled-tasks/{id}/runs` return the raw
  `payload`, which includes ad-hoc `set_password` arguments, at `levelRead`.
- **Unbounded reads.** `ListEventsByDeployedObject`, `ListBuildsByRepository` and the dashboard's
  `ListAgentHeartbeatsByBuild` have no `LIMIT`. The dashboard one scans 72 h of heartbeats for every
  object in the build; the 2 s shared cache softens it.

## 13. Open Decisions

1. **Keys for the CLI too?** The user asked for self-service API keys. This plan makes v1 keys MCP-only
   and read-only. A later `scope: 'cli'` could replace the CLI's GitHub-token credential file, but that
   is a write-capable credential and should be its own design. Recommend: MCP-only for now.
2. **Who can create keys?** Recommend any signed-in user. A key only ever sees what its owner can read.
   The alternative is to restrict creation to people with at least read on some repository.
3. **Expiry ceiling.** 90 days maximum? A competition weekend fits in 7.
4. **Content-backed step names** through a narrow `ContentReader` (recommended), or no step names in v1.
5. **Secrets.** Decided: everything a read-level user sees (§6).
6. **Instance-admin keys.** Should an admin's key read everything (as their session does), or should
   admin keys require a repository allow-list? Recommend: allowed, but the page nudges admins to scope them.
