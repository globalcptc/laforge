package incus

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"

	"github.com/gen0cide/laforge/ent"
)

const maxIncusNameLength = 63

var invalidNameCharacters = regexp.MustCompile(`[^a-z0-9-]+`)

func buildID(build *ent.Build) string {
	id := build.ID.String()
	if len(id) > 8 {
		return id[:8]
	}

	return id
}

func safeName(parts ...string) string {
	name := strings.ToLower(strings.Join(parts, "-"))
	name = invalidNameCharacters.ReplaceAllString(name, "-")
	name = strings.Trim(name, "-")
	if len(name) <= maxIncusNameLength {
		return name
	}

	sum := sha256.Sum256([]byte(name))
	suffix := hex.EncodeToString(sum[:])[:8]
	prefixLength := maxIncusNameLength - len(suffix) - 1
	prefix := strings.TrimRight(name[:prefixLength], "-")
	return prefix + "-" + suffix
}

func hashHex(length int, parts ...string) string {
	sum := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return hex.EncodeToString(sum[:])[:length]
}

func projectName(environment *ent.Environment, team *ent.Team, build *ent.Build) string {
	return safeName(
		"lf",
		environment.Name,
		fmt.Sprintf("team-%02d", team.TeamNumber),
		buildID(build),
	)
}

func gatewayName() string {
	return "lf-gateway"
}

// networkName returns the bridge name for a team network. Bridges live in
// the default project namespace and become host interfaces, so the name is
// a short hash (Linux interface names are limited to 15 characters) that is
// unique per build, team and network.
func networkName(network *ent.Network, team *ent.Team, build *ent.Build) string {
	return "lf" + hashHex(10, buildID(build), fmt.Sprint(team.TeamNumber), network.Name)
}

// gatewayInterface is the interface name inside the team gateway for a
// bridge network.
func gatewayInterface(bridge string) string {
	return "g" + strings.TrimPrefix(bridge, "lf")
}

func instanceName(host *ent.Host, build *ent.Build) string {
	return safeName("lf", host.Hostname, buildID(build))
}

// macAddress derives a stable locally administered MAC address so the
// gateway DHCP service can reserve the address before the NIC exists.
func macAddress(seed string) string {
	sum := sha256.Sum256([]byte(seed))
	return fmt.Sprintf("10:66:6a:%02x:%02x:%02x", sum[0], sum[1], sum[2])
}
