package planner

import (
	"context"
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"text/template"

	"github.com/gen0cide/laforge/ent"
	"github.com/gen0cide/laforge/ent/build"
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
	"github.com/gen0cide/laforge/ent/script"
	"github.com/gen0cide/laforge/ent/status"
	"github.com/gen0cide/laforge/ent/team"
	"github.com/gen0cide/laforge/grpc"
	"github.com/gen0cide/laforge/server/utils"
	_ "github.com/mattn/go-sqlite3"
)

var RenderFiles = false

func main() {

	client := ent.SQLLiteOpen("file:test.sqlite?_loc=auto&cache=shared&_fk=1")
	ctx := context.Background()
	defer client.Close()

	// Run the auto migration tool.
	if err := client.Schema.Create(ctx); err != nil {
		log.Fatalf("failed creating schema resources: %v", err)
	}

	entEnvironment, err := client.Environment.Query().Where(environment.IDEQ(1)).WithEnvironmentToBuild().Only(ctx)
	if err != nil {
		log.Fatalf("Failed to find Environment %v. Err: %v", 1, err)
	}

	entBuild, _ := CreateBuild(ctx, client, entEnvironment)
	if err != nil {
		log.Fatalf("Failed to create Build for Enviroment %v. Err: %v", 1, err)
	}
	fmt.Println(entBuild)
}

func createPlanningStatus(ctx context.Context, client *ent.Client, statusFor status.StatusFor) (*ent.Status, error) {
	entStatus, err := client.Status.Create().SetState(status.StatePLANNING).SetStatusFor(statusFor).Save(ctx)
	if err != nil {
		log.Fatalf("Failed to create Status for %v. Err: %v", statusFor, err)
		return nil, err
	}
	return entStatus, nil
}

func CreateBuild(ctx context.Context, client *ent.Client, entEnvironment *ent.Environment) (*ent.Build, error) {
	var wg sync.WaitGroup
	entStatus, err := createPlanningStatus(ctx, client, status.StatusForBuild)
	if err != nil {
		return nil, err
	}
	entCompetition, err := client.Competition.Query().Where(competition.HclIDEQ(entEnvironment.CompetitionID)).Only(ctx)
	if err != nil {
		log.Fatalf("Failed to Query Competition %v for Enviroment %v. Err: %v", len(entEnvironment.CompetitionID), entEnvironment.HclID, err)
		return nil, err
	}
	entBuild, err := client.Build.Create().
		SetRevision(len(entEnvironment.Edges.EnvironmentToBuild)).
		SetBuildToEnvironment(entEnvironment).
		SetBuildToStatus(entStatus).
		SetBuildToCompetition(entCompetition).
		Save(ctx)
	if err != nil {
		log.Fatalf("Failed to create Build %v for Enviroment %v. Err: %v", len(entEnvironment.Edges.EnvironmentToBuild), entEnvironment.HclID, err)
		return nil, err
	}
	_, err = client.Plan.Create().
		SetType(plan.TypeStartBuild).
		SetBuildID(entBuild.ID).
		SetPlanToBuild(entBuild).
		SetStepNumber(0).
		Save(ctx)
	if err != nil {
		log.Fatalf("Failed to create Plan Node for Build %v. Err: %v", entBuild.ID, err)
		return nil, err
	}
	for teamNumber := 0; teamNumber <= entEnvironment.TeamCount; teamNumber++ {
		wg.Add(1)
		go createTeam(ctx, client, entBuild, teamNumber, &wg)
	}
	wg.Wait()
	return entBuild, nil
}

func createTeam(ctx context.Context, client *ent.Client, entBuild *ent.Build, teamNumber int, wg *sync.WaitGroup) (*ent.Team, error) {
	defer wg.Done()

	entStatus, err := createPlanningStatus(ctx, client, status.StatusForTeam)
	if err != nil {
		return nil, err
	}
	entTeam, err := client.Team.Create().
		SetTeamNumber(teamNumber).
		SetTeamToBuild(entBuild).
		SetTeamToStatus(entStatus).
		Save(ctx)
	if err != nil {
		log.Fatalf("Failed to create Team Number %v for Build %v. Err: %v", teamNumber, entBuild.ID, err)
		return nil, err
	}
	buildPlanNode, err := entBuild.QueryBuildToPlan().Where(plan.StepNumberEQ(0)).Only(ctx)
	if err != nil {
		log.Fatalf("Failed to Query Plan Node for Build %v. Err: %v", entBuild.ID, err)
		return nil, err
	}
	_, err = client.Plan.Create().
		AddPrevPlan(buildPlanNode).
		SetType(plan.TypeStartTeam).
		SetBuildID(entBuild.ID).
		SetPlanToTeam(entTeam).
		SetPlanToBuild(entBuild).
		SetStepNumber(1).
		Save(ctx)
	if err != nil {
		log.Fatalf("Failed to create Plan Node for Team %v. Err: %v", teamNumber, err)
		return nil, err
	}
	buildNetworks, err := entBuild.QueryBuildToEnvironment().QueryEnvironmentToNetwork().All(ctx)
	if err != nil {
		log.Fatalf("Failed to Query Enviroment for Build %v. Err: %v", entBuild.ID, err)
		return nil, err
	}
	createProvisonedNetworks := []*ent.ProvisionedNetwork{}
	for _, buildNetwork := range buildNetworks {
		pNetwork, _ := createProvisionedNetworks(ctx, client, entBuild, entTeam, buildNetwork)
		createProvisonedNetworks = append(createProvisonedNetworks, pNetwork)
	}
	for _, pNetwork := range createProvisonedNetworks {
		entHosts, err := pNetwork.
			QueryProvisionedNetworkToNetwork().
			QueryNetworkToIncludedNetwork().
			QueryIncludedNetworkToHost().
			All(ctx)
		if err != nil {
			log.Fatalf("Failed to Query Hosts for Network %v. Err: %v", pNetwork.Name, err)
			return nil, err
		}
		networkPlan, err := pNetwork.QueryProvisionedNetworkToPlan().Only(ctx)
		if err != nil {
			log.Fatalf("Failed to Query Plan for Network %v. Err: %v", pNetwork.Name, err)
			return nil, err
		}
		for _, entHost := range entHosts {
			createProvisionedHosts(ctx, client, pNetwork, entHost, networkPlan)
		}
	}
	return entTeam, nil
}

func createProvisionedNetworks(ctx context.Context, client *ent.Client, entBuild *ent.Build, entTeam *ent.Team, entNetwork *ent.Network) (*ent.ProvisionedNetwork, error) {

	entStatus, err := createPlanningStatus(ctx, client, status.StatusForProvisionedNetwork)
	if err != nil {
		return nil, err
	}

	entProvisionedNetwork, err := client.ProvisionedNetwork.Create().
		SetName(entNetwork.Name).
		SetCidr(entNetwork.Cidr).
		SetProvisionedNetworkToStatus(entStatus).
		SetProvisionedNetworkToNetwork(entNetwork).
		SetProvisionedNetworkToTeam(entTeam).
		SetProvisionedNetworkToBuild(entBuild).
		Save(ctx)
	if err != nil {
		log.Fatalf("Failed to create Provisoned Network %v for Team %v. Err: %v", entNetwork.Name, entTeam.TeamNumber, err)
		return nil, err
	}
	teamPlanNode, err := entTeam.QueryTeamToPlan().Only(ctx)
	if err != nil {
		log.Fatalf("Failed to Query Plan Node for Build %v. Err: %v", entBuild.ID, err)
		return nil, err
	}
	_, err = client.Plan.Create().
		AddPrevPlan(teamPlanNode).
		SetType(plan.TypeProvisionNetwork).
		SetBuildID(entBuild.ID).
		SetPlanToProvisionedNetwork(entProvisionedNetwork).
		SetPlanToBuild(entBuild).
		SetStepNumber(teamPlanNode.StepNumber + 1).
		Save(ctx)
	if err != nil {
		log.Fatalf("Failed to create Plan Node for Provisioned Network  %v. Err: %v", entProvisionedNetwork.Name, err)
		return nil, err
	}
	return entProvisionedNetwork, nil
}

func createProvisionedHosts(ctx context.Context, client *ent.Client, pNetwork *ent.ProvisionedNetwork, entHost *ent.Host, prevPlan *ent.Plan) (*ent.ProvisionedHost, error) {
	prevPlans := []*ent.Plan{prevPlan}
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
			log.Fatalf("Failed to Query Existing Host %v. Err: %v", entHost.HclID, err)
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
				log.Fatalf("Failed to Query Depended On Host %v for Host %v. Err: %v", entHostDependency.Edges.HostDependencyToDependOnHost.HclID, entHost.HclID, err)
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
					log.Fatalf("Failed to Query Provined Network %v for Depended On Host %v. Err: %v", entHostDependency.Edges.HostDependencyToNetwork.HclID, entHostDependency.Edges.HostDependencyToDependOnHost.HclID, err)
				}
				entDependsOnHost, err = createProvisionedHosts(ctx, client, dependOnPnetwork, entHostDependency.Edges.HostDependencyToDependOnHost, prevPlan)
			}
		}
		dependOnPlan, err := entDependsOnHost.QueryProvisionedHostToEndStepPlan().Only(ctx)
		if err != nil && err != err.(*ent.NotFoundError) {
			log.Fatalf("Failed to Query Depended On Host %v Plan for Host %v. Err: %v", entHostDependency.Edges.HostDependencyToDependOnHost.HclID, entHost.HclID, err)
			return nil, err
		}
		prevPlans = append(prevPlans, dependOnPlan)
		if planStepNumber <= dependOnPlan.StepNumber {
			planStepNumber = dependOnPlan.StepNumber + 1
		}

	}

	subnetIP, err := calcIP(pNetwork.Cidr, entHost.LastOctet)
	if err != nil {
		return nil, err
	}

	entStatus, err := createPlanningStatus(ctx, client, status.StatusForProvisionedHost)
	if err != nil {
		return nil, err
	}

	entProvisionedHost, err = client.ProvisionedHost.Create().
		SetSubnetIP(subnetIP).
		SetProvisionedHostToStatus(entStatus).
		SetProvisionedHostToProvisionedNetwork(pNetwork).
		SetProvisionedHostToHost(entHost).
		Save(ctx)

	endPlanNode, err := client.Plan.Create().
		AddPrevPlan(prevPlans...).
		SetType(plan.TypeProvisionHost).
		SetBuildID(prevPlan.BuildID).
		SetPlanToProvisionedHost(entProvisionedHost).
		SetStepNumber(planStepNumber).
		SetPlanToBuild(currentBuild).
		Save(ctx)

	if err != nil {
		log.Fatalf("Failed to create Plan Node for Provisioned Host  %v. Err: %v", entHost.HclID, err)
		return nil, err
	}

	serverAddress, ok := os.LookupEnv("GRPC_SERVER")
	if !ok {
		serverAddress = "localhost:50051"
	}
	isWindowsHost := false
	if strings.Contains(entHost.OS, "w2k") {
		isWindowsHost = true
	}

	binaryPath := path.Join(currentBuild.Edges.BuildToEnvironment.Name, fmt.Sprint(currentBuild.Revision), fmt.Sprint(currentTeam.TeamNumber), pNetwork.Name, entHost.Hostname)
	os.MkdirAll(binaryPath, 0755)
	binaryName := path.Join(binaryPath, "laforgeAgent")
	if isWindowsHost {
		binaryName = binaryName + ".exe"
	}
	binaryName, err = filepath.Abs(binaryName)
	if err != nil {
		log.Fatalf("Unable to Resolve Absolute File Path. Err: %v", err)
		return nil, err
	}
	if RenderFiles {
		grpc.BuildAgent(fmt.Sprint(entProvisionedHost.ID), serverAddress, binaryName, isWindowsHost)
		entTmpUrl, err := utils.CreateTempURL(ctx, client, binaryName)
		if err != nil {
			return nil, err
		}
		_, err = entTmpUrl.Update().SetGinFileMiddlewareToProvisionedHost(entProvisionedHost).Save(ctx)
		if err != nil {
			return nil, err
		}
	}
	userDataScriptID, ok := entHost.Vars["user_data_script_id"]
	if ok {
		userDataScript, err := client.Script.Query().Where(script.HclIDEQ(userDataScriptID)).Only(ctx)
		if err != nil {
			log.Fatalf("Failed to Query Script %v. Err: %v", userDataScriptID, err)
			return nil, err
		}
		entUserDataProvisioningStep, err := client.ProvisioningStep.Create().
			SetStepNumber(0).
			SetType(provisioningstep.TypeScript).
			SetProvisioningStepToScript(userDataScript).
			SetProvisioningStepToProvisionedHost(entProvisionedHost).
			Save(ctx)
		if err != nil {
			log.Fatalf("Failed to Create Provisioning Step for Script %v. Err: %v", userDataScriptID, err)
			return nil, err
		}
		if RenderFiles {
			filePath, err := renderScript(ctx, client, entUserDataProvisioningStep)
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
		}

	}

	for stepNumber, pStep := range entHost.ProvisionSteps {
		stepNumber = stepNumber + 1
		entProvisioningStep, err := createProvisioningStep(ctx, client, pStep, stepNumber, entProvisionedHost, endPlanNode)
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
		log.Fatalf("Unable to Update The End Step. Err: %v", err)
		return nil, err
	}

	return entProvisionedHost, nil
}

func createProvisioningStep(ctx context.Context, client *ent.Client, hclID string, stepNumber int, pHost *ent.ProvisionedHost, prevPlan *ent.Plan) (*ent.ProvisioningStep, error) {
	var entProvisioningStep *ent.ProvisioningStep
	currentEnviroment, err := pHost.QueryProvisionedHostToHost().QueryHostToEnvironment().Only(ctx)
	currentBuild := pHost.QueryProvisionedHostToProvisionedNetwork().QueryProvisionedNetworkToBuild().WithBuildToEnvironment().OnlyX(ctx)
	if err != nil {
		log.Fatalf("Failed to Query Current Enviroment for Provisoned Host %v. Err: %v", pHost.ID, err)
		return nil, err
	}
	entStatus, err := createPlanningStatus(ctx, client, status.StatusForProvisionedHost)
	if err != nil {
		return nil, err
	}
	entScript, err := client.Script.Query().Where(
		script.And(
			script.HasScriptToEnvironmentWith(
				environment.IDEQ(currentEnviroment.ID),
			),
			script.HclIDEQ(hclID),
		),
	).Only(ctx)
	if err != nil {
		if err != err.(*ent.NotFoundError) {
			log.Fatalf("Failed to Query Script %v. Err: %v", hclID, err)
			return nil, err
		} else {
			entCommand, err := client.Command.Query().Where(
				command.And(
					command.HasCommandToEnvironmentWith(
						environment.IDEQ(currentEnviroment.ID),
					),
					command.HclIDEQ(hclID),
				)).Only(ctx)
			if err != nil {
				if err != err.(*ent.NotFoundError) {
					log.Fatalf("Failed to Query Command %v. Err: %v", hclID, err)
					return nil, err
				} else {
					entFileDownload, err := client.FileDownload.Query().Where(
						filedownload.And(
							filedownload.HasFileDownloadToEnvironmentWith(
								environment.IDEQ(currentEnviroment.ID),
							),
							filedownload.HclIDEQ(hclID),
						)).Only(ctx)
					if err != nil {
						if err != err.(*ent.NotFoundError) {
							log.Fatalf("Failed to Query FileDownload %v. Err: %v", hclID, err)
							return nil, err
						} else {
							entFileExtract, err := client.FileExtract.Query().Where(
								fileextract.And(
									fileextract.HasFileExtractToEnvironmentWith(
										environment.IDEQ(currentEnviroment.ID),
									),
									fileextract.HclIDEQ(hclID),
								)).Only(ctx)
							if err != nil {
								if err != err.(*ent.NotFoundError) {
									log.Fatalf("Failed to Query FileExtract %v. Err: %v", hclID, err)
									return nil, err
								} else {
									entFileDelete, err := client.FileDelete.Query().Where(
										filedelete.And(
											filedelete.HasFileDeleteToEnvironmentWith(
												environment.IDEQ(currentEnviroment.ID),
											),
											filedelete.HclIDEQ(hclID),
										)).Only(ctx)
									if err != nil {
										if err != err.(*ent.NotFoundError) {
											log.Fatalf("Failed to Query FileDelete %v. Err: %v", hclID, err)
											return nil, err
										} else {
											entDNSRecord, err := client.DNSRecord.Query().Where(
												dnsrecord.And(
													dnsrecord.HasDNSRecordToEnvironmentWith(
														environment.IDEQ(currentEnviroment.ID),
													),
													dnsrecord.HclIDEQ(hclID),
												)).Only(ctx)
											if err != nil {
												if err != err.(*ent.NotFoundError) {
													log.Fatalf("Failed to Query FileDelete %v. Err: %v", hclID, err)
													return nil, err
												} else {
													log.Fatalf("No Provisioning Steps found for %v. Err: %v", hclID, err)
													return nil, err
												}
											} else {
												entProvisioningStep, err = client.ProvisioningStep.Create().
													SetStepNumber(stepNumber).
													SetType(provisioningstep.TypeDNSRecord).SetProvisioningStepToDNSRecord(entDNSRecord).
													SetProvisioningStepToStatus(entStatus).
													SetProvisioningStepToProvisionedHost(pHost).
													Save(ctx)
												if err != nil {
													log.Fatalf("Failed to Create Provisioning Step for FileDelete %v. Err: %v", hclID, err)
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
											log.Fatalf("Failed to Create Provisioning Step for FileDelete %v. Err: %v", hclID, err)
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
									log.Fatalf("Failed to Create Provisioning Step for FileExtract %v. Err: %v", hclID, err)
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
							log.Fatalf("Failed to Create Provisioning Step for FileDownload %v. Err: %v", hclID, err)
							return nil, err
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
					log.Fatalf("Failed to Create Provisioning Step for Command %v. Err: %v", hclID, err)
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
			log.Fatalf("Failed to Create Provisioning Step for Script %v. Err: %v", hclID, err)
			return nil, err
		}
		if RenderFiles {
			filePath, err := renderScript(ctx, client, entProvisioningStep)
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
		}
	}

	_, err = client.Plan.Create().
		AddPrevPlan(prevPlan).
		SetType(plan.TypeExecuteStep).
		SetBuildID(prevPlan.BuildID).
		SetPlanToProvisioningStep(entProvisioningStep).
		SetStepNumber(prevPlan.StepNumber + 1).
		SetPlanToBuild(currentBuild).
		Save(ctx)

	if err != nil {
		log.Fatalf("Failed to Create Plan Node for Provisioning Step %v. Err: %v", entProvisioningStep.ID, err)
		return nil, err
	}

	return entProvisioningStep, nil

}

func renderScript(ctx context.Context, client *ent.Client, pStep *ent.ProvisioningStep) (string, error) {
	currentProvisionedHost := pStep.QueryProvisioningStepToProvisionedHost().OnlyX(ctx)
	currentScript := pStep.QueryProvisioningStepToScript().OnlyX(ctx)
	currentProvisionedNetwork := currentProvisionedHost.QueryProvisionedHostToProvisionedNetwork().OnlyX(ctx)
	currentTeam := currentProvisionedNetwork.QueryProvisionedNetworkToTeam().OnlyX(ctx)
	currentBuild := currentTeam.QueryTeamToBuild().OnlyX(ctx)
	currentEnvironment := currentBuild.QueryBuildToEnvironment().OnlyX(ctx)
	currentCompetition := currentBuild.QueryBuildToCompetition().OnlyX(ctx)
	currentNetwork := currentProvisionedNetwork.QueryProvisionedNetworkToNetwork().OnlyX(ctx)
	currentHost := currentProvisionedHost.QueryProvisionedHostToHost().OnlyX(ctx)
	// Need to Make Unique and change how it's loaded in
	currentDNS := currentCompetition.QueryCompetitionToDNS().FirstX(ctx)
	templeteData := TempleteContext{
		Build:              currentBuild,
		Competition:        currentCompetition,
		Environment:        currentEnvironment,
		Host:               currentHost,
		DNS:                currentDNS,
		Network:            currentNetwork,
		Script:             currentScript,
		Team:               currentTeam,
		ProvisionedNetwork: currentProvisionedNetwork,
		ProvisionedHost:    currentProvisionedHost,
		ProvisioningStep:   pStep,
	}
	t, err := template.ParseFiles(currentScript.AbsPath)
	if err != nil {
		log.Fatalf("Failed to Parse templete for script %v. Err: %v", currentScript.Name, err)
		return "", err
	}
	t.Funcs(TemplateFuncLib)
	fileRelativePath := path.Join(currentEnvironment.Name, fmt.Sprint(currentBuild.Revision), fmt.Sprint(currentTeam.TeamNumber), currentProvisionedNetwork.Name, currentHost.Hostname)
	os.MkdirAll(fileRelativePath, 0755)
	fileName := filepath.Base(currentScript.Source)
	fileName = path.Join(fileRelativePath, fileName)
	fileName, err = filepath.Abs(fileName)
	if err != nil {
		return "", err
	}
	f, err := os.Create(fileName)
	if err != nil {
		log.Fatalf("Error Generating Script %v. Err: %v", currentScript.Name, err)
		return "", err
	}
	err = t.Execute(f, templeteData)
	f.Close()
	return fileName, nil
}

// CalcIP is used to calculate the IP of a host within a given subnet
func calcIP(subnet string, lastOctect int) (string, error) {
	ip, _, err := net.ParseCIDR(subnet)
	if err != nil {
		log.Fatalf("Invalid Subner %v. Err: %v", subnet, err)
		return "", err
	}
	offset32 := uint32(lastOctect)
	ip32 := IPv42Int(ip)
	newIP := Int2IPv4(ip32 + offset32)
	return newIP.To4().String(), nil
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