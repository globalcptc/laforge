package incus

import (
	"context"
	"fmt"

	"github.com/gen0cide/laforge/ent"
	incusclient "github.com/lxc/incus/v6/client"
	"github.com/lxc/incus/v6/shared/api"
)

func (builder *IncusBuilder) DeployTeam(ctx context.Context, team *ent.Team) error {
	if err := builder.acquireDeployWorker(ctx); err != nil {
		return err
	}
	defer builder.DeployWorkerPool.Release(1)

	build, err := team.QueryBuild().Only(ctx)
	if err != nil {
		return fmt.Errorf("failed to query build for team %d: %w", team.TeamNumber, err)
	}
	environment, err := build.QueryEnvironment().Only(ctx)
	if err != nil {
		return fmt.Errorf("failed to query environment for team %d: %w", team.TeamNumber, err)
	}

	hostName, err := builder.selectHost(team)
	if err != nil {
		return err
	}
	host := builder.hosts[hostName]
	client := builder.clients[hostName]

	project := projectName(environment, team, build)
	description := fmt.Sprintf(
		"LaForge %s build %s team %d",
		environment.Name,
		buildID(build),
		team.TeamNumber,
	)
	if err := createOrUpdateProject(client, project, description); err != nil {
		return err
	}

	gateway := gatewayName()
	if err := builder.ensureGateway(client.UseProject(project), host, gateway, description); err != nil {
		return err
	}

	vars := cloneVars(team.Vars)
	vars[varHost] = hostName
	vars[varProject] = project
	vars[varGateway] = gateway
	if err := team.Update().SetVars(vars).Exec(ctx); err != nil {
		return fmt.Errorf("failed to update vars for team %d: %w", team.TeamNumber, err)
	}
	team.Vars = vars

	builder.Logger.Log.Infof(
		"Incus project %q for team %d placed on host %q",
		project,
		team.TeamNumber,
		hostName,
	)
	return nil
}

func (builder *IncusBuilder) ensureGateway(
	client incusclient.InstanceServer,
	host HostConfig,
	name string,
	description string,
) error {
	_, _, err := client.GetInstance(name)
	if err == nil {
		return nil
	}
	if !isNotFound(err) {
		return fmt.Errorf("failed to get Incus gateway %q: %w", name, err)
	}

	op, err := client.CreateInstance(api.InstancesPost{
		Name:  name,
		Type:  api.InstanceTypeContainer,
		Start: true,
		Source: api.InstanceSource{
			Type:  "image",
			Alias: builder.Config.GatewayImage,
		},
		InstancePut: api.InstancePut{
			Description: description + " gateway",
			Config: map[string]string{
				"cloud-init.user-data": gatewayUserData(),
			},
			Devices:  gatewayDevices(gatewaySpec{}, host),
			Profiles: []string{},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create Incus gateway %q: %w", name, err)
	}
	if err := op.Wait(); err != nil {
		return fmt.Errorf("failed waiting for Incus gateway %q: %w", name, err)
	}

	return nil
}

func (builder *IncusBuilder) TeardownTeam(ctx context.Context, team *ent.Team) error {
	if err := builder.acquireTeardownWorker(ctx); err != nil {
		return err
	}
	defer builder.TeardownWorkerPool.Release(1)

	hostName := team.Vars[varHost]
	if hostName == "" {
		// The team was never placed, so there is nothing on any host.
		return nil
	}
	client, ok := builder.clients[hostName]
	if !ok {
		return fmt.Errorf("team %d is pinned to unknown Incus host %q", team.TeamNumber, hostName)
	}

	project := team.Vars[varProject]
	if project == "" {
		build, err := team.QueryBuild().Only(ctx)
		if err != nil {
			return fmt.Errorf("failed to query build for team %d teardown: %w", team.TeamNumber, err)
		}
		environment, err := build.QueryEnvironment().Only(ctx)
		if err != nil {
			return fmt.Errorf("failed to query environment for team %d teardown: %w", team.TeamNumber, err)
		}
		project = projectName(environment, team, build)
	}

	gateway := team.Vars[varGateway]
	if gateway == "" {
		gateway = gatewayName()
	}
	if err := deleteInstance(client.UseProject(project), gateway); err != nil {
		return err
	}
	if err := client.DeleteProject(project); err != nil && !isNotFound(err) {
		return fmt.Errorf("failed to delete Incus project %q: %w", project, err)
	}

	vars := cloneVars(team.Vars)
	delete(vars, varHost)
	delete(vars, varProject)
	delete(vars, varGateway)
	delete(vars, varIngressAddr)
	delete(vars, varIngressPorts)
	if err := team.Update().SetVars(vars).Exec(ctx); err != nil {
		return fmt.Errorf("failed to clear vars for team %d: %w", team.TeamNumber, err)
	}
	team.Vars = vars

	return nil
}
