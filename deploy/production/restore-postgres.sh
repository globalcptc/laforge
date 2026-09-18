#!/bin/sh
set -eu

compose_file="${COMPOSE_FILE:-docker-compose.prod.yml}"
backup="${1:-}"

if [ -z "$backup" ]; then
	echo "usage: CONFIRM_RESTORE=laforge $0 <laforge-TIMESTAMP.dump>" >&2
	exit 2
fi
if [ "${CONFIRM_RESTORE:-}" != "laforge" ]; then
	echo "Refusing destructive restore; set CONFIRM_RESTORE=laforge" >&2
	exit 2
fi

backup="$(basename "$backup")"
case "$backup" in
	laforge-*.dump) ;;
	*)
		echo "Invalid backup filename: $backup" >&2
		exit 2
		;;
esac

docker compose -f "$compose_file" exec -T db-backup test -f "/backups/$backup"
docker compose -f "$compose_file" stop backend db-backup

restore_services() {
	docker compose -f "$compose_file" start db-backup backend >/dev/null
}
trap restore_services EXIT

docker compose -f "$compose_file" run --rm --no-deps --entrypoint sh db-backup -ec '
	export PGPASSWORD="$(cat /run/secrets/postgres_password)"
	pg_restore \
		--clean \
		--if-exists \
		--exit-on-error \
		--no-acl \
		--no-owner \
		--host=db \
		--username=laforger \
		--dbname=laforge \
		"$1"
' sh "/backups/$backup"

echo "Restored $backup"
