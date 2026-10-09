// Package microcloud is the MicroCloud builder -- its own implementation of the
// builder.Builder interface, SEPARATE from the Incus builder
// (internal/builder/incus). MicroCloud and Incus are distinct products:
// MicroCloud is Canonical's clustered stack (Ceph + OVN + LXD) and speaks the
// LXD REST API, while Incus is the LinuxContainers fork of LXD. They do not
// inherit from or wrap each other; they share only the builder.Builder
// interface (kind "microcloud" -> this package; kind "incus" -> a pool of
// internal/builder/incus.Builder). This package speaks the LXD REST API over
// mTLS; one Client talks to one MicroCloud cluster's single endpoint (any
// member answers for the whole cluster). The wire shapes here originated as a
// faithful copy of the proven Incus REST client and remain compatible because
// LXD and Incus share that REST lineage today -- but this is the MicroCloud/LXD
// builder, tracked and evolved on its own. Needs a smoke-test against a real
// MicroCloud cluster after the split.
package microcloud

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client talks to one Incus server. Every method call is scoped to
// Project (an Incus project -- see builder.go for why one project per
// team is the real isolation mechanism this builder uses).
type Client struct {
	BaseURL          string
	Project          string
	HTTPClient       *http.Client
	OperationTimeout time.Duration
}

// NewClient builds a Client that presents clientCertPEM/clientKeyPEM and
// trusts only serverCertPEM as the Incus server's identity -- "a pinned
// gateway public key/cert... baked in," the same pinning discipline the
// agent uses for the gateway, applied here to the builder's connection to
// the hoster. serverCertPEM is what a real builder config would store
// (fetched once, out of band, when the hoster was added -- see
// FetchServerCertificateInsecure for that one-time bootstrap step, never
// the steady-state path).
func NewClient(baseURL string, clientCertPEM, clientKeyPEM, serverCertPEM []byte, project string) (*Client, error) {
	cert, err := tls.X509KeyPair(clientCertPEM, clientKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("loading client keypair: %w", err)
	}
	pinnedBlock, _ := pem.Decode(serverCertPEM)
	if pinnedBlock == nil {
		return nil, fmt.Errorf("no certificate found in server cert PEM")
	}
	pinned := pinnedBlock.Bytes

	// A real Incus/MicroCloud server's certificate is self-signed and
	// typically has no meaningful hostname in it at all (a live daemon
	// tested this session presented a cert valid for its own container
	// ID, not "localhost" or any DNS name a client would actually dial)
	// -- exactly this case: "the server reaches
	// the MicroCloud cluster at https://<member>:8443 with a client
	// certificate," pinned by the server's own cert bytes, not by a
	// hostname match. Go's default verification checks BOTH chain-of-
	// trust AND hostname, so RootCAs+ServerName would reject a real
	// server exactly like this one -- found by running this against a
	// live daemon, not by reasoning about it in the abstract.
	// The textbook-correct fix for
	// certificate *pinning* (as opposed to CA-based trust) is to skip Go's
	// built-in verification and replace it with the actual, stronger
	// check pinning implies: is this exactly the one certificate we
	// trust, byte for byte -- which VerifyPeerCertificate below does.
	tlsCfg := &tls.Config{
		Certificates:       []tls.Certificate{cert},
		InsecureSkipVerify: true, //nolint:gosec // replaced by VerifyPeerCertificate's exact-byte pin, immediately below
		MinVersion:         tls.VersionTLS12,
		VerifyPeerCertificate: func(rawCerts [][]byte, _ [][]*x509.Certificate) error {
			for _, raw := range rawCerts {
				if bytes.Equal(raw, pinned) {
					return nil
				}
			}
			return fmt.Errorf("server presented a certificate that doesn't match the pinned one")
		},
	}
	return &Client{
		BaseURL: strings.TrimSuffix(baseURL, "/"),
		Project: project,
		// Deliberately no http.Client.Timeout here -- it would be an
		// absolute ceiling on every single request regardless of
		// context, including waitForOperation's own long-poll GET
		// (?timeout=N), silently overriding whatever OperationTimeout is
		// configured to. Found live against a real MicroCloud/Ceph
		// cluster: a
		// real VM's first image clone took well over two minutes
		// (network-attached Ceph RBD storage, not this package's own
		// fast local test daemon), and every deploy failed at exactly
		// 30s regardless of OperationTimeout being set to 120s or even
		// 300s explicitly -- an earlier version of this client set
		// Timeout: 30*time.Second right here. do and waitForOperation
		// each derive their own per-request context deadline instead
		// (context.WithTimeout), so a genuinely slow operation gets the
		// time OperationTimeout actually promises it, while an ordinary
		// call still fails fast on a truly dead connection.
		HTTPClient: &http.Client{Transport: &http.Transport{TLSClientConfig: tlsCfg}},
		// 5 minutes: a win2019 VM's first image clone onto network-attached
		// Ceph RBD genuinely runs past two minutes (found live -- deploys
		// failed at "context deadline exceeded" with the old 120s default),
		// and this budget only bounds waitForOperation's long-poll on async
		// operations, not ordinary calls (those use requestTimeout). A
		// builder config can raise it further via OperationTimeoutSeconds.
		OperationTimeout: 300 * time.Second,
	}, nil
}

// requestTimeout is do's own per-request deadline for everything that
// ISN'T waitForOperation's long-poll -- ordinary GET/POST/PUT/PATCH/DELETE
// calls, which should still fail fast on a truly dead connection rather
// than hang indefinitely just because the caller's own ctx has no
// deadline (context.Background(), the common case throughout this
// codebase's production callers).
const requestTimeout = 30 * time.Second

// FetchServerCertificateInsecure connects without verifying the server's
// identity and returns the certificate it presented, PEM-encoded -- the
// one-time, explicitly-insecure step a real onboarding flow uses to
// *obtain* the cert to pin, matching how `incus remote add` itself works
// on first contact with a new server. Never used by Client.do -- only to
// produce the serverCertPEM a real NewClient call then pins and verifies
// against on every subsequent connection.
func FetchServerCertificateInsecure(ctx context.Context, host string) ([]byte, error) {
	tlsCfg := &tls.Config{InsecureSkipVerify: true} //nolint:gosec // deliberate TOFU bootstrap, see doc comment
	conn, err := tls.DialWithDialer(&net.Dialer{Timeout: 10 * time.Second}, "tcp", host, tlsCfg)
	if err != nil {
		return nil, fmt.Errorf("connecting to %s: %w", host, err)
	}
	defer conn.Close()
	certs := conn.ConnectionState().PeerCertificates
	if len(certs) == 0 {
		return nil, fmt.Errorf("%s presented no certificate", host)
	}
	return pemEncodeCert(certs[0].Raw), nil
}

// apiResponse mirrors Incus's real response envelope -- verified
// field-for-field against a live daemon's actual JSON, not the API docs
// alone: {"type":"sync"|"async"|"error", "status", "status_code",
// "operation", "error_code", "error", "metadata"}.
type apiResponse struct {
	Type      string          `json:"type"`
	Operation string          `json:"operation"`
	ErrorCode int             `json:"error_code"`
	Error     string          `json:"error"`
	Metadata  json.RawMessage `json:"metadata"`
}

// APIError wraps a real Incus API error. AlreadyExists/NotFound match on
// the exact HTTP status and message text a live daemon was observed
// returning for these cases -- Incus has no dedicated machine-readable
// error code for "already exists," only a human-readable string, real
// error responses confirmed by hand against a live daemon:
//
//	create a network twice:  400, "The network already exists"
//	create an instance twice: 409 (surfaced via the operation wait), "... already exists"
//	delete something missing: 404, "... not found"
type APIError struct {
	HTTPStatus int
	ErrorCode  int
	Message    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("lxd api: http=%d code=%d: %s", e.HTTPStatus, e.ErrorCode, e.Message)
}

func (e *APIError) AlreadyExists() bool {
	return strings.Contains(e.Message, "already exists")
}

func (e *APIError) NotFound() bool {
	return e.HTTPStatus == http.StatusNotFound || strings.Contains(strings.ToLower(e.Message), "not found")
}

// AlreadyRunning matches the exact message a live daemon returns for
// PUT .../state {"action":"start"} on an instance that's already running
// -- the idempotency case for DeployHost/DeployContainer's start step.
func (e *APIError) AlreadyRunning() bool {
	return strings.Contains(e.Message, "already running")
}

// OVNUnavailable matches the exact message a live daemon with OVN's API
// extensions compiled in but no real OVN control plane running returns
// for a `type=ovn` network create -- confirmed live against this
// session's own test daemon (see Builder's own doc comment). Lets a test
// distinguish "this environment genuinely can't run OVN" (skip) from a
// real bug in the request this builder sends (fail).
func (e *APIError) OVNUnavailable() bool {
	return strings.Contains(e.Message, "OVN isn't currently available")
}

// Busy matches a real, transient race confirmed live once
// waitForOperation's own bug (see that function's doc comment) stopped
// silently swallowing async failures: an instance config update
// (removeNIC/restoreNIC's own PATCH/PUT) landing while the SAME
// instance still has another operation genuinely in flight -- observed
// live as "Failed to create instance update operation: Instance is busy
// running a \"stop\" operation" immediately after a close/open access
// cycle. Real and worth a bounded retry (the same idiom
// startInstance's own AlreadyRunning retry already uses), not a
// request-shape bug: the daemon is telling the truth about its own
// timing, not rejecting the request as malformed.
func (e *APIError) Busy() bool {
	return strings.Contains(e.Message, "is busy running")
}

func (c *Client) get(ctx context.Context, path string) (json.RawMessage, error) {
	return c.do(ctx, http.MethodGet, path, nil)
}
func (c *Client) post(ctx context.Context, path string, body interface{}) (json.RawMessage, error) {
	return c.do(ctx, http.MethodPost, path, body)
}
func (c *Client) put(ctx context.Context, path string, body interface{}) (json.RawMessage, error) {
	return c.do(ctx, http.MethodPut, path, body)
}
func (c *Client) patch(ctx context.Context, path string, body interface{}) (json.RawMessage, error) {
	return c.do(ctx, http.MethodPatch, path, body)
}
func (c *Client) delete(ctx context.Context, path string) (json.RawMessage, error) {
	return c.do(ctx, http.MethodDelete, path, nil)
}

type requestOptions struct {
	ifMatch string
	etag    *string
}

func (c *Client) do(ctx context.Context, method, path string, body interface{}, options ...requestOptions) (json.RawMessage, error) {
	// reqCtx, not a reassigned ctx: waitForOperation below needs the
	// ORIGINAL, undecorated ctx this function was called with, not this
	// request's own short-lived deadline -- a child context can never
	// outlive its parent's, so passing reqCtx into waitForOperation would
	// silently cap its own, much longer c.OperationTimeout+10s deadline
	// back down to requestTimeout (30s) regardless of what
	// OperationTimeout is configured to. Found exactly this way, live,
	// against a real MicroCloud/Ceph cluster -- see NewClient's own doc
	// comment on this same class of bug.
	timeout := requestTimeout
	// Some LXD cluster mutations (notably OVN network creation) complete
	// synchronously. Give them the configured operation budget too.
	if method != http.MethodGet && c.OperationTimeout > timeout {
		timeout = c.OperationTimeout
	}
	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	u := c.BaseURL + path
	if c.Project != "" {
		sep := "?"
		if strings.Contains(path, "?") {
			sep = "&"
		}
		u += sep + "project=" + url.QueryEscape(c.Project)
	}

	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(reqCtx, method, u, reqBody)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	if len(options) > 0 && options[0].ifMatch != "" {
		req.Header.Set("If-Match", options[0].ifMatch)
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s %s: %w", method, u, err)
	}
	defer resp.Body.Close()
	if len(options) > 0 && options[0].etag != nil {
		*options[0].etag = resp.Header.Get("ETag")
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var ar apiResponse
	if err := json.Unmarshal(data, &ar); err != nil {
		if resp.StatusCode >= 400 {
			return nil, &APIError{HTTPStatus: resp.StatusCode, ErrorCode: resp.StatusCode, Message: http.StatusText(resp.StatusCode)}
		}
		return nil, fmt.Errorf("decoding response from %s %s: %w (body: %s)", method, u, err, data)
	}
	if resp.StatusCode >= 400 {
		return nil, &APIError{HTTPStatus: resp.StatusCode, ErrorCode: ar.ErrorCode, Message: ar.Error}
	}

	switch ar.Type {
	case "error":
		return nil, &APIError{HTTPStatus: resp.StatusCode, ErrorCode: ar.ErrorCode, Message: ar.Error}
	case "async":
		return c.waitForOperation(ctx, ar.Operation)
	default: // "sync"
		return ar.Metadata, nil
	}
}

// waitForOperation blocks on an async operation (image pull, instance
// create/start/stop/delete -- everything that isn't instant) until it
// finishes, verified against a live daemon to return `type:"error"` at
// this same endpoint when the underlying task failed (e.g. a duplicate
// instance name), not merely a "sync" wrapper around a failed record.
// operationResult mirrors the real shape of an Incus operation's own
// metadata once /wait returns -- status/status_code/err describe the
// OPERATION's own outcome, a layer beneath apiResponse.Type, which the
// /wait endpoint always reports as "sync" the instant the operation
// finishes, succeeded or not. Verified against a live daemon's real
// response, not assumed: a VM create whose source image is larger than
// its requested disk size returns HTTP 200, apiResponse.Type=="sync",
// and this nested status_code==400 with a real, specific err message
// ("Source image size (53687091200) exceeds specified volume size
// (40000004096)") -- see waitForOperation's own doc comment for how this
// was found.
type operationResult struct {
	Status     string `json:"status"`
	StatusCode int    `json:"status_code"`
	Err        string `json:"err"`
}

// waitForOperation blocks on a real async Incus operation and returns
// its own result once it reaches a terminal state. Found live, not
// assumed: this used to only check the outer sync/async envelope's own
// Type field (apiResponse.Type == "error"), which the /wait endpoint
// itself never sets -- /wait's own response is always type "sync" once
// the operation stops running, REGARDLESS of whether the operation
// itself succeeded (status_code 200) or failed (400+, Incus's own
// convention -- see operationResult's doc comment). A real live test
// deploying a Windows VM whose disk size was smaller than its source
// image surfaced this directly: the daemon's own clear error ("Source
// image size... exceeds specified volume size...") was silently
// swallowed here, and the caller went on to try starting an instance
// that was never actually created, surfacing a confusing "instance not
// found" instead of the real, actionable cause. Every caller of do()/
// post()/put()/patch()/delete() that goes through an async operation
// (network/instance create, start/stop, ...) was affected equally.
func (c *Client) waitForOperation(ctx context.Context, opPath string) (result json.RawMessage, resultErr error) {
	terminal := false
	defer func() {
		if resultErr != nil && !terminal {
			resultErr = &pendingOperationError{Path: opPath, Err: resultErr}
		}
	}()
	// The client's own deadline must be strictly longer than what the
	// URL's ?timeout=N tells the SERVER to hold the long-poll for --
	// otherwise the client could give up and tear down the connection
	// right as the server was about to answer within its own budget. See
	// this function's own call site (NewClient's doc comment) for why
	// this can't just be c.HTTPClient's own Timeout field.
	ctx, cancel := context.WithTimeout(ctx, c.OperationTimeout+10*time.Second)
	defer cancel()
	opURL, err := url.Parse(opPath)
	if err != nil {
		return nil, fmt.Errorf("invalid operation URL: %w", err)
	}
	opURL.Path = strings.TrimRight(opURL.Path, "/") + "/wait"
	query := opURL.Query()
	query.Set("timeout", fmt.Sprint(int(c.OperationTimeout.Seconds())))
	if query.Get("project") == "" && c.Project != "" {
		query.Set("project", c.Project)
	}
	opURL.RawQuery = query.Encode()
	u := c.BaseURL + opURL.RequestURI()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("waiting for operation %s: %w", opPath, err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var ar apiResponse
	if err := json.Unmarshal(data, &ar); err != nil {
		if resp.StatusCode >= 400 {
			return nil, &APIError{HTTPStatus: resp.StatusCode, ErrorCode: resp.StatusCode, Message: http.StatusText(resp.StatusCode)}
		}
		return nil, fmt.Errorf("decoding operation-wait response: %w (body: %s)", err, data)
	}
	if ar.Type == "error" || resp.StatusCode >= 400 {
		return nil, &APIError{HTTPStatus: resp.StatusCode, ErrorCode: ar.ErrorCode, Message: ar.Error}
	}
	var op operationResult
	if err := json.Unmarshal(ar.Metadata, &op); err != nil {
		return nil, fmt.Errorf("decoding operation result: %w (body: %s)", err, ar.Metadata)
	}
	// Incus's own convention: 100-199 running, 200 success, 400+ failure/
	// cancelled -- mirrored from a live daemon's real response, not
	// documentation alone (see this function's own doc comment).
	if op.StatusCode >= 400 || op.Status == "Failure" || op.Status == "Cancelled" {
		terminal = true
		msg := op.Err
		if msg == "" {
			msg = op.Status
		}
		return nil, &APIError{HTTPStatus: resp.StatusCode, ErrorCode: op.StatusCode, Message: msg}
	}
	if op.StatusCode != 200 {
		return nil, fmt.Errorf("%w: %s (status %s, code %d)", errOperationRunning, opPath, op.Status, op.StatusCode)
	}
	return ar.Metadata, nil
}

func pemEncodeCert(der []byte) []byte {
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
}
