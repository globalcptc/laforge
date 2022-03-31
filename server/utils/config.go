package utils

import (
	"encoding/json"
	"fmt"
	"os"
)

type ServerConfig struct {
	Builders  map[string]BuilderConfig `json:"builders"`
	Database  DatabaseConfig           `json:"database"`
	Auth      AuthConfig               `json:"auth"`
	UI        UIConfig                 `json:"ui"`
	Agent     AgentConfig              `json:"agent"`
	Graphql   GraphqlConfig            `json:"graphql"`
	Debug     bool                     `json:"debug"`
	LogFolder string                   `json:"log_folder"`
	GinMode   string                   `json:"gin_mode"`
}

type BuilderConfig struct {
	Builder    string `json:"builder"`
	ConfigFile string `json:"config"`
}

type DatabaseConfig struct {
	PostgresUri string `json:"postgres_uri"`
	AdminUser   string `json:"admin_user"`
	AdminPass   string `json:"admin_password"`
}

type AuthConfig struct {
	GithubId      string `json:"github_id"`
	GithubSecret  string `json:"github_secret"`
	CookieTimeout int    `json:"cookie_timeout"`
}

type UIConfig struct {
	HttpsEnabled   bool     `json:"https_enabled"`
	AllowedOrigins []string `json:"allowed_origins"`
}

type AgentConfig struct {
	GrpcServerUri  string `json:"grpc_server_uri"`
	ApiDownloadUrl string `json:"api_download_url"`
}

type GraphqlConfig struct {
	Hostname       string `json:"hostname"`
	RedisServerUri string `json:"redis_server_uri"`
	RedisPassword  string `json:"redis_password"`
}

func LoadServerConfig() (*ServerConfig, error) {
	// Config file overrides. There might be a better way to define this
	configFile := "conf.json"
	if _, err := os.Stat("config.dev.json"); err == nil {
		configFile = "config.dev.json"
	}
	if _, err := os.Stat("config.prod.json"); err == nil {
		configFile = "config.prod.json"
	}
	// Read in the config file
	configBytes, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file \"%s\": %v", configFile, err)
	}
	// Marshal the config file into a ServerConfig object
	var loadedConfig ServerConfig
	err = json.Unmarshal(configBytes, &loadedConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal server config (\"%s\"): %v", configFile, err)
	}
	return &loadedConfig, nil
}
