package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// Ansible holds the schema definition for the Ansible entity.
type Ansible struct {
	ent.Schema
}

// Fields of the Ansible.
func (Ansible) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New),
		field.String("name").
			StructTag(`hcl:"name,attr"`),
		field.String("hcl_id").
			StructTag(`hcl:"id,label"`),
		field.String("description").
			StructTag(`hcl:"description,optional"`),
		field.String("source").
			StructTag(`hcl:"source,attr"`),
		field.String("playbook_name").
			StructTag(`hcl:"playbook_name,attr"`),
		field.Enum("method").Values(
			"LOCAL",
		).StructTag(`hcl:"method,optional"`),
		field.String("inventory").
			StructTag(`hcl:"inventory,optional"`),
		field.String("abs_path").
			StructTag(`hcl:"abs_path,optional"`),
		field.JSON("tags", map[string]string{}).
			StructTag(`hcl:"tags,optional"`),
	}
}

// Edges of the Ansible.
func (Ansible) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("Users", User.Type).
			StructTag(`hcl:"maintainer,block"`),
		edge.From("Environment", Environment.Type).
			Ref("Ansibles").
			Unique(),
	}
}
