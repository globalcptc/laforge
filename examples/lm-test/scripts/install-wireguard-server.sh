#!/usr/bin/env bash
set -euo pipefail
apt-get update && apt-get install -y wireguard
wg genkey | tee /etc/wireguard/server.key | wg pubkey > /etc/wireguard/server.pub
