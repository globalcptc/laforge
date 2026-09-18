#!/bin/sh
set -eu

interval="${BACKUP_INTERVAL:-86400}"
retention_days="${BACKUP_RETENTION_DAYS:-14}"
password_file="${PGPASSWORD_FILE:-/run/secrets/postgres_password}"

export PGPASSWORD="$(cat "$password_file")"

backup() {
	timestamp="$(date -u +%Y%m%dT%H%M%SZ)"
	output="/backups/laforge-${timestamp}.dump"
	temp="${output}.tmp"

	echo "Creating PostgreSQL backup ${output}"
	pg_dump --format=custom --compress=6 --no-acl --no-owner --file="$temp"
	chmod 0600 "$temp"
	mv "$temp" "$output"
	sha256sum "$output" > "${output}.sha256"
	date +%s > /backups/.last-success

	find /backups -type f \( -name 'laforge-*.dump' -o -name 'laforge-*.dump.sha256' \) \
		-mtime "+${retention_days}" -delete
}

trap 'exit 0' TERM INT

while :; do
	backup
	sleep "$interval" &
	wait "$!"
done
