// Code generated by entc, DO NOT EDIT.

package ent

import (
	"context"
	"fmt"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/schema/field"
	"github.com/gen0cide/laforge/ent/predicate"
	"github.com/gen0cide/laforge/ent/provisionednetwork"
)

// ProvisionedNetworkDelete is the builder for deleting a ProvisionedNetwork entity.
type ProvisionedNetworkDelete struct {
	config
	hooks    []Hook
	mutation *ProvisionedNetworkMutation
}

// Where appends a list predicates to the ProvisionedNetworkDelete builder.
func (pnd *ProvisionedNetworkDelete) Where(ps ...predicate.ProvisionedNetwork) *ProvisionedNetworkDelete {
	pnd.mutation.Where(ps...)
	return pnd
}

// Exec executes the deletion query and returns how many vertices were deleted.
func (pnd *ProvisionedNetworkDelete) Exec(ctx context.Context) (int, error) {
	var (
		err      error
		affected int
	)
	if len(pnd.hooks) == 0 {
		affected, err = pnd.sqlExec(ctx)
	} else {
		var mut Mutator = MutateFunc(func(ctx context.Context, m Mutation) (Value, error) {
			mutation, ok := m.(*ProvisionedNetworkMutation)
			if !ok {
				return nil, fmt.Errorf("unexpected mutation type %T", m)
			}
			pnd.mutation = mutation
			affected, err = pnd.sqlExec(ctx)
			mutation.done = true
			return affected, err
		})
		for i := len(pnd.hooks) - 1; i >= 0; i-- {
			if pnd.hooks[i] == nil {
				return 0, fmt.Errorf("ent: uninitialized hook (forgotten import ent/runtime?)")
			}
			mut = pnd.hooks[i](mut)
		}
		if _, err := mut.Mutate(ctx, pnd.mutation); err != nil {
			return 0, err
		}
	}
	return affected, err
}

// ExecX is like Exec, but panics if an error occurs.
func (pnd *ProvisionedNetworkDelete) ExecX(ctx context.Context) int {
	n, err := pnd.Exec(ctx)
	if err != nil {
		panic(err)
	}
	return n
}

func (pnd *ProvisionedNetworkDelete) sqlExec(ctx context.Context) (int, error) {
	_spec := &sqlgraph.DeleteSpec{
		Node: &sqlgraph.NodeSpec{
			Table: provisionednetwork.Table,
			ID: &sqlgraph.FieldSpec{
				Type:   field.TypeUUID,
				Column: provisionednetwork.FieldID,
			},
		},
	}
	if ps := pnd.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	return sqlgraph.DeleteNodes(ctx, pnd.driver, _spec)
}

// ProvisionedNetworkDeleteOne is the builder for deleting a single ProvisionedNetwork entity.
type ProvisionedNetworkDeleteOne struct {
	pnd *ProvisionedNetworkDelete
}

// Exec executes the deletion query.
func (pndo *ProvisionedNetworkDeleteOne) Exec(ctx context.Context) error {
	n, err := pndo.pnd.Exec(ctx)
	switch {
	case err != nil:
		return err
	case n == 0:
		return &NotFoundError{provisionednetwork.Label}
	default:
		return nil
	}
}

// ExecX is like Exec, but panics if an error occurs.
func (pndo *ProvisionedNetworkDeleteOne) ExecX(ctx context.Context) {
	pndo.pnd.ExecX(ctx)
}
