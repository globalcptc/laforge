package utils

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	"github.com/gen0cide/laforge/ent"
)

type buildConf struct {
	EnvironmentName string     `json:"environment_name"`
	RevisionNumber  int        `json:"revision_number"`
	Teams           []teamConf `json:"teams"`
}
type teamConf struct {
	TeamNumber int           `json:"team_number"`
	Networks   []networkConf `json:"networks"`
}
type networkConf struct {
	NetworkName string     `json:"network_name"`
	CIDR        string     `json:"cidr"`
	VDIVisible  bool       `json:"vdi_visible"`
	Hosts       []hostConf `json:"hosts"`
}
type hostConf struct {
	HostName         string   `json:"host_name"`
	SubnetIP         string   `json:"subnet_ip"`
	OS               string   `json:"os"`
	Instance_Size    string   `json:"instance_size"`
	AllowMacChanges  bool     `json:"allow_mac_changes"`
	ExposedTCPPorts  []string `json:"exposed_tcp_ports"`
	ExposedUDPPorts  []string `json:"exposed_udp_ports"`
	OverridePassword string   `json:"override_password"`
	DiskSize         int      `json:"disk_size"`
	AgentURL         string   `json:"agent_url"`
}

func GenerateBuildConf(ctx context.Context, client *ent.Client, entBuild *ent.Build) (string, error) {

	entEnvrioment, err := entBuild.QueryBuildToEnvironment().Only(ctx)
	if err != nil {
		return "", err
	}
	laforgeServerUrl, exists := entEnvrioment.Config["laforge_server_url"]
	if !exists {
		return "", errors.New("laforge_server_url doesn't exist in the environment configuration")
	}
	currentBuildConf := buildConf{
		EnvironmentName: entEnvrioment.Name,
		RevisionNumber:  entBuild.Revision,
		Teams:           []teamConf{},
	}
	entTeams, err := entBuild.QueryBuildToTeam().All(ctx)
	if err != nil {
		return "", err
	}
	for _, entTeam := range entTeams {
		currentTeamConf := teamConf{
			TeamNumber: entTeam.TeamNumber,
			Networks:   []networkConf{},
		}
		entProNetworks, err := entTeam.QueryTeamToProvisionedNetwork().All(ctx)
		if err != nil {
			return "", err
		}
		for _, entProNetwork := range entProNetworks {
			entNetwork, err := entProNetwork.QueryProvisionedNetworkToNetwork().Only(ctx)
			if err != nil {
				return "", err
			}
			currentNetworkConf := networkConf{
				NetworkName: entNetwork.Name,
				CIDR:        entProNetwork.Cidr,
				VDIVisible:  entNetwork.VdiVisible,
				Hosts:       []hostConf{},
			}
			entProHosts, err := entProNetwork.QueryProvisionedNetworkToProvisionedHost().All(ctx)
			if err != nil {
				return "", err
			}
			for _, entProHost := range entProHosts {
				entHost, err := entProHost.QueryProvisionedHostToHost().Only(ctx)
				if err != nil {
					return "", err
				}
				entDisk, err := entHost.QueryHostToDisk().Only(ctx)
				if err != nil {
					return "", err
				}
				entAgent, err := entProHost.QueryProvisionedHostToGinFileMiddleware().First(ctx)
				if err != nil {
					return "", err
				}

				agentUrl := fmt.Sprintf("%s/api/download/%s", laforgeServerUrl, entAgent.URLID)
				currentHostConf := hostConf{
					HostName:         entHost.Hostname,
					SubnetIP:         entProHost.SubnetIP,
					OS:               entHost.OS,
					Instance_Size:    entHost.InstanceSize,
					AllowMacChanges:  entHost.AllowMACChanges,
					ExposedTCPPorts:  entHost.ExposedTCPPorts,
					ExposedUDPPorts:  entHost.ExposedUDPPorts,
					OverridePassword: entHost.OverridePassword,
					DiskSize:         entDisk.Size,
					AgentURL:         agentUrl,
				}
				currentNetworkConf.Hosts = append(currentNetworkConf.Hosts, currentHostConf)
			}
			currentTeamConf.Networks = append(currentTeamConf.Networks, currentNetworkConf)

		}
		currentBuildConf.Teams = append(currentBuildConf.Teams, currentTeamConf)
	}
	jsonByteArray, err := json.MarshalIndent(currentBuildConf, "", "  ")
	if err != nil {
		return "", err
	}

	binaryPath := path.Join("builds", entEnvrioment.Name, fmt.Sprint(entBuild.Revision))
	os.MkdirAll(binaryPath, 0755)
	binaryName := path.Join(binaryPath, "export.json")
	binaryName, err = filepath.Abs(binaryName)
	if err != nil {
		return "", err
	}

	err = ioutil.WriteFile(binaryName, jsonByteArray, os.ModePerm)
	if err != nil {
		return "", err
	}
	entTmpUrl, err := CreateTempURL(ctx, client, binaryName)
	if err != nil {
		return "nil", err
	}

	exportUrl := fmt.Sprintf("%s/api/download/%s", laforgeServerUrl, entTmpUrl.URLID)
	return exportUrl, nil

}
