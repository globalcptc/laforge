# LaForge UI

The real React/TypeScript frontend from milestone 8 -- see
`../docs/milestone-8-status.md` for what's built and what isn't.

## Running locally

```bash
npm install
npm run dev
```

Needs `laforge-api` running (see `../cmd/laforge-api`) and reachable at
`VITE_API_BASE_URL` (`.env.development` defaults to
`http://localhost:8080`). Example:

```bash
DATABASE_URL="host=/tmp port=5432 user=<you> dbname=laforge_dev sslmode=disable" \
GITHUB_APP_WEBHOOK_SECRET="dev-secret" \
GITHUB_APP_CLIENT_ID="<real or placeholder>" \
GITHUB_APP_CLIENT_SECRET="<real or placeholder>" \
PUBLIC_BASE_URL="http://localhost:8080" \
UI_BASE_URL="http://localhost:5173" \
go run ../cmd/laforge-api
```

Without a real registered GitHub App, `/auth/github/login` builds a real
redirect but GitHub itself will reject it. To exercise the authenticated
app without one, create an `account` + `session` row directly and set
the `laforge_session` cookie (exactly what a real callback produces) --
see `docs/milestone-8-status.md`'s "real end-to-end proof" section for
the exact steps this was verified with. See `docs/github-app-setup.md`
for registering a real App (permissions, webhook events, and which
value goes in which environment variable, including `GITHUB_APP_ID`/
`GITHUB_APP_PRIVATE_KEY_PATH` for installation-token content fetching
and `LAFORGE_ADMIN_LOGINS` for approving installed repositories).
