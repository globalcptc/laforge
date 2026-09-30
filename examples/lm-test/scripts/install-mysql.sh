#!/usr/bin/env bash
set -euo pipefail
apt-get update && apt-get install -y mysql-server
systemctl enable --now mysql
