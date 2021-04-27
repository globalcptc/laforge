// Code generated by entc, DO NOT EDIT.

package ent

import (
	"context"
	"fmt"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/schema/field"
	"github.com/gen0cide/laforge/ent/environment"
	"github.com/gen0cide/laforge/ent/finding"
	"github.com/gen0cide/laforge/ent/host"
	"github.com/gen0cide/laforge/ent/predicate"
	"github.com/gen0cide/laforge/ent/script"
	"github.com/gen0cide/laforge/ent/user"
	"github.com/google/uuid"
)

// FindingUpdate is the builder for updating Finding entities.
type FindingUpdate struct {
	config
	hooks    []Hook
	mutation *FindingMutation
}

// Where adds a new predicate for the FindingUpdate builder.
func (fu *FindingUpdate) Where(ps ...predicate.Finding) *FindingUpdate {
	fu.mutation.predicates = append(fu.mutation.predicates, ps...)
	return fu
}

// SetName sets the "name" field.
func (fu *FindingUpdate) SetName(s string) *FindingUpdate {
	fu.mutation.SetName(s)
	return fu
}

// SetDescription sets the "description" field.
func (fu *FindingUpdate) SetDescription(s string) *FindingUpdate {
	fu.mutation.SetDescription(s)
	return fu
}

// SetSeverity sets the "severity" field.
func (fu *FindingUpdate) SetSeverity(f finding.Severity) *FindingUpdate {
	fu.mutation.SetSeverity(f)
	return fu
}

// SetDifficulty sets the "difficulty" field.
func (fu *FindingUpdate) SetDifficulty(f finding.Difficulty) *FindingUpdate {
	fu.mutation.SetDifficulty(f)
	return fu
}

// SetTags sets the "tags" field.
func (fu *FindingUpdate) SetTags(m map[string]string) *FindingUpdate {
	fu.mutation.SetTags(m)
	return fu
}

// AddFindingToUserIDs adds the "FindingToUser" edge to the User entity by IDs.
func (fu *FindingUpdate) AddFindingToUserIDs(ids ...uuid.UUID) *FindingUpdate {
	fu.mutation.AddFindingToUserIDs(ids...)
	return fu
}

// AddFindingToUser adds the "FindingToUser" edges to the User entity.
func (fu *FindingUpdate) AddFindingToUser(u ...*User) *FindingUpdate {
	ids := make([]uuid.UUID, len(u))
	for i := range u {
		ids[i] = u[i].ID
	}
	return fu.AddFindingToUserIDs(ids...)
}

// SetFindingToHostID sets the "FindingToHost" edge to the Host entity by ID.
func (fu *FindingUpdate) SetFindingToHostID(id uuid.UUID) *FindingUpdate {
	fu.mutation.SetFindingToHostID(id)
	return fu
}

// SetNillableFindingToHostID sets the "FindingToHost" edge to the Host entity by ID if the given value is not nil.
func (fu *FindingUpdate) SetNillableFindingToHostID(id *uuid.UUID) *FindingUpdate {
	if id != nil {
		fu = fu.SetFindingToHostID(*id)
	}
	return fu
}

// SetFindingToHost sets the "FindingToHost" edge to the Host entity.
func (fu *FindingUpdate) SetFindingToHost(h *Host) *FindingUpdate {
	return fu.SetFindingToHostID(h.ID)
}

// SetFindingToScriptID sets the "FindingToScript" edge to the Script entity by ID.
func (fu *FindingUpdate) SetFindingToScriptID(id uuid.UUID) *FindingUpdate {
	fu.mutation.SetFindingToScriptID(id)
	return fu
}

// SetNillableFindingToScriptID sets the "FindingToScript" edge to the Script entity by ID if the given value is not nil.
func (fu *FindingUpdate) SetNillableFindingToScriptID(id *uuid.UUID) *FindingUpdate {
	if id != nil {
		fu = fu.SetFindingToScriptID(*id)
	}
	return fu
}

// SetFindingToScript sets the "FindingToScript" edge to the Script entity.
func (fu *FindingUpdate) SetFindingToScript(s *Script) *FindingUpdate {
	return fu.SetFindingToScriptID(s.ID)
}

// SetFindingToEnvironmentID sets the "FindingToEnvironment" edge to the Environment entity by ID.
func (fu *FindingUpdate) SetFindingToEnvironmentID(id uuid.UUID) *FindingUpdate {
	fu.mutation.SetFindingToEnvironmentID(id)
	return fu
}

// SetNillableFindingToEnvironmentID sets the "FindingToEnvironment" edge to the Environment entity by ID if the given value is not nil.
func (fu *FindingUpdate) SetNillableFindingToEnvironmentID(id *uuid.UUID) *FindingUpdate {
	if id != nil {
		fu = fu.SetFindingToEnvironmentID(*id)
	}
	return fu
}

// SetFindingToEnvironment sets the "FindingToEnvironment" edge to the Environment entity.
func (fu *FindingUpdate) SetFindingToEnvironment(e *Environment) *FindingUpdate {
	return fu.SetFindingToEnvironmentID(e.ID)
}

// Mutation returns the FindingMutation object of the builder.
func (fu *FindingUpdate) Mutation() *FindingMutation {
	return fu.mutation
}

// ClearFindingToUser clears all "FindingToUser" edges to the User entity.
func (fu *FindingUpdate) ClearFindingToUser() *FindingUpdate {
	fu.mutation.ClearFindingToUser()
	return fu
}

// RemoveFindingToUserIDs removes the "FindingToUser" edge to User entities by IDs.
func (fu *FindingUpdate) RemoveFindingToUserIDs(ids ...uuid.UUID) *FindingUpdate {
	fu.mutation.RemoveFindingToUserIDs(ids...)
	return fu
}

// RemoveFindingToUser removes "FindingToUser" edges to User entities.
func (fu *FindingUpdate) RemoveFindingToUser(u ...*User) *FindingUpdate {
	ids := make([]uuid.UUID, len(u))
	for i := range u {
		ids[i] = u[i].ID
	}
	return fu.RemoveFindingToUserIDs(ids...)
}

// ClearFindingToHost clears the "FindingToHost" edge to the Host entity.
func (fu *FindingUpdate) ClearFindingToHost() *FindingUpdate {
	fu.mutation.ClearFindingToHost()
	return fu
}

// ClearFindingToScript clears the "FindingToScript" edge to the Script entity.
func (fu *FindingUpdate) ClearFindingToScript() *FindingUpdate {
	fu.mutation.ClearFindingToScript()
	return fu
}

// ClearFindingToEnvironment clears the "FindingToEnvironment" edge to the Environment entity.
func (fu *FindingUpdate) ClearFindingToEnvironment() *FindingUpdate {
	fu.mutation.ClearFindingToEnvironment()
	return fu
}

// Save executes the query and returns the number of nodes affected by the update operation.
func (fu *FindingUpdate) Save(ctx context.Context) (int, error) {
	var (
		err      error
		affected int
	)
	if len(fu.hooks) == 0 {
		if err = fu.check(); err != nil {
			return 0, err
		}
		affected, err = fu.sqlSave(ctx)
	} else {
		var mut Mutator = MutateFunc(func(ctx context.Context, m Mutation) (Value, error) {
			mutation, ok := m.(*FindingMutation)
			if !ok {
				return nil, fmt.Errorf("unexpected mutation type %T", m)
			}
			if err = fu.check(); err != nil {
				return 0, err
			}
			fu.mutation = mutation
			affected, err = fu.sqlSave(ctx)
			mutation.done = true
			return affected, err
		})
		for i := len(fu.hooks) - 1; i >= 0; i-- {
			mut = fu.hooks[i](mut)
		}
		if _, err := mut.Mutate(ctx, fu.mutation); err != nil {
			return 0, err
		}
	}
	return affected, err
}

// SaveX is like Save, but panics if an error occurs.
func (fu *FindingUpdate) SaveX(ctx context.Context) int {
	affected, err := fu.Save(ctx)
	if err != nil {
		panic(err)
	}
	return affected
}

// Exec executes the query.
func (fu *FindingUpdate) Exec(ctx context.Context) error {
	_, err := fu.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (fu *FindingUpdate) ExecX(ctx context.Context) {
	if err := fu.Exec(ctx); err != nil {
		panic(err)
	}
}

// check runs all checks and user-defined validators on the builder.
func (fu *FindingUpdate) check() error {
	if v, ok := fu.mutation.Severity(); ok {
		if err := finding.SeverityValidator(v); err != nil {
			return &ValidationError{Name: "severity", err: fmt.Errorf("ent: validator failed for field \"severity\": %w", err)}
		}
	}
	if v, ok := fu.mutation.Difficulty(); ok {
		if err := finding.DifficultyValidator(v); err != nil {
			return &ValidationError{Name: "difficulty", err: fmt.Errorf("ent: validator failed for field \"difficulty\": %w", err)}
		}
	}
	return nil
}

func (fu *FindingUpdate) sqlSave(ctx context.Context) (n int, err error) {
	_spec := &sqlgraph.UpdateSpec{
		Node: &sqlgraph.NodeSpec{
			Table:   finding.Table,
			Columns: finding.Columns,
			ID: &sqlgraph.FieldSpec{
				Type:   field.TypeUUID,
				Column: finding.FieldID,
			},
		},
	}
	if ps := fu.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if value, ok := fu.mutation.Name(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Value:  value,
			Column: finding.FieldName,
		})
	}
	if value, ok := fu.mutation.Description(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Value:  value,
			Column: finding.FieldDescription,
		})
	}
	if value, ok := fu.mutation.Severity(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeEnum,
			Value:  value,
			Column: finding.FieldSeverity,
		})
	}
	if value, ok := fu.mutation.Difficulty(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeEnum,
			Value:  value,
			Column: finding.FieldDifficulty,
		})
	}
	if value, ok := fu.mutation.Tags(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeJSON,
			Value:  value,
			Column: finding.FieldTags,
		})
	}
	if fu.mutation.FindingToUserCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: false,
			Table:   finding.FindingToUserTable,
			Columns: []string{finding.FindingToUserColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeUUID,
					Column: user.FieldID,
				},
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := fu.mutation.RemovedFindingToUserIDs(); len(nodes) > 0 && !fu.mutation.FindingToUserCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: false,
			Table:   finding.FindingToUserTable,
			Columns: []string{finding.FindingToUserColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeUUID,
					Column: user.FieldID,
				},
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := fu.mutation.FindingToUserIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: false,
			Table:   finding.FindingToUserTable,
			Columns: []string{finding.FindingToUserColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeUUID,
					Column: user.FieldID,
				},
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	if fu.mutation.FindingToHostCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: false,
			Table:   finding.FindingToHostTable,
			Columns: []string{finding.FindingToHostColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeUUID,
					Column: host.FieldID,
				},
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := fu.mutation.FindingToHostIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: false,
			Table:   finding.FindingToHostTable,
			Columns: []string{finding.FindingToHostColumn},
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
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	if fu.mutation.FindingToScriptCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: true,
			Table:   finding.FindingToScriptTable,
			Columns: []string{finding.FindingToScriptColumn},
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
	if nodes := fu.mutation.FindingToScriptIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: true,
			Table:   finding.FindingToScriptTable,
			Columns: []string{finding.FindingToScriptColumn},
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
	if fu.mutation.FindingToEnvironmentCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: true,
			Table:   finding.FindingToEnvironmentTable,
			Columns: []string{finding.FindingToEnvironmentColumn},
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
	if nodes := fu.mutation.FindingToEnvironmentIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: true,
			Table:   finding.FindingToEnvironmentTable,
			Columns: []string{finding.FindingToEnvironmentColumn},
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
	if n, err = sqlgraph.UpdateNodes(ctx, fu.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{finding.Label}
		} else if cerr, ok := isSQLConstraintError(err); ok {
			err = cerr
		}
		return 0, err
	}
	return n, nil
}

// FindingUpdateOne is the builder for updating a single Finding entity.
type FindingUpdateOne struct {
	config
	fields   []string
	hooks    []Hook
	mutation *FindingMutation
}

// SetName sets the "name" field.
func (fuo *FindingUpdateOne) SetName(s string) *FindingUpdateOne {
	fuo.mutation.SetName(s)
	return fuo
}

// SetDescription sets the "description" field.
func (fuo *FindingUpdateOne) SetDescription(s string) *FindingUpdateOne {
	fuo.mutation.SetDescription(s)
	return fuo
}

// SetSeverity sets the "severity" field.
func (fuo *FindingUpdateOne) SetSeverity(f finding.Severity) *FindingUpdateOne {
	fuo.mutation.SetSeverity(f)
	return fuo
}

// SetDifficulty sets the "difficulty" field.
func (fuo *FindingUpdateOne) SetDifficulty(f finding.Difficulty) *FindingUpdateOne {
	fuo.mutation.SetDifficulty(f)
	return fuo
}

// SetTags sets the "tags" field.
func (fuo *FindingUpdateOne) SetTags(m map[string]string) *FindingUpdateOne {
	fuo.mutation.SetTags(m)
	return fuo
}

// AddFindingToUserIDs adds the "FindingToUser" edge to the User entity by IDs.
func (fuo *FindingUpdateOne) AddFindingToUserIDs(ids ...uuid.UUID) *FindingUpdateOne {
	fuo.mutation.AddFindingToUserIDs(ids...)
	return fuo
}

// AddFindingToUser adds the "FindingToUser" edges to the User entity.
func (fuo *FindingUpdateOne) AddFindingToUser(u ...*User) *FindingUpdateOne {
	ids := make([]uuid.UUID, len(u))
	for i := range u {
		ids[i] = u[i].ID
	}
	return fuo.AddFindingToUserIDs(ids...)
}

// SetFindingToHostID sets the "FindingToHost" edge to the Host entity by ID.
func (fuo *FindingUpdateOne) SetFindingToHostID(id uuid.UUID) *FindingUpdateOne {
	fuo.mutation.SetFindingToHostID(id)
	return fuo
}

// SetNillableFindingToHostID sets the "FindingToHost" edge to the Host entity by ID if the given value is not nil.
func (fuo *FindingUpdateOne) SetNillableFindingToHostID(id *uuid.UUID) *FindingUpdateOne {
	if id != nil {
		fuo = fuo.SetFindingToHostID(*id)
	}
	return fuo
}

// SetFindingToHost sets the "FindingToHost" edge to the Host entity.
func (fuo *FindingUpdateOne) SetFindingToHost(h *Host) *FindingUpdateOne {
	return fuo.SetFindingToHostID(h.ID)
}

// SetFindingToScriptID sets the "FindingToScript" edge to the Script entity by ID.
func (fuo *FindingUpdateOne) SetFindingToScriptID(id uuid.UUID) *FindingUpdateOne {
	fuo.mutation.SetFindingToScriptID(id)
	return fuo
}

// SetNillableFindingToScriptID sets the "FindingToScript" edge to the Script entity by ID if the given value is not nil.
func (fuo *FindingUpdateOne) SetNillableFindingToScriptID(id *uuid.UUID) *FindingUpdateOne {
	if id != nil {
		fuo = fuo.SetFindingToScriptID(*id)
	}
	return fuo
}

// SetFindingToScript sets the "FindingToScript" edge to the Script entity.
func (fuo *FindingUpdateOne) SetFindingToScript(s *Script) *FindingUpdateOne {
	return fuo.SetFindingToScriptID(s.ID)
}

// SetFindingToEnvironmentID sets the "FindingToEnvironment" edge to the Environment entity by ID.
func (fuo *FindingUpdateOne) SetFindingToEnvironmentID(id uuid.UUID) *FindingUpdateOne {
	fuo.mutation.SetFindingToEnvironmentID(id)
	return fuo
}

// SetNillableFindingToEnvironmentID sets the "FindingToEnvironment" edge to the Environment entity by ID if the given value is not nil.
func (fuo *FindingUpdateOne) SetNillableFindingToEnvironmentID(id *uuid.UUID) *FindingUpdateOne {
	if id != nil {
		fuo = fuo.SetFindingToEnvironmentID(*id)
	}
	return fuo
}

// SetFindingToEnvironment sets the "FindingToEnvironment" edge to the Environment entity.
func (fuo *FindingUpdateOne) SetFindingToEnvironment(e *Environment) *FindingUpdateOne {
	return fuo.SetFindingToEnvironmentID(e.ID)
}

// Mutation returns the FindingMutation object of the builder.
func (fuo *FindingUpdateOne) Mutation() *FindingMutation {
	return fuo.mutation
}

// ClearFindingToUser clears all "FindingToUser" edges to the User entity.
func (fuo *FindingUpdateOne) ClearFindingToUser() *FindingUpdateOne {
	fuo.mutation.ClearFindingToUser()
	return fuo
}

// RemoveFindingToUserIDs removes the "FindingToUser" edge to User entities by IDs.
func (fuo *FindingUpdateOne) RemoveFindingToUserIDs(ids ...uuid.UUID) *FindingUpdateOne {
	fuo.mutation.RemoveFindingToUserIDs(ids...)
	return fuo
}

// RemoveFindingToUser removes "FindingToUser" edges to User entities.
func (fuo *FindingUpdateOne) RemoveFindingToUser(u ...*User) *FindingUpdateOne {
	ids := make([]uuid.UUID, len(u))
	for i := range u {
		ids[i] = u[i].ID
	}
	return fuo.RemoveFindingToUserIDs(ids...)
}

// ClearFindingToHost clears the "FindingToHost" edge to the Host entity.
func (fuo *FindingUpdateOne) ClearFindingToHost() *FindingUpdateOne {
	fuo.mutation.ClearFindingToHost()
	return fuo
}

// ClearFindingToScript clears the "FindingToScript" edge to the Script entity.
func (fuo *FindingUpdateOne) ClearFindingToScript() *FindingUpdateOne {
	fuo.mutation.ClearFindingToScript()
	return fuo
}

// ClearFindingToEnvironment clears the "FindingToEnvironment" edge to the Environment entity.
func (fuo *FindingUpdateOne) ClearFindingToEnvironment() *FindingUpdateOne {
	fuo.mutation.ClearFindingToEnvironment()
	return fuo
}

// Select allows selecting one or more fields (columns) of the returned entity.
// The default is selecting all fields defined in the entity schema.
func (fuo *FindingUpdateOne) Select(field string, fields ...string) *FindingUpdateOne {
	fuo.fields = append([]string{field}, fields...)
	return fuo
}

// Save executes the query and returns the updated Finding entity.
func (fuo *FindingUpdateOne) Save(ctx context.Context) (*Finding, error) {
	var (
		err  error
		node *Finding
	)
	if len(fuo.hooks) == 0 {
		if err = fuo.check(); err != nil {
			return nil, err
		}
		node, err = fuo.sqlSave(ctx)
	} else {
		var mut Mutator = MutateFunc(func(ctx context.Context, m Mutation) (Value, error) {
			mutation, ok := m.(*FindingMutation)
			if !ok {
				return nil, fmt.Errorf("unexpected mutation type %T", m)
			}
			if err = fuo.check(); err != nil {
				return nil, err
			}
			fuo.mutation = mutation
			node, err = fuo.sqlSave(ctx)
			mutation.done = true
			return node, err
		})
		for i := len(fuo.hooks) - 1; i >= 0; i-- {
			mut = fuo.hooks[i](mut)
		}
		if _, err := mut.Mutate(ctx, fuo.mutation); err != nil {
			return nil, err
		}
	}
	return node, err
}

// SaveX is like Save, but panics if an error occurs.
func (fuo *FindingUpdateOne) SaveX(ctx context.Context) *Finding {
	node, err := fuo.Save(ctx)
	if err != nil {
		panic(err)
	}
	return node
}

// Exec executes the query on the entity.
func (fuo *FindingUpdateOne) Exec(ctx context.Context) error {
	_, err := fuo.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (fuo *FindingUpdateOne) ExecX(ctx context.Context) {
	if err := fuo.Exec(ctx); err != nil {
		panic(err)
	}
}

// check runs all checks and user-defined validators on the builder.
func (fuo *FindingUpdateOne) check() error {
	if v, ok := fuo.mutation.Severity(); ok {
		if err := finding.SeverityValidator(v); err != nil {
			return &ValidationError{Name: "severity", err: fmt.Errorf("ent: validator failed for field \"severity\": %w", err)}
		}
	}
	if v, ok := fuo.mutation.Difficulty(); ok {
		if err := finding.DifficultyValidator(v); err != nil {
			return &ValidationError{Name: "difficulty", err: fmt.Errorf("ent: validator failed for field \"difficulty\": %w", err)}
		}
	}
	return nil
}

func (fuo *FindingUpdateOne) sqlSave(ctx context.Context) (_node *Finding, err error) {
	_spec := &sqlgraph.UpdateSpec{
		Node: &sqlgraph.NodeSpec{
			Table:   finding.Table,
			Columns: finding.Columns,
			ID: &sqlgraph.FieldSpec{
				Type:   field.TypeUUID,
				Column: finding.FieldID,
			},
		},
	}
	id, ok := fuo.mutation.ID()
	if !ok {
		return nil, &ValidationError{Name: "ID", err: fmt.Errorf("missing Finding.ID for update")}
	}
	_spec.Node.ID.Value = id
	if fields := fuo.fields; len(fields) > 0 {
		_spec.Node.Columns = make([]string, 0, len(fields))
		_spec.Node.Columns = append(_spec.Node.Columns, finding.FieldID)
		for _, f := range fields {
			if !finding.ValidColumn(f) {
				return nil, &ValidationError{Name: f, err: fmt.Errorf("ent: invalid field %q for query", f)}
			}
			if f != finding.FieldID {
				_spec.Node.Columns = append(_spec.Node.Columns, f)
			}
		}
	}
	if ps := fuo.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if value, ok := fuo.mutation.Name(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Value:  value,
			Column: finding.FieldName,
		})
	}
	if value, ok := fuo.mutation.Description(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Value:  value,
			Column: finding.FieldDescription,
		})
	}
	if value, ok := fuo.mutation.Severity(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeEnum,
			Value:  value,
			Column: finding.FieldSeverity,
		})
	}
	if value, ok := fuo.mutation.Difficulty(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeEnum,
			Value:  value,
			Column: finding.FieldDifficulty,
		})
	}
	if value, ok := fuo.mutation.Tags(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeJSON,
			Value:  value,
			Column: finding.FieldTags,
		})
	}
	if fuo.mutation.FindingToUserCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: false,
			Table:   finding.FindingToUserTable,
			Columns: []string{finding.FindingToUserColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeUUID,
					Column: user.FieldID,
				},
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := fuo.mutation.RemovedFindingToUserIDs(); len(nodes) > 0 && !fuo.mutation.FindingToUserCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: false,
			Table:   finding.FindingToUserTable,
			Columns: []string{finding.FindingToUserColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeUUID,
					Column: user.FieldID,
				},
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := fuo.mutation.FindingToUserIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: false,
			Table:   finding.FindingToUserTable,
			Columns: []string{finding.FindingToUserColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeUUID,
					Column: user.FieldID,
				},
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	if fuo.mutation.FindingToHostCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: false,
			Table:   finding.FindingToHostTable,
			Columns: []string{finding.FindingToHostColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeUUID,
					Column: host.FieldID,
				},
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := fuo.mutation.FindingToHostIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: false,
			Table:   finding.FindingToHostTable,
			Columns: []string{finding.FindingToHostColumn},
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
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	if fuo.mutation.FindingToScriptCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: true,
			Table:   finding.FindingToScriptTable,
			Columns: []string{finding.FindingToScriptColumn},
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
	if nodes := fuo.mutation.FindingToScriptIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: true,
			Table:   finding.FindingToScriptTable,
			Columns: []string{finding.FindingToScriptColumn},
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
	if fuo.mutation.FindingToEnvironmentCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: true,
			Table:   finding.FindingToEnvironmentTable,
			Columns: []string{finding.FindingToEnvironmentColumn},
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
	if nodes := fuo.mutation.FindingToEnvironmentIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: true,
			Table:   finding.FindingToEnvironmentTable,
			Columns: []string{finding.FindingToEnvironmentColumn},
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
	_node = &Finding{config: fuo.config}
	_spec.Assign = _node.assignValues
	_spec.ScanValues = _node.scanValues
	if err = sqlgraph.UpdateNode(ctx, fuo.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{finding.Label}
		} else if cerr, ok := isSQLConstraintError(err); ok {
			err = cerr
		}
		return nil, err
	}
	return _node, nil
}
