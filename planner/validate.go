package planner

import (
	"context"
	"fmt"

	"github.com/gen0cide/laforge/ent"
	"github.com/gen0cide/laforge/ent/competition"
	"github.com/gen0cide/laforge/ent/plan"
	"github.com/gen0cide/laforge/ent/status"
	"github.com/gen0cide/laforge/logging"
	"github.com/gen0cide/laforge/server/utils"
)

// ValidationSummary describes the plan generated for one environment.
type ValidationSummary struct {
	Environment       string `json:"environment"`
	Teams             int    `json:"teams"`
	Networks          int    `json:"networks"`
	Hosts             int    `json:"hosts"`
	ProvisioningSteps int    `json:"provisioning_steps"`
	ScheduledSteps    int    `json:"scheduled_steps"`
	Plans             int    `json:"plans"`
}

// ValidateBuild generates a complete build plan without rendering artifacts or
// contacting the configured infrastructure builder.
func ValidateBuild(
	ctx context.Context,
	client *ent.Client,
	laforgeConfig *utils.ServerConfig,
	logger *logging.Logger,
	entEnvironment *ent.Environment,
) (*ValidationSummary, error) {
	entCompetition, err := entEnvironment.
		QueryCompetitions().
		Where(competition.HCLIDEQ(entEnvironment.CompetitionID)).
		Only(ctx)
	if err != nil {
		return nil, fmt.Errorf(
			"query competition %q for environment %q: %w",
			entEnvironment.CompetitionID,
			entEnvironment.HCLID,
			err,
		)
	}

	buildStatus, err := createPlanningStatus(ctx, client, logger, status.StatusForBuild)
	if err != nil {
		return nil, fmt.Errorf("create build status: %w", err)
	}

	entBuild, err := client.Build.Create().
		SetRevision(0).
		SetEnvironmentRevision(entEnvironment.Revision).
		SetEnvironment(entEnvironment).
		SetStatus(buildStatus).
		SetCompetition(entCompetition).
		SetVars(map[string]string{}).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("create build for environment %q: %w", entEnvironment.HCLID, err)
	}

	planStatus, err := createPlanningStatus(ctx, client, logger, status.StatusForPlan)
	if err != nil {
		return nil, fmt.Errorf("create root plan status: %w", err)
	}
	if _, err := client.Plan.Create().
		SetType(plan.TypeStartBuild).
		SetBuild(entBuild).
		SetStepNumber(0).
		SetStatus(planStatus).
		Save(ctx); err != nil {
		return nil, fmt.Errorf("create root plan: %w", err)
	}

	for teamNumber := 1; teamNumber <= entEnvironment.TeamCount; teamNumber++ {
		if _, err := createTeam(client, laforgeConfig, logger, entBuild, teamNumber, false); err != nil {
			return nil, fmt.Errorf("plan team %d: %w", teamNumber, err)
		}
	}

	if err := entBuild.Update().SetCompletedPlan(true).Exec(ctx); err != nil {
		return nil, fmt.Errorf("mark build plan complete: %w", err)
	}

	summary := &ValidationSummary{
		Environment: entEnvironment.HCLID,
	}
	if summary.Teams, err = entBuild.QueryTeams().Count(ctx); err != nil {
		return nil, fmt.Errorf("count teams: %w", err)
	}
	if summary.Networks, err = entBuild.QueryProvisionedNetworks().Count(ctx); err != nil {
		return nil, fmt.Errorf("count provisioned networks: %w", err)
	}
	if summary.Hosts, err = entBuild.QueryProvisionedNetworks().QueryProvisionedHosts().Count(ctx); err != nil {
		return nil, fmt.Errorf("count provisioned hosts: %w", err)
	}
	if summary.ProvisioningSteps, err = entBuild.QueryProvisionedNetworks().
		QueryProvisionedHosts().
		QueryProvisioningSteps().
		Count(ctx); err != nil {
		return nil, fmt.Errorf("count provisioning steps: %w", err)
	}
	if summary.ScheduledSteps, err = entBuild.QueryProvisionedNetworks().
		QueryProvisionedHosts().
		QueryProvisioningScheduledSteps().
		Count(ctx); err != nil {
		return nil, fmt.Errorf("count scheduled steps: %w", err)
	}
	if summary.Plans, err = entBuild.QueryPlans().Count(ctx); err != nil {
		return nil, fmt.Errorf("count plans: %w", err)
	}

	return summary, nil
}
