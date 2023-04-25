package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// ProvisionedHost holds the schema definition for the ProvisionedHost entity.
type ProvisionedHost struct {
	ent.Schema
}

// Fields of the ProvisionedHost.
func (ProvisionedHost) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New),
		field.String("subnet_ip"),
		field.Enum("addon_type").Values("DNS").Nillable().Optional(),
		field.JSON("vars", map[string]string{}),
	}
}

// Edges of the ProvisionedHost.
func (ProvisionedHost) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("Status", Status.Type).
			Required().
			Unique().
			Annotations(entsql.Annotation{
				OnDelete: entsql.Cascade,
			}),
		edge.To("ProvisionedNetwork", ProvisionedNetwork.Type).
			Required().
			Unique(),
		edge.To("Host", Host.Type).
			Required().
			Unique(),
		edge.To("EndStepPlan", Plan.Type).
			Unique().
			Annotations(entsql.Annotation{
				OnDelete: entsql.Cascade,
			}),
		edge.To("Build", Build.Type).
			Unique().
			Required(),
		edge.From("ProvisioningSteps", ProvisioningStep.Type).
			Ref("ProvisionedHost"),
		edge.From("ProvisioningScheduledSteps", ProvisioningScheduledStep.Type).
			Ref("ProvisionedHost"),
		edge.From("AgentStatus", AgentStatus.Type).
			Ref("ProvisionedHost").
			Unique(),
		edge.From("AgentTasks", AgentTask.Type).
			Ref("ProvisionedHost"),
		edge.From("Plan", Plan.Type).
			Ref("ProvisionedHost").
			Unique(),
		edge.From("GinFileMiddleware", GinFileMiddleware.Type).
			Ref("ProvisionedHost").
			Unique(),
	}
}
