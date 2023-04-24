package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// BuildCommit holds the schema definition for the BuildCommit entity.
type BuildCommit struct {
	ent.Schema
}

// Fields of the BuildCommit.
func (BuildCommit) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New),
		field.Enum("type").Values("ROOT", "REBUILD", "DELETE"),
		field.Int("revision"),
		field.Enum("state").Values("PLANNING", "INPROGRESS", "APPLIED", "CANCELLED", "APPROVED"),
		field.Time("created_at").Default(time.Now),
	}
}

// Edges of the BuildCommit.
func (BuildCommit) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("Build", Build.Type).
			Unique().
			Required(),
		edge.From("ServerTasks", ServerTask.Type).
			Ref("ServerTaskToBuildCommit"),
		edge.From("PlanDiffs", PlanDiff.Type).
			Ref("BuildCommit").
			Annotations(entsql.Annotation{
				OnDelete: entsql.Cascade,
			}),
	}
}
