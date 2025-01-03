// Code generated by ent, DO NOT EDIT.

package provisionednetwork

import (
	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"github.com/gen0cide/laforge/ent/predicate"
	"github.com/google/uuid"
)

// ID filters vertices based on their ID field.
func ID(id uuid.UUID) predicate.ProvisionedNetwork {
	return predicate.ProvisionedNetwork(sql.FieldEQ(FieldID, id))
}

// IDEQ applies the EQ predicate on the ID field.
func IDEQ(id uuid.UUID) predicate.ProvisionedNetwork {
	return predicate.ProvisionedNetwork(sql.FieldEQ(FieldID, id))
}

// IDNEQ applies the NEQ predicate on the ID field.
func IDNEQ(id uuid.UUID) predicate.ProvisionedNetwork {
	return predicate.ProvisionedNetwork(sql.FieldNEQ(FieldID, id))
}

// IDIn applies the In predicate on the ID field.
func IDIn(ids ...uuid.UUID) predicate.ProvisionedNetwork {
	return predicate.ProvisionedNetwork(sql.FieldIn(FieldID, ids...))
}

// IDNotIn applies the NotIn predicate on the ID field.
func IDNotIn(ids ...uuid.UUID) predicate.ProvisionedNetwork {
	return predicate.ProvisionedNetwork(sql.FieldNotIn(FieldID, ids...))
}

// IDGT applies the GT predicate on the ID field.
func IDGT(id uuid.UUID) predicate.ProvisionedNetwork {
	return predicate.ProvisionedNetwork(sql.FieldGT(FieldID, id))
}

// IDGTE applies the GTE predicate on the ID field.
func IDGTE(id uuid.UUID) predicate.ProvisionedNetwork {
	return predicate.ProvisionedNetwork(sql.FieldGTE(FieldID, id))
}

// IDLT applies the LT predicate on the ID field.
func IDLT(id uuid.UUID) predicate.ProvisionedNetwork {
	return predicate.ProvisionedNetwork(sql.FieldLT(FieldID, id))
}

// IDLTE applies the LTE predicate on the ID field.
func IDLTE(id uuid.UUID) predicate.ProvisionedNetwork {
	return predicate.ProvisionedNetwork(sql.FieldLTE(FieldID, id))
}

// Name applies equality check predicate on the "name" field. It's identical to NameEQ.
func Name(v string) predicate.ProvisionedNetwork {
	return predicate.ProvisionedNetwork(sql.FieldEQ(FieldName, v))
}

// Cidr applies equality check predicate on the "cidr" field. It's identical to CidrEQ.
func Cidr(v string) predicate.ProvisionedNetwork {
	return predicate.ProvisionedNetwork(sql.FieldEQ(FieldCidr, v))
}

// NameEQ applies the EQ predicate on the "name" field.
func NameEQ(v string) predicate.ProvisionedNetwork {
	return predicate.ProvisionedNetwork(sql.FieldEQ(FieldName, v))
}

// NameNEQ applies the NEQ predicate on the "name" field.
func NameNEQ(v string) predicate.ProvisionedNetwork {
	return predicate.ProvisionedNetwork(sql.FieldNEQ(FieldName, v))
}

// NameIn applies the In predicate on the "name" field.
func NameIn(vs ...string) predicate.ProvisionedNetwork {
	return predicate.ProvisionedNetwork(sql.FieldIn(FieldName, vs...))
}

// NameNotIn applies the NotIn predicate on the "name" field.
func NameNotIn(vs ...string) predicate.ProvisionedNetwork {
	return predicate.ProvisionedNetwork(sql.FieldNotIn(FieldName, vs...))
}

// NameGT applies the GT predicate on the "name" field.
func NameGT(v string) predicate.ProvisionedNetwork {
	return predicate.ProvisionedNetwork(sql.FieldGT(FieldName, v))
}

// NameGTE applies the GTE predicate on the "name" field.
func NameGTE(v string) predicate.ProvisionedNetwork {
	return predicate.ProvisionedNetwork(sql.FieldGTE(FieldName, v))
}

// NameLT applies the LT predicate on the "name" field.
func NameLT(v string) predicate.ProvisionedNetwork {
	return predicate.ProvisionedNetwork(sql.FieldLT(FieldName, v))
}

// NameLTE applies the LTE predicate on the "name" field.
func NameLTE(v string) predicate.ProvisionedNetwork {
	return predicate.ProvisionedNetwork(sql.FieldLTE(FieldName, v))
}

// NameContains applies the Contains predicate on the "name" field.
func NameContains(v string) predicate.ProvisionedNetwork {
	return predicate.ProvisionedNetwork(sql.FieldContains(FieldName, v))
}

// NameHasPrefix applies the HasPrefix predicate on the "name" field.
func NameHasPrefix(v string) predicate.ProvisionedNetwork {
	return predicate.ProvisionedNetwork(sql.FieldHasPrefix(FieldName, v))
}

// NameHasSuffix applies the HasSuffix predicate on the "name" field.
func NameHasSuffix(v string) predicate.ProvisionedNetwork {
	return predicate.ProvisionedNetwork(sql.FieldHasSuffix(FieldName, v))
}

// NameEqualFold applies the EqualFold predicate on the "name" field.
func NameEqualFold(v string) predicate.ProvisionedNetwork {
	return predicate.ProvisionedNetwork(sql.FieldEqualFold(FieldName, v))
}

// NameContainsFold applies the ContainsFold predicate on the "name" field.
func NameContainsFold(v string) predicate.ProvisionedNetwork {
	return predicate.ProvisionedNetwork(sql.FieldContainsFold(FieldName, v))
}

// CidrEQ applies the EQ predicate on the "cidr" field.
func CidrEQ(v string) predicate.ProvisionedNetwork {
	return predicate.ProvisionedNetwork(sql.FieldEQ(FieldCidr, v))
}

// CidrNEQ applies the NEQ predicate on the "cidr" field.
func CidrNEQ(v string) predicate.ProvisionedNetwork {
	return predicate.ProvisionedNetwork(sql.FieldNEQ(FieldCidr, v))
}

// CidrIn applies the In predicate on the "cidr" field.
func CidrIn(vs ...string) predicate.ProvisionedNetwork {
	return predicate.ProvisionedNetwork(sql.FieldIn(FieldCidr, vs...))
}

// CidrNotIn applies the NotIn predicate on the "cidr" field.
func CidrNotIn(vs ...string) predicate.ProvisionedNetwork {
	return predicate.ProvisionedNetwork(sql.FieldNotIn(FieldCidr, vs...))
}

// CidrGT applies the GT predicate on the "cidr" field.
func CidrGT(v string) predicate.ProvisionedNetwork {
	return predicate.ProvisionedNetwork(sql.FieldGT(FieldCidr, v))
}

// CidrGTE applies the GTE predicate on the "cidr" field.
func CidrGTE(v string) predicate.ProvisionedNetwork {
	return predicate.ProvisionedNetwork(sql.FieldGTE(FieldCidr, v))
}

// CidrLT applies the LT predicate on the "cidr" field.
func CidrLT(v string) predicate.ProvisionedNetwork {
	return predicate.ProvisionedNetwork(sql.FieldLT(FieldCidr, v))
}

// CidrLTE applies the LTE predicate on the "cidr" field.
func CidrLTE(v string) predicate.ProvisionedNetwork {
	return predicate.ProvisionedNetwork(sql.FieldLTE(FieldCidr, v))
}

// CidrContains applies the Contains predicate on the "cidr" field.
func CidrContains(v string) predicate.ProvisionedNetwork {
	return predicate.ProvisionedNetwork(sql.FieldContains(FieldCidr, v))
}

// CidrHasPrefix applies the HasPrefix predicate on the "cidr" field.
func CidrHasPrefix(v string) predicate.ProvisionedNetwork {
	return predicate.ProvisionedNetwork(sql.FieldHasPrefix(FieldCidr, v))
}

// CidrHasSuffix applies the HasSuffix predicate on the "cidr" field.
func CidrHasSuffix(v string) predicate.ProvisionedNetwork {
	return predicate.ProvisionedNetwork(sql.FieldHasSuffix(FieldCidr, v))
}

// CidrEqualFold applies the EqualFold predicate on the "cidr" field.
func CidrEqualFold(v string) predicate.ProvisionedNetwork {
	return predicate.ProvisionedNetwork(sql.FieldEqualFold(FieldCidr, v))
}

// CidrContainsFold applies the ContainsFold predicate on the "cidr" field.
func CidrContainsFold(v string) predicate.ProvisionedNetwork {
	return predicate.ProvisionedNetwork(sql.FieldContainsFold(FieldCidr, v))
}

// HasStatus applies the HasEdge predicate on the "Status" edge.
func HasStatus() predicate.ProvisionedNetwork {
	return predicate.ProvisionedNetwork(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.Edge(sqlgraph.O2O, false, StatusTable, StatusColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasStatusWith applies the HasEdge predicate on the "Status" edge with a given conditions (other predicates).
func HasStatusWith(preds ...predicate.Status) predicate.ProvisionedNetwork {
	return predicate.ProvisionedNetwork(func(s *sql.Selector) {
		step := newStatusStep()
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// HasNetwork applies the HasEdge predicate on the "Network" edge.
func HasNetwork() predicate.ProvisionedNetwork {
	return predicate.ProvisionedNetwork(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.Edge(sqlgraph.M2O, false, NetworkTable, NetworkColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasNetworkWith applies the HasEdge predicate on the "Network" edge with a given conditions (other predicates).
func HasNetworkWith(preds ...predicate.Network) predicate.ProvisionedNetwork {
	return predicate.ProvisionedNetwork(func(s *sql.Selector) {
		step := newNetworkStep()
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// HasBuild applies the HasEdge predicate on the "Build" edge.
func HasBuild() predicate.ProvisionedNetwork {
	return predicate.ProvisionedNetwork(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.Edge(sqlgraph.M2O, false, BuildTable, BuildColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasBuildWith applies the HasEdge predicate on the "Build" edge with a given conditions (other predicates).
func HasBuildWith(preds ...predicate.Build) predicate.ProvisionedNetwork {
	return predicate.ProvisionedNetwork(func(s *sql.Selector) {
		step := newBuildStep()
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// HasTeam applies the HasEdge predicate on the "Team" edge.
func HasTeam() predicate.ProvisionedNetwork {
	return predicate.ProvisionedNetwork(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.Edge(sqlgraph.M2O, false, TeamTable, TeamColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasTeamWith applies the HasEdge predicate on the "Team" edge with a given conditions (other predicates).
func HasTeamWith(preds ...predicate.Team) predicate.ProvisionedNetwork {
	return predicate.ProvisionedNetwork(func(s *sql.Selector) {
		step := newTeamStep()
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// HasProvisionedHosts applies the HasEdge predicate on the "ProvisionedHosts" edge.
func HasProvisionedHosts() predicate.ProvisionedNetwork {
	return predicate.ProvisionedNetwork(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.Edge(sqlgraph.O2M, true, ProvisionedHostsTable, ProvisionedHostsColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasProvisionedHostsWith applies the HasEdge predicate on the "ProvisionedHosts" edge with a given conditions (other predicates).
func HasProvisionedHostsWith(preds ...predicate.ProvisionedHost) predicate.ProvisionedNetwork {
	return predicate.ProvisionedNetwork(func(s *sql.Selector) {
		step := newProvisionedHostsStep()
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// HasPlan applies the HasEdge predicate on the "Plan" edge.
func HasPlan() predicate.ProvisionedNetwork {
	return predicate.ProvisionedNetwork(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.Edge(sqlgraph.O2O, true, PlanTable, PlanColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasPlanWith applies the HasEdge predicate on the "Plan" edge with a given conditions (other predicates).
func HasPlanWith(preds ...predicate.Plan) predicate.ProvisionedNetwork {
	return predicate.ProvisionedNetwork(func(s *sql.Selector) {
		step := newPlanStep()
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// And groups predicates with the AND operator between them.
func And(predicates ...predicate.ProvisionedNetwork) predicate.ProvisionedNetwork {
	return predicate.ProvisionedNetwork(sql.AndPredicates(predicates...))
}

// Or groups predicates with the OR operator between them.
func Or(predicates ...predicate.ProvisionedNetwork) predicate.ProvisionedNetwork {
	return predicate.ProvisionedNetwork(sql.OrPredicates(predicates...))
}

// Not applies the not operator on the given predicate.
func Not(p predicate.ProvisionedNetwork) predicate.ProvisionedNetwork {
	return predicate.ProvisionedNetwork(sql.NotPredicates(p))
}
