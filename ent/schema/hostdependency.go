package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// HostDependency holds the schema definition for the HostDependency entity.
type HostDependency struct {
	ent.Schema
}

// Fields of the HostDependency.
func (HostDependency) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New),
		field.String("host_id").
			StructTag(`hcl:"host,attr"`),
		field.String("network_id").
			StructTag(`hcl:"network,attr"`),
	}
}

// Edges of the HostDependency.
func (HostDependency) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("RequiredBy", Host.Type).
			Unique().
			Annotations(entsql.Annotation{
				OnDelete: entsql.Cascade,
			}),
		edge.To("DependOnHost", Host.Type).
			Unique().
			Annotations(entsql.Annotation{
				OnDelete: entsql.Cascade,
			}),
		edge.To("DependOnNetwork", Network.Type).
			Unique(),
		edge.From("Environment", Environment.Type).
			Ref("HostDependencies").
			Unique(),
	}
}
