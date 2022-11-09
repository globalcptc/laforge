package main

import (
	"context"
	"time"

	"github.com/gen0cide/laforge/ent"
)

func main() {
	entScheduleSteps := []*ent.ScheduleStep{
		{
			ID:                                       [16]byte{},
			Type:                                     "",
			Repeated:                                 false,
			StartTime:                                time.Time{},
			EndTime:                                  time.Time{},
			Interval:                                 0,
			Edges:                                    ent.ScheduleStepEdges{},
			HCLScheduleStepToStatus:                  &ent.Status{},
			HCLScheduleStepToScript:                  &ent.Script{},
			HCLScheduleStepToCommand:                 &ent.Command{},
			HCLScheduleStepToFileDelete:              &ent.FileDelete{},
			HCLScheduleStepToFileDownload:            &ent.FileDownload{},
			HCLScheduleStepToFileExtract:             &ent.FileExtract{},
			HCLScheduleStepToAnsible:                 &ent.Ansible{},
			HCLScheduleStepToProvisionedScheduleStep: []*ent.ProvisionedScheduleStep{},
			HCLScheduleStepToHost:                    &ent.Host{},
		},
	}

}

// GenerateProvisionedScheduleStep is a skeleton of the scheduling function
func GenerateProvisionedScheduleStep(client *ent.Client, entScheduleSteps []*ent.ScheduleStep) ([]*ent.ProvisionedScheduleStepCreate, error) {
	entProvisionedScheduleStepCreate := make([]*ent.ProvisionedScheduleStepCreate, len(entScheduleSteps))

	// Loop to read through Sched. Steps
	for _, entScheduleStep := range entScheduleSteps {
		// Determine RunTime
		if entScheduleStep.Repeated {
			interval := entScheduleStep.EndTime.Sub(entScheduleStep.StartTime).Milliseconds()
			interval /= 1000 // convert to milliseconds
			interval /= int64(entScheduleStep.Interval)

			for i := int64(0); i < interval; i++ {
				runtime := entScheduleStep.StartTime
				runtime.Add(time.Duration(i * interval * 1000)) // interval in milliseconds, converts to nanoseconds
				entProvisionedScheduleStep := client.ProvisionedScheduleStep.Create().SetProvisionedScheduleStepToScheduleStep(entScheduleStep)
				entProvisionedScheduleStep.SetRunTime(runtime)
				entProvisionedScheduleStepCreate = append(entProvisionedScheduleStepCreate, entProvisionedScheduleStep)
			}
		} else {
			entProvisionedScheduleStep := client.ProvisionedScheduleStep.Create().SetProvisionedScheduleStepToScheduleStep(entScheduleStep)
			entProvisionedScheduleStep.SetRunTime(entScheduleStep.StartTime)
			entProvisionedScheduleStepCreate = append(entProvisionedScheduleStepCreate, entProvisionedScheduleStep)
		}
	}
	return entProvisionedScheduleStepCreate, nil
}

// ScheduleStepWatchdog will be used as a Go routine
// func ScheduleStepWatchdog(ctx context.Context, client *ent.Client) { <-- THIS IS THE REAL SIGNATURE
func ScheduleStepWatchdog(ctx context.Context, entProvisionedScheduledSteps []*ent.ProvisionedScheduleStep) {
	select {
	case <-ctx.Done():
		return
	default:
		// Query DB for all provisioned scheduled steps that haven't been run and their run_time is in the past

		// Loop over all queried steps (for now use the func param and filter)
		// Create AgentTask objects for those (https://github.com/globalcptc/laforge/blob/c865ba5f078f7168982b4bd1197fcb5a366ded43/planner/build.go#L849-L1085)
		// Mark ProvisionedSchuledStep as Complete (equivalent to run) via status

		// Sleep and then check again
		time.Sleep(1 * time.Minute)
	}
}
