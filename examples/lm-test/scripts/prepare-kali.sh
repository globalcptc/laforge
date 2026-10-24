#!/usr/bin/env bash
set -euo pipefail
apt-get update && apt-get -y dist-upgrade
echo "Team {{ .build.team }} — Kali VDI" > /etc/motd
