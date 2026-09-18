package grpc

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/gen0cide/laforge/ent"
	"github.com/gen0cide/laforge/logging"
	"github.com/gen0cide/laforge/server/utils"
	"github.com/sirupsen/logrus"
)

func BuildAgent(logger *logging.Logger, agentID string, serverAddress string, caCertPath string, useSystemRoots bool, binarypath string, isWindows bool, agentLogging bool) error {
	goos := "linux"
	if isWindows {
		goos = "windows"
	}

	ldflags := fmt.Sprintf("-X main.clientID=%s -X main.address=%s", agentID, serverAddress)
	if useSystemRoots {
		ldflags += " -X main.useSystemRoots=true"
	} else if caCertPath != "" {
		caCert, err := os.ReadFile(caCertPath)
		if err != nil {
			return fmt.Errorf("failed to read agent gRPC CA certificate %q: %w", caCertPath, err)
		}
		ldflags += " -X main.caCertPEMBase64=" + base64.StdEncoding.EncodeToString(caCert)
	}

	args := []string{"build"}
	if agentLogging {
		args = append(args, "-tags", "debug")
	}
	args = append(args, "-ldflags", ldflags, "-o", binarypath, "github.com/gen0cide/laforge/grpc/agent")

	logger.Log.Debugf("Building %s agent %s for %s", goos, agentID, serverAddress)
	cmd := exec.Command("go", args...)
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0", "GOOS="+goos, "GOARCH=amd64")
	stdoutStderr, err := cmd.CombinedOutput()
	if err != nil {
		logger.Log.Errorf("Agent for %s failed to create: %s", agentID, stdoutStderr)
		return fmt.Errorf("failed to build agent %s: %w", agentID, err)
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
		BuildAgent(&logger, fmt.Sprint(ph.ID), laforgeConfig.Agent.GrpcServerUri, laforgeConfig.Agent.GrpcCACertPath, laforgeConfig.Agent.GrpcUseSystemRoots, binaryName, false, laforgeConfig.AgentDebug)
	}
}
