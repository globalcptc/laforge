package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// Validation holds the schema definition for the Validation entity.
type Validation struct {
	ent.Schema
}

/*
  validation {
    name = "A Validator Name"
    source = "path/to/validation/file.laforge"
	type = "regex-content"
	regex = "/wew/ig"
  }
*/

// Fields of the Validation.
func (Validation) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New),
		field.String("hcl_id").
			StructTag(`hcl:"id,label"`),
		field.Enum("validation_type").
			Values(
				"linux-apt-installed",
				"net-tcp-open",
				"net-udp-open",
				"net-http-content-regex",
				"file-exists",
				"file-hash",
				"file-content-regex",
				"dir-exists",
				"user-exists",
				"user-group-membership",
				"host-port-open",
				"host-process-running",
				"host-service-state",
				"net-icmp",
				"file-content-string",
				"file-permission",
			).
			StructTag(`hcl:"validation_type"`),
		field.String("output").Default(""),
		field.Enum("state").Values("AWAITING", "INPROGRESS", "FAILED", "COMPLETE"),
		field.String("error_message").Default(""),
		field.String("hash").StructTag(`hcl:"hash,optional"`),
		field.String("regex").StructTag(`hcl:"regex,optional"`),
		field.String("ip").StructTag(`hcl:"ip,optional"`),
		field.Int("port").StructTag(`hcl:"port,optional"`),
		field.String("hostname").StructTag(`hcl:"hostname,optional"`),
		field.JSON("nameservers", []string{}).StructTag(`hcl:"nameservers,optional"`),
		field.String("package_name").StructTag(`hcl:"package_name,optional"`),
		field.String("username").StructTag(`hcl:"username,optional"`),
		field.String("group_name").StructTag(`hcl:"group_name,optional"`),
		field.String("file_path").StructTag(`hcl:"file_path,optional"`),
		field.String("search_string").StructTag(`hcl:"search_string,optional"`),
		field.String("service_name").StructTag(`hcl:"service_name,optional"`),
		field.String("service_status").StructTag(`hcl:"service_status,optional"`),
		field.String("process_name").StructTag(`hcl:"process_name,optional"`),
	}
}

// Edges of the Validation.
func (Validation) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("ValidationToAgentTask", AgentTask.Type).Ref("AgentTaskToValidation").Unique(),
		edge.From("ValidationToScript", Script.Type).Ref("ScriptToValidation"),
		edge.From("ValidationToEnvironment", Environment.Type).Ref("EnvironmentToValidation"),
	}
}
