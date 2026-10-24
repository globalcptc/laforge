#!/usr/bin/env bash
# Poll the tracked branch and, when it has moved on the remote, pull and
# rebuild the docker-compose stack. Run by the laforge-deploy systemd timer
# (see USAGE.md), or by hand. Idempotent: does nothing when there's nothing
# new, so it's safe to run every couple of minutes.
#
#   DEPLOY_BRANCH  branch to track (default: the branch currently checked out)
#   COMPOSE        how to invoke compose (default: "docker compose"; set to
#                  "sudo docker compose" when the service user needs sudo)
set -euo pipefail

cd "$(dirname "$0")/.."
BRANCH="${DEPLOY_BRANCH:-$(git rev-parse --abbrev-ref HEAD)}"
COMPOSE="${COMPOSE:-docker compose}"

git fetch --quiet origin "$BRANCH"
if [ "$(git rev-parse HEAD)" = "$(git rev-parse "origin/$BRANCH")" ]; then
  exit 0   # up to date -- nothing to do
fi

echo "[$(date -Is)] $BRANCH moved: $(git rev-parse --short HEAD) -> $(git rev-parse --short "origin/$BRANCH") -- deploying"
git checkout --quiet "$BRANCH"
# Match the remote exactly. Tracked files are reset; untracked/gitignored
# files (.env, .certs/, ui-dist, ...) are left untouched.
git reset --hard --quiet "origin/$BRANCH"

$COMPOSE up -d --build
echo "[$(date -Is)] deploy complete"
