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

	client, err := builderconfig.ResolveMicrocloudClient(pool, cfg)
	if err != nil {
		fail(err.Error())
		return
	}
	client.OperationTimeout = imageBuildTimeout

	src := microcloud.BaseImageSource{
		Server:   cfg.ContainerBaseServer,
		Protocol: "simplestreams",
		Alias:    cfg.ContainerBaseAlias,
	}
	fp, err := microcloud.BuildDockerBase(ctx, client, src, appendLog)
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
