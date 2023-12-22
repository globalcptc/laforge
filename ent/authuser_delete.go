// Code generated by ent, DO NOT EDIT.

package ent

import (
	"context"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/schema/field"
	"github.com/gen0cide/laforge/ent/authuser"
	"github.com/gen0cide/laforge/ent/predicate"
)

// AuthUserDelete is the builder for deleting a AuthUser entity.
type AuthUserDelete struct {
	config
	hooks    []Hook
	mutation *AuthUserMutation
}

// Where appends a list predicates to the AuthUserDelete builder.
func (aud *AuthUserDelete) Where(ps ...predicate.AuthUser) *AuthUserDelete {
	aud.mutation.Where(ps...)
	return aud
}

// Exec executes the deletion query and returns how many vertices were deleted.
func (aud *AuthUserDelete) Exec(ctx context.Context) (int, error) {
	return withHooks(ctx, aud.sqlExec, aud.mutation, aud.hooks)
}

// ExecX is like Exec, but panics if an error occurs.
func (aud *AuthUserDelete) ExecX(ctx context.Context) int {
	n, err := aud.Exec(ctx)
	if err != nil {
		panic(err)
	}
	return n
}

func (aud *AuthUserDelete) sqlExec(ctx context.Context) (int, error) {
	_spec := sqlgraph.NewDeleteSpec(authuser.Table, sqlgraph.NewFieldSpec(authuser.FieldID, field.TypeUUID))
	if ps := aud.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	affected, err := sqlgraph.DeleteNodes(ctx, aud.driver, _spec)
	if err != nil && sqlgraph.IsConstraintError(err) {
		err = &ConstraintError{msg: err.Error(), wrap: err}
	}
	aud.mutation.done = true
	return affected, err
}

// AuthUserDeleteOne is the builder for deleting a single AuthUser entity.
type AuthUserDeleteOne struct {
	aud *AuthUserDelete
}

// Where appends a list predicates to the AuthUserDelete builder.
func (audo *AuthUserDeleteOne) Where(ps ...predicate.AuthUser) *AuthUserDeleteOne {
	audo.aud.mutation.Where(ps...)
	return audo
}

// Exec executes the deletion query.
func (audo *AuthUserDeleteOne) Exec(ctx context.Context) error {
	n, err := audo.aud.Exec(ctx)
	switch {
	case err != nil:
		return err
	case n == 0:
		return &NotFoundError{authuser.Label}
	default:
		return nil
	}
}

// ExecX is like Exec, but panics if an error occurs.
func (audo *AuthUserDeleteOne) ExecX(ctx context.Context) {
	if err := audo.Exec(ctx); err != nil {
		panic(err)
	}
}
