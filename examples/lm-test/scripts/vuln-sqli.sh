#!/usr/bin/env bash
set -euo pipefail
mkdir -p /var/www/login
cat > /var/www/login/index.php <<'PHP'
<?php $q = "SELECT * FROM users WHERE user='" . $_GET['user'] . "'"; ?>
PHP
