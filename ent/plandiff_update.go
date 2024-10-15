// Code generated by ent, DO NOT EDIT.

package ent

import (
	"context"
	"errors"
	"fmt"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/schema/field"
	"github.com/gen0cide/laforge/ent/buildcommit"
	"github.com/gen0cide/laforge/ent/plan"
	"github.com/gen0cide/laforge/ent/plandiff"
	"github.com/gen0cide/laforge/ent/predicate"
	"github.com/google/uuid"
)

// PlanDiffUpdate is the builder for updating PlanDiff entities.
type PlanDiffUpdate struct {
	config
	hooks    []Hook
	mutation *PlanDiffMutation
}

// Where appends a list predicates to the PlanDiffUpdate builder.
func (pdu *PlanDiffUpdate) Where(ps ...predicate.PlanDiff) *PlanDiffUpdate {
	pdu.mutation.Where(ps...)
	return pdu
}

// SetRevision sets the "revision" field.
func (pdu *PlanDiffUpdate) SetRevision(i int) *PlanDiffUpdate {
	pdu.mutation.ResetRevision()
	pdu.mutation.SetRevision(i)
	return pdu
}

// SetNillableRevision sets the "revision" field if the given value is not nil.
func (pdu *PlanDiffUpdate) SetNillableRevision(i *int) *PlanDiffUpdate {
	if i != nil {
		pdu.SetRevision(*i)
	}
	return pdu
}

// AddRevision adds i to the "revision" field.
func (pdu *PlanDiffUpdate) AddRevision(i int) *PlanDiffUpdate {
	pdu.mutation.AddRevision(i)
	return pdu
}

// SetNewState sets the "new_state" field.
func (pdu *PlanDiffUpdate) SetNewState(ps plandiff.NewState) *PlanDiffUpdate {
	pdu.mutation.SetNewState(ps)
	return pdu
}

// SetNillableNewState sets the "new_state" field if the given value is not nil.
func (pdu *PlanDiffUpdate) SetNillableNewState(ps *plandiff.NewState) *PlanDiffUpdate {
	if ps != nil {
		pdu.SetNewState(*ps)
	}
	return pdu
}

// SetBuildCommitID sets the "BuildCommit" edge to the BuildCommit entity by ID.
func (pdu *PlanDiffUpdate) SetBuildCommitID(id uuid.UUID) *PlanDiffUpdate {
	pdu.mutation.SetBuildCommitID(id)
	return pdu
}

// SetBuildCommit sets the "BuildCommit" edge to the BuildCommit entity.
func (pdu *PlanDiffUpdate) SetBuildCommit(b *BuildCommit) *PlanDiffUpdate {
	return pdu.SetBuildCommitID(b.ID)
}

// SetPlanID sets the "Plan" edge to the Plan entity by ID.
func (pdu *PlanDiffUpdate) SetPlanID(id uuid.UUID) *PlanDiffUpdate {
	pdu.mutation.SetPlanID(id)
	return pdu
}

// SetPlan sets the "Plan" edge to the Plan entity.
func (pdu *PlanDiffUpdate) SetPlan(p *Plan) *PlanDiffUpdate {
	return pdu.SetPlanID(p.ID)
}

// Mutation returns the PlanDiffMutation object of the builder.
func (pdu *PlanDiffUpdate) Mutation() *PlanDiffMutation {
	return pdu.mutation
}

// ClearBuildCommit clears the "BuildCommit" edge to the BuildCommit entity.
func (pdu *PlanDiffUpdate) ClearBuildCommit() *PlanDiffUpdate {
	pdu.mutation.ClearBuildCommit()
	return pdu
}

// ClearPlan clears the "Plan" edge to the Plan entity.
func (pdu *PlanDiffUpdate) ClearPlan() *PlanDiffUpdate {
	pdu.mutation.ClearPlan()
	return pdu
}

// Save executes the query and returns the number of nodes affected by the update operation.
func (pdu *PlanDiffUpdate) Save(ctx context.Context) (int, error) {
	return withHooks(ctx, pdu.sqlSave, pdu.mutation, pdu.hooks)
}

// SaveX is like Save, but panics if an error occurs.
func (pdu *PlanDiffUpdate) SaveX(ctx context.Context) int {
	affected, err := pdu.Save(ctx)
	if err != nil {
		panic(err)
	}
	return affected
}

// Exec executes the query.
func (pdu *PlanDiffUpdate) Exec(ctx context.Context) error {
	_, err := pdu.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (pdu *PlanDiffUpdate) ExecX(ctx context.Context) {
	if err := pdu.Exec(ctx); err != nil {
		panic(err)
	}
}

// check runs all checks and user-defined validators on the builder.
func (pdu *PlanDiffUpdate) check() error {
	if v, ok := pdu.mutation.NewState(); ok {
		if err := plandiff.NewStateValidator(v); err != nil {
			return &ValidationError{Name: "new_state", err: fmt.Errorf(`ent: validator failed for field "PlanDiff.new_state": %w`, err)}
		}
	}
	if _, ok := pdu.mutation.BuildCommitID(); pdu.mutation.BuildCommitCleared() && !ok {
		return errors.New(`ent: clearing a required unique edge "PlanDiff.BuildCommit"`)
	}
	if _, ok := pdu.mutation.PlanID(); pdu.mutation.PlanCleared() && !ok {
		return errors.New(`ent: clearing a required unique edge "PlanDiff.Plan"`)
	}
	return nil
}

func (pdu *PlanDiffUpdate) sqlSave(ctx context.Context) (n int, err error) {
	if err := pdu.check(); err != nil {
		return n, err
	}
	_spec := sqlgraph.NewUpdateSpec(plandiff.Table, plandiff.Columns, sqlgraph.NewFieldSpec(plandiff.FieldID, field.TypeUUID))
	if ps := pdu.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if value, ok := pdu.mutation.Revision(); ok {
		_spec.SetField(plandiff.FieldRevision, field.TypeInt, value)
	}
	if value, ok := pdu.mutation.AddedRevision(); ok {
		_spec.AddField(plandiff.FieldRevision, field.TypeInt, value)
	}
	if value, ok := pdu.mutation.NewState(); ok {
		_spec.SetField(plandiff.FieldNewState, field.TypeEnum, value)
	}
	if pdu.mutation.BuildCommitCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: false,
			Table:   plandiff.BuildCommitTable,
			Columns: []string{plandiff.BuildCommitColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(buildcommit.FieldID, field.TypeUUID),
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := pdu.mutation.BuildCommitIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: false,
			Table:   plandiff.BuildCommitTable,
			Columns: []string{plandiff.BuildCommitColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(buildcommit.FieldID, field.TypeUUID),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	if pdu.mutation.PlanCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: false,
			Table:   plandiff.PlanTable,
			Columns: []string{plandiff.PlanColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(plan.FieldID, field.TypeUUID),
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := pdu.mutation.PlanIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: false,
			Table:   plandiff.PlanTable,
			Columns: []string{plandiff.PlanColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(plan.FieldID, field.TypeUUID),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	if n, err = sqlgraph.UpdateNodes(ctx, pdu.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{plandiff.Label}
		} else if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return 0, err
	}
	pdu.mutation.done = true
	return n, nil
}

// PlanDiffUpdateOne is the builder for updating a single PlanDiff entity.
type PlanDiffUpdateOne struct {
	config
	fields   []string
	hooks    []Hook
	mutation *PlanDiffMutation
}

// SetRevision sets the "revision" field.
func (pduo *PlanDiffUpdateOne) SetRevision(i int) *PlanDiffUpdateOne {
	pduo.mutation.ResetRevision()
	pduo.mutation.SetRevision(i)
	return pduo
}

// SetNillableRevision sets the "revision" field if the given value is not nil.
func (pduo *PlanDiffUpdateOne) SetNillableRevision(i *int) *PlanDiffUpdateOne {
	if i != nil {
		pduo.SetRevision(*i)
	}
	return pduo
}

// AddRevision adds i to the "revision" field.
func (pduo *PlanDiffUpdateOne) AddRevision(i int) *PlanDiffUpdateOne {
	pduo.mutation.AddRevision(i)
	return pduo
}

// SetNewState sets the "new_state" field.
func (pduo *PlanDiffUpdateOne) SetNewState(ps plandiff.NewState) *PlanDiffUpdateOne {
	pduo.mutation.SetNewState(ps)
	return pduo
}

// SetNillableNewState sets the "new_state" field if the given value is not nil.
func (pduo *PlanDiffUpdateOne) SetNillableNewState(ps *plandiff.NewState) *PlanDiffUpdateOne {
	if ps != nil {
		pduo.SetNewState(*ps)
	}
	return pduo
}

// SetBuildCommitID sets the "BuildCommit" edge to the BuildCommit entity by ID.
func (pduo *PlanDiffUpdateOne) SetBuildCommitID(id uuid.UUID) *PlanDiffUpdateOne {
	pduo.mutation.SetBuildCommitID(id)
	return pduo
}

// SetBuildCommit sets the "BuildCommit" edge to the BuildCommit entity.
func (pduo *PlanDiffUpdateOne) SetBuildCommit(b *BuildCommit) *PlanDiffUpdateOne {
	return pduo.SetBuildCommitID(b.ID)
}

// SetPlanID sets the "Plan" edge to the Plan entity by ID.
func (pduo *PlanDiffUpdateOne) SetPlanID(id uuid.UUID) *PlanDiffUpdateOne {
	pduo.mutation.SetPlanID(id)
	return pduo
}

// SetPlan sets the "Plan" edge to the Plan entity.
func (pduo *PlanDiffUpdateOne) SetPlan(p *Plan) *PlanDiffUpdateOne {
	return pduo.SetPlanID(p.ID)
}

// Mutation returns the PlanDiffMutation object of the builder.
func (pduo *PlanDiffUpdateOne) Mutation() *PlanDiffMutation {
	return pduo.mutation
}

// ClearBuildCommit clears the "BuildCommit" edge to the BuildCommit entity.
func (pduo *PlanDiffUpdateOne) ClearBuildCommit() *PlanDiffUpdateOne {
	pduo.mutation.ClearBuildCommit()
	return pduo
}

// ClearPlan clears the "Plan" edge to the Plan entity.
func (pduo *PlanDiffUpdateOne) ClearPlan() *PlanDiffUpdateOne {
	pduo.mutation.ClearPlan()
	return pduo
}

// Where appends a list predicates to the PlanDiffUpdate builder.
func (pduo *PlanDiffUpdateOne) Where(ps ...predicate.PlanDiff) *PlanDiffUpdateOne {
	pduo.mutation.Where(ps...)
	return pduo
}

// Select allows selecting one or more fields (columns) of the returned entity.
// The default is selecting all fields defined in the entity schema.
func (pduo *PlanDiffUpdateOne) Select(field string, fields ...string) *PlanDiffUpdateOne {
	pduo.fields = append([]string{field}, fields...)
	return pduo
}

// Save executes the query and returns the updated PlanDiff entity.
func (pduo *PlanDiffUpdateOne) Save(ctx context.Context) (*PlanDiff, error) {
	return withHooks(ctx, pduo.sqlSave, pduo.mutation, pduo.hooks)
}

// SaveX is like Save, but panics if an error occurs.
func (pduo *PlanDiffUpdateOne) SaveX(ctx context.Context) *PlanDiff {
	node, err := pduo.Save(ctx)
	if err != nil {
		panic(err)
	}
	return node
}

// Exec executes the query on the entity.
func (pduo *PlanDiffUpdateOne) Exec(ctx context.Context) error {
	_, err := pduo.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (pduo *PlanDiffUpdateOne) ExecX(ctx context.Context) {
	if err := pduo.Exec(ctx); err != nil {
		panic(err)
	}
}

// check runs all checks and user-defined validators on the builder.
func (pduo *PlanDiffUpdateOne) check() error {
	if v, ok := pduo.mutation.NewState(); ok {
		if err := plandiff.NewStateValidator(v); err != nil {
			return &ValidationError{Name: "new_state", err: fmt.Errorf(`ent: validator failed for field "PlanDiff.new_state": %w`, err)}
		}
	}
	if _, ok := pduo.mutation.BuildCommitID(); pduo.mutation.BuildCommitCleared() && !ok {
		return errors.New(`ent: clearing a required unique edge "PlanDiff.BuildCommit"`)
	}
	if _, ok := pduo.mutation.PlanID(); pduo.mutation.PlanCleared() && !ok {
		return errors.New(`ent: clearing a required unique edge "PlanDiff.Plan"`)
	}
	return nil
}

func (pduo *PlanDiffUpdateOne) sqlSave(ctx context.Context) (_node *PlanDiff, err error) {
	if err := pduo.check(); err != nil {
		return _node, err
	}
	_spec := sqlgraph.NewUpdateSpec(plandiff.Table, plandiff.Columns, sqlgraph.NewFieldSpec(plandiff.FieldID, field.TypeUUID))
	id, ok := pduo.mutation.ID()
	if !ok {
		return nil, &ValidationError{Name: "id", err: errors.New(`ent: missing "PlanDiff.id" for update`)}
	}
	_spec.Node.ID.Value = id
	if fields := pduo.fields; len(fields) > 0 {
		_spec.Node.Columns = make([]string, 0, len(fields))
		_spec.Node.Columns = append(_spec.Node.Columns, plandiff.FieldID)
		for _, f := range fields {
			if !plandiff.ValidColumn(f) {
				return nil, &ValidationError{Name: f, err: fmt.Errorf("ent: invalid field %q for query", f)}
			}
			if f != plandiff.FieldID {
				_spec.Node.Columns = append(_spec.Node.Columns, f)
			}
		}
	}
	if ps := pduo.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if value, ok := pduo.mutation.Revision(); ok {
		_spec.SetField(plandiff.FieldRevision, field.TypeInt, value)
	}
	if value, ok := pduo.mutation.AddedRevision(); ok {
		_spec.AddField(plandiff.FieldRevision, field.TypeInt, value)
	}
	if value, ok := pduo.mutation.NewState(); ok {
		_spec.SetField(plandiff.FieldNewState, field.TypeEnum, value)
	}
	if pduo.mutation.BuildCommitCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: false,
			Table:   plandiff.BuildCommitTable,
			Columns: []string{plandiff.BuildCommitColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(buildcommit.FieldID, field.TypeUUID),
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := pduo.mutation.BuildCommitIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: false,
			Table:   plandiff.BuildCommitTable,
			Columns: []string{plandiff.BuildCommitColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(buildcommit.FieldID, field.TypeUUID),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	if pduo.mutation.PlanCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: false,
			Table:   plandiff.PlanTable,
			Columns: []string{plandiff.PlanColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(plan.FieldID, field.TypeUUID),
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := pduo.mutation.PlanIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: false,
			Table:   plandiff.PlanTable,
			Columns: []string{plandiff.PlanColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(plan.FieldID, field.TypeUUID),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	_node = &PlanDiff{config: pduo.config}
	_spec.Assign = _node.assignValues
	_spec.ScanValues = _node.scanValues
	if err = sqlgraph.UpdateNode(ctx, pduo.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{plandiff.Label}
		} else if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return nil, err
	}
	pduo.mutation.done = true
	return _node, nil
}
