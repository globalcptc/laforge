// Code generated by ent, DO NOT EDIT.

package ent

import (
	"context"
	"errors"
	"fmt"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/schema/field"
	"github.com/gen0cide/laforge/ent/environment"
	"github.com/gen0cide/laforge/ent/fileextract"
	"github.com/gen0cide/laforge/ent/predicate"
	"github.com/google/uuid"
)

// FileExtractUpdate is the builder for updating FileExtract entities.
type FileExtractUpdate struct {
	config
	hooks    []Hook
	mutation *FileExtractMutation
}

// Where appends a list predicates to the FileExtractUpdate builder.
func (feu *FileExtractUpdate) Where(ps ...predicate.FileExtract) *FileExtractUpdate {
	feu.mutation.Where(ps...)
	return feu
}

// SetHCLID sets the "hcl_id" field.
func (feu *FileExtractUpdate) SetHCLID(s string) *FileExtractUpdate {
	feu.mutation.SetHCLID(s)
	return feu
}

// SetNillableHCLID sets the "hcl_id" field if the given value is not nil.
func (feu *FileExtractUpdate) SetNillableHCLID(s *string) *FileExtractUpdate {
	if s != nil {
		feu.SetHCLID(*s)
	}
	return feu
}

// SetSource sets the "source" field.
func (feu *FileExtractUpdate) SetSource(s string) *FileExtractUpdate {
	feu.mutation.SetSource(s)
	return feu
}

// SetNillableSource sets the "source" field if the given value is not nil.
func (feu *FileExtractUpdate) SetNillableSource(s *string) *FileExtractUpdate {
	if s != nil {
		feu.SetSource(*s)
	}
	return feu
}

// SetDestination sets the "destination" field.
func (feu *FileExtractUpdate) SetDestination(s string) *FileExtractUpdate {
	feu.mutation.SetDestination(s)
	return feu
}

// SetNillableDestination sets the "destination" field if the given value is not nil.
func (feu *FileExtractUpdate) SetNillableDestination(s *string) *FileExtractUpdate {
	if s != nil {
		feu.SetDestination(*s)
	}
	return feu
}

// SetType sets the "type" field.
func (feu *FileExtractUpdate) SetType(s string) *FileExtractUpdate {
	feu.mutation.SetType(s)
	return feu
}

// SetNillableType sets the "type" field if the given value is not nil.
func (feu *FileExtractUpdate) SetNillableType(s *string) *FileExtractUpdate {
	if s != nil {
		feu.SetType(*s)
	}
	return feu
}

// SetTags sets the "tags" field.
func (feu *FileExtractUpdate) SetTags(m map[string]string) *FileExtractUpdate {
	feu.mutation.SetTags(m)
	return feu
}

// SetEnvironmentID sets the "Environment" edge to the Environment entity by ID.
func (feu *FileExtractUpdate) SetEnvironmentID(id uuid.UUID) *FileExtractUpdate {
	feu.mutation.SetEnvironmentID(id)
	return feu
}

// SetNillableEnvironmentID sets the "Environment" edge to the Environment entity by ID if the given value is not nil.
func (feu *FileExtractUpdate) SetNillableEnvironmentID(id *uuid.UUID) *FileExtractUpdate {
	if id != nil {
		feu = feu.SetEnvironmentID(*id)
	}
	return feu
}

// SetEnvironment sets the "Environment" edge to the Environment entity.
func (feu *FileExtractUpdate) SetEnvironment(e *Environment) *FileExtractUpdate {
	return feu.SetEnvironmentID(e.ID)
}

// Mutation returns the FileExtractMutation object of the builder.
func (feu *FileExtractUpdate) Mutation() *FileExtractMutation {
	return feu.mutation
}

// ClearEnvironment clears the "Environment" edge to the Environment entity.
func (feu *FileExtractUpdate) ClearEnvironment() *FileExtractUpdate {
	feu.mutation.ClearEnvironment()
	return feu
}

// Save executes the query and returns the number of nodes affected by the update operation.
func (feu *FileExtractUpdate) Save(ctx context.Context) (int, error) {
	return withHooks(ctx, feu.sqlSave, feu.mutation, feu.hooks)
}

// SaveX is like Save, but panics if an error occurs.
func (feu *FileExtractUpdate) SaveX(ctx context.Context) int {
	affected, err := feu.Save(ctx)
	if err != nil {
		panic(err)
	}
	return affected
}

// Exec executes the query.
func (feu *FileExtractUpdate) Exec(ctx context.Context) error {
	_, err := feu.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (feu *FileExtractUpdate) ExecX(ctx context.Context) {
	if err := feu.Exec(ctx); err != nil {
		panic(err)
	}
}

func (feu *FileExtractUpdate) sqlSave(ctx context.Context) (n int, err error) {
	_spec := sqlgraph.NewUpdateSpec(fileextract.Table, fileextract.Columns, sqlgraph.NewFieldSpec(fileextract.FieldID, field.TypeUUID))
	if ps := feu.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if value, ok := feu.mutation.HCLID(); ok {
		_spec.SetField(fileextract.FieldHCLID, field.TypeString, value)
	}
	if value, ok := feu.mutation.Source(); ok {
		_spec.SetField(fileextract.FieldSource, field.TypeString, value)
	}
	if value, ok := feu.mutation.Destination(); ok {
		_spec.SetField(fileextract.FieldDestination, field.TypeString, value)
	}
	if value, ok := feu.mutation.GetType(); ok {
		_spec.SetField(fileextract.FieldType, field.TypeString, value)
	}
	if value, ok := feu.mutation.Tags(); ok {
		_spec.SetField(fileextract.FieldTags, field.TypeJSON, value)
	}
	if feu.mutation.EnvironmentCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: true,
			Table:   fileextract.EnvironmentTable,
			Columns: []string{fileextract.EnvironmentColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(environment.FieldID, field.TypeUUID),
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := feu.mutation.EnvironmentIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: true,
			Table:   fileextract.EnvironmentTable,
			Columns: []string{fileextract.EnvironmentColumn},
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
	if n, err = sqlgraph.UpdateNodes(ctx, feu.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{fileextract.Label}
		} else if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return 0, err
	}
	feu.mutation.done = true
	return n, nil
}

// FileExtractUpdateOne is the builder for updating a single FileExtract entity.
type FileExtractUpdateOne struct {
	config
	fields   []string
	hooks    []Hook
	mutation *FileExtractMutation
}

// SetHCLID sets the "hcl_id" field.
func (feuo *FileExtractUpdateOne) SetHCLID(s string) *FileExtractUpdateOne {
	feuo.mutation.SetHCLID(s)
	return feuo
}

// SetNillableHCLID sets the "hcl_id" field if the given value is not nil.
func (feuo *FileExtractUpdateOne) SetNillableHCLID(s *string) *FileExtractUpdateOne {
	if s != nil {
		feuo.SetHCLID(*s)
	}
	return feuo
}

// SetSource sets the "source" field.
func (feuo *FileExtractUpdateOne) SetSource(s string) *FileExtractUpdateOne {
	feuo.mutation.SetSource(s)
	return feuo
}

// SetNillableSource sets the "source" field if the given value is not nil.
func (feuo *FileExtractUpdateOne) SetNillableSource(s *string) *FileExtractUpdateOne {
	if s != nil {
		feuo.SetSource(*s)
	}
	return feuo
}

// SetDestination sets the "destination" field.
func (feuo *FileExtractUpdateOne) SetDestination(s string) *FileExtractUpdateOne {
	feuo.mutation.SetDestination(s)
	return feuo
}

// SetNillableDestination sets the "destination" field if the given value is not nil.
func (feuo *FileExtractUpdateOne) SetNillableDestination(s *string) *FileExtractUpdateOne {
	if s != nil {
		feuo.SetDestination(*s)
	}
	return feuo
}

// SetType sets the "type" field.
func (feuo *FileExtractUpdateOne) SetType(s string) *FileExtractUpdateOne {
	feuo.mutation.SetType(s)
	return feuo
}

// SetNillableType sets the "type" field if the given value is not nil.
func (feuo *FileExtractUpdateOne) SetNillableType(s *string) *FileExtractUpdateOne {
	if s != nil {
		feuo.SetType(*s)
	}
	return feuo
}

// SetTags sets the "tags" field.
func (feuo *FileExtractUpdateOne) SetTags(m map[string]string) *FileExtractUpdateOne {
	feuo.mutation.SetTags(m)
	return feuo
}

// SetEnvironmentID sets the "Environment" edge to the Environment entity by ID.
func (feuo *FileExtractUpdateOne) SetEnvironmentID(id uuid.UUID) *FileExtractUpdateOne {
	feuo.mutation.SetEnvironmentID(id)
	return feuo
}

// SetNillableEnvironmentID sets the "Environment" edge to the Environment entity by ID if the given value is not nil.
func (feuo *FileExtractUpdateOne) SetNillableEnvironmentID(id *uuid.UUID) *FileExtractUpdateOne {
	if id != nil {
		feuo = feuo.SetEnvironmentID(*id)
	}
	return feuo
}

// SetEnvironment sets the "Environment" edge to the Environment entity.
func (feuo *FileExtractUpdateOne) SetEnvironment(e *Environment) *FileExtractUpdateOne {
	return feuo.SetEnvironmentID(e.ID)
}

// Mutation returns the FileExtractMutation object of the builder.
func (feuo *FileExtractUpdateOne) Mutation() *FileExtractMutation {
	return feuo.mutation
}

// ClearEnvironment clears the "Environment" edge to the Environment entity.
func (feuo *FileExtractUpdateOne) ClearEnvironment() *FileExtractUpdateOne {
	feuo.mutation.ClearEnvironment()
	return feuo
}

// Where appends a list predicates to the FileExtractUpdate builder.
func (feuo *FileExtractUpdateOne) Where(ps ...predicate.FileExtract) *FileExtractUpdateOne {
	feuo.mutation.Where(ps...)
	return feuo
}

// Select allows selecting one or more fields (columns) of the returned entity.
// The default is selecting all fields defined in the entity schema.
func (feuo *FileExtractUpdateOne) Select(field string, fields ...string) *FileExtractUpdateOne {
	feuo.fields = append([]string{field}, fields...)
	return feuo
}

// Save executes the query and returns the updated FileExtract entity.
func (feuo *FileExtractUpdateOne) Save(ctx context.Context) (*FileExtract, error) {
	return withHooks(ctx, feuo.sqlSave, feuo.mutation, feuo.hooks)
}

// SaveX is like Save, but panics if an error occurs.
func (feuo *FileExtractUpdateOne) SaveX(ctx context.Context) *FileExtract {
	node, err := feuo.Save(ctx)
	if err != nil {
		panic(err)
	}
	return node
}

// Exec executes the query on the entity.
func (feuo *FileExtractUpdateOne) Exec(ctx context.Context) error {
	_, err := feuo.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (feuo *FileExtractUpdateOne) ExecX(ctx context.Context) {
	if err := feuo.Exec(ctx); err != nil {
		panic(err)
	}
}

func (feuo *FileExtractUpdateOne) sqlSave(ctx context.Context) (_node *FileExtract, err error) {
	_spec := sqlgraph.NewUpdateSpec(fileextract.Table, fileextract.Columns, sqlgraph.NewFieldSpec(fileextract.FieldID, field.TypeUUID))
	id, ok := feuo.mutation.ID()
	if !ok {
		return nil, &ValidationError{Name: "id", err: errors.New(`ent: missing "FileExtract.id" for update`)}
	}
	_spec.Node.ID.Value = id
	if fields := feuo.fields; len(fields) > 0 {
		_spec.Node.Columns = make([]string, 0, len(fields))
		_spec.Node.Columns = append(_spec.Node.Columns, fileextract.FieldID)
		for _, f := range fields {
			if !fileextract.ValidColumn(f) {
				return nil, &ValidationError{Name: f, err: fmt.Errorf("ent: invalid field %q for query", f)}
			}
			if f != fileextract.FieldID {
				_spec.Node.Columns = append(_spec.Node.Columns, f)
			}
		}
	}
	if ps := feuo.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if value, ok := feuo.mutation.HCLID(); ok {
		_spec.SetField(fileextract.FieldHCLID, field.TypeString, value)
	}
	if value, ok := feuo.mutation.Source(); ok {
		_spec.SetField(fileextract.FieldSource, field.TypeString, value)
	}
	if value, ok := feuo.mutation.Destination(); ok {
		_spec.SetField(fileextract.FieldDestination, field.TypeString, value)
	}
	if value, ok := feuo.mutation.GetType(); ok {
		_spec.SetField(fileextract.FieldType, field.TypeString, value)
	}
	if value, ok := feuo.mutation.Tags(); ok {
		_spec.SetField(fileextract.FieldTags, field.TypeJSON, value)
	}
	if feuo.mutation.EnvironmentCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: true,
			Table:   fileextract.EnvironmentTable,
			Columns: []string{fileextract.EnvironmentColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(environment.FieldID, field.TypeUUID),
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := feuo.mutation.EnvironmentIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: true,
			Table:   fileextract.EnvironmentTable,
			Columns: []string{fileextract.EnvironmentColumn},
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
	_node = &FileExtract{config: feuo.config}
	_spec.Assign = _node.assignValues
	_spec.ScanValues = _node.scanValues
	if err = sqlgraph.UpdateNode(ctx, feuo.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{fileextract.Label}
		} else if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return nil, err
	}
	feuo.mutation.done = true
	return _node, nil
}
