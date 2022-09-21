package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// ProvisionedScheduleStep holds the schema definition for the ProvisionedScheduleStep entity.
type ProvisionedScheduleStep struct {
	ent.Schema
}

// Fields of the ProvisionedScheduleStep.
func (ProvisionedScheduleStep) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New),
		field.Time("run_time"),
	}
}

// Edges of the ProvisionedScheduleStep.
func (ProvisionedScheduleStep) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("ProvisionedScheduleStepToStatus", Status.Type).
			Unique().
			Annotations(entsql.Annotation{
				OnDelete: entsql.Cascade,
			}),
		edge.To("ProvisionedScheduleStepToScheduleStep", ScheduleStep.Type).
			Unique(),
		edge.From("ProvisionedScheduleStepToAgentTask", AgentTask.Type).
			Ref("AgentTaskToProvisionedScheduleStep").
			Unique(),
	}
}
