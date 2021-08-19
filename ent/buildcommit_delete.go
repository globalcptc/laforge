// Code generated by entc, DO NOT EDIT.

package ent

import (
	"context"
	"fmt"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/schema/field"
	"github.com/gen0cide/laforge/ent/buildcommit"
	"github.com/gen0cide/laforge/ent/predicate"
)

// BuildCommitDelete is the builder for deleting a BuildCommit entity.
type BuildCommitDelete struct {
	config
	hooks    []Hook
	mutation *BuildCommitMutation
}

// Where adds a new predicate to the BuildCommitDelete builder.
func (bcd *BuildCommitDelete) Where(ps ...predicate.BuildCommit) *BuildCommitDelete {
	bcd.mutation.predicates = append(bcd.mutation.predicates, ps...)
	return bcd
}

// Exec executes the deletion query and returns how many vertices were deleted.
func (bcd *BuildCommitDelete) Exec(ctx context.Context) (int, error) {
	var (
		err      error
		affected int
	)
	if len(bcd.hooks) == 0 {
		affected, err = bcd.sqlExec(ctx)
	} else {
		var mut Mutator = MutateFunc(func(ctx context.Context, m Mutation) (Value, error) {
			mutation, ok := m.(*BuildCommitMutation)
			if !ok {
				return nil, fmt.Errorf("unexpected mutation type %T", m)
			}
			bcd.mutation = mutation
			affected, err = bcd.sqlExec(ctx)
			mutation.done = true
			return affected, err
		})
		for i := len(bcd.hooks) - 1; i >= 0; i-- {
			mut = bcd.hooks[i](mut)
		}
		if _, err := mut.Mutate(ctx, bcd.mutation); err != nil {
			return 0, err
		}
	}
	return affected, err
}

// ExecX is like Exec, but panics if an error occurs.
func (bcd *BuildCommitDelete) ExecX(ctx context.Context) int {
	n, err := bcd.Exec(ctx)
	if err != nil {
		panic(err)
	}
	return n
}

func (bcd *BuildCommitDelete) sqlExec(ctx context.Context) (int, error) {
	_spec := &sqlgraph.DeleteSpec{
		Node: &sqlgraph.NodeSpec{
			Table: buildcommit.Table,
			ID: &sqlgraph.FieldSpec{
				Type:   field.TypeUUID,
				Column: buildcommit.FieldID,
			},
		},
	}
	if ps := bcd.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	return sqlgraph.DeleteNodes(ctx, bcd.driver, _spec)
}

// BuildCommitDeleteOne is the builder for deleting a single BuildCommit entity.
type BuildCommitDeleteOne struct {
	bcd *BuildCommitDelete
}

// Exec executes the deletion query.
func (bcdo *BuildCommitDeleteOne) Exec(ctx context.Context) error {
	n, err := bcdo.bcd.Exec(ctx)
	switch {
	case err != nil:
		return err
	case n == 0:
		return &NotFoundError{buildcommit.Label}
	default:
		return nil
	}
}

// ExecX is like Exec, but panics if an error occurs.
func (bcdo *BuildCommitDeleteOne) ExecX(ctx context.Context) {
	bcdo.bcd.ExecX(ctx)
}