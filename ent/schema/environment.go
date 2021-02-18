package schema

import (
	"github.com/facebook/ent"
	"github.com/facebook/ent/schema/edge"
	"github.com/facebook/ent/schema/field"
)

// Environment holds the schema definition for the Environment entity.
type Environment struct {
	ent.Schema
}

// Fields of the Environment.
func (Environment) Fields() []ent.Field {
	return []ent.Field{
		field.String("hcl_id").
			StructTag(`hcl:"id,label"`),
		field.String("competition_id").
			StructTag(`hcl:"competition_id,attr"`),
		field.String("name").
			StructTag(`hcl:"name,attr"`),
		field.String("description").
			StructTag(`hcl:"description,attr"`),
		field.String("builder").
			StructTag(`hcl:"builder,attr"`),
		field.Int("team_count").
			StructTag(`hcl:"team_count,attr"`),
		field.Int("revision").
			StructTag(`hcl:"revision,optional"`),
		field.JSON("admin_cidrs", []string{}).
			StructTag(`hcl:"admin_ranges,attr"`),
		field.JSON("exposed_vdi_ports", []string{}).
			StructTag(`hcl:"vdi_allowed_tcp_ports"`),
		field.JSON("config", map[string]string{}).
			StructTag(`hcl:"config,optional"`),
		field.JSON("tags", map[string]string{}).
			StructTag(`hcl:"tags,optional"`),
	}
}

// Edges of the Environment.
func (Environment) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("EnvironmentToTag", Tag.Type),
		edge.To("EnvironmentToUser", User.Type).
			StructTag(`hcl:"maintainer,block"`),
		edge.To("EnvironmentToHost", Host.Type),
		edge.To("EnvironmentToCompetition", Competition.Type),
		edge.To("EnvironmentToBuild", Build.Type),
		edge.From("EnvironmentToIncludedNetwork", IncludedNetwork.Type).Ref("IncludedNetworkToEnvironment").
			StructTag(`hcl:"included_network,block"`),
		edge.From("EnvironmentToNetwork", Network.Type).Ref("NetworkToEnvironment"),
		edge.From("EnvironmentToTeam", Team.Type).Ref("TeamToEnvironment"),
	}
}
