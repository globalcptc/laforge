package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/globalcptc/laforge/internal/db"
)

// errDenied is a request for something the key can't read. Tools report it
// the same way whether the thing exists or not, so a key can't probe for
// builds it has no access to.
var errDenied = errors.New("not found, or this API key can't read it")

// errRateLimited is a call refused by a rate limit.
var errRateLimited = errors.New("rate limited")

// call is one tool call in progress: who's calling, and what they can read.
type call struct {
	s        *Server
	key      apiKey
	readable map[pgtype.UUID]bool
	// buildID is recorded on the audit row when the tool resolved one.
	buildID pgtype.UUID

	// What the activity log shows when the row is opened.
	trace  *trace
	timing callTiming
	// result is the encoded result, kept for the audit's preview.
	result []byte
	// phase is where a failed call stopped.
	phase string
}

// callTiming is how long each phase of a call took, in milliseconds.
type callTiming struct {
	QueueMs  float64 `json:"queue_ms"`
	AccessMs float64 `json:"access_ms"`
	RunMs    float64 `json:"run_ms"`
	EncodeMs float64 `json:"encode_ms"`
	TotalMs  float64 `json:"total_ms"`
}

func (c *call) q() *db.Queries { return c.s.cfg.Queries }

// canRead reports whether the key can read the repository.
func (c *call) canRead(repo pgtype.UUID) bool { return c.readable[repo] }

var uuidPrefix = regexp.MustCompile(`^[0-9a-fA-F-]{4,36}$`)

// build resolves a full build id or a unique prefix of one ("9531b5b4") to a
// build the key can read.
func (c *call) build(ctx context.Context, ref string) (pgtype.UUID, error) {
	ref = strings.TrimSpace(strings.ToLower(ref))
	if !uuidPrefix.MatchString(ref) {
		return pgtype.UUID{}, fmt.Errorf("build_id %q isn't a build id or a prefix of one", ref)
	}
	var id pgtype.UUID
	if len(ref) == 36 {
		if err := id.Scan(ref); err != nil {
			return pgtype.UUID{}, fmt.Errorf("build_id %q isn't a valid id", ref)
		}
	} else {
		ids, err := c.q().McpFindBuildsByIDPrefix(ctx, ref)
		if err != nil {
			return pgtype.UUID{}, err
		}
		switch len(ids) {
		case 0:
			return pgtype.UUID{}, errDenied
		case 2:
			return pgtype.UUID{}, fmt.Errorf("build_id %q matches more than one build; give more of the id", ref)
		}
		id = ids[0]
	}
	repo, err := c.q().McpGetBuildRepository(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return pgtype.UUID{}, errDenied
	}
	if err != nil {
		return pgtype.UUID{}, err
	}
	if !c.canRead(repo) {
		return pgtype.UUID{}, errDenied
	}
	c.buildID = id
	return id, nil
}

// readOnly are the annotations every tool carries.
func readOnly(title string) *sdk.ToolAnnotations {
	f := false
	return &sdk.ToolAnnotations{Title: title, ReadOnlyHint: true, IdempotentHint: true, DestructiveHint: &f, OpenWorldHint: &f}
}

// addTool registers a tool behind the checks every call goes through: a
// verified key, the per-owner rate limit, the global concurrency cap, a
// timeout, the result size cap, and an audit row.
func addTool[In, Out any](s *Server, name, title, description string, h func(context.Context, *call, In) (Out, error)) {
	addToolWith(s, &sdk.Tool{Name: name, Title: title, Description: description, Annotations: readOnly(title)}, h)
}

// destructive are the annotations of the one tool that changes something
// (power), so clients ask their user before running it.
func destructive(title string) *sdk.ToolAnnotations {
	t := true
	return &sdk.ToolAnnotations{Title: title, ReadOnlyHint: false, IdempotentHint: false, DestructiveHint: &t, OpenWorldHint: &t}
}

func addToolWith[In, Out any](s *Server, tool *sdk.Tool, h func(context.Context, *call, In) (Out, error)) {
	name := tool.Name
	sdk.AddTool(s.sdk, tool, func(ctx context.Context, req *sdk.CallToolRequest, in In) (*sdk.CallToolResult, Out, error) {
		var zero Out
		started := time.Now()
		var key apiKey
		if req.Extra != nil && req.Extra.TokenInfo != nil {
			key, _ = req.Extra.TokenInfo.Extra[tokenInfoKey].(apiKey)
		}
		if !key.ID.Valid {
			// RequireBearerToken guards the handler; reaching here without a
			// key means it was mounted without it.
			return nil, zero, errors.New("unauthenticated")
		}
		c := &call{s: s, key: key, trace: &trace{start: started}}
		ctx = withTrace(ctx, c.trace)
		out, size, status, err := runCall(ctx, s, c, func(ctx context.Context) (Out, error) { return h(ctx, c, in) })
		c.timing.TotalMs = ms(time.Since(started))
		s.audit(req, c, name, in, status, err, time.Since(started), size)
		if err != nil {
			return nil, zero, err
		}
		return nil, out, nil
	})
}

// runCall applies the limits around one tool body, and says how it went:
// "ok", "denied", "error" or "rate_limited".
func runCall[Out any](ctx context.Context, s *Server, c *call, body func(context.Context) (Out, error)) (Out, int, string, error) {
	var zero Out
	if ok, wait := s.lim.allow(c.key.AccountID, time.Now()); !ok {
		c.phase = "rate limit"
		return zero, 0, "rate_limited", fmt.Errorf("rate limited: too many calls from this account; retry in %s", wait.Round(100*time.Millisecond)+100*time.Millisecond)
	}
	queued := time.Now()
	queue, cancel := context.WithTimeout(ctx, s.cfg.QueueWait)
	defer cancel()
	select {
	case s.sem <- struct{}{}:
		defer func() { <-s.sem }()
	case <-queue.Done():
		c.timing.QueueMs = ms(time.Since(queued))
		c.phase = "queue"
		return zero, 0, "rate_limited", errors.New("LaForge is busy serving other agents; retry in a few seconds")
	}
	c.timing.QueueMs = ms(time.Since(queued))
	ctx, cancelCall := context.WithTimeout(ctx, s.cfg.CallTimeout)
	defer cancelCall()

	checked := time.Now()
	readable, err := s.readableRepos(ctx, c.key)
	c.timing.AccessMs = ms(time.Since(checked))
	step := traceStep{Kind: "access", Name: "repositories this key can read"}
	if err != nil {
		step.Error = err.Error()
	} else {
		step.Detail = fmt.Sprintf("%d repositories", len(readable))
	}
	c.trace.add(step, checked)
	if err != nil {
		c.phase = "access check"
		return zero, 0, "error", fmt.Errorf("checking access: %w", err)
	}
	c.readable = readable

	ran := time.Now()
	out, err := body(ctx)
	c.timing.RunMs = ms(time.Since(ran))
	if err != nil {
		c.phase = "tool"
	}
	if errors.Is(err, errDenied) {
		return zero, 0, "denied", err
	}
	if errors.Is(err, errRateLimited) {
		return zero, 0, "rate_limited", err
	}
	if err != nil {
		return zero, 0, "error", err
	}
	encoded := time.Now()
	b, err := json.Marshal(out)
	c.timing.EncodeMs = ms(time.Since(encoded))
	if err != nil {
		c.phase = "encoding the result"
		return zero, 0, "error", err
	}
	c.result = b
	if len(b) > maxResultBytes {
		c.phase = "result size"
		return zero, len(b), "error", fmt.Errorf("the result is %d KB, over the %d KB limit; narrow it with the tool's filters or a smaller limit", len(b)>>10, maxResultBytes>>10)
	}
	return out, len(b), "ok", nil
}

// audit records the call. Best effort: a failed insert is logged, never
// returned to the caller.
func (s *Server) audit(req *sdk.CallToolRequest, c *call, tool string, in any, status string, callErr error, took time.Duration, size int) {
	fullArgs, _ := json.Marshal(in)
	args := fullArgs
	if len(args) > 2048 {
		args, _ = json.Marshal(map[string]string{"truncated": string(args[:2000])})
	}
	errText := ""
	if callErr != nil {
		errText = callErr.Error()
	}
	var h http.Header
	if req.Extra != nil {
		h = req.Extra.Header
	}
	remote, ua := clientIP(h), h.Get("User-Agent")
	detail, err := json.Marshal(buildDetail(c, tool, fullArgs, h, status, errText, size))
	if err != nil {
		log.Printf("mcp: encoding audit detail: %v", err)
		detail = nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := s.cfg.Queries.McpInsertAudit(ctx, db.McpInsertAuditParams{
		ApiKeyID: c.key.ID, AccountID: c.key.AccountID, Tool: tool, Args: args, BuildID: c.buildID,
		Status: status, Error: errText, DurationMs: int32(took.Milliseconds()), ResultBytes: int32(size),
		RemoteAddr: remote, UserAgent: ua, Detail: detail,
	}); err != nil {
		log.Printf("mcp: writing audit row: %v", err)
	}
	if remote == "" {
		remote = "unknown"
	}
	build := ""
	if c.buildID.Valid {
		build = " build=" + c.buildID.String()[:8]
	}
	log.Printf("mcp: %s%s -> %s in %s (by %s key %s… from %s)", tool, build, status, took.Round(time.Millisecond), c.key.Login, c.key.Prefix, remote)
}

// clientIP is the caller's address as Cloudflare/Caddy report it.
func clientIP(h http.Header) string {
	if h == nil {
		return ""
	}
	if v := h.Get("CF-Connecting-IP"); v != "" {
		return v
	}
	if v := h.Get("X-Forwarded-For"); v != "" {
		first, _, _ := strings.Cut(v, ",")
		return strings.TrimSpace(first)
	}
	if v := h.Get("X-Real-IP"); v != "" {
		if host, _, err := net.SplitHostPort(v); err == nil {
			return host
		}
		return v
	}
	return ""
}
