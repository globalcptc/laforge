package registryclient

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const (
	testUser   = "alice"
	testSecret = "s3cret"
)

// basicRegistry is a fake registry that demands HTTP Basic auth, like a
// self-hosted Docker Distribution behind htpasswd.
func basicRegistry(t *testing.T, catalog []string, tags map[string][]string) *httptest.Server {
	t.Helper()
	ok := func(r *http.Request) bool {
		u, p, has := r.BasicAuth()
		return has && u == testUser && p == testSecret
	}
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !ok(r) {
			w.Header().Set("WWW-Authenticate", `Basic realm="registry"`)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		switch {
		case r.URL.Path == "/v2/":
			w.WriteHeader(http.StatusOK)
		case r.URL.Path == "/v2/_catalog":
			json.NewEncoder(w).Encode(catalogResponse{Repositories: catalog})
		case strings.HasSuffix(r.URL.Path, "/tags/list"):
			repo := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/v2/"), "/tags/list")
			json.NewEncoder(w).Encode(tagsResponse{Tags: tags[repo]})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func TestPingAndCatalogBasicAuth(t *testing.T) {
	srv := basicRegistry(t, []string{"team/web", "team/db"}, map[string][]string{
		"team/web": {"latest", "v1"},
	})
	defer srv.Close()
	c := New()
	host := httpHost(srv.URL)

	if err := c.Ping(context.Background(), host, testUser, testSecret); err != nil {
		t.Fatalf("Ping with good creds: %v", err)
	}
	if err := c.Ping(context.Background(), host, testUser, "wrong"); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("Ping with bad creds = %v, want ErrUnauthorized", err)
	}

	repos, err := c.Catalog(context.Background(), host, testUser, testSecret)
	if err != nil {
		t.Fatalf("Catalog: %v", err)
	}
	if len(repos) != 2 || repos[0] != "team/web" {
		t.Fatalf("Catalog = %v, want [team/web team/db]", repos)
	}
	tags, err := c.Tags(context.Background(), host, testUser, testSecret, "team/web")
	if err != nil {
		t.Fatalf("Tags: %v", err)
	}
	if len(tags) != 2 || tags[0] != "latest" {
		t.Fatalf("Tags = %v, want [latest v1]", tags)
	}
}

// TestBearerTokenAuth covers the Docker-Hub/GHCR-style flow: the registry
// answers 401 with a Bearer challenge naming a token realm; the client fetches
// a token there (authenticating with the credentials) and retries.
func TestBearerTokenAuth(t *testing.T) {
	var auth *httptest.Server
	issued := "tok-123"
	registry := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+issued {
			w.Header().Set("WWW-Authenticate", `Bearer realm="`+auth.URL+`/token",service="reg.example",scope="registry:catalog:*"`)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if r.URL.Path == "/v2/" {
			w.WriteHeader(http.StatusOK)
			return
		}
		if r.URL.Path == "/v2/_catalog" {
			json.NewEncoder(w).Encode(catalogResponse{Repositories: []string{"proj/app"}})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer registry.Close()
	auth = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, p, has := r.BasicAuth()
		if !has || u != testUser || p != testSecret {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		// Echo back the requested scope so we know it was forwarded.
		if r.URL.Query().Get("scope") == "" {
			t.Errorf("token request carried no scope")
		}
		json.NewEncoder(w).Encode(map[string]string{"token": issued})
	}))
	defer auth.Close()

	c := New()
	host := httpHost(registry.URL)
	if err := c.Ping(context.Background(), host, testUser, testSecret); err != nil {
		t.Fatalf("Ping (bearer): %v", err)
	}
	repos, err := c.Catalog(context.Background(), host, testUser, testSecret)
	if err != nil {
		t.Fatalf("Catalog (bearer): %v", err)
	}
	if len(repos) != 1 || repos[0] != "proj/app" {
		t.Fatalf("Catalog (bearer) = %v", repos)
	}
}

func TestCatalogUnsupported(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v2/" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound) // no catalog endpoint
	}))
	defer srv.Close()
	c := New()
	if _, err := c.Catalog(context.Background(), httpHost(srv.URL), "", ""); !errors.Is(err, ErrCatalogUnsupported) {
		t.Fatalf("Catalog on a registry without one = %v, want ErrCatalogUnsupported", err)
	}
}

func TestParseChallenge(t *testing.T) {
	scheme, p := parseChallenge(`Bearer realm="https://auth.example/token",service="reg",scope="repository:a/b:pull,push"`)
	if scheme != "Bearer" {
		t.Fatalf("scheme = %q", scheme)
	}
	if p["realm"] != "https://auth.example/token" || p["service"] != "reg" || p["scope"] != "repository:a/b:pull,push" {
		t.Fatalf("params = %#v (scope must survive its internal comma)", p)
	}
}

func TestBasicAuthHeader(t *testing.T) {
	got := basicAuth("u", "p")
	want := base64.StdEncoding.EncodeToString([]byte("u:p"))
	if got != want {
		t.Fatalf("basicAuth = %q, want %q", got, want)
	}
}

// httpHost turns an httptest URL (http://127.0.0.1:port) into the host form the
// client stores, keeping the explicit http:// so baseURL doesn't force HTTPS.
func httpHost(u string) string { return u }
