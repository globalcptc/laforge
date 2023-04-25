package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// ProvisioningStep holds the schema definition for the ProvisioningStep entity.
type ProvisioningStep struct {
	ent.Schema
}

// Fields of the ProvisioningStep.
func (ProvisioningStep) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New),
		field.Enum("type").
			Values(
				"Script",
				"Command",
				"DNSRecord",
				"FileDelete",
				"FileDownload",
				"FileExtract",
				"Ansible",
			),
		field.Int("step_number"),
	}
}

// Edges of the ProvisioningStep.
func (ProvisioningStep) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("Status", Status.Type).
			Unique().
			Annotations(entsql.Annotation{
				OnDelete: entsql.Cascade,
			}),
		edge.To("ProvisionedHost", ProvisionedHost.Type).
			Unique(),
		edge.To("Script", Script.Type).
			Unique(),
		edge.To("Command", Command.Type).
			Unique(),
		edge.To("DNSRecord", DNSRecord.Type).
			Unique(),
		edge.To("FileDelete", FileDelete.Type).
			Unique(),
		edge.To("FileDownload", FileDownload.Type).
			Unique(),
		edge.To("FileExtract", FileExtract.Type).
			Unique(),
		edge.To("Ansible", Ansible.Type).
			Unique(),
		edge.From("Plan", Plan.Type).
			Ref("ProvisioningStep").
			Unique(),
		edge.From("AgentTasks", AgentTask.Type).
			Ref("ProvisioningStep"),
		edge.From("GinFileMiddleware", GinFileMiddleware.Type).
			Ref("ProvisioningStep").
			Unique(),
	}
}
