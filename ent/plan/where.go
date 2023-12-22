// Code generated by ent, DO NOT EDIT.

package plan

import (
	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"github.com/gen0cide/laforge/ent/predicate"
	"github.com/google/uuid"
)

// ID filters vertices based on their ID field.
func ID(id uuid.UUID) predicate.Plan {
	return predicate.Plan(sql.FieldEQ(FieldID, id))
}

// IDEQ applies the EQ predicate on the ID field.
func IDEQ(id uuid.UUID) predicate.Plan {
	return predicate.Plan(sql.FieldEQ(FieldID, id))
}

// IDNEQ applies the NEQ predicate on the ID field.
func IDNEQ(id uuid.UUID) predicate.Plan {
	return predicate.Plan(sql.FieldNEQ(FieldID, id))
}

// IDIn applies the In predicate on the ID field.
func IDIn(ids ...uuid.UUID) predicate.Plan {
	return predicate.Plan(sql.FieldIn(FieldID, ids...))
}

// IDNotIn applies the NotIn predicate on the ID field.
func IDNotIn(ids ...uuid.UUID) predicate.Plan {
	return predicate.Plan(sql.FieldNotIn(FieldID, ids...))
}

// IDGT applies the GT predicate on the ID field.
func IDGT(id uuid.UUID) predicate.Plan {
	return predicate.Plan(sql.FieldGT(FieldID, id))
}

// IDGTE applies the GTE predicate on the ID field.
func IDGTE(id uuid.UUID) predicate.Plan {
	return predicate.Plan(sql.FieldGTE(FieldID, id))
}

// IDLT applies the LT predicate on the ID field.
func IDLT(id uuid.UUID) predicate.Plan {
	return predicate.Plan(sql.FieldLT(FieldID, id))
}

// IDLTE applies the LTE predicate on the ID field.
func IDLTE(id uuid.UUID) predicate.Plan {
	return predicate.Plan(sql.FieldLTE(FieldID, id))
}

// StepNumber applies equality check predicate on the "step_number" field. It's identical to StepNumberEQ.
func StepNumber(v int) predicate.Plan {
	return predicate.Plan(sql.FieldEQ(FieldStepNumber, v))
}

// StepNumberEQ applies the EQ predicate on the "step_number" field.
func StepNumberEQ(v int) predicate.Plan {
	return predicate.Plan(sql.FieldEQ(FieldStepNumber, v))
}

// StepNumberNEQ applies the NEQ predicate on the "step_number" field.
func StepNumberNEQ(v int) predicate.Plan {
	return predicate.Plan(sql.FieldNEQ(FieldStepNumber, v))
}

// StepNumberIn applies the In predicate on the "step_number" field.
func StepNumberIn(vs ...int) predicate.Plan {
	return predicate.Plan(sql.FieldIn(FieldStepNumber, vs...))
}

// StepNumberNotIn applies the NotIn predicate on the "step_number" field.
func StepNumberNotIn(vs ...int) predicate.Plan {
	return predicate.Plan(sql.FieldNotIn(FieldStepNumber, vs...))
}

// StepNumberGT applies the GT predicate on the "step_number" field.
func StepNumberGT(v int) predicate.Plan {
	return predicate.Plan(sql.FieldGT(FieldStepNumber, v))
}

// StepNumberGTE applies the GTE predicate on the "step_number" field.
func StepNumberGTE(v int) predicate.Plan {
	return predicate.Plan(sql.FieldGTE(FieldStepNumber, v))
}

// StepNumberLT applies the LT predicate on the "step_number" field.
func StepNumberLT(v int) predicate.Plan {
	return predicate.Plan(sql.FieldLT(FieldStepNumber, v))
}

// StepNumberLTE applies the LTE predicate on the "step_number" field.
func StepNumberLTE(v int) predicate.Plan {
	return predicate.Plan(sql.FieldLTE(FieldStepNumber, v))
}

// TypeEQ applies the EQ predicate on the "type" field.
func TypeEQ(v Type) predicate.Plan {
	return predicate.Plan(sql.FieldEQ(FieldType, v))
}

// TypeNEQ applies the NEQ predicate on the "type" field.
func TypeNEQ(v Type) predicate.Plan {
	return predicate.Plan(sql.FieldNEQ(FieldType, v))
}

// TypeIn applies the In predicate on the "type" field.
func TypeIn(vs ...Type) predicate.Plan {
	return predicate.Plan(sql.FieldIn(FieldType, vs...))
}

// TypeNotIn applies the NotIn predicate on the "type" field.
func TypeNotIn(vs ...Type) predicate.Plan {
	return predicate.Plan(sql.FieldNotIn(FieldType, vs...))
}

// HasPrevPlans applies the HasEdge predicate on the "PrevPlans" edge.
func HasPrevPlans() predicate.Plan {
	return predicate.Plan(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.Edge(sqlgraph.M2M, true, PrevPlansTable, PrevPlansPrimaryKey...),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasPrevPlansWith applies the HasEdge predicate on the "PrevPlans" edge with a given conditions (other predicates).
func HasPrevPlansWith(preds ...predicate.Plan) predicate.Plan {
	return predicate.Plan(func(s *sql.Selector) {
		step := newPrevPlansStep()
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// HasNextPlans applies the HasEdge predicate on the "NextPlans" edge.
func HasNextPlans() predicate.Plan {
	return predicate.Plan(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.Edge(sqlgraph.M2M, false, NextPlansTable, NextPlansPrimaryKey...),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasNextPlansWith applies the HasEdge predicate on the "NextPlans" edge with a given conditions (other predicates).
func HasNextPlansWith(preds ...predicate.Plan) predicate.Plan {
	return predicate.Plan(func(s *sql.Selector) {
		step := newNextPlansStep()
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// HasBuild applies the HasEdge predicate on the "Build" edge.
func HasBuild() predicate.Plan {
	return predicate.Plan(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.Edge(sqlgraph.M2O, false, BuildTable, BuildColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasBuildWith applies the HasEdge predicate on the "Build" edge with a given conditions (other predicates).
func HasBuildWith(preds ...predicate.Build) predicate.Plan {
	return predicate.Plan(func(s *sql.Selector) {
		step := newBuildStep()
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// HasTeam applies the HasEdge predicate on the "Team" edge.
func HasTeam() predicate.Plan {
	return predicate.Plan(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.Edge(sqlgraph.O2O, false, TeamTable, TeamColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasTeamWith applies the HasEdge predicate on the "Team" edge with a given conditions (other predicates).
func HasTeamWith(preds ...predicate.Team) predicate.Plan {
	return predicate.Plan(func(s *sql.Selector) {
		step := newTeamStep()
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// HasProvisionedNetwork applies the HasEdge predicate on the "ProvisionedNetwork" edge.
func HasProvisionedNetwork() predicate.Plan {
	return predicate.Plan(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.Edge(sqlgraph.O2O, false, ProvisionedNetworkTable, ProvisionedNetworkColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasProvisionedNetworkWith applies the HasEdge predicate on the "ProvisionedNetwork" edge with a given conditions (other predicates).
func HasProvisionedNetworkWith(preds ...predicate.ProvisionedNetwork) predicate.Plan {
	return predicate.Plan(func(s *sql.Selector) {
		step := newProvisionedNetworkStep()
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// HasProvisionedHost applies the HasEdge predicate on the "ProvisionedHost" edge.
func HasProvisionedHost() predicate.Plan {
	return predicate.Plan(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.Edge(sqlgraph.O2O, false, ProvisionedHostTable, ProvisionedHostColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasProvisionedHostWith applies the HasEdge predicate on the "ProvisionedHost" edge with a given conditions (other predicates).
func HasProvisionedHostWith(preds ...predicate.ProvisionedHost) predicate.Plan {
	return predicate.Plan(func(s *sql.Selector) {
		step := newProvisionedHostStep()
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// HasProvisioningStep applies the HasEdge predicate on the "ProvisioningStep" edge.
func HasProvisioningStep() predicate.Plan {
	return predicate.Plan(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.Edge(sqlgraph.O2O, false, ProvisioningStepTable, ProvisioningStepColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasProvisioningStepWith applies the HasEdge predicate on the "ProvisioningStep" edge with a given conditions (other predicates).
func HasProvisioningStepWith(preds ...predicate.ProvisioningStep) predicate.Plan {
	return predicate.Plan(func(s *sql.Selector) {
		step := newProvisioningStepStep()
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// HasProvisioningScheduledStep applies the HasEdge predicate on the "ProvisioningScheduledStep" edge.
func HasProvisioningScheduledStep() predicate.Plan {
	return predicate.Plan(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.Edge(sqlgraph.O2O, false, ProvisioningScheduledStepTable, ProvisioningScheduledStepColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasProvisioningScheduledStepWith applies the HasEdge predicate on the "ProvisioningScheduledStep" edge with a given conditions (other predicates).
func HasProvisioningScheduledStepWith(preds ...predicate.ProvisioningScheduledStep) predicate.Plan {
	return predicate.Plan(func(s *sql.Selector) {
		step := newProvisioningScheduledStepStep()
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// HasStatus applies the HasEdge predicate on the "Status" edge.
func HasStatus() predicate.Plan {
	return predicate.Plan(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.Edge(sqlgraph.O2O, false, StatusTable, StatusColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasStatusWith applies the HasEdge predicate on the "Status" edge with a given conditions (other predicates).
func HasStatusWith(preds ...predicate.Status) predicate.Plan {
	return predicate.Plan(func(s *sql.Selector) {
		step := newStatusStep()
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// HasPlanDiffs applies the HasEdge predicate on the "PlanDiffs" edge.
func HasPlanDiffs() predicate.Plan {
	return predicate.Plan(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.Edge(sqlgraph.O2M, true, PlanDiffsTable, PlanDiffsColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasPlanDiffsWith applies the HasEdge predicate on the "PlanDiffs" edge with a given conditions (other predicates).
func HasPlanDiffsWith(preds ...predicate.PlanDiff) predicate.Plan {
	return predicate.Plan(func(s *sql.Selector) {
		step := newPlanDiffsStep()
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// And groups predicates with the AND operator between them.
func And(predicates ...predicate.Plan) predicate.Plan {
	return predicate.Plan(sql.AndPredicates(predicates...))
}

// Or groups predicates with the OR operator between them.
func Or(predicates ...predicate.Plan) predicate.Plan {
	return predicate.Plan(sql.OrPredicates(predicates...))
}

// Not applies the not operator on the given predicate.
func Not(p predicate.Plan) predicate.Plan {
	return predicate.Plan(sql.NotPredicates(p))
}
