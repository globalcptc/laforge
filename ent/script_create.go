// Code generated by ent, DO NOT EDIT.

package ent

import (
	"context"
	"errors"
	"fmt"

	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/schema/field"
	"github.com/gen0cide/laforge/ent/environment"
	"github.com/gen0cide/laforge/ent/finding"
	"github.com/gen0cide/laforge/ent/script"
	"github.com/gen0cide/laforge/ent/user"
	"github.com/google/uuid"
)

// ScriptCreate is the builder for creating a Script entity.
type ScriptCreate struct {
	config
	mutation *ScriptMutation
	hooks    []Hook
}

// SetHCLID sets the "hcl_id" field.
func (sc *ScriptCreate) SetHCLID(s string) *ScriptCreate {
	sc.mutation.SetHCLID(s)
	return sc
}

// SetName sets the "name" field.
func (sc *ScriptCreate) SetName(s string) *ScriptCreate {
	sc.mutation.SetName(s)
	return sc
}

// SetLanguage sets the "language" field.
func (sc *ScriptCreate) SetLanguage(s string) *ScriptCreate {
	sc.mutation.SetLanguage(s)
	return sc
}

// SetDescription sets the "description" field.
func (sc *ScriptCreate) SetDescription(s string) *ScriptCreate {
	sc.mutation.SetDescription(s)
	return sc
}

// SetSource sets the "source" field.
func (sc *ScriptCreate) SetSource(s string) *ScriptCreate {
	sc.mutation.SetSource(s)
	return sc
}

// SetSourceType sets the "source_type" field.
func (sc *ScriptCreate) SetSourceType(s string) *ScriptCreate {
	sc.mutation.SetSourceType(s)
	return sc
}

// SetCooldown sets the "cooldown" field.
func (sc *ScriptCreate) SetCooldown(i int) *ScriptCreate {
	sc.mutation.SetCooldown(i)
	return sc
}

// SetTimeout sets the "timeout" field.
func (sc *ScriptCreate) SetTimeout(i int) *ScriptCreate {
	sc.mutation.SetTimeout(i)
	return sc
}

// SetIgnoreErrors sets the "ignore_errors" field.
func (sc *ScriptCreate) SetIgnoreErrors(b bool) *ScriptCreate {
	sc.mutation.SetIgnoreErrors(b)
	return sc
}

// SetArgs sets the "args" field.
func (sc *ScriptCreate) SetArgs(s []string) *ScriptCreate {
	sc.mutation.SetArgs(s)
	return sc
}

// SetDisabled sets the "disabled" field.
func (sc *ScriptCreate) SetDisabled(b bool) *ScriptCreate {
	sc.mutation.SetDisabled(b)
	return sc
}

// SetVars sets the "vars" field.
func (sc *ScriptCreate) SetVars(m map[string]string) *ScriptCreate {
	sc.mutation.SetVars(m)
	return sc
}

// SetAbsPath sets the "abs_path" field.
func (sc *ScriptCreate) SetAbsPath(s string) *ScriptCreate {
	sc.mutation.SetAbsPath(s)
	return sc
}

// SetTags sets the "tags" field.
func (sc *ScriptCreate) SetTags(m map[string]string) *ScriptCreate {
	sc.mutation.SetTags(m)
	return sc
}

// SetID sets the "id" field.
func (sc *ScriptCreate) SetID(u uuid.UUID) *ScriptCreate {
	sc.mutation.SetID(u)
	return sc
}

// SetNillableID sets the "id" field if the given value is not nil.
func (sc *ScriptCreate) SetNillableID(u *uuid.UUID) *ScriptCreate {
	if u != nil {
		sc.SetID(*u)
	}
	return sc
}

// AddUserIDs adds the "Users" edge to the User entity by IDs.
func (sc *ScriptCreate) AddUserIDs(ids ...uuid.UUID) *ScriptCreate {
	sc.mutation.AddUserIDs(ids...)
	return sc
}

// AddUsers adds the "Users" edges to the User entity.
func (sc *ScriptCreate) AddUsers(u ...*User) *ScriptCreate {
	ids := make([]uuid.UUID, len(u))
	for i := range u {
		ids[i] = u[i].ID
	}
	return sc.AddUserIDs(ids...)
}

// AddFindingIDs adds the "Findings" edge to the Finding entity by IDs.
func (sc *ScriptCreate) AddFindingIDs(ids ...uuid.UUID) *ScriptCreate {
	sc.mutation.AddFindingIDs(ids...)
	return sc
}

// AddFindings adds the "Findings" edges to the Finding entity.
func (sc *ScriptCreate) AddFindings(f ...*Finding) *ScriptCreate {
	ids := make([]uuid.UUID, len(f))
	for i := range f {
		ids[i] = f[i].ID
	}
	return sc.AddFindingIDs(ids...)
}

// SetEnvironmentID sets the "Environment" edge to the Environment entity by ID.
func (sc *ScriptCreate) SetEnvironmentID(id uuid.UUID) *ScriptCreate {
	sc.mutation.SetEnvironmentID(id)
	return sc
}

// SetNillableEnvironmentID sets the "Environment" edge to the Environment entity by ID if the given value is not nil.
func (sc *ScriptCreate) SetNillableEnvironmentID(id *uuid.UUID) *ScriptCreate {
	if id != nil {
		sc = sc.SetEnvironmentID(*id)
	}
	return sc
}

// SetEnvironment sets the "Environment" edge to the Environment entity.
func (sc *ScriptCreate) SetEnvironment(e *Environment) *ScriptCreate {
	return sc.SetEnvironmentID(e.ID)
}

// Mutation returns the ScriptMutation object of the builder.
func (sc *ScriptCreate) Mutation() *ScriptMutation {
	return sc.mutation
}

// Save creates the Script in the database.
func (sc *ScriptCreate) Save(ctx context.Context) (*Script, error) {
	sc.defaults()
	return withHooks(ctx, sc.sqlSave, sc.mutation, sc.hooks)
}

// SaveX calls Save and panics if Save returns an error.
func (sc *ScriptCreate) SaveX(ctx context.Context) *Script {
	v, err := sc.Save(ctx)
	if err != nil {
		panic(err)
	}
	return v
}

// Exec executes the query.
func (sc *ScriptCreate) Exec(ctx context.Context) error {
	_, err := sc.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (sc *ScriptCreate) ExecX(ctx context.Context) {
	if err := sc.Exec(ctx); err != nil {
		panic(err)
	}
}

// defaults sets the default values of the builder before save.
func (sc *ScriptCreate) defaults() {
	if _, ok := sc.mutation.ID(); !ok {
		v := script.DefaultID()
		sc.mutation.SetID(v)
	}
}

// check runs all checks and user-defined validators on the builder.
func (sc *ScriptCreate) check() error {
	if _, ok := sc.mutation.HCLID(); !ok {
		return &ValidationError{Name: "hcl_id", err: errors.New(`ent: missing required field "Script.hcl_id"`)}
	}
	if _, ok := sc.mutation.Name(); !ok {
		return &ValidationError{Name: "name", err: errors.New(`ent: missing required field "Script.name"`)}
	}
	if _, ok := sc.mutation.Language(); !ok {
		return &ValidationError{Name: "language", err: errors.New(`ent: missing required field "Script.language"`)}
	}
	if _, ok := sc.mutation.Description(); !ok {
		return &ValidationError{Name: "description", err: errors.New(`ent: missing required field "Script.description"`)}
	}
	if _, ok := sc.mutation.Source(); !ok {
		return &ValidationError{Name: "source", err: errors.New(`ent: missing required field "Script.source"`)}
	}
	if _, ok := sc.mutation.SourceType(); !ok {
		return &ValidationError{Name: "source_type", err: errors.New(`ent: missing required field "Script.source_type"`)}
	}
	if _, ok := sc.mutation.Cooldown(); !ok {
		return &ValidationError{Name: "cooldown", err: errors.New(`ent: missing required field "Script.cooldown"`)}
	}
	if _, ok := sc.mutation.Timeout(); !ok {
		return &ValidationError{Name: "timeout", err: errors.New(`ent: missing required field "Script.timeout"`)}
	}
	if _, ok := sc.mutation.IgnoreErrors(); !ok {
		return &ValidationError{Name: "ignore_errors", err: errors.New(`ent: missing required field "Script.ignore_errors"`)}
	}
	if _, ok := sc.mutation.Args(); !ok {
		return &ValidationError{Name: "args", err: errors.New(`ent: missing required field "Script.args"`)}
	}
	if _, ok := sc.mutation.Disabled(); !ok {
		return &ValidationError{Name: "disabled", err: errors.New(`ent: missing required field "Script.disabled"`)}
	}
	if _, ok := sc.mutation.Vars(); !ok {
		return &ValidationError{Name: "vars", err: errors.New(`ent: missing required field "Script.vars"`)}
	}
	if _, ok := sc.mutation.AbsPath(); !ok {
		return &ValidationError{Name: "abs_path", err: errors.New(`ent: missing required field "Script.abs_path"`)}
	}
	if _, ok := sc.mutation.Tags(); !ok {
		return &ValidationError{Name: "tags", err: errors.New(`ent: missing required field "Script.tags"`)}
	}
	return nil
}

func (sc *ScriptCreate) sqlSave(ctx context.Context) (*Script, error) {
	if err := sc.check(); err != nil {
		return nil, err
	}
	_node, _spec := sc.createSpec()
	if err := sqlgraph.CreateNode(ctx, sc.driver, _spec); err != nil {
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
	sc.mutation.id = &_node.ID
	sc.mutation.done = true
	return _node, nil
}

func (sc *ScriptCreate) createSpec() (*Script, *sqlgraph.CreateSpec) {
	var (
		_node = &Script{config: sc.config}
		_spec = sqlgraph.NewCreateSpec(script.Table, sqlgraph.NewFieldSpec(script.FieldID, field.TypeUUID))
	)
	if id, ok := sc.mutation.ID(); ok {
		_node.ID = id
		_spec.ID.Value = &id
	}
	if value, ok := sc.mutation.HCLID(); ok {
		_spec.SetField(script.FieldHCLID, field.TypeString, value)
		_node.HCLID = value
	}
	if value, ok := sc.mutation.Name(); ok {
		_spec.SetField(script.FieldName, field.TypeString, value)
		_node.Name = value
	}
	if value, ok := sc.mutation.Language(); ok {
		_spec.SetField(script.FieldLanguage, field.TypeString, value)
		_node.Language = value
	}
	if value, ok := sc.mutation.Description(); ok {
		_spec.SetField(script.FieldDescription, field.TypeString, value)
		_node.Description = value
	}
	if value, ok := sc.mutation.Source(); ok {
		_spec.SetField(script.FieldSource, field.TypeString, value)
		_node.Source = value
	}
	if value, ok := sc.mutation.SourceType(); ok {
		_spec.SetField(script.FieldSourceType, field.TypeString, value)
		_node.SourceType = value
	}
	if value, ok := sc.mutation.Cooldown(); ok {
		_spec.SetField(script.FieldCooldown, field.TypeInt, value)
		_node.Cooldown = value
	}
	if value, ok := sc.mutation.Timeout(); ok {
		_spec.SetField(script.FieldTimeout, field.TypeInt, value)
		_node.Timeout = value
	}
	if value, ok := sc.mutation.IgnoreErrors(); ok {
		_spec.SetField(script.FieldIgnoreErrors, field.TypeBool, value)
		_node.IgnoreErrors = value
	}
	if value, ok := sc.mutation.Args(); ok {
		_spec.SetField(script.FieldArgs, field.TypeJSON, value)
		_node.Args = value
	}
	if value, ok := sc.mutation.Disabled(); ok {
		_spec.SetField(script.FieldDisabled, field.TypeBool, value)
		_node.Disabled = value
	}
	if value, ok := sc.mutation.Vars(); ok {
		_spec.SetField(script.FieldVars, field.TypeJSON, value)
		_node.Vars = value
	}
	if value, ok := sc.mutation.AbsPath(); ok {
		_spec.SetField(script.FieldAbsPath, field.TypeString, value)
		_node.AbsPath = value
	}
	if value, ok := sc.mutation.Tags(); ok {
		_spec.SetField(script.FieldTags, field.TypeJSON, value)
		_node.Tags = value
	}
	if nodes := sc.mutation.UsersIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: false,
			Table:   script.UsersTable,
			Columns: []string{script.UsersColumn},
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
	if nodes := sc.mutation.FindingsIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: false,
			Table:   script.FindingsTable,
			Columns: []string{script.FindingsColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(finding.FieldID, field.TypeUUID),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges = append(_spec.Edges, edge)
	}
	if nodes := sc.mutation.EnvironmentIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: true,
			Table:   script.EnvironmentTable,
			Columns: []string{script.EnvironmentColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(environment.FieldID, field.TypeUUID),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_node.environment_scripts = &nodes[0]
		_spec.Edges = append(_spec.Edges, edge)
	}
	return _node, _spec
}

// ScriptCreateBulk is the builder for creating many Script entities in bulk.
type ScriptCreateBulk struct {
	config
	err      error
	builders []*ScriptCreate
}

// Save creates the Script entities in the database.
func (scb *ScriptCreateBulk) Save(ctx context.Context) ([]*Script, error) {
	if scb.err != nil {
		return nil, scb.err
	}
	specs := make([]*sqlgraph.CreateSpec, len(scb.builders))
	nodes := make([]*Script, len(scb.builders))
	mutators := make([]Mutator, len(scb.builders))
	for i := range scb.builders {
		func(i int, root context.Context) {
			builder := scb.builders[i]
			builder.defaults()
			var mut Mutator = MutateFunc(func(ctx context.Context, m Mutation) (Value, error) {
				mutation, ok := m.(*ScriptMutation)
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
					_, err = mutators[i+1].Mutate(root, scb.builders[i+1].mutation)
				} else {
					spec := &sqlgraph.BatchCreateSpec{Nodes: specs}
					// Invoke the actual operation on the latest mutation in the chain.
					if err = sqlgraph.BatchCreate(ctx, scb.driver, spec); err != nil {
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
		if _, err := mutators[0].Mutate(ctx, scb.builders[0].mutation); err != nil {
			return nil, err
		}
	}
	return nodes, nil
}

// SaveX is like Save, but panics if an error occurs.
func (scb *ScriptCreateBulk) SaveX(ctx context.Context) []*Script {
	v, err := scb.Save(ctx)
	if err != nil {
		panic(err)
	}
	return v
}

// Exec executes the query.
func (scb *ScriptCreateBulk) Exec(ctx context.Context) error {
	_, err := scb.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (scb *ScriptCreateBulk) ExecX(ctx context.Context) {
	if err := scb.Exec(ctx); err != nil {
		panic(err)
	}
}
