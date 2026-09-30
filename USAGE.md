# Running LaForge

This covers standing up LaForge locally, connecting it to GitHub, and installing the
VS Code authoring extension. For what LaForge *is*, see [README.md](README.md); for how
to write content, see [CONFIGURATION.md](CONFIGURATION.md).

- [Prerequisites](#prerequisites)
- [Quick start with Docker Compose](#quick-start-with-docker-compose)
- [The `.env` file](#the-env-file)
- [Dev certificates](#dev-certificates)
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
- **openssl** — for the dev certificate script.

You do **not** need a cloud account or any hoster to run LaForge locally: it ships with a
`fake` builder that simulates deployments, so the whole system runs end to end from
Compose.

---

## Quick start with Docker Compose

From the repository root:

```bash
cp .env.example .env          # then set at least GITHUB_APP_WEBHOOK_SECRET (any
                              # non-empty value) — laforge-api won't start without it
./scripts/gen-dev-certs.sh    # self-signed mTLS certs for the gateway (dev only)
docker compose up --build
```

That builds and starts every service. Once it's up:

| Service | URL |
| --- | --- |
| Operator UI | http://localhost:5173 |
| API | http://localhost:8080 |
| Gateway (mTLS, for agents) | localhost:8444 |
| Postgres (for `psql` from the host) | localhost:5433 |

Compose builds the Go services from the multi-stage `Dockerfile`, runs database
migrations (the `migrate` service) automatically, and starts the API, orchestrator,
runner, gateway, and UI. The `.dev-certs/` directory is mounted read-only into the
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
- **Addressing** — `PUBLIC_BASE_URL`, `UI_BASE_URL`, and mTLS cert paths
  (`GATEWAY_CA_CERT`, `GATEWAY_SERVER_CERT`, `GATEWAY_SERVER_KEY`).

Under Compose, the database URL and gateway CA paths are set by Compose itself; you only
touch `.env` for the GitHub App values and admin logins.

---

## Dev certificates

The gateway authenticates agents with mutual TLS. `scripts/gen-dev-certs.sh` creates a
self-signed CA and server certificate in `.dev-certs/` (gitignored) for local and Compose
use:

```bash
./scripts/gen-dev-certs.sh
```

If real agents on other machines will dial the gateway by a non-localhost address (a
LAN IP or public hostname), add it as a SAN so the certificate matches:

```bash
GATEWAY_EXTRA_SANS="DNS:gateway.example.com,IP:192.0.2.10" ./scripts/gen-dev-certs.sh
```

These certs are for development only. A real deployment issues its own CA and per-team
agent certificates through the agent factory — never reuse the dev certs.

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
| Homepage URL | Your LaForge UI's URL |
| Callback URL | `<PUBLIC_BASE_URL>/auth/github/callback` (must exactly match `PUBLIC_BASE_URL`) |
| Webhook → Active | Checked |
| Webhook URL | `<PUBLIC_BASE_URL>/webhook/github` |
| Webhook secret | Generate one (`openssl rand -hex 32`) and keep it → `GITHUB_APP_WEBHOOK_SECRET` |
| Request user authorization (OAuth) during installation | Checked — makes browser and CLI login work through this same App |
| Enable Device Flow | Checked — required for `laforge login` |
| Where can this GitHub App be installed? | **Only on this account**, unless you want other orgs to install it |

**Repository permissions:** Contents = Read-only, Commit statuses = Read and write,
Metadata = Read-only (automatic). Nothing else — least privilege is the point.

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
  `.dev-certs/` (gitignored and already mounted into the api container).

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

