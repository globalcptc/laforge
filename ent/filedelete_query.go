// Code generated by entc, DO NOT EDIT.

package ent

import (
	"context"
	"errors"
	"fmt"
	"math"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/schema/field"
	"github.com/gen0cide/laforge/ent/environment"
	"github.com/gen0cide/laforge/ent/filedelete"
	"github.com/gen0cide/laforge/ent/predicate"
	"github.com/google/uuid"
)

// FileDeleteQuery is the builder for querying FileDelete entities.
type FileDeleteQuery struct {
	config
	limit      *int
	offset     *int
	unique     *bool
	order      []OrderFunc
	fields     []string
	predicates []predicate.FileDelete
	// eager-loading edges.
	withFileDeleteToEnvironment *EnvironmentQuery
	withFKs                     bool
	// intermediate query (i.e. traversal path).
	sql  *sql.Selector
	path func(context.Context) (*sql.Selector, error)
}

// Where adds a new predicate for the FileDeleteQuery builder.
func (fdq *FileDeleteQuery) Where(ps ...predicate.FileDelete) *FileDeleteQuery {
	fdq.predicates = append(fdq.predicates, ps...)
	return fdq
}

// Limit adds a limit step to the query.
func (fdq *FileDeleteQuery) Limit(limit int) *FileDeleteQuery {
	fdq.limit = &limit
	return fdq
}

// Offset adds an offset step to the query.
func (fdq *FileDeleteQuery) Offset(offset int) *FileDeleteQuery {
	fdq.offset = &offset
	return fdq
}

// Unique configures the query builder to filter duplicate records on query.
// By default, unique is set to true, and can be disabled using this method.
func (fdq *FileDeleteQuery) Unique(unique bool) *FileDeleteQuery {
	fdq.unique = &unique
	return fdq
}

// Order adds an order step to the query.
func (fdq *FileDeleteQuery) Order(o ...OrderFunc) *FileDeleteQuery {
	fdq.order = append(fdq.order, o...)
	return fdq
}

// QueryFileDeleteToEnvironment chains the current query on the "FileDeleteToEnvironment" edge.
func (fdq *FileDeleteQuery) QueryFileDeleteToEnvironment() *EnvironmentQuery {
	query := &EnvironmentQuery{config: fdq.config}
	query.path = func(ctx context.Context) (fromU *sql.Selector, err error) {
		if err := fdq.prepareQuery(ctx); err != nil {
			return nil, err
		}
		selector := fdq.sqlQuery(ctx)
		if err := selector.Err(); err != nil {
			return nil, err
		}
		step := sqlgraph.NewStep(
			sqlgraph.From(filedelete.Table, filedelete.FieldID, selector),
			sqlgraph.To(environment.Table, environment.FieldID),
			sqlgraph.Edge(sqlgraph.M2O, true, filedelete.FileDeleteToEnvironmentTable, filedelete.FileDeleteToEnvironmentColumn),
		)
		fromU = sqlgraph.SetNeighbors(fdq.driver.Dialect(), step)
		return fromU, nil
	}
	return query
}

// First returns the first FileDelete entity from the query.
// Returns a *NotFoundError when no FileDelete was found.
func (fdq *FileDeleteQuery) First(ctx context.Context) (*FileDelete, error) {
	nodes, err := fdq.Limit(1).All(ctx)
	if err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nil, &NotFoundError{filedelete.Label}
	}
	return nodes[0], nil
}

// FirstX is like First, but panics if an error occurs.
func (fdq *FileDeleteQuery) FirstX(ctx context.Context) *FileDelete {
	node, err := fdq.First(ctx)
	if err != nil && !IsNotFound(err) {
		panic(err)
	}
	return node
}

// FirstID returns the first FileDelete ID from the query.
// Returns a *NotFoundError when no FileDelete ID was found.
func (fdq *FileDeleteQuery) FirstID(ctx context.Context) (id uuid.UUID, err error) {
	var ids []uuid.UUID
	if ids, err = fdq.Limit(1).IDs(ctx); err != nil {
		return
	}
	if len(ids) == 0 {
		err = &NotFoundError{filedelete.Label}
		return
	}
	return ids[0], nil
}

// FirstIDX is like FirstID, but panics if an error occurs.
func (fdq *FileDeleteQuery) FirstIDX(ctx context.Context) uuid.UUID {
	id, err := fdq.FirstID(ctx)
	if err != nil && !IsNotFound(err) {
		panic(err)
	}
	return id
}

// Only returns a single FileDelete entity found by the query, ensuring it only returns one.
// Returns a *NotSingularError when exactly one FileDelete entity is not found.
// Returns a *NotFoundError when no FileDelete entities are found.
func (fdq *FileDeleteQuery) Only(ctx context.Context) (*FileDelete, error) {
	nodes, err := fdq.Limit(2).All(ctx)
	if err != nil {
		return nil, err
	}
	switch len(nodes) {
	case 1:
		return nodes[0], nil
	case 0:
		return nil, &NotFoundError{filedelete.Label}
	default:
		return nil, &NotSingularError{filedelete.Label}
	}
}

// OnlyX is like Only, but panics if an error occurs.
func (fdq *FileDeleteQuery) OnlyX(ctx context.Context) *FileDelete {
	node, err := fdq.Only(ctx)
	if err != nil {
		panic(err)
	}
	return node
}

// OnlyID is like Only, but returns the only FileDelete ID in the query.
// Returns a *NotSingularError when exactly one FileDelete ID is not found.
// Returns a *NotFoundError when no entities are found.
func (fdq *FileDeleteQuery) OnlyID(ctx context.Context) (id uuid.UUID, err error) {
	var ids []uuid.UUID
	if ids, err = fdq.Limit(2).IDs(ctx); err != nil {
		return
	}
	switch len(ids) {
	case 1:
		id = ids[0]
	case 0:
		err = &NotFoundError{filedelete.Label}
	default:
		err = &NotSingularError{filedelete.Label}
	}
	return
}

// OnlyIDX is like OnlyID, but panics if an error occurs.
func (fdq *FileDeleteQuery) OnlyIDX(ctx context.Context) uuid.UUID {
	id, err := fdq.OnlyID(ctx)
	if err != nil {
		panic(err)
	}
	return id
}

// All executes the query and returns a list of FileDeletes.
func (fdq *FileDeleteQuery) All(ctx context.Context) ([]*FileDelete, error) {
	if err := fdq.prepareQuery(ctx); err != nil {
		return nil, err
	}
	return fdq.sqlAll(ctx)
}

// AllX is like All, but panics if an error occurs.
func (fdq *FileDeleteQuery) AllX(ctx context.Context) []*FileDelete {
	nodes, err := fdq.All(ctx)
	if err != nil {
		panic(err)
	}
	return nodes
}

// IDs executes the query and returns a list of FileDelete IDs.
func (fdq *FileDeleteQuery) IDs(ctx context.Context) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	if err := fdq.Select(filedelete.FieldID).Scan(ctx, &ids); err != nil {
		return nil, err
	}
	return ids, nil
}

// IDsX is like IDs, but panics if an error occurs.
func (fdq *FileDeleteQuery) IDsX(ctx context.Context) []uuid.UUID {
	ids, err := fdq.IDs(ctx)
	if err != nil {
		panic(err)
	}
	return ids
}

// Count returns the count of the given query.
func (fdq *FileDeleteQuery) Count(ctx context.Context) (int, error) {
	if err := fdq.prepareQuery(ctx); err != nil {
		return 0, err
	}
	return fdq.sqlCount(ctx)
}

// CountX is like Count, but panics if an error occurs.
func (fdq *FileDeleteQuery) CountX(ctx context.Context) int {
	count, err := fdq.Count(ctx)
	if err != nil {
		panic(err)
	}
	return count
}

// Exist returns true if the query has elements in the graph.
func (fdq *FileDeleteQuery) Exist(ctx context.Context) (bool, error) {
	if err := fdq.prepareQuery(ctx); err != nil {
		return false, err
	}
	return fdq.sqlExist(ctx)
}

// ExistX is like Exist, but panics if an error occurs.
func (fdq *FileDeleteQuery) ExistX(ctx context.Context) bool {
	exist, err := fdq.Exist(ctx)
	if err != nil {
		panic(err)
	}
	return exist
}

// Clone returns a duplicate of the FileDeleteQuery builder, including all associated steps. It can be
// used to prepare common query builders and use them differently after the clone is made.
func (fdq *FileDeleteQuery) Clone() *FileDeleteQuery {
	if fdq == nil {
		return nil
	}
	return &FileDeleteQuery{
		config:                      fdq.config,
		limit:                       fdq.limit,
		offset:                      fdq.offset,
		order:                       append([]OrderFunc{}, fdq.order...),
		predicates:                  append([]predicate.FileDelete{}, fdq.predicates...),
		withFileDeleteToEnvironment: fdq.withFileDeleteToEnvironment.Clone(),
		// clone intermediate query.
		sql:  fdq.sql.Clone(),
		path: fdq.path,
	}
}

// WithFileDeleteToEnvironment tells the query-builder to eager-load the nodes that are connected to
// the "FileDeleteToEnvironment" edge. The optional arguments are used to configure the query builder of the edge.
func (fdq *FileDeleteQuery) WithFileDeleteToEnvironment(opts ...func(*EnvironmentQuery)) *FileDeleteQuery {
	query := &EnvironmentQuery{config: fdq.config}
	for _, opt := range opts {
		opt(query)
	}
	fdq.withFileDeleteToEnvironment = query
	return fdq
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
//	client.FileDelete.Query().
//		GroupBy(filedelete.FieldHclID).
//		Aggregate(ent.Count()).
//		Scan(ctx, &v)
//
func (fdq *FileDeleteQuery) GroupBy(field string, fields ...string) *FileDeleteGroupBy {
	group := &FileDeleteGroupBy{config: fdq.config}
	group.fields = append([]string{field}, fields...)
	group.path = func(ctx context.Context) (prev *sql.Selector, err error) {
		if err := fdq.prepareQuery(ctx); err != nil {
			return nil, err
		}
		return fdq.sqlQuery(ctx), nil
	}
	return group
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
//	client.FileDelete.Query().
//		Select(filedelete.FieldHclID).
//		Scan(ctx, &v)
//
func (fdq *FileDeleteQuery) Select(field string, fields ...string) *FileDeleteSelect {
	fdq.fields = append([]string{field}, fields...)
	return &FileDeleteSelect{FileDeleteQuery: fdq}
}

func (fdq *FileDeleteQuery) prepareQuery(ctx context.Context) error {
	for _, f := range fdq.fields {
		if !filedelete.ValidColumn(f) {
			return &ValidationError{Name: f, err: fmt.Errorf("ent: invalid field %q for query", f)}
		}
	}
	if fdq.path != nil {
		prev, err := fdq.path(ctx)
		if err != nil {
			return err
		}
		fdq.sql = prev
	}
	return nil
}

func (fdq *FileDeleteQuery) sqlAll(ctx context.Context) ([]*FileDelete, error) {
	var (
		nodes       = []*FileDelete{}
		withFKs     = fdq.withFKs
		_spec       = fdq.querySpec()
		loadedTypes = [1]bool{
			fdq.withFileDeleteToEnvironment != nil,
		}
	)
	if fdq.withFileDeleteToEnvironment != nil {
		withFKs = true
	}
	if withFKs {
		_spec.Node.Columns = append(_spec.Node.Columns, filedelete.ForeignKeys...)
	}
	_spec.ScanValues = func(columns []string) ([]interface{}, error) {
		node := &FileDelete{config: fdq.config}
		nodes = append(nodes, node)
		return node.scanValues(columns)
	}
	_spec.Assign = func(columns []string, values []interface{}) error {
		if len(nodes) == 0 {
			return fmt.Errorf("ent: Assign called without calling ScanValues")
		}
		node := nodes[len(nodes)-1]
		node.Edges.loadedTypes = loadedTypes
		return node.assignValues(columns, values)
	}
	if err := sqlgraph.QueryNodes(ctx, fdq.driver, _spec); err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nodes, nil
	}

	if query := fdq.withFileDeleteToEnvironment; query != nil {
		ids := make([]uuid.UUID, 0, len(nodes))
		nodeids := make(map[uuid.UUID][]*FileDelete)
		for i := range nodes {
			if nodes[i].environment_environment_to_file_delete == nil {
				continue
			}
			fk := *nodes[i].environment_environment_to_file_delete
			if _, ok := nodeids[fk]; !ok {
				ids = append(ids, fk)
			}
			nodeids[fk] = append(nodeids[fk], nodes[i])
		}
		query.Where(environment.IDIn(ids...))
		neighbors, err := query.All(ctx)
		if err != nil {
			return nil, err
		}
		for _, n := range neighbors {
			nodes, ok := nodeids[n.ID]
			if !ok {
				return nil, fmt.Errorf(`unexpected foreign-key "environment_environment_to_file_delete" returned %v`, n.ID)
			}
			for i := range nodes {
				nodes[i].Edges.FileDeleteToEnvironment = n
			}
		}
	}

	return nodes, nil
}

func (fdq *FileDeleteQuery) sqlCount(ctx context.Context) (int, error) {
	_spec := fdq.querySpec()
	return sqlgraph.CountNodes(ctx, fdq.driver, _spec)
}

func (fdq *FileDeleteQuery) sqlExist(ctx context.Context) (bool, error) {
	n, err := fdq.sqlCount(ctx)
	if err != nil {
		return false, fmt.Errorf("ent: check existence: %w", err)
	}
	return n > 0, nil
}

func (fdq *FileDeleteQuery) querySpec() *sqlgraph.QuerySpec {
	_spec := &sqlgraph.QuerySpec{
		Node: &sqlgraph.NodeSpec{
			Table:   filedelete.Table,
			Columns: filedelete.Columns,
			ID: &sqlgraph.FieldSpec{
				Type:   field.TypeUUID,
				Column: filedelete.FieldID,
			},
		},
		From:   fdq.sql,
		Unique: true,
	}
	if unique := fdq.unique; unique != nil {
		_spec.Unique = *unique
	}
	if fields := fdq.fields; len(fields) > 0 {
		_spec.Node.Columns = make([]string, 0, len(fields))
		_spec.Node.Columns = append(_spec.Node.Columns, filedelete.FieldID)
		for i := range fields {
			if fields[i] != filedelete.FieldID {
				_spec.Node.Columns = append(_spec.Node.Columns, fields[i])
			}
		}
	}
	if ps := fdq.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if limit := fdq.limit; limit != nil {
		_spec.Limit = *limit
	}
	if offset := fdq.offset; offset != nil {
		_spec.Offset = *offset
	}
	if ps := fdq.order; len(ps) > 0 {
		_spec.Order = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	return _spec
}

func (fdq *FileDeleteQuery) sqlQuery(ctx context.Context) *sql.Selector {
	builder := sql.Dialect(fdq.driver.Dialect())
	t1 := builder.Table(filedelete.Table)
	selector := builder.Select(t1.Columns(filedelete.Columns...)...).From(t1)
	if fdq.sql != nil {
		selector = fdq.sql
		selector.Select(selector.Columns(filedelete.Columns...)...)
	}
	for _, p := range fdq.predicates {
		p(selector)
	}
	for _, p := range fdq.order {
		p(selector)
	}
	if offset := fdq.offset; offset != nil {
		// limit is mandatory for offset clause. We start
		// with default value, and override it below if needed.
		selector.Offset(*offset).Limit(math.MaxInt32)
	}
	if limit := fdq.limit; limit != nil {
		selector.Limit(*limit)
	}
	return selector
}

// FileDeleteGroupBy is the group-by builder for FileDelete entities.
type FileDeleteGroupBy struct {
	config
	fields []string
	fns    []AggregateFunc
	// intermediate query (i.e. traversal path).
	sql  *sql.Selector
	path func(context.Context) (*sql.Selector, error)
}

// Aggregate adds the given aggregation functions to the group-by query.
func (fdgb *FileDeleteGroupBy) Aggregate(fns ...AggregateFunc) *FileDeleteGroupBy {
	fdgb.fns = append(fdgb.fns, fns...)
	return fdgb
}

// Scan applies the group-by query and scans the result into the given value.
func (fdgb *FileDeleteGroupBy) Scan(ctx context.Context, v interface{}) error {
	query, err := fdgb.path(ctx)
	if err != nil {
		return err
	}
	fdgb.sql = query
	return fdgb.sqlScan(ctx, v)
}

// ScanX is like Scan, but panics if an error occurs.
func (fdgb *FileDeleteGroupBy) ScanX(ctx context.Context, v interface{}) {
	if err := fdgb.Scan(ctx, v); err != nil {
		panic(err)
	}
}

// Strings returns list of strings from group-by.
// It is only allowed when executing a group-by query with one field.
func (fdgb *FileDeleteGroupBy) Strings(ctx context.Context) ([]string, error) {
	if len(fdgb.fields) > 1 {
		return nil, errors.New("ent: FileDeleteGroupBy.Strings is not achievable when grouping more than 1 field")
	}
	var v []string
	if err := fdgb.Scan(ctx, &v); err != nil {
		return nil, err
	}
	return v, nil
}

// StringsX is like Strings, but panics if an error occurs.
func (fdgb *FileDeleteGroupBy) StringsX(ctx context.Context) []string {
	v, err := fdgb.Strings(ctx)
	if err != nil {
		panic(err)
	}
	return v
}

// String returns a single string from a group-by query.
// It is only allowed when executing a group-by query with one field.
func (fdgb *FileDeleteGroupBy) String(ctx context.Context) (_ string, err error) {
	var v []string
	if v, err = fdgb.Strings(ctx); err != nil {
		return
	}
	switch len(v) {
	case 1:
		return v[0], nil
	case 0:
		err = &NotFoundError{filedelete.Label}
	default:
		err = fmt.Errorf("ent: FileDeleteGroupBy.Strings returned %d results when one was expected", len(v))
	}
	return
}

// StringX is like String, but panics if an error occurs.
func (fdgb *FileDeleteGroupBy) StringX(ctx context.Context) string {
	v, err := fdgb.String(ctx)
	if err != nil {
		panic(err)
	}
	return v
}

// Ints returns list of ints from group-by.
// It is only allowed when executing a group-by query with one field.
func (fdgb *FileDeleteGroupBy) Ints(ctx context.Context) ([]int, error) {
	if len(fdgb.fields) > 1 {
		return nil, errors.New("ent: FileDeleteGroupBy.Ints is not achievable when grouping more than 1 field")
	}
	var v []int
	if err := fdgb.Scan(ctx, &v); err != nil {
		return nil, err
	}
	return v, nil
}

// IntsX is like Ints, but panics if an error occurs.
func (fdgb *FileDeleteGroupBy) IntsX(ctx context.Context) []int {
	v, err := fdgb.Ints(ctx)
	if err != nil {
		panic(err)
	}
	return v
}

// Int returns a single int from a group-by query.
// It is only allowed when executing a group-by query with one field.
func (fdgb *FileDeleteGroupBy) Int(ctx context.Context) (_ int, err error) {
	var v []int
	if v, err = fdgb.Ints(ctx); err != nil {
		return
	}
	switch len(v) {
	case 1:
		return v[0], nil
	case 0:
		err = &NotFoundError{filedelete.Label}
	default:
		err = fmt.Errorf("ent: FileDeleteGroupBy.Ints returned %d results when one was expected", len(v))
	}
	return
}

// IntX is like Int, but panics if an error occurs.
func (fdgb *FileDeleteGroupBy) IntX(ctx context.Context) int {
	v, err := fdgb.Int(ctx)
	if err != nil {
		panic(err)
	}
	return v
}

// Float64s returns list of float64s from group-by.
// It is only allowed when executing a group-by query with one field.
func (fdgb *FileDeleteGroupBy) Float64s(ctx context.Context) ([]float64, error) {
	if len(fdgb.fields) > 1 {
		return nil, errors.New("ent: FileDeleteGroupBy.Float64s is not achievable when grouping more than 1 field")
	}
	var v []float64
	if err := fdgb.Scan(ctx, &v); err != nil {
		return nil, err
	}
	return v, nil
}

// Float64sX is like Float64s, but panics if an error occurs.
func (fdgb *FileDeleteGroupBy) Float64sX(ctx context.Context) []float64 {
	v, err := fdgb.Float64s(ctx)
	if err != nil {
		panic(err)
	}
	return v
}

// Float64 returns a single float64 from a group-by query.
// It is only allowed when executing a group-by query with one field.
func (fdgb *FileDeleteGroupBy) Float64(ctx context.Context) (_ float64, err error) {
	var v []float64
	if v, err = fdgb.Float64s(ctx); err != nil {
		return
	}
	switch len(v) {
	case 1:
		return v[0], nil
	case 0:
		err = &NotFoundError{filedelete.Label}
	default:
		err = fmt.Errorf("ent: FileDeleteGroupBy.Float64s returned %d results when one was expected", len(v))
	}
	return
}

// Float64X is like Float64, but panics if an error occurs.
func (fdgb *FileDeleteGroupBy) Float64X(ctx context.Context) float64 {
	v, err := fdgb.Float64(ctx)
	if err != nil {
		panic(err)
	}
	return v
}

// Bools returns list of bools from group-by.
// It is only allowed when executing a group-by query with one field.
func (fdgb *FileDeleteGroupBy) Bools(ctx context.Context) ([]bool, error) {
	if len(fdgb.fields) > 1 {
		return nil, errors.New("ent: FileDeleteGroupBy.Bools is not achievable when grouping more than 1 field")
	}
	var v []bool
	if err := fdgb.Scan(ctx, &v); err != nil {
		return nil, err
	}
	return v, nil
}

// BoolsX is like Bools, but panics if an error occurs.
func (fdgb *FileDeleteGroupBy) BoolsX(ctx context.Context) []bool {
	v, err := fdgb.Bools(ctx)
	if err != nil {
		panic(err)
	}
	return v
}

// Bool returns a single bool from a group-by query.
// It is only allowed when executing a group-by query with one field.
func (fdgb *FileDeleteGroupBy) Bool(ctx context.Context) (_ bool, err error) {
	var v []bool
	if v, err = fdgb.Bools(ctx); err != nil {
		return
	}
	switch len(v) {
	case 1:
		return v[0], nil
	case 0:
		err = &NotFoundError{filedelete.Label}
	default:
		err = fmt.Errorf("ent: FileDeleteGroupBy.Bools returned %d results when one was expected", len(v))
	}
	return
}

// BoolX is like Bool, but panics if an error occurs.
func (fdgb *FileDeleteGroupBy) BoolX(ctx context.Context) bool {
	v, err := fdgb.Bool(ctx)
	if err != nil {
		panic(err)
	}
	return v
}

func (fdgb *FileDeleteGroupBy) sqlScan(ctx context.Context, v interface{}) error {
	for _, f := range fdgb.fields {
		if !filedelete.ValidColumn(f) {
			return &ValidationError{Name: f, err: fmt.Errorf("invalid field %q for group-by", f)}
		}
	}
	selector := fdgb.sqlQuery()
	if err := selector.Err(); err != nil {
		return err
	}
	rows := &sql.Rows{}
	query, args := selector.Query()
	if err := fdgb.driver.Query(ctx, query, args, rows); err != nil {
		return err
	}
	defer rows.Close()
	return sql.ScanSlice(rows, v)
}

func (fdgb *FileDeleteGroupBy) sqlQuery() *sql.Selector {
	selector := fdgb.sql
	columns := make([]string, 0, len(fdgb.fields)+len(fdgb.fns))
	columns = append(columns, fdgb.fields...)
	for _, fn := range fdgb.fns {
		columns = append(columns, fn(selector))
	}
	return selector.Select(columns...).GroupBy(fdgb.fields...)
}

// FileDeleteSelect is the builder for selecting fields of FileDelete entities.
type FileDeleteSelect struct {
	*FileDeleteQuery
	// intermediate query (i.e. traversal path).
	sql *sql.Selector
}

// Scan applies the selector query and scans the result into the given value.
func (fds *FileDeleteSelect) Scan(ctx context.Context, v interface{}) error {
	if err := fds.prepareQuery(ctx); err != nil {
		return err
	}
	fds.sql = fds.FileDeleteQuery.sqlQuery(ctx)
	return fds.sqlScan(ctx, v)
}

// ScanX is like Scan, but panics if an error occurs.
func (fds *FileDeleteSelect) ScanX(ctx context.Context, v interface{}) {
	if err := fds.Scan(ctx, v); err != nil {
		panic(err)
	}
}

// Strings returns list of strings from a selector. It is only allowed when selecting one field.
func (fds *FileDeleteSelect) Strings(ctx context.Context) ([]string, error) {
	if len(fds.fields) > 1 {
		return nil, errors.New("ent: FileDeleteSelect.Strings is not achievable when selecting more than 1 field")
	}
	var v []string
	if err := fds.Scan(ctx, &v); err != nil {
		return nil, err
	}
	return v, nil
}

// StringsX is like Strings, but panics if an error occurs.
func (fds *FileDeleteSelect) StringsX(ctx context.Context) []string {
	v, err := fds.Strings(ctx)
	if err != nil {
		panic(err)
	}
	return v
}

// String returns a single string from a selector. It is only allowed when selecting one field.
func (fds *FileDeleteSelect) String(ctx context.Context) (_ string, err error) {
	var v []string
	if v, err = fds.Strings(ctx); err != nil {
		return
	}
	switch len(v) {
	case 1:
		return v[0], nil
	case 0:
		err = &NotFoundError{filedelete.Label}
	default:
		err = fmt.Errorf("ent: FileDeleteSelect.Strings returned %d results when one was expected", len(v))
	}
	return
}

// StringX is like String, but panics if an error occurs.
func (fds *FileDeleteSelect) StringX(ctx context.Context) string {
	v, err := fds.String(ctx)
	if err != nil {
		panic(err)
	}
	return v
}

// Ints returns list of ints from a selector. It is only allowed when selecting one field.
func (fds *FileDeleteSelect) Ints(ctx context.Context) ([]int, error) {
	if len(fds.fields) > 1 {
		return nil, errors.New("ent: FileDeleteSelect.Ints is not achievable when selecting more than 1 field")
	}
	var v []int
	if err := fds.Scan(ctx, &v); err != nil {
		return nil, err
	}
	return v, nil
}

// IntsX is like Ints, but panics if an error occurs.
func (fds *FileDeleteSelect) IntsX(ctx context.Context) []int {
	v, err := fds.Ints(ctx)
	if err != nil {
		panic(err)
	}
	return v
}

// Int returns a single int from a selector. It is only allowed when selecting one field.
func (fds *FileDeleteSelect) Int(ctx context.Context) (_ int, err error) {
	var v []int
	if v, err = fds.Ints(ctx); err != nil {
		return
	}
	switch len(v) {
	case 1:
		return v[0], nil
	case 0:
		err = &NotFoundError{filedelete.Label}
	default:
		err = fmt.Errorf("ent: FileDeleteSelect.Ints returned %d results when one was expected", len(v))
	}
	return
}

// IntX is like Int, but panics if an error occurs.
func (fds *FileDeleteSelect) IntX(ctx context.Context) int {
	v, err := fds.Int(ctx)
	if err != nil {
		panic(err)
	}
	return v
}

// Float64s returns list of float64s from a selector. It is only allowed when selecting one field.
func (fds *FileDeleteSelect) Float64s(ctx context.Context) ([]float64, error) {
	if len(fds.fields) > 1 {
		return nil, errors.New("ent: FileDeleteSelect.Float64s is not achievable when selecting more than 1 field")
	}
	var v []float64
	if err := fds.Scan(ctx, &v); err != nil {
		return nil, err
	}
	return v, nil
}

// Float64sX is like Float64s, but panics if an error occurs.
func (fds *FileDeleteSelect) Float64sX(ctx context.Context) []float64 {
	v, err := fds.Float64s(ctx)
	if err != nil {
		panic(err)
	}
	return v
}

// Float64 returns a single float64 from a selector. It is only allowed when selecting one field.
func (fds *FileDeleteSelect) Float64(ctx context.Context) (_ float64, err error) {
	var v []float64
	if v, err = fds.Float64s(ctx); err != nil {
		return
	}
	switch len(v) {
	case 1:
		return v[0], nil
	case 0:
		err = &NotFoundError{filedelete.Label}
	default:
		err = fmt.Errorf("ent: FileDeleteSelect.Float64s returned %d results when one was expected", len(v))
	}
	return
}

// Float64X is like Float64, but panics if an error occurs.
func (fds *FileDeleteSelect) Float64X(ctx context.Context) float64 {
	v, err := fds.Float64(ctx)
	if err != nil {
		panic(err)
	}
	return v
}

// Bools returns list of bools from a selector. It is only allowed when selecting one field.
func (fds *FileDeleteSelect) Bools(ctx context.Context) ([]bool, error) {
	if len(fds.fields) > 1 {
		return nil, errors.New("ent: FileDeleteSelect.Bools is not achievable when selecting more than 1 field")
	}
	var v []bool
	if err := fds.Scan(ctx, &v); err != nil {
		return nil, err
	}
	return v, nil
}

// BoolsX is like Bools, but panics if an error occurs.
func (fds *FileDeleteSelect) BoolsX(ctx context.Context) []bool {
	v, err := fds.Bools(ctx)
	if err != nil {
		panic(err)
	}
	return v
}

// Bool returns a single bool from a selector. It is only allowed when selecting one field.
func (fds *FileDeleteSelect) Bool(ctx context.Context) (_ bool, err error) {
	var v []bool
	if v, err = fds.Bools(ctx); err != nil {
		return
	}
	switch len(v) {
	case 1:
		return v[0], nil
	case 0:
		err = &NotFoundError{filedelete.Label}
	default:
		err = fmt.Errorf("ent: FileDeleteSelect.Bools returned %d results when one was expected", len(v))
	}
	return
}

// BoolX is like Bool, but panics if an error occurs.
func (fds *FileDeleteSelect) BoolX(ctx context.Context) bool {
	v, err := fds.Bool(ctx)
	if err != nil {
		panic(err)
	}
	return v
}

func (fds *FileDeleteSelect) sqlScan(ctx context.Context, v interface{}) error {
	rows := &sql.Rows{}
	query, args := fds.sqlQuery().Query()
	if err := fds.driver.Query(ctx, query, args, rows); err != nil {
		return err
	}
	defer rows.Close()
	return sql.ScanSlice(rows, v)
}

func (fds *FileDeleteSelect) sqlQuery() sql.Querier {
	selector := fds.sql
	selector.Select(selector.Columns(fds.fields...)...)
	return selector
}
