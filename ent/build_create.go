// Code generated by entc, DO NOT EDIT.

package ent

import (
	"context"
	"errors"
	"fmt"

	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/schema/field"
	"github.com/gen0cide/laforge/ent/adhocplan"
	"github.com/gen0cide/laforge/ent/build"
	"github.com/gen0cide/laforge/ent/buildcommit"
	"github.com/gen0cide/laforge/ent/competition"
	"github.com/gen0cide/laforge/ent/environment"
	"github.com/gen0cide/laforge/ent/plan"
	"github.com/gen0cide/laforge/ent/provisionednetwork"
	"github.com/gen0cide/laforge/ent/status"
	"github.com/gen0cide/laforge/ent/team"
	"github.com/google/uuid"
)

// BuildCreate is the builder for creating a Build entity.
type BuildCreate struct {
	config
	mutation *BuildMutation
	hooks    []Hook
}

// SetRevision sets the "revision" field.
func (bc *BuildCreate) SetRevision(i int) *BuildCreate {
	bc.mutation.SetRevision(i)
	return bc
}

// SetEnvironmentRevision sets the "environment_revision" field.
func (bc *BuildCreate) SetEnvironmentRevision(i int) *BuildCreate {
	bc.mutation.SetEnvironmentRevision(i)
	return bc
}

// SetCompletedPlan sets the "completed_plan" field.
func (bc *BuildCreate) SetCompletedPlan(b bool) *BuildCreate {
	bc.mutation.SetCompletedPlan(b)
	return bc
}

// SetNillableCompletedPlan sets the "completed_plan" field if the given value is not nil.
func (bc *BuildCreate) SetNillableCompletedPlan(b *bool) *BuildCreate {
	if b != nil {
		bc.SetCompletedPlan(*b)
	}
	return bc
}

// SetID sets the "id" field.
func (bc *BuildCreate) SetID(u uuid.UUID) *BuildCreate {
	bc.mutation.SetID(u)
	return bc
}

// SetBuildToStatusID sets the "BuildToStatus" edge to the Status entity by ID.
func (bc *BuildCreate) SetBuildToStatusID(id uuid.UUID) *BuildCreate {
	bc.mutation.SetBuildToStatusID(id)
	return bc
}

// SetNillableBuildToStatusID sets the "BuildToStatus" edge to the Status entity by ID if the given value is not nil.
func (bc *BuildCreate) SetNillableBuildToStatusID(id *uuid.UUID) *BuildCreate {
	if id != nil {
		bc = bc.SetBuildToStatusID(*id)
	}
	return bc
}

// SetBuildToStatus sets the "BuildToStatus" edge to the Status entity.
func (bc *BuildCreate) SetBuildToStatus(s *Status) *BuildCreate {
	return bc.SetBuildToStatusID(s.ID)
}

// SetBuildToEnvironmentID sets the "BuildToEnvironment" edge to the Environment entity by ID.
func (bc *BuildCreate) SetBuildToEnvironmentID(id uuid.UUID) *BuildCreate {
	bc.mutation.SetBuildToEnvironmentID(id)
	return bc
}

// SetBuildToEnvironment sets the "BuildToEnvironment" edge to the Environment entity.
func (bc *BuildCreate) SetBuildToEnvironment(e *Environment) *BuildCreate {
	return bc.SetBuildToEnvironmentID(e.ID)
}

// SetBuildToCompetitionID sets the "BuildToCompetition" edge to the Competition entity by ID.
func (bc *BuildCreate) SetBuildToCompetitionID(id uuid.UUID) *BuildCreate {
	bc.mutation.SetBuildToCompetitionID(id)
	return bc
}

// SetBuildToCompetition sets the "BuildToCompetition" edge to the Competition entity.
func (bc *BuildCreate) SetBuildToCompetition(c *Competition) *BuildCreate {
	return bc.SetBuildToCompetitionID(c.ID)
}

// SetBuildToLatestBuildCommitID sets the "BuildToLatestBuildCommit" edge to the BuildCommit entity by ID.
func (bc *BuildCreate) SetBuildToLatestBuildCommitID(id uuid.UUID) *BuildCreate {
	bc.mutation.SetBuildToLatestBuildCommitID(id)
	return bc
}

// SetNillableBuildToLatestBuildCommitID sets the "BuildToLatestBuildCommit" edge to the BuildCommit entity by ID if the given value is not nil.
func (bc *BuildCreate) SetNillableBuildToLatestBuildCommitID(id *uuid.UUID) *BuildCreate {
	if id != nil {
		bc = bc.SetBuildToLatestBuildCommitID(*id)
	}
	return bc
}

// SetBuildToLatestBuildCommit sets the "BuildToLatestBuildCommit" edge to the BuildCommit entity.
func (bc *BuildCreate) SetBuildToLatestBuildCommit(b *BuildCommit) *BuildCreate {
	return bc.SetBuildToLatestBuildCommitID(b.ID)
}

// AddBuildToProvisionedNetworkIDs adds the "BuildToProvisionedNetwork" edge to the ProvisionedNetwork entity by IDs.
func (bc *BuildCreate) AddBuildToProvisionedNetworkIDs(ids ...uuid.UUID) *BuildCreate {
	bc.mutation.AddBuildToProvisionedNetworkIDs(ids...)
	return bc
}

// AddBuildToProvisionedNetwork adds the "BuildToProvisionedNetwork" edges to the ProvisionedNetwork entity.
func (bc *BuildCreate) AddBuildToProvisionedNetwork(p ...*ProvisionedNetwork) *BuildCreate {
	ids := make([]uuid.UUID, len(p))
	for i := range p {
		ids[i] = p[i].ID
	}
	return bc.AddBuildToProvisionedNetworkIDs(ids...)
}

// AddBuildToTeamIDs adds the "BuildToTeam" edge to the Team entity by IDs.
func (bc *BuildCreate) AddBuildToTeamIDs(ids ...uuid.UUID) *BuildCreate {
	bc.mutation.AddBuildToTeamIDs(ids...)
	return bc
}

// AddBuildToTeam adds the "BuildToTeam" edges to the Team entity.
func (bc *BuildCreate) AddBuildToTeam(t ...*Team) *BuildCreate {
	ids := make([]uuid.UUID, len(t))
	for i := range t {
		ids[i] = t[i].ID
	}
	return bc.AddBuildToTeamIDs(ids...)
}

// AddBuildToPlanIDs adds the "BuildToPlan" edge to the Plan entity by IDs.
func (bc *BuildCreate) AddBuildToPlanIDs(ids ...uuid.UUID) *BuildCreate {
	bc.mutation.AddBuildToPlanIDs(ids...)
	return bc
}

// AddBuildToPlan adds the "BuildToPlan" edges to the Plan entity.
func (bc *BuildCreate) AddBuildToPlan(p ...*Plan) *BuildCreate {
	ids := make([]uuid.UUID, len(p))
	for i := range p {
		ids[i] = p[i].ID
	}
	return bc.AddBuildToPlanIDs(ids...)
}

// AddBuildToBuildCommitIDs adds the "BuildToBuildCommits" edge to the BuildCommit entity by IDs.
func (bc *BuildCreate) AddBuildToBuildCommitIDs(ids ...uuid.UUID) *BuildCreate {
	bc.mutation.AddBuildToBuildCommitIDs(ids...)
	return bc
}

// AddBuildToBuildCommits adds the "BuildToBuildCommits" edges to the BuildCommit entity.
func (bc *BuildCreate) AddBuildToBuildCommits(b ...*BuildCommit) *BuildCreate {
	ids := make([]uuid.UUID, len(b))
	for i := range b {
		ids[i] = b[i].ID
	}
	return bc.AddBuildToBuildCommitIDs(ids...)
}

// AddBuildToAdhocPlanIDs adds the "BuildToAdhocPlans" edge to the AdhocPlan entity by IDs.
func (bc *BuildCreate) AddBuildToAdhocPlanIDs(ids ...uuid.UUID) *BuildCreate {
	bc.mutation.AddBuildToAdhocPlanIDs(ids...)
	return bc
}

// AddBuildToAdhocPlans adds the "BuildToAdhocPlans" edges to the AdhocPlan entity.
func (bc *BuildCreate) AddBuildToAdhocPlans(a ...*AdhocPlan) *BuildCreate {
	ids := make([]uuid.UUID, len(a))
	for i := range a {
		ids[i] = a[i].ID
	}
	return bc.AddBuildToAdhocPlanIDs(ids...)
}

// Mutation returns the BuildMutation object of the builder.
func (bc *BuildCreate) Mutation() *BuildMutation {
	return bc.mutation
}

// Save creates the Build in the database.
func (bc *BuildCreate) Save(ctx context.Context) (*Build, error) {
	var (
		err  error
		node *Build
	)
	bc.defaults()
	if len(bc.hooks) == 0 {
		if err = bc.check(); err != nil {
			return nil, err
		}
		node, err = bc.sqlSave(ctx)
	} else {
		var mut Mutator = MutateFunc(func(ctx context.Context, m Mutation) (Value, error) {
			mutation, ok := m.(*BuildMutation)
			if !ok {
				return nil, fmt.Errorf("unexpected mutation type %T", m)
			}
			if err = bc.check(); err != nil {
				return nil, err
			}
			bc.mutation = mutation
			if node, err = bc.sqlSave(ctx); err != nil {
				return nil, err
			}
			mutation.id = &node.ID
			mutation.done = true
			return node, err
		})
		for i := len(bc.hooks) - 1; i >= 0; i-- {
			if bc.hooks[i] == nil {
				return nil, fmt.Errorf("ent: uninitialized hook (forgotten import ent/runtime?)")
			}
			mut = bc.hooks[i](mut)
		}
		if _, err := mut.Mutate(ctx, bc.mutation); err != nil {
			return nil, err
		}
	}
	return node, err
}

// SaveX calls Save and panics if Save returns an error.
func (bc *BuildCreate) SaveX(ctx context.Context) *Build {
	v, err := bc.Save(ctx)
	if err != nil {
		panic(err)
	}
	return v
}

// Exec executes the query.
func (bc *BuildCreate) Exec(ctx context.Context) error {
	_, err := bc.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (bc *BuildCreate) ExecX(ctx context.Context) {
	if err := bc.Exec(ctx); err != nil {
		panic(err)
	}
}

// defaults sets the default values of the builder before save.
func (bc *BuildCreate) defaults() {
	if _, ok := bc.mutation.CompletedPlan(); !ok {
		v := build.DefaultCompletedPlan
		bc.mutation.SetCompletedPlan(v)
	}
	if _, ok := bc.mutation.ID(); !ok {
		v := build.DefaultID()
		bc.mutation.SetID(v)
	}
}

// check runs all checks and user-defined validators on the builder.
func (bc *BuildCreate) check() error {
	if _, ok := bc.mutation.Revision(); !ok {
		return &ValidationError{Name: "revision", err: errors.New(`ent: missing required field "revision"`)}
	}
	if _, ok := bc.mutation.EnvironmentRevision(); !ok {
		return &ValidationError{Name: "environment_revision", err: errors.New(`ent: missing required field "environment_revision"`)}
	}
	if _, ok := bc.mutation.CompletedPlan(); !ok {
		return &ValidationError{Name: "completed_plan", err: errors.New(`ent: missing required field "completed_plan"`)}
	}
	if _, ok := bc.mutation.BuildToEnvironmentID(); !ok {
		return &ValidationError{Name: "BuildToEnvironment", err: errors.New("ent: missing required edge \"BuildToEnvironment\"")}
	}
	if _, ok := bc.mutation.BuildToCompetitionID(); !ok {
		return &ValidationError{Name: "BuildToCompetition", err: errors.New("ent: missing required edge \"BuildToCompetition\"")}
	}
	return nil
}

func (bc *BuildCreate) sqlSave(ctx context.Context) (*Build, error) {
	_node, _spec := bc.createSpec()
	if err := sqlgraph.CreateNode(ctx, bc.driver, _spec); err != nil {
		if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{err.Error(), err}
		}
		return nil, err
	}
	if _spec.ID.Value != nil {
		_node.ID = _spec.ID.Value.(uuid.UUID)
	}
	return _node, nil
}

func (bc *BuildCreate) createSpec() (*Build, *sqlgraph.CreateSpec) {
	var (
		_node = &Build{config: bc.config}
		_spec = &sqlgraph.CreateSpec{
			Table: build.Table,
			ID: &sqlgraph.FieldSpec{
				Type:   field.TypeUUID,
				Column: build.FieldID,
			},
		}
	)
	if id, ok := bc.mutation.ID(); ok {
		_node.ID = id
		_spec.ID.Value = id
	}
	if value, ok := bc.mutation.Revision(); ok {
		_spec.Fields = append(_spec.Fields, &sqlgraph.FieldSpec{
			Type:   field.TypeInt,
			Value:  value,
			Column: build.FieldRevision,
		})
		_node.Revision = value
	}
	if value, ok := bc.mutation.EnvironmentRevision(); ok {
		_spec.Fields = append(_spec.Fields, &sqlgraph.FieldSpec{
			Type:   field.TypeInt,
			Value:  value,
			Column: build.FieldEnvironmentRevision,
		})
		_node.EnvironmentRevision = value
	}
	if value, ok := bc.mutation.CompletedPlan(); ok {
		_spec.Fields = append(_spec.Fields, &sqlgraph.FieldSpec{
			Type:   field.TypeBool,
			Value:  value,
			Column: build.FieldCompletedPlan,
		})
		_node.CompletedPlan = value
	}
	if nodes := bc.mutation.BuildToStatusIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2O,
			Inverse: false,
			Table:   build.BuildToStatusTable,
			Columns: []string{build.BuildToStatusColumn},
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
	if nodes := bc.mutation.BuildToEnvironmentIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: false,
			Table:   build.BuildToEnvironmentTable,
			Columns: []string{build.BuildToEnvironmentColumn},
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
		_node.build_build_to_environment = &nodes[0]
		_spec.Edges = append(_spec.Edges, edge)
	}
	if nodes := bc.mutation.BuildToCompetitionIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: false,
			Table:   build.BuildToCompetitionTable,
			Columns: []string{build.BuildToCompetitionColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeUUID,
					Column: competition.FieldID,
				},
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_node.build_build_to_competition = &nodes[0]
		_spec.Edges = append(_spec.Edges, edge)
	}
	if nodes := bc.mutation.BuildToLatestBuildCommitIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: false,
			Table:   build.BuildToLatestBuildCommitTable,
			Columns: []string{build.BuildToLatestBuildCommitColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeUUID,
					Column: buildcommit.FieldID,
				},
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_node.build_build_to_latest_build_commit = &nodes[0]
		_spec.Edges = append(_spec.Edges, edge)
	}
	if nodes := bc.mutation.BuildToProvisionedNetworkIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: true,
			Table:   build.BuildToProvisionedNetworkTable,
			Columns: []string{build.BuildToProvisionedNetworkColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeUUID,
					Column: provisionednetwork.FieldID,
				},
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges = append(_spec.Edges, edge)
	}
	if nodes := bc.mutation.BuildToTeamIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: true,
			Table:   build.BuildToTeamTable,
			Columns: []string{build.BuildToTeamColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeUUID,
					Column: team.FieldID,
				},
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges = append(_spec.Edges, edge)
	}
	if nodes := bc.mutation.BuildToPlanIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: true,
			Table:   build.BuildToPlanTable,
			Columns: []string{build.BuildToPlanColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeUUID,
					Column: plan.FieldID,
				},
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges = append(_spec.Edges, edge)
	}
	if nodes := bc.mutation.BuildToBuildCommitsIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: true,
			Table:   build.BuildToBuildCommitsTable,
			Columns: []string{build.BuildToBuildCommitsColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeUUID,
					Column: buildcommit.FieldID,
				},
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges = append(_spec.Edges, edge)
	}
	if nodes := bc.mutation.BuildToAdhocPlansIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: true,
			Table:   build.BuildToAdhocPlansTable,
			Columns: []string{build.BuildToAdhocPlansColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeUUID,
					Column: adhocplan.FieldID,
				},
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges = append(_spec.Edges, edge)
	}
	return _node, _spec
}

// BuildCreateBulk is the builder for creating many Build entities in bulk.
type BuildCreateBulk struct {
	config
	builders []*BuildCreate
}

// Save creates the Build entities in the database.
func (bcb *BuildCreateBulk) Save(ctx context.Context) ([]*Build, error) {
	specs := make([]*sqlgraph.CreateSpec, len(bcb.builders))
	nodes := make([]*Build, len(bcb.builders))
	mutators := make([]Mutator, len(bcb.builders))
	for i := range bcb.builders {
		func(i int, root context.Context) {
			builder := bcb.builders[i]
			builder.defaults()
			var mut Mutator = MutateFunc(func(ctx context.Context, m Mutation) (Value, error) {
				mutation, ok := m.(*BuildMutation)
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
					_, err = mutators[i+1].Mutate(root, bcb.builders[i+1].mutation)
				} else {
					spec := &sqlgraph.BatchCreateSpec{Nodes: specs}
					// Invoke the actual operation on the latest mutation in the chain.
					if err = sqlgraph.BatchCreate(ctx, bcb.driver, spec); err != nil {
						if sqlgraph.IsConstraintError(err) {
							err = &ConstraintError{err.Error(), err}
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
		if _, err := mutators[0].Mutate(ctx, bcb.builders[0].mutation); err != nil {
			return nil, err
		}
	}
	return nodes, nil
}

// SaveX is like Save, but panics if an error occurs.
func (bcb *BuildCreateBulk) SaveX(ctx context.Context) []*Build {
	v, err := bcb.Save(ctx)
	if err != nil {
		panic(err)
	}
	return v
}

// Exec executes the query.
func (bcb *BuildCreateBulk) Exec(ctx context.Context) error {
	_, err := bcb.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (bcb *BuildCreateBulk) ExecX(ctx context.Context) {
	if err := bcb.Exec(ctx); err != nil {
		panic(err)
	}
}
