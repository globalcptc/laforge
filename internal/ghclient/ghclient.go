// Package ghclient is the one place laforge talks to GitHub's own API:
// reading a repository's permissions and CI results, posting LaForge's own
// validation result back as a commit status, and the OAuth device flow the
// CLI uses to log a person in. Nothing here is GitHub-App-specific --
// every call takes the caller's own token, so the same client works
// whether that token came from device-flow login or (for local dev, since
// no OAuth App is registered yet) a
// plain personal access token dropped in an env var.
//
// Every endpoint is reachable through APIBaseURL / AuthBaseURL rather than
// a hardcoded https://api.github.com / https://github.com, specifically so
// tests can point this client at an httptest.Server and exercise the real
// request/response handling without a live GitHub account -- which is
// exactly how ghclient_test.go verifies it.
package ghclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	defaultAPIBaseURL  = "https://api.github.com"
	defaultAuthBaseURL = "https://github.com"
)

type Client struct {
	APIBaseURL  string
	AuthBaseURL string
	HTTPClient  *http.Client

	// sleep is used only between device-flow polls, pluggable so tests
	// don't burn real wall-clock time waiting out GitHub's interval.
	sleep func(time.Duration)
}

func New() *Client {
	return &Client{
		APIBaseURL:  defaultAPIBaseURL,
		AuthBaseURL: defaultAuthBaseURL,
		HTTPClient:  http.DefaultClient,
		sleep:       time.Sleep,
	}
}

// APIError is returned for any non-2xx response from the REST API, keeping
// GitHub's own status code and body text available rather than collapsing
// every failure into one opaque error string.
type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("github api: %d: %s", e.StatusCode, e.Body)
}

func (c *Client) doJSON(ctx context.Context, method, url, token string, body, out interface{}) error {
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reqBody = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &APIError{StatusCode: resp.StatusCode, Body: string(respBody)}
	}
	if out != nil {
		if err := json.Unmarshal(respBody, out); err != nil {
			return fmt.Errorf("decoding response: %w (body: %s)", err, respBody)
		}
	}
	return nil
}

// --- repository ---

type RepoPermissions struct {
	Admin bool `json:"admin"`
	Push  bool `json:"push"`
	Pull  bool `json:"pull"`
}

// Repo is deliberately a small projection of GitHub's real repository
// object -- only the fields "GitHub-based authorization" and "repo
// registration" actually need.
type Repo struct {
	// ID is only populated by calls that actually return it (GitHub's
	// installation-repositories listing does; GetRepo's own trimmed
	// projection never needed it before ListInstallationRepositories,
	// which is why every other caller of Repo leaves it zero) -- real for
	// "which repository is this" when it's set, not otherwise relied on.
	ID            int64           `json:"id"`
	FullName      string          `json:"full_name"`
	DefaultBranch string          `json:"default_branch"`
	Private       bool            `json:"private"`
	CloneURL      string          `json:"clone_url"`
	Permissions   RepoPermissions `json:"permissions"`
}

// GetRepo fetches a repository using the caller's own token. GitHub
// includes a `permissions` object scoped to whichever token made the
// request -- reading that directly is how authorization is enforced (see
// internal/api's auth middleware): "if you can push to the repo, you can
// build its environments" is answered by this one field, no separate
// permission table needed.
func (c *Client) GetRepo(ctx context.Context, token, owner, repo string) (*Repo, error) {
	u := fmt.Sprintf("%s/repos/%s/%s", c.APIBaseURL, url.PathEscape(owner), url.PathEscape(repo))
	var out Repo
	if err := c.doJSON(ctx, http.MethodGet, u, token, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

type User struct {
	ID        int64  `json:"id"`
	Login     string `json:"login"`
	AvatarURL string `json:"avatar_url"`
}

// GetAuthenticatedUser resolves whose token this is -- used right after
// device-flow login to know who just signed in, without asking them.
func (c *Client) GetAuthenticatedUser(ctx context.Context, token string) (*User, error) {
	var out User
	if err := c.doJSON(ctx, http.MethodGet, c.APIBaseURL+"/user", token, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetUserByLogin resolves an arbitrary GitHub username to their real,
// numeric id -- used when granting repository_access to someone by
// username (the "Adding someone means finding their GitHub
// identity"), who may never have signed in to LaForge yet, so there's no
// existing account row (and thus no known id) to grant against.
func (c *Client) GetUserByLogin(ctx context.Context, token, login string) (*User, error) {
	var out User
	if err := c.doJSON(ctx, http.MethodGet, c.APIBaseURL+"/users/"+login, token, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Collaborator is one person with access to a repository and what GitHub
// lets them do there. RoleName is GitHub's own word for it ("admin",
// "maintain", "write", "triage", "read", or a custom role's name).
type Collaborator struct {
	ID          int64           `json:"id"`
	Login       string          `json:"login"`
	AvatarURL   string          `json:"avatar_url"`
	RoleName    string          `json:"role_name"`
	Permissions RepoPermissions `json:"permissions"`
}

// ListCollaborators lists everyone with access to a repository --
// outside collaborators, and for an organization's repository its members
// with access directly, through a team, or through the base permission
// (affiliation=all). Works with an installation token (the App's
// metadata read permission covers it) or a user token with push access.
func (c *Client) ListCollaborators(ctx context.Context, token, owner, repo string) ([]Collaborator, error) {
	var all []Collaborator
	for page := 1; ; page++ {
		u := fmt.Sprintf("%s/repos/%s/%s/collaborators?affiliation=all&per_page=100&page=%d", c.APIBaseURL, owner, repo, page)
		var out []Collaborator
		if err := c.doJSON(ctx, http.MethodGet, u, token, nil, &out); err != nil {
			return nil, err
		}
		all = append(all, out...)
		if len(out) < 100 {
			return all, nil
		}
	}
}

// --- GitHub App installation repositories ---

type listInstallationRepositoriesResponse struct {
	TotalCount   int    `json:"total_count"`
	Repositories []Repo `json:"repositories"`
}

// ListInstallationRepositories is the real fix for "an 'all repositories'
// GitHub App install never lists its repos in the installation webhook
// payload at all": GET
// /installation/repositories, authenticated AS the installation itself
// (an installation access token, not the app JWT) -- the one real way to
// discover what an "all repositories" install actually covers, since the
// webhook payload's own `repositories` array is only ever populated for
// a "selected repositories" install.
func (c *Client) ListInstallationRepositories(ctx context.Context, installationToken string) ([]Repo, error) {
	var all []Repo
	for page := 1; ; page++ {
		u := fmt.Sprintf("%s/installation/repositories?per_page=100&page=%d", c.APIBaseURL, page)
		var out listInstallationRepositoriesResponse
		if err := c.doJSON(ctx, http.MethodGet, u, installationToken, nil, &out); err != nil {
			return nil, err
		}
		all = append(all, out.Repositories...)
		if len(out.Repositories) < 100 {
			return all, nil
		}
	}
}

// --- branches ---

// Branch is a small projection of GitHub's real branch object -- just
// enough for "which branches exist, and what commit is each one at right
// now" (a configured build's own branch picker, and resolving a branch's
// current tip to fetch a real checkout for it).
type Branch struct {
	Name   string `json:"name"`
	Commit struct {
		SHA string `json:"sha"`
	} `json:"commit"`
}

// ListBranches lists every real branch of a repository, paginating past
// GitHub's default 30-per-page (a real repo commonly has more than 30
// branches over its lifetime, even if only a handful are ever built).
func (c *Client) ListBranches(ctx context.Context, token, owner, repo string) ([]Branch, error) {
	var all []Branch
	for page := 1; ; page++ {
		u := fmt.Sprintf("%s/repos/%s/%s/branches?per_page=100&page=%d", c.APIBaseURL, url.PathEscape(owner), url.PathEscape(repo), page)
		var out []Branch
		if err := c.doJSON(ctx, http.MethodGet, u, token, nil, &out); err != nil {
			return nil, err
		}
		all = append(all, out...)
		if len(out) < 100 {
			return all, nil
		}
	}
}

// GetBranch resolves one specific branch by name -- used to find its
// current tip commit without paginating through every branch just to
// find one.
func (c *Client) GetBranch(ctx context.Context, token, owner, repo, branch string) (*Branch, error) {
	u := fmt.Sprintf("%s/repos/%s/%s/branches/%s", c.APIBaseURL, url.PathEscape(owner), url.PathEscape(repo), url.PathEscape(branch))
	var out Branch
	if err := c.doJSON(ctx, http.MethodGet, u, token, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// --- CI status gating ---

// CombinedStatus mirrors GET /repos/{owner}/{repo}/commits/{ref}/status,
// the legacy Status API. State is one of "success"/"pending"/"failure"/
// "error"; a repo with no statuses at all reports "pending" with zero
// statuses, per GitHub's own documented behavior for that endpoint.
type CombinedStatus struct {
	State      string `json:"state"`
	TotalCount int    `json:"total_count"`
}

func (c *Client) GetCombinedStatus(ctx context.Context, token, owner, repo, ref string) (*CombinedStatus, error) {
	u := fmt.Sprintf("%s/repos/%s/%s/commits/%s/status", c.APIBaseURL, url.PathEscape(owner), url.PathEscape(repo), url.PathEscape(ref))
	var out CombinedStatus
	if err := c.doJSON(ctx, http.MethodGet, u, token, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

type CheckRun struct {
	Name       string `json:"name"`
	Status     string `json:"status"`     // queued | in_progress | completed
	Conclusion string `json:"conclusion"` // success | failure | neutral | cancelled | timed_out | action_required | skipped | stale (only meaningful once status == completed)
}

type checkRunsResponse struct {
	TotalCount int        `json:"total_count"`
	CheckRuns  []CheckRun `json:"check_runs"`
}

// ListCheckRuns is the modern counterpart to GetCombinedStatus -- GitHub
// Actions and most third-party CI report through Checks, not the legacy
// Status API, so "CI passes" (see internal/ingest) has to look at both to
// mean what the plan says: "every check on that commit is green."
func (c *Client) ListCheckRuns(ctx context.Context, token, owner, repo, ref string) ([]CheckRun, error) {
	u := fmt.Sprintf("%s/repos/%s/%s/commits/%s/check-runs", c.APIBaseURL, url.PathEscape(owner), url.PathEscape(repo), url.PathEscape(ref))
	var out checkRunsResponse
	if err := c.doJSON(ctx, http.MethodGet, u, token, nil, &out); err != nil {
		return nil, err
	}
	return out.CheckRuns, nil
}

// CreateStatus posts LaForge's own validation result back to the commit,
// "PR check style" per the plan -- so a pull request shows pass/fail from
// LaForge the same way it shows any other CI check, before anyone can
// build off it.
func (c *Client) CreateStatus(ctx context.Context, token, owner, repo, sha, state, description, context_ string) error {
	u := fmt.Sprintf("%s/repos/%s/%s/statuses/%s", c.APIBaseURL, url.PathEscape(owner), url.PathEscape(repo), url.PathEscape(sha))
	body := map[string]string{"state": state, "description": description, "context": context_}
	return c.doJSON(ctx, http.MethodPost, u, token, body, nil)
}

// --- OAuth device flow (RFC 8628, as GitHub implements it) ---
// https://docs.github.com/en/apps/oauth-apps/building-oauth-apps/authorizing-oauth-apps#device-flow

type DeviceCodeResponse struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete"`
	ExpiresIn               int    `json:"expires_in"`
	Interval                int    `json:"interval"`
}

// RequestDeviceCode starts the device flow: GitHub returns a code for the
// CLI to poll with and a short user_code + URL to show the person, who
// enters it in a browser on a device that already has one (their phone,
// their normal browser -- the whole point of device flow on a CLI).
func (c *Client) RequestDeviceCode(ctx context.Context, clientID string, scopes []string) (*DeviceCodeResponse, error) {
	form := url.Values{"client_id": {clientID}}
	if len(scopes) > 0 {
		form.Set("scope", strings.Join(scopes, " "))
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.AuthBaseURL+"/login/device/code", strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &APIError{StatusCode: resp.StatusCode, Body: string(respBody)}
	}
	var out DeviceCodeResponse
	if err := json.Unmarshal(respBody, &out); err != nil {
		return nil, fmt.Errorf("decoding device code response: %w (body: %s)", err, respBody)
	}
	return &out, nil
}

type deviceTokenResponse struct {
	AccessToken string `json:"access_token"`
	// Present when the GitHub App has "Expire user authorization tokens" on:
	// the access token lasts expires_in seconds and refresh_token (good for
	// refresh_token_expires_in seconds) trades for a fresh one. Absent (zero)
	// when the App doesn't expire user tokens, in which case no refresh is ever
	// needed.
	RefreshToken          string `json:"refresh_token"`
	ExpiresIn             int    `json:"expires_in"`
	RefreshTokenExpiresIn int    `json:"refresh_token_expires_in"`
	Error                 string `json:"error"`
	ErrorDesc             string `json:"error_description"`
}

// OAuthToken is a user access token and, when the App expires user tokens, the
// refresh token and lifetimes (in seconds) needed to keep it fresh.
type OAuthToken struct {
	AccessToken           string
	RefreshToken          string
	ExpiresIn             int
	RefreshTokenExpiresIn int
}

// ErrAuthorizationDenied is returned when the person declines the sign-in
// on GitHub's side, distinct from a timeout or a transport error so the
// CLI can print the right message for each.
var ErrAuthorizationDenied = fmt.Errorf("authorization denied")

// ErrDeviceCodeExpired is returned once GitHub's expires_in window closes
// without the person completing sign-in.
var ErrDeviceCodeExpired = fmt.Errorf("device code expired before authorization completed")

// PollForToken polls GitHub's token endpoint at the interval GitHub asked
// for (widening it if told to slow down) until the person finishes signing
// in, the code expires, or they decline. This is the actual wait loop
// behind `laforge login` -- ctx cancellation (e.g. the person hits Ctrl-C)
// stops it immediately rather than waiting out the full interval.
func (c *Client) PollForToken(ctx context.Context, clientID, deviceCode string, interval, expiresIn int) (OAuthToken, error) {
	deadline := time.Now().Add(time.Duration(expiresIn) * time.Second)
	wait := time.Duration(interval) * time.Second
	if wait <= 0 {
		wait = 5 * time.Second
	}
	for {
		if time.Now().After(deadline) {
			return OAuthToken{}, ErrDeviceCodeExpired
		}
		select {
		case <-ctx.Done():
			return OAuthToken{}, ctx.Err()
		default:
		}
		c.sleep(wait)

		form := url.Values{
			"client_id":   {clientID},
			"device_code": {deviceCode},
			"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.AuthBaseURL+"/login/oauth/access_token", strings.NewReader(form.Encode()))
		if err != nil {
			return OAuthToken{}, err
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Accept", "application/json")
		resp, err := c.HTTPClient.Do(req)
		if err != nil {
			return OAuthToken{}, err
		}
		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return OAuthToken{}, err
		}
		var out deviceTokenResponse
		if err := json.Unmarshal(respBody, &out); err != nil {
			return OAuthToken{}, fmt.Errorf("decoding token response: %w (body: %s)", err, respBody)
		}
		switch out.Error {
		case "":
			if out.AccessToken == "" {
				return OAuthToken{}, fmt.Errorf("github returned no access_token and no error (body: %s)", respBody)
			}
			return OAuthToken{
				AccessToken:           out.AccessToken,
				RefreshToken:          out.RefreshToken,
				ExpiresIn:             out.ExpiresIn,
				RefreshTokenExpiresIn: out.RefreshTokenExpiresIn,
			}, nil
		case "authorization_pending":
			continue
		case "slow_down":
			wait += 5 * time.Second
			continue
		case "expired_token":
			return OAuthToken{}, ErrDeviceCodeExpired
		case "access_denied":
			return OAuthToken{}, ErrAuthorizationDenied
		default:
			return OAuthToken{}, fmt.Errorf("github device token error: %s: %s", out.Error, out.ErrorDesc)
		}
	}
}

// --- OAuth web application flow ---
// https://docs.github.com/en/apps/oauth-apps/building-oauth-apps/authorizing-oauth-apps#web-application-flow
//
// Device flow (above) is what the CLI uses -- no browser redirect
// available. A browser UI has one, so it uses the standard web flow
// instead: redirect to AuthorizeURL, GitHub redirects back with a `code`,
// ExchangeCode trades that for a real access token in one request, no
// polling. Same reasoning as device flow's own doc comment: this needs a
// registered OAuth App's client ID and secret, which this session doesn't
// have -- ExchangeCode itself is real
// and tested against a fake server exactly like PollForToken is, but a
// live browser round trip through github.com couldn't be exercised here.

// AuthorizeURL is the address to redirect the browser to. state is an
// opaque, caller-generated value (CSRF protection) that GitHub echoes
// back unchanged on the callback -- the caller must verify it matches
// what it handed out before trusting the returned code.
func (c *Client) AuthorizeURL(clientID, redirectURI, state string, scopes []string) string {
	q := url.Values{
		"client_id":    {clientID},
		"redirect_uri": {redirectURI},
		"state":        {state},
	}
	if len(scopes) > 0 {
		q.Set("scope", strings.Join(scopes, " "))
	}
	return c.AuthBaseURL + "/login/oauth/authorize?" + q.Encode()
}

// ExchangeCode trades the callback's `code` for a real access token (and, when
// the App expires user tokens, the refresh token + lifetimes) -- the one
// request that completes the web flow, no polling needed (unlike device flow,
// the person has already approved by the time this runs).
func (c *Client) ExchangeCode(ctx context.Context, clientID, clientSecret, code, redirectURI string) (OAuthToken, error) {
	return c.postOAuthToken(ctx, url.Values{
		"client_id":     {clientID},
		"client_secret": {clientSecret},
		"code":          {code},
		"redirect_uri":  {redirectURI},
	})
}

// RefreshUserToken trades a refresh token for a fresh access token (GitHub
// rotates the refresh token too, returning a new one), keeping a long LaForge
// session's short-lived GitHub token alive without re-prompting the person.
func (c *Client) RefreshUserToken(ctx context.Context, clientID, clientSecret, refreshToken string) (OAuthToken, error) {
	return c.postOAuthToken(ctx, url.Values{
		"client_id":     {clientID},
		"client_secret": {clientSecret},
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
	})
}

// postOAuthToken POSTs the token endpoint and parses the shared
// {access_token, refresh_token, expires_in, ...} / {error} response both the
// code exchange and the refresh use.
func (c *Client) postOAuthToken(ctx context.Context, form url.Values) (OAuthToken, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.AuthBaseURL+"/login/oauth/access_token", strings.NewReader(form.Encode()))
	if err != nil {
		return OAuthToken{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return OAuthToken{}, err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return OAuthToken{}, err
	}
	var out deviceTokenResponse
	if err := json.Unmarshal(respBody, &out); err != nil {
		return OAuthToken{}, fmt.Errorf("decoding token response: %w (body: %s)", err, respBody)
	}
	if out.Error != "" {
		return OAuthToken{}, fmt.Errorf("github oauth error: %s: %s", out.Error, out.ErrorDesc)
	}
	if out.AccessToken == "" {
		return OAuthToken{}, fmt.Errorf("github returned no access_token and no error (body: %s)", respBody)
	}
	return OAuthToken{
		AccessToken:           out.AccessToken,
		RefreshToken:          out.RefreshToken,
		ExpiresIn:             out.ExpiresIn,
		RefreshTokenExpiresIn: out.RefreshTokenExpiresIn,
	}, nil
}
