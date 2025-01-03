// Code generated by ent, DO NOT EDIT.

package ent

import (
	"context"
	"errors"
	"fmt"

	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/schema/field"
	"github.com/gen0cide/laforge/ent/environment"
	"github.com/gen0cide/laforge/ent/fileextract"
	"github.com/google/uuid"
)

// FileExtractCreate is the builder for creating a FileExtract entity.
type FileExtractCreate struct {
	config
	mutation *FileExtractMutation
	hooks    []Hook
}

// SetHCLID sets the "hcl_id" field.
func (fec *FileExtractCreate) SetHCLID(s string) *FileExtractCreate {
	fec.mutation.SetHCLID(s)
	return fec
}

// SetSource sets the "source" field.
func (fec *FileExtractCreate) SetSource(s string) *FileExtractCreate {
	fec.mutation.SetSource(s)
	return fec
}

// SetDestination sets the "destination" field.
func (fec *FileExtractCreate) SetDestination(s string) *FileExtractCreate {
	fec.mutation.SetDestination(s)
	return fec
}

// SetType sets the "type" field.
func (fec *FileExtractCreate) SetType(s string) *FileExtractCreate {
	fec.mutation.SetType(s)
	return fec
}

// SetTags sets the "tags" field.
func (fec *FileExtractCreate) SetTags(m map[string]string) *FileExtractCreate {
	fec.mutation.SetTags(m)
	return fec
}

// SetID sets the "id" field.
func (fec *FileExtractCreate) SetID(u uuid.UUID) *FileExtractCreate {
	fec.mutation.SetID(u)
	return fec
}

// SetNillableID sets the "id" field if the given value is not nil.
func (fec *FileExtractCreate) SetNillableID(u *uuid.UUID) *FileExtractCreate {
	if u != nil {
		fec.SetID(*u)
	}
	return fec
}

// SetEnvironmentID sets the "Environment" edge to the Environment entity by ID.
func (fec *FileExtractCreate) SetEnvironmentID(id uuid.UUID) *FileExtractCreate {
	fec.mutation.SetEnvironmentID(id)
	return fec
}

// SetNillableEnvironmentID sets the "Environment" edge to the Environment entity by ID if the given value is not nil.
func (fec *FileExtractCreate) SetNillableEnvironmentID(id *uuid.UUID) *FileExtractCreate {
	if id != nil {
		fec = fec.SetEnvironmentID(*id)
	}
	return fec
}

// SetEnvironment sets the "Environment" edge to the Environment entity.
func (fec *FileExtractCreate) SetEnvironment(e *Environment) *FileExtractCreate {
	return fec.SetEnvironmentID(e.ID)
}

// Mutation returns the FileExtractMutation object of the builder.
func (fec *FileExtractCreate) Mutation() *FileExtractMutation {
	return fec.mutation
}

// Save creates the FileExtract in the database.
func (fec *FileExtractCreate) Save(ctx context.Context) (*FileExtract, error) {
	fec.defaults()
	return withHooks(ctx, fec.sqlSave, fec.mutation, fec.hooks)
}

// SaveX calls Save and panics if Save returns an error.
func (fec *FileExtractCreate) SaveX(ctx context.Context) *FileExtract {
	v, err := fec.Save(ctx)
	if err != nil {
		panic(err)
	}
	return v
}

// Exec executes the query.
func (fec *FileExtractCreate) Exec(ctx context.Context) error {
	_, err := fec.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (fec *FileExtractCreate) ExecX(ctx context.Context) {
	if err := fec.Exec(ctx); err != nil {
		panic(err)
	}
}

// defaults sets the default values of the builder before save.
func (fec *FileExtractCreate) defaults() {
	if _, ok := fec.mutation.ID(); !ok {
		v := fileextract.DefaultID()
		fec.mutation.SetID(v)
	}
}

// check runs all checks and user-defined validators on the builder.
func (fec *FileExtractCreate) check() error {
	if _, ok := fec.mutation.HCLID(); !ok {
		return &ValidationError{Name: "hcl_id", err: errors.New(`ent: missing required field "FileExtract.hcl_id"`)}
	}
	if _, ok := fec.mutation.Source(); !ok {
		return &ValidationError{Name: "source", err: errors.New(`ent: missing required field "FileExtract.source"`)}
	}
	if _, ok := fec.mutation.Destination(); !ok {
		return &ValidationError{Name: "destination", err: errors.New(`ent: missing required field "FileExtract.destination"`)}
	}
	if _, ok := fec.mutation.GetType(); !ok {
		return &ValidationError{Name: "type", err: errors.New(`ent: missing required field "FileExtract.type"`)}
	}
	if _, ok := fec.mutation.Tags(); !ok {
		return &ValidationError{Name: "tags", err: errors.New(`ent: missing required field "FileExtract.tags"`)}
	}
	return nil
}

func (fec *FileExtractCreate) sqlSave(ctx context.Context) (*FileExtract, error) {
	if err := fec.check(); err != nil {
		return nil, err
	}
	_node, _spec := fec.createSpec()
	if err := sqlgraph.CreateNode(ctx, fec.driver, _spec); err != nil {
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
	fec.mutation.id = &_node.ID
	fec.mutation.done = true
	return _node, nil
}

func (fec *FileExtractCreate) createSpec() (*FileExtract, *sqlgraph.CreateSpec) {
	var (
		_node = &FileExtract{config: fec.config}
		_spec = sqlgraph.NewCreateSpec(fileextract.Table, sqlgraph.NewFieldSpec(fileextract.FieldID, field.TypeUUID))
	)
	if id, ok := fec.mutation.ID(); ok {
		_node.ID = id
		_spec.ID.Value = &id
	}
	if value, ok := fec.mutation.HCLID(); ok {
		_spec.SetField(fileextract.FieldHCLID, field.TypeString, value)
		_node.HCLID = value
	}
	if value, ok := fec.mutation.Source(); ok {
		_spec.SetField(fileextract.FieldSource, field.TypeString, value)
		_node.Source = value
	}
	if value, ok := fec.mutation.Destination(); ok {
		_spec.SetField(fileextract.FieldDestination, field.TypeString, value)
		_node.Destination = value
	}
	if value, ok := fec.mutation.GetType(); ok {
		_spec.SetField(fileextract.FieldType, field.TypeString, value)
		_node.Type = value
	}
	if value, ok := fec.mutation.Tags(); ok {
		_spec.SetField(fileextract.FieldTags, field.TypeJSON, value)
		_node.Tags = value
	}
	if nodes := fec.mutation.EnvironmentIDs(); len(nodes) > 0 {
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
		_node.environment_file_extracts = &nodes[0]
		_spec.Edges = append(_spec.Edges, edge)
	}
	return _node, _spec
}

// FileExtractCreateBulk is the builder for creating many FileExtract entities in bulk.
type FileExtractCreateBulk struct {
	config
	err      error
	builders []*FileExtractCreate
}

// Save creates the FileExtract entities in the database.
func (fecb *FileExtractCreateBulk) Save(ctx context.Context) ([]*FileExtract, error) {
	if fecb.err != nil {
		return nil, fecb.err
	}
	specs := make([]*sqlgraph.CreateSpec, len(fecb.builders))
	nodes := make([]*FileExtract, len(fecb.builders))
	mutators := make([]Mutator, len(fecb.builders))
	for i := range fecb.builders {
		func(i int, root context.Context) {
			builder := fecb.builders[i]
			builder.defaults()
			var mut Mutator = MutateFunc(func(ctx context.Context, m Mutation) (Value, error) {
				mutation, ok := m.(*FileExtractMutation)
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
					_, err = mutators[i+1].Mutate(root, fecb.builders[i+1].mutation)
				} else {
					spec := &sqlgraph.BatchCreateSpec{Nodes: specs}
					// Invoke the actual operation on the latest mutation in the chain.
					if err = sqlgraph.BatchCreate(ctx, fecb.driver, spec); err != nil {
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
		if _, err := mutators[0].Mutate(ctx, fecb.builders[0].mutation); err != nil {
			return nil, err
		}
	}
	return nodes, nil
}

// SaveX is like Save, but panics if an error occurs.
func (fecb *FileExtractCreateBulk) SaveX(ctx context.Context) []*FileExtract {
	v, err := fecb.Save(ctx)
	if err != nil {
		panic(err)
	}
	return v
}

// Exec executes the query.
func (fecb *FileExtractCreateBulk) Exec(ctx context.Context) error {
	_, err := fecb.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (fecb *FileExtractCreateBulk) ExecX(ctx context.Context) {
	if err := fecb.Exec(ctx); err != nil {
		panic(err)
	}
}
