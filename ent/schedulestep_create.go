// Code generated by ent, DO NOT EDIT.

package ent

import (
	"context"
	"errors"
	"fmt"
	"time"

	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/schema/field"
	"github.com/gen0cide/laforge/ent/ansible"
	"github.com/gen0cide/laforge/ent/command"
	"github.com/gen0cide/laforge/ent/filedelete"
	"github.com/gen0cide/laforge/ent/filedownload"
	"github.com/gen0cide/laforge/ent/fileextract"
	"github.com/gen0cide/laforge/ent/host"
	"github.com/gen0cide/laforge/ent/provisionedschedulestep"
	"github.com/gen0cide/laforge/ent/schedulestep"
	"github.com/gen0cide/laforge/ent/script"
	"github.com/gen0cide/laforge/ent/status"
	"github.com/google/uuid"
)

// ScheduleStepCreate is the builder for creating a ScheduleStep entity.
type ScheduleStepCreate struct {
	config
	mutation *ScheduleStepMutation
	hooks    []Hook
}

// SetType sets the "type" field.
func (ssc *ScheduleStepCreate) SetType(s schedulestep.Type) *ScheduleStepCreate {
	ssc.mutation.SetType(s)
	return ssc
}

// SetRepeated sets the "repeated" field.
func (ssc *ScheduleStepCreate) SetRepeated(b bool) *ScheduleStepCreate {
	ssc.mutation.SetRepeated(b)
	return ssc
}

// SetStartTime sets the "start_time" field.
func (ssc *ScheduleStepCreate) SetStartTime(t time.Time) *ScheduleStepCreate {
	ssc.mutation.SetStartTime(t)
	return ssc
}

// SetEndTime sets the "end_time" field.
func (ssc *ScheduleStepCreate) SetEndTime(t time.Time) *ScheduleStepCreate {
	ssc.mutation.SetEndTime(t)
	return ssc
}

// SetInterval sets the "interval" field.
func (ssc *ScheduleStepCreate) SetInterval(i int) *ScheduleStepCreate {
	ssc.mutation.SetInterval(i)
	return ssc
}

// SetID sets the "id" field.
func (ssc *ScheduleStepCreate) SetID(u uuid.UUID) *ScheduleStepCreate {
	ssc.mutation.SetID(u)
	return ssc
}

// SetNillableID sets the "id" field if the given value is not nil.
func (ssc *ScheduleStepCreate) SetNillableID(u *uuid.UUID) *ScheduleStepCreate {
	if u != nil {
		ssc.SetID(*u)
	}
	return ssc
}

// SetScheduleStepToStatusID sets the "ScheduleStepToStatus" edge to the Status entity by ID.
func (ssc *ScheduleStepCreate) SetScheduleStepToStatusID(id uuid.UUID) *ScheduleStepCreate {
	ssc.mutation.SetScheduleStepToStatusID(id)
	return ssc
}

// SetNillableScheduleStepToStatusID sets the "ScheduleStepToStatus" edge to the Status entity by ID if the given value is not nil.
func (ssc *ScheduleStepCreate) SetNillableScheduleStepToStatusID(id *uuid.UUID) *ScheduleStepCreate {
	if id != nil {
		ssc = ssc.SetScheduleStepToStatusID(*id)
	}
	return ssc
}

// SetScheduleStepToStatus sets the "ScheduleStepToStatus" edge to the Status entity.
func (ssc *ScheduleStepCreate) SetScheduleStepToStatus(s *Status) *ScheduleStepCreate {
	return ssc.SetScheduleStepToStatusID(s.ID)
}

// SetScheduleStepToScriptID sets the "ScheduleStepToScript" edge to the Script entity by ID.
func (ssc *ScheduleStepCreate) SetScheduleStepToScriptID(id uuid.UUID) *ScheduleStepCreate {
	ssc.mutation.SetScheduleStepToScriptID(id)
	return ssc
}

// SetNillableScheduleStepToScriptID sets the "ScheduleStepToScript" edge to the Script entity by ID if the given value is not nil.
func (ssc *ScheduleStepCreate) SetNillableScheduleStepToScriptID(id *uuid.UUID) *ScheduleStepCreate {
	if id != nil {
		ssc = ssc.SetScheduleStepToScriptID(*id)
	}
	return ssc
}

// SetScheduleStepToScript sets the "ScheduleStepToScript" edge to the Script entity.
func (ssc *ScheduleStepCreate) SetScheduleStepToScript(s *Script) *ScheduleStepCreate {
	return ssc.SetScheduleStepToScriptID(s.ID)
}

// SetScheduleStepToCommandID sets the "ScheduleStepToCommand" edge to the Command entity by ID.
func (ssc *ScheduleStepCreate) SetScheduleStepToCommandID(id uuid.UUID) *ScheduleStepCreate {
	ssc.mutation.SetScheduleStepToCommandID(id)
	return ssc
}

// SetNillableScheduleStepToCommandID sets the "ScheduleStepToCommand" edge to the Command entity by ID if the given value is not nil.
func (ssc *ScheduleStepCreate) SetNillableScheduleStepToCommandID(id *uuid.UUID) *ScheduleStepCreate {
	if id != nil {
		ssc = ssc.SetScheduleStepToCommandID(*id)
	}
	return ssc
}

// SetScheduleStepToCommand sets the "ScheduleStepToCommand" edge to the Command entity.
func (ssc *ScheduleStepCreate) SetScheduleStepToCommand(c *Command) *ScheduleStepCreate {
	return ssc.SetScheduleStepToCommandID(c.ID)
}

// SetScheduleStepToFileDeleteID sets the "ScheduleStepToFileDelete" edge to the FileDelete entity by ID.
func (ssc *ScheduleStepCreate) SetScheduleStepToFileDeleteID(id uuid.UUID) *ScheduleStepCreate {
	ssc.mutation.SetScheduleStepToFileDeleteID(id)
	return ssc
}

// SetNillableScheduleStepToFileDeleteID sets the "ScheduleStepToFileDelete" edge to the FileDelete entity by ID if the given value is not nil.
func (ssc *ScheduleStepCreate) SetNillableScheduleStepToFileDeleteID(id *uuid.UUID) *ScheduleStepCreate {
	if id != nil {
		ssc = ssc.SetScheduleStepToFileDeleteID(*id)
	}
	return ssc
}

// SetScheduleStepToFileDelete sets the "ScheduleStepToFileDelete" edge to the FileDelete entity.
func (ssc *ScheduleStepCreate) SetScheduleStepToFileDelete(f *FileDelete) *ScheduleStepCreate {
	return ssc.SetScheduleStepToFileDeleteID(f.ID)
}

// SetScheduleStepToFileDownloadID sets the "ScheduleStepToFileDownload" edge to the FileDownload entity by ID.
func (ssc *ScheduleStepCreate) SetScheduleStepToFileDownloadID(id uuid.UUID) *ScheduleStepCreate {
	ssc.mutation.SetScheduleStepToFileDownloadID(id)
	return ssc
}

// SetNillableScheduleStepToFileDownloadID sets the "ScheduleStepToFileDownload" edge to the FileDownload entity by ID if the given value is not nil.
func (ssc *ScheduleStepCreate) SetNillableScheduleStepToFileDownloadID(id *uuid.UUID) *ScheduleStepCreate {
	if id != nil {
		ssc = ssc.SetScheduleStepToFileDownloadID(*id)
	}
	return ssc
}

// SetScheduleStepToFileDownload sets the "ScheduleStepToFileDownload" edge to the FileDownload entity.
func (ssc *ScheduleStepCreate) SetScheduleStepToFileDownload(f *FileDownload) *ScheduleStepCreate {
	return ssc.SetScheduleStepToFileDownloadID(f.ID)
}

// SetScheduleStepToFileExtractID sets the "ScheduleStepToFileExtract" edge to the FileExtract entity by ID.
func (ssc *ScheduleStepCreate) SetScheduleStepToFileExtractID(id uuid.UUID) *ScheduleStepCreate {
	ssc.mutation.SetScheduleStepToFileExtractID(id)
	return ssc
}

// SetNillableScheduleStepToFileExtractID sets the "ScheduleStepToFileExtract" edge to the FileExtract entity by ID if the given value is not nil.
func (ssc *ScheduleStepCreate) SetNillableScheduleStepToFileExtractID(id *uuid.UUID) *ScheduleStepCreate {
	if id != nil {
		ssc = ssc.SetScheduleStepToFileExtractID(*id)
	}
	return ssc
}

// SetScheduleStepToFileExtract sets the "ScheduleStepToFileExtract" edge to the FileExtract entity.
func (ssc *ScheduleStepCreate) SetScheduleStepToFileExtract(f *FileExtract) *ScheduleStepCreate {
	return ssc.SetScheduleStepToFileExtractID(f.ID)
}

// SetScheduleStepToAnsibleID sets the "ScheduleStepToAnsible" edge to the Ansible entity by ID.
func (ssc *ScheduleStepCreate) SetScheduleStepToAnsibleID(id uuid.UUID) *ScheduleStepCreate {
	ssc.mutation.SetScheduleStepToAnsibleID(id)
	return ssc
}

// SetNillableScheduleStepToAnsibleID sets the "ScheduleStepToAnsible" edge to the Ansible entity by ID if the given value is not nil.
func (ssc *ScheduleStepCreate) SetNillableScheduleStepToAnsibleID(id *uuid.UUID) *ScheduleStepCreate {
	if id != nil {
		ssc = ssc.SetScheduleStepToAnsibleID(*id)
	}
	return ssc
}

// SetScheduleStepToAnsible sets the "ScheduleStepToAnsible" edge to the Ansible entity.
func (ssc *ScheduleStepCreate) SetScheduleStepToAnsible(a *Ansible) *ScheduleStepCreate {
	return ssc.SetScheduleStepToAnsibleID(a.ID)
}

// AddScheduleStepToProvisionedScheduleStepIDs adds the "ScheduleStepToProvisionedScheduleStep" edge to the ProvisionedScheduleStep entity by IDs.
func (ssc *ScheduleStepCreate) AddScheduleStepToProvisionedScheduleStepIDs(ids ...uuid.UUID) *ScheduleStepCreate {
	ssc.mutation.AddScheduleStepToProvisionedScheduleStepIDs(ids...)
	return ssc
}

// AddScheduleStepToProvisionedScheduleStep adds the "ScheduleStepToProvisionedScheduleStep" edges to the ProvisionedScheduleStep entity.
func (ssc *ScheduleStepCreate) AddScheduleStepToProvisionedScheduleStep(p ...*ProvisionedScheduleStep) *ScheduleStepCreate {
	ids := make([]uuid.UUID, len(p))
	for i := range p {
		ids[i] = p[i].ID
	}
	return ssc.AddScheduleStepToProvisionedScheduleStepIDs(ids...)
}

// SetScheduleStepToHostID sets the "ScheduleStepToHost" edge to the Host entity by ID.
func (ssc *ScheduleStepCreate) SetScheduleStepToHostID(id uuid.UUID) *ScheduleStepCreate {
	ssc.mutation.SetScheduleStepToHostID(id)
	return ssc
}

// SetScheduleStepToHost sets the "ScheduleStepToHost" edge to the Host entity.
func (ssc *ScheduleStepCreate) SetScheduleStepToHost(h *Host) *ScheduleStepCreate {
	return ssc.SetScheduleStepToHostID(h.ID)
}

// Mutation returns the ScheduleStepMutation object of the builder.
func (ssc *ScheduleStepCreate) Mutation() *ScheduleStepMutation {
	return ssc.mutation
}

// Save creates the ScheduleStep in the database.
func (ssc *ScheduleStepCreate) Save(ctx context.Context) (*ScheduleStep, error) {
	var (
		err  error
		node *ScheduleStep
	)
	ssc.defaults()
	if len(ssc.hooks) == 0 {
		if err = ssc.check(); err != nil {
			return nil, err
		}
		node, err = ssc.sqlSave(ctx)
	} else {
		var mut Mutator = MutateFunc(func(ctx context.Context, m Mutation) (Value, error) {
			mutation, ok := m.(*ScheduleStepMutation)
			if !ok {
				return nil, fmt.Errorf("unexpected mutation type %T", m)
			}
			if err = ssc.check(); err != nil {
				return nil, err
			}
			ssc.mutation = mutation
			if node, err = ssc.sqlSave(ctx); err != nil {
				return nil, err
			}
			mutation.id = &node.ID
			mutation.done = true
			return node, err
		})
		for i := len(ssc.hooks) - 1; i >= 0; i-- {
			if ssc.hooks[i] == nil {
				return nil, fmt.Errorf("ent: uninitialized hook (forgotten import ent/runtime?)")
			}
			mut = ssc.hooks[i](mut)
		}
		v, err := mut.Mutate(ctx, ssc.mutation)
		if err != nil {
			return nil, err
		}
		nv, ok := v.(*ScheduleStep)
		if !ok {
			return nil, fmt.Errorf("unexpected node type %T returned from ScheduleStepMutation", v)
		}
		node = nv
	}
	return node, err
}

// SaveX calls Save and panics if Save returns an error.
func (ssc *ScheduleStepCreate) SaveX(ctx context.Context) *ScheduleStep {
	v, err := ssc.Save(ctx)
	if err != nil {
		panic(err)
	}
	return v
}

// Exec executes the query.
func (ssc *ScheduleStepCreate) Exec(ctx context.Context) error {
	_, err := ssc.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (ssc *ScheduleStepCreate) ExecX(ctx context.Context) {
	if err := ssc.Exec(ctx); err != nil {
		panic(err)
	}
}

// defaults sets the default values of the builder before save.
func (ssc *ScheduleStepCreate) defaults() {
	if _, ok := ssc.mutation.ID(); !ok {
		v := schedulestep.DefaultID()
		ssc.mutation.SetID(v)
	}
}

// check runs all checks and user-defined validators on the builder.
func (ssc *ScheduleStepCreate) check() error {
	if _, ok := ssc.mutation.GetType(); !ok {
		return &ValidationError{Name: "type", err: errors.New(`ent: missing required field "ScheduleStep.type"`)}
	}
	if v, ok := ssc.mutation.GetType(); ok {
		if err := schedulestep.TypeValidator(v); err != nil {
			return &ValidationError{Name: "type", err: fmt.Errorf(`ent: validator failed for field "ScheduleStep.type": %w`, err)}
		}
	}
	if _, ok := ssc.mutation.Repeated(); !ok {
		return &ValidationError{Name: "repeated", err: errors.New(`ent: missing required field "ScheduleStep.repeated"`)}
	}
	if _, ok := ssc.mutation.StartTime(); !ok {
		return &ValidationError{Name: "start_time", err: errors.New(`ent: missing required field "ScheduleStep.start_time"`)}
	}
	if _, ok := ssc.mutation.EndTime(); !ok {
		return &ValidationError{Name: "end_time", err: errors.New(`ent: missing required field "ScheduleStep.end_time"`)}
	}
	if _, ok := ssc.mutation.Interval(); !ok {
		return &ValidationError{Name: "interval", err: errors.New(`ent: missing required field "ScheduleStep.interval"`)}
	}
	if _, ok := ssc.mutation.ScheduleStepToHostID(); !ok {
		return &ValidationError{Name: "ScheduleStepToHost", err: errors.New(`ent: missing required edge "ScheduleStep.ScheduleStepToHost"`)}
	}
	return nil
}

func (ssc *ScheduleStepCreate) sqlSave(ctx context.Context) (*ScheduleStep, error) {
	_node, _spec := ssc.createSpec()
	if err := sqlgraph.CreateNode(ctx, ssc.driver, _spec); err != nil {
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
	return _node, nil
}

func (ssc *ScheduleStepCreate) createSpec() (*ScheduleStep, *sqlgraph.CreateSpec) {
	var (
		_node = &ScheduleStep{config: ssc.config}
		_spec = &sqlgraph.CreateSpec{
			Table: schedulestep.Table,
			ID: &sqlgraph.FieldSpec{
				Type:   field.TypeUUID,
				Column: schedulestep.FieldID,
			},
		}
	)
	if id, ok := ssc.mutation.ID(); ok {
		_node.ID = id
		_spec.ID.Value = &id
	}
	if value, ok := ssc.mutation.GetType(); ok {
		_spec.Fields = append(_spec.Fields, &sqlgraph.FieldSpec{
			Type:   field.TypeEnum,
			Value:  value,
			Column: schedulestep.FieldType,
		})
		_node.Type = value
	}
	if value, ok := ssc.mutation.Repeated(); ok {
		_spec.Fields = append(_spec.Fields, &sqlgraph.FieldSpec{
			Type:   field.TypeBool,
			Value:  value,
			Column: schedulestep.FieldRepeated,
		})
		_node.Repeated = value
	}
	if value, ok := ssc.mutation.StartTime(); ok {
		_spec.Fields = append(_spec.Fields, &sqlgraph.FieldSpec{
			Type:   field.TypeTime,
			Value:  value,
			Column: schedulestep.FieldStartTime,
		})
		_node.StartTime = value
	}
	if value, ok := ssc.mutation.EndTime(); ok {
		_spec.Fields = append(_spec.Fields, &sqlgraph.FieldSpec{
			Type:   field.TypeTime,
			Value:  value,
			Column: schedulestep.FieldEndTime,
		})
		_node.EndTime = value
	}
	if value, ok := ssc.mutation.Interval(); ok {
		_spec.Fields = append(_spec.Fields, &sqlgraph.FieldSpec{
			Type:   field.TypeInt,
			Value:  value,
			Column: schedulestep.FieldInterval,
		})
		_node.Interval = value
	}
	if nodes := ssc.mutation.ScheduleStepToStatusIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2O,
			Inverse: false,
			Table:   schedulestep.ScheduleStepToStatusTable,
			Columns: []string{schedulestep.ScheduleStepToStatusColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeUUID,
					Column: status.FieldID,
				},
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges = append(_spec.Edges, edge)
	}
	if nodes := ssc.mutation.ScheduleStepToScriptIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: false,
			Table:   schedulestep.ScheduleStepToScriptTable,
			Columns: []string{schedulestep.ScheduleStepToScriptColumn},
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
		_node.schedule_step_schedule_step_to_script = &nodes[0]
		_spec.Edges = append(_spec.Edges, edge)
	}
	if nodes := ssc.mutation.ScheduleStepToCommandIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: false,
			Table:   schedulestep.ScheduleStepToCommandTable,
			Columns: []string{schedulestep.ScheduleStepToCommandColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeUUID,
					Column: command.FieldID,
				},
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_node.schedule_step_schedule_step_to_command = &nodes[0]
		_spec.Edges = append(_spec.Edges, edge)
	}
	if nodes := ssc.mutation.ScheduleStepToFileDeleteIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: false,
			Table:   schedulestep.ScheduleStepToFileDeleteTable,
			Columns: []string{schedulestep.ScheduleStepToFileDeleteColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeUUID,
					Column: filedelete.FieldID,
				},
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_node.schedule_step_schedule_step_to_file_delete = &nodes[0]
		_spec.Edges = append(_spec.Edges, edge)
	}
	if nodes := ssc.mutation.ScheduleStepToFileDownloadIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: false,
			Table:   schedulestep.ScheduleStepToFileDownloadTable,
			Columns: []string{schedulestep.ScheduleStepToFileDownloadColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeUUID,
					Column: filedownload.FieldID,
				},
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_node.schedule_step_schedule_step_to_file_download = &nodes[0]
		_spec.Edges = append(_spec.Edges, edge)
	}
	if nodes := ssc.mutation.ScheduleStepToFileExtractIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: false,
			Table:   schedulestep.ScheduleStepToFileExtractTable,
			Columns: []string{schedulestep.ScheduleStepToFileExtractColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeUUID,
					Column: fileextract.FieldID,
				},
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_node.schedule_step_schedule_step_to_file_extract = &nodes[0]
		_spec.Edges = append(_spec.Edges, edge)
	}
	if nodes := ssc.mutation.ScheduleStepToAnsibleIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: false,
			Table:   schedulestep.ScheduleStepToAnsibleTable,
			Columns: []string{schedulestep.ScheduleStepToAnsibleColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeUUID,
					Column: ansible.FieldID,
				},
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_node.schedule_step_schedule_step_to_ansible = &nodes[0]
		_spec.Edges = append(_spec.Edges, edge)
	}
	if nodes := ssc.mutation.ScheduleStepToProvisionedScheduleStepIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: false,
			Table:   schedulestep.ScheduleStepToProvisionedScheduleStepTable,
			Columns: []string{schedulestep.ScheduleStepToProvisionedScheduleStepColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeUUID,
					Column: provisionedschedulestep.FieldID,
				},
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges = append(_spec.Edges, edge)
	}
	if nodes := ssc.mutation.ScheduleStepToHostIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: true,
			Table:   schedulestep.ScheduleStepToHostTable,
			Columns: []string{schedulestep.ScheduleStepToHostColumn},
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
		_node.host_host_to_schedule_step = &nodes[0]
		_spec.Edges = append(_spec.Edges, edge)
	}
	return _node, _spec
}

// ScheduleStepCreateBulk is the builder for creating many ScheduleStep entities in bulk.
type ScheduleStepCreateBulk struct {
	config
	builders []*ScheduleStepCreate
}

// Save creates the ScheduleStep entities in the database.
func (sscb *ScheduleStepCreateBulk) Save(ctx context.Context) ([]*ScheduleStep, error) {
	specs := make([]*sqlgraph.CreateSpec, len(sscb.builders))
	nodes := make([]*ScheduleStep, len(sscb.builders))
	mutators := make([]Mutator, len(sscb.builders))
	for i := range sscb.builders {
		func(i int, root context.Context) {
			builder := sscb.builders[i]
			builder.defaults()
			var mut Mutator = MutateFunc(func(ctx context.Context, m Mutation) (Value, error) {
				mutation, ok := m.(*ScheduleStepMutation)
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
					_, err = mutators[i+1].Mutate(root, sscb.builders[i+1].mutation)
				} else {
					spec := &sqlgraph.BatchCreateSpec{Nodes: specs}
					// Invoke the actual operation on the latest mutation in the chain.
					if err = sqlgraph.BatchCreate(ctx, sscb.driver, spec); err != nil {
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
		if _, err := mutators[0].Mutate(ctx, sscb.builders[0].mutation); err != nil {
			return nil, err
		}
	}
	return nodes, nil
}

// SaveX is like Save, but panics if an error occurs.
func (sscb *ScheduleStepCreateBulk) SaveX(ctx context.Context) []*ScheduleStep {
	v, err := sscb.Save(ctx)
	if err != nil {
		panic(err)
	}
	return v
}

// Exec executes the query.
func (sscb *ScheduleStepCreateBulk) Exec(ctx context.Context) error {
	_, err := sscb.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (sscb *ScheduleStepCreateBulk) ExecX(ctx context.Context) {
	if err := sscb.Exec(ctx); err != nil {
		panic(err)
	}
}
