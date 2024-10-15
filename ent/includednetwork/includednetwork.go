// Code generated by ent, DO NOT EDIT.

package includednetwork

import (
	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"github.com/google/uuid"
)

const (
	// Label holds the string label denoting the includednetwork type in the database.
	Label = "included_network"
	// FieldID holds the string denoting the id field in the database.
	FieldID = "id"
	// FieldName holds the string denoting the name field in the database.
	FieldName = "name"
	// FieldIncludedHosts holds the string denoting the included_hosts field in the database.
	FieldIncludedHosts = "included_hosts"
	// EdgeTags holds the string denoting the tags edge name in mutations.
	EdgeTags = "Tags"
	// EdgeHosts holds the string denoting the hosts edge name in mutations.
	EdgeHosts = "Hosts"
	// EdgeNetwork holds the string denoting the network edge name in mutations.
	EdgeNetwork = "Network"
	// EdgeEnvironments holds the string denoting the environments edge name in mutations.
	EdgeEnvironments = "Environments"
	// Table holds the table name of the includednetwork in the database.
	Table = "included_networks"
	// TagsTable is the table that holds the Tags relation/edge.
	TagsTable = "tags"
	// TagsInverseTable is the table name for the Tag entity.
	// It exists in this package in order to avoid circular dependency with the "tag" package.
	TagsInverseTable = "tags"
	// TagsColumn is the table column denoting the Tags relation/edge.
	TagsColumn = "included_network_tags"
	// HostsTable is the table that holds the Hosts relation/edge. The primary key declared below.
	HostsTable = "included_network_Hosts"
	// HostsInverseTable is the table name for the Host entity.
	// It exists in this package in order to avoid circular dependency with the "host" package.
	HostsInverseTable = "hosts"
	// NetworkTable is the table that holds the Network relation/edge.
	NetworkTable = "included_networks"
	// NetworkInverseTable is the table name for the Network entity.
	// It exists in this package in order to avoid circular dependency with the "network" package.
	NetworkInverseTable = "networks"
	// NetworkColumn is the table column denoting the Network relation/edge.
	NetworkColumn = "included_network_network"
	// EnvironmentsTable is the table that holds the Environments relation/edge. The primary key declared below.
	EnvironmentsTable = "environment_IncludedNetworks"
	// EnvironmentsInverseTable is the table name for the Environment entity.
	// It exists in this package in order to avoid circular dependency with the "environment" package.
	EnvironmentsInverseTable = "environments"
)

// Columns holds all SQL columns for includednetwork fields.
var Columns = []string{
	FieldID,
	FieldName,
	FieldIncludedHosts,
}

// ForeignKeys holds the SQL foreign-keys that are owned by the "included_networks"
// table and are not defined as standalone fields in the schema.
var ForeignKeys = []string{
	"included_network_network",
}

var (
	// HostsPrimaryKey and HostsColumn2 are the table columns denoting the
	// primary key for the Hosts relation (M2M).
	HostsPrimaryKey = []string{"included_network_id", "host_id"}
	// EnvironmentsPrimaryKey and EnvironmentsColumn2 are the table columns denoting the
	// primary key for the Environments relation (M2M).
	EnvironmentsPrimaryKey = []string{"environment_id", "included_network_id"}
)

// ValidColumn reports if the column name is valid (part of the table columns).
func ValidColumn(column string) bool {
	for i := range Columns {
		if column == Columns[i] {
			return true
		}
	}
	for i := range ForeignKeys {
		if column == ForeignKeys[i] {
			return true
		}
	}
	return false
}

var (
	// DefaultID holds the default value on creation for the "id" field.
	DefaultID func() uuid.UUID
)

// OrderOption defines the ordering options for the IncludedNetwork queries.
type OrderOption func(*sql.Selector)

// ByID orders the results by the id field.
func ByID(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldID, opts...).ToFunc()
}

// ByName orders the results by the name field.
func ByName(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldName, opts...).ToFunc()
}

// ByTagsCount orders the results by Tags count.
func ByTagsCount(opts ...sql.OrderTermOption) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborsCount(s, newTagsStep(), opts...)
	}
}

// ByTags orders the results by Tags terms.
func ByTags(term sql.OrderTerm, terms ...sql.OrderTerm) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborTerms(s, newTagsStep(), append([]sql.OrderTerm{term}, terms...)...)
	}
}

// ByHostsCount orders the results by Hosts count.
func ByHostsCount(opts ...sql.OrderTermOption) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborsCount(s, newHostsStep(), opts...)
	}
}

// ByHosts orders the results by Hosts terms.
func ByHosts(term sql.OrderTerm, terms ...sql.OrderTerm) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborTerms(s, newHostsStep(), append([]sql.OrderTerm{term}, terms...)...)
	}
}

// ByNetworkField orders the results by Network field.
func ByNetworkField(field string, opts ...sql.OrderTermOption) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborTerms(s, newNetworkStep(), sql.OrderByField(field, opts...))
	}
}

// ByEnvironmentsCount orders the results by Environments count.
func ByEnvironmentsCount(opts ...sql.OrderTermOption) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborsCount(s, newEnvironmentsStep(), opts...)
	}
}

// ByEnvironments orders the results by Environments terms.
func ByEnvironments(term sql.OrderTerm, terms ...sql.OrderTerm) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborTerms(s, newEnvironmentsStep(), append([]sql.OrderTerm{term}, terms...)...)
	}
}
func newTagsStep() *sqlgraph.Step {
	return sqlgraph.NewStep(
		sqlgraph.From(Table, FieldID),
		sqlgraph.To(TagsInverseTable, FieldID),
		sqlgraph.Edge(sqlgraph.O2M, false, TagsTable, TagsColumn),
	)
}
func newHostsStep() *sqlgraph.Step {
	return sqlgraph.NewStep(
		sqlgraph.From(Table, FieldID),
		sqlgraph.To(HostsInverseTable, FieldID),
		sqlgraph.Edge(sqlgraph.M2M, false, HostsTable, HostsPrimaryKey...),
	)
}
func newNetworkStep() *sqlgraph.Step {
	return sqlgraph.NewStep(
		sqlgraph.From(Table, FieldID),
		sqlgraph.To(NetworkInverseTable, FieldID),
		sqlgraph.Edge(sqlgraph.M2O, false, NetworkTable, NetworkColumn),
	)
}
func newEnvironmentsStep() *sqlgraph.Step {
	return sqlgraph.NewStep(
		sqlgraph.From(Table, FieldID),
		sqlgraph.To(EnvironmentsInverseTable, FieldID),
		sqlgraph.Edge(sqlgraph.M2M, true, EnvironmentsTable, EnvironmentsPrimaryKey...),
	)
}
