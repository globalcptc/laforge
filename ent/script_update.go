// Code generated by ent, DO NOT EDIT.

package ent

import (
	"context"
	"errors"
	"fmt"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/dialect/sql/sqljson"
	"entgo.io/ent/schema/field"
	"github.com/gen0cide/laforge/ent/environment"
	"github.com/gen0cide/laforge/ent/finding"
	"github.com/gen0cide/laforge/ent/predicate"
	"github.com/gen0cide/laforge/ent/script"
	"github.com/gen0cide/laforge/ent/user"
	"github.com/google/uuid"
)

// ScriptUpdate is the builder for updating Script entities.
type ScriptUpdate struct {
	config
	hooks    []Hook
	mutation *ScriptMutation
}

// Where appends a list predicates to the ScriptUpdate builder.
func (su *ScriptUpdate) Where(ps ...predicate.Script) *ScriptUpdate {
	su.mutation.Where(ps...)
	return su
}

// SetHCLID sets the "hcl_id" field.
func (su *ScriptUpdate) SetHCLID(s string) *ScriptUpdate {
	su.mutation.SetHCLID(s)
	return su
}

// SetNillableHCLID sets the "hcl_id" field if the given value is not nil.
func (su *ScriptUpdate) SetNillableHCLID(s *string) *ScriptUpdate {
	if s != nil {
		su.SetHCLID(*s)
	}
	return su
}

// SetName sets the "name" field.
func (su *ScriptUpdate) SetName(s string) *ScriptUpdate {
	su.mutation.SetName(s)
	return su
}

// SetNillableName sets the "name" field if the given value is not nil.
func (su *ScriptUpdate) SetNillableName(s *string) *ScriptUpdate {
	if s != nil {
		su.SetName(*s)
	}
	return su
}

// SetLanguage sets the "language" field.
func (su *ScriptUpdate) SetLanguage(s string) *ScriptUpdate {
	su.mutation.SetLanguage(s)
	return su
}

// SetNillableLanguage sets the "language" field if the given value is not nil.
func (su *ScriptUpdate) SetNillableLanguage(s *string) *ScriptUpdate {
	if s != nil {
		su.SetLanguage(*s)
	}
	return su
}

// SetDescription sets the "description" field.
func (su *ScriptUpdate) SetDescription(s string) *ScriptUpdate {
	su.mutation.SetDescription(s)
	return su
}

// SetNillableDescription sets the "description" field if the given value is not nil.
func (su *ScriptUpdate) SetNillableDescription(s *string) *ScriptUpdate {
	if s != nil {
		su.SetDescription(*s)
	}
	return su
}

// SetSource sets the "source" field.
func (su *ScriptUpdate) SetSource(s string) *ScriptUpdate {
	su.mutation.SetSource(s)
	return su
}

// SetNillableSource sets the "source" field if the given value is not nil.
func (su *ScriptUpdate) SetNillableSource(s *string) *ScriptUpdate {
	if s != nil {
		su.SetSource(*s)
	}
	return su
}

// SetSourceType sets the "source_type" field.
func (su *ScriptUpdate) SetSourceType(s string) *ScriptUpdate {
	su.mutation.SetSourceType(s)
	return su
}

// SetNillableSourceType sets the "source_type" field if the given value is not nil.
func (su *ScriptUpdate) SetNillableSourceType(s *string) *ScriptUpdate {
	if s != nil {
		su.SetSourceType(*s)
	}
	return su
}

// SetCooldown sets the "cooldown" field.
func (su *ScriptUpdate) SetCooldown(i int) *ScriptUpdate {
	su.mutation.ResetCooldown()
	su.mutation.SetCooldown(i)
	return su
}

// SetNillableCooldown sets the "cooldown" field if the given value is not nil.
func (su *ScriptUpdate) SetNillableCooldown(i *int) *ScriptUpdate {
	if i != nil {
		su.SetCooldown(*i)
	}
	return su
}

// AddCooldown adds i to the "cooldown" field.
func (su *ScriptUpdate) AddCooldown(i int) *ScriptUpdate {
	su.mutation.AddCooldown(i)
	return su
}

// SetTimeout sets the "timeout" field.
func (su *ScriptUpdate) SetTimeout(i int) *ScriptUpdate {
	su.mutation.ResetTimeout()
	su.mutation.SetTimeout(i)
	return su
}

// SetNillableTimeout sets the "timeout" field if the given value is not nil.
func (su *ScriptUpdate) SetNillableTimeout(i *int) *ScriptUpdate {
	if i != nil {
		su.SetTimeout(*i)
	}
	return su
}

// AddTimeout adds i to the "timeout" field.
func (su *ScriptUpdate) AddTimeout(i int) *ScriptUpdate {
	su.mutation.AddTimeout(i)
	return su
}

// SetIgnoreErrors sets the "ignore_errors" field.
func (su *ScriptUpdate) SetIgnoreErrors(b bool) *ScriptUpdate {
	su.mutation.SetIgnoreErrors(b)
	return su
}

// SetNillableIgnoreErrors sets the "ignore_errors" field if the given value is not nil.
func (su *ScriptUpdate) SetNillableIgnoreErrors(b *bool) *ScriptUpdate {
	if b != nil {
		su.SetIgnoreErrors(*b)
	}
	return su
}

// SetArgs sets the "args" field.
func (su *ScriptUpdate) SetArgs(s []string) *ScriptUpdate {
	su.mutation.SetArgs(s)
	return su
}

// AppendArgs appends s to the "args" field.
func (su *ScriptUpdate) AppendArgs(s []string) *ScriptUpdate {
	su.mutation.AppendArgs(s)
	return su
}

// SetDisabled sets the "disabled" field.
func (su *ScriptUpdate) SetDisabled(b bool) *ScriptUpdate {
	su.mutation.SetDisabled(b)
	return su
}

// SetNillableDisabled sets the "disabled" field if the given value is not nil.
func (su *ScriptUpdate) SetNillableDisabled(b *bool) *ScriptUpdate {
	if b != nil {
		su.SetDisabled(*b)
	}
	return su
}

// SetVars sets the "vars" field.
func (su *ScriptUpdate) SetVars(m map[string]string) *ScriptUpdate {
	su.mutation.SetVars(m)
	return su
}

// SetAbsPath sets the "abs_path" field.
func (su *ScriptUpdate) SetAbsPath(s string) *ScriptUpdate {
	su.mutation.SetAbsPath(s)
	return su
}

// SetNillableAbsPath sets the "abs_path" field if the given value is not nil.
func (su *ScriptUpdate) SetNillableAbsPath(s *string) *ScriptUpdate {
	if s != nil {
		su.SetAbsPath(*s)
	}
	return su
}

// SetTags sets the "tags" field.
func (su *ScriptUpdate) SetTags(m map[string]string) *ScriptUpdate {
	su.mutation.SetTags(m)
	return su
}

// AddScriptToUserIDs adds the "ScriptToUser" edge to the User entity by IDs.
func (su *ScriptUpdate) AddScriptToUserIDs(ids ...uuid.UUID) *ScriptUpdate {
	su.mutation.AddScriptToUserIDs(ids...)
	return su
}

// AddScriptToUser adds the "ScriptToUser" edges to the User entity.
func (su *ScriptUpdate) AddScriptToUser(u ...*User) *ScriptUpdate {
	ids := make([]uuid.UUID, len(u))
	for i := range u {
		ids[i] = u[i].ID
	}
	return su.AddScriptToUserIDs(ids...)
}

// AddScriptToFindingIDs adds the "ScriptToFinding" edge to the Finding entity by IDs.
func (su *ScriptUpdate) AddScriptToFindingIDs(ids ...uuid.UUID) *ScriptUpdate {
	su.mutation.AddScriptToFindingIDs(ids...)
	return su
}

// AddScriptToFinding adds the "ScriptToFinding" edges to the Finding entity.
func (su *ScriptUpdate) AddScriptToFinding(f ...*Finding) *ScriptUpdate {
	ids := make([]uuid.UUID, len(f))
	for i := range f {
		ids[i] = f[i].ID
	}
	return su.AddScriptToFindingIDs(ids...)
}

// SetScriptToEnvironmentID sets the "ScriptToEnvironment" edge to the Environment entity by ID.
func (su *ScriptUpdate) SetScriptToEnvironmentID(id uuid.UUID) *ScriptUpdate {
	su.mutation.SetScriptToEnvironmentID(id)
	return su
}

// SetNillableScriptToEnvironmentID sets the "ScriptToEnvironment" edge to the Environment entity by ID if the given value is not nil.
func (su *ScriptUpdate) SetNillableScriptToEnvironmentID(id *uuid.UUID) *ScriptUpdate {
	if id != nil {
		su = su.SetScriptToEnvironmentID(*id)
	}
	return su
}

// SetScriptToEnvironment sets the "ScriptToEnvironment" edge to the Environment entity.
func (su *ScriptUpdate) SetScriptToEnvironment(e *Environment) *ScriptUpdate {
	return su.SetScriptToEnvironmentID(e.ID)
}

// Mutation returns the ScriptMutation object of the builder.
func (su *ScriptUpdate) Mutation() *ScriptMutation {
	return su.mutation
}

// ClearScriptToUser clears all "ScriptToUser" edges to the User entity.
func (su *ScriptUpdate) ClearScriptToUser() *ScriptUpdate {
	su.mutation.ClearScriptToUser()
	return su
}

// RemoveScriptToUserIDs removes the "ScriptToUser" edge to User entities by IDs.
func (su *ScriptUpdate) RemoveScriptToUserIDs(ids ...uuid.UUID) *ScriptUpdate {
	su.mutation.RemoveScriptToUserIDs(ids...)
	return su
}

// RemoveScriptToUser removes "ScriptToUser" edges to User entities.
func (su *ScriptUpdate) RemoveScriptToUser(u ...*User) *ScriptUpdate {
	ids := make([]uuid.UUID, len(u))
	for i := range u {
		ids[i] = u[i].ID
	}
	return su.RemoveScriptToUserIDs(ids...)
}

// ClearScriptToFinding clears all "ScriptToFinding" edges to the Finding entity.
func (su *ScriptUpdate) ClearScriptToFinding() *ScriptUpdate {
	su.mutation.ClearScriptToFinding()
	return su
}

// RemoveScriptToFindingIDs removes the "ScriptToFinding" edge to Finding entities by IDs.
func (su *ScriptUpdate) RemoveScriptToFindingIDs(ids ...uuid.UUID) *ScriptUpdate {
	su.mutation.RemoveScriptToFindingIDs(ids...)
	return su
}

// RemoveScriptToFinding removes "ScriptToFinding" edges to Finding entities.
func (su *ScriptUpdate) RemoveScriptToFinding(f ...*Finding) *ScriptUpdate {
	ids := make([]uuid.UUID, len(f))
	for i := range f {
		ids[i] = f[i].ID
	}
	return su.RemoveScriptToFindingIDs(ids...)
}

// ClearScriptToEnvironment clears the "ScriptToEnvironment" edge to the Environment entity.
func (su *ScriptUpdate) ClearScriptToEnvironment() *ScriptUpdate {
	su.mutation.ClearScriptToEnvironment()
	return su
}

// Save executes the query and returns the number of nodes affected by the update operation.
func (su *ScriptUpdate) Save(ctx context.Context) (int, error) {
	return withHooks(ctx, su.sqlSave, su.mutation, su.hooks)
}

// SaveX is like Save, but panics if an error occurs.
func (su *ScriptUpdate) SaveX(ctx context.Context) int {
	affected, err := su.Save(ctx)
	if err != nil {
		panic(err)
	}
	return affected
}

// Exec executes the query.
func (su *ScriptUpdate) Exec(ctx context.Context) error {
	_, err := su.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (su *ScriptUpdate) ExecX(ctx context.Context) {
	if err := su.Exec(ctx); err != nil {
		panic(err)
	}
}

func (su *ScriptUpdate) sqlSave(ctx context.Context) (n int, err error) {
	_spec := sqlgraph.NewUpdateSpec(script.Table, script.Columns, sqlgraph.NewFieldSpec(script.FieldID, field.TypeUUID))
	if ps := su.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if value, ok := su.mutation.HCLID(); ok {
		_spec.SetField(script.FieldHCLID, field.TypeString, value)
	}
	if value, ok := su.mutation.Name(); ok {
		_spec.SetField(script.FieldName, field.TypeString, value)
	}
	if value, ok := su.mutation.Language(); ok {
		_spec.SetField(script.FieldLanguage, field.TypeString, value)
	}
	if value, ok := su.mutation.Description(); ok {
		_spec.SetField(script.FieldDescription, field.TypeString, value)
	}
	if value, ok := su.mutation.Source(); ok {
		_spec.SetField(script.FieldSource, field.TypeString, value)
	}
	if value, ok := su.mutation.SourceType(); ok {
		_spec.SetField(script.FieldSourceType, field.TypeString, value)
	}
	if value, ok := su.mutation.Cooldown(); ok {
		_spec.SetField(script.FieldCooldown, field.TypeInt, value)
	}
	if value, ok := su.mutation.AddedCooldown(); ok {
		_spec.AddField(script.FieldCooldown, field.TypeInt, value)
	}
	if value, ok := su.mutation.Timeout(); ok {
		_spec.SetField(script.FieldTimeout, field.TypeInt, value)
	}
	if value, ok := su.mutation.AddedTimeout(); ok {
		_spec.AddField(script.FieldTimeout, field.TypeInt, value)
	}
	if value, ok := su.mutation.IgnoreErrors(); ok {
		_spec.SetField(script.FieldIgnoreErrors, field.TypeBool, value)
	}
	if value, ok := su.mutation.Args(); ok {
		_spec.SetField(script.FieldArgs, field.TypeJSON, value)
	}
	if value, ok := su.mutation.AppendedArgs(); ok {
		_spec.AddModifier(func(u *sql.UpdateBuilder) {
			sqljson.Append(u, script.FieldArgs, value)
		})
	}
	if value, ok := su.mutation.Disabled(); ok {
		_spec.SetField(script.FieldDisabled, field.TypeBool, value)
	}
	if value, ok := su.mutation.Vars(); ok {
		_spec.SetField(script.FieldVars, field.TypeJSON, value)
	}
	if value, ok := su.mutation.AbsPath(); ok {
		_spec.SetField(script.FieldAbsPath, field.TypeString, value)
	}
	if value, ok := su.mutation.Tags(); ok {
		_spec.SetField(script.FieldTags, field.TypeJSON, value)
	}
	if su.mutation.ScriptToUserCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: false,
			Table:   script.ScriptToUserTable,
			Columns: []string{script.ScriptToUserColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(user.FieldID, field.TypeUUID),
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := su.mutation.RemovedScriptToUserIDs(); len(nodes) > 0 && !su.mutation.ScriptToUserCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: false,
			Table:   script.ScriptToUserTable,
			Columns: []string{script.ScriptToUserColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(user.FieldID, field.TypeUUID),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := su.mutation.ScriptToUserIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: false,
			Table:   script.ScriptToUserTable,
			Columns: []string{script.ScriptToUserColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(user.FieldID, field.TypeUUID),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	if su.mutation.ScriptToFindingCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: false,
			Table:   script.ScriptToFindingTable,
			Columns: []string{script.ScriptToFindingColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(finding.FieldID, field.TypeUUID),
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := su.mutation.RemovedScriptToFindingIDs(); len(nodes) > 0 && !su.mutation.ScriptToFindingCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: false,
			Table:   script.ScriptToFindingTable,
			Columns: []string{script.ScriptToFindingColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(finding.FieldID, field.TypeUUID),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := su.mutation.ScriptToFindingIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: false,
			Table:   script.ScriptToFindingTable,
			Columns: []string{script.ScriptToFindingColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(finding.FieldID, field.TypeUUID),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	if su.mutation.ScriptToEnvironmentCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: true,
			Table:   script.ScriptToEnvironmentTable,
			Columns: []string{script.ScriptToEnvironmentColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(environment.FieldID, field.TypeUUID),
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := su.mutation.ScriptToEnvironmentIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: true,
			Table:   script.ScriptToEnvironmentTable,
			Columns: []string{script.ScriptToEnvironmentColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(environment.FieldID, field.TypeUUID),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	if n, err = sqlgraph.UpdateNodes(ctx, su.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{script.Label}
		} else if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return 0, err
	}
	su.mutation.done = true
	return n, nil
}

// ScriptUpdateOne is the builder for updating a single Script entity.
type ScriptUpdateOne struct {
	config
	fields   []string
	hooks    []Hook
	mutation *ScriptMutation
}

// SetHCLID sets the "hcl_id" field.
func (suo *ScriptUpdateOne) SetHCLID(s string) *ScriptUpdateOne {
	suo.mutation.SetHCLID(s)
	return suo
}

// SetNillableHCLID sets the "hcl_id" field if the given value is not nil.
func (suo *ScriptUpdateOne) SetNillableHCLID(s *string) *ScriptUpdateOne {
	if s != nil {
		suo.SetHCLID(*s)
	}
	return suo
}

// SetName sets the "name" field.
func (suo *ScriptUpdateOne) SetName(s string) *ScriptUpdateOne {
	suo.mutation.SetName(s)
	return suo
}

// SetNillableName sets the "name" field if the given value is not nil.
func (suo *ScriptUpdateOne) SetNillableName(s *string) *ScriptUpdateOne {
	if s != nil {
		suo.SetName(*s)
	}
	return suo
}

// SetLanguage sets the "language" field.
func (suo *ScriptUpdateOne) SetLanguage(s string) *ScriptUpdateOne {
	suo.mutation.SetLanguage(s)
	return suo
}

// SetNillableLanguage sets the "language" field if the given value is not nil.
func (suo *ScriptUpdateOne) SetNillableLanguage(s *string) *ScriptUpdateOne {
	if s != nil {
		suo.SetLanguage(*s)
	}
	return suo
}

// SetDescription sets the "description" field.
func (suo *ScriptUpdateOne) SetDescription(s string) *ScriptUpdateOne {
	suo.mutation.SetDescription(s)
	return suo
}

// SetNillableDescription sets the "description" field if the given value is not nil.
func (suo *ScriptUpdateOne) SetNillableDescription(s *string) *ScriptUpdateOne {
	if s != nil {
		suo.SetDescription(*s)
	}
	return suo
}

// SetSource sets the "source" field.
func (suo *ScriptUpdateOne) SetSource(s string) *ScriptUpdateOne {
	suo.mutation.SetSource(s)
	return suo
}

// SetNillableSource sets the "source" field if the given value is not nil.
func (suo *ScriptUpdateOne) SetNillableSource(s *string) *ScriptUpdateOne {
	if s != nil {
		suo.SetSource(*s)
	}
	return suo
}

// SetSourceType sets the "source_type" field.
func (suo *ScriptUpdateOne) SetSourceType(s string) *ScriptUpdateOne {
	suo.mutation.SetSourceType(s)
	return suo
}

// SetNillableSourceType sets the "source_type" field if the given value is not nil.
func (suo *ScriptUpdateOne) SetNillableSourceType(s *string) *ScriptUpdateOne {
	if s != nil {
		suo.SetSourceType(*s)
	}
	return suo
}

// SetCooldown sets the "cooldown" field.
func (suo *ScriptUpdateOne) SetCooldown(i int) *ScriptUpdateOne {
	suo.mutation.ResetCooldown()
	suo.mutation.SetCooldown(i)
	return suo
}

// SetNillableCooldown sets the "cooldown" field if the given value is not nil.
func (suo *ScriptUpdateOne) SetNillableCooldown(i *int) *ScriptUpdateOne {
	if i != nil {
		suo.SetCooldown(*i)
	}
	return suo
}

// AddCooldown adds i to the "cooldown" field.
func (suo *ScriptUpdateOne) AddCooldown(i int) *ScriptUpdateOne {
	suo.mutation.AddCooldown(i)
	return suo
}

// SetTimeout sets the "timeout" field.
func (suo *ScriptUpdateOne) SetTimeout(i int) *ScriptUpdateOne {
	suo.mutation.ResetTimeout()
	suo.mutation.SetTimeout(i)
	return suo
}

// SetNillableTimeout sets the "timeout" field if the given value is not nil.
func (suo *ScriptUpdateOne) SetNillableTimeout(i *int) *ScriptUpdateOne {
	if i != nil {
		suo.SetTimeout(*i)
	}
	return suo
}

// AddTimeout adds i to the "timeout" field.
func (suo *ScriptUpdateOne) AddTimeout(i int) *ScriptUpdateOne {
	suo.mutation.AddTimeout(i)
	return suo
}

// SetIgnoreErrors sets the "ignore_errors" field.
func (suo *ScriptUpdateOne) SetIgnoreErrors(b bool) *ScriptUpdateOne {
	suo.mutation.SetIgnoreErrors(b)
	return suo
}

// SetNillableIgnoreErrors sets the "ignore_errors" field if the given value is not nil.
func (suo *ScriptUpdateOne) SetNillableIgnoreErrors(b *bool) *ScriptUpdateOne {
	if b != nil {
		suo.SetIgnoreErrors(*b)
	}
	return suo
}

// SetArgs sets the "args" field.
func (suo *ScriptUpdateOne) SetArgs(s []string) *ScriptUpdateOne {
	suo.mutation.SetArgs(s)
	return suo
}

// AppendArgs appends s to the "args" field.
func (suo *ScriptUpdateOne) AppendArgs(s []string) *ScriptUpdateOne {
	suo.mutation.AppendArgs(s)
	return suo
}

// SetDisabled sets the "disabled" field.
func (suo *ScriptUpdateOne) SetDisabled(b bool) *ScriptUpdateOne {
	suo.mutation.SetDisabled(b)
	return suo
}

// SetNillableDisabled sets the "disabled" field if the given value is not nil.
func (suo *ScriptUpdateOne) SetNillableDisabled(b *bool) *ScriptUpdateOne {
	if b != nil {
		suo.SetDisabled(*b)
	}
	return suo
}

// SetVars sets the "vars" field.
func (suo *ScriptUpdateOne) SetVars(m map[string]string) *ScriptUpdateOne {
	suo.mutation.SetVars(m)
	return suo
}

// SetAbsPath sets the "abs_path" field.
func (suo *ScriptUpdateOne) SetAbsPath(s string) *ScriptUpdateOne {
	suo.mutation.SetAbsPath(s)
	return suo
}

// SetNillableAbsPath sets the "abs_path" field if the given value is not nil.
func (suo *ScriptUpdateOne) SetNillableAbsPath(s *string) *ScriptUpdateOne {
	if s != nil {
		suo.SetAbsPath(*s)
	}
	return suo
}

// SetTags sets the "tags" field.
func (suo *ScriptUpdateOne) SetTags(m map[string]string) *ScriptUpdateOne {
	suo.mutation.SetTags(m)
	return suo
}

// AddScriptToUserIDs adds the "ScriptToUser" edge to the User entity by IDs.
func (suo *ScriptUpdateOne) AddScriptToUserIDs(ids ...uuid.UUID) *ScriptUpdateOne {
	suo.mutation.AddScriptToUserIDs(ids...)
	return suo
}

// AddScriptToUser adds the "ScriptToUser" edges to the User entity.
func (suo *ScriptUpdateOne) AddScriptToUser(u ...*User) *ScriptUpdateOne {
	ids := make([]uuid.UUID, len(u))
	for i := range u {
		ids[i] = u[i].ID
	}
	return suo.AddScriptToUserIDs(ids...)
}

// AddScriptToFindingIDs adds the "ScriptToFinding" edge to the Finding entity by IDs.
func (suo *ScriptUpdateOne) AddScriptToFindingIDs(ids ...uuid.UUID) *ScriptUpdateOne {
	suo.mutation.AddScriptToFindingIDs(ids...)
	return suo
}

// AddScriptToFinding adds the "ScriptToFinding" edges to the Finding entity.
func (suo *ScriptUpdateOne) AddScriptToFinding(f ...*Finding) *ScriptUpdateOne {
	ids := make([]uuid.UUID, len(f))
	for i := range f {
		ids[i] = f[i].ID
	}
	return suo.AddScriptToFindingIDs(ids...)
}

// SetScriptToEnvironmentID sets the "ScriptToEnvironment" edge to the Environment entity by ID.
func (suo *ScriptUpdateOne) SetScriptToEnvironmentID(id uuid.UUID) *ScriptUpdateOne {
	suo.mutation.SetScriptToEnvironmentID(id)
	return suo
}

// SetNillableScriptToEnvironmentID sets the "ScriptToEnvironment" edge to the Environment entity by ID if the given value is not nil.
func (suo *ScriptUpdateOne) SetNillableScriptToEnvironmentID(id *uuid.UUID) *ScriptUpdateOne {
	if id != nil {
		suo = suo.SetScriptToEnvironmentID(*id)
	}
	return suo
}

// SetScriptToEnvironment sets the "ScriptToEnvironment" edge to the Environment entity.
func (suo *ScriptUpdateOne) SetScriptToEnvironment(e *Environment) *ScriptUpdateOne {
	return suo.SetScriptToEnvironmentID(e.ID)
}

// Mutation returns the ScriptMutation object of the builder.
func (suo *ScriptUpdateOne) Mutation() *ScriptMutation {
	return suo.mutation
}

// ClearScriptToUser clears all "ScriptToUser" edges to the User entity.
func (suo *ScriptUpdateOne) ClearScriptToUser() *ScriptUpdateOne {
	suo.mutation.ClearScriptToUser()
	return suo
}

// RemoveScriptToUserIDs removes the "ScriptToUser" edge to User entities by IDs.
func (suo *ScriptUpdateOne) RemoveScriptToUserIDs(ids ...uuid.UUID) *ScriptUpdateOne {
	suo.mutation.RemoveScriptToUserIDs(ids...)
	return suo
}

// RemoveScriptToUser removes "ScriptToUser" edges to User entities.
func (suo *ScriptUpdateOne) RemoveScriptToUser(u ...*User) *ScriptUpdateOne {
	ids := make([]uuid.UUID, len(u))
	for i := range u {
		ids[i] = u[i].ID
	}
	return suo.RemoveScriptToUserIDs(ids...)
}

// ClearScriptToFinding clears all "ScriptToFinding" edges to the Finding entity.
func (suo *ScriptUpdateOne) ClearScriptToFinding() *ScriptUpdateOne {
	suo.mutation.ClearScriptToFinding()
	return suo
}

// RemoveScriptToFindingIDs removes the "ScriptToFinding" edge to Finding entities by IDs.
func (suo *ScriptUpdateOne) RemoveScriptToFindingIDs(ids ...uuid.UUID) *ScriptUpdateOne {
	suo.mutation.RemoveScriptToFindingIDs(ids...)
	return suo
}

// RemoveScriptToFinding removes "ScriptToFinding" edges to Finding entities.
func (suo *ScriptUpdateOne) RemoveScriptToFinding(f ...*Finding) *ScriptUpdateOne {
	ids := make([]uuid.UUID, len(f))
	for i := range f {
		ids[i] = f[i].ID
	}
	return suo.RemoveScriptToFindingIDs(ids...)
}

// ClearScriptToEnvironment clears the "ScriptToEnvironment" edge to the Environment entity.
func (suo *ScriptUpdateOne) ClearScriptToEnvironment() *ScriptUpdateOne {
	suo.mutation.ClearScriptToEnvironment()
	return suo
}

// Where appends a list predicates to the ScriptUpdate builder.
func (suo *ScriptUpdateOne) Where(ps ...predicate.Script) *ScriptUpdateOne {
	suo.mutation.Where(ps...)
	return suo
}

// Select allows selecting one or more fields (columns) of the returned entity.
// The default is selecting all fields defined in the entity schema.
func (suo *ScriptUpdateOne) Select(field string, fields ...string) *ScriptUpdateOne {
	suo.fields = append([]string{field}, fields...)
	return suo
}

// Save executes the query and returns the updated Script entity.
func (suo *ScriptUpdateOne) Save(ctx context.Context) (*Script, error) {
	return withHooks(ctx, suo.sqlSave, suo.mutation, suo.hooks)
}

// SaveX is like Save, but panics if an error occurs.
func (suo *ScriptUpdateOne) SaveX(ctx context.Context) *Script {
	node, err := suo.Save(ctx)
	if err != nil {
		panic(err)
	}
	return node
}

// Exec executes the query on the entity.
func (suo *ScriptUpdateOne) Exec(ctx context.Context) error {
	_, err := suo.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (suo *ScriptUpdateOne) ExecX(ctx context.Context) {
	if err := suo.Exec(ctx); err != nil {
		panic(err)
	}
}

func (suo *ScriptUpdateOne) sqlSave(ctx context.Context) (_node *Script, err error) {
	_spec := sqlgraph.NewUpdateSpec(script.Table, script.Columns, sqlgraph.NewFieldSpec(script.FieldID, field.TypeUUID))
	id, ok := suo.mutation.ID()
	if !ok {
		return nil, &ValidationError{Name: "id", err: errors.New(`ent: missing "Script.id" for update`)}
	}
	_spec.Node.ID.Value = id
	if fields := suo.fields; len(fields) > 0 {
		_spec.Node.Columns = make([]string, 0, len(fields))
		_spec.Node.Columns = append(_spec.Node.Columns, script.FieldID)
		for _, f := range fields {
			if !script.ValidColumn(f) {
				return nil, &ValidationError{Name: f, err: fmt.Errorf("ent: invalid field %q for query", f)}
			}
			if f != script.FieldID {
				_spec.Node.Columns = append(_spec.Node.Columns, f)
			}
		}
	}
	if ps := suo.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if value, ok := suo.mutation.HCLID(); ok {
		_spec.SetField(script.FieldHCLID, field.TypeString, value)
	}
	if value, ok := suo.mutation.Name(); ok {
		_spec.SetField(script.FieldName, field.TypeString, value)
	}
	if value, ok := suo.mutation.Language(); ok {
		_spec.SetField(script.FieldLanguage, field.TypeString, value)
	}
	if value, ok := suo.mutation.Description(); ok {
		_spec.SetField(script.FieldDescription, field.TypeString, value)
	}
	if value, ok := suo.mutation.Source(); ok {
		_spec.SetField(script.FieldSource, field.TypeString, value)
	}
	if value, ok := suo.mutation.SourceType(); ok {
		_spec.SetField(script.FieldSourceType, field.TypeString, value)
	}
	if value, ok := suo.mutation.Cooldown(); ok {
		_spec.SetField(script.FieldCooldown, field.TypeInt, value)
	}
	if value, ok := suo.mutation.AddedCooldown(); ok {
		_spec.AddField(script.FieldCooldown, field.TypeInt, value)
	}
	if value, ok := suo.mutation.Timeout(); ok {
		_spec.SetField(script.FieldTimeout, field.TypeInt, value)
	}
	if value, ok := suo.mutation.AddedTimeout(); ok {
		_spec.AddField(script.FieldTimeout, field.TypeInt, value)
	}
	if value, ok := suo.mutation.IgnoreErrors(); ok {
		_spec.SetField(script.FieldIgnoreErrors, field.TypeBool, value)
	}
	if value, ok := suo.mutation.Args(); ok {
		_spec.SetField(script.FieldArgs, field.TypeJSON, value)
	}
	if value, ok := suo.mutation.AppendedArgs(); ok {
		_spec.AddModifier(func(u *sql.UpdateBuilder) {
			sqljson.Append(u, script.FieldArgs, value)
		})
	}
	if value, ok := suo.mutation.Disabled(); ok {
		_spec.SetField(script.FieldDisabled, field.TypeBool, value)
	}
	if value, ok := suo.mutation.Vars(); ok {
		_spec.SetField(script.FieldVars, field.TypeJSON, value)
	}
	if value, ok := suo.mutation.AbsPath(); ok {
		_spec.SetField(script.FieldAbsPath, field.TypeString, value)
	}
	if value, ok := suo.mutation.Tags(); ok {
		_spec.SetField(script.FieldTags, field.TypeJSON, value)
	}
	if suo.mutation.ScriptToUserCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: false,
			Table:   script.ScriptToUserTable,
			Columns: []string{script.ScriptToUserColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(user.FieldID, field.TypeUUID),
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := suo.mutation.RemovedScriptToUserIDs(); len(nodes) > 0 && !suo.mutation.ScriptToUserCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: false,
			Table:   script.ScriptToUserTable,
			Columns: []string{script.ScriptToUserColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(user.FieldID, field.TypeUUID),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := suo.mutation.ScriptToUserIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: false,
			Table:   script.ScriptToUserTable,
			Columns: []string{script.ScriptToUserColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(user.FieldID, field.TypeUUID),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	if suo.mutation.ScriptToFindingCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: false,
			Table:   script.ScriptToFindingTable,
			Columns: []string{script.ScriptToFindingColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(finding.FieldID, field.TypeUUID),
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := suo.mutation.RemovedScriptToFindingIDs(); len(nodes) > 0 && !suo.mutation.ScriptToFindingCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: false,
			Table:   script.ScriptToFindingTable,
			Columns: []string{script.ScriptToFindingColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(finding.FieldID, field.TypeUUID),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := suo.mutation.ScriptToFindingIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: false,
			Table:   script.ScriptToFindingTable,
			Columns: []string{script.ScriptToFindingColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(finding.FieldID, field.TypeUUID),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	if suo.mutation.ScriptToEnvironmentCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: true,
			Table:   script.ScriptToEnvironmentTable,
			Columns: []string{script.ScriptToEnvironmentColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(environment.FieldID, field.TypeUUID),
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := suo.mutation.ScriptToEnvironmentIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: true,
			Table:   script.ScriptToEnvironmentTable,
			Columns: []string{script.ScriptToEnvironmentColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(environment.FieldID, field.TypeUUID),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	_node = &Script{config: suo.config}
	_spec.Assign = _node.assignValues
	_spec.ScanValues = _node.scanValues
	if err = sqlgraph.UpdateNode(ctx, suo.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{script.Label}
		} else if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return nil, err
	}
	suo.mutation.done = true
	return _node, nil
}
