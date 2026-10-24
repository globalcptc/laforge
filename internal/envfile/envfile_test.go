package envfile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSetsRealValues(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	content := "# a comment\n\nGITHUB_APP_ID=123456\nGITHUB_APP_CLIENT_SECRET=\"has a space and a # not a comment\"\nQUOTED_SINGLE='also quoted'\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	for _, k := range []string{"GITHUB_APP_ID", "GITHUB_APP_CLIENT_SECRET", "QUOTED_SINGLE", "ALREADY_SET"} {
		os.Unsetenv(k)
	}
	t.Cleanup(func() {
		for _, k := range []string{"GITHUB_APP_ID", "GITHUB_APP_CLIENT_SECRET", "QUOTED_SINGLE", "ALREADY_SET"} {
			os.Unsetenv(k)
		}
	})

	if err := Load(path); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := os.Getenv("GITHUB_APP_ID"); got != "123456" {
		t.Errorf("GITHUB_APP_ID = %q, want 123456", got)
	}
	if got := os.Getenv("GITHUB_APP_CLIENT_SECRET"); got != "has a space and a # not a comment" {
		t.Errorf("GITHUB_APP_CLIENT_SECRET = %q, want the real quoted value including the # character", got)
	}
	if got := os.Getenv("QUOTED_SINGLE"); got != "also quoted" {
		t.Errorf("QUOTED_SINGLE = %q, want \"also quoted\"", got)
	}
}

func TestLoadDoesNotOverrideAlreadySetEnv(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	if err := os.WriteFile(path, []byte("ALREADY_SET=from-file\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	os.Setenv("ALREADY_SET", "from-real-environment")
	t.Cleanup(func() { os.Unsetenv("ALREADY_SET") })

	if err := Load(path); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := os.Getenv("ALREADY_SET"); got != "from-real-environment" {
		t.Errorf("ALREADY_SET = %q, want the real environment's value to win over the file", got)
	}
}

func TestLoadMissingFileIsNotAnError(t *testing.T) {
	if err := Load(filepath.Join(t.TempDir(), "does-not-exist.env")); err != nil {
		t.Fatalf("Load on a missing file = %v, want nil (a real deployment has no .env at all)", err)
	}
}
