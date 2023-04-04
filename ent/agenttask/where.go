// Code generated by ent, DO NOT EDIT.

package agenttask

import (
	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"github.com/gen0cide/laforge/ent/predicate"
	"github.com/google/uuid"
)

// ID filters vertices based on their ID field.
func ID(id uuid.UUID) predicate.AgentTask {
	return predicate.AgentTask(func(s *sql.Selector) {
		s.Where(sql.EQ(s.C(FieldID), id))
	})
}

// IDEQ applies the EQ predicate on the ID field.
func IDEQ(id uuid.UUID) predicate.AgentTask {
	return predicate.AgentTask(func(s *sql.Selector) {
		s.Where(sql.EQ(s.C(FieldID), id))
	})
}

// IDNEQ applies the NEQ predicate on the ID field.
func IDNEQ(id uuid.UUID) predicate.AgentTask {
	return predicate.AgentTask(func(s *sql.Selector) {
		s.Where(sql.NEQ(s.C(FieldID), id))
	})
}

// IDIn applies the In predicate on the ID field.
func IDIn(ids ...uuid.UUID) predicate.AgentTask {
	return predicate.AgentTask(func(s *sql.Selector) {
		v := make([]interface{}, len(ids))
		for i := range v {
			v[i] = ids[i]
		}
		s.Where(sql.In(s.C(FieldID), v...))
	})
}

// IDNotIn applies the NotIn predicate on the ID field.
func IDNotIn(ids ...uuid.UUID) predicate.AgentTask {
	return predicate.AgentTask(func(s *sql.Selector) {
		v := make([]interface{}, len(ids))
		for i := range v {
			v[i] = ids[i]
		}
		s.Where(sql.NotIn(s.C(FieldID), v...))
	})
}

// IDGT applies the GT predicate on the ID field.
func IDGT(id uuid.UUID) predicate.AgentTask {
	return predicate.AgentTask(func(s *sql.Selector) {
		s.Where(sql.GT(s.C(FieldID), id))
	})
}

// IDGTE applies the GTE predicate on the ID field.
func IDGTE(id uuid.UUID) predicate.AgentTask {
	return predicate.AgentTask(func(s *sql.Selector) {
		s.Where(sql.GTE(s.C(FieldID), id))
	})
}

// IDLT applies the LT predicate on the ID field.
func IDLT(id uuid.UUID) predicate.AgentTask {
	return predicate.AgentTask(func(s *sql.Selector) {
		s.Where(sql.LT(s.C(FieldID), id))
	})
}

// IDLTE applies the LTE predicate on the ID field.
func IDLTE(id uuid.UUID) predicate.AgentTask {
	return predicate.AgentTask(func(s *sql.Selector) {
		s.Where(sql.LTE(s.C(FieldID), id))
	})
}

// Args applies equality check predicate on the "args" field. It's identical to ArgsEQ.
func Args(v string) predicate.AgentTask {
	return predicate.AgentTask(func(s *sql.Selector) {
		s.Where(sql.EQ(s.C(FieldArgs), v))
	})
}

// Number applies equality check predicate on the "number" field. It's identical to NumberEQ.
func Number(v int) predicate.AgentTask {
	return predicate.AgentTask(func(s *sql.Selector) {
		s.Where(sql.EQ(s.C(FieldNumber), v))
	})
}

// Output applies equality check predicate on the "output" field. It's identical to OutputEQ.
func Output(v string) predicate.AgentTask {
	return predicate.AgentTask(func(s *sql.Selector) {
		s.Where(sql.EQ(s.C(FieldOutput), v))
	})
}

// ErrorMessage applies equality check predicate on the "error_message" field. It's identical to ErrorMessageEQ.
func ErrorMessage(v string) predicate.AgentTask {
	return predicate.AgentTask(func(s *sql.Selector) {
		s.Where(sql.EQ(s.C(FieldErrorMessage), v))
	})
}

// CommandEQ applies the EQ predicate on the "command" field.
func CommandEQ(v Command) predicate.AgentTask {
	return predicate.AgentTask(func(s *sql.Selector) {
		s.Where(sql.EQ(s.C(FieldCommand), v))
	})
}

// CommandNEQ applies the NEQ predicate on the "command" field.
func CommandNEQ(v Command) predicate.AgentTask {
	return predicate.AgentTask(func(s *sql.Selector) {
		s.Where(sql.NEQ(s.C(FieldCommand), v))
	})
}

// CommandIn applies the In predicate on the "command" field.
func CommandIn(vs ...Command) predicate.AgentTask {
	v := make([]interface{}, len(vs))
	for i := range v {
		v[i] = vs[i]
	}
	return predicate.AgentTask(func(s *sql.Selector) {
		s.Where(sql.In(s.C(FieldCommand), v...))
	})
}

// CommandNotIn applies the NotIn predicate on the "command" field.
func CommandNotIn(vs ...Command) predicate.AgentTask {
	v := make([]interface{}, len(vs))
	for i := range v {
		v[i] = vs[i]
	}
	return predicate.AgentTask(func(s *sql.Selector) {
		s.Where(sql.NotIn(s.C(FieldCommand), v...))
	})
}

// ArgsEQ applies the EQ predicate on the "args" field.
func ArgsEQ(v string) predicate.AgentTask {
	return predicate.AgentTask(func(s *sql.Selector) {
		s.Where(sql.EQ(s.C(FieldArgs), v))
	})
}

// ArgsNEQ applies the NEQ predicate on the "args" field.
func ArgsNEQ(v string) predicate.AgentTask {
	return predicate.AgentTask(func(s *sql.Selector) {
		s.Where(sql.NEQ(s.C(FieldArgs), v))
	})
}

// ArgsIn applies the In predicate on the "args" field.
func ArgsIn(vs ...string) predicate.AgentTask {
	v := make([]interface{}, len(vs))
	for i := range v {
		v[i] = vs[i]
	}
	return predicate.AgentTask(func(s *sql.Selector) {
		s.Where(sql.In(s.C(FieldArgs), v...))
	})
}

// ArgsNotIn applies the NotIn predicate on the "args" field.
func ArgsNotIn(vs ...string) predicate.AgentTask {
	v := make([]interface{}, len(vs))
	for i := range v {
		v[i] = vs[i]
	}
	return predicate.AgentTask(func(s *sql.Selector) {
		s.Where(sql.NotIn(s.C(FieldArgs), v...))
	})
}

// ArgsGT applies the GT predicate on the "args" field.
func ArgsGT(v string) predicate.AgentTask {
	return predicate.AgentTask(func(s *sql.Selector) {
		s.Where(sql.GT(s.C(FieldArgs), v))
	})
}

// ArgsGTE applies the GTE predicate on the "args" field.
func ArgsGTE(v string) predicate.AgentTask {
	return predicate.AgentTask(func(s *sql.Selector) {
		s.Where(sql.GTE(s.C(FieldArgs), v))
	})
}

// ArgsLT applies the LT predicate on the "args" field.
func ArgsLT(v string) predicate.AgentTask {
	return predicate.AgentTask(func(s *sql.Selector) {
		s.Where(sql.LT(s.C(FieldArgs), v))
	})
}

// ArgsLTE applies the LTE predicate on the "args" field.
func ArgsLTE(v string) predicate.AgentTask {
	return predicate.AgentTask(func(s *sql.Selector) {
		s.Where(sql.LTE(s.C(FieldArgs), v))
	})
}

// ArgsContains applies the Contains predicate on the "args" field.
func ArgsContains(v string) predicate.AgentTask {
	return predicate.AgentTask(func(s *sql.Selector) {
		s.Where(sql.Contains(s.C(FieldArgs), v))
	})
}

// ArgsHasPrefix applies the HasPrefix predicate on the "args" field.
func ArgsHasPrefix(v string) predicate.AgentTask {
	return predicate.AgentTask(func(s *sql.Selector) {
		s.Where(sql.HasPrefix(s.C(FieldArgs), v))
	})
}

// ArgsHasSuffix applies the HasSuffix predicate on the "args" field.
func ArgsHasSuffix(v string) predicate.AgentTask {
	return predicate.AgentTask(func(s *sql.Selector) {
		s.Where(sql.HasSuffix(s.C(FieldArgs), v))
	})
}

// ArgsEqualFold applies the EqualFold predicate on the "args" field.
func ArgsEqualFold(v string) predicate.AgentTask {
	return predicate.AgentTask(func(s *sql.Selector) {
		s.Where(sql.EqualFold(s.C(FieldArgs), v))
	})
}

// ArgsContainsFold applies the ContainsFold predicate on the "args" field.
func ArgsContainsFold(v string) predicate.AgentTask {
	return predicate.AgentTask(func(s *sql.Selector) {
		s.Where(sql.ContainsFold(s.C(FieldArgs), v))
	})
}

// NumberEQ applies the EQ predicate on the "number" field.
func NumberEQ(v int) predicate.AgentTask {
	return predicate.AgentTask(func(s *sql.Selector) {
		s.Where(sql.EQ(s.C(FieldNumber), v))
	})
}

// NumberNEQ applies the NEQ predicate on the "number" field.
func NumberNEQ(v int) predicate.AgentTask {
	return predicate.AgentTask(func(s *sql.Selector) {
		s.Where(sql.NEQ(s.C(FieldNumber), v))
	})
}

// NumberIn applies the In predicate on the "number" field.
func NumberIn(vs ...int) predicate.AgentTask {
	v := make([]interface{}, len(vs))
	for i := range v {
		v[i] = vs[i]
	}
	return predicate.AgentTask(func(s *sql.Selector) {
		s.Where(sql.In(s.C(FieldNumber), v...))
	})
}

// NumberNotIn applies the NotIn predicate on the "number" field.
func NumberNotIn(vs ...int) predicate.AgentTask {
	v := make([]interface{}, len(vs))
	for i := range v {
		v[i] = vs[i]
	}
	return predicate.AgentTask(func(s *sql.Selector) {
		s.Where(sql.NotIn(s.C(FieldNumber), v...))
	})
}

// NumberGT applies the GT predicate on the "number" field.
func NumberGT(v int) predicate.AgentTask {
	return predicate.AgentTask(func(s *sql.Selector) {
		s.Where(sql.GT(s.C(FieldNumber), v))
	})
}

// NumberGTE applies the GTE predicate on the "number" field.
func NumberGTE(v int) predicate.AgentTask {
	return predicate.AgentTask(func(s *sql.Selector) {
		s.Where(sql.GTE(s.C(FieldNumber), v))
	})
}

// NumberLT applies the LT predicate on the "number" field.
func NumberLT(v int) predicate.AgentTask {
	return predicate.AgentTask(func(s *sql.Selector) {
		s.Where(sql.LT(s.C(FieldNumber), v))
	})
}

// NumberLTE applies the LTE predicate on the "number" field.
func NumberLTE(v int) predicate.AgentTask {
	return predicate.AgentTask(func(s *sql.Selector) {
		s.Where(sql.LTE(s.C(FieldNumber), v))
	})
}

// OutputEQ applies the EQ predicate on the "output" field.
func OutputEQ(v string) predicate.AgentTask {
	return predicate.AgentTask(func(s *sql.Selector) {
		s.Where(sql.EQ(s.C(FieldOutput), v))
	})
}

// OutputNEQ applies the NEQ predicate on the "output" field.
func OutputNEQ(v string) predicate.AgentTask {
	return predicate.AgentTask(func(s *sql.Selector) {
		s.Where(sql.NEQ(s.C(FieldOutput), v))
	})
}

// OutputIn applies the In predicate on the "output" field.
func OutputIn(vs ...string) predicate.AgentTask {
	v := make([]interface{}, len(vs))
	for i := range v {
		v[i] = vs[i]
	}
	return predicate.AgentTask(func(s *sql.Selector) {
		s.Where(sql.In(s.C(FieldOutput), v...))
	})
}

// OutputNotIn applies the NotIn predicate on the "output" field.
func OutputNotIn(vs ...string) predicate.AgentTask {
	v := make([]interface{}, len(vs))
	for i := range v {
		v[i] = vs[i]
	}
	return predicate.AgentTask(func(s *sql.Selector) {
		s.Where(sql.NotIn(s.C(FieldOutput), v...))
	})
}

// OutputGT applies the GT predicate on the "output" field.
func OutputGT(v string) predicate.AgentTask {
	return predicate.AgentTask(func(s *sql.Selector) {
		s.Where(sql.GT(s.C(FieldOutput), v))
	})
}

// OutputGTE applies the GTE predicate on the "output" field.
func OutputGTE(v string) predicate.AgentTask {
	return predicate.AgentTask(func(s *sql.Selector) {
		s.Where(sql.GTE(s.C(FieldOutput), v))
	})
}

// OutputLT applies the LT predicate on the "output" field.
func OutputLT(v string) predicate.AgentTask {
	return predicate.AgentTask(func(s *sql.Selector) {
		s.Where(sql.LT(s.C(FieldOutput), v))
	})
}

// OutputLTE applies the LTE predicate on the "output" field.
func OutputLTE(v string) predicate.AgentTask {
	return predicate.AgentTask(func(s *sql.Selector) {
		s.Where(sql.LTE(s.C(FieldOutput), v))
	})
}

// OutputContains applies the Contains predicate on the "output" field.
func OutputContains(v string) predicate.AgentTask {
	return predicate.AgentTask(func(s *sql.Selector) {
		s.Where(sql.Contains(s.C(FieldOutput), v))
	})
}

// OutputHasPrefix applies the HasPrefix predicate on the "output" field.
func OutputHasPrefix(v string) predicate.AgentTask {
	return predicate.AgentTask(func(s *sql.Selector) {
		s.Where(sql.HasPrefix(s.C(FieldOutput), v))
	})
}

// OutputHasSuffix applies the HasSuffix predicate on the "output" field.
func OutputHasSuffix(v string) predicate.AgentTask {
	return predicate.AgentTask(func(s *sql.Selector) {
		s.Where(sql.HasSuffix(s.C(FieldOutput), v))
	})
}

// OutputEqualFold applies the EqualFold predicate on the "output" field.
func OutputEqualFold(v string) predicate.AgentTask {
	return predicate.AgentTask(func(s *sql.Selector) {
		s.Where(sql.EqualFold(s.C(FieldOutput), v))
	})
}

// OutputContainsFold applies the ContainsFold predicate on the "output" field.
func OutputContainsFold(v string) predicate.AgentTask {
	return predicate.AgentTask(func(s *sql.Selector) {
		s.Where(sql.ContainsFold(s.C(FieldOutput), v))
	})
}

// StateEQ applies the EQ predicate on the "state" field.
func StateEQ(v State) predicate.AgentTask {
	return predicate.AgentTask(func(s *sql.Selector) {
		s.Where(sql.EQ(s.C(FieldState), v))
	})
}

// StateNEQ applies the NEQ predicate on the "state" field.
func StateNEQ(v State) predicate.AgentTask {
	return predicate.AgentTask(func(s *sql.Selector) {
		s.Where(sql.NEQ(s.C(FieldState), v))
	})
}

// StateIn applies the In predicate on the "state" field.
func StateIn(vs ...State) predicate.AgentTask {
	v := make([]interface{}, len(vs))
	for i := range v {
		v[i] = vs[i]
	}
	return predicate.AgentTask(func(s *sql.Selector) {
		s.Where(sql.In(s.C(FieldState), v...))
	})
}

// StateNotIn applies the NotIn predicate on the "state" field.
func StateNotIn(vs ...State) predicate.AgentTask {
	v := make([]interface{}, len(vs))
	for i := range v {
		v[i] = vs[i]
	}
	return predicate.AgentTask(func(s *sql.Selector) {
		s.Where(sql.NotIn(s.C(FieldState), v...))
	})
}

// ErrorMessageEQ applies the EQ predicate on the "error_message" field.
func ErrorMessageEQ(v string) predicate.AgentTask {
	return predicate.AgentTask(func(s *sql.Selector) {
		s.Where(sql.EQ(s.C(FieldErrorMessage), v))
	})
}

// ErrorMessageNEQ applies the NEQ predicate on the "error_message" field.
func ErrorMessageNEQ(v string) predicate.AgentTask {
	return predicate.AgentTask(func(s *sql.Selector) {
		s.Where(sql.NEQ(s.C(FieldErrorMessage), v))
	})
}

// ErrorMessageIn applies the In predicate on the "error_message" field.
func ErrorMessageIn(vs ...string) predicate.AgentTask {
	v := make([]interface{}, len(vs))
	for i := range v {
		v[i] = vs[i]
	}
	return predicate.AgentTask(func(s *sql.Selector) {
		s.Where(sql.In(s.C(FieldErrorMessage), v...))
	})
}

// ErrorMessageNotIn applies the NotIn predicate on the "error_message" field.
func ErrorMessageNotIn(vs ...string) predicate.AgentTask {
	v := make([]interface{}, len(vs))
	for i := range v {
		v[i] = vs[i]
	}
	return predicate.AgentTask(func(s *sql.Selector) {
		s.Where(sql.NotIn(s.C(FieldErrorMessage), v...))
	})
}

// ErrorMessageGT applies the GT predicate on the "error_message" field.
func ErrorMessageGT(v string) predicate.AgentTask {
	return predicate.AgentTask(func(s *sql.Selector) {
		s.Where(sql.GT(s.C(FieldErrorMessage), v))
	})
}

// ErrorMessageGTE applies the GTE predicate on the "error_message" field.
func ErrorMessageGTE(v string) predicate.AgentTask {
	return predicate.AgentTask(func(s *sql.Selector) {
		s.Where(sql.GTE(s.C(FieldErrorMessage), v))
	})
}

// ErrorMessageLT applies the LT predicate on the "error_message" field.
func ErrorMessageLT(v string) predicate.AgentTask {
	return predicate.AgentTask(func(s *sql.Selector) {
		s.Where(sql.LT(s.C(FieldErrorMessage), v))
	})
}

// ErrorMessageLTE applies the LTE predicate on the "error_message" field.
func ErrorMessageLTE(v string) predicate.AgentTask {
	return predicate.AgentTask(func(s *sql.Selector) {
		s.Where(sql.LTE(s.C(FieldErrorMessage), v))
	})
}

// ErrorMessageContains applies the Contains predicate on the "error_message" field.
func ErrorMessageContains(v string) predicate.AgentTask {
	return predicate.AgentTask(func(s *sql.Selector) {
		s.Where(sql.Contains(s.C(FieldErrorMessage), v))
	})
}

// ErrorMessageHasPrefix applies the HasPrefix predicate on the "error_message" field.
func ErrorMessageHasPrefix(v string) predicate.AgentTask {
	return predicate.AgentTask(func(s *sql.Selector) {
		s.Where(sql.HasPrefix(s.C(FieldErrorMessage), v))
	})
}

// ErrorMessageHasSuffix applies the HasSuffix predicate on the "error_message" field.
func ErrorMessageHasSuffix(v string) predicate.AgentTask {
	return predicate.AgentTask(func(s *sql.Selector) {
		s.Where(sql.HasSuffix(s.C(FieldErrorMessage), v))
	})
}

// ErrorMessageEqualFold applies the EqualFold predicate on the "error_message" field.
func ErrorMessageEqualFold(v string) predicate.AgentTask {
	return predicate.AgentTask(func(s *sql.Selector) {
		s.Where(sql.EqualFold(s.C(FieldErrorMessage), v))
	})
}

// ErrorMessageContainsFold applies the ContainsFold predicate on the "error_message" field.
func ErrorMessageContainsFold(v string) predicate.AgentTask {
	return predicate.AgentTask(func(s *sql.Selector) {
		s.Where(sql.ContainsFold(s.C(FieldErrorMessage), v))
	})
}

// HasAgentTaskToProvisioningStep applies the HasEdge predicate on the "AgentTaskToProvisioningStep" edge.
func HasAgentTaskToProvisioningStep() predicate.AgentTask {
	return predicate.AgentTask(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.To(AgentTaskToProvisioningStepTable, FieldID),
			sqlgraph.Edge(sqlgraph.M2O, false, AgentTaskToProvisioningStepTable, AgentTaskToProvisioningStepColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasAgentTaskToProvisioningStepWith applies the HasEdge predicate on the "AgentTaskToProvisioningStep" edge with a given conditions (other predicates).
func HasAgentTaskToProvisioningStepWith(preds ...predicate.ProvisioningStep) predicate.AgentTask {
	return predicate.AgentTask(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.To(AgentTaskToProvisioningStepInverseTable, FieldID),
			sqlgraph.Edge(sqlgraph.M2O, false, AgentTaskToProvisioningStepTable, AgentTaskToProvisioningStepColumn),
		)
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// HasAgentTaskToProvisioningScheduledStep applies the HasEdge predicate on the "AgentTaskToProvisioningScheduledStep" edge.
func HasAgentTaskToProvisioningScheduledStep() predicate.AgentTask {
	return predicate.AgentTask(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.To(AgentTaskToProvisioningScheduledStepTable, FieldID),
			sqlgraph.Edge(sqlgraph.O2O, false, AgentTaskToProvisioningScheduledStepTable, AgentTaskToProvisioningScheduledStepColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasAgentTaskToProvisioningScheduledStepWith applies the HasEdge predicate on the "AgentTaskToProvisioningScheduledStep" edge with a given conditions (other predicates).
func HasAgentTaskToProvisioningScheduledStepWith(preds ...predicate.ProvisioningScheduledStep) predicate.AgentTask {
	return predicate.AgentTask(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.To(AgentTaskToProvisioningScheduledStepInverseTable, FieldID),
			sqlgraph.Edge(sqlgraph.O2O, false, AgentTaskToProvisioningScheduledStepTable, AgentTaskToProvisioningScheduledStepColumn),
		)
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// HasAgentTaskToProvisionedHost applies the HasEdge predicate on the "AgentTaskToProvisionedHost" edge.
func HasAgentTaskToProvisionedHost() predicate.AgentTask {
	return predicate.AgentTask(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.To(AgentTaskToProvisionedHostTable, FieldID),
			sqlgraph.Edge(sqlgraph.M2O, false, AgentTaskToProvisionedHostTable, AgentTaskToProvisionedHostColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasAgentTaskToProvisionedHostWith applies the HasEdge predicate on the "AgentTaskToProvisionedHost" edge with a given conditions (other predicates).
func HasAgentTaskToProvisionedHostWith(preds ...predicate.ProvisionedHost) predicate.AgentTask {
	return predicate.AgentTask(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.To(AgentTaskToProvisionedHostInverseTable, FieldID),
			sqlgraph.Edge(sqlgraph.M2O, false, AgentTaskToProvisionedHostTable, AgentTaskToProvisionedHostColumn),
		)
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// HasAgentTaskToAdhocPlan applies the HasEdge predicate on the "AgentTaskToAdhocPlan" edge.
func HasAgentTaskToAdhocPlan() predicate.AgentTask {
	return predicate.AgentTask(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.To(AgentTaskToAdhocPlanTable, FieldID),
			sqlgraph.Edge(sqlgraph.O2M, true, AgentTaskToAdhocPlanTable, AgentTaskToAdhocPlanColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasAgentTaskToAdhocPlanWith applies the HasEdge predicate on the "AgentTaskToAdhocPlan" edge with a given conditions (other predicates).
func HasAgentTaskToAdhocPlanWith(preds ...predicate.AdhocPlan) predicate.AgentTask {
	return predicate.AgentTask(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.To(AgentTaskToAdhocPlanInverseTable, FieldID),
			sqlgraph.Edge(sqlgraph.O2M, true, AgentTaskToAdhocPlanTable, AgentTaskToAdhocPlanColumn),
		)
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// HasAgentTaskToValidation applies the HasEdge predicate on the "AgentTaskToValidation" edge.
func HasAgentTaskToValidation() predicate.AgentTask {
	return predicate.AgentTask(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.To(AgentTaskToValidationTable, FieldID),
			sqlgraph.Edge(sqlgraph.O2O, false, AgentTaskToValidationTable, AgentTaskToValidationColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasAgentTaskToValidationWith applies the HasEdge predicate on the "AgentTaskToValidation" edge with a given conditions (other predicates).
func HasAgentTaskToValidationWith(preds ...predicate.Validation) predicate.AgentTask {
	return predicate.AgentTask(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.To(AgentTaskToValidationInverseTable, FieldID),
			sqlgraph.Edge(sqlgraph.O2O, false, AgentTaskToValidationTable, AgentTaskToValidationColumn),
		)
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// And groups predicates with the AND operator between them.
func And(predicates ...predicate.AgentTask) predicate.AgentTask {
	return predicate.AgentTask(func(s *sql.Selector) {
		s1 := s.Clone().SetP(nil)
		for _, p := range predicates {
			p(s1)
		}
		s.Where(s1.P())
	})
}

// Or groups predicates with the OR operator between them.
func Or(predicates ...predicate.AgentTask) predicate.AgentTask {
	return predicate.AgentTask(func(s *sql.Selector) {
		s1 := s.Clone().SetP(nil)
		for i, p := range predicates {
			if i > 0 {
				s1.Or()
			}
			p(s1)
		}
		s.Where(s1.P())
	})
}

// Not applies the not operator on the given predicate.
func Not(p predicate.AgentTask) predicate.AgentTask {
	return predicate.AgentTask(func(s *sql.Selector) {
		p(s.Not())
	})
}
