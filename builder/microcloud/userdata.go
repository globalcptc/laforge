package microcloud

import (
	"fmt"
	"strings"
)

func userData(osName string, agentURL string, password string) string {
	if isWindowsOS(osName) {
		return fmt.Sprintf(`<script>
powershell -Command "New-Item -ItemType Directory -Path $env:PROGRAMDATA\Laforge -Force"
powershell -Command "do { $ready = Test-NetConnection 1.1.1.1 -InformationLevel Quiet; Start-Sleep -Seconds 5 } until ($ready)"
powershell -Command "[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12; Invoke-WebRequest '%s' -OutFile $env:PROGRAMDATA\Laforge\laforge.exe"
powershell -Command "& $env:PROGRAMDATA\Laforge\laforge.exe -service install"
powershell -Command "& $env:PROGRAMDATA\Laforge\laforge.exe -service start"
</script>`, agentURL)
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
`, shellSingleQuote(password), agentURL)
}

func isWindowsOS(osName string) bool {
	osName = strings.ToLower(osName)
	return strings.HasPrefix(osName, "win") || strings.HasPrefix(osName, "w2k")
}

func shellSingleQuote(value string) string {
	return strings.ReplaceAll(value, "'", `'"'"'`)
}
