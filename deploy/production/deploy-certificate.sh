#!/bin/sh
set -eu

lineage="${RENEWED_LINEAGE:-/etc/letsencrypt/live/${GRPC_DOMAIN:?GRPC_DOMAIN is required}}"
destination=/run/grpc-tls

mkdir -p "$destination"
cp "$lineage/fullchain.pem" "$destination/fullchain.pem.tmp"
cp "$lineage/privkey.pem" "$destination/privkey.pem.tmp"
chown 10001:10001 "$destination/fullchain.pem.tmp" "$destination/privkey.pem.tmp"
chmod 0644 "$destination/fullchain.pem.tmp"
chmod 0600 "$destination/privkey.pem.tmp"
mv "$destination/fullchain.pem.tmp" "$destination/fullchain.pem"
mv "$destination/privkey.pem.tmp" "$destination/privkey.pem"
