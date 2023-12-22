// Code generated by ent, DO NOT EDIT.

package ent

import (
	"context"
	"errors"
	"fmt"

	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/schema/field"
	"github.com/gen0cide/laforge/ent/agenttask"
	"github.com/gen0cide/laforge/ent/ansible"
	"github.com/gen0cide/laforge/ent/command"
	"github.com/gen0cide/laforge/ent/dnsrecord"
	"github.com/gen0cide/laforge/ent/filedelete"
	"github.com/gen0cide/laforge/ent/filedownload"
	"github.com/gen0cide/laforge/ent/fileextract"
	"github.com/gen0cide/laforge/ent/ginfilemiddleware"
	"github.com/gen0cide/laforge/ent/plan"
	"github.com/gen0cide/laforge/ent/provisionedhost"
	"github.com/gen0cide/laforge/ent/provisioningstep"
	"github.com/gen0cide/laforge/ent/script"
	"github.com/gen0cide/laforge/ent/status"
	"github.com/google/uuid"
)

// ProvisioningStepCreate is the builder for creating a ProvisioningStep entity.
type ProvisioningStepCreate struct {
	config
	mutation *ProvisioningStepMutation
	hooks    []Hook
}

// SetType sets the "type" field.
func (psc *ProvisioningStepCreate) SetType(pr provisioningstep.Type) *ProvisioningStepCreate {
	psc.mutation.SetType(pr)
	return psc
}

// SetStepNumber sets the "step_number" field.
func (psc *ProvisioningStepCreate) SetStepNumber(i int) *ProvisioningStepCreate {
	psc.mutation.SetStepNumber(i)
	return psc
}

// SetID sets the "id" field.
func (psc *ProvisioningStepCreate) SetID(u uuid.UUID) *ProvisioningStepCreate {
	psc.mutation.SetID(u)
	return psc
}

// SetNillableID sets the "id" field if the given value is not nil.
func (psc *ProvisioningStepCreate) SetNillableID(u *uuid.UUID) *ProvisioningStepCreate {
	if u != nil {
		psc.SetID(*u)
	}
	return psc
}

// SetStatusID sets the "Status" edge to the Status entity by ID.
func (psc *ProvisioningStepCreate) SetStatusID(id uuid.UUID) *ProvisioningStepCreate {
	psc.mutation.SetStatusID(id)
	return psc
}

// SetNillableStatusID sets the "Status" edge to the Status entity by ID if the given value is not nil.
func (psc *ProvisioningStepCreate) SetNillableStatusID(id *uuid.UUID) *ProvisioningStepCreate {
	if id != nil {
		psc = psc.SetStatusID(*id)
	}
	return psc
}

// SetStatus sets the "Status" edge to the Status entity.
func (psc *ProvisioningStepCreate) SetStatus(s *Status) *ProvisioningStepCreate {
	return psc.SetStatusID(s.ID)
}

// SetProvisionedHostID sets the "ProvisionedHost" edge to the ProvisionedHost entity by ID.
func (psc *ProvisioningStepCreate) SetProvisionedHostID(id uuid.UUID) *ProvisioningStepCreate {
	psc.mutation.SetProvisionedHostID(id)
	return psc
}

// SetNillableProvisionedHostID sets the "ProvisionedHost" edge to the ProvisionedHost entity by ID if the given value is not nil.
func (psc *ProvisioningStepCreate) SetNillableProvisionedHostID(id *uuid.UUID) *ProvisioningStepCreate {
	if id != nil {
		psc = psc.SetProvisionedHostID(*id)
	}
	return psc
}

// SetProvisionedHost sets the "ProvisionedHost" edge to the ProvisionedHost entity.
func (psc *ProvisioningStepCreate) SetProvisionedHost(p *ProvisionedHost) *ProvisioningStepCreate {
	return psc.SetProvisionedHostID(p.ID)
}

// SetScriptID sets the "Script" edge to the Script entity by ID.
func (psc *ProvisioningStepCreate) SetScriptID(id uuid.UUID) *ProvisioningStepCreate {
	psc.mutation.SetScriptID(id)
	return psc
}

// SetNillableScriptID sets the "Script" edge to the Script entity by ID if the given value is not nil.
func (psc *ProvisioningStepCreate) SetNillableScriptID(id *uuid.UUID) *ProvisioningStepCreate {
	if id != nil {
		psc = psc.SetScriptID(*id)
	}
	return psc
}

// SetScript sets the "Script" edge to the Script entity.
func (psc *ProvisioningStepCreate) SetScript(s *Script) *ProvisioningStepCreate {
	return psc.SetScriptID(s.ID)
}

// SetCommandID sets the "Command" edge to the Command entity by ID.
func (psc *ProvisioningStepCreate) SetCommandID(id uuid.UUID) *ProvisioningStepCreate {
	psc.mutation.SetCommandID(id)
	return psc
}

// SetNillableCommandID sets the "Command" edge to the Command entity by ID if the given value is not nil.
func (psc *ProvisioningStepCreate) SetNillableCommandID(id *uuid.UUID) *ProvisioningStepCreate {
	if id != nil {
		psc = psc.SetCommandID(*id)
	}
	return psc
}

// SetCommand sets the "Command" edge to the Command entity.
func (psc *ProvisioningStepCreate) SetCommand(c *Command) *ProvisioningStepCreate {
	return psc.SetCommandID(c.ID)
}

// SetDNSRecordID sets the "DNSRecord" edge to the DNSRecord entity by ID.
func (psc *ProvisioningStepCreate) SetDNSRecordID(id uuid.UUID) *ProvisioningStepCreate {
	psc.mutation.SetDNSRecordID(id)
	return psc
}

// SetNillableDNSRecordID sets the "DNSRecord" edge to the DNSRecord entity by ID if the given value is not nil.
func (psc *ProvisioningStepCreate) SetNillableDNSRecordID(id *uuid.UUID) *ProvisioningStepCreate {
	if id != nil {
		psc = psc.SetDNSRecordID(*id)
	}
	return psc
}

// SetDNSRecord sets the "DNSRecord" edge to the DNSRecord entity.
func (psc *ProvisioningStepCreate) SetDNSRecord(d *DNSRecord) *ProvisioningStepCreate {
	return psc.SetDNSRecordID(d.ID)
}

// SetFileDeleteID sets the "FileDelete" edge to the FileDelete entity by ID.
func (psc *ProvisioningStepCreate) SetFileDeleteID(id uuid.UUID) *ProvisioningStepCreate {
	psc.mutation.SetFileDeleteID(id)
	return psc
}

// SetNillableFileDeleteID sets the "FileDelete" edge to the FileDelete entity by ID if the given value is not nil.
func (psc *ProvisioningStepCreate) SetNillableFileDeleteID(id *uuid.UUID) *ProvisioningStepCreate {
	if id != nil {
		psc = psc.SetFileDeleteID(*id)
	}
	return psc
}

// SetFileDelete sets the "FileDelete" edge to the FileDelete entity.
func (psc *ProvisioningStepCreate) SetFileDelete(f *FileDelete) *ProvisioningStepCreate {
	return psc.SetFileDeleteID(f.ID)
}

// SetFileDownloadID sets the "FileDownload" edge to the FileDownload entity by ID.
func (psc *ProvisioningStepCreate) SetFileDownloadID(id uuid.UUID) *ProvisioningStepCreate {
	psc.mutation.SetFileDownloadID(id)
	return psc
}

// SetNillableFileDownloadID sets the "FileDownload" edge to the FileDownload entity by ID if the given value is not nil.
func (psc *ProvisioningStepCreate) SetNillableFileDownloadID(id *uuid.UUID) *ProvisioningStepCreate {
	if id != nil {
		psc = psc.SetFileDownloadID(*id)
	}
	return psc
}

// SetFileDownload sets the "FileDownload" edge to the FileDownload entity.
func (psc *ProvisioningStepCreate) SetFileDownload(f *FileDownload) *ProvisioningStepCreate {
	return psc.SetFileDownloadID(f.ID)
}

// SetFileExtractID sets the "FileExtract" edge to the FileExtract entity by ID.
func (psc *ProvisioningStepCreate) SetFileExtractID(id uuid.UUID) *ProvisioningStepCreate {
	psc.mutation.SetFileExtractID(id)
	return psc
}

// SetNillableFileExtractID sets the "FileExtract" edge to the FileExtract entity by ID if the given value is not nil.
func (psc *ProvisioningStepCreate) SetNillableFileExtractID(id *uuid.UUID) *ProvisioningStepCreate {
	if id != nil {
		psc = psc.SetFileExtractID(*id)
	}
	return psc
}

// SetFileExtract sets the "FileExtract" edge to the FileExtract entity.
func (psc *ProvisioningStepCreate) SetFileExtract(f *FileExtract) *ProvisioningStepCreate {
	return psc.SetFileExtractID(f.ID)
}

// SetAnsibleID sets the "Ansible" edge to the Ansible entity by ID.
func (psc *ProvisioningStepCreate) SetAnsibleID(id uuid.UUID) *ProvisioningStepCreate {
	psc.mutation.SetAnsibleID(id)
	return psc
}

// SetNillableAnsibleID sets the "Ansible" edge to the Ansible entity by ID if the given value is not nil.
func (psc *ProvisioningStepCreate) SetNillableAnsibleID(id *uuid.UUID) *ProvisioningStepCreate {
	if id != nil {
		psc = psc.SetAnsibleID(*id)
	}
	return psc
}

// SetAnsible sets the "Ansible" edge to the Ansible entity.
func (psc *ProvisioningStepCreate) SetAnsible(a *Ansible) *ProvisioningStepCreate {
	return psc.SetAnsibleID(a.ID)
}

// SetPlanID sets the "Plan" edge to the Plan entity by ID.
func (psc *ProvisioningStepCreate) SetPlanID(id uuid.UUID) *ProvisioningStepCreate {
	psc.mutation.SetPlanID(id)
	return psc
}

// SetNillablePlanID sets the "Plan" edge to the Plan entity by ID if the given value is not nil.
func (psc *ProvisioningStepCreate) SetNillablePlanID(id *uuid.UUID) *ProvisioningStepCreate {
	if id != nil {
		psc = psc.SetPlanID(*id)
	}
	return psc
}

// SetPlan sets the "Plan" edge to the Plan entity.
func (psc *ProvisioningStepCreate) SetPlan(p *Plan) *ProvisioningStepCreate {
	return psc.SetPlanID(p.ID)
}

// AddAgentTaskIDs adds the "AgentTasks" edge to the AgentTask entity by IDs.
func (psc *ProvisioningStepCreate) AddAgentTaskIDs(ids ...uuid.UUID) *ProvisioningStepCreate {
	psc.mutation.AddAgentTaskIDs(ids...)
	return psc
}

// AddAgentTasks adds the "AgentTasks" edges to the AgentTask entity.
func (psc *ProvisioningStepCreate) AddAgentTasks(a ...*AgentTask) *ProvisioningStepCreate {
	ids := make([]uuid.UUID, len(a))
	for i := range a {
		ids[i] = a[i].ID
	}
	return psc.AddAgentTaskIDs(ids...)
}

// SetGinFileMiddlewareID sets the "GinFileMiddleware" edge to the GinFileMiddleware entity by ID.
func (psc *ProvisioningStepCreate) SetGinFileMiddlewareID(id uuid.UUID) *ProvisioningStepCreate {
	psc.mutation.SetGinFileMiddlewareID(id)
	return psc
}

// SetNillableGinFileMiddlewareID sets the "GinFileMiddleware" edge to the GinFileMiddleware entity by ID if the given value is not nil.
func (psc *ProvisioningStepCreate) SetNillableGinFileMiddlewareID(id *uuid.UUID) *ProvisioningStepCreate {
	if id != nil {
		psc = psc.SetGinFileMiddlewareID(*id)
	}
	return psc
}

// SetGinFileMiddleware sets the "GinFileMiddleware" edge to the GinFileMiddleware entity.
func (psc *ProvisioningStepCreate) SetGinFileMiddleware(g *GinFileMiddleware) *ProvisioningStepCreate {
	return psc.SetGinFileMiddlewareID(g.ID)
}

// Mutation returns the ProvisioningStepMutation object of the builder.
func (psc *ProvisioningStepCreate) Mutation() *ProvisioningStepMutation {
	return psc.mutation
}

// Save creates the ProvisioningStep in the database.
func (psc *ProvisioningStepCreate) Save(ctx context.Context) (*ProvisioningStep, error) {
	psc.defaults()
	return withHooks(ctx, psc.sqlSave, psc.mutation, psc.hooks)
}

// SaveX calls Save and panics if Save returns an error.
func (psc *ProvisioningStepCreate) SaveX(ctx context.Context) *ProvisioningStep {
	v, err := psc.Save(ctx)
	if err != nil {
		panic(err)
	}
	return v
}

// Exec executes the query.
func (psc *ProvisioningStepCreate) Exec(ctx context.Context) error {
	_, err := psc.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (psc *ProvisioningStepCreate) ExecX(ctx context.Context) {
	if err := psc.Exec(ctx); err != nil {
		panic(err)
	}
}

// defaults sets the default values of the builder before save.
func (psc *ProvisioningStepCreate) defaults() {
	if _, ok := psc.mutation.ID(); !ok {
		v := provisioningstep.DefaultID()
		psc.mutation.SetID(v)
	}
}

// check runs all checks and user-defined validators on the builder.
func (psc *ProvisioningStepCreate) check() error {
	if _, ok := psc.mutation.GetType(); !ok {
		return &ValidationError{Name: "type", err: errors.New(`ent: missing required field "ProvisioningStep.type"`)}
	}
	if v, ok := psc.mutation.GetType(); ok {
		if err := provisioningstep.TypeValidator(v); err != nil {
			return &ValidationError{Name: "type", err: fmt.Errorf(`ent: validator failed for field "ProvisioningStep.type": %w`, err)}
		}
	}
	if _, ok := psc.mutation.StepNumber(); !ok {
		return &ValidationError{Name: "step_number", err: errors.New(`ent: missing required field "ProvisioningStep.step_number"`)}
	}
	return nil
}

func (psc *ProvisioningStepCreate) sqlSave(ctx context.Context) (*ProvisioningStep, error) {
	if err := psc.check(); err != nil {
		return nil, err
	}
	_node, _spec := psc.createSpec()
	if err := sqlgraph.CreateNode(ctx, psc.driver, _spec); err != nil {
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
	psc.mutation.id = &_node.ID
	psc.mutation.done = true
	return _node, nil
}

func (psc *ProvisioningStepCreate) createSpec() (*ProvisioningStep, *sqlgraph.CreateSpec) {
	var (
		_node = &ProvisioningStep{config: psc.config}
		_spec = sqlgraph.NewCreateSpec(provisioningstep.Table, sqlgraph.NewFieldSpec(provisioningstep.FieldID, field.TypeUUID))
	)
	if id, ok := psc.mutation.ID(); ok {
		_node.ID = id
		_spec.ID.Value = &id
	}
	if value, ok := psc.mutation.GetType(); ok {
		_spec.SetField(provisioningstep.FieldType, field.TypeEnum, value)
		_node.Type = value
	}
	if value, ok := psc.mutation.StepNumber(); ok {
		_spec.SetField(provisioningstep.FieldStepNumber, field.TypeInt, value)
		_node.StepNumber = value
	}
	if nodes := psc.mutation.StatusIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2O,
			Inverse: false,
			Table:   provisioningstep.StatusTable,
			Columns: []string{provisioningstep.StatusColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(status.FieldID, field.TypeUUID),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges = append(_spec.Edges, edge)
	}
	if nodes := psc.mutation.ProvisionedHostIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: false,
			Table:   provisioningstep.ProvisionedHostTable,
			Columns: []string{provisioningstep.ProvisionedHostColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(provisionedhost.FieldID, field.TypeUUID),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_node.provisioning_step_provisioned_host = &nodes[0]
		_spec.Edges = append(_spec.Edges, edge)
	}
	if nodes := psc.mutation.ScriptIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: false,
			Table:   provisioningstep.ScriptTable,
			Columns: []string{provisioningstep.ScriptColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(script.FieldID, field.TypeUUID),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_node.provisioning_step_script = &nodes[0]
		_spec.Edges = append(_spec.Edges, edge)
	}
	if nodes := psc.mutation.CommandIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: false,
			Table:   provisioningstep.CommandTable,
			Columns: []string{provisioningstep.CommandColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(command.FieldID, field.TypeUUID),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_node.provisioning_step_command = &nodes[0]
		_spec.Edges = append(_spec.Edges, edge)
	}
	if nodes := psc.mutation.DNSRecordIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: false,
			Table:   provisioningstep.DNSRecordTable,
			Columns: []string{provisioningstep.DNSRecordColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(dnsrecord.FieldID, field.TypeUUID),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_node.provisioning_step_dns_record = &nodes[0]
		_spec.Edges = append(_spec.Edges, edge)
	}
	if nodes := psc.mutation.FileDeleteIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: false,
			Table:   provisioningstep.FileDeleteTable,
			Columns: []string{provisioningstep.FileDeleteColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(filedelete.FieldID, field.TypeUUID),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_node.provisioning_step_file_delete = &nodes[0]
		_spec.Edges = append(_spec.Edges, edge)
	}
	if nodes := psc.mutation.FileDownloadIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: false,
			Table:   provisioningstep.FileDownloadTable,
			Columns: []string{provisioningstep.FileDownloadColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(filedownload.FieldID, field.TypeUUID),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_node.provisioning_step_file_download = &nodes[0]
		_spec.Edges = append(_spec.Edges, edge)
	}
	if nodes := psc.mutation.FileExtractIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: false,
			Table:   provisioningstep.FileExtractTable,
			Columns: []string{provisioningstep.FileExtractColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(fileextract.FieldID, field.TypeUUID),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_node.provisioning_step_file_extract = &nodes[0]
		_spec.Edges = append(_spec.Edges, edge)
	}
	if nodes := psc.mutation.AnsibleIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: false,
			Table:   provisioningstep.AnsibleTable,
			Columns: []string{provisioningstep.AnsibleColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(ansible.FieldID, field.TypeUUID),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_node.provisioning_step_ansible = &nodes[0]
		_spec.Edges = append(_spec.Edges, edge)
	}
	if nodes := psc.mutation.PlanIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2O,
			Inverse: true,
			Table:   provisioningstep.PlanTable,
			Columns: []string{provisioningstep.PlanColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(plan.FieldID, field.TypeUUID),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_node.plan_provisioning_step = &nodes[0]
		_spec.Edges = append(_spec.Edges, edge)
	}
	if nodes := psc.mutation.AgentTasksIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: true,
			Table:   provisioningstep.AgentTasksTable,
			Columns: []string{provisioningstep.AgentTasksColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(agenttask.FieldID, field.TypeUUID),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges = append(_spec.Edges, edge)
	}
	if nodes := psc.mutation.GinFileMiddlewareIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2O,
			Inverse: true,
			Table:   provisioningstep.GinFileMiddlewareTable,
			Columns: []string{provisioningstep.GinFileMiddlewareColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(ginfilemiddleware.FieldID, field.TypeUUID),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_node.gin_file_middleware_provisioning_step = &nodes[0]
		_spec.Edges = append(_spec.Edges, edge)
	}
	return _node, _spec
}

// ProvisioningStepCreateBulk is the builder for creating many ProvisioningStep entities in bulk.
type ProvisioningStepCreateBulk struct {
	config
	err      error
	builders []*ProvisioningStepCreate
}

// Save creates the ProvisioningStep entities in the database.
func (pscb *ProvisioningStepCreateBulk) Save(ctx context.Context) ([]*ProvisioningStep, error) {
	if pscb.err != nil {
		return nil, pscb.err
	}
	specs := make([]*sqlgraph.CreateSpec, len(pscb.builders))
	nodes := make([]*ProvisioningStep, len(pscb.builders))
	mutators := make([]Mutator, len(pscb.builders))
	for i := range pscb.builders {
		func(i int, root context.Context) {
			builder := pscb.builders[i]
			builder.defaults()
			var mut Mutator = MutateFunc(func(ctx context.Context, m Mutation) (Value, error) {
				mutation, ok := m.(*ProvisioningStepMutation)
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
					_, err = mutators[i+1].Mutate(root, pscb.builders[i+1].mutation)
				} else {
					spec := &sqlgraph.BatchCreateSpec{Nodes: specs}
					// Invoke the actual operation on the latest mutation in the chain.
					if err = sqlgraph.BatchCreate(ctx, pscb.driver, spec); err != nil {
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
		if _, err := mutators[0].Mutate(ctx, pscb.builders[0].mutation); err != nil {
			return nil, err
		}
	}
	return nodes, nil
}

// SaveX is like Save, but panics if an error occurs.
func (pscb *ProvisioningStepCreateBulk) SaveX(ctx context.Context) []*ProvisioningStep {
	v, err := pscb.Save(ctx)
	if err != nil {
		panic(err)
	}
	return v
}

// Exec executes the query.
func (pscb *ProvisioningStepCreateBulk) Exec(ctx context.Context) error {
	_, err := pscb.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (pscb *ProvisioningStepCreateBulk) ExecX(ctx context.Context) {
	if err := pscb.Exec(ctx); err != nil {
		panic(err)
	}
}
