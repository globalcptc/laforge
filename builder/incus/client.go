package incus

import (
	"fmt"
	"net/http"
	"os"

	incusclient "github.com/lxc/incus/v6/client"
	"github.com/lxc/incus/v6/shared/api"
)

var requiredExtensions = []string{
	"instance_create_start",
	"projects",
}

func connect(config BuilderConfig, host HostConfig) (incusclient.InstanceServer, error) {
	clientCert, err := os.ReadFile(host.ClientCertPath)
	if err != nil {
		return nil, fmt.Errorf("host %q: failed to read Incus client certificate: %w", host.Name, err)
	}
	clientKey, err := os.ReadFile(host.ClientKeyPath)
	if err != nil {
		return nil, fmt.Errorf("host %q: failed to read Incus client key: %w", host.Name, err)
	}
	serverCert, err := os.ReadFile(host.ServerCertPath)
	if err != nil {
		return nil, fmt.Errorf("host %q: failed to read Incus server certificate: %w", host.Name, err)
	}

	client, err := incusclient.ConnectIncus(host.BaseURL, &incusclient.ConnectionArgs{
		TLSClientCert: string(clientCert),
		TLSClientKey:  string(clientKey),
		TLSServerCert: string(serverCert),
		UserAgent:     "laforge-incus-builder",
	})
	if err != nil {
		return nil, fmt.Errorf("host %q: failed to connect to Incus at %q: %w", host.Name, host.BaseURL, err)
	}

	for _, extension := range requiredExtensions {
		if !client.HasExtension(extension) {
			return nil, fmt.Errorf("host %q: Incus server does not support required API extension %q", host.Name, extension)
		}
	}

	transit, _, err := client.GetNetwork(host.TransitNetwork)
	if err != nil {
		return nil, fmt.Errorf("host %q: failed to get transit network %q: %w", host.Name, host.TransitNetwork, err)
	}
	if transit.Type != "bridge" {
		return nil, fmt.Errorf("host %q: transit network %q is type %q, want bridge", host.Name, host.TransitNetwork, transit.Type)
	}

	if _, _, err := client.GetStoragePool(host.StoragePool); err != nil {
		return nil, fmt.Errorf("host %q: failed to get storage pool %q: %w", host.Name, host.StoragePool, err)
	}

	aliases := map[string]struct{}{config.GatewayImage: {}}
	for _, alias := range config.Images {
		aliases[alias] = struct{}{}
	}
	for alias := range aliases {
		if _, _, err := client.GetImageAlias(alias); err != nil {
			return nil, fmt.Errorf("host %q: failed to get image alias %q: %w", host.Name, alias, err)
		}
	}

	return client, nil
}

func isNotFound(err error) bool {
	return api.StatusErrorCheck(err, http.StatusNotFound)
}

func createOrUpdateProject(
	client incusclient.InstanceServer,
	name string,
	description string,
) error {
	desired := map[string]string{
		"features.images":   "false",
		"features.networks": "false",
		"features.profiles": "true",
	}

	project, etag, err := client.GetProject(name)
	if err != nil && !isNotFound(err) {
		return fmt.Errorf("failed to get Incus project %q: %w", name, err)
	}

	if isNotFound(err) {
		err = client.CreateProject(api.ProjectsPost{
			Name: name,
			ProjectPut: api.ProjectPut{
				Description: description,
				Config:      desired,
			},
		})
		if err != nil {
			return fmt.Errorf("failed to create Incus project %q: %w", name, err)
		}
		return nil
	}

	config := cloneVars(project.Config)
	for key, value := range desired {
		config[key] = value
	}
	if err := client.UpdateProject(name, api.ProjectPut{
		Description: description,
		Config:      config,
	}, etag); err != nil {
		return fmt.Errorf("failed to update Incus project %q: %w", name, err)
	}

	return nil
}
