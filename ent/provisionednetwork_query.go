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
	"github.com/gen0cide/laforge/ent/build"
	"github.com/gen0cide/laforge/ent/network"
	"github.com/gen0cide/laforge/ent/plan"
	"github.com/gen0cide/laforge/ent/predicate"
	"github.com/gen0cide/laforge/ent/provisionedhost"
	"github.com/gen0cide/laforge/ent/provisionednetwork"
	"github.com/gen0cide/laforge/ent/status"
	"github.com/gen0cide/laforge/ent/team"
	"github.com/google/uuid"
)

// ProvisionedNetworkQuery is the builder for querying ProvisionedNetwork entities.
type ProvisionedNetworkQuery struct {
	config
	limit                *int
	offset               *int
	unique               *bool
	order                []OrderFunc
	fields               []string
	predicates           []predicate.ProvisionedNetwork
	withStatus           *StatusQuery
	withNetwork          *NetworkQuery
	withBuild            *BuildQuery
	withTeam             *TeamQuery
	withProvisionedHosts *ProvisionedHostQuery
	withPlan             *PlanQuery
	withFKs              bool
	// intermediate query (i.e. traversal path).
	sql  *sql.Selector
	path func(context.Context) (*sql.Selector, error)
}

// Where adds a new predicate for the ProvisionedNetworkQuery builder.
func (pnq *ProvisionedNetworkQuery) Where(ps ...predicate.ProvisionedNetwork) *ProvisionedNetworkQuery {
	pnq.predicates = append(pnq.predicates, ps...)
	return pnq
}

// Limit adds a limit step to the query.
func (pnq *ProvisionedNetworkQuery) Limit(limit int) *ProvisionedNetworkQuery {
	pnq.limit = &limit
	return pnq
}

// Offset adds an offset step to the query.
func (pnq *ProvisionedNetworkQuery) Offset(offset int) *ProvisionedNetworkQuery {
	pnq.offset = &offset
	return pnq
}

// Unique configures the query builder to filter duplicate records on query.
// By default, unique is set to true, and can be disabled using this method.
func (pnq *ProvisionedNetworkQuery) Unique(unique bool) *ProvisionedNetworkQuery {
	pnq.unique = &unique
	return pnq
}

// Order adds an order step to the query.
func (pnq *ProvisionedNetworkQuery) Order(o ...OrderFunc) *ProvisionedNetworkQuery {
	pnq.order = append(pnq.order, o...)
	return pnq
}

// QueryStatus chains the current query on the "Status" edge.
func (pnq *ProvisionedNetworkQuery) QueryStatus() *StatusQuery {
	query := &StatusQuery{config: pnq.config}
	query.path = func(ctx context.Context) (fromU *sql.Selector, err error) {
		if err := pnq.prepareQuery(ctx); err != nil {
			return nil, err
		}
		selector := pnq.sqlQuery(ctx)
		if err := selector.Err(); err != nil {
			return nil, err
		}
		step := sqlgraph.NewStep(
			sqlgraph.From(provisionednetwork.Table, provisionednetwork.FieldID, selector),
			sqlgraph.To(status.Table, status.FieldID),
			sqlgraph.Edge(sqlgraph.O2O, false, provisionednetwork.StatusTable, provisionednetwork.StatusColumn),
		)
		fromU = sqlgraph.SetNeighbors(pnq.driver.Dialect(), step)
		return fromU, nil
	}
	return query
}

// QueryNetwork chains the current query on the "Network" edge.
func (pnq *ProvisionedNetworkQuery) QueryNetwork() *NetworkQuery {
	query := &NetworkQuery{config: pnq.config}
	query.path = func(ctx context.Context) (fromU *sql.Selector, err error) {
		if err := pnq.prepareQuery(ctx); err != nil {
			return nil, err
		}
		selector := pnq.sqlQuery(ctx)
		if err := selector.Err(); err != nil {
			return nil, err
		}
		step := sqlgraph.NewStep(
			sqlgraph.From(provisionednetwork.Table, provisionednetwork.FieldID, selector),
			sqlgraph.To(network.Table, network.FieldID),
			sqlgraph.Edge(sqlgraph.M2O, false, provisionednetwork.NetworkTable, provisionednetwork.NetworkColumn),
		)
		fromU = sqlgraph.SetNeighbors(pnq.driver.Dialect(), step)
		return fromU, nil
	}
	return query
}

// QueryBuild chains the current query on the "Build" edge.
func (pnq *ProvisionedNetworkQuery) QueryBuild() *BuildQuery {
	query := &BuildQuery{config: pnq.config}
	query.path = func(ctx context.Context) (fromU *sql.Selector, err error) {
		if err := pnq.prepareQuery(ctx); err != nil {
			return nil, err
		}
		selector := pnq.sqlQuery(ctx)
		if err := selector.Err(); err != nil {
			return nil, err
		}
		step := sqlgraph.NewStep(
			sqlgraph.From(provisionednetwork.Table, provisionednetwork.FieldID, selector),
			sqlgraph.To(build.Table, build.FieldID),
			sqlgraph.Edge(sqlgraph.M2O, false, provisionednetwork.BuildTable, provisionednetwork.BuildColumn),
		)
		fromU = sqlgraph.SetNeighbors(pnq.driver.Dialect(), step)
		return fromU, nil
	}
	return query
}

// QueryTeam chains the current query on the "Team" edge.
func (pnq *ProvisionedNetworkQuery) QueryTeam() *TeamQuery {
	query := &TeamQuery{config: pnq.config}
	query.path = func(ctx context.Context) (fromU *sql.Selector, err error) {
		if err := pnq.prepareQuery(ctx); err != nil {
			return nil, err
		}
		selector := pnq.sqlQuery(ctx)
		if err := selector.Err(); err != nil {
			return nil, err
		}
		step := sqlgraph.NewStep(
			sqlgraph.From(provisionednetwork.Table, provisionednetwork.FieldID, selector),
			sqlgraph.To(team.Table, team.FieldID),
			sqlgraph.Edge(sqlgraph.M2O, false, provisionednetwork.TeamTable, provisionednetwork.TeamColumn),
		)
		fromU = sqlgraph.SetNeighbors(pnq.driver.Dialect(), step)
		return fromU, nil
	}
	return query
}

// QueryProvisionedHosts chains the current query on the "ProvisionedHosts" edge.
func (pnq *ProvisionedNetworkQuery) QueryProvisionedHosts() *ProvisionedHostQuery {
	query := &ProvisionedHostQuery{config: pnq.config}
	query.path = func(ctx context.Context) (fromU *sql.Selector, err error) {
		if err := pnq.prepareQuery(ctx); err != nil {
			return nil, err
		}
		selector := pnq.sqlQuery(ctx)
		if err := selector.Err(); err != nil {
			return nil, err
		}
		step := sqlgraph.NewStep(
			sqlgraph.From(provisionednetwork.Table, provisionednetwork.FieldID, selector),
			sqlgraph.To(provisionedhost.Table, provisionedhost.FieldID),
			sqlgraph.Edge(sqlgraph.O2M, true, provisionednetwork.ProvisionedHostsTable, provisionednetwork.ProvisionedHostsColumn),
		)
		fromU = sqlgraph.SetNeighbors(pnq.driver.Dialect(), step)
		return fromU, nil
	}
	return query
}

// QueryPlan chains the current query on the "Plan" edge.
func (pnq *ProvisionedNetworkQuery) QueryPlan() *PlanQuery {
	query := &PlanQuery{config: pnq.config}
	query.path = func(ctx context.Context) (fromU *sql.Selector, err error) {
		if err := pnq.prepareQuery(ctx); err != nil {
			return nil, err
		}
		selector := pnq.sqlQuery(ctx)
		if err := selector.Err(); err != nil {
			return nil, err
		}
		step := sqlgraph.NewStep(
			sqlgraph.From(provisionednetwork.Table, provisionednetwork.FieldID, selector),
			sqlgraph.To(plan.Table, plan.FieldID),
			sqlgraph.Edge(sqlgraph.O2O, true, provisionednetwork.PlanTable, provisionednetwork.PlanColumn),
		)
		fromU = sqlgraph.SetNeighbors(pnq.driver.Dialect(), step)
		return fromU, nil
	}
	return query
}

// First returns the first ProvisionedNetwork entity from the query.
// Returns a *NotFoundError when no ProvisionedNetwork was found.
func (pnq *ProvisionedNetworkQuery) First(ctx context.Context) (*ProvisionedNetwork, error) {
	nodes, err := pnq.Limit(1).All(ctx)
	if err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nil, &NotFoundError{provisionednetwork.Label}
	}
	return nodes[0], nil
}

// FirstX is like First, but panics if an error occurs.
func (pnq *ProvisionedNetworkQuery) FirstX(ctx context.Context) *ProvisionedNetwork {
	node, err := pnq.First(ctx)
	if err != nil && !IsNotFound(err) {
		panic(err)
	}
	return node
}

// FirstID returns the first ProvisionedNetwork ID from the query.
// Returns a *NotFoundError when no ProvisionedNetwork ID was found.
func (pnq *ProvisionedNetworkQuery) FirstID(ctx context.Context) (id uuid.UUID, err error) {
	var ids []uuid.UUID
	if ids, err = pnq.Limit(1).IDs(ctx); err != nil {
		return
	}
	if len(ids) == 0 {
		err = &NotFoundError{provisionednetwork.Label}
		return
	}
	return ids[0], nil
}

// FirstIDX is like FirstID, but panics if an error occurs.
func (pnq *ProvisionedNetworkQuery) FirstIDX(ctx context.Context) uuid.UUID {
	id, err := pnq.FirstID(ctx)
	if err != nil && !IsNotFound(err) {
		panic(err)
	}
	return id
}

// Only returns a single ProvisionedNetwork entity found by the query, ensuring it only returns one.
// Returns a *NotSingularError when more than one ProvisionedNetwork entity is found.
// Returns a *NotFoundError when no ProvisionedNetwork entities are found.
func (pnq *ProvisionedNetworkQuery) Only(ctx context.Context) (*ProvisionedNetwork, error) {
	nodes, err := pnq.Limit(2).All(ctx)
	if err != nil {
		return nil, err
	}
	switch len(nodes) {
	case 1:
		return nodes[0], nil
	case 0:
		return nil, &NotFoundError{provisionednetwork.Label}
	default:
		return nil, &NotSingularError{provisionednetwork.Label}
	}
}

// OnlyX is like Only, but panics if an error occurs.
func (pnq *ProvisionedNetworkQuery) OnlyX(ctx context.Context) *ProvisionedNetwork {
	node, err := pnq.Only(ctx)
	if err != nil {
		panic(err)
	}
	return node
}

// OnlyID is like Only, but returns the only ProvisionedNetwork ID in the query.
// Returns a *NotSingularError when more than one ProvisionedNetwork ID is found.
// Returns a *NotFoundError when no entities are found.
func (pnq *ProvisionedNetworkQuery) OnlyID(ctx context.Context) (id uuid.UUID, err error) {
	var ids []uuid.UUID
	if ids, err = pnq.Limit(2).IDs(ctx); err != nil {
		return
	}
	switch len(ids) {
	case 1:
		id = ids[0]
	case 0:
		err = &NotFoundError{provisionednetwork.Label}
	default:
		err = &NotSingularError{provisionednetwork.Label}
	}
	return
}

// OnlyIDX is like OnlyID, but panics if an error occurs.
func (pnq *ProvisionedNetworkQuery) OnlyIDX(ctx context.Context) uuid.UUID {
	id, err := pnq.OnlyID(ctx)
	if err != nil {
		panic(err)
	}
	return id
}

// All executes the query and returns a list of ProvisionedNetworks.
func (pnq *ProvisionedNetworkQuery) All(ctx context.Context) ([]*ProvisionedNetwork, error) {
	if err := pnq.prepareQuery(ctx); err != nil {
		return nil, err
	}
	return pnq.sqlAll(ctx)
}

// AllX is like All, but panics if an error occurs.
func (pnq *ProvisionedNetworkQuery) AllX(ctx context.Context) []*ProvisionedNetwork {
	nodes, err := pnq.All(ctx)
	if err != nil {
		panic(err)
	}
	return nodes
}

// IDs executes the query and returns a list of ProvisionedNetwork IDs.
func (pnq *ProvisionedNetworkQuery) IDs(ctx context.Context) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	if err := pnq.Select(provisionednetwork.FieldID).Scan(ctx, &ids); err != nil {
		return nil, err
	}
	return ids, nil
}

// IDsX is like IDs, but panics if an error occurs.
func (pnq *ProvisionedNetworkQuery) IDsX(ctx context.Context) []uuid.UUID {
	ids, err := pnq.IDs(ctx)
	if err != nil {
		panic(err)
	}
	return ids
}

// Count returns the count of the given query.
func (pnq *ProvisionedNetworkQuery) Count(ctx context.Context) (int, error) {
	if err := pnq.prepareQuery(ctx); err != nil {
		return 0, err
	}
	return pnq.sqlCount(ctx)
}

// CountX is like Count, but panics if an error occurs.
func (pnq *ProvisionedNetworkQuery) CountX(ctx context.Context) int {
	count, err := pnq.Count(ctx)
	if err != nil {
		panic(err)
	}
	return count
}

// Exist returns true if the query has elements in the graph.
func (pnq *ProvisionedNetworkQuery) Exist(ctx context.Context) (bool, error) {
	if err := pnq.prepareQuery(ctx); err != nil {
		return false, err
	}
	return pnq.sqlExist(ctx)
}

// ExistX is like Exist, but panics if an error occurs.
func (pnq *ProvisionedNetworkQuery) ExistX(ctx context.Context) bool {
	exist, err := pnq.Exist(ctx)
	if err != nil {
		panic(err)
	}
	return exist
}

// Clone returns a duplicate of the ProvisionedNetworkQuery builder, including all associated steps. It can be
// used to prepare common query builders and use them differently after the clone is made.
func (pnq *ProvisionedNetworkQuery) Clone() *ProvisionedNetworkQuery {
	if pnq == nil {
		return nil
	}
	return &ProvisionedNetworkQuery{
		config:               pnq.config,
		limit:                pnq.limit,
		offset:               pnq.offset,
		order:                append([]OrderFunc{}, pnq.order...),
		predicates:           append([]predicate.ProvisionedNetwork{}, pnq.predicates...),
		withStatus:           pnq.withStatus.Clone(),
		withNetwork:          pnq.withNetwork.Clone(),
		withBuild:            pnq.withBuild.Clone(),
		withTeam:             pnq.withTeam.Clone(),
		withProvisionedHosts: pnq.withProvisionedHosts.Clone(),
		withPlan:             pnq.withPlan.Clone(),
		// clone intermediate query.
		sql:    pnq.sql.Clone(),
		path:   pnq.path,
		unique: pnq.unique,
	}
}

// WithStatus tells the query-builder to eager-load the nodes that are connected to
// the "Status" edge. The optional arguments are used to configure the query builder of the edge.
func (pnq *ProvisionedNetworkQuery) WithStatus(opts ...func(*StatusQuery)) *ProvisionedNetworkQuery {
	query := &StatusQuery{config: pnq.config}
	for _, opt := range opts {
		opt(query)
	}
	pnq.withStatus = query
	return pnq
}

// WithNetwork tells the query-builder to eager-load the nodes that are connected to
// the "Network" edge. The optional arguments are used to configure the query builder of the edge.
func (pnq *ProvisionedNetworkQuery) WithNetwork(opts ...func(*NetworkQuery)) *ProvisionedNetworkQuery {
	query := &NetworkQuery{config: pnq.config}
	for _, opt := range opts {
		opt(query)
	}
	pnq.withNetwork = query
	return pnq
}

// WithBuild tells the query-builder to eager-load the nodes that are connected to
// the "Build" edge. The optional arguments are used to configure the query builder of the edge.
func (pnq *ProvisionedNetworkQuery) WithBuild(opts ...func(*BuildQuery)) *ProvisionedNetworkQuery {
	query := &BuildQuery{config: pnq.config}
	for _, opt := range opts {
		opt(query)
	}
	pnq.withBuild = query
	return pnq
}

// WithTeam tells the query-builder to eager-load the nodes that are connected to
// the "Team" edge. The optional arguments are used to configure the query builder of the edge.
func (pnq *ProvisionedNetworkQuery) WithTeam(opts ...func(*TeamQuery)) *ProvisionedNetworkQuery {
	query := &TeamQuery{config: pnq.config}
	for _, opt := range opts {
		opt(query)
	}
	pnq.withTeam = query
	return pnq
}

// WithProvisionedHosts tells the query-builder to eager-load the nodes that are connected to
// the "ProvisionedHosts" edge. The optional arguments are used to configure the query builder of the edge.
func (pnq *ProvisionedNetworkQuery) WithProvisionedHosts(opts ...func(*ProvisionedHostQuery)) *ProvisionedNetworkQuery {
	query := &ProvisionedHostQuery{config: pnq.config}
	for _, opt := range opts {
		opt(query)
	}
	pnq.withProvisionedHosts = query
	return pnq
}

// WithPlan tells the query-builder to eager-load the nodes that are connected to
// the "Plan" edge. The optional arguments are used to configure the query builder of the edge.
func (pnq *ProvisionedNetworkQuery) WithPlan(opts ...func(*PlanQuery)) *ProvisionedNetworkQuery {
	query := &PlanQuery{config: pnq.config}
	for _, opt := range opts {
		opt(query)
	}
	pnq.withPlan = query
	return pnq
}

// GroupBy is used to group vertices by one or more fields/columns.
// It is often used with aggregate functions, like: count, max, mean, min, sum.
//
// Example:
//
//	var v []struct {
//		Name string `json:"name,omitempty"`
//		Count int `json:"count,omitempty"`
//	}
//
//	client.ProvisionedNetwork.Query().
//		GroupBy(provisionednetwork.FieldName).
//		Aggregate(ent.Count()).
//		Scan(ctx, &v)
func (pnq *ProvisionedNetworkQuery) GroupBy(field string, fields ...string) *ProvisionedNetworkGroupBy {
	grbuild := &ProvisionedNetworkGroupBy{config: pnq.config}
	grbuild.fields = append([]string{field}, fields...)
	grbuild.path = func(ctx context.Context) (prev *sql.Selector, err error) {
		if err := pnq.prepareQuery(ctx); err != nil {
			return nil, err
		}
		return pnq.sqlQuery(ctx), nil
	}
	grbuild.label = provisionednetwork.Label
	grbuild.flds, grbuild.scan = &grbuild.fields, grbuild.Scan
	return grbuild
}

// Select allows the selection one or more fields/columns for the given query,
// instead of selecting all fields in the entity.
//
// Example:
//
//	var v []struct {
//		Name string `json:"name,omitempty"`
//	}
//
//	client.ProvisionedNetwork.Query().
//		Select(provisionednetwork.FieldName).
//		Scan(ctx, &v)
func (pnq *ProvisionedNetworkQuery) Select(fields ...string) *ProvisionedNetworkSelect {
	pnq.fields = append(pnq.fields, fields...)
	selbuild := &ProvisionedNetworkSelect{ProvisionedNetworkQuery: pnq}
	selbuild.label = provisionednetwork.Label
	selbuild.flds, selbuild.scan = &pnq.fields, selbuild.Scan
	return selbuild
}

func (pnq *ProvisionedNetworkQuery) prepareQuery(ctx context.Context) error {
	for _, f := range pnq.fields {
		if !provisionednetwork.ValidColumn(f) {
			return &ValidationError{Name: f, err: fmt.Errorf("ent: invalid field %q for query", f)}
		}
	}
	if pnq.path != nil {
		prev, err := pnq.path(ctx)
		if err != nil {
			return err
		}
		pnq.sql = prev
	}
	return nil
}

func (pnq *ProvisionedNetworkQuery) sqlAll(ctx context.Context, hooks ...queryHook) ([]*ProvisionedNetwork, error) {
	var (
		nodes       = []*ProvisionedNetwork{}
		withFKs     = pnq.withFKs
		_spec       = pnq.querySpec()
		loadedTypes = [6]bool{
			pnq.withStatus != nil,
			pnq.withNetwork != nil,
			pnq.withBuild != nil,
			pnq.withTeam != nil,
			pnq.withProvisionedHosts != nil,
			pnq.withPlan != nil,
		}
	)
	if pnq.withNetwork != nil || pnq.withBuild != nil || pnq.withTeam != nil || pnq.withPlan != nil {
		withFKs = true
	}
	if withFKs {
		_spec.Node.Columns = append(_spec.Node.Columns, provisionednetwork.ForeignKeys...)
	}
	_spec.ScanValues = func(columns []string) ([]interface{}, error) {
		return (*ProvisionedNetwork).scanValues(nil, columns)
	}
	_spec.Assign = func(columns []string, values []interface{}) error {
		node := &ProvisionedNetwork{config: pnq.config}
		nodes = append(nodes, node)
		node.Edges.loadedTypes = loadedTypes
		return node.assignValues(columns, values)
	}
	for i := range hooks {
		hooks[i](ctx, _spec)
	}
	if err := sqlgraph.QueryNodes(ctx, pnq.driver, _spec); err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nodes, nil
	}
	if query := pnq.withStatus; query != nil {
		if err := pnq.loadStatus(ctx, query, nodes, nil,
			func(n *ProvisionedNetwork, e *Status) { n.Edges.Status = e }); err != nil {
			return nil, err
		}
	}
	if query := pnq.withNetwork; query != nil {
		if err := pnq.loadNetwork(ctx, query, nodes, nil,
			func(n *ProvisionedNetwork, e *Network) { n.Edges.Network = e }); err != nil {
			return nil, err
		}
	}
	if query := pnq.withBuild; query != nil {
		if err := pnq.loadBuild(ctx, query, nodes, nil,
			func(n *ProvisionedNetwork, e *Build) { n.Edges.Build = e }); err != nil {
			return nil, err
		}
	}
	if query := pnq.withTeam; query != nil {
		if err := pnq.loadTeam(ctx, query, nodes, nil,
			func(n *ProvisionedNetwork, e *Team) { n.Edges.Team = e }); err != nil {
			return nil, err
		}
	}
	if query := pnq.withProvisionedHosts; query != nil {
		if err := pnq.loadProvisionedHosts(ctx, query, nodes,
			func(n *ProvisionedNetwork) { n.Edges.ProvisionedHosts = []*ProvisionedHost{} },
			func(n *ProvisionedNetwork, e *ProvisionedHost) {
				n.Edges.ProvisionedHosts = append(n.Edges.ProvisionedHosts, e)
			}); err != nil {
			return nil, err
		}
	}
	if query := pnq.withPlan; query != nil {
		if err := pnq.loadPlan(ctx, query, nodes, nil,
			func(n *ProvisionedNetwork, e *Plan) { n.Edges.Plan = e }); err != nil {
			return nil, err
		}
	}
	return nodes, nil
}

func (pnq *ProvisionedNetworkQuery) loadStatus(ctx context.Context, query *StatusQuery, nodes []*ProvisionedNetwork, init func(*ProvisionedNetwork), assign func(*ProvisionedNetwork, *Status)) error {
	fks := make([]driver.Value, 0, len(nodes))
	nodeids := make(map[uuid.UUID]*ProvisionedNetwork)
	for i := range nodes {
		fks = append(fks, nodes[i].ID)
		nodeids[nodes[i].ID] = nodes[i]
	}
	query.withFKs = true
	query.Where(predicate.Status(func(s *sql.Selector) {
		s.Where(sql.InValues(provisionednetwork.StatusColumn, fks...))
	}))
	neighbors, err := query.All(ctx)
	if err != nil {
		return err
	}
	for _, n := range neighbors {
		fk := n.provisioned_network_status
		if fk == nil {
			return fmt.Errorf(`foreign-key "provisioned_network_status" is nil for node %v`, n.ID)
		}
		node, ok := nodeids[*fk]
		if !ok {
			return fmt.Errorf(`unexpected foreign-key "provisioned_network_status" returned %v for node %v`, *fk, n.ID)
		}
		assign(node, n)
	}
	return nil
}
func (pnq *ProvisionedNetworkQuery) loadNetwork(ctx context.Context, query *NetworkQuery, nodes []*ProvisionedNetwork, init func(*ProvisionedNetwork), assign func(*ProvisionedNetwork, *Network)) error {
	ids := make([]uuid.UUID, 0, len(nodes))
	nodeids := make(map[uuid.UUID][]*ProvisionedNetwork)
	for i := range nodes {
		if nodes[i].provisioned_network_network == nil {
			continue
		}
		fk := *nodes[i].provisioned_network_network
		if _, ok := nodeids[fk]; !ok {
			ids = append(ids, fk)
		}
		nodeids[fk] = append(nodeids[fk], nodes[i])
	}
	query.Where(network.IDIn(ids...))
	neighbors, err := query.All(ctx)
	if err != nil {
		return err
	}
	for _, n := range neighbors {
		nodes, ok := nodeids[n.ID]
		if !ok {
			return fmt.Errorf(`unexpected foreign-key "provisioned_network_network" returned %v`, n.ID)
		}
		for i := range nodes {
			assign(nodes[i], n)
		}
	}
	return nil
}
func (pnq *ProvisionedNetworkQuery) loadBuild(ctx context.Context, query *BuildQuery, nodes []*ProvisionedNetwork, init func(*ProvisionedNetwork), assign func(*ProvisionedNetwork, *Build)) error {
	ids := make([]uuid.UUID, 0, len(nodes))
	nodeids := make(map[uuid.UUID][]*ProvisionedNetwork)
	for i := range nodes {
		if nodes[i].provisioned_network_build == nil {
			continue
		}
		fk := *nodes[i].provisioned_network_build
		if _, ok := nodeids[fk]; !ok {
			ids = append(ids, fk)
		}
		nodeids[fk] = append(nodeids[fk], nodes[i])
	}
	query.Where(build.IDIn(ids...))
	neighbors, err := query.All(ctx)
	if err != nil {
		return err
	}
	for _, n := range neighbors {
		nodes, ok := nodeids[n.ID]
		if !ok {
			return fmt.Errorf(`unexpected foreign-key "provisioned_network_build" returned %v`, n.ID)
		}
		for i := range nodes {
			assign(nodes[i], n)
		}
	}
	return nil
}
func (pnq *ProvisionedNetworkQuery) loadTeam(ctx context.Context, query *TeamQuery, nodes []*ProvisionedNetwork, init func(*ProvisionedNetwork), assign func(*ProvisionedNetwork, *Team)) error {
	ids := make([]uuid.UUID, 0, len(nodes))
	nodeids := make(map[uuid.UUID][]*ProvisionedNetwork)
	for i := range nodes {
		if nodes[i].provisioned_network_team == nil {
			continue
		}
		fk := *nodes[i].provisioned_network_team
		if _, ok := nodeids[fk]; !ok {
			ids = append(ids, fk)
		}
		nodeids[fk] = append(nodeids[fk], nodes[i])
	}
	query.Where(team.IDIn(ids...))
	neighbors, err := query.All(ctx)
	if err != nil {
		return err
	}
	for _, n := range neighbors {
		nodes, ok := nodeids[n.ID]
		if !ok {
			return fmt.Errorf(`unexpected foreign-key "provisioned_network_team" returned %v`, n.ID)
		}
		for i := range nodes {
			assign(nodes[i], n)
		}
	}
	return nil
}
func (pnq *ProvisionedNetworkQuery) loadProvisionedHosts(ctx context.Context, query *ProvisionedHostQuery, nodes []*ProvisionedNetwork, init func(*ProvisionedNetwork), assign func(*ProvisionedNetwork, *ProvisionedHost)) error {
	fks := make([]driver.Value, 0, len(nodes))
	nodeids := make(map[uuid.UUID]*ProvisionedNetwork)
	for i := range nodes {
		fks = append(fks, nodes[i].ID)
		nodeids[nodes[i].ID] = nodes[i]
		if init != nil {
			init(nodes[i])
		}
	}
	query.withFKs = true
	query.Where(predicate.ProvisionedHost(func(s *sql.Selector) {
		s.Where(sql.InValues(provisionednetwork.ProvisionedHostsColumn, fks...))
	}))
	neighbors, err := query.All(ctx)
	if err != nil {
		return err
	}
	for _, n := range neighbors {
		fk := n.provisioned_host_provisioned_network
		if fk == nil {
			return fmt.Errorf(`foreign-key "provisioned_host_provisioned_network" is nil for node %v`, n.ID)
		}
		node, ok := nodeids[*fk]
		if !ok {
			return fmt.Errorf(`unexpected foreign-key "provisioned_host_provisioned_network" returned %v for node %v`, *fk, n.ID)
		}
		assign(node, n)
	}
	return nil
}
func (pnq *ProvisionedNetworkQuery) loadPlan(ctx context.Context, query *PlanQuery, nodes []*ProvisionedNetwork, init func(*ProvisionedNetwork), assign func(*ProvisionedNetwork, *Plan)) error {
	ids := make([]uuid.UUID, 0, len(nodes))
	nodeids := make(map[uuid.UUID][]*ProvisionedNetwork)
	for i := range nodes {
		if nodes[i].plan_provisioned_network == nil {
			continue
		}
		fk := *nodes[i].plan_provisioned_network
		if _, ok := nodeids[fk]; !ok {
			ids = append(ids, fk)
		}
		nodeids[fk] = append(nodeids[fk], nodes[i])
	}
	query.Where(plan.IDIn(ids...))
	neighbors, err := query.All(ctx)
	if err != nil {
		return err
	}
	for _, n := range neighbors {
		nodes, ok := nodeids[n.ID]
		if !ok {
			return fmt.Errorf(`unexpected foreign-key "plan_provisioned_network" returned %v`, n.ID)
		}
		for i := range nodes {
			assign(nodes[i], n)
		}
	}
	return nil
}

func (pnq *ProvisionedNetworkQuery) sqlCount(ctx context.Context) (int, error) {
	_spec := pnq.querySpec()
	_spec.Node.Columns = pnq.fields
	if len(pnq.fields) > 0 {
		_spec.Unique = pnq.unique != nil && *pnq.unique
	}
	return sqlgraph.CountNodes(ctx, pnq.driver, _spec)
}

func (pnq *ProvisionedNetworkQuery) sqlExist(ctx context.Context) (bool, error) {
	n, err := pnq.sqlCount(ctx)
	if err != nil {
		return false, fmt.Errorf("ent: check existence: %w", err)
	}
	return n > 0, nil
}

func (pnq *ProvisionedNetworkQuery) querySpec() *sqlgraph.QuerySpec {
	_spec := &sqlgraph.QuerySpec{
		Node: &sqlgraph.NodeSpec{
			Table:   provisionednetwork.Table,
			Columns: provisionednetwork.Columns,
			ID: &sqlgraph.FieldSpec{
				Type:   field.TypeUUID,
				Column: provisionednetwork.FieldID,
			},
		},
		From:   pnq.sql,
		Unique: true,
	}
	if unique := pnq.unique; unique != nil {
		_spec.Unique = *unique
	}
	if fields := pnq.fields; len(fields) > 0 {
		_spec.Node.Columns = make([]string, 0, len(fields))
		_spec.Node.Columns = append(_spec.Node.Columns, provisionednetwork.FieldID)
		for i := range fields {
			if fields[i] != provisionednetwork.FieldID {
				_spec.Node.Columns = append(_spec.Node.Columns, fields[i])
			}
		}
	}
	if ps := pnq.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if limit := pnq.limit; limit != nil {
		_spec.Limit = *limit
	}
	if offset := pnq.offset; offset != nil {
		_spec.Offset = *offset
	}
	if ps := pnq.order; len(ps) > 0 {
		_spec.Order = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	return _spec
}

func (pnq *ProvisionedNetworkQuery) sqlQuery(ctx context.Context) *sql.Selector {
	builder := sql.Dialect(pnq.driver.Dialect())
	t1 := builder.Table(provisionednetwork.Table)
	columns := pnq.fields
	if len(columns) == 0 {
		columns = provisionednetwork.Columns
	}
	selector := builder.Select(t1.Columns(columns...)...).From(t1)
	if pnq.sql != nil {
		selector = pnq.sql
		selector.Select(selector.Columns(columns...)...)
	}
	if pnq.unique != nil && *pnq.unique {
		selector.Distinct()
	}
	for _, p := range pnq.predicates {
		p(selector)
	}
	for _, p := range pnq.order {
		p(selector)
	}
	if offset := pnq.offset; offset != nil {
		// limit is mandatory for offset clause. We start
		// with default value, and override it below if needed.
		selector.Offset(*offset).Limit(math.MaxInt32)
	}
	if limit := pnq.limit; limit != nil {
		selector.Limit(*limit)
	}
	return selector
}

// ProvisionedNetworkGroupBy is the group-by builder for ProvisionedNetwork entities.
type ProvisionedNetworkGroupBy struct {
	config
	selector
	fields []string
	fns    []AggregateFunc
	// intermediate query (i.e. traversal path).
	sql  *sql.Selector
	path func(context.Context) (*sql.Selector, error)
}

// Aggregate adds the given aggregation functions to the group-by query.
func (pngb *ProvisionedNetworkGroupBy) Aggregate(fns ...AggregateFunc) *ProvisionedNetworkGroupBy {
	pngb.fns = append(pngb.fns, fns...)
	return pngb
}

// Scan applies the group-by query and scans the result into the given value.
func (pngb *ProvisionedNetworkGroupBy) Scan(ctx context.Context, v interface{}) error {
	query, err := pngb.path(ctx)
	if err != nil {
		return err
	}
	pngb.sql = query
	return pngb.sqlScan(ctx, v)
}

func (pngb *ProvisionedNetworkGroupBy) sqlScan(ctx context.Context, v interface{}) error {
	for _, f := range pngb.fields {
		if !provisionednetwork.ValidColumn(f) {
			return &ValidationError{Name: f, err: fmt.Errorf("invalid field %q for group-by", f)}
		}
	}
	selector := pngb.sqlQuery()
	if err := selector.Err(); err != nil {
		return err
	}
	rows := &sql.Rows{}
	query, args := selector.Query()
	if err := pngb.driver.Query(ctx, query, args, rows); err != nil {
		return err
	}
	defer rows.Close()
	return sql.ScanSlice(rows, v)
}

func (pngb *ProvisionedNetworkGroupBy) sqlQuery() *sql.Selector {
	selector := pngb.sql.Select()
	aggregation := make([]string, 0, len(pngb.fns))
	for _, fn := range pngb.fns {
		aggregation = append(aggregation, fn(selector))
	}
	// If no columns were selected in a custom aggregation function, the default
	// selection is the fields used for "group-by", and the aggregation functions.
	if len(selector.SelectedColumns()) == 0 {
		columns := make([]string, 0, len(pngb.fields)+len(pngb.fns))
		for _, f := range pngb.fields {
			columns = append(columns, selector.C(f))
		}
		columns = append(columns, aggregation...)
		selector.Select(columns...)
	}
	return selector.GroupBy(selector.Columns(pngb.fields...)...)
}

// ProvisionedNetworkSelect is the builder for selecting fields of ProvisionedNetwork entities.
type ProvisionedNetworkSelect struct {
	*ProvisionedNetworkQuery
	selector
	// intermediate query (i.e. traversal path).
	sql *sql.Selector
}

// Scan applies the selector query and scans the result into the given value.
func (pns *ProvisionedNetworkSelect) Scan(ctx context.Context, v interface{}) error {
	if err := pns.prepareQuery(ctx); err != nil {
		return err
	}
	pns.sql = pns.ProvisionedNetworkQuery.sqlQuery(ctx)
	return pns.sqlScan(ctx, v)
}

func (pns *ProvisionedNetworkSelect) sqlScan(ctx context.Context, v interface{}) error {
	rows := &sql.Rows{}
	query, args := pns.sql.Query()
	if err := pns.driver.Query(ctx, query, args, rows); err != nil {
		return err
	}
	defer rows.Close()
	return sql.ScanSlice(rows, v)
}
