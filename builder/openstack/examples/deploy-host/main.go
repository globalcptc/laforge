package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/gen0cide/laforge/builder"
	"github.com/gen0cide/laforge/ent"
	"github.com/gen0cide/laforge/logging"
	"github.com/gen0cide/laforge/server/utils"
	"github.com/sirupsen/logrus"
)

const (
	CONFIG_FILE = "./configs/test_openstack.json"
	LOG_FILE    = "./test_openstack.log"
)

func main() {
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

	defaultLogger := logging.CreateNewLogger(LOG_FILE)

	env, err := client.Environment.Query().Only(ctx)
	if err != nil {
		log.Fatalf("error querying env: %v", err)
	}

	fmt.Println("Creating Openstack builder...")
	_, err = builder.NewOpenstackBuilder(CONFIG_FILE, env, &defaultLogger)
	if err != nil {
		defaultLogger.Log.Errorf("failed to create Openstack builder: %v", err)
		os.Exit(1)
	}

	build, err := env.QueryEnvironmentToBuild().Only(ctx)
	if err != nil {
		log.Fatalf("error querying build from env: %v", err)
	}
	_, err = build.QueryBuildToTeam().All(ctx)
	if err != nil {
		log.Fatalf("error querying teams from build: %v", err)
	}
}
