package incus

import (
	"context"
	"encoding/json"
	"net/url"
	"sync"
)

// serverNameCache holds the Incus server's own name, read once: it identifies
// which pool host a team's instances and networks are on, and it doesn't
// change while the builder is in use.
type serverNameCache struct {
	mu   sync.Mutex
	name string
}

// serverName is the name this Incus server reports for itself (GET /1.0,
// environment.server_name). If that can't be read, the API endpoint's host
// stands in, so placement is still told apart; the next call tries again.
func (b *Builder) serverName(ctx context.Context) string {
	b.server.mu.Lock()
	defer b.server.mu.Unlock()
	if b.server.name != "" {
		return b.server.name
	}
	raw, err := b.Client.get(ctx, "/1.0")
	if err == nil {
		var info struct {
			Environment struct {
				ServerName string `json:"server_name"`
			} `json:"environment"`
		}
		if json.Unmarshal(raw, &info) == nil && info.Environment.ServerName != "" {
			b.server.name = info.Environment.ServerName
			return b.server.name
		}
	}
	if u, perr := url.Parse(b.Client.BaseURL); perr == nil && u.Hostname() != "" {
		return u.Hostname()
	}
	return ""
}

// instanceLocation is where an instance runs: its cluster member when the
// server is clustered, otherwise the server itself. A standalone server
// reports location "none".
func (b *Builder) instanceLocation(ctx context.Context, location string) string {
	if location != "" && location != "none" {
		return location
	}
	return b.serverName(ctx)
}
