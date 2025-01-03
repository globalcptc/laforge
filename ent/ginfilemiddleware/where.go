// Code generated by ent, DO NOT EDIT.

package ginfilemiddleware

import (
	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"github.com/gen0cide/laforge/ent/predicate"
	"github.com/google/uuid"
)

// ID filters vertices based on their ID field.
func ID(id uuid.UUID) predicate.GinFileMiddleware {
	return predicate.GinFileMiddleware(sql.FieldEQ(FieldID, id))
}

// IDEQ applies the EQ predicate on the ID field.
func IDEQ(id uuid.UUID) predicate.GinFileMiddleware {
	return predicate.GinFileMiddleware(sql.FieldEQ(FieldID, id))
}

// IDNEQ applies the NEQ predicate on the ID field.
func IDNEQ(id uuid.UUID) predicate.GinFileMiddleware {
	return predicate.GinFileMiddleware(sql.FieldNEQ(FieldID, id))
}

// IDIn applies the In predicate on the ID field.
func IDIn(ids ...uuid.UUID) predicate.GinFileMiddleware {
	return predicate.GinFileMiddleware(sql.FieldIn(FieldID, ids...))
}

// IDNotIn applies the NotIn predicate on the ID field.
func IDNotIn(ids ...uuid.UUID) predicate.GinFileMiddleware {
	return predicate.GinFileMiddleware(sql.FieldNotIn(FieldID, ids...))
}

// IDGT applies the GT predicate on the ID field.
func IDGT(id uuid.UUID) predicate.GinFileMiddleware {
	return predicate.GinFileMiddleware(sql.FieldGT(FieldID, id))
}

// IDGTE applies the GTE predicate on the ID field.
func IDGTE(id uuid.UUID) predicate.GinFileMiddleware {
	return predicate.GinFileMiddleware(sql.FieldGTE(FieldID, id))
}

// IDLT applies the LT predicate on the ID field.
func IDLT(id uuid.UUID) predicate.GinFileMiddleware {
	return predicate.GinFileMiddleware(sql.FieldLT(FieldID, id))
}

// IDLTE applies the LTE predicate on the ID field.
func IDLTE(id uuid.UUID) predicate.GinFileMiddleware {
	return predicate.GinFileMiddleware(sql.FieldLTE(FieldID, id))
}

// URLID applies equality check predicate on the "url_id" field. It's identical to URLIDEQ.
func URLID(v string) predicate.GinFileMiddleware {
	return predicate.GinFileMiddleware(sql.FieldEQ(FieldURLID, v))
}

// FilePath applies equality check predicate on the "file_path" field. It's identical to FilePathEQ.
func FilePath(v string) predicate.GinFileMiddleware {
	return predicate.GinFileMiddleware(sql.FieldEQ(FieldFilePath, v))
}

// Accessed applies equality check predicate on the "accessed" field. It's identical to AccessedEQ.
func Accessed(v bool) predicate.GinFileMiddleware {
	return predicate.GinFileMiddleware(sql.FieldEQ(FieldAccessed, v))
}

// URLIDEQ applies the EQ predicate on the "url_id" field.
func URLIDEQ(v string) predicate.GinFileMiddleware {
	return predicate.GinFileMiddleware(sql.FieldEQ(FieldURLID, v))
}

// URLIDNEQ applies the NEQ predicate on the "url_id" field.
func URLIDNEQ(v string) predicate.GinFileMiddleware {
	return predicate.GinFileMiddleware(sql.FieldNEQ(FieldURLID, v))
}

// URLIDIn applies the In predicate on the "url_id" field.
func URLIDIn(vs ...string) predicate.GinFileMiddleware {
	return predicate.GinFileMiddleware(sql.FieldIn(FieldURLID, vs...))
}

// URLIDNotIn applies the NotIn predicate on the "url_id" field.
func URLIDNotIn(vs ...string) predicate.GinFileMiddleware {
	return predicate.GinFileMiddleware(sql.FieldNotIn(FieldURLID, vs...))
}

// URLIDGT applies the GT predicate on the "url_id" field.
func URLIDGT(v string) predicate.GinFileMiddleware {
	return predicate.GinFileMiddleware(sql.FieldGT(FieldURLID, v))
}

// URLIDGTE applies the GTE predicate on the "url_id" field.
func URLIDGTE(v string) predicate.GinFileMiddleware {
	return predicate.GinFileMiddleware(sql.FieldGTE(FieldURLID, v))
}

// URLIDLT applies the LT predicate on the "url_id" field.
func URLIDLT(v string) predicate.GinFileMiddleware {
	return predicate.GinFileMiddleware(sql.FieldLT(FieldURLID, v))
}

// URLIDLTE applies the LTE predicate on the "url_id" field.
func URLIDLTE(v string) predicate.GinFileMiddleware {
	return predicate.GinFileMiddleware(sql.FieldLTE(FieldURLID, v))
}

// URLIDContains applies the Contains predicate on the "url_id" field.
func URLIDContains(v string) predicate.GinFileMiddleware {
	return predicate.GinFileMiddleware(sql.FieldContains(FieldURLID, v))
}

// URLIDHasPrefix applies the HasPrefix predicate on the "url_id" field.
func URLIDHasPrefix(v string) predicate.GinFileMiddleware {
	return predicate.GinFileMiddleware(sql.FieldHasPrefix(FieldURLID, v))
}

// URLIDHasSuffix applies the HasSuffix predicate on the "url_id" field.
func URLIDHasSuffix(v string) predicate.GinFileMiddleware {
	return predicate.GinFileMiddleware(sql.FieldHasSuffix(FieldURLID, v))
}

// URLIDEqualFold applies the EqualFold predicate on the "url_id" field.
func URLIDEqualFold(v string) predicate.GinFileMiddleware {
	return predicate.GinFileMiddleware(sql.FieldEqualFold(FieldURLID, v))
}

// URLIDContainsFold applies the ContainsFold predicate on the "url_id" field.
func URLIDContainsFold(v string) predicate.GinFileMiddleware {
	return predicate.GinFileMiddleware(sql.FieldContainsFold(FieldURLID, v))
}

// FilePathEQ applies the EQ predicate on the "file_path" field.
func FilePathEQ(v string) predicate.GinFileMiddleware {
	return predicate.GinFileMiddleware(sql.FieldEQ(FieldFilePath, v))
}

// FilePathNEQ applies the NEQ predicate on the "file_path" field.
func FilePathNEQ(v string) predicate.GinFileMiddleware {
	return predicate.GinFileMiddleware(sql.FieldNEQ(FieldFilePath, v))
}

// FilePathIn applies the In predicate on the "file_path" field.
func FilePathIn(vs ...string) predicate.GinFileMiddleware {
	return predicate.GinFileMiddleware(sql.FieldIn(FieldFilePath, vs...))
}

// FilePathNotIn applies the NotIn predicate on the "file_path" field.
func FilePathNotIn(vs ...string) predicate.GinFileMiddleware {
	return predicate.GinFileMiddleware(sql.FieldNotIn(FieldFilePath, vs...))
}

// FilePathGT applies the GT predicate on the "file_path" field.
func FilePathGT(v string) predicate.GinFileMiddleware {
	return predicate.GinFileMiddleware(sql.FieldGT(FieldFilePath, v))
}

// FilePathGTE applies the GTE predicate on the "file_path" field.
func FilePathGTE(v string) predicate.GinFileMiddleware {
	return predicate.GinFileMiddleware(sql.FieldGTE(FieldFilePath, v))
}

// FilePathLT applies the LT predicate on the "file_path" field.
func FilePathLT(v string) predicate.GinFileMiddleware {
	return predicate.GinFileMiddleware(sql.FieldLT(FieldFilePath, v))
}

// FilePathLTE applies the LTE predicate on the "file_path" field.
func FilePathLTE(v string) predicate.GinFileMiddleware {
	return predicate.GinFileMiddleware(sql.FieldLTE(FieldFilePath, v))
}

// FilePathContains applies the Contains predicate on the "file_path" field.
func FilePathContains(v string) predicate.GinFileMiddleware {
	return predicate.GinFileMiddleware(sql.FieldContains(FieldFilePath, v))
}

// FilePathHasPrefix applies the HasPrefix predicate on the "file_path" field.
func FilePathHasPrefix(v string) predicate.GinFileMiddleware {
	return predicate.GinFileMiddleware(sql.FieldHasPrefix(FieldFilePath, v))
}

// FilePathHasSuffix applies the HasSuffix predicate on the "file_path" field.
func FilePathHasSuffix(v string) predicate.GinFileMiddleware {
	return predicate.GinFileMiddleware(sql.FieldHasSuffix(FieldFilePath, v))
}

// FilePathEqualFold applies the EqualFold predicate on the "file_path" field.
func FilePathEqualFold(v string) predicate.GinFileMiddleware {
	return predicate.GinFileMiddleware(sql.FieldEqualFold(FieldFilePath, v))
}

// FilePathContainsFold applies the ContainsFold predicate on the "file_path" field.
func FilePathContainsFold(v string) predicate.GinFileMiddleware {
	return predicate.GinFileMiddleware(sql.FieldContainsFold(FieldFilePath, v))
}

// AccessedEQ applies the EQ predicate on the "accessed" field.
func AccessedEQ(v bool) predicate.GinFileMiddleware {
	return predicate.GinFileMiddleware(sql.FieldEQ(FieldAccessed, v))
}

// AccessedNEQ applies the NEQ predicate on the "accessed" field.
func AccessedNEQ(v bool) predicate.GinFileMiddleware {
	return predicate.GinFileMiddleware(sql.FieldNEQ(FieldAccessed, v))
}

// HasProvisionedHost applies the HasEdge predicate on the "ProvisionedHost" edge.
func HasProvisionedHost() predicate.GinFileMiddleware {
	return predicate.GinFileMiddleware(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.Edge(sqlgraph.O2O, false, ProvisionedHostTable, ProvisionedHostColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasProvisionedHostWith applies the HasEdge predicate on the "ProvisionedHost" edge with a given conditions (other predicates).
func HasProvisionedHostWith(preds ...predicate.ProvisionedHost) predicate.GinFileMiddleware {
	return predicate.GinFileMiddleware(func(s *sql.Selector) {
		step := newProvisionedHostStep()
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// HasProvisioningStep applies the HasEdge predicate on the "ProvisioningStep" edge.
func HasProvisioningStep() predicate.GinFileMiddleware {
	return predicate.GinFileMiddleware(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.Edge(sqlgraph.O2O, false, ProvisioningStepTable, ProvisioningStepColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasProvisioningStepWith applies the HasEdge predicate on the "ProvisioningStep" edge with a given conditions (other predicates).
func HasProvisioningStepWith(preds ...predicate.ProvisioningStep) predicate.GinFileMiddleware {
	return predicate.GinFileMiddleware(func(s *sql.Selector) {
		step := newProvisioningStepStep()
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// HasProvisioningScheduledStep applies the HasEdge predicate on the "ProvisioningScheduledStep" edge.
func HasProvisioningScheduledStep() predicate.GinFileMiddleware {
	return predicate.GinFileMiddleware(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.Edge(sqlgraph.O2O, false, ProvisioningScheduledStepTable, ProvisioningScheduledStepColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasProvisioningScheduledStepWith applies the HasEdge predicate on the "ProvisioningScheduledStep" edge with a given conditions (other predicates).
func HasProvisioningScheduledStepWith(preds ...predicate.ProvisioningScheduledStep) predicate.GinFileMiddleware {
	return predicate.GinFileMiddleware(func(s *sql.Selector) {
		step := newProvisioningScheduledStepStep()
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// And groups predicates with the AND operator between them.
func And(predicates ...predicate.GinFileMiddleware) predicate.GinFileMiddleware {
	return predicate.GinFileMiddleware(sql.AndPredicates(predicates...))
}

// Or groups predicates with the OR operator between them.
func Or(predicates ...predicate.GinFileMiddleware) predicate.GinFileMiddleware {
	return predicate.GinFileMiddleware(sql.OrPredicates(predicates...))
}

// Not applies the not operator on the given predicate.
func Not(p predicate.GinFileMiddleware) predicate.GinFileMiddleware {
	return predicate.GinFileMiddleware(sql.NotPredicates(p))
}
