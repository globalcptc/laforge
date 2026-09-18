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

Enable Azure managed-disk snapshots or Azure Backup for that disk. The Compose
backup service provides logical PostgreSQL dumps, but those dumps still need an
off-host copy to protect against VM or disk loss.

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
10001, `db-backup` as `postgres`). Directory mode 700 still keeps them
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

Load the deployment variables for the certificate bootstrap commands:

```sh
set -a
. ./.env.production
set +a
```

Start Caddy by itself. It obtains the UI certificate and serves the ACME
webroot for the DNS-only gRPC hostname:

```sh
docker compose --env-file .env.production -f docker-compose.prod.yml \
  up -d --no-deps proxy
```

Issue the initial gRPC certificate:

```sh
docker compose --env-file .env.production -f docker-compose.prod.yml \
  run --rm --entrypoint certbot certbot \
  certonly --webroot -w /var/www/certbot \
  --email "$ACME_EMAIL" --agree-tos --no-eff-email \
  --deploy-hook "sh /usr/local/bin/deploy-certificate" \
  -d "$GRPC_DOMAIN"
```

Start the complete stack:

```sh
docker compose --env-file .env.production -f docker-compose.prod.yml \
  up -d --build
docker compose --env-file .env.production -f docker-compose.prod.yml ps
```

Set Cloudflare SSL/TLS mode to **Full (strict)** after Caddy has issued the
certificate. Do not enable Cloudflare caching for `/api/*` or `/auth/*`.

Certbot checks for renewal every 12 hours. Its deploy hook copies the renewed
certificate into a backend-readable volume without exposing Certbot's account
state or private directories to the application. The backend reads that copy
during each new TLS handshake, so renewal does not require rebuilding the
backend or agents.

## Backup and restore

`db-backup` creates a compressed custom-format dump immediately at startup and
then once per `BACKUP_INTERVAL`. Dumps and SHA-256 files are retained for
`BACKUP_RETENTION_DAYS` in the `postgres-backups` volume.

List backups:

```sh
docker compose --env-file .env.production -f docker-compose.prod.yml \
  exec db-backup ls -lh /backups
```

Restore a dump:

```sh
CONFIRM_RESTORE=laforge \
  COMPOSE_FILE=docker-compose.prod.yml \
  sh deploy/production/restore-postgres.sh laforge-YYYYMMDDTHHMMSSZ.dump
```

The restore script stops the backend and backup worker, restores with
`--clean --if-exists`, and starts both services again. Test restores on a
non-production VM before relying on the backup process.

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

Take a fresh PostgreSQL dump and managed-disk snapshot before application or
database upgrades.
