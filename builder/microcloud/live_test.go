package microcloud

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	lxd "github.com/canonical/lxd/client"
	"github.com/canonical/lxd/shared/api"
	"github.com/gen0cide/laforge/ent"
	"github.com/gen0cide/laforge/ent/enttest"
	"github.com/gen0cide/laforge/logging"
)

func TestLiveConnection(t *testing.T) {
	configPath := os.Getenv("LAFORGE_MICROCLOUD_LIVE_CONFIG")
	if configPath == "" {
		t.Skip("LAFORGE_MICROCLOUD_LIVE_CONFIG is not set")
	}

	configBytes, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read live config: %v", err)
	}

	var config BuilderConfig
	if err := json.Unmarshal(configBytes, &config); err != nil {
		t.Fatalf("decode live config: %v", err)
	}

	logger := logging.CreateNewLogger(filepath.Join(t.TempDir(), "microcloud-live.log"))
	builder, err := New(config, &logger)
	if err != nil {
		t.Fatalf("connect to MicroCloud: %v", err)
	}
	if len(builder.clusterMembers) == 0 {
		t.Fatal("MicroCloud returned no cluster members")
	}

	t.Logf("connected to %d MicroCloud members", len(builder.clusterMembers))
}

func TestLiveOverlappingNetworks(t *testing.T) {
	configPath := os.Getenv("LAFORGE_MICROCLOUD_LIVE_CONFIG")
	if configPath == "" {
		t.Skip("LAFORGE_MICROCLOUD_LIVE_CONFIG is not set")
	}

	configBytes, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read live config: %v", err)
	}
	var config BuilderConfig
	if err := json.Unmarshal(configBytes, &config); err != nil {
		t.Fatalf("decode live config: %v", err)
	}

	logger := logging.CreateNewLogger(filepath.Join(t.TempDir(), "microcloud-live.log"))
	builder, err := New(config, &logger)
	if err != nil {
		t.Fatalf("connect to MicroCloud: %v", err)
	}

	ctx := context.Background()
	db := enttest.Open(t, dialect.SQLite, "file:microcloud-live?mode=memory&cache=shared&_fk=1")
	environment := db.Environment.Create().
		SetHCLID("microcloud-live").
		SetCompetitionID("microcloud-live").
		SetName("MicroCloud Live Test").
		SetDescription("Ephemeral MicroCloud builder integration test").
		SetBuilder("microcloud-test").
		SetTeamCount(2).
		SetRevision(1).
		SetAdminCidrs([]string{}).
		SetExposedVdiPorts([]string{}).
		SetConfig(map[string]string{}).
		SetTags(map[string]string{}).
		SaveX(ctx)
	competition := db.Competition.Create().
		SetHCLID("microcloud-live").
		SetRootPassword("not-used").
		SetConfig(map[string]string{}).
		SetTags(map[string]string{}).
		SetEnvironment(environment).
		SaveX(ctx)
	build := db.Build.Create().
		SetRevision(1).
		SetEnvironmentRevision(1).
		SetVars(map[string]string{}).
		SetEnvironment(environment).
		SetCompetition(competition).
		SaveX(ctx)
	networks := []*struct {
		name string
		cidr string
	}{
		{name: "vdi", cidr: "10.50.10.0/24"},
		{name: "competition", cidr: "10.50.20.0/24"},
	}

	for teamNumber := 1; teamNumber <= 2; teamNumber++ {
		team := db.Team.Create().
			SetTeamNumber(teamNumber).
			SetVars(map[string]string{}).
			SetBuild(build).
			SaveX(ctx)
		if err := builder.DeployTeam(ctx, team); err != nil {
			t.Fatalf("deploy team %d: %v", teamNumber, err)
		}
		t.Cleanup(func() {
			if err := builder.TeardownTeam(ctx, team); err != nil {
				t.Errorf("teardown team %d: %v", teamNumber, err)
			}
		})

		provisionedNetworks := make([]provisionedNetworkRef, 0, len(networks))
		for _, spec := range networks {
			network := db.Network.Create().
				SetHCLID(spec.name).
				SetName(spec.name).
				SetCidr(spec.cidr).
				SetVdiVisible(spec.name == "vdi").
				SetVars(map[string]string{}).
				SetTags(map[string]string{}).
				SetEnvironment(environment).
				SaveX(ctx)
			provisionedNetwork := db.ProvisionedNetwork.Create().
				SetName(spec.name).
				SetCidr(spec.cidr).
				SetVars(map[string]string{}).
				SetNetwork(network).
				SetBuild(build).
				SetTeam(team).
				SaveX(ctx)
			t.Cleanup(func() {
				if err := builder.TeardownNetwork(ctx, provisionedNetwork); err != nil {
					t.Errorf(
						"cleanup team %d network %s: %v",
						teamNumber,
						spec.name,
						err,
					)
				}
			})
			if err := builder.DeployNetwork(ctx, provisionedNetwork); err != nil {
				t.Fatalf("deploy team %d network %s: %v", teamNumber, spec.name, err)
			}
			provisionedNetworks = append(provisionedNetworks, provisionedNetworkRef{
				entity: provisionedNetwork,
				name:   provisionedNetwork.Vars["lxd_network"],
			})
		}

		projectClient := builder.Client.UseProject(team.Vars["lxd_project"])
		for _, network := range provisionedNetworks {
			peers, err := projectClient.GetNetworkPeers(network.name)
			if err != nil {
				t.Fatalf("list peers for team %d network %s: %v", teamNumber, network.name, err)
			}
			if len(peers) != 1 {
				t.Fatalf(
					"team %d network %s has %d peers, want 1",
					teamNumber,
					network.name,
					len(peers),
				)
			}
		}

		if teamNumber == 1 {
			disk := db.Disk.Create().
				SetSize(10).
				SaveX(ctx)
			host := db.Host.Create().
				SetHCLID("ubuntu").
				SetHostname("ubuntu").
				SetDescription("MicroCloud live test VM").
				SetOS("ubuntu22").
				SetLastOctet(10).
				SetInstanceSize("nano").
				SetAllowMACChanges(false).
				SetExposedTCPPorts([]string{"22"}).
				SetExposedUDPPorts([]string{}).
				SetOverridePassword("").
				SetVars(map[string]string{"public_ingress": "true"}).
				SetUserGroups([]string{}).
				SetProvisionSteps([]string{}).
				SetScheduledSteps([]string{}).
				SetTags(map[string]string{}).
				SetDisk(disk).
				SetEnvironment(environment).
				SaveX(ctx)
			status := db.Status.Create().
				SetState("AWAITING").
				SetStatusFor("ProvisionedHost").
				SaveX(ctx)
			provisionedHost := db.ProvisionedHost.Create().
				SetSubnetIP("10.50.10.10").
				SetVars(map[string]string{}).
				SetStatus(status).
				SetProvisionedNetwork(provisionedNetworks[0].entity).
				SetHost(host).
				SetBuild(build).
				SaveX(ctx)
			db.GinFileMiddleware.Create().
				SetURLID("microcloud-live-agent").
				SetFilePath("/tmp/microcloud-live-agent").
				SetAccessed(false).
				SetProvisionedHost(provisionedHost).
				SaveX(ctx)
			t.Cleanup(func() {
				if err := builder.TeardownHost(ctx, provisionedHost); err != nil {
					t.Errorf("cleanup live test host: %v", err)
				}
			})

			if err := builder.DeployHost(ctx, provisionedHost); err != nil {
				t.Fatalf("deploy live test host: %v", err)
			}
			instanceName := provisionedHost.Vars["lxd_instance"]
			instance, _, err := projectClient.GetInstance(instanceName)
			if err != nil {
				t.Fatalf("get live test instance %q: %v", instanceName, err)
			}
			if !instance.IsActive() {
				t.Fatalf("live test instance %q is not active: %s", instanceName, instance.Status)
			}
			if err := waitForInstanceHTTP(
				ctx,
				projectClient,
				instanceName,
				config.LaForgeServerURL,
				2*time.Minute,
			); err != nil {
				t.Fatalf("verify VM access to LaForge: %v", err)
			}
			forwardAddress := provisionedHost.Vars["lxd_forward_address"]
			if forwardAddress == "" {
				t.Fatal("live test instance has no allocated ingress forward")
			}
			forward, _, err := projectClient.GetNetworkForward(
				provisionedNetworks[0].name,
				forwardAddress,
			)
			if err != nil {
				t.Fatalf("get ingress forward %q: %v", forwardAddress, err)
			}
			if len(forward.Ports) != 1 ||
				forward.Ports[0].Protocol != "tcp" ||
				forward.Ports[0].ListenPort != "22" ||
				forward.Ports[0].TargetAddress != provisionedHost.SubnetIP {
				t.Fatalf("unexpected ingress forward ports: %#v", forward.Ports)
			}
			if err := builder.TeardownHost(ctx, provisionedHost); err != nil {
				t.Fatalf("teardown live test host: %v", err)
			}
		}

		for i := len(provisionedNetworks) - 1; i >= 0; i-- {
			if err := builder.TeardownNetwork(ctx, provisionedNetworks[i].entity); err != nil {
				t.Fatalf(
					"teardown team %d network %s: %v",
					teamNumber,
					provisionedNetworks[i].name,
					err,
				)
			}
		}
	}
}

type provisionedNetworkRef struct {
	entity *ent.ProvisionedNetwork
	name   string
}

func waitForInstanceHTTP(
	ctx context.Context,
	client lxd.InstanceServer,
	instanceName string,
	url string,
	timeout time.Duration,
) error {
	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		op, err := client.ExecInstance(instanceName, api.InstanceExecPost{
			Command:   []string{"curl", "-fsS", "--max-time", "5", url},
			WaitForWS: true,
		}, &lxd.InstanceExecArgs{
			Stdout:   &stdout,
			Stderr:   &stderr,
			DataDone: make(chan bool),
		})
		if err == nil {
			err = op.Wait()
			if err == nil {
				returnCode, _ := op.Get().Metadata["return"].(float64)
				if returnCode == 0 {
					return nil
				}
				err = fmt.Errorf("curl exit code %.0f: %s", returnCode, stderr.String())
			}
		}
		lastErr = err

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(5 * time.Second):
		}
	}

	return fmt.Errorf("instance %q could not reach %q: %w", instanceName, url, lastErr)
}
