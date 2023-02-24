// Code generated by ent, DO NOT EDIT.

package provisioningscheduledstep

import (
	"time"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"github.com/gen0cide/laforge/ent/predicate"
	"github.com/google/uuid"
)

// ID filters vertices based on their ID field.
func ID(id uuid.UUID) predicate.ProvisioningScheduledStep {
	return predicate.ProvisioningScheduledStep(func(s *sql.Selector) {
		s.Where(sql.EQ(s.C(FieldID), id))
	})
}

// IDEQ applies the EQ predicate on the ID field.
func IDEQ(id uuid.UUID) predicate.ProvisioningScheduledStep {
	return predicate.ProvisioningScheduledStep(func(s *sql.Selector) {
		s.Where(sql.EQ(s.C(FieldID), id))
	})
}

// IDNEQ applies the NEQ predicate on the ID field.
func IDNEQ(id uuid.UUID) predicate.ProvisioningScheduledStep {
	return predicate.ProvisioningScheduledStep(func(s *sql.Selector) {
		s.Where(sql.NEQ(s.C(FieldID), id))
	})
}

// IDIn applies the In predicate on the ID field.
func IDIn(ids ...uuid.UUID) predicate.ProvisioningScheduledStep {
	return predicate.ProvisioningScheduledStep(func(s *sql.Selector) {
		v := make([]interface{}, len(ids))
		for i := range v {
			v[i] = ids[i]
		}
		s.Where(sql.In(s.C(FieldID), v...))
	})
}

// IDNotIn applies the NotIn predicate on the ID field.
func IDNotIn(ids ...uuid.UUID) predicate.ProvisioningScheduledStep {
	return predicate.ProvisioningScheduledStep(func(s *sql.Selector) {
		v := make([]interface{}, len(ids))
		for i := range v {
			v[i] = ids[i]
		}
		s.Where(sql.NotIn(s.C(FieldID), v...))
	})
}

// IDGT applies the GT predicate on the ID field.
func IDGT(id uuid.UUID) predicate.ProvisioningScheduledStep {
	return predicate.ProvisioningScheduledStep(func(s *sql.Selector) {
		s.Where(sql.GT(s.C(FieldID), id))
	})
}

// IDGTE applies the GTE predicate on the ID field.
func IDGTE(id uuid.UUID) predicate.ProvisioningScheduledStep {
	return predicate.ProvisioningScheduledStep(func(s *sql.Selector) {
		s.Where(sql.GTE(s.C(FieldID), id))
	})
}

// IDLT applies the LT predicate on the ID field.
func IDLT(id uuid.UUID) predicate.ProvisioningScheduledStep {
	return predicate.ProvisioningScheduledStep(func(s *sql.Selector) {
		s.Where(sql.LT(s.C(FieldID), id))
	})
}

// IDLTE applies the LTE predicate on the ID field.
func IDLTE(id uuid.UUID) predicate.ProvisioningScheduledStep {
	return predicate.ProvisioningScheduledStep(func(s *sql.Selector) {
		s.Where(sql.LTE(s.C(FieldID), id))
	})
}

// RunTime applies equality check predicate on the "run_time" field. It's identical to RunTimeEQ.
func RunTime(v time.Time) predicate.ProvisioningScheduledStep {
	return predicate.ProvisioningScheduledStep(func(s *sql.Selector) {
		s.Where(sql.EQ(s.C(FieldRunTime), v))
	})
}

// TypeEQ applies the EQ predicate on the "type" field.
func TypeEQ(v Type) predicate.ProvisioningScheduledStep {
	return predicate.ProvisioningScheduledStep(func(s *sql.Selector) {
		s.Where(sql.EQ(s.C(FieldType), v))
	})
}

// TypeNEQ applies the NEQ predicate on the "type" field.
func TypeNEQ(v Type) predicate.ProvisioningScheduledStep {
	return predicate.ProvisioningScheduledStep(func(s *sql.Selector) {
		s.Where(sql.NEQ(s.C(FieldType), v))
	})
}

// TypeIn applies the In predicate on the "type" field.
func TypeIn(vs ...Type) predicate.ProvisioningScheduledStep {
	v := make([]interface{}, len(vs))
	for i := range v {
		v[i] = vs[i]
	}
	return predicate.ProvisioningScheduledStep(func(s *sql.Selector) {
		s.Where(sql.In(s.C(FieldType), v...))
	})
}

// TypeNotIn applies the NotIn predicate on the "type" field.
func TypeNotIn(vs ...Type) predicate.ProvisioningScheduledStep {
	v := make([]interface{}, len(vs))
	for i := range v {
		v[i] = vs[i]
	}
	return predicate.ProvisioningScheduledStep(func(s *sql.Selector) {
		s.Where(sql.NotIn(s.C(FieldType), v...))
	})
}

// RunTimeEQ applies the EQ predicate on the "run_time" field.
func RunTimeEQ(v time.Time) predicate.ProvisioningScheduledStep {
	return predicate.ProvisioningScheduledStep(func(s *sql.Selector) {
		s.Where(sql.EQ(s.C(FieldRunTime), v))
	})
}

// RunTimeNEQ applies the NEQ predicate on the "run_time" field.
func RunTimeNEQ(v time.Time) predicate.ProvisioningScheduledStep {
	return predicate.ProvisioningScheduledStep(func(s *sql.Selector) {
		s.Where(sql.NEQ(s.C(FieldRunTime), v))
	})
}

// RunTimeIn applies the In predicate on the "run_time" field.
func RunTimeIn(vs ...time.Time) predicate.ProvisioningScheduledStep {
	v := make([]interface{}, len(vs))
	for i := range v {
		v[i] = vs[i]
	}
	return predicate.ProvisioningScheduledStep(func(s *sql.Selector) {
		s.Where(sql.In(s.C(FieldRunTime), v...))
	})
}

// RunTimeNotIn applies the NotIn predicate on the "run_time" field.
func RunTimeNotIn(vs ...time.Time) predicate.ProvisioningScheduledStep {
	v := make([]interface{}, len(vs))
	for i := range v {
		v[i] = vs[i]
	}
	return predicate.ProvisioningScheduledStep(func(s *sql.Selector) {
		s.Where(sql.NotIn(s.C(FieldRunTime), v...))
	})
}

// RunTimeGT applies the GT predicate on the "run_time" field.
func RunTimeGT(v time.Time) predicate.ProvisioningScheduledStep {
	return predicate.ProvisioningScheduledStep(func(s *sql.Selector) {
		s.Where(sql.GT(s.C(FieldRunTime), v))
	})
}

// RunTimeGTE applies the GTE predicate on the "run_time" field.
func RunTimeGTE(v time.Time) predicate.ProvisioningScheduledStep {
	return predicate.ProvisioningScheduledStep(func(s *sql.Selector) {
		s.Where(sql.GTE(s.C(FieldRunTime), v))
	})
}

// RunTimeLT applies the LT predicate on the "run_time" field.
func RunTimeLT(v time.Time) predicate.ProvisioningScheduledStep {
	return predicate.ProvisioningScheduledStep(func(s *sql.Selector) {
		s.Where(sql.LT(s.C(FieldRunTime), v))
	})
}

// RunTimeLTE applies the LTE predicate on the "run_time" field.
func RunTimeLTE(v time.Time) predicate.ProvisioningScheduledStep {
	return predicate.ProvisioningScheduledStep(func(s *sql.Selector) {
		s.Where(sql.LTE(s.C(FieldRunTime), v))
	})
}

// HasProvisioningScheduledStepToStatus applies the HasEdge predicate on the "ProvisioningScheduledStepToStatus" edge.
func HasProvisioningScheduledStepToStatus() predicate.ProvisioningScheduledStep {
	return predicate.ProvisioningScheduledStep(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.To(ProvisioningScheduledStepToStatusTable, FieldID),
			sqlgraph.Edge(sqlgraph.O2O, false, ProvisioningScheduledStepToStatusTable, ProvisioningScheduledStepToStatusColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasProvisioningScheduledStepToStatusWith applies the HasEdge predicate on the "ProvisioningScheduledStepToStatus" edge with a given conditions (other predicates).
func HasProvisioningScheduledStepToStatusWith(preds ...predicate.Status) predicate.ProvisioningScheduledStep {
	return predicate.ProvisioningScheduledStep(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.To(ProvisioningScheduledStepToStatusInverseTable, FieldID),
			sqlgraph.Edge(sqlgraph.O2O, false, ProvisioningScheduledStepToStatusTable, ProvisioningScheduledStepToStatusColumn),
		)
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// HasProvisioningScheduledStepToScheduledStep applies the HasEdge predicate on the "ProvisioningScheduledStepToScheduledStep" edge.
func HasProvisioningScheduledStepToScheduledStep() predicate.ProvisioningScheduledStep {
	return predicate.ProvisioningScheduledStep(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.To(ProvisioningScheduledStepToScheduledStepTable, FieldID),
			sqlgraph.Edge(sqlgraph.M2O, false, ProvisioningScheduledStepToScheduledStepTable, ProvisioningScheduledStepToScheduledStepColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasProvisioningScheduledStepToScheduledStepWith applies the HasEdge predicate on the "ProvisioningScheduledStepToScheduledStep" edge with a given conditions (other predicates).
func HasProvisioningScheduledStepToScheduledStepWith(preds ...predicate.ScheduledStep) predicate.ProvisioningScheduledStep {
	return predicate.ProvisioningScheduledStep(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.To(ProvisioningScheduledStepToScheduledStepInverseTable, FieldID),
			sqlgraph.Edge(sqlgraph.M2O, false, ProvisioningScheduledStepToScheduledStepTable, ProvisioningScheduledStepToScheduledStepColumn),
		)
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// HasProvisioningScheduledStepToProvisionedHost applies the HasEdge predicate on the "ProvisioningScheduledStepToProvisionedHost" edge.
func HasProvisioningScheduledStepToProvisionedHost() predicate.ProvisioningScheduledStep {
	return predicate.ProvisioningScheduledStep(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.To(ProvisioningScheduledStepToProvisionedHostTable, FieldID),
			sqlgraph.Edge(sqlgraph.M2O, false, ProvisioningScheduledStepToProvisionedHostTable, ProvisioningScheduledStepToProvisionedHostColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasProvisioningScheduledStepToProvisionedHostWith applies the HasEdge predicate on the "ProvisioningScheduledStepToProvisionedHost" edge with a given conditions (other predicates).
func HasProvisioningScheduledStepToProvisionedHostWith(preds ...predicate.ProvisionedHost) predicate.ProvisioningScheduledStep {
	return predicate.ProvisioningScheduledStep(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.To(ProvisioningScheduledStepToProvisionedHostInverseTable, FieldID),
			sqlgraph.Edge(sqlgraph.M2O, false, ProvisioningScheduledStepToProvisionedHostTable, ProvisioningScheduledStepToProvisionedHostColumn),
		)
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// HasProvisioningScheduledStepToScript applies the HasEdge predicate on the "ProvisioningScheduledStepToScript" edge.
func HasProvisioningScheduledStepToScript() predicate.ProvisioningScheduledStep {
	return predicate.ProvisioningScheduledStep(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.To(ProvisioningScheduledStepToScriptTable, FieldID),
			sqlgraph.Edge(sqlgraph.M2O, false, ProvisioningScheduledStepToScriptTable, ProvisioningScheduledStepToScriptColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasProvisioningScheduledStepToScriptWith applies the HasEdge predicate on the "ProvisioningScheduledStepToScript" edge with a given conditions (other predicates).
func HasProvisioningScheduledStepToScriptWith(preds ...predicate.Script) predicate.ProvisioningScheduledStep {
	return predicate.ProvisioningScheduledStep(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.To(ProvisioningScheduledStepToScriptInverseTable, FieldID),
			sqlgraph.Edge(sqlgraph.M2O, false, ProvisioningScheduledStepToScriptTable, ProvisioningScheduledStepToScriptColumn),
		)
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// HasProvisioningScheduledStepToCommand applies the HasEdge predicate on the "ProvisioningScheduledStepToCommand" edge.
func HasProvisioningScheduledStepToCommand() predicate.ProvisioningScheduledStep {
	return predicate.ProvisioningScheduledStep(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.To(ProvisioningScheduledStepToCommandTable, FieldID),
			sqlgraph.Edge(sqlgraph.M2O, false, ProvisioningScheduledStepToCommandTable, ProvisioningScheduledStepToCommandColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasProvisioningScheduledStepToCommandWith applies the HasEdge predicate on the "ProvisioningScheduledStepToCommand" edge with a given conditions (other predicates).
func HasProvisioningScheduledStepToCommandWith(preds ...predicate.Command) predicate.ProvisioningScheduledStep {
	return predicate.ProvisioningScheduledStep(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.To(ProvisioningScheduledStepToCommandInverseTable, FieldID),
			sqlgraph.Edge(sqlgraph.M2O, false, ProvisioningScheduledStepToCommandTable, ProvisioningScheduledStepToCommandColumn),
		)
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// HasProvisioningScheduledStepToDNSRecord applies the HasEdge predicate on the "ProvisioningScheduledStepToDNSRecord" edge.
func HasProvisioningScheduledStepToDNSRecord() predicate.ProvisioningScheduledStep {
	return predicate.ProvisioningScheduledStep(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.To(ProvisioningScheduledStepToDNSRecordTable, FieldID),
			sqlgraph.Edge(sqlgraph.M2O, false, ProvisioningScheduledStepToDNSRecordTable, ProvisioningScheduledStepToDNSRecordColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasProvisioningScheduledStepToDNSRecordWith applies the HasEdge predicate on the "ProvisioningScheduledStepToDNSRecord" edge with a given conditions (other predicates).
func HasProvisioningScheduledStepToDNSRecordWith(preds ...predicate.DNSRecord) predicate.ProvisioningScheduledStep {
	return predicate.ProvisioningScheduledStep(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.To(ProvisioningScheduledStepToDNSRecordInverseTable, FieldID),
			sqlgraph.Edge(sqlgraph.M2O, false, ProvisioningScheduledStepToDNSRecordTable, ProvisioningScheduledStepToDNSRecordColumn),
		)
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// HasProvisioningScheduledStepToFileDelete applies the HasEdge predicate on the "ProvisioningScheduledStepToFileDelete" edge.
func HasProvisioningScheduledStepToFileDelete() predicate.ProvisioningScheduledStep {
	return predicate.ProvisioningScheduledStep(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.To(ProvisioningScheduledStepToFileDeleteTable, FieldID),
			sqlgraph.Edge(sqlgraph.M2O, false, ProvisioningScheduledStepToFileDeleteTable, ProvisioningScheduledStepToFileDeleteColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasProvisioningScheduledStepToFileDeleteWith applies the HasEdge predicate on the "ProvisioningScheduledStepToFileDelete" edge with a given conditions (other predicates).
func HasProvisioningScheduledStepToFileDeleteWith(preds ...predicate.FileDelete) predicate.ProvisioningScheduledStep {
	return predicate.ProvisioningScheduledStep(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.To(ProvisioningScheduledStepToFileDeleteInverseTable, FieldID),
			sqlgraph.Edge(sqlgraph.M2O, false, ProvisioningScheduledStepToFileDeleteTable, ProvisioningScheduledStepToFileDeleteColumn),
		)
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// HasProvisioningScheduledStepToFileDownload applies the HasEdge predicate on the "ProvisioningScheduledStepToFileDownload" edge.
func HasProvisioningScheduledStepToFileDownload() predicate.ProvisioningScheduledStep {
	return predicate.ProvisioningScheduledStep(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.To(ProvisioningScheduledStepToFileDownloadTable, FieldID),
			sqlgraph.Edge(sqlgraph.M2O, false, ProvisioningScheduledStepToFileDownloadTable, ProvisioningScheduledStepToFileDownloadColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasProvisioningScheduledStepToFileDownloadWith applies the HasEdge predicate on the "ProvisioningScheduledStepToFileDownload" edge with a given conditions (other predicates).
func HasProvisioningScheduledStepToFileDownloadWith(preds ...predicate.FileDownload) predicate.ProvisioningScheduledStep {
	return predicate.ProvisioningScheduledStep(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.To(ProvisioningScheduledStepToFileDownloadInverseTable, FieldID),
			sqlgraph.Edge(sqlgraph.M2O, false, ProvisioningScheduledStepToFileDownloadTable, ProvisioningScheduledStepToFileDownloadColumn),
		)
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// HasProvisioningScheduledStepToFileExtract applies the HasEdge predicate on the "ProvisioningScheduledStepToFileExtract" edge.
func HasProvisioningScheduledStepToFileExtract() predicate.ProvisioningScheduledStep {
	return predicate.ProvisioningScheduledStep(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.To(ProvisioningScheduledStepToFileExtractTable, FieldID),
			sqlgraph.Edge(sqlgraph.M2O, false, ProvisioningScheduledStepToFileExtractTable, ProvisioningScheduledStepToFileExtractColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasProvisioningScheduledStepToFileExtractWith applies the HasEdge predicate on the "ProvisioningScheduledStepToFileExtract" edge with a given conditions (other predicates).
func HasProvisioningScheduledStepToFileExtractWith(preds ...predicate.FileExtract) predicate.ProvisioningScheduledStep {
	return predicate.ProvisioningScheduledStep(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.To(ProvisioningScheduledStepToFileExtractInverseTable, FieldID),
			sqlgraph.Edge(sqlgraph.M2O, false, ProvisioningScheduledStepToFileExtractTable, ProvisioningScheduledStepToFileExtractColumn),
		)
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// HasProvisioningScheduledStepToAnsible applies the HasEdge predicate on the "ProvisioningScheduledStepToAnsible" edge.
func HasProvisioningScheduledStepToAnsible() predicate.ProvisioningScheduledStep {
	return predicate.ProvisioningScheduledStep(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.To(ProvisioningScheduledStepToAnsibleTable, FieldID),
			sqlgraph.Edge(sqlgraph.M2O, false, ProvisioningScheduledStepToAnsibleTable, ProvisioningScheduledStepToAnsibleColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasProvisioningScheduledStepToAnsibleWith applies the HasEdge predicate on the "ProvisioningScheduledStepToAnsible" edge with a given conditions (other predicates).
func HasProvisioningScheduledStepToAnsibleWith(preds ...predicate.Ansible) predicate.ProvisioningScheduledStep {
	return predicate.ProvisioningScheduledStep(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.To(ProvisioningScheduledStepToAnsibleInverseTable, FieldID),
			sqlgraph.Edge(sqlgraph.M2O, false, ProvisioningScheduledStepToAnsibleTable, ProvisioningScheduledStepToAnsibleColumn),
		)
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// HasProvisioningScheduledStepToAgentTask applies the HasEdge predicate on the "ProvisioningScheduledStepToAgentTask" edge.
func HasProvisioningScheduledStepToAgentTask() predicate.ProvisioningScheduledStep {
	return predicate.ProvisioningScheduledStep(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.To(ProvisioningScheduledStepToAgentTaskTable, FieldID),
			sqlgraph.Edge(sqlgraph.O2O, true, ProvisioningScheduledStepToAgentTaskTable, ProvisioningScheduledStepToAgentTaskColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasProvisioningScheduledStepToAgentTaskWith applies the HasEdge predicate on the "ProvisioningScheduledStepToAgentTask" edge with a given conditions (other predicates).
func HasProvisioningScheduledStepToAgentTaskWith(preds ...predicate.AgentTask) predicate.ProvisioningScheduledStep {
	return predicate.ProvisioningScheduledStep(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.To(ProvisioningScheduledStepToAgentTaskInverseTable, FieldID),
			sqlgraph.Edge(sqlgraph.O2O, true, ProvisioningScheduledStepToAgentTaskTable, ProvisioningScheduledStepToAgentTaskColumn),
		)
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// HasProvisioningStepToPlan applies the HasEdge predicate on the "ProvisioningStepToPlan" edge.
func HasProvisioningStepToPlan() predicate.ProvisioningScheduledStep {
	return predicate.ProvisioningScheduledStep(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.To(ProvisioningStepToPlanTable, FieldID),
			sqlgraph.Edge(sqlgraph.O2O, true, ProvisioningStepToPlanTable, ProvisioningStepToPlanColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasProvisioningStepToPlanWith applies the HasEdge predicate on the "ProvisioningStepToPlan" edge with a given conditions (other predicates).
func HasProvisioningStepToPlanWith(preds ...predicate.Plan) predicate.ProvisioningScheduledStep {
	return predicate.ProvisioningScheduledStep(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.To(ProvisioningStepToPlanInverseTable, FieldID),
			sqlgraph.Edge(sqlgraph.O2O, true, ProvisioningStepToPlanTable, ProvisioningStepToPlanColumn),
		)
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// HasProvisioningScheduledStepToGinFileMiddleware applies the HasEdge predicate on the "ProvisioningScheduledStepToGinFileMiddleware" edge.
func HasProvisioningScheduledStepToGinFileMiddleware() predicate.ProvisioningScheduledStep {
	return predicate.ProvisioningScheduledStep(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.To(ProvisioningScheduledStepToGinFileMiddlewareTable, FieldID),
			sqlgraph.Edge(sqlgraph.O2O, true, ProvisioningScheduledStepToGinFileMiddlewareTable, ProvisioningScheduledStepToGinFileMiddlewareColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasProvisioningScheduledStepToGinFileMiddlewareWith applies the HasEdge predicate on the "ProvisioningScheduledStepToGinFileMiddleware" edge with a given conditions (other predicates).
func HasProvisioningScheduledStepToGinFileMiddlewareWith(preds ...predicate.GinFileMiddleware) predicate.ProvisioningScheduledStep {
	return predicate.ProvisioningScheduledStep(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.To(ProvisioningScheduledStepToGinFileMiddlewareInverseTable, FieldID),
			sqlgraph.Edge(sqlgraph.O2O, true, ProvisioningScheduledStepToGinFileMiddlewareTable, ProvisioningScheduledStepToGinFileMiddlewareColumn),
		)
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// And groups predicates with the AND operator between them.
func And(predicates ...predicate.ProvisioningScheduledStep) predicate.ProvisioningScheduledStep {
	return predicate.ProvisioningScheduledStep(func(s *sql.Selector) {
		s1 := s.Clone().SetP(nil)
		for _, p := range predicates {
			p(s1)
		}
		s.Where(s1.P())
	})
}

// Or groups predicates with the OR operator between them.
func Or(predicates ...predicate.ProvisioningScheduledStep) predicate.ProvisioningScheduledStep {
	return predicate.ProvisioningScheduledStep(func(s *sql.Selector) {
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
func Not(p predicate.ProvisioningScheduledStep) predicate.ProvisioningScheduledStep {
	return predicate.ProvisioningScheduledStep(func(s *sql.Selector) {
		p(s.Not())
	})
}
