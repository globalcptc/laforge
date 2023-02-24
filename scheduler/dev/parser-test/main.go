package main

import (
	"github.com/gen0cide/laforge/ent"
	hcl2 "github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	hcl2parse "github.com/hashicorp/hcl/v2/hclparse"
	"github.com/sirupsen/logrus"
)

var TEST_SCHEDULED_STEP = []byte(`
scheduled_step "/schedule/linux/ping" {
	name = "Ping Google"
	description = "Ping Google every 5 minutes"

	step = "/scripts/linux/ping"

	start_time = 1677216701
	end_time = 1677217701
	interval = 5
	repeated = true
}`)

type DefinedConfigs struct {
	ScheduledSteps []*ent.ScheduledStep `hcl:"scheduled_step,block"`
}

func main() {
	p := hcl2parse.NewParser()

	_, diags := p.ParseHCL(TEST_SCHEDULED_STEP, "test.hcl")
	if diags.HasErrors() {
		for _, e := range diags.Errs() {
			ne, ok := e.(*hcl2.Diagnostic)
			if ok {
				logrus.Errorf("Laforge failed to parse a config file:\n Location: %v\n    Issue: %v\n   Detail: %v", ne.Subject, ne.Summary, ne.Detail)
			}
		}
		logrus.Panic("failed to parse HCL")
	}

	var testConfig DefinedConfigs

	for _, f := range p.Files() {
		diags := gohcl.DecodeBody(f.Body, nil, &testConfig)
		if diags.HasErrors() {
			for _, e := range diags.Errs() {
				ne, ok := e.(*hcl2.Diagnostic)
				if ok {
					logrus.Errorf("Laforge failed to parse a config file:\n Location: %v\n    Issue: %v\n   Detail: %v", ne.Subject, ne.Summary, ne.Detail)
				}
			}
			logrus.Panic("failed to parse HCL")
		}
	}

	logrus.Infof("%v", testConfig)
}

// // ScheduleStepWatchdog will be used as a Go routine
// // func ScheduleStepWatchdog(ctx context.Context, client *ent.Client) { <-- THIS IS THE REAL SIGNATURE
// func ScheduleStepWatchdog(ctx context.Context, entScheduledSteps []*ent.ScheduleStep) {
// 	select {
// 	case <-ctx.Done():
// 		return
// 	default:
// 		for _, entScheduledStep := range entScheduledSteps {

// 			// TODO: Query DB once integrated
// 			// entScheduledStep, err := entProvisionedScheduledStep.QueryProvisionedScheduleStepToScheduleStep().Only(ctx)
// 			// if err != nil {
// 			// 	// TODO: Log error
// 			// 	continue
// 			// }
// 			switch entScheduledStep.Type {
// 			case schedulestep.TypeScript:
// 				fmt.Println("Starting " + entScheduledStep.Type.String() +
// 					" at " + entScheduledStep.StartTime.String() +
// 					" Ending at " + entScheduledStep.EndTime.String() +
// 					" and is repeated for " + strconv.FormatBool(entScheduledStep.Repeated))
// 			case schedulestep.TypeCommand:
// 				fmt.Println("Starting " + entScheduledStep.Type.String() +
// 					" at " + entScheduledStep.StartTime.String() +
// 					" Ending at " + entScheduledStep.EndTime.String() +
// 					" and is repeated for " + strconv.FormatBool(entScheduledStep.Repeated))
// 			case schedulestep.TypeFileDelete:
// 				fmt.Println("Starting " + entScheduledStep.Type.String() +
// 					" at " + entScheduledStep.StartTime.String() +
// 					" Ending at " + entScheduledStep.EndTime.String() +
// 					" and is repeated for " + strconv.FormatBool(entScheduledStep.Repeated))
// 			case schedulestep.TypeFileDownload:
// 				fmt.Println("Starting " + entScheduledStep.Type.String() +
// 					" at " + entScheduledStep.StartTime.String() +
// 					" Ending at " + entScheduledStep.EndTime.String() +
// 					" and is repeated for " + strconv.FormatBool(entScheduledStep.Repeated))
// 			case schedulestep.TypeFileExtract:
// 				fmt.Println("Starting " + entScheduledStep.Type.String() +
// 					" at " + entScheduledStep.StartTime.String() +
// 					" Ending at " + entScheduledStep.EndTime.String() +
// 					" and is repeated for " + strconv.FormatBool(entScheduledStep.Repeated))
// 			case schedulestep.TypeAnsible:
// 				fmt.Println("Starting " + entScheduledStep.Type.String() +
// 					" at " + entScheduledStep.StartTime.String() +
// 					" Ending at " + entScheduledStep.EndTime.String() +
// 					" and is repeated for " + strconv.FormatBool(entScheduledStep.Repeated))
// 			default:
// 				fmt.Println("Should not be here")
// 				break
// 			}
// 			// Query DB for all provisioned scheduled steps that haven't been run and their run_time is in the past

// 			// Loop over all queried steps (for now use the func param and filter)
// 			// Create AgentTask objects for those (https://github.com/globalcptc/laforge/blob/c865ba5f078f7168982b4bd1197fcb5a366ded43/planner/build.go#L849-L1085)
// 			// Mark ProvisionedSchuledStep as Complete (equivalent to run) via status

// 			// Sleep and then check again
// 			time.Sleep(1 * time.Second)
// 		}
// 	}
// }
