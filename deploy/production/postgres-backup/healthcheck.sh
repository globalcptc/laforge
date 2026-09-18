#!/bin/sh
set -eu

last_success="$(cat /backups/.last-success)"
now="$(date +%s)"
max_age="$(( ${BACKUP_INTERVAL:-86400} * 2 + 3600 ))"

test "$((now - last_success))" -le "$max_age"
