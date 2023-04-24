package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// Plan holds the schema definition for the Plan entity.
type Plan struct {
	ent.Schema
}

// Fields of the Plan.
func (Plan) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New),
		field.Int("step_number"),
		field.Enum("type").
			Values(
				"start_build",
				"start_team",
				"provision_network",
				"provision_host",
				"execute_step",
				"start_scheduled_step",
			),
		field.String("build_id"),
	}
}

// Edges of the Plan.
func (Plan) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("NextPlans", Plan.Type).
			From("PrevPlans").
			Annotations(entsql.Annotation{
				OnDelete: entsql.Cascade,
			}),
		edge.To("Build", Build.Type).
			Unique(),
		edge.To("Team", Team.Type).
			Unique(),
		edge.To("ProvisionedNetwork", ProvisionedNetwork.Type).
			Unique(),
		edge.To("ProvisionedHost", ProvisionedHost.Type).
			Unique(),
		edge.To("ProvisioningStep", ProvisioningStep.Type).
			Unique(),
		edge.To("ProvisioningScheduledStep", ProvisioningScheduledStep.Type).
			Unique(),
		edge.To("Status", Status.Type).
			Unique().
			Required().
			Annotations(entsql.Annotation{
				OnDelete: entsql.Cascade,
			}),
		edge.From("PlanDiffs", PlanDiff.Type).
			Ref("PlanDiffToPlan"),
	}
}
