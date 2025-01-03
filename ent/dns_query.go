// Code generated by ent, DO NOT EDIT.

package ent

import (
	"context"
	"database/sql/driver"
	"fmt"
	"math"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/schema/field"
	"github.com/gen0cide/laforge/ent/competition"
	"github.com/gen0cide/laforge/ent/dns"
	"github.com/gen0cide/laforge/ent/environment"
	"github.com/gen0cide/laforge/ent/predicate"
	"github.com/google/uuid"
)

// DNSQuery is the builder for querying DNS entities.
type DNSQuery struct {
	config
	ctx                   *QueryContext
	order                 []dns.OrderOption
	inters                []Interceptor
	predicates            []predicate.DNS
	withEnvironments      *EnvironmentQuery
	withCompetitions      *CompetitionQuery
	modifiers             []func(*sql.Selector)
	loadTotal             []func(context.Context, []*DNS) error
	withNamedEnvironments map[string]*EnvironmentQuery
	withNamedCompetitions map[string]*CompetitionQuery
	// intermediate query (i.e. traversal path).
	sql  *sql.Selector
	path func(context.Context) (*sql.Selector, error)
}

// Where adds a new predicate for the DNSQuery builder.
func (dq *DNSQuery) Where(ps ...predicate.DNS) *DNSQuery {
	dq.predicates = append(dq.predicates, ps...)
	return dq
}

// Limit the number of records to be returned by this query.
func (dq *DNSQuery) Limit(limit int) *DNSQuery {
	dq.ctx.Limit = &limit
	return dq
}

// Offset to start from.
func (dq *DNSQuery) Offset(offset int) *DNSQuery {
	dq.ctx.Offset = &offset
	return dq
}

// Unique configures the query builder to filter duplicate records on query.
// By default, unique is set to true, and can be disabled using this method.
func (dq *DNSQuery) Unique(unique bool) *DNSQuery {
	dq.ctx.Unique = &unique
	return dq
}

// Order specifies how the records should be ordered.
func (dq *DNSQuery) Order(o ...dns.OrderOption) *DNSQuery {
	dq.order = append(dq.order, o...)
	return dq
}

// QueryEnvironments chains the current query on the "Environments" edge.
func (dq *DNSQuery) QueryEnvironments() *EnvironmentQuery {
	query := (&EnvironmentClient{config: dq.config}).Query()
	query.path = func(ctx context.Context) (fromU *sql.Selector, err error) {
		if err := dq.prepareQuery(ctx); err != nil {
			return nil, err
		}
		selector := dq.sqlQuery(ctx)
		if err := selector.Err(); err != nil {
			return nil, err
		}
		step := sqlgraph.NewStep(
			sqlgraph.From(dns.Table, dns.FieldID, selector),
			sqlgraph.To(environment.Table, environment.FieldID),
			sqlgraph.Edge(sqlgraph.M2M, true, dns.EnvironmentsTable, dns.EnvironmentsPrimaryKey...),
		)
		fromU = sqlgraph.SetNeighbors(dq.driver.Dialect(), step)
		return fromU, nil
	}
	return query
}

// QueryCompetitions chains the current query on the "Competitions" edge.
func (dq *DNSQuery) QueryCompetitions() *CompetitionQuery {
	query := (&CompetitionClient{config: dq.config}).Query()
	query.path = func(ctx context.Context) (fromU *sql.Selector, err error) {
		if err := dq.prepareQuery(ctx); err != nil {
			return nil, err
		}
		selector := dq.sqlQuery(ctx)
		if err := selector.Err(); err != nil {
			return nil, err
		}
		step := sqlgraph.NewStep(
			sqlgraph.From(dns.Table, dns.FieldID, selector),
			sqlgraph.To(competition.Table, competition.FieldID),
			sqlgraph.Edge(sqlgraph.M2M, true, dns.CompetitionsTable, dns.CompetitionsPrimaryKey...),
		)
		fromU = sqlgraph.SetNeighbors(dq.driver.Dialect(), step)
		return fromU, nil
	}
	return query
}

// First returns the first DNS entity from the query.
// Returns a *NotFoundError when no DNS was found.
func (dq *DNSQuery) First(ctx context.Context) (*DNS, error) {
	nodes, err := dq.Limit(1).All(setContextOp(ctx, dq.ctx, "First"))
	if err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nil, &NotFoundError{dns.Label}
	}
	return nodes[0], nil
}

// FirstX is like First, but panics if an error occurs.
func (dq *DNSQuery) FirstX(ctx context.Context) *DNS {
	node, err := dq.First(ctx)
	if err != nil && !IsNotFound(err) {
		panic(err)
	}
	return node
}

// FirstID returns the first DNS ID from the query.
// Returns a *NotFoundError when no DNS ID was found.
func (dq *DNSQuery) FirstID(ctx context.Context) (id uuid.UUID, err error) {
	var ids []uuid.UUID
	if ids, err = dq.Limit(1).IDs(setContextOp(ctx, dq.ctx, "FirstID")); err != nil {
		return
	}
	if len(ids) == 0 {
		err = &NotFoundError{dns.Label}
		return
	}
	return ids[0], nil
}

// FirstIDX is like FirstID, but panics if an error occurs.
func (dq *DNSQuery) FirstIDX(ctx context.Context) uuid.UUID {
	id, err := dq.FirstID(ctx)
	if err != nil && !IsNotFound(err) {
		panic(err)
	}
	return id
}

// Only returns a single DNS entity found by the query, ensuring it only returns one.
// Returns a *NotSingularError when more than one DNS entity is found.
// Returns a *NotFoundError when no DNS entities are found.
func (dq *DNSQuery) Only(ctx context.Context) (*DNS, error) {
	nodes, err := dq.Limit(2).All(setContextOp(ctx, dq.ctx, "Only"))
	if err != nil {
		return nil, err
	}
	switch len(nodes) {
	case 1:
		return nodes[0], nil
	case 0:
		return nil, &NotFoundError{dns.Label}
	default:
		return nil, &NotSingularError{dns.Label}
	}
}

// OnlyX is like Only, but panics if an error occurs.
func (dq *DNSQuery) OnlyX(ctx context.Context) *DNS {
	node, err := dq.Only(ctx)
	if err != nil {
		panic(err)
	}
	return node
}

// OnlyID is like Only, but returns the only DNS ID in the query.
// Returns a *NotSingularError when more than one DNS ID is found.
// Returns a *NotFoundError when no entities are found.
func (dq *DNSQuery) OnlyID(ctx context.Context) (id uuid.UUID, err error) {
	var ids []uuid.UUID
	if ids, err = dq.Limit(2).IDs(setContextOp(ctx, dq.ctx, "OnlyID")); err != nil {
		return
	}
	switch len(ids) {
	case 1:
		id = ids[0]
	case 0:
		err = &NotFoundError{dns.Label}
	default:
		err = &NotSingularError{dns.Label}
	}
	return
}

// OnlyIDX is like OnlyID, but panics if an error occurs.
func (dq *DNSQuery) OnlyIDX(ctx context.Context) uuid.UUID {
	id, err := dq.OnlyID(ctx)
	if err != nil {
		panic(err)
	}
	return id
}

// All executes the query and returns a list of DNSs.
func (dq *DNSQuery) All(ctx context.Context) ([]*DNS, error) {
	ctx = setContextOp(ctx, dq.ctx, "All")
	if err := dq.prepareQuery(ctx); err != nil {
		return nil, err
	}
	qr := querierAll[[]*DNS, *DNSQuery]()
	return withInterceptors[[]*DNS](ctx, dq, qr, dq.inters)
}

// AllX is like All, but panics if an error occurs.
func (dq *DNSQuery) AllX(ctx context.Context) []*DNS {
	nodes, err := dq.All(ctx)
	if err != nil {
		panic(err)
	}
	return nodes
}

// IDs executes the query and returns a list of DNS IDs.
func (dq *DNSQuery) IDs(ctx context.Context) (ids []uuid.UUID, err error) {
	if dq.ctx.Unique == nil && dq.path != nil {
		dq.Unique(true)
	}
	ctx = setContextOp(ctx, dq.ctx, "IDs")
	if err = dq.Select(dns.FieldID).Scan(ctx, &ids); err != nil {
		return nil, err
	}
	return ids, nil
}

// IDsX is like IDs, but panics if an error occurs.
func (dq *DNSQuery) IDsX(ctx context.Context) []uuid.UUID {
	ids, err := dq.IDs(ctx)
	if err != nil {
		panic(err)
	}
	return ids
}

// Count returns the count of the given query.
func (dq *DNSQuery) Count(ctx context.Context) (int, error) {
	ctx = setContextOp(ctx, dq.ctx, "Count")
	if err := dq.prepareQuery(ctx); err != nil {
		return 0, err
	}
	return withInterceptors[int](ctx, dq, querierCount[*DNSQuery](), dq.inters)
}

// CountX is like Count, but panics if an error occurs.
func (dq *DNSQuery) CountX(ctx context.Context) int {
	count, err := dq.Count(ctx)
	if err != nil {
		panic(err)
	}
	return count
}

// Exist returns true if the query has elements in the graph.
func (dq *DNSQuery) Exist(ctx context.Context) (bool, error) {
	ctx = setContextOp(ctx, dq.ctx, "Exist")
	switch _, err := dq.FirstID(ctx); {
	case IsNotFound(err):
		return false, nil
	case err != nil:
		return false, fmt.Errorf("ent: check existence: %w", err)
	default:
		return true, nil
	}
}

// ExistX is like Exist, but panics if an error occurs.
func (dq *DNSQuery) ExistX(ctx context.Context) bool {
	exist, err := dq.Exist(ctx)
	if err != nil {
		panic(err)
	}
	return exist
}

// Clone returns a duplicate of the DNSQuery builder, including all associated steps. It can be
// used to prepare common query builders and use them differently after the clone is made.
func (dq *DNSQuery) Clone() *DNSQuery {
	if dq == nil {
		return nil
	}
	return &DNSQuery{
		config:           dq.config,
		ctx:              dq.ctx.Clone(),
		order:            append([]dns.OrderOption{}, dq.order...),
		inters:           append([]Interceptor{}, dq.inters...),
		predicates:       append([]predicate.DNS{}, dq.predicates...),
		withEnvironments: dq.withEnvironments.Clone(),
		withCompetitions: dq.withCompetitions.Clone(),
		// clone intermediate query.
		sql:  dq.sql.Clone(),
		path: dq.path,
	}
}

// WithEnvironments tells the query-builder to eager-load the nodes that are connected to
// the "Environments" edge. The optional arguments are used to configure the query builder of the edge.
func (dq *DNSQuery) WithEnvironments(opts ...func(*EnvironmentQuery)) *DNSQuery {
	query := (&EnvironmentClient{config: dq.config}).Query()
	for _, opt := range opts {
		opt(query)
	}
	dq.withEnvironments = query
	return dq
}

// WithCompetitions tells the query-builder to eager-load the nodes that are connected to
// the "Competitions" edge. The optional arguments are used to configure the query builder of the edge.
func (dq *DNSQuery) WithCompetitions(opts ...func(*CompetitionQuery)) *DNSQuery {
	query := (&CompetitionClient{config: dq.config}).Query()
	for _, opt := range opts {
		opt(query)
	}
	dq.withCompetitions = query
	return dq
}

// GroupBy is used to group vertices by one or more fields/columns.
// It is often used with aggregate functions, like: count, max, mean, min, sum.
//
// Example:
//
//	var v []struct {
//		HCLID string `json:"hcl_id,omitempty" hcl:"id,label"`
//		Count int `json:"count,omitempty"`
//	}
//
//	client.DNS.Query().
//		GroupBy(dns.FieldHCLID).
//		Aggregate(ent.Count()).
//		Scan(ctx, &v)
func (dq *DNSQuery) GroupBy(field string, fields ...string) *DNSGroupBy {
	dq.ctx.Fields = append([]string{field}, fields...)
	grbuild := &DNSGroupBy{build: dq}
	grbuild.flds = &dq.ctx.Fields
	grbuild.label = dns.Label
	grbuild.scan = grbuild.Scan
	return grbuild
}

// Select allows the selection one or more fields/columns for the given query,
// instead of selecting all fields in the entity.
//
// Example:
//
//	var v []struct {
//		HCLID string `json:"hcl_id,omitempty" hcl:"id,label"`
//	}
//
//	client.DNS.Query().
//		Select(dns.FieldHCLID).
//		Scan(ctx, &v)
func (dq *DNSQuery) Select(fields ...string) *DNSSelect {
	dq.ctx.Fields = append(dq.ctx.Fields, fields...)
	sbuild := &DNSSelect{DNSQuery: dq}
	sbuild.label = dns.Label
	sbuild.flds, sbuild.scan = &dq.ctx.Fields, sbuild.Scan
	return sbuild
}

// Aggregate returns a DNSSelect configured with the given aggregations.
func (dq *DNSQuery) Aggregate(fns ...AggregateFunc) *DNSSelect {
	return dq.Select().Aggregate(fns...)
}

func (dq *DNSQuery) prepareQuery(ctx context.Context) error {
	for _, inter := range dq.inters {
		if inter == nil {
			return fmt.Errorf("ent: uninitialized interceptor (forgotten import ent/runtime?)")
		}
		if trv, ok := inter.(Traverser); ok {
			if err := trv.Traverse(ctx, dq); err != nil {
				return err
			}
		}
	}
	for _, f := range dq.ctx.Fields {
		if !dns.ValidColumn(f) {
			return &ValidationError{Name: f, err: fmt.Errorf("ent: invalid field %q for query", f)}
		}
	}
	if dq.path != nil {
		prev, err := dq.path(ctx)
		if err != nil {
			return err
		}
		dq.sql = prev
	}
	return nil
}

func (dq *DNSQuery) sqlAll(ctx context.Context, hooks ...queryHook) ([]*DNS, error) {
	var (
		nodes       = []*DNS{}
		_spec       = dq.querySpec()
		loadedTypes = [2]bool{
			dq.withEnvironments != nil,
			dq.withCompetitions != nil,
		}
	)
	_spec.ScanValues = func(columns []string) ([]any, error) {
		return (*DNS).scanValues(nil, columns)
	}
	_spec.Assign = func(columns []string, values []any) error {
		node := &DNS{config: dq.config}
		nodes = append(nodes, node)
		node.Edges.loadedTypes = loadedTypes
		return node.assignValues(columns, values)
	}
	if len(dq.modifiers) > 0 {
		_spec.Modifiers = dq.modifiers
	}
	for i := range hooks {
		hooks[i](ctx, _spec)
	}
	if err := sqlgraph.QueryNodes(ctx, dq.driver, _spec); err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nodes, nil
	}
	if query := dq.withEnvironments; query != nil {
		if err := dq.loadEnvironments(ctx, query, nodes,
			func(n *DNS) { n.Edges.Environments = []*Environment{} },
			func(n *DNS, e *Environment) { n.Edges.Environments = append(n.Edges.Environments, e) }); err != nil {
			return nil, err
		}
	}
	if query := dq.withCompetitions; query != nil {
		if err := dq.loadCompetitions(ctx, query, nodes,
			func(n *DNS) { n.Edges.Competitions = []*Competition{} },
			func(n *DNS, e *Competition) { n.Edges.Competitions = append(n.Edges.Competitions, e) }); err != nil {
			return nil, err
		}
	}
	for name, query := range dq.withNamedEnvironments {
		if err := dq.loadEnvironments(ctx, query, nodes,
			func(n *DNS) { n.appendNamedEnvironments(name) },
			func(n *DNS, e *Environment) { n.appendNamedEnvironments(name, e) }); err != nil {
			return nil, err
		}
	}
	for name, query := range dq.withNamedCompetitions {
		if err := dq.loadCompetitions(ctx, query, nodes,
			func(n *DNS) { n.appendNamedCompetitions(name) },
			func(n *DNS, e *Competition) { n.appendNamedCompetitions(name, e) }); err != nil {
			return nil, err
		}
	}
	for i := range dq.loadTotal {
		if err := dq.loadTotal[i](ctx, nodes); err != nil {
			return nil, err
		}
	}
	return nodes, nil
}

func (dq *DNSQuery) loadEnvironments(ctx context.Context, query *EnvironmentQuery, nodes []*DNS, init func(*DNS), assign func(*DNS, *Environment)) error {
	edgeIDs := make([]driver.Value, len(nodes))
	byID := make(map[uuid.UUID]*DNS)
	nids := make(map[uuid.UUID]map[*DNS]struct{})
	for i, node := range nodes {
		edgeIDs[i] = node.ID
		byID[node.ID] = node
		if init != nil {
			init(node)
		}
	}
	query.Where(func(s *sql.Selector) {
		joinT := sql.Table(dns.EnvironmentsTable)
		s.Join(joinT).On(s.C(environment.FieldID), joinT.C(dns.EnvironmentsPrimaryKey[0]))
		s.Where(sql.InValues(joinT.C(dns.EnvironmentsPrimaryKey[1]), edgeIDs...))
		columns := s.SelectedColumns()
		s.Select(joinT.C(dns.EnvironmentsPrimaryKey[1]))
		s.AppendSelect(columns...)
		s.SetDistinct(false)
	})
	if err := query.prepareQuery(ctx); err != nil {
		return err
	}
	qr := QuerierFunc(func(ctx context.Context, q Query) (Value, error) {
		return query.sqlAll(ctx, func(_ context.Context, spec *sqlgraph.QuerySpec) {
			assign := spec.Assign
			values := spec.ScanValues
			spec.ScanValues = func(columns []string) ([]any, error) {
				values, err := values(columns[1:])
				if err != nil {
					return nil, err
				}
				return append([]any{new(uuid.UUID)}, values...), nil
			}
			spec.Assign = func(columns []string, values []any) error {
				outValue := *values[0].(*uuid.UUID)
				inValue := *values[1].(*uuid.UUID)
				if nids[inValue] == nil {
					nids[inValue] = map[*DNS]struct{}{byID[outValue]: {}}
					return assign(columns[1:], values[1:])
				}
				nids[inValue][byID[outValue]] = struct{}{}
				return nil
			}
		})
	})
	neighbors, err := withInterceptors[[]*Environment](ctx, query, qr, query.inters)
	if err != nil {
		return err
	}
	for _, n := range neighbors {
		nodes, ok := nids[n.ID]
		if !ok {
			return fmt.Errorf(`unexpected "Environments" node returned %v`, n.ID)
		}
		for kn := range nodes {
			assign(kn, n)
		}
	}
	return nil
}
func (dq *DNSQuery) loadCompetitions(ctx context.Context, query *CompetitionQuery, nodes []*DNS, init func(*DNS), assign func(*DNS, *Competition)) error {
	edgeIDs := make([]driver.Value, len(nodes))
	byID := make(map[uuid.UUID]*DNS)
	nids := make(map[uuid.UUID]map[*DNS]struct{})
	for i, node := range nodes {
		edgeIDs[i] = node.ID
		byID[node.ID] = node
		if init != nil {
			init(node)
		}
	}
	query.Where(func(s *sql.Selector) {
		joinT := sql.Table(dns.CompetitionsTable)
		s.Join(joinT).On(s.C(competition.FieldID), joinT.C(dns.CompetitionsPrimaryKey[0]))
		s.Where(sql.InValues(joinT.C(dns.CompetitionsPrimaryKey[1]), edgeIDs...))
		columns := s.SelectedColumns()
		s.Select(joinT.C(dns.CompetitionsPrimaryKey[1]))
		s.AppendSelect(columns...)
		s.SetDistinct(false)
	})
	if err := query.prepareQuery(ctx); err != nil {
		return err
	}
	qr := QuerierFunc(func(ctx context.Context, q Query) (Value, error) {
		return query.sqlAll(ctx, func(_ context.Context, spec *sqlgraph.QuerySpec) {
			assign := spec.Assign
			values := spec.ScanValues
			spec.ScanValues = func(columns []string) ([]any, error) {
				values, err := values(columns[1:])
				if err != nil {
					return nil, err
				}
				return append([]any{new(uuid.UUID)}, values...), nil
			}
			spec.Assign = func(columns []string, values []any) error {
				outValue := *values[0].(*uuid.UUID)
				inValue := *values[1].(*uuid.UUID)
				if nids[inValue] == nil {
					nids[inValue] = map[*DNS]struct{}{byID[outValue]: {}}
					return assign(columns[1:], values[1:])
				}
				nids[inValue][byID[outValue]] = struct{}{}
				return nil
			}
		})
	})
	neighbors, err := withInterceptors[[]*Competition](ctx, query, qr, query.inters)
	if err != nil {
		return err
	}
	for _, n := range neighbors {
		nodes, ok := nids[n.ID]
		if !ok {
			return fmt.Errorf(`unexpected "Competitions" node returned %v`, n.ID)
		}
		for kn := range nodes {
			assign(kn, n)
		}
	}
	return nil
}

func (dq *DNSQuery) sqlCount(ctx context.Context) (int, error) {
	_spec := dq.querySpec()
	if len(dq.modifiers) > 0 {
		_spec.Modifiers = dq.modifiers
	}
	_spec.Node.Columns = dq.ctx.Fields
	if len(dq.ctx.Fields) > 0 {
		_spec.Unique = dq.ctx.Unique != nil && *dq.ctx.Unique
	}
	return sqlgraph.CountNodes(ctx, dq.driver, _spec)
}

func (dq *DNSQuery) querySpec() *sqlgraph.QuerySpec {
	_spec := sqlgraph.NewQuerySpec(dns.Table, dns.Columns, sqlgraph.NewFieldSpec(dns.FieldID, field.TypeUUID))
	_spec.From = dq.sql
	if unique := dq.ctx.Unique; unique != nil {
		_spec.Unique = *unique
	} else if dq.path != nil {
		_spec.Unique = true
	}
	if fields := dq.ctx.Fields; len(fields) > 0 {
		_spec.Node.Columns = make([]string, 0, len(fields))
		_spec.Node.Columns = append(_spec.Node.Columns, dns.FieldID)
		for i := range fields {
			if fields[i] != dns.FieldID {
				_spec.Node.Columns = append(_spec.Node.Columns, fields[i])
			}
		}
	}
	if ps := dq.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if limit := dq.ctx.Limit; limit != nil {
		_spec.Limit = *limit
	}
	if offset := dq.ctx.Offset; offset != nil {
		_spec.Offset = *offset
	}
	if ps := dq.order; len(ps) > 0 {
		_spec.Order = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	return _spec
}

func (dq *DNSQuery) sqlQuery(ctx context.Context) *sql.Selector {
	builder := sql.Dialect(dq.driver.Dialect())
	t1 := builder.Table(dns.Table)
	columns := dq.ctx.Fields
	if len(columns) == 0 {
		columns = dns.Columns
	}
	selector := builder.Select(t1.Columns(columns...)...).From(t1)
	if dq.sql != nil {
		selector = dq.sql
		selector.Select(selector.Columns(columns...)...)
	}
	if dq.ctx.Unique != nil && *dq.ctx.Unique {
		selector.Distinct()
	}
	for _, p := range dq.predicates {
		p(selector)
	}
	for _, p := range dq.order {
		p(selector)
	}
	if offset := dq.ctx.Offset; offset != nil {
		// limit is mandatory for offset clause. We start
		// with default value, and override it below if needed.
		selector.Offset(*offset).Limit(math.MaxInt32)
	}
	if limit := dq.ctx.Limit; limit != nil {
		selector.Limit(*limit)
	}
	return selector
}

// WithNamedEnvironments tells the query-builder to eager-load the nodes that are connected to the "Environments"
// edge with the given name. The optional arguments are used to configure the query builder of the edge.
func (dq *DNSQuery) WithNamedEnvironments(name string, opts ...func(*EnvironmentQuery)) *DNSQuery {
	query := (&EnvironmentClient{config: dq.config}).Query()
	for _, opt := range opts {
		opt(query)
	}
	if dq.withNamedEnvironments == nil {
		dq.withNamedEnvironments = make(map[string]*EnvironmentQuery)
	}
	dq.withNamedEnvironments[name] = query
	return dq
}

// WithNamedCompetitions tells the query-builder to eager-load the nodes that are connected to the "Competitions"
// edge with the given name. The optional arguments are used to configure the query builder of the edge.
func (dq *DNSQuery) WithNamedCompetitions(name string, opts ...func(*CompetitionQuery)) *DNSQuery {
	query := (&CompetitionClient{config: dq.config}).Query()
	for _, opt := range opts {
		opt(query)
	}
	if dq.withNamedCompetitions == nil {
		dq.withNamedCompetitions = make(map[string]*CompetitionQuery)
	}
	dq.withNamedCompetitions[name] = query
	return dq
}

// DNSGroupBy is the group-by builder for DNS entities.
type DNSGroupBy struct {
	selector
	build *DNSQuery
}

// Aggregate adds the given aggregation functions to the group-by query.
func (dgb *DNSGroupBy) Aggregate(fns ...AggregateFunc) *DNSGroupBy {
	dgb.fns = append(dgb.fns, fns...)
	return dgb
}

// Scan applies the selector query and scans the result into the given value.
func (dgb *DNSGroupBy) Scan(ctx context.Context, v any) error {
	ctx = setContextOp(ctx, dgb.build.ctx, "GroupBy")
	if err := dgb.build.prepareQuery(ctx); err != nil {
		return err
	}
	return scanWithInterceptors[*DNSQuery, *DNSGroupBy](ctx, dgb.build, dgb, dgb.build.inters, v)
}

func (dgb *DNSGroupBy) sqlScan(ctx context.Context, root *DNSQuery, v any) error {
	selector := root.sqlQuery(ctx).Select()
	aggregation := make([]string, 0, len(dgb.fns))
	for _, fn := range dgb.fns {
		aggregation = append(aggregation, fn(selector))
	}
	if len(selector.SelectedColumns()) == 0 {
		columns := make([]string, 0, len(*dgb.flds)+len(dgb.fns))
		for _, f := range *dgb.flds {
			columns = append(columns, selector.C(f))
		}
		columns = append(columns, aggregation...)
		selector.Select(columns...)
	}
	selector.GroupBy(selector.Columns(*dgb.flds...)...)
	if err := selector.Err(); err != nil {
		return err
	}
	rows := &sql.Rows{}
	query, args := selector.Query()
	if err := dgb.build.driver.Query(ctx, query, args, rows); err != nil {
		return err
	}
	defer rows.Close()
	return sql.ScanSlice(rows, v)
}

// DNSSelect is the builder for selecting fields of DNS entities.
type DNSSelect struct {
	*DNSQuery
	selector
}

// Aggregate adds the given aggregation functions to the selector query.
func (ds *DNSSelect) Aggregate(fns ...AggregateFunc) *DNSSelect {
	ds.fns = append(ds.fns, fns...)
	return ds
}

// Scan applies the selector query and scans the result into the given value.
func (ds *DNSSelect) Scan(ctx context.Context, v any) error {
	ctx = setContextOp(ctx, ds.ctx, "Select")
	if err := ds.prepareQuery(ctx); err != nil {
		return err
	}
	return scanWithInterceptors[*DNSQuery, *DNSSelect](ctx, ds.DNSQuery, ds, ds.inters, v)
}

func (ds *DNSSelect) sqlScan(ctx context.Context, root *DNSQuery, v any) error {
	selector := root.sqlQuery(ctx)
	aggregation := make([]string, 0, len(ds.fns))
	for _, fn := range ds.fns {
		aggregation = append(aggregation, fn(selector))
	}
	switch n := len(*ds.selector.flds); {
	case n == 0 && len(aggregation) > 0:
		selector.Select(aggregation...)
	case n != 0 && len(aggregation) > 0:
		selector.AppendSelect(aggregation...)
	}
	rows := &sql.Rows{}
	query, args := selector.Query()
	if err := ds.driver.Query(ctx, query, args, rows); err != nil {
		return err
	}
	defer rows.Close()
	return sql.ScanSlice(rows, v)
}
