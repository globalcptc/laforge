package mcp

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// A request with no key gets a 401 that points at the OAuth metadata: that's
// how ChatGPT and Claude.ai find the sign-in.
func TestUnauthenticatedPointsAtResourceMetadata(t *testing.T) {
	const meta = "https://api.example/.well-known/oauth-protected-resource/mcp"
	h := New(Config{ResourceMetadataURL: meta}).Handler()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(`{}`)))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
	if got := rec.Header().Get("WWW-Authenticate"); !strings.Contains(got, `resource_metadata="`+meta+`"`) {
		t.Fatalf("WWW-Authenticate = %q, want resource_metadata", got)
	}
}
