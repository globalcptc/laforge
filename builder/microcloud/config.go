package microcloud

import (
	"fmt"
	"strings"
)

type InstanceSize struct {
	CPU    string `json:"cpu"`
	Memory string `json:"memory"`
}

type BuilderConfig struct {
	LaForgeServerURL   string                  `json:"laforge_server_url"`
	MaxBuildWorkers    int                     `json:"max_build_workers"`
	MaxTeardownWorkers int                     `json:"max_teardown_workers"`
	BaseURL            string                  `json:"base_url"`
	ClientCertPath     string                  `json:"client_cert_path"`
	ClientKeyPath      string                  `json:"client_key_path"`
	ServerCertPath     string                  `json:"server_cert_path"`
	StoragePool        string                  `json:"storage_pool"`
	UplinkNetwork      string                  `json:"uplink_network"`
	InstanceSizes      map[string]InstanceSize `json:"instance_sizes"`
	Images             map[string]string       `json:"images"`
}

func (config BuilderConfig) Validate() error {
	required := map[string]string{
		"laforge_server_url": config.LaForgeServerURL,
		"base_url":           config.BaseURL,
		"client_cert_path":   config.ClientCertPath,
		"client_key_path":    config.ClientKeyPath,
		"server_cert_path":   config.ServerCertPath,
		"storage_pool":       config.StoragePool,
		"uplink_network":     config.UplinkNetwork,
	}

	for name, value := range required {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("microcloud builder config %q is required", name)
		}
	}

	if config.MaxBuildWorkers < 1 {
		return fmt.Errorf("microcloud builder config max_build_workers must be positive")
	}
	if config.MaxTeardownWorkers < 1 {
		return fmt.Errorf("microcloud builder config max_teardown_workers must be positive")
	}
	if len(config.InstanceSizes) == 0 {
		return fmt.Errorf("microcloud builder config instance_sizes must not be empty")
	}
	if len(config.Images) == 0 {
		return fmt.Errorf("microcloud builder config images must not be empty")
	}

	for name, size := range config.InstanceSizes {
		if strings.TrimSpace(size.CPU) == "" || strings.TrimSpace(size.Memory) == "" {
			return fmt.Errorf("microcloud instance size %q requires cpu and memory", name)
		}
	}

	return nil
}
