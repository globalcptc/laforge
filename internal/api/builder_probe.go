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
	"time"

	"github.com/globalcptc/laforge/internal/builder"
	"github.com/globalcptc/laforge/internal/builder/incus"
	"github.com/globalcptc/laforge/internal/builder/microcloud"
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
		// reach or enroll the hoster is statusUpstreamFailed (424, not 502 -- see json.go).
		if errors.Is(err, builder.ErrInvalidOnboardRequest) {
			writeError(w, http.StatusBadRequest, err)
		} else {
			writeError(w, statusUpstreamFailed, err)
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
		// ?project= scopes the images to the project the builder will use.
		Project: r.URL.Query().Get("project"),
	})
	if err != nil {
		writeError(w, statusUpstreamFailed, err)
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

// handleListBuilderConfigImages re-discovers what a builder's hoster actually
// holds -- the images (and pools/networks) you could map an os to -- keyed by
// builder NAME, resolving its stored credential. Unlike handleGetBuilderConnection
// (which takes a raw credential id), this is what `laforge images <b> --available`
// and the config editor both want: "show me what this builder can see."
func (s *Server) handleListBuilderConfigImages(w http.ResponseWriter, r *http.Request) {
	if _, err := s.requireInstanceAdmin(r.Context(), r); err != nil {
		writeAuthError(w, err)
		return
	}
	cfg, err := s.Queries.GetBuilderConfigByName(r.Context(), r.PathValue("name"))
	if err != nil {
		writeError(w, http.StatusNotFound, errors.New("no such builder"))
		return
	}
	if !cfg.IncusCredentialID.Valid {
		writeError(w, http.StatusBadRequest, errors.New("this builder has no stored connection to discover from (env-credential builders like AWS/OpenStack don't keep one)"))
		return
	}
	cred, err := s.Queries.GetBuilderCredential(r.Context(), cfg.IncusCredentialID)
	if err != nil {
		writeError(w, http.StatusNotFound, errors.New("the builder's stored connection credential is missing"))
		return
	}
	onb, err := builder.OnboarderFor(onboarderKindOrDefault(cfg.Kind))
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	disc, err := onb.Rediscover(r.Context(), builder.Connection{
		APIURL: cred.ApiUrl, ServerName: cred.ServerName, ServerFingerprint: cred.ServerFingerprint,
		ServerCertPEM: []byte(cred.ServerCertPem), ClientCertPEM: []byte(cred.ClientCertPem), ClientKeyPEM: []byte(cred.ClientKeyPem),
		Project: db.StrOrEmpty(cfg.IncusProject),
	})
	if err != nil {
		writeError(w, statusUpstreamFailed, err)
		return
	}
	writeJSON(w, http.StatusOK, disc)
}

// tokenCheckView is what checking a trust token found, without redeeming it.
type tokenCheckView struct {
	ClientName  string `json:"client_name"`
	Fingerprint string `json:"fingerprint"`
	ExpiresAt   string `json:"expires_at,omitempty"`
	// Type is set for an LXD identity token (redeemed through the identities
	// API) and empty for a plain trust token.
	Type string `json:"type,omitempty"`
	// Addresses are probed in the order Connect tries them; WillUse is the one
	// Connect would enroll through (the first whose certificate matches), or
	// empty if none would work.
	Addresses []addressCheckView `json:"addresses"`
	WillUse   string             `json:"will_use"`
}

type addressCheckView struct {
	Address            string `json:"address"`
	Reachable          bool   `json:"reachable"`
	FingerprintMatches bool   `json:"fingerprint_matches"`
	Error              string `json:"error,omitempty"`
	Millis             int64  `json:"millis"`
}

// handleCheckBuilderToken tries every address a trust token (or the override
// address) names from the API server -- exactly the probe Connect starts with
// -- and reports each one, without redeeming the token. It answers "why won't
// this connect?" before a single-use token is spent on finding out.
func (s *Server) handleCheckBuilderToken(w http.ResponseWriter, r *http.Request) {
	if _, err := s.requireInstanceAdmin(r.Context(), r); err != nil {
		writeAuthError(w, err)
		return
	}
	var req connectBuilderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	var view tokenCheckView
	var addresses []string
	switch onboarderKindOrDefault(req.Kind) {
	case "microcloud":
		t, err := microcloud.ParseTrustToken(req.Token)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		view = tokenCheckView{ClientName: t.ClientName, Fingerprint: t.Fingerprint, Type: t.Type}
		if !t.ExpiresAt.IsZero() {
			view.ExpiresAt = t.ExpiresAt.Format(time.RFC3339)
		}
		addresses = microcloud.TokenAddresses(t, req.Address)
		for _, c := range microcloud.CheckAddresses(r.Context(), addresses, t.Fingerprint) {
			view.Addresses = append(view.Addresses, addressCheckView{c.Address, c.Reachable, c.FingerprintMatches, c.Error, c.Millis})
		}
	case "incus":
		t, err := incus.ParseTrustToken(req.Token)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		view = tokenCheckView{ClientName: t.ClientName, Fingerprint: t.Fingerprint, Type: t.Type}
		if !t.ExpiresAt.IsZero() {
			view.ExpiresAt = t.ExpiresAt.Format(time.RFC3339)
		}
		addresses = incus.TokenAddresses(t, req.Address)
		for _, c := range incus.CheckAddresses(r.Context(), addresses, t.Fingerprint) {
			view.Addresses = append(view.Addresses, addressCheckView{c.Address, c.Reachable, c.FingerprintMatches, c.Error, c.Millis})
		}
	default:
		writeError(w, http.StatusBadRequest, errors.New("only Incus and MicroCloud builders connect with a trust token"))
		return
	}
	for _, a := range view.Addresses {
		if a.FingerprintMatches {
			view.WillUse = a.Address
			break
		}
	}
	if view.Addresses == nil {
		view.Addresses = []addressCheckView{}
	}
	writeJSON(w, http.StatusOK, view)
}
