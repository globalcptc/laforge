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
	"github.com/gen0cide/laforge/ent/finding"
	"github.com/gen0cide/laforge/ent/host"
	"github.com/gen0cide/laforge/ent/predicate"
	"github.com/gen0cide/laforge/ent/script"
	"github.com/gen0cide/laforge/ent/user"
	"github.com/google/uuid"
)

// FindingQuery is the builder for querying Finding entities.
type FindingQuery struct {
	config
	ctx                      *QueryContext
	order                    []finding.OrderOption
	inters                   []Interceptor
	predicates               []predicate.Finding
	withFindingToUser        *UserQuery
	withFindingToHost        *HostQuery
	withFindingToScript      *ScriptQuery
	withFindingToEnvironment *EnvironmentQuery
	withFKs                  bool
	modifiers                []func(*sql.Selector)
	loadTotal                []func(context.Context, []*Finding) error
	withNamedFindingToUser   map[string]*UserQuery
	// intermediate query (i.e. traversal path).
	sql  *sql.Selector
	path func(context.Context) (*sql.Selector, error)
}

// Where adds a new predicate for the FindingQuery builder.
func (fq *FindingQuery) Where(ps ...predicate.Finding) *FindingQuery {
	fq.predicates = append(fq.predicates, ps...)
	return fq
}

// Limit the number of records to be returned by this query.
func (fq *FindingQuery) Limit(limit int) *FindingQuery {
	fq.ctx.Limit = &limit
	return fq
}

// Offset to start from.
func (fq *FindingQuery) Offset(offset int) *FindingQuery {
	fq.ctx.Offset = &offset
	return fq
}

// Unique configures the query builder to filter duplicate records on query.
// By default, unique is set to true, and can be disabled using this method.
func (fq *FindingQuery) Unique(unique bool) *FindingQuery {
	fq.ctx.Unique = &unique
	return fq
}

// Order specifies how the records should be ordered.
func (fq *FindingQuery) Order(o ...finding.OrderOption) *FindingQuery {
	fq.order = append(fq.order, o...)
	return fq
}

// QueryFindingToUser chains the current query on the "FindingToUser" edge.
func (fq *FindingQuery) QueryFindingToUser() *UserQuery {
	query := (&UserClient{config: fq.config}).Query()
	query.path = func(ctx context.Context) (fromU *sql.Selector, err error) {
		if err := fq.prepareQuery(ctx); err != nil {
			return nil, err
		}
		selector := fq.sqlQuery(ctx)
		if err := selector.Err(); err != nil {
			return nil, err
		}
		step := sqlgraph.NewStep(
			sqlgraph.From(finding.Table, finding.FieldID, selector),
			sqlgraph.To(user.Table, user.FieldID),
			sqlgraph.Edge(sqlgraph.O2M, false, finding.FindingToUserTable, finding.FindingToUserColumn),
		)
		fromU = sqlgraph.SetNeighbors(fq.driver.Dialect(), step)
		return fromU, nil
	}
	return query
}

// QueryFindingToHost chains the current query on the "FindingToHost" edge.
func (fq *FindingQuery) QueryFindingToHost() *HostQuery {
	query := (&HostClient{config: fq.config}).Query()
	query.path = func(ctx context.Context) (fromU *sql.Selector, err error) {
		if err := fq.prepareQuery(ctx); err != nil {
			return nil, err
		}
		selector := fq.sqlQuery(ctx)
		if err := selector.Err(); err != nil {
			return nil, err
		}
		step := sqlgraph.NewStep(
			sqlgraph.From(finding.Table, finding.FieldID, selector),
			sqlgraph.To(host.Table, host.FieldID),
			sqlgraph.Edge(sqlgraph.M2O, false, finding.FindingToHostTable, finding.FindingToHostColumn),
		)
		fromU = sqlgraph.SetNeighbors(fq.driver.Dialect(), step)
		return fromU, nil
	}
	return query
}

// QueryFindingToScript chains the current query on the "FindingToScript" edge.
func (fq *FindingQuery) QueryFindingToScript() *ScriptQuery {
	query := (&ScriptClient{config: fq.config}).Query()
	query.path = func(ctx context.Context) (fromU *sql.Selector, err error) {
		if err := fq.prepareQuery(ctx); err != nil {
			return nil, err
		}
		selector := fq.sqlQuery(ctx)
		if err := selector.Err(); err != nil {
			return nil, err
		}
		step := sqlgraph.NewStep(
			sqlgraph.From(finding.Table, finding.FieldID, selector),
			sqlgraph.To(script.Table, script.FieldID),
			sqlgraph.Edge(sqlgraph.M2O, true, finding.FindingToScriptTable, finding.FindingToScriptColumn),
		)
		fromU = sqlgraph.SetNeighbors(fq.driver.Dialect(), step)
		return fromU, nil
	}
	return query
}

// QueryFindingToEnvironment chains the current query on the "FindingToEnvironment" edge.
func (fq *FindingQuery) QueryFindingToEnvironment() *EnvironmentQuery {
	query := (&EnvironmentClient{config: fq.config}).Query()
	query.path = func(ctx context.Context) (fromU *sql.Selector, err error) {
		if err := fq.prepareQuery(ctx); err != nil {
			return nil, err
		}
		selector := fq.sqlQuery(ctx)
		if err := selector.Err(); err != nil {
			return nil, err
		}
		step := sqlgraph.NewStep(
			sqlgraph.From(finding.Table, finding.FieldID, selector),
			sqlgraph.To(environment.Table, environment.FieldID),
			sqlgraph.Edge(sqlgraph.M2O, true, finding.FindingToEnvironmentTable, finding.FindingToEnvironmentColumn),
		)
		fromU = sqlgraph.SetNeighbors(fq.driver.Dialect(), step)
		return fromU, nil
	}
	return query
}

// First returns the first Finding entity from the query.
// Returns a *NotFoundError when no Finding was found.
func (fq *FindingQuery) First(ctx context.Context) (*Finding, error) {
	nodes, err := fq.Limit(1).All(setContextOp(ctx, fq.ctx, "First"))
	if err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nil, &NotFoundError{finding.Label}
	}
	return nodes[0], nil
}

// FirstX is like First, but panics if an error occurs.
func (fq *FindingQuery) FirstX(ctx context.Context) *Finding {
	node, err := fq.First(ctx)
	if err != nil && !IsNotFound(err) {
		panic(err)
	}
	return node
}

// FirstID returns the first Finding ID from the query.
// Returns a *NotFoundError when no Finding ID was found.
func (fq *FindingQuery) FirstID(ctx context.Context) (id uuid.UUID, err error) {
	var ids []uuid.UUID
	if ids, err = fq.Limit(1).IDs(setContextOp(ctx, fq.ctx, "FirstID")); err != nil {
		return
	}
	if len(ids) == 0 {
		err = &NotFoundError{finding.Label}
		return
	}
	return ids[0], nil
}

// FirstIDX is like FirstID, but panics if an error occurs.
func (fq *FindingQuery) FirstIDX(ctx context.Context) uuid.UUID {
	id, err := fq.FirstID(ctx)
	if err != nil && !IsNotFound(err) {
		panic(err)
	}
	return id
}

// Only returns a single Finding entity found by the query, ensuring it only returns one.
// Returns a *NotSingularError when more than one Finding entity is found.
// Returns a *NotFoundError when no Finding entities are found.
func (fq *FindingQuery) Only(ctx context.Context) (*Finding, error) {
	nodes, err := fq.Limit(2).All(setContextOp(ctx, fq.ctx, "Only"))
	if err != nil {
		return nil, err
	}
	switch len(nodes) {
	case 1:
		return nodes[0], nil
	case 0:
		return nil, &NotFoundError{finding.Label}
	default:
		return nil, &NotSingularError{finding.Label}
	}
}

// OnlyX is like Only, but panics if an error occurs.
func (fq *FindingQuery) OnlyX(ctx context.Context) *Finding {
	node, err := fq.Only(ctx)
	if err != nil {
		panic(err)
	}
	return node
}

// OnlyID is like Only, but returns the only Finding ID in the query.
// Returns a *NotSingularError when more than one Finding ID is found.
// Returns a *NotFoundError when no entities are found.
func (fq *FindingQuery) OnlyID(ctx context.Context) (id uuid.UUID, err error) {
	var ids []uuid.UUID
	if ids, err = fq.Limit(2).IDs(setContextOp(ctx, fq.ctx, "OnlyID")); err != nil {
		return
	}
	switch len(ids) {
	case 1:
		id = ids[0]
	case 0:
		err = &NotFoundError{finding.Label}
	default:
		err = &NotSingularError{finding.Label}
	}
	return
}

// OnlyIDX is like OnlyID, but panics if an error occurs.
func (fq *FindingQuery) OnlyIDX(ctx context.Context) uuid.UUID {
	id, err := fq.OnlyID(ctx)
	if err != nil {
		panic(err)
	}
	return id
}

// All executes the query and returns a list of Findings.
func (fq *FindingQuery) All(ctx context.Context) ([]*Finding, error) {
	ctx = setContextOp(ctx, fq.ctx, "All")
	if err := fq.prepareQuery(ctx); err != nil {
		return nil, err
	}
	qr := querierAll[[]*Finding, *FindingQuery]()
	return withInterceptors[[]*Finding](ctx, fq, qr, fq.inters)
}

// AllX is like All, but panics if an error occurs.
func (fq *FindingQuery) AllX(ctx context.Context) []*Finding {
	nodes, err := fq.All(ctx)
	if err != nil {
		panic(err)
	}
	return nodes
}

// IDs executes the query and returns a list of Finding IDs.
func (fq *FindingQuery) IDs(ctx context.Context) (ids []uuid.UUID, err error) {
	if fq.ctx.Unique == nil && fq.path != nil {
		fq.Unique(true)
	}
	ctx = setContextOp(ctx, fq.ctx, "IDs")
	if err = fq.Select(finding.FieldID).Scan(ctx, &ids); err != nil {
		return nil, err
	}
	return ids, nil
}

// IDsX is like IDs, but panics if an error occurs.
func (fq *FindingQuery) IDsX(ctx context.Context) []uuid.UUID {
	ids, err := fq.IDs(ctx)
	if err != nil {
		panic(err)
	}
	return ids
}

// Count returns the count of the given query.
func (fq *FindingQuery) Count(ctx context.Context) (int, error) {
	ctx = setContextOp(ctx, fq.ctx, "Count")
	if err := fq.prepareQuery(ctx); err != nil {
		return 0, err
	}
	return withInterceptors[int](ctx, fq, querierCount[*FindingQuery](), fq.inters)
}

// CountX is like Count, but panics if an error occurs.
func (fq *FindingQuery) CountX(ctx context.Context) int {
	count, err := fq.Count(ctx)
	if err != nil {
		panic(err)
	}
	return count
}

// Exist returns true if the query has elements in the graph.
func (fq *FindingQuery) Exist(ctx context.Context) (bool, error) {
	ctx = setContextOp(ctx, fq.ctx, "Exist")
	switch _, err := fq.FirstID(ctx); {
	case IsNotFound(err):
		return false, nil
	case err != nil:
		return false, fmt.Errorf("ent: check existence: %w", err)
	default:
		return true, nil
	}
}

// ExistX is like Exist, but panics if an error occurs.
func (fq *FindingQuery) ExistX(ctx context.Context) bool {
	exist, err := fq.Exist(ctx)
	if err != nil {
		panic(err)
	}
	return exist
}

// Clone returns a duplicate of the FindingQuery builder, including all associated steps. It can be
// used to prepare common query builders and use them differently after the clone is made.
func (fq *FindingQuery) Clone() *FindingQuery {
	if fq == nil {
		return nil
	}
	return &FindingQuery{
		config:                   fq.config,
		ctx:                      fq.ctx.Clone(),
		order:                    append([]finding.OrderOption{}, fq.order...),
		inters:                   append([]Interceptor{}, fq.inters...),
		predicates:               append([]predicate.Finding{}, fq.predicates...),
		withFindingToUser:        fq.withFindingToUser.Clone(),
		withFindingToHost:        fq.withFindingToHost.Clone(),
		withFindingToScript:      fq.withFindingToScript.Clone(),
		withFindingToEnvironment: fq.withFindingToEnvironment.Clone(),
		// clone intermediate query.
		sql:  fq.sql.Clone(),
		path: fq.path,
	}
}

// WithFindingToUser tells the query-builder to eager-load the nodes that are connected to
// the "FindingToUser" edge. The optional arguments are used to configure the query builder of the edge.
func (fq *FindingQuery) WithFindingToUser(opts ...func(*UserQuery)) *FindingQuery {
	query := (&UserClient{config: fq.config}).Query()
	for _, opt := range opts {
		opt(query)
	}
	fq.withFindingToUser = query
	return fq
}

// WithFindingToHost tells the query-builder to eager-load the nodes that are connected to
// the "FindingToHost" edge. The optional arguments are used to configure the query builder of the edge.
func (fq *FindingQuery) WithFindingToHost(opts ...func(*HostQuery)) *FindingQuery {
	query := (&HostClient{config: fq.config}).Query()
	for _, opt := range opts {
		opt(query)
	}
	fq.withFindingToHost = query
	return fq
}

// WithFindingToScript tells the query-builder to eager-load the nodes that are connected to
// the "FindingToScript" edge. The optional arguments are used to configure the query builder of the edge.
func (fq *FindingQuery) WithFindingToScript(opts ...func(*ScriptQuery)) *FindingQuery {
	query := (&ScriptClient{config: fq.config}).Query()
	for _, opt := range opts {
		opt(query)
	}
	fq.withFindingToScript = query
	return fq
}

// WithFindingToEnvironment tells the query-builder to eager-load the nodes that are connected to
// the "FindingToEnvironment" edge. The optional arguments are used to configure the query builder of the edge.
func (fq *FindingQuery) WithFindingToEnvironment(opts ...func(*EnvironmentQuery)) *FindingQuery {
	query := (&EnvironmentClient{config: fq.config}).Query()
	for _, opt := range opts {
		opt(query)
	}
	fq.withFindingToEnvironment = query
	return fq
}

// GroupBy is used to group vertices by one or more fields/columns.
// It is often used with aggregate functions, like: count, max, mean, min, sum.
//
// Example:
//
//	var v []struct {
//		Name string `json:"name,omitempty" hcl:"name,attr"`
//		Count int `json:"count,omitempty"`
//	}
//
//	client.Finding.Query().
//		GroupBy(finding.FieldName).
//		Aggregate(ent.Count()).
//		Scan(ctx, &v)
func (fq *FindingQuery) GroupBy(field string, fields ...string) *FindingGroupBy {
	fq.ctx.Fields = append([]string{field}, fields...)
	grbuild := &FindingGroupBy{build: fq}
	grbuild.flds = &fq.ctx.Fields
	grbuild.label = finding.Label
	grbuild.scan = grbuild.Scan
	return grbuild
}

// Select allows the selection one or more fields/columns for the given query,
// instead of selecting all fields in the entity.
//
// Example:
//
//	var v []struct {
//		Name string `json:"name,omitempty" hcl:"name,attr"`
//	}
//
//	client.Finding.Query().
//		Select(finding.FieldName).
//		Scan(ctx, &v)
func (fq *FindingQuery) Select(fields ...string) *FindingSelect {
	fq.ctx.Fields = append(fq.ctx.Fields, fields...)
	sbuild := &FindingSelect{FindingQuery: fq}
	sbuild.label = finding.Label
	sbuild.flds, sbuild.scan = &fq.ctx.Fields, sbuild.Scan
	return sbuild
}

// Aggregate returns a FindingSelect configured with the given aggregations.
func (fq *FindingQuery) Aggregate(fns ...AggregateFunc) *FindingSelect {
	return fq.Select().Aggregate(fns...)
}

func (fq *FindingQuery) prepareQuery(ctx context.Context) error {
	for _, inter := range fq.inters {
		if inter == nil {
			return fmt.Errorf("ent: uninitialized interceptor (forgotten import ent/runtime?)")
		}
		if trv, ok := inter.(Traverser); ok {
			if err := trv.Traverse(ctx, fq); err != nil {
				return err
			}
		}
	}
	for _, f := range fq.ctx.Fields {
		if !finding.ValidColumn(f) {
			return &ValidationError{Name: f, err: fmt.Errorf("ent: invalid field %q for query", f)}
		}
	}
	if fq.path != nil {
		prev, err := fq.path(ctx)
		if err != nil {
			return err
		}
		fq.sql = prev
	}
	return nil
}

func (fq *FindingQuery) sqlAll(ctx context.Context, hooks ...queryHook) ([]*Finding, error) {
	var (
		nodes       = []*Finding{}
		withFKs     = fq.withFKs
		_spec       = fq.querySpec()
		loadedTypes = [4]bool{
			fq.withFindingToUser != nil,
			fq.withFindingToHost != nil,
			fq.withFindingToScript != nil,
			fq.withFindingToEnvironment != nil,
		}
	)
	if fq.withFindingToHost != nil || fq.withFindingToScript != nil || fq.withFindingToEnvironment != nil {
		withFKs = true
	}
	if withFKs {
		_spec.Node.Columns = append(_spec.Node.Columns, finding.ForeignKeys...)
	}
	_spec.ScanValues = func(columns []string) ([]any, error) {
		return (*Finding).scanValues(nil, columns)
	}
	_spec.Assign = func(columns []string, values []any) error {
		node := &Finding{config: fq.config}
		nodes = append(nodes, node)
		node.Edges.loadedTypes = loadedTypes
		return node.assignValues(columns, values)
	}
	if len(fq.modifiers) > 0 {
		_spec.Modifiers = fq.modifiers
	}
	for i := range hooks {
		hooks[i](ctx, _spec)
	}
	if err := sqlgraph.QueryNodes(ctx, fq.driver, _spec); err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nodes, nil
	}
	if query := fq.withFindingToUser; query != nil {
		if err := fq.loadFindingToUser(ctx, query, nodes,
			func(n *Finding) { n.Edges.FindingToUser = []*User{} },
			func(n *Finding, e *User) { n.Edges.FindingToUser = append(n.Edges.FindingToUser, e) }); err != nil {
			return nil, err
		}
	}
	if query := fq.withFindingToHost; query != nil {
		if err := fq.loadFindingToHost(ctx, query, nodes, nil,
			func(n *Finding, e *Host) { n.Edges.FindingToHost = e }); err != nil {
			return nil, err
		}
	}
	if query := fq.withFindingToScript; query != nil {
		if err := fq.loadFindingToScript(ctx, query, nodes, nil,
			func(n *Finding, e *Script) { n.Edges.FindingToScript = e }); err != nil {
			return nil, err
		}
	}
	if query := fq.withFindingToEnvironment; query != nil {
		if err := fq.loadFindingToEnvironment(ctx, query, nodes, nil,
			func(n *Finding, e *Environment) { n.Edges.FindingToEnvironment = e }); err != nil {
			return nil, err
		}
	}
	for name, query := range fq.withNamedFindingToUser {
		if err := fq.loadFindingToUser(ctx, query, nodes,
			func(n *Finding) { n.appendNamedFindingToUser(name) },
			func(n *Finding, e *User) { n.appendNamedFindingToUser(name, e) }); err != nil {
			return nil, err
		}
	}
	for i := range fq.loadTotal {
		if err := fq.loadTotal[i](ctx, nodes); err != nil {
			return nil, err
		}
	}
	return nodes, nil
}

func (fq *FindingQuery) loadFindingToUser(ctx context.Context, query *UserQuery, nodes []*Finding, init func(*Finding), assign func(*Finding, *User)) error {
	fks := make([]driver.Value, 0, len(nodes))
	nodeids := make(map[uuid.UUID]*Finding)
	for i := range nodes {
		fks = append(fks, nodes[i].ID)
		nodeids[nodes[i].ID] = nodes[i]
		if init != nil {
			init(nodes[i])
		}
	}
	query.withFKs = true
	query.Where(predicate.User(func(s *sql.Selector) {
		s.Where(sql.InValues(s.C(finding.FindingToUserColumn), fks...))
	}))
	neighbors, err := query.All(ctx)
	if err != nil {
		return err
	}
	for _, n := range neighbors {
		fk := n.finding_finding_to_user
		if fk == nil {
			return fmt.Errorf(`foreign-key "finding_finding_to_user" is nil for node %v`, n.ID)
		}
		node, ok := nodeids[*fk]
		if !ok {
			return fmt.Errorf(`unexpected referenced foreign-key "finding_finding_to_user" returned %v for node %v`, *fk, n.ID)
		}
		assign(node, n)
	}
	return nil
}
func (fq *FindingQuery) loadFindingToHost(ctx context.Context, query *HostQuery, nodes []*Finding, init func(*Finding), assign func(*Finding, *Host)) error {
	ids := make([]uuid.UUID, 0, len(nodes))
	nodeids := make(map[uuid.UUID][]*Finding)
	for i := range nodes {
		if nodes[i].finding_finding_to_host == nil {
			continue
		}
		fk := *nodes[i].finding_finding_to_host
		if _, ok := nodeids[fk]; !ok {
			ids = append(ids, fk)
		}
		nodeids[fk] = append(nodeids[fk], nodes[i])
	}
	if len(ids) == 0 {
		return nil
	}
	query.Where(host.IDIn(ids...))
	neighbors, err := query.All(ctx)
	if err != nil {
		return err
	}
	for _, n := range neighbors {
		nodes, ok := nodeids[n.ID]
		if !ok {
			return fmt.Errorf(`unexpected foreign-key "finding_finding_to_host" returned %v`, n.ID)
		}
		for i := range nodes {
			assign(nodes[i], n)
		}
	}
	return nil
}
func (fq *FindingQuery) loadFindingToScript(ctx context.Context, query *ScriptQuery, nodes []*Finding, init func(*Finding), assign func(*Finding, *Script)) error {
	ids := make([]uuid.UUID, 0, len(nodes))
	nodeids := make(map[uuid.UUID][]*Finding)
	for i := range nodes {
		if nodes[i].script_script_to_finding == nil {
			continue
		}
		fk := *nodes[i].script_script_to_finding
		if _, ok := nodeids[fk]; !ok {
			ids = append(ids, fk)
		}
		nodeids[fk] = append(nodeids[fk], nodes[i])
	}
	if len(ids) == 0 {
		return nil
	}
	query.Where(script.IDIn(ids...))
	neighbors, err := query.All(ctx)
	if err != nil {
		return err
	}
	for _, n := range neighbors {
		nodes, ok := nodeids[n.ID]
		if !ok {
			return fmt.Errorf(`unexpected foreign-key "script_script_to_finding" returned %v`, n.ID)
		}
		for i := range nodes {
			assign(nodes[i], n)
		}
	}
	return nil
}
func (fq *FindingQuery) loadFindingToEnvironment(ctx context.Context, query *EnvironmentQuery, nodes []*Finding, init func(*Finding), assign func(*Finding, *Environment)) error {
	ids := make([]uuid.UUID, 0, len(nodes))
	nodeids := make(map[uuid.UUID][]*Finding)
	for i := range nodes {
		if nodes[i].environment_environment_to_finding == nil {
			continue
		}
		fk := *nodes[i].environment_environment_to_finding
		if _, ok := nodeids[fk]; !ok {
			ids = append(ids, fk)
		}
		nodeids[fk] = append(nodeids[fk], nodes[i])
	}
	if len(ids) == 0 {
		return nil
	}
	query.Where(environment.IDIn(ids...))
	neighbors, err := query.All(ctx)
	if err != nil {
		return err
	}
	for _, n := range neighbors {
		nodes, ok := nodeids[n.ID]
		if !ok {
			return fmt.Errorf(`unexpected foreign-key "environment_environment_to_finding" returned %v`, n.ID)
		}
		for i := range nodes {
			assign(nodes[i], n)
		}
	}
	return nil
}

func (fq *FindingQuery) sqlCount(ctx context.Context) (int, error) {
	_spec := fq.querySpec()
	if len(fq.modifiers) > 0 {
		_spec.Modifiers = fq.modifiers
	}
	_spec.Node.Columns = fq.ctx.Fields
	if len(fq.ctx.Fields) > 0 {
		_spec.Unique = fq.ctx.Unique != nil && *fq.ctx.Unique
	}
	return sqlgraph.CountNodes(ctx, fq.driver, _spec)
}

func (fq *FindingQuery) querySpec() *sqlgraph.QuerySpec {
	_spec := sqlgraph.NewQuerySpec(finding.Table, finding.Columns, sqlgraph.NewFieldSpec(finding.FieldID, field.TypeUUID))
	_spec.From = fq.sql
	if unique := fq.ctx.Unique; unique != nil {
		_spec.Unique = *unique
	} else if fq.path != nil {
		_spec.Unique = true
	}
	if fields := fq.ctx.Fields; len(fields) > 0 {
		_spec.Node.Columns = make([]string, 0, len(fields))
		_spec.Node.Columns = append(_spec.Node.Columns, finding.FieldID)
		for i := range fields {
			if fields[i] != finding.FieldID {
				_spec.Node.Columns = append(_spec.Node.Columns, fields[i])
			}
		}
	}
	if ps := fq.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if limit := fq.ctx.Limit; limit != nil {
		_spec.Limit = *limit
	}
	if offset := fq.ctx.Offset; offset != nil {
		_spec.Offset = *offset
	}
	if ps := fq.order; len(ps) > 0 {
		_spec.Order = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	return _spec
}

func (fq *FindingQuery) sqlQuery(ctx context.Context) *sql.Selector {
	builder := sql.Dialect(fq.driver.Dialect())
	t1 := builder.Table(finding.Table)
	columns := fq.ctx.Fields
	if len(columns) == 0 {
		columns = finding.Columns
	}
	selector := builder.Select(t1.Columns(columns...)...).From(t1)
	if fq.sql != nil {
		selector = fq.sql
		selector.Select(selector.Columns(columns...)...)
	}
	if fq.ctx.Unique != nil && *fq.ctx.Unique {
		selector.Distinct()
	}
	for _, p := range fq.predicates {
		p(selector)
	}
	for _, p := range fq.order {
		p(selector)
	}
	if offset := fq.ctx.Offset; offset != nil {
		// limit is mandatory for offset clause. We start
		// with default value, and override it below if needed.
		selector.Offset(*offset).Limit(math.MaxInt32)
	}
	if limit := fq.ctx.Limit; limit != nil {
		selector.Limit(*limit)
	}
	return selector
}

// WithNamedFindingToUser tells the query-builder to eager-load the nodes that are connected to the "FindingToUser"
// edge with the given name. The optional arguments are used to configure the query builder of the edge.
func (fq *FindingQuery) WithNamedFindingToUser(name string, opts ...func(*UserQuery)) *FindingQuery {
	query := (&UserClient{config: fq.config}).Query()
	for _, opt := range opts {
		opt(query)
	}
	if fq.withNamedFindingToUser == nil {
		fq.withNamedFindingToUser = make(map[string]*UserQuery)
	}
	fq.withNamedFindingToUser[name] = query
	return fq
}

// FindingGroupBy is the group-by builder for Finding entities.
type FindingGroupBy struct {
	selector
	build *FindingQuery
}

// Aggregate adds the given aggregation functions to the group-by query.
func (fgb *FindingGroupBy) Aggregate(fns ...AggregateFunc) *FindingGroupBy {
	fgb.fns = append(fgb.fns, fns...)
	return fgb
}

// Scan applies the selector query and scans the result into the given value.
func (fgb *FindingGroupBy) Scan(ctx context.Context, v any) error {
	ctx = setContextOp(ctx, fgb.build.ctx, "GroupBy")
	if err := fgb.build.prepareQuery(ctx); err != nil {
		return err
	}
	return scanWithInterceptors[*FindingQuery, *FindingGroupBy](ctx, fgb.build, fgb, fgb.build.inters, v)
}

func (fgb *FindingGroupBy) sqlScan(ctx context.Context, root *FindingQuery, v any) error {
	selector := root.sqlQuery(ctx).Select()
	aggregation := make([]string, 0, len(fgb.fns))
	for _, fn := range fgb.fns {
		aggregation = append(aggregation, fn(selector))
	}
	if len(selector.SelectedColumns()) == 0 {
		columns := make([]string, 0, len(*fgb.flds)+len(fgb.fns))
		for _, f := range *fgb.flds {
			columns = append(columns, selector.C(f))
		}
		columns = append(columns, aggregation...)
		selector.Select(columns...)
	}
	selector.GroupBy(selector.Columns(*fgb.flds...)...)
	if err := selector.Err(); err != nil {
		return err
	}
	rows := &sql.Rows{}
	query, args := selector.Query()
	if err := fgb.build.driver.Query(ctx, query, args, rows); err != nil {
		return err
	}
	defer rows.Close()
	return sql.ScanSlice(rows, v)
}

// FindingSelect is the builder for selecting fields of Finding entities.
type FindingSelect struct {
	*FindingQuery
	selector
}

// Aggregate adds the given aggregation functions to the selector query.
func (fs *FindingSelect) Aggregate(fns ...AggregateFunc) *FindingSelect {
	fs.fns = append(fs.fns, fns...)
	return fs
}

// Scan applies the selector query and scans the result into the given value.
func (fs *FindingSelect) Scan(ctx context.Context, v any) error {
	ctx = setContextOp(ctx, fs.ctx, "Select")
	if err := fs.prepareQuery(ctx); err != nil {
		return err
	}
	return scanWithInterceptors[*FindingQuery, *FindingSelect](ctx, fs.FindingQuery, fs, fs.inters, v)
}

func (fs *FindingSelect) sqlScan(ctx context.Context, root *FindingQuery, v any) error {
	selector := root.sqlQuery(ctx)
	aggregation := make([]string, 0, len(fs.fns))
	for _, fn := range fs.fns {
		aggregation = append(aggregation, fn(selector))
	}
	switch n := len(*fs.selector.flds); {
	case n == 0 && len(aggregation) > 0:
		selector.Select(aggregation...)
	case n != 0 && len(aggregation) > 0:
		selector.AppendSelect(aggregation...)
	}
	rows := &sql.Rows{}
	query, args := selector.Query()
	if err := fs.driver.Query(ctx, query, args, rows); err != nil {
		return err
	}
	defer rows.Close()
	return sql.ScanSlice(rows, v)
}
