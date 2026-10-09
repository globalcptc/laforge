// Package registryclient is a small Docker Registry HTTP API (v2) client, used
// to check that stored registry credentials actually work and to list the
// images a registry holds -- so an operator can confirm a login before a build
// depends on it, and a content author can see whether the image they pushed is
// actually there. It speaks only what those two jobs need: an authenticated
// GET, handling both Basic auth (self-hosted registries) and Bearer token auth
// (Docker Hub, GHCR, Harbor), plus the catalog and tags-list endpoints.
package registryclient

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client is stateless apart from its HTTP client; one can be shared.
type Client struct {
	HTTP *http.Client
}

func New() *Client {
	return &Client{HTTP: &http.Client{Timeout: 15 * time.Second}}
}

// ErrCatalogUnsupported is returned by Catalog when a registry doesn't
// implement the /v2/_catalog API (Docker Hub and GHCR, for example, don't) --
// the credentials can still be valid and pulls still work.
var ErrCatalogUnsupported = errors.New("this registry doesn't expose an image catalog (the /v2/_catalog API); Docker Hub and GHCR, for example, don't -- credentials can still be valid and images still pullable by exact name")

// ErrUnauthorized means the registry rejected the username/secret.
var ErrUnauthorized = errors.New("the registry rejected the credentials -- check the username and secret")

// ErrPullDenied means the credentials authenticate, but the account is not
// authorized to pull a given repository -- the registry issued a token with no
// pull access, so the request still came back 401/403. This is the signature of
// a valid login that lacks pull permission on the project/namespace (common with
// a Harbor robot account that was never granted Pull on the project). Distinct
// from ErrUnauthorized, which is a flat-out bad credential.
var ErrPullDenied = errors.New("authenticated, but not authorized to pull -- grant this account Pull permission on the registry/project")

// ErrPullUnverifiable means the credentials authenticate but pull access could
// not be probed automatically, because the registry exposes no catalog to pick
// a repository to test against (Docker Hub and GHCR, for example). The login is
// fine; pulls by exact image name may still work.
var ErrPullUnverifiable = errors.New("authenticated, but pull access couldn't be verified automatically (this registry exposes no catalog to pick a test repository)")

// baseURL turns a stored registry host into its v2 API base. It honors an
// explicit http:// (a plain-HTTP registry) and maps the Docker Hub aliases to
// the real endpoint; everything else defaults to HTTPS, matching how the
// deploy-time `docker login` reaches it.
func baseURL(host string) string {
	h := strings.TrimSuffix(strings.TrimSpace(host), "/")
	scheme := "https"
	switch {
	case strings.HasPrefix(h, "http://"):
		scheme, h = "http", strings.TrimPrefix(h, "http://")
	case strings.HasPrefix(h, "https://"):
		h = strings.TrimPrefix(h, "https://")
	}
	if h == "docker.io" || h == "index.docker.io" {
		h = "registry-1.docker.io"
	}
	return scheme + "://" + h
}

// Ping confirms the credentials are accepted: GET /v2/ must ultimately answer
// 200 after whatever auth the registry demands. A 401 that survives auth means
// the credentials are wrong; any other status is reported as-is.
func (c *Client) Ping(ctx context.Context, host, user, secret string) error {
	status, _, err := c.get(ctx, host, user, secret, "/v2/")
	if err != nil {
		return err
	}
	switch {
	case status == http.StatusOK:
		return nil
	case status == http.StatusUnauthorized:
		return ErrUnauthorized
	default:
		return fmt.Errorf("registry returned HTTP %d", status)
	}
}

type catalogResponse struct {
	Repositories []string `json:"repositories"`
}

// Catalog lists the repositories a registry holds. Returns ErrCatalogUnsupported
// for registries that don't implement it.
func (c *Client) Catalog(ctx context.Context, host, user, secret string) ([]string, error) {
	status, body, err := c.get(ctx, host, user, secret, "/v2/_catalog?n=1000")
	if err != nil {
		return nil, err
	}
	switch status {
	case http.StatusOK:
		var out catalogResponse
		if err := json.Unmarshal(body, &out); err != nil {
			return nil, fmt.Errorf("decoding catalog: %w", err)
		}
		return out.Repositories, nil
	case http.StatusNotFound:
		return nil, ErrCatalogUnsupported
	case http.StatusUnauthorized, http.StatusForbidden:
		// Some registries (Docker Hub, GHCR) answer the catalog path with a
		// 401/403 rather than 404 because they simply don't offer it to anyone.
		return nil, ErrCatalogUnsupported
	default:
		return nil, fmt.Errorf("registry returned HTTP %d for the catalog", status)
	}
}

type tagsResponse struct {
	Tags []string `json:"tags"`
}

// Tags lists the tags of one repository. Listing tags needs a
// repository:<repo>:pull token -- the same authorization a real `docker pull`
// needs -- so a 401/403 here is reported as ErrPullDenied (authenticated but not
// authorized to pull this repo), distinct from a transport or other HTTP error.
func (c *Client) Tags(ctx context.Context, host, user, secret, repo string) ([]string, error) {
	status, body, err := c.get(ctx, host, user, secret, "/v2/"+repo+"/tags/list")
	if err != nil {
		return nil, err
	}
	switch status {
	case http.StatusOK:
		var out tagsResponse
		if err := json.Unmarshal(body, &out); err != nil {
			return nil, fmt.Errorf("decoding tags: %w", err)
		}
		return out.Tags, nil
	case http.StatusUnauthorized, http.StatusForbidden:
		return nil, fmt.Errorf("%w (%s)", ErrPullDenied, repo)
	default:
		return nil, fmt.Errorf("registry returned HTTP %d listing tags for %s", status, repo)
	}
}

// VerifyPull confirms the credentials can actually PULL, not merely
// authenticate. It pings (to check the login), then picks a repository from the
// catalog and lists its tags -- which requires a repository:<repo>:pull token,
// exactly the authorization a deploy-time `docker pull` needs. On success it
// returns the repository it tested against. It returns ErrPullDenied when the
// login works but pull is refused, and ErrPullUnverifiable when the login works
// but there is no catalog (so no repo to probe) -- the caller treats the login
// as usable in that case, since pulls by exact name may still work.
func (c *Client) VerifyPull(ctx context.Context, host, user, secret string) (repo string, err error) {
	if err := c.Ping(ctx, host, user, secret); err != nil {
		return "", err
	}
	repos, err := c.Catalog(ctx, host, user, secret)
	switch {
	case errors.Is(err, ErrCatalogUnsupported):
		return "", ErrPullUnverifiable
	case err != nil:
		return "", err
	case len(repos) == 0:
		return "", ErrPullUnverifiable
	}
	repo = repos[0]
	if _, err := c.Tags(ctx, host, user, secret, repo); err != nil {
		return repo, err // ErrPullDenied for a 401/403, else the real error
	}
	return repo, nil
}

// get performs a GET against the registry's v2 API, transparently satisfying
// whichever auth the registry challenges with -- Basic (retry with the
// credentials) or Bearer (fetch a token from the named realm using the
// credentials, then retry). The token realm and scope come from the challenge
// itself, so this works for a catalog request (scope registry:catalog:*) and a
// tags request (repository:<repo>:pull) without special-casing either.
func (c *Client) get(ctx context.Context, host, user, secret, path string) (int, []byte, error) {
	full := baseURL(host) + path
	resp, body, err := c.rawGet(ctx, full, "")
	if err != nil {
		return 0, nil, err
	}
	if resp.StatusCode != http.StatusUnauthorized {
		return resp.StatusCode, body, nil
	}

	scheme, params := parseChallenge(resp.Header.Get("WWW-Authenticate"))
	switch strings.ToLower(scheme) {
	case "basic":
		resp2, body2, err := c.rawGet(ctx, full, "Basic "+basicAuth(user, secret))
		if err != nil {
			return 0, nil, err
		}
		return resp2.StatusCode, body2, nil
	case "bearer":
		token, err := c.fetchToken(ctx, params, user, secret)
		if err != nil {
			return 0, nil, err
		}
		if token == "" {
			return resp.StatusCode, body, nil // no token issued; surface the original 401
		}
		resp2, body2, err := c.rawGet(ctx, full, "Bearer "+token)
		if err != nil {
			return 0, nil, err
		}
		return resp2.StatusCode, body2, nil
	default:
		return resp.StatusCode, body, nil
	}
}

func (c *Client) rawGet(ctx context.Context, urlStr, authz string) (*http.Response, []byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
	if err != nil {
		return nil, nil, err
	}
	if authz != "" {
		req.Header.Set("Authorization", authz)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, nil, err
	}
	return resp, body, nil
}

// fetchToken redeems a Bearer challenge: GET the realm with the challenge's
// service and scope, authenticating with the stored credentials, and return
// the issued token.
func (c *Client) fetchToken(ctx context.Context, params map[string]string, user, secret string) (string, error) {
	realm := params["realm"]
	if realm == "" {
		return "", errors.New("registry asked for a bearer token but named no token realm")
	}
	u, err := url.Parse(realm)
	if err != nil {
		return "", fmt.Errorf("parsing token realm: %w", err)
	}
	q := u.Query()
	if svc := params["service"]; svc != "" {
		q.Set("service", svc)
	}
	if scope := params["scope"]; scope != "" {
		q.Set("scope", scope)
	}
	u.RawQuery = q.Encode()

	authz := ""
	if user != "" || secret != "" {
		authz = "Basic " + basicAuth(user, secret)
	}
	resp, body, err := c.rawGet(ctx, u.String(), authz)
	if err != nil {
		return "", err
	}
	if resp.StatusCode == http.StatusUnauthorized {
		return "", ErrUnauthorized
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("token endpoint returned HTTP %d", resp.StatusCode)
	}
	var tok struct {
		Token       string `json:"token"`
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(body, &tok); err != nil {
		return "", fmt.Errorf("decoding token response: %w", err)
	}
	if tok.Token != "" {
		return tok.Token, nil
	}
	return tok.AccessToken, nil
}

// parseChallenge splits a WWW-Authenticate header into its scheme and the
// comma-separated key="value" parameters a Bearer challenge carries.
func parseChallenge(header string) (scheme string, params map[string]string) {
	params = map[string]string{}
	header = strings.TrimSpace(header)
	if header == "" {
		return "", params
	}
	sp := strings.IndexByte(header, ' ')
	if sp < 0 {
		return header, params
	}
	scheme = header[:sp]
	for _, part := range splitParams(header[sp+1:]) {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}
		params[strings.TrimSpace(kv[0])] = strings.Trim(strings.TrimSpace(kv[1]), `"`)
	}
	return scheme, params
}

// splitParams splits on commas that aren't inside quotes (a scope value can
// itself contain commas).
func splitParams(s string) []string {
	var out []string
	var cur strings.Builder
	inQuote := false
	for _, r := range s {
		switch {
		case r == '"':
			inQuote = !inQuote
			cur.WriteRune(r)
		case r == ',' && !inQuote:
			out = append(out, strings.TrimSpace(cur.String()))
			cur.Reset()
		default:
			cur.WriteRune(r)
		}
	}
	if cur.Len() > 0 {
		out = append(out, strings.TrimSpace(cur.String()))
	}
	return out
}

func basicAuth(user, secret string) string {
	return base64.StdEncoding.EncodeToString([]byte(user + ":" + secret))
}
