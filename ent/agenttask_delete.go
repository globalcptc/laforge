// Code generated by ent, DO NOT EDIT.

package ent

import (
	"context"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/schema/field"
	"github.com/gen0cide/laforge/ent/agenttask"
	"github.com/gen0cide/laforge/ent/predicate"
)

// AgentTaskDelete is the builder for deleting a AgentTask entity.
type AgentTaskDelete struct {
	config
	hooks    []Hook
	mutation *AgentTaskMutation
}

// Where appends a list predicates to the AgentTaskDelete builder.
func (atd *AgentTaskDelete) Where(ps ...predicate.AgentTask) *AgentTaskDelete {
	atd.mutation.Where(ps...)
	return atd
}

// Exec executes the deletion query and returns how many vertices were deleted.
func (atd *AgentTaskDelete) Exec(ctx context.Context) (int, error) {
	return withHooks(ctx, atd.sqlExec, atd.mutation, atd.hooks)
}

// ExecX is like Exec, but panics if an error occurs.
func (atd *AgentTaskDelete) ExecX(ctx context.Context) int {
	n, err := atd.Exec(ctx)
	if err != nil {
		panic(err)
	}
	return n
}

func (atd *AgentTaskDelete) sqlExec(ctx context.Context) (int, error) {
	_spec := sqlgraph.NewDeleteSpec(agenttask.Table, sqlgraph.NewFieldSpec(agenttask.FieldID, field.TypeUUID))
	if ps := atd.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	affected, err := sqlgraph.DeleteNodes(ctx, atd.driver, _spec)
	if err != nil && sqlgraph.IsConstraintError(err) {
		err = &ConstraintError{msg: err.Error(), wrap: err}
	}
	atd.mutation.done = true
	return affected, err
}

// AgentTaskDeleteOne is the builder for deleting a single AgentTask entity.
type AgentTaskDeleteOne struct {
	atd *AgentTaskDelete
}

// Where appends a list predicates to the AgentTaskDelete builder.
func (atdo *AgentTaskDeleteOne) Where(ps ...predicate.AgentTask) *AgentTaskDeleteOne {
	atdo.atd.mutation.Where(ps...)
	return atdo
}

// Exec executes the deletion query.
func (atdo *AgentTaskDeleteOne) Exec(ctx context.Context) error {
	n, err := atdo.atd.Exec(ctx)
	switch {
	case err != nil:
		return err
	case n == 0:
		return &NotFoundError{agenttask.Label}
	default:
		return nil
	}
}

// ExecX is like Exec, but panics if an error occurs.
func (atdo *AgentTaskDeleteOne) ExecX(ctx context.Context) {
	if err := atdo.Exec(ctx); err != nil {
		panic(err)
	}
}
