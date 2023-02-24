// Code generated by ent, DO NOT EDIT.

package ent

import (
	"context"
	"errors"
	"fmt"

	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/schema/field"
	"github.com/gen0cide/laforge/ent/environment"
	"github.com/gen0cide/laforge/ent/scheduledstep"
	"github.com/google/uuid"
)

// ScheduledStepCreate is the builder for creating a ScheduledStep entity.
type ScheduledStepCreate struct {
	config
	mutation *ScheduledStepMutation
	hooks    []Hook
}

// SetHclID sets the "hcl_id" field.
func (ssc *ScheduledStepCreate) SetHclID(s string) *ScheduledStepCreate {
	ssc.mutation.SetHclID(s)
	return ssc
}

// SetName sets the "name" field.
func (ssc *ScheduledStepCreate) SetName(s string) *ScheduledStepCreate {
	ssc.mutation.SetName(s)
	return ssc
}

// SetDescription sets the "description" field.
func (ssc *ScheduledStepCreate) SetDescription(s string) *ScheduledStepCreate {
	ssc.mutation.SetDescription(s)
	return ssc
}

// SetStep sets the "step" field.
func (ssc *ScheduledStepCreate) SetStep(s string) *ScheduledStepCreate {
	ssc.mutation.SetStep(s)
	return ssc
}

// SetStartTime sets the "start_time" field.
func (ssc *ScheduledStepCreate) SetStartTime(i int64) *ScheduledStepCreate {
	ssc.mutation.SetStartTime(i)
	return ssc
}

// SetEndTime sets the "end_time" field.
func (ssc *ScheduledStepCreate) SetEndTime(i int64) *ScheduledStepCreate {
	ssc.mutation.SetEndTime(i)
	return ssc
}

// SetInterval sets the "interval" field.
func (ssc *ScheduledStepCreate) SetInterval(i int) *ScheduledStepCreate {
	ssc.mutation.SetInterval(i)
	return ssc
}

// SetRepeated sets the "repeated" field.
func (ssc *ScheduledStepCreate) SetRepeated(b bool) *ScheduledStepCreate {
	ssc.mutation.SetRepeated(b)
	return ssc
}

// SetID sets the "id" field.
func (ssc *ScheduledStepCreate) SetID(u uuid.UUID) *ScheduledStepCreate {
	ssc.mutation.SetID(u)
	return ssc
}

// SetNillableID sets the "id" field if the given value is not nil.
func (ssc *ScheduledStepCreate) SetNillableID(u *uuid.UUID) *ScheduledStepCreate {
	if u != nil {
		ssc.SetID(*u)
	}
	return ssc
}

// SetScheduledStepToEnvironmentID sets the "ScheduledStepToEnvironment" edge to the Environment entity by ID.
func (ssc *ScheduledStepCreate) SetScheduledStepToEnvironmentID(id uuid.UUID) *ScheduledStepCreate {
	ssc.mutation.SetScheduledStepToEnvironmentID(id)
	return ssc
}

// SetNillableScheduledStepToEnvironmentID sets the "ScheduledStepToEnvironment" edge to the Environment entity by ID if the given value is not nil.
func (ssc *ScheduledStepCreate) SetNillableScheduledStepToEnvironmentID(id *uuid.UUID) *ScheduledStepCreate {
	if id != nil {
		ssc = ssc.SetScheduledStepToEnvironmentID(*id)
	}
	return ssc
}

// SetScheduledStepToEnvironment sets the "ScheduledStepToEnvironment" edge to the Environment entity.
func (ssc *ScheduledStepCreate) SetScheduledStepToEnvironment(e *Environment) *ScheduledStepCreate {
	return ssc.SetScheduledStepToEnvironmentID(e.ID)
}

// Mutation returns the ScheduledStepMutation object of the builder.
func (ssc *ScheduledStepCreate) Mutation() *ScheduledStepMutation {
	return ssc.mutation
}

// Save creates the ScheduledStep in the database.
func (ssc *ScheduledStepCreate) Save(ctx context.Context) (*ScheduledStep, error) {
	var (
		err  error
		node *ScheduledStep
	)
	ssc.defaults()
	if len(ssc.hooks) == 0 {
		if err = ssc.check(); err != nil {
			return nil, err
		}
		node, err = ssc.sqlSave(ctx)
	} else {
		var mut Mutator = MutateFunc(func(ctx context.Context, m Mutation) (Value, error) {
			mutation, ok := m.(*ScheduledStepMutation)
			if !ok {
				return nil, fmt.Errorf("unexpected mutation type %T", m)
			}
			if err = ssc.check(); err != nil {
				return nil, err
			}
			ssc.mutation = mutation
			if node, err = ssc.sqlSave(ctx); err != nil {
				return nil, err
			}
			mutation.id = &node.ID
			mutation.done = true
			return node, err
		})
		for i := len(ssc.hooks) - 1; i >= 0; i-- {
			if ssc.hooks[i] == nil {
				return nil, fmt.Errorf("ent: uninitialized hook (forgotten import ent/runtime?)")
			}
			mut = ssc.hooks[i](mut)
		}
		v, err := mut.Mutate(ctx, ssc.mutation)
		if err != nil {
			return nil, err
		}
		nv, ok := v.(*ScheduledStep)
		if !ok {
			return nil, fmt.Errorf("unexpected node type %T returned from ScheduledStepMutation", v)
		}
		node = nv
	}
	return node, err
}

// SaveX calls Save and panics if Save returns an error.
func (ssc *ScheduledStepCreate) SaveX(ctx context.Context) *ScheduledStep {
	v, err := ssc.Save(ctx)
	if err != nil {
		panic(err)
	}
	return v
}

// Exec executes the query.
func (ssc *ScheduledStepCreate) Exec(ctx context.Context) error {
	_, err := ssc.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (ssc *ScheduledStepCreate) ExecX(ctx context.Context) {
	if err := ssc.Exec(ctx); err != nil {
		panic(err)
	}
}

// defaults sets the default values of the builder before save.
func (ssc *ScheduledStepCreate) defaults() {
	if _, ok := ssc.mutation.ID(); !ok {
		v := scheduledstep.DefaultID()
		ssc.mutation.SetID(v)
	}
}

// check runs all checks and user-defined validators on the builder.
func (ssc *ScheduledStepCreate) check() error {
	if _, ok := ssc.mutation.HclID(); !ok {
		return &ValidationError{Name: "hcl_id", err: errors.New(`ent: missing required field "ScheduledStep.hcl_id"`)}
	}
	if _, ok := ssc.mutation.Name(); !ok {
		return &ValidationError{Name: "name", err: errors.New(`ent: missing required field "ScheduledStep.name"`)}
	}
	if _, ok := ssc.mutation.Description(); !ok {
		return &ValidationError{Name: "description", err: errors.New(`ent: missing required field "ScheduledStep.description"`)}
	}
	if _, ok := ssc.mutation.Step(); !ok {
		return &ValidationError{Name: "step", err: errors.New(`ent: missing required field "ScheduledStep.step"`)}
	}
	if _, ok := ssc.mutation.StartTime(); !ok {
		return &ValidationError{Name: "start_time", err: errors.New(`ent: missing required field "ScheduledStep.start_time"`)}
	}
	if _, ok := ssc.mutation.EndTime(); !ok {
		return &ValidationError{Name: "end_time", err: errors.New(`ent: missing required field "ScheduledStep.end_time"`)}
	}
	if _, ok := ssc.mutation.Interval(); !ok {
		return &ValidationError{Name: "interval", err: errors.New(`ent: missing required field "ScheduledStep.interval"`)}
	}
	if _, ok := ssc.mutation.Repeated(); !ok {
		return &ValidationError{Name: "repeated", err: errors.New(`ent: missing required field "ScheduledStep.repeated"`)}
	}
	return nil
}

func (ssc *ScheduledStepCreate) sqlSave(ctx context.Context) (*ScheduledStep, error) {
	_node, _spec := ssc.createSpec()
	if err := sqlgraph.CreateNode(ctx, ssc.driver, _spec); err != nil {
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
	return _node, nil
}

func (ssc *ScheduledStepCreate) createSpec() (*ScheduledStep, *sqlgraph.CreateSpec) {
	var (
		_node = &ScheduledStep{config: ssc.config}
		_spec = &sqlgraph.CreateSpec{
			Table: scheduledstep.Table,
			ID: &sqlgraph.FieldSpec{
				Type:   field.TypeUUID,
				Column: scheduledstep.FieldID,
			},
		}
	)
	if id, ok := ssc.mutation.ID(); ok {
		_node.ID = id
		_spec.ID.Value = &id
	}
	if value, ok := ssc.mutation.HclID(); ok {
		_spec.Fields = append(_spec.Fields, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Value:  value,
			Column: scheduledstep.FieldHclID,
		})
		_node.HclID = value
	}
	if value, ok := ssc.mutation.Name(); ok {
		_spec.Fields = append(_spec.Fields, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Value:  value,
			Column: scheduledstep.FieldName,
		})
		_node.Name = value
	}
	if value, ok := ssc.mutation.Description(); ok {
		_spec.Fields = append(_spec.Fields, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Value:  value,
			Column: scheduledstep.FieldDescription,
		})
		_node.Description = value
	}
	if value, ok := ssc.mutation.Step(); ok {
		_spec.Fields = append(_spec.Fields, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Value:  value,
			Column: scheduledstep.FieldStep,
		})
		_node.Step = value
	}
	if value, ok := ssc.mutation.StartTime(); ok {
		_spec.Fields = append(_spec.Fields, &sqlgraph.FieldSpec{
			Type:   field.TypeInt64,
			Value:  value,
			Column: scheduledstep.FieldStartTime,
		})
		_node.StartTime = value
	}
	if value, ok := ssc.mutation.EndTime(); ok {
		_spec.Fields = append(_spec.Fields, &sqlgraph.FieldSpec{
			Type:   field.TypeInt64,
			Value:  value,
			Column: scheduledstep.FieldEndTime,
		})
		_node.EndTime = value
	}
	if value, ok := ssc.mutation.Interval(); ok {
		_spec.Fields = append(_spec.Fields, &sqlgraph.FieldSpec{
			Type:   field.TypeInt,
			Value:  value,
			Column: scheduledstep.FieldInterval,
		})
		_node.Interval = value
	}
	if value, ok := ssc.mutation.Repeated(); ok {
		_spec.Fields = append(_spec.Fields, &sqlgraph.FieldSpec{
			Type:   field.TypeBool,
			Value:  value,
			Column: scheduledstep.FieldRepeated,
		})
		_node.Repeated = value
	}
	if nodes := ssc.mutation.ScheduledStepToEnvironmentIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: true,
			Table:   scheduledstep.ScheduledStepToEnvironmentTable,
			Columns: []string{scheduledstep.ScheduledStepToEnvironmentColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeUUID,
					Column: environment.FieldID,
				},
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_node.environment_environment_to_scheduled_step = &nodes[0]
		_spec.Edges = append(_spec.Edges, edge)
	}
	return _node, _spec
}

// ScheduledStepCreateBulk is the builder for creating many ScheduledStep entities in bulk.
type ScheduledStepCreateBulk struct {
	config
	builders []*ScheduledStepCreate
}

// Save creates the ScheduledStep entities in the database.
func (sscb *ScheduledStepCreateBulk) Save(ctx context.Context) ([]*ScheduledStep, error) {
	specs := make([]*sqlgraph.CreateSpec, len(sscb.builders))
	nodes := make([]*ScheduledStep, len(sscb.builders))
	mutators := make([]Mutator, len(sscb.builders))
	for i := range sscb.builders {
		func(i int, root context.Context) {
			builder := sscb.builders[i]
			builder.defaults()
			var mut Mutator = MutateFunc(func(ctx context.Context, m Mutation) (Value, error) {
				mutation, ok := m.(*ScheduledStepMutation)
				if !ok {
					return nil, fmt.Errorf("unexpected mutation type %T", m)
				}
				if err := builder.check(); err != nil {
					return nil, err
				}
				builder.mutation = mutation
				nodes[i], specs[i] = builder.createSpec()
				var err error
				if i < len(mutators)-1 {
					_, err = mutators[i+1].Mutate(root, sscb.builders[i+1].mutation)
				} else {
					spec := &sqlgraph.BatchCreateSpec{Nodes: specs}
					// Invoke the actual operation on the latest mutation in the chain.
					if err = sqlgraph.BatchCreate(ctx, sscb.driver, spec); err != nil {
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
		if _, err := mutators[0].Mutate(ctx, sscb.builders[0].mutation); err != nil {
			return nil, err
		}
	}
	return nodes, nil
}

// SaveX is like Save, but panics if an error occurs.
func (sscb *ScheduledStepCreateBulk) SaveX(ctx context.Context) []*ScheduledStep {
	v, err := sscb.Save(ctx)
	if err != nil {
		panic(err)
	}
	return v
}

// Exec executes the query.
func (sscb *ScheduledStepCreateBulk) Exec(ctx context.Context) error {
	_, err := sscb.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (sscb *ScheduledStepCreateBulk) ExecX(ctx context.Context) {
	if err := sscb.Exec(ctx); err != nil {
		panic(err)
	}
}
