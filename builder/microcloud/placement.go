package microcloud

import (
	"context"
	"fmt"

	"github.com/canonical/lxd/shared/api"
	"github.com/gen0cide/laforge/ent"
)

func (builder *MicroCloudBuilder) DeployTeam(ctx context.Context, team *ent.Team) error {
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

	project := projectName(environment, team, build)
	group := clusterGroupName(environment, team, build)
	member := team.Vars["lxd_cluster_member"]
	if member != "" {
		found := false
		for _, onlineMember := range builder.clusterMembers {
			if onlineMember == member {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf(
				"team %d is pinned to unavailable LXD cluster member %q; explicit migration or reassignment is required",
				team.TeamNumber,
				member,
			)
		}
	} else {
		memberIndex := team.TeamNumber - 1
		if memberIndex < 0 {
			memberIndex = 0
		}
		member = builder.clusterMembers[memberIndex%len(builder.clusterMembers)]
	}
	description := fmt.Sprintf(
		"LaForge %s build %s team %d",
		environment.Name,
		buildID(build),
		team.TeamNumber,
	)

	if err := createOrUpdateClusterGroup(
		builder.Client,
		group,
		member,
		description,
	); err != nil {
		return err
	}
	if err := createOrUpdateProject(
		builder.Client,
		project,
		group,
		builder.Config.UplinkNetwork,
		description,
	); err != nil {
		return err
	}

	vars := cloneVars(team.Vars)
	vars["lxd_project"] = project
	vars["lxd_cluster_group"] = group
	vars["lxd_cluster_member"] = member
	if err := team.Update().SetVars(vars).Exec(ctx); err != nil {
		return fmt.Errorf("failed to update vars for team %d: %w", team.TeamNumber, err)
	}
	team.Vars = vars

	builder.Logger.Log.Infof(
		"MicroCloud project %q for team %d pinned to member %q",
		project,
		team.TeamNumber,
		member,
	)
	return nil
}

func (builder *MicroCloudBuilder) TeardownTeam(ctx context.Context, team *ent.Team) error {
	if err := builder.acquireTeardownWorker(ctx); err != nil {
		return err
	}
	defer builder.TeardownWorkerPool.Release(1)

	project := team.Vars["lxd_project"]
	group := team.Vars["lxd_cluster_group"]
	if project == "" || group == "" {
		build, err := team.QueryBuild().Only(ctx)
		if err != nil {
			return fmt.Errorf("failed to query build for team %d teardown: %w", team.TeamNumber, err)
		}
		environment, err := build.QueryEnvironment().Only(ctx)
		if err != nil {
			return fmt.Errorf("failed to query environment for team %d teardown: %w", team.TeamNumber, err)
		}
		project = projectName(environment, team, build)
		group = clusterGroupName(environment, team, build)
	}

	if err := builder.Client.DeleteProject(project); err != nil && !isNotFound(err) {
		return fmt.Errorf("failed to delete LXD project %q: %w", project, err)
	}

	existingGroup, etag, err := builder.Client.GetClusterGroup(group)
	if err != nil && !isNotFound(err) {
		return fmt.Errorf("failed to get LXD cluster group %q: %w", group, err)
	}
	if err == nil {
		if len(existingGroup.Members) > 0 {
			if err := builder.Client.UpdateClusterGroup(group, api.ClusterGroupPut{
				Description: existingGroup.Description,
				Members:     []string{},
			}, etag); err != nil {
				return fmt.Errorf("failed to empty LXD cluster group %q: %w", group, err)
			}
		}
		if err := builder.Client.DeleteClusterGroup(group); err != nil && !isNotFound(err) {
			return fmt.Errorf("failed to delete LXD cluster group %q: %w", group, err)
		}
	}

	vars := cloneVars(team.Vars)
	delete(vars, "lxd_project")
	delete(vars, "lxd_cluster_group")
	delete(vars, "lxd_cluster_member")
	delete(vars, "lxd_networks")
	delete(vars, "ingress_address")
	if err := team.Update().SetVars(vars).Exec(ctx); err != nil {
		return fmt.Errorf("failed to clear vars for team %d: %w", team.TeamNumber, err)
	}

	return nil
}
