package api

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/jackc/pgx/v5/pgtype"
)

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// statusUpstreamFailed is the status for "a service LaForge depends on --
// GitHub, a builder's hoster, a registry -- failed or refused". It is 424
// Failed Dependency rather than 502 Bad Gateway on purpose: behind Cloudflare
// an origin's 502 or 504 is replaced by Cloudflare's own error page, which
// drops this response's CORS headers and its JSON error, so the browser
// reports a CORS failure and the real reason never reaches the operator.
const statusUpstreamFailed = http.StatusFailedDependency

func writeError(w http.ResponseWriter, status int, err error) {
	// Server-side and upstream failures are logged: they're what an operator
	// goes looking for in the API's log when something in the UI fails.
	if status >= 500 || status == statusUpstreamFailed {
		log.Printf("api: responding %d: %v", status, err)
	}
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

// parseUUID turns a path parameter into a pgtype.UUID, rather than
// trusting whatever the client sent straight into a SQL query -- an
// unparseable id is a 400, not a confusing 500 from Postgres.
func parseUUID(s string) (pgtype.UUID, error) {
	var u pgtype.UUID
	err := u.Scan(s)
	return u, err
}
