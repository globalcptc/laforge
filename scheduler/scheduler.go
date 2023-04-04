package scheduler

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/gen0cide/laforge/ent"
	"github.com/gen0cide/laforge/ent/provisioningscheduledstep"
	"github.com/gen0cide/laforge/ent/status"
	"github.com/gen0cide/laforge/logging"
	"github.com/gen0cide/laforge/server/utils"
	"github.com/sirupsen/logrus"
)

func SchedulerWatchdog(ctx context.Context, client *ent.Client, laforgeConfig *utils.ServerConfig) {
	schedulerLogger, err := InitSchedulerLogger(laforgeConfig)
	if err != nil {
		logrus.Errorf("failed to initialize scheduler watchdog logger: %v", err)
		return
	}
	// A loop which queries for all provisioning scheduled steps which:
	//   1) haven't been run (status is AWAITING)
	//   2) don't have a run_time of 0 (these are waiting on other things)
	ticker := time.NewTicker(time.Second)
	for {
		select {
		case <-ctx.Done():
			break
		case <-ticker.C:
			provisioningStepsToExecute, err := GetStepsToExecute(ctx, client)
			if err != nil {
				schedulerLogger.Log.Errorf("failed to query provisioning scheduled steps to execute: %v", err)
				continue
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
			provisioningscheduledstep.HasProvisioningScheduledStepToStatusWith(
				status.StateEQ(status.StateAWAITING),
			),
			provisioningscheduledstep.RunTimeNEQ(time.Unix(0, 0)),
		),
	).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query provisioning scheduled steps: %v", err)
	}
	return entProvisioningScheduledSteps, nil
}
