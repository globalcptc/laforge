# Running LaForge

This covers standing up LaForge — on your own domain for real use, or locally to try it —
connecting it to GitHub, and installing the VS Code authoring extension. For what LaForge
*is*, see [README.md](README.md); for how to write content, see
[CONFIGURATION.md](CONFIGURATION.md).

- [Prerequisites](#prerequisites)
- [Quick start with Docker Compose](#quick-start-with-docker-compose)
- [The `.env` file](#the-env-file)
- [Addressing (the URLs, explained)](#addressing-the-urls-explained)
- [Certificates](#certificates)
- [The `laforge` CLI](#the-laforge-cli)
- [Connecting GitHub (the GitHub App)](#connecting-github-the-github-app)
- [The VS Code extension](#the-vs-code-extension)
- [Running services directly (without Compose)](#running-services-directly)

---

## Prerequisites

- **Docker + Docker Compose** — the supported way to run the whole stack.
- **Go 1.22+** — to build the `laforge` CLI and the `laforge-lsp` language server (used
  by the VS Code extension), and to run services directly.
- **Node 18+ / npm** — only to build the VS Code extension.
- **openssl** — for the certificate script (`scripts/gen-certs.sh`).

You do **not** need a cloud account or any hoster to run LaForge locally: it ships with a
`fake` builder that simulates deployments, so the whole system runs end to end from
Compose.

---

## Quick start with Docker Compose

From the repository root:

```bash
cp .env.example .env          # then set GITHUB_APP_WEBHOOK_SECRET (any non-empty
                              # value) — laforge-api won't start without it — and
                              # clear PUBLIC_BASE_URL / UI_BASE_URL for a local run
                              # (they default to example.com; empty → localhost)
./scripts/gen-certs.sh localhost   # self-signed CA + gateway mTLS cert (see Certificates below)
docker compose up --build
```

That builds and starts every service. Once it's up:

| Service | URL |
| --- | --- |
| Operator UI | http://localhost:8081 |
| API | http://localhost:8080 |
| Gateway (mTLS, for agents) | localhost:8444 |
| Postgres (for `psql` from the host) | localhost:5433 |

Compose builds the Go services from the multi-stage `Dockerfile`, runs database
migrations (the `migrate` service) automatically, and starts the API, orchestrator,
runner, gateway, and UI. The `.certs/` directory is mounted read-only into the
services that need it.

To stop: `docker compose down` (add `-v` to also drop the database volume).

---

## The `.env` file

`.env` (copied from `.env.example`) is read at startup by the services. It's gitignored —
**never commit real secrets**. The only value required to start is
`GITHUB_APP_WEBHOOK_SECRET`; everything else is optional and controls GitHub integration
and addressing. The file is fully commented; the important groups:

- **GitHub App identity** — `GITHUB_APP_WEBHOOK_SECRET`, `GITHUB_APP_CLIENT_ID`,
  `GITHUB_APP_CLIENT_SECRET`, `GITHUB_APP_ID`, `GITHUB_APP_PRIVATE_KEY_PATH`,
  `GITHUB_APP_SLUG`. See [Connecting GitHub](#connecting-github-the-github-app). A
  deployment can run with no App configured at all, falling back to
  `GITHUB_SERVICE_TOKEN`.
- **Admins** — `LAFORGE_ADMIN_LOGINS`: comma-separated GitHub logins with instance-wide
  admin (needed to approve a repository into LaForge). Empty means nobody can approve
  anything yet — a safe default.
- **Addressing** — the URLs and gateway address the deployment answers on. This is the
  easiest thing to get wrong, so it has its own section: **[Addressing](#addressing-the-urls-explained)**.
- **Certificates** — the mTLS cert paths (`GATEWAY_CA_CERT`, `GATEWAY_SERVER_CERT`,
  `GATEWAY_SERVER_KEY`); see [Certificates](#certificates).

Under Compose, the database connection and gateway CA paths are set by Compose itself;
you mainly touch `.env` for the GitHub App values, admin logins, and the addresses below.

---

## Addressing (the URLs, explained)

LaForge is reached at a few different addresses, and they trip people up because **the
API appears twice** — once for browsers and once for the deployed hosts. Here is the
whole picture. There are two audiences.

**Operators' browsers** talk to the UI and the API:

| Setting | What it is | Example |
| --- | --- | --- |
| `UI_BASE_URL` | the operator console, where you sign in | `https://laforge.example.com` |
| `PUBLIC_BASE_URL` | the HTTP API, as browsers reach it | `https://api.laforge.example.com` |

**Deployed hosts** (the VMs and containers LaForge creates on your hoster) talk to the
gateway and the API:

| Setting | What it is | Example |
| --- | --- | --- |
| `GATEWAY_PUBLIC_ADDR` | the agent gateway — mTLS, a raw `host:port`, **not** a URL | `gateway.example.com:8444` |
| `API_PUBLIC_URL` | the **same API** as `PUBLIC_BASE_URL`, as deployed hosts reach it | `https://api.laforge.example.com` |

The three things that clear up the confusion:

1. **`PUBLIC_BASE_URL` and `API_PUBLIC_URL` are the same API.** There are two settings
   only because browsers and deployed hosts may reach it over different network paths. If
   your hosts reach the API at the same address your browser does, **set them equal.**
2. **`GATEWAY_PUBLIC_ADDR` is a `host:port`, not a URL** — no `https://`. It's a separate
   service from the API, on its own port, speaking mTLS. Its host must be a SAN on the
   gateway certificate (see [Certificates](#certificates)).
3. **The UI and API need not be separate hosts.** Behind one reverse proxy they can share
   an origin, in which case `UI_BASE_URL` and `PUBLIC_BASE_URL` are the same URL.

The agent addresses (`GATEWAY_PUBLIC_ADDR`, `API_PUBLIC_URL`) are only needed when you
deploy to a real hoster with agents. Leave them empty to deploy **without** agents — the
default, and what the `fake` builder uses.

### A worked example

One domain, `example.com`, everything behind a TLS-terminating reverse proxy, deploying
to a real hoster. The complete addressing part of `.env`:

```bash
# what operators' browsers reach
UI_BASE_URL=https://laforge.example.com
PUBLIC_BASE_URL=https://api.laforge.example.com

# what deployed hosts reach (the same API, plus the mTLS gateway on its own port)
API_PUBLIC_URL=https://api.laforge.example.com
GATEWAY_PUBLIC_ADDR=gateway.example.com:8444
```

`PUBLIC_BASE_URL` must **exactly** match the Callback URL on your GitHub App
(`<PUBLIC_BASE_URL>/auth/github/callback`) — see [Connecting GitHub](#connecting-github-the-github-app).

### Cookies across domains

Sign-in uses a session cookie. It works as-is whenever the UI and API share a registrable
domain — the subdomain split above (`laforge.example.com` + `api.laforge.example.com`), or
a single shared origin. **Only** if the UI and API sit on genuinely unrelated domains
(say `laforge.example.com` and `laforge-api.example.net`) set
`LAFORGE_CROSS_DOMAIN_COOKIES=true` and serve both over HTTPS; the cookie then uses
`SameSite=None; Secure`, which browsers require to send it across sites.

### Just trying it locally?

For a local `docker compose` run, clear `UI_BASE_URL` and `PUBLIC_BASE_URL` (they fall
back to `http://localhost:8081` and `http://localhost:8080`) and leave the agent
addresses empty. The [quick start](#quick-start-with-docker-compose) table lists the
local ports.

---

## Certificates

The gateway authenticates agents with mutual TLS, and this is the same setup for
development and production. `scripts/gen-certs.sh` creates a self-signed CA and a gateway
server certificate in `.certs/` (gitignored). It asks for one thing — the address agents
reach the gateway on — and issues the certificate for exactly that name:

```bash
./scripts/gen-certs.sh gateway.example.com   # or an IP, or "localhost" for a local trial
```

(Run it with no argument and it prompts.) It produces two things you set up by hand:

- **`ca.crt` / `ca.key`** — the CA the gateway trusts, and the key the runner uses to
  sign every host's agent certificate at deploy time.
- **`server.crt` / `server.key`** — the gateway's own TLS certificate, signed by that CA.

The address you give must be the host in `GATEWAY_PUBLIC_ADDR` (without the port), because
agents pin the gateway by that name. The certificate carries that single name and nothing
else — no `localhost` or wildcard defaults — so it can't be reused for any other host.

You do **not** generate per-host agent certificates yourself: when agent delivery is
configured, the runner mints one per host automatically and patches it into that host's
agent binary. The CA is the only signing material you manage.

### For production

One thing changes beyond passing your real gateway address: **`ca.key` becomes a real
secret.** In a local run only the gateway reads it; once you deploy with agents the
**runner** also loads it to sign agent certificates, so it's a live signing key — back it
up, restrict its permissions, and never commit it.

Then wire the runner for agent delivery (see the [`.env`](#the-env-file) variables
`GATEWAY_PUBLIC_ADDR`, `API_PUBLIC_URL`, `GATEWAY_CA_CERT`, `GATEWAY_CA_KEY`, and
`AGENT_BASE_DIR`). If any of those are unset, the runner deploys **without** agents.

---

## The `laforge` CLI

The easiest way to get the CLI (for validating content with `laforge check`) is to
**download a prebuilt binary** for your OS from the repository's
[Releases](https://github.com/globalcptc/laforge/releases) page — `laforge-<version>-<os>-<arch>`
for Linux, Windows, and macOS. Rename it to `laforge` (or `laforge.exe`), put it on your
`PATH`, and run `laforge version` to confirm.

Or build and install it from source:

```bash
go install ./cmd/laforge
laforge check ./path/to/your-content-repo
```

`laforge check` runs the same validation the server does on a push — schema checks, all
cross-file checks, and a full render of every script for every host in every team — and
touches no hoster. A clean run means the content will build.

---

## Connecting GitHub (the GitHub App)

Every LaForge deployment registers and owns its **own** GitHub App — nothing in the
codebase names a specific App, organization, or installation. This is the one-time setup
that ties your deployment to your GitHub org.

### 1. Create the App

In the GitHub account or organization you want LaForge tied to:
**Settings → Developer settings → GitHub Apps → New GitHub App.**

| Field | Value |
| --- | --- |
| GitHub App name | Anything unique, e.g. `yourorg-laforge` |
| Homepage URL | Your `UI_BASE_URL`, e.g. `https://laforge.example.com` |
| Callback URL | `<PUBLIC_BASE_URL>/auth/github/callback` — e.g. `https://api.laforge.example.com/auth/github/callback` (must match `PUBLIC_BASE_URL` exactly) |
| Webhook → Active | Checked |
| Webhook URL | `<PUBLIC_BASE_URL>/webhook/github` — e.g. `https://api.laforge.example.com/webhook/github` |
| Webhook secret | Generate one (`openssl rand -hex 32`) and keep it → `GITHUB_APP_WEBHOOK_SECRET` |
| Request user authorization (OAuth) during installation | Checked — makes browser and CLI login work through this same App |
| Enable Device Flow | Checked — required for `laforge login` |
| Where can this GitHub App be installed? | **Only on this account**, unless you want other orgs to install it |

**Repository permissions:** Contents = Read-only, Commit statuses = Read and write,
Metadata = Read-only (automatic).

**Organization permissions:** Members = Read-only. This is what lets LaForge list
everyone with repository access — without it, the installation token can only see org
members whose membership is **public**, so anyone with access purely through a team (and
private membership, the GitHub default) silently won't appear in a repository's access
list. Nothing beyond these — least privilege is the point.

> **Note:** adding or changing a permission after the App is installed doesn't take effect
> until an org owner **approves** the new permission on the installation (GitHub emails
> them / shows a "Review request" under the org's installed Apps). No re-install or
> webhook re-run is needed — LaForge mints a fresh installation token per request.

**Subscribe to events:** Push, Installation, Installation repositories. Then **Create
GitHub App**.

### 2. Collect its identity

From the App's settings page:

- **App ID** (a number) → `GITHUB_APP_ID`
- **Client ID** → `GITHUB_APP_CLIENT_ID`
- **Client secret** (Generate a new one) → `GITHUB_APP_CLIENT_SECRET`
- **App slug** (the part of the App URL after `github.com/apps/`) → `GITHUB_APP_SLUG`
  (only used to build "Install on GitHub" links in the admin UI)
- **Private key** (Generate a private key → downloads a `.pem`) → save it somewhere
  `laforge-api` can read and set `GITHUB_APP_PRIVATE_KEY_PATH` to that path. Treat it like
  any private key: never commit it, restrict its permissions. A convenient spot is
  `.certs/` (gitignored and already mounted into the api container).

### 3. Configure and restart

Set the values above in `.env`, plus `LAFORGE_ADMIN_LOGINS` (your GitHub login), and
restart `laforge-api`. `GITHUB_APP_ID` and `GITHUB_APP_PRIVATE_KEY_PATH` are optional as a
**pair** — set both or neither; setting one without the other fails loudly at startup.
Without an App, everything falls back to `GITHUB_SERVICE_TOKEN` (a personal access token
with `repo` scope).

### 4. Install and approve

From the App's page (`https://github.com/apps/<your-app-slug>`), click **Install**, choose
your org, and select the repositories to install it on. Then sign in to the LaForge UI as
a `LAFORGE_ADMIN_LOGINS` account, open **Installations**, and **Approve** each repository
you want to build from. Installing makes a repo reachable; approving is what makes LaForge
track it.

> **Tip:** choose **Only select repositories** (not **All repositories**) when installing —
> GitHub's install webhook only includes the repository list for the selective form.

From there: configure a build (repo + branch + environment file + builder), push, and
LaForge validates, builds, and — on your go — deploys.

---

## The VS Code extension

The extension (`editors/vscode/`) gives content authors schema-aware autocomplete, hover
documentation, and live `laforge check` diagnostics. It talks to a language server,
`laforge-lsp`, which must be on your `PATH`.

> **Quickest path (prebuilt):** from the
> [Releases](https://github.com/globalcptc/laforge/releases) page, download the
> `laforge-vscode-<version>.vsix` and the `laforge-lsp-<version>-<os>-<arch>` for your OS.
> Install the extension with `code --install-extension laforge-vscode-<version>.vsix`,
> then rename the `laforge-lsp` binary to `laforge-lsp` (or `laforge-lsp.exe`) and put it
> on your `PATH`. Reload VS Code. Building from source (below) is only needed for
> development.

### 1. Install the language server

```bash
go install ./cmd/laforge-lsp
```

This puts `laforge-lsp` on your `GOBIN` (usually `~/go/bin`) — make sure that's on your
`PATH`. The extension runs `laforge-lsp` from your `PATH` by default; you can point it at a
specific binary with the `laforge.serverPath` setting.

### 2. Build the extension

```bash
cd editors/vscode
npm install
npm run compile        # compiles TypeScript into ./out
```

### 3. Install it

**For local development**, open `editors/vscode/` in VS Code and press **F5** to launch an
Extension Development Host with the extension loaded.

**To package and install it as a real extension**, use `vsce`:

```bash
cd editors/vscode
npm install -g @vscode/vsce      # if you don't have it
vsce package                     # produces laforge-<version>.vsix
code --install-extension laforge-*.vsix
```

Reload VS Code. Open any content repository's `.yaml` files and you'll get completion,
hover docs, and diagnostics. If the extension can't find `laforge-lsp`, it offers to run
`go install ./cmd/laforge-lsp` for you and tells you to reload.

---

## Running services directly

You can run any service on the host without Compose (useful for iterating on one):

```bash
# Point at your own Postgres and run migrations first:
go run ./cmd/laforge-migrate "host=/tmp port=5432 user=you dbname=laforge_dev sslmode=disable" up

# Then a service, reading .env from the repo root:
go run ./cmd/laforge-api
go run ./cmd/laforge-orchestrator -db-url "…" -repo /path/to/content
go run ./cmd/laforge-runner       -db-url "…" -repo /path/to/content -id local-runner
go run ./cmd/laforge-gateway
```

`LISTEN_ADDR` is read by both the API (default `:8080`) and the gateway (default `:8443`)
with different built-in defaults, so override it inline per service if you run them from
the same `.env` (e.g. `LISTEN_ADDR=:8081 go run ./cmd/laforge-api`).

---

## Building the release (maintainers)

The `Makefile` produces the downloadable volunteer artifacts into `./dist`:

```bash
make release VERSION=v3.0.0     # binaries + .vsix + SHA256SUMS.txt
```

That cross-compiles the `laforge` CLI and `laforge-lsp` for Linux, Windows, and macOS
(no CGO, no cross toolchain needed), packages the `.vsix`, and writes checksums. Upload
the contents of `./dist` to the GitHub Release.

Pushing a `vX.Y.Z` tag does this automatically: `.github/workflows/release.yml` runs
`make release` and attaches the artifacts to that tag's Release.

