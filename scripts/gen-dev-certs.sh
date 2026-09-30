#!/usr/bin/env bash
# Real, self-signed dev CA + server cert for laforge-gateway's mTLS
# (internal/agentpki, cmd/laforge-gateway) -- for local dev and
# docker-compose only. Never commit the output (.dev-certs/ is
# gitignored) and never use these for anything real: a genuine
# deployment issues its own CA and per-team agent certs through the
# agent factory (internal/agentfactory), not this script.
set -euo pipefail
# GATEWAY_EXTRA_SANS: extra comma-separated SANs for the gateway server cert
# (e.g. "IP:192.0.2.10" for an address real agents dial). Needed
# when agents connect to the gateway by an address other than localhost.
cd "$(dirname "$0")/.."

out=.dev-certs
mkdir -p "$out"
cd "$out"

openssl req -x509 -newkey rsa:2048 -nodes -days 3650 \
  -subj "/CN=laforge-dev-ca" -keyout ca.key -out ca.crt

openssl req -newkey rsa:2048 -nodes \
  -subj "/CN=localhost" -keyout server.key -out server.csr
openssl x509 -req -in server.csr -CA ca.crt -CAkey ca.key -CAcreateserial \
  -days 3650 -out server.crt \
  -extfile <(printf "subjectAltName=DNS:localhost,DNS:gateway,IP:127.0.0.1${GATEWAY_EXTRA_SANS:+,${GATEWAY_EXTRA_SANS}}")

rm -f server.csr
echo "dev certs written to $out/:"
ls -la
