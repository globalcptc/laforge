// The admin-facing half of "what about the MicroCloud builder":
// internal/builderconfig.Resolve is the real
// lookup internal/runner uses at deploy time; these endpoints are how a
// real builder_config row gets there in the first place. Instance-admin
// gated like installations.go's own approve flow -- a builder config is
// shared, global infrastructure (a MicroCloud cluster isn't owned by one
// repository), not a per-repository concern, so requireInstanceAdmin,
// not requireLevel, is the right floor.
package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	awsbuilder "github.com/globalcptc/laforge/internal/builder/aws"
	"github.com/globalcptc/laforge/internal/builder/incus"
	"github.com/globalcptc/laforge/internal/builder/microcloud"
	"github.com/globalcptc/laforge/internal/builderconfig"
	"github.com/globalcptc/laforge/internal/db"
)

// builderConfigRequest is both the create and update request body. kind
// "microcloud" (one real cluster, any member answers for the whole
// thing) uses the single incus_* fields below; kind "incus" (a POOL of
// independent, non-clustered hosts -- see migration 00014's own doc
// comment) uses IncusHosts instead, one entry per host. Either way,
// incus_client_cert_path/incus_client_key_path (per host, for "incus")
// are filesystem paths (the runner's own, real credential file -- see
// migration 00012's own doc comment for why this isn't key material in
// the request/row at all), and incus_server_cert_pem is the pinned server
// certificate itself (a public value, not a secret -- see
// FetchServerCertificateInsecure's own doc comment for how an admin
// obtains it once, out of band).
type builderConfigRequest struct {
	IncusOneTeamPerHost    bool                           `json:"incus_one_team_per_host"`
	MicrocloudPublicAccess *microcloud.PublicAccessConfig `json:"microcloud_public_access,omitempty"`
	// MicrocloudClusterGroup is the LXD cluster group every instance is created
	// into (MicroCloud only); LXD schedules across its members. Empty defaults to
	// "default" (the whole cluster). LaForge never picks a specific member.
	MicrocloudClusterGroup string `json:"microcloud_cluster_group,omitempty"`
	// AwsConfig is kind "aws"'s account infrastructure (ECS cluster,
	// task-execution role, log group, region, agent-init image, assign-public-IP).
	// Empty fields fall back to the runner's AWS_* env vars. Ignored by every
	// other kind.
	AwsConfig             *awsbuilder.ConfigInput `json:"aws_config,omitempty"`
	Kind                  string                  `json:"kind"`
	IncusApiUrl           string                  `json:"incus_api_url,omitempty"`
	IncusClientCertPath   string                  `json:"incus_client_cert_path,omitempty"`
	IncusClientKeyPath    string                  `json:"incus_client_key_path,omitempty"`
	IncusServerCertPem    string                  `json:"incus_server_cert_pem,omitempty"`
	IncusOvnUplinkNetwork string                  `json:"incus_ovn_uplink_network,omitempty"`
	IncusStoragePool      string                  `json:"incus_storage_pool,omitempty"`
	// IncusProject is the LXD project a kind "microcloud" builder creates
	// everything in; empty is `default`. Ignored by every other kind.
	IncusProject                 string                    `json:"incus_project,omitempty"`
	IncusOperationTimeoutSeconds int32                     `json:"incus_operation_timeout_seconds,omitempty"`
	IncusImages                  map[string]incus.ImageRef `json:"incus_images,omitempty"`
	IncusSizes                   map[string]incus.SizeSpec `json:"incus_sizes,omitempty"`
	// IncusHosts is only meaningful for kind "incus" -- one pool member
	// per entry. Ignored (and left empty) for every other kind.
	IncusHosts []incus.HostConfig `json:"incus_hosts,omitempty"`
	// IncusCredentialID is kind "microcloud"'s enrolled connection (see
	// builder_probe.go's handleConnectBuilder) -- when set, the older
	// incus_api_url/cert path/server cert fields are left empty.
	IncusCredentialID string `json:"incus_credential_id,omitempty"`
	// AwsCredentialID is kind "aws"'s enrolled credential (the access key stored
	// by the connect step). Empty means the runner's ambient AWS chain.
	AwsCredentialID string `json:"aws_credential_id,omitempty"`
	// ExternalAccessIP + the port window realize content `public:` ports on a
	// shared-IP builder (Incus/MicroCloud): the single external IP, and the
	// range of external ports allocated on it. Empty IP = external access off.
	// 0 ports = builder defaults. Public-IP builders ignore these.
	ExternalAccessIP string `json:"external_access_ip,omitempty"`
	ExternalPortMin  int32  `json:"external_port_min,omitempty"`
	ExternalPortMax  int32  `json:"external_port_max,omitempty"`
	// Color is a user-chosen display color for the builder's name pill ("" = none,
	// else a "#rrggbb" hex). See validate() for the format check.
	Color string `json:"color,omitempty"`
	// MaxConcurrency caps how many of THIS builder's tasks may deploy at once,
	// below the global RUNNER_CONCURRENCY pool -- a knob to turn down a
	// capacity-limited hoster (e.g. a MicroCloud cluster). 0 = unlimited.
	// Enforced as a soft cap in the task-lease query, so it applies to any kind.
	MaxConcurrency int32 `json:"max_concurrency,omitempty"`
	// IncusHostMaxConcurrency caps how many instance deploys run at once on EACH
	// host of a kind "incus" pool (where the contention is: one team's VMs all
	// created on one host). Over the cap, deploys queue in the runner and are
	// admitted critical-path first. 0 = unbounded. Incus pools only.
	IncusHostMaxConcurrency int32 `json:"incus_host_max_concurrency,omitempty"`
}

// validate turns req into the columns CreateBuilderConfig/UpdateBuilderConfig
// need, and -- the real point of doing this here rather than just
// marshaling straight through -- proves the result is actually usable by
// running it through internal/builderconfig.Resolve (discarding the
// Builder it returns): an admin gets a clear, immediate error for a
// broken config (an unreadable cert file, a missing required field) at
// the moment they register it, not only once a real deploy tries to use
// it. Resolve's own "microcloud"/"incus" paths never dial the network --
// they only parse and pin locally -- so this needs no live cluster to
// validate.
// validBuilderColor accepts an empty string (no color chosen) or a lowercase or
// uppercase "#rrggbb" hex string. The UI's color picker only ever sends the
// latter, but a config could be posted directly, so the format is checked here.
func validBuilderColor(c string) bool {
	if c == "" {
		return true
	}
	if len(c) != 7 || c[0] != '#' {
		return false
	}
	for _, r := range c[1:] {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')) {
			return false
		}
	}
	return true
}

func (req builderConfigRequest) validate(pool *pgxpool.Pool) (db.CreateBuilderConfigParams, error) {
	if req.Kind != "fake" && req.Kind != "incus" && req.Kind != "microcloud" && req.Kind != "aws" && req.Kind != "openstack" {
		return db.CreateBuilderConfigParams{}, errors.New(`kind must be "fake", "incus", "microcloud", "aws", or "openstack"`)
	}
	if req.IncusOneTeamPerHost && req.Kind != "incus" {
		return db.CreateBuilderConfigParams{}, errors.New("incus_one_team_per_host is only supported by the Incus builder")
	}
	if !validBuilderColor(req.Color) {
		return db.CreateBuilderConfigParams{}, errors.New(`color must be empty or a "#rrggbb" hex string`)
	}
	var publicAccess []byte
	if req.MicrocloudPublicAccess != nil {
		if req.Kind != "microcloud" {
			return db.CreateBuilderConfigParams{}, errors.New("microcloud_public_access is only supported by the MicroCloud builder")
		}
		if err := req.MicrocloudPublicAccess.Validate(); err != nil {
			return db.CreateBuilderConfigParams{}, err
		}
		publicAccess, _ = json.Marshal(req.MicrocloudPublicAccess)
	}
	clusterGroup := strings.TrimSpace(req.MicrocloudClusterGroup)
	if clusterGroup != "" && req.Kind != "microcloud" {
		return db.CreateBuilderConfigParams{}, errors.New("microcloud_cluster_group is only supported by the MicroCloud builder")
	}
	// The column is NOT NULL -- an unset group means the whole cluster, "default".
	if clusterGroup == "" {
		clusterGroup = "default"
	}
	var awsConfig []byte
	if req.AwsConfig != nil {
		if req.Kind != "aws" {
			return db.CreateBuilderConfigParams{}, errors.New("aws_config is only supported by the AWS builder")
		}
		awsConfig, _ = json.Marshal(req.AwsConfig)
	}
	images, err := json.Marshal(req.IncusImages)
	if err != nil {
		return db.CreateBuilderConfigParams{}, err
	}
	if req.IncusImages == nil {
		images = []byte("{}")
	}
	sizes, err := json.Marshal(req.IncusSizes)
	if err != nil {
		return db.CreateBuilderConfigParams{}, err
	}
	if req.IncusSizes == nil {
		sizes = []byte("{}")
	}
	hosts, err := json.Marshal(req.IncusHosts)
	if err != nil {
		return db.CreateBuilderConfigParams{}, err
	}
	if req.IncusHosts == nil {
		hosts = []byte("[]")
	}
	// A negative cap is meaningless; treat it as 0 = unlimited. No upper clamp --
	// a value above the global worker pool simply never binds.
	maxConcurrency := req.MaxConcurrency
	if maxConcurrency < 0 {
		maxConcurrency = 0
	}
	incusHostMaxConcurrency := req.IncusHostMaxConcurrency
	if incusHostMaxConcurrency < 0 {
		incusHostMaxConcurrency = 0
	}
	if incusHostMaxConcurrency > 0 && req.Kind != "incus" {
		return db.CreateBuilderConfigParams{}, errors.New("incus_host_max_concurrency is only supported by the Incus builder")
	}
	params := db.CreateBuilderConfigParams{
		Kind:                    req.Kind,
		IncusOneTeamPerHost:     req.IncusOneTeamPerHost,
		MicrocloudPublicAccess:  publicAccess,
		AwsConfig:               awsConfig,
		IncusApiUrl:             db.StrPtr(req.IncusApiUrl),
		IncusClientCertPath:     db.StrPtr(req.IncusClientCertPath),
		IncusClientKeyPath:      db.StrPtr(req.IncusClientKeyPath),
		IncusServerCertPem:      db.StrPtr(req.IncusServerCertPem),
		IncusOvnUplinkNetwork:   db.StrPtr(req.IncusOvnUplinkNetwork),
		IncusStoragePool:        db.StrPtr(req.IncusStoragePool),
		IncusImages:             images,
		IncusSizes:              sizes,
		IncusHosts:              hosts,
		ExternalAccessIp:        db.StrPtr(req.ExternalAccessIP),
		Color:                   req.Color,
		MicrocloudClusterGroup:  clusterGroup,
		MaxConcurrency:          maxConcurrency,
		IncusHostMaxConcurrency: incusHostMaxConcurrency,
	}
	if req.Kind == "microcloud" && req.IncusProject != "default" {
		params.IncusProject = db.StrPtr(req.IncusProject)
	}
	if req.IncusOperationTimeoutSeconds > 0 {
		params.IncusOperationTimeoutSeconds = &req.IncusOperationTimeoutSeconds
	}
	if req.ExternalPortMin > 0 {
		params.ExternalPortMin = &req.ExternalPortMin
	}
	if req.ExternalPortMax > 0 {
		params.ExternalPortMax = &req.ExternalPortMax
	}
	if req.IncusCredentialID != "" {
		if err := params.IncusCredentialID.Scan(req.IncusCredentialID); err != nil {
			return db.CreateBuilderConfigParams{}, fmt.Errorf("invalid incus_credential_id: %w", err)
		}
	}
	if req.AwsCredentialID != "" {
		if req.Kind != "aws" {
			return db.CreateBuilderConfigParams{}, errors.New("aws_credential_id is only supported by the AWS builder")
		}
		if err := params.AwsCredentialID.Scan(req.AwsCredentialID); err != nil {
			return db.CreateBuilderConfigParams{}, fmt.Errorf("invalid aws_credential_id: %w", err)
		}
	}

	// Run the exact same resolution real deploys use, purely to validate
	// -- see this method's own doc comment. pool is only read here (to load
	// an enrolled credential), never used to dial the hoster.
	//
	// The DRAFT cloud kinds ("aws"/"openstack") are skipped: their Resolve
	// authenticates against the live cloud (Keystone / the AWS credential
	// chain), so probing at config-creation time would need real cloud
	// credentials this validation deliberately doesn't require -- and there's
	// no test infrastructure for them yet. They're accepted and stored;
	// their first real deploy is where a misconfiguration surfaces.
	if params.Kind == "aws" || params.Kind == "openstack" {
		return params, nil
	}
	if _, err := builderconfig.Resolve(pool, db.BuilderConfig{
		Name: "(validation)", Kind: params.Kind,
		IncusApiUrl: params.IncusApiUrl, IncusClientCertPath: params.IncusClientCertPath,
		IncusClientKeyPath: params.IncusClientKeyPath, IncusServerCertPem: params.IncusServerCertPem,
		IncusOvnUplinkNetwork: params.IncusOvnUplinkNetwork, IncusStoragePool: params.IncusStoragePool,
		IncusOperationTimeoutSeconds: params.IncusOperationTimeoutSeconds,
		IncusImages:                  params.IncusImages, IncusSizes: params.IncusSizes,
		IncusHosts: params.IncusHosts, IncusCredentialID: params.IncusCredentialID,
		IncusOneTeamPerHost:    params.IncusOneTeamPerHost,
		IncusProject:           params.IncusProject,
		MicrocloudPublicAccess: params.MicrocloudPublicAccess,
	}); err != nil {
		return db.CreateBuilderConfigParams{}, err
	}
	return params, nil
}

// verifyImages checks, through each host's own API, that every image the
// config offers is really on every host (builderconfig.VerifyImages) --
// done at save time so a gap is caught by the admin, not by a failed
// deploy mid-event.
func (s *Server) verifyImages(r *http.Request, params db.CreateBuilderConfigParams) error {
	// Each host client bounds its reads using the configured operation timeout.
	// A fixed deadline around the whole catalog would truncate slow reads and
	// make otherwise healthy hosts fail as the number of checks grows.
	return builderconfig.VerifyImages(r.Context(), s.Pool, db.BuilderConfig{
		Kind: params.Kind, IncusApiUrl: params.IncusApiUrl, IncusClientCertPath: params.IncusClientCertPath,
		IncusClientKeyPath: params.IncusClientKeyPath, IncusServerCertPem: params.IncusServerCertPem,
		IncusImages: params.IncusImages, IncusHosts: params.IncusHosts, IncusCredentialID: params.IncusCredentialID,
		IncusOperationTimeoutSeconds: params.IncusOperationTimeoutSeconds,
		IncusOneTeamPerHost:          params.IncusOneTeamPerHost,
		IncusProject:                 params.IncusProject,
		MicrocloudPublicAccess:       params.MicrocloudPublicAccess,
	})
}

// authAndAdmin resolves the caller from a session cookie OR a bearer token and
// reports whether they are an instance admin. err is non-nil only when the
// caller isn't authenticated at all -- so a GET gated by this is readable by any
// signed-in user, with admin-only fields redacted for the rest.
func (s *Server) authAndAdmin(ctx context.Context, r *http.Request) (bool, error) {
	sess, err := s.authSessionForRequest(ctx, r)
	if err != nil {
		return false, err
	}
	return s.isInstanceAdmin(sess), nil
}

// redactBuilderConfig strips the connection/credential material from a builder
// config so a non-admin can see WHAT is configured (name, kind, hoster URL, the
// os->image and size maps) without its cert paths, server cert, pooled-host
// connection list, or credential reference -- the same "none of its connection
// details" line builderSummary already draws for non-admins.
func redactBuilderConfig(c db.BuilderConfig) db.BuilderConfig {
	c.IncusClientCertPath = nil
	c.IncusClientKeyPath = nil
	c.IncusServerCertPem = nil
	c.IncusHosts = nil
	c.IncusCredentialID = pgtype.UUID{}
	return c
}

func (s *Server) handleListBuilderConfigs(w http.ResponseWriter, r *http.Request) {
	admin, err := s.authAndAdmin(r.Context(), r)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	rows, err := s.Queries.ListBuilderConfigs(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if rows == nil {
		rows = []db.BuilderConfig{}
	}
	if !admin {
		for i := range rows {
			rows[i] = redactBuilderConfig(rows[i])
		}
	}
	writeJSON(w, http.StatusOK, rows)
}

// builderSummary is what anyone configuring a build needs to pick a
// builder: its name and type, none of its connection details.
type builderSummary struct {
	Name  string `json:"name"`
	Kind  string `json:"kind"`
	Color string `json:"color,omitempty"`
}

// handleListBuilders lists every builder by name and type for any signed-in
// person -- the New Build picker needs it, and someone with build access to
// a repository isn't necessarily an instance admin.
func (s *Server) handleListBuilders(w http.ResponseWriter, r *http.Request) {
	if _, err := s.sessionFromRequest(r); err != nil {
		writeAuthError(w, err)
		return
	}
	rows, err := s.Queries.ListBuilderConfigs(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	out := make([]builderSummary, 0, len(rows))
	for _, row := range rows {
		out = append(out, builderSummary{Name: row.Name, Kind: row.Kind, Color: row.Color})
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleGetBuilderConfig(w http.ResponseWriter, r *http.Request) {
	admin, err := s.authAndAdmin(r.Context(), r)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	row, err := s.Queries.GetBuilderConfigByName(r.Context(), r.PathValue("name"))
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	if !admin {
		row = redactBuilderConfig(row)
	}
	writeJSON(w, http.StatusOK, row)
}

func (s *Server) handleCreateBuilderConfig(w http.ResponseWriter, r *http.Request) {
	if _, err := s.requireInstanceAdmin(r.Context(), r); err != nil {
		writeAuthError(w, err)
		return
	}
	name := r.PathValue("name")
	if name == "" {
		writeError(w, http.StatusBadRequest, errors.New("name is required"))
		return
	}
	var req builderConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	params, err := req.validate(s.Pool)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := s.verifyImages(r, params); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	params.Name = name
	row, err := s.Queries.CreateBuilderConfig(r.Context(), params)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	// Kick off the docker base image build for a newly-added Incus/MicroCloud
	// builder so its container runtime is ready without a manual step.
	s.queueDockerBaseBuild(r, row)
	writeJSON(w, http.StatusCreated, row)
}

func (s *Server) handleUpdateBuilderConfig(w http.ResponseWriter, r *http.Request) {
	if _, err := s.requireInstanceAdmin(r.Context(), r); err != nil {
		writeAuthError(w, err)
		return
	}
	name := r.PathValue("name")
	var req builderConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	params, err := req.validate(s.Pool)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	// Re-verify images against the hoster only when something that affects
	// them changed -- the images, the hosts, the connection, the project. A
	// save that changes the concurrency limit or the sizes shouldn't wait on
	// a snapshot read per image on a loaded cluster (minutes, and a timed-out
	// save). `?verify=false` skips it on request, logged.
	skip := r.URL.Query().Get("verify") == "false"
	if existing, err := s.Queries.GetBuilderConfigByName(r.Context(), name); err == nil && !imagesOrConnectionChanged(existing, params) {
		log.Printf("api: builder %q: saved without re-verifying images (images, hosts and connection unchanged)", name)
		skip = true
	} else if skip {
		log.Printf("api: builder %q: saved without verifying images, as requested (verify=false)", name)
	}
	if !skip {
		started := time.Now()
		if err := s.verifyImages(r, params); err != nil {
			log.Printf("api: builder %q: image verification failed after %s: %v", name, time.Since(started).Round(time.Millisecond), err)
			writeError(w, http.StatusBadRequest, fmt.Errorf("%w -- to save anyway, repeat the save without verifying images", err))
			return
		}
		log.Printf("api: builder %q: images verified in %s", name, time.Since(started).Round(time.Millisecond))
	}
	row, err := s.Queries.UpdateBuilderConfig(r.Context(), db.UpdateBuilderConfigParams{
		Name: name, Kind: params.Kind,
		IncusApiUrl: params.IncusApiUrl, IncusClientCertPath: params.IncusClientCertPath,
		IncusClientKeyPath: params.IncusClientKeyPath, IncusServerCertPem: params.IncusServerCertPem,
		IncusOvnUplinkNetwork: params.IncusOvnUplinkNetwork, IncusStoragePool: params.IncusStoragePool,
		IncusOperationTimeoutSeconds: params.IncusOperationTimeoutSeconds,
		IncusImages:                  params.IncusImages, IncusSizes: params.IncusSizes,
		IncusHosts: params.IncusHosts, IncusCredentialID: params.IncusCredentialID,
		IncusOneTeamPerHost: params.IncusOneTeamPerHost,
		ExternalAccessIp:    params.ExternalAccessIp,
		ExternalPortMin:     params.ExternalPortMin, ExternalPortMax: params.ExternalPortMax,
		IncusProject:            params.IncusProject,
		MicrocloudPublicAccess:  params.MicrocloudPublicAccess,
		AwsConfig:               params.AwsConfig,
		AwsCredentialID:         params.AwsCredentialID,
		Color:                   params.Color,
		MicrocloudClusterGroup:  params.MicrocloudClusterGroup,
		MaxConcurrency:          params.MaxConcurrency,
		IncusHostMaxConcurrency: params.IncusHostMaxConcurrency,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, errors.New("no such builder config"))
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, row)
}

func (s *Server) handleDeleteBuilderConfig(w http.ResponseWriter, r *http.Request) {
	if _, err := s.requireInstanceAdmin(r.Context(), r); err != nil {
		writeAuthError(w, err)
		return
	}
	if err := s.Queries.DeleteBuilderConfig(r.Context(), r.PathValue("name")); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// imagesOrConnectionChanged reports whether a builder update touches anything
// image verification depends on: which images, which hosts, how the hoster is
// reached, and which project.
func imagesOrConnectionChanged(existing db.BuilderConfig, params db.CreateBuilderConfigParams) bool {
	return existing.Kind != params.Kind ||
		string(existing.IncusImages) != string(params.IncusImages) ||
		string(existing.IncusHosts) != string(params.IncusHosts) ||
		existing.IncusApiUrl != params.IncusApiUrl ||
		existing.IncusServerCertPem != params.IncusServerCertPem ||
		existing.IncusClientCertPath != params.IncusClientCertPath ||
		existing.IncusClientKeyPath != params.IncusClientKeyPath ||
		existing.IncusCredentialID != params.IncusCredentialID ||
		existing.IncusProject != params.IncusProject
}
