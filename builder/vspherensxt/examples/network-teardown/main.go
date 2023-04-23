package main

// import (
// 	"context"
// 	"fmt"
// 	"log"
// 	"os"

// 	"github.com/gen0cide/laforge/builder"
// 	"github.com/gen0cide/laforge/ent"
// 	"github.com/gen0cide/laforge/ent/build"
// 	"github.com/gen0cide/laforge/ent/environment"
// 	"github.com/gen0cide/laforge/logging"
// 	"github.com/gen0cide/laforge/server/utils"
// 	"github.com/sirupsen/logrus"
// )

// func main() {

// 	laforgeConfig, err := utils.LoadServerConfig()
// 	if err != nil {
// 		logrus.Errorf("failed to load LaForge config: %v", err)
// 		return
// 	}

// 	if laforgeConfig.Database.PostgresUri == "" {
// 		logrus.Errorf("Database.PostgresUri not set in LaForge config")
// 		os.Exit(1)
// 	}

// 	client := ent.PGOpen(laforgeConfig.Database.PostgresUri)

// 	ctx := context.Background()
// 	defer ctx.Done()
// 	defer client.Close()

// 	// Run the auto migration tool.
// 	if err := client.Schema.Create(ctx); err != nil {
// 		log.Fatalf("failed creating schema resources: %v", err)
// 	}

// 	env, err := client.Environment.Query().Where(environment.NameEQ("fred")).Only(ctx)
// 	if err != nil {
// 		log.Fatalf("error querying env: %v", err)
// 	}

// 	defaultLogger := logging.CreateNewLogger("./output.lfglog")

// 	fmt.Println("Creating vSphere/NSX-T builder...")
// 	vsphereNsxt, err := builder.NewVSphereNSXTBuilder(env, &defaultLogger)
// 	if err != nil {
// 		log.Fatalf("error while creating vCenter/NSX-T builder: %v", err)
// 	}

// 	build, err := env.QueryBuilds().Order(ent.Desc(build.FieldRevision)).First(ctx)
// 	if err != nil {
// 		log.Fatalf("error querying build from env: %v", err)
// 	}
// 	teams, err := build.QueryTeams().All(ctx)
// 	if err != nil {
// 		log.Fatalf("error querying teams from build: %v", err)
// 	}

// 	for _, team := range teams {
// 		fmt.Printf("Networks for Team %d\n", team.TeamNumber)
// 		pnets, err := team.QueryTeamToProvisionedNetwork().All(ctx)
// 		if err != nil {
// 			log.Fatalf("error while querying provisioned netowrks from team: %v", err)
// 		}
// 		for _, pnet := range pnets {
// 			fmt.Printf("\t%s | %s\n", pnet.Name, pnet.Cidr)
// 			err := vsphereNsxt.TeardownNetwork(ctx, pnet)
// 			if err != nil {
// 				fmt.Printf("\tERROR: %v\n", err)
// 			} else {
// 				fmt.Println("\tOK")
// 			}
// 		}
// 	}
// }
