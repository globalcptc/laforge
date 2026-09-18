package utils

import (
	"net/url"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSecrets(t *testing.T) {
	tempDir := t.TempDir()
	writeSecret := func(name, value string) string {
		t.Helper()
		path := filepath.Join(tempDir, name)
		if err := os.WriteFile(path, []byte(value+"\n"), 0600); err != nil {
			t.Fatal(err)
		}
		return path
	}

	config := ServerConfig{
		ConfigFile: filepath.Join(tempDir, "conf.prod.json"),
		Database: DatabaseConfig{
			PostgresUri:          "postgresql://laforger@db:5432/laforge?sslmode=disable",
			PostgresPasswordFile: writeSecret("postgres", "p@ss:/word"),
			AdminPassFile:        writeSecret("admin", "admin-secret"),
		},
		Auth: AuthConfig{
			GithubSecretFile:  writeSecret("github", "github-secret"),
			SessionSecretFile: writeSecret("session", "session-secret"),
		},
		Graphql: GraphqlConfig{
			RedisPasswordFile: writeSecret("redis", "redis-secret"),
		},
	}

	if err := config.loadSecrets(); err != nil {
		t.Fatal(err)
	}

	postgresURL, err := url.Parse(config.Database.PostgresUri)
	if err != nil {
		t.Fatal(err)
	}
	password, ok := postgresURL.User.Password()
	if !ok || password != "p@ss:/word" {
		t.Fatalf("unexpected PostgreSQL password: %q", password)
	}
	if config.Database.AdminPass != "admin-secret" {
		t.Fatalf("unexpected admin password: %q", config.Database.AdminPass)
	}
	if config.Auth.GithubSecret != "github-secret" {
		t.Fatalf("unexpected GitHub secret: %q", config.Auth.GithubSecret)
	}
	if config.Auth.SessionSecret != "session-secret" {
		t.Fatalf("unexpected session secret: %q", config.Auth.SessionSecret)
	}
	if config.Graphql.RedisPassword != "redis-secret" {
		t.Fatalf("unexpected Redis password: %q", config.Graphql.RedisPassword)
	}
}

func TestLoadSecretsRejectsPasswordURIWithoutUsername(t *testing.T) {
	tempDir := t.TempDir()
	passwordPath := filepath.Join(tempDir, "postgres")
	if err := os.WriteFile(passwordPath, []byte("secret"), 0600); err != nil {
		t.Fatal(err)
	}
	config := ServerConfig{
		ConfigFile: filepath.Join(tempDir, "conf.prod.json"),
		Database: DatabaseConfig{
			PostgresUri:          "postgresql://db:5432/laforge",
			PostgresPasswordFile: passwordPath,
		},
	}

	if err := config.loadSecrets(); err == nil {
		t.Fatal("expected missing PostgreSQL username to fail")
	}
}
