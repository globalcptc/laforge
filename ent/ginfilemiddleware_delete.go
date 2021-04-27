// Code generated by entc, DO NOT EDIT.

package ent

import (
	"context"
	"fmt"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/schema/field"
	"github.com/gen0cide/laforge/ent/ginfilemiddleware"
	"github.com/gen0cide/laforge/ent/predicate"
)

// GinFileMiddlewareDelete is the builder for deleting a GinFileMiddleware entity.
type GinFileMiddlewareDelete struct {
	config
	hooks    []Hook
	mutation *GinFileMiddlewareMutation
}

// Where adds a new predicate to the GinFileMiddlewareDelete builder.
func (gfmd *GinFileMiddlewareDelete) Where(ps ...predicate.GinFileMiddleware) *GinFileMiddlewareDelete {
	gfmd.mutation.predicates = append(gfmd.mutation.predicates, ps...)
	return gfmd
}

// Exec executes the deletion query and returns how many vertices were deleted.
func (gfmd *GinFileMiddlewareDelete) Exec(ctx context.Context) (int, error) {
	var (
		err      error
		affected int
	)
	if len(gfmd.hooks) == 0 {
		affected, err = gfmd.sqlExec(ctx)
	} else {
		var mut Mutator = MutateFunc(func(ctx context.Context, m Mutation) (Value, error) {
			mutation, ok := m.(*GinFileMiddlewareMutation)
			if !ok {
				return nil, fmt.Errorf("unexpected mutation type %T", m)
			}
			gfmd.mutation = mutation
			affected, err = gfmd.sqlExec(ctx)
			mutation.done = true
			return affected, err
		})
		for i := len(gfmd.hooks) - 1; i >= 0; i-- {
			mut = gfmd.hooks[i](mut)
		}
		if _, err := mut.Mutate(ctx, gfmd.mutation); err != nil {
			return 0, err
		}
	}
	return affected, err
}

// ExecX is like Exec, but panics if an error occurs.
func (gfmd *GinFileMiddlewareDelete) ExecX(ctx context.Context) int {
	n, err := gfmd.Exec(ctx)
	if err != nil {
		panic(err)
	}
	return n
}

func (gfmd *GinFileMiddlewareDelete) sqlExec(ctx context.Context) (int, error) {
	_spec := &sqlgraph.DeleteSpec{
		Node: &sqlgraph.NodeSpec{
			Table: ginfilemiddleware.Table,
			ID: &sqlgraph.FieldSpec{
				Type:   field.TypeUUID,
				Column: ginfilemiddleware.FieldID,
			},
		},
	}
	if ps := gfmd.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	return sqlgraph.DeleteNodes(ctx, gfmd.driver, _spec)
}

// GinFileMiddlewareDeleteOne is the builder for deleting a single GinFileMiddleware entity.
type GinFileMiddlewareDeleteOne struct {
	gfmd *GinFileMiddlewareDelete
}

// Exec executes the deletion query.
func (gfmdo *GinFileMiddlewareDeleteOne) Exec(ctx context.Context) error {
	n, err := gfmdo.gfmd.Exec(ctx)
	switch {
	case err != nil:
		return err
	case n == 0:
		return &NotFoundError{ginfilemiddleware.Label}
	default:
		return nil
	}
}

// ExecX is like Exec, but panics if an error occurs.
func (gfmdo *GinFileMiddlewareDeleteOne) ExecX(ctx context.Context) {
	gfmdo.gfmd.ExecX(ctx)
}
