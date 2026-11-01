// OAuth sign-in for the MCP server (docs/plans/mcp-readonly.md §14). ChatGPT
// and Claude.ai can't send a pasted API key, so LaForge acts as their OAuth
// 2.1 authorization server: the client registers itself (RFC 7591), sends the
// person through GitHub sign-in and a consent page, and trades the one-time
// code (with PKCE) for an ordinary lfk_ API key. That key is what the MCP
// server then sees, so what a connected app can do -- and revoking it -- is
// exactly what it is for a pasted key.
//
// Deliberately small: authorization code + S256 only, no refresh tokens (the
// key lasts connectedAppLifetime), and pending sign-ins and codes live in
// memory, since there's one API process.
package api

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"log"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/globalcptc/laforge/internal/db"
)

const (
	// connectedAppLifetime is how long the key an OAuth sign-in hands out
	// lasts. There's no refresh: when it lapses, the app asks the person to
	// connect again.
	connectedAppLifetime = 30 * 24 * time.Hour
	// maxConnectedApps caps one person's live connected apps (apart from
	// their pasted keys).
	maxConnectedApps = 20

	oauthRequestTTL = 10 * time.Minute
	oauthCodeTTL    = 60 * time.Second
	// maxPendingOAuth bounds the in-memory pending sign-ins and codes.
	maxPendingOAuth = 2000

	// Open registration is capped per IP and in total.
	oauthRegisterPerMinute = 10
	maxUnusedOAuthClients  = 1000
	maxOAuthRedirectURIs   = 10
	maxOAuthClientName     = 100

	oauthScope = "read"
)

// defaultOAuthRedirectHosts are the hosts a client's redirect URI may point
// at: only the hosted apps that need OAuth. Anything else could be someone
// registering a client called "Claude" to phish a volunteer's consent.
var defaultOAuthRedirectHosts = []string{"claude.ai", "claude.com", "chatgpt.com"}

// oauthPending is a validated /oauth/authorize request waiting on consent.
type oauthPending struct {
	client      db.OauthClient
	redirectURI string
	state       string
	challenge   string
	expires     time.Time
}

// oauthCode is an approved request, waiting for the client to trade it in.
type oauthCode struct {
	client      db.OauthClient
	accountID   pgtype.UUID
	redirectURI string
	challenge   string
	expires     time.Time
}

// oauthState is the in-memory half of the server: pending requests by id,
// codes by hash, and recent registrations per IP. The zero value is ready.
type oauthState struct {
	mu        sync.Mutex
	pending   map[string]oauthPending
	codes     map[string]oauthCode
	registers map[string][]time.Time
}

func (o *oauthState) sweep(now time.Time) {
	for id, p := range o.pending {
		if now.After(p.expires) {
			delete(o.pending, id)
		}
	}
	for h, c := range o.codes {
		if now.After(c.expires) {
			delete(o.codes, h)
		}
	}
	for ip, ts := range o.registers {
		ts = slices.DeleteFunc(ts, func(t time.Time) bool { return now.Sub(t) > time.Minute })
		if len(ts) == 0 {
			delete(o.registers, ip)
		} else {
			o.registers[ip] = ts
		}
	}
}

func (o *oauthState) addPending(p oauthPending) (string, error) {
	id, err := randomToken()
	if err != nil {
		return "", err
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	o.sweep(time.Now())
	if o.pending == nil {
		o.pending = map[string]oauthPending{}
	}
	if len(o.pending) >= maxPendingOAuth {
		return "", errors.New("too many sign-ins in progress; try again in a few minutes")
	}
	o.pending[id] = p
	return id, nil
}

func (o *oauthState) getPending(id string) (oauthPending, bool) {
	o.mu.Lock()
	defer o.mu.Unlock()
	p, ok := o.pending[id]
	if !ok || time.Now().After(p.expires) {
		return oauthPending{}, false
	}
	return p, true
}

// takePending removes a pending request, so it can be decided only once.
func (o *oauthState) takePending(id string) (oauthPending, bool) {
	o.mu.Lock()
	defer o.mu.Unlock()
	p, ok := o.pending[id]
	delete(o.pending, id)
	if !ok || time.Now().After(p.expires) {
		return oauthPending{}, false
	}
	return p, true
}

func (o *oauthState) addCode(c oauthCode) (string, error) {
	code, err := randomToken()
	if err != nil {
		return "", err
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	o.sweep(time.Now())
	if o.codes == nil {
		o.codes = map[string]oauthCode{}
	}
	if len(o.codes) >= maxPendingOAuth {
		return "", errors.New("too many sign-ins in progress; try again in a few minutes")
	}
	o.codes[hashToken(code)] = c
	return code, nil
}

// takeCode removes a code: it works once, whatever happens next.
func (o *oauthState) takeCode(code string) (oauthCode, bool) {
	o.mu.Lock()
	defer o.mu.Unlock()
	h := hashToken(code)
	c, ok := o.codes[h]
	delete(o.codes, h)
	if !ok || time.Now().After(c.expires) {
		return oauthCode{}, false
	}
	return c, true
}

// allowRegister records a registration from ip, refusing past the per-minute cap.
func (o *oauthState) allowRegister(ip string) bool {
	o.mu.Lock()
	defer o.mu.Unlock()
	now := time.Now()
	o.sweep(now)
	if o.registers == nil {
		o.registers = map[string][]time.Time{}
	}
	if len(o.registers[ip]) >= oauthRegisterPerMinute {
		return false
	}
	o.registers[ip] = append(o.registers[ip], now)
	return true
}

// oauthIssuer is this authorization server's issuer, and mcpResource the one
// resource it issues keys for. Both come from PublicBaseURL; OAuth is off
// without it.
func (s *Server) oauthIssuer() string { return strings.TrimRight(s.PublicBaseURL, "/") }
func (s *Server) mcpResource() string { return s.oauthIssuer() + "/mcp" }

func (s *Server) oauthEnabled() bool { return s.MCP != nil && s.PublicBaseURL != "" }

func (s *Server) redirectHostAllowed(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.Fragment != "" || u.User != nil || u.Port() != "" {
		return false
	}
	hosts := s.OAuthRedirectHosts
	if hosts == nil {
		hosts = defaultOAuthRedirectHosts
	}
	return slices.Contains(hosts, strings.ToLower(u.Hostname()))
}

// redirectHost is the host part of a client's first redirect URI, shown so a
// person can see where they're sending access.
func redirectHost(uris []string) string {
	if len(uris) == 0 {
		return ""
	}
	u, err := url.Parse(uris[0])
	if err != nil {
		return ""
	}
	return u.Hostname()
}

// sameResource compares a client's resource parameter to /mcp: scheme and
// host case-insensitively, a trailing slash ignored.
func sameResource(a, b string) bool {
	ua, err1 := url.Parse(a)
	ub, err2 := url.Parse(b)
	if err1 != nil || err2 != nil {
		return false
	}
	return strings.EqualFold(ua.Scheme, ub.Scheme) && strings.EqualFold(ua.Host, ub.Host) &&
		strings.TrimRight(ua.Path, "/") == strings.TrimRight(ub.Path, "/") && ua.RawQuery == "" && ua.Fragment == ""
}

// serveOAuth answers the machine-facing OAuth endpoints, which sit ahead of
// the cookie CORS (like /mcp): any origin may read them, with no credentials.
// It reports whether the path was one of them.
func (s *Server) serveOAuth(w http.ResponseWriter, r *http.Request) bool {
	var h http.HandlerFunc
	switch r.URL.Path {
	case "/.well-known/oauth-protected-resource", "/.well-known/oauth-protected-resource/mcp":
		h = s.handleProtectedResourceMetadata
	case "/.well-known/oauth-authorization-server":
		h = s.handleAuthServerMetadata
	case "/oauth/register":
		h = s.handleOAuthRegister
	case "/oauth/token":
		h = s.handleOAuthToken
	case "/oauth/authorize":
		h = s.handleOAuthAuthorize
	default:
		return false
	}
	if !s.oauthEnabled() {
		writeError(w, http.StatusNotFound, errors.New("OAuth sign-in for the MCP server isn't enabled on this LaForge"))
		return true
	}
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, MCP-Protocol-Version")
		w.WriteHeader(http.StatusNoContent)
		return true
	}
	h(w, r)
	return true
}

func (s *Server) handleProtectedResourceMetadata(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		writeError(w, http.StatusMethodNotAllowed, errors.New("use GET"))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"resource":                 s.mcpResource(),
		"authorization_servers":    []string{s.oauthIssuer()},
		"scopes_supported":         []string{oauthScope},
		"bearer_methods_supported": []string{"header"},
		"resource_name":            "LaForge",
	})
}

func (s *Server) handleAuthServerMetadata(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		writeError(w, http.StatusMethodNotAllowed, errors.New("use GET"))
		return
	}
	iss := s.oauthIssuer()
	writeJSON(w, http.StatusOK, map[string]any{
		"issuer":                                iss,
		"authorization_endpoint":                iss + "/oauth/authorize",
		"token_endpoint":                        iss + "/oauth/token",
		"registration_endpoint":                 iss + "/oauth/register",
		"response_types_supported":              []string{"code"},
		"grant_types_supported":                 []string{"authorization_code"},
		"code_challenge_methods_supported":      []string{"S256"},
		"token_endpoint_auth_methods_supported": []string{"none", "client_secret_post", "client_secret_basic"},
		"scopes_supported":                      []string{oauthScope},
		// The iss parameter on every redirect back (RFC 9207). It lets
		// ChatGPT use its fixed redirect URI.
		"authorization_response_iss_parameter_supported": true,
	})
}

// oauthError is an OAuth error response (RFC 6749 §5.2, RFC 7591 §3.2.2).
func oauthError(w http.ResponseWriter, status int, code, description string) {
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, status, map[string]string{"error": code, "error_description": description})
}

// requestIP is the caller's address: Cloudflare's header when it's there
// (prod is behind Cloudflare), else the peer.
func requestIP(r *http.Request) string {
	if ip := strings.TrimSpace(r.Header.Get("CF-Connecting-IP")); ip != "" {
		return ip
	}
	return clientAddr(r)
}

type oauthRegisterRequest struct {
	ClientName              string   `json:"client_name"`
	RedirectURIs            []string `json:"redirect_uris"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
	GrantTypes              []string `json:"grant_types"`
	ResponseTypes           []string `json:"response_types"`
}

// handleOAuthRegister is dynamic client registration (RFC 7591).
func (s *Server) handleOAuthRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, errors.New("use POST"))
		return
	}
	ip := requestIP(r)
	if !s.oauth.allowRegister(ip) {
		w.Header().Set("Retry-After", "60")
		oauthError(w, http.StatusTooManyRequests, "invalid_request", "too many registrations; try again in a minute")
		return
	}
	var req oauthRegisterRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10)).Decode(&req); err != nil {
		oauthError(w, http.StatusBadRequest, "invalid_client_metadata", "the body must be JSON client metadata")
		return
	}
	if len(req.RedirectURIs) == 0 || len(req.RedirectURIs) > maxOAuthRedirectURIs {
		oauthError(w, http.StatusBadRequest, "invalid_redirect_uri", fmt.Sprintf("send 1 to %d redirect_uris", maxOAuthRedirectURIs))
		return
	}
	for _, u := range req.RedirectURIs {
		if !s.redirectHostAllowed(u) {
			oauthError(w, http.StatusBadRequest, "invalid_redirect_uri", fmt.Sprintf("%q isn't allowed: LaForge only signs in apps that redirect to an https URL on one of its allowed hosts", u))
			return
		}
	}
	for _, g := range req.GrantTypes {
		// refresh_token is accepted in the request (clients ask for it by
		// habit) but never issued; the response lists what is.
		if g != "authorization_code" && g != "refresh_token" {
			oauthError(w, http.StatusBadRequest, "invalid_client_metadata", fmt.Sprintf("grant type %q isn't supported", g))
			return
		}
	}
	for _, t := range req.ResponseTypes {
		if t != "code" {
			oauthError(w, http.StatusBadRequest, "invalid_client_metadata", fmt.Sprintf("response type %q isn't supported", t))
			return
		}
	}
	method := req.TokenEndpointAuthMethod
	if method == "" {
		method = "client_secret_basic" // RFC 7591's default
	}
	if method != "none" && method != "client_secret_post" && method != "client_secret_basic" {
		oauthError(w, http.StatusBadRequest, "invalid_client_metadata", fmt.Sprintf("token_endpoint_auth_method %q isn't supported", method))
		return
	}
	name := strings.TrimSpace(req.ClientName)
	if name == "" {
		name = redirectHost(req.RedirectURIs)
	}
	if utf8.RuneCountInString(name) > maxOAuthClientName {
		name = string([]rune(name)[:maxOAuthClientName])
	}

	unused, err := s.Queries.CountUnusedOAuthClients(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if unused >= maxUnusedOAuthClients {
		oauthError(w, http.StatusServiceUnavailable, "temporarily_unavailable", "too many registered clients; try again later")
		return
	}

	clientID, err := randomToken()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	clientID = "lfc_" + clientID[:32]
	var secret string
	var secretHash *string
	if method != "none" {
		if secret, err = randomToken(); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		h := hashToken(secret)
		secretHash = &h
	}
	c, err := s.Queries.CreateOAuthClient(r.Context(), db.CreateOAuthClientParams{
		ClientID: clientID, ClientName: name, RedirectUris: req.RedirectURIs, ClientSecretHash: secretHash,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	log.Printf("api: oauth client %q registered for %s (from %s)", name, redirectHost(req.RedirectURIs), ip)
	resp := map[string]any{
		"client_id":                  c.ClientID,
		"client_id_issued_at":        c.CreatedAt.Time.Unix(),
		"client_name":                c.ClientName,
		"redirect_uris":              c.RedirectUris,
		"grant_types":                []string{"authorization_code"},
		"response_types":             []string{"code"},
		"token_endpoint_auth_method": method,
		"scope":                      oauthScope,
	}
	if secret != "" {
		resp["client_secret"] = secret
		resp["client_secret_expires_at"] = 0
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusCreated, resp)
}

// oauthErrorPage is for authorize errors that mustn't be redirected (an
// unknown client or redirect URI): a plain page for the person to read.
func oauthErrorPage(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	fmt.Fprintf(w, `<!doctype html><meta charset="utf-8"><title>LaForge Sign-In</title>`+
		`<body style="font-family:system-ui,sans-serif;max-width:32rem;margin:4rem auto;padding:0 1rem">`+
		`<h1 style="font-size:1.25rem">Can't Connect To LaForge</h1><p>%s</p></body>`, html.EscapeString(msg))
}

// redirectWith is a client redirect URI with params added, plus iss (RFC 9207).
func (s *Server) redirectWith(redirectURI string, params url.Values) string {
	u, err := url.Parse(redirectURI)
	if err != nil {
		return redirectURI
	}
	q := u.Query()
	for k, vs := range params {
		for _, v := range vs {
			if v != "" {
				q.Set(k, v)
			}
		}
	}
	q.Set("iss", s.oauthIssuer())
	u.RawQuery = q.Encode()
	return u.String()
}

// validChallenge is a PKCE S256 challenge: 43 unpadded base64url characters.
func validChallenge(c string) bool {
	if len(c) != 43 {
		return false
	}
	_, err := base64.RawURLEncoding.DecodeString(c)
	return err == nil
}

// handleOAuthAuthorize checks an authorization request and sends the browser
// to the consent page (which handles sign-in). Nothing is shown or redirected
// until the client and its redirect URI check out.
func (s *Server) handleOAuthAuthorize(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		oauthErrorPage(w, http.StatusMethodNotAllowed, "Use GET.")
		return
	}
	q := r.URL.Query()
	client, err := s.Queries.GetOAuthClientByClientID(r.Context(), q.Get("client_id"))
	if errors.Is(err, pgx.ErrNoRows) {
		oauthErrorPage(w, http.StatusBadRequest, "This app isn't registered with LaForge. Remove the connection in the app and add it again.")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	redirectURI := q.Get("redirect_uri")
	if !slices.Contains(client.RedirectUris, redirectURI) || !s.redirectHostAllowed(redirectURI) {
		oauthErrorPage(w, http.StatusBadRequest, "This app asked to send you somewhere LaForge doesn't allow, so the sign-in was stopped.")
		return
	}
	state := q.Get("state")
	fail := func(code, desc string) {
		http.Redirect(w, r, s.redirectWith(redirectURI, url.Values{"error": {code}, "error_description": {desc}, "state": {state}}), http.StatusFound)
	}
	if q.Get("response_type") != "code" {
		fail("unsupported_response_type", "only response_type=code is supported")
		return
	}
	challenge := q.Get("code_challenge")
	if q.Get("code_challenge_method") != "S256" || !validChallenge(challenge) {
		fail("invalid_request", "PKCE is required: send a code_challenge with code_challenge_method=S256")
		return
	}
	if res := q.Get("resource"); res != "" && !sameResource(res, s.mcpResource()) {
		fail("invalid_target", "this server only issues access to "+s.mcpResource())
		return
	}
	id, err := s.oauth.addPending(oauthPending{
		client: client, redirectURI: redirectURI, state: state, challenge: challenge,
		expires: time.Now().Add(oauthRequestTTL),
	})
	if err != nil {
		fail("temporarily_unavailable", err.Error())
		return
	}
	http.Redirect(w, r, strings.TrimRight(s.UIBaseURL, "/")+"/oauth/consent?request="+url.QueryEscape(id), http.StatusFound)
}

// oauthRequestView is what the consent page shows about a pending request.
type oauthRequestView struct {
	ClientName    string `json:"client_name"`
	RedirectHost  string `json:"redirect_host"`
	ExpiresInDays int    `json:"expires_in_days"`
}

func (s *Server) handleGetOAuthRequest(w http.ResponseWriter, r *http.Request) {
	if _, err := s.sessionFromRequest(r); err != nil {
		writeAuthError(w, err)
		return
	}
	p, ok := s.oauth.getPending(r.PathValue("id"))
	if !ok {
		writeError(w, http.StatusNotFound, errors.New("this sign-in link expired; start again from the app"))
		return
	}
	writeJSON(w, http.StatusOK, oauthRequestView{
		ClientName: p.client.ClientName, RedirectHost: redirectHost([]string{p.redirectURI}),
		ExpiresInDays: int(connectedAppLifetime / (24 * time.Hour)),
	})
}

// handleDecideOAuthRequest is the consent page's Approve or Deny. It answers
// with the URL to send the browser to; the code itself is made here, and the
// key only when the app trades the code in.
func (s *Server) handleDecideOAuthRequest(w http.ResponseWriter, r *http.Request) {
	sess, err := s.sessionFromRequest(r)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	// Only the LaForge UI can approve: another site can't send this POST
	// with the person's cookie and a matching Origin.
	if origin := r.Header.Get("Origin"); s.UIBaseURL == "" || origin != strings.TrimRight(s.UIBaseURL, "/") {
		writeError(w, http.StatusForbidden, errors.New("approve from the LaForge consent page"))
		return
	}
	var body struct {
		Approve bool `json:"approve"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid body: %w", err))
		return
	}
	if body.Approve {
		active, err := s.Queries.CountActiveOAuthKeysByAccount(r.Context(), sess.AccountID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		if active >= maxConnectedApps {
			writeError(w, http.StatusConflict, fmt.Errorf("you already have %d connected apps; revoke one on the API Keys page first", maxConnectedApps))
			return
		}
	}
	p, ok := s.oauth.takePending(r.PathValue("id"))
	if !ok {
		writeError(w, http.StatusNotFound, errors.New("this sign-in link expired; start again from the app"))
		return
	}
	if !body.Approve {
		writeJSON(w, http.StatusOK, map[string]string{"redirect_to": s.redirectWith(p.redirectURI, url.Values{
			"error": {"access_denied"}, "error_description": {"the person declined"}, "state": {p.state},
		})})
		return
	}
	code, err := s.oauth.addCode(oauthCode{
		client: p.client, accountID: sess.AccountID, redirectURI: p.redirectURI, challenge: p.challenge,
		expires: time.Now().Add(oauthCodeTTL),
	})
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"redirect_to": s.redirectWith(p.redirectURI, url.Values{
		"code": {code}, "state": {p.state},
	})})
}

// clientCredentials reads the client id and secret from Basic auth or the form.
func clientCredentials(r *http.Request) (id, secret string, basic bool) {
	if u, p, ok := r.BasicAuth(); ok {
		id, _ = url.QueryUnescape(u)
		secret, _ = url.QueryUnescape(p)
		return id, secret, true
	}
	return r.PostFormValue("client_id"), r.PostFormValue("client_secret"), false
}

// handleOAuthToken trades an approved code for an API key.
func (s *Server) handleOAuthToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, errors.New("use POST"))
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	if err := r.ParseForm(); err != nil {
		oauthError(w, http.StatusBadRequest, "invalid_request", "send a form-encoded body")
		return
	}
	if gt := r.PostFormValue("grant_type"); gt != "authorization_code" {
		oauthError(w, http.StatusBadRequest, "unsupported_grant_type", "only authorization_code is supported; when the key expires, connect again")
		return
	}
	clientID, secret, _ := clientCredentials(r)
	client, err := s.Queries.GetOAuthClientByClientID(r.Context(), clientID)
	if errors.Is(err, pgx.ErrNoRows) {
		oauthError(w, http.StatusUnauthorized, "invalid_client", "unknown client")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if client.ClientSecretHash != nil && subtle.ConstantTimeCompare([]byte(hashToken(secret)), []byte(*client.ClientSecretHash)) != 1 {
		oauthError(w, http.StatusUnauthorized, "invalid_client", "wrong client secret")
		return
	}
	code, ok := s.oauth.takeCode(r.PostFormValue("code"))
	if !ok || code.client.ID != client.ID {
		oauthError(w, http.StatusBadRequest, "invalid_grant", "the code is unknown, expired or already used")
		return
	}
	if r.PostFormValue("redirect_uri") != code.redirectURI {
		oauthError(w, http.StatusBadRequest, "invalid_grant", "redirect_uri doesn't match the authorization request")
		return
	}
	verifier := r.PostFormValue("code_verifier")
	sum := sha256.Sum256([]byte(verifier))
	if len(verifier) < 43 || len(verifier) > 128 ||
		subtle.ConstantTimeCompare([]byte(base64.RawURLEncoding.EncodeToString(sum[:])), []byte(code.challenge)) != 1 {
		oauthError(w, http.StatusBadRequest, "invalid_grant", "code_verifier doesn't match the code_challenge")
		return
	}
	if res := r.PostFormValue("resource"); res != "" && !sameResource(res, s.mcpResource()) {
		oauthError(w, http.StatusBadRequest, "invalid_target", "this server only issues access to "+s.mcpResource())
		return
	}

	token, err := newAPIKey()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	expires := time.Now().Add(connectedAppLifetime)
	tx, err := s.Pool.Begin(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	defer tx.Rollback(r.Context())
	q := s.Queries.WithTx(tx)
	// Connecting the same app again replaces its earlier key.
	replaced, err := q.RevokeOAuthKeysForClient(r.Context(), db.RevokeOAuthKeysForClientParams{AccountID: code.accountID, OauthClientID: client.ID})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if _, err := q.CreateOAuthAPIKey(r.Context(), db.CreateOAuthAPIKeyParams{
		AccountID: code.accountID, Name: client.ClientName, Prefix: token[:apiKeyDisplayLen], TokenHash: hashToken(token),
		ExpiresAt: pgtype.Timestamptz{Time: expires, Valid: true}, OauthClientID: client.ID,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if replaced > 0 && s.MCPKeysChanged != nil {
		s.MCPKeysChanged()
	}
	log.Printf("api: oauth key issued to %q for account %s", client.ClientName, code.accountID.String())
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	writeJSON(w, http.StatusOK, map[string]any{
		"access_token": token,
		"token_type":   "Bearer",
		"expires_in":   int(time.Until(expires).Seconds()),
		"scope":        oauthScope,
	})
}
