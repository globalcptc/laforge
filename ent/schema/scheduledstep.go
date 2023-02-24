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
		field.Int64("start_time").
			StructTag(`hcl:"start_time,attr"`),
		field.Int64("end_time").
			StructTag(`hcl:"end_time,attr"`),
		field.Int("interval").
			StructTag(`hcl:"interval,optional"`),
		field.Bool("repeated").
			StructTag(`hcl:"repeated,optional"`),
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
