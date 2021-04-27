// Code generated by entc, DO NOT EDIT.

package ent

import (
	"context"
	"errors"
	"fmt"

	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/schema/field"
	"github.com/gen0cide/laforge/ent/environment"
	"github.com/gen0cide/laforge/ent/filedownload"
	"github.com/google/uuid"
)

// FileDownloadCreate is the builder for creating a FileDownload entity.
type FileDownloadCreate struct {
	config
	mutation *FileDownloadMutation
	hooks    []Hook
}

// SetHclID sets the "hcl_id" field.
func (fdc *FileDownloadCreate) SetHclID(s string) *FileDownloadCreate {
	fdc.mutation.SetHclID(s)
	return fdc
}

// SetSourceType sets the "source_type" field.
func (fdc *FileDownloadCreate) SetSourceType(s string) *FileDownloadCreate {
	fdc.mutation.SetSourceType(s)
	return fdc
}

// SetSource sets the "source" field.
func (fdc *FileDownloadCreate) SetSource(s string) *FileDownloadCreate {
	fdc.mutation.SetSource(s)
	return fdc
}

// SetDestination sets the "destination" field.
func (fdc *FileDownloadCreate) SetDestination(s string) *FileDownloadCreate {
	fdc.mutation.SetDestination(s)
	return fdc
}

// SetTemplate sets the "template" field.
func (fdc *FileDownloadCreate) SetTemplate(b bool) *FileDownloadCreate {
	fdc.mutation.SetTemplate(b)
	return fdc
}

// SetPerms sets the "perms" field.
func (fdc *FileDownloadCreate) SetPerms(s string) *FileDownloadCreate {
	fdc.mutation.SetPerms(s)
	return fdc
}

// SetDisabled sets the "disabled" field.
func (fdc *FileDownloadCreate) SetDisabled(b bool) *FileDownloadCreate {
	fdc.mutation.SetDisabled(b)
	return fdc
}

// SetMd5 sets the "md5" field.
func (fdc *FileDownloadCreate) SetMd5(s string) *FileDownloadCreate {
	fdc.mutation.SetMd5(s)
	return fdc
}

// SetAbsPath sets the "abs_path" field.
func (fdc *FileDownloadCreate) SetAbsPath(s string) *FileDownloadCreate {
	fdc.mutation.SetAbsPath(s)
	return fdc
}

// SetTags sets the "tags" field.
func (fdc *FileDownloadCreate) SetTags(m map[string]string) *FileDownloadCreate {
	fdc.mutation.SetTags(m)
	return fdc
}

// SetID sets the "id" field.
func (fdc *FileDownloadCreate) SetID(u uuid.UUID) *FileDownloadCreate {
	fdc.mutation.SetID(u)
	return fdc
}

// SetFileDownloadToEnvironmentID sets the "FileDownloadToEnvironment" edge to the Environment entity by ID.
func (fdc *FileDownloadCreate) SetFileDownloadToEnvironmentID(id uuid.UUID) *FileDownloadCreate {
	fdc.mutation.SetFileDownloadToEnvironmentID(id)
	return fdc
}

// SetNillableFileDownloadToEnvironmentID sets the "FileDownloadToEnvironment" edge to the Environment entity by ID if the given value is not nil.
func (fdc *FileDownloadCreate) SetNillableFileDownloadToEnvironmentID(id *uuid.UUID) *FileDownloadCreate {
	if id != nil {
		fdc = fdc.SetFileDownloadToEnvironmentID(*id)
	}
	return fdc
}

// SetFileDownloadToEnvironment sets the "FileDownloadToEnvironment" edge to the Environment entity.
func (fdc *FileDownloadCreate) SetFileDownloadToEnvironment(e *Environment) *FileDownloadCreate {
	return fdc.SetFileDownloadToEnvironmentID(e.ID)
}

// Mutation returns the FileDownloadMutation object of the builder.
func (fdc *FileDownloadCreate) Mutation() *FileDownloadMutation {
	return fdc.mutation
}

// Save creates the FileDownload in the database.
func (fdc *FileDownloadCreate) Save(ctx context.Context) (*FileDownload, error) {
	var (
		err  error
		node *FileDownload
	)
	fdc.defaults()
	if len(fdc.hooks) == 0 {
		if err = fdc.check(); err != nil {
			return nil, err
		}
		node, err = fdc.sqlSave(ctx)
	} else {
		var mut Mutator = MutateFunc(func(ctx context.Context, m Mutation) (Value, error) {
			mutation, ok := m.(*FileDownloadMutation)
			if !ok {
				return nil, fmt.Errorf("unexpected mutation type %T", m)
			}
			if err = fdc.check(); err != nil {
				return nil, err
			}
			fdc.mutation = mutation
			node, err = fdc.sqlSave(ctx)
			mutation.done = true
			return node, err
		})
		for i := len(fdc.hooks) - 1; i >= 0; i-- {
			mut = fdc.hooks[i](mut)
		}
		if _, err := mut.Mutate(ctx, fdc.mutation); err != nil {
			return nil, err
		}
	}
	return node, err
}

// SaveX calls Save and panics if Save returns an error.
func (fdc *FileDownloadCreate) SaveX(ctx context.Context) *FileDownload {
	v, err := fdc.Save(ctx)
	if err != nil {
		panic(err)
	}
	return v
}

// defaults sets the default values of the builder before save.
func (fdc *FileDownloadCreate) defaults() {
	if _, ok := fdc.mutation.ID(); !ok {
		v := filedownload.DefaultID()
		fdc.mutation.SetID(v)
	}
}

// check runs all checks and user-defined validators on the builder.
func (fdc *FileDownloadCreate) check() error {
	if _, ok := fdc.mutation.HclID(); !ok {
		return &ValidationError{Name: "hcl_id", err: errors.New("ent: missing required field \"hcl_id\"")}
	}
	if _, ok := fdc.mutation.SourceType(); !ok {
		return &ValidationError{Name: "source_type", err: errors.New("ent: missing required field \"source_type\"")}
	}
	if _, ok := fdc.mutation.Source(); !ok {
		return &ValidationError{Name: "source", err: errors.New("ent: missing required field \"source\"")}
	}
	if _, ok := fdc.mutation.Destination(); !ok {
		return &ValidationError{Name: "destination", err: errors.New("ent: missing required field \"destination\"")}
	}
	if _, ok := fdc.mutation.Template(); !ok {
		return &ValidationError{Name: "template", err: errors.New("ent: missing required field \"template\"")}
	}
	if _, ok := fdc.mutation.Perms(); !ok {
		return &ValidationError{Name: "perms", err: errors.New("ent: missing required field \"perms\"")}
	}
	if _, ok := fdc.mutation.Disabled(); !ok {
		return &ValidationError{Name: "disabled", err: errors.New("ent: missing required field \"disabled\"")}
	}
	if _, ok := fdc.mutation.Md5(); !ok {
		return &ValidationError{Name: "md5", err: errors.New("ent: missing required field \"md5\"")}
	}
	if _, ok := fdc.mutation.AbsPath(); !ok {
		return &ValidationError{Name: "abs_path", err: errors.New("ent: missing required field \"abs_path\"")}
	}
	if _, ok := fdc.mutation.Tags(); !ok {
		return &ValidationError{Name: "tags", err: errors.New("ent: missing required field \"tags\"")}
	}
	return nil
}

func (fdc *FileDownloadCreate) sqlSave(ctx context.Context) (*FileDownload, error) {
	_node, _spec := fdc.createSpec()
	if err := sqlgraph.CreateNode(ctx, fdc.driver, _spec); err != nil {
		if cerr, ok := isSQLConstraintError(err); ok {
			err = cerr
		}
		return nil, err
	}
	return _node, nil
}

func (fdc *FileDownloadCreate) createSpec() (*FileDownload, *sqlgraph.CreateSpec) {
	var (
		_node = &FileDownload{config: fdc.config}
		_spec = &sqlgraph.CreateSpec{
			Table: filedownload.Table,
			ID: &sqlgraph.FieldSpec{
				Type:   field.TypeUUID,
				Column: filedownload.FieldID,
			},
		}
	)
	if id, ok := fdc.mutation.ID(); ok {
		_node.ID = id
		_spec.ID.Value = id
	}
	if value, ok := fdc.mutation.HclID(); ok {
		_spec.Fields = append(_spec.Fields, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Value:  value,
			Column: filedownload.FieldHclID,
		})
		_node.HclID = value
	}
	if value, ok := fdc.mutation.SourceType(); ok {
		_spec.Fields = append(_spec.Fields, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Value:  value,
			Column: filedownload.FieldSourceType,
		})
		_node.SourceType = value
	}
	if value, ok := fdc.mutation.Source(); ok {
		_spec.Fields = append(_spec.Fields, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Value:  value,
			Column: filedownload.FieldSource,
		})
		_node.Source = value
	}
	if value, ok := fdc.mutation.Destination(); ok {
		_spec.Fields = append(_spec.Fields, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Value:  value,
			Column: filedownload.FieldDestination,
		})
		_node.Destination = value
	}
	if value, ok := fdc.mutation.Template(); ok {
		_spec.Fields = append(_spec.Fields, &sqlgraph.FieldSpec{
			Type:   field.TypeBool,
			Value:  value,
			Column: filedownload.FieldTemplate,
		})
		_node.Template = value
	}
	if value, ok := fdc.mutation.Perms(); ok {
		_spec.Fields = append(_spec.Fields, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Value:  value,
			Column: filedownload.FieldPerms,
		})
		_node.Perms = value
	}
	if value, ok := fdc.mutation.Disabled(); ok {
		_spec.Fields = append(_spec.Fields, &sqlgraph.FieldSpec{
			Type:   field.TypeBool,
			Value:  value,
			Column: filedownload.FieldDisabled,
		})
		_node.Disabled = value
	}
	if value, ok := fdc.mutation.Md5(); ok {
		_spec.Fields = append(_spec.Fields, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Value:  value,
			Column: filedownload.FieldMd5,
		})
		_node.Md5 = value
	}
	if value, ok := fdc.mutation.AbsPath(); ok {
		_spec.Fields = append(_spec.Fields, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Value:  value,
			Column: filedownload.FieldAbsPath,
		})
		_node.AbsPath = value
	}
	if value, ok := fdc.mutation.Tags(); ok {
		_spec.Fields = append(_spec.Fields, &sqlgraph.FieldSpec{
			Type:   field.TypeJSON,
			Value:  value,
			Column: filedownload.FieldTags,
		})
		_node.Tags = value
	}
	if nodes := fdc.mutation.FileDownloadToEnvironmentIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: true,
			Table:   filedownload.FileDownloadToEnvironmentTable,
			Columns: []string{filedownload.FileDownloadToEnvironmentColumn},
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
		_node.environment_environment_to_file_download = &nodes[0]
		_spec.Edges = append(_spec.Edges, edge)
	}
	return _node, _spec
}

// FileDownloadCreateBulk is the builder for creating many FileDownload entities in bulk.
type FileDownloadCreateBulk struct {
	config
	builders []*FileDownloadCreate
}

// Save creates the FileDownload entities in the database.
func (fdcb *FileDownloadCreateBulk) Save(ctx context.Context) ([]*FileDownload, error) {
	specs := make([]*sqlgraph.CreateSpec, len(fdcb.builders))
	nodes := make([]*FileDownload, len(fdcb.builders))
	mutators := make([]Mutator, len(fdcb.builders))
	for i := range fdcb.builders {
		func(i int, root context.Context) {
			builder := fdcb.builders[i]
			builder.defaults()
			var mut Mutator = MutateFunc(func(ctx context.Context, m Mutation) (Value, error) {
				mutation, ok := m.(*FileDownloadMutation)
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
					_, err = mutators[i+1].Mutate(root, fdcb.builders[i+1].mutation)
				} else {
					// Invoke the actual operation on the latest mutation in the chain.
					if err = sqlgraph.BatchCreate(ctx, fdcb.driver, &sqlgraph.BatchCreateSpec{Nodes: specs}); err != nil {
						if cerr, ok := isSQLConstraintError(err); ok {
							err = cerr
						}
					}
				}
				mutation.done = true
				if err != nil {
					return nil, err
				}
				return nodes[i], nil
			})
			for i := len(builder.hooks) - 1; i >= 0; i-- {
				mut = builder.hooks[i](mut)
			}
			mutators[i] = mut
		}(i, ctx)
	}
	if len(mutators) > 0 {
		if _, err := mutators[0].Mutate(ctx, fdcb.builders[0].mutation); err != nil {
			return nil, err
		}
	}
	return nodes, nil
}

// SaveX is like Save, but panics if an error occurs.
func (fdcb *FileDownloadCreateBulk) SaveX(ctx context.Context) []*FileDownload {
	v, err := fdcb.Save(ctx)
	if err != nil {
		panic(err)
	}
	return v
}
