package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// Environment holds the schema definition for the Environment entity.
type Environment struct {
	ent.Schema
}

// Annotations of the Environment.
func (Environment) Annotations() []schema.Annotation {
	return []schema.Annotation{
		edge.Annotation{
			StructTag: `json:"edges"`,
		},
	}
}

// Fields of the Environment.
func (Environment) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New),
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
		edge.To("Users", User.Type).
			StructTag(`hcl:"maintainer,block"`).
			Annotations(entsql.Annotation{
				OnDelete: entsql.Cascade,
			}),
		edge.To("Hosts", Host.Type).
			Annotations(entsql.Annotation{
				OnDelete: entsql.Cascade,
			}),
		edge.To("Competitions", Competition.Type).
			Annotations(entsql.Annotation{
				OnDelete: entsql.Cascade,
			}),
		edge.To("Identities", Identity.Type).
			Annotations(entsql.Annotation{
				OnDelete: entsql.Cascade,
			}),
		edge.To("Commands", Command.Type).
			Annotations(entsql.Annotation{
				OnDelete: entsql.Cascade,
			}),
		edge.To("Scripts", Script.Type).
			Annotations(entsql.Annotation{
				OnDelete: entsql.Cascade,
			}),
		edge.To("FileDownloads", FileDownload.Type).
			Annotations(entsql.Annotation{
				OnDelete: entsql.Cascade,
			}),
		edge.To("FileDeletes", FileDelete.Type).
			Annotations(entsql.Annotation{
				OnDelete: entsql.Cascade,
			}),
		edge.To("FileExtracts", FileExtract.Type).
			Annotations(entsql.Annotation{
				OnDelete: entsql.Cascade,
			}),
		edge.To("IncludedNetworks", IncludedNetwork.Type).
			StructTag(`hcl:"included_network,block"`).
			Annotations(entsql.Annotation{
				OnDelete: entsql.Cascade,
			}),
		edge.To("Findings", Finding.Type).
			Annotations(entsql.Annotation{
				OnDelete: entsql.Cascade,
			}),
		edge.To("DNSRecords", DNSRecord.Type).
			Annotations(entsql.Annotation{
				OnDelete: entsql.Cascade,
			}),
		edge.To("DNS", DNS.Type).
			Annotations(entsql.Annotation{
				OnDelete: entsql.Cascade,
			}),
		edge.To("Networks", Network.Type).
			Annotations(entsql.Annotation{
				OnDelete: entsql.Cascade,
			}),
		edge.To("HostDependencies", HostDependency.Type).
			Annotations(entsql.Annotation{
				OnDelete: entsql.Cascade,
			}),
		edge.To("Ansibles", Ansible.Type).
			Annotations(entsql.Annotation{
				OnDelete: entsql.Cascade,
			}),
		edge.To("ScheduledSteps", ScheduledStep.Type).
			Annotations(entsql.Annotation{
				OnDelete: entsql.Cascade,
			}),
		edge.From("Builds", Build.Type).
			Ref("Environment"),
		edge.From("Repositories", Repository.Type).
			Ref("RepositoryToEnvironment"),
		edge.From("ServerTasks", ServerTask.Type).
			Ref("ServerTaskToEnvironment"),
	}
}
