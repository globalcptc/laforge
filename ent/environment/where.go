// Code generated by ent, DO NOT EDIT.

package environment

import (
	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"github.com/gen0cide/laforge/ent/predicate"
	"github.com/google/uuid"
)

// ID filters vertices based on their ID field.
func ID(id uuid.UUID) predicate.Environment {
	return predicate.Environment(sql.FieldEQ(FieldID, id))
}

// IDEQ applies the EQ predicate on the ID field.
func IDEQ(id uuid.UUID) predicate.Environment {
	return predicate.Environment(sql.FieldEQ(FieldID, id))
}

// IDNEQ applies the NEQ predicate on the ID field.
func IDNEQ(id uuid.UUID) predicate.Environment {
	return predicate.Environment(sql.FieldNEQ(FieldID, id))
}

// IDIn applies the In predicate on the ID field.
func IDIn(ids ...uuid.UUID) predicate.Environment {
	return predicate.Environment(sql.FieldIn(FieldID, ids...))
}

// IDNotIn applies the NotIn predicate on the ID field.
func IDNotIn(ids ...uuid.UUID) predicate.Environment {
	return predicate.Environment(sql.FieldNotIn(FieldID, ids...))
}

// IDGT applies the GT predicate on the ID field.
func IDGT(id uuid.UUID) predicate.Environment {
	return predicate.Environment(sql.FieldGT(FieldID, id))
}

// IDGTE applies the GTE predicate on the ID field.
func IDGTE(id uuid.UUID) predicate.Environment {
	return predicate.Environment(sql.FieldGTE(FieldID, id))
}

// IDLT applies the LT predicate on the ID field.
func IDLT(id uuid.UUID) predicate.Environment {
	return predicate.Environment(sql.FieldLT(FieldID, id))
}

// IDLTE applies the LTE predicate on the ID field.
func IDLTE(id uuid.UUID) predicate.Environment {
	return predicate.Environment(sql.FieldLTE(FieldID, id))
}

// HCLID applies equality check predicate on the "hcl_id" field. It's identical to HCLIDEQ.
func HCLID(v string) predicate.Environment {
	return predicate.Environment(sql.FieldEQ(FieldHCLID, v))
}

// CompetitionID applies equality check predicate on the "competition_id" field. It's identical to CompetitionIDEQ.
func CompetitionID(v string) predicate.Environment {
	return predicate.Environment(sql.FieldEQ(FieldCompetitionID, v))
}

// Name applies equality check predicate on the "name" field. It's identical to NameEQ.
func Name(v string) predicate.Environment {
	return predicate.Environment(sql.FieldEQ(FieldName, v))
}

// Description applies equality check predicate on the "description" field. It's identical to DescriptionEQ.
func Description(v string) predicate.Environment {
	return predicate.Environment(sql.FieldEQ(FieldDescription, v))
}

// Builder applies equality check predicate on the "builder" field. It's identical to BuilderEQ.
func Builder(v string) predicate.Environment {
	return predicate.Environment(sql.FieldEQ(FieldBuilder, v))
}

// TeamCount applies equality check predicate on the "team_count" field. It's identical to TeamCountEQ.
func TeamCount(v int) predicate.Environment {
	return predicate.Environment(sql.FieldEQ(FieldTeamCount, v))
}

// Revision applies equality check predicate on the "revision" field. It's identical to RevisionEQ.
func Revision(v int) predicate.Environment {
	return predicate.Environment(sql.FieldEQ(FieldRevision, v))
}

// HCLIDEQ applies the EQ predicate on the "hcl_id" field.
func HCLIDEQ(v string) predicate.Environment {
	return predicate.Environment(sql.FieldEQ(FieldHCLID, v))
}

// HCLIDNEQ applies the NEQ predicate on the "hcl_id" field.
func HCLIDNEQ(v string) predicate.Environment {
	return predicate.Environment(sql.FieldNEQ(FieldHCLID, v))
}

// HCLIDIn applies the In predicate on the "hcl_id" field.
func HCLIDIn(vs ...string) predicate.Environment {
	return predicate.Environment(sql.FieldIn(FieldHCLID, vs...))
}

// HCLIDNotIn applies the NotIn predicate on the "hcl_id" field.
func HCLIDNotIn(vs ...string) predicate.Environment {
	return predicate.Environment(sql.FieldNotIn(FieldHCLID, vs...))
}

// HCLIDGT applies the GT predicate on the "hcl_id" field.
func HCLIDGT(v string) predicate.Environment {
	return predicate.Environment(sql.FieldGT(FieldHCLID, v))
}

// HCLIDGTE applies the GTE predicate on the "hcl_id" field.
func HCLIDGTE(v string) predicate.Environment {
	return predicate.Environment(sql.FieldGTE(FieldHCLID, v))
}

// HCLIDLT applies the LT predicate on the "hcl_id" field.
func HCLIDLT(v string) predicate.Environment {
	return predicate.Environment(sql.FieldLT(FieldHCLID, v))
}

// HCLIDLTE applies the LTE predicate on the "hcl_id" field.
func HCLIDLTE(v string) predicate.Environment {
	return predicate.Environment(sql.FieldLTE(FieldHCLID, v))
}

// HCLIDContains applies the Contains predicate on the "hcl_id" field.
func HCLIDContains(v string) predicate.Environment {
	return predicate.Environment(sql.FieldContains(FieldHCLID, v))
}

// HCLIDHasPrefix applies the HasPrefix predicate on the "hcl_id" field.
func HCLIDHasPrefix(v string) predicate.Environment {
	return predicate.Environment(sql.FieldHasPrefix(FieldHCLID, v))
}

// HCLIDHasSuffix applies the HasSuffix predicate on the "hcl_id" field.
func HCLIDHasSuffix(v string) predicate.Environment {
	return predicate.Environment(sql.FieldHasSuffix(FieldHCLID, v))
}

// HCLIDEqualFold applies the EqualFold predicate on the "hcl_id" field.
func HCLIDEqualFold(v string) predicate.Environment {
	return predicate.Environment(sql.FieldEqualFold(FieldHCLID, v))
}

// HCLIDContainsFold applies the ContainsFold predicate on the "hcl_id" field.
func HCLIDContainsFold(v string) predicate.Environment {
	return predicate.Environment(sql.FieldContainsFold(FieldHCLID, v))
}

// CompetitionIDEQ applies the EQ predicate on the "competition_id" field.
func CompetitionIDEQ(v string) predicate.Environment {
	return predicate.Environment(sql.FieldEQ(FieldCompetitionID, v))
}

// CompetitionIDNEQ applies the NEQ predicate on the "competition_id" field.
func CompetitionIDNEQ(v string) predicate.Environment {
	return predicate.Environment(sql.FieldNEQ(FieldCompetitionID, v))
}

// CompetitionIDIn applies the In predicate on the "competition_id" field.
func CompetitionIDIn(vs ...string) predicate.Environment {
	return predicate.Environment(sql.FieldIn(FieldCompetitionID, vs...))
}

// CompetitionIDNotIn applies the NotIn predicate on the "competition_id" field.
func CompetitionIDNotIn(vs ...string) predicate.Environment {
	return predicate.Environment(sql.FieldNotIn(FieldCompetitionID, vs...))
}

// CompetitionIDGT applies the GT predicate on the "competition_id" field.
func CompetitionIDGT(v string) predicate.Environment {
	return predicate.Environment(sql.FieldGT(FieldCompetitionID, v))
}

// CompetitionIDGTE applies the GTE predicate on the "competition_id" field.
func CompetitionIDGTE(v string) predicate.Environment {
	return predicate.Environment(sql.FieldGTE(FieldCompetitionID, v))
}

// CompetitionIDLT applies the LT predicate on the "competition_id" field.
func CompetitionIDLT(v string) predicate.Environment {
	return predicate.Environment(sql.FieldLT(FieldCompetitionID, v))
}

// CompetitionIDLTE applies the LTE predicate on the "competition_id" field.
func CompetitionIDLTE(v string) predicate.Environment {
	return predicate.Environment(sql.FieldLTE(FieldCompetitionID, v))
}

// CompetitionIDContains applies the Contains predicate on the "competition_id" field.
func CompetitionIDContains(v string) predicate.Environment {
	return predicate.Environment(sql.FieldContains(FieldCompetitionID, v))
}

// CompetitionIDHasPrefix applies the HasPrefix predicate on the "competition_id" field.
func CompetitionIDHasPrefix(v string) predicate.Environment {
	return predicate.Environment(sql.FieldHasPrefix(FieldCompetitionID, v))
}

// CompetitionIDHasSuffix applies the HasSuffix predicate on the "competition_id" field.
func CompetitionIDHasSuffix(v string) predicate.Environment {
	return predicate.Environment(sql.FieldHasSuffix(FieldCompetitionID, v))
}

// CompetitionIDEqualFold applies the EqualFold predicate on the "competition_id" field.
func CompetitionIDEqualFold(v string) predicate.Environment {
	return predicate.Environment(sql.FieldEqualFold(FieldCompetitionID, v))
}

// CompetitionIDContainsFold applies the ContainsFold predicate on the "competition_id" field.
func CompetitionIDContainsFold(v string) predicate.Environment {
	return predicate.Environment(sql.FieldContainsFold(FieldCompetitionID, v))
}

// NameEQ applies the EQ predicate on the "name" field.
func NameEQ(v string) predicate.Environment {
	return predicate.Environment(sql.FieldEQ(FieldName, v))
}

// NameNEQ applies the NEQ predicate on the "name" field.
func NameNEQ(v string) predicate.Environment {
	return predicate.Environment(sql.FieldNEQ(FieldName, v))
}

// NameIn applies the In predicate on the "name" field.
func NameIn(vs ...string) predicate.Environment {
	return predicate.Environment(sql.FieldIn(FieldName, vs...))
}

// NameNotIn applies the NotIn predicate on the "name" field.
func NameNotIn(vs ...string) predicate.Environment {
	return predicate.Environment(sql.FieldNotIn(FieldName, vs...))
}

// NameGT applies the GT predicate on the "name" field.
func NameGT(v string) predicate.Environment {
	return predicate.Environment(sql.FieldGT(FieldName, v))
}

// NameGTE applies the GTE predicate on the "name" field.
func NameGTE(v string) predicate.Environment {
	return predicate.Environment(sql.FieldGTE(FieldName, v))
}

// NameLT applies the LT predicate on the "name" field.
func NameLT(v string) predicate.Environment {
	return predicate.Environment(sql.FieldLT(FieldName, v))
}

// NameLTE applies the LTE predicate on the "name" field.
func NameLTE(v string) predicate.Environment {
	return predicate.Environment(sql.FieldLTE(FieldName, v))
}

// NameContains applies the Contains predicate on the "name" field.
func NameContains(v string) predicate.Environment {
	return predicate.Environment(sql.FieldContains(FieldName, v))
}

// NameHasPrefix applies the HasPrefix predicate on the "name" field.
func NameHasPrefix(v string) predicate.Environment {
	return predicate.Environment(sql.FieldHasPrefix(FieldName, v))
}

// NameHasSuffix applies the HasSuffix predicate on the "name" field.
func NameHasSuffix(v string) predicate.Environment {
	return predicate.Environment(sql.FieldHasSuffix(FieldName, v))
}

// NameEqualFold applies the EqualFold predicate on the "name" field.
func NameEqualFold(v string) predicate.Environment {
	return predicate.Environment(sql.FieldEqualFold(FieldName, v))
}

// NameContainsFold applies the ContainsFold predicate on the "name" field.
func NameContainsFold(v string) predicate.Environment {
	return predicate.Environment(sql.FieldContainsFold(FieldName, v))
}

// DescriptionEQ applies the EQ predicate on the "description" field.
func DescriptionEQ(v string) predicate.Environment {
	return predicate.Environment(sql.FieldEQ(FieldDescription, v))
}

// DescriptionNEQ applies the NEQ predicate on the "description" field.
func DescriptionNEQ(v string) predicate.Environment {
	return predicate.Environment(sql.FieldNEQ(FieldDescription, v))
}

// DescriptionIn applies the In predicate on the "description" field.
func DescriptionIn(vs ...string) predicate.Environment {
	return predicate.Environment(sql.FieldIn(FieldDescription, vs...))
}

// DescriptionNotIn applies the NotIn predicate on the "description" field.
func DescriptionNotIn(vs ...string) predicate.Environment {
	return predicate.Environment(sql.FieldNotIn(FieldDescription, vs...))
}

// DescriptionGT applies the GT predicate on the "description" field.
func DescriptionGT(v string) predicate.Environment {
	return predicate.Environment(sql.FieldGT(FieldDescription, v))
}

// DescriptionGTE applies the GTE predicate on the "description" field.
func DescriptionGTE(v string) predicate.Environment {
	return predicate.Environment(sql.FieldGTE(FieldDescription, v))
}

// DescriptionLT applies the LT predicate on the "description" field.
func DescriptionLT(v string) predicate.Environment {
	return predicate.Environment(sql.FieldLT(FieldDescription, v))
}

// DescriptionLTE applies the LTE predicate on the "description" field.
func DescriptionLTE(v string) predicate.Environment {
	return predicate.Environment(sql.FieldLTE(FieldDescription, v))
}

// DescriptionContains applies the Contains predicate on the "description" field.
func DescriptionContains(v string) predicate.Environment {
	return predicate.Environment(sql.FieldContains(FieldDescription, v))
}

// DescriptionHasPrefix applies the HasPrefix predicate on the "description" field.
func DescriptionHasPrefix(v string) predicate.Environment {
	return predicate.Environment(sql.FieldHasPrefix(FieldDescription, v))
}

// DescriptionHasSuffix applies the HasSuffix predicate on the "description" field.
func DescriptionHasSuffix(v string) predicate.Environment {
	return predicate.Environment(sql.FieldHasSuffix(FieldDescription, v))
}

// DescriptionEqualFold applies the EqualFold predicate on the "description" field.
func DescriptionEqualFold(v string) predicate.Environment {
	return predicate.Environment(sql.FieldEqualFold(FieldDescription, v))
}

// DescriptionContainsFold applies the ContainsFold predicate on the "description" field.
func DescriptionContainsFold(v string) predicate.Environment {
	return predicate.Environment(sql.FieldContainsFold(FieldDescription, v))
}

// BuilderEQ applies the EQ predicate on the "builder" field.
func BuilderEQ(v string) predicate.Environment {
	return predicate.Environment(sql.FieldEQ(FieldBuilder, v))
}

// BuilderNEQ applies the NEQ predicate on the "builder" field.
func BuilderNEQ(v string) predicate.Environment {
	return predicate.Environment(sql.FieldNEQ(FieldBuilder, v))
}

// BuilderIn applies the In predicate on the "builder" field.
func BuilderIn(vs ...string) predicate.Environment {
	return predicate.Environment(sql.FieldIn(FieldBuilder, vs...))
}

// BuilderNotIn applies the NotIn predicate on the "builder" field.
func BuilderNotIn(vs ...string) predicate.Environment {
	return predicate.Environment(sql.FieldNotIn(FieldBuilder, vs...))
}

// BuilderGT applies the GT predicate on the "builder" field.
func BuilderGT(v string) predicate.Environment {
	return predicate.Environment(sql.FieldGT(FieldBuilder, v))
}

// BuilderGTE applies the GTE predicate on the "builder" field.
func BuilderGTE(v string) predicate.Environment {
	return predicate.Environment(sql.FieldGTE(FieldBuilder, v))
}

// BuilderLT applies the LT predicate on the "builder" field.
func BuilderLT(v string) predicate.Environment {
	return predicate.Environment(sql.FieldLT(FieldBuilder, v))
}

// BuilderLTE applies the LTE predicate on the "builder" field.
func BuilderLTE(v string) predicate.Environment {
	return predicate.Environment(sql.FieldLTE(FieldBuilder, v))
}

// BuilderContains applies the Contains predicate on the "builder" field.
func BuilderContains(v string) predicate.Environment {
	return predicate.Environment(sql.FieldContains(FieldBuilder, v))
}

// BuilderHasPrefix applies the HasPrefix predicate on the "builder" field.
func BuilderHasPrefix(v string) predicate.Environment {
	return predicate.Environment(sql.FieldHasPrefix(FieldBuilder, v))
}

// BuilderHasSuffix applies the HasSuffix predicate on the "builder" field.
func BuilderHasSuffix(v string) predicate.Environment {
	return predicate.Environment(sql.FieldHasSuffix(FieldBuilder, v))
}

// BuilderEqualFold applies the EqualFold predicate on the "builder" field.
func BuilderEqualFold(v string) predicate.Environment {
	return predicate.Environment(sql.FieldEqualFold(FieldBuilder, v))
}

// BuilderContainsFold applies the ContainsFold predicate on the "builder" field.
func BuilderContainsFold(v string) predicate.Environment {
	return predicate.Environment(sql.FieldContainsFold(FieldBuilder, v))
}

// TeamCountEQ applies the EQ predicate on the "team_count" field.
func TeamCountEQ(v int) predicate.Environment {
	return predicate.Environment(sql.FieldEQ(FieldTeamCount, v))
}

// TeamCountNEQ applies the NEQ predicate on the "team_count" field.
func TeamCountNEQ(v int) predicate.Environment {
	return predicate.Environment(sql.FieldNEQ(FieldTeamCount, v))
}

// TeamCountIn applies the In predicate on the "team_count" field.
func TeamCountIn(vs ...int) predicate.Environment {
	return predicate.Environment(sql.FieldIn(FieldTeamCount, vs...))
}

// TeamCountNotIn applies the NotIn predicate on the "team_count" field.
func TeamCountNotIn(vs ...int) predicate.Environment {
	return predicate.Environment(sql.FieldNotIn(FieldTeamCount, vs...))
}

// TeamCountGT applies the GT predicate on the "team_count" field.
func TeamCountGT(v int) predicate.Environment {
	return predicate.Environment(sql.FieldGT(FieldTeamCount, v))
}

// TeamCountGTE applies the GTE predicate on the "team_count" field.
func TeamCountGTE(v int) predicate.Environment {
	return predicate.Environment(sql.FieldGTE(FieldTeamCount, v))
}

// TeamCountLT applies the LT predicate on the "team_count" field.
func TeamCountLT(v int) predicate.Environment {
	return predicate.Environment(sql.FieldLT(FieldTeamCount, v))
}

// TeamCountLTE applies the LTE predicate on the "team_count" field.
func TeamCountLTE(v int) predicate.Environment {
	return predicate.Environment(sql.FieldLTE(FieldTeamCount, v))
}

// RevisionEQ applies the EQ predicate on the "revision" field.
func RevisionEQ(v int) predicate.Environment {
	return predicate.Environment(sql.FieldEQ(FieldRevision, v))
}

// RevisionNEQ applies the NEQ predicate on the "revision" field.
func RevisionNEQ(v int) predicate.Environment {
	return predicate.Environment(sql.FieldNEQ(FieldRevision, v))
}

// RevisionIn applies the In predicate on the "revision" field.
func RevisionIn(vs ...int) predicate.Environment {
	return predicate.Environment(sql.FieldIn(FieldRevision, vs...))
}

// RevisionNotIn applies the NotIn predicate on the "revision" field.
func RevisionNotIn(vs ...int) predicate.Environment {
	return predicate.Environment(sql.FieldNotIn(FieldRevision, vs...))
}

// RevisionGT applies the GT predicate on the "revision" field.
func RevisionGT(v int) predicate.Environment {
	return predicate.Environment(sql.FieldGT(FieldRevision, v))
}

// RevisionGTE applies the GTE predicate on the "revision" field.
func RevisionGTE(v int) predicate.Environment {
	return predicate.Environment(sql.FieldGTE(FieldRevision, v))
}

// RevisionLT applies the LT predicate on the "revision" field.
func RevisionLT(v int) predicate.Environment {
	return predicate.Environment(sql.FieldLT(FieldRevision, v))
}

// RevisionLTE applies the LTE predicate on the "revision" field.
func RevisionLTE(v int) predicate.Environment {
	return predicate.Environment(sql.FieldLTE(FieldRevision, v))
}

// HasUsers applies the HasEdge predicate on the "Users" edge.
func HasUsers() predicate.Environment {
	return predicate.Environment(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.Edge(sqlgraph.M2M, false, UsersTable, UsersPrimaryKey...),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasUsersWith applies the HasEdge predicate on the "Users" edge with a given conditions (other predicates).
func HasUsersWith(preds ...predicate.User) predicate.Environment {
	return predicate.Environment(func(s *sql.Selector) {
		step := newUsersStep()
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// HasHosts applies the HasEdge predicate on the "Hosts" edge.
func HasHosts() predicate.Environment {
	return predicate.Environment(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.Edge(sqlgraph.O2M, false, HostsTable, HostsColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasHostsWith applies the HasEdge predicate on the "Hosts" edge with a given conditions (other predicates).
func HasHostsWith(preds ...predicate.Host) predicate.Environment {
	return predicate.Environment(func(s *sql.Selector) {
		step := newHostsStep()
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// HasCompetitions applies the HasEdge predicate on the "Competitions" edge.
func HasCompetitions() predicate.Environment {
	return predicate.Environment(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.Edge(sqlgraph.O2M, false, CompetitionsTable, CompetitionsColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasCompetitionsWith applies the HasEdge predicate on the "Competitions" edge with a given conditions (other predicates).
func HasCompetitionsWith(preds ...predicate.Competition) predicate.Environment {
	return predicate.Environment(func(s *sql.Selector) {
		step := newCompetitionsStep()
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// HasIdentities applies the HasEdge predicate on the "Identities" edge.
func HasIdentities() predicate.Environment {
	return predicate.Environment(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.Edge(sqlgraph.O2M, false, IdentitiesTable, IdentitiesColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasIdentitiesWith applies the HasEdge predicate on the "Identities" edge with a given conditions (other predicates).
func HasIdentitiesWith(preds ...predicate.Identity) predicate.Environment {
	return predicate.Environment(func(s *sql.Selector) {
		step := newIdentitiesStep()
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// HasCommands applies the HasEdge predicate on the "Commands" edge.
func HasCommands() predicate.Environment {
	return predicate.Environment(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.Edge(sqlgraph.O2M, false, CommandsTable, CommandsColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasCommandsWith applies the HasEdge predicate on the "Commands" edge with a given conditions (other predicates).
func HasCommandsWith(preds ...predicate.Command) predicate.Environment {
	return predicate.Environment(func(s *sql.Selector) {
		step := newCommandsStep()
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// HasScripts applies the HasEdge predicate on the "Scripts" edge.
func HasScripts() predicate.Environment {
	return predicate.Environment(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.Edge(sqlgraph.O2M, false, ScriptsTable, ScriptsColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasScriptsWith applies the HasEdge predicate on the "Scripts" edge with a given conditions (other predicates).
func HasScriptsWith(preds ...predicate.Script) predicate.Environment {
	return predicate.Environment(func(s *sql.Selector) {
		step := newScriptsStep()
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// HasFileDownloads applies the HasEdge predicate on the "FileDownloads" edge.
func HasFileDownloads() predicate.Environment {
	return predicate.Environment(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.Edge(sqlgraph.O2M, false, FileDownloadsTable, FileDownloadsColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasFileDownloadsWith applies the HasEdge predicate on the "FileDownloads" edge with a given conditions (other predicates).
func HasFileDownloadsWith(preds ...predicate.FileDownload) predicate.Environment {
	return predicate.Environment(func(s *sql.Selector) {
		step := newFileDownloadsStep()
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// HasFileDeletes applies the HasEdge predicate on the "FileDeletes" edge.
func HasFileDeletes() predicate.Environment {
	return predicate.Environment(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.Edge(sqlgraph.O2M, false, FileDeletesTable, FileDeletesColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasFileDeletesWith applies the HasEdge predicate on the "FileDeletes" edge with a given conditions (other predicates).
func HasFileDeletesWith(preds ...predicate.FileDelete) predicate.Environment {
	return predicate.Environment(func(s *sql.Selector) {
		step := newFileDeletesStep()
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// HasFileExtracts applies the HasEdge predicate on the "FileExtracts" edge.
func HasFileExtracts() predicate.Environment {
	return predicate.Environment(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.Edge(sqlgraph.O2M, false, FileExtractsTable, FileExtractsColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasFileExtractsWith applies the HasEdge predicate on the "FileExtracts" edge with a given conditions (other predicates).
func HasFileExtractsWith(preds ...predicate.FileExtract) predicate.Environment {
	return predicate.Environment(func(s *sql.Selector) {
		step := newFileExtractsStep()
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// HasIncludedNetworks applies the HasEdge predicate on the "IncludedNetworks" edge.
func HasIncludedNetworks() predicate.Environment {
	return predicate.Environment(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.Edge(sqlgraph.M2M, false, IncludedNetworksTable, IncludedNetworksPrimaryKey...),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasIncludedNetworksWith applies the HasEdge predicate on the "IncludedNetworks" edge with a given conditions (other predicates).
func HasIncludedNetworksWith(preds ...predicate.IncludedNetwork) predicate.Environment {
	return predicate.Environment(func(s *sql.Selector) {
		step := newIncludedNetworksStep()
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// HasFindings applies the HasEdge predicate on the "Findings" edge.
func HasFindings() predicate.Environment {
	return predicate.Environment(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.Edge(sqlgraph.O2M, false, FindingsTable, FindingsColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasFindingsWith applies the HasEdge predicate on the "Findings" edge with a given conditions (other predicates).
func HasFindingsWith(preds ...predicate.Finding) predicate.Environment {
	return predicate.Environment(func(s *sql.Selector) {
		step := newFindingsStep()
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// HasDNSRecords applies the HasEdge predicate on the "DNSRecords" edge.
func HasDNSRecords() predicate.Environment {
	return predicate.Environment(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.Edge(sqlgraph.O2M, false, DNSRecordsTable, DNSRecordsColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasDNSRecordsWith applies the HasEdge predicate on the "DNSRecords" edge with a given conditions (other predicates).
func HasDNSRecordsWith(preds ...predicate.DNSRecord) predicate.Environment {
	return predicate.Environment(func(s *sql.Selector) {
		step := newDNSRecordsStep()
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// HasDNS applies the HasEdge predicate on the "DNS" edge.
func HasDNS() predicate.Environment {
	return predicate.Environment(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.Edge(sqlgraph.M2M, false, DNSTable, DNSPrimaryKey...),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasDNSWith applies the HasEdge predicate on the "DNS" edge with a given conditions (other predicates).
func HasDNSWith(preds ...predicate.DNS) predicate.Environment {
	return predicate.Environment(func(s *sql.Selector) {
		step := newDNSStep()
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// HasNetworks applies the HasEdge predicate on the "Networks" edge.
func HasNetworks() predicate.Environment {
	return predicate.Environment(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.Edge(sqlgraph.O2M, false, NetworksTable, NetworksColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasNetworksWith applies the HasEdge predicate on the "Networks" edge with a given conditions (other predicates).
func HasNetworksWith(preds ...predicate.Network) predicate.Environment {
	return predicate.Environment(func(s *sql.Selector) {
		step := newNetworksStep()
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// HasHostDependencies applies the HasEdge predicate on the "HostDependencies" edge.
func HasHostDependencies() predicate.Environment {
	return predicate.Environment(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.Edge(sqlgraph.O2M, false, HostDependenciesTable, HostDependenciesColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasHostDependenciesWith applies the HasEdge predicate on the "HostDependencies" edge with a given conditions (other predicates).
func HasHostDependenciesWith(preds ...predicate.HostDependency) predicate.Environment {
	return predicate.Environment(func(s *sql.Selector) {
		step := newHostDependenciesStep()
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// HasAnsibles applies the HasEdge predicate on the "Ansibles" edge.
func HasAnsibles() predicate.Environment {
	return predicate.Environment(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.Edge(sqlgraph.O2M, false, AnsiblesTable, AnsiblesColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasAnsiblesWith applies the HasEdge predicate on the "Ansibles" edge with a given conditions (other predicates).
func HasAnsiblesWith(preds ...predicate.Ansible) predicate.Environment {
	return predicate.Environment(func(s *sql.Selector) {
		step := newAnsiblesStep()
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// HasScheduledSteps applies the HasEdge predicate on the "ScheduledSteps" edge.
func HasScheduledSteps() predicate.Environment {
	return predicate.Environment(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.Edge(sqlgraph.O2M, false, ScheduledStepsTable, ScheduledStepsColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasScheduledStepsWith applies the HasEdge predicate on the "ScheduledSteps" edge with a given conditions (other predicates).
func HasScheduledStepsWith(preds ...predicate.ScheduledStep) predicate.Environment {
	return predicate.Environment(func(s *sql.Selector) {
		step := newScheduledStepsStep()
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// HasBuilds applies the HasEdge predicate on the "Builds" edge.
func HasBuilds() predicate.Environment {
	return predicate.Environment(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.Edge(sqlgraph.O2M, true, BuildsTable, BuildsColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasBuildsWith applies the HasEdge predicate on the "Builds" edge with a given conditions (other predicates).
func HasBuildsWith(preds ...predicate.Build) predicate.Environment {
	return predicate.Environment(func(s *sql.Selector) {
		step := newBuildsStep()
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// HasRepositories applies the HasEdge predicate on the "Repositories" edge.
func HasRepositories() predicate.Environment {
	return predicate.Environment(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.Edge(sqlgraph.M2M, true, RepositoriesTable, RepositoriesPrimaryKey...),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasRepositoriesWith applies the HasEdge predicate on the "Repositories" edge with a given conditions (other predicates).
func HasRepositoriesWith(preds ...predicate.Repository) predicate.Environment {
	return predicate.Environment(func(s *sql.Selector) {
		step := newRepositoriesStep()
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// HasServerTasks applies the HasEdge predicate on the "ServerTasks" edge.
func HasServerTasks() predicate.Environment {
	return predicate.Environment(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.Edge(sqlgraph.O2M, true, ServerTasksTable, ServerTasksColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasServerTasksWith applies the HasEdge predicate on the "ServerTasks" edge with a given conditions (other predicates).
func HasServerTasksWith(preds ...predicate.ServerTask) predicate.Environment {
	return predicate.Environment(func(s *sql.Selector) {
		step := newServerTasksStep()
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// And groups predicates with the AND operator between them.
func And(predicates ...predicate.Environment) predicate.Environment {
	return predicate.Environment(sql.AndPredicates(predicates...))
}

// Or groups predicates with the OR operator between them.
func Or(predicates ...predicate.Environment) predicate.Environment {
	return predicate.Environment(sql.OrPredicates(predicates...))
}

// Not applies the not operator on the given predicate.
func Not(p predicate.Environment) predicate.Environment {
	return predicate.Environment(sql.NotPredicates(p))
}
