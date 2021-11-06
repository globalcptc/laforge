// Code generated by entc, DO NOT EDIT.

package ent

import (
	"context"
	"errors"
	"fmt"

	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/schema/field"
	"github.com/gen0cide/laforge/ent/disk"
	"github.com/gen0cide/laforge/ent/host"
	"github.com/google/uuid"
)

// DiskCreate is the builder for creating a Disk entity.
type DiskCreate struct {
	config
	mutation *DiskMutation
	hooks    []Hook
}

// SetSize sets the "size" field.
func (dc *DiskCreate) SetSize(i int) *DiskCreate {
	dc.mutation.SetSize(i)
	return dc
}

// SetID sets the "id" field.
func (dc *DiskCreate) SetID(u uuid.UUID) *DiskCreate {
	dc.mutation.SetID(u)
	return dc
}

// SetDiskToHostID sets the "DiskToHost" edge to the Host entity by ID.
func (dc *DiskCreate) SetDiskToHostID(id uuid.UUID) *DiskCreate {
	dc.mutation.SetDiskToHostID(id)
	return dc
}

// SetNillableDiskToHostID sets the "DiskToHost" edge to the Host entity by ID if the given value is not nil.
func (dc *DiskCreate) SetNillableDiskToHostID(id *uuid.UUID) *DiskCreate {
	if id != nil {
		dc = dc.SetDiskToHostID(*id)
	}
	return dc
}

// SetDiskToHost sets the "DiskToHost" edge to the Host entity.
func (dc *DiskCreate) SetDiskToHost(h *Host) *DiskCreate {
	return dc.SetDiskToHostID(h.ID)
}

// Mutation returns the DiskMutation object of the builder.
func (dc *DiskCreate) Mutation() *DiskMutation {
	return dc.mutation
}

// Save creates the Disk in the database.
func (dc *DiskCreate) Save(ctx context.Context) (*Disk, error) {
	var (
		err  error
		node *Disk
	)
	dc.defaults()
	if len(dc.hooks) == 0 {
		if err = dc.check(); err != nil {
			return nil, err
		}
		node, err = dc.sqlSave(ctx)
	} else {
		var mut Mutator = MutateFunc(func(ctx context.Context, m Mutation) (Value, error) {
			mutation, ok := m.(*DiskMutation)
			if !ok {
				return nil, fmt.Errorf("unexpected mutation type %T", m)
			}
			if err = dc.check(); err != nil {
				return nil, err
			}
			dc.mutation = mutation
			if node, err = dc.sqlSave(ctx); err != nil {
				return nil, err
			}
			mutation.id = &node.ID
			mutation.done = true
			return node, err
		})
		for i := len(dc.hooks) - 1; i >= 0; i-- {
			if dc.hooks[i] == nil {
				return nil, fmt.Errorf("ent: uninitialized hook (forgotten import ent/runtime?)")
			}
			mut = dc.hooks[i](mut)
		}
		if _, err := mut.Mutate(ctx, dc.mutation); err != nil {
			return nil, err
		}
	}
	return node, err
}

// SaveX calls Save and panics if Save returns an error.
func (dc *DiskCreate) SaveX(ctx context.Context) *Disk {
	v, err := dc.Save(ctx)
	if err != nil {
		panic(err)
	}
	return v
}

// Exec executes the query.
func (dc *DiskCreate) Exec(ctx context.Context) error {
	_, err := dc.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (dc *DiskCreate) ExecX(ctx context.Context) {
	if err := dc.Exec(ctx); err != nil {
		panic(err)
	}
}

// defaults sets the default values of the builder before save.
func (dc *DiskCreate) defaults() {
	if _, ok := dc.mutation.ID(); !ok {
		v := disk.DefaultID()
		dc.mutation.SetID(v)
	}
}

// check runs all checks and user-defined validators on the builder.
func (dc *DiskCreate) check() error {
	if _, ok := dc.mutation.Size(); !ok {
		return &ValidationError{Name: "size", err: errors.New(`ent: missing required field "size"`)}
	}
	if v, ok := dc.mutation.Size(); ok {
		if err := disk.SizeValidator(v); err != nil {
			return &ValidationError{Name: "size", err: fmt.Errorf(`ent: validator failed for field "size": %w`, err)}
		}
	}
	return nil
}

func (dc *DiskCreate) sqlSave(ctx context.Context) (*Disk, error) {
	_node, _spec := dc.createSpec()
	if err := sqlgraph.CreateNode(ctx, dc.driver, _spec); err != nil {
		if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{err.Error(), err}
		}
		return nil, err
	}
	if _spec.ID.Value != nil {
		_node.ID = _spec.ID.Value.(uuid.UUID)
	}
	return _node, nil
}

func (dc *DiskCreate) createSpec() (*Disk, *sqlgraph.CreateSpec) {
	var (
		_node = &Disk{config: dc.config}
		_spec = &sqlgraph.CreateSpec{
			Table: disk.Table,
			ID: &sqlgraph.FieldSpec{
				Type:   field.TypeUUID,
				Column: disk.FieldID,
			},
		}
	)
	if id, ok := dc.mutation.ID(); ok {
		_node.ID = id
		_spec.ID.Value = id
	}
	if value, ok := dc.mutation.Size(); ok {
		_spec.Fields = append(_spec.Fields, &sqlgraph.FieldSpec{
			Type:   field.TypeInt,
			Value:  value,
			Column: disk.FieldSize,
		})
		_node.Size = value
	}
	if nodes := dc.mutation.DiskToHostIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2O,
			Inverse: true,
			Table:   disk.DiskToHostTable,
			Columns: []string{disk.DiskToHostColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeUUID,
					Column: host.FieldID,
				},
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_node.host_host_to_disk = &nodes[0]
		_spec.Edges = append(_spec.Edges, edge)
	}
	return _node, _spec
}

// DiskCreateBulk is the builder for creating many Disk entities in bulk.
type DiskCreateBulk struct {
	config
	builders []*DiskCreate
}

// Save creates the Disk entities in the database.
func (dcb *DiskCreateBulk) Save(ctx context.Context) ([]*Disk, error) {
	specs := make([]*sqlgraph.CreateSpec, len(dcb.builders))
	nodes := make([]*Disk, len(dcb.builders))
	mutators := make([]Mutator, len(dcb.builders))
	for i := range dcb.builders {
		func(i int, root context.Context) {
			builder := dcb.builders[i]
			builder.defaults()
			var mut Mutator = MutateFunc(func(ctx context.Context, m Mutation) (Value, error) {
				mutation, ok := m.(*DiskMutation)
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
					_, err = mutators[i+1].Mutate(root, dcb.builders[i+1].mutation)
				} else {
					spec := &sqlgraph.BatchCreateSpec{Nodes: specs}
					// Invoke the actual operation on the latest mutation in the chain.
					if err = sqlgraph.BatchCreate(ctx, dcb.driver, spec); err != nil {
						if sqlgraph.IsConstraintError(err) {
							err = &ConstraintError{err.Error(), err}
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
		if _, err := mutators[0].Mutate(ctx, dcb.builders[0].mutation); err != nil {
			return nil, err
		}
	}
	return nodes, nil
}

// SaveX is like Save, but panics if an error occurs.
func (dcb *DiskCreateBulk) SaveX(ctx context.Context) []*Disk {
	v, err := dcb.Save(ctx)
	if err != nil {
		panic(err)
	}
	return v
}

// Exec executes the query.
func (dcb *DiskCreateBulk) Exec(ctx context.Context) error {
	_, err := dcb.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (dcb *DiskCreateBulk) ExecX(ctx context.Context) {
	if err := dcb.Exec(ctx); err != nil {
		panic(err)
	}
}
