package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"

	"github.com/gen0cide/laforge/ent"
	_ "github.com/gen0cide/laforge/ent/runtime"
	"github.com/gen0cide/laforge/loader"
	"github.com/gen0cide/laforge/logging"
	"github.com/gen0cide/laforge/planner"
	"github.com/gen0cide/laforge/server/utils"
	"github.com/sirupsen/logrus"
)

var errUsage = errors.New("expected exactly one environment file")

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "laforge-test: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	flags := flag.NewFlagSet("laforge-test", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	jsonOutput := flags.Bool("json", false, "write the validation summary as JSON")
	verbose := flags.Bool("verbose", false, "enable detailed loader and planner logs")
	flags.Usage = func() {
		fmt.Fprintln(flags.Output(), "Usage: laforge-test [options] <environment.laforge|environment.json>")
		flags.PrintDefaults()
	}
	if err := flags.Parse(os.Args[1:]); err != nil {
		return err
	}
	if flags.NArg() != 1 {
		flags.Usage()
		return errUsage
	}

	envPath, err := filepath.Abs(flags.Arg(0))
	if err != nil {
		return fmt.Errorf("resolve environment path: %w", err)
	}
	info, err := os.Stat(envPath)
	if err != nil {
		return fmt.Errorf("open environment file %q: %w", envPath, err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("environment path %q is not a regular file", envPath)
	}

	level := logrus.WarnLevel
	if *verbose {
		level = logrus.DebugLevel
	}
	logrus.SetLevel(level)
	commandLogger := logrus.New()
	commandLogger.SetLevel(level)
	commandLogger.SetOutput(os.Stderr)
	commandLogger.SetFormatter(&logrus.TextFormatter{
		DisableTimestamp: true,
	})
	logger := &logging.Logger{Log: commandLogger}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	client := ent.SQLLiteOpen("file:laforge-test?mode=memory&cache=shared&_fk=1")
	if client == nil {
		return errors.New("open temporary SQLite database")
	}
	defer client.Close()
	if err := client.Schema.Create(ctx); err != nil {
		return fmt.Errorf("create temporary database schema: %w", err)
	}

	environments, err := loader.LoadEnvironment(ctx, client, logger, envPath)
	if err != nil {
		return fmt.Errorf("load environment: %w", err)
	}
	if len(environments) == 0 {
		return errors.New("environment file did not define any environments")
	}

	config := &utils.ServerConfig{}
	summaries := make([]*planner.ValidationSummary, 0, len(environments))
	for _, environment := range environments {
		summary, err := planner.ValidateBuild(ctx, client, config, logger, environment)
		if err != nil {
			return fmt.Errorf("validate environment %q: %w", environment.HCLID, err)
		}
		summaries = append(summaries, summary)
	}

	if *jsonOutput {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(summaries)
	}
	for _, summary := range summaries {
		fmt.Printf(
			"PASS %s: %d teams, %d networks, %d hosts, %d provisioning steps, %d scheduled steps, %d plans\n",
			summary.Environment,
			summary.Teams,
			summary.Networks,
			summary.Hosts,
			summary.ProvisioningSteps,
			summary.ScheduledSteps,
			summary.Plans,
		)
	}

	return nil
}
