package planner_test

import (
	"context"
	"io"
	"path/filepath"
	"testing"

	"github.com/gen0cide/laforge/ent"
	_ "github.com/gen0cide/laforge/ent/runtime"
	"github.com/gen0cide/laforge/loader"
	"github.com/gen0cide/laforge/logging"
	"github.com/gen0cide/laforge/planner"
	"github.com/gen0cide/laforge/server/utils"
	"github.com/sirupsen/logrus"
)

func TestValidateBuild(t *testing.T) {
	ctx := context.Background()
	client := ent.SQLLiteOpen("file:validate-build-test?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() {
		client.Close()
	})
	if err := client.Schema.Create(ctx); err != nil {
		t.Fatalf("create schema: %v", err)
	}

	testLog := logrus.New()
	testLog.SetOutput(io.Discard)
	logger := &logging.Logger{Log: testLog}
	envPath := filepath.Join("..", "cmd", "laforge-test", "testdata", "minimal.laforge")
	environments, err := loader.LoadEnvironment(ctx, client, logger, envPath)
	if err != nil {
		t.Fatalf("load environment: %v", err)
	}
	if len(environments) != 1 {
		t.Fatalf("got %d environments, want 1", len(environments))
	}

	summary, err := planner.ValidateBuild(
		ctx,
		client,
		&utils.ServerConfig{},
		logger,
		environments[0],
	)
	if err != nil {
		t.Fatalf("validate build: %v", err)
	}
	if summary.Teams != 1 || summary.Networks != 1 || summary.Hosts != 1 || summary.Plans != 4 {
		t.Fatalf("unexpected summary: %+v", summary)
	}
}
