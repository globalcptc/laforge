// Code generated by entc, DO NOT EDIT.

package ent

import (
	"context"
	"errors"
	"fmt"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/schema/field"
	"github.com/gen0cide/laforge/ent/agenttask"
	"github.com/gen0cide/laforge/ent/environment"
	"github.com/gen0cide/laforge/ent/predicate"
	"github.com/gen0cide/laforge/ent/script"
	"github.com/gen0cide/laforge/ent/validation"
	"github.com/google/uuid"
)

// ValidationUpdate is the builder for updating Validation entities.
type ValidationUpdate struct {
	config
	hooks    []Hook
	mutation *ValidationMutation
}

// Where appends a list predicates to the ValidationUpdate builder.
func (vu *ValidationUpdate) Where(ps ...predicate.Validation) *ValidationUpdate {
	vu.mutation.Where(ps...)
	return vu
}

// SetHclID sets the "hcl_id" field.
func (vu *ValidationUpdate) SetHclID(s string) *ValidationUpdate {
	vu.mutation.SetHclID(s)
	return vu
}

// SetValidationType sets the "validation_type" field.
func (vu *ValidationUpdate) SetValidationType(s string) *ValidationUpdate {
	vu.mutation.SetValidationType(s)
	return vu
}

// SetNillableValidationType sets the "validation_type" field if the given value is not nil.
func (vu *ValidationUpdate) SetNillableValidationType(s *string) *ValidationUpdate {
	if s != nil {
		vu.SetValidationType(*s)
	}
	return vu
}

// SetOutput sets the "output" field.
func (vu *ValidationUpdate) SetOutput(s string) *ValidationUpdate {
	vu.mutation.SetOutput(s)
	return vu
}

// SetNillableOutput sets the "output" field if the given value is not nil.
func (vu *ValidationUpdate) SetNillableOutput(s *string) *ValidationUpdate {
	if s != nil {
		vu.SetOutput(*s)
	}
	return vu
}

// SetState sets the "state" field.
func (vu *ValidationUpdate) SetState(v validation.State) *ValidationUpdate {
	vu.mutation.SetState(v)
	return vu
}

// SetErrorMessage sets the "error_message" field.
func (vu *ValidationUpdate) SetErrorMessage(s string) *ValidationUpdate {
	vu.mutation.SetErrorMessage(s)
	return vu
}

// SetNillableErrorMessage sets the "error_message" field if the given value is not nil.
func (vu *ValidationUpdate) SetNillableErrorMessage(s *string) *ValidationUpdate {
	if s != nil {
		vu.SetErrorMessage(*s)
	}
	return vu
}

// SetHash sets the "hash" field.
func (vu *ValidationUpdate) SetHash(s string) *ValidationUpdate {
	vu.mutation.SetHash(s)
	return vu
}

// SetRegex sets the "regex" field.
func (vu *ValidationUpdate) SetRegex(s string) *ValidationUpdate {
	vu.mutation.SetRegex(s)
	return vu
}

// SetIP sets the "ip" field.
func (vu *ValidationUpdate) SetIP(s string) *ValidationUpdate {
	vu.mutation.SetIP(s)
	return vu
}

// SetPort sets the "port" field.
func (vu *ValidationUpdate) SetPort(i int) *ValidationUpdate {
	vu.mutation.ResetPort()
	vu.mutation.SetPort(i)
	return vu
}

// AddPort adds i to the "port" field.
func (vu *ValidationUpdate) AddPort(i int) *ValidationUpdate {
	vu.mutation.AddPort(i)
	return vu
}

// SetHostname sets the "hostname" field.
func (vu *ValidationUpdate) SetHostname(s string) *ValidationUpdate {
	vu.mutation.SetHostname(s)
	return vu
}

// SetNameservers sets the "nameservers" field.
func (vu *ValidationUpdate) SetNameservers(s []string) *ValidationUpdate {
	vu.mutation.SetNameservers(s)
	return vu
}

// SetPackageName sets the "package_name" field.
func (vu *ValidationUpdate) SetPackageName(s string) *ValidationUpdate {
	vu.mutation.SetPackageName(s)
	return vu
}

// SetUsername sets the "username" field.
func (vu *ValidationUpdate) SetUsername(s string) *ValidationUpdate {
	vu.mutation.SetUsername(s)
	return vu
}

// SetGroupName sets the "group_name" field.
func (vu *ValidationUpdate) SetGroupName(s string) *ValidationUpdate {
	vu.mutation.SetGroupName(s)
	return vu
}

// SetFilePath sets the "file_path" field.
func (vu *ValidationUpdate) SetFilePath(s string) *ValidationUpdate {
	vu.mutation.SetFilePath(s)
	return vu
}

// SetSearchString sets the "search_string" field.
func (vu *ValidationUpdate) SetSearchString(s string) *ValidationUpdate {
	vu.mutation.SetSearchString(s)
	return vu
}

// SetServiceName sets the "service_name" field.
func (vu *ValidationUpdate) SetServiceName(s string) *ValidationUpdate {
	vu.mutation.SetServiceName(s)
	return vu
}

// SetServiceStatus sets the "service_status" field.
func (vu *ValidationUpdate) SetServiceStatus(s string) *ValidationUpdate {
	vu.mutation.SetServiceStatus(s)
	return vu
}

// SetProcessName sets the "process_name" field.
func (vu *ValidationUpdate) SetProcessName(s string) *ValidationUpdate {
	vu.mutation.SetProcessName(s)
	return vu
}

// SetValidationToAgentTaskID sets the "ValidationToAgentTask" edge to the AgentTask entity by ID.
func (vu *ValidationUpdate) SetValidationToAgentTaskID(id uuid.UUID) *ValidationUpdate {
	vu.mutation.SetValidationToAgentTaskID(id)
	return vu
}

// SetNillableValidationToAgentTaskID sets the "ValidationToAgentTask" edge to the AgentTask entity by ID if the given value is not nil.
func (vu *ValidationUpdate) SetNillableValidationToAgentTaskID(id *uuid.UUID) *ValidationUpdate {
	if id != nil {
		vu = vu.SetValidationToAgentTaskID(*id)
	}
	return vu
}

// SetValidationToAgentTask sets the "ValidationToAgentTask" edge to the AgentTask entity.
func (vu *ValidationUpdate) SetValidationToAgentTask(a *AgentTask) *ValidationUpdate {
	return vu.SetValidationToAgentTaskID(a.ID)
}

// AddValidationToScriptIDs adds the "ValidationToScript" edge to the Script entity by IDs.
func (vu *ValidationUpdate) AddValidationToScriptIDs(ids ...uuid.UUID) *ValidationUpdate {
	vu.mutation.AddValidationToScriptIDs(ids...)
	return vu
}

// AddValidationToScript adds the "ValidationToScript" edges to the Script entity.
func (vu *ValidationUpdate) AddValidationToScript(s ...*Script) *ValidationUpdate {
	ids := make([]uuid.UUID, len(s))
	for i := range s {
		ids[i] = s[i].ID
	}
	return vu.AddValidationToScriptIDs(ids...)
}

// AddValidationToEnvironmentIDs adds the "ValidationToEnvironment" edge to the Environment entity by IDs.
func (vu *ValidationUpdate) AddValidationToEnvironmentIDs(ids ...uuid.UUID) *ValidationUpdate {
	vu.mutation.AddValidationToEnvironmentIDs(ids...)
	return vu
}

// AddValidationToEnvironment adds the "ValidationToEnvironment" edges to the Environment entity.
func (vu *ValidationUpdate) AddValidationToEnvironment(e ...*Environment) *ValidationUpdate {
	ids := make([]uuid.UUID, len(e))
	for i := range e {
		ids[i] = e[i].ID
	}
	return vu.AddValidationToEnvironmentIDs(ids...)
}

// Mutation returns the ValidationMutation object of the builder.
func (vu *ValidationUpdate) Mutation() *ValidationMutation {
	return vu.mutation
}

// ClearValidationToAgentTask clears the "ValidationToAgentTask" edge to the AgentTask entity.
func (vu *ValidationUpdate) ClearValidationToAgentTask() *ValidationUpdate {
	vu.mutation.ClearValidationToAgentTask()
	return vu
}

// ClearValidationToScript clears all "ValidationToScript" edges to the Script entity.
func (vu *ValidationUpdate) ClearValidationToScript() *ValidationUpdate {
	vu.mutation.ClearValidationToScript()
	return vu
}

// RemoveValidationToScriptIDs removes the "ValidationToScript" edge to Script entities by IDs.
func (vu *ValidationUpdate) RemoveValidationToScriptIDs(ids ...uuid.UUID) *ValidationUpdate {
	vu.mutation.RemoveValidationToScriptIDs(ids...)
	return vu
}

// RemoveValidationToScript removes "ValidationToScript" edges to Script entities.
func (vu *ValidationUpdate) RemoveValidationToScript(s ...*Script) *ValidationUpdate {
	ids := make([]uuid.UUID, len(s))
	for i := range s {
		ids[i] = s[i].ID
	}
	return vu.RemoveValidationToScriptIDs(ids...)
}

// ClearValidationToEnvironment clears all "ValidationToEnvironment" edges to the Environment entity.
func (vu *ValidationUpdate) ClearValidationToEnvironment() *ValidationUpdate {
	vu.mutation.ClearValidationToEnvironment()
	return vu
}

// RemoveValidationToEnvironmentIDs removes the "ValidationToEnvironment" edge to Environment entities by IDs.
func (vu *ValidationUpdate) RemoveValidationToEnvironmentIDs(ids ...uuid.UUID) *ValidationUpdate {
	vu.mutation.RemoveValidationToEnvironmentIDs(ids...)
	return vu
}

// RemoveValidationToEnvironment removes "ValidationToEnvironment" edges to Environment entities.
func (vu *ValidationUpdate) RemoveValidationToEnvironment(e ...*Environment) *ValidationUpdate {
	ids := make([]uuid.UUID, len(e))
	for i := range e {
		ids[i] = e[i].ID
	}
	return vu.RemoveValidationToEnvironmentIDs(ids...)
}

// Save executes the query and returns the number of nodes affected by the update operation.
func (vu *ValidationUpdate) Save(ctx context.Context) (int, error) {
	var (
		err      error
		affected int
	)
	if len(vu.hooks) == 0 {
		if err = vu.check(); err != nil {
			return 0, err
		}
		affected, err = vu.sqlSave(ctx)
	} else {
		var mut Mutator = MutateFunc(func(ctx context.Context, m Mutation) (Value, error) {
			mutation, ok := m.(*ValidationMutation)
			if !ok {
				return nil, fmt.Errorf("unexpected mutation type %T", m)
			}
			if err = vu.check(); err != nil {
				return 0, err
			}
			vu.mutation = mutation
			affected, err = vu.sqlSave(ctx)
			mutation.done = true
			return affected, err
		})
		for i := len(vu.hooks) - 1; i >= 0; i-- {
			if vu.hooks[i] == nil {
				return 0, fmt.Errorf("ent: uninitialized hook (forgotten import ent/runtime?)")
			}
			mut = vu.hooks[i](mut)
		}
		if _, err := mut.Mutate(ctx, vu.mutation); err != nil {
			return 0, err
		}
	}
	return affected, err
}

// SaveX is like Save, but panics if an error occurs.
func (vu *ValidationUpdate) SaveX(ctx context.Context) int {
	affected, err := vu.Save(ctx)
	if err != nil {
		panic(err)
	}
	return affected
}

// Exec executes the query.
func (vu *ValidationUpdate) Exec(ctx context.Context) error {
	_, err := vu.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (vu *ValidationUpdate) ExecX(ctx context.Context) {
	if err := vu.Exec(ctx); err != nil {
		panic(err)
	}
}

// check runs all checks and user-defined validators on the builder.
func (vu *ValidationUpdate) check() error {
	if v, ok := vu.mutation.State(); ok {
		if err := validation.StateValidator(v); err != nil {
			return &ValidationError{Name: "state", err: fmt.Errorf(`ent: validator failed for field "Validation.state": %w`, err)}
		}
	}
	return nil
}

func (vu *ValidationUpdate) sqlSave(ctx context.Context) (n int, err error) {
	_spec := &sqlgraph.UpdateSpec{
		Node: &sqlgraph.NodeSpec{
			Table:   validation.Table,
			Columns: validation.Columns,
			ID: &sqlgraph.FieldSpec{
				Type:   field.TypeUUID,
				Column: validation.FieldID,
			},
		},
	}
	if ps := vu.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if value, ok := vu.mutation.HclID(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Value:  value,
			Column: validation.FieldHclID,
		})
	}
	if value, ok := vu.mutation.ValidationType(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Value:  value,
			Column: validation.FieldValidationType,
		})
	}
	if value, ok := vu.mutation.Output(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Value:  value,
			Column: validation.FieldOutput,
		})
	}
	if value, ok := vu.mutation.State(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeEnum,
			Value:  value,
			Column: validation.FieldState,
		})
	}
	if value, ok := vu.mutation.ErrorMessage(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Value:  value,
			Column: validation.FieldErrorMessage,
		})
	}
	if value, ok := vu.mutation.Hash(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Value:  value,
			Column: validation.FieldHash,
		})
	}
	if value, ok := vu.mutation.Regex(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Value:  value,
			Column: validation.FieldRegex,
		})
	}
	if value, ok := vu.mutation.IP(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Value:  value,
			Column: validation.FieldIP,
		})
	}
	if value, ok := vu.mutation.Port(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeInt,
			Value:  value,
			Column: validation.FieldPort,
		})
	}
	if value, ok := vu.mutation.AddedPort(); ok {
		_spec.Fields.Add = append(_spec.Fields.Add, &sqlgraph.FieldSpec{
			Type:   field.TypeInt,
			Value:  value,
			Column: validation.FieldPort,
		})
	}
	if value, ok := vu.mutation.Hostname(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Value:  value,
			Column: validation.FieldHostname,
		})
	}
	if value, ok := vu.mutation.Nameservers(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeJSON,
			Value:  value,
			Column: validation.FieldNameservers,
		})
	}
	if value, ok := vu.mutation.PackageName(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Value:  value,
			Column: validation.FieldPackageName,
		})
	}
	if value, ok := vu.mutation.Username(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Value:  value,
			Column: validation.FieldUsername,
		})
	}
	if value, ok := vu.mutation.GroupName(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Value:  value,
			Column: validation.FieldGroupName,
		})
	}
	if value, ok := vu.mutation.FilePath(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Value:  value,
			Column: validation.FieldFilePath,
		})
	}
	if value, ok := vu.mutation.SearchString(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Value:  value,
			Column: validation.FieldSearchString,
		})
	}
	if value, ok := vu.mutation.ServiceName(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Value:  value,
			Column: validation.FieldServiceName,
		})
	}
	if value, ok := vu.mutation.ServiceStatus(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Value:  value,
			Column: validation.FieldServiceStatus,
		})
	}
	if value, ok := vu.mutation.ProcessName(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Value:  value,
			Column: validation.FieldProcessName,
		})
	}
	if vu.mutation.ValidationToAgentTaskCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2O,
			Inverse: true,
			Table:   validation.ValidationToAgentTaskTable,
			Columns: []string{validation.ValidationToAgentTaskColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeUUID,
					Column: agenttask.FieldID,
				},
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := vu.mutation.ValidationToAgentTaskIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2O,
			Inverse: true,
			Table:   validation.ValidationToAgentTaskTable,
			Columns: []string{validation.ValidationToAgentTaskColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeUUID,
					Column: agenttask.FieldID,
				},
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	if vu.mutation.ValidationToScriptCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: true,
			Table:   validation.ValidationToScriptTable,
			Columns: []string{validation.ValidationToScriptColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeUUID,
					Column: script.FieldID,
				},
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := vu.mutation.RemovedValidationToScriptIDs(); len(nodes) > 0 && !vu.mutation.ValidationToScriptCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: true,
			Table:   validation.ValidationToScriptTable,
			Columns: []string{validation.ValidationToScriptColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeUUID,
					Column: script.FieldID,
				},
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := vu.mutation.ValidationToScriptIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: true,
			Table:   validation.ValidationToScriptTable,
			Columns: []string{validation.ValidationToScriptColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeUUID,
					Column: script.FieldID,
				},
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	if vu.mutation.ValidationToEnvironmentCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2M,
			Inverse: true,
			Table:   validation.ValidationToEnvironmentTable,
			Columns: validation.ValidationToEnvironmentPrimaryKey,
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeUUID,
					Column: environment.FieldID,
				},
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := vu.mutation.RemovedValidationToEnvironmentIDs(); len(nodes) > 0 && !vu.mutation.ValidationToEnvironmentCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2M,
			Inverse: true,
			Table:   validation.ValidationToEnvironmentTable,
			Columns: validation.ValidationToEnvironmentPrimaryKey,
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
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := vu.mutation.ValidationToEnvironmentIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2M,
			Inverse: true,
			Table:   validation.ValidationToEnvironmentTable,
			Columns: validation.ValidationToEnvironmentPrimaryKey,
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
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	if n, err = sqlgraph.UpdateNodes(ctx, vu.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{validation.Label}
		} else if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{err.Error(), err}
		}
		return 0, err
	}
	return n, nil
}

// ValidationUpdateOne is the builder for updating a single Validation entity.
type ValidationUpdateOne struct {
	config
	fields   []string
	hooks    []Hook
	mutation *ValidationMutation
}

// SetHclID sets the "hcl_id" field.
func (vuo *ValidationUpdateOne) SetHclID(s string) *ValidationUpdateOne {
	vuo.mutation.SetHclID(s)
	return vuo
}

// SetValidationType sets the "validation_type" field.
func (vuo *ValidationUpdateOne) SetValidationType(s string) *ValidationUpdateOne {
	vuo.mutation.SetValidationType(s)
	return vuo
}

// SetNillableValidationType sets the "validation_type" field if the given value is not nil.
func (vuo *ValidationUpdateOne) SetNillableValidationType(s *string) *ValidationUpdateOne {
	if s != nil {
		vuo.SetValidationType(*s)
	}
	return vuo
}

// SetOutput sets the "output" field.
func (vuo *ValidationUpdateOne) SetOutput(s string) *ValidationUpdateOne {
	vuo.mutation.SetOutput(s)
	return vuo
}

// SetNillableOutput sets the "output" field if the given value is not nil.
func (vuo *ValidationUpdateOne) SetNillableOutput(s *string) *ValidationUpdateOne {
	if s != nil {
		vuo.SetOutput(*s)
	}
	return vuo
}

// SetState sets the "state" field.
func (vuo *ValidationUpdateOne) SetState(v validation.State) *ValidationUpdateOne {
	vuo.mutation.SetState(v)
	return vuo
}

// SetErrorMessage sets the "error_message" field.
func (vuo *ValidationUpdateOne) SetErrorMessage(s string) *ValidationUpdateOne {
	vuo.mutation.SetErrorMessage(s)
	return vuo
}

// SetNillableErrorMessage sets the "error_message" field if the given value is not nil.
func (vuo *ValidationUpdateOne) SetNillableErrorMessage(s *string) *ValidationUpdateOne {
	if s != nil {
		vuo.SetErrorMessage(*s)
	}
	return vuo
}

// SetHash sets the "hash" field.
func (vuo *ValidationUpdateOne) SetHash(s string) *ValidationUpdateOne {
	vuo.mutation.SetHash(s)
	return vuo
}

// SetRegex sets the "regex" field.
func (vuo *ValidationUpdateOne) SetRegex(s string) *ValidationUpdateOne {
	vuo.mutation.SetRegex(s)
	return vuo
}

// SetIP sets the "ip" field.
func (vuo *ValidationUpdateOne) SetIP(s string) *ValidationUpdateOne {
	vuo.mutation.SetIP(s)
	return vuo
}

// SetPort sets the "port" field.
func (vuo *ValidationUpdateOne) SetPort(i int) *ValidationUpdateOne {
	vuo.mutation.ResetPort()
	vuo.mutation.SetPort(i)
	return vuo
}

// AddPort adds i to the "port" field.
func (vuo *ValidationUpdateOne) AddPort(i int) *ValidationUpdateOne {
	vuo.mutation.AddPort(i)
	return vuo
}

// SetHostname sets the "hostname" field.
func (vuo *ValidationUpdateOne) SetHostname(s string) *ValidationUpdateOne {
	vuo.mutation.SetHostname(s)
	return vuo
}

// SetNameservers sets the "nameservers" field.
func (vuo *ValidationUpdateOne) SetNameservers(s []string) *ValidationUpdateOne {
	vuo.mutation.SetNameservers(s)
	return vuo
}

// SetPackageName sets the "package_name" field.
func (vuo *ValidationUpdateOne) SetPackageName(s string) *ValidationUpdateOne {
	vuo.mutation.SetPackageName(s)
	return vuo
}

// SetUsername sets the "username" field.
func (vuo *ValidationUpdateOne) SetUsername(s string) *ValidationUpdateOne {
	vuo.mutation.SetUsername(s)
	return vuo
}

// SetGroupName sets the "group_name" field.
func (vuo *ValidationUpdateOne) SetGroupName(s string) *ValidationUpdateOne {
	vuo.mutation.SetGroupName(s)
	return vuo
}

// SetFilePath sets the "file_path" field.
func (vuo *ValidationUpdateOne) SetFilePath(s string) *ValidationUpdateOne {
	vuo.mutation.SetFilePath(s)
	return vuo
}

// SetSearchString sets the "search_string" field.
func (vuo *ValidationUpdateOne) SetSearchString(s string) *ValidationUpdateOne {
	vuo.mutation.SetSearchString(s)
	return vuo
}

// SetServiceName sets the "service_name" field.
func (vuo *ValidationUpdateOne) SetServiceName(s string) *ValidationUpdateOne {
	vuo.mutation.SetServiceName(s)
	return vuo
}

// SetServiceStatus sets the "service_status" field.
func (vuo *ValidationUpdateOne) SetServiceStatus(s string) *ValidationUpdateOne {
	vuo.mutation.SetServiceStatus(s)
	return vuo
}

// SetProcessName sets the "process_name" field.
func (vuo *ValidationUpdateOne) SetProcessName(s string) *ValidationUpdateOne {
	vuo.mutation.SetProcessName(s)
	return vuo
}

// SetValidationToAgentTaskID sets the "ValidationToAgentTask" edge to the AgentTask entity by ID.
func (vuo *ValidationUpdateOne) SetValidationToAgentTaskID(id uuid.UUID) *ValidationUpdateOne {
	vuo.mutation.SetValidationToAgentTaskID(id)
	return vuo
}

// SetNillableValidationToAgentTaskID sets the "ValidationToAgentTask" edge to the AgentTask entity by ID if the given value is not nil.
func (vuo *ValidationUpdateOne) SetNillableValidationToAgentTaskID(id *uuid.UUID) *ValidationUpdateOne {
	if id != nil {
		vuo = vuo.SetValidationToAgentTaskID(*id)
	}
	return vuo
}

// SetValidationToAgentTask sets the "ValidationToAgentTask" edge to the AgentTask entity.
func (vuo *ValidationUpdateOne) SetValidationToAgentTask(a *AgentTask) *ValidationUpdateOne {
	return vuo.SetValidationToAgentTaskID(a.ID)
}

// AddValidationToScriptIDs adds the "ValidationToScript" edge to the Script entity by IDs.
func (vuo *ValidationUpdateOne) AddValidationToScriptIDs(ids ...uuid.UUID) *ValidationUpdateOne {
	vuo.mutation.AddValidationToScriptIDs(ids...)
	return vuo
}

// AddValidationToScript adds the "ValidationToScript" edges to the Script entity.
func (vuo *ValidationUpdateOne) AddValidationToScript(s ...*Script) *ValidationUpdateOne {
	ids := make([]uuid.UUID, len(s))
	for i := range s {
		ids[i] = s[i].ID
	}
	return vuo.AddValidationToScriptIDs(ids...)
}

// AddValidationToEnvironmentIDs adds the "ValidationToEnvironment" edge to the Environment entity by IDs.
func (vuo *ValidationUpdateOne) AddValidationToEnvironmentIDs(ids ...uuid.UUID) *ValidationUpdateOne {
	vuo.mutation.AddValidationToEnvironmentIDs(ids...)
	return vuo
}

// AddValidationToEnvironment adds the "ValidationToEnvironment" edges to the Environment entity.
func (vuo *ValidationUpdateOne) AddValidationToEnvironment(e ...*Environment) *ValidationUpdateOne {
	ids := make([]uuid.UUID, len(e))
	for i := range e {
		ids[i] = e[i].ID
	}
	return vuo.AddValidationToEnvironmentIDs(ids...)
}

// Mutation returns the ValidationMutation object of the builder.
func (vuo *ValidationUpdateOne) Mutation() *ValidationMutation {
	return vuo.mutation
}

// ClearValidationToAgentTask clears the "ValidationToAgentTask" edge to the AgentTask entity.
func (vuo *ValidationUpdateOne) ClearValidationToAgentTask() *ValidationUpdateOne {
	vuo.mutation.ClearValidationToAgentTask()
	return vuo
}

// ClearValidationToScript clears all "ValidationToScript" edges to the Script entity.
func (vuo *ValidationUpdateOne) ClearValidationToScript() *ValidationUpdateOne {
	vuo.mutation.ClearValidationToScript()
	return vuo
}

// RemoveValidationToScriptIDs removes the "ValidationToScript" edge to Script entities by IDs.
func (vuo *ValidationUpdateOne) RemoveValidationToScriptIDs(ids ...uuid.UUID) *ValidationUpdateOne {
	vuo.mutation.RemoveValidationToScriptIDs(ids...)
	return vuo
}

// RemoveValidationToScript removes "ValidationToScript" edges to Script entities.
func (vuo *ValidationUpdateOne) RemoveValidationToScript(s ...*Script) *ValidationUpdateOne {
	ids := make([]uuid.UUID, len(s))
	for i := range s {
		ids[i] = s[i].ID
	}
	return vuo.RemoveValidationToScriptIDs(ids...)
}

// ClearValidationToEnvironment clears all "ValidationToEnvironment" edges to the Environment entity.
func (vuo *ValidationUpdateOne) ClearValidationToEnvironment() *ValidationUpdateOne {
	vuo.mutation.ClearValidationToEnvironment()
	return vuo
}

// RemoveValidationToEnvironmentIDs removes the "ValidationToEnvironment" edge to Environment entities by IDs.
func (vuo *ValidationUpdateOne) RemoveValidationToEnvironmentIDs(ids ...uuid.UUID) *ValidationUpdateOne {
	vuo.mutation.RemoveValidationToEnvironmentIDs(ids...)
	return vuo
}

// RemoveValidationToEnvironment removes "ValidationToEnvironment" edges to Environment entities.
func (vuo *ValidationUpdateOne) RemoveValidationToEnvironment(e ...*Environment) *ValidationUpdateOne {
	ids := make([]uuid.UUID, len(e))
	for i := range e {
		ids[i] = e[i].ID
	}
	return vuo.RemoveValidationToEnvironmentIDs(ids...)
}

// Select allows selecting one or more fields (columns) of the returned entity.
// The default is selecting all fields defined in the entity schema.
func (vuo *ValidationUpdateOne) Select(field string, fields ...string) *ValidationUpdateOne {
	vuo.fields = append([]string{field}, fields...)
	return vuo
}

// Save executes the query and returns the updated Validation entity.
func (vuo *ValidationUpdateOne) Save(ctx context.Context) (*Validation, error) {
	var (
		err  error
		node *Validation
	)
	if len(vuo.hooks) == 0 {
		if err = vuo.check(); err != nil {
			return nil, err
		}
		node, err = vuo.sqlSave(ctx)
	} else {
		var mut Mutator = MutateFunc(func(ctx context.Context, m Mutation) (Value, error) {
			mutation, ok := m.(*ValidationMutation)
			if !ok {
				return nil, fmt.Errorf("unexpected mutation type %T", m)
			}
			if err = vuo.check(); err != nil {
				return nil, err
			}
			vuo.mutation = mutation
			node, err = vuo.sqlSave(ctx)
			mutation.done = true
			return node, err
		})
		for i := len(vuo.hooks) - 1; i >= 0; i-- {
			if vuo.hooks[i] == nil {
				return nil, fmt.Errorf("ent: uninitialized hook (forgotten import ent/runtime?)")
			}
			mut = vuo.hooks[i](mut)
		}
		if _, err := mut.Mutate(ctx, vuo.mutation); err != nil {
			return nil, err
		}
	}
	return node, err
}

// SaveX is like Save, but panics if an error occurs.
func (vuo *ValidationUpdateOne) SaveX(ctx context.Context) *Validation {
	node, err := vuo.Save(ctx)
	if err != nil {
		panic(err)
	}
	return node
}

// Exec executes the query on the entity.
func (vuo *ValidationUpdateOne) Exec(ctx context.Context) error {
	_, err := vuo.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (vuo *ValidationUpdateOne) ExecX(ctx context.Context) {
	if err := vuo.Exec(ctx); err != nil {
		panic(err)
	}
}

// check runs all checks and user-defined validators on the builder.
func (vuo *ValidationUpdateOne) check() error {
	if v, ok := vuo.mutation.State(); ok {
		if err := validation.StateValidator(v); err != nil {
			return &ValidationError{Name: "state", err: fmt.Errorf(`ent: validator failed for field "Validation.state": %w`, err)}
		}
	}
	return nil
}

func (vuo *ValidationUpdateOne) sqlSave(ctx context.Context) (_node *Validation, err error) {
	_spec := &sqlgraph.UpdateSpec{
		Node: &sqlgraph.NodeSpec{
			Table:   validation.Table,
			Columns: validation.Columns,
			ID: &sqlgraph.FieldSpec{
				Type:   field.TypeUUID,
				Column: validation.FieldID,
			},
		},
	}
	id, ok := vuo.mutation.ID()
	if !ok {
		return nil, &ValidationError{Name: "id", err: errors.New(`ent: missing "Validation.id" for update`)}
	}
	_spec.Node.ID.Value = id
	if fields := vuo.fields; len(fields) > 0 {
		_spec.Node.Columns = make([]string, 0, len(fields))
		_spec.Node.Columns = append(_spec.Node.Columns, validation.FieldID)
		for _, f := range fields {
			if !validation.ValidColumn(f) {
				return nil, &ValidationError{Name: f, err: fmt.Errorf("ent: invalid field %q for query", f)}
			}
			if f != validation.FieldID {
				_spec.Node.Columns = append(_spec.Node.Columns, f)
			}
		}
	}
	if ps := vuo.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if value, ok := vuo.mutation.HclID(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Value:  value,
			Column: validation.FieldHclID,
		})
	}
	if value, ok := vuo.mutation.ValidationType(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Value:  value,
			Column: validation.FieldValidationType,
		})
	}
	if value, ok := vuo.mutation.Output(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Value:  value,
			Column: validation.FieldOutput,
		})
	}
	if value, ok := vuo.mutation.State(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeEnum,
			Value:  value,
			Column: validation.FieldState,
		})
	}
	if value, ok := vuo.mutation.ErrorMessage(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Value:  value,
			Column: validation.FieldErrorMessage,
		})
	}
	if value, ok := vuo.mutation.Hash(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Value:  value,
			Column: validation.FieldHash,
		})
	}
	if value, ok := vuo.mutation.Regex(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Value:  value,
			Column: validation.FieldRegex,
		})
	}
	if value, ok := vuo.mutation.IP(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Value:  value,
			Column: validation.FieldIP,
		})
	}
	if value, ok := vuo.mutation.Port(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeInt,
			Value:  value,
			Column: validation.FieldPort,
		})
	}
	if value, ok := vuo.mutation.AddedPort(); ok {
		_spec.Fields.Add = append(_spec.Fields.Add, &sqlgraph.FieldSpec{
			Type:   field.TypeInt,
			Value:  value,
			Column: validation.FieldPort,
		})
	}
	if value, ok := vuo.mutation.Hostname(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Value:  value,
			Column: validation.FieldHostname,
		})
	}
	if value, ok := vuo.mutation.Nameservers(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeJSON,
			Value:  value,
			Column: validation.FieldNameservers,
		})
	}
	if value, ok := vuo.mutation.PackageName(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Value:  value,
			Column: validation.FieldPackageName,
		})
	}
	if value, ok := vuo.mutation.Username(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Value:  value,
			Column: validation.FieldUsername,
		})
	}
	if value, ok := vuo.mutation.GroupName(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Value:  value,
			Column: validation.FieldGroupName,
		})
	}
	if value, ok := vuo.mutation.FilePath(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Value:  value,
			Column: validation.FieldFilePath,
		})
	}
	if value, ok := vuo.mutation.SearchString(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Value:  value,
			Column: validation.FieldSearchString,
		})
	}
	if value, ok := vuo.mutation.ServiceName(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Value:  value,
			Column: validation.FieldServiceName,
		})
	}
	if value, ok := vuo.mutation.ServiceStatus(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Value:  value,
			Column: validation.FieldServiceStatus,
		})
	}
	if value, ok := vuo.mutation.ProcessName(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Value:  value,
			Column: validation.FieldProcessName,
		})
	}
	if vuo.mutation.ValidationToAgentTaskCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2O,
			Inverse: true,
			Table:   validation.ValidationToAgentTaskTable,
			Columns: []string{validation.ValidationToAgentTaskColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeUUID,
					Column: agenttask.FieldID,
				},
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := vuo.mutation.ValidationToAgentTaskIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2O,
			Inverse: true,
			Table:   validation.ValidationToAgentTaskTable,
			Columns: []string{validation.ValidationToAgentTaskColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeUUID,
					Column: agenttask.FieldID,
				},
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	if vuo.mutation.ValidationToScriptCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: true,
			Table:   validation.ValidationToScriptTable,
			Columns: []string{validation.ValidationToScriptColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeUUID,
					Column: script.FieldID,
				},
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := vuo.mutation.RemovedValidationToScriptIDs(); len(nodes) > 0 && !vuo.mutation.ValidationToScriptCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: true,
			Table:   validation.ValidationToScriptTable,
			Columns: []string{validation.ValidationToScriptColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeUUID,
					Column: script.FieldID,
				},
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := vuo.mutation.ValidationToScriptIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: true,
			Table:   validation.ValidationToScriptTable,
			Columns: []string{validation.ValidationToScriptColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeUUID,
					Column: script.FieldID,
				},
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	if vuo.mutation.ValidationToEnvironmentCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2M,
			Inverse: true,
			Table:   validation.ValidationToEnvironmentTable,
			Columns: validation.ValidationToEnvironmentPrimaryKey,
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeUUID,
					Column: environment.FieldID,
				},
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := vuo.mutation.RemovedValidationToEnvironmentIDs(); len(nodes) > 0 && !vuo.mutation.ValidationToEnvironmentCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2M,
			Inverse: true,
			Table:   validation.ValidationToEnvironmentTable,
			Columns: validation.ValidationToEnvironmentPrimaryKey,
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
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := vuo.mutation.ValidationToEnvironmentIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2M,
			Inverse: true,
			Table:   validation.ValidationToEnvironmentTable,
			Columns: validation.ValidationToEnvironmentPrimaryKey,
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
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	_node = &Validation{config: vuo.config}
	_spec.Assign = _node.assignValues
	_spec.ScanValues = _node.scanValues
	if err = sqlgraph.UpdateNode(ctx, vuo.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{validation.Label}
		} else if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{err.Error(), err}
		}
		return nil, err
	}
	return _node, nil
}