package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// ServerTask holds the schema definition for the AgentTask entity.
type ServerTask struct {
	ent.Schema
}

// Fields of the ServerTask.
func (ServerTask) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New),
		field.Enum("type").Values(
			"LOADENV",
			"CREATEBUILD",
			"RENDERFILES",
			"DELETEBUILD",
			"REBUILD",
			"EXECUTEBUILD",
		),
		field.Time("start_time").Optional(),
		field.Time("end_time").Optional(),
		field.Strings("errors").Optional(),
		field.String("log_file_path").Optional(),
	}
}

// Edges of the ServerTask.
func (ServerTask) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("AuthUser", AuthUser.Type).
			Unique().
			Required(),
		edge.To("Status", Status.Type).
			Unique().
			Required().
			Annotations(entsql.Annotation{
				OnDelete: entsql.Cascade,
			}),
		edge.To("Environment", Environment.Type).
			Unique(),
		edge.To("Build", Build.Type).
			Unique(),
		edge.To("BuildCommit", BuildCommit.Type).
			Unique(),
		edge.To("GinFileMiddleware", GinFileMiddleware.Type),
	}
}
