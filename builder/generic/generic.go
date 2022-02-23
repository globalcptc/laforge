package generic

import (
	"context"

	"github.com/gen0cide/laforge/ent"
	"github.com/gen0cide/laforge/logging"
)

const (
	ID          = "generic"
	Name        = "Generic"
	Description = "Builder that interfaces with Nothing"
	Author      = "Fred Rybin <github.com/frybin>"
	Version     = "0.1"
)

type GenericBuilder struct {
	Logger *logging.Logger
}

func (builder GenericBuilder) ID() string {
	return ID
}

func (builder GenericBuilder) Name() string {
	return Name
}

func (builder GenericBuilder) Description() string {
	return Description
}

func (builder GenericBuilder) Author() string {
	return Author
}

func (builder GenericBuilder) Version() string {
	return Version
}

func (builder GenericBuilder) DeployHost(ctx context.Context, provisionedHost *ent.ProvisionedHost) (err error) {
	return
}

func (builder GenericBuilder) DeployNetwork(ctx context.Context, provisionedNetwork *ent.ProvisionedNetwork) (err error) {
	return
}

func (builder GenericBuilder) DeployTeam(ctx context.Context, entTeam *ent.Team) (err error) {
	return
}

func (builder GenericBuilder) TeardownHost(ctx context.Context, provisionedHost *ent.ProvisionedHost) (err error) {
	return
}

func (builder GenericBuilder) TeardownNetwork(ctx context.Context, provisionedNetwork *ent.ProvisionedNetwork) (err error) {
	return
}

func (builder GenericBuilder) TeardownTeam(ctx context.Context, entTeam *ent.Team) (err error) {
	return
}
