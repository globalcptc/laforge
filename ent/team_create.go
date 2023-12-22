// Code generated by ent, DO NOT EDIT.

package ent

import (
	"context"
	"errors"
	"fmt"

	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/schema/field"
	"github.com/gen0cide/laforge/ent/build"
	"github.com/gen0cide/laforge/ent/plan"
	"github.com/gen0cide/laforge/ent/provisionednetwork"
	"github.com/gen0cide/laforge/ent/status"
	"github.com/gen0cide/laforge/ent/team"
	"github.com/google/uuid"
)

// TeamCreate is the builder for creating a Team entity.
type TeamCreate struct {
	config
	mutation *TeamMutation
	hooks    []Hook
}

// SetTeamNumber sets the "team_number" field.
func (tc *TeamCreate) SetTeamNumber(i int) *TeamCreate {
	tc.mutation.SetTeamNumber(i)
	return tc
}

// SetVars sets the "vars" field.
func (tc *TeamCreate) SetVars(m map[string]string) *TeamCreate {
	tc.mutation.SetVars(m)
	return tc
}

// SetID sets the "id" field.
func (tc *TeamCreate) SetID(u uuid.UUID) *TeamCreate {
	tc.mutation.SetID(u)
	return tc
}

// SetNillableID sets the "id" field if the given value is not nil.
func (tc *TeamCreate) SetNillableID(u *uuid.UUID) *TeamCreate {
	if u != nil {
		tc.SetID(*u)
	}
	return tc
}

// SetTeamToBuildID sets the "TeamToBuild" edge to the Build entity by ID.
func (tc *TeamCreate) SetTeamToBuildID(id uuid.UUID) *TeamCreate {
	tc.mutation.SetTeamToBuildID(id)
	return tc
}

// SetTeamToBuild sets the "TeamToBuild" edge to the Build entity.
func (tc *TeamCreate) SetTeamToBuild(b *Build) *TeamCreate {
	return tc.SetTeamToBuildID(b.ID)
}

// SetTeamToStatusID sets the "TeamToStatus" edge to the Status entity by ID.
func (tc *TeamCreate) SetTeamToStatusID(id uuid.UUID) *TeamCreate {
	tc.mutation.SetTeamToStatusID(id)
	return tc
}

// SetNillableTeamToStatusID sets the "TeamToStatus" edge to the Status entity by ID if the given value is not nil.
func (tc *TeamCreate) SetNillableTeamToStatusID(id *uuid.UUID) *TeamCreate {
	if id != nil {
		tc = tc.SetTeamToStatusID(*id)
	}
	return tc
}

// SetTeamToStatus sets the "TeamToStatus" edge to the Status entity.
func (tc *TeamCreate) SetTeamToStatus(s *Status) *TeamCreate {
	return tc.SetTeamToStatusID(s.ID)
}

// AddTeamToProvisionedNetworkIDs adds the "TeamToProvisionedNetwork" edge to the ProvisionedNetwork entity by IDs.
func (tc *TeamCreate) AddTeamToProvisionedNetworkIDs(ids ...uuid.UUID) *TeamCreate {
	tc.mutation.AddTeamToProvisionedNetworkIDs(ids...)
	return tc
}

// AddTeamToProvisionedNetwork adds the "TeamToProvisionedNetwork" edges to the ProvisionedNetwork entity.
func (tc *TeamCreate) AddTeamToProvisionedNetwork(p ...*ProvisionedNetwork) *TeamCreate {
	ids := make([]uuid.UUID, len(p))
	for i := range p {
		ids[i] = p[i].ID
	}
	return tc.AddTeamToProvisionedNetworkIDs(ids...)
}

// SetTeamToPlanID sets the "TeamToPlan" edge to the Plan entity by ID.
func (tc *TeamCreate) SetTeamToPlanID(id uuid.UUID) *TeamCreate {
	tc.mutation.SetTeamToPlanID(id)
	return tc
}

// SetNillableTeamToPlanID sets the "TeamToPlan" edge to the Plan entity by ID if the given value is not nil.
func (tc *TeamCreate) SetNillableTeamToPlanID(id *uuid.UUID) *TeamCreate {
	if id != nil {
		tc = tc.SetTeamToPlanID(*id)
	}
	return tc
}

// SetTeamToPlan sets the "TeamToPlan" edge to the Plan entity.
func (tc *TeamCreate) SetTeamToPlan(p *Plan) *TeamCreate {
	return tc.SetTeamToPlanID(p.ID)
}

// Mutation returns the TeamMutation object of the builder.
func (tc *TeamCreate) Mutation() *TeamMutation {
	return tc.mutation
}

// Save creates the Team in the database.
func (tc *TeamCreate) Save(ctx context.Context) (*Team, error) {
	tc.defaults()
	return withHooks(ctx, tc.sqlSave, tc.mutation, tc.hooks)
}

// SaveX calls Save and panics if Save returns an error.
func (tc *TeamCreate) SaveX(ctx context.Context) *Team {
	v, err := tc.Save(ctx)
	if err != nil {
		panic(err)
	}
	return v
}

// Exec executes the query.
func (tc *TeamCreate) Exec(ctx context.Context) error {
	_, err := tc.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (tc *TeamCreate) ExecX(ctx context.Context) {
	if err := tc.Exec(ctx); err != nil {
		panic(err)
	}
}

// defaults sets the default values of the builder before save.
func (tc *TeamCreate) defaults() {
	if _, ok := tc.mutation.ID(); !ok {
		v := team.DefaultID()
		tc.mutation.SetID(v)
	}
}

// check runs all checks and user-defined validators on the builder.
func (tc *TeamCreate) check() error {
	if _, ok := tc.mutation.TeamNumber(); !ok {
		return &ValidationError{Name: "team_number", err: errors.New(`ent: missing required field "Team.team_number"`)}
	}
	if _, ok := tc.mutation.Vars(); !ok {
		return &ValidationError{Name: "vars", err: errors.New(`ent: missing required field "Team.vars"`)}
	}
	if _, ok := tc.mutation.TeamToBuildID(); !ok {
		return &ValidationError{Name: "TeamToBuild", err: errors.New(`ent: missing required edge "Team.TeamToBuild"`)}
	}
	return nil
}

func (tc *TeamCreate) sqlSave(ctx context.Context) (*Team, error) {
	if err := tc.check(); err != nil {
		return nil, err
	}
	_node, _spec := tc.createSpec()
	if err := sqlgraph.CreateNode(ctx, tc.driver, _spec); err != nil {
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
	tc.mutation.id = &_node.ID
	tc.mutation.done = true
	return _node, nil
}

func (tc *TeamCreate) createSpec() (*Team, *sqlgraph.CreateSpec) {
	var (
		_node = &Team{config: tc.config}
		_spec = sqlgraph.NewCreateSpec(team.Table, sqlgraph.NewFieldSpec(team.FieldID, field.TypeUUID))
	)
	if id, ok := tc.mutation.ID(); ok {
		_node.ID = id
		_spec.ID.Value = &id
	}
	if value, ok := tc.mutation.TeamNumber(); ok {
		_spec.SetField(team.FieldTeamNumber, field.TypeInt, value)
		_node.TeamNumber = value
	}
	if value, ok := tc.mutation.Vars(); ok {
		_spec.SetField(team.FieldVars, field.TypeJSON, value)
		_node.Vars = value
	}
	if nodes := tc.mutation.TeamToBuildIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: false,
			Table:   team.TeamToBuildTable,
			Columns: []string{team.TeamToBuildColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(build.FieldID, field.TypeUUID),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_node.team_team_to_build = &nodes[0]
		_spec.Edges = append(_spec.Edges, edge)
	}
	if nodes := tc.mutation.TeamToStatusIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2O,
			Inverse: false,
			Table:   team.TeamToStatusTable,
			Columns: []string{team.TeamToStatusColumn},
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
	if nodes := tc.mutation.TeamToProvisionedNetworkIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: true,
			Table:   team.TeamToProvisionedNetworkTable,
			Columns: []string{team.TeamToProvisionedNetworkColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(provisionednetwork.FieldID, field.TypeUUID),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges = append(_spec.Edges, edge)
	}
	if nodes := tc.mutation.TeamToPlanIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2O,
			Inverse: true,
			Table:   team.TeamToPlanTable,
			Columns: []string{team.TeamToPlanColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(plan.FieldID, field.TypeUUID),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_node.plan_plan_to_team = &nodes[0]
		_spec.Edges = append(_spec.Edges, edge)
	}
	return _node, _spec
}

// TeamCreateBulk is the builder for creating many Team entities in bulk.
type TeamCreateBulk struct {
	config
	err      error
	builders []*TeamCreate
}

// Save creates the Team entities in the database.
func (tcb *TeamCreateBulk) Save(ctx context.Context) ([]*Team, error) {
	if tcb.err != nil {
		return nil, tcb.err
	}
	specs := make([]*sqlgraph.CreateSpec, len(tcb.builders))
	nodes := make([]*Team, len(tcb.builders))
	mutators := make([]Mutator, len(tcb.builders))
	for i := range tcb.builders {
		func(i int, root context.Context) {
			builder := tcb.builders[i]
			builder.defaults()
			var mut Mutator = MutateFunc(func(ctx context.Context, m Mutation) (Value, error) {
				mutation, ok := m.(*TeamMutation)
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
					_, err = mutators[i+1].Mutate(root, tcb.builders[i+1].mutation)
				} else {
					spec := &sqlgraph.BatchCreateSpec{Nodes: specs}
					// Invoke the actual operation on the latest mutation in the chain.
					if err = sqlgraph.BatchCreate(ctx, tcb.driver, spec); err != nil {
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
		if _, err := mutators[0].Mutate(ctx, tcb.builders[0].mutation); err != nil {
			return nil, err
		}
	}
	return nodes, nil
}

// SaveX is like Save, but panics if an error occurs.
func (tcb *TeamCreateBulk) SaveX(ctx context.Context) []*Team {
	v, err := tcb.Save(ctx)
	if err != nil {
		panic(err)
	}
	return v
}

// Exec executes the query.
func (tcb *TeamCreateBulk) Exec(ctx context.Context) error {
	_, err := tcb.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (tcb *TeamCreateBulk) ExecX(ctx context.Context) {
	if err := tcb.Exec(ctx); err != nil {
		panic(err)
	}
}
