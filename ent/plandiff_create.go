// Code generated by ent, DO NOT EDIT.

package ent

import (
	"context"
	"errors"
	"fmt"

	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/schema/field"
	"github.com/gen0cide/laforge/ent/buildcommit"
	"github.com/gen0cide/laforge/ent/plan"
	"github.com/gen0cide/laforge/ent/plandiff"
	"github.com/google/uuid"
)

// PlanDiffCreate is the builder for creating a PlanDiff entity.
type PlanDiffCreate struct {
	config
	mutation *PlanDiffMutation
	hooks    []Hook
}

// SetRevision sets the "revision" field.
func (pdc *PlanDiffCreate) SetRevision(i int) *PlanDiffCreate {
	pdc.mutation.SetRevision(i)
	return pdc
}

// SetNewState sets the "new_state" field.
func (pdc *PlanDiffCreate) SetNewState(ps plandiff.NewState) *PlanDiffCreate {
	pdc.mutation.SetNewState(ps)
	return pdc
}

// SetID sets the "id" field.
func (pdc *PlanDiffCreate) SetID(u uuid.UUID) *PlanDiffCreate {
	pdc.mutation.SetID(u)
	return pdc
}

// SetNillableID sets the "id" field if the given value is not nil.
func (pdc *PlanDiffCreate) SetNillableID(u *uuid.UUID) *PlanDiffCreate {
	if u != nil {
		pdc.SetID(*u)
	}
	return pdc
}

// SetBuildCommitID sets the "BuildCommit" edge to the BuildCommit entity by ID.
func (pdc *PlanDiffCreate) SetBuildCommitID(id uuid.UUID) *PlanDiffCreate {
	pdc.mutation.SetBuildCommitID(id)
	return pdc
}

// SetBuildCommit sets the "BuildCommit" edge to the BuildCommit entity.
func (pdc *PlanDiffCreate) SetBuildCommit(b *BuildCommit) *PlanDiffCreate {
	return pdc.SetBuildCommitID(b.ID)
}

// SetPlanID sets the "Plan" edge to the Plan entity by ID.
func (pdc *PlanDiffCreate) SetPlanID(id uuid.UUID) *PlanDiffCreate {
	pdc.mutation.SetPlanID(id)
	return pdc
}

// SetPlan sets the "Plan" edge to the Plan entity.
func (pdc *PlanDiffCreate) SetPlan(p *Plan) *PlanDiffCreate {
	return pdc.SetPlanID(p.ID)
}

// Mutation returns the PlanDiffMutation object of the builder.
func (pdc *PlanDiffCreate) Mutation() *PlanDiffMutation {
	return pdc.mutation
}

// Save creates the PlanDiff in the database.
func (pdc *PlanDiffCreate) Save(ctx context.Context) (*PlanDiff, error) {
	pdc.defaults()
	return withHooks(ctx, pdc.sqlSave, pdc.mutation, pdc.hooks)
}

// SaveX calls Save and panics if Save returns an error.
func (pdc *PlanDiffCreate) SaveX(ctx context.Context) *PlanDiff {
	v, err := pdc.Save(ctx)
	if err != nil {
		panic(err)
	}
	return v
}

// Exec executes the query.
func (pdc *PlanDiffCreate) Exec(ctx context.Context) error {
	_, err := pdc.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (pdc *PlanDiffCreate) ExecX(ctx context.Context) {
	if err := pdc.Exec(ctx); err != nil {
		panic(err)
	}
}

// defaults sets the default values of the builder before save.
func (pdc *PlanDiffCreate) defaults() {
	if _, ok := pdc.mutation.ID(); !ok {
		v := plandiff.DefaultID()
		pdc.mutation.SetID(v)
	}
}

// check runs all checks and user-defined validators on the builder.
func (pdc *PlanDiffCreate) check() error {
	if _, ok := pdc.mutation.Revision(); !ok {
		return &ValidationError{Name: "revision", err: errors.New(`ent: missing required field "PlanDiff.revision"`)}
	}
	if _, ok := pdc.mutation.NewState(); !ok {
		return &ValidationError{Name: "new_state", err: errors.New(`ent: missing required field "PlanDiff.new_state"`)}
	}
	if v, ok := pdc.mutation.NewState(); ok {
		if err := plandiff.NewStateValidator(v); err != nil {
			return &ValidationError{Name: "new_state", err: fmt.Errorf(`ent: validator failed for field "PlanDiff.new_state": %w`, err)}
		}
	}
	if _, ok := pdc.mutation.BuildCommitID(); !ok {
		return &ValidationError{Name: "BuildCommit", err: errors.New(`ent: missing required edge "PlanDiff.BuildCommit"`)}
	}
	if _, ok := pdc.mutation.PlanID(); !ok {
		return &ValidationError{Name: "Plan", err: errors.New(`ent: missing required edge "PlanDiff.Plan"`)}
	}
	return nil
}

func (pdc *PlanDiffCreate) sqlSave(ctx context.Context) (*PlanDiff, error) {
	if err := pdc.check(); err != nil {
		return nil, err
	}
	_node, _spec := pdc.createSpec()
	if err := sqlgraph.CreateNode(ctx, pdc.driver, _spec); err != nil {
		if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return nil, err
	}
	if _spec.ID.Value != nil {
		if id, ok := _spec.ID.Value.(*uuid.UUID); ok {
			_node.ID = *id
		} else if err := _node.ID.Scan(_spec.ID.Value); err != nil {
			return nil, err
		}
	}
	pdc.mutation.id = &_node.ID
	pdc.mutation.done = true
	return _node, nil
}

func (pdc *PlanDiffCreate) createSpec() (*PlanDiff, *sqlgraph.CreateSpec) {
	var (
		_node = &PlanDiff{config: pdc.config}
		_spec = sqlgraph.NewCreateSpec(plandiff.Table, sqlgraph.NewFieldSpec(plandiff.FieldID, field.TypeUUID))
	)
	if id, ok := pdc.mutation.ID(); ok {
		_node.ID = id
		_spec.ID.Value = &id
	}
	if value, ok := pdc.mutation.Revision(); ok {
		_spec.SetField(plandiff.FieldRevision, field.TypeInt, value)
		_node.Revision = value
	}
	if value, ok := pdc.mutation.NewState(); ok {
		_spec.SetField(plandiff.FieldNewState, field.TypeEnum, value)
		_node.NewState = value
	}
	if nodes := pdc.mutation.BuildCommitIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: false,
			Table:   plandiff.BuildCommitTable,
			Columns: []string{plandiff.BuildCommitColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(buildcommit.FieldID, field.TypeUUID),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_node.plan_diff_build_commit = &nodes[0]
		_spec.Edges = append(_spec.Edges, edge)
	}
	if nodes := pdc.mutation.PlanIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: false,
			Table:   plandiff.PlanTable,
			Columns: []string{plandiff.PlanColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(plan.FieldID, field.TypeUUID),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_node.plan_diff_plan = &nodes[0]
		_spec.Edges = append(_spec.Edges, edge)
	}
	return _node, _spec
}

// PlanDiffCreateBulk is the builder for creating many PlanDiff entities in bulk.
type PlanDiffCreateBulk struct {
	config
	err      error
	builders []*PlanDiffCreate
}

// Save creates the PlanDiff entities in the database.
func (pdcb *PlanDiffCreateBulk) Save(ctx context.Context) ([]*PlanDiff, error) {
	if pdcb.err != nil {
		return nil, pdcb.err
	}
	specs := make([]*sqlgraph.CreateSpec, len(pdcb.builders))
	nodes := make([]*PlanDiff, len(pdcb.builders))
	mutators := make([]Mutator, len(pdcb.builders))
	for i := range pdcb.builders {
		func(i int, root context.Context) {
			builder := pdcb.builders[i]
			builder.defaults()
			var mut Mutator = MutateFunc(func(ctx context.Context, m Mutation) (Value, error) {
				mutation, ok := m.(*PlanDiffMutation)
				if !ok {
					return nil, fmt.Errorf("unexpected mutation type %T", m)
				}
				if err := builder.check(); err != nil {
					return nil, err
				}
				builder.mutation = mutation
				var err error
				nodes[i], specs[i] = builder.createSpec()
				if i < len(mutators)-1 {
					_, err = mutators[i+1].Mutate(root, pdcb.builders[i+1].mutation)
				} else {
					spec := &sqlgraph.BatchCreateSpec{Nodes: specs}
					// Invoke the actual operation on the latest mutation in the chain.
					if err = sqlgraph.BatchCreate(ctx, pdcb.driver, spec); err != nil {
						if sqlgraph.IsConstraintError(err) {
							err = &ConstraintError{msg: err.Error(), wrap: err}
						}
					}
				}
				if err != nil {
					return nil, err
				}
				mutation.id = &nodes[i].ID
				mutation.done = true
				return nodes[i], nil
			})
			for i := len(builder.hooks) - 1; i >= 0; i-- {
				mut = builder.hooks[i](mut)
			}
			mutators[i] = mut
		}(i, ctx)
	}
	if len(mutators) > 0 {
		if _, err := mutators[0].Mutate(ctx, pdcb.builders[0].mutation); err != nil {
			return nil, err
		}
	}
	return nodes, nil
}

// SaveX is like Save, but panics if an error occurs.
func (pdcb *PlanDiffCreateBulk) SaveX(ctx context.Context) []*PlanDiff {
	v, err := pdcb.Save(ctx)
	if err != nil {
		panic(err)
	}
	return v
}

// Exec executes the query.
func (pdcb *PlanDiffCreateBulk) Exec(ctx context.Context) error {
	_, err := pdcb.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (pdcb *PlanDiffCreateBulk) ExecX(ctx context.Context) {
	if err := pdcb.Exec(ctx); err != nil {
		panic(err)
	}
}
