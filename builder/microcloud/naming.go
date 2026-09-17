package microcloud

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"

	"github.com/gen0cide/laforge/ent"
)

const maxLXDNameLength = 63

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
	if len(name) <= maxLXDNameLength {
		return name
	}

	sum := sha256.Sum256([]byte(name))
	suffix := hex.EncodeToString(sum[:])[:8]
	prefixLength := maxLXDNameLength - len(suffix) - 1
	prefix := strings.TrimRight(name[:prefixLength], "-")
	return prefix + "-" + suffix
}

func projectName(environment *ent.Environment, team *ent.Team, build *ent.Build) string {
	return safeName(
		"lf",
		environment.Name,
		fmt.Sprintf("team-%02d", team.TeamNumber),
		buildID(build),
	)
}

func clusterGroupName(environment *ent.Environment, team *ent.Team, build *ent.Build) string {
	return safeName(
		"lf",
		environment.Name,
		fmt.Sprintf("team-%02d", team.TeamNumber),
		buildID(build),
		"group",
	)
}

func networkName(network *ent.Network, build *ent.Build) string {
	return safeName("lf", network.Name, buildID(build))
}

func instanceName(host *ent.Host, build *ent.Build) string {
	return safeName("lf", host.Hostname, buildID(build))
}

func peerName(targetNetworkName string) string {
	return safeName("to", targetNetworkName)
}
