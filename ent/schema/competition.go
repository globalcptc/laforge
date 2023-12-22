package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// Competition holds the schema definition for the Competition entity.
type Competition struct {
	ent.Schema
}

// Annotations of the Competition.
func (Competition) Annotations() []schema.Annotation {
	return []schema.Annotation{
		edge.Annotation{
			StructTag: `json:"edges"`,
		},
	}
}

// Fields of the Competition.
func (Competition) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New),
		field.String("hcl_id").
			StructTag(`hcl:"id,label"`),
		field.String("root_password").
			StructTag(`hcl:"root_password,attr"`),
		field.Int64("start_time").
			Optional().
			StructTag(`hcl:"start_time,optional"`),
		field.Int64("stop_time").
			Optional().
			StructTag(`hcl:"stop_time,optional"`),
		field.JSON("config", map[string]string{}).
			StructTag(`hcl:"config,optional"`),
		field.JSON("tags", map[string]string{}).
			StructTag(`hcl:"tags,optional"`),
	}
}

// Edges of the Competition.
func (Competition) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("DNS", DNS.Type).
			StructTag(`hcl:"dns,block"`),
		edge.From("Environment", Environment.Type).
			Ref("Competitions").
			Unique(),
		edge.From("Builds", Build.Type).
			Ref("Competition"),
	}
}
