// Code generated by entc, DO NOT EDIT.

package provisioningstep

import (
	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"github.com/gen0cide/laforge/ent/predicate"
	"github.com/google/uuid"
)

// ID filters vertices based on their ID field.
func ID(id uuid.UUID) predicate.ProvisioningStep {
	return predicate.ProvisioningStep(func(s *sql.Selector) {
		s.Where(sql.EQ(s.C(FieldID), id))
	})
}

// IDEQ applies the EQ predicate on the ID field.
func IDEQ(id uuid.UUID) predicate.ProvisioningStep {
	return predicate.ProvisioningStep(func(s *sql.Selector) {
		s.Where(sql.EQ(s.C(FieldID), id))
	})
}

// IDNEQ applies the NEQ predicate on the ID field.
func IDNEQ(id uuid.UUID) predicate.ProvisioningStep {
	return predicate.ProvisioningStep(func(s *sql.Selector) {
		s.Where(sql.NEQ(s.C(FieldID), id))
	})
}

// IDIn applies the In predicate on the ID field.
func IDIn(ids ...uuid.UUID) predicate.ProvisioningStep {
	return predicate.ProvisioningStep(func(s *sql.Selector) {
		// if not arguments were provided, append the FALSE constants,
		// since we can't apply "IN ()". This will make this predicate falsy.
		if len(ids) == 0 {
			s.Where(sql.False())
			return
		}
		v := make([]interface{}, len(ids))
		for i := range v {
			v[i] = ids[i]
		}
		s.Where(sql.In(s.C(FieldID), v...))
	})
}

// IDNotIn applies the NotIn predicate on the ID field.
func IDNotIn(ids ...uuid.UUID) predicate.ProvisioningStep {
	return predicate.ProvisioningStep(func(s *sql.Selector) {
		// if not arguments were provided, append the FALSE constants,
		// since we can't apply "IN ()". This will make this predicate falsy.
		if len(ids) == 0 {
			s.Where(sql.False())
			return
		}
		v := make([]interface{}, len(ids))
		for i := range v {
			v[i] = ids[i]
		}
		s.Where(sql.NotIn(s.C(FieldID), v...))
	})
}

// IDGT applies the GT predicate on the ID field.
func IDGT(id uuid.UUID) predicate.ProvisioningStep {
	return predicate.ProvisioningStep(func(s *sql.Selector) {
		s.Where(sql.GT(s.C(FieldID), id))
	})
}

// IDGTE applies the GTE predicate on the ID field.
func IDGTE(id uuid.UUID) predicate.ProvisioningStep {
	return predicate.ProvisioningStep(func(s *sql.Selector) {
		s.Where(sql.GTE(s.C(FieldID), id))
	})
}

// IDLT applies the LT predicate on the ID field.
func IDLT(id uuid.UUID) predicate.ProvisioningStep {
	return predicate.ProvisioningStep(func(s *sql.Selector) {
		s.Where(sql.LT(s.C(FieldID), id))
	})
}

// IDLTE applies the LTE predicate on the ID field.
func IDLTE(id uuid.UUID) predicate.ProvisioningStep {
	return predicate.ProvisioningStep(func(s *sql.Selector) {
		s.Where(sql.LTE(s.C(FieldID), id))
	})
}

// StepNumber applies equality check predicate on the "step_number" field. It's identical to StepNumberEQ.
func StepNumber(v int) predicate.ProvisioningStep {
	return predicate.ProvisioningStep(func(s *sql.Selector) {
		s.Where(sql.EQ(s.C(FieldStepNumber), v))
	})
}

// TypeEQ applies the EQ predicate on the "type" field.
func TypeEQ(v Type) predicate.ProvisioningStep {
	return predicate.ProvisioningStep(func(s *sql.Selector) {
		s.Where(sql.EQ(s.C(FieldType), v))
	})
}

// TypeNEQ applies the NEQ predicate on the "type" field.
func TypeNEQ(v Type) predicate.ProvisioningStep {
	return predicate.ProvisioningStep(func(s *sql.Selector) {
		s.Where(sql.NEQ(s.C(FieldType), v))
	})
}

// TypeIn applies the In predicate on the "type" field.
func TypeIn(vs ...Type) predicate.ProvisioningStep {
	v := make([]interface{}, len(vs))
	for i := range v {
		v[i] = vs[i]
	}
	return predicate.ProvisioningStep(func(s *sql.Selector) {
		// if not arguments were provided, append the FALSE constants,
		// since we can't apply "IN ()". This will make this predicate falsy.
		if len(v) == 0 {
			s.Where(sql.False())
			return
		}
		s.Where(sql.In(s.C(FieldType), v...))
	})
}

// TypeNotIn applies the NotIn predicate on the "type" field.
func TypeNotIn(vs ...Type) predicate.ProvisioningStep {
	v := make([]interface{}, len(vs))
	for i := range v {
		v[i] = vs[i]
	}
	return predicate.ProvisioningStep(func(s *sql.Selector) {
		// if not arguments were provided, append the FALSE constants,
		// since we can't apply "IN ()". This will make this predicate falsy.
		if len(v) == 0 {
			s.Where(sql.False())
			return
		}
		s.Where(sql.NotIn(s.C(FieldType), v...))
	})
}

// StepNumberEQ applies the EQ predicate on the "step_number" field.
func StepNumberEQ(v int) predicate.ProvisioningStep {
	return predicate.ProvisioningStep(func(s *sql.Selector) {
		s.Where(sql.EQ(s.C(FieldStepNumber), v))
	})
}

// StepNumberNEQ applies the NEQ predicate on the "step_number" field.
func StepNumberNEQ(v int) predicate.ProvisioningStep {
	return predicate.ProvisioningStep(func(s *sql.Selector) {
		s.Where(sql.NEQ(s.C(FieldStepNumber), v))
	})
}

// StepNumberIn applies the In predicate on the "step_number" field.
func StepNumberIn(vs ...int) predicate.ProvisioningStep {
	v := make([]interface{}, len(vs))
	for i := range v {
		v[i] = vs[i]
	}
	return predicate.ProvisioningStep(func(s *sql.Selector) {
		// if not arguments were provided, append the FALSE constants,
		// since we can't apply "IN ()". This will make this predicate falsy.
		if len(v) == 0 {
			s.Where(sql.False())
			return
		}
		s.Where(sql.In(s.C(FieldStepNumber), v...))
	})
}

// StepNumberNotIn applies the NotIn predicate on the "step_number" field.
func StepNumberNotIn(vs ...int) predicate.ProvisioningStep {
	v := make([]interface{}, len(vs))
	for i := range v {
		v[i] = vs[i]
	}
	return predicate.ProvisioningStep(func(s *sql.Selector) {
		// if not arguments were provided, append the FALSE constants,
		// since we can't apply "IN ()". This will make this predicate falsy.
		if len(v) == 0 {
			s.Where(sql.False())
			return
		}
		s.Where(sql.NotIn(s.C(FieldStepNumber), v...))
	})
}

// StepNumberGT applies the GT predicate on the "step_number" field.
func StepNumberGT(v int) predicate.ProvisioningStep {
	return predicate.ProvisioningStep(func(s *sql.Selector) {
		s.Where(sql.GT(s.C(FieldStepNumber), v))
	})
}

// StepNumberGTE applies the GTE predicate on the "step_number" field.
func StepNumberGTE(v int) predicate.ProvisioningStep {
	return predicate.ProvisioningStep(func(s *sql.Selector) {
		s.Where(sql.GTE(s.C(FieldStepNumber), v))
	})
}

// StepNumberLT applies the LT predicate on the "step_number" field.
func StepNumberLT(v int) predicate.ProvisioningStep {
	return predicate.ProvisioningStep(func(s *sql.Selector) {
		s.Where(sql.LT(s.C(FieldStepNumber), v))
	})
}

// StepNumberLTE applies the LTE predicate on the "step_number" field.
func StepNumberLTE(v int) predicate.ProvisioningStep {
	return predicate.ProvisioningStep(func(s *sql.Selector) {
		s.Where(sql.LTE(s.C(FieldStepNumber), v))
	})
}

// HasProvisioningStepToStatus applies the HasEdge predicate on the "ProvisioningStepToStatus" edge.
func HasProvisioningStepToStatus() predicate.ProvisioningStep {
	return predicate.ProvisioningStep(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.To(ProvisioningStepToStatusTable, FieldID),
			sqlgraph.Edge(sqlgraph.O2O, false, ProvisioningStepToStatusTable, ProvisioningStepToStatusColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasProvisioningStepToStatusWith applies the HasEdge predicate on the "ProvisioningStepToStatus" edge with a given conditions (other predicates).
func HasProvisioningStepToStatusWith(preds ...predicate.Status) predicate.ProvisioningStep {
	return predicate.ProvisioningStep(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.To(ProvisioningStepToStatusInverseTable, FieldID),
			sqlgraph.Edge(sqlgraph.O2O, false, ProvisioningStepToStatusTable, ProvisioningStepToStatusColumn),
		)
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// HasProvisioningStepToProvisionedHost applies the HasEdge predicate on the "ProvisioningStepToProvisionedHost" edge.
func HasProvisioningStepToProvisionedHost() predicate.ProvisioningStep {
	return predicate.ProvisioningStep(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.To(ProvisioningStepToProvisionedHostTable, FieldID),
			sqlgraph.Edge(sqlgraph.M2O, false, ProvisioningStepToProvisionedHostTable, ProvisioningStepToProvisionedHostColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasProvisioningStepToProvisionedHostWith applies the HasEdge predicate on the "ProvisioningStepToProvisionedHost" edge with a given conditions (other predicates).
func HasProvisioningStepToProvisionedHostWith(preds ...predicate.ProvisionedHost) predicate.ProvisioningStep {
	return predicate.ProvisioningStep(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.To(ProvisioningStepToProvisionedHostInverseTable, FieldID),
			sqlgraph.Edge(sqlgraph.M2O, false, ProvisioningStepToProvisionedHostTable, ProvisioningStepToProvisionedHostColumn),
		)
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// HasProvisioningStepToScript applies the HasEdge predicate on the "ProvisioningStepToScript" edge.
func HasProvisioningStepToScript() predicate.ProvisioningStep {
	return predicate.ProvisioningStep(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.To(ProvisioningStepToScriptTable, FieldID),
			sqlgraph.Edge(sqlgraph.M2O, false, ProvisioningStepToScriptTable, ProvisioningStepToScriptColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasProvisioningStepToScriptWith applies the HasEdge predicate on the "ProvisioningStepToScript" edge with a given conditions (other predicates).
func HasProvisioningStepToScriptWith(preds ...predicate.Script) predicate.ProvisioningStep {
	return predicate.ProvisioningStep(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.To(ProvisioningStepToScriptInverseTable, FieldID),
			sqlgraph.Edge(sqlgraph.M2O, false, ProvisioningStepToScriptTable, ProvisioningStepToScriptColumn),
		)
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// HasProvisioningStepToCommand applies the HasEdge predicate on the "ProvisioningStepToCommand" edge.
func HasProvisioningStepToCommand() predicate.ProvisioningStep {
	return predicate.ProvisioningStep(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.To(ProvisioningStepToCommandTable, FieldID),
			sqlgraph.Edge(sqlgraph.M2O, false, ProvisioningStepToCommandTable, ProvisioningStepToCommandColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasProvisioningStepToCommandWith applies the HasEdge predicate on the "ProvisioningStepToCommand" edge with a given conditions (other predicates).
func HasProvisioningStepToCommandWith(preds ...predicate.Command) predicate.ProvisioningStep {
	return predicate.ProvisioningStep(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.To(ProvisioningStepToCommandInverseTable, FieldID),
			sqlgraph.Edge(sqlgraph.M2O, false, ProvisioningStepToCommandTable, ProvisioningStepToCommandColumn),
		)
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// HasProvisioningStepToDNSRecord applies the HasEdge predicate on the "ProvisioningStepToDNSRecord" edge.
func HasProvisioningStepToDNSRecord() predicate.ProvisioningStep {
	return predicate.ProvisioningStep(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.To(ProvisioningStepToDNSRecordTable, FieldID),
			sqlgraph.Edge(sqlgraph.M2O, false, ProvisioningStepToDNSRecordTable, ProvisioningStepToDNSRecordColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasProvisioningStepToDNSRecordWith applies the HasEdge predicate on the "ProvisioningStepToDNSRecord" edge with a given conditions (other predicates).
func HasProvisioningStepToDNSRecordWith(preds ...predicate.DNSRecord) predicate.ProvisioningStep {
	return predicate.ProvisioningStep(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.To(ProvisioningStepToDNSRecordInverseTable, FieldID),
			sqlgraph.Edge(sqlgraph.M2O, false, ProvisioningStepToDNSRecordTable, ProvisioningStepToDNSRecordColumn),
		)
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// HasProvisioningStepToFileDelete applies the HasEdge predicate on the "ProvisioningStepToFileDelete" edge.
func HasProvisioningStepToFileDelete() predicate.ProvisioningStep {
	return predicate.ProvisioningStep(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.To(ProvisioningStepToFileDeleteTable, FieldID),
			sqlgraph.Edge(sqlgraph.M2O, false, ProvisioningStepToFileDeleteTable, ProvisioningStepToFileDeleteColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasProvisioningStepToFileDeleteWith applies the HasEdge predicate on the "ProvisioningStepToFileDelete" edge with a given conditions (other predicates).
func HasProvisioningStepToFileDeleteWith(preds ...predicate.FileDelete) predicate.ProvisioningStep {
	return predicate.ProvisioningStep(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.To(ProvisioningStepToFileDeleteInverseTable, FieldID),
			sqlgraph.Edge(sqlgraph.M2O, false, ProvisioningStepToFileDeleteTable, ProvisioningStepToFileDeleteColumn),
		)
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// HasProvisioningStepToFileDownload applies the HasEdge predicate on the "ProvisioningStepToFileDownload" edge.
func HasProvisioningStepToFileDownload() predicate.ProvisioningStep {
	return predicate.ProvisioningStep(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.To(ProvisioningStepToFileDownloadTable, FieldID),
			sqlgraph.Edge(sqlgraph.M2O, false, ProvisioningStepToFileDownloadTable, ProvisioningStepToFileDownloadColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasProvisioningStepToFileDownloadWith applies the HasEdge predicate on the "ProvisioningStepToFileDownload" edge with a given conditions (other predicates).
func HasProvisioningStepToFileDownloadWith(preds ...predicate.FileDownload) predicate.ProvisioningStep {
	return predicate.ProvisioningStep(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.To(ProvisioningStepToFileDownloadInverseTable, FieldID),
			sqlgraph.Edge(sqlgraph.M2O, false, ProvisioningStepToFileDownloadTable, ProvisioningStepToFileDownloadColumn),
		)
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// HasProvisioningStepToFileExtract applies the HasEdge predicate on the "ProvisioningStepToFileExtract" edge.
func HasProvisioningStepToFileExtract() predicate.ProvisioningStep {
	return predicate.ProvisioningStep(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.To(ProvisioningStepToFileExtractTable, FieldID),
			sqlgraph.Edge(sqlgraph.M2O, false, ProvisioningStepToFileExtractTable, ProvisioningStepToFileExtractColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasProvisioningStepToFileExtractWith applies the HasEdge predicate on the "ProvisioningStepToFileExtract" edge with a given conditions (other predicates).
func HasProvisioningStepToFileExtractWith(preds ...predicate.FileExtract) predicate.ProvisioningStep {
	return predicate.ProvisioningStep(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.To(ProvisioningStepToFileExtractInverseTable, FieldID),
			sqlgraph.Edge(sqlgraph.M2O, false, ProvisioningStepToFileExtractTable, ProvisioningStepToFileExtractColumn),
		)
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// HasProvisioningStepToPlan applies the HasEdge predicate on the "ProvisioningStepToPlan" edge.
func HasProvisioningStepToPlan() predicate.ProvisioningStep {
	return predicate.ProvisioningStep(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.To(ProvisioningStepToPlanTable, FieldID),
			sqlgraph.Edge(sqlgraph.O2O, true, ProvisioningStepToPlanTable, ProvisioningStepToPlanColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasProvisioningStepToPlanWith applies the HasEdge predicate on the "ProvisioningStepToPlan" edge with a given conditions (other predicates).
func HasProvisioningStepToPlanWith(preds ...predicate.Plan) predicate.ProvisioningStep {
	return predicate.ProvisioningStep(func(s *sql.Selector) {
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

// HasProvisioningStepToGinFileMiddleware applies the HasEdge predicate on the "ProvisioningStepToGinFileMiddleware" edge.
func HasProvisioningStepToGinFileMiddleware() predicate.ProvisioningStep {
	return predicate.ProvisioningStep(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.To(ProvisioningStepToGinFileMiddlewareTable, FieldID),
			sqlgraph.Edge(sqlgraph.O2O, true, ProvisioningStepToGinFileMiddlewareTable, ProvisioningStepToGinFileMiddlewareColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasProvisioningStepToGinFileMiddlewareWith applies the HasEdge predicate on the "ProvisioningStepToGinFileMiddleware" edge with a given conditions (other predicates).
func HasProvisioningStepToGinFileMiddlewareWith(preds ...predicate.GinFileMiddleware) predicate.ProvisioningStep {
	return predicate.ProvisioningStep(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.To(ProvisioningStepToGinFileMiddlewareInverseTable, FieldID),
			sqlgraph.Edge(sqlgraph.O2O, true, ProvisioningStepToGinFileMiddlewareTable, ProvisioningStepToGinFileMiddlewareColumn),
		)
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// And groups predicates with the AND operator between them.
func And(predicates ...predicate.ProvisioningStep) predicate.ProvisioningStep {
	return predicate.ProvisioningStep(func(s *sql.Selector) {
		s1 := s.Clone().SetP(nil)
		for _, p := range predicates {
			p(s1)
		}
		s.Where(s1.P())
	})
}

// Or groups predicates with the OR operator between them.
func Or(predicates ...predicate.ProvisioningStep) predicate.ProvisioningStep {
	return predicate.ProvisioningStep(func(s *sql.Selector) {
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
func Not(p predicate.ProvisioningStep) predicate.ProvisioningStep {
	return predicate.ProvisioningStep(func(s *sql.Selector) {
		p(s.Not())
	})
}
