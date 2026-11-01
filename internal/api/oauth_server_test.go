package api

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"testing"
)

const testRedirect = "https://app.example/callback"

// oauthTestEnv is an API with OAuth on: an MCP handler set (any will do) and
// app.example as the one allowed redirect host.
func oauthTestEnv(t *testing.T) *apiTestEnv {
	t.Helper()
	env := setupAPITest(t)
	env.server.MCP = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusUnauthorized) })
	env.server.OAuthRedirectHosts = []string{"app.example"}
	return env
}

// anon is a client that doesn't follow redirects.
var anon = &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}

func registerClient(t *testing.T, env *apiTestEnv, body map[string]any) (int, map[string]any) {
	t.Helper()
	var out map[string]any
	status := doJSON(t, anon, http.MethodPost, env.httpURL+"/oauth/register", body, &out)
	if id, ok := out["client_id"].(string); ok {
		t.Cleanup(func() {
			env.server.Pool.Exec(context.Background(), `DELETE FROM api_key WHERE oauth_client_id = (SELECT id FROM oauth_client WHERE client_id = $1)`, id)
			env.server.Pool.Exec(context.Background(), `DELETE FROM oauth_client WHERE client_id = $1`, id)
		})
	}
	return status, out
}

func pkce(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

const testVerifier = "a-verifier-that-is-at-least-forty-three-characters-long"

// authorize starts a sign-in and returns where the API sent the browser.
func authorize(t *testing.T, env *apiTestEnv, params url.Values) *url.URL {
	t.Helper()
	resp, err := anon.Get(env.httpURL + "/oauth/authorize?" + params.Encode())
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusFound {
		t.Fatalf("authorize: status = %d, want 302", resp.StatusCode)
	}
	loc, err := resp.Location()
	if err != nil {
		t.Fatal(err)
	}
	return loc
}

func authorizeParams(env *apiTestEnv, clientID string) url.Values {
	return url.Values{
		"response_type": {"code"}, "client_id": {clientID}, "redirect_uri": {testRedirect},
		"code_challenge": {pkce(testVerifier)}, "code_challenge_method": {"S256"},
		"state": {"xyz"}, "scope": {"read"}, "resource": {env.httpURL + "/mcp"},
	}
}

// decide approves or denies a pending request as the signed-in person, from
// the UI's origin, and returns the redirect it answers with.
func decide(t *testing.T, env *apiTestEnv, browser *http.Client, requestID string, approve bool, origin string) (int, *url.URL) {
	t.Helper()
	req, _ := http.NewRequest(http.MethodPost, env.httpURL+"/oauth/requests/"+requestID, strings.NewReader(`{"approve":`+map[bool]string{true: "true", false: "false"}[approve]+`}`))
	req.Header.Set("Content-Type", "application/json")
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	resp, err := browser.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var out struct {
		RedirectTo string `json:"redirect_to"`
	}
	json.NewDecoder(resp.Body).Decode(&out)
	u, _ := url.Parse(out.RedirectTo)
	return resp.StatusCode, u
}

func exchange(t *testing.T, env *apiTestEnv, form url.Values) (int, map[string]any) {
	t.Helper()
	resp, err := anon.PostForm(env.httpURL+"/oauth/token", form)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var out map[string]any
	json.NewDecoder(resp.Body).Decode(&out)
	return resp.StatusCode, out
}

// signInCode runs authorize + approve and returns the one-time code.
func signInCode(t *testing.T, env *apiTestEnv, browser *http.Client, clientID string) string {
	t.Helper()
	loc := authorize(t, env, authorizeParams(env, clientID))
	if loc.Host != "ui.example" || loc.Path != "/oauth/consent" {
		t.Fatalf("authorize sent the browser to %s, want the consent page", loc)
	}
	status, back := decide(t, env, browser, loc.Query().Get("request"), true, "https://ui.example")
	if status != http.StatusOK {
		t.Fatalf("approve: status = %d", status)
	}
	if back.Host != "app.example" || back.Query().Get("state") != "xyz" || back.Query().Get("iss") != env.httpURL {
		t.Fatalf("approve redirect = %s, want app.example with state and iss", back)
	}
	return back.Query().Get("code")
}

func TestOAuthMetadata(t *testing.T) {
	env := oauthTestEnv(t)
	var prm map[string]any
	if s := doJSON(t, anon, http.MethodGet, env.httpURL+"/.well-known/oauth-protected-resource/mcp", nil, &prm); s != http.StatusOK {
		t.Fatalf("resource metadata: %d", s)
	}
	if prm["resource"] != env.httpURL+"/mcp" {
		t.Fatalf("resource = %v", prm["resource"])
	}
	var asm map[string]any
	doJSON(t, anon, http.MethodGet, env.httpURL+"/.well-known/oauth-authorization-server", nil, &asm)
	if asm["issuer"] != env.httpURL || asm["token_endpoint"] != env.httpURL+"/oauth/token" {
		t.Fatalf("auth server metadata = %v", asm)
	}
	if m, _ := asm["code_challenge_methods_supported"].([]any); len(m) != 1 || m[0] != "S256" {
		t.Fatalf("code_challenge_methods_supported = %v", asm["code_challenge_methods_supported"])
	}

	// Without MCP, there's nothing to sign in to.
	env.server.MCP = nil
	if s := doJSON(t, anon, http.MethodGet, env.httpURL+"/.well-known/oauth-authorization-server", nil, nil); s != http.StatusNotFound {
		t.Fatalf("metadata with MCP off: %d, want 404", s)
	}
}

func TestOAuthRegisterRefusesOtherRedirectHosts(t *testing.T) {
	env := oauthTestEnv(t)
	for _, uri := range []string{"https://evil.example/cb", "http://app.example/cb", "https://app.example:8443/cb", "https://app.example/cb#frag"} {
		status, out := registerClient(t, env, map[string]any{"client_name": "Claude", "redirect_uris": []string{uri}, "token_endpoint_auth_method": "none"})
		if status != http.StatusBadRequest || out["error"] != "invalid_redirect_uri" {
			t.Errorf("register %s: %d %v, want 400 invalid_redirect_uri", uri, status, out)
		}
	}
}

func TestOAuthSignInIssuesAConnectedAppKey(t *testing.T) {
	env := oauthTestEnv(t)
	browser, _ := apiKeyClient(t, env, "oauth-owner")

	status, reg := registerClient(t, env, map[string]any{"client_name": "ChatGPT", "redirect_uris": []string{testRedirect}, "token_endpoint_auth_method": "none"})
	if status != http.StatusCreated {
		t.Fatalf("register: %d %v", status, reg)
	}
	clientID := reg["client_id"].(string)
	if _, ok := reg["client_secret"]; ok {
		t.Fatal("a public client got a secret")
	}

	// The consent page's view needs a session, and approving needs the UI's origin.
	loc := authorize(t, env, authorizeParams(env, clientID))
	reqID := loc.Query().Get("request")
	if s := doJSON(t, anon, http.MethodGet, env.httpURL+"/oauth/requests/"+reqID, nil, nil); s != http.StatusUnauthorized {
		t.Fatalf("request view without a session: %d, want 401", s)
	}
	var view oauthRequestView
	if s := doJSON(t, browser, http.MethodGet, env.httpURL+"/oauth/requests/"+reqID, nil, &view); s != http.StatusOK || view.ClientName != "ChatGPT" || view.RedirectHost != "app.example" {
		t.Fatalf("request view: %d %+v", s, view)
	}
	if s, _ := decide(t, env, browser, reqID, true, "https://evil.example"); s != http.StatusForbidden {
		t.Fatalf("approve from another origin: %d, want 403", s)
	}

	code := signInCode(t, env, browser, clientID)
	form := url.Values{"grant_type": {"authorization_code"}, "code": {code}, "client_id": {clientID},
		"redirect_uri": {testRedirect}, "code_verifier": {testVerifier}, "resource": {env.httpURL + "/mcp"}}
	status, tok := exchange(t, env, form)
	if status != http.StatusOK {
		t.Fatalf("token: %d %v", status, tok)
	}
	key, _ := tok["access_token"].(string)
	if !strings.HasPrefix(key, "lfk_") || tok["token_type"] != "Bearer" || tok["scope"] != "read" {
		t.Fatalf("token response = %v", tok)
	}
	if _, ok := tok["refresh_token"]; ok {
		t.Fatal("a refresh token was issued")
	}

	// A code works once.
	if status, out := exchange(t, env, form); status != http.StatusBadRequest || out["error"] != "invalid_grant" {
		t.Fatalf("code reuse: %d %v, want invalid_grant", status, out)
	}

	// The key is a connected app on the API Keys page.
	var keys []apiKeyView
	doJSON(t, browser, http.MethodGet, env.httpURL+"/auth/api-keys", nil, &keys)
	if len(keys) != 1 || keys[0].Kind != "app" || keys[0].ClientHost != "app.example" || keys[0].Name != "ChatGPT" || keys[0].Status != "active" {
		t.Fatalf("keys = %+v", keys)
	}
	first := keys[0].ID

	// Connecting the same app again replaces its key.
	form.Set("code", signInCode(t, env, browser, clientID))
	if status, out := exchange(t, env, form); status != http.StatusOK {
		t.Fatalf("second token: %d %v", status, out)
	}
	keys = nil
	doJSON(t, browser, http.MethodGet, env.httpURL+"/auth/api-keys", nil, &keys)
	active := 0
	for _, k := range keys {
		if k.Status == "active" {
			active++
			if k.ID == first {
				t.Fatal("the first key is still active after reconnecting")
			}
		}
	}
	if len(keys) != 2 || active != 1 {
		t.Fatalf("after reconnecting: %d keys, %d active; want 2 and 1", len(keys), active)
	}
}

func TestOAuthTokenChecks(t *testing.T) {
	env := oauthTestEnv(t)
	browser, _ := apiKeyClient(t, env, "oauth-checks")
	_, reg := registerClient(t, env, map[string]any{"client_name": "Claude", "redirect_uris": []string{testRedirect}, "token_endpoint_auth_method": "client_secret_post"})
	clientID, secret := reg["client_id"].(string), reg["client_secret"].(string)
	good := func() url.Values {
		return url.Values{"grant_type": {"authorization_code"}, "code": {signInCode(t, env, browser, clientID)}, "client_id": {clientID},
			"client_secret": {secret}, "redirect_uri": {testRedirect}, "code_verifier": {testVerifier}}
	}

	cases := []struct {
		name   string
		change func(url.Values)
		status int
		code   string
	}{
		{"wrong verifier", func(f url.Values) { f.Set("code_verifier", strings.Repeat("x", 50)) }, 400, "invalid_grant"},
		{"wrong redirect", func(f url.Values) { f.Set("redirect_uri", "https://app.example/other") }, 400, "invalid_grant"},
		{"wrong secret", func(f url.Values) { f.Set("client_secret", "nope") }, 401, "invalid_client"},
		{"foreign resource", func(f url.Values) { f.Set("resource", "https://other.example/mcp") }, 400, "invalid_target"},
		{"refresh grant", func(f url.Values) { f.Set("grant_type", "refresh_token") }, 400, "unsupported_grant_type"},
	}
	for _, c := range cases {
		f := good()
		c.change(f)
		status, out := exchange(t, env, f)
		if status != c.status || out["error"] != c.code {
			t.Errorf("%s: %d %v, want %d %s", c.name, status, out, c.status, c.code)
		}
	}
	if status, out := exchange(t, env, good()); status != http.StatusOK {
		t.Fatalf("good exchange with a secret: %d %v", status, out)
	}
}

func TestOAuthAuthorizeRefusals(t *testing.T) {
	env := oauthTestEnv(t)
	browser, _ := apiKeyClient(t, env, "oauth-refusals")
	_, reg := registerClient(t, env, map[string]any{"client_name": "x", "redirect_uris": []string{testRedirect}, "token_endpoint_auth_method": "none"})
	clientID := reg["client_id"].(string)

	// Unknown client or redirect URI: an error page, never a redirect.
	for _, p := range []url.Values{
		func() url.Values { p := authorizeParams(env, "lfc_nope"); return p }(),
		func() url.Values {
			p := authorizeParams(env, clientID)
			p.Set("redirect_uri", "https://app.example/elsewhere")
			return p
		}(),
	} {
		resp, err := anon.Get(env.httpURL + "/oauth/authorize?" + p.Encode())
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("authorize %v: %d, want 400", p, resp.StatusCode)
		}
	}

	// A good client and redirect, but a bad request: back to the app with an error.
	for name, change := range map[string]func(url.Values){
		"no PKCE":          func(p url.Values) { p.Del("code_challenge") },
		"plain PKCE":       func(p url.Values) { p.Set("code_challenge_method", "plain") },
		"token response":   func(p url.Values) { p.Set("response_type", "token") },
		"foreign resource": func(p url.Values) { p.Set("resource", "https://other.example/mcp") },
	} {
		p := authorizeParams(env, clientID)
		change(p)
		loc := authorize(t, env, p)
		if loc.Host != "app.example" || loc.Query().Get("error") == "" || loc.Query().Get("state") != "xyz" {
			t.Errorf("%s: redirect = %s, want an error back to the app", name, loc)
		}
	}

	// Deny sends access_denied back, and the request can't be decided twice.
	loc := authorize(t, env, authorizeParams(env, clientID))
	reqID := loc.Query().Get("request")
	status, back := decide(t, env, browser, reqID, false, "https://ui.example")
	if status != http.StatusOK || back.Query().Get("error") != "access_denied" {
		t.Fatalf("deny: %d %s", status, back)
	}
	if status, _ := decide(t, env, browser, reqID, true, "https://ui.example"); status != http.StatusNotFound {
		t.Fatalf("approve after deny: %d, want 404", status)
	}
}

func TestSignInReturnTo(t *testing.T) {
	env := setupAPITest(t)
	for path, want := range map[string]string{
		"/oauth/consent?request=abc": "https://ui.example/oauth/consent?request=abc",
		"//evil.example/x":           "https://ui.example/",
		"https://evil.example/":      "https://ui.example/",
	} {
		client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
		client.Jar = newJar(t)
		start, err := client.Get(env.httpURL + "/auth/github/login?return_to=" + url.QueryEscape(path))
		if err != nil {
			t.Fatal(err)
		}
		start.Body.Close()
		loc, _ := start.Location()
		cb, err := client.Get(env.httpURL + "/auth/github/callback?code=return-to&state=" + loc.Query().Get("state"))
		if err != nil {
			t.Fatal(err)
		}
		cb.Body.Close()
		if got := cb.Header.Get("Location"); got != want {
			t.Errorf("return_to %q: landed on %q, want %q", path, got, want)
		}
	}
}

func newJar(t *testing.T) *cookiejar.Jar {
	t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	return jar
}
