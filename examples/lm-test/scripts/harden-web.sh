#!/usr/bin/env bash
set -euo pipefail
echo "server_tokens off;" >> /etc/nginx/nginx.conf
rm -f /var/www/html/index.nginx-debian.html
systemctl reload nginx
