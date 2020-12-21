// Code generated by entc, DO NOT EDIT.

package ent

import (
	"context"
	"fmt"

	"github.com/facebook/ent/dialect/sql"
	"github.com/facebook/ent/dialect/sql/sqlgraph"
	"github.com/facebook/ent/schema/field"
	"github.com/gen0cide/laforge/ent/includednetwork"
	"github.com/gen0cide/laforge/ent/predicate"
)

// IncludedNetworkDelete is the builder for deleting a IncludedNetwork entity.
type IncludedNetworkDelete struct {
	config
	hooks    []Hook
	mutation *IncludedNetworkMutation
}

// Where adds a new predicate to the delete builder.
func (ind *IncludedNetworkDelete) Where(ps ...predicate.IncludedNetwork) *IncludedNetworkDelete {
	ind.mutation.predicates = append(ind.mutation.predicates, ps...)
	return ind
}

// Exec executes the deletion query and returns how many vertices were deleted.
func (ind *IncludedNetworkDelete) Exec(ctx context.Context) (int, error) {
	var (
		err      error
		affected int
	)
	if len(ind.hooks) == 0 {
		affected, err = ind.sqlExec(ctx)
	} else {
		var mut Mutator = MutateFunc(func(ctx context.Context, m Mutation) (Value, error) {
			mutation, ok := m.(*IncludedNetworkMutation)
			if !ok {
				return nil, fmt.Errorf("unexpected mutation type %T", m)
			}
			ind.mutation = mutation
			affected, err = ind.sqlExec(ctx)
			mutation.done = true
			return affected, err
		})
		for i := len(ind.hooks) - 1; i >= 0; i-- {
			mut = ind.hooks[i](mut)
		}
		if _, err := mut.Mutate(ctx, ind.mutation); err != nil {
			return 0, err
		}
	}
	return affected, err
}

// ExecX is like Exec, but panics if an error occurs.
func (ind *IncludedNetworkDelete) ExecX(ctx context.Context) int {
	n, err := ind.Exec(ctx)
	if err != nil {
		panic(err)
	}
	return n
}

func (ind *IncludedNetworkDelete) sqlExec(ctx context.Context) (int, error) {
	_spec := &sqlgraph.DeleteSpec{
		Node: &sqlgraph.NodeSpec{
			Table: includednetwork.Table,
			ID: &sqlgraph.FieldSpec{
				Type:   field.TypeInt,
				Column: includednetwork.FieldID,
			},
		},
	}
	if ps := ind.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	return sqlgraph.DeleteNodes(ctx, ind.driver, _spec)
}

// IncludedNetworkDeleteOne is the builder for deleting a single IncludedNetwork entity.
type IncludedNetworkDeleteOne struct {
	ind *IncludedNetworkDelete
}

// Exec executes the deletion query.
func (indo *IncludedNetworkDeleteOne) Exec(ctx context.Context) error {
	n, err := indo.ind.Exec(ctx)
	switch {
	case err != nil:
		return err
	case n == 0:
		return &NotFoundError{includednetwork.Label}
	default:
		return nil
	}
}

// ExecX is like Exec, but panics if an error occurs.
func (indo *IncludedNetworkDeleteOne) ExecX(ctx context.Context) {
	indo.ind.ExecX(ctx)
}