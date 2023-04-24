package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// ProvisionedNetwork holds the schema definition for the ProvisionedNetwork entity.
type ProvisionedNetwork struct {
	ent.Schema
}

// Fields of the ProvisionedNetwork.
func (ProvisionedNetwork) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New),
		field.String("name"),
		field.String("cidr"),
		field.JSON("vars", map[string]string{}),
	}
}

// Edges of the ProvisionedNetwork.
func (ProvisionedNetwork) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("Status", Status.Type).
			Unique().
			Annotations(entsql.Annotation{
				OnDelete: entsql.Cascade,
			}),
		edge.To("Network", Network.Type).
			Unique(),
		edge.To("Build", Build.Type).
			Unique(),
		edge.To("Team", Team.Type).
			Unique(),
		edge.From("ProvisionedHosts", ProvisionedHost.Type).
			Ref("ProvisionedNetwork"),
		edge.From("Plan", Plan.Type).
			Ref("ProvisionedNetwork").
			Unique(),
	}
}
