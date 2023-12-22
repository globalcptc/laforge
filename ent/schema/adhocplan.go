package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// AdhocPlan holds the schema definition for the AdhocPlan entity.
type AdhocPlan struct {
	ent.Schema
}

// Fields of the AdhocPlan.
func (AdhocPlan) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New),
	}
}

// Edges of the AdhocPlan.
func (AdhocPlan) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("NextAdhocPlans", AdhocPlan.Type).
			From("PrevAdhocPlans").
			Annotations(entsql.Annotation{
				OnDelete: entsql.Cascade,
			}),
		edge.To("Build", Build.Type).
			Unique().
			Required(),
		edge.To("Status", Status.Type).
			Unique().
			Required().
			Annotations(entsql.Annotation{
				OnDelete: entsql.Cascade,
			}),
		edge.To("AgentTask", AgentTask.Type).
			Unique().
			Required(),
	}
}
