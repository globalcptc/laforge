// Code generated by entc, DO NOT EDIT.

package ent

import (
	"context"
	"fmt"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/schema/field"
	"github.com/gen0cide/laforge/ent/environment"
	"github.com/gen0cide/laforge/ent/hostdependency"
	"github.com/gen0cide/laforge/ent/includednetwork"
	"github.com/gen0cide/laforge/ent/network"
	"github.com/gen0cide/laforge/ent/predicate"
	"github.com/google/uuid"
)

// NetworkUpdate is the builder for updating Network entities.
type NetworkUpdate struct {
	config
	hooks    []Hook
	mutation *NetworkMutation
}

// Where appends a list predicates to the NetworkUpdate builder.
func (nu *NetworkUpdate) Where(ps ...predicate.Network) *NetworkUpdate {
	nu.mutation.Where(ps...)
	return nu
}

// SetHclID sets the "hcl_id" field.
func (nu *NetworkUpdate) SetHclID(s string) *NetworkUpdate {
	nu.mutation.SetHclID(s)
	return nu
}

// SetName sets the "name" field.
func (nu *NetworkUpdate) SetName(s string) *NetworkUpdate {
	nu.mutation.SetName(s)
	return nu
}

// SetCidr sets the "cidr" field.
func (nu *NetworkUpdate) SetCidr(s string) *NetworkUpdate {
	nu.mutation.SetCidr(s)
	return nu
}

// SetVdiVisible sets the "vdi_visible" field.
func (nu *NetworkUpdate) SetVdiVisible(b bool) *NetworkUpdate {
	nu.mutation.SetVdiVisible(b)
	return nu
}

// SetVars sets the "vars" field.
func (nu *NetworkUpdate) SetVars(m map[string]string) *NetworkUpdate {
	nu.mutation.SetVars(m)
	return nu
}

// SetTags sets the "tags" field.
func (nu *NetworkUpdate) SetTags(m map[string]string) *NetworkUpdate {
	nu.mutation.SetTags(m)
	return nu
}

// SetNetworkToEnvironmentID sets the "NetworkToEnvironment" edge to the Environment entity by ID.
func (nu *NetworkUpdate) SetNetworkToEnvironmentID(id uuid.UUID) *NetworkUpdate {
	nu.mutation.SetNetworkToEnvironmentID(id)
	return nu
}

// SetNillableNetworkToEnvironmentID sets the "NetworkToEnvironment" edge to the Environment entity by ID if the given value is not nil.
func (nu *NetworkUpdate) SetNillableNetworkToEnvironmentID(id *uuid.UUID) *NetworkUpdate {
	if id != nil {
		nu = nu.SetNetworkToEnvironmentID(*id)
	}
	return nu
}

// SetNetworkToEnvironment sets the "NetworkToEnvironment" edge to the Environment entity.
func (nu *NetworkUpdate) SetNetworkToEnvironment(e *Environment) *NetworkUpdate {
	return nu.SetNetworkToEnvironmentID(e.ID)
}

// AddNetworkToHostDependencyIDs adds the "NetworkToHostDependency" edge to the HostDependency entity by IDs.
func (nu *NetworkUpdate) AddNetworkToHostDependencyIDs(ids ...uuid.UUID) *NetworkUpdate {
	nu.mutation.AddNetworkToHostDependencyIDs(ids...)
	return nu
}

// AddNetworkToHostDependency adds the "NetworkToHostDependency" edges to the HostDependency entity.
func (nu *NetworkUpdate) AddNetworkToHostDependency(h ...*HostDependency) *NetworkUpdate {
	ids := make([]uuid.UUID, len(h))
	for i := range h {
		ids[i] = h[i].ID
	}
	return nu.AddNetworkToHostDependencyIDs(ids...)
}

// AddNetworkToIncludedNetworkIDs adds the "NetworkToIncludedNetwork" edge to the IncludedNetwork entity by IDs.
func (nu *NetworkUpdate) AddNetworkToIncludedNetworkIDs(ids ...uuid.UUID) *NetworkUpdate {
	nu.mutation.AddNetworkToIncludedNetworkIDs(ids...)
	return nu
}

// AddNetworkToIncludedNetwork adds the "NetworkToIncludedNetwork" edges to the IncludedNetwork entity.
func (nu *NetworkUpdate) AddNetworkToIncludedNetwork(i ...*IncludedNetwork) *NetworkUpdate {
	ids := make([]uuid.UUID, len(i))
	for j := range i {
		ids[j] = i[j].ID
	}
	return nu.AddNetworkToIncludedNetworkIDs(ids...)
}

// Mutation returns the NetworkMutation object of the builder.
func (nu *NetworkUpdate) Mutation() *NetworkMutation {
	return nu.mutation
}

// ClearNetworkToEnvironment clears the "NetworkToEnvironment" edge to the Environment entity.
func (nu *NetworkUpdate) ClearNetworkToEnvironment() *NetworkUpdate {
	nu.mutation.ClearNetworkToEnvironment()
	return nu
}

// ClearNetworkToHostDependency clears all "NetworkToHostDependency" edges to the HostDependency entity.
func (nu *NetworkUpdate) ClearNetworkToHostDependency() *NetworkUpdate {
	nu.mutation.ClearNetworkToHostDependency()
	return nu
}

// RemoveNetworkToHostDependencyIDs removes the "NetworkToHostDependency" edge to HostDependency entities by IDs.
func (nu *NetworkUpdate) RemoveNetworkToHostDependencyIDs(ids ...uuid.UUID) *NetworkUpdate {
	nu.mutation.RemoveNetworkToHostDependencyIDs(ids...)
	return nu
}

// RemoveNetworkToHostDependency removes "NetworkToHostDependency" edges to HostDependency entities.
func (nu *NetworkUpdate) RemoveNetworkToHostDependency(h ...*HostDependency) *NetworkUpdate {
	ids := make([]uuid.UUID, len(h))
	for i := range h {
		ids[i] = h[i].ID
	}
	return nu.RemoveNetworkToHostDependencyIDs(ids...)
}

// ClearNetworkToIncludedNetwork clears all "NetworkToIncludedNetwork" edges to the IncludedNetwork entity.
func (nu *NetworkUpdate) ClearNetworkToIncludedNetwork() *NetworkUpdate {
	nu.mutation.ClearNetworkToIncludedNetwork()
	return nu
}

// RemoveNetworkToIncludedNetworkIDs removes the "NetworkToIncludedNetwork" edge to IncludedNetwork entities by IDs.
func (nu *NetworkUpdate) RemoveNetworkToIncludedNetworkIDs(ids ...uuid.UUID) *NetworkUpdate {
	nu.mutation.RemoveNetworkToIncludedNetworkIDs(ids...)
	return nu
}

// RemoveNetworkToIncludedNetwork removes "NetworkToIncludedNetwork" edges to IncludedNetwork entities.
func (nu *NetworkUpdate) RemoveNetworkToIncludedNetwork(i ...*IncludedNetwork) *NetworkUpdate {
	ids := make([]uuid.UUID, len(i))
	for j := range i {
		ids[j] = i[j].ID
	}
	return nu.RemoveNetworkToIncludedNetworkIDs(ids...)
}

// Save executes the query and returns the number of nodes affected by the update operation.
func (nu *NetworkUpdate) Save(ctx context.Context) (int, error) {
	var (
		err      error
		affected int
	)
	if len(nu.hooks) == 0 {
		affected, err = nu.sqlSave(ctx)
	} else {
		var mut Mutator = MutateFunc(func(ctx context.Context, m Mutation) (Value, error) {
			mutation, ok := m.(*NetworkMutation)
			if !ok {
				return nil, fmt.Errorf("unexpected mutation type %T", m)
			}
			nu.mutation = mutation
			affected, err = nu.sqlSave(ctx)
			mutation.done = true
			return affected, err
		})
		for i := len(nu.hooks) - 1; i >= 0; i-- {
			if nu.hooks[i] == nil {
				return 0, fmt.Errorf("ent: uninitialized hook (forgotten import ent/runtime?)")
			}
			mut = nu.hooks[i](mut)
		}
		if _, err := mut.Mutate(ctx, nu.mutation); err != nil {
			return 0, err
		}
	}
	return affected, err
}

// SaveX is like Save, but panics if an error occurs.
func (nu *NetworkUpdate) SaveX(ctx context.Context) int {
	affected, err := nu.Save(ctx)
	if err != nil {
		panic(err)
	}
	return affected
}

// Exec executes the query.
func (nu *NetworkUpdate) Exec(ctx context.Context) error {
	_, err := nu.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (nu *NetworkUpdate) ExecX(ctx context.Context) {
	if err := nu.Exec(ctx); err != nil {
		panic(err)
	}
}

func (nu *NetworkUpdate) sqlSave(ctx context.Context) (n int, err error) {
	_spec := &sqlgraph.UpdateSpec{
		Node: &sqlgraph.NodeSpec{
			Table:   network.Table,
			Columns: network.Columns,
			ID: &sqlgraph.FieldSpec{
				Type:   field.TypeUUID,
				Column: network.FieldID,
			},
		},
	}
	if ps := nu.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if value, ok := nu.mutation.HclID(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Value:  value,
			Column: network.FieldHclID,
		})
	}
	if value, ok := nu.mutation.Name(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Value:  value,
			Column: network.FieldName,
		})
	}
	if value, ok := nu.mutation.Cidr(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Value:  value,
			Column: network.FieldCidr,
		})
	}
	if value, ok := nu.mutation.VdiVisible(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeBool,
			Value:  value,
			Column: network.FieldVdiVisible,
		})
	}
	if value, ok := nu.mutation.Vars(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeJSON,
			Value:  value,
			Column: network.FieldVars,
		})
	}
	if value, ok := nu.mutation.Tags(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeJSON,
			Value:  value,
			Column: network.FieldTags,
		})
	}
	if nu.mutation.NetworkToEnvironmentCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: true,
			Table:   network.NetworkToEnvironmentTable,
			Columns: []string{network.NetworkToEnvironmentColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeUUID,
					Column: environment.FieldID,
				},
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := nu.mutation.NetworkToEnvironmentIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: true,
			Table:   network.NetworkToEnvironmentTable,
			Columns: []string{network.NetworkToEnvironmentColumn},
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
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	if nu.mutation.NetworkToHostDependencyCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: true,
			Table:   network.NetworkToHostDependencyTable,
			Columns: []string{network.NetworkToHostDependencyColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeUUID,
					Column: hostdependency.FieldID,
				},
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := nu.mutation.RemovedNetworkToHostDependencyIDs(); len(nodes) > 0 && !nu.mutation.NetworkToHostDependencyCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: true,
			Table:   network.NetworkToHostDependencyTable,
			Columns: []string{network.NetworkToHostDependencyColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeUUID,
					Column: hostdependency.FieldID,
				},
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := nu.mutation.NetworkToHostDependencyIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: true,
			Table:   network.NetworkToHostDependencyTable,
			Columns: []string{network.NetworkToHostDependencyColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeUUID,
					Column: hostdependency.FieldID,
				},
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	if nu.mutation.NetworkToIncludedNetworkCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: true,
			Table:   network.NetworkToIncludedNetworkTable,
			Columns: []string{network.NetworkToIncludedNetworkColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeUUID,
					Column: includednetwork.FieldID,
				},
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := nu.mutation.RemovedNetworkToIncludedNetworkIDs(); len(nodes) > 0 && !nu.mutation.NetworkToIncludedNetworkCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: true,
			Table:   network.NetworkToIncludedNetworkTable,
			Columns: []string{network.NetworkToIncludedNetworkColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeUUID,
					Column: includednetwork.FieldID,
				},
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := nu.mutation.NetworkToIncludedNetworkIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: true,
			Table:   network.NetworkToIncludedNetworkTable,
			Columns: []string{network.NetworkToIncludedNetworkColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeUUID,
					Column: includednetwork.FieldID,
				},
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	if n, err = sqlgraph.UpdateNodes(ctx, nu.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{network.Label}
		} else if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{err.Error(), err}
		}
		return 0, err
	}
	return n, nil
}

// NetworkUpdateOne is the builder for updating a single Network entity.
type NetworkUpdateOne struct {
	config
	fields   []string
	hooks    []Hook
	mutation *NetworkMutation
}

// SetHclID sets the "hcl_id" field.
func (nuo *NetworkUpdateOne) SetHclID(s string) *NetworkUpdateOne {
	nuo.mutation.SetHclID(s)
	return nuo
}

// SetName sets the "name" field.
func (nuo *NetworkUpdateOne) SetName(s string) *NetworkUpdateOne {
	nuo.mutation.SetName(s)
	return nuo
}

// SetCidr sets the "cidr" field.
func (nuo *NetworkUpdateOne) SetCidr(s string) *NetworkUpdateOne {
	nuo.mutation.SetCidr(s)
	return nuo
}

// SetVdiVisible sets the "vdi_visible" field.
func (nuo *NetworkUpdateOne) SetVdiVisible(b bool) *NetworkUpdateOne {
	nuo.mutation.SetVdiVisible(b)
	return nuo
}

// SetVars sets the "vars" field.
func (nuo *NetworkUpdateOne) SetVars(m map[string]string) *NetworkUpdateOne {
	nuo.mutation.SetVars(m)
	return nuo
}

// SetTags sets the "tags" field.
func (nuo *NetworkUpdateOne) SetTags(m map[string]string) *NetworkUpdateOne {
	nuo.mutation.SetTags(m)
	return nuo
}

// SetNetworkToEnvironmentID sets the "NetworkToEnvironment" edge to the Environment entity by ID.
func (nuo *NetworkUpdateOne) SetNetworkToEnvironmentID(id uuid.UUID) *NetworkUpdateOne {
	nuo.mutation.SetNetworkToEnvironmentID(id)
	return nuo
}

// SetNillableNetworkToEnvironmentID sets the "NetworkToEnvironment" edge to the Environment entity by ID if the given value is not nil.
func (nuo *NetworkUpdateOne) SetNillableNetworkToEnvironmentID(id *uuid.UUID) *NetworkUpdateOne {
	if id != nil {
		nuo = nuo.SetNetworkToEnvironmentID(*id)
	}
	return nuo
}

// SetNetworkToEnvironment sets the "NetworkToEnvironment" edge to the Environment entity.
func (nuo *NetworkUpdateOne) SetNetworkToEnvironment(e *Environment) *NetworkUpdateOne {
	return nuo.SetNetworkToEnvironmentID(e.ID)
}

// AddNetworkToHostDependencyIDs adds the "NetworkToHostDependency" edge to the HostDependency entity by IDs.
func (nuo *NetworkUpdateOne) AddNetworkToHostDependencyIDs(ids ...uuid.UUID) *NetworkUpdateOne {
	nuo.mutation.AddNetworkToHostDependencyIDs(ids...)
	return nuo
}

// AddNetworkToHostDependency adds the "NetworkToHostDependency" edges to the HostDependency entity.
func (nuo *NetworkUpdateOne) AddNetworkToHostDependency(h ...*HostDependency) *NetworkUpdateOne {
	ids := make([]uuid.UUID, len(h))
	for i := range h {
		ids[i] = h[i].ID
	}
	return nuo.AddNetworkToHostDependencyIDs(ids...)
}

// AddNetworkToIncludedNetworkIDs adds the "NetworkToIncludedNetwork" edge to the IncludedNetwork entity by IDs.
func (nuo *NetworkUpdateOne) AddNetworkToIncludedNetworkIDs(ids ...uuid.UUID) *NetworkUpdateOne {
	nuo.mutation.AddNetworkToIncludedNetworkIDs(ids...)
	return nuo
}

// AddNetworkToIncludedNetwork adds the "NetworkToIncludedNetwork" edges to the IncludedNetwork entity.
func (nuo *NetworkUpdateOne) AddNetworkToIncludedNetwork(i ...*IncludedNetwork) *NetworkUpdateOne {
	ids := make([]uuid.UUID, len(i))
	for j := range i {
		ids[j] = i[j].ID
	}
	return nuo.AddNetworkToIncludedNetworkIDs(ids...)
}

// Mutation returns the NetworkMutation object of the builder.
func (nuo *NetworkUpdateOne) Mutation() *NetworkMutation {
	return nuo.mutation
}

// ClearNetworkToEnvironment clears the "NetworkToEnvironment" edge to the Environment entity.
func (nuo *NetworkUpdateOne) ClearNetworkToEnvironment() *NetworkUpdateOne {
	nuo.mutation.ClearNetworkToEnvironment()
	return nuo
}

// ClearNetworkToHostDependency clears all "NetworkToHostDependency" edges to the HostDependency entity.
func (nuo *NetworkUpdateOne) ClearNetworkToHostDependency() *NetworkUpdateOne {
	nuo.mutation.ClearNetworkToHostDependency()
	return nuo
}

// RemoveNetworkToHostDependencyIDs removes the "NetworkToHostDependency" edge to HostDependency entities by IDs.
func (nuo *NetworkUpdateOne) RemoveNetworkToHostDependencyIDs(ids ...uuid.UUID) *NetworkUpdateOne {
	nuo.mutation.RemoveNetworkToHostDependencyIDs(ids...)
	return nuo
}

// RemoveNetworkToHostDependency removes "NetworkToHostDependency" edges to HostDependency entities.
func (nuo *NetworkUpdateOne) RemoveNetworkToHostDependency(h ...*HostDependency) *NetworkUpdateOne {
	ids := make([]uuid.UUID, len(h))
	for i := range h {
		ids[i] = h[i].ID
	}
	return nuo.RemoveNetworkToHostDependencyIDs(ids...)
}

// ClearNetworkToIncludedNetwork clears all "NetworkToIncludedNetwork" edges to the IncludedNetwork entity.
func (nuo *NetworkUpdateOne) ClearNetworkToIncludedNetwork() *NetworkUpdateOne {
	nuo.mutation.ClearNetworkToIncludedNetwork()
	return nuo
}

// RemoveNetworkToIncludedNetworkIDs removes the "NetworkToIncludedNetwork" edge to IncludedNetwork entities by IDs.
func (nuo *NetworkUpdateOne) RemoveNetworkToIncludedNetworkIDs(ids ...uuid.UUID) *NetworkUpdateOne {
	nuo.mutation.RemoveNetworkToIncludedNetworkIDs(ids...)
	return nuo
}

// RemoveNetworkToIncludedNetwork removes "NetworkToIncludedNetwork" edges to IncludedNetwork entities.
func (nuo *NetworkUpdateOne) RemoveNetworkToIncludedNetwork(i ...*IncludedNetwork) *NetworkUpdateOne {
	ids := make([]uuid.UUID, len(i))
	for j := range i {
		ids[j] = i[j].ID
	}
	return nuo.RemoveNetworkToIncludedNetworkIDs(ids...)
}

// Select allows selecting one or more fields (columns) of the returned entity.
// The default is selecting all fields defined in the entity schema.
func (nuo *NetworkUpdateOne) Select(field string, fields ...string) *NetworkUpdateOne {
	nuo.fields = append([]string{field}, fields...)
	return nuo
}

// Save executes the query and returns the updated Network entity.
func (nuo *NetworkUpdateOne) Save(ctx context.Context) (*Network, error) {
	var (
		err  error
		node *Network
	)
	if len(nuo.hooks) == 0 {
		node, err = nuo.sqlSave(ctx)
	} else {
		var mut Mutator = MutateFunc(func(ctx context.Context, m Mutation) (Value, error) {
			mutation, ok := m.(*NetworkMutation)
			if !ok {
				return nil, fmt.Errorf("unexpected mutation type %T", m)
			}
			nuo.mutation = mutation
			node, err = nuo.sqlSave(ctx)
			mutation.done = true
			return node, err
		})
		for i := len(nuo.hooks) - 1; i >= 0; i-- {
			if nuo.hooks[i] == nil {
				return nil, fmt.Errorf("ent: uninitialized hook (forgotten import ent/runtime?)")
			}
			mut = nuo.hooks[i](mut)
		}
		if _, err := mut.Mutate(ctx, nuo.mutation); err != nil {
			return nil, err
		}
	}
	return node, err
}

// SaveX is like Save, but panics if an error occurs.
func (nuo *NetworkUpdateOne) SaveX(ctx context.Context) *Network {
	node, err := nuo.Save(ctx)
	if err != nil {
		panic(err)
	}
	return node
}

// Exec executes the query on the entity.
func (nuo *NetworkUpdateOne) Exec(ctx context.Context) error {
	_, err := nuo.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (nuo *NetworkUpdateOne) ExecX(ctx context.Context) {
	if err := nuo.Exec(ctx); err != nil {
		panic(err)
	}
}

func (nuo *NetworkUpdateOne) sqlSave(ctx context.Context) (_node *Network, err error) {
	_spec := &sqlgraph.UpdateSpec{
		Node: &sqlgraph.NodeSpec{
			Table:   network.Table,
			Columns: network.Columns,
			ID: &sqlgraph.FieldSpec{
				Type:   field.TypeUUID,
				Column: network.FieldID,
			},
		},
	}
	id, ok := nuo.mutation.ID()
	if !ok {
		return nil, &ValidationError{Name: "ID", err: fmt.Errorf("missing Network.ID for update")}
	}
	_spec.Node.ID.Value = id
	if fields := nuo.fields; len(fields) > 0 {
		_spec.Node.Columns = make([]string, 0, len(fields))
		_spec.Node.Columns = append(_spec.Node.Columns, network.FieldID)
		for _, f := range fields {
			if !network.ValidColumn(f) {
				return nil, &ValidationError{Name: f, err: fmt.Errorf("ent: invalid field %q for query", f)}
			}
			if f != network.FieldID {
				_spec.Node.Columns = append(_spec.Node.Columns, f)
			}
		}
	}
	if ps := nuo.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if value, ok := nuo.mutation.HclID(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Value:  value,
			Column: network.FieldHclID,
		})
	}
	if value, ok := nuo.mutation.Name(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Value:  value,
			Column: network.FieldName,
		})
	}
	if value, ok := nuo.mutation.Cidr(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Value:  value,
			Column: network.FieldCidr,
		})
	}
	if value, ok := nuo.mutation.VdiVisible(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeBool,
			Value:  value,
			Column: network.FieldVdiVisible,
		})
	}
	if value, ok := nuo.mutation.Vars(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeJSON,
			Value:  value,
			Column: network.FieldVars,
		})
	}
	if value, ok := nuo.mutation.Tags(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeJSON,
			Value:  value,
			Column: network.FieldTags,
		})
	}
	if nuo.mutation.NetworkToEnvironmentCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: true,
			Table:   network.NetworkToEnvironmentTable,
			Columns: []string{network.NetworkToEnvironmentColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeUUID,
					Column: environment.FieldID,
				},
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := nuo.mutation.NetworkToEnvironmentIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: true,
			Table:   network.NetworkToEnvironmentTable,
			Columns: []string{network.NetworkToEnvironmentColumn},
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
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	if nuo.mutation.NetworkToHostDependencyCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: true,
			Table:   network.NetworkToHostDependencyTable,
			Columns: []string{network.NetworkToHostDependencyColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeUUID,
					Column: hostdependency.FieldID,
				},
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := nuo.mutation.RemovedNetworkToHostDependencyIDs(); len(nodes) > 0 && !nuo.mutation.NetworkToHostDependencyCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: true,
			Table:   network.NetworkToHostDependencyTable,
			Columns: []string{network.NetworkToHostDependencyColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeUUID,
					Column: hostdependency.FieldID,
				},
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := nuo.mutation.NetworkToHostDependencyIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: true,
			Table:   network.NetworkToHostDependencyTable,
			Columns: []string{network.NetworkToHostDependencyColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeUUID,
					Column: hostdependency.FieldID,
				},
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	if nuo.mutation.NetworkToIncludedNetworkCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: true,
			Table:   network.NetworkToIncludedNetworkTable,
			Columns: []string{network.NetworkToIncludedNetworkColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeUUID,
					Column: includednetwork.FieldID,
				},
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := nuo.mutation.RemovedNetworkToIncludedNetworkIDs(); len(nodes) > 0 && !nuo.mutation.NetworkToIncludedNetworkCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: true,
			Table:   network.NetworkToIncludedNetworkTable,
			Columns: []string{network.NetworkToIncludedNetworkColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeUUID,
					Column: includednetwork.FieldID,
				},
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := nuo.mutation.NetworkToIncludedNetworkIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: true,
			Table:   network.NetworkToIncludedNetworkTable,
			Columns: []string{network.NetworkToIncludedNetworkColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeUUID,
					Column: includednetwork.FieldID,
				},
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	_node = &Network{config: nuo.config}
	_spec.Assign = _node.assignValues
	_spec.ScanValues = _node.scanValues
	if err = sqlgraph.UpdateNode(ctx, nuo.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{network.Label}
		} else if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{err.Error(), err}
		}
		return nil, err
	}
	return _node, nil
}
