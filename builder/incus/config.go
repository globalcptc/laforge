package incus

import (
	"fmt"
	"net"
	"strings"
)

type InstanceSize struct {
	CPU    string `json:"cpu"`
	Memory string `json:"memory"`
}

// HostConfig describes one standalone Incus host. Teams are pinned to a
// single host and never span hosts.
type HostConfig struct {
	Name           string `json:"name"`
	BaseURL        string `json:"base_url"`
	ClientCertPath string `json:"client_cert_path"`
	ClientKeyPath  string `json:"client_key_path"`
	ServerCertPath string `json:"server_cert_path"`
	StoragePool    string `json:"storage_pool"`
	// TransitNetwork is an existing NAT'd managed bridge (for example
	// incusbr0) that every team gateway uses for egress.
	TransitNetwork string `json:"transit_network"`
	// IngressListenAddress is the host address that team ingress ports are
	// published on. Leave empty to disable public ingress on this host.
	IngressListenAddress string `json:"ingress_listen_address"`
	// IngressPortBase and IngressPortStride allocate ingress ports per team:
	// base + team_number*stride + index.
	IngressPortBase   int `json:"ingress_port_base"`
	IngressPortStride int `json:"ingress_port_stride"`
}

type BuilderConfig struct {
	LaForgeServerURL   string       `json:"laforge_server_url"`
	MaxBuildWorkers    int          `json:"max_build_workers"`
	MaxTeardownWorkers int          `json:"max_teardown_workers"`
	Hosts              []HostConfig `json:"hosts"`
	// GatewayImage is a local container image alias with cloud-init.
	GatewayImage string `json:"gateway_image"`
	// DNSServers are handed out by the gateway DHCP service.
	DNSServers    []string                `json:"dns_servers"`
	InstanceSizes map[string]InstanceSize `json:"instance_sizes"`
	// Images maps a LaForge host os to a local VM image alias.
	Images map[string]string `json:"images"`
	// ImageConfig maps a LaForge host os to extra instance config keys, for
	// example security.secureboot=false for a Windows image.
	ImageConfig map[string]map[string]string `json:"image_config"`
}

var defaultDNSServers = []string{"1.1.1.1", "8.8.8.8"}

func (config BuilderConfig) dnsServers() []string {
	if len(config.DNSServers) == 0 {
		return defaultDNSServers
	}
	return config.DNSServers
}

func (config BuilderConfig) Validate() error {
	if strings.TrimSpace(config.LaForgeServerURL) == "" {
		return fmt.Errorf("incus builder config %q is required", "laforge_server_url")
	}
	if strings.TrimSpace(config.GatewayImage) == "" {
		return fmt.Errorf("incus builder config %q is required", "gateway_image")
	}
	if config.MaxBuildWorkers < 1 {
		return fmt.Errorf("incus builder config max_build_workers must be positive")
	}
	if config.MaxTeardownWorkers < 1 {
		return fmt.Errorf("incus builder config max_teardown_workers must be positive")
	}
	if len(config.Hosts) == 0 {
		return fmt.Errorf("incus builder config hosts must not be empty")
	}
	if len(config.InstanceSizes) == 0 {
		return fmt.Errorf("incus builder config instance_sizes must not be empty")
	}
	if len(config.Images) == 0 {
		return fmt.Errorf("incus builder config images must not be empty")
	}

	for name, size := range config.InstanceSizes {
		if strings.TrimSpace(size.CPU) == "" || strings.TrimSpace(size.Memory) == "" {
			return fmt.Errorf("incus instance size %q requires cpu and memory", name)
		}
	}
	for _, server := range config.DNSServers {
		if net.ParseIP(server) == nil {
			return fmt.Errorf("incus builder config dns_servers entry %q is not an IP address", server)
		}
	}

	seen := make(map[string]struct{}, len(config.Hosts))
	for index, host := range config.Hosts {
		if err := host.validate(); err != nil {
			return fmt.Errorf("incus builder config hosts[%d]: %w", index, err)
		}
		if _, duplicate := seen[host.Name]; duplicate {
			return fmt.Errorf("incus builder config hosts[%d]: duplicate name %q", index, host.Name)
		}
		seen[host.Name] = struct{}{}
	}

	return nil
}

func (host HostConfig) validate() error {
	required := map[string]string{
		"name":             host.Name,
		"base_url":         host.BaseURL,
		"client_cert_path": host.ClientCertPath,
		"client_key_path":  host.ClientKeyPath,
		"server_cert_path": host.ServerCertPath,
		"storage_pool":     host.StoragePool,
		"transit_network":  host.TransitNetwork,
	}
	for name, value := range required {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%q is required", name)
		}
	}

	if host.IngressListenAddress == "" {
		return nil
	}
	if net.ParseIP(host.IngressListenAddress) == nil {
		return fmt.Errorf("ingress_listen_address %q is not an IP address", host.IngressListenAddress)
	}
	if host.IngressPortBase < 1024 || host.IngressPortBase > 65535 {
		return fmt.Errorf("ingress_port_base must be between 1024 and 65535")
	}
	if host.IngressPortStride < 1 {
		return fmt.Errorf("ingress_port_stride must be positive when ingress is enabled")
	}
	if host.IngressPortBase+host.IngressPortStride > 65535 {
		return fmt.Errorf("ingress_port_base + ingress_port_stride exceeds 65535")
	}

	return nil
}
