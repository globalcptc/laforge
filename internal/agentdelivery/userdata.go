package agentdelivery

import "fmt"

// The functions here RENDER the agent install into a hoster's user-data format.
// Which one a builder calls is the builder's choice -- the format (cloud-init vs
// cloudbase-init vs EC2Launch) is a property of the image/hoster, so the builder
// owns it (see internal/builder). agentdelivery only hands over the neutral
// intent (Delivery.DownloadURL + Platform) and these reusable renderers.

// LinuxUserData is a cloud-init cloud-config that downloads the per-host agent
// binary and runs it under systemd, so it restarts if it dies and comes back on
// reboot. The binary already carries its identity, so this needs no certificates
// -- just fetch and run. curl retries because the api may be reached before its
// first-boot networking fully settles.
func LinuxUserData(binaryURL string) string {
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

// windowsAgentPowerShell is the first-boot PowerShell that downloads the agent
// and registers it as a boot-time task. It carries NO format wrapper -- the two
// exported Windows renderers add the cloudbase-init shebang or the EC2Launch
// <powershell> tags around this one shared body, so the install logic stays
// identical across hosters.
//
// Run as a scheduled task, not a Windows service: the agent is a plain console
// program with no Service Control Manager handler, so `sc create` starts it but
// the SCM kills it within ~30s for not reporting as a service. A scheduled task
// (onstart, SYSTEM) runs a console program correctly and restarts it on boot.
func windowsAgentPowerShell(binaryURL string) string {
	return fmt.Sprintf(`$ErrorActionPreference = "Stop"
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

// WindowsCloudbaseUserData wraps the agent install for cloudbase-init (the
// "#ps1_sysnative" shebang). Used by builders whose Windows images run
// cloudbase-init -- the images LaForge builds for Incus/MicroCloud/OpenStack.
func WindowsCloudbaseUserData(binaryURL string) string {
	return "#ps1_sysnative\n" + windowsAgentPowerShell(binaryURL)
}

// WindowsEC2LaunchUserData wraps the same install in the <powershell>...</powershell>
// tags AWS EC2Launch (v1/v2) and EC2Config execute. Used by the AWS builder,
// whose stock Windows AMIs run EC2Launch and ignore the cloudbase-init shebang
// entirely (so the agent would never install). cloudbase-init also accepts
// <powershell>, so this is safe whichever agent the chosen AMI ships.
func WindowsEC2LaunchUserData(binaryURL string) string {
	return "<powershell>\n" + windowsAgentPowerShell(binaryURL) + "</powershell>\n"
}
