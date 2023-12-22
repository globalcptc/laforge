package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// Status holds the schema definition for the Status entity.
type Status struct {
	ent.Schema
}

// Fields of the Status.
func (Status) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New),
		field.Enum("state").Values("PLANNING", "AWAITING", "PARENTAWAITING", "INPROGRESS", "FAILED", "COMPLETE", "TAINTED", "TODELETE", "DELETEINPROGRESS", "DELETED", "TOREBUILD", "CANCELLED"),
		field.Enum("status_for").Values("Build", "Team", "Plan", "ProvisionedNetwork", "ProvisionedHost", "ProvisioningStep", "ProvisioningScheduledStep", "ServerTask"),
		field.Time("started_at").Optional(),
		field.Time("ended_at").Optional(),
		field.Bool("failed").Default(false),
		field.Bool("completed").Default(false),
		field.String("error").Optional(),
	}
}

// Edges of the Status.
func (Status) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("Build", Build.Type).
			Ref("Status").
			Unique(),
		edge.From("ProvisionedNetwork", ProvisionedNetwork.Type).
			Ref("Status").
			Unique(),
		edge.From("ProvisionedHost", ProvisionedHost.Type).
			Ref("Status").
			Unique(),
		edge.From("ProvisioningStep", ProvisioningStep.Type).
			Ref("Status").
			Unique(),
		edge.From("Team", Team.Type).
			Ref("Status").
			Unique(),
		edge.From("Plan", Plan.Type).
			Ref("Status").
			Unique(),
		edge.From("ServerTask", ServerTask.Type).
			Ref("Status").
			Unique(),
		edge.From("AdhocPlan", AdhocPlan.Type).
			Ref("Status").
			Unique(),
		edge.From("ProvisioningScheduledStep", ProvisioningScheduledStep.Type).
			Ref("Status").
			Unique(),
	}
}
