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
	"github.com/gen0cide/laforge/ent/environment"
	"github.com/gen0cide/laforge/ent/host"
	"github.com/gen0cide/laforge/ent/includednetwork"
	"github.com/gen0cide/laforge/ent/network"
	"github.com/gen0cide/laforge/ent/predicate"
	"github.com/gen0cide/laforge/ent/tag"
	"github.com/google/uuid"
)

// IncludedNetworkQuery is the builder for querying IncludedNetwork entities.
type IncludedNetworkQuery struct {
	config
	ctx                   *QueryContext
	order                 []includednetwork.OrderOption
	inters                []Interceptor
	predicates            []predicate.IncludedNetwork
	withTags              *TagQuery
	withHosts             *HostQuery
	withNetwork           *NetworkQuery
	withEnvironments      *EnvironmentQuery
	withFKs               bool
	modifiers             []func(*sql.Selector)
	loadTotal             []func(context.Context, []*IncludedNetwork) error
	withNamedTags         map[string]*TagQuery
	withNamedHosts        map[string]*HostQuery
	withNamedEnvironments map[string]*EnvironmentQuery
	// intermediate query (i.e. traversal path).
	sql  *sql.Selector
	path func(context.Context) (*sql.Selector, error)
}

// Where adds a new predicate for the IncludedNetworkQuery builder.
func (inq *IncludedNetworkQuery) Where(ps ...predicate.IncludedNetwork) *IncludedNetworkQuery {
	inq.predicates = append(inq.predicates, ps...)
	return inq
}

// Limit the number of records to be returned by this query.
func (inq *IncludedNetworkQuery) Limit(limit int) *IncludedNetworkQuery {
	inq.ctx.Limit = &limit
	return inq
}

// Offset to start from.
func (inq *IncludedNetworkQuery) Offset(offset int) *IncludedNetworkQuery {
	inq.ctx.Offset = &offset
	return inq
}

// Unique configures the query builder to filter duplicate records on query.
// By default, unique is set to true, and can be disabled using this method.
func (inq *IncludedNetworkQuery) Unique(unique bool) *IncludedNetworkQuery {
	inq.ctx.Unique = &unique
	return inq
}

// Order specifies how the records should be ordered.
func (inq *IncludedNetworkQuery) Order(o ...includednetwork.OrderOption) *IncludedNetworkQuery {
	inq.order = append(inq.order, o...)
	return inq
}

// QueryTags chains the current query on the "Tags" edge.
func (inq *IncludedNetworkQuery) QueryTags() *TagQuery {
	query := (&TagClient{config: inq.config}).Query()
	query.path = func(ctx context.Context) (fromU *sql.Selector, err error) {
		if err := inq.prepareQuery(ctx); err != nil {
			return nil, err
		}
		selector := inq.sqlQuery(ctx)
		if err := selector.Err(); err != nil {
			return nil, err
		}
		step := sqlgraph.NewStep(
			sqlgraph.From(includednetwork.Table, includednetwork.FieldID, selector),
			sqlgraph.To(tag.Table, tag.FieldID),
			sqlgraph.Edge(sqlgraph.O2M, false, includednetwork.TagsTable, includednetwork.TagsColumn),
		)
		fromU = sqlgraph.SetNeighbors(inq.driver.Dialect(), step)
		return fromU, nil
	}
	return query
}

// QueryHosts chains the current query on the "Hosts" edge.
func (inq *IncludedNetworkQuery) QueryHosts() *HostQuery {
	query := (&HostClient{config: inq.config}).Query()
	query.path = func(ctx context.Context) (fromU *sql.Selector, err error) {
		if err := inq.prepareQuery(ctx); err != nil {
			return nil, err
		}
		selector := inq.sqlQuery(ctx)
		if err := selector.Err(); err != nil {
			return nil, err
		}
		step := sqlgraph.NewStep(
			sqlgraph.From(includednetwork.Table, includednetwork.FieldID, selector),
			sqlgraph.To(host.Table, host.FieldID),
			sqlgraph.Edge(sqlgraph.M2M, false, includednetwork.HostsTable, includednetwork.HostsPrimaryKey...),
		)
		fromU = sqlgraph.SetNeighbors(inq.driver.Dialect(), step)
		return fromU, nil
	}
	return query
}

// QueryNetwork chains the current query on the "Network" edge.
func (inq *IncludedNetworkQuery) QueryNetwork() *NetworkQuery {
	query := (&NetworkClient{config: inq.config}).Query()
	query.path = func(ctx context.Context) (fromU *sql.Selector, err error) {
		if err := inq.prepareQuery(ctx); err != nil {
			return nil, err
		}
		selector := inq.sqlQuery(ctx)
		if err := selector.Err(); err != nil {
			return nil, err
		}
		step := sqlgraph.NewStep(
			sqlgraph.From(includednetwork.Table, includednetwork.FieldID, selector),
			sqlgraph.To(network.Table, network.FieldID),
			sqlgraph.Edge(sqlgraph.M2O, false, includednetwork.NetworkTable, includednetwork.NetworkColumn),
		)
		fromU = sqlgraph.SetNeighbors(inq.driver.Dialect(), step)
		return fromU, nil
	}
	return query
}

// QueryEnvironments chains the current query on the "Environments" edge.
func (inq *IncludedNetworkQuery) QueryEnvironments() *EnvironmentQuery {
	query := (&EnvironmentClient{config: inq.config}).Query()
	query.path = func(ctx context.Context) (fromU *sql.Selector, err error) {
		if err := inq.prepareQuery(ctx); err != nil {
			return nil, err
		}
		selector := inq.sqlQuery(ctx)
		if err := selector.Err(); err != nil {
			return nil, err
		}
		step := sqlgraph.NewStep(
			sqlgraph.From(includednetwork.Table, includednetwork.FieldID, selector),
			sqlgraph.To(environment.Table, environment.FieldID),
			sqlgraph.Edge(sqlgraph.M2M, true, includednetwork.EnvironmentsTable, includednetwork.EnvironmentsPrimaryKey...),
		)
		fromU = sqlgraph.SetNeighbors(inq.driver.Dialect(), step)
		return fromU, nil
	}
	return query
}

// First returns the first IncludedNetwork entity from the query.
// Returns a *NotFoundError when no IncludedNetwork was found.
func (inq *IncludedNetworkQuery) First(ctx context.Context) (*IncludedNetwork, error) {
	nodes, err := inq.Limit(1).All(setContextOp(ctx, inq.ctx, "First"))
	if err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nil, &NotFoundError{includednetwork.Label}
	}
	return nodes[0], nil
}

// FirstX is like First, but panics if an error occurs.
func (inq *IncludedNetworkQuery) FirstX(ctx context.Context) *IncludedNetwork {
	node, err := inq.First(ctx)
	if err != nil && !IsNotFound(err) {
		panic(err)
	}
	return node
}

// FirstID returns the first IncludedNetwork ID from the query.
// Returns a *NotFoundError when no IncludedNetwork ID was found.
func (inq *IncludedNetworkQuery) FirstID(ctx context.Context) (id uuid.UUID, err error) {
	var ids []uuid.UUID
	if ids, err = inq.Limit(1).IDs(setContextOp(ctx, inq.ctx, "FirstID")); err != nil {
		return
	}
	if len(ids) == 0 {
		err = &NotFoundError{includednetwork.Label}
		return
	}
	return ids[0], nil
}

// FirstIDX is like FirstID, but panics if an error occurs.
func (inq *IncludedNetworkQuery) FirstIDX(ctx context.Context) uuid.UUID {
	id, err := inq.FirstID(ctx)
	if err != nil && !IsNotFound(err) {
		panic(err)
	}
	return id
}

// Only returns a single IncludedNetwork entity found by the query, ensuring it only returns one.
// Returns a *NotSingularError when more than one IncludedNetwork entity is found.
// Returns a *NotFoundError when no IncludedNetwork entities are found.
func (inq *IncludedNetworkQuery) Only(ctx context.Context) (*IncludedNetwork, error) {
	nodes, err := inq.Limit(2).All(setContextOp(ctx, inq.ctx, "Only"))
	if err != nil {
		return nil, err
	}
	switch len(nodes) {
	case 1:
		return nodes[0], nil
	case 0:
		return nil, &NotFoundError{includednetwork.Label}
	default:
		return nil, &NotSingularError{includednetwork.Label}
	}
}

// OnlyX is like Only, but panics if an error occurs.
func (inq *IncludedNetworkQuery) OnlyX(ctx context.Context) *IncludedNetwork {
	node, err := inq.Only(ctx)
	if err != nil {
		panic(err)
	}
	return node
}

// OnlyID is like Only, but returns the only IncludedNetwork ID in the query.
// Returns a *NotSingularError when more than one IncludedNetwork ID is found.
// Returns a *NotFoundError when no entities are found.
func (inq *IncludedNetworkQuery) OnlyID(ctx context.Context) (id uuid.UUID, err error) {
	var ids []uuid.UUID
	if ids, err = inq.Limit(2).IDs(setContextOp(ctx, inq.ctx, "OnlyID")); err != nil {
		return
	}
	switch len(ids) {
	case 1:
		id = ids[0]
	case 0:
		err = &NotFoundError{includednetwork.Label}
	default:
		err = &NotSingularError{includednetwork.Label}
	}
	return
}

// OnlyIDX is like OnlyID, but panics if an error occurs.
func (inq *IncludedNetworkQuery) OnlyIDX(ctx context.Context) uuid.UUID {
	id, err := inq.OnlyID(ctx)
	if err != nil {
		panic(err)
	}
	return id
}

// All executes the query and returns a list of IncludedNetworks.
func (inq *IncludedNetworkQuery) All(ctx context.Context) ([]*IncludedNetwork, error) {
	ctx = setContextOp(ctx, inq.ctx, "All")
	if err := inq.prepareQuery(ctx); err != nil {
		return nil, err
	}
	qr := querierAll[[]*IncludedNetwork, *IncludedNetworkQuery]()
	return withInterceptors[[]*IncludedNetwork](ctx, inq, qr, inq.inters)
}

// AllX is like All, but panics if an error occurs.
func (inq *IncludedNetworkQuery) AllX(ctx context.Context) []*IncludedNetwork {
	nodes, err := inq.All(ctx)
	if err != nil {
		panic(err)
	}
	return nodes
}

// IDs executes the query and returns a list of IncludedNetwork IDs.
func (inq *IncludedNetworkQuery) IDs(ctx context.Context) (ids []uuid.UUID, err error) {
	if inq.ctx.Unique == nil && inq.path != nil {
		inq.Unique(true)
	}
	ctx = setContextOp(ctx, inq.ctx, "IDs")
	if err = inq.Select(includednetwork.FieldID).Scan(ctx, &ids); err != nil {
		return nil, err
	}
	return ids, nil
}

// IDsX is like IDs, but panics if an error occurs.
func (inq *IncludedNetworkQuery) IDsX(ctx context.Context) []uuid.UUID {
	ids, err := inq.IDs(ctx)
	if err != nil {
		panic(err)
	}
	return ids
}

// Count returns the count of the given query.
func (inq *IncludedNetworkQuery) Count(ctx context.Context) (int, error) {
	ctx = setContextOp(ctx, inq.ctx, "Count")
	if err := inq.prepareQuery(ctx); err != nil {
		return 0, err
	}
	return withInterceptors[int](ctx, inq, querierCount[*IncludedNetworkQuery](), inq.inters)
}

// CountX is like Count, but panics if an error occurs.
func (inq *IncludedNetworkQuery) CountX(ctx context.Context) int {
	count, err := inq.Count(ctx)
	if err != nil {
		panic(err)
	}
	return count
}

// Exist returns true if the query has elements in the graph.
func (inq *IncludedNetworkQuery) Exist(ctx context.Context) (bool, error) {
	ctx = setContextOp(ctx, inq.ctx, "Exist")
	switch _, err := inq.FirstID(ctx); {
	case IsNotFound(err):
		return false, nil
	case err != nil:
		return false, fmt.Errorf("ent: check existence: %w", err)
	default:
		return true, nil
	}
}

// ExistX is like Exist, but panics if an error occurs.
func (inq *IncludedNetworkQuery) ExistX(ctx context.Context) bool {
	exist, err := inq.Exist(ctx)
	if err != nil {
		panic(err)
	}
	return exist
}

// Clone returns a duplicate of the IncludedNetworkQuery builder, including all associated steps. It can be
// used to prepare common query builders and use them differently after the clone is made.
func (inq *IncludedNetworkQuery) Clone() *IncludedNetworkQuery {
	if inq == nil {
		return nil
	}
	return &IncludedNetworkQuery{
		config:           inq.config,
		ctx:              inq.ctx.Clone(),
		order:            append([]includednetwork.OrderOption{}, inq.order...),
		inters:           append([]Interceptor{}, inq.inters...),
		predicates:       append([]predicate.IncludedNetwork{}, inq.predicates...),
		withTags:         inq.withTags.Clone(),
		withHosts:        inq.withHosts.Clone(),
		withNetwork:      inq.withNetwork.Clone(),
		withEnvironments: inq.withEnvironments.Clone(),
		// clone intermediate query.
		sql:  inq.sql.Clone(),
		path: inq.path,
	}
}

// WithTags tells the query-builder to eager-load the nodes that are connected to
// the "Tags" edge. The optional arguments are used to configure the query builder of the edge.
func (inq *IncludedNetworkQuery) WithTags(opts ...func(*TagQuery)) *IncludedNetworkQuery {
	query := (&TagClient{config: inq.config}).Query()
	for _, opt := range opts {
		opt(query)
	}
	inq.withTags = query
	return inq
}

// WithHosts tells the query-builder to eager-load the nodes that are connected to
// the "Hosts" edge. The optional arguments are used to configure the query builder of the edge.
func (inq *IncludedNetworkQuery) WithHosts(opts ...func(*HostQuery)) *IncludedNetworkQuery {
	query := (&HostClient{config: inq.config}).Query()
	for _, opt := range opts {
		opt(query)
	}
	inq.withHosts = query
	return inq
}

// WithNetwork tells the query-builder to eager-load the nodes that are connected to
// the "Network" edge. The optional arguments are used to configure the query builder of the edge.
func (inq *IncludedNetworkQuery) WithNetwork(opts ...func(*NetworkQuery)) *IncludedNetworkQuery {
	query := (&NetworkClient{config: inq.config}).Query()
	for _, opt := range opts {
		opt(query)
	}
	inq.withNetwork = query
	return inq
}

// WithEnvironments tells the query-builder to eager-load the nodes that are connected to
// the "Environments" edge. The optional arguments are used to configure the query builder of the edge.
func (inq *IncludedNetworkQuery) WithEnvironments(opts ...func(*EnvironmentQuery)) *IncludedNetworkQuery {
	query := (&EnvironmentClient{config: inq.config}).Query()
	for _, opt := range opts {
		opt(query)
	}
	inq.withEnvironments = query
	return inq
}

// GroupBy is used to group vertices by one or more fields/columns.
// It is often used with aggregate functions, like: count, max, mean, min, sum.
//
// Example:
//
//	var v []struct {
//		Name string `json:"name,omitempty" hcl:"name,label"`
//		Count int `json:"count,omitempty"`
//	}
//
//	client.IncludedNetwork.Query().
//		GroupBy(includednetwork.FieldName).
//		Aggregate(ent.Count()).
//		Scan(ctx, &v)
func (inq *IncludedNetworkQuery) GroupBy(field string, fields ...string) *IncludedNetworkGroupBy {
	inq.ctx.Fields = append([]string{field}, fields...)
	grbuild := &IncludedNetworkGroupBy{build: inq}
	grbuild.flds = &inq.ctx.Fields
	grbuild.label = includednetwork.Label
	grbuild.scan = grbuild.Scan
	return grbuild
}

// Select allows the selection one or more fields/columns for the given query,
// instead of selecting all fields in the entity.
//
// Example:
//
//	var v []struct {
//		Name string `json:"name,omitempty" hcl:"name,label"`
//	}
//
//	client.IncludedNetwork.Query().
//		Select(includednetwork.FieldName).
//		Scan(ctx, &v)
func (inq *IncludedNetworkQuery) Select(fields ...string) *IncludedNetworkSelect {
	inq.ctx.Fields = append(inq.ctx.Fields, fields...)
	sbuild := &IncludedNetworkSelect{IncludedNetworkQuery: inq}
	sbuild.label = includednetwork.Label
	sbuild.flds, sbuild.scan = &inq.ctx.Fields, sbuild.Scan
	return sbuild
}

// Aggregate returns a IncludedNetworkSelect configured with the given aggregations.
func (inq *IncludedNetworkQuery) Aggregate(fns ...AggregateFunc) *IncludedNetworkSelect {
	return inq.Select().Aggregate(fns...)
}

func (inq *IncludedNetworkQuery) prepareQuery(ctx context.Context) error {
	for _, inter := range inq.inters {
		if inter == nil {
			return fmt.Errorf("ent: uninitialized interceptor (forgotten import ent/runtime?)")
		}
		if trv, ok := inter.(Traverser); ok {
			if err := trv.Traverse(ctx, inq); err != nil {
				return err
			}
		}
	}
	for _, f := range inq.ctx.Fields {
		if !includednetwork.ValidColumn(f) {
			return &ValidationError{Name: f, err: fmt.Errorf("ent: invalid field %q for query", f)}
		}
	}
	if inq.path != nil {
		prev, err := inq.path(ctx)
		if err != nil {
			return err
		}
		inq.sql = prev
	}
	return nil
}

func (inq *IncludedNetworkQuery) sqlAll(ctx context.Context, hooks ...queryHook) ([]*IncludedNetwork, error) {
	var (
		nodes       = []*IncludedNetwork{}
		withFKs     = inq.withFKs
		_spec       = inq.querySpec()
		loadedTypes = [4]bool{
			inq.withTags != nil,
			inq.withHosts != nil,
			inq.withNetwork != nil,
			inq.withEnvironments != nil,
		}
	)
	if inq.withNetwork != nil {
		withFKs = true
	}
	if withFKs {
		_spec.Node.Columns = append(_spec.Node.Columns, includednetwork.ForeignKeys...)
	}
	_spec.ScanValues = func(columns []string) ([]any, error) {
		return (*IncludedNetwork).scanValues(nil, columns)
	}
	_spec.Assign = func(columns []string, values []any) error {
		node := &IncludedNetwork{config: inq.config}
		nodes = append(nodes, node)
		node.Edges.loadedTypes = loadedTypes
		return node.assignValues(columns, values)
	}
	if len(inq.modifiers) > 0 {
		_spec.Modifiers = inq.modifiers
	}
	for i := range hooks {
		hooks[i](ctx, _spec)
	}
	if err := sqlgraph.QueryNodes(ctx, inq.driver, _spec); err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nodes, nil
	}
	if query := inq.withTags; query != nil {
		if err := inq.loadTags(ctx, query, nodes,
			func(n *IncludedNetwork) { n.Edges.Tags = []*Tag{} },
			func(n *IncludedNetwork, e *Tag) { n.Edges.Tags = append(n.Edges.Tags, e) }); err != nil {
			return nil, err
		}
	}
	if query := inq.withHosts; query != nil {
		if err := inq.loadHosts(ctx, query, nodes,
			func(n *IncludedNetwork) { n.Edges.Hosts = []*Host{} },
			func(n *IncludedNetwork, e *Host) { n.Edges.Hosts = append(n.Edges.Hosts, e) }); err != nil {
			return nil, err
		}
	}
	if query := inq.withNetwork; query != nil {
		if err := inq.loadNetwork(ctx, query, nodes, nil,
			func(n *IncludedNetwork, e *Network) { n.Edges.Network = e }); err != nil {
			return nil, err
		}
	}
	if query := inq.withEnvironments; query != nil {
		if err := inq.loadEnvironments(ctx, query, nodes,
			func(n *IncludedNetwork) { n.Edges.Environments = []*Environment{} },
			func(n *IncludedNetwork, e *Environment) { n.Edges.Environments = append(n.Edges.Environments, e) }); err != nil {
			return nil, err
		}
	}
	for name, query := range inq.withNamedTags {
		if err := inq.loadTags(ctx, query, nodes,
			func(n *IncludedNetwork) { n.appendNamedTags(name) },
			func(n *IncludedNetwork, e *Tag) { n.appendNamedTags(name, e) }); err != nil {
			return nil, err
		}
	}
	for name, query := range inq.withNamedHosts {
		if err := inq.loadHosts(ctx, query, nodes,
			func(n *IncludedNetwork) { n.appendNamedHosts(name) },
			func(n *IncludedNetwork, e *Host) { n.appendNamedHosts(name, e) }); err != nil {
			return nil, err
		}
	}
	for name, query := range inq.withNamedEnvironments {
		if err := inq.loadEnvironments(ctx, query, nodes,
			func(n *IncludedNetwork) { n.appendNamedEnvironments(name) },
			func(n *IncludedNetwork, e *Environment) { n.appendNamedEnvironments(name, e) }); err != nil {
			return nil, err
		}
	}
	for i := range inq.loadTotal {
		if err := inq.loadTotal[i](ctx, nodes); err != nil {
			return nil, err
		}
	}
	return nodes, nil
}

func (inq *IncludedNetworkQuery) loadTags(ctx context.Context, query *TagQuery, nodes []*IncludedNetwork, init func(*IncludedNetwork), assign func(*IncludedNetwork, *Tag)) error {
	fks := make([]driver.Value, 0, len(nodes))
	nodeids := make(map[uuid.UUID]*IncludedNetwork)
	for i := range nodes {
		fks = append(fks, nodes[i].ID)
		nodeids[nodes[i].ID] = nodes[i]
		if init != nil {
			init(nodes[i])
		}
	}
	query.withFKs = true
	query.Where(predicate.Tag(func(s *sql.Selector) {
		s.Where(sql.InValues(s.C(includednetwork.TagsColumn), fks...))
	}))
	neighbors, err := query.All(ctx)
	if err != nil {
		return err
	}
	for _, n := range neighbors {
		fk := n.included_network_tags
		if fk == nil {
			return fmt.Errorf(`foreign-key "included_network_tags" is nil for node %v`, n.ID)
		}
		node, ok := nodeids[*fk]
		if !ok {
			return fmt.Errorf(`unexpected referenced foreign-key "included_network_tags" returned %v for node %v`, *fk, n.ID)
		}
		assign(node, n)
	}
	return nil
}
func (inq *IncludedNetworkQuery) loadHosts(ctx context.Context, query *HostQuery, nodes []*IncludedNetwork, init func(*IncludedNetwork), assign func(*IncludedNetwork, *Host)) error {
	edgeIDs := make([]driver.Value, len(nodes))
	byID := make(map[uuid.UUID]*IncludedNetwork)
	nids := make(map[uuid.UUID]map[*IncludedNetwork]struct{})
	for i, node := range nodes {
		edgeIDs[i] = node.ID
		byID[node.ID] = node
		if init != nil {
			init(node)
		}
	}
	query.Where(func(s *sql.Selector) {
		joinT := sql.Table(includednetwork.HostsTable)
		s.Join(joinT).On(s.C(host.FieldID), joinT.C(includednetwork.HostsPrimaryKey[1]))
		s.Where(sql.InValues(joinT.C(includednetwork.HostsPrimaryKey[0]), edgeIDs...))
		columns := s.SelectedColumns()
		s.Select(joinT.C(includednetwork.HostsPrimaryKey[0]))
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
					nids[inValue] = map[*IncludedNetwork]struct{}{byID[outValue]: {}}
					return assign(columns[1:], values[1:])
				}
				nids[inValue][byID[outValue]] = struct{}{}
				return nil
			}
		})
	})
	neighbors, err := withInterceptors[[]*Host](ctx, query, qr, query.inters)
	if err != nil {
		return err
	}
	for _, n := range neighbors {
		nodes, ok := nids[n.ID]
		if !ok {
			return fmt.Errorf(`unexpected "Hosts" node returned %v`, n.ID)
		}
		for kn := range nodes {
			assign(kn, n)
		}
	}
	return nil
}
func (inq *IncludedNetworkQuery) loadNetwork(ctx context.Context, query *NetworkQuery, nodes []*IncludedNetwork, init func(*IncludedNetwork), assign func(*IncludedNetwork, *Network)) error {
	ids := make([]uuid.UUID, 0, len(nodes))
	nodeids := make(map[uuid.UUID][]*IncludedNetwork)
	for i := range nodes {
		if nodes[i].included_network_network == nil {
			continue
		}
		fk := *nodes[i].included_network_network
		if _, ok := nodeids[fk]; !ok {
			ids = append(ids, fk)
		}
		nodeids[fk] = append(nodeids[fk], nodes[i])
	}
	if len(ids) == 0 {
		return nil
	}
	query.Where(network.IDIn(ids...))
	neighbors, err := query.All(ctx)
	if err != nil {
		return err
	}
	for _, n := range neighbors {
		nodes, ok := nodeids[n.ID]
		if !ok {
			return fmt.Errorf(`unexpected foreign-key "included_network_network" returned %v`, n.ID)
		}
		for i := range nodes {
			assign(nodes[i], n)
		}
	}
	return nil
}
func (inq *IncludedNetworkQuery) loadEnvironments(ctx context.Context, query *EnvironmentQuery, nodes []*IncludedNetwork, init func(*IncludedNetwork), assign func(*IncludedNetwork, *Environment)) error {
	edgeIDs := make([]driver.Value, len(nodes))
	byID := make(map[uuid.UUID]*IncludedNetwork)
	nids := make(map[uuid.UUID]map[*IncludedNetwork]struct{})
	for i, node := range nodes {
		edgeIDs[i] = node.ID
		byID[node.ID] = node
		if init != nil {
			init(node)
		}
	}
	query.Where(func(s *sql.Selector) {
		joinT := sql.Table(includednetwork.EnvironmentsTable)
		s.Join(joinT).On(s.C(environment.FieldID), joinT.C(includednetwork.EnvironmentsPrimaryKey[0]))
		s.Where(sql.InValues(joinT.C(includednetwork.EnvironmentsPrimaryKey[1]), edgeIDs...))
		columns := s.SelectedColumns()
		s.Select(joinT.C(includednetwork.EnvironmentsPrimaryKey[1]))
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
					nids[inValue] = map[*IncludedNetwork]struct{}{byID[outValue]: {}}
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

func (inq *IncludedNetworkQuery) sqlCount(ctx context.Context) (int, error) {
	_spec := inq.querySpec()
	if len(inq.modifiers) > 0 {
		_spec.Modifiers = inq.modifiers
	}
	_spec.Node.Columns = inq.ctx.Fields
	if len(inq.ctx.Fields) > 0 {
		_spec.Unique = inq.ctx.Unique != nil && *inq.ctx.Unique
	}
	return sqlgraph.CountNodes(ctx, inq.driver, _spec)
}

func (inq *IncludedNetworkQuery) querySpec() *sqlgraph.QuerySpec {
	_spec := sqlgraph.NewQuerySpec(includednetwork.Table, includednetwork.Columns, sqlgraph.NewFieldSpec(includednetwork.FieldID, field.TypeUUID))
	_spec.From = inq.sql
	if unique := inq.ctx.Unique; unique != nil {
		_spec.Unique = *unique
	} else if inq.path != nil {
		_spec.Unique = true
	}
	if fields := inq.ctx.Fields; len(fields) > 0 {
		_spec.Node.Columns = make([]string, 0, len(fields))
		_spec.Node.Columns = append(_spec.Node.Columns, includednetwork.FieldID)
		for i := range fields {
			if fields[i] != includednetwork.FieldID {
				_spec.Node.Columns = append(_spec.Node.Columns, fields[i])
			}
		}
	}
	if ps := inq.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if limit := inq.ctx.Limit; limit != nil {
		_spec.Limit = *limit
	}
	if offset := inq.ctx.Offset; offset != nil {
		_spec.Offset = *offset
	}
	if ps := inq.order; len(ps) > 0 {
		_spec.Order = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	return _spec
}

func (inq *IncludedNetworkQuery) sqlQuery(ctx context.Context) *sql.Selector {
	builder := sql.Dialect(inq.driver.Dialect())
	t1 := builder.Table(includednetwork.Table)
	columns := inq.ctx.Fields
	if len(columns) == 0 {
		columns = includednetwork.Columns
	}
	selector := builder.Select(t1.Columns(columns...)...).From(t1)
	if inq.sql != nil {
		selector = inq.sql
		selector.Select(selector.Columns(columns...)...)
	}
	if inq.ctx.Unique != nil && *inq.ctx.Unique {
		selector.Distinct()
	}
	for _, p := range inq.predicates {
		p(selector)
	}
	for _, p := range inq.order {
		p(selector)
	}
	if offset := inq.ctx.Offset; offset != nil {
		// limit is mandatory for offset clause. We start
		// with default value, and override it below if needed.
		selector.Offset(*offset).Limit(math.MaxInt32)
	}
	if limit := inq.ctx.Limit; limit != nil {
		selector.Limit(*limit)
	}
	return selector
}

// WithNamedTags tells the query-builder to eager-load the nodes that are connected to the "Tags"
// edge with the given name. The optional arguments are used to configure the query builder of the edge.
func (inq *IncludedNetworkQuery) WithNamedTags(name string, opts ...func(*TagQuery)) *IncludedNetworkQuery {
	query := (&TagClient{config: inq.config}).Query()
	for _, opt := range opts {
		opt(query)
	}
	if inq.withNamedTags == nil {
		inq.withNamedTags = make(map[string]*TagQuery)
	}
	inq.withNamedTags[name] = query
	return inq
}

// WithNamedHosts tells the query-builder to eager-load the nodes that are connected to the "Hosts"
// edge with the given name. The optional arguments are used to configure the query builder of the edge.
func (inq *IncludedNetworkQuery) WithNamedHosts(name string, opts ...func(*HostQuery)) *IncludedNetworkQuery {
	query := (&HostClient{config: inq.config}).Query()
	for _, opt := range opts {
		opt(query)
	}
	if inq.withNamedHosts == nil {
		inq.withNamedHosts = make(map[string]*HostQuery)
	}
	inq.withNamedHosts[name] = query
	return inq
}

// WithNamedEnvironments tells the query-builder to eager-load the nodes that are connected to the "Environments"
// edge with the given name. The optional arguments are used to configure the query builder of the edge.
func (inq *IncludedNetworkQuery) WithNamedEnvironments(name string, opts ...func(*EnvironmentQuery)) *IncludedNetworkQuery {
	query := (&EnvironmentClient{config: inq.config}).Query()
	for _, opt := range opts {
		opt(query)
	}
	if inq.withNamedEnvironments == nil {
		inq.withNamedEnvironments = make(map[string]*EnvironmentQuery)
	}
	inq.withNamedEnvironments[name] = query
	return inq
}

// IncludedNetworkGroupBy is the group-by builder for IncludedNetwork entities.
type IncludedNetworkGroupBy struct {
	selector
	build *IncludedNetworkQuery
}

// Aggregate adds the given aggregation functions to the group-by query.
func (ingb *IncludedNetworkGroupBy) Aggregate(fns ...AggregateFunc) *IncludedNetworkGroupBy {
	ingb.fns = append(ingb.fns, fns...)
	return ingb
}

// Scan applies the selector query and scans the result into the given value.
func (ingb *IncludedNetworkGroupBy) Scan(ctx context.Context, v any) error {
	ctx = setContextOp(ctx, ingb.build.ctx, "GroupBy")
	if err := ingb.build.prepareQuery(ctx); err != nil {
		return err
	}
	return scanWithInterceptors[*IncludedNetworkQuery, *IncludedNetworkGroupBy](ctx, ingb.build, ingb, ingb.build.inters, v)
}

func (ingb *IncludedNetworkGroupBy) sqlScan(ctx context.Context, root *IncludedNetworkQuery, v any) error {
	selector := root.sqlQuery(ctx).Select()
	aggregation := make([]string, 0, len(ingb.fns))
	for _, fn := range ingb.fns {
		aggregation = append(aggregation, fn(selector))
	}
	if len(selector.SelectedColumns()) == 0 {
		columns := make([]string, 0, len(*ingb.flds)+len(ingb.fns))
		for _, f := range *ingb.flds {
			columns = append(columns, selector.C(f))
		}
		columns = append(columns, aggregation...)
		selector.Select(columns...)
	}
	selector.GroupBy(selector.Columns(*ingb.flds...)...)
	if err := selector.Err(); err != nil {
		return err
	}
	rows := &sql.Rows{}
	query, args := selector.Query()
	if err := ingb.build.driver.Query(ctx, query, args, rows); err != nil {
		return err
	}
	defer rows.Close()
	return sql.ScanSlice(rows, v)
}

// IncludedNetworkSelect is the builder for selecting fields of IncludedNetwork entities.
type IncludedNetworkSelect struct {
	*IncludedNetworkQuery
	selector
}

// Aggregate adds the given aggregation functions to the selector query.
func (ins *IncludedNetworkSelect) Aggregate(fns ...AggregateFunc) *IncludedNetworkSelect {
	ins.fns = append(ins.fns, fns...)
	return ins
}

// Scan applies the selector query and scans the result into the given value.
func (ins *IncludedNetworkSelect) Scan(ctx context.Context, v any) error {
	ctx = setContextOp(ctx, ins.ctx, "Select")
	if err := ins.prepareQuery(ctx); err != nil {
		return err
	}
	return scanWithInterceptors[*IncludedNetworkQuery, *IncludedNetworkSelect](ctx, ins.IncludedNetworkQuery, ins, ins.inters, v)
}

func (ins *IncludedNetworkSelect) sqlScan(ctx context.Context, root *IncludedNetworkQuery, v any) error {
	selector := root.sqlQuery(ctx)
	aggregation := make([]string, 0, len(ins.fns))
	for _, fn := range ins.fns {
		aggregation = append(aggregation, fn(selector))
	}
	switch n := len(*ins.selector.flds); {
	case n == 0 && len(aggregation) > 0:
		selector.Select(aggregation...)
	case n != 0 && len(aggregation) > 0:
		selector.AppendSelect(aggregation...)
	}
	rows := &sql.Rows{}
	query, args := selector.Query()
	if err := ins.driver.Query(ctx, query, args, rows); err != nil {
		return err
	}
	defer rows.Close()
	return sql.ScanSlice(rows, v)
}
