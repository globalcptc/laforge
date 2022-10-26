package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// ScheduleStep holds the schema definition for the ScheduleStep entity.
type ScheduleStep struct {
	ent.Schema
}

// Fields of the ScheduleStep.
func (ScheduleStep) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New),
		field.Enum("type").
			Values(
				"Script",
				"Command",
				"FileDelete",
				"FileDownload",
				"FileExtract",
				"Ansible",
			),
		field.Bool("repeated").Comment("If repeated, this step is run on the interval specified. Otherwise, it is executed at the start_time."),
		field.Time("start_time").Comment("This is the start time of the run window."),
		field.Time("end_time").Comment("This is the end time of the run window."),
		field.Int("interval").Comment("This is the time between runs of the step in milliseconds."),
	}
}

// Edges of the ScheduleStep.
func (ScheduleStep) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("ScheduleStepToStatus", Status.Type).
			Unique().
			Annotations(entsql.Annotation{
				OnDelete: entsql.Cascade,
			}),
		edge.To("ScheduleStepToScript", Script.Type).
			Unique(),
		edge.To("ScheduleStepToCommand", Command.Type).
			Unique(),
		edge.To("ScheduleStepToFileDelete", FileDelete.Type).
			Unique(),
		edge.To("ScheduleStepToFileDownload", FileDownload.Type).
			Unique(),
		edge.To("ScheduleStepToFileExtract", FileExtract.Type).
			Unique(),
		edge.To("ScheduleStepToAnsible", Ansible.Type).
			Unique(),
		edge.To("ScheduleStepToProvisionedScheduleStep", ProvisionedScheduleStep.Type),
		edge.From("ScheduleStepToHost", Host.Type).
			Ref("HostToScheduleStep").
			Required().
			Unique(),
	}
}
