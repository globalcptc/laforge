package compose

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const flakyish = `
name: ignored-by-laforge
x-app: &app
  build: { context: ., target: base }
  image: registry.example/team/app:1.0
  env_file: .env
  volumes: [ "media:/media" ]
services:
  db:
    image: postgres:16-alpine
    volumes: [ "pgdata:/var/lib/postgresql/data" ]
  web:
    <<: *app
  worker:
    <<: *app
  nginx:
    image: ${NGINX_IMAGE:-nginx:1.27-alpine}
    ports: [ "80:80" ]
    volumes:
      - ./nginx/nginx.conf:/etc/nginx/nginx.conf:ro
      - ./nginx/certs:/etc/nginx/certs:ro
      - media:/media:ro
volumes: { pgdata: {}, media: {} }
`

// writeStack lays out a project directory, containers/<name>/, under a fresh
// repo root.
func writeStack(t *testing.T, name string, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for rel, content := range files {
		path := filepath.Join(root, "containers", name, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func untar(t *testing.T, archive []byte) map[string]string {
	t.Helper()
	gz, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]string{}
	tr := tar.NewReader(gz)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			return out
		}
		if err != nil {
			t.Fatal(err)
		}
		data, _ := io.ReadAll(tr)
		out[h.Name] = string(data)
	}
}

func TestLoadShipsTheWholeDirectoryAndFindsImages(t *testing.T) {
	files := map[string]string{
		"compose.yaml":         flakyish,
		".env":                 "POSTGRES_PASSWORD=x\n",
		"nginx/nginx.conf":     "events {}\n",
		"nginx/certs/site.key": "\x00\x01binary\xff",
	}
	root := writeStack(t, "flaky", files)
	b, err := Load(root, "containers/flaky/compose.yaml")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if b.File != "compose.yaml" {
		t.Errorf("File = %q, want compose.yaml", b.File)
	}
	// The anchored app image once, Docker Hub postgres, and not the
	// interpolated nginx ref.
	if got := strings.Join(b.Images, ","); got != "postgres:16-alpine,registry.example/team/app:1.0" {
		t.Errorf("Images = %s", got)
	}
	got := untar(t, b.Archive)
	for rel, want := range files {
		if got[rel] != want {
			t.Errorf("archive %s = %q, want %q", rel, got[rel], want)
		}
	}
	again, err := Load(root, "containers/flaky/compose.yaml")
	if err != nil || !bytes.Equal(b.Archive, again.Archive) {
		t.Errorf("archive is not stable across loads (err %v)", err)
	}
}

func TestLoadRejectsWhatCannotWorkOnAHost(t *testing.T) {
	cases := []struct {
		name, compose, wantErr string
		extra                  map[string]string
	}{
		{"build without image", "services:\n  app:\n    build: .\n", "LaForge does not build images", nil},
		{"no image", "services:\n  app:\n    command: x\n", "has no image", nil},
		{"no services", "volumes: {}\n", "no services", nil},
		{"missing bind source", "services:\n  app:\n    image: a\n    volumes: [ './conf:/etc/app' ]\n", "does not exist next to the compose file", nil},
		{"bind outside directory", "services:\n  app:\n    image: a\n    volumes: [ '../secrets:/s' ]\n", "outside the compose file's directory", nil},
		{"long-form bind", "services:\n  app:\n    image: a\n    volumes:\n      - { type: bind, source: ./gone, target: /x }\n", "does not exist next to the compose file", nil},
		{"too large", "services:\n  app:\n    image: a\n", "over the", map[string]string{"blob": incompressible(2 * MaxArchiveBytes)}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			files := map[string]string{"compose.yaml": tc.compose}
			for k, v := range tc.extra {
				files[k] = v
			}
			_, err := Load(writeStack(t, "s", files), "containers/s/compose.yaml")
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("err = %v, want it to mention %q", err, tc.wantErr)
			}
		})
	}

	if _, err := Load(t.TempDir(), "containers/nope/compose.yaml"); err == nil || !strings.Contains(err.Error(), "not found in the content repo") {
		t.Errorf("missing compose file: err = %v", err)
	}
}

// incompressible is n bytes gzip can't shrink.
func incompressible(n int) string {
	b := make([]byte, n)
	x := uint32(2463534242)
	for i := range b {
		x ^= x << 13
		x ^= x >> 17
		x ^= x << 5
		b[i] = byte(x)
	}
	return string(b)
}

func TestProjectName(t *testing.T) {
	for name, want := range map[string]string{
		"flakyframes": "flakyframes", "Flaky Frames": "flaky-frames", "web_01": "web-01", "-x": "x", "!!!": "stack",
	} {
		if got := ProjectName(name); got != want {
			t.Errorf("ProjectName(%q) = %q, want %q", name, got, want)
		}
	}
}

func TestRegistryHost(t *testing.T) {
	for image, want := range map[string]string{
		"nginx": "", "nginx:alpine": "", "library/nginx": "", "myuser/app:1": "",
		"registry.internal/team/app:1.2": "registry.internal", "localhost/app": "localhost", "host:5000/app": "host:5000",
	} {
		if got := RegistryHost(image); got != want {
			t.Errorf("RegistryHost(%q) = %q, want %q", image, got, want)
		}
	}
}
