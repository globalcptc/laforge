package main

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/gen0cide/laforge/ent"
	"github.com/gen0cide/laforge/ent/schedulestep"
)

func main() {
	entScheduleSteps := []*ent.ScheduleStep{
		{
			ID:                                       [16]byte{},
			Type:                                     schedulestep.TypeScript,
			Repeated:                                 false,
			StartTime:                                time.Now().Add(1 * time.Minute),
			EndTime:                                  time.Now().Add(10 * time.Minute),
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
		{
			ID:                                       [16]byte{},
			Type:                                     schedulestep.TypeCommand,
			Repeated:                                 true,
			StartTime:                                time.Now().Add(1 * time.Minute),
			EndTime:                                  time.Now().Add(30 * time.Minute),
			Interval:                                 5,
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
		{
			ID:                                       [16]byte{},
			Type:                                     schedulestep.TypeFileDownload,
			Repeated:                                 false,
			StartTime:                                time.Now().Add(1 * time.Minute),
			EndTime:                                  time.Now().Add(5 * time.Minute),
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

	ctx, _ := context.WithCancel(context.Background())

	ScheduleStepWatchdog(ctx, entScheduleSteps)
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
func ScheduleStepWatchdog(ctx context.Context, entScheduledSteps []*ent.ScheduleStep) {
	select {
	case <-ctx.Done():
		return
	default:
		for _, entScheduledStep := range entScheduledSteps {

			// TODO: Query DB once integrated
			// entScheduledStep, err := entProvisionedScheduledStep.QueryProvisionedScheduleStepToScheduleStep().Only(ctx)
			// if err != nil {
			// 	// TODO: Log error
			// 	continue
			// }
			switch entScheduledStep.Type {
			case schedulestep.TypeScript:
				fmt.Println("Starting " + entScheduledStep.Type.String() +
					" at " + entScheduledStep.StartTime.String() +
					" Ending at " + entScheduledStep.EndTime.String() +
					" and is repeated for " + strconv.FormatBool(entScheduledStep.Repeated))
			case schedulestep.TypeCommand:
				fmt.Println("Starting " + entScheduledStep.Type.String() +
					" at " + entScheduledStep.StartTime.String() +
					" Ending at " + entScheduledStep.EndTime.String() +
					" and is repeated for " + strconv.FormatBool(entScheduledStep.Repeated))
			case schedulestep.TypeFileDelete:
				fmt.Println("Starting " + entScheduledStep.Type.String() +
					" at " + entScheduledStep.StartTime.String() +
					" Ending at " + entScheduledStep.EndTime.String() +
					" and is repeated for " + strconv.FormatBool(entScheduledStep.Repeated))
			case schedulestep.TypeFileDownload:
				fmt.Println("Starting " + entScheduledStep.Type.String() +
					" at " + entScheduledStep.StartTime.String() +
					" Ending at " + entScheduledStep.EndTime.String() +
					" and is repeated for " + strconv.FormatBool(entScheduledStep.Repeated))
			case schedulestep.TypeFileExtract:
				fmt.Println("Starting " + entScheduledStep.Type.String() +
					" at " + entScheduledStep.StartTime.String() +
					" Ending at " + entScheduledStep.EndTime.String() +
					" and is repeated for " + strconv.FormatBool(entScheduledStep.Repeated))
			case schedulestep.TypeAnsible:
				fmt.Println("Starting " + entScheduledStep.Type.String() +
					" at " + entScheduledStep.StartTime.String() +
					" Ending at " + entScheduledStep.EndTime.String() +
					" and is repeated for " + strconv.FormatBool(entScheduledStep.Repeated))
			default:
				fmt.Println("Should not be here")
				break
			}
			// Query DB for all provisioned scheduled steps that haven't been run and their run_time is in the past

			// Loop over all queried steps (for now use the func param and filter)
			// Create AgentTask objects for those (https://github.com/globalcptc/laforge/blob/c865ba5f078f7168982b4bd1197fcb5a366ded43/planner/build.go#L849-L1085)
			// Mark ProvisionedSchuledStep as Complete (equivalent to run) via status

			// Sleep and then check again
			time.Sleep(1 * time.Second)
		}
	}
}
