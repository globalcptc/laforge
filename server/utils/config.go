package utils

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
)

type ServerConfig struct {
	ConfigFile string                   `json:"-"`
	Builders   map[string]BuilderConfig `json:"builders"`
	Database   DatabaseConfig           `json:"database"`
	Auth       AuthConfig               `json:"auth"`
	UI         UIConfig                 `json:"ui"`
	Agent      AgentConfig              `json:"agent"`
	Graphql    GraphqlConfig            `json:"graphql"`
	Debug      bool                     `json:"debug"`
	AgentDebug bool                     `json:"agent_debug"`
	LogFolder  string                   `json:"log_folder"`
	GinMode    string                   `json:"gin_mode"`
}

type BuilderConfig struct {
	Builder    string `json:"builder"`
	ConfigFile string `json:"config"`
}

type DatabaseConfig struct {
	PostgresUri          string `json:"postgres_uri"`
	PostgresPasswordFile string `json:"postgres_password_file"`
	AdminUser            string `json:"admin_user"`
	AdminPass            string `json:"admin_password"`
	AdminPassFile        string `json:"admin_password_file"`
}

type AuthConfig struct {
	GithubId          string `json:"github_id"`
	GithubSecret      string `json:"github_secret"`
	GithubSecretFile  string `json:"github_secret_file"`
	SessionSecret     string `json:"session_secret"`
	SessionSecretFile string `json:"session_secret_file"`
	CookieTimeout     int    `json:"cookie_timeout"`
}

type UIConfig struct {
	HttpsEnabled   bool     `json:"https_enabled"`
	AllowedOrigins []string `json:"allowed_origins"`
}

type AgentConfig struct {
	GrpcServerUri      string `json:"grpc_server_uri"`
	ApiDownloadUrl     string `json:"api_download_url"`
	GrpcUseSystemRoots bool   `json:"grpc_use_system_roots"`
	GrpcCACertPath     string `json:"grpc_ca_cert_path"`
	GrpcTLSCertPath    string `json:"grpc_tls_cert_path"`
	GrpcTLSKeyPath     string `json:"grpc_tls_key_path"`
}

type GraphqlConfig struct {
	Hostname          string `json:"hostname"`
	RedisServerUri    string `json:"redis_server_uri"`
	RedisPassword     string `json:"redis_password"`
	RedisPasswordFile string `json:"redis_password_file"`
}

func readConfigSecret(configFile, secretFile string) (string, error) {
	if secretFile == "" {
		return "", nil
	}
	if !filepath.IsAbs(secretFile) {
		secretFile = filepath.Join(filepath.Dir(configFile), secretFile)
	}
	value, err := os.ReadFile(secretFile)
	if err != nil {
		return "", fmt.Errorf("failed to read secret file %q: %w", secretFile, err)
	}
	secret := strings.TrimSpace(string(value))
	if secret == "" {
		return "", fmt.Errorf("secret file %q is empty", secretFile)
	}
	return secret, nil
}

func (config *ServerConfig) loadSecrets() error {
	type secretTarget struct {
		file  string
		value *string
	}
	targets := []secretTarget{
		{config.Database.AdminPassFile, &config.Database.AdminPass},
		{config.Auth.GithubSecretFile, &config.Auth.GithubSecret},
		{config.Auth.SessionSecretFile, &config.Auth.SessionSecret},
		{config.Graphql.RedisPasswordFile, &config.Graphql.RedisPassword},
	}
	for _, target := range targets {
		if target.file == "" {
			continue
		}
		value, err := readConfigSecret(config.ConfigFile, target.file)
		if err != nil {
			return err
		}
		*target.value = value
	}

	if config.Agent.GrpcUseSystemRoots && config.Agent.GrpcCACertPath != "" {
		return fmt.Errorf("agent.grpc_use_system_roots and agent.grpc_ca_cert_path are mutually exclusive")
	}
	if config.Database.PostgresPasswordFile == "" {
		return nil
	}
	password, err := readConfigSecret(config.ConfigFile, config.Database.PostgresPasswordFile)
	if err != nil {
		return err
	}
	postgresURL, err := url.Parse(config.Database.PostgresUri)
	if err != nil {
		return fmt.Errorf("failed to parse database.postgres_uri: %w", err)
	}
	if postgresURL.User == nil || postgresURL.User.Username() == "" {
		return fmt.Errorf("database.postgres_uri must include a username when postgres_password_file is set")
	}
	postgresURL.User = url.UserPassword(postgresURL.User.Username(), password)
	config.Database.PostgresUri = postgresURL.String()
	return nil
}

func LoadServerConfig() (*ServerConfig, error) {
	// Config file overrides. There might be a better way to define this
	cwd, err := os.Getwd()
	if err != nil {
		logrus.Warn("failed to get current working directory, using relative paths for config file instead")
		cwd = "./"
	}
	configFile := path.Join(cwd, "conf.json")
	if _, err := os.Stat(path.Join(cwd, "conf.dev.json")); err == nil {
		configFile = path.Join(cwd, "conf.dev.json")
	}
	if _, err := os.Stat(path.Join(cwd, "conf.prod.json")); err == nil {
		configFile = path.Join(cwd, "conf.prod.json")
	}
	// Read in the config file
	configBytes, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file \"%s\": %v", configFile, err)
	}
	configBytes = []byte(os.ExpandEnv(string(configBytes)))
	// Marshal the config file into a ServerConfig object
	var loadedConfig ServerConfig
	err = json.Unmarshal(configBytes, &loadedConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal server config (\"%s\"): %v", configFile, err)
	}
	loadedConfig.ConfigFile = configFile
	if err := loadedConfig.loadSecrets(); err != nil {
		return nil, err
	}
	return &loadedConfig, nil
}
