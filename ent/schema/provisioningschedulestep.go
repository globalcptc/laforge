package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// ProvisioningScheduledStep holds the schema definition for the ProvisioningScheduledStep entity.
type ProvisioningScheduledStep struct {
	ent.Schema
}

// Fields of the ProvisioningScheduledStep.
func (ProvisioningScheduledStep) Fields() []ent.Field {
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
		field.Time("run_time"),
	}
}

// Edges of the ProvisioningScheduledStep.
func (ProvisioningScheduledStep) Edges() []ent.Edge {
	return []ent.Edge{
		// Status
		edge.To("Status", Status.Type).
			Unique().
			Annotations(entsql.Annotation{
				OnDelete: entsql.Cascade,
			}),
		// ScheduledStep
		edge.To("ScheduledStep", ScheduledStep.Type).
			Unique().
			Required(),
		// ProvisionedHost
		edge.To("ProvisionedHost", ProvisionedHost.Type).
			Required().
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
		// AgentTask
		edge.From("AgentTasks", AgentTask.Type).
			Ref("ProvisioningScheduledStep"),
		// Plan
		edge.From("Plan", Plan.Type).
			Ref("ProvisioningScheduledStep").
			Unique(),
		edge.From("GinFileMiddleware", GinFileMiddleware.Type).
			Ref("ProvisioningScheduledStep").
			Unique(),
	}
}
