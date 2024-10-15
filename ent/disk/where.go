// Code generated by ent, DO NOT EDIT.

package disk

import (
	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"github.com/gen0cide/laforge/ent/predicate"
	"github.com/google/uuid"
)

// ID filters vertices based on their ID field.
func ID(id uuid.UUID) predicate.Disk {
	return predicate.Disk(sql.FieldEQ(FieldID, id))
}

// IDEQ applies the EQ predicate on the ID field.
func IDEQ(id uuid.UUID) predicate.Disk {
	return predicate.Disk(sql.FieldEQ(FieldID, id))
}

// IDNEQ applies the NEQ predicate on the ID field.
func IDNEQ(id uuid.UUID) predicate.Disk {
	return predicate.Disk(sql.FieldNEQ(FieldID, id))
}

// IDIn applies the In predicate on the ID field.
func IDIn(ids ...uuid.UUID) predicate.Disk {
	return predicate.Disk(sql.FieldIn(FieldID, ids...))
}

// IDNotIn applies the NotIn predicate on the ID field.
func IDNotIn(ids ...uuid.UUID) predicate.Disk {
	return predicate.Disk(sql.FieldNotIn(FieldID, ids...))
}

// IDGT applies the GT predicate on the ID field.
func IDGT(id uuid.UUID) predicate.Disk {
	return predicate.Disk(sql.FieldGT(FieldID, id))
}

// IDGTE applies the GTE predicate on the ID field.
func IDGTE(id uuid.UUID) predicate.Disk {
	return predicate.Disk(sql.FieldGTE(FieldID, id))
}

// IDLT applies the LT predicate on the ID field.
func IDLT(id uuid.UUID) predicate.Disk {
	return predicate.Disk(sql.FieldLT(FieldID, id))
}

// IDLTE applies the LTE predicate on the ID field.
func IDLTE(id uuid.UUID) predicate.Disk {
	return predicate.Disk(sql.FieldLTE(FieldID, id))
}

// Size applies equality check predicate on the "size" field. It's identical to SizeEQ.
func Size(v int) predicate.Disk {
	return predicate.Disk(sql.FieldEQ(FieldSize, v))
}

// SizeEQ applies the EQ predicate on the "size" field.
func SizeEQ(v int) predicate.Disk {
	return predicate.Disk(sql.FieldEQ(FieldSize, v))
}

// SizeNEQ applies the NEQ predicate on the "size" field.
func SizeNEQ(v int) predicate.Disk {
	return predicate.Disk(sql.FieldNEQ(FieldSize, v))
}

// SizeIn applies the In predicate on the "size" field.
func SizeIn(vs ...int) predicate.Disk {
	return predicate.Disk(sql.FieldIn(FieldSize, vs...))
}

// SizeNotIn applies the NotIn predicate on the "size" field.
func SizeNotIn(vs ...int) predicate.Disk {
	return predicate.Disk(sql.FieldNotIn(FieldSize, vs...))
}

// SizeGT applies the GT predicate on the "size" field.
func SizeGT(v int) predicate.Disk {
	return predicate.Disk(sql.FieldGT(FieldSize, v))
}

// SizeGTE applies the GTE predicate on the "size" field.
func SizeGTE(v int) predicate.Disk {
	return predicate.Disk(sql.FieldGTE(FieldSize, v))
}

// SizeLT applies the LT predicate on the "size" field.
func SizeLT(v int) predicate.Disk {
	return predicate.Disk(sql.FieldLT(FieldSize, v))
}

// SizeLTE applies the LTE predicate on the "size" field.
func SizeLTE(v int) predicate.Disk {
	return predicate.Disk(sql.FieldLTE(FieldSize, v))
}

// HasHost applies the HasEdge predicate on the "Host" edge.
func HasHost() predicate.Disk {
	return predicate.Disk(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.Edge(sqlgraph.O2O, true, HostTable, HostColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasHostWith applies the HasEdge predicate on the "Host" edge with a given conditions (other predicates).
func HasHostWith(preds ...predicate.Host) predicate.Disk {
	return predicate.Disk(func(s *sql.Selector) {
		step := newHostStep()
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// And groups predicates with the AND operator between them.
func And(predicates ...predicate.Disk) predicate.Disk {
	return predicate.Disk(sql.AndPredicates(predicates...))
}

// Or groups predicates with the OR operator between them.
func Or(predicates ...predicate.Disk) predicate.Disk {
	return predicate.Disk(sql.OrPredicates(predicates...))
}

// Not applies the not operator on the given predicate.
func Not(p predicate.Disk) predicate.Disk {
	return predicate.Disk(sql.NotPredicates(p))
}
