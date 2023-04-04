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
	"github.com/gen0cide/laforge/ent/predicate"
	"github.com/gen0cide/laforge/ent/script"
	"github.com/gen0cide/laforge/ent/user"
	"github.com/gen0cide/laforge/ent/validation"
	"github.com/google/uuid"
)

// ScriptQuery is the builder for querying Script entities.
type ScriptQuery struct {
	config
	limit                   *int
	offset                  *int
	unique                  *bool
	order                   []OrderFunc
	fields                  []string
	predicates              []predicate.Script
	withScriptToUser        *UserQuery
	withScriptToFinding     *FindingQuery
	withScriptToEnvironment *EnvironmentQuery
	withScriptToValidation  *ValidationQuery
	withFKs                 bool
	// intermediate query (i.e. traversal path).
	sql  *sql.Selector
	path func(context.Context) (*sql.Selector, error)
}

// Where adds a new predicate for the ScriptQuery builder.
func (sq *ScriptQuery) Where(ps ...predicate.Script) *ScriptQuery {
	sq.predicates = append(sq.predicates, ps...)
	return sq
}

// Limit adds a limit step to the query.
func (sq *ScriptQuery) Limit(limit int) *ScriptQuery {
	sq.limit = &limit
	return sq
}

// Offset adds an offset step to the query.
func (sq *ScriptQuery) Offset(offset int) *ScriptQuery {
	sq.offset = &offset
	return sq
}

// Unique configures the query builder to filter duplicate records on query.
// By default, unique is set to true, and can be disabled using this method.
func (sq *ScriptQuery) Unique(unique bool) *ScriptQuery {
	sq.unique = &unique
	return sq
}

// Order adds an order step to the query.
func (sq *ScriptQuery) Order(o ...OrderFunc) *ScriptQuery {
	sq.order = append(sq.order, o...)
	return sq
}

// QueryScriptToUser chains the current query on the "ScriptToUser" edge.
func (sq *ScriptQuery) QueryScriptToUser() *UserQuery {
	query := &UserQuery{config: sq.config}
	query.path = func(ctx context.Context) (fromU *sql.Selector, err error) {
		if err := sq.prepareQuery(ctx); err != nil {
			return nil, err
		}
		selector := sq.sqlQuery(ctx)
		if err := selector.Err(); err != nil {
			return nil, err
		}
		step := sqlgraph.NewStep(
			sqlgraph.From(script.Table, script.FieldID, selector),
			sqlgraph.To(user.Table, user.FieldID),
			sqlgraph.Edge(sqlgraph.O2M, false, script.ScriptToUserTable, script.ScriptToUserColumn),
		)
		fromU = sqlgraph.SetNeighbors(sq.driver.Dialect(), step)
		return fromU, nil
	}
	return query
}

// QueryScriptToFinding chains the current query on the "ScriptToFinding" edge.
func (sq *ScriptQuery) QueryScriptToFinding() *FindingQuery {
	query := &FindingQuery{config: sq.config}
	query.path = func(ctx context.Context) (fromU *sql.Selector, err error) {
		if err := sq.prepareQuery(ctx); err != nil {
			return nil, err
		}
		selector := sq.sqlQuery(ctx)
		if err := selector.Err(); err != nil {
			return nil, err
		}
		step := sqlgraph.NewStep(
			sqlgraph.From(script.Table, script.FieldID, selector),
			sqlgraph.To(finding.Table, finding.FieldID),
			sqlgraph.Edge(sqlgraph.O2M, false, script.ScriptToFindingTable, script.ScriptToFindingColumn),
		)
		fromU = sqlgraph.SetNeighbors(sq.driver.Dialect(), step)
		return fromU, nil
	}
	return query
}

// QueryScriptToEnvironment chains the current query on the "ScriptToEnvironment" edge.
func (sq *ScriptQuery) QueryScriptToEnvironment() *EnvironmentQuery {
	query := &EnvironmentQuery{config: sq.config}
	query.path = func(ctx context.Context) (fromU *sql.Selector, err error) {
		if err := sq.prepareQuery(ctx); err != nil {
			return nil, err
		}
		selector := sq.sqlQuery(ctx)
		if err := selector.Err(); err != nil {
			return nil, err
		}
		step := sqlgraph.NewStep(
			sqlgraph.From(script.Table, script.FieldID, selector),
			sqlgraph.To(environment.Table, environment.FieldID),
			sqlgraph.Edge(sqlgraph.M2O, true, script.ScriptToEnvironmentTable, script.ScriptToEnvironmentColumn),
		)
		fromU = sqlgraph.SetNeighbors(sq.driver.Dialect(), step)
		return fromU, nil
	}
	return query
}

// QueryScriptToValidation chains the current query on the "ScriptToValidation" edge.
func (sq *ScriptQuery) QueryScriptToValidation() *ValidationQuery {
	query := &ValidationQuery{config: sq.config}
	query.path = func(ctx context.Context) (fromU *sql.Selector, err error) {
		if err := sq.prepareQuery(ctx); err != nil {
			return nil, err
		}
		selector := sq.sqlQuery(ctx)
		if err := selector.Err(); err != nil {
			return nil, err
		}
		step := sqlgraph.NewStep(
			sqlgraph.From(script.Table, script.FieldID, selector),
			sqlgraph.To(validation.Table, validation.FieldID),
			sqlgraph.Edge(sqlgraph.M2O, false, script.ScriptToValidationTable, script.ScriptToValidationColumn),
		)
		fromU = sqlgraph.SetNeighbors(sq.driver.Dialect(), step)
		return fromU, nil
	}
	return query
}

// First returns the first Script entity from the query.
// Returns a *NotFoundError when no Script was found.
func (sq *ScriptQuery) First(ctx context.Context) (*Script, error) {
	nodes, err := sq.Limit(1).All(ctx)
	if err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nil, &NotFoundError{script.Label}
	}
	return nodes[0], nil
}

// FirstX is like First, but panics if an error occurs.
func (sq *ScriptQuery) FirstX(ctx context.Context) *Script {
	node, err := sq.First(ctx)
	if err != nil && !IsNotFound(err) {
		panic(err)
	}
	return node
}

// FirstID returns the first Script ID from the query.
// Returns a *NotFoundError when no Script ID was found.
func (sq *ScriptQuery) FirstID(ctx context.Context) (id uuid.UUID, err error) {
	var ids []uuid.UUID
	if ids, err = sq.Limit(1).IDs(ctx); err != nil {
		return
	}
	if len(ids) == 0 {
		err = &NotFoundError{script.Label}
		return
	}
	return ids[0], nil
}

// FirstIDX is like FirstID, but panics if an error occurs.
func (sq *ScriptQuery) FirstIDX(ctx context.Context) uuid.UUID {
	id, err := sq.FirstID(ctx)
	if err != nil && !IsNotFound(err) {
		panic(err)
	}
	return id
}

// Only returns a single Script entity found by the query, ensuring it only returns one.
// Returns a *NotSingularError when more than one Script entity is found.
// Returns a *NotFoundError when no Script entities are found.
func (sq *ScriptQuery) Only(ctx context.Context) (*Script, error) {
	nodes, err := sq.Limit(2).All(ctx)
	if err != nil {
		return nil, err
	}
	switch len(nodes) {
	case 1:
		return nodes[0], nil
	case 0:
		return nil, &NotFoundError{script.Label}
	default:
		return nil, &NotSingularError{script.Label}
	}
}

// OnlyX is like Only, but panics if an error occurs.
func (sq *ScriptQuery) OnlyX(ctx context.Context) *Script {
	node, err := sq.Only(ctx)
	if err != nil {
		panic(err)
	}
	return node
}

// OnlyID is like Only, but returns the only Script ID in the query.
// Returns a *NotSingularError when more than one Script ID is found.
// Returns a *NotFoundError when no entities are found.
func (sq *ScriptQuery) OnlyID(ctx context.Context) (id uuid.UUID, err error) {
	var ids []uuid.UUID
	if ids, err = sq.Limit(2).IDs(ctx); err != nil {
		return
	}
	switch len(ids) {
	case 1:
		id = ids[0]
	case 0:
		err = &NotFoundError{script.Label}
	default:
		err = &NotSingularError{script.Label}
	}
	return
}

// OnlyIDX is like OnlyID, but panics if an error occurs.
func (sq *ScriptQuery) OnlyIDX(ctx context.Context) uuid.UUID {
	id, err := sq.OnlyID(ctx)
	if err != nil {
		panic(err)
	}
	return id
}

// All executes the query and returns a list of Scripts.
func (sq *ScriptQuery) All(ctx context.Context) ([]*Script, error) {
	if err := sq.prepareQuery(ctx); err != nil {
		return nil, err
	}
	return sq.sqlAll(ctx)
}

// AllX is like All, but panics if an error occurs.
func (sq *ScriptQuery) AllX(ctx context.Context) []*Script {
	nodes, err := sq.All(ctx)
	if err != nil {
		panic(err)
	}
	return nodes
}

// IDs executes the query and returns a list of Script IDs.
func (sq *ScriptQuery) IDs(ctx context.Context) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	if err := sq.Select(script.FieldID).Scan(ctx, &ids); err != nil {
		return nil, err
	}
	return ids, nil
}

// IDsX is like IDs, but panics if an error occurs.
func (sq *ScriptQuery) IDsX(ctx context.Context) []uuid.UUID {
	ids, err := sq.IDs(ctx)
	if err != nil {
		panic(err)
	}
	return ids
}

// Count returns the count of the given query.
func (sq *ScriptQuery) Count(ctx context.Context) (int, error) {
	if err := sq.prepareQuery(ctx); err != nil {
		return 0, err
	}
	return sq.sqlCount(ctx)
}

// CountX is like Count, but panics if an error occurs.
func (sq *ScriptQuery) CountX(ctx context.Context) int {
	count, err := sq.Count(ctx)
	if err != nil {
		panic(err)
	}
	return count
}

// Exist returns true if the query has elements in the graph.
func (sq *ScriptQuery) Exist(ctx context.Context) (bool, error) {
	if err := sq.prepareQuery(ctx); err != nil {
		return false, err
	}
	return sq.sqlExist(ctx)
}

// ExistX is like Exist, but panics if an error occurs.
func (sq *ScriptQuery) ExistX(ctx context.Context) bool {
	exist, err := sq.Exist(ctx)
	if err != nil {
		panic(err)
	}
	return exist
}

// Clone returns a duplicate of the ScriptQuery builder, including all associated steps. It can be
// used to prepare common query builders and use them differently after the clone is made.
func (sq *ScriptQuery) Clone() *ScriptQuery {
	if sq == nil {
		return nil
	}
	return &ScriptQuery{
		config:                  sq.config,
		limit:                   sq.limit,
		offset:                  sq.offset,
		order:                   append([]OrderFunc{}, sq.order...),
		predicates:              append([]predicate.Script{}, sq.predicates...),
		withScriptToUser:        sq.withScriptToUser.Clone(),
		withScriptToFinding:     sq.withScriptToFinding.Clone(),
		withScriptToEnvironment: sq.withScriptToEnvironment.Clone(),
		withScriptToValidation:  sq.withScriptToValidation.Clone(),
		// clone intermediate query.
		sql:    sq.sql.Clone(),
		path:   sq.path,
		unique: sq.unique,
	}
}

// WithScriptToUser tells the query-builder to eager-load the nodes that are connected to
// the "ScriptToUser" edge. The optional arguments are used to configure the query builder of the edge.
func (sq *ScriptQuery) WithScriptToUser(opts ...func(*UserQuery)) *ScriptQuery {
	query := &UserQuery{config: sq.config}
	for _, opt := range opts {
		opt(query)
	}
	sq.withScriptToUser = query
	return sq
}

// WithScriptToFinding tells the query-builder to eager-load the nodes that are connected to
// the "ScriptToFinding" edge. The optional arguments are used to configure the query builder of the edge.
func (sq *ScriptQuery) WithScriptToFinding(opts ...func(*FindingQuery)) *ScriptQuery {
	query := &FindingQuery{config: sq.config}
	for _, opt := range opts {
		opt(query)
	}
	sq.withScriptToFinding = query
	return sq
}

// WithScriptToEnvironment tells the query-builder to eager-load the nodes that are connected to
// the "ScriptToEnvironment" edge. The optional arguments are used to configure the query builder of the edge.
func (sq *ScriptQuery) WithScriptToEnvironment(opts ...func(*EnvironmentQuery)) *ScriptQuery {
	query := &EnvironmentQuery{config: sq.config}
	for _, opt := range opts {
		opt(query)
	}
	sq.withScriptToEnvironment = query
	return sq
}

// WithScriptToValidation tells the query-builder to eager-load the nodes that are connected to
// the "ScriptToValidation" edge. The optional arguments are used to configure the query builder of the edge.
func (sq *ScriptQuery) WithScriptToValidation(opts ...func(*ValidationQuery)) *ScriptQuery {
	query := &ValidationQuery{config: sq.config}
	for _, opt := range opts {
		opt(query)
	}
	sq.withScriptToValidation = query
	return sq
}

// GroupBy is used to group vertices by one or more fields/columns.
// It is often used with aggregate functions, like: count, max, mean, min, sum.
//
// Example:
//
//	var v []struct {
//		HclID string `json:"hcl_id,omitempty" hcl:"id,label"`
//		Count int `json:"count,omitempty"`
//	}
//
//	client.Script.Query().
//		GroupBy(script.FieldHclID).
//		Aggregate(ent.Count()).
//		Scan(ctx, &v)
func (sq *ScriptQuery) GroupBy(field string, fields ...string) *ScriptGroupBy {
	grbuild := &ScriptGroupBy{config: sq.config}
	grbuild.fields = append([]string{field}, fields...)
	grbuild.path = func(ctx context.Context) (prev *sql.Selector, err error) {
		if err := sq.prepareQuery(ctx); err != nil {
			return nil, err
		}
		return sq.sqlQuery(ctx), nil
	}
	grbuild.label = script.Label
	grbuild.flds, grbuild.scan = &grbuild.fields, grbuild.Scan
	return grbuild
}

// Select allows the selection one or more fields/columns for the given query,
// instead of selecting all fields in the entity.
//
// Example:
//
//	var v []struct {
//		HclID string `json:"hcl_id,omitempty" hcl:"id,label"`
//	}
//
//	client.Script.Query().
//		Select(script.FieldHclID).
//		Scan(ctx, &v)
func (sq *ScriptQuery) Select(fields ...string) *ScriptSelect {
	sq.fields = append(sq.fields, fields...)
	selbuild := &ScriptSelect{ScriptQuery: sq}
	selbuild.label = script.Label
	selbuild.flds, selbuild.scan = &sq.fields, selbuild.Scan
	return selbuild
}

func (sq *ScriptQuery) prepareQuery(ctx context.Context) error {
	for _, f := range sq.fields {
		if !script.ValidColumn(f) {
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

func (sq *ScriptQuery) sqlAll(ctx context.Context, hooks ...queryHook) ([]*Script, error) {
	var (
		nodes       = []*Script{}
		withFKs     = sq.withFKs
		_spec       = sq.querySpec()
		loadedTypes = [4]bool{
			sq.withScriptToUser != nil,
			sq.withScriptToFinding != nil,
			sq.withScriptToEnvironment != nil,
			sq.withScriptToValidation != nil,
		}
	)
	if sq.withScriptToEnvironment != nil || sq.withScriptToValidation != nil {
		withFKs = true
	}
	if withFKs {
		_spec.Node.Columns = append(_spec.Node.Columns, script.ForeignKeys...)
	}
	_spec.ScanValues = func(columns []string) ([]interface{}, error) {
		return (*Script).scanValues(nil, columns)
	}
	_spec.Assign = func(columns []string, values []interface{}) error {
		node := &Script{config: sq.config}
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
	if query := sq.withScriptToUser; query != nil {
		if err := sq.loadScriptToUser(ctx, query, nodes,
			func(n *Script) { n.Edges.ScriptToUser = []*User{} },
			func(n *Script, e *User) { n.Edges.ScriptToUser = append(n.Edges.ScriptToUser, e) }); err != nil {
			return nil, err
		}
	}
	if query := sq.withScriptToFinding; query != nil {
		if err := sq.loadScriptToFinding(ctx, query, nodes,
			func(n *Script) { n.Edges.ScriptToFinding = []*Finding{} },
			func(n *Script, e *Finding) { n.Edges.ScriptToFinding = append(n.Edges.ScriptToFinding, e) }); err != nil {
			return nil, err
		}
	}
	if query := sq.withScriptToEnvironment; query != nil {
		if err := sq.loadScriptToEnvironment(ctx, query, nodes, nil,
			func(n *Script, e *Environment) { n.Edges.ScriptToEnvironment = e }); err != nil {
			return nil, err
		}
	}
	if query := sq.withScriptToValidation; query != nil {
		if err := sq.loadScriptToValidation(ctx, query, nodes, nil,
			func(n *Script, e *Validation) { n.Edges.ScriptToValidation = e }); err != nil {
			return nil, err
		}
	}
	return nodes, nil
}

func (sq *ScriptQuery) loadScriptToUser(ctx context.Context, query *UserQuery, nodes []*Script, init func(*Script), assign func(*Script, *User)) error {
	fks := make([]driver.Value, 0, len(nodes))
	nodeids := make(map[uuid.UUID]*Script)
	for i := range nodes {
		fks = append(fks, nodes[i].ID)
		nodeids[nodes[i].ID] = nodes[i]
		if init != nil {
			init(nodes[i])
		}
	}
	query.withFKs = true
	query.Where(predicate.User(func(s *sql.Selector) {
		s.Where(sql.InValues(script.ScriptToUserColumn, fks...))
	}))
	neighbors, err := query.All(ctx)
	if err != nil {
		return err
	}
	for _, n := range neighbors {
		fk := n.script_script_to_user
		if fk == nil {
			return fmt.Errorf(`foreign-key "script_script_to_user" is nil for node %v`, n.ID)
		}
		node, ok := nodeids[*fk]
		if !ok {
			return fmt.Errorf(`unexpected foreign-key "script_script_to_user" returned %v for node %v`, *fk, n.ID)
		}
		assign(node, n)
	}
	return nil
}
func (sq *ScriptQuery) loadScriptToFinding(ctx context.Context, query *FindingQuery, nodes []*Script, init func(*Script), assign func(*Script, *Finding)) error {
	fks := make([]driver.Value, 0, len(nodes))
	nodeids := make(map[uuid.UUID]*Script)
	for i := range nodes {
		fks = append(fks, nodes[i].ID)
		nodeids[nodes[i].ID] = nodes[i]
		if init != nil {
			init(nodes[i])
		}
	}
	query.withFKs = true
	query.Where(predicate.Finding(func(s *sql.Selector) {
		s.Where(sql.InValues(script.ScriptToFindingColumn, fks...))
	}))
	neighbors, err := query.All(ctx)
	if err != nil {
		return err
	}
	for _, n := range neighbors {
		fk := n.script_script_to_finding
		if fk == nil {
			return fmt.Errorf(`foreign-key "script_script_to_finding" is nil for node %v`, n.ID)
		}
		node, ok := nodeids[*fk]
		if !ok {
			return fmt.Errorf(`unexpected foreign-key "script_script_to_finding" returned %v for node %v`, *fk, n.ID)
		}
		assign(node, n)
	}
	return nil
}
func (sq *ScriptQuery) loadScriptToEnvironment(ctx context.Context, query *EnvironmentQuery, nodes []*Script, init func(*Script), assign func(*Script, *Environment)) error {
	ids := make([]uuid.UUID, 0, len(nodes))
	nodeids := make(map[uuid.UUID][]*Script)
	for i := range nodes {
		if nodes[i].environment_environment_to_script == nil {
			continue
		}
		fk := *nodes[i].environment_environment_to_script
		if _, ok := nodeids[fk]; !ok {
			ids = append(ids, fk)
		}
		nodeids[fk] = append(nodeids[fk], nodes[i])
	}
	query.Where(environment.IDIn(ids...))
	neighbors, err := query.All(ctx)
	if err != nil {
		return err
	}
	for _, n := range neighbors {
		nodes, ok := nodeids[n.ID]
		if !ok {
			return fmt.Errorf(`unexpected foreign-key "environment_environment_to_script" returned %v`, n.ID)
		}
		for i := range nodes {
			assign(nodes[i], n)
		}
	}
	return nil
}
func (sq *ScriptQuery) loadScriptToValidation(ctx context.Context, query *ValidationQuery, nodes []*Script, init func(*Script), assign func(*Script, *Validation)) error {
	ids := make([]uuid.UUID, 0, len(nodes))
	nodeids := make(map[uuid.UUID][]*Script)
	for i := range nodes {
		if nodes[i].script_script_to_validation == nil {
			continue
		}
		fk := *nodes[i].script_script_to_validation
		if _, ok := nodeids[fk]; !ok {
			ids = append(ids, fk)
		}
		nodeids[fk] = append(nodeids[fk], nodes[i])
	}
	query.Where(validation.IDIn(ids...))
	neighbors, err := query.All(ctx)
	if err != nil {
		return err
	}
	for _, n := range neighbors {
		nodes, ok := nodeids[n.ID]
		if !ok {
			return fmt.Errorf(`unexpected foreign-key "script_script_to_validation" returned %v`, n.ID)
		}
		for i := range nodes {
			assign(nodes[i], n)
		}
	}
	return nil
}

func (sq *ScriptQuery) sqlCount(ctx context.Context) (int, error) {
	_spec := sq.querySpec()
	_spec.Node.Columns = sq.fields
	if len(sq.fields) > 0 {
		_spec.Unique = sq.unique != nil && *sq.unique
	}
	return sqlgraph.CountNodes(ctx, sq.driver, _spec)
}

func (sq *ScriptQuery) sqlExist(ctx context.Context) (bool, error) {
	n, err := sq.sqlCount(ctx)
	if err != nil {
		return false, fmt.Errorf("ent: check existence: %w", err)
	}
	return n > 0, nil
}

func (sq *ScriptQuery) querySpec() *sqlgraph.QuerySpec {
	_spec := &sqlgraph.QuerySpec{
		Node: &sqlgraph.NodeSpec{
			Table:   script.Table,
			Columns: script.Columns,
			ID: &sqlgraph.FieldSpec{
				Type:   field.TypeUUID,
				Column: script.FieldID,
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
		_spec.Node.Columns = append(_spec.Node.Columns, script.FieldID)
		for i := range fields {
			if fields[i] != script.FieldID {
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

func (sq *ScriptQuery) sqlQuery(ctx context.Context) *sql.Selector {
	builder := sql.Dialect(sq.driver.Dialect())
	t1 := builder.Table(script.Table)
	columns := sq.fields
	if len(columns) == 0 {
		columns = script.Columns
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

// ScriptGroupBy is the group-by builder for Script entities.
type ScriptGroupBy struct {
	config
	selector
	fields []string
	fns    []AggregateFunc
	// intermediate query (i.e. traversal path).
	sql  *sql.Selector
	path func(context.Context) (*sql.Selector, error)
}

// Aggregate adds the given aggregation functions to the group-by query.
func (sgb *ScriptGroupBy) Aggregate(fns ...AggregateFunc) *ScriptGroupBy {
	sgb.fns = append(sgb.fns, fns...)
	return sgb
}

// Scan applies the group-by query and scans the result into the given value.
func (sgb *ScriptGroupBy) Scan(ctx context.Context, v interface{}) error {
	query, err := sgb.path(ctx)
	if err != nil {
		return err
	}
	sgb.sql = query
	return sgb.sqlScan(ctx, v)
}

func (sgb *ScriptGroupBy) sqlScan(ctx context.Context, v interface{}) error {
	for _, f := range sgb.fields {
		if !script.ValidColumn(f) {
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

func (sgb *ScriptGroupBy) sqlQuery() *sql.Selector {
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

// ScriptSelect is the builder for selecting fields of Script entities.
type ScriptSelect struct {
	*ScriptQuery
	selector
	// intermediate query (i.e. traversal path).
	sql *sql.Selector
}

// Scan applies the selector query and scans the result into the given value.
func (ss *ScriptSelect) Scan(ctx context.Context, v interface{}) error {
	if err := ss.prepareQuery(ctx); err != nil {
		return err
	}
	ss.sql = ss.ScriptQuery.sqlQuery(ctx)
	return ss.sqlScan(ctx, v)
}

func (ss *ScriptSelect) sqlScan(ctx context.Context, v interface{}) error {
	rows := &sql.Rows{}
	query, args := ss.sql.Query()
	if err := ss.driver.Query(ctx, query, args, rows); err != nil {
		return err
	}
	defer rows.Close()
	return sql.ScanSlice(rows, v)
}
