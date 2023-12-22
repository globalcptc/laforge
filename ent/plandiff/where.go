// Code generated by ent, DO NOT EDIT.

package plandiff

import (
	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"github.com/gen0cide/laforge/ent/predicate"
	"github.com/google/uuid"
)

// ID filters vertices based on their ID field.
func ID(id uuid.UUID) predicate.PlanDiff {
	return predicate.PlanDiff(sql.FieldEQ(FieldID, id))
}

// IDEQ applies the EQ predicate on the ID field.
func IDEQ(id uuid.UUID) predicate.PlanDiff {
	return predicate.PlanDiff(sql.FieldEQ(FieldID, id))
}

// IDNEQ applies the NEQ predicate on the ID field.
func IDNEQ(id uuid.UUID) predicate.PlanDiff {
	return predicate.PlanDiff(sql.FieldNEQ(FieldID, id))
}

// IDIn applies the In predicate on the ID field.
func IDIn(ids ...uuid.UUID) predicate.PlanDiff {
	return predicate.PlanDiff(sql.FieldIn(FieldID, ids...))
}

// IDNotIn applies the NotIn predicate on the ID field.
func IDNotIn(ids ...uuid.UUID) predicate.PlanDiff {
	return predicate.PlanDiff(sql.FieldNotIn(FieldID, ids...))
}

// IDGT applies the GT predicate on the ID field.
func IDGT(id uuid.UUID) predicate.PlanDiff {
	return predicate.PlanDiff(sql.FieldGT(FieldID, id))
}

// IDGTE applies the GTE predicate on the ID field.
func IDGTE(id uuid.UUID) predicate.PlanDiff {
	return predicate.PlanDiff(sql.FieldGTE(FieldID, id))
}

// IDLT applies the LT predicate on the ID field.
func IDLT(id uuid.UUID) predicate.PlanDiff {
	return predicate.PlanDiff(sql.FieldLT(FieldID, id))
}

// IDLTE applies the LTE predicate on the ID field.
func IDLTE(id uuid.UUID) predicate.PlanDiff {
	return predicate.PlanDiff(sql.FieldLTE(FieldID, id))
}

// Revision applies equality check predicate on the "revision" field. It's identical to RevisionEQ.
func Revision(v int) predicate.PlanDiff {
	return predicate.PlanDiff(sql.FieldEQ(FieldRevision, v))
}

// RevisionEQ applies the EQ predicate on the "revision" field.
func RevisionEQ(v int) predicate.PlanDiff {
	return predicate.PlanDiff(sql.FieldEQ(FieldRevision, v))
}

// RevisionNEQ applies the NEQ predicate on the "revision" field.
func RevisionNEQ(v int) predicate.PlanDiff {
	return predicate.PlanDiff(sql.FieldNEQ(FieldRevision, v))
}

// RevisionIn applies the In predicate on the "revision" field.
func RevisionIn(vs ...int) predicate.PlanDiff {
	return predicate.PlanDiff(sql.FieldIn(FieldRevision, vs...))
}

// RevisionNotIn applies the NotIn predicate on the "revision" field.
func RevisionNotIn(vs ...int) predicate.PlanDiff {
	return predicate.PlanDiff(sql.FieldNotIn(FieldRevision, vs...))
}

// RevisionGT applies the GT predicate on the "revision" field.
func RevisionGT(v int) predicate.PlanDiff {
	return predicate.PlanDiff(sql.FieldGT(FieldRevision, v))
}

// RevisionGTE applies the GTE predicate on the "revision" field.
func RevisionGTE(v int) predicate.PlanDiff {
	return predicate.PlanDiff(sql.FieldGTE(FieldRevision, v))
}

// RevisionLT applies the LT predicate on the "revision" field.
func RevisionLT(v int) predicate.PlanDiff {
	return predicate.PlanDiff(sql.FieldLT(FieldRevision, v))
}

// RevisionLTE applies the LTE predicate on the "revision" field.
func RevisionLTE(v int) predicate.PlanDiff {
	return predicate.PlanDiff(sql.FieldLTE(FieldRevision, v))
}

// NewStateEQ applies the EQ predicate on the "new_state" field.
func NewStateEQ(v NewState) predicate.PlanDiff {
	return predicate.PlanDiff(sql.FieldEQ(FieldNewState, v))
}

// NewStateNEQ applies the NEQ predicate on the "new_state" field.
func NewStateNEQ(v NewState) predicate.PlanDiff {
	return predicate.PlanDiff(sql.FieldNEQ(FieldNewState, v))
}

// NewStateIn applies the In predicate on the "new_state" field.
func NewStateIn(vs ...NewState) predicate.PlanDiff {
	return predicate.PlanDiff(sql.FieldIn(FieldNewState, vs...))
}

// NewStateNotIn applies the NotIn predicate on the "new_state" field.
func NewStateNotIn(vs ...NewState) predicate.PlanDiff {
	return predicate.PlanDiff(sql.FieldNotIn(FieldNewState, vs...))
}

// HasPlanDiffToBuildCommit applies the HasEdge predicate on the "PlanDiffToBuildCommit" edge.
func HasPlanDiffToBuildCommit() predicate.PlanDiff {
	return predicate.PlanDiff(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.Edge(sqlgraph.M2O, false, PlanDiffToBuildCommitTable, PlanDiffToBuildCommitColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasPlanDiffToBuildCommitWith applies the HasEdge predicate on the "PlanDiffToBuildCommit" edge with a given conditions (other predicates).
func HasPlanDiffToBuildCommitWith(preds ...predicate.BuildCommit) predicate.PlanDiff {
	return predicate.PlanDiff(func(s *sql.Selector) {
		step := newPlanDiffToBuildCommitStep()
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// HasPlanDiffToPlan applies the HasEdge predicate on the "PlanDiffToPlan" edge.
func HasPlanDiffToPlan() predicate.PlanDiff {
	return predicate.PlanDiff(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.Edge(sqlgraph.M2O, false, PlanDiffToPlanTable, PlanDiffToPlanColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasPlanDiffToPlanWith applies the HasEdge predicate on the "PlanDiffToPlan" edge with a given conditions (other predicates).
func HasPlanDiffToPlanWith(preds ...predicate.Plan) predicate.PlanDiff {
	return predicate.PlanDiff(func(s *sql.Selector) {
		step := newPlanDiffToPlanStep()
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// And groups predicates with the AND operator between them.
func And(predicates ...predicate.PlanDiff) predicate.PlanDiff {
	return predicate.PlanDiff(sql.AndPredicates(predicates...))
}

// Or groups predicates with the OR operator between them.
func Or(predicates ...predicate.PlanDiff) predicate.PlanDiff {
	return predicate.PlanDiff(sql.OrPredicates(predicates...))
}

// Not applies the not operator on the given predicate.
func Not(p predicate.PlanDiff) predicate.PlanDiff {
	return predicate.PlanDiff(sql.NotPredicates(p))
}
