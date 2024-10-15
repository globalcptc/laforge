// Code generated by ent, DO NOT EDIT.

package ent

import (
	"context"
	"errors"
	"fmt"

	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/schema/field"
	"github.com/gen0cide/laforge/ent/ginfilemiddleware"
	"github.com/gen0cide/laforge/ent/provisionedhost"
	"github.com/gen0cide/laforge/ent/provisioningscheduledstep"
	"github.com/gen0cide/laforge/ent/provisioningstep"
	"github.com/google/uuid"
)

// GinFileMiddlewareCreate is the builder for creating a GinFileMiddleware entity.
type GinFileMiddlewareCreate struct {
	config
	mutation *GinFileMiddlewareMutation
	hooks    []Hook
}

// SetURLID sets the "url_id" field.
func (gfmc *GinFileMiddlewareCreate) SetURLID(s string) *GinFileMiddlewareCreate {
	gfmc.mutation.SetURLID(s)
	return gfmc
}

// SetFilePath sets the "file_path" field.
func (gfmc *GinFileMiddlewareCreate) SetFilePath(s string) *GinFileMiddlewareCreate {
	gfmc.mutation.SetFilePath(s)
	return gfmc
}

// SetAccessed sets the "accessed" field.
func (gfmc *GinFileMiddlewareCreate) SetAccessed(b bool) *GinFileMiddlewareCreate {
	gfmc.mutation.SetAccessed(b)
	return gfmc
}

// SetNillableAccessed sets the "accessed" field if the given value is not nil.
func (gfmc *GinFileMiddlewareCreate) SetNillableAccessed(b *bool) *GinFileMiddlewareCreate {
	if b != nil {
		gfmc.SetAccessed(*b)
	}
	return gfmc
}

// SetID sets the "id" field.
func (gfmc *GinFileMiddlewareCreate) SetID(u uuid.UUID) *GinFileMiddlewareCreate {
	gfmc.mutation.SetID(u)
	return gfmc
}

// SetNillableID sets the "id" field if the given value is not nil.
func (gfmc *GinFileMiddlewareCreate) SetNillableID(u *uuid.UUID) *GinFileMiddlewareCreate {
	if u != nil {
		gfmc.SetID(*u)
	}
	return gfmc
}

// SetProvisionedHostID sets the "ProvisionedHost" edge to the ProvisionedHost entity by ID.
func (gfmc *GinFileMiddlewareCreate) SetProvisionedHostID(id uuid.UUID) *GinFileMiddlewareCreate {
	gfmc.mutation.SetProvisionedHostID(id)
	return gfmc
}

// SetNillableProvisionedHostID sets the "ProvisionedHost" edge to the ProvisionedHost entity by ID if the given value is not nil.
func (gfmc *GinFileMiddlewareCreate) SetNillableProvisionedHostID(id *uuid.UUID) *GinFileMiddlewareCreate {
	if id != nil {
		gfmc = gfmc.SetProvisionedHostID(*id)
	}
	return gfmc
}

// SetProvisionedHost sets the "ProvisionedHost" edge to the ProvisionedHost entity.
func (gfmc *GinFileMiddlewareCreate) SetProvisionedHost(p *ProvisionedHost) *GinFileMiddlewareCreate {
	return gfmc.SetProvisionedHostID(p.ID)
}

// SetProvisioningStepID sets the "ProvisioningStep" edge to the ProvisioningStep entity by ID.
func (gfmc *GinFileMiddlewareCreate) SetProvisioningStepID(id uuid.UUID) *GinFileMiddlewareCreate {
	gfmc.mutation.SetProvisioningStepID(id)
	return gfmc
}

// SetNillableProvisioningStepID sets the "ProvisioningStep" edge to the ProvisioningStep entity by ID if the given value is not nil.
func (gfmc *GinFileMiddlewareCreate) SetNillableProvisioningStepID(id *uuid.UUID) *GinFileMiddlewareCreate {
	if id != nil {
		gfmc = gfmc.SetProvisioningStepID(*id)
	}
	return gfmc
}

// SetProvisioningStep sets the "ProvisioningStep" edge to the ProvisioningStep entity.
func (gfmc *GinFileMiddlewareCreate) SetProvisioningStep(p *ProvisioningStep) *GinFileMiddlewareCreate {
	return gfmc.SetProvisioningStepID(p.ID)
}

// SetProvisioningScheduledStepID sets the "ProvisioningScheduledStep" edge to the ProvisioningScheduledStep entity by ID.
func (gfmc *GinFileMiddlewareCreate) SetProvisioningScheduledStepID(id uuid.UUID) *GinFileMiddlewareCreate {
	gfmc.mutation.SetProvisioningScheduledStepID(id)
	return gfmc
}

// SetNillableProvisioningScheduledStepID sets the "ProvisioningScheduledStep" edge to the ProvisioningScheduledStep entity by ID if the given value is not nil.
func (gfmc *GinFileMiddlewareCreate) SetNillableProvisioningScheduledStepID(id *uuid.UUID) *GinFileMiddlewareCreate {
	if id != nil {
		gfmc = gfmc.SetProvisioningScheduledStepID(*id)
	}
	return gfmc
}

// SetProvisioningScheduledStep sets the "ProvisioningScheduledStep" edge to the ProvisioningScheduledStep entity.
func (gfmc *GinFileMiddlewareCreate) SetProvisioningScheduledStep(p *ProvisioningScheduledStep) *GinFileMiddlewareCreate {
	return gfmc.SetProvisioningScheduledStepID(p.ID)
}

// Mutation returns the GinFileMiddlewareMutation object of the builder.
func (gfmc *GinFileMiddlewareCreate) Mutation() *GinFileMiddlewareMutation {
	return gfmc.mutation
}

// Save creates the GinFileMiddleware in the database.
func (gfmc *GinFileMiddlewareCreate) Save(ctx context.Context) (*GinFileMiddleware, error) {
	gfmc.defaults()
	return withHooks(ctx, gfmc.sqlSave, gfmc.mutation, gfmc.hooks)
}

// SaveX calls Save and panics if Save returns an error.
func (gfmc *GinFileMiddlewareCreate) SaveX(ctx context.Context) *GinFileMiddleware {
	v, err := gfmc.Save(ctx)
	if err != nil {
		panic(err)
	}
	return v
}

// Exec executes the query.
func (gfmc *GinFileMiddlewareCreate) Exec(ctx context.Context) error {
	_, err := gfmc.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (gfmc *GinFileMiddlewareCreate) ExecX(ctx context.Context) {
	if err := gfmc.Exec(ctx); err != nil {
		panic(err)
	}
}

// defaults sets the default values of the builder before save.
func (gfmc *GinFileMiddlewareCreate) defaults() {
	if _, ok := gfmc.mutation.Accessed(); !ok {
		v := ginfilemiddleware.DefaultAccessed
		gfmc.mutation.SetAccessed(v)
	}
	if _, ok := gfmc.mutation.ID(); !ok {
		v := ginfilemiddleware.DefaultID()
		gfmc.mutation.SetID(v)
	}
}

// check runs all checks and user-defined validators on the builder.
func (gfmc *GinFileMiddlewareCreate) check() error {
	if _, ok := gfmc.mutation.URLID(); !ok {
		return &ValidationError{Name: "url_id", err: errors.New(`ent: missing required field "GinFileMiddleware.url_id"`)}
	}
	if _, ok := gfmc.mutation.FilePath(); !ok {
		return &ValidationError{Name: "file_path", err: errors.New(`ent: missing required field "GinFileMiddleware.file_path"`)}
	}
	if _, ok := gfmc.mutation.Accessed(); !ok {
		return &ValidationError{Name: "accessed", err: errors.New(`ent: missing required field "GinFileMiddleware.accessed"`)}
	}
	return nil
}

func (gfmc *GinFileMiddlewareCreate) sqlSave(ctx context.Context) (*GinFileMiddleware, error) {
	if err := gfmc.check(); err != nil {
		return nil, err
	}
	_node, _spec := gfmc.createSpec()
	if err := sqlgraph.CreateNode(ctx, gfmc.driver, _spec); err != nil {
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
	gfmc.mutation.id = &_node.ID
	gfmc.mutation.done = true
	return _node, nil
}

func (gfmc *GinFileMiddlewareCreate) createSpec() (*GinFileMiddleware, *sqlgraph.CreateSpec) {
	var (
		_node = &GinFileMiddleware{config: gfmc.config}
		_spec = sqlgraph.NewCreateSpec(ginfilemiddleware.Table, sqlgraph.NewFieldSpec(ginfilemiddleware.FieldID, field.TypeUUID))
	)
	if id, ok := gfmc.mutation.ID(); ok {
		_node.ID = id
		_spec.ID.Value = &id
	}
	if value, ok := gfmc.mutation.URLID(); ok {
		_spec.SetField(ginfilemiddleware.FieldURLID, field.TypeString, value)
		_node.URLID = value
	}
	if value, ok := gfmc.mutation.FilePath(); ok {
		_spec.SetField(ginfilemiddleware.FieldFilePath, field.TypeString, value)
		_node.FilePath = value
	}
	if value, ok := gfmc.mutation.Accessed(); ok {
		_spec.SetField(ginfilemiddleware.FieldAccessed, field.TypeBool, value)
		_node.Accessed = value
	}
	if nodes := gfmc.mutation.ProvisionedHostIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2O,
			Inverse: false,
			Table:   ginfilemiddleware.ProvisionedHostTable,
			Columns: []string{ginfilemiddleware.ProvisionedHostColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(provisionedhost.FieldID, field.TypeUUID),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges = append(_spec.Edges, edge)
	}
	if nodes := gfmc.mutation.ProvisioningStepIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2O,
			Inverse: false,
			Table:   ginfilemiddleware.ProvisioningStepTable,
			Columns: []string{ginfilemiddleware.ProvisioningStepColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(provisioningstep.FieldID, field.TypeUUID),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges = append(_spec.Edges, edge)
	}
	if nodes := gfmc.mutation.ProvisioningScheduledStepIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2O,
			Inverse: false,
			Table:   ginfilemiddleware.ProvisioningScheduledStepTable,
			Columns: []string{ginfilemiddleware.ProvisioningScheduledStepColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(provisioningscheduledstep.FieldID, field.TypeUUID),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges = append(_spec.Edges, edge)
	}
	return _node, _spec
}

// GinFileMiddlewareCreateBulk is the builder for creating many GinFileMiddleware entities in bulk.
type GinFileMiddlewareCreateBulk struct {
	config
	err      error
	builders []*GinFileMiddlewareCreate
}

// Save creates the GinFileMiddleware entities in the database.
func (gfmcb *GinFileMiddlewareCreateBulk) Save(ctx context.Context) ([]*GinFileMiddleware, error) {
	if gfmcb.err != nil {
		return nil, gfmcb.err
	}
	specs := make([]*sqlgraph.CreateSpec, len(gfmcb.builders))
	nodes := make([]*GinFileMiddleware, len(gfmcb.builders))
	mutators := make([]Mutator, len(gfmcb.builders))
	for i := range gfmcb.builders {
		func(i int, root context.Context) {
			builder := gfmcb.builders[i]
			builder.defaults()
			var mut Mutator = MutateFunc(func(ctx context.Context, m Mutation) (Value, error) {
				mutation, ok := m.(*GinFileMiddlewareMutation)
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
					_, err = mutators[i+1].Mutate(root, gfmcb.builders[i+1].mutation)
				} else {
					spec := &sqlgraph.BatchCreateSpec{Nodes: specs}
					// Invoke the actual operation on the latest mutation in the chain.
					if err = sqlgraph.BatchCreate(ctx, gfmcb.driver, spec); err != nil {
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
		if _, err := mutators[0].Mutate(ctx, gfmcb.builders[0].mutation); err != nil {
			return nil, err
		}
	}
	return nodes, nil
}

// SaveX is like Save, but panics if an error occurs.
func (gfmcb *GinFileMiddlewareCreateBulk) SaveX(ctx context.Context) []*GinFileMiddleware {
	v, err := gfmcb.Save(ctx)
	if err != nil {
		panic(err)
	}
	return v
}

// Exec executes the query.
func (gfmcb *GinFileMiddlewareCreateBulk) Exec(ctx context.Context) error {
	_, err := gfmcb.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (gfmcb *GinFileMiddlewareCreateBulk) ExecX(ctx context.Context) {
	if err := gfmcb.Exec(ctx); err != nil {
		panic(err)
	}
}
