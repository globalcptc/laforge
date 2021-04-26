// Code generated by entc, DO NOT EDIT.

package ent

import (
	"context"
	"fmt"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/schema/field"
	"github.com/gen0cide/laforge/ent/build"
	"github.com/gen0cide/laforge/ent/plan"
	"github.com/gen0cide/laforge/ent/predicate"
	"github.com/gen0cide/laforge/ent/provisionedhost"
	"github.com/gen0cide/laforge/ent/provisionednetwork"
	"github.com/gen0cide/laforge/ent/provisioningstep"
	"github.com/gen0cide/laforge/ent/team"
)

// PlanUpdate is the builder for updating Plan entities.
type PlanUpdate struct {
	config
	hooks    []Hook
	mutation *PlanMutation
}

// Where adds a new predicate for the PlanUpdate builder.
func (pu *PlanUpdate) Where(ps ...predicate.Plan) *PlanUpdate {
	pu.mutation.predicates = append(pu.mutation.predicates, ps...)
	return pu
}

// SetStepNumber sets the "step_number" field.
func (pu *PlanUpdate) SetStepNumber(i int) *PlanUpdate {
	pu.mutation.ResetStepNumber()
	pu.mutation.SetStepNumber(i)
	return pu
}

// AddStepNumber adds i to the "step_number" field.
func (pu *PlanUpdate) AddStepNumber(i int) *PlanUpdate {
	pu.mutation.AddStepNumber(i)
	return pu
}

// SetType sets the "type" field.
func (pu *PlanUpdate) SetType(pl plan.Type) *PlanUpdate {
	pu.mutation.SetType(pl)
	return pu
}

// SetBuildID sets the "build_id" field.
func (pu *PlanUpdate) SetBuildID(i int) *PlanUpdate {
	pu.mutation.ResetBuildID()
	pu.mutation.SetBuildID(i)
	return pu
}

// AddBuildID adds i to the "build_id" field.
func (pu *PlanUpdate) AddBuildID(i int) *PlanUpdate {
	pu.mutation.AddBuildID(i)
	return pu
}

// AddPrevPlanIDs adds the "PrevPlan" edge to the Plan entity by IDs.
func (pu *PlanUpdate) AddPrevPlanIDs(ids ...int) *PlanUpdate {
	pu.mutation.AddPrevPlanIDs(ids...)
	return pu
}

// AddPrevPlan adds the "PrevPlan" edges to the Plan entity.
func (pu *PlanUpdate) AddPrevPlan(p ...*Plan) *PlanUpdate {
	ids := make([]int, len(p))
	for i := range p {
		ids[i] = p[i].ID
	}
	return pu.AddPrevPlanIDs(ids...)
}

// AddNextPlanIDs adds the "NextPlan" edge to the Plan entity by IDs.
func (pu *PlanUpdate) AddNextPlanIDs(ids ...int) *PlanUpdate {
	pu.mutation.AddNextPlanIDs(ids...)
	return pu
}

// AddNextPlan adds the "NextPlan" edges to the Plan entity.
func (pu *PlanUpdate) AddNextPlan(p ...*Plan) *PlanUpdate {
	ids := make([]int, len(p))
	for i := range p {
		ids[i] = p[i].ID
	}
	return pu.AddNextPlanIDs(ids...)
}

// SetPlanToBuildID sets the "PlanToBuild" edge to the Build entity by ID.
func (pu *PlanUpdate) SetPlanToBuildID(id int) *PlanUpdate {
	pu.mutation.SetPlanToBuildID(id)
	return pu
}

// SetNillablePlanToBuildID sets the "PlanToBuild" edge to the Build entity by ID if the given value is not nil.
func (pu *PlanUpdate) SetNillablePlanToBuildID(id *int) *PlanUpdate {
	if id != nil {
		pu = pu.SetPlanToBuildID(*id)
	}
	return pu
}

// SetPlanToBuild sets the "PlanToBuild" edge to the Build entity.
func (pu *PlanUpdate) SetPlanToBuild(b *Build) *PlanUpdate {
	return pu.SetPlanToBuildID(b.ID)
}

// SetPlanToTeamID sets the "PlanToTeam" edge to the Team entity by ID.
func (pu *PlanUpdate) SetPlanToTeamID(id int) *PlanUpdate {
	pu.mutation.SetPlanToTeamID(id)
	return pu
}

// SetNillablePlanToTeamID sets the "PlanToTeam" edge to the Team entity by ID if the given value is not nil.
func (pu *PlanUpdate) SetNillablePlanToTeamID(id *int) *PlanUpdate {
	if id != nil {
		pu = pu.SetPlanToTeamID(*id)
	}
	return pu
}

// SetPlanToTeam sets the "PlanToTeam" edge to the Team entity.
func (pu *PlanUpdate) SetPlanToTeam(t *Team) *PlanUpdate {
	return pu.SetPlanToTeamID(t.ID)
}

// SetPlanToProvisionedNetworkID sets the "PlanToProvisionedNetwork" edge to the ProvisionedNetwork entity by ID.
func (pu *PlanUpdate) SetPlanToProvisionedNetworkID(id int) *PlanUpdate {
	pu.mutation.SetPlanToProvisionedNetworkID(id)
	return pu
}

// SetNillablePlanToProvisionedNetworkID sets the "PlanToProvisionedNetwork" edge to the ProvisionedNetwork entity by ID if the given value is not nil.
func (pu *PlanUpdate) SetNillablePlanToProvisionedNetworkID(id *int) *PlanUpdate {
	if id != nil {
		pu = pu.SetPlanToProvisionedNetworkID(*id)
	}
	return pu
}

// SetPlanToProvisionedNetwork sets the "PlanToProvisionedNetwork" edge to the ProvisionedNetwork entity.
func (pu *PlanUpdate) SetPlanToProvisionedNetwork(p *ProvisionedNetwork) *PlanUpdate {
	return pu.SetPlanToProvisionedNetworkID(p.ID)
}

// SetPlanToProvisionedHostID sets the "PlanToProvisionedHost" edge to the ProvisionedHost entity by ID.
func (pu *PlanUpdate) SetPlanToProvisionedHostID(id int) *PlanUpdate {
	pu.mutation.SetPlanToProvisionedHostID(id)
	return pu
}

// SetNillablePlanToProvisionedHostID sets the "PlanToProvisionedHost" edge to the ProvisionedHost entity by ID if the given value is not nil.
func (pu *PlanUpdate) SetNillablePlanToProvisionedHostID(id *int) *PlanUpdate {
	if id != nil {
		pu = pu.SetPlanToProvisionedHostID(*id)
	}
	return pu
}

// SetPlanToProvisionedHost sets the "PlanToProvisionedHost" edge to the ProvisionedHost entity.
func (pu *PlanUpdate) SetPlanToProvisionedHost(p *ProvisionedHost) *PlanUpdate {
	return pu.SetPlanToProvisionedHostID(p.ID)
}

// SetPlanToProvisioningStepID sets the "PlanToProvisioningStep" edge to the ProvisioningStep entity by ID.
func (pu *PlanUpdate) SetPlanToProvisioningStepID(id int) *PlanUpdate {
	pu.mutation.SetPlanToProvisioningStepID(id)
	return pu
}

// SetNillablePlanToProvisioningStepID sets the "PlanToProvisioningStep" edge to the ProvisioningStep entity by ID if the given value is not nil.
func (pu *PlanUpdate) SetNillablePlanToProvisioningStepID(id *int) *PlanUpdate {
	if id != nil {
		pu = pu.SetPlanToProvisioningStepID(*id)
	}
	return pu
}

// SetPlanToProvisioningStep sets the "PlanToProvisioningStep" edge to the ProvisioningStep entity.
func (pu *PlanUpdate) SetPlanToProvisioningStep(p *ProvisioningStep) *PlanUpdate {
	return pu.SetPlanToProvisioningStepID(p.ID)
}

// Mutation returns the PlanMutation object of the builder.
func (pu *PlanUpdate) Mutation() *PlanMutation {
	return pu.mutation
}

// ClearPrevPlan clears all "PrevPlan" edges to the Plan entity.
func (pu *PlanUpdate) ClearPrevPlan() *PlanUpdate {
	pu.mutation.ClearPrevPlan()
	return pu
}

// RemovePrevPlanIDs removes the "PrevPlan" edge to Plan entities by IDs.
func (pu *PlanUpdate) RemovePrevPlanIDs(ids ...int) *PlanUpdate {
	pu.mutation.RemovePrevPlanIDs(ids...)
	return pu
}

// RemovePrevPlan removes "PrevPlan" edges to Plan entities.
func (pu *PlanUpdate) RemovePrevPlan(p ...*Plan) *PlanUpdate {
	ids := make([]int, len(p))
	for i := range p {
		ids[i] = p[i].ID
	}
	return pu.RemovePrevPlanIDs(ids...)
}

// ClearNextPlan clears all "NextPlan" edges to the Plan entity.
func (pu *PlanUpdate) ClearNextPlan() *PlanUpdate {
	pu.mutation.ClearNextPlan()
	return pu
}

// RemoveNextPlanIDs removes the "NextPlan" edge to Plan entities by IDs.
func (pu *PlanUpdate) RemoveNextPlanIDs(ids ...int) *PlanUpdate {
	pu.mutation.RemoveNextPlanIDs(ids...)
	return pu
}

// RemoveNextPlan removes "NextPlan" edges to Plan entities.
func (pu *PlanUpdate) RemoveNextPlan(p ...*Plan) *PlanUpdate {
	ids := make([]int, len(p))
	for i := range p {
		ids[i] = p[i].ID
	}
	return pu.RemoveNextPlanIDs(ids...)
}

// ClearPlanToBuild clears the "PlanToBuild" edge to the Build entity.
func (pu *PlanUpdate) ClearPlanToBuild() *PlanUpdate {
	pu.mutation.ClearPlanToBuild()
	return pu
}

// ClearPlanToTeam clears the "PlanToTeam" edge to the Team entity.
func (pu *PlanUpdate) ClearPlanToTeam() *PlanUpdate {
	pu.mutation.ClearPlanToTeam()
	return pu
}

// ClearPlanToProvisionedNetwork clears the "PlanToProvisionedNetwork" edge to the ProvisionedNetwork entity.
func (pu *PlanUpdate) ClearPlanToProvisionedNetwork() *PlanUpdate {
	pu.mutation.ClearPlanToProvisionedNetwork()
	return pu
}

// ClearPlanToProvisionedHost clears the "PlanToProvisionedHost" edge to the ProvisionedHost entity.
func (pu *PlanUpdate) ClearPlanToProvisionedHost() *PlanUpdate {
	pu.mutation.ClearPlanToProvisionedHost()
	return pu
}

// ClearPlanToProvisioningStep clears the "PlanToProvisioningStep" edge to the ProvisioningStep entity.
func (pu *PlanUpdate) ClearPlanToProvisioningStep() *PlanUpdate {
	pu.mutation.ClearPlanToProvisioningStep()
	return pu
}

// Save executes the query and returns the number of nodes affected by the update operation.
func (pu *PlanUpdate) Save(ctx context.Context) (int, error) {
	var (
		err      error
		affected int
	)
	if len(pu.hooks) == 0 {
		if err = pu.check(); err != nil {
			return 0, err
		}
		affected, err = pu.sqlSave(ctx)
	} else {
		var mut Mutator = MutateFunc(func(ctx context.Context, m Mutation) (Value, error) {
			mutation, ok := m.(*PlanMutation)
			if !ok {
				return nil, fmt.Errorf("unexpected mutation type %T", m)
			}
			if err = pu.check(); err != nil {
				return 0, err
			}
			pu.mutation = mutation
			affected, err = pu.sqlSave(ctx)
			mutation.done = true
			return affected, err
		})
		for i := len(pu.hooks) - 1; i >= 0; i-- {
			mut = pu.hooks[i](mut)
		}
		if _, err := mut.Mutate(ctx, pu.mutation); err != nil {
			return 0, err
		}
	}
	return affected, err
}

// SaveX is like Save, but panics if an error occurs.
func (pu *PlanUpdate) SaveX(ctx context.Context) int {
	affected, err := pu.Save(ctx)
	if err != nil {
		panic(err)
	}
	return affected
}

// Exec executes the query.
func (pu *PlanUpdate) Exec(ctx context.Context) error {
	_, err := pu.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (pu *PlanUpdate) ExecX(ctx context.Context) {
	if err := pu.Exec(ctx); err != nil {
		panic(err)
	}
}

// check runs all checks and user-defined validators on the builder.
func (pu *PlanUpdate) check() error {
	if v, ok := pu.mutation.GetType(); ok {
		if err := plan.TypeValidator(v); err != nil {
			return &ValidationError{Name: "type", err: fmt.Errorf("ent: validator failed for field \"type\": %w", err)}
		}
	}
	return nil
}

func (pu *PlanUpdate) sqlSave(ctx context.Context) (n int, err error) {
	_spec := &sqlgraph.UpdateSpec{
		Node: &sqlgraph.NodeSpec{
			Table:   plan.Table,
			Columns: plan.Columns,
			ID: &sqlgraph.FieldSpec{
				Type:   field.TypeInt,
				Column: plan.FieldID,
			},
		},
	}
	if ps := pu.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if value, ok := pu.mutation.StepNumber(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeInt,
			Value:  value,
			Column: plan.FieldStepNumber,
		})
	}
	if value, ok := pu.mutation.AddedStepNumber(); ok {
		_spec.Fields.Add = append(_spec.Fields.Add, &sqlgraph.FieldSpec{
			Type:   field.TypeInt,
			Value:  value,
			Column: plan.FieldStepNumber,
		})
	}
	if value, ok := pu.mutation.GetType(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeEnum,
			Value:  value,
			Column: plan.FieldType,
		})
	}
	if value, ok := pu.mutation.BuildID(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeInt,
			Value:  value,
			Column: plan.FieldBuildID,
		})
	}
	if value, ok := pu.mutation.AddedBuildID(); ok {
		_spec.Fields.Add = append(_spec.Fields.Add, &sqlgraph.FieldSpec{
			Type:   field.TypeInt,
			Value:  value,
			Column: plan.FieldBuildID,
		})
	}
	if pu.mutation.PrevPlanCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2M,
			Inverse: true,
			Table:   plan.PrevPlanTable,
			Columns: plan.PrevPlanPrimaryKey,
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeInt,
					Column: plan.FieldID,
				},
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := pu.mutation.RemovedPrevPlanIDs(); len(nodes) > 0 && !pu.mutation.PrevPlanCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2M,
			Inverse: true,
			Table:   plan.PrevPlanTable,
			Columns: plan.PrevPlanPrimaryKey,
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeInt,
					Column: plan.FieldID,
				},
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := pu.mutation.PrevPlanIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2M,
			Inverse: true,
			Table:   plan.PrevPlanTable,
			Columns: plan.PrevPlanPrimaryKey,
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeInt,
					Column: plan.FieldID,
				},
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	if pu.mutation.NextPlanCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2M,
			Inverse: false,
			Table:   plan.NextPlanTable,
			Columns: plan.NextPlanPrimaryKey,
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeInt,
					Column: plan.FieldID,
				},
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := pu.mutation.RemovedNextPlanIDs(); len(nodes) > 0 && !pu.mutation.NextPlanCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2M,
			Inverse: false,
			Table:   plan.NextPlanTable,
			Columns: plan.NextPlanPrimaryKey,
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeInt,
					Column: plan.FieldID,
				},
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := pu.mutation.NextPlanIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2M,
			Inverse: false,
			Table:   plan.NextPlanTable,
			Columns: plan.NextPlanPrimaryKey,
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeInt,
					Column: plan.FieldID,
				},
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	if pu.mutation.PlanToBuildCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: false,
			Table:   plan.PlanToBuildTable,
			Columns: []string{plan.PlanToBuildColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeInt,
					Column: build.FieldID,
				},
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := pu.mutation.PlanToBuildIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: false,
			Table:   plan.PlanToBuildTable,
			Columns: []string{plan.PlanToBuildColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeInt,
					Column: build.FieldID,
				},
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	if pu.mutation.PlanToTeamCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2O,
			Inverse: false,
			Table:   plan.PlanToTeamTable,
			Columns: []string{plan.PlanToTeamColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeInt,
					Column: team.FieldID,
				},
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := pu.mutation.PlanToTeamIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2O,
			Inverse: false,
			Table:   plan.PlanToTeamTable,
			Columns: []string{plan.PlanToTeamColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeInt,
					Column: team.FieldID,
				},
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	if pu.mutation.PlanToProvisionedNetworkCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2O,
			Inverse: false,
			Table:   plan.PlanToProvisionedNetworkTable,
			Columns: []string{plan.PlanToProvisionedNetworkColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeInt,
					Column: provisionednetwork.FieldID,
				},
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := pu.mutation.PlanToProvisionedNetworkIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2O,
			Inverse: false,
			Table:   plan.PlanToProvisionedNetworkTable,
			Columns: []string{plan.PlanToProvisionedNetworkColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeInt,
					Column: provisionednetwork.FieldID,
				},
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	if pu.mutation.PlanToProvisionedHostCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2O,
			Inverse: false,
			Table:   plan.PlanToProvisionedHostTable,
			Columns: []string{plan.PlanToProvisionedHostColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeInt,
					Column: provisionedhost.FieldID,
				},
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := pu.mutation.PlanToProvisionedHostIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2O,
			Inverse: false,
			Table:   plan.PlanToProvisionedHostTable,
			Columns: []string{plan.PlanToProvisionedHostColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeInt,
					Column: provisionedhost.FieldID,
				},
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	if pu.mutation.PlanToProvisioningStepCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2O,
			Inverse: false,
			Table:   plan.PlanToProvisioningStepTable,
			Columns: []string{plan.PlanToProvisioningStepColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeInt,
					Column: provisioningstep.FieldID,
				},
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := pu.mutation.PlanToProvisioningStepIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2O,
			Inverse: false,
			Table:   plan.PlanToProvisioningStepTable,
			Columns: []string{plan.PlanToProvisioningStepColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeInt,
					Column: provisioningstep.FieldID,
				},
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	if n, err = sqlgraph.UpdateNodes(ctx, pu.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{plan.Label}
		} else if cerr, ok := isSQLConstraintError(err); ok {
			err = cerr
		}
		return 0, err
	}
	return n, nil
}

// PlanUpdateOne is the builder for updating a single Plan entity.
type PlanUpdateOne struct {
	config
	hooks    []Hook
	mutation *PlanMutation
}

// SetStepNumber sets the "step_number" field.
func (puo *PlanUpdateOne) SetStepNumber(i int) *PlanUpdateOne {
	puo.mutation.ResetStepNumber()
	puo.mutation.SetStepNumber(i)
	return puo
}

// AddStepNumber adds i to the "step_number" field.
func (puo *PlanUpdateOne) AddStepNumber(i int) *PlanUpdateOne {
	puo.mutation.AddStepNumber(i)
	return puo
}

// SetType sets the "type" field.
func (puo *PlanUpdateOne) SetType(pl plan.Type) *PlanUpdateOne {
	puo.mutation.SetType(pl)
	return puo
}

// SetBuildID sets the "build_id" field.
func (puo *PlanUpdateOne) SetBuildID(i int) *PlanUpdateOne {
	puo.mutation.ResetBuildID()
	puo.mutation.SetBuildID(i)
	return puo
}

// AddBuildID adds i to the "build_id" field.
func (puo *PlanUpdateOne) AddBuildID(i int) *PlanUpdateOne {
	puo.mutation.AddBuildID(i)
	return puo
}

// AddPrevPlanIDs adds the "PrevPlan" edge to the Plan entity by IDs.
func (puo *PlanUpdateOne) AddPrevPlanIDs(ids ...int) *PlanUpdateOne {
	puo.mutation.AddPrevPlanIDs(ids...)
	return puo
}

// AddPrevPlan adds the "PrevPlan" edges to the Plan entity.
func (puo *PlanUpdateOne) AddPrevPlan(p ...*Plan) *PlanUpdateOne {
	ids := make([]int, len(p))
	for i := range p {
		ids[i] = p[i].ID
	}
	return puo.AddPrevPlanIDs(ids...)
}

// AddNextPlanIDs adds the "NextPlan" edge to the Plan entity by IDs.
func (puo *PlanUpdateOne) AddNextPlanIDs(ids ...int) *PlanUpdateOne {
	puo.mutation.AddNextPlanIDs(ids...)
	return puo
}

// AddNextPlan adds the "NextPlan" edges to the Plan entity.
func (puo *PlanUpdateOne) AddNextPlan(p ...*Plan) *PlanUpdateOne {
	ids := make([]int, len(p))
	for i := range p {
		ids[i] = p[i].ID
	}
	return puo.AddNextPlanIDs(ids...)
}

// SetPlanToBuildID sets the "PlanToBuild" edge to the Build entity by ID.
func (puo *PlanUpdateOne) SetPlanToBuildID(id int) *PlanUpdateOne {
	puo.mutation.SetPlanToBuildID(id)
	return puo
}

// SetNillablePlanToBuildID sets the "PlanToBuild" edge to the Build entity by ID if the given value is not nil.
func (puo *PlanUpdateOne) SetNillablePlanToBuildID(id *int) *PlanUpdateOne {
	if id != nil {
		puo = puo.SetPlanToBuildID(*id)
	}
	return puo
}

// SetPlanToBuild sets the "PlanToBuild" edge to the Build entity.
func (puo *PlanUpdateOne) SetPlanToBuild(b *Build) *PlanUpdateOne {
	return puo.SetPlanToBuildID(b.ID)
}

// SetPlanToTeamID sets the "PlanToTeam" edge to the Team entity by ID.
func (puo *PlanUpdateOne) SetPlanToTeamID(id int) *PlanUpdateOne {
	puo.mutation.SetPlanToTeamID(id)
	return puo
}

// SetNillablePlanToTeamID sets the "PlanToTeam" edge to the Team entity by ID if the given value is not nil.
func (puo *PlanUpdateOne) SetNillablePlanToTeamID(id *int) *PlanUpdateOne {
	if id != nil {
		puo = puo.SetPlanToTeamID(*id)
	}
	return puo
}

// SetPlanToTeam sets the "PlanToTeam" edge to the Team entity.
func (puo *PlanUpdateOne) SetPlanToTeam(t *Team) *PlanUpdateOne {
	return puo.SetPlanToTeamID(t.ID)
}

// SetPlanToProvisionedNetworkID sets the "PlanToProvisionedNetwork" edge to the ProvisionedNetwork entity by ID.
func (puo *PlanUpdateOne) SetPlanToProvisionedNetworkID(id int) *PlanUpdateOne {
	puo.mutation.SetPlanToProvisionedNetworkID(id)
	return puo
}

// SetNillablePlanToProvisionedNetworkID sets the "PlanToProvisionedNetwork" edge to the ProvisionedNetwork entity by ID if the given value is not nil.
func (puo *PlanUpdateOne) SetNillablePlanToProvisionedNetworkID(id *int) *PlanUpdateOne {
	if id != nil {
		puo = puo.SetPlanToProvisionedNetworkID(*id)
	}
	return puo
}

// SetPlanToProvisionedNetwork sets the "PlanToProvisionedNetwork" edge to the ProvisionedNetwork entity.
func (puo *PlanUpdateOne) SetPlanToProvisionedNetwork(p *ProvisionedNetwork) *PlanUpdateOne {
	return puo.SetPlanToProvisionedNetworkID(p.ID)
}

// SetPlanToProvisionedHostID sets the "PlanToProvisionedHost" edge to the ProvisionedHost entity by ID.
func (puo *PlanUpdateOne) SetPlanToProvisionedHostID(id int) *PlanUpdateOne {
	puo.mutation.SetPlanToProvisionedHostID(id)
	return puo
}

// SetNillablePlanToProvisionedHostID sets the "PlanToProvisionedHost" edge to the ProvisionedHost entity by ID if the given value is not nil.
func (puo *PlanUpdateOne) SetNillablePlanToProvisionedHostID(id *int) *PlanUpdateOne {
	if id != nil {
		puo = puo.SetPlanToProvisionedHostID(*id)
	}
	return puo
}

// SetPlanToProvisionedHost sets the "PlanToProvisionedHost" edge to the ProvisionedHost entity.
func (puo *PlanUpdateOne) SetPlanToProvisionedHost(p *ProvisionedHost) *PlanUpdateOne {
	return puo.SetPlanToProvisionedHostID(p.ID)
}

// SetPlanToProvisioningStepID sets the "PlanToProvisioningStep" edge to the ProvisioningStep entity by ID.
func (puo *PlanUpdateOne) SetPlanToProvisioningStepID(id int) *PlanUpdateOne {
	puo.mutation.SetPlanToProvisioningStepID(id)
	return puo
}

// SetNillablePlanToProvisioningStepID sets the "PlanToProvisioningStep" edge to the ProvisioningStep entity by ID if the given value is not nil.
func (puo *PlanUpdateOne) SetNillablePlanToProvisioningStepID(id *int) *PlanUpdateOne {
	if id != nil {
		puo = puo.SetPlanToProvisioningStepID(*id)
	}
	return puo
}

// SetPlanToProvisioningStep sets the "PlanToProvisioningStep" edge to the ProvisioningStep entity.
func (puo *PlanUpdateOne) SetPlanToProvisioningStep(p *ProvisioningStep) *PlanUpdateOne {
	return puo.SetPlanToProvisioningStepID(p.ID)
}

// Mutation returns the PlanMutation object of the builder.
func (puo *PlanUpdateOne) Mutation() *PlanMutation {
	return puo.mutation
}

// ClearPrevPlan clears all "PrevPlan" edges to the Plan entity.
func (puo *PlanUpdateOne) ClearPrevPlan() *PlanUpdateOne {
	puo.mutation.ClearPrevPlan()
	return puo
}

// RemovePrevPlanIDs removes the "PrevPlan" edge to Plan entities by IDs.
func (puo *PlanUpdateOne) RemovePrevPlanIDs(ids ...int) *PlanUpdateOne {
	puo.mutation.RemovePrevPlanIDs(ids...)
	return puo
}

// RemovePrevPlan removes "PrevPlan" edges to Plan entities.
func (puo *PlanUpdateOne) RemovePrevPlan(p ...*Plan) *PlanUpdateOne {
	ids := make([]int, len(p))
	for i := range p {
		ids[i] = p[i].ID
	}
	return puo.RemovePrevPlanIDs(ids...)
}

// ClearNextPlan clears all "NextPlan" edges to the Plan entity.
func (puo *PlanUpdateOne) ClearNextPlan() *PlanUpdateOne {
	puo.mutation.ClearNextPlan()
	return puo
}

// RemoveNextPlanIDs removes the "NextPlan" edge to Plan entities by IDs.
func (puo *PlanUpdateOne) RemoveNextPlanIDs(ids ...int) *PlanUpdateOne {
	puo.mutation.RemoveNextPlanIDs(ids...)
	return puo
}

// RemoveNextPlan removes "NextPlan" edges to Plan entities.
func (puo *PlanUpdateOne) RemoveNextPlan(p ...*Plan) *PlanUpdateOne {
	ids := make([]int, len(p))
	for i := range p {
		ids[i] = p[i].ID
	}
	return puo.RemoveNextPlanIDs(ids...)
}

// ClearPlanToBuild clears the "PlanToBuild" edge to the Build entity.
func (puo *PlanUpdateOne) ClearPlanToBuild() *PlanUpdateOne {
	puo.mutation.ClearPlanToBuild()
	return puo
}

// ClearPlanToTeam clears the "PlanToTeam" edge to the Team entity.
func (puo *PlanUpdateOne) ClearPlanToTeam() *PlanUpdateOne {
	puo.mutation.ClearPlanToTeam()
	return puo
}

// ClearPlanToProvisionedNetwork clears the "PlanToProvisionedNetwork" edge to the ProvisionedNetwork entity.
func (puo *PlanUpdateOne) ClearPlanToProvisionedNetwork() *PlanUpdateOne {
	puo.mutation.ClearPlanToProvisionedNetwork()
	return puo
}

// ClearPlanToProvisionedHost clears the "PlanToProvisionedHost" edge to the ProvisionedHost entity.
func (puo *PlanUpdateOne) ClearPlanToProvisionedHost() *PlanUpdateOne {
	puo.mutation.ClearPlanToProvisionedHost()
	return puo
}

// ClearPlanToProvisioningStep clears the "PlanToProvisioningStep" edge to the ProvisioningStep entity.
func (puo *PlanUpdateOne) ClearPlanToProvisioningStep() *PlanUpdateOne {
	puo.mutation.ClearPlanToProvisioningStep()
	return puo
}

// Save executes the query and returns the updated Plan entity.
func (puo *PlanUpdateOne) Save(ctx context.Context) (*Plan, error) {
	var (
		err  error
		node *Plan
	)
	if len(puo.hooks) == 0 {
		if err = puo.check(); err != nil {
			return nil, err
		}
		node, err = puo.sqlSave(ctx)
	} else {
		var mut Mutator = MutateFunc(func(ctx context.Context, m Mutation) (Value, error) {
			mutation, ok := m.(*PlanMutation)
			if !ok {
				return nil, fmt.Errorf("unexpected mutation type %T", m)
			}
			if err = puo.check(); err != nil {
				return nil, err
			}
			puo.mutation = mutation
			node, err = puo.sqlSave(ctx)
			mutation.done = true
			return node, err
		})
		for i := len(puo.hooks) - 1; i >= 0; i-- {
			mut = puo.hooks[i](mut)
		}
		if _, err := mut.Mutate(ctx, puo.mutation); err != nil {
			return nil, err
		}
	}
	return node, err
}

// SaveX is like Save, but panics if an error occurs.
func (puo *PlanUpdateOne) SaveX(ctx context.Context) *Plan {
	node, err := puo.Save(ctx)
	if err != nil {
		panic(err)
	}
	return node
}

// Exec executes the query on the entity.
func (puo *PlanUpdateOne) Exec(ctx context.Context) error {
	_, err := puo.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (puo *PlanUpdateOne) ExecX(ctx context.Context) {
	if err := puo.Exec(ctx); err != nil {
		panic(err)
	}
}

// check runs all checks and user-defined validators on the builder.
func (puo *PlanUpdateOne) check() error {
	if v, ok := puo.mutation.GetType(); ok {
		if err := plan.TypeValidator(v); err != nil {
			return &ValidationError{Name: "type", err: fmt.Errorf("ent: validator failed for field \"type\": %w", err)}
		}
	}
	return nil
}

func (puo *PlanUpdateOne) sqlSave(ctx context.Context) (_node *Plan, err error) {
	_spec := &sqlgraph.UpdateSpec{
		Node: &sqlgraph.NodeSpec{
			Table:   plan.Table,
			Columns: plan.Columns,
			ID: &sqlgraph.FieldSpec{
				Type:   field.TypeInt,
				Column: plan.FieldID,
			},
		},
	}
	id, ok := puo.mutation.ID()
	if !ok {
		return nil, &ValidationError{Name: "ID", err: fmt.Errorf("missing Plan.ID for update")}
	}
	_spec.Node.ID.Value = id
	if ps := puo.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if value, ok := puo.mutation.StepNumber(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeInt,
			Value:  value,
			Column: plan.FieldStepNumber,
		})
	}
	if value, ok := puo.mutation.AddedStepNumber(); ok {
		_spec.Fields.Add = append(_spec.Fields.Add, &sqlgraph.FieldSpec{
			Type:   field.TypeInt,
			Value:  value,
			Column: plan.FieldStepNumber,
		})
	}
	if value, ok := puo.mutation.GetType(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeEnum,
			Value:  value,
			Column: plan.FieldType,
		})
	}
	if value, ok := puo.mutation.BuildID(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeInt,
			Value:  value,
			Column: plan.FieldBuildID,
		})
	}
	if value, ok := puo.mutation.AddedBuildID(); ok {
		_spec.Fields.Add = append(_spec.Fields.Add, &sqlgraph.FieldSpec{
			Type:   field.TypeInt,
			Value:  value,
			Column: plan.FieldBuildID,
		})
	}
	if puo.mutation.PrevPlanCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2M,
			Inverse: true,
			Table:   plan.PrevPlanTable,
			Columns: plan.PrevPlanPrimaryKey,
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeInt,
					Column: plan.FieldID,
				},
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := puo.mutation.RemovedPrevPlanIDs(); len(nodes) > 0 && !puo.mutation.PrevPlanCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2M,
			Inverse: true,
			Table:   plan.PrevPlanTable,
			Columns: plan.PrevPlanPrimaryKey,
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeInt,
					Column: plan.FieldID,
				},
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := puo.mutation.PrevPlanIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2M,
			Inverse: true,
			Table:   plan.PrevPlanTable,
			Columns: plan.PrevPlanPrimaryKey,
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeInt,
					Column: plan.FieldID,
				},
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	if puo.mutation.NextPlanCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2M,
			Inverse: false,
			Table:   plan.NextPlanTable,
			Columns: plan.NextPlanPrimaryKey,
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeInt,
					Column: plan.FieldID,
				},
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := puo.mutation.RemovedNextPlanIDs(); len(nodes) > 0 && !puo.mutation.NextPlanCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2M,
			Inverse: false,
			Table:   plan.NextPlanTable,
			Columns: plan.NextPlanPrimaryKey,
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeInt,
					Column: plan.FieldID,
				},
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := puo.mutation.NextPlanIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2M,
			Inverse: false,
			Table:   plan.NextPlanTable,
			Columns: plan.NextPlanPrimaryKey,
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeInt,
					Column: plan.FieldID,
				},
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	if puo.mutation.PlanToBuildCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: false,
			Table:   plan.PlanToBuildTable,
			Columns: []string{plan.PlanToBuildColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeInt,
					Column: build.FieldID,
				},
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := puo.mutation.PlanToBuildIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: false,
			Table:   plan.PlanToBuildTable,
			Columns: []string{plan.PlanToBuildColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeInt,
					Column: build.FieldID,
				},
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	if puo.mutation.PlanToTeamCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2O,
			Inverse: false,
			Table:   plan.PlanToTeamTable,
			Columns: []string{plan.PlanToTeamColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeInt,
					Column: team.FieldID,
				},
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := puo.mutation.PlanToTeamIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2O,
			Inverse: false,
			Table:   plan.PlanToTeamTable,
			Columns: []string{plan.PlanToTeamColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeInt,
					Column: team.FieldID,
				},
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	if puo.mutation.PlanToProvisionedNetworkCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2O,
			Inverse: false,
			Table:   plan.PlanToProvisionedNetworkTable,
			Columns: []string{plan.PlanToProvisionedNetworkColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeInt,
					Column: provisionednetwork.FieldID,
				},
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := puo.mutation.PlanToProvisionedNetworkIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2O,
			Inverse: false,
			Table:   plan.PlanToProvisionedNetworkTable,
			Columns: []string{plan.PlanToProvisionedNetworkColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeInt,
					Column: provisionednetwork.FieldID,
				},
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	if puo.mutation.PlanToProvisionedHostCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2O,
			Inverse: false,
			Table:   plan.PlanToProvisionedHostTable,
			Columns: []string{plan.PlanToProvisionedHostColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeInt,
					Column: provisionedhost.FieldID,
				},
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := puo.mutation.PlanToProvisionedHostIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2O,
			Inverse: false,
			Table:   plan.PlanToProvisionedHostTable,
			Columns: []string{plan.PlanToProvisionedHostColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeInt,
					Column: provisionedhost.FieldID,
				},
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	if puo.mutation.PlanToProvisioningStepCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2O,
			Inverse: false,
			Table:   plan.PlanToProvisioningStepTable,
			Columns: []string{plan.PlanToProvisioningStepColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeInt,
					Column: provisioningstep.FieldID,
				},
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := puo.mutation.PlanToProvisioningStepIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2O,
			Inverse: false,
			Table:   plan.PlanToProvisioningStepTable,
			Columns: []string{plan.PlanToProvisioningStepColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeInt,
					Column: provisioningstep.FieldID,
				},
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	_node = &Plan{config: puo.config}
	_spec.Assign = _node.assignValues
	_spec.ScanValues = _node.scanValues
	if err = sqlgraph.UpdateNode(ctx, puo.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{plan.Label}
		} else if cerr, ok := isSQLConstraintError(err); ok {
			err = cerr
		}
		return nil, err
	}
	return _node, nil
}