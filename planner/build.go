package planner

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gen0cide/laforge/builder"
	"github.com/gen0cide/laforge/ent"
	"github.com/gen0cide/laforge/ent/agenttask"
	"github.com/gen0cide/laforge/ent/buildcommit"
	"github.com/gen0cide/laforge/ent/plan"
	"github.com/gen0cide/laforge/ent/provisionedhost"
	"github.com/gen0cide/laforge/ent/provisionednetwork"
	"github.com/gen0cide/laforge/ent/provisioningstep"
	"github.com/gen0cide/laforge/ent/scheduledstep"
	"github.com/gen0cide/laforge/ent/status"
	"github.com/gen0cide/laforge/ent/validation"
	"github.com/gen0cide/laforge/logging"
	"github.com/gen0cide/laforge/server/utils"
	"github.com/google/uuid"
	"github.com/gorhill/cronexpr"
	"github.com/sirupsen/logrus"
)

var cancelMap = map[uuid.UUID]context.CancelFunc{}

func CancelBuild(id uuid.UUID) bool {
	if cancelMap[id] != nil {
		cancelMap[id]()
		delete(cancelMap, id)
		return true
	}
	return false
}

func StartBuild(client *ent.Client, laforgeConfig *utils.ServerConfig, logger *logging.Logger, currentUser *ent.AuthUser, serverTask *ent.ServerTask, taskStatus *ent.Status, entBuild *ent.Build) error {
	logger.Log.Debug("BUILDER | START BUILD")
	ctx, cancel := context.WithCancel(context.Background())
	defer ctx.Done()
	cancelMap[entBuild.ID] = cancel
	ctxClosing := context.Background()
	defer ctxClosing.Done()
	entPlans, err := entBuild.QueryPlans().Where(plan.HasStatusWith(status.StateEQ(status.StatePLANNING))).All(ctx)

	if err != nil {
		taskStatus, serverTask, err = utils.FailServerTask(ctx, client, rdb, taskStatus, serverTask)
		if err != nil {
			logger.Log.Errorf("Failed to Query Plan Nodes %v. Err: %v", entPlans, err)
			return err
		}
		logger.Log.Errorf("Failed to Query Plan Nodes %v. Err: %v", entPlans, err)
		return err
	}

	var wg sync.WaitGroup

	for _, entPlan := range entPlans {
		entStatus, err := entPlan.QueryStatus().Only(ctx)

		if err != nil {
			logger.Log.Errorf("Failed to Query Status %v. Err: %v", entPlan, err)
			return err
		}

		wg.Add(1)

		go func(wg *sync.WaitGroup, entStatus *ent.Status) {
			defer wg.Done()
			ctx := context.Background()
			defer ctx.Done()
			entStatus.Update().SetState(status.StateAWAITING).Save(ctx)
			rdb.Publish(ctx, "updatedStatus", entStatus.ID.String())
		}(&wg, entStatus)

		wg.Add(1)
		go func(wg *sync.WaitGroup, entPlan *ent.Plan) {
			defer wg.Done()
			ctx := context.Background()
			defer ctx.Done()
			switch entPlan.Type {
			case plan.TypeProvisionNetwork:
				entProNetwork, err := entPlan.QueryProvisionedNetwork().Only(ctx)
				if err != nil {
					logger.Log.Errorf("Failed to Query Provisioned Network. Err: %v", err)
					return
				}
				entStatus, err := entProNetwork.QueryStatus().Only(ctx)
				if err != nil {
					logger.Log.Errorf("Failed to Query Status %v. Err: %v", entPlan, err)
					return
				}
				entStatus.Update().SetState(status.StateAWAITING).Save(ctx)
				rdb.Publish(ctx, "updatedStatus", entStatus.ID.String())
			case plan.TypeProvisionHost:
				entProHost, err := entPlan.QueryProvisionedHost().Only(ctx)
				if err != nil {
					logger.Log.Errorf("Failed to Query Provisioned Host. Err: %v", err)
					return
				}
				entStatus, err := entProHost.QueryStatus().Only(ctx)
				if err != nil {
					logger.Log.Errorf("Failed to Query Status %v. Err: %v", entPlan, err)
					return
				}
				entStatus.Update().SetState(status.StateAWAITING).Save(ctx)
				rdb.Publish(ctx, "updatedStatus", entStatus.ID.String())
			case plan.TypeExecuteStep:
				entProvisioningStep, err := entPlan.QueryProvisioningStep().Only(ctx)
				if err != nil {
					logger.Log.Errorf("Failed to Query Provisioning Step. Err: %v", err)
					return
				}
				entStatus, err := entProvisioningStep.QueryStatus().Only(ctx)
				if err != nil {
					logger.Log.Errorf("Failed to Query Status %v. Err: %v", entPlan, err)
					return
				}
				entStatus.Update().SetState(status.StateAWAITING).Save(ctx)
				rdb.Publish(ctx, "updatedStatus", entStatus.ID.String())
			case plan.TypeStartTeam:
				entTeam, err := entPlan.QueryTeam().Only(ctx)
				if err != nil {
					logger.Log.Errorf("Failed to Query Provisioning Step. Err: %v", err)
					return
				}
				entStatus, err := entTeam.QueryStatus().Only(ctx)
				if err != nil {
					logger.Log.Errorf("Failed to Query Status %v. Err: %v", entPlan, err)
					return
				}
				entStatus.Update().SetState(status.StateAWAITING).Save(ctx)
				rdb.Publish(ctx, "updatedStatus", entStatus.ID.String())
			case plan.TypeStartBuild:
				entBuild, err := entPlan.QueryBuild().Only(ctx)
				if err != nil {
					logger.Log.Errorf("Failed to Query Provisioning Step. Err: %v", err)
					return
				}
				entStatus, err := entBuild.QueryStatus().Only(ctx)
				if err != nil {
					logger.Log.Errorf("Failed to Query Status %v. Err: %v", entPlan, err)
					return
				}
				entStatus.Update().SetState(status.StateAWAITING).Save(ctx)
				rdb.Publish(ctx, "updatedStatus", entStatus.ID.String())
			default:
				break
			}
		}(&wg, entPlan)
	}

	wg.Wait()

	rootPlans, err := entBuild.QueryPlans().Where(plan.TypeEQ(plan.TypeStartBuild)).All(ctx)
	if err != nil {
		taskStatus, serverTask, err = utils.FailServerTask(ctxClosing, client, rdb, taskStatus, serverTask)
		if err != nil {
			logger.Log.Errorf("error failing execute build server task: %v", err)
			return err
		}
		logger.Log.Errorf("Failed to Query Start Plan Nodes. Err: %v", err)
		return err
	}
	environment, err := entBuild.QueryEnvironment().Only(ctx)
	if err != nil {
		taskStatus, serverTask, err = utils.FailServerTask(ctxClosing, client, rdb, taskStatus, serverTask)
		if err != nil {
			logger.Log.Errorf("error failing execute build server task: %v", err)
			return err
		}
		logger.Log.Errorf("Failed to Query Environment. Err: %v", err)
		return err
	}

	genericBuilder, err := builder.BuilderFromEnvironment(laforgeConfig.Builders, environment, logger)
	if err != nil {
		logger.Log.Errorf("error generating builder: %v", err)
		taskStatus, serverTask, err = utils.FailServerTask(ctxClosing, client, rdb, taskStatus, serverTask)
		if err != nil {
			logger.Log.Errorf("error failing execute build server task: %v", err)
			return err
		}
		return err
	}

	entRootCommit, err := entBuild.QueryLatestBuildCommit().Only(ctx)
	if err != nil {
		logger.Log.Errorf("error while querying lastest commit from build: %v", err)
		return err
	}

	err = entRootCommit.Update().SetState(buildcommit.StateINPROGRESS).Exec(ctxClosing)
	if err != nil {
		logger.Log.Errorf("error while cancelling rebuild commit: %v", err)
		return err
	}
	rdb.Publish(ctxClosing, "updatedBuildCommit", entRootCommit.ID.String())

	for _, entPlan := range rootPlans {
		wg.Add(1)
		go buildRoutine(client, laforgeConfig, logger, &genericBuilder, ctx, entPlan, &wg)
	}

	wg.Wait()

	if ctx.Err() != nil {
		taskStatus, serverTask, err = utils.FailServerTask(ctxClosing, client, rdb, taskStatus, serverTask)
		if err != nil {
			logger.Log.Errorf("error failing execute build server task: %v", err)
			return err
		}
	} else {
		taskStatus, serverTask, err = utils.CompleteServerTask(ctxClosing, client, rdb, taskStatus, serverTask)
		if err != nil {
			logger.Log.Errorf("error completing execute build server task: %v", err)
			return err
		}
		delete(cancelMap, entBuild.ID)
	}

	err = entRootCommit.Update().SetState(buildcommit.StateAPPLIED).Exec(ctxClosing)
	if err != nil {
		logger.Log.Errorf("error while cancelling rebuild commit: %v", err)
		return err
	}
	rdb.Publish(ctxClosing, "updatedBuildCommit", entRootCommit.ID.String())

	return nil
}

func buildRoutine(client *ent.Client, laforgeConfig *utils.ServerConfig, logger *logging.Logger, builder *builder.Builder, ctx context.Context, entPlan *ent.Plan, wg *sync.WaitGroup) {
	logger.Log.WithFields(logrus.Fields{
		"plan": entPlan.ID,
	}).Debugf("BUILDER | BUILD ROUTINE START")
	defer wg.Done()
	ctxClosing := context.Background()
	defer ctxClosing.Done()

	entStatus, err := entPlan.QueryStatus().Only(ctx)

	if err != nil {
		logger.Log.Errorf("Failed to Query Status %v. Err: %v", entPlan, err)
		return
	}

	// If it isn't marked for planning, don't worry about traversing to it
	if entStatus.State != status.StateAWAITING {
		logger.Log.WithFields(logrus.Fields{
			"plan": entPlan.ID,
		}).Debugf("BUILDER | already awaiting. EXITING")
		return
	}

	// If it's already in progress, don't worry about traversing to it
	if entStatus.State == status.StatePARENTAWAITING || entStatus.State == status.StateINPROGRESS || entStatus.State == status.StateCOMPLETE {
		logger.Log.WithFields(logrus.Fields{
			"plan": entPlan.ID,
		}).Debugf("BUILDER | node already in progress. EXITING")
		return
	}

	prevNodes, err := entPlan.QueryPrevPlans().All(ctx)

	if err != nil {
		logger.Log.Errorf("Failed to Query Plan Start %v. Err: %v", prevNodes, err)
		return
	}

	logger.Log.WithFields(logrus.Fields{
		"plan": entPlan.ID,
	}).Debugf("BUILDER | waiting on parents")

	parentNodeFailed := false

	entStatus, err = entStatus.Update().SetState(status.StatePARENTAWAITING).Save(ctxClosing)
	if err != nil {
		logger.Log.WithFields(logrus.Fields{
			"plan": entPlan.ID,
		}).Error("BUILDER | failed to set PARENTAWAITING status. EXITING")
		return
	}

	for _, prevNode := range prevNodes {
		for {

			if parentNodeFailed {
				break
			}

			prevCompletedStatus, err := prevNode.QueryStatus().Where(
				status.StateNEQ(
					status.StateCOMPLETE,
				),
			).Exist(ctx)

			if err != nil {
				logger.Log.Errorf("Failed to Query Status %v. Err: %v", prevNode, err)
				return
			}

			prevFailedStatus, err := prevNode.QueryStatus().Where(
				status.StateEQ(
					status.StateFAILED,
				),
			).Exist(ctx)

			if err != nil {
				logger.Log.Errorf("Failed to Query Status %v. Err: %v", prevNode, err)
				return
			}

			if !prevCompletedStatus {
				break
			}

			if prevFailedStatus {
				parentNodeFailed = true
				break
			}

			time.Sleep(time.Second)
		}
	}
	logger.Log.WithFields(logrus.Fields{
		"plan": entPlan.ID,
	}).Debugf("BUILDER | done waiting on parents")
	entStatus, err = entPlan.QueryStatus().Only(ctx)

	if err != nil {
		logger.Log.Errorf("Failed to Query Status %v. Err: %v", entPlan, err)
		return
	}

	entStatus.Update().SetState(status.StateINPROGRESS).Save(ctxClosing)
	rdb.Publish(ctxClosing, "updatedStatus", entStatus.ID.String())

	var planErr error = nil
	switch entPlan.Type {
	case plan.TypeProvisionNetwork:
		entProNetwork, err := entPlan.QueryProvisionedNetwork().Only(ctx)
		if err != nil {
			logger.Log.Errorf("Failed to Query Provisioned Network. Err: %v", err)
			return
		}
		if parentNodeFailed {
			networkStatus, err := entProNetwork.QueryStatus().Only(ctxClosing)
			if err != nil {
				logger.Log.Errorf("Error while getting Provisioned Network status: %v", err)
				return
			}
			_, saveErr := networkStatus.Update().SetFailed(true).SetState(status.StateFAILED).Save(ctxClosing)
			if saveErr != nil {
				logger.Log.Errorf("Error while setting Provisioned Network status to FAILED: %v", saveErr)
				return
			}
			rdb.Publish(ctxClosing, "updatedStatus", networkStatus.ID.String())
			planErr = fmt.Errorf("parent node for Provionded Network has failed")
		} else {
			planErr = buildNetwork(client, logger, builder, ctx, entProNetwork)
		}
	case plan.TypeProvisionHost:
		entProHost, err := entPlan.QueryProvisionedHost().Only(ctx)
		if err != nil {
			logger.Log.Errorf("Failed to Query Provisioned Host. Err: %v", err)
			return
		}
		if parentNodeFailed {
			hostStatus, err := entProHost.QueryStatus().Only(ctxClosing)
			if err != nil {
				logger.Log.Errorf("Error while getting Provisioned Network status: %v", err)
				return
			}
			_, saveErr := hostStatus.Update().SetFailed(true).SetState(status.StateFAILED).Save(ctxClosing)
			if saveErr != nil {
				logger.Log.Errorf("Error while setting Provisioned Network status to FAILED: %v", saveErr)
				return
			}
			rdb.Publish(ctxClosing, "updatedStatus", hostStatus.ID.String())
			planErr = fmt.Errorf("parent node for Provionded Host has failed")
		} else {
			planErr = buildHost(client, logger, builder, ctx, entProHost)
		}
	case plan.TypeExecuteStep:
		entProvisioningStep, err := entPlan.QueryProvisioningStep().Only(ctx)
		if err != nil {
			logger.Log.Errorf("Failed to Query Provisioning Step. Err: %v", err)
			return
		}
		if parentNodeFailed {
			stepStatus, err := entProvisioningStep.QueryStatus().Only(ctxClosing)
			if err != nil {
				logger.Log.Errorf("Failed to Query Provisioning Step Status. Err: %v", err)
				return
			}
			_, err = stepStatus.Update().SetFailed(true).SetState(status.StateFAILED).Save(ctxClosing)
			if err != nil {
				logger.Log.Errorf("error while trying to set ent.ProvisioningStep.Status.State to status.StateFAILED: %v", err)
				return
			}
			rdb.Publish(ctxClosing, "updatedStatus", stepStatus.ID.String())
			planErr = fmt.Errorf("parent node for Provisioning Step has failed")
		} else {
			planErr = execStep(client, laforgeConfig, logger, ctx, entProvisioningStep)
		}
	case plan.TypeStartTeam:
		entTeam, err := entPlan.QueryTeam().Only(ctx)
		if err != nil {
			logger.Log.Errorf("Failed to Query Ent Tean. Err: %v", err)
			return
		}
		if parentNodeFailed {
			teamStatus, err := entTeam.QueryStatus().Only(ctxClosing)
			if err != nil {
				logger.Log.Errorf("Failed to Query Provisioning Step Status. Err: %v", err)
				return
			}
			_, err = teamStatus.Update().SetFailed(true).SetState(status.StateFAILED).Save(ctxClosing)
			if err != nil {
				logger.Log.Errorf("error while trying to set ent.ProvisioningStep.Status.State to status.StateFAILED: %v", err)
				return
			}
			rdb.Publish(ctxClosing, "updatedStatus", teamStatus.ID.String())
			planErr = fmt.Errorf("parent node for Team has failed")
		} else {
			planErr = buildTeam(client, logger, builder, ctx, entTeam)
		}
	case plan.TypeStartBuild:
		entBuild, err := entPlan.QueryBuild().Only(ctx)
		if err != nil {
			logger.Log.Errorf("Failed to Query Provisioning Step. Err: %v", err)
			return
		}
		entStatus, err := entBuild.QueryStatus().Only(ctx)
		if err != nil {
			logger.Log.Errorf("Failed to Query Status %v. Err: %v", entPlan, err)
			return
		}
		entStatus.Update().SetState(status.StateCOMPLETE).Save(ctxClosing)
		rdb.Publish(ctxClosing, "updatedStatus", entStatus.ID.String())
	case plan.TypeStartScheduledStep:
		entProvisioningScheduledStep, err := entPlan.QueryProvisioningScheduledStep().Only(ctx)
		if err != nil {
			logger.Log.Errorf("Failed to Query Provisioning Scheduled Step. Err: %v", err)
			return
		}
		if parentNodeFailed {
			planErr = fmt.Errorf("parent node for Provisioning Step has failed")
		} else {
			planErr = startScheduledStep(client, laforgeConfig, logger, ctx, entProvisioningScheduledStep)
		}
	default:
		break
	}

	if planErr != nil {
		entStatus.Update().SetState(status.StateFAILED).SetFailed(true).Save(ctxClosing)
		rdb.Publish(ctxClosing, "updatedStatus", entStatus.ID.String())
		logger.Log.WithFields(logrus.Fields{
			"type":    entPlan.Type,
			"builder": (*builder).ID(),
		}).Errorf("error while executing plan: %v", planErr)
	} else {

		entStatus.Update().SetState(status.StateCOMPLETE).SetCompleted(true).Save(ctxClosing)
		rdb.Publish(ctxClosing, "updatedStatus", entStatus.ID.String())
	}

	logger.Log.WithFields(logrus.Fields{
		"plan": entPlan.ID,
	}).Debugf("BUILDER | plan done. SPAWNING CHILDREN")

	nextPlans, err := entPlan.QueryNextPlans().All(ctx)
	for _, nextPlan := range nextPlans {
		wg.Add(1)
		go buildRoutine(client, laforgeConfig, logger, builder, ctx, nextPlan, wg)
	}

}

func buildHost(client *ent.Client, logger *logging.Logger, builder *builder.Builder, ctx context.Context, entProHost *ent.ProvisionedHost) error {
	entProNet, err := entProHost.QueryProvisionedNetwork().First(ctx)
	if err != nil {
		logger.Log.WithFields(logrus.Fields{
			"entProHost": entProHost.ID,
		}).Error("error querying host and provisioned network from provisioned host")
		return err
	} else {
		entTeam, err := entProNet.QueryTeam().First(ctx)
		if err != nil {
			logger.Log.WithFields(logrus.Fields{
				"entProNet": entProNet.ID,
			}).Error("error querying team from provisioned network")
			return err
		} else {
			logger.Log.WithFields(logrus.Fields{
				"subnetIp":  entProHost.SubnetIP,
				"entProNet": entProNet.Name,
				"entTeam":   entTeam.TeamNumber,
			}).Debugf("BUILDER | BUILD ROUTINE START")
		}
	}
	logger.Log.Infof("deploying %s", entProHost.SubnetIP)
	hostStatus, err := entProHost.QueryStatus().Only(ctx)
	if err != nil {
		logger.Log.Errorf("Error while getting Provisioned Host status: %v", err)
		return err
	}
	entProNetwork, err := entProHost.QueryProvisionedNetwork().Only(ctx)
	if err != nil {
		logger.Log.Errorf("Error while checking if host step is failed: %v", err)
		return err
	}

	_, saveErr := hostStatus.Update().SetState(status.StateINPROGRESS).Save(ctx)
	if saveErr != nil {
		logger.Log.Errorf("Error while setting Provisioned Host status to INPROGRESS: %v", saveErr)
		return saveErr
	}
	rdb.Publish(ctx, "updatedStatus", hostStatus.ID.String())
	err = (*builder).DeployHost(ctx, entProHost)
	if err != nil {
		logger.Log.Errorf("Error while deploying host: %v", err)
		_, saveErr := hostStatus.Update().SetFailed(true).SetState(status.StateFAILED).Save(ctx)
		if saveErr != nil {
			logger.Log.Errorf("Error while setting Provisioned Host status to FAILED: %v", saveErr)
			return saveErr
		}
		rdb.Publish(ctx, "updatedStatus", hostStatus.ID.String())
		checkNetworkStatus(client, logger, ctx, entProNetwork)
		return err
	}
	logger.Log.Infof("deployed %s successfully", entProHost.SubnetIP)
	// _, saveErr = hostStatus.Update().SetCompleted(true).SetState(status.StateCOMPLETE).Save(ctx)
	// if saveErr != nil {
	// 	logger.Log.Errorf("Error while setting Provisioned Host status to COMPLETE: %v", saveErr)
	// 	return saveErr
	// }
	rdb.Publish(ctx, "updatedStatus", hostStatus.ID.String())
	return nil
}

func buildNetwork(client *ent.Client, logger *logging.Logger, builder *builder.Builder, ctx context.Context, entProNetwork *ent.ProvisionedNetwork) error {
	logger.Log.Infof("deploying %s", entProNetwork.Name)
	networkStatus, err := entProNetwork.QueryStatus().Only(ctx)
	if err != nil {
		logger.Log.Errorf("Error while getting Provisioned Network status: %v", err)
		return err
	}
	entTeam, err := entProNetwork.QueryTeam().Only(ctx)
	if err != nil {
		logger.Log.Errorf("Error while getting team: %v", err)
		return err
	}
	_, saveErr := networkStatus.Update().SetState(status.StateINPROGRESS).Save(ctx)
	if saveErr != nil {
		logger.Log.Errorf("Error while setting Provisioned Network status to INPROGRESS: %v", saveErr)
		return saveErr
	}
	rdb.Publish(ctx, "updatedStatus", networkStatus.ID.String())
	err = (*builder).DeployNetwork(ctx, entProNetwork)
	if err != nil {
		logger.Log.Errorf("Error while deploying network: %v", err)
		_, saveErr := networkStatus.Update().SetFailed(true).SetState(status.StateFAILED).Save(ctx)
		if saveErr != nil {
			logger.Log.Errorf("Error while setting Provisioned Network status to FAILED: %v", saveErr)
			return saveErr
		}
		rdb.Publish(ctx, "updatedStatus", networkStatus.ID.String())
		checkTeamStatus(client, logger, ctx, entTeam)
		return err
	}
	logger.Log.Infof("deployed %s successfully", entProNetwork.Name)

	// _, saveErr = networkStatus.Update().SetCompleted(true).SetState(status.StateCOMPLETE).Save(ctx)
	// if saveErr != nil {
	// 	logger.Log.Errorf("Error while setting Provisioned Network status to COMPLETE: %v", saveErr)
	// 	return saveErr
	// }
	// rdb.Publish(ctx, "updatedStatus", networkStatus.ID.String())
	return nil
}

func buildTeam(client *ent.Client, logger *logging.Logger, builder *builder.Builder, ctx context.Context, entTeam *ent.Team) error {
	logger.Log.Infof("deploying Team: %d", entTeam.TeamNumber)

	teamStatus, err := entTeam.QueryStatus().Only(ctx)
	if err != nil {
		logger.Log.Errorf("Error while getting Team status: %v", err)
		return err
	}
	_, saveErr := teamStatus.Update().SetState(status.StateINPROGRESS).Save(ctx)
	if saveErr != nil {
		logger.Log.Errorf("Error while setting Team status to INPROGRESS: %v", saveErr)
		return saveErr
	}
	rdb.Publish(ctx, "updatedStatus", teamStatus.ID.String())
	err = (*builder).DeployTeam(ctx, entTeam)
	if err != nil {
		logger.Log.Errorf("Error while deploying network: %v", err)
		_, saveErr := teamStatus.Update().SetFailed(true).SetState(status.StateFAILED).Save(ctx)
		if saveErr != nil {
			logger.Log.Errorf("Error while setting Provisioned Network status to FAILED: %v", saveErr)
			return saveErr
		}
		rdb.Publish(ctx, "updatedStatus", teamStatus.ID.String())
		checkTeamStatus(client, logger, ctx, entTeam)
		return err
	}
	logger.Log.Infof("deployed %d successfully", entTeam.TeamNumber)
	return nil
}

func checkTeamStatus(client *ent.Client, logger *logging.Logger, ctx context.Context, entTeam *ent.Team) error {
	stepAwaitingInProgress, err := entTeam.
		QueryProvisionedNetworks().
		Where(
			provisionednetwork.
				HasStatusWith(
					status.Or(
						status.StateEQ(status.StateAWAITING),
						status.StateEQ(status.StateINPROGRESS),
					),
				),
		).Exist(ctx)
	if err != nil {
		logger.Log.Errorf("Error while checking if host step is in progress: %v", err)
		return err
	}
	if stepAwaitingInProgress {
		logger.Log.Debug("team %s is in progress", entTeam.ID)
		return nil
	}

	teamStatus, err := entTeam.QueryStatus().Only(ctx)
	if teamStatus.State != status.StateINPROGRESS {
		return nil
	}

	hostFailed, err := entTeam.
		QueryProvisionedNetworks().
		Where(
			provisionednetwork.
				HasStatusWith(
					status.Or(
						status.StateEQ(status.StateFAILED),
						status.StateEQ(status.StateTAINTED),
					),
				),
		).Exist(ctx)
	if err != nil {
		logger.Log.Errorf("Error while checking if host step is failed: %v", err)
		return err
	}
	if hostFailed {
		_, saveErr := teamStatus.Update().SetCompleted(false).SetFailed(true).SetState(status.StateTAINTED).Save(ctx)
		if saveErr != nil {
			logger.Log.Errorf("Error while setting Provisioned Network status to Tainted: %v", saveErr)
			return saveErr
		}
		rdb.Publish(ctx, "updatedStatus", teamStatus.ID.String())
		logger.Log.Debug("host %s is failed", teamStatus.ID)
		return nil
	}

	stepNotCompleted, err := entTeam.
		QueryProvisionedNetworks().
		Where(
			provisionednetwork.
				HasStatusWith(
					status.StateNEQ(status.StateCOMPLETE),
				),
		).Exist(ctx)
	if err != nil {
		logger.Log.Errorf("Error while checking if host step is failed: %v", err)
		return err
	}
	if !stepNotCompleted {
		_, saveErr := teamStatus.Update().SetCompleted(true).SetFailed(false).SetState(status.StateCOMPLETE).Save(ctx)
		if saveErr != nil {
			logger.Log.Errorf("Error while setting Provisioned Network status to Completed: %v", saveErr)
			return saveErr
		}
		rdb.Publish(ctx, "updatedStatus", teamStatus.ID.String())
		logger.Log.Debug("host %s is failed", teamStatus.ID)
		return nil
	}
	return nil
}

func checkNetworkStatus(client *ent.Client, logger *logging.Logger, ctx context.Context, entProNetwork *ent.ProvisionedNetwork) error {
	stepAwaitingInProgress, err := entProNetwork.
		QueryProvisionedHosts().
		Where(
			provisionedhost.
				HasStatusWith(
					status.Or(
						status.StateEQ(status.StateAWAITING),
						status.StateEQ(status.StateINPROGRESS),
					),
				),
		).Exist(ctx)
	if err != nil {
		logger.Log.Errorf("Error while checking if host step is in progress: %v", err)
		return err
	}
	if stepAwaitingInProgress {
		logger.Log.Debug("network %s is in progress", entProNetwork.ID)
		return nil
	}

	networkStatus, err := entProNetwork.QueryStatus().Only(ctx)
	if networkStatus.State != status.StateINPROGRESS {
		return nil
	}
	entTeam, err := entProNetwork.QueryTeam().Only(ctx)
	if err != nil {
		logger.Log.Errorf("Error while getting team: %v", err)
		return err
	}

	hostFailed, err := entProNetwork.
		QueryProvisionedHosts().
		Where(
			provisionedhost.
				HasStatusWith(
					status.Or(
						status.StateEQ(status.StateFAILED),
						status.StateEQ(status.StateTAINTED),
					),
				),
		).Exist(ctx)
	if err != nil {
		logger.Log.Errorf("Error while checking if host step is failed: %v", err)
		return err
	}
	if hostFailed {
		_, saveErr := networkStatus.Update().SetCompleted(false).SetFailed(true).SetState(status.StateTAINTED).Save(ctx)
		if saveErr != nil {
			logger.Log.Errorf("Error while setting Provisioned Network status to Tainted: %v", saveErr)
			return saveErr
		}
		rdb.Publish(ctx, "updatedStatus", networkStatus.ID.String())
		logger.Log.Debug("host %s is failed", networkStatus.ID)
		checkTeamStatus(client, logger, ctx, entTeam)
		return nil
	}

	stepNotCompleted, err := entProNetwork.
		QueryProvisionedHosts().
		Where(
			provisionedhost.
				HasStatusWith(
					status.StateNEQ(status.StateCOMPLETE),
				),
		).Exist(ctx)
	if err != nil {
		logger.Log.Errorf("Error while checking if host step is failed: %v", err)
		return err
	}
	if !stepNotCompleted {
		_, saveErr := networkStatus.Update().SetCompleted(true).SetFailed(false).SetState(status.StateCOMPLETE).Save(ctx)
		if saveErr != nil {
			logger.Log.Errorf("Error while setting Provisioned Network status to Completed: %v", saveErr)
			return saveErr
		}
		rdb.Publish(ctx, "updatedStatus", networkStatus.ID.String())
		logger.Log.Debug("host %s is Completed", networkStatus.ID)
		checkTeamStatus(client, logger, ctx, entTeam)
		return nil
	}
	return nil
}

func checkHostStatus(client *ent.Client, logger *logging.Logger, ctx context.Context, entProHost *ent.ProvisionedHost) error {
	hostStatus, err := entProHost.QueryStatus().Only(ctx)
	if hostStatus.State != status.StateINPROGRESS {
		return nil
	}
	entProNetwork, err := entProHost.QueryProvisionedNetwork().Only(ctx)
	if err != nil {
		logger.Log.Errorf("Error while checking if host step is failed: %v", err)
		return err
	}

	stepFailed, err := entProHost.
		QueryProvisioningSteps().
		Where(
			provisioningstep.
				HasStatusWith(
					status.StateEQ(status.StateFAILED),
				),
		).Exist(ctx)
	if err != nil {
		logger.Log.Errorf("Error while checking if host step is failed: %v", err)
		return err
	}
	if stepFailed {
		_, saveErr := hostStatus.Update().SetCompleted(false).SetFailed(true).SetState(status.StateTAINTED).Save(ctx)
		if saveErr != nil {
			logger.Log.Errorf("Error while setting Provisioned Host status to Tainted: %v", saveErr)
			return saveErr
		}
		rdb.Publish(ctx, "updatedStatus", hostStatus.ID.String())
		logger.Log.Debug("host %s is failed", entProHost.ID)
		checkNetworkStatus(client, logger, ctx, entProNetwork)
		return nil
	}

	stepNotCompleted, err := entProHost.
		QueryProvisioningSteps().
		Where(
			provisioningstep.
				HasStatusWith(
					status.StateNEQ(status.StateCOMPLETE),
				),
		).Exist(ctx)
	if err != nil {
		logger.Log.Errorf("Error while checking if host step is failed: %v", err)
		return err
	}
	if !stepNotCompleted {
		_, saveErr := hostStatus.Update().SetCompleted(true).SetFailed(false).SetState(status.StateCOMPLETE).Save(ctx)
		if saveErr != nil {
			logger.Log.Errorf("Error while setting Provisioned Host status to Completed: %v", saveErr)
			return saveErr
		}
		rdb.Publish(ctx, "updatedStatus", hostStatus.ID.String())
		logger.Log.Debug("host %s is completed", entProHost.ID)
		checkNetworkStatus(client, logger, ctx, entProNetwork)
		return nil
	}
	return nil
}

func execStep(client *ent.Client, laforgeConfig *utils.ServerConfig, logger *logging.Logger, ctx context.Context, entStep *ent.ProvisioningStep) error {
	stepStatus, err := entStep.QueryStatus().Only(ctx)
	if err != nil {
		logger.Log.Errorf("Failed to Query Provisioning Step Status. Err: %v", err)
		return err
	}
	_, err = stepStatus.Update().SetState(status.StateINPROGRESS).Save(ctx)
	if err != nil {
		logger.Log.Errorf("error while trying to set ent.ProvisioningStep.Status.State to status.StateCOMPLETED: %v", err)
		return err
	}
	rdb.Publish(ctx, "updatedStatus", stepStatus.ID.String())

	entProvisionedHost, err := entStep.QueryProvisionedHost().Only(ctx)
	if err != nil {
		logger.Log.Errorf("failed querying Provisioned Host for Provioning Step: %v", err)
		return err
	}

	taskCount, err := entProvisionedHost.QueryAgentTasks().Count(ctx)
	if err != nil {
		logger.Log.Errorf("failed querying Number of Tasks: %v", err)
		return err
	}

	switch entStep.Type {
	case provisioningstep.TypeScript:
		entScript, err := entStep.QueryScript().Only(ctx)
		if err != nil {
			logger.Log.Errorf("failed querying Script for Provioning Step: %v", err)
			return err
		}
		if _, ok := entScript.Vars["build_render"]; ok {
			_, err := RenderScript(ctx, client, logger, entStep)
			if err != nil {
				logger.Log.Errorf("failed rerendering Script: %v", err)
				return err
			}
			logger.Log.Debug("sucessful rerendering for Script: %v", err)
		}
		entGinMiddleware, err := entStep.QueryGinFileMiddleware().Only(ctx)
		if err != nil {
			logger.Log.Errorf("failed querying Gin File Middleware for Script: %v", err)
			return err
		}
		_, err = client.AgentTask.Create().
			SetCommand(agenttask.CommandDOWNLOAD).
			SetArgs(entScript.Source + "💔" + laforgeConfig.Agent.ApiDownloadUrl + entGinMiddleware.URLID + "💔" + "true").
			SetNumber(taskCount).
			SetState(agenttask.StateAWAITING).
			SetProvisionedHost(entProvisionedHost).
			SetProvisioningStep(entStep).
			Save(ctx)
		if err != nil {
			logger.Log.Errorf("failed Creating Agent Task for Script Download: %v", err)
			return err
		}
		// TODO: Add the Ability to change permissions of a file into the agent
		_, err = client.AgentTask.Create().
			SetCommand(agenttask.CommandEXECUTE).
			SetArgs(entScript.Source + "💔" + strings.Join(entScript.Args, " ")).
			SetNumber(taskCount + 1).
			SetState(agenttask.StateAWAITING).
			SetProvisionedHost(entProvisionedHost).
			SetProvisioningStep(entStep).
			Save(ctx)
		if err != nil {
			logger.Log.Errorf("failed Creating Agent Task for Script Execute: %v", err)
			return err
		}
		_, err = client.AgentTask.Create().
			SetCommand(agenttask.CommandDELETE).
			SetArgs(entScript.Source).
			SetNumber(taskCount + 2).
			SetState(agenttask.StateAWAITING).
			SetProvisionedHost(entProvisionedHost).
			SetProvisioningStep(entStep).
			Save(ctx)
		if err != nil {
			logger.Log.Errorf("failed Creating Agent Task for Script Delete: %v", err)
			return err
		}
		for i, validationHCLID := range entScript.Validations {
			err = createValidation(client, logger, ctx, validationHCLID, taskCount+3+i, entProvisionedHost, entStep)
			if err != nil {
				logger.Log.Errorf("failed Creating Agent Task for Script Validation: %v", err)
			}

		}
	case provisioningstep.TypeCommand:
		entCommand, err := entStep.QueryCommand().Only(ctx)
		if err != nil {
			logger.Log.Errorf("failed querying Command for Provioning Step: %v", err)
			return err
		}
		// Check if reboot command
		if entCommand.Program == "REBOOT" {
			_, err = client.AgentTask.Create().
				SetCommand(agenttask.CommandREBOOT).
				SetArgs("").
				SetNumber(taskCount).
				SetState(agenttask.StateAWAITING).
				SetProvisionedHost(entProvisionedHost).
				SetProvisioningStep(entStep).
				Save(ctx)
			if err != nil {
				logger.Log.Errorf("failed Creating Agent Task for Reboot Command: %v", err)
				return err
			}
		} else {
			_, err = client.AgentTask.Create().
				SetCommand(agenttask.CommandEXECUTE).
				SetArgs(entCommand.Program + "💔" + strings.Join(entCommand.Args, " ")).
				SetNumber(taskCount).
				SetState(agenttask.StateAWAITING).
				SetProvisionedHost(entProvisionedHost).
				SetProvisioningStep(entStep).
				Save(ctx)
			if err != nil {
				logger.Log.Errorf("failed Creating Agent Task for Command: %v", err)
				return err
			}
		}
		for i, validationHCLID := range entCommand.Validations {
			err = createValidation(client, logger, ctx, validationHCLID, taskCount+3+i, entProvisionedHost, entStep)
			if err != nil {
				logger.Log.Errorf("failed Creating Agent Task for Command Validation: %v", err)
			}

		}
	case provisioningstep.TypeFileDelete:
		entFileDelete, err := entStep.QueryFileDelete().Only(ctx)
		if err != nil {
			logger.Log.Errorf("failed querying File Delete for Provioning Step: %v", err)
			return err
		}
		_, err = client.AgentTask.Create().
			SetCommand(agenttask.CommandDELETE).
			SetArgs(entFileDelete.Path).
			SetNumber(taskCount).
			SetState(agenttask.StateAWAITING).
			SetProvisionedHost(entProvisionedHost).
			SetProvisioningStep(entStep).
			Save(ctx)
		if err != nil {
			logger.Log.Errorf("failed Creating Agent Task for File Delete: %v", err)
			return err
		}
		for i, validationHCLID := range entFileDelete.Validations {
			err = createValidation(client, logger, ctx, validationHCLID, taskCount+3+i, entProvisionedHost, entStep)
			if err != nil {
				logger.Log.Errorf("failed Creating Agent Task for FileDelete Validation: %v", err)
			}

		}
	case provisioningstep.TypeFileDownload:
		entFileDownload, err := entStep.QueryFileDownload().Only(ctx)
		if err != nil {
			logger.Log.Errorf("failed querying File Download for Provioning Step: %v", err)
			return err
		}
		entGinMiddleware, err := entStep.QueryGinFileMiddleware().Only(ctx)
		if err != nil {
			logger.Log.Errorf("failed querying Gin File Middleware for File Download: %v", err)
			return err
		}
		if entFileDownload.SourceType == "remote" {
			_, err = client.AgentTask.Create().
				SetCommand(agenttask.CommandDOWNLOAD).
				SetArgs(entFileDownload.Destination + "💔" + entFileDownload.Source + "💔" + strings.ToLower(fmt.Sprintf("%v", entFileDownload.IsTxt))).
				SetNumber(taskCount).
				SetState(agenttask.StateAWAITING).
				SetProvisionedHost(entProvisionedHost).
				SetProvisioningStep(entStep).
				Save(ctx)
		} else {
			_, err = client.AgentTask.Create().
				SetCommand(agenttask.CommandDOWNLOAD).
				SetArgs(entFileDownload.Destination + "💔" + laforgeConfig.Agent.ApiDownloadUrl + entGinMiddleware.URLID + "💔" + strings.ToLower(fmt.Sprintf("%v", entFileDownload.IsTxt))).
				SetNumber(taskCount).
				SetState(agenttask.StateAWAITING).
				SetProvisionedHost(entProvisionedHost).
				SetProvisioningStep(entStep).
				Save(ctx)
		}
		if err != nil {
			logger.Log.Errorf("failed Creating Agent Task for File Download: %v", err)
			return err
		}
		for i, validationHCLID := range entFileDownload.Validations {
			err = createValidation(client, logger, ctx, validationHCLID, taskCount+3+i, entProvisionedHost, entStep)
			if err != nil {
				logger.Log.Errorf("failed Creating Agent Task for FileDownloads Validation: %v", err)
			}

		}
	case provisioningstep.TypeFileExtract:
		entFileExtract, err := entStep.QueryFileExtract().Only(ctx)
		if err != nil {
			logger.Log.Errorf("failed querying File Extract for Provioning Step: %v", err)
			return err
		}
		_, err = client.AgentTask.Create().
			SetCommand(agenttask.CommandEXTRACT).
			SetArgs(entFileExtract.Source + "💔" + entFileExtract.Destination).
			SetNumber(taskCount).
			SetState(agenttask.StateAWAITING).
			SetProvisionedHost(entProvisionedHost).
			SetProvisioningStep(entStep).
			Save(ctx)
		if err != nil {
			logger.Log.Errorf("failed Creating Agent Task for File Extract: %v", err)
			return err
		}
		for i, validationHCLID := range entFileExtract.Validations {
			err = createValidation(client, logger, ctx, validationHCLID, taskCount+3+i, entProvisionedHost, entStep)
			if err != nil {
				logger.Log.Errorf("failed Creating Agent Task for FileExtract Validation: %v", err)
			}

		}
	case provisioningstep.TypeDNSRecord:
		break
	case provisioningstep.TypeAnsible:
		entAnsible, err := entStep.QueryAnsible().Only(ctx)
		if err != nil {
			logger.Log.Errorf("failed querying Ansible for Provioning Step: %v", err)
			return err
		}
		entGinMiddleware, err := entStep.QueryGinFileMiddleware().Only(ctx)
		if err != nil {
			logger.Log.Errorf("failed querying Gin File Middleware for Script: %v", err)
			return err
		}
		_, err = client.AgentTask.Create().
			SetCommand(agenttask.CommandDOWNLOAD).
			SetArgs("/tmp/" + entAnsible.Name + ".zip" + "💔" + laforgeConfig.Agent.ApiDownloadUrl + entGinMiddleware.URLID + "💔" + "false").
			SetNumber(taskCount).
			SetState(agenttask.StateAWAITING).
			SetProvisionedHost(entProvisionedHost).
			SetProvisioningStep(entStep).
			Save(ctx)
		if err != nil {
			logger.Log.Errorf("failed Creating Agent Task for Script Download: %v", err)
			return err
		}
		_, err = client.AgentTask.Create().
			SetCommand(agenttask.CommandEXTRACT).
			SetArgs("/tmp/" + entAnsible.Name + ".zip" + "💔" + "/tmp").
			SetNumber(taskCount + 1).
			SetState(agenttask.StateAWAITING).
			SetProvisionedHost(entProvisionedHost).
			SetProvisioningStep(entStep).
			Save(ctx)
		if err != nil {
			logger.Log.Errorf("failed Creating Agent Task for Script Download: %v", err)
			return err
		}
		_, err = client.AgentTask.Create().
			SetCommand(agenttask.CommandANSIBLE).
			SetArgs("/tmp/" + entAnsible.Name + "/" + entAnsible.PlaybookName + "💔" + string(entAnsible.Method) + "💔" + entAnsible.Inventory).
			SetNumber(taskCount + 2).
			SetState(agenttask.StateAWAITING).
			SetProvisionedHost(entProvisionedHost).
			SetProvisioningStep(entStep).
			Save(ctx)
		if err != nil {
			logger.Log.Errorf("failed Creating Agent Task for Script Execute: %v", err)
			return err
		}
		_, err = client.AgentTask.Create().
			SetCommand(agenttask.CommandDELETE).
			SetArgs("/tmp/" + entAnsible.Name).
			SetNumber(taskCount + 3).
			SetState(agenttask.StateAWAITING).
			SetProvisionedHost(entProvisionedHost).
			SetProvisioningStep(entStep).
			Save(ctx)
		if err != nil {
			logger.Log.Errorf("failed Creating Agent Task for Script Delete: %v", err)
			return err
		}
		_, err = client.AgentTask.Create().
			SetCommand(agenttask.CommandDELETE).
			SetArgs("/tmp/" + entAnsible.Name + ".zip").
			SetNumber(taskCount + 4).
			SetState(agenttask.StateAWAITING).
			SetProvisionedHost(entProvisionedHost).
			SetProvisioningStep(entStep).
			Save(ctx)
		if err != nil {
			logger.Log.Errorf("failed Creating Agent Task for Script Delete: %v", err)
			return err
		}
		for i, validationHCLID := range entAnsible.Validations {
			err = createValidation(client, logger, ctx, validationHCLID, taskCount+3+i, entProvisionedHost, entStep)
			if err != nil {
				logger.Log.Errorf("failed Creating Agent Task for Ansible Validation: %v", err)
			}

		}
	default:
		break
	}

	for {
		taskFailed, err := entStep.QueryAgentTasks().Where(
			agenttask.StateEQ(
				agenttask.StateFAILED,
			),
		).Exist(ctx)

		if err != nil {
			logger.Log.Errorf("Failed to Query Agent Task State. Err: %v", err)
			return err
		}

		if taskFailed {
			_, err = stepStatus.Update().SetFailed(true).SetState(status.StateFAILED).Save(ctx)
			if err != nil {
				logger.Log.Errorf("error while trying to set ent.ProvisioningStep.Status.State to status.StateFAILED: %v", err)
				return err
			}
			checkHostStatus(client, logger, ctx, entProvisionedHost)
			rdb.Publish(ctx, "updatedStatus", stepStatus.ID.String())
			return fmt.Errorf("one or more agent tasks failed")
		}

		taskRunning, err := entStep.QueryAgentTasks().Where(
			agenttask.StateNEQ(
				agenttask.StateCOMPLETE,
			),
		).Exist(ctx)

		if err != nil {
			logger.Log.Errorf("Failed to Query Agent Task State. Err: %v", err)
			return err
		}

		if !taskRunning {
			break
		}

		time.Sleep(time.Second)
	}
	_, err = stepStatus.Update().SetCompleted(true).SetState(status.StateCOMPLETE).Save(ctx)

	if err != nil {
		logger.Log.Errorf("error while trying to set ent.ProvisioningStep.Status.State to status.StateCOMPLETED: %v", err)
		return err
	}
	checkHostStatus(client, logger, ctx, entProvisionedHost)
	rdb.Publish(ctx, "updatedStatus", stepStatus.ID.String())

	return nil

}

func createValidation(client *ent.Client, logger *logging.Logger, ctx context.Context, validationHCLID string, taskCount int, entProvisionedHost *ent.ProvisionedHost, entStep *ent.ProvisioningStep) error {
	entValidation, err := client.Validation.Query().
		Where(
			validation.HclIDEQ(
				validationHCLID,
			),
		).
		Only(ctx)
	if err != nil {
		logger.Log.Errorf("failed Query Validation for Script: %v", err)
		return err
	}

	switch entValidation.ValidationType {
	case "linux-apt-installed":
		_, err = client.AgentTask.Create().
			SetCommand(agenttask.CommandVALIDATOR).
			SetArgs("linux-apt-installed" + "💔" + entValidation.PackageName).
			SetNumber(taskCount).
			SetState(agenttask.StateAWAITING).
			SetProvisionedHost(entProvisionedHost).
			SetProvisioningStep(entStep).
			SetValidation(entValidation).
			Save(ctx)
	case "net-tcp-open":
		_, err = client.AgentTask.Create().
			SetCommand(agenttask.CommandVALIDATOR).
			SetArgs("net-tcp-open" + "💔" + entValidation.IP + "💔" + strconv.Itoa(entValidation.Port)).
			SetNumber(taskCount).
			SetState(agenttask.StateAWAITING).
			SetProvisionedHost(entProvisionedHost).
			SetProvisioningStep(entStep).
			SetValidation(entValidation).
			Save(ctx)
	case "net-udp-open":
		_, err = client.AgentTask.Create().
			SetCommand(agenttask.CommandVALIDATOR).
			SetArgs("net-udp-open" + "💔" + entValidation.IP + "💔" + strconv.Itoa(entValidation.Port)).
			SetNumber(taskCount).
			SetState(agenttask.StateAWAITING).
			SetProvisionedHost(entProvisionedHost).
			SetProvisioningStep(entStep).
			SetValidation(entValidation).
			Save(ctx)
	case "net-http-content-regex":
		_, err = client.AgentTask.Create().
			SetCommand(agenttask.CommandVALIDATOR).
			SetArgs("net-http-content-regex" + "💔" + entValidation.IP).
			SetNumber(taskCount).
			SetState(agenttask.StateAWAITING).
			SetProvisionedHost(entProvisionedHost).
			SetProvisioningStep(entStep).
			SetValidation(entValidation).
			Save(ctx)
	case "file-exists":
		_, err = client.AgentTask.Create().
			SetCommand(agenttask.CommandVALIDATOR).
			SetArgs("file-exists" + "💔" + entValidation.FilePath).
			SetNumber(taskCount).
			SetState(agenttask.StateAWAITING).
			SetProvisionedHost(entProvisionedHost).
			SetProvisioningStep(entStep).
			SetValidation(entValidation).
			Save(ctx)
	case "file-hash":
		_, err = client.AgentTask.Create().
			SetCommand(agenttask.CommandVALIDATOR).
			SetArgs("file-hash" + "💔" + entValidation.FilePath).
			SetNumber(taskCount).
			SetState(agenttask.StateAWAITING).
			SetProvisionedHost(entProvisionedHost).
			SetProvisioningStep(entStep).
			SetValidation(entValidation).
			Save(ctx)
	case "file-content-regex":
		_, err = client.AgentTask.Create().
			SetCommand(agenttask.CommandVALIDATOR).
			SetArgs("file-content-regex" + "💔" + entValidation.FilePath + "💔" + entValidation.Regex).
			SetNumber(taskCount).
			SetState(agenttask.StateAWAITING).
			SetProvisionedHost(entProvisionedHost).
			SetProvisioningStep(entStep).
			SetValidation(entValidation).
			Save(ctx)
	case "dir-exists":
		_, err = client.AgentTask.Create().
			SetCommand(agenttask.CommandVALIDATOR).
			SetArgs("dir-exists" + "💔" + entValidation.FilePath).
			SetNumber(taskCount).
			SetState(agenttask.StateAWAITING).
			SetProvisionedHost(entProvisionedHost).
			SetProvisioningStep(entStep).
			SetValidation(entValidation).
			Save(ctx)
	case "user-exists":
		_, err = client.AgentTask.Create().
			SetCommand(agenttask.CommandVALIDATOR).
			SetArgs("user-exists" + "💔" + entValidation.Username).
			SetNumber(taskCount).
			SetState(agenttask.StateAWAITING).
			SetProvisionedHost(entProvisionedHost).
			SetProvisioningStep(entStep).
			SetValidation(entValidation).
			Save(ctx)
	case "user-group-membership":
		_, err = client.AgentTask.Create().
			SetCommand(agenttask.CommandVALIDATOR).
			SetArgs("user-group-membership" + "💔" + entValidation.Username + "💔" + entValidation.GroupName).
			SetNumber(taskCount).
			SetState(agenttask.StateAWAITING).
			SetProvisionedHost(entProvisionedHost).
			SetProvisioningStep(entStep).
			SetValidation(entValidation).
			Save(ctx)
	case "host-port-open":
		_, err = client.AgentTask.Create().
			SetCommand(agenttask.CommandVALIDATOR).
			SetArgs("host-port-open" + "💔" + strconv.Itoa(entValidation.Port)).
			SetNumber(taskCount).
			SetState(agenttask.StateAWAITING).
			SetProvisionedHost(entProvisionedHost).
			SetProvisioningStep(entStep).
			SetValidation(entValidation).
			Save(ctx)
	case "host-process-running":
		_, err = client.AgentTask.Create().
			SetCommand(agenttask.CommandVALIDATOR).
			SetArgs("host-process-running" + "💔" + entValidation.ProcessName).
			SetNumber(taskCount).
			SetState(agenttask.StateAWAITING).
			SetProvisionedHost(entProvisionedHost).
			SetProvisioningStep(entStep).
			SetValidation(entValidation).
			Save(ctx)
	case "host-service-state":
		_, err = client.AgentTask.Create().
			SetCommand(agenttask.CommandVALIDATOR).
			SetArgs("host-service-state" + "💔" + entValidation.ServiceName + "💔" + entValidation.ServiceStatus.String()).
			SetNumber(taskCount).
			SetState(agenttask.StateAWAITING).
			SetProvisionedHost(entProvisionedHost).
			SetProvisioningStep(entStep).
			SetValidation(entValidation).
			Save(ctx)
	case "net-icmp":
		_, err = client.AgentTask.Create().
			SetCommand(agenttask.CommandVALIDATOR).
			SetArgs("net-icmp" + "💔" + entValidation.IP).
			SetNumber(taskCount).
			SetState(agenttask.StateAWAITING).
			SetProvisionedHost(entProvisionedHost).
			SetProvisioningStep(entStep).
			SetValidation(entValidation).
			Save(ctx)
	case "net-http-content-hash":
		_, err = client.AgentTask.Create().
			SetCommand(agenttask.CommandVALIDATOR).
			SetArgs("net-http-content-hash" + "💔" + entValidation.IP).
			SetNumber(taskCount).
			SetState(agenttask.StateAWAITING).
			SetProvisionedHost(entProvisionedHost).
			SetProvisioningStep(entStep).
			SetValidation(entValidation).
			Save(ctx)
	case "file-content-string":
		_, err = client.AgentTask.Create().
			SetCommand(agenttask.CommandVALIDATOR).
			SetArgs("file-content-string" + "💔" + entValidation.FilePath + "💔" + entValidation.SearchString).
			SetNumber(taskCount).
			SetState(agenttask.StateAWAITING).
			SetProvisionedHost(entProvisionedHost).
			SetProvisioningStep(entStep).
			SetValidation(entValidation).
			Save(ctx)
	case "file-permission":
		_, err = client.AgentTask.Create().
			SetCommand(agenttask.CommandVALIDATOR).
			SetArgs("file-permission" + "💔" + entValidation.FilePath).
			SetNumber(taskCount).
			SetState(agenttask.StateAWAITING).
			SetProvisionedHost(entProvisionedHost).
			SetProvisioningStep(entStep).
			SetValidation(entValidation).
			Save(ctx)
	}
	if err != nil {
		logger.Log.Errorf("Agent task failed with error: %v", err)
		return err
	}
	return nil
}

func startScheduledStep(client *ent.Client, laforgeConfig *utils.ServerConfig, logger *logging.Logger, ctx context.Context, entProvisioningScheduledStep *ent.ProvisioningScheduledStep) error {
	entScheduledStep, err := entProvisioningScheduledStep.QueryScheduledStep().Only(ctx)
	if err != nil {
		return fmt.Errorf("failed to query scheduled step from provisioning scheduled step: %v", err)
	}
	entStatus, err := entProvisioningScheduledStep.QueryStatus().Only(ctx)
	if err != nil {
		return fmt.Errorf("failed to query status from provisioning scheduled step: %v", err)
	}
	if entScheduledStep.Type == scheduledstep.TypeRUNONCE {
		// We can ignore RUNONCE steps as their run_at time is already accurate
		//   and will be handled by the scheduling watchdog
		return nil
	}
	if entScheduledStep.Type == scheduledstep.TypeCRON && entProvisioningScheduledStep.RunTime.Unix() > 0 {
		// These steps were already scheduled during the planning phase and we
		//   can ignore at build time
		return nil
	}
	scheduleExpr, err := cronexpr.Parse(entScheduledStep.Schedule)
	if err != nil {
		return fmt.Errorf("failed to parse scheduled step schedule: %v", err)
	}
	// determine the next time this step should run after this
	runTime := scheduleExpr.Next(time.Now())

	// set the proper run_time, watchdog will handle scheduling the further ones
	err = entProvisioningScheduledStep.Update().SetRunTime(runTime).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update run_time of provisioning scheduled step: %v", err)
	}

	err = entStatus.Update().SetState(status.StateAWAITING).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to set provisioning scheduled step status to AWAITING: %v", err)
	}

	return nil
}
