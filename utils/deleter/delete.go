package main

import (
	"context"
	"log"
	"os"

	"github.com/gen0cide/laforge/ent"
	"github.com/gen0cide/laforge/ent/command"
	"github.com/gen0cide/laforge/ent/competition"
	"github.com/gen0cide/laforge/ent/dns"
	"github.com/gen0cide/laforge/ent/dnsrecord"
	"github.com/gen0cide/laforge/ent/filedownload"
	"github.com/gen0cide/laforge/ent/finding"
	"github.com/gen0cide/laforge/ent/host"
	"github.com/gen0cide/laforge/ent/hostdependency"
	"github.com/gen0cide/laforge/ent/identity"
	"github.com/gen0cide/laforge/ent/includednetwork"
	"github.com/gen0cide/laforge/ent/network"
	"github.com/gen0cide/laforge/ent/script"
	"github.com/gen0cide/laforge/server/utils"
	"github.com/sirupsen/logrus"
)

func main() {
	logrus.SetLevel(logrus.DebugLevel)

	laforgeConfig, err := utils.LoadServerConfig()
	if err != nil {
		logrus.Errorf("failed to load LaForge config: %v", err)
		return
	}

	if laforgeConfig.Database.PostgresUri == "" {
		logrus.Errorf("Database.PostgresUri not set in LaForge config")
		os.Exit(1)
	}

	client := ent.PGOpen(laforgeConfig.Database.PostgresUri)

	ctx := context.Background()
	defer ctx.Done()
	defer client.Close()

	// Run the auto migration tool.
	if err := client.Schema.Create(ctx); err != nil {
		log.Fatalf("failed creating schema resources: %v", err)
	}

	//GinFileMiddleware --
	//AgentStatus --
	//AgentTask --
	//ProvisioningStep --
	//ProvisionedHost --
	//ProvisionedNetwork --
	//Team --
	//BuildCommit --
	//Plan --
	//AdhocPlan --

	//Build --
	//Enviroment --

	deletedCount, err := client.Host.Delete().Where(host.Not(host.HasEnvironment())).Exec(ctx)
	if err != nil {
		log.Fatalf("failed to get env: %v", err)
	}
	println(deletedCount)
	deletedCount, err = client.Competition.Delete().Where(competition.Not(competition.HasEnvironment())).Exec(ctx)
	if err != nil {
		log.Fatalf("failed to get env: %v", err)
	}
	println(deletedCount)
	deletedCount, err = client.Identity.Delete().Where(identity.Not(identity.HasEnvironment())).Exec(ctx)
	if err != nil {
		log.Fatalf("failed to get env: %v", err)
	}
	println(deletedCount)
	deletedCount, err = client.Command.Delete().Where(command.Not(command.HasEnvironment())).Exec(ctx)
	if err != nil {
		log.Fatalf("failed to get env: %v", err)
	}
	println(deletedCount)
	deletedCount, err = client.Script.Delete().Where(script.Not(script.HasScriptToEnvironment())).Exec(ctx)
	if err != nil {
		log.Fatalf("failed to get env: %v", err)
	}
	println(deletedCount)
	deletedCount, err = client.FileDownload.Delete().Where(filedownload.Not(filedownload.HasEnvironment())).Exec(ctx)
	if err != nil {
		log.Fatalf("failed to get env: %v", err)
	}
	println(deletedCount)
	deletedCount, err = client.IncludedNetwork.Delete().Where(includednetwork.Not(includednetwork.HasEnvironments())).Exec(ctx)
	if err != nil {
		log.Fatalf("failed to get env: %v", err)
	}
	println(deletedCount)
	deletedCount, err = client.Finding.Delete().Where(finding.Not(finding.HasEnvironment())).Exec(ctx)
	if err != nil {
		log.Fatalf("failed to get env: %v", err)
	}
	println(deletedCount)
	deletedCount, err = client.DNSRecord.Delete().Where(dnsrecord.Not(dnsrecord.HasEnvironment())).Exec(ctx)
	if err != nil {
		log.Fatalf("failed to get env: %v", err)
	}
	println(deletedCount)
	deletedCount, err = client.DNS.Delete().Where(dns.Not(dns.HasEnvironments())).Exec(ctx)
	if err != nil {
		log.Fatalf("failed to get env: %v", err)
	}
	println(deletedCount)
	deletedCount, err = client.Network.Delete().Where(network.Not(network.HasEnvironment())).Exec(ctx)
	if err != nil {
		log.Fatalf("failed to get env: %v", err)
	}
	println(deletedCount)
	deletedCount, err = client.HostDependency.Delete().Where(hostdependency.Not(hostdependency.HasEnvironment())).Exec(ctx)
	if err != nil {
		log.Fatalf("failed to get env: %v", err)
	}
	println(deletedCount)

	// entEnvironment, err := client.Environment.Query().Where(environment.HclIDEQ("/envs/jrwr-2021-regional-dev")).Only(ctx)
	// if err != nil {
	// 	log.Fatalf("failed to get env: %v", err)
	// }
	// entBuilds, err := entEnvironment.QueryBuilds().All(ctx)
	// if err != nil {
	// 	log.Fatalf("failed to get builds: %v", err)
	// }
	// for _, entBuild := range entBuilds {
	// 	amountDeleted, err := client.GinFileMiddleware.Delete().Where(
	// 		ginfilemiddleware.HasProvisionedHostWith(
	// 			provisionedhost.HasProvisionedNetworkWith(
	// 				provisionednetwork.HasBuildWith(
	// 					build.IDEQ(entBuild.ID),
	// 				),
	// 			),
	// 		),
	// 	).Exec(ctx)
	// 	if err != nil {
	// 		logrus.Fatalf("failed to delete middleware: %v", err)
	// 	}
	// 	logrus.Infof("Deleted %v amount of GinMiddleware for Build %v for env %v", amountDeleted, entBuild.Revision, entEnvironment.HclID)
	// 	amountDeleted, err = client.AgentStatus.Delete().Where(
	// 		agentstatus.HasAgentStatusToBuildWith(
	// 			build.IDEQ(entBuild.ID),
	// 		),
	// 	).Exec(ctx)
	// 	if err != nil {
	// 		logrus.Fatalf("failed to delete agentstatus: %v", err)
	// 	}
	// 	logrus.Infof("Deleted %v amount of agentstatus for Build %v for env %v", amountDeleted, entBuild.Revision, entEnvironment.HclID)
	// 	amountDeleted, err = client.AgentTask.Delete().Where(
	// 		agenttask.HasAgentTaskToProvisionedHostWith(
	// 			provisionedhost.HasProvisionedNetworkWith(
	// 				provisionednetwork.HasBuildWith(
	// 					build.IDEQ(entBuild.ID),
	// 				),
	// 			),
	// 		),
	// 	).Exec(ctx)
	// 	if err != nil {
	// 		logrus.Fatalf("failed to delete agenttask: %v", err)
	// 	}
	// 	logrus.Infof("Deleted %v amount of agenttask for Build %v for env %v", amountDeleted, entBuild.Revision, entEnvironment.HclID)
	// 	amountDeleted, err = client.ProvisioningStep.Delete().Where(
	// 		provisioningstep.HasProvisionedHostWith(
	// 			provisionedhost.HasProvisionedNetworkWith(
	// 				provisionednetwork.HasBuildWith(
	// 					build.IDEQ(entBuild.ID),
	// 				),
	// 			),
	// 		),
	// 	).Exec(ctx)
	// 	if err != nil {
	// 		logrus.Fatalf("failed to delete pstep: %v", err)
	// 	}
	// 	logrus.Infof("Deleted %v amount of pstep for Build %v for env %v", amountDeleted, entBuild.Revision, entEnvironment.HclID)
	// 	amountDeleted, err = client.ProvisionedHost.Delete().Where(
	// 		provisionedhost.HasProvisionedNetworkWith(
	// 			provisionednetwork.HasBuildWith(
	// 				build.IDEQ(entBuild.ID),
	// 			),
	// 		),
	// 	).Exec(ctx)
	// 	if err != nil {
	// 		logrus.Fatalf("failed to delete phost: %v", err)
	// 	}
	// 	logrus.Infof("Deleted %v amount of phost for Build %v for env %v", amountDeleted, entBuild.Revision, entEnvironment.HclID)
	// 	amountDeleted, err = client.ProvisionedNetwork.Delete().Where(
	// 		provisionednetwork.HasBuildWith(
	// 			build.IDEQ(entBuild.ID),
	// 		),
	// 	).Exec(ctx)
	// 	amountDeleted, err = client.Team.Delete().Where(
	// 		team.HasTeamToBuildWith(
	// 			build.IDEQ(entBuild.ID),
	// 		),
	// 	).Exec(ctx)
	// 	if err != nil {
	// 		logrus.Fatalf("failed to delete ProvisionedNetwork: %v", err)
	// 	}
	// 	logrus.Infof("Deleted %v amount of ProvisionedNetwork for Build %v for env %v", amountDeleted, entBuild.Revision, entEnvironment.HclID)
	// 	amountDeleted, err = client.BuildCommit.Delete().Where(
	// 		buildcommit.HasBuildCommitToBuildWith(
	// 			build.IDEQ(entBuild.ID),
	// 		),
	// 	).Exec(ctx)
	// 	if err != nil {
	// 		logrus.Fatalf("failed to delete BuildCommit: %v", err)
	// 	}
	// 	logrus.Infof("Deleted %v amount of BuildCommit for Build %v for env %v", amountDeleted, entBuild.Revision, entEnvironment.HclID)
	// 	amountDeleted, err = client.Plan.Delete().Where(
	// 		plan.HasBuildWith(
	// 			build.IDEQ(entBuild.ID),
	// 		),
	// 	).Exec(ctx)
	// 	if err != nil {
	// 		logrus.Fatalf("failed to delete plan: %v", err)
	// 	}
	// 	logrus.Infof("Deleted %v amount of plan for Build %v for env %v", amountDeleted, entBuild.Revision, entEnvironment.HclID)
	// 	amountDeleted, err = client.AdhocPlan.Delete().Where(
	// 		adhocplan.HasAdhocBuildWith(
	// 			build.IDEQ(entBuild.ID),
	// 		),
	// 	).Exec(ctx)
	// 	if err != nil {
	// 		logrus.Fatalf("failed to delete AdhocPlan: %v", err)
	// 	}
	// 	logrus.Infof("Deleted %v amount of AdhocPlan for Build %v for env %v", amountDeleted, entBuild.Revision, entEnvironment.HclID)
	// 	amountDeleted, err = client.Build.Delete().Where(
	// 		build.IDEQ(entBuild.ID),
	// 	).Exec(ctx)
	// 	if err != nil {
	// 		logrus.Fatalf("failed to delete Build: %v", err)
	// 	}
	// 	logrus.Infof("Deleted Build %v for env %v", amountDeleted, entBuild.Revision, entEnvironment.HclID)
	// }
	// amountDeleted, err := client.Environment.Delete().Where(
	// 	environment.IDEQ(entEnvironment.ID),
	// ).Exec(ctx)
	// if err != nil {
	// 	logrus.Fatalf("failed to delete Build: %v", err)
	// }
	// logrus.Infof("Deleted Env %v", amountDeleted, entEnvironment.HclID)
}
