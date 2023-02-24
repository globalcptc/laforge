package main

import (
	"context"
	"fmt"
	"time"

	"github.com/gen0cide/laforge/ent"
	"github.com/gen0cide/laforge/ent/ansible"
	"github.com/gen0cide/laforge/ent/command"
	"github.com/gen0cide/laforge/ent/dnsrecord"
	"github.com/gen0cide/laforge/ent/environment"
	"github.com/gen0cide/laforge/ent/filedelete"
	"github.com/gen0cide/laforge/ent/filedownload"
	"github.com/gen0cide/laforge/ent/fileextract"
	"github.com/gen0cide/laforge/ent/provisioningscheduledstep"
	"github.com/gen0cide/laforge/ent/script"
)

func main() {
	ctx := context.Background()
	createProvisioningScheduleStep(ctx, nil, nil, nil)
}

// createProvisionedScheduleStep is a skeleton of the scheduling function
func createProvisioningScheduleStep(ctx context.Context, client *ent.Client, entProvisionedHost *ent.ProvisionedHost, entScheduledSteps []*ent.ScheduledStep) error {
	// Loop to read through Sched. Steps
	for _, entScheduledStep := range entScheduledSteps {
		// Determine RunTime
		if entScheduledStep.Repeated {
			bulkProvisioningScheduledSteps := make([]*ent.ProvisioningScheduledStepCreate, 0)
			for run_time := entScheduledStep.StartTime; run_time <= entScheduledStep.EndTime; run_time += int64(entScheduledStep.Interval) {
				// entStatus, err := createPlanningStatus(ctx, client, logger, status.StatusForProvisioningScheduledStep)
				// if err != nil {
				// 	return nil, fmt.Errorf("failed to create status for provisioning scheduled step: %v", err)
				// }
				entProvisionedScheduleStepCreate, err := generateProvisioningScheduledStepByType(ctx, client, entScheduledStep)
				if err != nil {
					return fmt.Errorf("failed to generate provisioning scheduled step by type: %v", err)
				}
				entProvisionedScheduleStepCreate = entProvisionedScheduleStepCreate.
					SetProvisioningScheduledStepToProvisionedHost(entProvisionedHost).
					SetRunTime(time.Unix(run_time, 0))
					// SetProvisioningScheduledStepToStatus(entStatus)
				bulkProvisioningScheduledSteps = append(bulkProvisioningScheduledSteps, entProvisionedScheduleStepCreate)
			}
			err := client.ProvisioningScheduledStep.CreateBulk(bulkProvisioningScheduledSteps...).Exec(ctx)
			if err != nil {
				return fmt.Errorf("failed to bulk create repeated provisioning scheduled steps: %v", err)
			}
		} else {
			// entStatus, err := createPlanningStatus(ctx, client, logger, status.StatusForProvisioningScheduledStep)
			// if err != nil {
			// 	return nil, fmt.Errorf("failed to create status for provisioning scheduled step: %v", err)
			// }
			entProvisionedScheduleStepCreate, err := generateProvisioningScheduledStepByType(ctx, client, entScheduledStep)
			if err != nil {
				return fmt.Errorf("failed to generate provisioning scheduled step by type: %v", err)
			}
			err = entProvisionedScheduleStepCreate.
				SetProvisioningScheduledStepToProvisionedHost(entProvisionedHost).
				SetRunTime(time.Unix(entScheduledStep.StartTime, 0)).
				// SetProvisioningScheduleStepToStatus(entStatus)
				Exec(ctx)
			if err != nil {
				return fmt.Errorf("failed to create provisioning scheduled step: %v", err)
			}
		}
		// if RenderFiles {
		// 	filePath, err := renderScript(ctx, client, logger, entProvisioningStep)
		// 	if err != nil {
		// 		return nil, err
		// 	}
		// 	entTmpUrl, err := utils.CreateTempURL(ctx, client, filePath)
		// 	if err != nil {
		// 		return nil, err
		// 	}
		// 	_, err = entTmpUrl.Update().SetGinFileMiddlewareToProvisioningStep(entProvisioningStep).Save(ctx)
		// 	if err != nil {
		// 		return nil, err
		// 	}
		// 	if RenderFilesTask != nil {
		// 		RenderFilesTask, err = RenderFilesTask.Update().AddServerTaskToGinFileMiddleware(entTmpUrl).Save(ctx)
		// 		if err != nil {
		// 			return nil, err
		// 		}
		// 	}
		// }
	}
	return nil
}

func generateProvisioningScheduledStepByType(ctx context.Context, client *ent.Client, entScheduledStep *ent.ScheduledStep) (*ent.ProvisioningScheduledStepCreate, error) {
	entEnvironment, err := entScheduledStep.QueryScheduledStepToEnvironment().Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query environment from scheduled step: %v", err)
	}
	// Check if step is script
	entScript, err := client.Script.Query().Where(
		script.And(
			script.HasScriptToEnvironmentWith(
				environment.IDEQ(entEnvironment.ID),
			),
			script.HclIDEQ(entScheduledStep.Step),
		),
	).Only(ctx)
	if err == nil {
		// Step is a script
		return client.ProvisioningScheduledStep.Create().
			SetType(provisioningscheduledstep.TypeScript).
			SetProvisioningScheduledStepToScript(entScript), nil
	} else if err != nil && !ent.IsNotFound(err) {
		return nil, fmt.Errorf("failed to query for script based on hcl_id from scheduled step: %v", err)
	}
	// Check if step is command
	entCommand, err := client.Command.Query().Where(
		command.And(
			command.HasCommandToEnvironmentWith(
				environment.IDEQ(entEnvironment.ID),
			),
			command.HclIDEQ(entScheduledStep.Step),
		),
	).Only(ctx)
	if err == nil {
		// Step is a command
		return client.ProvisioningScheduledStep.Create().
			SetType(provisioningscheduledstep.TypeCommand).
			SetProvisioningScheduledStepToCommand(entCommand), nil
	} else if err != nil && !ent.IsNotFound(err) {
		return nil, fmt.Errorf("failed to query for command based on hcl_id from scheduled step: %v", err)
	}
	// Check if step is file download
	entFileDownload, err := client.FileDownload.Query().Where(
		filedownload.And(
			filedownload.HasFileDownloadToEnvironmentWith(
				environment.IDEQ(entEnvironment.ID),
			),
			filedownload.HclIDEQ(entScheduledStep.Step),
		),
	).Only(ctx)
	if err == nil {
		// Step is a file download
		return client.ProvisioningScheduledStep.Create().
			SetType(provisioningscheduledstep.TypeFileDownload).
			SetProvisioningScheduledStepToFileDownload(entFileDownload), nil
	} else if err != nil && !ent.IsNotFound(err) {
		return nil, fmt.Errorf("failed to query for file download based on hcl_id from scheduled step: %v", err)
	}
	// Check if step is file extract
	entFileExtract, err := client.FileExtract.Query().Where(
		fileextract.And(
			fileextract.HasFileExtractToEnvironmentWith(
				environment.IDEQ(entEnvironment.ID),
			),
			fileextract.HclIDEQ(entScheduledStep.Step),
		),
	).Only(ctx)
	if err == nil {
		// Step is a file extract
		return client.ProvisioningScheduledStep.Create().
			SetType(provisioningscheduledstep.TypeFileExtract).
			SetProvisioningScheduledStepToFileExtract(entFileExtract), nil
	} else if err != nil && !ent.IsNotFound(err) {
		return nil, fmt.Errorf("failed to query for file extract based on hcl_id from scheduled step: %v", err)
	}
	// Check if step is file delete
	entFileDelete, err := client.FileDelete.Query().Where(
		filedelete.And(
			filedelete.HasFileDeleteToEnvironmentWith(
				environment.IDEQ(entEnvironment.ID),
			),
			filedelete.HclIDEQ(entScheduledStep.Step),
		),
	).Only(ctx)
	if err == nil {
		// Step is a file delete
		return client.ProvisioningScheduledStep.Create().
			SetType(provisioningscheduledstep.TypeFileDelete).
			SetProvisioningScheduledStepToFileDelete(entFileDelete), nil
	} else if err != nil && !ent.IsNotFound(err) {
		return nil, fmt.Errorf("failed to query for file delete based on hcl_id from scheduled step: %v", err)
	}
	// Check if step is dns record
	entDNSRecord, err := client.DNSRecord.Query().Where(
		dnsrecord.And(
			dnsrecord.HasDNSRecordToEnvironmentWith(
				environment.IDEQ(entEnvironment.ID),
			),
			dnsrecord.HclIDEQ(entScheduledStep.Step),
		),
	).Only(ctx)
	if err == nil {
		// Step is a dns record
		return client.ProvisioningScheduledStep.Create().
			SetType(provisioningscheduledstep.TypeDNSRecord).
			SetProvisioningScheduledStepToDNSRecord(entDNSRecord), nil
	} else if err != nil && !ent.IsNotFound(err) {
		return nil, fmt.Errorf("failed to query for dns record based on hcl_id from scheduled step: %v", err)
	}
	// Check if step is ansible
	entAnsible, err := client.Ansible.Query().Where(
		ansible.And(
			ansible.HasAnsibleFromEnvironmentWith(
				environment.IDEQ(entEnvironment.ID),
			),
			ansible.HclIDEQ(entScheduledStep.Step),
		),
	).Only(ctx)
	if err == nil {
		// Step is a ansible
		return client.ProvisioningScheduledStep.Create().
			SetType(provisioningscheduledstep.TypeAnsible).
			SetProvisioningScheduledStepToAnsible(entAnsible), nil
	} else if err != nil && !ent.IsNotFound(err) {
		return nil, fmt.Errorf("failed to query for ansible based on hcl_id from scheduled step: %v", err)
	}
	return nil, fmt.Errorf("unknown scheduled step type")
}
