// Per-builder "docker base image" build job endpoints (migration 00024).
// The job runs in the orchestrator (internal/orchestrator/image_build.go);
// these endpoints queue it (on builder create and on demand via Rebuild),
// and expose its status + streamed log for the live console on the builder
// config page. Instance-admin only, like the rest of builder-config
// management.
package api

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/globalcptc/laforge/internal/db"
)

// usesDockerBase reports whether a builder kind has a docker-ready base image
// this job builds. MicroCloud (LXD) has no OCI runtime, so every `container:`
// boots from it. Incus runs a single-image `container:` as native OCI and needs
// it only for a container that runs a Docker Compose project. fake/aws/
// openstack don't use one.
func usesDockerBase(kind string) bool { return kind == "microcloud" || kind == "incus" }

// queueDockerBaseBuild enqueues a docker-base build when a builder is created,
// for the kind that can't run any container without one (MicroCloud). An Incus
// builder's is built on demand (Rebuild), by an operator who wants compose
// containers. Best-effort: a failure to queue must not fail creating the
// builder itself (the operator can always hit Rebuild).
func (s *Server) queueDockerBaseBuild(r *http.Request, cfg db.BuilderConfig) {
	if cfg.Kind != "microcloud" {
		return
	}
	_, _ = s.Queries.CreateImageBuild(r.Context(), db.CreateImageBuildParams{
		BuilderConfigID: cfg.ID, Kind: "docker_base",
	})
}

// handleRebuildBuilderImage queues a fresh docker-base build for a builder.
func (s *Server) handleRebuildBuilderImage(w http.ResponseWriter, r *http.Request) {
	if _, err := s.requireInstanceAdmin(r.Context(), r); err != nil {
		writeAuthError(w, err)
		return
	}
	cfg, err := s.Queries.GetBuilderConfigByName(r.Context(), r.PathValue("name"))
	if err != nil {
		writeError(w, http.StatusNotFound, errors.New("no such builder config"))
		return
	}
	if !usesDockerBase(cfg.Kind) {
		writeError(w, http.StatusBadRequest, errors.New("this builder kind does not use a docker base image"))
		return
	}
	build, err := s.Queries.CreateImageBuild(r.Context(), db.CreateImageBuildParams{
		BuilderConfigID: cfg.ID, Kind: "docker_base",
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusCreated, build)
}

// handleListBuilderImageBuilds lists a builder's recent image builds.
func (s *Server) handleListBuilderImageBuilds(w http.ResponseWriter, r *http.Request) {
	if _, err := s.requireInstanceAdmin(r.Context(), r); err != nil {
		writeAuthError(w, err)
		return
	}
	cfg, err := s.Queries.GetBuilderConfigByName(r.Context(), r.PathValue("name"))
	if err != nil {
		writeError(w, http.StatusNotFound, errors.New("no such builder config"))
		return
	}
	builds, err := s.Queries.ListImageBuildsByBuilder(r.Context(), cfg.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if builds == nil {
		builds = []db.BuilderImageBuild{}
	}
	writeJSON(w, http.StatusOK, builds)
}

type imageBuildLogLine struct {
	Seq  int32  `json:"seq"`
	Line string `json:"line"`
}

// imageBuildLogResponse is the live console's poll payload: the build's
// current status plus every log line after `since`. The UI polls with the
// last seq it has while status is pending/running, and stops once done.
type imageBuildLogResponse struct {
	ID               string              `json:"id"`
	Status           string              `json:"status"`
	ImageFingerprint string              `json:"image_fingerprint"`
	Error            string              `json:"error"`
	Done             bool                `json:"done"`
	Lines            []imageBuildLogLine `json:"lines"`
}

// handleImageBuildLog returns the build's status and its log lines after
// ?since=<seq> (default 0). The live console tails it.
func (s *Server) handleImageBuildLog(w http.ResponseWriter, r *http.Request) {
	if _, err := s.requireInstanceAdmin(r.Context(), r); err != nil {
		writeAuthError(w, err)
		return
	}
	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, errors.New("invalid build id"))
		return
	}
	build, err := s.Queries.GetImageBuild(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, errors.New("no such image build"))
		return
	}
	var since int32
	if v := r.URL.Query().Get("since"); v != "" {
		n, convErr := strconv.Atoi(v)
		if convErr == nil {
			since = int32(n)
		}
	}
	rows, err := s.Queries.ListImageBuildLogSince(r.Context(), db.ListImageBuildLogSinceParams{BuildID: id, Seq: since})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	lines := make([]imageBuildLogLine, 0, len(rows))
	for _, row := range rows {
		lines = append(lines, imageBuildLogLine{Seq: row.Seq, Line: row.Line})
	}
	writeJSON(w, http.StatusOK, imageBuildLogResponse{
		ID:               build.ID.String(),
		Status:           build.Status,
		ImageFingerprint: build.ImageFingerprint,
		Error:            build.Error,
		Done:             build.Status == "succeeded" || build.Status == "failed",
		Lines:            lines,
	})
}
