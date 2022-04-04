package configs

import (
	"encoding/json"
	"fmt"
	"os"
)

func LoadBuilderConfig(configFilePath string, config interface{}) error {
	// Read in the config file
	configBytes, err := os.ReadFile(configFilePath)
	if err != nil {
		return fmt.Errorf("failed to read config file \"%s\": %v", configFilePath, err)
	}
	// Marshal the config file into a ServerConfig object
	err = json.Unmarshal(configBytes, &config)
	if err != nil {
		return fmt.Errorf("failed to unmarshal server config (\"%s\"): %v", configFilePath, err)
	}
	return nil
}
