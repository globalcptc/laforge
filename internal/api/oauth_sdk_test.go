package api

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/oauthex"

	"github.com/globalcptc/laforge/internal/mcp"
)

// The MCP SDK's own OAuth client, run against LaForge end to end: from the
// 401 on /mcp through discovery, registration, sign-in and the code exchange
// to an API key. If this passes, a spec-following client (ChatGPT, Claude)
// gets through the same way.
func TestOAuthWithTheMCPSDKClient(t *testing.T) {
	env := oauthTestEnv(t)
	env.server.MCP = mcp.New(mcp.Config{ResourceMetadataURL: env.httpURL + "/.well-known/oauth-protected-resource/mcp"}).Handler()
	browser, _ := apiKeyClient(t, env, "oauth-sdk")
	t.Cleanup(func() {
		env.server.Pool.Exec(context.Background(), `DELETE FROM oauth_client WHERE client_name = 'SDK Test'`)
	})

	h, err := auth.NewAuthorizationCodeHandler(&auth.AuthorizationCodeHandlerConfig{
		DynamicClientRegistrationConfig: &auth.DynamicClientRegistrationConfig{Metadata: &oauthex.ClientRegistrationMetadata{
			ClientName: "SDK Test", RedirectURIs: []string{testRedirect},
		}},
		RedirectURL: testRedirect,
		// The person: open the authorize URL, approve on the consent page,
		// and hand back what the app's redirect would receive.
		AuthorizationCodeFetcher: func(ctx context.Context, args *auth.AuthorizationArgs) (*auth.AuthorizationResult, error) {
			resp, err := browser.Get(args.URL)
			if err != nil {
				return nil, err
			}
			resp.Body.Close()
			consent, err := resp.Location()
			if err != nil {
				return nil, err
			}
			_, back := decide(t, env, browser, consent.Query().Get("request"), true, "https://ui.example")
			q := back.Query()
			return &auth.AuthorizationResult{Code: q.Get("code"), State: q.Get("state"), Iss: q.Get("iss")}, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	req, _ := http.NewRequest(http.MethodPost, env.httpURL+"/mcp", strings.NewReader(`{}`))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("/mcp without a key: %d, want 401", resp.StatusCode)
	}
	if err := h.Authorize(context.Background(), req, resp); err != nil {
		t.Fatalf("Authorize: %v", err)
	}
	ts, err := h.TokenSource(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	tok, err := ts.Token()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(tok.AccessToken, "lfk_") {
		t.Fatalf("access token = %q, want an lfk_ key", tok.AccessToken)
	}
}
