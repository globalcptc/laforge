package main

import (
	"context"
	"fmt"
	"time"

	"github.com/gorhill/cronexpr"
	hcl2 "github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	hcl2parse "github.com/hashicorp/hcl/v2/hclparse"
	"github.com/sirupsen/logrus"

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
	// ctx := context.Background()
	testConfig, err := parseTestConfigs()
	if err != nil {
		logrus.Errorf("failed to parse test configs: %v", err)
		return
	}

	logrus.Infof("Competitions: %v", testConfig.Competitions)
	logrus.Infof("Scheduled Steps: %v", testConfig.ScheduledSteps)
	// createProvisioningScheduleStep(ctx, nil, nil, nil)
}

var TEST_ENV = []byte(`
competition "laforge-demo" {
  root_password = "test123"

	start_time = 1678208400
	stop_time = 1678294800

  dns "default" {
    type = "bind"
    root_domain = "cp.tc"

    dns_servers = [
      "8.8.8.8",
      "8.8.4.4",
    ]

    ntp_servers = [
      "129.6.15.28",
      "129.6.15.29",
    ]
  }
}

scheduled "/scheduled/windows/reboot" {
	name = "Periodic Reboot"
	description = "Reboot windows every 3 hours"

	step = "/commands/generic/reboot"

	type = "cron" // cron, runonce

	// Cron schedule
	schedule = "0 */3 * * *" // Every 3 hours

	// Unix time to run at
	// run_time = 1677296367
}`)

type DefinedConfigs struct {
	Competitions   []*ent.Competition   `hcl:"competition,block"`
	ScheduledSteps []*ent.ScheduledStep `hcl:"scheduled,block"`
}

func parseTestConfigs() (*DefinedConfigs, error) {
	p := hcl2parse.NewParser()

	_, diags := p.ParseHCL(TEST_ENV, "test_env.hcl")
	if diags.HasErrors() {
		for _, e := range diags.Errs() {
			ne, ok := e.(*hcl2.Diagnostic)
			if ok {
				logrus.Errorf("Laforge failed to parse a config file:\n Location: %v\n    Issue: %v\n   Detail: %v", ne.Subject, ne.Summary, ne.Detail)
			}
		}
		return nil, fmt.Errorf("failed to parse HCL")
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
			return nil, fmt.Errorf("failed to parse HCL")
		}
	}

	return &testConfig, nil
}

// createProvisionedScheduleStep is used when competitions have a well-defined start and stop time and are cron scheduled
func createProvisioningScheduleStep(ctx context.Context, client *ent.Client, entCompetition *ent.Competition, entProvisionedHost *ent.ProvisionedHost, entScheduledSteps []*ent.ScheduledStep) error {
	// Loop to read through Sched. Steps
	for _, entScheduledStep := range entScheduledSteps {
		// Determine RunTime
		scheduleExpr, err := cronexpr.Parse(entScheduledStep.Schedule)
		if err != nil {
			return fmt.Errorf("failed to parse scheduled step schedule: %v", err)
		}
		runTime := scheduleExpr.Next(time.Unix(entCompetition.StartTime, 0))
		for runTime.Unix() <= entCompetition.StopTime {
			// entStatus, err := createPlanningStatus(ctx, client, logger, status.StatusForProvisioningScheduledStep)
			// if err != nil {
			// 	return nil, fmt.Errorf("failed to create status for provisioning scheduled step: %v", err)
			// }
			entProvisionedScheduleStepCreate, err := generateProvisioningScheduledStepByType(ctx, client, entScheduledStep)
			if err != nil {
				return fmt.Errorf("failed to generate provisioning scheduled step by type: %v", err)
			}
			err = entProvisionedScheduleStepCreate.
				SetProvisionedHost(entProvisionedHost).
				SetRunTime(runTime).
				// SetProvisioningScheduleStepToStatus(entStatus)
				Exec(ctx)
			if err != nil {
				return fmt.Errorf("failed to create provisioning scheduled step: %v", err)
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
			runTime = scheduleExpr.Next(runTime)
		}
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
			SetScript(entScript), nil
	} else if err != nil && !ent.IsNotFound(err) {
		return nil, fmt.Errorf("failed to query for script based on hcl_id from scheduled step: %v", err)
	}
	// Check if step is command
	entCommand, err := client.Command.Query().Where(
		command.And(
			command.HasEnvironmentWith(
				environment.IDEQ(entEnvironment.ID),
			),
			command.HclIDEQ(entScheduledStep.Step),
		),
	).Only(ctx)
	if err == nil {
		// Step is a command
		return client.ProvisioningScheduledStep.Create().
			SetType(provisioningscheduledstep.TypeCommand).
			SetCommand(entCommand), nil
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
			SetFileDownload(entFileDownload), nil
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
			SetFileExtract(entFileExtract), nil
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
			SetFileDelete(entFileDelete), nil
	} else if err != nil && !ent.IsNotFound(err) {
		return nil, fmt.Errorf("failed to query for file delete based on hcl_id from scheduled step: %v", err)
	}
	// Check if step is dns record
	entDNSRecord, err := client.DNSRecord.Query().Where(
		dnsrecord.And(
			dnsrecord.HasEnvironmentWith(
				environment.IDEQ(entEnvironment.ID),
			),
			dnsrecord.HclIDEQ(entScheduledStep.Step),
		),
	).Only(ctx)
	if err == nil {
		// Step is a dns record
		return client.ProvisioningScheduledStep.Create().
			SetType(provisioningscheduledstep.TypeDNSRecord).
			SetDNSRecord(entDNSRecord), nil
	} else if err != nil && !ent.IsNotFound(err) {
		return nil, fmt.Errorf("failed to query for dns record based on hcl_id from scheduled step: %v", err)
	}
	// Check if step is ansible
	entAnsible, err := client.Ansible.Query().Where(
		ansible.And(
			ansible.HasEnvironmentWith(
				environment.IDEQ(entEnvironment.ID),
			),
			ansible.HclIDEQ(entScheduledStep.Step),
		),
	).Only(ctx)
	if err == nil {
		// Step is a ansible
		return client.ProvisioningScheduledStep.Create().
			SetType(provisioningscheduledstep.TypeAnsible).
			SetAnsible(entAnsible), nil
	} else if err != nil && !ent.IsNotFound(err) {
		return nil, fmt.Errorf("failed to query for ansible based on hcl_id from scheduled step: %v", err)
	}
	return nil, fmt.Errorf("unknown scheduled step type")
}
