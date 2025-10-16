package grpc

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/gen0cide/laforge/ent"
	"github.com/gen0cide/laforge/logging"
	"github.com/gen0cide/laforge/server/utils"
	"github.com/sirupsen/logrus"
)

func BuildAgent(logger *logging.Logger, agentID string, serverAddress string, binarypath string, isWindows bool, agentLogging bool) error {
	command := ""
	agentDebugString := ""
	if agentLogging {
		agentDebugString = " -tags debug "
	}

	if isWindows {
		command = "CGO_ENABLED=1 GOOS=windows GOARCH=amd64 CC=\"zcc\" go build" + agentDebugString + " -ldflags=\" -X 'main.clientID=" + agentID + "' -X 'main.address=" + serverAddress + "'\" -o " + binarypath + " github.com/gen0cide/laforge/grpc/agent"
	} else {
		command = "CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build" + agentDebugString + " -ldflags=\" -X 'main.clientID=" + agentID + "' -X 'main.address=" + serverAddress + "'\" -o " + binarypath + " github.com/gen0cide/laforge/grpc/agent"
	}
	logger.Log.Debugf("Executing command to build agent: %s", command)
	cmd := exec.Command("bash", "-c", command)
	stdoutStderr, err := cmd.CombinedOutput()
	cmd.Run()
	if err != nil {
		logger.Log.Errorf("Agent for %s failed to create: %v", agentID, stdoutStderr)
		return err
	}
	logger.Log.Debugf("Created %s, Output %s\n", binarypath, stdoutStderr)
	return nil
}

func main() {
	laforgeConfig, err := utils.LoadServerConfig()
	if err != nil {
		logrus.Errorf("failed to load LaForge Config: %v", err)
		return
	}

	client := &ent.Client{}

	ctx := context.Background()
	defer client.Close()

	// Run the auto migration tool.
	if err := client.Schema.Create(ctx); err != nil {
		logrus.Errorf("failed creating schema resources: %v", err)
	}

	phs, err := client.ProvisionedHost.Query().All(ctx)
	if err != nil {
		logrus.Errorf("Failed to Query All Provisioned Hosts: %v", err)
	}

	for _, ph := range phs {
		host, err := ph.QueryHost().Only(ctx)
		if err != nil {
			logrus.Errorf("Failed to Query Host: %v", err)
		}
		hostName := host.Hostname

		switch runtime.GOOS {
		case "windows":
			if !strings.Contains(host.OS, "w2k") {
				continue
			}
		case "linux":
			if strings.Contains(host.OS, "w2k") {
				continue
			}
		}

		pn, err := ph.QueryProvisionedNetwork().Only(ctx)
		if err != nil {
			logrus.Errorf("Failed to Query Provisioned Network: %v", err)
		}
		networkName := pn.Name

		team, err := pn.QueryTeam().Only(ctx)
		if err != nil {
			logrus.Errorf("Failed to Query Team: %v", err)
		}
		teamName := team.TeamNumber
		env, err := team.QueryBuild().QueryEnvironment().Only(ctx)
		if err != nil {
			logrus.Errorf("Failed to Query Enviroment: %v", err)
		}
		envName := env.Name

		logger := logging.Logger{
			Log:     &logrus.Logger{},
			LogFile: "",
		}

		binaryName := filepath.Join(envName, "team", fmt.Sprint(teamName), networkName, hostName)
		BuildAgent(&logger, fmt.Sprint(ph.ID), laforgeConfig.Agent.GrpcServerUri, binaryName, false, laforgeConfig.AgentDebug)
	}
}
