package schema

import (
	"github.com/facebook/ent"
	"github.com/facebook/ent/schema/edge"
	"github.com/facebook/ent/schema/field"
)

// Script holds the schema definition for the Script entity.
type Script struct {
	ent.Schema
}

// Fields of the Script.
func (Script) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			StructTag(`hcl:"name,attr"`),
		field.String("language").
			StructTag(`hcl:"language,attr"`),
		field.String("description").
			StructTag(`hcl:"description,optional"`),
		field.String("source").
			StructTag(`hcl:"source,attr"`),
		field.String("source_type").
			StructTag(`hcl:"source_type,attr"`),
		field.Int("cooldown").
			StructTag(`hcl:"cooldown,optional"`),
		field.Int("timeout").
			StructTag(`hcl:"timeout,optional"`),
		field.Bool("ignore_errors").
			StructTag(`hcl:"ignore_errors,optional"`),
		field.JSON("args", []string{}).
			StructTag(`hcl:"args,optional"`),
		field.Bool("disabled").
			StructTag(`hcl:"disabled,optional" `),
		field.JSON("vars", map[string]string{}).
			StructTag(`hcl:"vars,optional"`),
		field.String("abs_path").
			StructTag(`hcl:"abs_path,optional"`),
		field.JSON("tags", map[string]string{}).
			StructTag(`hcl:"tags,optional"`),
	}
}

// Edges of the Script.
func (Script) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("ScriptToTag", Tag.Type),
		edge.To("ScriptToUser", User.Type).
			StructTag(`hcl:"maintainer,block"`),
		edge.From("ScriptToFinding", Finding.Type).
			Ref("FindingToScript").
			StructTag(`hcl:"finding,block"`),
	}
}
