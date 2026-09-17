package microcloud

import (
	"context"
	"fmt"
	"sync"

	lxd "github.com/canonical/lxd/client"
	"github.com/gen0cide/laforge/ent"
	entteam "github.com/gen0cide/laforge/ent/team"
	"github.com/gen0cide/laforge/logging"
	"golang.org/x/sync/semaphore"
)

const (
	ID          = "microcloud"
	Name        = "Canonical MicroCloud"
	Description = "Builder that deploys LaForge environments to Canonical MicroCloud"
	Author      = "Global CPTC"
	Version     = "0.1.0"
)

type MicroCloudBuilder struct {
	Config             BuilderConfig
	Client             lxd.InstanceServer
	Logger             *logging.Logger
	DeployWorkerPool   *semaphore.Weighted
	TeardownWorkerPool *semaphore.Weighted

	clusterMembers []string
	teamLocks      sync.Map
}

func New(config BuilderConfig, logger *logging.Logger) (*MicroCloudBuilder, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	client, clusterMembers, err := connect(config)
	if err != nil {
		return nil, err
	}

	return &MicroCloudBuilder{
		Config:             config,
		Client:             client,
		Logger:             logger,
		DeployWorkerPool:   semaphore.NewWeighted(int64(config.MaxBuildWorkers)),
		TeardownWorkerPool: semaphore.NewWeighted(int64(config.MaxTeardownWorkers)),
		clusterMembers:     clusterMembers,
	}, nil
}

func (builder *MicroCloudBuilder) ID() string {
	return ID
}

func (builder *MicroCloudBuilder) Name() string {
	return Name
}

func (builder *MicroCloudBuilder) Description() string {
	return Description
}

func (builder *MicroCloudBuilder) Author() string {
	return Author
}

func (builder *MicroCloudBuilder) Version() string {
	return Version
}

func (builder *MicroCloudBuilder) acquireDeployWorker(ctx context.Context) error {
	if err := builder.DeployWorkerPool.Acquire(ctx, 1); err != nil {
		return fmt.Errorf("failed to acquire MicroCloud deploy worker: %w", err)
	}
	return nil
}

func (builder *MicroCloudBuilder) acquireTeardownWorker(ctx context.Context) error {
	if err := builder.TeardownWorkerPool.Acquire(ctx, 1); err != nil {
		return fmt.Errorf("failed to acquire MicroCloud teardown worker: %w", err)
	}
	return nil
}

func (builder *MicroCloudBuilder) teamLock(team *ent.Team) *sync.Mutex {
	lock, _ := builder.teamLocks.LoadOrStore(team.ID, &sync.Mutex{})
	return lock.(*sync.Mutex)
}

func cloneVars(vars map[string]string) map[string]string {
	cloned := make(map[string]string, len(vars))
	for key, value := range vars {
		cloned[key] = value
	}
	return cloned
}

func refreshTeam(ctx context.Context, team *ent.Team) (*ent.Team, error) {
	fresh, err := team.QueryBuild().
		QueryTeams().
		Where(entteam.IDEQ(team.ID)).
		Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh team %d: %w", team.TeamNumber, err)
	}
	return fresh, nil
}
