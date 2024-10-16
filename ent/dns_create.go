// Code generated by ent, DO NOT EDIT.

package ent

import (
	"context"
	"errors"
	"fmt"

	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/schema/field"
	"github.com/gen0cide/laforge/ent/competition"
	"github.com/gen0cide/laforge/ent/dns"
	"github.com/gen0cide/laforge/ent/environment"
	"github.com/google/uuid"
)

// DNSCreate is the builder for creating a DNS entity.
type DNSCreate struct {
	config
	mutation *DNSMutation
	hooks    []Hook
}

// SetHCLID sets the "hcl_id" field.
func (dc *DNSCreate) SetHCLID(s string) *DNSCreate {
	dc.mutation.SetHCLID(s)
	return dc
}

// SetType sets the "type" field.
func (dc *DNSCreate) SetType(s string) *DNSCreate {
	dc.mutation.SetType(s)
	return dc
}

// SetRootDomain sets the "root_domain" field.
func (dc *DNSCreate) SetRootDomain(s string) *DNSCreate {
	dc.mutation.SetRootDomain(s)
	return dc
}

// SetDNSServers sets the "dns_servers" field.
func (dc *DNSCreate) SetDNSServers(s []string) *DNSCreate {
	dc.mutation.SetDNSServers(s)
	return dc
}

// SetNtpServers sets the "ntp_servers" field.
func (dc *DNSCreate) SetNtpServers(s []string) *DNSCreate {
	dc.mutation.SetNtpServers(s)
	return dc
}

// SetConfig sets the "config" field.
func (dc *DNSCreate) SetConfig(m map[string]string) *DNSCreate {
	dc.mutation.SetConfig(m)
	return dc
}

// SetID sets the "id" field.
func (dc *DNSCreate) SetID(u uuid.UUID) *DNSCreate {
	dc.mutation.SetID(u)
	return dc
}

// SetNillableID sets the "id" field if the given value is not nil.
func (dc *DNSCreate) SetNillableID(u *uuid.UUID) *DNSCreate {
	if u != nil {
		dc.SetID(*u)
	}
	return dc
}

// AddEnvironmentIDs adds the "Environments" edge to the Environment entity by IDs.
func (dc *DNSCreate) AddEnvironmentIDs(ids ...uuid.UUID) *DNSCreate {
	dc.mutation.AddEnvironmentIDs(ids...)
	return dc
}

// AddEnvironments adds the "Environments" edges to the Environment entity.
func (dc *DNSCreate) AddEnvironments(e ...*Environment) *DNSCreate {
	ids := make([]uuid.UUID, len(e))
	for i := range e {
		ids[i] = e[i].ID
	}
	return dc.AddEnvironmentIDs(ids...)
}

// AddCompetitionIDs adds the "Competitions" edge to the Competition entity by IDs.
func (dc *DNSCreate) AddCompetitionIDs(ids ...uuid.UUID) *DNSCreate {
	dc.mutation.AddCompetitionIDs(ids...)
	return dc
}

// AddCompetitions adds the "Competitions" edges to the Competition entity.
func (dc *DNSCreate) AddCompetitions(c ...*Competition) *DNSCreate {
	ids := make([]uuid.UUID, len(c))
	for i := range c {
		ids[i] = c[i].ID
	}
	return dc.AddCompetitionIDs(ids...)
}

// Mutation returns the DNSMutation object of the builder.
func (dc *DNSCreate) Mutation() *DNSMutation {
	return dc.mutation
}

// Save creates the DNS in the database.
func (dc *DNSCreate) Save(ctx context.Context) (*DNS, error) {
	dc.defaults()
	return withHooks(ctx, dc.sqlSave, dc.mutation, dc.hooks)
}

// SaveX calls Save and panics if Save returns an error.
func (dc *DNSCreate) SaveX(ctx context.Context) *DNS {
	v, err := dc.Save(ctx)
	if err != nil {
		panic(err)
	}
	return v
}

// Exec executes the query.
func (dc *DNSCreate) Exec(ctx context.Context) error {
	_, err := dc.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (dc *DNSCreate) ExecX(ctx context.Context) {
	if err := dc.Exec(ctx); err != nil {
		panic(err)
	}
}

// defaults sets the default values of the builder before save.
func (dc *DNSCreate) defaults() {
	if _, ok := dc.mutation.ID(); !ok {
		v := dns.DefaultID()
		dc.mutation.SetID(v)
	}
}

// check runs all checks and user-defined validators on the builder.
func (dc *DNSCreate) check() error {
	if _, ok := dc.mutation.HCLID(); !ok {
		return &ValidationError{Name: "hcl_id", err: errors.New(`ent: missing required field "DNS.hcl_id"`)}
	}
	if _, ok := dc.mutation.GetType(); !ok {
		return &ValidationError{Name: "type", err: errors.New(`ent: missing required field "DNS.type"`)}
	}
	if _, ok := dc.mutation.RootDomain(); !ok {
		return &ValidationError{Name: "root_domain", err: errors.New(`ent: missing required field "DNS.root_domain"`)}
	}
	if _, ok := dc.mutation.DNSServers(); !ok {
		return &ValidationError{Name: "dns_servers", err: errors.New(`ent: missing required field "DNS.dns_servers"`)}
	}
	if _, ok := dc.mutation.NtpServers(); !ok {
		return &ValidationError{Name: "ntp_servers", err: errors.New(`ent: missing required field "DNS.ntp_servers"`)}
	}
	if _, ok := dc.mutation.Config(); !ok {
		return &ValidationError{Name: "config", err: errors.New(`ent: missing required field "DNS.config"`)}
	}
	return nil
}

func (dc *DNSCreate) sqlSave(ctx context.Context) (*DNS, error) {
	if err := dc.check(); err != nil {
		return nil, err
	}
	_node, _spec := dc.createSpec()
	if err := sqlgraph.CreateNode(ctx, dc.driver, _spec); err != nil {
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
	dc.mutation.id = &_node.ID
	dc.mutation.done = true
	return _node, nil
}

func (dc *DNSCreate) createSpec() (*DNS, *sqlgraph.CreateSpec) {
	var (
		_node = &DNS{config: dc.config}
		_spec = sqlgraph.NewCreateSpec(dns.Table, sqlgraph.NewFieldSpec(dns.FieldID, field.TypeUUID))
	)
	if id, ok := dc.mutation.ID(); ok {
		_node.ID = id
		_spec.ID.Value = &id
	}
	if value, ok := dc.mutation.HCLID(); ok {
		_spec.SetField(dns.FieldHCLID, field.TypeString, value)
		_node.HCLID = value
	}
	if value, ok := dc.mutation.GetType(); ok {
		_spec.SetField(dns.FieldType, field.TypeString, value)
		_node.Type = value
	}
	if value, ok := dc.mutation.RootDomain(); ok {
		_spec.SetField(dns.FieldRootDomain, field.TypeString, value)
		_node.RootDomain = value
	}
	if value, ok := dc.mutation.DNSServers(); ok {
		_spec.SetField(dns.FieldDNSServers, field.TypeJSON, value)
		_node.DNSServers = value
	}
	if value, ok := dc.mutation.NtpServers(); ok {
		_spec.SetField(dns.FieldNtpServers, field.TypeJSON, value)
		_node.NtpServers = value
	}
	if value, ok := dc.mutation.Config(); ok {
		_spec.SetField(dns.FieldConfig, field.TypeJSON, value)
		_node.Config = value
	}
	if nodes := dc.mutation.EnvironmentsIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2M,
			Inverse: true,
			Table:   dns.EnvironmentsTable,
			Columns: dns.EnvironmentsPrimaryKey,
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(environment.FieldID, field.TypeUUID),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges = append(_spec.Edges, edge)
	}
	if nodes := dc.mutation.CompetitionsIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2M,
			Inverse: true,
			Table:   dns.CompetitionsTable,
			Columns: dns.CompetitionsPrimaryKey,
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(competition.FieldID, field.TypeUUID),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges = append(_spec.Edges, edge)
	}
	return _node, _spec
}

// DNSCreateBulk is the builder for creating many DNS entities in bulk.
type DNSCreateBulk struct {
	config
	err      error
	builders []*DNSCreate
}

// Save creates the DNS entities in the database.
func (dcb *DNSCreateBulk) Save(ctx context.Context) ([]*DNS, error) {
	if dcb.err != nil {
		return nil, dcb.err
	}
	specs := make([]*sqlgraph.CreateSpec, len(dcb.builders))
	nodes := make([]*DNS, len(dcb.builders))
	mutators := make([]Mutator, len(dcb.builders))
	for i := range dcb.builders {
		func(i int, root context.Context) {
			builder := dcb.builders[i]
			builder.defaults()
			var mut Mutator = MutateFunc(func(ctx context.Context, m Mutation) (Value, error) {
				mutation, ok := m.(*DNSMutation)
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
					_, err = mutators[i+1].Mutate(root, dcb.builders[i+1].mutation)
				} else {
					spec := &sqlgraph.BatchCreateSpec{Nodes: specs}
					// Invoke the actual operation on the latest mutation in the chain.
					if err = sqlgraph.BatchCreate(ctx, dcb.driver, spec); err != nil {
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
		if _, err := mutators[0].Mutate(ctx, dcb.builders[0].mutation); err != nil {
			return nil, err
		}
	}
	return nodes, nil
}

// SaveX is like Save, but panics if an error occurs.
func (dcb *DNSCreateBulk) SaveX(ctx context.Context) []*DNS {
	v, err := dcb.Save(ctx)
	if err != nil {
		panic(err)
	}
	return v
}

// Exec executes the query.
func (dcb *DNSCreateBulk) Exec(ctx context.Context) error {
	_, err := dcb.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (dcb *DNSCreateBulk) ExecX(ctx context.Context) {
	if err := dcb.Exec(ctx); err != nil {
		panic(err)
	}
}
