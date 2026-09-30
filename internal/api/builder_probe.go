// The builder config workflow's backend: connecting a new hoster and reading
// what it has, so every later choice in the wizard is a pick from what actually
// exists. Dispatches by builder kind through the builder.Onboarder registry
// (internal/builder.Onboarder) -- the MicroCloud and Incus builders each enroll
// a trust token (`incus config trust add laforge` / the LXD equivalent on the
// host); the cloud builders authenticate from environment credentials and store
// nothing. Instance-admin gated, same as the builder config CRUD endpoints.
package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/globalcptc/laforge/internal/builder"
	"github.com/globalcptc/laforge/internal/db"
)

type connectBuilderRequest struct {
	// Kind selects which builder type's onboarding to run. Historically this
	// endpoint served only the trust-token family, so an absent kind defaults to
	// "microcloud" for backward compatibility with an older UI.
	Kind string `json:"kind"`
	// Token is exactly what `incus config trust add <name>` printed (trust-token
	// kinds only).
	Token string `json:"token"`
	// Address is optional: only needed when none of the addresses the token
	// lists are reachable from LaForge (e.g. the host sits behind NAT).
	Address string `json:"address,omitempty"`
}

type connectionView struct {
	Credential db.GetBuilderCredentialSummaryRow `json:"credential"`
	Discovery  builder.Discovery                 `json:"discovery"`
}

func onboarderKindOrDefault(kind string) string {
	if kind == "" {
		return "microcloud"
	}
	return kind
}

// handleConnectBuilder runs a builder kind's onboarding: for trust-token kinds
// it enrolls LaForge as a trusted client of the host the token describes (the
// host's certificate is checked against the token's fingerprint, not trusted on
// first use), stores the resulting credential server-side, and returns what the
// host has. The private key never leaves the server; the UI only ever sees the
// credential's id. For env-credential kinds nothing is stored.
func (s *Server) handleConnectBuilder(w http.ResponseWriter, r *http.Request) {
	if _, err := s.requireInstanceAdmin(r.Context(), r); err != nil {
		writeAuthError(w, err)
		return
	}
	var req connectBuilderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	onb, err := builder.OnboarderFor(onboarderKindOrDefault(req.Kind))
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	result, err := onb.Onboard(r.Context(), builder.OnboardRequest{Token: req.Token, Address: req.Address})
	if err != nil {
		// Bad operator input (a malformed token) is a 400; a genuine failure to
		// reach or enroll the hoster is a 502.
		if errors.Is(err, builder.ErrInvalidOnboardRequest) {
			writeError(w, http.StatusBadRequest, err)
		} else {
			writeError(w, http.StatusBadGateway, err)
		}
		return
	}

	// Env-credential kinds (AWS/OpenStack) produce no stored credential -- return
	// discovery only, no credential row.
	if !result.Connection.HasCredential() {
		writeJSON(w, http.StatusCreated, connectionView{Discovery: result.Discovery})
		return
	}

	conn := result.Connection
	cred, err := s.Queries.CreateBuilderCredential(r.Context(), db.CreateBuilderCredentialParams{
		ApiUrl: conn.APIURL, ServerName: conn.ServerName, ServerFingerprint: conn.ServerFingerprint,
		ServerCertPem: string(conn.ServerCertPEM), ClientCertPem: string(conn.ClientCertPEM), ClientKeyPem: string(conn.ClientKeyPEM),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusCreated, connectionView{
		Credential: db.GetBuilderCredentialSummaryRow{
			ID: cred.ID, ApiUrl: cred.ApiUrl, ServerName: cred.ServerName,
			ServerFingerprint: cred.ServerFingerprint, CreatedAt: cred.CreatedAt,
		},
		Discovery: result.Discovery,
	})
}

// handleGetBuilderConnection re-reads what an already-enrolled host has -- used
// when editing an existing builder config, so its choices are still picks from
// the host's real, current state. The kind query param selects the onboarder
// (defaulting to the trust-token family, the only kinds that store a credential).
func (s *Server) handleGetBuilderConnection(w http.ResponseWriter, r *http.Request) {
	if _, err := s.requireInstanceAdmin(r.Context(), r); err != nil {
		writeAuthError(w, err)
		return
	}
	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	onb, err := builder.OnboarderFor(onboarderKindOrDefault(r.URL.Query().Get("kind")))
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	cred, err := s.Queries.GetBuilderCredential(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, errors.New("no such builder connection"))
		return
	}
	disc, err := onb.Rediscover(r.Context(), builder.Connection{
		APIURL: cred.ApiUrl, ServerName: cred.ServerName, ServerFingerprint: cred.ServerFingerprint,
		ServerCertPEM: []byte(cred.ServerCertPem), ClientCertPEM: []byte(cred.ClientCertPem), ClientKeyPEM: []byte(cred.ClientKeyPem),
	})
	if err != nil {
		writeError(w, http.StatusBadGateway, err)
		return
	}
	writeJSON(w, http.StatusOK, connectionView{
		Credential: db.GetBuilderCredentialSummaryRow{
			ID: cred.ID, ApiUrl: cred.ApiUrl, ServerName: cred.ServerName,
			ServerFingerprint: cred.ServerFingerprint, CreatedAt: cred.CreatedAt,
		},
		Discovery: *disc,
	})
}
