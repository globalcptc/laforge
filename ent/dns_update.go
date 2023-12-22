// Code generated by ent, DO NOT EDIT.

package ent

import (
	"context"
	"errors"
	"fmt"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/dialect/sql/sqljson"
	"entgo.io/ent/schema/field"
	"github.com/gen0cide/laforge/ent/competition"
	"github.com/gen0cide/laforge/ent/dns"
	"github.com/gen0cide/laforge/ent/environment"
	"github.com/gen0cide/laforge/ent/predicate"
	"github.com/google/uuid"
)

// DNSUpdate is the builder for updating DNS entities.
type DNSUpdate struct {
	config
	hooks    []Hook
	mutation *DNSMutation
}

// Where appends a list predicates to the DNSUpdate builder.
func (du *DNSUpdate) Where(ps ...predicate.DNS) *DNSUpdate {
	du.mutation.Where(ps...)
	return du
}

// SetHCLID sets the "hcl_id" field.
func (du *DNSUpdate) SetHCLID(s string) *DNSUpdate {
	du.mutation.SetHCLID(s)
	return du
}

// SetNillableHCLID sets the "hcl_id" field if the given value is not nil.
func (du *DNSUpdate) SetNillableHCLID(s *string) *DNSUpdate {
	if s != nil {
		du.SetHCLID(*s)
	}
	return du
}

// SetType sets the "type" field.
func (du *DNSUpdate) SetType(s string) *DNSUpdate {
	du.mutation.SetType(s)
	return du
}

// SetNillableType sets the "type" field if the given value is not nil.
func (du *DNSUpdate) SetNillableType(s *string) *DNSUpdate {
	if s != nil {
		du.SetType(*s)
	}
	return du
}

// SetRootDomain sets the "root_domain" field.
func (du *DNSUpdate) SetRootDomain(s string) *DNSUpdate {
	du.mutation.SetRootDomain(s)
	return du
}

// SetNillableRootDomain sets the "root_domain" field if the given value is not nil.
func (du *DNSUpdate) SetNillableRootDomain(s *string) *DNSUpdate {
	if s != nil {
		du.SetRootDomain(*s)
	}
	return du
}

// SetDNSServers sets the "dns_servers" field.
func (du *DNSUpdate) SetDNSServers(s []string) *DNSUpdate {
	du.mutation.SetDNSServers(s)
	return du
}

// AppendDNSServers appends s to the "dns_servers" field.
func (du *DNSUpdate) AppendDNSServers(s []string) *DNSUpdate {
	du.mutation.AppendDNSServers(s)
	return du
}

// SetNtpServers sets the "ntp_servers" field.
func (du *DNSUpdate) SetNtpServers(s []string) *DNSUpdate {
	du.mutation.SetNtpServers(s)
	return du
}

// AppendNtpServers appends s to the "ntp_servers" field.
func (du *DNSUpdate) AppendNtpServers(s []string) *DNSUpdate {
	du.mutation.AppendNtpServers(s)
	return du
}

// SetConfig sets the "config" field.
func (du *DNSUpdate) SetConfig(m map[string]string) *DNSUpdate {
	du.mutation.SetConfig(m)
	return du
}

// AddEnvironmentIDs adds the "Environments" edge to the Environment entity by IDs.
func (du *DNSUpdate) AddEnvironmentIDs(ids ...uuid.UUID) *DNSUpdate {
	du.mutation.AddEnvironmentIDs(ids...)
	return du
}

// AddEnvironments adds the "Environments" edges to the Environment entity.
func (du *DNSUpdate) AddEnvironments(e ...*Environment) *DNSUpdate {
	ids := make([]uuid.UUID, len(e))
	for i := range e {
		ids[i] = e[i].ID
	}
	return du.AddEnvironmentIDs(ids...)
}

// AddCompetitionIDs adds the "Competitions" edge to the Competition entity by IDs.
func (du *DNSUpdate) AddCompetitionIDs(ids ...uuid.UUID) *DNSUpdate {
	du.mutation.AddCompetitionIDs(ids...)
	return du
}

// AddCompetitions adds the "Competitions" edges to the Competition entity.
func (du *DNSUpdate) AddCompetitions(c ...*Competition) *DNSUpdate {
	ids := make([]uuid.UUID, len(c))
	for i := range c {
		ids[i] = c[i].ID
	}
	return du.AddCompetitionIDs(ids...)
}

// Mutation returns the DNSMutation object of the builder.
func (du *DNSUpdate) Mutation() *DNSMutation {
	return du.mutation
}

// ClearEnvironments clears all "Environments" edges to the Environment entity.
func (du *DNSUpdate) ClearEnvironments() *DNSUpdate {
	du.mutation.ClearEnvironments()
	return du
}

// RemoveEnvironmentIDs removes the "Environments" edge to Environment entities by IDs.
func (du *DNSUpdate) RemoveEnvironmentIDs(ids ...uuid.UUID) *DNSUpdate {
	du.mutation.RemoveEnvironmentIDs(ids...)
	return du
}

// RemoveEnvironments removes "Environments" edges to Environment entities.
func (du *DNSUpdate) RemoveEnvironments(e ...*Environment) *DNSUpdate {
	ids := make([]uuid.UUID, len(e))
	for i := range e {
		ids[i] = e[i].ID
	}
	return du.RemoveEnvironmentIDs(ids...)
}

// ClearCompetitions clears all "Competitions" edges to the Competition entity.
func (du *DNSUpdate) ClearCompetitions() *DNSUpdate {
	du.mutation.ClearCompetitions()
	return du
}

// RemoveCompetitionIDs removes the "Competitions" edge to Competition entities by IDs.
func (du *DNSUpdate) RemoveCompetitionIDs(ids ...uuid.UUID) *DNSUpdate {
	du.mutation.RemoveCompetitionIDs(ids...)
	return du
}

// RemoveCompetitions removes "Competitions" edges to Competition entities.
func (du *DNSUpdate) RemoveCompetitions(c ...*Competition) *DNSUpdate {
	ids := make([]uuid.UUID, len(c))
	for i := range c {
		ids[i] = c[i].ID
	}
	return du.RemoveCompetitionIDs(ids...)
}

// Save executes the query and returns the number of nodes affected by the update operation.
func (du *DNSUpdate) Save(ctx context.Context) (int, error) {
	return withHooks(ctx, du.sqlSave, du.mutation, du.hooks)
}

// SaveX is like Save, but panics if an error occurs.
func (du *DNSUpdate) SaveX(ctx context.Context) int {
	affected, err := du.Save(ctx)
	if err != nil {
		panic(err)
	}
	return affected
}

// Exec executes the query.
func (du *DNSUpdate) Exec(ctx context.Context) error {
	_, err := du.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (du *DNSUpdate) ExecX(ctx context.Context) {
	if err := du.Exec(ctx); err != nil {
		panic(err)
	}
}

func (du *DNSUpdate) sqlSave(ctx context.Context) (n int, err error) {
	_spec := sqlgraph.NewUpdateSpec(dns.Table, dns.Columns, sqlgraph.NewFieldSpec(dns.FieldID, field.TypeUUID))
	if ps := du.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if value, ok := du.mutation.HCLID(); ok {
		_spec.SetField(dns.FieldHCLID, field.TypeString, value)
	}
	if value, ok := du.mutation.GetType(); ok {
		_spec.SetField(dns.FieldType, field.TypeString, value)
	}
	if value, ok := du.mutation.RootDomain(); ok {
		_spec.SetField(dns.FieldRootDomain, field.TypeString, value)
	}
	if value, ok := du.mutation.DNSServers(); ok {
		_spec.SetField(dns.FieldDNSServers, field.TypeJSON, value)
	}
	if value, ok := du.mutation.AppendedDNSServers(); ok {
		_spec.AddModifier(func(u *sql.UpdateBuilder) {
			sqljson.Append(u, dns.FieldDNSServers, value)
		})
	}
	if value, ok := du.mutation.NtpServers(); ok {
		_spec.SetField(dns.FieldNtpServers, field.TypeJSON, value)
	}
	if value, ok := du.mutation.AppendedNtpServers(); ok {
		_spec.AddModifier(func(u *sql.UpdateBuilder) {
			sqljson.Append(u, dns.FieldNtpServers, value)
		})
	}
	if value, ok := du.mutation.Config(); ok {
		_spec.SetField(dns.FieldConfig, field.TypeJSON, value)
	}
	if du.mutation.EnvironmentsCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2M,
			Inverse: true,
			Table:   dns.EnvironmentsTable,
			Columns: dns.EnvironmentsPrimaryKey,
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(environment.FieldID, field.TypeUUID),
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := du.mutation.RemovedEnvironmentsIDs(); len(nodes) > 0 && !du.mutation.EnvironmentsCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2M,
			Inverse: true,
			Table:   dns.EnvironmentsTable,
			Columns: dns.EnvironmentsPrimaryKey,
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(environment.FieldID, field.TypeUUID),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := du.mutation.EnvironmentsIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2M,
			Inverse: true,
			Table:   dns.EnvironmentsTable,
			Columns: dns.EnvironmentsPrimaryKey,
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(environment.FieldID, field.TypeUUID),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	if du.mutation.CompetitionsCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2M,
			Inverse: true,
			Table:   dns.CompetitionsTable,
			Columns: dns.CompetitionsPrimaryKey,
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(competition.FieldID, field.TypeUUID),
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := du.mutation.RemovedCompetitionsIDs(); len(nodes) > 0 && !du.mutation.CompetitionsCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2M,
			Inverse: true,
			Table:   dns.CompetitionsTable,
			Columns: dns.CompetitionsPrimaryKey,
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(competition.FieldID, field.TypeUUID),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := du.mutation.CompetitionsIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2M,
			Inverse: true,
			Table:   dns.CompetitionsTable,
			Columns: dns.CompetitionsPrimaryKey,
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(competition.FieldID, field.TypeUUID),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	if n, err = sqlgraph.UpdateNodes(ctx, du.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{dns.Label}
		} else if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return 0, err
	}
	du.mutation.done = true
	return n, nil
}

// DNSUpdateOne is the builder for updating a single DNS entity.
type DNSUpdateOne struct {
	config
	fields   []string
	hooks    []Hook
	mutation *DNSMutation
}

// SetHCLID sets the "hcl_id" field.
func (duo *DNSUpdateOne) SetHCLID(s string) *DNSUpdateOne {
	duo.mutation.SetHCLID(s)
	return duo
}

// SetNillableHCLID sets the "hcl_id" field if the given value is not nil.
func (duo *DNSUpdateOne) SetNillableHCLID(s *string) *DNSUpdateOne {
	if s != nil {
		duo.SetHCLID(*s)
	}
	return duo
}

// SetType sets the "type" field.
func (duo *DNSUpdateOne) SetType(s string) *DNSUpdateOne {
	duo.mutation.SetType(s)
	return duo
}

// SetNillableType sets the "type" field if the given value is not nil.
func (duo *DNSUpdateOne) SetNillableType(s *string) *DNSUpdateOne {
	if s != nil {
		duo.SetType(*s)
	}
	return duo
}

// SetRootDomain sets the "root_domain" field.
func (duo *DNSUpdateOne) SetRootDomain(s string) *DNSUpdateOne {
	duo.mutation.SetRootDomain(s)
	return duo
}

// SetNillableRootDomain sets the "root_domain" field if the given value is not nil.
func (duo *DNSUpdateOne) SetNillableRootDomain(s *string) *DNSUpdateOne {
	if s != nil {
		duo.SetRootDomain(*s)
	}
	return duo
}

// SetDNSServers sets the "dns_servers" field.
func (duo *DNSUpdateOne) SetDNSServers(s []string) *DNSUpdateOne {
	duo.mutation.SetDNSServers(s)
	return duo
}

// AppendDNSServers appends s to the "dns_servers" field.
func (duo *DNSUpdateOne) AppendDNSServers(s []string) *DNSUpdateOne {
	duo.mutation.AppendDNSServers(s)
	return duo
}

// SetNtpServers sets the "ntp_servers" field.
func (duo *DNSUpdateOne) SetNtpServers(s []string) *DNSUpdateOne {
	duo.mutation.SetNtpServers(s)
	return duo
}

// AppendNtpServers appends s to the "ntp_servers" field.
func (duo *DNSUpdateOne) AppendNtpServers(s []string) *DNSUpdateOne {
	duo.mutation.AppendNtpServers(s)
	return duo
}

// SetConfig sets the "config" field.
func (duo *DNSUpdateOne) SetConfig(m map[string]string) *DNSUpdateOne {
	duo.mutation.SetConfig(m)
	return duo
}

// AddEnvironmentIDs adds the "Environments" edge to the Environment entity by IDs.
func (duo *DNSUpdateOne) AddEnvironmentIDs(ids ...uuid.UUID) *DNSUpdateOne {
	duo.mutation.AddEnvironmentIDs(ids...)
	return duo
}

// AddEnvironments adds the "Environments" edges to the Environment entity.
func (duo *DNSUpdateOne) AddEnvironments(e ...*Environment) *DNSUpdateOne {
	ids := make([]uuid.UUID, len(e))
	for i := range e {
		ids[i] = e[i].ID
	}
	return duo.AddEnvironmentIDs(ids...)
}

// AddCompetitionIDs adds the "Competitions" edge to the Competition entity by IDs.
func (duo *DNSUpdateOne) AddCompetitionIDs(ids ...uuid.UUID) *DNSUpdateOne {
	duo.mutation.AddCompetitionIDs(ids...)
	return duo
}

// AddCompetitions adds the "Competitions" edges to the Competition entity.
func (duo *DNSUpdateOne) AddCompetitions(c ...*Competition) *DNSUpdateOne {
	ids := make([]uuid.UUID, len(c))
	for i := range c {
		ids[i] = c[i].ID
	}
	return duo.AddCompetitionIDs(ids...)
}

// Mutation returns the DNSMutation object of the builder.
func (duo *DNSUpdateOne) Mutation() *DNSMutation {
	return duo.mutation
}

// ClearEnvironments clears all "Environments" edges to the Environment entity.
func (duo *DNSUpdateOne) ClearEnvironments() *DNSUpdateOne {
	duo.mutation.ClearEnvironments()
	return duo
}

// RemoveEnvironmentIDs removes the "Environments" edge to Environment entities by IDs.
func (duo *DNSUpdateOne) RemoveEnvironmentIDs(ids ...uuid.UUID) *DNSUpdateOne {
	duo.mutation.RemoveEnvironmentIDs(ids...)
	return duo
}

// RemoveEnvironments removes "Environments" edges to Environment entities.
func (duo *DNSUpdateOne) RemoveEnvironments(e ...*Environment) *DNSUpdateOne {
	ids := make([]uuid.UUID, len(e))
	for i := range e {
		ids[i] = e[i].ID
	}
	return duo.RemoveEnvironmentIDs(ids...)
}

// ClearCompetitions clears all "Competitions" edges to the Competition entity.
func (duo *DNSUpdateOne) ClearCompetitions() *DNSUpdateOne {
	duo.mutation.ClearCompetitions()
	return duo
}

// RemoveCompetitionIDs removes the "Competitions" edge to Competition entities by IDs.
func (duo *DNSUpdateOne) RemoveCompetitionIDs(ids ...uuid.UUID) *DNSUpdateOne {
	duo.mutation.RemoveCompetitionIDs(ids...)
	return duo
}

// RemoveCompetitions removes "Competitions" edges to Competition entities.
func (duo *DNSUpdateOne) RemoveCompetitions(c ...*Competition) *DNSUpdateOne {
	ids := make([]uuid.UUID, len(c))
	for i := range c {
		ids[i] = c[i].ID
	}
	return duo.RemoveCompetitionIDs(ids...)
}

// Where appends a list predicates to the DNSUpdate builder.
func (duo *DNSUpdateOne) Where(ps ...predicate.DNS) *DNSUpdateOne {
	duo.mutation.Where(ps...)
	return duo
}

// Select allows selecting one or more fields (columns) of the returned entity.
// The default is selecting all fields defined in the entity schema.
func (duo *DNSUpdateOne) Select(field string, fields ...string) *DNSUpdateOne {
	duo.fields = append([]string{field}, fields...)
	return duo
}

// Save executes the query and returns the updated DNS entity.
func (duo *DNSUpdateOne) Save(ctx context.Context) (*DNS, error) {
	return withHooks(ctx, duo.sqlSave, duo.mutation, duo.hooks)
}

// SaveX is like Save, but panics if an error occurs.
func (duo *DNSUpdateOne) SaveX(ctx context.Context) *DNS {
	node, err := duo.Save(ctx)
	if err != nil {
		panic(err)
	}
	return node
}

// Exec executes the query on the entity.
func (duo *DNSUpdateOne) Exec(ctx context.Context) error {
	_, err := duo.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (duo *DNSUpdateOne) ExecX(ctx context.Context) {
	if err := duo.Exec(ctx); err != nil {
		panic(err)
	}
}

func (duo *DNSUpdateOne) sqlSave(ctx context.Context) (_node *DNS, err error) {
	_spec := sqlgraph.NewUpdateSpec(dns.Table, dns.Columns, sqlgraph.NewFieldSpec(dns.FieldID, field.TypeUUID))
	id, ok := duo.mutation.ID()
	if !ok {
		return nil, &ValidationError{Name: "id", err: errors.New(`ent: missing "DNS.id" for update`)}
	}
	_spec.Node.ID.Value = id
	if fields := duo.fields; len(fields) > 0 {
		_spec.Node.Columns = make([]string, 0, len(fields))
		_spec.Node.Columns = append(_spec.Node.Columns, dns.FieldID)
		for _, f := range fields {
			if !dns.ValidColumn(f) {
				return nil, &ValidationError{Name: f, err: fmt.Errorf("ent: invalid field %q for query", f)}
			}
			if f != dns.FieldID {
				_spec.Node.Columns = append(_spec.Node.Columns, f)
			}
		}
	}
	if ps := duo.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if value, ok := duo.mutation.HCLID(); ok {
		_spec.SetField(dns.FieldHCLID, field.TypeString, value)
	}
	if value, ok := duo.mutation.GetType(); ok {
		_spec.SetField(dns.FieldType, field.TypeString, value)
	}
	if value, ok := duo.mutation.RootDomain(); ok {
		_spec.SetField(dns.FieldRootDomain, field.TypeString, value)
	}
	if value, ok := duo.mutation.DNSServers(); ok {
		_spec.SetField(dns.FieldDNSServers, field.TypeJSON, value)
	}
	if value, ok := duo.mutation.AppendedDNSServers(); ok {
		_spec.AddModifier(func(u *sql.UpdateBuilder) {
			sqljson.Append(u, dns.FieldDNSServers, value)
		})
	}
	if value, ok := duo.mutation.NtpServers(); ok {
		_spec.SetField(dns.FieldNtpServers, field.TypeJSON, value)
	}
	if value, ok := duo.mutation.AppendedNtpServers(); ok {
		_spec.AddModifier(func(u *sql.UpdateBuilder) {
			sqljson.Append(u, dns.FieldNtpServers, value)
		})
	}
	if value, ok := duo.mutation.Config(); ok {
		_spec.SetField(dns.FieldConfig, field.TypeJSON, value)
	}
	if duo.mutation.EnvironmentsCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2M,
			Inverse: true,
			Table:   dns.EnvironmentsTable,
			Columns: dns.EnvironmentsPrimaryKey,
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(environment.FieldID, field.TypeUUID),
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := duo.mutation.RemovedEnvironmentsIDs(); len(nodes) > 0 && !duo.mutation.EnvironmentsCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2M,
			Inverse: true,
			Table:   dns.EnvironmentsTable,
			Columns: dns.EnvironmentsPrimaryKey,
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(environment.FieldID, field.TypeUUID),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := duo.mutation.EnvironmentsIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2M,
			Inverse: true,
			Table:   dns.EnvironmentsTable,
			Columns: dns.EnvironmentsPrimaryKey,
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(environment.FieldID, field.TypeUUID),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	if duo.mutation.CompetitionsCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2M,
			Inverse: true,
			Table:   dns.CompetitionsTable,
			Columns: dns.CompetitionsPrimaryKey,
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(competition.FieldID, field.TypeUUID),
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := duo.mutation.RemovedCompetitionsIDs(); len(nodes) > 0 && !duo.mutation.CompetitionsCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2M,
			Inverse: true,
			Table:   dns.CompetitionsTable,
			Columns: dns.CompetitionsPrimaryKey,
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(competition.FieldID, field.TypeUUID),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := duo.mutation.CompetitionsIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2M,
			Inverse: true,
			Table:   dns.CompetitionsTable,
			Columns: dns.CompetitionsPrimaryKey,
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(competition.FieldID, field.TypeUUID),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	_node = &DNS{config: duo.config}
	_spec.Assign = _node.assignValues
	_spec.ScanValues = _node.scanValues
	if err = sqlgraph.UpdateNode(ctx, duo.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{dns.Label}
		} else if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return nil, err
	}
	duo.mutation.done = true
	return _node, nil
}
