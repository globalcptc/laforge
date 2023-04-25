// Code generated by ent, DO NOT EDIT.

package provisionedhost

import (
	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"github.com/gen0cide/laforge/ent/predicate"
	"github.com/google/uuid"
)

// ID filters vertices based on their ID field.
func ID(id uuid.UUID) predicate.ProvisionedHost {
	return predicate.ProvisionedHost(func(s *sql.Selector) {
		s.Where(sql.EQ(s.C(FieldID), id))
	})
}

// IDEQ applies the EQ predicate on the ID field.
func IDEQ(id uuid.UUID) predicate.ProvisionedHost {
	return predicate.ProvisionedHost(func(s *sql.Selector) {
		s.Where(sql.EQ(s.C(FieldID), id))
	})
}

// IDNEQ applies the NEQ predicate on the ID field.
func IDNEQ(id uuid.UUID) predicate.ProvisionedHost {
	return predicate.ProvisionedHost(func(s *sql.Selector) {
		s.Where(sql.NEQ(s.C(FieldID), id))
	})
}

// IDIn applies the In predicate on the ID field.
func IDIn(ids ...uuid.UUID) predicate.ProvisionedHost {
	return predicate.ProvisionedHost(func(s *sql.Selector) {
		v := make([]interface{}, len(ids))
		for i := range v {
			v[i] = ids[i]
		}
		s.Where(sql.In(s.C(FieldID), v...))
	})
}

// IDNotIn applies the NotIn predicate on the ID field.
func IDNotIn(ids ...uuid.UUID) predicate.ProvisionedHost {
	return predicate.ProvisionedHost(func(s *sql.Selector) {
		v := make([]interface{}, len(ids))
		for i := range v {
			v[i] = ids[i]
		}
		s.Where(sql.NotIn(s.C(FieldID), v...))
	})
}

// IDGT applies the GT predicate on the ID field.
func IDGT(id uuid.UUID) predicate.ProvisionedHost {
	return predicate.ProvisionedHost(func(s *sql.Selector) {
		s.Where(sql.GT(s.C(FieldID), id))
	})
}

// IDGTE applies the GTE predicate on the ID field.
func IDGTE(id uuid.UUID) predicate.ProvisionedHost {
	return predicate.ProvisionedHost(func(s *sql.Selector) {
		s.Where(sql.GTE(s.C(FieldID), id))
	})
}

// IDLT applies the LT predicate on the ID field.
func IDLT(id uuid.UUID) predicate.ProvisionedHost {
	return predicate.ProvisionedHost(func(s *sql.Selector) {
		s.Where(sql.LT(s.C(FieldID), id))
	})
}

// IDLTE applies the LTE predicate on the ID field.
func IDLTE(id uuid.UUID) predicate.ProvisionedHost {
	return predicate.ProvisionedHost(func(s *sql.Selector) {
		s.Where(sql.LTE(s.C(FieldID), id))
	})
}

// SubnetIP applies equality check predicate on the "subnet_ip" field. It's identical to SubnetIPEQ.
func SubnetIP(v string) predicate.ProvisionedHost {
	return predicate.ProvisionedHost(func(s *sql.Selector) {
		s.Where(sql.EQ(s.C(FieldSubnetIP), v))
	})
}

// SubnetIPEQ applies the EQ predicate on the "subnet_ip" field.
func SubnetIPEQ(v string) predicate.ProvisionedHost {
	return predicate.ProvisionedHost(func(s *sql.Selector) {
		s.Where(sql.EQ(s.C(FieldSubnetIP), v))
	})
}

// SubnetIPNEQ applies the NEQ predicate on the "subnet_ip" field.
func SubnetIPNEQ(v string) predicate.ProvisionedHost {
	return predicate.ProvisionedHost(func(s *sql.Selector) {
		s.Where(sql.NEQ(s.C(FieldSubnetIP), v))
	})
}

// SubnetIPIn applies the In predicate on the "subnet_ip" field.
func SubnetIPIn(vs ...string) predicate.ProvisionedHost {
	v := make([]interface{}, len(vs))
	for i := range v {
		v[i] = vs[i]
	}
	return predicate.ProvisionedHost(func(s *sql.Selector) {
		s.Where(sql.In(s.C(FieldSubnetIP), v...))
	})
}

// SubnetIPNotIn applies the NotIn predicate on the "subnet_ip" field.
func SubnetIPNotIn(vs ...string) predicate.ProvisionedHost {
	v := make([]interface{}, len(vs))
	for i := range v {
		v[i] = vs[i]
	}
	return predicate.ProvisionedHost(func(s *sql.Selector) {
		s.Where(sql.NotIn(s.C(FieldSubnetIP), v...))
	})
}

// SubnetIPGT applies the GT predicate on the "subnet_ip" field.
func SubnetIPGT(v string) predicate.ProvisionedHost {
	return predicate.ProvisionedHost(func(s *sql.Selector) {
		s.Where(sql.GT(s.C(FieldSubnetIP), v))
	})
}

// SubnetIPGTE applies the GTE predicate on the "subnet_ip" field.
func SubnetIPGTE(v string) predicate.ProvisionedHost {
	return predicate.ProvisionedHost(func(s *sql.Selector) {
		s.Where(sql.GTE(s.C(FieldSubnetIP), v))
	})
}

// SubnetIPLT applies the LT predicate on the "subnet_ip" field.
func SubnetIPLT(v string) predicate.ProvisionedHost {
	return predicate.ProvisionedHost(func(s *sql.Selector) {
		s.Where(sql.LT(s.C(FieldSubnetIP), v))
	})
}

// SubnetIPLTE applies the LTE predicate on the "subnet_ip" field.
func SubnetIPLTE(v string) predicate.ProvisionedHost {
	return predicate.ProvisionedHost(func(s *sql.Selector) {
		s.Where(sql.LTE(s.C(FieldSubnetIP), v))
	})
}

// SubnetIPContains applies the Contains predicate on the "subnet_ip" field.
func SubnetIPContains(v string) predicate.ProvisionedHost {
	return predicate.ProvisionedHost(func(s *sql.Selector) {
		s.Where(sql.Contains(s.C(FieldSubnetIP), v))
	})
}

// SubnetIPHasPrefix applies the HasPrefix predicate on the "subnet_ip" field.
func SubnetIPHasPrefix(v string) predicate.ProvisionedHost {
	return predicate.ProvisionedHost(func(s *sql.Selector) {
		s.Where(sql.HasPrefix(s.C(FieldSubnetIP), v))
	})
}

// SubnetIPHasSuffix applies the HasSuffix predicate on the "subnet_ip" field.
func SubnetIPHasSuffix(v string) predicate.ProvisionedHost {
	return predicate.ProvisionedHost(func(s *sql.Selector) {
		s.Where(sql.HasSuffix(s.C(FieldSubnetIP), v))
	})
}

// SubnetIPEqualFold applies the EqualFold predicate on the "subnet_ip" field.
func SubnetIPEqualFold(v string) predicate.ProvisionedHost {
	return predicate.ProvisionedHost(func(s *sql.Selector) {
		s.Where(sql.EqualFold(s.C(FieldSubnetIP), v))
	})
}

// SubnetIPContainsFold applies the ContainsFold predicate on the "subnet_ip" field.
func SubnetIPContainsFold(v string) predicate.ProvisionedHost {
	return predicate.ProvisionedHost(func(s *sql.Selector) {
		s.Where(sql.ContainsFold(s.C(FieldSubnetIP), v))
	})
}

// AddonTypeEQ applies the EQ predicate on the "addon_type" field.
func AddonTypeEQ(v AddonType) predicate.ProvisionedHost {
	return predicate.ProvisionedHost(func(s *sql.Selector) {
		s.Where(sql.EQ(s.C(FieldAddonType), v))
	})
}

// AddonTypeNEQ applies the NEQ predicate on the "addon_type" field.
func AddonTypeNEQ(v AddonType) predicate.ProvisionedHost {
	return predicate.ProvisionedHost(func(s *sql.Selector) {
		s.Where(sql.NEQ(s.C(FieldAddonType), v))
	})
}

// AddonTypeIn applies the In predicate on the "addon_type" field.
func AddonTypeIn(vs ...AddonType) predicate.ProvisionedHost {
	v := make([]interface{}, len(vs))
	for i := range v {
		v[i] = vs[i]
	}
	return predicate.ProvisionedHost(func(s *sql.Selector) {
		s.Where(sql.In(s.C(FieldAddonType), v...))
	})
}

// AddonTypeNotIn applies the NotIn predicate on the "addon_type" field.
func AddonTypeNotIn(vs ...AddonType) predicate.ProvisionedHost {
	v := make([]interface{}, len(vs))
	for i := range v {
		v[i] = vs[i]
	}
	return predicate.ProvisionedHost(func(s *sql.Selector) {
		s.Where(sql.NotIn(s.C(FieldAddonType), v...))
	})
}

// AddonTypeIsNil applies the IsNil predicate on the "addon_type" field.
func AddonTypeIsNil() predicate.ProvisionedHost {
	return predicate.ProvisionedHost(func(s *sql.Selector) {
		s.Where(sql.IsNull(s.C(FieldAddonType)))
	})
}

// AddonTypeNotNil applies the NotNil predicate on the "addon_type" field.
func AddonTypeNotNil() predicate.ProvisionedHost {
	return predicate.ProvisionedHost(func(s *sql.Selector) {
		s.Where(sql.NotNull(s.C(FieldAddonType)))
	})
}

// HasStatus applies the HasEdge predicate on the "Status" edge.
func HasStatus() predicate.ProvisionedHost {
	return predicate.ProvisionedHost(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.To(StatusTable, FieldID),
			sqlgraph.Edge(sqlgraph.O2O, false, StatusTable, StatusColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasStatusWith applies the HasEdge predicate on the "Status" edge with a given conditions (other predicates).
func HasStatusWith(preds ...predicate.Status) predicate.ProvisionedHost {
	return predicate.ProvisionedHost(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.To(StatusInverseTable, FieldID),
			sqlgraph.Edge(sqlgraph.O2O, false, StatusTable, StatusColumn),
		)
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// HasProvisionedNetwork applies the HasEdge predicate on the "ProvisionedNetwork" edge.
func HasProvisionedNetwork() predicate.ProvisionedHost {
	return predicate.ProvisionedHost(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.To(ProvisionedNetworkTable, FieldID),
			sqlgraph.Edge(sqlgraph.M2O, false, ProvisionedNetworkTable, ProvisionedNetworkColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasProvisionedNetworkWith applies the HasEdge predicate on the "ProvisionedNetwork" edge with a given conditions (other predicates).
func HasProvisionedNetworkWith(preds ...predicate.ProvisionedNetwork) predicate.ProvisionedHost {
	return predicate.ProvisionedHost(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.To(ProvisionedNetworkInverseTable, FieldID),
			sqlgraph.Edge(sqlgraph.M2O, false, ProvisionedNetworkTable, ProvisionedNetworkColumn),
		)
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// HasHost applies the HasEdge predicate on the "Host" edge.
func HasHost() predicate.ProvisionedHost {
	return predicate.ProvisionedHost(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.To(HostTable, FieldID),
			sqlgraph.Edge(sqlgraph.M2O, false, HostTable, HostColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasHostWith applies the HasEdge predicate on the "Host" edge with a given conditions (other predicates).
func HasHostWith(preds ...predicate.Host) predicate.ProvisionedHost {
	return predicate.ProvisionedHost(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.To(HostInverseTable, FieldID),
			sqlgraph.Edge(sqlgraph.M2O, false, HostTable, HostColumn),
		)
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// HasEndStepPlan applies the HasEdge predicate on the "EndStepPlan" edge.
func HasEndStepPlan() predicate.ProvisionedHost {
	return predicate.ProvisionedHost(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.To(EndStepPlanTable, FieldID),
			sqlgraph.Edge(sqlgraph.M2O, false, EndStepPlanTable, EndStepPlanColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasEndStepPlanWith applies the HasEdge predicate on the "EndStepPlan" edge with a given conditions (other predicates).
func HasEndStepPlanWith(preds ...predicate.Plan) predicate.ProvisionedHost {
	return predicate.ProvisionedHost(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.To(EndStepPlanInverseTable, FieldID),
			sqlgraph.Edge(sqlgraph.M2O, false, EndStepPlanTable, EndStepPlanColumn),
		)
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// HasBuild applies the HasEdge predicate on the "Build" edge.
func HasBuild() predicate.ProvisionedHost {
	return predicate.ProvisionedHost(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.To(BuildTable, FieldID),
			sqlgraph.Edge(sqlgraph.M2O, false, BuildTable, BuildColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasBuildWith applies the HasEdge predicate on the "Build" edge with a given conditions (other predicates).
func HasBuildWith(preds ...predicate.Build) predicate.ProvisionedHost {
	return predicate.ProvisionedHost(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.To(BuildInverseTable, FieldID),
			sqlgraph.Edge(sqlgraph.M2O, false, BuildTable, BuildColumn),
		)
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// HasProvisioningSteps applies the HasEdge predicate on the "ProvisioningSteps" edge.
func HasProvisioningSteps() predicate.ProvisionedHost {
	return predicate.ProvisionedHost(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.To(ProvisioningStepsTable, FieldID),
			sqlgraph.Edge(sqlgraph.O2M, true, ProvisioningStepsTable, ProvisioningStepsColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasProvisioningStepsWith applies the HasEdge predicate on the "ProvisioningSteps" edge with a given conditions (other predicates).
func HasProvisioningStepsWith(preds ...predicate.ProvisioningStep) predicate.ProvisionedHost {
	return predicate.ProvisionedHost(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.To(ProvisioningStepsInverseTable, FieldID),
			sqlgraph.Edge(sqlgraph.O2M, true, ProvisioningStepsTable, ProvisioningStepsColumn),
		)
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// HasProvisioningScheduledSteps applies the HasEdge predicate on the "ProvisioningScheduledSteps" edge.
func HasProvisioningScheduledSteps() predicate.ProvisionedHost {
	return predicate.ProvisionedHost(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.To(ProvisioningScheduledStepsTable, FieldID),
			sqlgraph.Edge(sqlgraph.O2M, true, ProvisioningScheduledStepsTable, ProvisioningScheduledStepsColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasProvisioningScheduledStepsWith applies the HasEdge predicate on the "ProvisioningScheduledSteps" edge with a given conditions (other predicates).
func HasProvisioningScheduledStepsWith(preds ...predicate.ProvisioningScheduledStep) predicate.ProvisionedHost {
	return predicate.ProvisionedHost(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.To(ProvisioningScheduledStepsInverseTable, FieldID),
			sqlgraph.Edge(sqlgraph.O2M, true, ProvisioningScheduledStepsTable, ProvisioningScheduledStepsColumn),
		)
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// HasAgentStatus applies the HasEdge predicate on the "AgentStatus" edge.
func HasAgentStatus() predicate.ProvisionedHost {
	return predicate.ProvisionedHost(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.To(AgentStatusTable, FieldID),
			sqlgraph.Edge(sqlgraph.O2O, true, AgentStatusTable, AgentStatusColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasAgentStatusWith applies the HasEdge predicate on the "AgentStatus" edge with a given conditions (other predicates).
func HasAgentStatusWith(preds ...predicate.AgentStatus) predicate.ProvisionedHost {
	return predicate.ProvisionedHost(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.To(AgentStatusInverseTable, FieldID),
			sqlgraph.Edge(sqlgraph.O2O, true, AgentStatusTable, AgentStatusColumn),
		)
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// HasAgentTasks applies the HasEdge predicate on the "AgentTasks" edge.
func HasAgentTasks() predicate.ProvisionedHost {
	return predicate.ProvisionedHost(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.To(AgentTasksTable, FieldID),
			sqlgraph.Edge(sqlgraph.O2M, true, AgentTasksTable, AgentTasksColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasAgentTasksWith applies the HasEdge predicate on the "AgentTasks" edge with a given conditions (other predicates).
func HasAgentTasksWith(preds ...predicate.AgentTask) predicate.ProvisionedHost {
	return predicate.ProvisionedHost(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.To(AgentTasksInverseTable, FieldID),
			sqlgraph.Edge(sqlgraph.O2M, true, AgentTasksTable, AgentTasksColumn),
		)
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// HasPlan applies the HasEdge predicate on the "Plan" edge.
func HasPlan() predicate.ProvisionedHost {
	return predicate.ProvisionedHost(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.To(PlanTable, FieldID),
			sqlgraph.Edge(sqlgraph.O2O, true, PlanTable, PlanColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasPlanWith applies the HasEdge predicate on the "Plan" edge with a given conditions (other predicates).
func HasPlanWith(preds ...predicate.Plan) predicate.ProvisionedHost {
	return predicate.ProvisionedHost(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.To(PlanInverseTable, FieldID),
			sqlgraph.Edge(sqlgraph.O2O, true, PlanTable, PlanColumn),
		)
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// HasGinFileMiddleware applies the HasEdge predicate on the "GinFileMiddleware" edge.
func HasGinFileMiddleware() predicate.ProvisionedHost {
	return predicate.ProvisionedHost(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.To(GinFileMiddlewareTable, FieldID),
			sqlgraph.Edge(sqlgraph.O2O, true, GinFileMiddlewareTable, GinFileMiddlewareColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasGinFileMiddlewareWith applies the HasEdge predicate on the "GinFileMiddleware" edge with a given conditions (other predicates).
func HasGinFileMiddlewareWith(preds ...predicate.GinFileMiddleware) predicate.ProvisionedHost {
	return predicate.ProvisionedHost(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.To(GinFileMiddlewareInverseTable, FieldID),
			sqlgraph.Edge(sqlgraph.O2O, true, GinFileMiddlewareTable, GinFileMiddlewareColumn),
		)
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// And groups predicates with the AND operator between them.
func And(predicates ...predicate.ProvisionedHost) predicate.ProvisionedHost {
	return predicate.ProvisionedHost(func(s *sql.Selector) {
		s1 := s.Clone().SetP(nil)
		for _, p := range predicates {
			p(s1)
		}
		s.Where(s1.P())
	})
}

// Or groups predicates with the OR operator between them.
func Or(predicates ...predicate.ProvisionedHost) predicate.ProvisionedHost {
	return predicate.ProvisionedHost(func(s *sql.Selector) {
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
func Not(p predicate.ProvisionedHost) predicate.ProvisionedHost {
	return predicate.ProvisionedHost(func(s *sql.Selector) {
		p(s.Not())
	})
}
