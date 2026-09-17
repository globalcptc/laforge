package microcloud

import (
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"

	lxd "github.com/canonical/lxd/client"
	"github.com/canonical/lxd/shared/api"
)

var requiredExtensions = []string{
	"clustering",
	"clustering_groups",
	"instance_create_start",
	"network_allocate_external_ips",
	"network_forward",
	"network_type_ovn",
	"network_peer",
	"projects",
	"projects_restricted_cluster_target",
}

func connect(config BuilderConfig) (lxd.InstanceServer, []string, error) {
	clientCert, err := os.ReadFile(config.ClientCertPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read LXD client certificate: %w", err)
	}

	clientKey, err := os.ReadFile(config.ClientKeyPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read LXD client key: %w", err)
	}

	serverCert, err := os.ReadFile(config.ServerCertPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read LXD server certificate: %w", err)
	}

	client, err := lxd.ConnectLXD(config.BaseURL, &lxd.ConnectionArgs{
		TLSClientCert: string(clientCert),
		TLSClientKey:  string(clientKey),
		TLSServerCert: string(serverCert),
		UserAgent:     "laforge-microcloud-builder",
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to LXD at %q: %w", config.BaseURL, err)
	}

	for _, extension := range requiredExtensions {
		if !client.HasExtension(extension) {
			return nil, nil, fmt.Errorf("LXD server does not support required API extension %q", extension)
		}
	}

	members, err := client.GetClusterMembers()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list LXD cluster members: %w", err)
	}
	if len(members) == 0 {
		return nil, nil, fmt.Errorf("LXD cluster has no members")
	}

	memberNames := make([]string, 0, len(members))
	for _, member := range members {
		if !strings.EqualFold(member.Status, "online") {
			return nil, nil, fmt.Errorf(
				"LXD cluster member %q is not online: %s",
				member.ServerName,
				member.Message,
			)
		}
		memberNames = append(memberNames, member.ServerName)
	}
	sort.Strings(memberNames)

	uplink, _, err := client.GetNetwork(config.UplinkNetwork)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get LXD uplink network %q: %w", config.UplinkNetwork, err)
	}
	if uplink.Type != "physical" {
		return nil, nil, fmt.Errorf("LXD uplink network %q is type %q, want physical", config.UplinkNetwork, uplink.Type)
	}
	if strings.TrimSpace(uplink.Config["ipv4.ovn.ranges"]) == "" {
		return nil, nil, fmt.Errorf("LXD uplink network %q has no ipv4.ovn.ranges", config.UplinkNetwork)
	}
	if strings.TrimSpace(uplink.Config["ipv4.routes"]) == "" {
		return nil, nil, fmt.Errorf("LXD uplink network %q has no ipv4.routes for ingress allocation", config.UplinkNetwork)
	}

	if _, _, err := client.GetStoragePool(config.StoragePool); err != nil {
		return nil, nil, fmt.Errorf("failed to get LXD storage pool %q: %w", config.StoragePool, err)
	}

	checkedImages := make(map[string]struct{}, len(config.Images))
	for _, alias := range config.Images {
		if _, checked := checkedImages[alias]; checked {
			continue
		}
		if _, _, err := client.GetImageAlias(alias); err != nil {
			return nil, nil, fmt.Errorf("failed to get LXD image alias %q: %w", alias, err)
		}
		checkedImages[alias] = struct{}{}
	}

	return client, memberNames, nil
}

func isNotFound(err error) bool {
	return api.StatusErrorCheck(err, http.StatusNotFound)
}

func createOrUpdateClusterGroup(
	client lxd.InstanceServer,
	name string,
	member string,
	description string,
) error {
	group, etag, err := client.GetClusterGroup(name)
	if err != nil && !isNotFound(err) {
		return fmt.Errorf("failed to get LXD cluster group %q: %w", name, err)
	}

	put := api.ClusterGroupPut{
		Description: description,
		Members:     []string{member},
	}
	if isNotFound(err) {
		err = client.CreateClusterGroup(api.ClusterGroupsPost{
			Name:            name,
			ClusterGroupPut: put,
		})
		if err != nil {
			return fmt.Errorf("failed to create LXD cluster group %q: %w", name, err)
		}
		return nil
	}

	if len(group.Members) == 1 &&
		group.Members[0] == member &&
		group.Description == description {
		return nil
	}

	if err := client.UpdateClusterGroup(name, put, etag); err != nil {
		return fmt.Errorf("failed to update LXD cluster group %q: %w", name, err)
	}
	return nil
}

func createOrUpdateProject(
	client lxd.InstanceServer,
	name string,
	groupName string,
	uplinkNetwork string,
	description string,
) error {
	project, etag, err := client.GetProject(name)
	if err != nil && !isNotFound(err) {
		return fmt.Errorf("failed to get LXD project %q: %w", name, err)
	}

	if isNotFound(err) {
		err = client.CreateProject(api.ProjectsPost{
			Name: name,
			ProjectPut: api.ProjectPut{
				Description: description,
				Config: map[string]string{
					"features.images":             "false",
					"features.networks":           "true",
					"features.profiles":           "true",
					"restricted":                  "true",
					"restricted.cluster.groups":   groupName,
					"restricted.networks.uplinks": uplinkNetwork,
				},
			},
		})
		if err != nil {
			return fmt.Errorf("failed to create LXD project %q: %w", name, err)
		}
		return nil
	}

	config := cloneVars(project.Config)
	config["features.images"] = "false"
	config["features.networks"] = "true"
	config["features.profiles"] = "true"
	config["restricted"] = "true"
	config["restricted.cluster.groups"] = groupName
	config["restricted.networks.uplinks"] = uplinkNetwork
	if err := client.UpdateProject(name, api.ProjectPut{
		Description: description,
		Config:      config,
	}, etag); err != nil {
		return fmt.Errorf("failed to update LXD project %q: %w", name, err)
	}

	return nil
}
