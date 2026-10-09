package orchestrator

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/globalcptc/laforge/internal/builder/incus"
	"github.com/globalcptc/laforge/internal/builder/microcloud"
	"github.com/globalcptc/laforge/internal/builderconfig"
	"github.com/globalcptc/laforge/internal/db"
)

// imageBuildTimeout bounds one docker-base build: the apt install + Docker Hub
// warm-up + publish runs a few minutes on a real cluster, so the Incus
// client's per-operation deadline is raised well past its 300s default.
const imageBuildTimeout = 20 * time.Minute

// RunPendingImageBuilds picks up every queued per-builder image build and runs
// each in its own goroutine. Leasing (StartImageBuild flips pending->running
// atomically) means a build already running is never re-picked, so this is
// safe to call every tick and returns immediately -- a build takes minutes and
// must not block the orchestrator's poll loop.
func RunPendingImageBuilds(ctx context.Context, pool *pgxpool.Pool) error {
	q := db.New(pool)
	pending, err := q.ListPendingImageBuilds(ctx)
	if err != nil {
		return fmt.Errorf("listing pending image builds: %w", err)
	}
	for _, b := range pending {
		go runImageBuild(ctx, pool, b.ID)
	}
	return nil
}

func runImageBuild(ctx context.Context, pool *pgxpool.Pool, buildID pgtype.UUID) {
	q := db.New(pool)

	// Lease it: only the goroutine that flips pending->running proceeds.
	leased, err := q.StartImageBuild(ctx, buildID)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			log.Printf("image build %s: leasing: %v", buildID.String(), err)
		}
		return // already leased/finished by someone else, or gone
	}
	appendLog := func(line string) {
		if err := q.AppendImageBuildLog(ctx, db.AppendImageBuildLogParams{BuildID: buildID, Line: line}); err != nil {
			log.Printf("image build %s: appending log: %v", buildID.String(), err)
		}
	}
	fail := func(msg string) {
		appendLog("BUILD FAILED: " + msg)
		if err := q.FinishImageBuildFailed(ctx, db.FinishImageBuildFailedParams{ID: buildID, Error: msg}); err != nil {
			log.Printf("image build %s: recording failure: %v", buildID.String(), err)
		}
	}

	cfg, err := q.GetBuilderConfigByID(ctx, leased.BuilderConfigID)
	if err != nil {
		fail(fmt.Sprintf("loading builder config: %v", err))
		return
	}
	appendLog(fmt.Sprintf("Building docker base image for builder %q (%s)…", cfg.Name, cfg.Kind))

	var fp string
	switch cfg.Kind {
	case "incus":
		fp, err = buildIncusDockerBase(ctx, pool, cfg, appendLog)
	default:
		fp, err = buildMicrocloudDockerBase(ctx, pool, cfg, appendLog)
	}
	if err != nil {
		fail(err.Error())
		return
	}

	appendLog(fmt.Sprintf("BUILD SUCCEEDED: published %s (%s)", microcloud.DockerBaseAlias, fp))
	if err := q.FinishImageBuildSuccess(ctx, db.FinishImageBuildSuccessParams{ID: buildID, ImageFingerprint: fp}); err != nil {
		log.Printf("image build %s: recording success: %v", buildID.String(), err)
	}
	// Record the result on the builder config so the container runtime knows
	// which image to boot from.
	if err := q.SetBuilderConfigDockerBaseFingerprint(ctx, db.SetBuilderConfigDockerBaseFingerprintParams{
		ID: cfg.ID, DockerBaseFingerprint: fp,
	}); err != nil {
		log.Printf("image build %s: recording fingerprint on builder config: %v", buildID.String(), err)
	}
}

func buildMicrocloudDockerBase(ctx context.Context, pool *pgxpool.Pool, cfg db.BuilderConfig, appendLog func(string)) (string, error) {
	client, err := builderconfig.ResolveMicrocloudClient(pool, cfg)
	if err != nil {
		return "", err
	}
	client.OperationTimeout = imageBuildTimeout
	return microcloud.BuildDockerBase(ctx, client, microcloud.BaseImageSource{
		Server:   cfg.ContainerBaseServer,
		Protocol: "simplestreams",
		Alias:    cfg.ContainerBaseAlias,
	}, db.StrOrEmpty(cfg.IncusStoragePool), appendLog)
}

// lxdDefaultBaseServer is builder_config.container_base_server's column
// default -- Canonical's image server, which serves LXD but not Incus.
const lxdDefaultBaseServer = "https://cloud-images.ubuntu.com/releases"

// buildIncusDockerBase builds the image on every host of an Incus builder --
// they are independent, with no shared image store -- and returns the first
// host's fingerprint as the record that the build happened. Any host failing
// fails the build: a team placed on a host without the image couldn't deploy.
func buildIncusDockerBase(ctx context.Context, pool *pgxpool.Pool, cfg db.BuilderConfig, appendLog func(string)) (string, error) {
	hosts, err := builderconfig.ResolveIncusHosts(pool, cfg)
	if err != nil {
		return "", err
	}
	// A config still on the column default names a source Incus can't pull
	// from; the zero value makes BuildDockerBase use its own default.
	var src incus.BaseImageSource
	if cfg.ContainerBaseServer != lxdDefaultBaseServer {
		src = incus.BaseImageSource{Server: cfg.ContainerBaseServer, Protocol: "simplestreams", Alias: cfg.ContainerBaseAlias}
	}
	var first string
	for i, h := range hosts {
		appendLog(fmt.Sprintf("Host %d of %d: %s", i+1, len(hosts), h.Label))
		h.Client.OperationTimeout = imageBuildTimeout
		fp, err := incus.BuildDockerBase(ctx, h.Client, src, h.StoragePool, appendLog)
		if err != nil {
			return "", fmt.Errorf("%s: %w", h.Label, err)
		}
		if first == "" {
			first = fp
		}
	}
	return first, nil
}
