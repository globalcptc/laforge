// Code generated by ent, DO NOT EDIT.

package dns

import (
	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"github.com/gen0cide/laforge/ent/predicate"
	"github.com/google/uuid"
)

// ID filters vertices based on their ID field.
func ID(id uuid.UUID) predicate.DNS {
	return predicate.DNS(sql.FieldEQ(FieldID, id))
}

// IDEQ applies the EQ predicate on the ID field.
func IDEQ(id uuid.UUID) predicate.DNS {
	return predicate.DNS(sql.FieldEQ(FieldID, id))
}

// IDNEQ applies the NEQ predicate on the ID field.
func IDNEQ(id uuid.UUID) predicate.DNS {
	return predicate.DNS(sql.FieldNEQ(FieldID, id))
}

// IDIn applies the In predicate on the ID field.
func IDIn(ids ...uuid.UUID) predicate.DNS {
	return predicate.DNS(sql.FieldIn(FieldID, ids...))
}

// IDNotIn applies the NotIn predicate on the ID field.
func IDNotIn(ids ...uuid.UUID) predicate.DNS {
	return predicate.DNS(sql.FieldNotIn(FieldID, ids...))
}

// IDGT applies the GT predicate on the ID field.
func IDGT(id uuid.UUID) predicate.DNS {
	return predicate.DNS(sql.FieldGT(FieldID, id))
}

// IDGTE applies the GTE predicate on the ID field.
func IDGTE(id uuid.UUID) predicate.DNS {
	return predicate.DNS(sql.FieldGTE(FieldID, id))
}

// IDLT applies the LT predicate on the ID field.
func IDLT(id uuid.UUID) predicate.DNS {
	return predicate.DNS(sql.FieldLT(FieldID, id))
}

// IDLTE applies the LTE predicate on the ID field.
func IDLTE(id uuid.UUID) predicate.DNS {
	return predicate.DNS(sql.FieldLTE(FieldID, id))
}

// HCLID applies equality check predicate on the "hcl_id" field. It's identical to HCLIDEQ.
func HCLID(v string) predicate.DNS {
	return predicate.DNS(sql.FieldEQ(FieldHCLID, v))
}

// Type applies equality check predicate on the "type" field. It's identical to TypeEQ.
func Type(v string) predicate.DNS {
	return predicate.DNS(sql.FieldEQ(FieldType, v))
}

// RootDomain applies equality check predicate on the "root_domain" field. It's identical to RootDomainEQ.
func RootDomain(v string) predicate.DNS {
	return predicate.DNS(sql.FieldEQ(FieldRootDomain, v))
}

// HCLIDEQ applies the EQ predicate on the "hcl_id" field.
func HCLIDEQ(v string) predicate.DNS {
	return predicate.DNS(sql.FieldEQ(FieldHCLID, v))
}

// HCLIDNEQ applies the NEQ predicate on the "hcl_id" field.
func HCLIDNEQ(v string) predicate.DNS {
	return predicate.DNS(sql.FieldNEQ(FieldHCLID, v))
}

// HCLIDIn applies the In predicate on the "hcl_id" field.
func HCLIDIn(vs ...string) predicate.DNS {
	return predicate.DNS(sql.FieldIn(FieldHCLID, vs...))
}

// HCLIDNotIn applies the NotIn predicate on the "hcl_id" field.
func HCLIDNotIn(vs ...string) predicate.DNS {
	return predicate.DNS(sql.FieldNotIn(FieldHCLID, vs...))
}

// HCLIDGT applies the GT predicate on the "hcl_id" field.
func HCLIDGT(v string) predicate.DNS {
	return predicate.DNS(sql.FieldGT(FieldHCLID, v))
}

// HCLIDGTE applies the GTE predicate on the "hcl_id" field.
func HCLIDGTE(v string) predicate.DNS {
	return predicate.DNS(sql.FieldGTE(FieldHCLID, v))
}

// HCLIDLT applies the LT predicate on the "hcl_id" field.
func HCLIDLT(v string) predicate.DNS {
	return predicate.DNS(sql.FieldLT(FieldHCLID, v))
}

// HCLIDLTE applies the LTE predicate on the "hcl_id" field.
func HCLIDLTE(v string) predicate.DNS {
	return predicate.DNS(sql.FieldLTE(FieldHCLID, v))
}

// HCLIDContains applies the Contains predicate on the "hcl_id" field.
func HCLIDContains(v string) predicate.DNS {
	return predicate.DNS(sql.FieldContains(FieldHCLID, v))
}

// HCLIDHasPrefix applies the HasPrefix predicate on the "hcl_id" field.
func HCLIDHasPrefix(v string) predicate.DNS {
	return predicate.DNS(sql.FieldHasPrefix(FieldHCLID, v))
}

// HCLIDHasSuffix applies the HasSuffix predicate on the "hcl_id" field.
func HCLIDHasSuffix(v string) predicate.DNS {
	return predicate.DNS(sql.FieldHasSuffix(FieldHCLID, v))
}

// HCLIDEqualFold applies the EqualFold predicate on the "hcl_id" field.
func HCLIDEqualFold(v string) predicate.DNS {
	return predicate.DNS(sql.FieldEqualFold(FieldHCLID, v))
}

// HCLIDContainsFold applies the ContainsFold predicate on the "hcl_id" field.
func HCLIDContainsFold(v string) predicate.DNS {
	return predicate.DNS(sql.FieldContainsFold(FieldHCLID, v))
}

// TypeEQ applies the EQ predicate on the "type" field.
func TypeEQ(v string) predicate.DNS {
	return predicate.DNS(sql.FieldEQ(FieldType, v))
}

// TypeNEQ applies the NEQ predicate on the "type" field.
func TypeNEQ(v string) predicate.DNS {
	return predicate.DNS(sql.FieldNEQ(FieldType, v))
}

// TypeIn applies the In predicate on the "type" field.
func TypeIn(vs ...string) predicate.DNS {
	return predicate.DNS(sql.FieldIn(FieldType, vs...))
}

// TypeNotIn applies the NotIn predicate on the "type" field.
func TypeNotIn(vs ...string) predicate.DNS {
	return predicate.DNS(sql.FieldNotIn(FieldType, vs...))
}

// TypeGT applies the GT predicate on the "type" field.
func TypeGT(v string) predicate.DNS {
	return predicate.DNS(sql.FieldGT(FieldType, v))
}

// TypeGTE applies the GTE predicate on the "type" field.
func TypeGTE(v string) predicate.DNS {
	return predicate.DNS(sql.FieldGTE(FieldType, v))
}

// TypeLT applies the LT predicate on the "type" field.
func TypeLT(v string) predicate.DNS {
	return predicate.DNS(sql.FieldLT(FieldType, v))
}

// TypeLTE applies the LTE predicate on the "type" field.
func TypeLTE(v string) predicate.DNS {
	return predicate.DNS(sql.FieldLTE(FieldType, v))
}

// TypeContains applies the Contains predicate on the "type" field.
func TypeContains(v string) predicate.DNS {
	return predicate.DNS(sql.FieldContains(FieldType, v))
}

// TypeHasPrefix applies the HasPrefix predicate on the "type" field.
func TypeHasPrefix(v string) predicate.DNS {
	return predicate.DNS(sql.FieldHasPrefix(FieldType, v))
}

// TypeHasSuffix applies the HasSuffix predicate on the "type" field.
func TypeHasSuffix(v string) predicate.DNS {
	return predicate.DNS(sql.FieldHasSuffix(FieldType, v))
}

// TypeEqualFold applies the EqualFold predicate on the "type" field.
func TypeEqualFold(v string) predicate.DNS {
	return predicate.DNS(sql.FieldEqualFold(FieldType, v))
}

// TypeContainsFold applies the ContainsFold predicate on the "type" field.
func TypeContainsFold(v string) predicate.DNS {
	return predicate.DNS(sql.FieldContainsFold(FieldType, v))
}

// RootDomainEQ applies the EQ predicate on the "root_domain" field.
func RootDomainEQ(v string) predicate.DNS {
	return predicate.DNS(sql.FieldEQ(FieldRootDomain, v))
}

// RootDomainNEQ applies the NEQ predicate on the "root_domain" field.
func RootDomainNEQ(v string) predicate.DNS {
	return predicate.DNS(sql.FieldNEQ(FieldRootDomain, v))
}

// RootDomainIn applies the In predicate on the "root_domain" field.
func RootDomainIn(vs ...string) predicate.DNS {
	return predicate.DNS(sql.FieldIn(FieldRootDomain, vs...))
}

// RootDomainNotIn applies the NotIn predicate on the "root_domain" field.
func RootDomainNotIn(vs ...string) predicate.DNS {
	return predicate.DNS(sql.FieldNotIn(FieldRootDomain, vs...))
}

// RootDomainGT applies the GT predicate on the "root_domain" field.
func RootDomainGT(v string) predicate.DNS {
	return predicate.DNS(sql.FieldGT(FieldRootDomain, v))
}

// RootDomainGTE applies the GTE predicate on the "root_domain" field.
func RootDomainGTE(v string) predicate.DNS {
	return predicate.DNS(sql.FieldGTE(FieldRootDomain, v))
}

// RootDomainLT applies the LT predicate on the "root_domain" field.
func RootDomainLT(v string) predicate.DNS {
	return predicate.DNS(sql.FieldLT(FieldRootDomain, v))
}

// RootDomainLTE applies the LTE predicate on the "root_domain" field.
func RootDomainLTE(v string) predicate.DNS {
	return predicate.DNS(sql.FieldLTE(FieldRootDomain, v))
}

// RootDomainContains applies the Contains predicate on the "root_domain" field.
func RootDomainContains(v string) predicate.DNS {
	return predicate.DNS(sql.FieldContains(FieldRootDomain, v))
}

// RootDomainHasPrefix applies the HasPrefix predicate on the "root_domain" field.
func RootDomainHasPrefix(v string) predicate.DNS {
	return predicate.DNS(sql.FieldHasPrefix(FieldRootDomain, v))
}

// RootDomainHasSuffix applies the HasSuffix predicate on the "root_domain" field.
func RootDomainHasSuffix(v string) predicate.DNS {
	return predicate.DNS(sql.FieldHasSuffix(FieldRootDomain, v))
}

// RootDomainEqualFold applies the EqualFold predicate on the "root_domain" field.
func RootDomainEqualFold(v string) predicate.DNS {
	return predicate.DNS(sql.FieldEqualFold(FieldRootDomain, v))
}

// RootDomainContainsFold applies the ContainsFold predicate on the "root_domain" field.
func RootDomainContainsFold(v string) predicate.DNS {
	return predicate.DNS(sql.FieldContainsFold(FieldRootDomain, v))
}

// HasEnvironments applies the HasEdge predicate on the "Environments" edge.
func HasEnvironments() predicate.DNS {
	return predicate.DNS(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.Edge(sqlgraph.M2M, true, EnvironmentsTable, EnvironmentsPrimaryKey...),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasEnvironmentsWith applies the HasEdge predicate on the "Environments" edge with a given conditions (other predicates).
func HasEnvironmentsWith(preds ...predicate.Environment) predicate.DNS {
	return predicate.DNS(func(s *sql.Selector) {
		step := newEnvironmentsStep()
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// HasCompetitions applies the HasEdge predicate on the "Competitions" edge.
func HasCompetitions() predicate.DNS {
	return predicate.DNS(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.Edge(sqlgraph.M2M, true, CompetitionsTable, CompetitionsPrimaryKey...),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasCompetitionsWith applies the HasEdge predicate on the "Competitions" edge with a given conditions (other predicates).
func HasCompetitionsWith(preds ...predicate.Competition) predicate.DNS {
	return predicate.DNS(func(s *sql.Selector) {
		step := newCompetitionsStep()
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// And groups predicates with the AND operator between them.
func And(predicates ...predicate.DNS) predicate.DNS {
	return predicate.DNS(sql.AndPredicates(predicates...))
}

// Or groups predicates with the OR operator between them.
func Or(predicates ...predicate.DNS) predicate.DNS {
	return predicate.DNS(sql.OrPredicates(predicates...))
}

// Not applies the not operator on the given predicate.
func Not(p predicate.DNS) predicate.DNS {
	return predicate.DNS(sql.NotPredicates(p))
}
