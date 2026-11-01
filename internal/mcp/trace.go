package mcp

import (
	"context"
	"database/sql/driver"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// A trace is everything one tool call did, in order: each database query
// (by its sqlc name, with arguments, rows and time), each read of a LaForge
// API page and each power request. It's stored with the call's audit row so
// the activity log can show what really happened.
type trace struct {
	mu      sync.Mutex
	start   time.Time
	steps   []traceStep
	dropped int
}

type traceStep struct {
	// AtMs is when the step started, from the start of the call.
	AtMs float64 `json:"at_ms"`
	Ms   float64 `json:"ms"`
	// Kind is sql, api, power or access.
	Kind   string `json:"kind"`
	Name   string `json:"name"`
	Detail string `json:"detail,omitempty"`
	Rows   *int64 `json:"rows,omitempty"`
	Status int    `json:"status,omitempty"`
	Bytes  int    `json:"bytes,omitempty"`
	// Response is a power request's answer (the per-host results).
	Response string `json:"response,omitempty"`
	Error    string `json:"error,omitempty"`
}

// maxTraceSteps bounds a trace; a call past it counts the rest as dropped.
const maxTraceSteps = 300

type traceKey struct{}

func withTrace(ctx context.Context, t *trace) context.Context {
	return context.WithValue(ctx, traceKey{}, t)
}

func traceFrom(ctx context.Context) *trace {
	t, _ := ctx.Value(traceKey{}).(*trace)
	return t
}

// add records a step that began at began and ends now. Safe on a nil trace.
func (t *trace) add(s traceStep, began time.Time) {
	if t == nil {
		return
	}
	now := time.Now()
	s.AtMs = ms(began.Sub(t.start))
	s.Ms = ms(now.Sub(began))
	t.mu.Lock()
	defer t.mu.Unlock()
	if len(t.steps) >= maxTraceSteps {
		t.dropped++
		return
	}
	t.steps = append(t.steps, s)
}

func (t *trace) snapshot() ([]traceStep, int) {
	if t == nil {
		return nil, 0
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	return append([]traceStep(nil), t.steps...), t.dropped
}

func ms(d time.Duration) float64 { return float64(d.Microseconds()) / 1000 }

// QueryTracer adds every query run on the MCP pool during a tool call to that
// call's trace. Queries outside a call (key checks, the audit insert) aren't
// traced.
type QueryTracer struct{}

type queryStartKey struct{}

type queryStart struct {
	began time.Time
	name  string
	args  string
}

func (QueryTracer) TraceQueryStart(ctx context.Context, _ *pgx.Conn, d pgx.TraceQueryStartData) context.Context {
	if traceFrom(ctx) == nil {
		return ctx
	}
	return context.WithValue(ctx, queryStartKey{}, queryStart{began: time.Now(), name: queryName(d.SQL), args: formatArgs(d.Args)})
}

func (QueryTracer) TraceQueryEnd(ctx context.Context, _ *pgx.Conn, d pgx.TraceQueryEndData) {
	t := traceFrom(ctx)
	st, ok := ctx.Value(queryStartKey{}).(queryStart)
	if t == nil || !ok {
		return
	}
	step := traceStep{Kind: "sql", Name: st.name, Detail: st.args}
	if d.Err != nil {
		step.Error = d.Err.Error()
	} else {
		n := d.CommandTag.RowsAffected()
		step.Rows = &n
	}
	t.add(step, st.began)
}

// queryName is a sqlc query's name, from the "-- name: X :one" line it
// starts with, or the start of the SQL for anything else.
func queryName(sql string) string {
	if rest, ok := strings.CutPrefix(strings.TrimSpace(sql), "-- name: "); ok {
		name, _, _ := strings.Cut(rest, " ")
		return name
	}
	sql = strings.Join(strings.Fields(sql), " ")
	if len(sql) > 80 {
		sql = sql[:80] + "…"
	}
	return sql
}

func formatArgs(args []any) string {
	if len(args) == 0 {
		return ""
	}
	parts := make([]string, len(args))
	for i, a := range args {
		parts[i] = fmt.Sprintf("$%d=%s", i+1, formatArg(a))
	}
	return strings.Join(parts, " ")
}

func formatArg(a any) string {
	var s string
	switch v := a.(type) {
	case nil:
		s = "NULL"
	case []byte:
		s = fmt.Sprintf("<%d bytes>", len(v))
	case []pgtype.UUID:
		ids := make([]string, len(v))
		for i, id := range v {
			ids[i] = formatArg(id)
		}
		s = "[" + strings.Join(ids, ",") + "]"
	case string:
		s = fmt.Sprintf("%q", v)
	case driver.Valuer:
		val, err := v.Value()
		switch {
		case err != nil:
			s = "?"
		case val == nil:
			s = "NULL"
		case isString(val):
			s = fmt.Sprintf("%q", val)
		default:
			s = fmt.Sprintf("%v", val)
		}
	default:
		s = fmt.Sprintf("%v", v)
	}
	if len(s) > 200 {
		s = s[:200] + "…"
	}
	return s
}

func isString(v any) bool { _, ok := v.(string); return ok }
