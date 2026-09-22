package incus

import (
	"fmt"
	"strings"
)

// gatewayUserData installs the gateway services and registers a boot unit
// that re-applies the configuration pushed by reconcileGateway.
func gatewayUserData() string {
	return fmt.Sprintf(`#cloud-config
package_update: true
packages:
  - dnsmasq
  - nftables
write_files:
  - path: /etc/systemd/system/laforge-gateway.service
    content: |
      [Unit]
      Description=LaForge team gateway configuration
      After=network-online.target
      Wants=network-online.target

      [Service]
      Type=oneshot
      RemainAfterExit=yes
      ExecStart=/bin/sh -c 'test -x %[1]s && %[1]s || true'

      [Install]
      WantedBy=multi-user.target
runcmd:
  - systemctl daemon-reload
  - systemctl enable laforge-gateway.service
`, gatewayApplyPath)
}

func userData(osName string, agentURL string, password string) string {
	if isWindowsOS(osName) {
		return fmt.Sprintf(`<powershell>
$ErrorActionPreference = 'Stop'
net user Administrator '%s'
New-Item -ItemType Directory -Path "$env:PROGRAMDATA\Laforge" -Force | Out-Null
[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
do {
	$downloaded = $false
	try {
		Invoke-WebRequest -UseBasicParsing '%s' -OutFile "$env:PROGRAMDATA\Laforge\laforge.exe"
		$downloaded = $true
	} catch {
		Start-Sleep -Seconds 5
	}
} until ($downloaded)
& "$env:PROGRAMDATA\Laforge\laforge.exe" -service install
& "$env:PROGRAMDATA\Laforge\laforge.exe" -service start
</powershell>`, powershellSingleQuote(password), powershellSingleQuote(agentURL))
	}

	return fmt.Sprintf(`#!/bin/sh
set -eu
if ! id laforge >/dev/null 2>&1; then
	useradd --create-home --shell /bin/sh laforge
	if getent group sudo >/dev/null 2>&1; then
		usermod --append --groups sudo laforge
	fi
fi
printf 'laforge:%%s\n' '%s' | chpasswd
until curl -fSL -o /laforge.bin '%s'; do
	sleep 10
done
chmod 0755 /laforge.bin
cd /
./laforge.bin -service install
./laforge.bin -service start
`, shellSingleQuote(password), shellSingleQuote(agentURL))
}

func isWindowsOS(osName string) bool {
	osName = strings.ToLower(osName)
	return strings.HasPrefix(osName, "win") || strings.HasPrefix(osName, "w2k")
}

func shellSingleQuote(value string) string {
	return strings.ReplaceAll(value, "'", `'"'"'`)
}

func powershellSingleQuote(value string) string {
	return strings.ReplaceAll(value, "'", "''")
}
