// Code generated by entc, DO NOT EDIT.

package ent

import (
	"context"
	"fmt"
	"time"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/schema/field"
	"github.com/gen0cide/laforge/ent/build"
	"github.com/gen0cide/laforge/ent/predicate"
	"github.com/gen0cide/laforge/ent/provisionedhost"
	"github.com/gen0cide/laforge/ent/provisionednetwork"
	"github.com/gen0cide/laforge/ent/provisioningstep"
	"github.com/gen0cide/laforge/ent/status"
	"github.com/gen0cide/laforge/ent/team"
	"github.com/google/uuid"
)

// StatusUpdate is the builder for updating Status entities.
type StatusUpdate struct {
	config
	hooks    []Hook
	mutation *StatusMutation
}

// Where adds a new predicate for the StatusUpdate builder.
func (su *StatusUpdate) Where(ps ...predicate.Status) *StatusUpdate {
	su.mutation.predicates = append(su.mutation.predicates, ps...)
	return su
}

// SetState sets the "state" field.
func (su *StatusUpdate) SetState(s status.State) *StatusUpdate {
	su.mutation.SetState(s)
	return su
}

// SetStatusFor sets the "status_for" field.
func (su *StatusUpdate) SetStatusFor(sf status.StatusFor) *StatusUpdate {
	su.mutation.SetStatusFor(sf)
	return su
}

// SetStartedAt sets the "started_at" field.
func (su *StatusUpdate) SetStartedAt(t time.Time) *StatusUpdate {
	su.mutation.SetStartedAt(t)
	return su
}

// SetNillableStartedAt sets the "started_at" field if the given value is not nil.
func (su *StatusUpdate) SetNillableStartedAt(t *time.Time) *StatusUpdate {
	if t != nil {
		su.SetStartedAt(*t)
	}
	return su
}

// ClearStartedAt clears the value of the "started_at" field.
func (su *StatusUpdate) ClearStartedAt() *StatusUpdate {
	su.mutation.ClearStartedAt()
	return su
}

// SetEndedAt sets the "ended_at" field.
func (su *StatusUpdate) SetEndedAt(t time.Time) *StatusUpdate {
	su.mutation.SetEndedAt(t)
	return su
}

// SetNillableEndedAt sets the "ended_at" field if the given value is not nil.
func (su *StatusUpdate) SetNillableEndedAt(t *time.Time) *StatusUpdate {
	if t != nil {
		su.SetEndedAt(*t)
	}
	return su
}

// ClearEndedAt clears the value of the "ended_at" field.
func (su *StatusUpdate) ClearEndedAt() *StatusUpdate {
	su.mutation.ClearEndedAt()
	return su
}

// SetFailed sets the "failed" field.
func (su *StatusUpdate) SetFailed(b bool) *StatusUpdate {
	su.mutation.SetFailed(b)
	return su
}

// SetNillableFailed sets the "failed" field if the given value is not nil.
func (su *StatusUpdate) SetNillableFailed(b *bool) *StatusUpdate {
	if b != nil {
		su.SetFailed(*b)
	}
	return su
}

// SetCompleted sets the "completed" field.
func (su *StatusUpdate) SetCompleted(b bool) *StatusUpdate {
	su.mutation.SetCompleted(b)
	return su
}

// SetNillableCompleted sets the "completed" field if the given value is not nil.
func (su *StatusUpdate) SetNillableCompleted(b *bool) *StatusUpdate {
	if b != nil {
		su.SetCompleted(*b)
	}
	return su
}

// SetError sets the "error" field.
func (su *StatusUpdate) SetError(s string) *StatusUpdate {
	su.mutation.SetError(s)
	return su
}

// SetNillableError sets the "error" field if the given value is not nil.
func (su *StatusUpdate) SetNillableError(s *string) *StatusUpdate {
	if s != nil {
		su.SetError(*s)
	}
	return su
}

// ClearError clears the value of the "error" field.
func (su *StatusUpdate) ClearError() *StatusUpdate {
	su.mutation.ClearError()
	return su
}

// SetStatusToBuildID sets the "StatusToBuild" edge to the Build entity by ID.
func (su *StatusUpdate) SetStatusToBuildID(id uuid.UUID) *StatusUpdate {
	su.mutation.SetStatusToBuildID(id)
	return su
}

// SetNillableStatusToBuildID sets the "StatusToBuild" edge to the Build entity by ID if the given value is not nil.
func (su *StatusUpdate) SetNillableStatusToBuildID(id *uuid.UUID) *StatusUpdate {
	if id != nil {
		su = su.SetStatusToBuildID(*id)
	}
	return su
}

// SetStatusToBuild sets the "StatusToBuild" edge to the Build entity.
func (su *StatusUpdate) SetStatusToBuild(b *Build) *StatusUpdate {
	return su.SetStatusToBuildID(b.ID)
}

// SetStatusToProvisionedNetworkID sets the "StatusToProvisionedNetwork" edge to the ProvisionedNetwork entity by ID.
func (su *StatusUpdate) SetStatusToProvisionedNetworkID(id uuid.UUID) *StatusUpdate {
	su.mutation.SetStatusToProvisionedNetworkID(id)
	return su
}

// SetNillableStatusToProvisionedNetworkID sets the "StatusToProvisionedNetwork" edge to the ProvisionedNetwork entity by ID if the given value is not nil.
func (su *StatusUpdate) SetNillableStatusToProvisionedNetworkID(id *uuid.UUID) *StatusUpdate {
	if id != nil {
		su = su.SetStatusToProvisionedNetworkID(*id)
	}
	return su
}

// SetStatusToProvisionedNetwork sets the "StatusToProvisionedNetwork" edge to the ProvisionedNetwork entity.
func (su *StatusUpdate) SetStatusToProvisionedNetwork(p *ProvisionedNetwork) *StatusUpdate {
	return su.SetStatusToProvisionedNetworkID(p.ID)
}

// SetStatusToProvisionedHostID sets the "StatusToProvisionedHost" edge to the ProvisionedHost entity by ID.
func (su *StatusUpdate) SetStatusToProvisionedHostID(id uuid.UUID) *StatusUpdate {
	su.mutation.SetStatusToProvisionedHostID(id)
	return su
}

// SetNillableStatusToProvisionedHostID sets the "StatusToProvisionedHost" edge to the ProvisionedHost entity by ID if the given value is not nil.
func (su *StatusUpdate) SetNillableStatusToProvisionedHostID(id *uuid.UUID) *StatusUpdate {
	if id != nil {
		su = su.SetStatusToProvisionedHostID(*id)
	}
	return su
}

// SetStatusToProvisionedHost sets the "StatusToProvisionedHost" edge to the ProvisionedHost entity.
func (su *StatusUpdate) SetStatusToProvisionedHost(p *ProvisionedHost) *StatusUpdate {
	return su.SetStatusToProvisionedHostID(p.ID)
}

// SetStatusToProvisioningStepID sets the "StatusToProvisioningStep" edge to the ProvisioningStep entity by ID.
func (su *StatusUpdate) SetStatusToProvisioningStepID(id uuid.UUID) *StatusUpdate {
	su.mutation.SetStatusToProvisioningStepID(id)
	return su
}

// SetNillableStatusToProvisioningStepID sets the "StatusToProvisioningStep" edge to the ProvisioningStep entity by ID if the given value is not nil.
func (su *StatusUpdate) SetNillableStatusToProvisioningStepID(id *uuid.UUID) *StatusUpdate {
	if id != nil {
		su = su.SetStatusToProvisioningStepID(*id)
	}
	return su
}

// SetStatusToProvisioningStep sets the "StatusToProvisioningStep" edge to the ProvisioningStep entity.
func (su *StatusUpdate) SetStatusToProvisioningStep(p *ProvisioningStep) *StatusUpdate {
	return su.SetStatusToProvisioningStepID(p.ID)
}

// SetStatusToTeamID sets the "StatusToTeam" edge to the Team entity by ID.
func (su *StatusUpdate) SetStatusToTeamID(id uuid.UUID) *StatusUpdate {
	su.mutation.SetStatusToTeamID(id)
	return su
}

// SetNillableStatusToTeamID sets the "StatusToTeam" edge to the Team entity by ID if the given value is not nil.
func (su *StatusUpdate) SetNillableStatusToTeamID(id *uuid.UUID) *StatusUpdate {
	if id != nil {
		su = su.SetStatusToTeamID(*id)
	}
	return su
}

// SetStatusToTeam sets the "StatusToTeam" edge to the Team entity.
func (su *StatusUpdate) SetStatusToTeam(t *Team) *StatusUpdate {
	return su.SetStatusToTeamID(t.ID)
}

// Mutation returns the StatusMutation object of the builder.
func (su *StatusUpdate) Mutation() *StatusMutation {
	return su.mutation
}

// ClearStatusToBuild clears the "StatusToBuild" edge to the Build entity.
func (su *StatusUpdate) ClearStatusToBuild() *StatusUpdate {
	su.mutation.ClearStatusToBuild()
	return su
}

// ClearStatusToProvisionedNetwork clears the "StatusToProvisionedNetwork" edge to the ProvisionedNetwork entity.
func (su *StatusUpdate) ClearStatusToProvisionedNetwork() *StatusUpdate {
	su.mutation.ClearStatusToProvisionedNetwork()
	return su
}

// ClearStatusToProvisionedHost clears the "StatusToProvisionedHost" edge to the ProvisionedHost entity.
func (su *StatusUpdate) ClearStatusToProvisionedHost() *StatusUpdate {
	su.mutation.ClearStatusToProvisionedHost()
	return su
}

// ClearStatusToProvisioningStep clears the "StatusToProvisioningStep" edge to the ProvisioningStep entity.
func (su *StatusUpdate) ClearStatusToProvisioningStep() *StatusUpdate {
	su.mutation.ClearStatusToProvisioningStep()
	return su
}

// ClearStatusToTeam clears the "StatusToTeam" edge to the Team entity.
func (su *StatusUpdate) ClearStatusToTeam() *StatusUpdate {
	su.mutation.ClearStatusToTeam()
	return su
}

// Save executes the query and returns the number of nodes affected by the update operation.
func (su *StatusUpdate) Save(ctx context.Context) (int, error) {
	var (
		err      error
		affected int
	)
	if len(su.hooks) == 0 {
		if err = su.check(); err != nil {
			return 0, err
		}
		affected, err = su.sqlSave(ctx)
	} else {
		var mut Mutator = MutateFunc(func(ctx context.Context, m Mutation) (Value, error) {
			mutation, ok := m.(*StatusMutation)
			if !ok {
				return nil, fmt.Errorf("unexpected mutation type %T", m)
			}
			if err = su.check(); err != nil {
				return 0, err
			}
			su.mutation = mutation
			affected, err = su.sqlSave(ctx)
			mutation.done = true
			return affected, err
		})
		for i := len(su.hooks) - 1; i >= 0; i-- {
			mut = su.hooks[i](mut)
		}
		if _, err := mut.Mutate(ctx, su.mutation); err != nil {
			return 0, err
		}
	}
	return affected, err
}

// SaveX is like Save, but panics if an error occurs.
func (su *StatusUpdate) SaveX(ctx context.Context) int {
	affected, err := su.Save(ctx)
	if err != nil {
		panic(err)
	}
	return affected
}

// Exec executes the query.
func (su *StatusUpdate) Exec(ctx context.Context) error {
	_, err := su.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (su *StatusUpdate) ExecX(ctx context.Context) {
	if err := su.Exec(ctx); err != nil {
		panic(err)
	}
}

// check runs all checks and user-defined validators on the builder.
func (su *StatusUpdate) check() error {
	if v, ok := su.mutation.State(); ok {
		if err := status.StateValidator(v); err != nil {
			return &ValidationError{Name: "state", err: fmt.Errorf("ent: validator failed for field \"state\": %w", err)}
		}
	}
	if v, ok := su.mutation.StatusFor(); ok {
		if err := status.StatusForValidator(v); err != nil {
			return &ValidationError{Name: "status_for", err: fmt.Errorf("ent: validator failed for field \"status_for\": %w", err)}
		}
	}
	return nil
}

func (su *StatusUpdate) sqlSave(ctx context.Context) (n int, err error) {
	_spec := &sqlgraph.UpdateSpec{
		Node: &sqlgraph.NodeSpec{
			Table:   status.Table,
			Columns: status.Columns,
			ID: &sqlgraph.FieldSpec{
				Type:   field.TypeUUID,
				Column: status.FieldID,
			},
		},
	}
	if ps := su.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if value, ok := su.mutation.State(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeEnum,
			Value:  value,
			Column: status.FieldState,
		})
	}
	if value, ok := su.mutation.StatusFor(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeEnum,
			Value:  value,
			Column: status.FieldStatusFor,
		})
	}
	if value, ok := su.mutation.StartedAt(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeTime,
			Value:  value,
			Column: status.FieldStartedAt,
		})
	}
	if su.mutation.StartedAtCleared() {
		_spec.Fields.Clear = append(_spec.Fields.Clear, &sqlgraph.FieldSpec{
			Type:   field.TypeTime,
			Column: status.FieldStartedAt,
		})
	}
	if value, ok := su.mutation.EndedAt(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeTime,
			Value:  value,
			Column: status.FieldEndedAt,
		})
	}
	if su.mutation.EndedAtCleared() {
		_spec.Fields.Clear = append(_spec.Fields.Clear, &sqlgraph.FieldSpec{
			Type:   field.TypeTime,
			Column: status.FieldEndedAt,
		})
	}
	if value, ok := su.mutation.Failed(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeBool,
			Value:  value,
			Column: status.FieldFailed,
		})
	}
	if value, ok := su.mutation.Completed(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeBool,
			Value:  value,
			Column: status.FieldCompleted,
		})
	}
	if value, ok := su.mutation.Error(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Value:  value,
			Column: status.FieldError,
		})
	}
	if su.mutation.ErrorCleared() {
		_spec.Fields.Clear = append(_spec.Fields.Clear, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Column: status.FieldError,
		})
	}
	if su.mutation.StatusToBuildCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2O,
			Inverse: true,
			Table:   status.StatusToBuildTable,
			Columns: []string{status.StatusToBuildColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeUUID,
					Column: build.FieldID,
				},
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := su.mutation.StatusToBuildIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2O,
			Inverse: true,
			Table:   status.StatusToBuildTable,
			Columns: []string{status.StatusToBuildColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeUUID,
					Column: build.FieldID,
				},
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	if su.mutation.StatusToProvisionedNetworkCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2O,
			Inverse: true,
			Table:   status.StatusToProvisionedNetworkTable,
			Columns: []string{status.StatusToProvisionedNetworkColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeUUID,
					Column: provisionednetwork.FieldID,
				},
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := su.mutation.StatusToProvisionedNetworkIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2O,
			Inverse: true,
			Table:   status.StatusToProvisionedNetworkTable,
			Columns: []string{status.StatusToProvisionedNetworkColumn},
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
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	if su.mutation.StatusToProvisionedHostCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2O,
			Inverse: true,
			Table:   status.StatusToProvisionedHostTable,
			Columns: []string{status.StatusToProvisionedHostColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeUUID,
					Column: provisionedhost.FieldID,
				},
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := su.mutation.StatusToProvisionedHostIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2O,
			Inverse: true,
			Table:   status.StatusToProvisionedHostTable,
			Columns: []string{status.StatusToProvisionedHostColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeUUID,
					Column: provisionedhost.FieldID,
				},
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	if su.mutation.StatusToProvisioningStepCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2O,
			Inverse: true,
			Table:   status.StatusToProvisioningStepTable,
			Columns: []string{status.StatusToProvisioningStepColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeUUID,
					Column: provisioningstep.FieldID,
				},
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := su.mutation.StatusToProvisioningStepIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2O,
			Inverse: true,
			Table:   status.StatusToProvisioningStepTable,
			Columns: []string{status.StatusToProvisioningStepColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeUUID,
					Column: provisioningstep.FieldID,
				},
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	if su.mutation.StatusToTeamCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2O,
			Inverse: true,
			Table:   status.StatusToTeamTable,
			Columns: []string{status.StatusToTeamColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeUUID,
					Column: team.FieldID,
				},
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := su.mutation.StatusToTeamIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2O,
			Inverse: true,
			Table:   status.StatusToTeamTable,
			Columns: []string{status.StatusToTeamColumn},
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
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	if n, err = sqlgraph.UpdateNodes(ctx, su.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{status.Label}
		} else if cerr, ok := isSQLConstraintError(err); ok {
			err = cerr
		}
		return 0, err
	}
	return n, nil
}

// StatusUpdateOne is the builder for updating a single Status entity.
type StatusUpdateOne struct {
	config
	fields   []string
	hooks    []Hook
	mutation *StatusMutation
}

// SetState sets the "state" field.
func (suo *StatusUpdateOne) SetState(s status.State) *StatusUpdateOne {
	suo.mutation.SetState(s)
	return suo
}

// SetStatusFor sets the "status_for" field.
func (suo *StatusUpdateOne) SetStatusFor(sf status.StatusFor) *StatusUpdateOne {
	suo.mutation.SetStatusFor(sf)
	return suo
}

// SetStartedAt sets the "started_at" field.
func (suo *StatusUpdateOne) SetStartedAt(t time.Time) *StatusUpdateOne {
	suo.mutation.SetStartedAt(t)
	return suo
}

// SetNillableStartedAt sets the "started_at" field if the given value is not nil.
func (suo *StatusUpdateOne) SetNillableStartedAt(t *time.Time) *StatusUpdateOne {
	if t != nil {
		suo.SetStartedAt(*t)
	}
	return suo
}

// ClearStartedAt clears the value of the "started_at" field.
func (suo *StatusUpdateOne) ClearStartedAt() *StatusUpdateOne {
	suo.mutation.ClearStartedAt()
	return suo
}

// SetEndedAt sets the "ended_at" field.
func (suo *StatusUpdateOne) SetEndedAt(t time.Time) *StatusUpdateOne {
	suo.mutation.SetEndedAt(t)
	return suo
}

// SetNillableEndedAt sets the "ended_at" field if the given value is not nil.
func (suo *StatusUpdateOne) SetNillableEndedAt(t *time.Time) *StatusUpdateOne {
	if t != nil {
		suo.SetEndedAt(*t)
	}
	return suo
}

// ClearEndedAt clears the value of the "ended_at" field.
func (suo *StatusUpdateOne) ClearEndedAt() *StatusUpdateOne {
	suo.mutation.ClearEndedAt()
	return suo
}

// SetFailed sets the "failed" field.
func (suo *StatusUpdateOne) SetFailed(b bool) *StatusUpdateOne {
	suo.mutation.SetFailed(b)
	return suo
}

// SetNillableFailed sets the "failed" field if the given value is not nil.
func (suo *StatusUpdateOne) SetNillableFailed(b *bool) *StatusUpdateOne {
	if b != nil {
		suo.SetFailed(*b)
	}
	return suo
}

// SetCompleted sets the "completed" field.
func (suo *StatusUpdateOne) SetCompleted(b bool) *StatusUpdateOne {
	suo.mutation.SetCompleted(b)
	return suo
}

// SetNillableCompleted sets the "completed" field if the given value is not nil.
func (suo *StatusUpdateOne) SetNillableCompleted(b *bool) *StatusUpdateOne {
	if b != nil {
		suo.SetCompleted(*b)
	}
	return suo
}

// SetError sets the "error" field.
func (suo *StatusUpdateOne) SetError(s string) *StatusUpdateOne {
	suo.mutation.SetError(s)
	return suo
}

// SetNillableError sets the "error" field if the given value is not nil.
func (suo *StatusUpdateOne) SetNillableError(s *string) *StatusUpdateOne {
	if s != nil {
		suo.SetError(*s)
	}
	return suo
}

// ClearError clears the value of the "error" field.
func (suo *StatusUpdateOne) ClearError() *StatusUpdateOne {
	suo.mutation.ClearError()
	return suo
}

// SetStatusToBuildID sets the "StatusToBuild" edge to the Build entity by ID.
func (suo *StatusUpdateOne) SetStatusToBuildID(id uuid.UUID) *StatusUpdateOne {
	suo.mutation.SetStatusToBuildID(id)
	return suo
}

// SetNillableStatusToBuildID sets the "StatusToBuild" edge to the Build entity by ID if the given value is not nil.
func (suo *StatusUpdateOne) SetNillableStatusToBuildID(id *uuid.UUID) *StatusUpdateOne {
	if id != nil {
		suo = suo.SetStatusToBuildID(*id)
	}
	return suo
}

// SetStatusToBuild sets the "StatusToBuild" edge to the Build entity.
func (suo *StatusUpdateOne) SetStatusToBuild(b *Build) *StatusUpdateOne {
	return suo.SetStatusToBuildID(b.ID)
}

// SetStatusToProvisionedNetworkID sets the "StatusToProvisionedNetwork" edge to the ProvisionedNetwork entity by ID.
func (suo *StatusUpdateOne) SetStatusToProvisionedNetworkID(id uuid.UUID) *StatusUpdateOne {
	suo.mutation.SetStatusToProvisionedNetworkID(id)
	return suo
}

// SetNillableStatusToProvisionedNetworkID sets the "StatusToProvisionedNetwork" edge to the ProvisionedNetwork entity by ID if the given value is not nil.
func (suo *StatusUpdateOne) SetNillableStatusToProvisionedNetworkID(id *uuid.UUID) *StatusUpdateOne {
	if id != nil {
		suo = suo.SetStatusToProvisionedNetworkID(*id)
	}
	return suo
}

// SetStatusToProvisionedNetwork sets the "StatusToProvisionedNetwork" edge to the ProvisionedNetwork entity.
func (suo *StatusUpdateOne) SetStatusToProvisionedNetwork(p *ProvisionedNetwork) *StatusUpdateOne {
	return suo.SetStatusToProvisionedNetworkID(p.ID)
}

// SetStatusToProvisionedHostID sets the "StatusToProvisionedHost" edge to the ProvisionedHost entity by ID.
func (suo *StatusUpdateOne) SetStatusToProvisionedHostID(id uuid.UUID) *StatusUpdateOne {
	suo.mutation.SetStatusToProvisionedHostID(id)
	return suo
}

// SetNillableStatusToProvisionedHostID sets the "StatusToProvisionedHost" edge to the ProvisionedHost entity by ID if the given value is not nil.
func (suo *StatusUpdateOne) SetNillableStatusToProvisionedHostID(id *uuid.UUID) *StatusUpdateOne {
	if id != nil {
		suo = suo.SetStatusToProvisionedHostID(*id)
	}
	return suo
}

// SetStatusToProvisionedHost sets the "StatusToProvisionedHost" edge to the ProvisionedHost entity.
func (suo *StatusUpdateOne) SetStatusToProvisionedHost(p *ProvisionedHost) *StatusUpdateOne {
	return suo.SetStatusToProvisionedHostID(p.ID)
}

// SetStatusToProvisioningStepID sets the "StatusToProvisioningStep" edge to the ProvisioningStep entity by ID.
func (suo *StatusUpdateOne) SetStatusToProvisioningStepID(id uuid.UUID) *StatusUpdateOne {
	suo.mutation.SetStatusToProvisioningStepID(id)
	return suo
}

// SetNillableStatusToProvisioningStepID sets the "StatusToProvisioningStep" edge to the ProvisioningStep entity by ID if the given value is not nil.
func (suo *StatusUpdateOne) SetNillableStatusToProvisioningStepID(id *uuid.UUID) *StatusUpdateOne {
	if id != nil {
		suo = suo.SetStatusToProvisioningStepID(*id)
	}
	return suo
}

// SetStatusToProvisioningStep sets the "StatusToProvisioningStep" edge to the ProvisioningStep entity.
func (suo *StatusUpdateOne) SetStatusToProvisioningStep(p *ProvisioningStep) *StatusUpdateOne {
	return suo.SetStatusToProvisioningStepID(p.ID)
}

// SetStatusToTeamID sets the "StatusToTeam" edge to the Team entity by ID.
func (suo *StatusUpdateOne) SetStatusToTeamID(id uuid.UUID) *StatusUpdateOne {
	suo.mutation.SetStatusToTeamID(id)
	return suo
}

// SetNillableStatusToTeamID sets the "StatusToTeam" edge to the Team entity by ID if the given value is not nil.
func (suo *StatusUpdateOne) SetNillableStatusToTeamID(id *uuid.UUID) *StatusUpdateOne {
	if id != nil {
		suo = suo.SetStatusToTeamID(*id)
	}
	return suo
}

// SetStatusToTeam sets the "StatusToTeam" edge to the Team entity.
func (suo *StatusUpdateOne) SetStatusToTeam(t *Team) *StatusUpdateOne {
	return suo.SetStatusToTeamID(t.ID)
}

// Mutation returns the StatusMutation object of the builder.
func (suo *StatusUpdateOne) Mutation() *StatusMutation {
	return suo.mutation
}

// ClearStatusToBuild clears the "StatusToBuild" edge to the Build entity.
func (suo *StatusUpdateOne) ClearStatusToBuild() *StatusUpdateOne {
	suo.mutation.ClearStatusToBuild()
	return suo
}

// ClearStatusToProvisionedNetwork clears the "StatusToProvisionedNetwork" edge to the ProvisionedNetwork entity.
func (suo *StatusUpdateOne) ClearStatusToProvisionedNetwork() *StatusUpdateOne {
	suo.mutation.ClearStatusToProvisionedNetwork()
	return suo
}

// ClearStatusToProvisionedHost clears the "StatusToProvisionedHost" edge to the ProvisionedHost entity.
func (suo *StatusUpdateOne) ClearStatusToProvisionedHost() *StatusUpdateOne {
	suo.mutation.ClearStatusToProvisionedHost()
	return suo
}

// ClearStatusToProvisioningStep clears the "StatusToProvisioningStep" edge to the ProvisioningStep entity.
func (suo *StatusUpdateOne) ClearStatusToProvisioningStep() *StatusUpdateOne {
	suo.mutation.ClearStatusToProvisioningStep()
	return suo
}

// ClearStatusToTeam clears the "StatusToTeam" edge to the Team entity.
func (suo *StatusUpdateOne) ClearStatusToTeam() *StatusUpdateOne {
	suo.mutation.ClearStatusToTeam()
	return suo
}

// Select allows selecting one or more fields (columns) of the returned entity.
// The default is selecting all fields defined in the entity schema.
func (suo *StatusUpdateOne) Select(field string, fields ...string) *StatusUpdateOne {
	suo.fields = append([]string{field}, fields...)
	return suo
}

// Save executes the query and returns the updated Status entity.
func (suo *StatusUpdateOne) Save(ctx context.Context) (*Status, error) {
	var (
		err  error
		node *Status
	)
	if len(suo.hooks) == 0 {
		if err = suo.check(); err != nil {
			return nil, err
		}
		node, err = suo.sqlSave(ctx)
	} else {
		var mut Mutator = MutateFunc(func(ctx context.Context, m Mutation) (Value, error) {
			mutation, ok := m.(*StatusMutation)
			if !ok {
				return nil, fmt.Errorf("unexpected mutation type %T", m)
			}
			if err = suo.check(); err != nil {
				return nil, err
			}
			suo.mutation = mutation
			node, err = suo.sqlSave(ctx)
			mutation.done = true
			return node, err
		})
		for i := len(suo.hooks) - 1; i >= 0; i-- {
			mut = suo.hooks[i](mut)
		}
		if _, err := mut.Mutate(ctx, suo.mutation); err != nil {
			return nil, err
		}
	}
	return node, err
}

// SaveX is like Save, but panics if an error occurs.
func (suo *StatusUpdateOne) SaveX(ctx context.Context) *Status {
	node, err := suo.Save(ctx)
	if err != nil {
		panic(err)
	}
	return node
}

// Exec executes the query on the entity.
func (suo *StatusUpdateOne) Exec(ctx context.Context) error {
	_, err := suo.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (suo *StatusUpdateOne) ExecX(ctx context.Context) {
	if err := suo.Exec(ctx); err != nil {
		panic(err)
	}
}

// check runs all checks and user-defined validators on the builder.
func (suo *StatusUpdateOne) check() error {
	if v, ok := suo.mutation.State(); ok {
		if err := status.StateValidator(v); err != nil {
			return &ValidationError{Name: "state", err: fmt.Errorf("ent: validator failed for field \"state\": %w", err)}
		}
	}
	if v, ok := suo.mutation.StatusFor(); ok {
		if err := status.StatusForValidator(v); err != nil {
			return &ValidationError{Name: "status_for", err: fmt.Errorf("ent: validator failed for field \"status_for\": %w", err)}
		}
	}
	return nil
}

func (suo *StatusUpdateOne) sqlSave(ctx context.Context) (_node *Status, err error) {
	_spec := &sqlgraph.UpdateSpec{
		Node: &sqlgraph.NodeSpec{
			Table:   status.Table,
			Columns: status.Columns,
			ID: &sqlgraph.FieldSpec{
				Type:   field.TypeUUID,
				Column: status.FieldID,
			},
		},
	}
	id, ok := suo.mutation.ID()
	if !ok {
		return nil, &ValidationError{Name: "ID", err: fmt.Errorf("missing Status.ID for update")}
	}
	_spec.Node.ID.Value = id
	if fields := suo.fields; len(fields) > 0 {
		_spec.Node.Columns = make([]string, 0, len(fields))
		_spec.Node.Columns = append(_spec.Node.Columns, status.FieldID)
		for _, f := range fields {
			if !status.ValidColumn(f) {
				return nil, &ValidationError{Name: f, err: fmt.Errorf("ent: invalid field %q for query", f)}
			}
			if f != status.FieldID {
				_spec.Node.Columns = append(_spec.Node.Columns, f)
			}
		}
	}
	if ps := suo.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if value, ok := suo.mutation.State(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeEnum,
			Value:  value,
			Column: status.FieldState,
		})
	}
	if value, ok := suo.mutation.StatusFor(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeEnum,
			Value:  value,
			Column: status.FieldStatusFor,
		})
	}
	if value, ok := suo.mutation.StartedAt(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeTime,
			Value:  value,
			Column: status.FieldStartedAt,
		})
	}
	if suo.mutation.StartedAtCleared() {
		_spec.Fields.Clear = append(_spec.Fields.Clear, &sqlgraph.FieldSpec{
			Type:   field.TypeTime,
			Column: status.FieldStartedAt,
		})
	}
	if value, ok := suo.mutation.EndedAt(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeTime,
			Value:  value,
			Column: status.FieldEndedAt,
		})
	}
	if suo.mutation.EndedAtCleared() {
		_spec.Fields.Clear = append(_spec.Fields.Clear, &sqlgraph.FieldSpec{
			Type:   field.TypeTime,
			Column: status.FieldEndedAt,
		})
	}
	if value, ok := suo.mutation.Failed(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeBool,
			Value:  value,
			Column: status.FieldFailed,
		})
	}
	if value, ok := suo.mutation.Completed(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeBool,
			Value:  value,
			Column: status.FieldCompleted,
		})
	}
	if value, ok := suo.mutation.Error(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Value:  value,
			Column: status.FieldError,
		})
	}
	if suo.mutation.ErrorCleared() {
		_spec.Fields.Clear = append(_spec.Fields.Clear, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Column: status.FieldError,
		})
	}
	if suo.mutation.StatusToBuildCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2O,
			Inverse: true,
			Table:   status.StatusToBuildTable,
			Columns: []string{status.StatusToBuildColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeUUID,
					Column: build.FieldID,
				},
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := suo.mutation.StatusToBuildIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2O,
			Inverse: true,
			Table:   status.StatusToBuildTable,
			Columns: []string{status.StatusToBuildColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeUUID,
					Column: build.FieldID,
				},
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	if suo.mutation.StatusToProvisionedNetworkCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2O,
			Inverse: true,
			Table:   status.StatusToProvisionedNetworkTable,
			Columns: []string{status.StatusToProvisionedNetworkColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeUUID,
					Column: provisionednetwork.FieldID,
				},
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := suo.mutation.StatusToProvisionedNetworkIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2O,
			Inverse: true,
			Table:   status.StatusToProvisionedNetworkTable,
			Columns: []string{status.StatusToProvisionedNetworkColumn},
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
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	if suo.mutation.StatusToProvisionedHostCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2O,
			Inverse: true,
			Table:   status.StatusToProvisionedHostTable,
			Columns: []string{status.StatusToProvisionedHostColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeUUID,
					Column: provisionedhost.FieldID,
				},
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := suo.mutation.StatusToProvisionedHostIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2O,
			Inverse: true,
			Table:   status.StatusToProvisionedHostTable,
			Columns: []string{status.StatusToProvisionedHostColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeUUID,
					Column: provisionedhost.FieldID,
				},
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	if suo.mutation.StatusToProvisioningStepCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2O,
			Inverse: true,
			Table:   status.StatusToProvisioningStepTable,
			Columns: []string{status.StatusToProvisioningStepColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeUUID,
					Column: provisioningstep.FieldID,
				},
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := suo.mutation.StatusToProvisioningStepIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2O,
			Inverse: true,
			Table:   status.StatusToProvisioningStepTable,
			Columns: []string{status.StatusToProvisioningStepColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeUUID,
					Column: provisioningstep.FieldID,
				},
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	if suo.mutation.StatusToTeamCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2O,
			Inverse: true,
			Table:   status.StatusToTeamTable,
			Columns: []string{status.StatusToTeamColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeUUID,
					Column: team.FieldID,
				},
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := suo.mutation.StatusToTeamIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2O,
			Inverse: true,
			Table:   status.StatusToTeamTable,
			Columns: []string{status.StatusToTeamColumn},
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
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	_node = &Status{config: suo.config}
	_spec.Assign = _node.assignValues
	_spec.ScanValues = _node.scanValues
	if err = sqlgraph.UpdateNode(ctx, suo.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{status.Label}
		} else if cerr, ok := isSQLConstraintError(err); ok {
			err = cerr
		}
		return nil, err
	}
	return _node, nil
}
