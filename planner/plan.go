package planner

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/gen0cide/laforge/ent"
	"github.com/gen0cide/laforge/ent/ansible"
	"github.com/gen0cide/laforge/ent/build"
	"github.com/gen0cide/laforge/ent/buildcommit"
	"github.com/gen0cide/laforge/ent/command"
	"github.com/gen0cide/laforge/ent/competition"
	"github.com/gen0cide/laforge/ent/dnsrecord"
	"github.com/gen0cide/laforge/ent/environment"
	"github.com/gen0cide/laforge/ent/filedelete"
	"github.com/gen0cide/laforge/ent/filedownload"
	"github.com/gen0cide/laforge/ent/fileextract"
	"github.com/gen0cide/laforge/ent/host"
	"github.com/gen0cide/laforge/ent/network"
	"github.com/gen0cide/laforge/ent/plan"
	"github.com/gen0cide/laforge/ent/provisionedhost"
	"github.com/gen0cide/laforge/ent/provisionednetwork"
	"github.com/gen0cide/laforge/ent/provisioningstep"
	"github.com/gen0cide/laforge/ent/repocommit"
	"github.com/gen0cide/laforge/ent/script"
	"github.com/gen0cide/laforge/ent/servertask"
	"github.com/gen0cide/laforge/ent/status"
	"github.com/gen0cide/laforge/ent/team"
	"github.com/gen0cide/laforge/grpc"
	"github.com/gen0cide/laforge/logging"
	"github.com/gen0cide/laforge/server/utils"
	"github.com/ghodss/yaml"
	"github.com/go-redis/redis/v8"
	_ "github.com/mattn/go-sqlite3"
	"github.com/mholt/archiver/v3"
	"github.com/sirupsen/logrus"
)

var RenderFiles = false
var RenderFilesTask *ent.ServerTask = nil
var RenderFilesTaskStatus *ent.Status = nil

// func main() {

// 	client := ent.SQLLiteOpen("file:test.sqlite?_loc=auto&cache=shared&_fk=1")
// 	ctx := context.Background()
// 	defer client.Close()

// 	// Run the auto migration tool.
// 	if err := client.Schema.Create(ctx); err != nil {
// 		logrus.Errorf("failed creating schema resources: %v", err)
// 	}
// 	uuidString := "36579b83-cc50-4f9f-a007-6da25467dc8a"
// 	envID, err := uuid.Parse(uuidString)
// 	if err != nil {
// 		logrus.Errorf("Unable to parse UUID %v. Err: %v", uuidString, err)
// 	}

// 	entEnvironment, err := client.Environment.Query().Where(environment.ID(envID)).WithEnvironmentToBuild().Only(ctx)
// 	if err != nil {
// 		logrus.Errorf("Failed to find Environment %v. Err: %v", uuidString, err)
// 	}

// 	entBuild, _ := CreateBuild(ctx, client, entEnvironment)
// 	if err != nil {
// 		logrus.Errorf("Failed to create Build for Environment %v. Err: %v", 1, err)
// 	}
// 	fmt.Println(entBuild)
// }

func createPlanningStatus(ctx context.Context, client *ent.Client, logger *logging.Logger, statusFor status.StatusFor) (*ent.Status, error) {
	logger.Log.WithFields(logrus.Fields{
		"statusFor": statusFor,
	}).Debugf("creating planning status")
	entStatus, err := client.Status.Create().SetState(status.StatePLANNING).SetStatusFor(statusFor).Save(ctx)
	if err != nil {
		logger.Log.Errorf("Failed to create Status for %v. Err: %v", statusFor, err)
		return nil, err
	}
	return entStatus, nil
}

func CreateBuild(ctx context.Context, client *ent.Client, rdb *redis.Client, laforgeConfig *utils.ServerConfig, currentUser *ent.AuthUser, entEnvironment *ent.Environment) (*ent.Build, error) {
	taskStatus, serverTask, err := utils.CreateServerTask(ctx, client, rdb, currentUser, servertask.TypeCREATEBUILD)
	if err != nil {
		return nil, fmt.Errorf("error creating server task: %v", err)
	}
	logger, err := logging.CreateLoggerForServerTask(laforgeConfig, serverTask)
	if err != nil {
		return nil, err
	}
	serverTask, err = client.ServerTask.UpdateOne(serverTask).SetServerTaskToEnvironment(entEnvironment).Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("error assigning environment to create build server task: %v", err)
	}
	rdb.Publish(ctx, "updatedServerTask", serverTask.ID.String())
	if RenderFiles {
		RenderFilesTask, err = client.ServerTask.UpdateOne(RenderFilesTask).SetServerTaskToEnvironment(entEnvironment).Save(ctx)
		if err != nil {
			return nil, fmt.Errorf("error assigning environment to render files server task: %v", err)
		}
		rdb.Publish(ctx, "updatedServerTask", RenderFilesTask.ID.String())
	}
	logger.Log.WithFields(logrus.Fields{
		"environment": entEnvironment.ID,
	}).Debugf("creating build from environment")

	var wg sync.WaitGroup
	entStatus, err := createPlanningStatus(ctx, client, logger, status.StatusForBuild)
	if err != nil {
		_, _, err = utils.FailServerTask(ctx, client, rdb, taskStatus, serverTask, err)
		if err != nil {
			return nil, fmt.Errorf("error failing server task: %v", err)
		}
		return nil, err
	}
	entCompetition, err := entEnvironment.QueryEnvironmentToCompetition().Where(competition.HCLIDEQ(entEnvironment.CompetitionID)).Only(ctx)
	if err != nil {
		logger.Log.Errorf("Failed to Query Competition %v for Environment %v. Err: %v", len(entEnvironment.CompetitionID), entEnvironment.HCLID, err)
		_, _, err = utils.FailServerTask(ctx, client, rdb, taskStatus, serverTask, err)
		if err != nil {
			return nil, fmt.Errorf("error failing server task: %v", err)
		}
		return nil, err
	}
	entRepoCommit, err := entEnvironment.QueryEnvironmentToRepository().QueryRepositoryToRepoCommit().Order(ent.Desc(repocommit.FieldRevision)).First(ctx)
	if err != nil {
		logger.Log.Errorf("Failed to Query Repository from Environment %v. Err: %v", entEnvironment.HCLID, err)
		_, _, err = utils.FailServerTask(ctx, client, rdb, taskStatus, serverTask, err)
		if err != nil {
			return nil, fmt.Errorf("error failing server task: %v", err)
		}
		return nil, err
	}
	entBuild, err := client.Build.Create().
		SetRevision(len(entEnvironment.Edges.EnvironmentToBuild)).
		SetEnvironmentRevision(entEnvironment.Revision).
		SetBuildToRepoCommit(entRepoCommit).
		SetBuildToEnvironment(entEnvironment).
		SetBuildToStatus(entStatus).
		SetBuildToCompetition(entCompetition).
		SetVars(map[string]string{}).
		Save(ctx)
	if err != nil {
		logger.Log.Errorf("Failed to create Build %v for Environment %v. Err: %v", len(entEnvironment.Edges.EnvironmentToBuild), entEnvironment.HCLID, err)
		_, _, err = utils.FailServerTask(ctx, client, rdb, taskStatus, serverTask, err)
		if err != nil {
			return nil, fmt.Errorf("error failing server task: %v", err)
		}
		return nil, err
	}
	serverTask, err = client.ServerTask.UpdateOne(serverTask).SetServerTaskToBuild(entBuild).Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("error assigning environment to create build server task: %v", err)
	}
	rdb.Publish(ctx, "updatedServerTask", serverTask.ID.String())
	if RenderFiles {
		RenderFilesTask, err = client.ServerTask.UpdateOne(RenderFilesTask).SetServerTaskToBuild(entBuild).Save(ctx)
		if err != nil {
			return nil, fmt.Errorf("error linking build to render files server task: %v", err)
		}
		rdb.Publish(ctx, "updatedServerTask", RenderFilesTask.ID.String())
	}
	entPlanStatus, err := createPlanningStatus(ctx, client, logger, status.StatusForPlan)
	if err != nil {
		_, _, err = utils.FailServerTask(ctx, client, rdb, taskStatus, serverTask, err)
		if err != nil {
			return nil, fmt.Errorf("error failing server task: %v", err)
		}
		return nil, err
	}

	_, err = client.Plan.Create().
		SetType(plan.TypeStartBuild).
		SetBuildID(entBuild.ID.String()).
		SetPlanToBuild(entBuild).
		SetStepNumber(0).
		SetPlanToStatus(entPlanStatus).
		Save(ctx)
	if err != nil {
		logger.Log.Errorf("Failed to create Plan Node for Build %v. Err: %v", entBuild.ID, err)
		_, _, err = utils.FailServerTask(ctx, client, rdb, taskStatus, serverTask, err)
		if err != nil {
			return nil, fmt.Errorf("error failing server task: %v", err)
		}
		return nil, err
	}
	for teamNumber := 1; teamNumber <= entEnvironment.TeamCount; teamNumber++ {
		wg.Add(1)
		go func(wg *sync.WaitGroup, teamNumber int, logger *logging.Logger, entBuild *ent.Build, client *ent.Client) {
			_, err := createTeam(client, laforgeConfig, logger, entBuild, teamNumber, wg)
			if err != nil {
				logrus.Errorf("error creating team: %v", err)
				logger.Log.Errorf("error creating team: %v", err)
			}
		}(&wg, teamNumber, logger, entBuild, client)
	}

	wg.Wait()

	go func(wg *sync.WaitGroup, entBuild *ent.Build) {
		wg.Wait()

		ctx := context.Background()
		defer ctx.Done()

		entCommit, err := utils.CreateRootCommit(client, rdb, entBuild)
		if err != nil {
			_, _, err = utils.FailServerTask(ctx, client, rdb, taskStatus, serverTask, err)
			return
		}
		err = entCommit.Update().AddBuildCommitToServerTask(serverTask).Exec(ctx)
		if err != nil {
			_, _, err = utils.FailServerTask(ctx, client, rdb, taskStatus, serverTask, err)
			return
		}
		err = entBuild.Update().SetBuildToLatestBuildCommit(entCommit).Exec(ctx)
		if err != nil {
			_, _, err = utils.FailServerTask(ctx, client, rdb, taskStatus, serverTask, err)
			return
		}
		rdb.Publish(ctx, "updatedBuild", entBuild.ID.String())

		if RenderFilesTask != nil {
			RenderFilesTaskStatus, RenderFilesTask, err = utils.CompleteServerTask(ctx, client, rdb, RenderFilesTaskStatus, RenderFilesTask)
			if err != nil {
				_, _, err = utils.FailServerTask(ctx, client, rdb, RenderFilesTaskStatus, RenderFilesTask, err)
				return
			}
		}
		_, serverTask, err = utils.CompleteServerTask(ctx, client, rdb, taskStatus, serverTask)
		if err != nil {
			_, _, err = utils.FailServerTask(ctx, client, rdb, taskStatus, serverTask, err)
			return
		}
		serverTask, err = client.ServerTask.UpdateOne(serverTask).SetServerTaskToBuild(entBuild).Save(ctx)
		if err != nil {
			_, _, err = utils.FailServerTask(ctx, client, rdb, taskStatus, serverTask, err)
			return
		}
		rdb.Publish(ctx, "updatedServerTask", serverTask.ID.String())
		rdb.Publish(ctx, "updatedBuild", entBuild.ID.String())
		rdb.Publish(ctx, "updatedBuildCommit", entCommit.ID.String())
		// entBuild.Update().SetCompletedPlan(true).SaveX(ctx)

		logger.Log.Debug("-----\nWAITING FOR COMMIT REVIEW\n-----")
		isApproved, err := utils.WaitForCommitReview(client, entCommit, 20*time.Minute)
		if err != nil {
			logger.Log.Errorf("error while waiting for root commit to be approved: %v", err)
			entCommit.Update().SetState(buildcommit.StateCANCELLED).Exec(ctx)
			rdb.Publish(ctx, "updatedBuildCommit", entCommit.ID.String())
			return
		}
		if isApproved {
			logger.Log.Debug("-----\nCOMMIT APPROVED\n-----")
			entEnvironment, err := entBuild.QueryBuildToEnvironment().Only(ctx)
			if err != nil {
				logger.Log.Errorf("failed to query environment from build: %v", err)
				return
			}

			taskStatus, serverTask, err := utils.CreateServerTask(ctx, client, rdb, currentUser, servertask.TypeEXECUTEBUILD)
			if err != nil {
				logger.Log.Errorf("error creating server task: %v", err)
				return
			}
			serverTask, err = client.ServerTask.UpdateOne(serverTask).SetServerTaskToBuild(entBuild).SetServerTaskToEnvironment(entEnvironment).SetServerTaskToBuildCommit(entCommit).Save(ctx)
			if err != nil {
				taskStatus, serverTask, err = utils.FailServerTask(ctx, client, rdb, taskStatus, serverTask)
				if err != nil {
					logger.Log.Errorf("error failing execute build server task: %v", err)
					return
				}
				logger.Log.Errorf("error assigning environment and build to execute build server task: %v", err)
				return
			}
			rdb.Publish(ctx, "updatedServerTask", serverTask.ID.String())

			executeLogger, err := logging.CreateLoggerForServerTask(laforgeConfig, serverTask)
			if err != nil {
				logger.Log.Errorf("error creating logger for execute build: %v", err)
				return
			}
			go StartBuild(client, laforgeConfig, executeLogger, currentUser, serverTask, taskStatus, entBuild)
		} else {
			logger.Log.Debug("-----\nCOMMIT CANCELLED/TIMED OUT\n-----")
			logger.Log.Errorf("root commit has been cancelled or 20 minute timeout has been reached")
			entCommit.Update().SetState(buildcommit.StateCANCELLED).Exec(ctx)
			rdb.Publish(ctx, "updatedBuildCommit", entCommit.ID.String())
		}
	}(&wg, entBuild)

	return entBuild, nil
}

func createTeam(client *ent.Client, laforgeConfig *utils.ServerConfig, logger *logging.Logger, entBuild *ent.Build, teamNumber int, wg *sync.WaitGroup) (*ent.Team, error) {
	logger.Log.WithFields(logrus.Fields{
		"teamNumber": teamNumber,
	}).Debug("creating team")
	defer wg.Done()

	ctx := context.Background()
	defer ctx.Done()

	entStatus, err := createPlanningStatus(ctx, client, logger, status.StatusForTeam)
	if err != nil {
		return nil, err
	}
	entTeam, err := client.Team.Create().
		SetTeamNumber(teamNumber).
		SetTeamToBuild(entBuild).
		SetTeamToStatus(entStatus).
		SetVars(map[string]string{}).
		Save(ctx)
	if err != nil {
		logger.Log.Errorf("Failed to create Team Number %v for Build %v. Err: %v", teamNumber, entBuild.ID, err)
		return nil, err
	}
	buildPlanNode, err := entBuild.QueryBuildToPlan().Where(plan.StepNumberEQ(0)).Only(ctx)
	if err != nil {
		logger.Log.Errorf("Failed to Query Plan Node for Build %v. Err: %v", entBuild.ID, err)
		return nil, err
	}
	entPlanStatus, err := createPlanningStatus(ctx, client, logger, status.StatusForPlan)
	if err != nil {
		return nil, err
	}

	_, err = client.Plan.Create().
		AddPrevPlan(buildPlanNode).
		SetType(plan.TypeStartTeam).
		SetBuildID(entBuild.ID.String()).
		SetPlanToTeam(entTeam).
		SetPlanToBuild(entBuild).
		SetStepNumber(1).
		SetPlanToStatus(entPlanStatus).
		Save(ctx)
	if err != nil {
		logger.Log.Errorf("Failed to create Plan Node for Team %v. Err: %v", teamNumber, err)
		return nil, err
	}
	buildNetworks, err := entBuild.QueryBuildToEnvironment().QueryEnvironmentToNetwork().All(ctx)
	if err != nil {
		logger.Log.Errorf("Failed to Query Environment for Build %v. Err: %v", entBuild.ID, err)
		return nil, err
	}
	createProvisonedNetworks := []*ent.ProvisionedNetwork{}
	for _, buildNetwork := range buildNetworks {
		pNetwork, err := createProvisionedNetworks(ctx, client, laforgeConfig, logger, entBuild, entTeam, buildNetwork)
		if err != nil {
			logger.Log.Errorf("Failed to Create Provisioned Network for Network %v. Err: %v", buildNetwork.Name, err)
			return nil, err
		}
		createProvisonedNetworks = append(createProvisonedNetworks, pNetwork)
	}
	for _, pNetwork := range createProvisonedNetworks {
		entHosts, err := pNetwork.
			QueryProvisionedNetworkToNetwork().
			QueryNetworkToIncludedNetwork().
			QueryIncludedNetworkToHost().
			All(ctx)
		if err != nil {
			logger.Log.Errorf("Failed to Query Hosts for Network %v. Err: %v", pNetwork.Name, err)
			return nil, err
		}
		networkPlan, err := pNetwork.QueryProvisionedNetworkToPlan().Only(ctx)
		if err != nil {
			logger.Log.Errorf("Failed to Query Plan for Network %v. Err: %v", pNetwork.Name, err)
			return nil, err
		}
		for _, entHost := range entHosts {
			_, err = createProvisionedHosts(ctx, client, laforgeConfig, logger, pNetwork, entHost, networkPlan)
			if err != nil {
				logrus.Errorf("Failed to create provisioned hosts")
				logger.Log.Errorf("Failed to create provisioned hosts")
				return nil, err
			}
		}
	}
	return entTeam, nil
}

func createProvisionedNetworks(ctx context.Context, client *ent.Client, laforgeConfig *utils.ServerConfig, logger *logging.Logger, entBuild *ent.Build, entTeam *ent.Team, entNetwork *ent.Network) (*ent.ProvisionedNetwork, error) {
	logger.Log.WithFields(logrus.Fields{
		"team":            entTeam.ID,
		"team.teamNumber": entTeam.TeamNumber,
		"network":         entNetwork.ID,
		"network.Name":    entNetwork.Name,
	}).Debug("creating provisioned network")
	entStatus, err := createPlanningStatus(ctx, client, logger, status.StatusForProvisionedNetwork)
	if err != nil {
		return nil, err
	}

	entProvisionedNetwork, err := client.ProvisionedNetwork.Create().
		SetName(entNetwork.Name).
		SetCidr(entNetwork.Cidr).
		SetVars(map[string]string{}).
		SetProvisionedNetworkToStatus(entStatus).
		SetProvisionedNetworkToNetwork(entNetwork).
		SetProvisionedNetworkToTeam(entTeam).
		SetProvisionedNetworkToBuild(entBuild).
		Save(ctx)
	if err != nil {
		logger.Log.Errorf("Failed to create Provisoned Network %v for Team %v. Err: %v", entNetwork.Name, entTeam.TeamNumber, err)
		return nil, err
	}
	teamPlanNode, err := entTeam.QueryTeamToPlan().Only(ctx)
	if err != nil {
		logger.Log.Errorf("Failed to Query Plan Node for Build %v. Err: %v", entBuild.ID, err)
		return nil, err
	}

	entPlanStatus, err := createPlanningStatus(ctx, client, logger, status.StatusForPlan)
	if err != nil {
		return nil, err
	}
	_, err = client.Plan.Create().
		AddPrevPlan(teamPlanNode).
		SetType(plan.TypeProvisionNetwork).
		SetBuildID(entBuild.ID.String()).
		SetPlanToProvisionedNetwork(entProvisionedNetwork).
		SetPlanToBuild(entBuild).
		SetStepNumber(teamPlanNode.StepNumber + 1).
		SetPlanToStatus(entPlanStatus).
		Save(ctx)
	if err != nil {
		logger.Log.Errorf("Failed to create Plan Node for Provisioned Network  %v. Err: %v", entProvisionedNetwork.Name, err)
		return nil, err
	}
	return entProvisionedNetwork, nil
}

func createProvisionedHosts(ctx context.Context, client *ent.Client, laforgeConfig *utils.ServerConfig, logger *logging.Logger, pNetwork *ent.ProvisionedNetwork, entHost *ent.Host, prevPlan *ent.Plan) (*ent.ProvisionedHost, error) {
	logger.Log.WithFields(logrus.Fields{
		"pNetwork":      pNetwork.ID,
		"pNetwork.Name": pNetwork.Name,
		"host":          entHost.ID,
		"host.Hostname": entHost.Hostname,
	}).Debug("creating provisioned host")

	prevPlans := []*ent.Plan{prevPlan}
	// logger.Log.Infof("START  %s | %s | %v", pNetwork.Name, entHost.Hostname, prevPlans)
	planStepNumber := prevPlan.StepNumber + 1
	entProvisionedHost, err := client.ProvisionedHost.Query().Where(
		provisionedhost.And(
			provisionedhost.HasProvisionedHostToProvisionedNetworkWith(
				provisionednetwork.IDEQ(pNetwork.ID),
			),
			provisionedhost.HasProvisionedHostToHostWith(
				host.IDEQ(entHost.ID),
			),
		),
	).Only(ctx)
	if err != nil {
		if err != err.(*ent.NotFoundError) {
			logger.Log.Errorf("Failed to Query Existing Host %v. Err: %v", entHost.HCLID, err)
			return nil, err
		}
	} else {
		return entProvisionedHost, nil
	}

	entHostDependencies, err := entHost.QueryDependByHostToHostDependency().
		WithHostDependencyToDependOnHost().
		WithHostDependencyToNetwork().
		All(ctx)

	currentBuild := pNetwork.QueryProvisionedNetworkToBuild().WithBuildToEnvironment().OnlyX(ctx)
	currentTeam := pNetwork.QueryProvisionedNetworkToTeam().OnlyX(ctx)

	for _, entHostDependency := range entHostDependencies {
		entDependsOnHost, err := client.ProvisionedHost.Query().Where(
			provisionedhost.And(
				provisionedhost.HasProvisionedHostToProvisionedNetworkWith(
					provisionednetwork.And(
						provisionednetwork.HasProvisionedNetworkToNetworkWith(
							network.IDEQ(entHostDependency.Edges.HostDependencyToNetwork.ID),
						),
						provisionednetwork.HasProvisionedNetworkToBuildWith(
							build.IDEQ(currentBuild.ID),
						),
						provisionednetwork.HasProvisionedNetworkToTeamWith(
							team.IDEQ(currentTeam.ID),
						),
					),
				),
				provisionedhost.HasProvisionedHostToHostWith(
					host.IDEQ(entHostDependency.Edges.HostDependencyToDependOnHost.ID),
				),
			),
		).WithProvisionedHostToPlan().Only(ctx)
		if err != nil {
			if err != err.(*ent.NotFoundError) {
				logger.Log.Errorf("Failed to Query Depended On Host %v for Host %v. Err: %v", entHostDependency.Edges.HostDependencyToDependOnHost.HCLID, entHost.HCLID, err)
				return nil, err
			} else {
				dependOnPnetwork, err := client.ProvisionedNetwork.Query().Where(
					provisionednetwork.And(
						provisionednetwork.HasProvisionedNetworkToNetworkWith(
							network.IDEQ(entHostDependency.Edges.HostDependencyToNetwork.ID),
						),
						provisionednetwork.HasProvisionedNetworkToBuildWith(
							build.IDEQ(currentBuild.ID),
						),
						provisionednetwork.HasProvisionedNetworkToTeamWith(
							team.IDEQ(currentTeam.ID),
						),
					),
				).Only(ctx)
				if err != nil {
					logger.Log.Errorf("Failed to Query Provined Network %v for Depended On Host %v. Err: %v", entHostDependency.Edges.HostDependencyToNetwork.HCLID, entHostDependency.Edges.HostDependencyToDependOnHost.HCLID, err)
				}
				dependOnPnetworkPlan, err := dependOnPnetwork.QueryProvisionedNetworkToPlan().Only(ctx)
				if err != nil {
					logger.Log.Errorf("error while retrieving plan from provisioned network: %v", err)
					return nil, err
				}
				entDependsOnHost, err = createProvisionedHosts(ctx, client, laforgeConfig, logger, dependOnPnetwork, entHostDependency.Edges.HostDependencyToDependOnHost, dependOnPnetworkPlan)
				if err != nil {
					logger.Log.Errorf("error creating depends on host: %v", err)
					return nil, err
				}
			}
		}
		dependOnPlan, err := entDependsOnHost.QueryProvisionedHostToEndStepPlan().Only(ctx)
		if err != nil && err != err.(*ent.NotFoundError) {
			logger.Log.Errorf("Failed to Query Depended On Host %v Plan for Host %v. Err: %v", entHostDependency.Edges.HostDependencyToDependOnHost.HCLID, entHost.HCLID, err)
			return nil, err
		}
		prevPlans = append(prevPlans, dependOnPlan)
		if planStepNumber <= dependOnPlan.StepNumber {
			planStepNumber = dependOnPlan.StepNumber + 1
		}

	}

	subnetIP, err := CalcIP(pNetwork.Cidr, entHost.LastOctet)
	if err != nil {
		return nil, err
	}

	entStatus, err := createPlanningStatus(ctx, client, logger, status.StatusForProvisionedHost)
	if err != nil {
		return nil, err
	}

	entProvisionedHost, err = client.ProvisionedHost.Create().
		SetSubnetIP(subnetIP).
		SetVars(map[string]string{}).
		SetProvisionedHostToStatus(entStatus).
		SetProvisionedHostToProvisionedNetwork(pNetwork).
		SetProvisionedHostToHost(entHost).
		SetProvisionedHostToBuild(currentBuild).
		Save(ctx)
	if err != nil {
		logrus.Errorf("error while creating provisioned host: %v", err)
		logger.Log.Errorf("error while creating provisioned host: %v", err)
		return nil, err
	}

	if entHost.Tags["root-dns"] == "true" {
		entProvisionedHost, err = entProvisionedHost.Update().SetAddonType(provisionedhost.AddonTypeDNS).Save(ctx)
		if err != nil {
			return nil, err
		}
	}

	entPlanStatus, err := createPlanningStatus(ctx, client, logger, status.StatusForPlan)
	if err != nil {
		return nil, err
	}

	// logger.Log.Infof("CREATE %s | %s | %v", pNetwork.Name, entHost.Hostname, prevPlans)
	endPlanNode, err := client.Plan.Create().
		AddPrevPlan(prevPlans...).
		SetType(plan.TypeProvisionHost).
		SetBuildID(prevPlan.BuildID).
		SetPlanToProvisionedHost(entProvisionedHost).
		SetStepNumber(planStepNumber).
		SetPlanToBuild(currentBuild).
		SetPlanToStatus(entPlanStatus).
		Save(ctx)

	if err != nil {
		logger.Log.Errorf("Failed to create Plan Node for Provisioned Host  %v. Err: %v", entHost.HCLID, err)
		return nil, err
	}

	isWindowsHost := false
	if strings.Contains(entHost.OS, "w2k") {
		isWindowsHost = true
	}

	binaryPath := path.Join("builds", currentBuild.Edges.BuildToEnvironment.Name, fmt.Sprint(currentBuild.Revision), fmt.Sprint(currentTeam.TeamNumber), pNetwork.Name, entHost.Hostname)
	os.MkdirAll(binaryPath, 0755)
	binaryName := path.Join(binaryPath, "laforgeAgent")
	if isWindowsHost {
		binaryName = binaryName + ".exe"
	}
	binaryName, err = filepath.Abs(binaryName)
	if err != nil {
		logger.Log.Errorf("Unable to Resolve Absolute File Path. Err: %v", err)
		return nil, err
	}
	if RenderFiles {
		err = grpc.BuildAgent(logger, fmt.Sprint(entProvisionedHost.ID), laforgeConfig.Agent.GrpcServerUri, binaryName, isWindowsHost)
		if err != nil {
			return nil, err
		}
		entTmpUrl, err := utils.CreateTempURL(ctx, client, binaryName)
		if err != nil {
			return nil, err
		}
		_, err = entTmpUrl.Update().SetGinFileMiddlewareToProvisionedHost(entProvisionedHost).Save(ctx)
		if err != nil {
			return nil, err
		}
		if RenderFilesTask != nil {
			RenderFilesTask, err = RenderFilesTask.Update().AddServerTaskToGinFileMiddleware(entTmpUrl).Save(ctx)
			if err != nil {
				return nil, err
			}
		}
	}
	userDataScriptID, ok := entHost.Vars["user_data_script_id"]
	if ok {
		currentEnvironment, err := entProvisionedHost.QueryProvisionedHostToHost().QueryHostToEnvironment().Only(ctx)
		userDataScript, err := client.Script.Query().Where(
			script.And(
				script.HasScriptToEnvironmentWith(
					environment.IDEQ(currentEnvironment.ID),
				),
				script.HCLIDEQ(userDataScriptID),
			),
		).Only(ctx)
		if err != nil {
			logger.Log.Errorf("Failed to Query Script %v. Err: %v", userDataScriptID, err)
			return nil, err
		}
		entUserDataStatus, err := client.Status.Create().SetState(status.StateCOMPLETE).SetStatusFor(status.StatusForProvisioningStep).Save(ctx)
		if err != nil {
			logger.Log.Errorf("Failed to Create Provisioning Step Status for Script %v. Err: %v", userDataScriptID, err)
			return nil, err
		}
		entUserDataProvisioningStep, err := client.ProvisioningStep.Create().
			SetStepNumber(0).
			SetType(provisioningstep.TypeScript).
			SetProvisioningStepToScript(userDataScript).
			SetProvisioningStepToProvisionedHost(entProvisionedHost).
			SetProvisioningStepToStatus(entUserDataStatus).
			Save(ctx)
		if err != nil {
			logger.Log.Errorf("Failed to Create Provisioning Step for Script %v. Err: %v", userDataScriptID, err)
			return nil, err
		}
		if RenderFiles {
			filePath, err := renderScript(ctx, client, logger, entUserDataProvisioningStep)
			if err != nil {
				return nil, err
			}
			entTmpUrl, err := utils.CreateTempURL(ctx, client, filePath)
			if err != nil {
				return nil, err
			}
			_, err = entTmpUrl.Update().SetGinFileMiddlewareToProvisioningStep(entUserDataProvisioningStep).Save(ctx)
			if err != nil {
				return nil, err
			}
			if RenderFilesTask != nil {
				RenderFilesTask, err = RenderFilesTask.Update().AddServerTaskToGinFileMiddleware(entTmpUrl).Save(ctx)
				if err != nil {
					return nil, err
				}
			}
		}

	}

	for stepNumber, pStep := range entHost.ProvisionSteps {
		stepNumber = stepNumber + 1
		entProvisioningStep, err := createProvisioningStep(ctx, client, logger, pStep, stepNumber, entProvisionedHost, endPlanNode)
		if err != nil {
			return nil, err
		}
		endPlanNode, err = entProvisioningStep.QueryProvisioningStepToPlan().Only(ctx)
		if err != nil {
			return nil, err
		}
	}
	_, err = entProvisionedHost.Update().SetProvisionedHostToEndStepPlan(endPlanNode).Save(ctx)
	if err != nil {
		logger.Log.Errorf("Unable to Update The End Step. Err: %v", err)
		return nil, err
	}

	return entProvisionedHost, nil
}

func createProvisioningStep(ctx context.Context, client *ent.Client, logger *logging.Logger, hclID string, stepNumber int, pHost *ent.ProvisionedHost, prevPlan *ent.Plan) (*ent.ProvisioningStep, error) {
	entHost, err := pHost.QueryProvisionedHostToHost().Only(ctx)
	if err != nil {
		logger.Log.Errorf("Failed to Query Host for Provisoned Host %v. Err: %v", pHost.ID, err)
		return nil, err
	}
	logger.Log.WithFields(logrus.Fields{
		"pHost":               pHost.ID,
		"pHost.HCLID":         entHost.HCLID,
		"pHost.SubnetIP":      pHost.SubnetIP,
		"stepNumber":          stepNumber,
		"prevPlan":            prevPlan.ID,
		"prevPlan.Type":       prevPlan.Type,
		"prevPlan.StepNumber": prevPlan.StepNumber,
	}).Debug("creating provisioned step")
	var entProvisioningStep *ent.ProvisioningStep
	currentEnvironment, err := pHost.QueryProvisionedHostToHost().QueryHostToEnvironment().Only(ctx)
	currentBuild := pHost.QueryProvisionedHostToProvisionedNetwork().QueryProvisionedNetworkToBuild().WithBuildToEnvironment().OnlyX(ctx)
	if err != nil {
		logger.Log.Errorf("Failed to Query Current Environment for Provisoned Host %v. Err: %v", pHost.ID, err)
		return nil, err
	}
	entStatus, err := createPlanningStatus(ctx, client, logger, status.StatusForProvisioningStep)
	if err != nil {
		return nil, err
	}
	entScript, err := client.Script.Query().Where(
		script.And(
			script.HasScriptToEnvironmentWith(
				environment.IDEQ(currentEnvironment.ID),
			),
			script.HCLIDEQ(hclID),
		),
	).Only(ctx)
	if err != nil {
		if err != err.(*ent.NotFoundError) {
			logger.Log.WithFields(logrus.Fields{
				"pHost":               pHost.ID,
				"pHost.HCLID":         entHost.HCLID,
				"pHost.SubnetIP":      pHost.SubnetIP,
				"stepNumber":          stepNumber,
				"prevPlan":            prevPlan.ID,
				"prevPlan.Type":       prevPlan.Type,
				"prevPlan.StepNumber": prevPlan.StepNumber,
			}).Errorf("Failed to Query Script %v. Err: %v", hclID, err)
			return nil, err
		} else {
			entCommand, err := client.Command.Query().Where(
				command.And(
					command.HasCommandToEnvironmentWith(
						environment.IDEQ(currentEnvironment.ID),
					),
					command.HCLIDEQ(hclID),
				)).Only(ctx)
			if err != nil {
				if err != err.(*ent.NotFoundError) {
					logger.Log.WithFields(logrus.Fields{
						"pHost":               pHost.ID,
						"pHost.HCLID":         entHost.HCLID,
						"pHost.SubnetIP":      pHost.SubnetIP,
						"stepNumber":          stepNumber,
						"prevPlan":            prevPlan.ID,
						"prevPlan.Type":       prevPlan.Type,
						"prevPlan.StepNumber": prevPlan.StepNumber,
					}).Errorf("Failed to Query Command %v. Err: %v", hclID, err)
					return nil, err
				} else {
					entFileDownload, err := client.FileDownload.Query().Where(
						filedownload.And(
							filedownload.HasFileDownloadToEnvironmentWith(
								environment.IDEQ(currentEnvironment.ID),
							),
							filedownload.HCLIDEQ(hclID),
						)).Only(ctx)
					if err != nil {
						if err != err.(*ent.NotFoundError) {
							logger.Log.WithFields(logrus.Fields{
								"pHost":               pHost.ID,
								"pHost.HCLID":         entHost.HCLID,
								"pHost.SubnetIP":      pHost.SubnetIP,
								"stepNumber":          stepNumber,
								"prevPlan":            prevPlan.ID,
								"prevPlan.Type":       prevPlan.Type,
								"prevPlan.StepNumber": prevPlan.StepNumber,
							}).Errorf("Failed to Query FileDownload %v. Err: %v", hclID, err)
							return nil, err
						} else {
							entFileExtract, err := client.FileExtract.Query().Where(
								fileextract.And(
									fileextract.HasFileExtractToEnvironmentWith(
										environment.IDEQ(currentEnvironment.ID),
									),
									fileextract.HCLIDEQ(hclID),
								)).Only(ctx)
							if err != nil {
								if err != err.(*ent.NotFoundError) {
									logger.Log.WithFields(logrus.Fields{
										"pHost":               pHost.ID,
										"pHost.HCLID":         entHost.HCLID,
										"pHost.SubnetIP":      pHost.SubnetIP,
										"stepNumber":          stepNumber,
										"prevPlan":            prevPlan.ID,
										"prevPlan.Type":       prevPlan.Type,
										"prevPlan.StepNumber": prevPlan.StepNumber,
									}).Errorf("Failed to Query FileExtract %v. Err: %v", hclID, err)
									return nil, err
								} else {
									entFileDelete, err := client.FileDelete.Query().Where(
										filedelete.And(
											filedelete.HasFileDeleteToEnvironmentWith(
												environment.IDEQ(currentEnvironment.ID),
											),
											filedelete.HCLIDEQ(hclID),
										)).Only(ctx)
									if err != nil {
										if err != err.(*ent.NotFoundError) {
											logger.Log.WithFields(logrus.Fields{
												"pHost":               pHost.ID,
												"pHost.HCLID":         entHost.HCLID,
												"pHost.SubnetIP":      pHost.SubnetIP,
												"stepNumber":          stepNumber,
												"prevPlan":            prevPlan.ID,
												"prevPlan.Type":       prevPlan.Type,
												"prevPlan.StepNumber": prevPlan.StepNumber,
											}).Errorf("Failed to Query FileDelete %v. Err: %v", hclID, err)
											return nil, err
										} else {
											entDNSRecord, err := client.DNSRecord.Query().Where(
												dnsrecord.And(
													dnsrecord.HasDNSRecordToEnvironmentWith(
														environment.IDEQ(currentEnvironment.ID),
													),
													dnsrecord.HCLIDEQ(hclID),
												)).Only(ctx)
											if err != nil {
												if err != err.(*ent.NotFoundError) {
													logger.Log.WithFields(logrus.Fields{
														"pHost":               pHost.ID,
														"pHost.HCLID":         entHost.HCLID,
														"pHost.SubnetIP":      pHost.SubnetIP,
														"stepNumber":          stepNumber,
														"prevPlan":            prevPlan.ID,
														"prevPlan.Type":       prevPlan.Type,
														"prevPlan.StepNumber": prevPlan.StepNumber,
													}).Errorf("Failed to Query FileDelete %v. Err: %v", hclID, err)
													return nil, err
												} else {
													entAnsible, err := client.Ansible.Query().Where(
														ansible.And(
															ansible.HasAnsibleFromEnvironmentWith(
																environment.IDEQ(currentEnvironment.ID),
															),
															ansible.HCLIDEQ(hclID),
														)).Only(ctx)
													if err != nil {
														if err != err.(*ent.NotFoundError) {
															logger.Log.WithFields(logrus.Fields{
																"pHost":               pHost.ID,
																"pHost.HCLID":         entHost.HCLID,
																"pHost.SubnetIP":      pHost.SubnetIP,
																"stepNumber":          stepNumber,
																"prevPlan":            prevPlan.ID,
																"prevPlan.Type":       prevPlan.Type,
																"prevPlan.StepNumber": prevPlan.StepNumber,
															}).Errorf("Failed to Query Ansible %v. Err: %v", hclID, err)
															return nil, err
														} else {
															logger.Log.WithFields(logrus.Fields{
																"pHost":               pHost.ID,
																"pHost.HCLID":         entHost.HCLID,
																"pHost.SubnetIP":      pHost.SubnetIP,
																"stepNumber":          stepNumber,
																"prevPlan":            prevPlan.ID,
																"prevPlan.Type":       prevPlan.Type,
																"prevPlan.StepNumber": prevPlan.StepNumber,
															}).Errorf("No Provisioning Steps found for %v. Err: %v", hclID, err)
															return nil, err
														}
													} else {
														entProvisioningStep, err = client.ProvisioningStep.Create().
															SetStepNumber(stepNumber).
															SetType(provisioningstep.TypeAnsible).
															SetProvisioningStepToAnsible(entAnsible).
															SetProvisioningStepToStatus(entStatus).
															SetProvisioningStepToProvisionedHost(pHost).
															Save(ctx)
														if err != nil {
															logger.Log.WithFields(logrus.Fields{
																"pHost":               pHost.ID,
																"pHost.HCLID":         entHost.HCLID,
																"pHost.SubnetIP":      pHost.SubnetIP,
																"stepNumber":          stepNumber,
																"prevPlan":            prevPlan.ID,
																"prevPlan.Type":       prevPlan.Type,
																"prevPlan.StepNumber": prevPlan.StepNumber,
															}).Errorf("Failed to Create Provisioning Step for Ansible %v. Err: %v", hclID, err)
															return nil, err
														}
														if RenderFiles {
															filePath, err := renderAnsible(ctx, client, logger, entProvisioningStep)
															if err != nil {
																return nil, err
															}
															entTmpUrl, err := utils.CreateTempURL(ctx, client, filePath)
															if err != nil {
																return nil, err
															}
															_, err = entTmpUrl.Update().SetGinFileMiddlewareToProvisioningStep(entProvisioningStep).Save(ctx)
															if err != nil {
																return nil, err
															}
															if RenderFilesTask != nil {
																RenderFilesTask, err = RenderFilesTask.Update().AddServerTaskToGinFileMiddleware(entTmpUrl).Save(ctx)
																if err != nil {
																	return nil, err
																}
															}
														}
													}
												}
											} else {
												entProvisioningStep, err = client.ProvisioningStep.Create().
													SetStepNumber(stepNumber).
													SetType(provisioningstep.TypeDNSRecord).SetProvisioningStepToDNSRecord(entDNSRecord).
													SetProvisioningStepToStatus(entStatus).
													SetProvisioningStepToProvisionedHost(pHost).
													Save(ctx)
												if err != nil {
													logger.Log.WithFields(logrus.Fields{
														"pHost":               pHost.ID,
														"pHost.HCLID":         entHost.HCLID,
														"pHost.SubnetIP":      pHost.SubnetIP,
														"stepNumber":          stepNumber,
														"prevPlan":            prevPlan.ID,
														"prevPlan.Type":       prevPlan.Type,
														"prevPlan.StepNumber": prevPlan.StepNumber,
													}).Errorf("Failed to Create Provisioning Step for FileDelete %v. Err: %v", hclID, err)
													return nil, err
												}
											}
										}
									} else {
										entProvisioningStep, err = client.ProvisioningStep.Create().
											SetStepNumber(stepNumber).
											SetType(provisioningstep.TypeFileDelete).SetProvisioningStepToFileDelete(entFileDelete).
											SetProvisioningStepToStatus(entStatus).
											SetProvisioningStepToProvisionedHost(pHost).
											Save(ctx)
										if err != nil {
											logger.Log.WithFields(logrus.Fields{
												"pHost":               pHost.ID,
												"pHost.HCLID":         entHost.HCLID,
												"pHost.SubnetIP":      pHost.SubnetIP,
												"stepNumber":          stepNumber,
												"prevPlan":            prevPlan.ID,
												"prevPlan.Type":       prevPlan.Type,
												"prevPlan.StepNumber": prevPlan.StepNumber,
											}).Errorf("Failed to Create Provisioning Step for FileDelete %v. Err: %v", hclID, err)
											return nil, err
										}
									}
								}
							} else {
								entProvisioningStep, err = client.ProvisioningStep.Create().
									SetStepNumber(stepNumber).
									SetType(provisioningstep.TypeFileExtract).
									SetProvisioningStepToFileExtract(entFileExtract).
									SetProvisioningStepToStatus(entStatus).
									SetProvisioningStepToProvisionedHost(pHost).
									Save(ctx)
								if err != nil {
									logger.Log.WithFields(logrus.Fields{
										"pHost":               pHost.ID,
										"pHost.HCLID":         entHost.HCLID,
										"pHost.SubnetIP":      pHost.SubnetIP,
										"stepNumber":          stepNumber,
										"prevPlan":            prevPlan.ID,
										"prevPlan.Type":       prevPlan.Type,
										"prevPlan.StepNumber": prevPlan.StepNumber,
									}).Errorf("Failed to Create Provisioning Step for FileExtract %v. Err: %v", hclID, err)
									return nil, err
								}
							}
						}
					} else {
						entProvisioningStep, err = client.ProvisioningStep.Create().
							SetStepNumber(stepNumber).
							SetType(provisioningstep.TypeFileDownload).
							SetProvisioningStepToFileDownload(entFileDownload).
							SetProvisioningStepToStatus(entStatus).
							SetProvisioningStepToProvisionedHost(pHost).
							Save(ctx)
						if err != nil {
							logger.Log.WithFields(logrus.Fields{
								"pHost":               pHost.ID,
								"pHost.HCLID":         entHost.HCLID,
								"pHost.SubnetIP":      pHost.SubnetIP,
								"stepNumber":          stepNumber,
								"prevPlan":            prevPlan.ID,
								"prevPlan.Type":       prevPlan.Type,
								"prevPlan.StepNumber": prevPlan.StepNumber,
							}).Errorf("Failed to Create Provisioning Step for FileDownload %v. Err: %v", hclID, err)
							return nil, err
						}
						if RenderFiles {
							filePath, err := renderFileDownload(ctx, logger, entProvisioningStep)
							if err != nil {
								return nil, err
							}
							entTmpUrl, err := utils.CreateTempURL(ctx, client, filePath)
							if err != nil {
								return nil, err
							}
							_, err = entTmpUrl.Update().SetGinFileMiddlewareToProvisioningStep(entProvisioningStep).Save(ctx)
							if err != nil {
								return nil, err
							}
							if RenderFilesTask != nil {
								RenderFilesTask, err = RenderFilesTask.Update().AddServerTaskToGinFileMiddleware(entTmpUrl).Save(ctx)
								if err != nil {
									return nil, err
								}
							}
						}
					}
				}
			} else {
				entProvisioningStep, err = client.ProvisioningStep.Create().
					SetStepNumber(stepNumber).
					SetType(provisioningstep.TypeCommand).
					SetProvisioningStepToCommand(entCommand).
					SetProvisioningStepToStatus(entStatus).
					SetProvisioningStepToProvisionedHost(pHost).
					Save(ctx)
				if err != nil {
					logger.Log.WithFields(logrus.Fields{
						"pHost":               pHost.ID,
						"pHost.HCLID":         entHost.HCLID,
						"pHost.SubnetIP":      pHost.SubnetIP,
						"stepNumber":          stepNumber,
						"prevPlan":            prevPlan.ID,
						"prevPlan.Type":       prevPlan.Type,
						"prevPlan.StepNumber": prevPlan.StepNumber,
					}).Errorf("Failed to Create Provisioning Step for Command %v. Err: %v", hclID, err)
					return nil, err
				}
			}
		}
	} else {
		entProvisioningStep, err = client.ProvisioningStep.Create().
			SetStepNumber(stepNumber).
			SetType(provisioningstep.TypeScript).
			SetProvisioningStepToScript(entScript).
			SetProvisioningStepToStatus(entStatus).
			SetProvisioningStepToProvisionedHost(pHost).
			Save(ctx)
		if err != nil {
			logger.Log.WithFields(logrus.Fields{
				"pHost":               pHost.ID,
				"pHost.HCLID":         entHost.HCLID,
				"pHost.SubnetIP":      pHost.SubnetIP,
				"stepNumber":          stepNumber,
				"prevPlan":            prevPlan.ID,
				"prevPlan.Type":       prevPlan.Type,
				"prevPlan.StepNumber": prevPlan.StepNumber,
			}).Errorf("Failed to Create Provisioning Step for Script %v. Err: %v", hclID, err)
			return nil, err
		}
		if RenderFiles {
			filePath, err := renderScript(ctx, client, logger, entProvisioningStep)
			if err != nil {
				return nil, err
			}
			entTmpUrl, err := utils.CreateTempURL(ctx, client, filePath)
			if err != nil {
				return nil, err
			}
			_, err = entTmpUrl.Update().SetGinFileMiddlewareToProvisioningStep(entProvisioningStep).Save(ctx)
			if err != nil {
				return nil, err
			}
			if RenderFilesTask != nil {
				RenderFilesTask, err = RenderFilesTask.Update().AddServerTaskToGinFileMiddleware(entTmpUrl).Save(ctx)
				if err != nil {
					return nil, err
				}
			}
		}
	}

	entPlanStatus, err := createPlanningStatus(ctx, client, logger, status.StatusForPlan)
	if err != nil {
		return nil, err
	}

	_, err = client.Plan.Create().
		AddPrevPlan(prevPlan).
		SetType(plan.TypeExecuteStep).
		SetBuildID(prevPlan.BuildID).
		SetPlanToProvisioningStep(entProvisioningStep).
		SetStepNumber(prevPlan.StepNumber + 1).
		SetPlanToBuild(currentBuild).
		SetPlanToStatus(entPlanStatus).
		Save(ctx)

	if err != nil {
		logger.Log.WithFields(logrus.Fields{
			"pHost":               pHost.ID,
			"pHost.HCLID":         entHost.HCLID,
			"pHost.SubnetIP":      pHost.SubnetIP,
			"stepNumber":          stepNumber,
			"prevPlan":            prevPlan.ID,
			"prevPlan.Type":       prevPlan.Type,
			"prevPlan.StepNumber": prevPlan.StepNumber,
		}).Errorf("Failed to Create Plan Node for Provisioning Step %v. Err: %v", entProvisioningStep.ID, err)
		return nil, err
	}

	return entProvisioningStep, nil

}

func renderScript(ctx context.Context, client *ent.Client, logger *logging.Logger, pStep *ent.ProvisioningStep) (string, error) {
	logger.Log.WithFields(logrus.Fields{
		"pStep":            pStep.ID,
		"pStep.StepNumber": pStep.StepNumber,
		"pStep.Type":       pStep.Type,
	}).Debug("render script")
	currentProvisionedHost := pStep.QueryProvisioningStepToProvisionedHost().OnlyX(ctx)
	currentScript := pStep.QueryProvisioningStepToScript().OnlyX(ctx)
	currentProvisionedNetwork := currentProvisionedHost.QueryProvisionedHostToProvisionedNetwork().OnlyX(ctx)
	currentTeam := currentProvisionedNetwork.QueryProvisionedNetworkToTeam().OnlyX(ctx)
	currentBuild := currentTeam.QueryTeamToBuild().OnlyX(ctx)
	currentEnvironment := currentBuild.QueryBuildToEnvironment().OnlyX(ctx)
	currentIncludedNetwork := currentEnvironment.QueryEnvironmentToIncludedNetwork().WithIncludedNetworkToHost().WithIncludedNetworkToNetwork().AllX(ctx)
	currentCompetition := currentBuild.QueryBuildToCompetition().OnlyX(ctx)
	currentNetwork := currentProvisionedNetwork.QueryProvisionedNetworkToNetwork().OnlyX(ctx)
	currentHost := currentProvisionedHost.QueryProvisionedHostToHost().OnlyX(ctx)
	currentIdentities := currentEnvironment.QueryEnvironmentToIdentity().AllX(ctx)
	agentScriptFile := currentProvisionedHost.QueryProvisionedHostToGinFileMiddleware().OnlyX(ctx)
	// Need to Make Unique and change how it's loaded in
	currentDNS := currentCompetition.QueryCompetitionToDNS().FirstX(ctx)
	templateData := TempleteContext{
		Build:              currentBuild,
		Competition:        currentCompetition,
		Environment:        currentEnvironment,
		Host:               currentHost,
		DNS:                currentDNS,
		IncludedNetworks:   currentIncludedNetwork,
		Network:            currentNetwork,
		Script:             currentScript,
		Team:               currentTeam,
		Identities:         currentIdentities,
		ProvisionedNetwork: currentProvisionedNetwork,
		ProvisionedHost:    currentProvisionedHost,
		ProvisioningStep:   pStep,
		AgentSlug:          agentScriptFile.URLID,
	}
	t, err := template.New(strings.Replace(currentScript.Source, "./", "", -1)).Funcs(TemplateFuncLib).ParseFiles(currentScript.AbsPath)
	if err != nil {
		logger.Log.Errorf("Failed to Parse template for script %v. Err: %v", currentScript.Name, err)
		return "", err
	}
	fileRelativePath := path.Join("builds", currentEnvironment.Name, fmt.Sprint(currentBuild.Revision), fmt.Sprint(currentTeam.TeamNumber), currentProvisionedNetwork.Name, currentHost.Hostname)
	os.MkdirAll(fileRelativePath, 0755)
	fileName := filepath.Base(currentScript.Source)
	fileName = path.Join(fileRelativePath, fileName)
	fileName, err = filepath.Abs(fileName)
	if err != nil {
		return "", err
	}
	f, err := os.Create(fileName)
	if err != nil {
		logger.Log.Errorf("Error Generating Script %v. Err: %v", currentScript.Name, err)
		return "", err
	}
	err = t.Execute(f, templateData)
	if err != nil {
		logger.Log.WithFields(logrus.Fields{
			"scriptName": currentScript.Name,
			"path":       currentScript.AbsPath,
		}).Errorf("error while parsing template for script: %v", err)
	}
	f.Close()
	return fileName, nil
}

func renderFileDownload(ctx context.Context, logger *logging.Logger, pStep *ent.ProvisioningStep) (string, error) {
	logger.Log.WithFields(logrus.Fields{
		"pStep":            pStep.ID,
		"pStep.StepNumber": pStep.StepNumber,
		"pStep.Type":       pStep.Type,
	}).Debug("render file download")
	currentFileDownload := pStep.QueryProvisioningStepToFileDownload().OnlyX(ctx)
	currentProvisionedHost := pStep.QueryProvisioningStepToProvisionedHost().OnlyX(ctx)
	currentProvisionedNetwork := currentProvisionedHost.QueryProvisionedHostToProvisionedNetwork().OnlyX(ctx)
	currentHost := currentProvisionedHost.QueryProvisionedHostToHost().OnlyX(ctx)
	currentTeam := currentProvisionedNetwork.QueryProvisionedNetworkToTeam().OnlyX(ctx)
	currentBuild := currentTeam.QueryTeamToBuild().OnlyX(ctx)
	currentEnvironment := currentBuild.QueryBuildToEnvironment().OnlyX(ctx)

	fileRelativePath := path.Join("builds", currentEnvironment.Name, fmt.Sprint(currentBuild.Revision), fmt.Sprint(currentTeam.TeamNumber), currentProvisionedNetwork.Name, currentHost.Hostname)
	os.MkdirAll(fileRelativePath, 0755)
	fileName := filepath.Base(currentFileDownload.Source)
	fileName = path.Join(fileRelativePath, fileName)
	fileName, err := filepath.Abs(fileName)
	if err != nil {
		return "", err
	}
	destFile, err := os.Create(fileName)
	if err != nil {
		err = fmt.Errorf("error creating file download: %v", err)
		return "", err
	}
	defer destFile.Close()

	// TODO: SOMETHING
	if currentFileDownload.SourceType == "remote" {
		// http.Get(currentFileDownload.Source)
	} else {
		srcFile, err := os.Open(currentFileDownload.AbsPath)
		if err != nil {
			return "", err
		}
		defer srcFile.Close()
		_, err = io.Copy(destFile, srcFile)
		if err != nil {
			return "", err
		}
	}
	return fileName, nil
}

func renderAnsible(ctx context.Context, client *ent.Client, logger *logging.Logger, pStep *ent.ProvisioningStep) (string, error) {
	logger.Log.WithFields(logrus.Fields{
		"pStep":            pStep.ID,
		"pStep.StepNumber": pStep.StepNumber,
		"pStep.Type":       pStep.Type,
	}).Debug("render ansible")
	currentProvisionedHost := pStep.QueryProvisioningStepToProvisionedHost().OnlyX(ctx)
	currentAnsible := pStep.QueryProvisioningStepToAnsible().OnlyX(ctx)
	currentProvisionedNetwork := currentProvisionedHost.QueryProvisionedHostToProvisionedNetwork().OnlyX(ctx)
	currentTeam := currentProvisionedNetwork.QueryProvisionedNetworkToTeam().OnlyX(ctx)
	currentBuild := currentTeam.QueryTeamToBuild().OnlyX(ctx)
	currentEnvironment := currentBuild.QueryBuildToEnvironment().OnlyX(ctx)
	currentIncludedNetwork := currentEnvironment.QueryEnvironmentToIncludedNetwork().WithIncludedNetworkToHost().WithIncludedNetworkToNetwork().AllX(ctx)
	currentCompetition := currentBuild.QueryBuildToCompetition().OnlyX(ctx)
	currentNetwork := currentProvisionedNetwork.QueryProvisionedNetworkToNetwork().OnlyX(ctx)
	currentHost := currentProvisionedHost.QueryProvisionedHostToHost().OnlyX(ctx)
	currentIdentities := currentEnvironment.QueryEnvironmentToIdentity().AllX(ctx)
	agentScriptFile := currentProvisionedHost.QueryProvisionedHostToGinFileMiddleware().OnlyX(ctx)
	// Need to Make Unique and change how it's loaded in
	currentDNS := currentCompetition.QueryCompetitionToDNS().FirstX(ctx)
	templateData := TempleteContext{
		Build:              currentBuild,
		Competition:        currentCompetition,
		Environment:        currentEnvironment,
		Host:               currentHost,
		DNS:                currentDNS,
		IncludedNetworks:   currentIncludedNetwork,
		Network:            currentNetwork,
		Ansible:            currentAnsible,
		Team:               currentTeam,
		Identities:         currentIdentities,
		ProvisionedNetwork: currentProvisionedNetwork,
		ProvisionedHost:    currentProvisionedHost,
		ProvisioningStep:   pStep,
		AgentSlug:          agentScriptFile.URLID,
	}

	dirRelativePath := path.Join("builds", currentEnvironment.Name, fmt.Sprint(currentBuild.Revision), fmt.Sprint(currentTeam.TeamNumber), currentProvisionedNetwork.Name, currentHost.Hostname)
	dirAbsPath, err := filepath.Abs(dirRelativePath)
	if err != nil {
		logger.Log.Errorf("Error Generating Absolute Directory Path for Ansible %v. Err: %v", currentAnsible.Name, err)
		return "", err
	}

	ansibleFolder := path.Join(dirAbsPath, currentAnsible.Name)
	err = os.MkdirAll(ansibleFolder, 0755)
	if err != nil {
		logger.Log.Errorf("Failed to create folder for ansible %v. Err: %v", currentAnsible.HCLID, err)
		return "", err
	}

	data, err := yaml.Marshal(templateData)
	if err != nil {
		logger.Log.Errorf("Failed to render vars file for ansible %v. Err: %v", currentAnsible.HCLID, err)
		return "", err
	}
	varFileName := path.Join(ansibleFolder, "laforge_vars.yml")
	err = ioutil.WriteFile(varFileName, data, 0755)
	if err != nil {
		logger.Log.Errorf("Failed to create vars file for ansible %v. Err: %v", currentAnsible.HCLID, err)
		return "", err
	}

	err = CopyDir(currentAnsible.AbsPath, ansibleFolder)
	if err != nil {
		logger.Log.Errorf("Failed to copy folder for ansible %v. Err: %v", currentAnsible.HCLID, err)
		return "", err
	}

	zipFileName := path.Join(dirAbsPath, currentAnsible.Name+".zip")
	err = archiver.Archive([]string{ansibleFolder}, zipFileName)
	if err != nil {
		logger.Log.WithFields(logrus.Fields{
			"ansibleName": currentAnsible.Name,
			"path":        currentAnsible.AbsPath,
		}).Errorf("error while creating zip for ansible: %v", err)
	}

	return zipFileName, nil
}

// IPv42Int converts net.IP address objects to their uint32 representation
func IPv42Int(ip net.IP) uint32 {
	if len(ip) == 16 {
		return binary.BigEndian.Uint32(ip[12:16])
	}
	return binary.BigEndian.Uint32(ip)
}

// Int2IPv4 converts uint32s to their net.IP object
func Int2IPv4(nn uint32) net.IP {
	ip := make(net.IP, 4)
	binary.BigEndian.PutUint32(ip, nn)
	return ip
}

func CopyDir(src string, dest string) error {

	if dest == src {
		return fmt.Errorf("cannot copy a folder into the folder itself")
	}

	f, err := os.Open(src)
	if err != nil {
		return err
	}

	file, err := f.Stat()
	if err != nil {
		return err
	}
	if !file.IsDir() {
		return fmt.Errorf("Source " + file.Name() + " is not a directory!")
	}

	err = os.MkdirAll(dest, 0755)
	if err != nil {
		return err
	}

	files, err := ioutil.ReadDir(src)
	if err != nil {
		return err
	}

	for _, f := range files {

		if f.IsDir() {

			err = CopyDir(src+"/"+f.Name(), dest+"/"+f.Name())
			if err != nil {
				return err
			}

		}

		if !f.IsDir() {

			content, err := ioutil.ReadFile(src + "/" + f.Name())
			if err != nil {
				return err

			}

			err = ioutil.WriteFile(dest+"/"+f.Name(), content, 0755)
			if err != nil {
				return err

			}

		}

	}

	return nil
}
