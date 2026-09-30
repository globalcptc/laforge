#!/usr/bin/env bash
set -euo pipefail
{{ range people "employees" }}
useradd -m -s /bin/bash -c "{{ .first_name }} {{ .last_name }}" {{ .username }}
echo '{{ .username }}:{{ .password }}' | chpasswd
{{ if eq .sudo "true" }}usermod -aG sudo {{ .username }}{{ end }}
{{ end }}
