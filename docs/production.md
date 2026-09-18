# Production deployment

This deployment targets one Azure Linux VM with Docker Compose, a static public
IP, Cloudflare DNS, and a separate Azure managed disk for persistent data.

## Network layout

Create these DNS records:

- `laforge.cp.tc`: Cloudflare-proxied `A` record to the Azure public IP.
- `grpc.laforge.cp.tc`: DNS-only `A` record to the same IP.

The gRPC record must remain DNS-only. Cloudflare's normal proxy does not forward
port 50051.

Allow these inbound ports in the Azure NSG:

- TCP 22 from administrative source addresses only.
- TCP 80 and 443 from the internet. Caddy uses port 80 for ACME validation.
- TCP 50051 from the internet for LaForge agents.

The backend also needs outbound access to GitHub, operating-system package
repositories, and the MicroCloud LXD API. Prefer a site-to-site VPN for LXD. If
LXD port 8443 is published instead, restrict it at the remote firewall to the
Azure static IP and continue to require LXD mutual TLS.

## Host preparation

Use a current Ubuntu LTS VM with at least 8 vCPUs, 32 GiB RAM, and a separate
managed disk. Agent compilation and environment builds make CPU and temporary
disk performance more important than they are for a conventional web service.

Enable Redis background saves. Without this, Redis logs a warning and AOF
rewrites can fail under memory pressure:

```sh
sudo sysctl -w vm.overcommit_memory=1
echo 'vm.overcommit_memory = 1' | sudo tee /etc/sysctl.d/80-laforge-redis.conf
```

Mount the data disk and move Docker's data root onto it before starting LaForge:

```sh
sudo mkdir -p /mnt/laforge/docker
sudo tee /etc/docker/daemon.json >/dev/null <<'EOF'
{
  "data-root": "/mnt/laforge/docker",
  "log-driver": "json-file",
  "log-opts": {
    "max-size": "25m",
    "max-file": "5"
  }
}
EOF
sudo systemctl restart docker
```

Enable Azure managed-disk snapshots or Azure Backup for the Docker data-root
disk. That snapshot is the backup: Postgres, Redis, and LaForge state all live
there. There is no in-compose dump sidecar; dumps on the same disk do not
protect against disk loss.

## Configuration and secrets

From the repository root:

```sh
cp .env.production.example .env.production
cp conf.prod.json.example conf.prod.json
mkdir -m 700 secrets
openssl rand -base64 36 > secrets/postgres_password
openssl rand -base64 36 > secrets/redis_password
openssl rand -base64 48 > secrets/session_secret
openssl rand -base64 24 > secrets/admin_password
chmod 600 .env.production
chmod 644 conf.prod.json secrets/*
```

Keep `secrets/` mode 700. The files themselves must be world-readable
because Compose bind-mounts them into non-root containers (backend uid
10001). Directory mode 700 still keeps them
off-limits to other host users.

Edit `.env.production` and `conf.prod.json`. In particular:

- Set `ACME_EMAIL`, `LAFORGE_DOMAIN`, `GRPC_DOMAIN`, and `GITHUB_CLIENT_ID`.
- Put the GitHub OAuth client secret in `secrets/github_client_secret`.
- Set the GitHub OAuth callback to
  `https://laforge.cp.tc/auth/github/callback`.
- Add each production builder to `conf.prod.json` and place its configuration
  under `configs/`.

The UI uses Font Awesome Pro packages. Put the npm registry credentials in
`secrets/npmrc`:

```ini
@fortawesome:registry=https://npm.fontawesome.com/
//npm.fontawesome.com/:_authToken=REPLACE_ME
```

Create `secrets/environment.prod.ts` for the Angular production build:

```typescript
const websocketProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';

export const environment = {
  production: true,
  appVersion: 'v710demo1',
  USERDATA_KEY: 'authf649fc9a5f55',
  isMockEnabled: false,
  apiUrl: 'api',
  graphqlUrl: '/api/query',
  wsUrl: `${websocketProtocol}//${window.location.host}/api/query`,
  isMockApi: false,
  authBaseUrl: '/auth'
};
```

This file is deployment-specific and intentionally ignored by Git. It is passed
to the UI image build as a BuildKit secret. Nothing in it is confidential—the
compiled Angular application is downloaded by every browser.

Secret files are mounted under `/run/secrets`; they are not copied into either
application image. Keep `conf.prod.json`, `.env.production`, and `secrets/`
outside source control.

## First start

Install Caddy on the VM (not in Compose). It terminates TLS for the UI on
443 and for agents on 50051, then reverse-proxies to loopback ports published
by the `ui` and `backend` containers.

```sh
sudo apt-get install -y debian-keyring debian-archive-keyring apt-transport-https curl
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/gpg.key' \
  | sudo gpg --dearmor -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt' \
  | sudo tee /etc/apt/sources.list.d/caddy-stable.list
sudo apt-get update
sudo apt-get install -y caddy
```

Copy the Caddyfile and point systemd at `.env.production` so
`{$LAFORGE_DOMAIN}`, `{$GRPC_DOMAIN}`, and the loopback upstreams expand:

```sh
sudo cp deploy/production/Caddyfile /etc/caddy/Caddyfile
sudo mkdir -p /etc/systemd/system/caddy.service.d
sudo cp deploy/production/caddy-systemd.conf \
  /etc/systemd/system/caddy.service.d/laforge.conf
sudo sed -i "s|/opt/laforge/.env.production|$(pwd)/.env.production|" \
  /etc/systemd/system/caddy.service.d/laforge.conf
sudo systemctl daemon-reload
sudo systemctl enable --now caddy
sudo systemctl reload caddy
```

Leave `laforge.cp.tc` DNS-only until Caddy has issued the certificate, then
set Cloudflare SSL/TLS to **Full (strict)**. `grpc.laforge.cp.tc` stays
DNS-only; Cloudflare's proxy does not forward port 50051. Do not cache
`/api/*` or `/auth/*`.

Start the application stack:

```sh
docker compose --env-file .env.production -f docker-compose.prod.yml \
  up -d --build
docker compose --env-file .env.production -f docker-compose.prod.yml ps
```

Host Caddy should show listeners on `0.0.0.0:80`, `:443`, and `:50051`.
Compose should only publish `127.0.0.1:8080->80` (`ui`) and
`127.0.0.1:15051->50051` (`backend`). A `5432/tcp` entry on `db` with no
`0.0.0.0:` or `127.0.0.1:` prefix is the Postgres image's `EXPOSE` metadata,
not a host port.

## Backup and restore

Snapshot the Docker data-root managed disk. That captures Postgres, Redis,
and LaForge volumes together. Restore is a disk restore, not a logical
`pg_restore`.

## Operations

Inspect health and logs:

```sh
docker compose --env-file .env.production -f docker-compose.prod.yml ps
docker compose --env-file .env.production -f docker-compose.prod.yml logs -f backend
curl --fail https://laforge.cp.tc/
openssl s_client -connect grpc.laforge.cp.tc:50051 \
  -servername grpc.laforge.cp.tc </dev/null
```

Upgrade with:

```sh
docker compose --env-file .env.production -f docker-compose.prod.yml pull
docker compose --env-file .env.production -f docker-compose.prod.yml \
  up -d --build --remove-orphans
```

Take a managed-disk snapshot before application or database upgrades.
