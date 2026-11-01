package mcp

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"sort"
	"time"
	"unicode/utf8"
)

// auditDetail is the full record of one call, stored as mcp_audit.detail and
// shown when its row on the activity log is opened.
type auditDetail struct {
	Request detailRequest `json:"request"`
	Key     detailKey     `json:"key"`
	Client  detailClient  `json:"client"`
	Timing  callTiming    `json:"timing"`
	Trace   []traceStep   `json:"trace"`
	Dropped int           `json:"trace_dropped,omitempty"`
	Result  detailResult  `json:"result"`
	Outcome detailOutcome `json:"outcome"`
	Server  detailServer  `json:"server"`
}

type detailRequest struct {
	Tool string `json:"tool"`
	// Arguments are the call's full arguments (Truncated when over the cap).
	Arguments      json.RawMessage `json:"arguments,omitempty"`
	ArgumentsBytes int             `json:"arguments_bytes"`
	Truncated      bool            `json:"arguments_truncated,omitempty"`
	BuildID        string          `json:"build_id,omitempty"`
}

type detailKey struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Prefix    string    `json:"prefix"`
	Owner     string    `json:"owner"`
	ExpiresAt time.Time `json:"expires_at"`
	LimitedTo []string  `json:"limited_to,omitempty"`
	Readable  int       `json:"readable_repositories"`
}

type detailClient struct {
	IP              string            `json:"ip,omitempty"`
	UserAgent       string            `json:"user_agent,omitempty"`
	ProtocolVersion string            `json:"protocol_version,omitempty"`
	SessionID       string            `json:"session_id,omitempty"`
	Headers         map[string]string `json:"headers,omitempty"`
}

type detailResult struct {
	Bytes int `json:"bytes"`
	// Lists counts the entries of each list at the top of the result.
	Lists map[string]int `json:"lists,omitempty"`
	// Clipped counts long strings cut to max_field_chars.
	Clipped          int    `json:"clipped_fields,omitempty"`
	Preview          string `json:"preview,omitempty"`
	PreviewTruncated bool   `json:"preview_truncated,omitempty"`
}

type detailOutcome struct {
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
	// Phase is where a failed call stopped: rate limit, queue, access check,
	// tool, encoding the result or result size.
	Phase string `json:"phase,omitempty"`
}

type detailServer struct {
	Host string `json:"host,omitempty"`
}

const (
	maxDetailArgs    = 64 << 10
	maxDetailPreview = 32 << 10
)

// Request headers worth keeping, beyond the ones with their own field.
// Authorization is never kept.
var detailHeaders = []string{"X-Forwarded-For", "X-Real-IP", "CF-Ray", "CF-IPCountry", "Accept", "Content-Length", "Origin"}

var hostname, _ = os.Hostname()

func buildDetail(c *call, tool string, args []byte, h http.Header, status, errText string, size int) auditDetail {
	steps, dropped := c.trace.snapshot()
	d := auditDetail{
		Request: detailRequest{Tool: tool, ArgumentsBytes: len(args)},
		Key: detailKey{
			ID: c.key.ID.String(), Name: c.key.Name, Prefix: c.key.Prefix, Owner: c.key.Login,
			ExpiresAt: c.key.ExpiresAt, Readable: len(c.readable),
		},
		Client: detailClient{
			IP: clientIP(h), UserAgent: h.Get("User-Agent"),
			ProtocolVersion: h.Get("Mcp-Protocol-Version"), SessionID: h.Get("Mcp-Session-Id"),
		},
		Timing:  c.timing,
		Trace:   steps,
		Dropped: dropped,
		Result:  detailResult{Bytes: size},
		Outcome: detailOutcome{Status: status, Error: errText, Phase: c.phase},
		Server:  detailServer{Host: hostname},
	}
	if status == "ok" {
		d.Outcome.Phase = ""
	}
	if d.Trace == nil {
		d.Trace = []traceStep{}
	}
	if len(args) <= maxDetailArgs && json.Valid(args) {
		d.Request.Arguments = args
	} else if len(args) > 0 {
		d.Request.Arguments, _ = json.Marshal(truncUTF8(string(args), maxDetailArgs))
		d.Request.Truncated = true
	}
	if c.buildID.Valid {
		d.Request.BuildID = c.buildID.String()
	}
	for _, r := range c.key.Repos {
		d.Key.LimitedTo = append(d.Key.LimitedTo, r.String())
	}
	for _, name := range detailHeaders {
		if v := h.Get(name); v != "" {
			if d.Client.Headers == nil {
				d.Client.Headers = map[string]string{}
			}
			d.Client.Headers[name] = v
		}
	}
	if len(c.result) > 0 {
		d.Result.Lists = topLevelLists(c.result)
		d.Result.Clipped = bytes.Count(c.result, []byte("[cut: "))
		var pretty bytes.Buffer
		if json.Indent(&pretty, c.result, "", "  ") == nil {
			d.Result.Preview = pretty.String()
		} else {
			d.Result.Preview = string(c.result)
		}
		if len(d.Result.Preview) > maxDetailPreview {
			d.Result.Preview = truncUTF8(d.Result.Preview, maxDetailPreview)
			d.Result.PreviewTruncated = true
		}
	}
	return d
}

// topLevelLists counts the entries of each list at the top of a JSON object.
func topLevelLists(b []byte) map[string]int {
	var top map[string]json.RawMessage
	if json.Unmarshal(b, &top) != nil {
		return nil
	}
	keys := make([]string, 0, len(top))
	for k := range top {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var out map[string]int
	for _, k := range keys {
		var list []json.RawMessage
		if json.Unmarshal(top[k], &list) == nil && list != nil {
			if out == nil {
				out = map[string]int{}
			}
			out[k] = len(list)
		}
	}
	return out
}

func truncUTF8(s string, n int) string {
	if len(s) <= n {
		return s
	}
	for n > 0 && !utf8.RuneStart(s[n]) {
		n--
	}
	return s[:n]
}
