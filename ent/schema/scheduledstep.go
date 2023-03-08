package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// ScheduledStep holds the schema definition for the ScheduledStep entity.
type ScheduledStep struct {
	ent.Schema
}

// Fields of the ScheduledStep.
func (ScheduledStep) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New),
		field.String("hcl_id").
			StructTag(`hcl:"id,label"`),
		field.String("name").
			StructTag(`hcl:"name,attr"`),
		field.String("description").
			StructTag(`hcl:"description,optional"`),
		field.String("step").
			StructTag(`hcl:"step,attr"`),
		field.Enum("type").Values("CRON", "RUNONCE").
			StructTag(`hcl:"type,attr"`),
		field.String("schedule").
			StructTag(`hcl:"schedule,optional"`),
		field.String("run_at").
			StructTag(`hcl:"run_at,optional"`),
	}
}

// Edges of the ScheduledStep.
func (ScheduledStep) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("ScheduledStepToEnvironment", Environment.Type).
			Ref("EnvironmentToScheduledStep").
			Unique(),
	}
}
