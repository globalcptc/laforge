// Code generated by ent, DO NOT EDIT.

package ent

import (
	"context"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/schema/field"
	"github.com/gen0cide/laforge/ent/agentstatus"
	"github.com/gen0cide/laforge/ent/predicate"
)

// AgentStatusDelete is the builder for deleting a AgentStatus entity.
type AgentStatusDelete struct {
	config
	hooks    []Hook
	mutation *AgentStatusMutation
}

// Where appends a list predicates to the AgentStatusDelete builder.
func (asd *AgentStatusDelete) Where(ps ...predicate.AgentStatus) *AgentStatusDelete {
	asd.mutation.Where(ps...)
	return asd
}

// Exec executes the deletion query and returns how many vertices were deleted.
func (asd *AgentStatusDelete) Exec(ctx context.Context) (int, error) {
	return withHooks(ctx, asd.sqlExec, asd.mutation, asd.hooks)
}

// ExecX is like Exec, but panics if an error occurs.
func (asd *AgentStatusDelete) ExecX(ctx context.Context) int {
	n, err := asd.Exec(ctx)
	if err != nil {
		panic(err)
	}
	return n
}

func (asd *AgentStatusDelete) sqlExec(ctx context.Context) (int, error) {
	_spec := sqlgraph.NewDeleteSpec(agentstatus.Table, sqlgraph.NewFieldSpec(agentstatus.FieldID, field.TypeUUID))
	if ps := asd.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	affected, err := sqlgraph.DeleteNodes(ctx, asd.driver, _spec)
	if err != nil && sqlgraph.IsConstraintError(err) {
		err = &ConstraintError{msg: err.Error(), wrap: err}
	}
	asd.mutation.done = true
	return affected, err
}

// AgentStatusDeleteOne is the builder for deleting a single AgentStatus entity.
type AgentStatusDeleteOne struct {
	asd *AgentStatusDelete
}

// Where appends a list predicates to the AgentStatusDelete builder.
func (asdo *AgentStatusDeleteOne) Where(ps ...predicate.AgentStatus) *AgentStatusDeleteOne {
	asdo.asd.mutation.Where(ps...)
	return asdo
}

// Exec executes the deletion query.
func (asdo *AgentStatusDeleteOne) Exec(ctx context.Context) error {
	n, err := asdo.asd.Exec(ctx)
	switch {
	case err != nil:
		return err
	case n == 0:
		return &NotFoundError{agentstatus.Label}
	default:
		return nil
	}
}

// ExecX is like Exec, but panics if an error occurs.
func (asdo *AgentStatusDeleteOne) ExecX(ctx context.Context) {
	if err := asdo.Exec(ctx); err != nil {
		panic(err)
	}
}
