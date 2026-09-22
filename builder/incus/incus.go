package incus

import (
	"context"
	"fmt"
	"sync"

	"github.com/gen0cide/laforge/ent"
	entteam "github.com/gen0cide/laforge/ent/team"
	"github.com/gen0cide/laforge/logging"
	incusclient "github.com/lxc/incus/v6/client"
	"golang.org/x/sync/semaphore"
)

const (
	ID          = "incus"
	Name        = "Incus"
	Description = "Builder that deploys LaForge environments to standalone Incus hosts"
	Author      = "Global CPTC"
	Version     = "0.1.0"
)

// Team.Vars keys written by this builder.
const (
	varHost         = "incus_host"
	varProject      = "incus_project"
	varGateway      = "incus_gateway"
	varIngressAddr  = "ingress_address"
	varIngressPorts = "ingress_ports"
)

// ProvisionedNetwork.Vars and ProvisionedHost.Vars keys.
const (
	varNetwork  = "incus_network"
	varInstance = "incus_instance"
)

type IncusBuilder struct {
	Config             BuilderConfig
	Logger             *logging.Logger
	DeployWorkerPool   *semaphore.Weighted
	TeardownWorkerPool *semaphore.Weighted

	clients   map[string]incusclient.InstanceServer
	hosts     map[string]HostConfig
	hostOrder []string
	teamLocks sync.Map
}

func New(config BuilderConfig, logger *logging.Logger) (*IncusBuilder, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	builder := &IncusBuilder{
		Config:             config,
		Logger:             logger,
		DeployWorkerPool:   semaphore.NewWeighted(int64(config.MaxBuildWorkers)),
		TeardownWorkerPool: semaphore.NewWeighted(int64(config.MaxTeardownWorkers)),
		clients:            make(map[string]incusclient.InstanceServer, len(config.Hosts)),
		hosts:              make(map[string]HostConfig, len(config.Hosts)),
	}

	for _, host := range config.Hosts {
		client, err := connect(config, host)
		if err != nil {
			return nil, err
		}
		builder.clients[host.Name] = client
		builder.hosts[host.Name] = host
		builder.hostOrder = append(builder.hostOrder, host.Name)
	}

	return builder, nil
}

func (builder *IncusBuilder) ID() string {
	return ID
}

func (builder *IncusBuilder) Name() string {
	return Name
}

func (builder *IncusBuilder) Description() string {
	return Description
}

func (builder *IncusBuilder) Author() string {
	return Author
}

func (builder *IncusBuilder) Version() string {
	return Version
}

func (builder *IncusBuilder) acquireDeployWorker(ctx context.Context) error {
	if err := builder.DeployWorkerPool.Acquire(ctx, 1); err != nil {
		return fmt.Errorf("failed to acquire Incus deploy worker: %w", err)
	}
	return nil
}

func (builder *IncusBuilder) acquireTeardownWorker(ctx context.Context) error {
	if err := builder.TeardownWorkerPool.Acquire(ctx, 1); err != nil {
		return fmt.Errorf("failed to acquire Incus teardown worker: %w", err)
	}
	return nil
}

func (builder *IncusBuilder) teamLock(team *ent.Team) *sync.Mutex {
	lock, _ := builder.teamLocks.LoadOrStore(team.ID, &sync.Mutex{})
	return lock.(*sync.Mutex)
}

// selectHost returns the configured host a team is pinned to, or the
// deterministic round-robin choice for an unpinned team.
func (builder *IncusBuilder) selectHost(team *ent.Team) (string, error) {
	if pinned := team.Vars[varHost]; pinned != "" {
		if _, ok := builder.hosts[pinned]; !ok {
			return "", fmt.Errorf(
				"team %d is pinned to unknown Incus host %q; explicit reassignment is required",
				team.TeamNumber,
				pinned,
			)
		}
		return pinned, nil
	}

	return pickHost(builder.hostOrder, team.TeamNumber), nil
}

func pickHost(hostOrder []string, teamNumber int) string {
	index := teamNumber - 1
	if index < 0 {
		index = 0
	}
	return hostOrder[index%len(hostOrder)]
}

// teamTarget resolves the client, host config and project for a team that
// DeployTeam has already placed.
func (builder *IncusBuilder) teamTarget(team *ent.Team) (incusclient.InstanceServer, HostConfig, string, error) {
	hostName := team.Vars[varHost]
	project := team.Vars[varProject]
	if hostName == "" || project == "" {
		return nil, HostConfig{}, "", fmt.Errorf("team %d has not been placed on an Incus host", team.TeamNumber)
	}
	host, ok := builder.hosts[hostName]
	if !ok {
		return nil, HostConfig{}, "", fmt.Errorf("team %d is pinned to unknown Incus host %q", team.TeamNumber, hostName)
	}

	return builder.clients[hostName], host, project, nil
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
