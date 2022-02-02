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

type ValidatorConfigBlock struct {
	Name        string `json:"name" hcl:"name"`
	Regex       string `json:"name,omitempty" hcl:"regex,optional"`
	Ip          string `json:"name,omitempty" hcl:"ip,optional"`
	Port        int
	Hostname    string   `json:"name,omitempty" hcl:"hostname,optional"`
	Nameservers []string `json:"name_servers,omitempty" hcl:"name_servers,optional"`
	PackageName string   `json:"name,omitempty" hcl:"packagename,optional"`
	Username    string   `json:"name,omitempty" hcl:"username,optional"`
	Groupname   string   `json:"name,omitempty" hcl:"groupname,optional"`
	Filepath    string   `json:"name,omitempty" hcl:"filepath,optional"`
	ServiceName string   `json:"name,omitempty" hcl:"servicename,optional"`
	ProcessName string   `json:"process_name,omitempty" hcl:"process_name,optional"`
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
		edge.To("Validator", ValidatorBlockType).StructTag(`hcl:"dns,block"`).Unique(),
	}
}
