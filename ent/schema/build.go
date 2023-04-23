package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// Build holds the schema definition for the Build entity.
type Build struct {
	ent.Schema
}

// Fields of the Build.
func (Build) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New),
		field.Int("revision"),
		field.Int("environment_revision"),
		field.JSON("vars", map[string]string{}),
		field.Bool("completed_plan").
			Default(false),
	}
}

// Edges of the Build.
func (Build) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("Status", Status.Type).
			Unique().
			Annotations(entsql.Annotation{
				OnDelete: entsql.Cascade,
			}),
		edge.To("Environment", Environment.Type).
			Unique().
			Required(),
		edge.To("Competition", Competition.Type).
			Unique().
			Required(),
		edge.To("LatestBuildCommit", BuildCommit.Type).
			Unique().
			Annotations(entsql.Annotation{
				OnDelete: entsql.Cascade,
			}),
		edge.To("RepoCommits", RepoCommit.Type).Unique(),
		edge.From("ProvisionedNetworks", ProvisionedNetwork.Type).
			Ref("ProvisionedNetworkToBuild"),
		edge.From("Teams", Team.Type).
			Ref("TeamToBuild"),
		edge.From("Plans", Plan.Type).
			Ref("PlanToBuild"),
		edge.From("BuildCommits", BuildCommit.Type).
			Ref("BuildCommitToBuild"),
		edge.From("AdhocPlans", AdhocPlan.Type).
			Ref("Build"),
		edge.From("AgentStatuses", AgentStatus.Type).
			Ref("Build"),
		edge.From("ServerTasks", ServerTask.Type).
			Ref("ServerTaskToBuild"),
	}
}
