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
	"errors"
	"fmt"
	"io"
	"log/slog"
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
	NetworkLock      NetworkLocker
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
		// time OperationTimeout actually promises it. Ordinary calls also
		// honor that budget: cluster reads can exceed 30 seconds under load.
		HTTPClient: &http.Client{Transport: &http.Transport{TLSClientConfig: tlsCfg}},
		// Busy Ceph clusters and large image copies can take several minutes.
		// The builder can override this budget; caller cancellation always wins.
		OperationTimeout: DefaultOperationTimeout,
	}, nil
}

// DefaultOperationTimeout bounds cluster reads and operation waits when unset.
const DefaultOperationTimeout = 10 * time.Minute

// requestTimeout is the minimum per-request budget when a client does not
// configure a longer cluster operation timeout. Every request remains bounded,
// even when its caller has no deadline.
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

// getRaw does a GET and returns the untouched response body, for endpoints that
// don't speak the LXD JSON envelope -- notably /1.0/metrics, which is OpenMetrics
// text. The normal do()/get() path would fail trying to json.Unmarshal it.
func (c *Client) getRaw(ctx context.Context, path string) ([]byte, error) {
	reqCtx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()
	u := c.BaseURL + path
	if c.Project != "" {
		sep := "?"
		if strings.Contains(path, "?") {
			sep = "&"
		}
		u += sep + "project=" + url.QueryEscape(c.Project)
	}
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GET %s: %w", u, err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: HTTP %d", u, resp.StatusCode)
	}
	return data, nil
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

// do is every request this client makes. Each one is logged -- method, path,
// how long the hoster took, and what came back (sync, an async operation and
// how that ended, or an error) -- because when a deploy or a rebuild stalls,
// the question is always "is the cluster slow, is it us, or is nothing
// happening at all?", and only the request log answers it.
func (c *Client) do(ctx context.Context, method, path string, body interface{}, options ...requestOptions) (json.RawMessage, error) {
	started := time.Now()
	// A request that's slow to answer is said while it's slow, not only when
	// it ends: a hung request is otherwise invisible until its timeout.
	stop := logWaitProgress(ctx, "MicroCloud", method+" "+path, started)
	defer stop()
	result, err := c.doLogged(ctx, method, path, body, options...)
	logRequest(ctx, "MicroCloud", method, path, started, err)
	return result, err
}

// logRequest is the one line per hoster API request (see Client.do).
func logRequest(ctx context.Context, hoster, method, path string, started time.Time, err error) {
	elapsed := time.Since(started).Round(time.Millisecond)
	switch {
	case err == nil:
		slog.InfoContext(ctx, hoster+" API request", "method", method, "path", path, "elapsed", elapsed)
	case errors.Is(err, context.Canceled):
		slog.WarnContext(ctx, hoster+" API request cancelled", "method", method, "path", path, "elapsed", elapsed)
	default:
		slog.WarnContext(ctx, hoster+" API request failed", "method", method, "path", path, "elapsed", elapsed, "error", err)
	}
}

func (c *Client) doLogged(ctx context.Context, method, path string, body interface{}, options ...requestOptions) (json.RawMessage, error) {
	if isNetworkingMutation(method, path) {
		lockStarted := time.Now()
		lockedCtx, release, err := c.lockNetworking(ctx)
		if err != nil {
			slog.WarnContext(ctx, "MicroCloud networking lock not acquired", "method", method, "path", path, "waited", time.Since(lockStarted).Round(time.Millisecond), "error", err)
			return nil, err
		}
		if waited := time.Since(lockStarted); waited > time.Second {
			slog.InfoContext(ctx, "MicroCloud networking lock acquired after a wait", "method", method, "path", path, "waited", waited.Round(time.Millisecond))
		}
		defer release()
		ctx = lockedCtx
	}
	// Keep the original caller context for waitForOperation. Submission
	// and waiting each need their own bounded budget; time spent submitting
	// must not consume the operation wait's budget. Both still obey the
	// caller's overall deadline and cancellation.
	timeout := requestTimeout
	// Cluster reads can also be slow: a live OVN network read succeeded
	// after the old fixed 30-second deadline had repeatedly cancelled it.
	// Use the configured budget for reads and synchronous mutations alike.
	if c.OperationTimeout > timeout {
		timeout = c.OperationTimeout
	}
	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	reqStarted := time.Now()
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
		slog.InfoContext(ctx, "MicroCloud API accepted an async operation", "method", method, "path", path, "operation", ar.Operation, "submit_elapsed", time.Since(reqStarted).Round(time.Millisecond))
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
	started := time.Now()
	// Say so while the wait is long: a single long-poll can sit for minutes
	// with nothing else in the log, which looks exactly like a hang.
	stopProgress := logWaitProgress(ctx, "MicroCloud", opPath, started)
	defer func() {
		stopProgress()
		if resultErr != nil && !terminal {
			resultErr = &pendingOperationError{Path: opPath, Err: resultErr}
		}
		elapsed := time.Since(started).Round(time.Millisecond)
		switch {
		case resultErr == nil:
			slog.InfoContext(ctx, "MicroCloud operation finished", "operation", opPath, "elapsed", elapsed)
		case terminal:
			slog.WarnContext(ctx, "MicroCloud operation failed", "operation", opPath, "elapsed", elapsed, "error", resultErr)
		default:
			slog.WarnContext(ctx, "MicroCloud operation still unfinished when the wait ended", "operation", opPath, "elapsed", elapsed, "wait_limit", c.OperationTimeout, "error", resultErr)
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

// logWaitProgress logs every 30 seconds that an operation is still being
// waited on, until the returned stop is called. Without it a slow cluster
// looks like a stalled runner.
func logWaitProgress(ctx context.Context, hoster, opPath string, started time.Time) (stop func()) {
	done := make(chan struct{})
	go func() {
		t := time.NewTicker(30 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-done:
				return
			case <-ctx.Done():
				return
			case <-t.C:
				slog.InfoContext(ctx, hoster+" still waiting on the hoster", "for", opPath, "waited", time.Since(started).Round(time.Second))
			}
		}
	}()
	return func() { close(done) }
}
