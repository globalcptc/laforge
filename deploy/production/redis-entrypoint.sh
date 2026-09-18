#!/bin/sh
set -eu

# Read the Compose-mounted secret as root, then hand off to the official
# image entrypoint so it can chown /data and drop to the redis user.
PASS=$(cat /run/secrets/redis_password)
if [ -z "$PASS" ]; then
	echo "redis password secret is empty" >&2
	exit 1
fi

exec docker-entrypoint.sh redis-server --appendonly yes --requirepass "$PASS"
