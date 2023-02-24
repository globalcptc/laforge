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

// Fields of the ProvisionedScheduleStep.
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

// Edges of the ProvisionedScheduleStep.
func (ProvisioningScheduledStep) Edges() []ent.Edge {
	return []ent.Edge{
		// Status
		edge.To("ProvisioningScheduledStepToStatus", Status.Type).
			Unique().
			Annotations(entsql.Annotation{
				OnDelete: entsql.Cascade,
			}),
		// ScheduledStep
		edge.To("ProvisioningScheduledStepToScheduledStep", ScheduledStep.Type).
			Unique().
			Required(),
		// ProvisionedHost
		edge.To("ProvisioningScheduleStepToProvisionedHost", ProvisionedHost.Type).
			Required().
			Unique(),
		edge.To("ProvisioningScheduledStepToScript", Script.Type).
			Unique(),
		edge.To("ProvisioningScheduledStepToCommand", Command.Type).
			Unique(),
		edge.To("ProvisioningScheduledStepToDNSRecord", DNSRecord.Type).
			Unique(),
		edge.To("ProvisioningScheduledStepToFileDelete", FileDelete.Type).
			Unique(),
		edge.To("ProvisioningScheduledStepToFileDownload", FileDownload.Type).
			Unique(),
		edge.To("ProvisioningScheduledStepToFileExtract", FileExtract.Type).
			Unique(),
		edge.To("ProvisioningScheduledStepToAnsible", Ansible.Type).
			Unique(),
		// AgentTask
		edge.From("ProvisioningScheduledStepToAgentTask", AgentTask.Type).
			Ref("AgentTaskToProvisioningScheduledStep").
			Unique(),
		// Plan
		edge.From("ProvisioningStepToPlan", Plan.Type).
			Ref("PlanToProvisioningScheduledStep").
			Unique(),
		edge.From("ProvisioningScheduledStepToGinFileMiddleware", GinFileMiddleware.Type).
			Ref("GinFileMiddlewareToProvisioningScheduledStep").
			Unique(),
	}
}
