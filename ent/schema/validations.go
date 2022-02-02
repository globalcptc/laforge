package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// AgentTask holds the schema definition for the AgentTask entity.
type Validator struct {
	ent.Schema
}

// Fields of the AgentTask.
func (Validator) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New),
		field.Enum("command").Values(
			//this should be only execute?
			//this section is pending review
			"EXECUTE",
		),
		field.String("args"),
		field.Int("number"),
		field.String("output").Default(""),
		field.Enum("state").Values("AWAITING", "INPROGRESS", "FAILED", "COMPLETE"),
		field.String("error_message").Default(""),
	}
}

// Edges of the AgentTask.
func (Validator) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("Script", ProvisioningStep.Type).
			StructTag(`hcl:"`),
		edge.To("Command", ProvisionedHost.Type).
			Unique(),
		edge.To("DNSRecord", ProvisioningStep.Type).
			Unique(),
		edge.To("FileDelete", ProvisioningStep.Type).
			Unique(),
		edge.To("FileDownload", ProvisioningStep.Type).
			Unique(),
		edge.To("FileExtract", ProvisioningStep.Type).
			Unique(),
		edge.From("Environment", ProvisioningStep.Type).
			Unique(),
	}
}
