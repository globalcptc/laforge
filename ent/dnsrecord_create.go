// Code generated by ent, DO NOT EDIT.

package ent

import (
	"context"
	"errors"
	"fmt"

	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/schema/field"
	"github.com/gen0cide/laforge/ent/dnsrecord"
	"github.com/gen0cide/laforge/ent/environment"
	"github.com/google/uuid"
)

// DNSRecordCreate is the builder for creating a DNSRecord entity.
type DNSRecordCreate struct {
	config
	mutation *DNSRecordMutation
	hooks    []Hook
}

// SetHCLID sets the "hcl_id" field.
func (drc *DNSRecordCreate) SetHCLID(s string) *DNSRecordCreate {
	drc.mutation.SetHCLID(s)
	return drc
}

// SetName sets the "name" field.
func (drc *DNSRecordCreate) SetName(s string) *DNSRecordCreate {
	drc.mutation.SetName(s)
	return drc
}

// SetValues sets the "values" field.
func (drc *DNSRecordCreate) SetValues(s []string) *DNSRecordCreate {
	drc.mutation.SetValues(s)
	return drc
}

// SetType sets the "type" field.
func (drc *DNSRecordCreate) SetType(s string) *DNSRecordCreate {
	drc.mutation.SetType(s)
	return drc
}

// SetZone sets the "zone" field.
func (drc *DNSRecordCreate) SetZone(s string) *DNSRecordCreate {
	drc.mutation.SetZone(s)
	return drc
}

// SetVars sets the "vars" field.
func (drc *DNSRecordCreate) SetVars(m map[string]string) *DNSRecordCreate {
	drc.mutation.SetVars(m)
	return drc
}

// SetDisabled sets the "disabled" field.
func (drc *DNSRecordCreate) SetDisabled(b bool) *DNSRecordCreate {
	drc.mutation.SetDisabled(b)
	return drc
}

// SetTags sets the "tags" field.
func (drc *DNSRecordCreate) SetTags(m map[string]string) *DNSRecordCreate {
	drc.mutation.SetTags(m)
	return drc
}

// SetID sets the "id" field.
func (drc *DNSRecordCreate) SetID(u uuid.UUID) *DNSRecordCreate {
	drc.mutation.SetID(u)
	return drc
}

// SetNillableID sets the "id" field if the given value is not nil.
func (drc *DNSRecordCreate) SetNillableID(u *uuid.UUID) *DNSRecordCreate {
	if u != nil {
		drc.SetID(*u)
	}
	return drc
}

// SetEnvironmentID sets the "Environment" edge to the Environment entity by ID.
func (drc *DNSRecordCreate) SetEnvironmentID(id uuid.UUID) *DNSRecordCreate {
	drc.mutation.SetEnvironmentID(id)
	return drc
}

// SetNillableEnvironmentID sets the "Environment" edge to the Environment entity by ID if the given value is not nil.
func (drc *DNSRecordCreate) SetNillableEnvironmentID(id *uuid.UUID) *DNSRecordCreate {
	if id != nil {
		drc = drc.SetEnvironmentID(*id)
	}
	return drc
}

// SetEnvironment sets the "Environment" edge to the Environment entity.
func (drc *DNSRecordCreate) SetEnvironment(e *Environment) *DNSRecordCreate {
	return drc.SetEnvironmentID(e.ID)
}

// Mutation returns the DNSRecordMutation object of the builder.
func (drc *DNSRecordCreate) Mutation() *DNSRecordMutation {
	return drc.mutation
}

// Save creates the DNSRecord in the database.
func (drc *DNSRecordCreate) Save(ctx context.Context) (*DNSRecord, error) {
	drc.defaults()
	return withHooks(ctx, drc.sqlSave, drc.mutation, drc.hooks)
}

// SaveX calls Save and panics if Save returns an error.
func (drc *DNSRecordCreate) SaveX(ctx context.Context) *DNSRecord {
	v, err := drc.Save(ctx)
	if err != nil {
		panic(err)
	}
	return v
}

// Exec executes the query.
func (drc *DNSRecordCreate) Exec(ctx context.Context) error {
	_, err := drc.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (drc *DNSRecordCreate) ExecX(ctx context.Context) {
	if err := drc.Exec(ctx); err != nil {
		panic(err)
	}
}

// defaults sets the default values of the builder before save.
func (drc *DNSRecordCreate) defaults() {
	if _, ok := drc.mutation.ID(); !ok {
		v := dnsrecord.DefaultID()
		drc.mutation.SetID(v)
	}
}

// check runs all checks and user-defined validators on the builder.
func (drc *DNSRecordCreate) check() error {
	if _, ok := drc.mutation.HCLID(); !ok {
		return &ValidationError{Name: "hcl_id", err: errors.New(`ent: missing required field "DNSRecord.hcl_id"`)}
	}
	if _, ok := drc.mutation.Name(); !ok {
		return &ValidationError{Name: "name", err: errors.New(`ent: missing required field "DNSRecord.name"`)}
	}
	if _, ok := drc.mutation.Values(); !ok {
		return &ValidationError{Name: "values", err: errors.New(`ent: missing required field "DNSRecord.values"`)}
	}
	if _, ok := drc.mutation.GetType(); !ok {
		return &ValidationError{Name: "type", err: errors.New(`ent: missing required field "DNSRecord.type"`)}
	}
	if _, ok := drc.mutation.Zone(); !ok {
		return &ValidationError{Name: "zone", err: errors.New(`ent: missing required field "DNSRecord.zone"`)}
	}
	if _, ok := drc.mutation.Vars(); !ok {
		return &ValidationError{Name: "vars", err: errors.New(`ent: missing required field "DNSRecord.vars"`)}
	}
	if _, ok := drc.mutation.Disabled(); !ok {
		return &ValidationError{Name: "disabled", err: errors.New(`ent: missing required field "DNSRecord.disabled"`)}
	}
	if _, ok := drc.mutation.Tags(); !ok {
		return &ValidationError{Name: "tags", err: errors.New(`ent: missing required field "DNSRecord.tags"`)}
	}
	return nil
}

func (drc *DNSRecordCreate) sqlSave(ctx context.Context) (*DNSRecord, error) {
	if err := drc.check(); err != nil {
		return nil, err
	}
	_node, _spec := drc.createSpec()
	if err := sqlgraph.CreateNode(ctx, drc.driver, _spec); err != nil {
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
	drc.mutation.id = &_node.ID
	drc.mutation.done = true
	return _node, nil
}

func (drc *DNSRecordCreate) createSpec() (*DNSRecord, *sqlgraph.CreateSpec) {
	var (
		_node = &DNSRecord{config: drc.config}
		_spec = sqlgraph.NewCreateSpec(dnsrecord.Table, sqlgraph.NewFieldSpec(dnsrecord.FieldID, field.TypeUUID))
	)
	if id, ok := drc.mutation.ID(); ok {
		_node.ID = id
		_spec.ID.Value = &id
	}
	if value, ok := drc.mutation.HCLID(); ok {
		_spec.SetField(dnsrecord.FieldHCLID, field.TypeString, value)
		_node.HCLID = value
	}
	if value, ok := drc.mutation.Name(); ok {
		_spec.SetField(dnsrecord.FieldName, field.TypeString, value)
		_node.Name = value
	}
	if value, ok := drc.mutation.Values(); ok {
		_spec.SetField(dnsrecord.FieldValues, field.TypeJSON, value)
		_node.Values = value
	}
	if value, ok := drc.mutation.GetType(); ok {
		_spec.SetField(dnsrecord.FieldType, field.TypeString, value)
		_node.Type = value
	}
	if value, ok := drc.mutation.Zone(); ok {
		_spec.SetField(dnsrecord.FieldZone, field.TypeString, value)
		_node.Zone = value
	}
	if value, ok := drc.mutation.Vars(); ok {
		_spec.SetField(dnsrecord.FieldVars, field.TypeJSON, value)
		_node.Vars = value
	}
	if value, ok := drc.mutation.Disabled(); ok {
		_spec.SetField(dnsrecord.FieldDisabled, field.TypeBool, value)
		_node.Disabled = value
	}
	if value, ok := drc.mutation.Tags(); ok {
		_spec.SetField(dnsrecord.FieldTags, field.TypeJSON, value)
		_node.Tags = value
	}
	if nodes := drc.mutation.EnvironmentIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: true,
			Table:   dnsrecord.EnvironmentTable,
			Columns: []string{dnsrecord.EnvironmentColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(environment.FieldID, field.TypeUUID),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_node.environment_dns_records = &nodes[0]
		_spec.Edges = append(_spec.Edges, edge)
	}
	return _node, _spec
}

// DNSRecordCreateBulk is the builder for creating many DNSRecord entities in bulk.
type DNSRecordCreateBulk struct {
	config
	err      error
	builders []*DNSRecordCreate
}

// Save creates the DNSRecord entities in the database.
func (drcb *DNSRecordCreateBulk) Save(ctx context.Context) ([]*DNSRecord, error) {
	if drcb.err != nil {
		return nil, drcb.err
	}
	specs := make([]*sqlgraph.CreateSpec, len(drcb.builders))
	nodes := make([]*DNSRecord, len(drcb.builders))
	mutators := make([]Mutator, len(drcb.builders))
	for i := range drcb.builders {
		func(i int, root context.Context) {
			builder := drcb.builders[i]
			builder.defaults()
			var mut Mutator = MutateFunc(func(ctx context.Context, m Mutation) (Value, error) {
				mutation, ok := m.(*DNSRecordMutation)
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
					_, err = mutators[i+1].Mutate(root, drcb.builders[i+1].mutation)
				} else {
					spec := &sqlgraph.BatchCreateSpec{Nodes: specs}
					// Invoke the actual operation on the latest mutation in the chain.
					if err = sqlgraph.BatchCreate(ctx, drcb.driver, spec); err != nil {
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
		if _, err := mutators[0].Mutate(ctx, drcb.builders[0].mutation); err != nil {
			return nil, err
		}
	}
	return nodes, nil
}

// SaveX is like Save, but panics if an error occurs.
func (drcb *DNSRecordCreateBulk) SaveX(ctx context.Context) []*DNSRecord {
	v, err := drcb.Save(ctx)
	if err != nil {
		panic(err)
	}
	return v
}

// Exec executes the query.
func (drcb *DNSRecordCreateBulk) Exec(ctx context.Context) error {
	_, err := drcb.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (drcb *DNSRecordCreateBulk) ExecX(ctx context.Context) {
	if err := drcb.Exec(ctx); err != nil {
		panic(err)
	}
}
