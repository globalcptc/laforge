package agentdelivery

import "fmt"

// linuxUserData is a cloud-init cloud-config that downloads the per-host
// agent binary and runs it under systemd, so it restarts if it dies and
// comes back on reboot. The binary already carries its identity, so this
// needs no certificates -- just fetch and run. curl retries because the
// api may be reached before its first-boot networking fully settles.
func linuxUserData(binaryURL string) string {
	return fmt.Sprintf(`#cloud-config
write_files:
  - path: /etc/systemd/system/laforge-agent.service
    permissions: '0644'
    content: |
      [Unit]
      Description=LaForge agent
      After=network-online.target
      Wants=network-online.target
      [Service]
      ExecStart=/usr/local/bin/laforge-agent
      Restart=always
      RestartSec=5
      [Install]
      WantedBy=multi-user.target
runcmd:
  - [ sh, -c, "for i in $(seq 1 30); do curl -fsSL '%s' -o /usr/local/bin/laforge-agent && break || sleep 5; done" ]
  - [ chmod, "0755", /usr/local/bin/laforge-agent ]
  - [ systemctl, daemon-reload ]
  - [ systemctl, enable, --now, laforge-agent.service ]
`, binaryURL)
}

// windowsUserData is the cloudbase-init equivalent: a PowerShell first-boot
// script that downloads the agent and registers it as a service that starts
// at boot. Only images built with cloudbase-init consume this. Run as a
// scheduled task, not a Windows service: the agent is a plain console
// program with no Service Control Manager handler, so `sc create` starts
// it but the SCM kills it within ~30s for not reporting as a service. A
// scheduled task (onstart, SYSTEM) runs a console program correctly and
// restarts it on every boot.
func windowsUserData(binaryURL string) string {
	return fmt.Sprintf(`#ps1_sysnative
$ErrorActionPreference = "Stop"
# Clear "user must change password at next logon" on the built-in local admin.
# cloudbase-init sets that flag when it injects the admin password
# (first_logon_behaviour), which blocks the very login the password was set up
# for. This user-data runs after the password plugin, so clearing it here sticks.
# Best-effort -- a differently-named admin (set via a LaForge set_password step)
# is handled in the agent itself; don't abort the agent install if this fails.
try {
  $a = [ADSI]('WinNT://./Administrator,user')
  $a.PasswordExpired = 0
  $a.SetInfo()
} catch { Write-Host "clearing Administrator password-expiry: $_" }
$dst = "C:\laforge-agent.exe"
for ($i = 0; $i -lt 60; $i++) {
  try { Invoke-WebRequest -Uri "%s" -OutFile $dst -UseBasicParsing; break } catch { Start-Sleep -Seconds 5 }
}
schtasks /create /tn LaForgeAgent /tr "$dst" /sc onstart /ru SYSTEM /rl highest /f
schtasks /run /tn LaForgeAgent
`, binaryURL)
}
