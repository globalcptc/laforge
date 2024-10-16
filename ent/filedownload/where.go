// Code generated by ent, DO NOT EDIT.

package filedownload

import (
	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"github.com/gen0cide/laforge/ent/predicate"
	"github.com/google/uuid"
)

// ID filters vertices based on their ID field.
func ID(id uuid.UUID) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldEQ(FieldID, id))
}

// IDEQ applies the EQ predicate on the ID field.
func IDEQ(id uuid.UUID) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldEQ(FieldID, id))
}

// IDNEQ applies the NEQ predicate on the ID field.
func IDNEQ(id uuid.UUID) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldNEQ(FieldID, id))
}

// IDIn applies the In predicate on the ID field.
func IDIn(ids ...uuid.UUID) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldIn(FieldID, ids...))
}

// IDNotIn applies the NotIn predicate on the ID field.
func IDNotIn(ids ...uuid.UUID) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldNotIn(FieldID, ids...))
}

// IDGT applies the GT predicate on the ID field.
func IDGT(id uuid.UUID) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldGT(FieldID, id))
}

// IDGTE applies the GTE predicate on the ID field.
func IDGTE(id uuid.UUID) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldGTE(FieldID, id))
}

// IDLT applies the LT predicate on the ID field.
func IDLT(id uuid.UUID) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldLT(FieldID, id))
}

// IDLTE applies the LTE predicate on the ID field.
func IDLTE(id uuid.UUID) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldLTE(FieldID, id))
}

// HCLID applies equality check predicate on the "hcl_id" field. It's identical to HCLIDEQ.
func HCLID(v string) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldEQ(FieldHCLID, v))
}

// SourceType applies equality check predicate on the "source_type" field. It's identical to SourceTypeEQ.
func SourceType(v string) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldEQ(FieldSourceType, v))
}

// Source applies equality check predicate on the "source" field. It's identical to SourceEQ.
func Source(v string) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldEQ(FieldSource, v))
}

// Destination applies equality check predicate on the "destination" field. It's identical to DestinationEQ.
func Destination(v string) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldEQ(FieldDestination, v))
}

// Template applies equality check predicate on the "template" field. It's identical to TemplateEQ.
func Template(v bool) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldEQ(FieldTemplate, v))
}

// Perms applies equality check predicate on the "perms" field. It's identical to PermsEQ.
func Perms(v string) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldEQ(FieldPerms, v))
}

// Disabled applies equality check predicate on the "disabled" field. It's identical to DisabledEQ.
func Disabled(v bool) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldEQ(FieldDisabled, v))
}

// Md5 applies equality check predicate on the "md5" field. It's identical to Md5EQ.
func Md5(v string) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldEQ(FieldMd5, v))
}

// AbsPath applies equality check predicate on the "abs_path" field. It's identical to AbsPathEQ.
func AbsPath(v string) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldEQ(FieldAbsPath, v))
}

// IsTxt applies equality check predicate on the "is_txt" field. It's identical to IsTxtEQ.
func IsTxt(v bool) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldEQ(FieldIsTxt, v))
}

// HCLIDEQ applies the EQ predicate on the "hcl_id" field.
func HCLIDEQ(v string) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldEQ(FieldHCLID, v))
}

// HCLIDNEQ applies the NEQ predicate on the "hcl_id" field.
func HCLIDNEQ(v string) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldNEQ(FieldHCLID, v))
}

// HCLIDIn applies the In predicate on the "hcl_id" field.
func HCLIDIn(vs ...string) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldIn(FieldHCLID, vs...))
}

// HCLIDNotIn applies the NotIn predicate on the "hcl_id" field.
func HCLIDNotIn(vs ...string) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldNotIn(FieldHCLID, vs...))
}

// HCLIDGT applies the GT predicate on the "hcl_id" field.
func HCLIDGT(v string) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldGT(FieldHCLID, v))
}

// HCLIDGTE applies the GTE predicate on the "hcl_id" field.
func HCLIDGTE(v string) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldGTE(FieldHCLID, v))
}

// HCLIDLT applies the LT predicate on the "hcl_id" field.
func HCLIDLT(v string) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldLT(FieldHCLID, v))
}

// HCLIDLTE applies the LTE predicate on the "hcl_id" field.
func HCLIDLTE(v string) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldLTE(FieldHCLID, v))
}

// HCLIDContains applies the Contains predicate on the "hcl_id" field.
func HCLIDContains(v string) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldContains(FieldHCLID, v))
}

// HCLIDHasPrefix applies the HasPrefix predicate on the "hcl_id" field.
func HCLIDHasPrefix(v string) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldHasPrefix(FieldHCLID, v))
}

// HCLIDHasSuffix applies the HasSuffix predicate on the "hcl_id" field.
func HCLIDHasSuffix(v string) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldHasSuffix(FieldHCLID, v))
}

// HCLIDEqualFold applies the EqualFold predicate on the "hcl_id" field.
func HCLIDEqualFold(v string) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldEqualFold(FieldHCLID, v))
}

// HCLIDContainsFold applies the ContainsFold predicate on the "hcl_id" field.
func HCLIDContainsFold(v string) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldContainsFold(FieldHCLID, v))
}

// SourceTypeEQ applies the EQ predicate on the "source_type" field.
func SourceTypeEQ(v string) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldEQ(FieldSourceType, v))
}

// SourceTypeNEQ applies the NEQ predicate on the "source_type" field.
func SourceTypeNEQ(v string) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldNEQ(FieldSourceType, v))
}

// SourceTypeIn applies the In predicate on the "source_type" field.
func SourceTypeIn(vs ...string) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldIn(FieldSourceType, vs...))
}

// SourceTypeNotIn applies the NotIn predicate on the "source_type" field.
func SourceTypeNotIn(vs ...string) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldNotIn(FieldSourceType, vs...))
}

// SourceTypeGT applies the GT predicate on the "source_type" field.
func SourceTypeGT(v string) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldGT(FieldSourceType, v))
}

// SourceTypeGTE applies the GTE predicate on the "source_type" field.
func SourceTypeGTE(v string) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldGTE(FieldSourceType, v))
}

// SourceTypeLT applies the LT predicate on the "source_type" field.
func SourceTypeLT(v string) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldLT(FieldSourceType, v))
}

// SourceTypeLTE applies the LTE predicate on the "source_type" field.
func SourceTypeLTE(v string) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldLTE(FieldSourceType, v))
}

// SourceTypeContains applies the Contains predicate on the "source_type" field.
func SourceTypeContains(v string) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldContains(FieldSourceType, v))
}

// SourceTypeHasPrefix applies the HasPrefix predicate on the "source_type" field.
func SourceTypeHasPrefix(v string) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldHasPrefix(FieldSourceType, v))
}

// SourceTypeHasSuffix applies the HasSuffix predicate on the "source_type" field.
func SourceTypeHasSuffix(v string) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldHasSuffix(FieldSourceType, v))
}

// SourceTypeEqualFold applies the EqualFold predicate on the "source_type" field.
func SourceTypeEqualFold(v string) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldEqualFold(FieldSourceType, v))
}

// SourceTypeContainsFold applies the ContainsFold predicate on the "source_type" field.
func SourceTypeContainsFold(v string) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldContainsFold(FieldSourceType, v))
}

// SourceEQ applies the EQ predicate on the "source" field.
func SourceEQ(v string) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldEQ(FieldSource, v))
}

// SourceNEQ applies the NEQ predicate on the "source" field.
func SourceNEQ(v string) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldNEQ(FieldSource, v))
}

// SourceIn applies the In predicate on the "source" field.
func SourceIn(vs ...string) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldIn(FieldSource, vs...))
}

// SourceNotIn applies the NotIn predicate on the "source" field.
func SourceNotIn(vs ...string) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldNotIn(FieldSource, vs...))
}

// SourceGT applies the GT predicate on the "source" field.
func SourceGT(v string) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldGT(FieldSource, v))
}

// SourceGTE applies the GTE predicate on the "source" field.
func SourceGTE(v string) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldGTE(FieldSource, v))
}

// SourceLT applies the LT predicate on the "source" field.
func SourceLT(v string) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldLT(FieldSource, v))
}

// SourceLTE applies the LTE predicate on the "source" field.
func SourceLTE(v string) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldLTE(FieldSource, v))
}

// SourceContains applies the Contains predicate on the "source" field.
func SourceContains(v string) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldContains(FieldSource, v))
}

// SourceHasPrefix applies the HasPrefix predicate on the "source" field.
func SourceHasPrefix(v string) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldHasPrefix(FieldSource, v))
}

// SourceHasSuffix applies the HasSuffix predicate on the "source" field.
func SourceHasSuffix(v string) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldHasSuffix(FieldSource, v))
}

// SourceEqualFold applies the EqualFold predicate on the "source" field.
func SourceEqualFold(v string) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldEqualFold(FieldSource, v))
}

// SourceContainsFold applies the ContainsFold predicate on the "source" field.
func SourceContainsFold(v string) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldContainsFold(FieldSource, v))
}

// DestinationEQ applies the EQ predicate on the "destination" field.
func DestinationEQ(v string) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldEQ(FieldDestination, v))
}

// DestinationNEQ applies the NEQ predicate on the "destination" field.
func DestinationNEQ(v string) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldNEQ(FieldDestination, v))
}

// DestinationIn applies the In predicate on the "destination" field.
func DestinationIn(vs ...string) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldIn(FieldDestination, vs...))
}

// DestinationNotIn applies the NotIn predicate on the "destination" field.
func DestinationNotIn(vs ...string) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldNotIn(FieldDestination, vs...))
}

// DestinationGT applies the GT predicate on the "destination" field.
func DestinationGT(v string) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldGT(FieldDestination, v))
}

// DestinationGTE applies the GTE predicate on the "destination" field.
func DestinationGTE(v string) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldGTE(FieldDestination, v))
}

// DestinationLT applies the LT predicate on the "destination" field.
func DestinationLT(v string) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldLT(FieldDestination, v))
}

// DestinationLTE applies the LTE predicate on the "destination" field.
func DestinationLTE(v string) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldLTE(FieldDestination, v))
}

// DestinationContains applies the Contains predicate on the "destination" field.
func DestinationContains(v string) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldContains(FieldDestination, v))
}

// DestinationHasPrefix applies the HasPrefix predicate on the "destination" field.
func DestinationHasPrefix(v string) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldHasPrefix(FieldDestination, v))
}

// DestinationHasSuffix applies the HasSuffix predicate on the "destination" field.
func DestinationHasSuffix(v string) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldHasSuffix(FieldDestination, v))
}

// DestinationEqualFold applies the EqualFold predicate on the "destination" field.
func DestinationEqualFold(v string) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldEqualFold(FieldDestination, v))
}

// DestinationContainsFold applies the ContainsFold predicate on the "destination" field.
func DestinationContainsFold(v string) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldContainsFold(FieldDestination, v))
}

// TemplateEQ applies the EQ predicate on the "template" field.
func TemplateEQ(v bool) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldEQ(FieldTemplate, v))
}

// TemplateNEQ applies the NEQ predicate on the "template" field.
func TemplateNEQ(v bool) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldNEQ(FieldTemplate, v))
}

// PermsEQ applies the EQ predicate on the "perms" field.
func PermsEQ(v string) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldEQ(FieldPerms, v))
}

// PermsNEQ applies the NEQ predicate on the "perms" field.
func PermsNEQ(v string) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldNEQ(FieldPerms, v))
}

// PermsIn applies the In predicate on the "perms" field.
func PermsIn(vs ...string) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldIn(FieldPerms, vs...))
}

// PermsNotIn applies the NotIn predicate on the "perms" field.
func PermsNotIn(vs ...string) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldNotIn(FieldPerms, vs...))
}

// PermsGT applies the GT predicate on the "perms" field.
func PermsGT(v string) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldGT(FieldPerms, v))
}

// PermsGTE applies the GTE predicate on the "perms" field.
func PermsGTE(v string) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldGTE(FieldPerms, v))
}

// PermsLT applies the LT predicate on the "perms" field.
func PermsLT(v string) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldLT(FieldPerms, v))
}

// PermsLTE applies the LTE predicate on the "perms" field.
func PermsLTE(v string) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldLTE(FieldPerms, v))
}

// PermsContains applies the Contains predicate on the "perms" field.
func PermsContains(v string) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldContains(FieldPerms, v))
}

// PermsHasPrefix applies the HasPrefix predicate on the "perms" field.
func PermsHasPrefix(v string) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldHasPrefix(FieldPerms, v))
}

// PermsHasSuffix applies the HasSuffix predicate on the "perms" field.
func PermsHasSuffix(v string) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldHasSuffix(FieldPerms, v))
}

// PermsEqualFold applies the EqualFold predicate on the "perms" field.
func PermsEqualFold(v string) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldEqualFold(FieldPerms, v))
}

// PermsContainsFold applies the ContainsFold predicate on the "perms" field.
func PermsContainsFold(v string) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldContainsFold(FieldPerms, v))
}

// DisabledEQ applies the EQ predicate on the "disabled" field.
func DisabledEQ(v bool) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldEQ(FieldDisabled, v))
}

// DisabledNEQ applies the NEQ predicate on the "disabled" field.
func DisabledNEQ(v bool) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldNEQ(FieldDisabled, v))
}

// Md5EQ applies the EQ predicate on the "md5" field.
func Md5EQ(v string) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldEQ(FieldMd5, v))
}

// Md5NEQ applies the NEQ predicate on the "md5" field.
func Md5NEQ(v string) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldNEQ(FieldMd5, v))
}

// Md5In applies the In predicate on the "md5" field.
func Md5In(vs ...string) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldIn(FieldMd5, vs...))
}

// Md5NotIn applies the NotIn predicate on the "md5" field.
func Md5NotIn(vs ...string) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldNotIn(FieldMd5, vs...))
}

// Md5GT applies the GT predicate on the "md5" field.
func Md5GT(v string) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldGT(FieldMd5, v))
}

// Md5GTE applies the GTE predicate on the "md5" field.
func Md5GTE(v string) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldGTE(FieldMd5, v))
}

// Md5LT applies the LT predicate on the "md5" field.
func Md5LT(v string) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldLT(FieldMd5, v))
}

// Md5LTE applies the LTE predicate on the "md5" field.
func Md5LTE(v string) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldLTE(FieldMd5, v))
}

// Md5Contains applies the Contains predicate on the "md5" field.
func Md5Contains(v string) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldContains(FieldMd5, v))
}

// Md5HasPrefix applies the HasPrefix predicate on the "md5" field.
func Md5HasPrefix(v string) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldHasPrefix(FieldMd5, v))
}

// Md5HasSuffix applies the HasSuffix predicate on the "md5" field.
func Md5HasSuffix(v string) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldHasSuffix(FieldMd5, v))
}

// Md5EqualFold applies the EqualFold predicate on the "md5" field.
func Md5EqualFold(v string) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldEqualFold(FieldMd5, v))
}

// Md5ContainsFold applies the ContainsFold predicate on the "md5" field.
func Md5ContainsFold(v string) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldContainsFold(FieldMd5, v))
}

// AbsPathEQ applies the EQ predicate on the "abs_path" field.
func AbsPathEQ(v string) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldEQ(FieldAbsPath, v))
}

// AbsPathNEQ applies the NEQ predicate on the "abs_path" field.
func AbsPathNEQ(v string) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldNEQ(FieldAbsPath, v))
}

// AbsPathIn applies the In predicate on the "abs_path" field.
func AbsPathIn(vs ...string) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldIn(FieldAbsPath, vs...))
}

// AbsPathNotIn applies the NotIn predicate on the "abs_path" field.
func AbsPathNotIn(vs ...string) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldNotIn(FieldAbsPath, vs...))
}

// AbsPathGT applies the GT predicate on the "abs_path" field.
func AbsPathGT(v string) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldGT(FieldAbsPath, v))
}

// AbsPathGTE applies the GTE predicate on the "abs_path" field.
func AbsPathGTE(v string) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldGTE(FieldAbsPath, v))
}

// AbsPathLT applies the LT predicate on the "abs_path" field.
func AbsPathLT(v string) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldLT(FieldAbsPath, v))
}

// AbsPathLTE applies the LTE predicate on the "abs_path" field.
func AbsPathLTE(v string) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldLTE(FieldAbsPath, v))
}

// AbsPathContains applies the Contains predicate on the "abs_path" field.
func AbsPathContains(v string) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldContains(FieldAbsPath, v))
}

// AbsPathHasPrefix applies the HasPrefix predicate on the "abs_path" field.
func AbsPathHasPrefix(v string) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldHasPrefix(FieldAbsPath, v))
}

// AbsPathHasSuffix applies the HasSuffix predicate on the "abs_path" field.
func AbsPathHasSuffix(v string) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldHasSuffix(FieldAbsPath, v))
}

// AbsPathEqualFold applies the EqualFold predicate on the "abs_path" field.
func AbsPathEqualFold(v string) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldEqualFold(FieldAbsPath, v))
}

// AbsPathContainsFold applies the ContainsFold predicate on the "abs_path" field.
func AbsPathContainsFold(v string) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldContainsFold(FieldAbsPath, v))
}

// IsTxtEQ applies the EQ predicate on the "is_txt" field.
func IsTxtEQ(v bool) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldEQ(FieldIsTxt, v))
}

// IsTxtNEQ applies the NEQ predicate on the "is_txt" field.
func IsTxtNEQ(v bool) predicate.FileDownload {
	return predicate.FileDownload(sql.FieldNEQ(FieldIsTxt, v))
}

// HasEnvironment applies the HasEdge predicate on the "Environment" edge.
func HasEnvironment() predicate.FileDownload {
	return predicate.FileDownload(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.Edge(sqlgraph.M2O, true, EnvironmentTable, EnvironmentColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasEnvironmentWith applies the HasEdge predicate on the "Environment" edge with a given conditions (other predicates).
func HasEnvironmentWith(preds ...predicate.Environment) predicate.FileDownload {
	return predicate.FileDownload(func(s *sql.Selector) {
		step := newEnvironmentStep()
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// And groups predicates with the AND operator between them.
func And(predicates ...predicate.FileDownload) predicate.FileDownload {
	return predicate.FileDownload(sql.AndPredicates(predicates...))
}

// Or groups predicates with the OR operator between them.
func Or(predicates ...predicate.FileDownload) predicate.FileDownload {
	return predicate.FileDownload(sql.OrPredicates(predicates...))
}

// Not applies the not operator on the given predicate.
func Not(p predicate.FileDownload) predicate.FileDownload {
	return predicate.FileDownload(sql.NotPredicates(p))
}
