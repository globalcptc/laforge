// Code generated by ent, DO NOT EDIT.

package ent

import (
	"context"
	"errors"
	"fmt"

	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/schema/field"
	"github.com/gen0cide/laforge/ent/ansible"
	"github.com/gen0cide/laforge/ent/environment"
	"github.com/gen0cide/laforge/ent/user"
	"github.com/google/uuid"
)

// AnsibleCreate is the builder for creating a Ansible entity.
type AnsibleCreate struct {
	config
	mutation *AnsibleMutation
	hooks    []Hook
}

// SetName sets the "name" field.
func (ac *AnsibleCreate) SetName(s string) *AnsibleCreate {
	ac.mutation.SetName(s)
	return ac
}

// SetHCLID sets the "hcl_id" field.
func (ac *AnsibleCreate) SetHCLID(s string) *AnsibleCreate {
	ac.mutation.SetHCLID(s)
	return ac
}

// SetDescription sets the "description" field.
func (ac *AnsibleCreate) SetDescription(s string) *AnsibleCreate {
	ac.mutation.SetDescription(s)
	return ac
}

// SetSource sets the "source" field.
func (ac *AnsibleCreate) SetSource(s string) *AnsibleCreate {
	ac.mutation.SetSource(s)
	return ac
}

// SetPlaybookName sets the "playbook_name" field.
func (ac *AnsibleCreate) SetPlaybookName(s string) *AnsibleCreate {
	ac.mutation.SetPlaybookName(s)
	return ac
}

// SetMethod sets the "method" field.
func (ac *AnsibleCreate) SetMethod(a ansible.Method) *AnsibleCreate {
	ac.mutation.SetMethod(a)
	return ac
}

// SetInventory sets the "inventory" field.
func (ac *AnsibleCreate) SetInventory(s string) *AnsibleCreate {
	ac.mutation.SetInventory(s)
	return ac
}

// SetAbsPath sets the "abs_path" field.
func (ac *AnsibleCreate) SetAbsPath(s string) *AnsibleCreate {
	ac.mutation.SetAbsPath(s)
	return ac
}

// SetTags sets the "tags" field.
func (ac *AnsibleCreate) SetTags(m map[string]string) *AnsibleCreate {
	ac.mutation.SetTags(m)
	return ac
}

// SetID sets the "id" field.
func (ac *AnsibleCreate) SetID(u uuid.UUID) *AnsibleCreate {
	ac.mutation.SetID(u)
	return ac
}

// SetNillableID sets the "id" field if the given value is not nil.
func (ac *AnsibleCreate) SetNillableID(u *uuid.UUID) *AnsibleCreate {
	if u != nil {
		ac.SetID(*u)
	}
	return ac
}

// AddUserIDs adds the "Users" edge to the User entity by IDs.
func (ac *AnsibleCreate) AddUserIDs(ids ...uuid.UUID) *AnsibleCreate {
	ac.mutation.AddUserIDs(ids...)
	return ac
}

// AddUsers adds the "Users" edges to the User entity.
func (ac *AnsibleCreate) AddUsers(u ...*User) *AnsibleCreate {
	ids := make([]uuid.UUID, len(u))
	for i := range u {
		ids[i] = u[i].ID
	}
	return ac.AddUserIDs(ids...)
}

// SetEnvironmentID sets the "Environment" edge to the Environment entity by ID.
func (ac *AnsibleCreate) SetEnvironmentID(id uuid.UUID) *AnsibleCreate {
	ac.mutation.SetEnvironmentID(id)
	return ac
}

// SetNillableEnvironmentID sets the "Environment" edge to the Environment entity by ID if the given value is not nil.
func (ac *AnsibleCreate) SetNillableEnvironmentID(id *uuid.UUID) *AnsibleCreate {
	if id != nil {
		ac = ac.SetEnvironmentID(*id)
	}
	return ac
}

// SetEnvironment sets the "Environment" edge to the Environment entity.
func (ac *AnsibleCreate) SetEnvironment(e *Environment) *AnsibleCreate {
	return ac.SetEnvironmentID(e.ID)
}

// Mutation returns the AnsibleMutation object of the builder.
func (ac *AnsibleCreate) Mutation() *AnsibleMutation {
	return ac.mutation
}

// Save creates the Ansible in the database.
func (ac *AnsibleCreate) Save(ctx context.Context) (*Ansible, error) {
	ac.defaults()
	return withHooks(ctx, ac.sqlSave, ac.mutation, ac.hooks)
}

// SaveX calls Save and panics if Save returns an error.
func (ac *AnsibleCreate) SaveX(ctx context.Context) *Ansible {
	v, err := ac.Save(ctx)
	if err != nil {
		panic(err)
	}
	return v
}

// Exec executes the query.
func (ac *AnsibleCreate) Exec(ctx context.Context) error {
	_, err := ac.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (ac *AnsibleCreate) ExecX(ctx context.Context) {
	if err := ac.Exec(ctx); err != nil {
		panic(err)
	}
}

// defaults sets the default values of the builder before save.
func (ac *AnsibleCreate) defaults() {
	if _, ok := ac.mutation.ID(); !ok {
		v := ansible.DefaultID()
		ac.mutation.SetID(v)
	}
}

// check runs all checks and user-defined validators on the builder.
func (ac *AnsibleCreate) check() error {
	if _, ok := ac.mutation.Name(); !ok {
		return &ValidationError{Name: "name", err: errors.New(`ent: missing required field "Ansible.name"`)}
	}
	if _, ok := ac.mutation.HCLID(); !ok {
		return &ValidationError{Name: "hcl_id", err: errors.New(`ent: missing required field "Ansible.hcl_id"`)}
	}
	if _, ok := ac.mutation.Description(); !ok {
		return &ValidationError{Name: "description", err: errors.New(`ent: missing required field "Ansible.description"`)}
	}
	if _, ok := ac.mutation.Source(); !ok {
		return &ValidationError{Name: "source", err: errors.New(`ent: missing required field "Ansible.source"`)}
	}
	if _, ok := ac.mutation.PlaybookName(); !ok {
		return &ValidationError{Name: "playbook_name", err: errors.New(`ent: missing required field "Ansible.playbook_name"`)}
	}
	if _, ok := ac.mutation.Method(); !ok {
		return &ValidationError{Name: "method", err: errors.New(`ent: missing required field "Ansible.method"`)}
	}
	if v, ok := ac.mutation.Method(); ok {
		if err := ansible.MethodValidator(v); err != nil {
			return &ValidationError{Name: "method", err: fmt.Errorf(`ent: validator failed for field "Ansible.method": %w`, err)}
		}
	}
	if _, ok := ac.mutation.Inventory(); !ok {
		return &ValidationError{Name: "inventory", err: errors.New(`ent: missing required field "Ansible.inventory"`)}
	}
	if _, ok := ac.mutation.AbsPath(); !ok {
		return &ValidationError{Name: "abs_path", err: errors.New(`ent: missing required field "Ansible.abs_path"`)}
	}
	if _, ok := ac.mutation.Tags(); !ok {
		return &ValidationError{Name: "tags", err: errors.New(`ent: missing required field "Ansible.tags"`)}
	}
	return nil
}

func (ac *AnsibleCreate) sqlSave(ctx context.Context) (*Ansible, error) {
	if err := ac.check(); err != nil {
		return nil, err
	}
	_node, _spec := ac.createSpec()
	if err := sqlgraph.CreateNode(ctx, ac.driver, _spec); err != nil {
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
	ac.mutation.id = &_node.ID
	ac.mutation.done = true
	return _node, nil
}

func (ac *AnsibleCreate) createSpec() (*Ansible, *sqlgraph.CreateSpec) {
	var (
		_node = &Ansible{config: ac.config}
		_spec = sqlgraph.NewCreateSpec(ansible.Table, sqlgraph.NewFieldSpec(ansible.FieldID, field.TypeUUID))
	)
	if id, ok := ac.mutation.ID(); ok {
		_node.ID = id
		_spec.ID.Value = &id
	}
	if value, ok := ac.mutation.Name(); ok {
		_spec.SetField(ansible.FieldName, field.TypeString, value)
		_node.Name = value
	}
	if value, ok := ac.mutation.HCLID(); ok {
		_spec.SetField(ansible.FieldHCLID, field.TypeString, value)
		_node.HCLID = value
	}
	if value, ok := ac.mutation.Description(); ok {
		_spec.SetField(ansible.FieldDescription, field.TypeString, value)
		_node.Description = value
	}
	if value, ok := ac.mutation.Source(); ok {
		_spec.SetField(ansible.FieldSource, field.TypeString, value)
		_node.Source = value
	}
	if value, ok := ac.mutation.PlaybookName(); ok {
		_spec.SetField(ansible.FieldPlaybookName, field.TypeString, value)
		_node.PlaybookName = value
	}
	if value, ok := ac.mutation.Method(); ok {
		_spec.SetField(ansible.FieldMethod, field.TypeEnum, value)
		_node.Method = value
	}
	if value, ok := ac.mutation.Inventory(); ok {
		_spec.SetField(ansible.FieldInventory, field.TypeString, value)
		_node.Inventory = value
	}
	if value, ok := ac.mutation.AbsPath(); ok {
		_spec.SetField(ansible.FieldAbsPath, field.TypeString, value)
		_node.AbsPath = value
	}
	if value, ok := ac.mutation.Tags(); ok {
		_spec.SetField(ansible.FieldTags, field.TypeJSON, value)
		_node.Tags = value
	}
	if nodes := ac.mutation.UsersIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: false,
			Table:   ansible.UsersTable,
			Columns: []string{ansible.UsersColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(user.FieldID, field.TypeUUID),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges = append(_spec.Edges, edge)
	}
	if nodes := ac.mutation.EnvironmentIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: true,
			Table:   ansible.EnvironmentTable,
			Columns: []string{ansible.EnvironmentColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(environment.FieldID, field.TypeUUID),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_node.environment_ansibles = &nodes[0]
		_spec.Edges = append(_spec.Edges, edge)
	}
	return _node, _spec
}

// AnsibleCreateBulk is the builder for creating many Ansible entities in bulk.
type AnsibleCreateBulk struct {
	config
	err      error
	builders []*AnsibleCreate
}

// Save creates the Ansible entities in the database.
func (acb *AnsibleCreateBulk) Save(ctx context.Context) ([]*Ansible, error) {
	if acb.err != nil {
		return nil, acb.err
	}
	specs := make([]*sqlgraph.CreateSpec, len(acb.builders))
	nodes := make([]*Ansible, len(acb.builders))
	mutators := make([]Mutator, len(acb.builders))
	for i := range acb.builders {
		func(i int, root context.Context) {
			builder := acb.builders[i]
			builder.defaults()
			var mut Mutator = MutateFunc(func(ctx context.Context, m Mutation) (Value, error) {
				mutation, ok := m.(*AnsibleMutation)
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
					_, err = mutators[i+1].Mutate(root, acb.builders[i+1].mutation)
				} else {
					spec := &sqlgraph.BatchCreateSpec{Nodes: specs}
					// Invoke the actual operation on the latest mutation in the chain.
					if err = sqlgraph.BatchCreate(ctx, acb.driver, spec); err != nil {
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
		if _, err := mutators[0].Mutate(ctx, acb.builders[0].mutation); err != nil {
			return nil, err
		}
	}
	return nodes, nil
}

// SaveX is like Save, but panics if an error occurs.
func (acb *AnsibleCreateBulk) SaveX(ctx context.Context) []*Ansible {
	v, err := acb.Save(ctx)
	if err != nil {
		panic(err)
	}
	return v
}

// Exec executes the query.
func (acb *AnsibleCreateBulk) Exec(ctx context.Context) error {
	_, err := acb.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (acb *AnsibleCreateBulk) ExecX(ctx context.Context) {
	if err := acb.Exec(ctx); err != nil {
		panic(err)
	}
}
