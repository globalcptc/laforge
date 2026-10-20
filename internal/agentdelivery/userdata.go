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
      # Never give up restarting: disable systemd's start-rate limiter so even a
      # fast crash-loop keeps being restarted (RestartSec=5 already keeps us under
      # the default burst, but this makes "restart forever" explicit).
      StartLimitIntervalSec=0
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
// and installs it as a Windows service. It carries NO format wrapper -- the two
// exported Windows renderers add the cloudbase-init shebang or the EC2Launch
// <powershell> tags around this one shared body, so the install logic stays
// identical across hosters.
//
// Installed as a real Windows SERVICE (the agent speaks the Service Control
// Manager protocol natively -- see agent/src/winsvc.rs). The SCM then owns:
//   - boot-start (start= auto), with NO time trigger, so a clock skewed into the
//     future -- e.g. a UTC+4 region before cloudbase-init fixes the timezone --
//     can't stop it starting (the failure mode a boot scheduled task has);
//   - crash-restart (failure actions: restart after 5s, indefinitely), so a dead
//     agent comes straight back instead of staying down until the next reboot.
//
// If the service doesn't come up, it falls back to a hardened boot-time
// scheduled task running the same binary (which detects it wasn't launched by
// the SCM and runs as a console program) -- a backup for an unusual image.
// NOTE: in PowerShell "sc" is an alias for Set-Content, so the service control
// tool must be invoked as "sc.exe"; its "key= value" argument syntax (the space
// after "=" is required) is passed through cmd, which parses it reliably.
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
  try { Invoke-WebRequest -Uri "%s" -OutFile $dst -UseBasicParsing -ErrorAction Stop; break } catch { Start-Sleep -Seconds 5 }
}
# CRITICAL: drop out of "Stop" before the install. The retry-guarded download
# above needs Stop so a failed Invoke-WebRequest throws into its catch; but the
# idempotent cleanup/create below runs native tools (sc.exe, schtasks) that write
# expected "service/task does not exist" noise to stderr on a fresh box, and
# under $ErrorActionPreference='Stop' PowerShell turns ANY native stderr into a
# TERMINATING error -- which aborts the whole script before sc create ever
# runs. That is the "agent downloaded but no service installed" failure. The
# explicit Get-Service check further down is the real error handling here, so
# Continue is correct: let every install step run, then decide via the result.
$ErrorActionPreference = "Continue"
if (-not (Test-Path $dst)) { Write-Host "laforge-agent.exe did not download after retries; cannot install"; exit 1 }
# Remove any prior install (a legacy scheduled-task version, or a re-run) so this
# is idempotent; native-command failures here are expected and don't abort.
schtasks /delete /tn LaForgeAgent /f 2>$null
cmd /c "sc.exe stop LaForgeAgent" 2>$null
cmd /c "sc.exe delete LaForgeAgent" 2>$null
Start-Sleep -Seconds 2
cmd /c 'sc.exe create LaForgeAgent binPath= "C:\laforge-agent.exe" start= auto obj= LocalSystem DisplayName= "LaForge Agent"'
cmd /c 'sc.exe description LaForgeAgent "LaForge agent"'
cmd /c 'sc.exe failure LaForgeAgent reset= 0 actions= restart/5000/restart/5000/restart/5000'
cmd /c "sc.exe failureflag LaForgeAgent 1"
cmd /c "sc.exe start LaForgeAgent"
Start-Sleep -Seconds 5
# Fallback: if the service didn't come up (an unusual image, an SCM refusal), run
# the agent from a hardened boot-time scheduled task instead -- the SAME binary
# detects it wasn't launched by the SCM and runs as a console program. The failed
# service is deleted first, so the two are mutually exclusive (never two agents).
# The task mirrors the service's robustness: a BootTrigger with NO StartBoundary
# (immune to a future-skewed clock), a 1-minute repetition with IgnoreNew for
# crash recovery, and no execution time limit (the default is 72h).
$svc = Get-Service -Name LaForgeAgent -ErrorAction SilentlyContinue
if (-not $svc -or $svc.Status -ne 'Running') {
  Write-Host "LaForge service is not running; falling back to a scheduled task"
  cmd /c "sc.exe delete LaForgeAgent"
  $taskXml = @'
<?xml version="1.0" encoding="UTF-16"?>
<Task version="1.2" xmlns="http://schemas.microsoft.com/windows/2004/02/mit/task">
  <RegistrationInfo>
    <Description>LaForge agent (service fallback)</Description>
  </RegistrationInfo>
  <Triggers>
    <BootTrigger>
      <Enabled>true</Enabled>
      <Repetition>
        <Interval>PT1M</Interval>
        <StopAtDurationEnd>false</StopAtDurationEnd>
      </Repetition>
    </BootTrigger>
  </Triggers>
  <Principals>
    <Principal id="Author">
      <UserId>S-1-5-18</UserId>
      <RunLevel>HighestAvailable</RunLevel>
    </Principal>
  </Principals>
  <Settings>
    <MultipleInstancesPolicy>IgnoreNew</MultipleInstancesPolicy>
    <DisallowStartIfOnBatteries>false</DisallowStartIfOnBatteries>
    <StopIfGoingOnBatteries>false</StopIfGoingOnBatteries>
    <AllowHardTerminate>true</AllowHardTerminate>
    <StartWhenAvailable>true</StartWhenAvailable>
    <RunOnlyIfNetworkAvailable>false</RunOnlyIfNetworkAvailable>
    <AllowStartOnDemand>true</AllowStartOnDemand>
    <Enabled>true</Enabled>
    <Hidden>false</Hidden>
    <ExecutionTimeLimit>PT0S</ExecutionTimeLimit>
    <Priority>5</Priority>
    <RestartOnFailure>
      <Interval>PT1M</Interval>
      <Count>999</Count>
    </RestartOnFailure>
  </Settings>
  <Actions Context="Author">
    <Exec>
      <Command>C:\laforge-agent.exe</Command>
    </Exec>
  </Actions>
</Task>
'@
  $xmlPath = "C:\laforge-agent-task.xml"
  Set-Content -Path $xmlPath -Value $taskXml -Encoding Unicode
  schtasks /create /tn LaForgeAgent /xml "$xmlPath" /f
  schtasks /run /tn LaForgeAgent
}
`, binaryURL)
}

// DeliveryFormat names how a host's agent-install user-data is framed for the
// image that consumes it -- the read-only "what first-boot config did this host
// get?" view the UI's Render tab shows uses it.
type DeliveryFormat string

const (
	FormatCloudInit DeliveryFormat = "cloud-init"     // Linux: #cloud-config
	FormatCloudbase DeliveryFormat = "cloudbase-init" // Windows on Incus/MicroCloud/OpenStack: #ps1_sysnative
	FormatEC2Launch DeliveryFormat = "ec2launch"      // Windows on AWS: <powershell>
)

// DeliveredUserData renders the agent-install user-data LaForge delivers to a
// host and reports how it's framed. It is the read side of delivery: it calls
// the SAME LinuxUserData / WindowsCloudbaseUserData / WindowsEC2LaunchUserData
// renderers the builders deliver and the SAME PlatformFor OS split, so it can't
// drift from what a host actually received. The only builder-specific choice it
// mirrors is Windows cloudbase-vs-EC2Launch, keyed on builderKind (the
// builder_config.kind: incus/microcloud/openstack run cloudbase-init, aws runs
// EC2Launch). url is the per-host agent download URL; a caller without the real
// one-time URL (e.g. a read-only UI request after the deploy consumed the
// token) passes a placeholder -- the URL isn't reproducible, and isn't what
// this view exists to answer.
func DeliveredUserData(os, builderKind, url string) (userData string, format DeliveryFormat) {
	if PlatformFor(os) != Windows {
		return LinuxUserData(url), FormatCloudInit
	}
	if builderKind == "aws" {
		return WindowsEC2LaunchUserData(url), FormatEC2Launch
	}
	return WindowsCloudbaseUserData(url), FormatCloudbase
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
