package scheduler

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/gen0cide/laforge/ent"
	"github.com/gen0cide/laforge/ent/agenttask"
	"github.com/gen0cide/laforge/ent/provisioningscheduledstep"
	"github.com/gen0cide/laforge/ent/status"
	"github.com/gen0cide/laforge/logging"
	"github.com/gen0cide/laforge/planner"
	"github.com/gen0cide/laforge/server/utils"
	"github.com/go-redis/redis/v8"
	"github.com/sirupsen/logrus"
)

func SchedulerWatchdog(ctx context.Context, client *ent.Client, rdb *redis.Client, laforgeConfig *utils.ServerConfig) {
	schedulerLogger, err := InitSchedulerLogger(laforgeConfig)
	if err != nil {
		logrus.Errorf("failed to initialize scheduler watchdog logger: %v", err)
		return
	}
	// A loop which queries for all provisioning scheduled steps which:
	//   1) haven't been run (status is AWAITING)
	//   2) don't have a run_time of 0 (these are waiting on other things)
	//   3) have a run_time in the past
	ticker := time.NewTicker(time.Second)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			provisioningScheduledStepsToExecute, err := GetStepsToExecute(ctx, client)
			if err != nil {
				schedulerLogger.Log.Errorf("failed to query provisioning scheduled steps to execute: %v", err)
				continue
			}
			for _, entProvisioningScheduledStep := range provisioningScheduledStepsToExecute {
				go ExecuteScheduledStep(ctx, client, rdb, schedulerLogger, laforgeConfig, entProvisioningScheduledStep)
			}
		}
	}
}

func InitSchedulerLogger(laforgeConfig *utils.ServerConfig) (*logging.Logger, error) {
	logFolder := laforgeConfig.LogFolder
	if logFolder == "" {
		// Default log location
		logFolder = "/var/log/laforge"
	}
	absPath, err := filepath.Abs(logFolder)
	if err != nil {
		return nil, fmt.Errorf("error getting absolute path from log folder: %v", err)
	}
	err = os.MkdirAll(absPath, 0755)
	if err != nil {
		return nil, fmt.Errorf("error creating log folder: %v", err)
	}
	filename := fmt.Sprintf("%s_%s.lfglog", time.Now().Format("20060102-15-04-05"), "InternalScheduler")
	logPath := path.Join(absPath, filename)
	logrus.Info(logPath)
	schedulerLogger := logging.CreateNewLogger(logPath)
	return &schedulerLogger, nil
}

func GetStepsToExecute(ctx context.Context, client *ent.Client) ([]*ent.ProvisioningScheduledStep, error) {
	entProvisioningScheduledSteps, err := client.ProvisioningScheduledStep.Query().Where(
		provisioningscheduledstep.And(
			provisioningscheduledstep.HasStatusWith(
				status.StateEQ(status.StateAWAITING), // Is of status AWAITING (has been queued by the builder)
			),
			provisioningscheduledstep.RunTimeNEQ(time.Unix(0, 0)), // Has a non-zero run time
			provisioningscheduledstep.RunTimeLTE(time.Now()),      // Should be run now or in the past
		),
	).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query provisioning scheduled steps: %v", err)
	}
	return entProvisioningScheduledSteps, nil
}

func ExecuteScheduledStep(ctx context.Context, client *ent.Client, rdb *redis.Client, logger *logging.Logger, laforgeConfig *utils.ServerConfig, entProvisioningScheduledStep *ent.ProvisioningScheduledStep) {
	entStatus, err := entProvisioningScheduledStep.Status(ctx)
	if err != nil {
		logger.Log.Errorf("failed to query provisioned scheduled step status: %v", err)
		return
	}
	err = entStatus.Update().SetState(status.StateINPROGRESS).Exec(ctx)
	if err != nil {
		logger.Log.Errorf("failed to update provisioned scheduled step status: %v", err)
		return
	}
	rdb.Publish(ctx, "updatedStatus", entStatus.ID.String())

	entProvisionedHost, err := entProvisioningScheduledStep.ProvisionedHost(ctx)
	if err != nil {
		logger.Log.Errorf("failed to query provisioned host: %v", err)
		return
	}

	taskCount, err := entProvisionedHost.QueryProvisionedHostToAgentTask().Count(ctx)
	if err != nil {
		logger.Log.Errorf("failed querying umber of tasks: %v", err)
		return
	}

	switch entProvisioningScheduledStep.Type {
	case provisioningscheduledstep.TypeScript:
		entScript, err := entProvisioningScheduledStep.Script(ctx)
		if err != nil {
			logger.Log.Errorf("failed querying Script for provioning scheduled step: %v", err)
			return
		}
		if _, ok := entScript.Vars["build_render"]; ok {
			_, err := planner.RenderScript(ctx, client, logger, entProvisioningScheduledStep)
			if err != nil {
				logger.Log.Errorf("failed rerendering Script: %v", err)
				return
			}
			logger.Log.Debug("successful rerendering for Script: %v", err)
		}
		entGinMiddleware, err := entProvisioningScheduledStep.GinFileMiddleware(ctx)
		if err != nil {
			logger.Log.Errorf("failed querying Gin File Middleware for Script: %v", err)
			return
		}
		_, err = client.AgentTask.Create().
			SetCommand(agenttask.CommandDOWNLOAD).
			SetArgs(entScript.Source + "💔" + laforgeConfig.Agent.ApiDownloadUrl + entGinMiddleware.URLID + "💔" + "true").
			SetNumber(taskCount).
			SetState(agenttask.StateAWAITING).
			SetAgentTaskToProvisionedHost(entProvisionedHost).
			SetAgentTaskToProvisioningScheduledStep(entProvisioningScheduledStep).
			Save(ctx)
		if err != nil {
			logger.Log.Errorf("failed Creating Agent Task for Script Download: %v", err)
			return
		}
		// TODO: Add the Ability to change permissions of a file into the agent
		_, err = client.AgentTask.Create().
			SetCommand(agenttask.CommandEXECUTE).
			SetArgs(entScript.Source + "💔" + strings.Join(entScript.Args, " ")).
			SetNumber(taskCount + 1).
			SetState(agenttask.StateAWAITING).
			SetAgentTaskToProvisionedHost(entProvisionedHost).
			SetAgentTaskToProvisioningScheduledStep(entProvisioningScheduledStep).
			Save(ctx)
		if err != nil {
			logger.Log.Errorf("failed Creating Agent Task for Script Execute: %v", err)
			return
		}
		_, err = client.AgentTask.Create().
			SetCommand(agenttask.CommandDELETE).
			SetArgs(entScript.Source).
			SetNumber(taskCount + 2).
			SetState(agenttask.StateAWAITING).
			SetAgentTaskToProvisionedHost(entProvisionedHost).
			SetAgentTaskToProvisioningScheduledStep(entProvisioningScheduledStep).
			Save(ctx)
		if err != nil {
			logger.Log.Errorf("failed Creating Agent Task for Script Delete: %v", err)
			return
		}
	case provisioningscheduledstep.TypeCommand:
		entCommand, err := entProvisioningScheduledStep.Command(ctx)
		if err != nil {
			logger.Log.Errorf("failed querying command for provioning scheduled step: %v", err)
			return
		}
		// Check if reboot command
		if entCommand.Program == "REBOOT" {
			_, err = client.AgentTask.Create().
				SetCommand(agenttask.CommandREBOOT).
				SetArgs("").
				SetNumber(taskCount).
				SetState(agenttask.StateAWAITING).
				SetAgentTaskToProvisionedHost(entProvisionedHost).
				SetAgentTaskToProvisioningScheduledStep(entProvisioningScheduledStep).
				Save(ctx)
			if err != nil {
				logger.Log.Errorf("failed Creating Agent Task for Reboot Command: %v", err)
				return
			}
		} else {
			_, err = client.AgentTask.Create().
				SetCommand(agenttask.CommandEXECUTE).
				SetArgs(entCommand.Program + "💔" + strings.Join(entCommand.Args, " ")).
				SetNumber(taskCount).
				SetState(agenttask.StateAWAITING).
				SetAgentTaskToProvisionedHost(entProvisionedHost).
				SetAgentTaskToProvisioningScheduledStep(entProvisioningScheduledStep).
				Save(ctx)
			if err != nil {
				logger.Log.Errorf("failed Creating Agent Task for Command: %v", err)
				return
			}
		}
	case provisioningscheduledstep.TypeFileDelete:
		entFileDelete, err := entProvisioningScheduledStep.FileDelete(ctx)
		if err != nil {
			logger.Log.Errorf("failed querying File Delete for provioning scheduled step: %v", err)
			return
		}
		_, err = client.AgentTask.Create().
			SetCommand(agenttask.CommandDELETE).
			SetArgs(entFileDelete.Path).
			SetNumber(taskCount).
			SetState(agenttask.StateAWAITING).
			SetAgentTaskToProvisionedHost(entProvisionedHost).
			SetAgentTaskToProvisioningScheduledStep(entProvisioningScheduledStep).
			Save(ctx)
		if err != nil {
			logger.Log.Errorf("failed Creating Agent Task for File Delete: %v", err)
			return
		}
	case provisioningscheduledstep.TypeFileDownload:
		entFileDownload, err := entProvisioningScheduledStep.FileDownload(ctx)
		if err != nil {
			logger.Log.Errorf("failed querying File Download for Provioning scheduled Step: %v", err)
			return
		}
		entGinMiddleware, err := entProvisioningScheduledStep.GinFileMiddleware(ctx)
		if err != nil {
			logger.Log.Errorf("failed querying Gin File Middleware for File Download: %v", err)
			return
		}
		if entFileDownload.SourceType == "remote" {
			_, err = client.AgentTask.Create().
				SetCommand(agenttask.CommandDOWNLOAD).
				SetArgs(entFileDownload.Destination + "💔" + entFileDownload.Source + "💔" + strings.ToLower(fmt.Sprintf("%v", entFileDownload.IsTxt))).
				SetNumber(taskCount).
				SetState(agenttask.StateAWAITING).
				SetAgentTaskToProvisionedHost(entProvisionedHost).
				SetAgentTaskToProvisioningScheduledStep(entProvisioningScheduledStep).
				Save(ctx)
		} else {
			_, err = client.AgentTask.Create().
				SetCommand(agenttask.CommandDOWNLOAD).
				SetArgs(entFileDownload.Destination + "💔" + laforgeConfig.Agent.ApiDownloadUrl + entGinMiddleware.URLID + "💔" + strings.ToLower(fmt.Sprintf("%v", entFileDownload.IsTxt))).
				SetNumber(taskCount).
				SetState(agenttask.StateAWAITING).
				SetAgentTaskToProvisionedHost(entProvisionedHost).
				SetAgentTaskToProvisioningScheduledStep(entProvisioningScheduledStep).
				Save(ctx)
		}
		if err != nil {
			logger.Log.Errorf("failed Creating Agent Task for File Download: %v", err)
			return
		}
	case provisioningscheduledstep.TypeFileExtract:
		entFileExtract, err := entProvisioningScheduledStep.FileExtract(ctx)
		if err != nil {
			logger.Log.Errorf("failed querying File Extract for Provioning scheduled Step: %v", err)
			return
		}
		_, err = client.AgentTask.Create().
			SetCommand(agenttask.CommandEXTRACT).
			SetArgs(entFileExtract.Source + "💔" + entFileExtract.Destination).
			SetNumber(taskCount).
			SetState(agenttask.StateAWAITING).
			SetAgentTaskToProvisionedHost(entProvisionedHost).
			SetAgentTaskToProvisioningScheduledStep(entProvisioningScheduledStep).
			Save(ctx)
		if err != nil {
			logger.Log.Errorf("failed Creating Agent Task for File Extract: %v", err)
			return
		}
	case provisioningscheduledstep.TypeDNSRecord:
		break
	case provisioningscheduledstep.TypeAnsible:
		entAnsible, err := entProvisioningScheduledStep.Ansible(ctx)
		if err != nil {
			logger.Log.Errorf("failed querying Ansible for Provioning scheduled Step: %v", err)
			return
		}
		entGinMiddleware, err := entProvisioningScheduledStep.GinFileMiddleware(ctx)
		if err != nil {
			logger.Log.Errorf("failed querying Gin File Middleware for Script: %v", err)
			return
		}
		_, err = client.AgentTask.Create().
			SetCommand(agenttask.CommandDOWNLOAD).
			SetArgs("/tmp/" + entAnsible.Name + ".zip" + "💔" + laforgeConfig.Agent.ApiDownloadUrl + entGinMiddleware.URLID + "💔" + "false").
			SetNumber(taskCount).
			SetState(agenttask.StateAWAITING).
			SetAgentTaskToProvisionedHost(entProvisionedHost).
			SetAgentTaskToProvisioningScheduledStep(entProvisioningScheduledStep).
			Save(ctx)
		if err != nil {
			logger.Log.Errorf("failed Creating Agent Task for Script Download: %v", err)
			return
		}
		_, err = client.AgentTask.Create().
			SetCommand(agenttask.CommandEXTRACT).
			SetArgs("/tmp/" + entAnsible.Name + ".zip" + "💔" + "/tmp").
			SetNumber(taskCount + 1).
			SetState(agenttask.StateAWAITING).
			SetAgentTaskToProvisionedHost(entProvisionedHost).
			SetAgentTaskToProvisioningScheduledStep(entProvisioningScheduledStep).
			Save(ctx)
		if err != nil {
			logger.Log.Errorf("failed Creating Agent Task for Script Download: %v", err)
			return
		}
		_, err = client.AgentTask.Create().
			SetCommand(agenttask.CommandANSIBLE).
			SetArgs("/tmp/" + entAnsible.Name + "/" + entAnsible.PlaybookName + "💔" + string(entAnsible.Method) + "💔" + entAnsible.Inventory).
			SetNumber(taskCount + 2).
			SetState(agenttask.StateAWAITING).
			SetAgentTaskToProvisionedHost(entProvisionedHost).
			SetAgentTaskToProvisioningScheduledStep(entProvisioningScheduledStep).
			Save(ctx)
		if err != nil {
			logger.Log.Errorf("failed Creating Agent Task for Script Execute: %v", err)
			return
		}
		_, err = client.AgentTask.Create().
			SetCommand(agenttask.CommandDELETE).
			SetArgs("/tmp/" + entAnsible.Name).
			SetNumber(taskCount + 3).
			SetState(agenttask.StateAWAITING).
			SetAgentTaskToProvisionedHost(entProvisionedHost).
			SetAgentTaskToProvisioningScheduledStep(entProvisioningScheduledStep).
			Save(ctx)
		if err != nil {
			logger.Log.Errorf("failed Creating Agent Task for Script Delete: %v", err)
			return
		}
		_, err = client.AgentTask.Create().
			SetCommand(agenttask.CommandDELETE).
			SetArgs("/tmp/" + entAnsible.Name + ".zip").
			SetNumber(taskCount + 4).
			SetState(agenttask.StateAWAITING).
			SetAgentTaskToProvisionedHost(entProvisionedHost).
			SetAgentTaskToProvisioningScheduledStep(entProvisioningScheduledStep).
			Save(ctx)
		if err != nil {
			logger.Log.Errorf("failed Creating Agent Task for Script Delete: %v", err)
			return
		}
	default:
		break
	}

	for {
		taskFailed, err := entProvisioningScheduledStep.QueryAgentTask().Where(
			agenttask.StateEQ(
				agenttask.StateFAILED,
			),
		).Exist(ctx)

		if err != nil {
			logger.Log.Errorf("Failed to Query Agent Task State. Err: %v", err)
			return
		}

		if taskFailed {
			_, err = entStatus.Update().SetFailed(true).SetState(status.StateFAILED).Save(ctx)
			if err != nil {
				logger.Log.Errorf("error while trying to set entProvisioningScheduledStep.Status.State to FAILED: %v", err)
				return
			}
			rdb.Publish(ctx, "updatedStatus", entStatus.ID.String())
			logger.Log.Errorf("one or more agent tasks failed")
			return
		}

		taskRunning, err := entProvisioningScheduledStep.QueryAgentTask().Where(
			agenttask.StateNEQ(
				agenttask.StateCOMPLETE,
			),
		).Exist(ctx)

		if err != nil {
			logger.Log.Errorf("Failed to Query Agent Task State. Err: %v", err)
			return
		}

		if !taskRunning {
			break
		}

		time.Sleep(time.Second)
	}
	err = entStatus.Update().SetCompleted(true).SetState(status.StateCOMPLETE).Exec(ctx)
	if err != nil {
		logger.Log.Errorf("error while trying to set entProvisioningScheduledStep.Status.State to COMPLETED: %v", err)
		return
	}
	rdb.Publish(ctx, "updatedStatus", entStatus.ID.String())
}
