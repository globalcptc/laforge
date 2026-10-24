#!/usr/bin/env bash
# Self-signed CA + gateway server certificate for laforge-gateway's mTLS
# (internal/agentpki, cmd/laforge-gateway). The same setup is used for
# development and production -- the CA is not a dev-only artifact: the
# gateway trusts it, and the runner signs every per-host agent client
# certificate with its key at deploy time (internal/agentdelivery).
# Per-host agent certs are minted automatically; this script produces the
# two things you set up by hand:
#
#   ca.crt / ca.key          the CA the gateway trusts and the runner signs with
#   server.crt / server.key  the gateway's own TLS cert, signed by that CA
#
# You give it ONE thing: the address agents reach the gateway on -- the
# host part of GATEWAY_PUBLIC_ADDR, with no port. The server certificate is
# issued for exactly that name and nothing else (no localhost / wildcard
# defaults), so it can't be repurposed for any other host. Pass it as an
# argument, or the script asks:
#
#   ./scripts/gen-certs.sh gateway.example.com
#   ./scripts/gen-certs.sh 203.0.113.10
#   ./scripts/gen-certs.sh localhost          # a purely local trial
#
# In production ca.key is a real signing secret (the runner uses it): back
# it up, restrict its permissions, and never commit it. .certs/ is gitignored.
set -euo pipefail
cd "$(dirname "$0")/.."

host="${1:-}"
if [ -z "$host" ]; then
  printf 'Gateway address agents will connect to (hostname or IP, no port): '
  read -r host
fi
host="${host%%:*}"   # tolerate a pasted host:port -- keep only the host
if [ -z "$host" ]; then
  echo "error: no gateway address given" >&2
  exit 1
fi

# Exactly one SAN, matching that host: IP:<addr> for a bare IPv4 literal,
# DNS:<name> otherwise.
if printf '%s' "$host" | grep -qE '^[0-9]+(\.[0-9]+){3}$'; then
  san="IP:$host"
else
  san="DNS:$host"
fi

out=.certs
mkdir -p "$out"
cd "$out"

openssl req -x509 -newkey rsa:2048 -nodes -days 3650 \
  -subj "/CN=laforge-ca" -keyout ca.key -out ca.crt

openssl req -newkey rsa:2048 -nodes \
  -subj "/CN=$host" -keyout server.key -out server.csr
openssl x509 -req -in server.csr -CA ca.crt -CAkey ca.key -CAcreateserial \
  -days 3650 -out server.crt \
  -extfile <(printf 'subjectAltName=%s' "$san")

rm -f server.csr
# ca.key is the CA signing secret -- only the runner reads it, and that
# container is root, so keep it 600. The gateway, however, runs as a NON-root
# container user (distroless :nonroot) and must read the server key + the CA
# cert, so those stay world-readable (644) or the gateway can't start.
chmod 600 ca.key
chmod 644 ca.crt server.crt server.key
echo
echo "certs written to $out/ for gateway host: $host ($san)"
echo "ca.key is the runner's signing key -- keep it secret; back it up and restrict it in production."
