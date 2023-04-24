// Code generated by ent, DO NOT EDIT.

package ent

import (
	"context"
	"fmt"
	"math"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/schema/field"
	"github.com/gen0cide/laforge/ent/adhocplan"
	"github.com/gen0cide/laforge/ent/build"
	"github.com/gen0cide/laforge/ent/plan"
	"github.com/gen0cide/laforge/ent/predicate"
	"github.com/gen0cide/laforge/ent/provisionedhost"
	"github.com/gen0cide/laforge/ent/provisionednetwork"
	"github.com/gen0cide/laforge/ent/provisioningscheduledstep"
	"github.com/gen0cide/laforge/ent/provisioningstep"
	"github.com/gen0cide/laforge/ent/servertask"
	"github.com/gen0cide/laforge/ent/status"
	"github.com/gen0cide/laforge/ent/team"
	"github.com/google/uuid"
)

// StatusQuery is the builder for querying Status entities.
type StatusQuery struct {
	config
	limit                                 *int
	offset                                *int
	unique                                *bool
	order                                 []OrderFunc
	fields                                []string
	predicates                            []predicate.Status
	withStatusToBuild                     *BuildQuery
	withStatusToProvisionedNetwork        *ProvisionedNetworkQuery
	withStatusToProvisionedHost           *ProvisionedHostQuery
	withStatusToProvisioningStep          *ProvisioningStepQuery
	withStatusToTeam                      *TeamQuery
	withStatusToPlan                      *PlanQuery
	withStatusToServerTask                *ServerTaskQuery
	withStatusToAdhocPlan                 *AdhocPlanQuery
	withStatusToProvisioningScheduledStep *ProvisioningScheduledStepQuery
	withFKs                               bool
	// intermediate query (i.e. traversal path).
	sql  *sql.Selector
	path func(context.Context) (*sql.Selector, error)
}

// Where adds a new predicate for the StatusQuery builder.
func (sq *StatusQuery) Where(ps ...predicate.Status) *StatusQuery {
	sq.predicates = append(sq.predicates, ps...)
	return sq
}

// Limit adds a limit step to the query.
func (sq *StatusQuery) Limit(limit int) *StatusQuery {
	sq.limit = &limit
	return sq
}

// Offset adds an offset step to the query.
func (sq *StatusQuery) Offset(offset int) *StatusQuery {
	sq.offset = &offset
	return sq
}

// Unique configures the query builder to filter duplicate records on query.
// By default, unique is set to true, and can be disabled using this method.
func (sq *StatusQuery) Unique(unique bool) *StatusQuery {
	sq.unique = &unique
	return sq
}

// Order adds an order step to the query.
func (sq *StatusQuery) Order(o ...OrderFunc) *StatusQuery {
	sq.order = append(sq.order, o...)
	return sq
}

// QueryStatusToBuild chains the current query on the "StatusToBuild" edge.
func (sq *StatusQuery) QueryStatusToBuild() *BuildQuery {
	query := &BuildQuery{config: sq.config}
	query.path = func(ctx context.Context) (fromU *sql.Selector, err error) {
		if err := sq.prepareQuery(ctx); err != nil {
			return nil, err
		}
		selector := sq.sqlQuery(ctx)
		if err := selector.Err(); err != nil {
			return nil, err
		}
		step := sqlgraph.NewStep(
			sqlgraph.From(status.Table, status.FieldID, selector),
			sqlgraph.To(build.Table, build.FieldID),
			sqlgraph.Edge(sqlgraph.O2O, true, status.StatusToBuildTable, status.StatusToBuildColumn),
		)
		fromU = sqlgraph.SetNeighbors(sq.driver.Dialect(), step)
		return fromU, nil
	}
	return query
}

// QueryStatusToProvisionedNetwork chains the current query on the "StatusToProvisionedNetwork" edge.
func (sq *StatusQuery) QueryStatusToProvisionedNetwork() *ProvisionedNetworkQuery {
	query := &ProvisionedNetworkQuery{config: sq.config}
	query.path = func(ctx context.Context) (fromU *sql.Selector, err error) {
		if err := sq.prepareQuery(ctx); err != nil {
			return nil, err
		}
		selector := sq.sqlQuery(ctx)
		if err := selector.Err(); err != nil {
			return nil, err
		}
		step := sqlgraph.NewStep(
			sqlgraph.From(status.Table, status.FieldID, selector),
			sqlgraph.To(provisionednetwork.Table, provisionednetwork.FieldID),
			sqlgraph.Edge(sqlgraph.O2O, true, status.StatusToProvisionedNetworkTable, status.StatusToProvisionedNetworkColumn),
		)
		fromU = sqlgraph.SetNeighbors(sq.driver.Dialect(), step)
		return fromU, nil
	}
	return query
}

// QueryStatusToProvisionedHost chains the current query on the "StatusToProvisionedHost" edge.
func (sq *StatusQuery) QueryStatusToProvisionedHost() *ProvisionedHostQuery {
	query := &ProvisionedHostQuery{config: sq.config}
	query.path = func(ctx context.Context) (fromU *sql.Selector, err error) {
		if err := sq.prepareQuery(ctx); err != nil {
			return nil, err
		}
		selector := sq.sqlQuery(ctx)
		if err := selector.Err(); err != nil {
			return nil, err
		}
		step := sqlgraph.NewStep(
			sqlgraph.From(status.Table, status.FieldID, selector),
			sqlgraph.To(provisionedhost.Table, provisionedhost.FieldID),
			sqlgraph.Edge(sqlgraph.O2O, true, status.StatusToProvisionedHostTable, status.StatusToProvisionedHostColumn),
		)
		fromU = sqlgraph.SetNeighbors(sq.driver.Dialect(), step)
		return fromU, nil
	}
	return query
}

// QueryStatusToProvisioningStep chains the current query on the "StatusToProvisioningStep" edge.
func (sq *StatusQuery) QueryStatusToProvisioningStep() *ProvisioningStepQuery {
	query := &ProvisioningStepQuery{config: sq.config}
	query.path = func(ctx context.Context) (fromU *sql.Selector, err error) {
		if err := sq.prepareQuery(ctx); err != nil {
			return nil, err
		}
		selector := sq.sqlQuery(ctx)
		if err := selector.Err(); err != nil {
			return nil, err
		}
		step := sqlgraph.NewStep(
			sqlgraph.From(status.Table, status.FieldID, selector),
			sqlgraph.To(provisioningstep.Table, provisioningstep.FieldID),
			sqlgraph.Edge(sqlgraph.O2O, true, status.StatusToProvisioningStepTable, status.StatusToProvisioningStepColumn),
		)
		fromU = sqlgraph.SetNeighbors(sq.driver.Dialect(), step)
		return fromU, nil
	}
	return query
}

// QueryStatusToTeam chains the current query on the "StatusToTeam" edge.
func (sq *StatusQuery) QueryStatusToTeam() *TeamQuery {
	query := &TeamQuery{config: sq.config}
	query.path = func(ctx context.Context) (fromU *sql.Selector, err error) {
		if err := sq.prepareQuery(ctx); err != nil {
			return nil, err
		}
		selector := sq.sqlQuery(ctx)
		if err := selector.Err(); err != nil {
			return nil, err
		}
		step := sqlgraph.NewStep(
			sqlgraph.From(status.Table, status.FieldID, selector),
			sqlgraph.To(team.Table, team.FieldID),
			sqlgraph.Edge(sqlgraph.O2O, true, status.StatusToTeamTable, status.StatusToTeamColumn),
		)
		fromU = sqlgraph.SetNeighbors(sq.driver.Dialect(), step)
		return fromU, nil
	}
	return query
}

// QueryStatusToPlan chains the current query on the "StatusToPlan" edge.
func (sq *StatusQuery) QueryStatusToPlan() *PlanQuery {
	query := &PlanQuery{config: sq.config}
	query.path = func(ctx context.Context) (fromU *sql.Selector, err error) {
		if err := sq.prepareQuery(ctx); err != nil {
			return nil, err
		}
		selector := sq.sqlQuery(ctx)
		if err := selector.Err(); err != nil {
			return nil, err
		}
		step := sqlgraph.NewStep(
			sqlgraph.From(status.Table, status.FieldID, selector),
			sqlgraph.To(plan.Table, plan.FieldID),
			sqlgraph.Edge(sqlgraph.O2O, true, status.StatusToPlanTable, status.StatusToPlanColumn),
		)
		fromU = sqlgraph.SetNeighbors(sq.driver.Dialect(), step)
		return fromU, nil
	}
	return query
}

// QueryStatusToServerTask chains the current query on the "StatusToServerTask" edge.
func (sq *StatusQuery) QueryStatusToServerTask() *ServerTaskQuery {
	query := &ServerTaskQuery{config: sq.config}
	query.path = func(ctx context.Context) (fromU *sql.Selector, err error) {
		if err := sq.prepareQuery(ctx); err != nil {
			return nil, err
		}
		selector := sq.sqlQuery(ctx)
		if err := selector.Err(); err != nil {
			return nil, err
		}
		step := sqlgraph.NewStep(
			sqlgraph.From(status.Table, status.FieldID, selector),
			sqlgraph.To(servertask.Table, servertask.FieldID),
			sqlgraph.Edge(sqlgraph.O2O, true, status.StatusToServerTaskTable, status.StatusToServerTaskColumn),
		)
		fromU = sqlgraph.SetNeighbors(sq.driver.Dialect(), step)
		return fromU, nil
	}
	return query
}

// QueryStatusToAdhocPlan chains the current query on the "StatusToAdhocPlan" edge.
func (sq *StatusQuery) QueryStatusToAdhocPlan() *AdhocPlanQuery {
	query := &AdhocPlanQuery{config: sq.config}
	query.path = func(ctx context.Context) (fromU *sql.Selector, err error) {
		if err := sq.prepareQuery(ctx); err != nil {
			return nil, err
		}
		selector := sq.sqlQuery(ctx)
		if err := selector.Err(); err != nil {
			return nil, err
		}
		step := sqlgraph.NewStep(
			sqlgraph.From(status.Table, status.FieldID, selector),
			sqlgraph.To(adhocplan.Table, adhocplan.FieldID),
			sqlgraph.Edge(sqlgraph.O2O, true, status.StatusToAdhocPlanTable, status.StatusToAdhocPlanColumn),
		)
		fromU = sqlgraph.SetNeighbors(sq.driver.Dialect(), step)
		return fromU, nil
	}
	return query
}

// QueryStatusToProvisioningScheduledStep chains the current query on the "StatusToProvisioningScheduledStep" edge.
func (sq *StatusQuery) QueryStatusToProvisioningScheduledStep() *ProvisioningScheduledStepQuery {
	query := &ProvisioningScheduledStepQuery{config: sq.config}
	query.path = func(ctx context.Context) (fromU *sql.Selector, err error) {
		if err := sq.prepareQuery(ctx); err != nil {
			return nil, err
		}
		selector := sq.sqlQuery(ctx)
		if err := selector.Err(); err != nil {
			return nil, err
		}
		step := sqlgraph.NewStep(
			sqlgraph.From(status.Table, status.FieldID, selector),
			sqlgraph.To(provisioningscheduledstep.Table, provisioningscheduledstep.FieldID),
			sqlgraph.Edge(sqlgraph.O2O, true, status.StatusToProvisioningScheduledStepTable, status.StatusToProvisioningScheduledStepColumn),
		)
		fromU = sqlgraph.SetNeighbors(sq.driver.Dialect(), step)
		return fromU, nil
	}
	return query
}

// First returns the first Status entity from the query.
// Returns a *NotFoundError when no Status was found.
func (sq *StatusQuery) First(ctx context.Context) (*Status, error) {
	nodes, err := sq.Limit(1).All(ctx)
	if err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nil, &NotFoundError{status.Label}
	}
	return nodes[0], nil
}

// FirstX is like First, but panics if an error occurs.
func (sq *StatusQuery) FirstX(ctx context.Context) *Status {
	node, err := sq.First(ctx)
	if err != nil && !IsNotFound(err) {
		panic(err)
	}
	return node
}

// FirstID returns the first Status ID from the query.
// Returns a *NotFoundError when no Status ID was found.
func (sq *StatusQuery) FirstID(ctx context.Context) (id uuid.UUID, err error) {
	var ids []uuid.UUID
	if ids, err = sq.Limit(1).IDs(ctx); err != nil {
		return
	}
	if len(ids) == 0 {
		err = &NotFoundError{status.Label}
		return
	}
	return ids[0], nil
}

// FirstIDX is like FirstID, but panics if an error occurs.
func (sq *StatusQuery) FirstIDX(ctx context.Context) uuid.UUID {
	id, err := sq.FirstID(ctx)
	if err != nil && !IsNotFound(err) {
		panic(err)
	}
	return id
}

// Only returns a single Status entity found by the query, ensuring it only returns one.
// Returns a *NotSingularError when more than one Status entity is found.
// Returns a *NotFoundError when no Status entities are found.
func (sq *StatusQuery) Only(ctx context.Context) (*Status, error) {
	nodes, err := sq.Limit(2).All(ctx)
	if err != nil {
		return nil, err
	}
	switch len(nodes) {
	case 1:
		return nodes[0], nil
	case 0:
		return nil, &NotFoundError{status.Label}
	default:
		return nil, &NotSingularError{status.Label}
	}
}

// OnlyX is like Only, but panics if an error occurs.
func (sq *StatusQuery) OnlyX(ctx context.Context) *Status {
	node, err := sq.Only(ctx)
	if err != nil {
		panic(err)
	}
	return node
}

// OnlyID is like Only, but returns the only Status ID in the query.
// Returns a *NotSingularError when more than one Status ID is found.
// Returns a *NotFoundError when no entities are found.
func (sq *StatusQuery) OnlyID(ctx context.Context) (id uuid.UUID, err error) {
	var ids []uuid.UUID
	if ids, err = sq.Limit(2).IDs(ctx); err != nil {
		return
	}
	switch len(ids) {
	case 1:
		id = ids[0]
	case 0:
		err = &NotFoundError{status.Label}
	default:
		err = &NotSingularError{status.Label}
	}
	return
}

// OnlyIDX is like OnlyID, but panics if an error occurs.
func (sq *StatusQuery) OnlyIDX(ctx context.Context) uuid.UUID {
	id, err := sq.OnlyID(ctx)
	if err != nil {
		panic(err)
	}
	return id
}

// All executes the query and returns a list of StatusSlice.
func (sq *StatusQuery) All(ctx context.Context) ([]*Status, error) {
	if err := sq.prepareQuery(ctx); err != nil {
		return nil, err
	}
	return sq.sqlAll(ctx)
}

// AllX is like All, but panics if an error occurs.
func (sq *StatusQuery) AllX(ctx context.Context) []*Status {
	nodes, err := sq.All(ctx)
	if err != nil {
		panic(err)
	}
	return nodes
}

// IDs executes the query and returns a list of Status IDs.
func (sq *StatusQuery) IDs(ctx context.Context) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	if err := sq.Select(status.FieldID).Scan(ctx, &ids); err != nil {
		return nil, err
	}
	return ids, nil
}

// IDsX is like IDs, but panics if an error occurs.
func (sq *StatusQuery) IDsX(ctx context.Context) []uuid.UUID {
	ids, err := sq.IDs(ctx)
	if err != nil {
		panic(err)
	}
	return ids
}

// Count returns the count of the given query.
func (sq *StatusQuery) Count(ctx context.Context) (int, error) {
	if err := sq.prepareQuery(ctx); err != nil {
		return 0, err
	}
	return sq.sqlCount(ctx)
}

// CountX is like Count, but panics if an error occurs.
func (sq *StatusQuery) CountX(ctx context.Context) int {
	count, err := sq.Count(ctx)
	if err != nil {
		panic(err)
	}
	return count
}

// Exist returns true if the query has elements in the graph.
func (sq *StatusQuery) Exist(ctx context.Context) (bool, error) {
	if err := sq.prepareQuery(ctx); err != nil {
		return false, err
	}
	return sq.sqlExist(ctx)
}

// ExistX is like Exist, but panics if an error occurs.
func (sq *StatusQuery) ExistX(ctx context.Context) bool {
	exist, err := sq.Exist(ctx)
	if err != nil {
		panic(err)
	}
	return exist
}

// Clone returns a duplicate of the StatusQuery builder, including all associated steps. It can be
// used to prepare common query builders and use them differently after the clone is made.
func (sq *StatusQuery) Clone() *StatusQuery {
	if sq == nil {
		return nil
	}
	return &StatusQuery{
		config:                                sq.config,
		limit:                                 sq.limit,
		offset:                                sq.offset,
		order:                                 append([]OrderFunc{}, sq.order...),
		predicates:                            append([]predicate.Status{}, sq.predicates...),
		withStatusToBuild:                     sq.withStatusToBuild.Clone(),
		withStatusToProvisionedNetwork:        sq.withStatusToProvisionedNetwork.Clone(),
		withStatusToProvisionedHost:           sq.withStatusToProvisionedHost.Clone(),
		withStatusToProvisioningStep:          sq.withStatusToProvisioningStep.Clone(),
		withStatusToTeam:                      sq.withStatusToTeam.Clone(),
		withStatusToPlan:                      sq.withStatusToPlan.Clone(),
		withStatusToServerTask:                sq.withStatusToServerTask.Clone(),
		withStatusToAdhocPlan:                 sq.withStatusToAdhocPlan.Clone(),
		withStatusToProvisioningScheduledStep: sq.withStatusToProvisioningScheduledStep.Clone(),
		// clone intermediate query.
		sql:    sq.sql.Clone(),
		path:   sq.path,
		unique: sq.unique,
	}
}

// WithStatusToBuild tells the query-builder to eager-load the nodes that are connected to
// the "StatusToBuild" edge. The optional arguments are used to configure the query builder of the edge.
func (sq *StatusQuery) WithStatusToBuild(opts ...func(*BuildQuery)) *StatusQuery {
	query := &BuildQuery{config: sq.config}
	for _, opt := range opts {
		opt(query)
	}
	sq.withStatusToBuild = query
	return sq
}

// WithStatusToProvisionedNetwork tells the query-builder to eager-load the nodes that are connected to
// the "StatusToProvisionedNetwork" edge. The optional arguments are used to configure the query builder of the edge.
func (sq *StatusQuery) WithStatusToProvisionedNetwork(opts ...func(*ProvisionedNetworkQuery)) *StatusQuery {
	query := &ProvisionedNetworkQuery{config: sq.config}
	for _, opt := range opts {
		opt(query)
	}
	sq.withStatusToProvisionedNetwork = query
	return sq
}

// WithStatusToProvisionedHost tells the query-builder to eager-load the nodes that are connected to
// the "StatusToProvisionedHost" edge. The optional arguments are used to configure the query builder of the edge.
func (sq *StatusQuery) WithStatusToProvisionedHost(opts ...func(*ProvisionedHostQuery)) *StatusQuery {
	query := &ProvisionedHostQuery{config: sq.config}
	for _, opt := range opts {
		opt(query)
	}
	sq.withStatusToProvisionedHost = query
	return sq
}

// WithStatusToProvisioningStep tells the query-builder to eager-load the nodes that are connected to
// the "StatusToProvisioningStep" edge. The optional arguments are used to configure the query builder of the edge.
func (sq *StatusQuery) WithStatusToProvisioningStep(opts ...func(*ProvisioningStepQuery)) *StatusQuery {
	query := &ProvisioningStepQuery{config: sq.config}
	for _, opt := range opts {
		opt(query)
	}
	sq.withStatusToProvisioningStep = query
	return sq
}

// WithStatusToTeam tells the query-builder to eager-load the nodes that are connected to
// the "StatusToTeam" edge. The optional arguments are used to configure the query builder of the edge.
func (sq *StatusQuery) WithStatusToTeam(opts ...func(*TeamQuery)) *StatusQuery {
	query := &TeamQuery{config: sq.config}
	for _, opt := range opts {
		opt(query)
	}
	sq.withStatusToTeam = query
	return sq
}

// WithStatusToPlan tells the query-builder to eager-load the nodes that are connected to
// the "StatusToPlan" edge. The optional arguments are used to configure the query builder of the edge.
func (sq *StatusQuery) WithStatusToPlan(opts ...func(*PlanQuery)) *StatusQuery {
	query := &PlanQuery{config: sq.config}
	for _, opt := range opts {
		opt(query)
	}
	sq.withStatusToPlan = query
	return sq
}

// WithStatusToServerTask tells the query-builder to eager-load the nodes that are connected to
// the "StatusToServerTask" edge. The optional arguments are used to configure the query builder of the edge.
func (sq *StatusQuery) WithStatusToServerTask(opts ...func(*ServerTaskQuery)) *StatusQuery {
	query := &ServerTaskQuery{config: sq.config}
	for _, opt := range opts {
		opt(query)
	}
	sq.withStatusToServerTask = query
	return sq
}

// WithStatusToAdhocPlan tells the query-builder to eager-load the nodes that are connected to
// the "StatusToAdhocPlan" edge. The optional arguments are used to configure the query builder of the edge.
func (sq *StatusQuery) WithStatusToAdhocPlan(opts ...func(*AdhocPlanQuery)) *StatusQuery {
	query := &AdhocPlanQuery{config: sq.config}
	for _, opt := range opts {
		opt(query)
	}
	sq.withStatusToAdhocPlan = query
	return sq
}

// WithStatusToProvisioningScheduledStep tells the query-builder to eager-load the nodes that are connected to
// the "StatusToProvisioningScheduledStep" edge. The optional arguments are used to configure the query builder of the edge.
func (sq *StatusQuery) WithStatusToProvisioningScheduledStep(opts ...func(*ProvisioningScheduledStepQuery)) *StatusQuery {
	query := &ProvisioningScheduledStepQuery{config: sq.config}
	for _, opt := range opts {
		opt(query)
	}
	sq.withStatusToProvisioningScheduledStep = query
	return sq
}

// GroupBy is used to group vertices by one or more fields/columns.
// It is often used with aggregate functions, like: count, max, mean, min, sum.
//
// Example:
//
//	var v []struct {
//		State status.State `json:"state,omitempty"`
//		Count int `json:"count,omitempty"`
//	}
//
//	client.Status.Query().
//		GroupBy(status.FieldState).
//		Aggregate(ent.Count()).
//		Scan(ctx, &v)
func (sq *StatusQuery) GroupBy(field string, fields ...string) *StatusGroupBy {
	grbuild := &StatusGroupBy{config: sq.config}
	grbuild.fields = append([]string{field}, fields...)
	grbuild.path = func(ctx context.Context) (prev *sql.Selector, err error) {
		if err := sq.prepareQuery(ctx); err != nil {
			return nil, err
		}
		return sq.sqlQuery(ctx), nil
	}
	grbuild.label = status.Label
	grbuild.flds, grbuild.scan = &grbuild.fields, grbuild.Scan
	return grbuild
}

// Select allows the selection one or more fields/columns for the given query,
// instead of selecting all fields in the entity.
//
// Example:
//
//	var v []struct {
//		State status.State `json:"state,omitempty"`
//	}
//
//	client.Status.Query().
//		Select(status.FieldState).
//		Scan(ctx, &v)
func (sq *StatusQuery) Select(fields ...string) *StatusSelect {
	sq.fields = append(sq.fields, fields...)
	selbuild := &StatusSelect{StatusQuery: sq}
	selbuild.label = status.Label
	selbuild.flds, selbuild.scan = &sq.fields, selbuild.Scan
	return selbuild
}

func (sq *StatusQuery) prepareQuery(ctx context.Context) error {
	for _, f := range sq.fields {
		if !status.ValidColumn(f) {
			return &ValidationError{Name: f, err: fmt.Errorf("ent: invalid field %q for query", f)}
		}
	}
	if sq.path != nil {
		prev, err := sq.path(ctx)
		if err != nil {
			return err
		}
		sq.sql = prev
	}
	return nil
}

func (sq *StatusQuery) sqlAll(ctx context.Context, hooks ...queryHook) ([]*Status, error) {
	var (
		nodes       = []*Status{}
		withFKs     = sq.withFKs
		_spec       = sq.querySpec()
		loadedTypes = [9]bool{
			sq.withStatusToBuild != nil,
			sq.withStatusToProvisionedNetwork != nil,
			sq.withStatusToProvisionedHost != nil,
			sq.withStatusToProvisioningStep != nil,
			sq.withStatusToTeam != nil,
			sq.withStatusToPlan != nil,
			sq.withStatusToServerTask != nil,
			sq.withStatusToAdhocPlan != nil,
			sq.withStatusToProvisioningScheduledStep != nil,
		}
	)
	if sq.withStatusToBuild != nil || sq.withStatusToProvisionedNetwork != nil || sq.withStatusToProvisionedHost != nil || sq.withStatusToProvisioningStep != nil || sq.withStatusToTeam != nil || sq.withStatusToPlan != nil || sq.withStatusToServerTask != nil || sq.withStatusToAdhocPlan != nil || sq.withStatusToProvisioningScheduledStep != nil {
		withFKs = true
	}
	if withFKs {
		_spec.Node.Columns = append(_spec.Node.Columns, status.ForeignKeys...)
	}
	_spec.ScanValues = func(columns []string) ([]interface{}, error) {
		return (*Status).scanValues(nil, columns)
	}
	_spec.Assign = func(columns []string, values []interface{}) error {
		node := &Status{config: sq.config}
		nodes = append(nodes, node)
		node.Edges.loadedTypes = loadedTypes
		return node.assignValues(columns, values)
	}
	for i := range hooks {
		hooks[i](ctx, _spec)
	}
	if err := sqlgraph.QueryNodes(ctx, sq.driver, _spec); err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nodes, nil
	}
	if query := sq.withStatusToBuild; query != nil {
		if err := sq.loadStatusToBuild(ctx, query, nodes, nil,
			func(n *Status, e *Build) { n.Edges.StatusToBuild = e }); err != nil {
			return nil, err
		}
	}
	if query := sq.withStatusToProvisionedNetwork; query != nil {
		if err := sq.loadStatusToProvisionedNetwork(ctx, query, nodes, nil,
			func(n *Status, e *ProvisionedNetwork) { n.Edges.StatusToProvisionedNetwork = e }); err != nil {
			return nil, err
		}
	}
	if query := sq.withStatusToProvisionedHost; query != nil {
		if err := sq.loadStatusToProvisionedHost(ctx, query, nodes, nil,
			func(n *Status, e *ProvisionedHost) { n.Edges.StatusToProvisionedHost = e }); err != nil {
			return nil, err
		}
	}
	if query := sq.withStatusToProvisioningStep; query != nil {
		if err := sq.loadStatusToProvisioningStep(ctx, query, nodes, nil,
			func(n *Status, e *ProvisioningStep) { n.Edges.StatusToProvisioningStep = e }); err != nil {
			return nil, err
		}
	}
	if query := sq.withStatusToTeam; query != nil {
		if err := sq.loadStatusToTeam(ctx, query, nodes, nil,
			func(n *Status, e *Team) { n.Edges.StatusToTeam = e }); err != nil {
			return nil, err
		}
	}
	if query := sq.withStatusToPlan; query != nil {
		if err := sq.loadStatusToPlan(ctx, query, nodes, nil,
			func(n *Status, e *Plan) { n.Edges.StatusToPlan = e }); err != nil {
			return nil, err
		}
	}
	if query := sq.withStatusToServerTask; query != nil {
		if err := sq.loadStatusToServerTask(ctx, query, nodes, nil,
			func(n *Status, e *ServerTask) { n.Edges.StatusToServerTask = e }); err != nil {
			return nil, err
		}
	}
	if query := sq.withStatusToAdhocPlan; query != nil {
		if err := sq.loadStatusToAdhocPlan(ctx, query, nodes, nil,
			func(n *Status, e *AdhocPlan) { n.Edges.StatusToAdhocPlan = e }); err != nil {
			return nil, err
		}
	}
	if query := sq.withStatusToProvisioningScheduledStep; query != nil {
		if err := sq.loadStatusToProvisioningScheduledStep(ctx, query, nodes, nil,
			func(n *Status, e *ProvisioningScheduledStep) { n.Edges.StatusToProvisioningScheduledStep = e }); err != nil {
			return nil, err
		}
	}
	return nodes, nil
}

func (sq *StatusQuery) loadStatusToBuild(ctx context.Context, query *BuildQuery, nodes []*Status, init func(*Status), assign func(*Status, *Build)) error {
	ids := make([]uuid.UUID, 0, len(nodes))
	nodeids := make(map[uuid.UUID][]*Status)
	for i := range nodes {
		if nodes[i].build_status == nil {
			continue
		}
		fk := *nodes[i].build_status
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
			return fmt.Errorf(`unexpected foreign-key "build_status" returned %v`, n.ID)
		}
		for i := range nodes {
			assign(nodes[i], n)
		}
	}
	return nil
}
func (sq *StatusQuery) loadStatusToProvisionedNetwork(ctx context.Context, query *ProvisionedNetworkQuery, nodes []*Status, init func(*Status), assign func(*Status, *ProvisionedNetwork)) error {
	ids := make([]uuid.UUID, 0, len(nodes))
	nodeids := make(map[uuid.UUID][]*Status)
	for i := range nodes {
		if nodes[i].provisioned_network_status == nil {
			continue
		}
		fk := *nodes[i].provisioned_network_status
		if _, ok := nodeids[fk]; !ok {
			ids = append(ids, fk)
		}
		nodeids[fk] = append(nodeids[fk], nodes[i])
	}
	query.Where(provisionednetwork.IDIn(ids...))
	neighbors, err := query.All(ctx)
	if err != nil {
		return err
	}
	for _, n := range neighbors {
		nodes, ok := nodeids[n.ID]
		if !ok {
			return fmt.Errorf(`unexpected foreign-key "provisioned_network_status" returned %v`, n.ID)
		}
		for i := range nodes {
			assign(nodes[i], n)
		}
	}
	return nil
}
func (sq *StatusQuery) loadStatusToProvisionedHost(ctx context.Context, query *ProvisionedHostQuery, nodes []*Status, init func(*Status), assign func(*Status, *ProvisionedHost)) error {
	ids := make([]uuid.UUID, 0, len(nodes))
	nodeids := make(map[uuid.UUID][]*Status)
	for i := range nodes {
		if nodes[i].provisioned_host_status == nil {
			continue
		}
		fk := *nodes[i].provisioned_host_status
		if _, ok := nodeids[fk]; !ok {
			ids = append(ids, fk)
		}
		nodeids[fk] = append(nodeids[fk], nodes[i])
	}
	query.Where(provisionedhost.IDIn(ids...))
	neighbors, err := query.All(ctx)
	if err != nil {
		return err
	}
	for _, n := range neighbors {
		nodes, ok := nodeids[n.ID]
		if !ok {
			return fmt.Errorf(`unexpected foreign-key "provisioned_host_status" returned %v`, n.ID)
		}
		for i := range nodes {
			assign(nodes[i], n)
		}
	}
	return nil
}
func (sq *StatusQuery) loadStatusToProvisioningStep(ctx context.Context, query *ProvisioningStepQuery, nodes []*Status, init func(*Status), assign func(*Status, *ProvisioningStep)) error {
	ids := make([]uuid.UUID, 0, len(nodes))
	nodeids := make(map[uuid.UUID][]*Status)
	for i := range nodes {
		if nodes[i].provisioning_step_status == nil {
			continue
		}
		fk := *nodes[i].provisioning_step_status
		if _, ok := nodeids[fk]; !ok {
			ids = append(ids, fk)
		}
		nodeids[fk] = append(nodeids[fk], nodes[i])
	}
	query.Where(provisioningstep.IDIn(ids...))
	neighbors, err := query.All(ctx)
	if err != nil {
		return err
	}
	for _, n := range neighbors {
		nodes, ok := nodeids[n.ID]
		if !ok {
			return fmt.Errorf(`unexpected foreign-key "provisioning_step_status" returned %v`, n.ID)
		}
		for i := range nodes {
			assign(nodes[i], n)
		}
	}
	return nil
}
func (sq *StatusQuery) loadStatusToTeam(ctx context.Context, query *TeamQuery, nodes []*Status, init func(*Status), assign func(*Status, *Team)) error {
	ids := make([]uuid.UUID, 0, len(nodes))
	nodeids := make(map[uuid.UUID][]*Status)
	for i := range nodes {
		if nodes[i].team_team_to_status == nil {
			continue
		}
		fk := *nodes[i].team_team_to_status
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
			return fmt.Errorf(`unexpected foreign-key "team_team_to_status" returned %v`, n.ID)
		}
		for i := range nodes {
			assign(nodes[i], n)
		}
	}
	return nil
}
func (sq *StatusQuery) loadStatusToPlan(ctx context.Context, query *PlanQuery, nodes []*Status, init func(*Status), assign func(*Status, *Plan)) error {
	ids := make([]uuid.UUID, 0, len(nodes))
	nodeids := make(map[uuid.UUID][]*Status)
	for i := range nodes {
		if nodes[i].plan_status == nil {
			continue
		}
		fk := *nodes[i].plan_status
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
			return fmt.Errorf(`unexpected foreign-key "plan_status" returned %v`, n.ID)
		}
		for i := range nodes {
			assign(nodes[i], n)
		}
	}
	return nil
}
func (sq *StatusQuery) loadStatusToServerTask(ctx context.Context, query *ServerTaskQuery, nodes []*Status, init func(*Status), assign func(*Status, *ServerTask)) error {
	ids := make([]uuid.UUID, 0, len(nodes))
	nodeids := make(map[uuid.UUID][]*Status)
	for i := range nodes {
		if nodes[i].server_task_server_task_to_status == nil {
			continue
		}
		fk := *nodes[i].server_task_server_task_to_status
		if _, ok := nodeids[fk]; !ok {
			ids = append(ids, fk)
		}
		nodeids[fk] = append(nodeids[fk], nodes[i])
	}
	query.Where(servertask.IDIn(ids...))
	neighbors, err := query.All(ctx)
	if err != nil {
		return err
	}
	for _, n := range neighbors {
		nodes, ok := nodeids[n.ID]
		if !ok {
			return fmt.Errorf(`unexpected foreign-key "server_task_server_task_to_status" returned %v`, n.ID)
		}
		for i := range nodes {
			assign(nodes[i], n)
		}
	}
	return nil
}
func (sq *StatusQuery) loadStatusToAdhocPlan(ctx context.Context, query *AdhocPlanQuery, nodes []*Status, init func(*Status), assign func(*Status, *AdhocPlan)) error {
	ids := make([]uuid.UUID, 0, len(nodes))
	nodeids := make(map[uuid.UUID][]*Status)
	for i := range nodes {
		if nodes[i].adhoc_plan_status == nil {
			continue
		}
		fk := *nodes[i].adhoc_plan_status
		if _, ok := nodeids[fk]; !ok {
			ids = append(ids, fk)
		}
		nodeids[fk] = append(nodeids[fk], nodes[i])
	}
	query.Where(adhocplan.IDIn(ids...))
	neighbors, err := query.All(ctx)
	if err != nil {
		return err
	}
	for _, n := range neighbors {
		nodes, ok := nodeids[n.ID]
		if !ok {
			return fmt.Errorf(`unexpected foreign-key "adhoc_plan_status" returned %v`, n.ID)
		}
		for i := range nodes {
			assign(nodes[i], n)
		}
	}
	return nil
}
func (sq *StatusQuery) loadStatusToProvisioningScheduledStep(ctx context.Context, query *ProvisioningScheduledStepQuery, nodes []*Status, init func(*Status), assign func(*Status, *ProvisioningScheduledStep)) error {
	ids := make([]uuid.UUID, 0, len(nodes))
	nodeids := make(map[uuid.UUID][]*Status)
	for i := range nodes {
		if nodes[i].provisioning_scheduled_step_status == nil {
			continue
		}
		fk := *nodes[i].provisioning_scheduled_step_status
		if _, ok := nodeids[fk]; !ok {
			ids = append(ids, fk)
		}
		nodeids[fk] = append(nodeids[fk], nodes[i])
	}
	query.Where(provisioningscheduledstep.IDIn(ids...))
	neighbors, err := query.All(ctx)
	if err != nil {
		return err
	}
	for _, n := range neighbors {
		nodes, ok := nodeids[n.ID]
		if !ok {
			return fmt.Errorf(`unexpected foreign-key "provisioning_scheduled_step_status" returned %v`, n.ID)
		}
		for i := range nodes {
			assign(nodes[i], n)
		}
	}
	return nil
}

func (sq *StatusQuery) sqlCount(ctx context.Context) (int, error) {
	_spec := sq.querySpec()
	_spec.Node.Columns = sq.fields
	if len(sq.fields) > 0 {
		_spec.Unique = sq.unique != nil && *sq.unique
	}
	return sqlgraph.CountNodes(ctx, sq.driver, _spec)
}

func (sq *StatusQuery) sqlExist(ctx context.Context) (bool, error) {
	n, err := sq.sqlCount(ctx)
	if err != nil {
		return false, fmt.Errorf("ent: check existence: %w", err)
	}
	return n > 0, nil
}

func (sq *StatusQuery) querySpec() *sqlgraph.QuerySpec {
	_spec := &sqlgraph.QuerySpec{
		Node: &sqlgraph.NodeSpec{
			Table:   status.Table,
			Columns: status.Columns,
			ID: &sqlgraph.FieldSpec{
				Type:   field.TypeUUID,
				Column: status.FieldID,
			},
		},
		From:   sq.sql,
		Unique: true,
	}
	if unique := sq.unique; unique != nil {
		_spec.Unique = *unique
	}
	if fields := sq.fields; len(fields) > 0 {
		_spec.Node.Columns = make([]string, 0, len(fields))
		_spec.Node.Columns = append(_spec.Node.Columns, status.FieldID)
		for i := range fields {
			if fields[i] != status.FieldID {
				_spec.Node.Columns = append(_spec.Node.Columns, fields[i])
			}
		}
	}
	if ps := sq.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if limit := sq.limit; limit != nil {
		_spec.Limit = *limit
	}
	if offset := sq.offset; offset != nil {
		_spec.Offset = *offset
	}
	if ps := sq.order; len(ps) > 0 {
		_spec.Order = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	return _spec
}

func (sq *StatusQuery) sqlQuery(ctx context.Context) *sql.Selector {
	builder := sql.Dialect(sq.driver.Dialect())
	t1 := builder.Table(status.Table)
	columns := sq.fields
	if len(columns) == 0 {
		columns = status.Columns
	}
	selector := builder.Select(t1.Columns(columns...)...).From(t1)
	if sq.sql != nil {
		selector = sq.sql
		selector.Select(selector.Columns(columns...)...)
	}
	if sq.unique != nil && *sq.unique {
		selector.Distinct()
	}
	for _, p := range sq.predicates {
		p(selector)
	}
	for _, p := range sq.order {
		p(selector)
	}
	if offset := sq.offset; offset != nil {
		// limit is mandatory for offset clause. We start
		// with default value, and override it below if needed.
		selector.Offset(*offset).Limit(math.MaxInt32)
	}
	if limit := sq.limit; limit != nil {
		selector.Limit(*limit)
	}
	return selector
}

// StatusGroupBy is the group-by builder for Status entities.
type StatusGroupBy struct {
	config
	selector
	fields []string
	fns    []AggregateFunc
	// intermediate query (i.e. traversal path).
	sql  *sql.Selector
	path func(context.Context) (*sql.Selector, error)
}

// Aggregate adds the given aggregation functions to the group-by query.
func (sgb *StatusGroupBy) Aggregate(fns ...AggregateFunc) *StatusGroupBy {
	sgb.fns = append(sgb.fns, fns...)
	return sgb
}

// Scan applies the group-by query and scans the result into the given value.
func (sgb *StatusGroupBy) Scan(ctx context.Context, v interface{}) error {
	query, err := sgb.path(ctx)
	if err != nil {
		return err
	}
	sgb.sql = query
	return sgb.sqlScan(ctx, v)
}

func (sgb *StatusGroupBy) sqlScan(ctx context.Context, v interface{}) error {
	for _, f := range sgb.fields {
		if !status.ValidColumn(f) {
			return &ValidationError{Name: f, err: fmt.Errorf("invalid field %q for group-by", f)}
		}
	}
	selector := sgb.sqlQuery()
	if err := selector.Err(); err != nil {
		return err
	}
	rows := &sql.Rows{}
	query, args := selector.Query()
	if err := sgb.driver.Query(ctx, query, args, rows); err != nil {
		return err
	}
	defer rows.Close()
	return sql.ScanSlice(rows, v)
}

func (sgb *StatusGroupBy) sqlQuery() *sql.Selector {
	selector := sgb.sql.Select()
	aggregation := make([]string, 0, len(sgb.fns))
	for _, fn := range sgb.fns {
		aggregation = append(aggregation, fn(selector))
	}
	// If no columns were selected in a custom aggregation function, the default
	// selection is the fields used for "group-by", and the aggregation functions.
	if len(selector.SelectedColumns()) == 0 {
		columns := make([]string, 0, len(sgb.fields)+len(sgb.fns))
		for _, f := range sgb.fields {
			columns = append(columns, selector.C(f))
		}
		columns = append(columns, aggregation...)
		selector.Select(columns...)
	}
	return selector.GroupBy(selector.Columns(sgb.fields...)...)
}

// StatusSelect is the builder for selecting fields of Status entities.
type StatusSelect struct {
	*StatusQuery
	selector
	// intermediate query (i.e. traversal path).
	sql *sql.Selector
}

// Scan applies the selector query and scans the result into the given value.
func (ss *StatusSelect) Scan(ctx context.Context, v interface{}) error {
	if err := ss.prepareQuery(ctx); err != nil {
		return err
	}
	ss.sql = ss.StatusQuery.sqlQuery(ctx)
	return ss.sqlScan(ctx, v)
}

func (ss *StatusSelect) sqlScan(ctx context.Context, v interface{}) error {
	rows := &sql.Rows{}
	query, args := ss.sql.Query()
	if err := ss.driver.Query(ctx, query, args, rows); err != nil {
		return err
	}
	defer rows.Close()
	return sql.ScanSlice(rows, v)
}
