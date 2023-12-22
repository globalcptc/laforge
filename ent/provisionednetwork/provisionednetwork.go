// Code generated by ent, DO NOT EDIT.

package provisionednetwork

import (
	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"github.com/google/uuid"
)

const (
	// Label holds the string label denoting the provisionednetwork type in the database.
	Label = "provisioned_network"
	// FieldID holds the string denoting the id field in the database.
	FieldID = "id"
	// FieldName holds the string denoting the name field in the database.
	FieldName = "name"
	// FieldCidr holds the string denoting the cidr field in the database.
	FieldCidr = "cidr"
	// FieldVars holds the string denoting the vars field in the database.
	FieldVars = "vars"
	// EdgeProvisionedNetworkToStatus holds the string denoting the provisionednetworktostatus edge name in mutations.
	EdgeProvisionedNetworkToStatus = "ProvisionedNetworkToStatus"
	// EdgeProvisionedNetworkToNetwork holds the string denoting the provisionednetworktonetwork edge name in mutations.
	EdgeProvisionedNetworkToNetwork = "ProvisionedNetworkToNetwork"
	// EdgeProvisionedNetworkToBuild holds the string denoting the provisionednetworktobuild edge name in mutations.
	EdgeProvisionedNetworkToBuild = "ProvisionedNetworkToBuild"
	// EdgeProvisionedNetworkToTeam holds the string denoting the provisionednetworktoteam edge name in mutations.
	EdgeProvisionedNetworkToTeam = "ProvisionedNetworkToTeam"
	// EdgeProvisionedNetworkToProvisionedHost holds the string denoting the provisionednetworktoprovisionedhost edge name in mutations.
	EdgeProvisionedNetworkToProvisionedHost = "ProvisionedNetworkToProvisionedHost"
	// EdgeProvisionedNetworkToPlan holds the string denoting the provisionednetworktoplan edge name in mutations.
	EdgeProvisionedNetworkToPlan = "ProvisionedNetworkToPlan"
	// Table holds the table name of the provisionednetwork in the database.
	Table = "provisioned_networks"
	// ProvisionedNetworkToStatusTable is the table that holds the ProvisionedNetworkToStatus relation/edge.
	ProvisionedNetworkToStatusTable = "status"
	// ProvisionedNetworkToStatusInverseTable is the table name for the Status entity.
	// It exists in this package in order to avoid circular dependency with the "status" package.
	ProvisionedNetworkToStatusInverseTable = "status"
	// ProvisionedNetworkToStatusColumn is the table column denoting the ProvisionedNetworkToStatus relation/edge.
	ProvisionedNetworkToStatusColumn = "provisioned_network_provisioned_network_to_status"
	// ProvisionedNetworkToNetworkTable is the table that holds the ProvisionedNetworkToNetwork relation/edge.
	ProvisionedNetworkToNetworkTable = "provisioned_networks"
	// ProvisionedNetworkToNetworkInverseTable is the table name for the Network entity.
	// It exists in this package in order to avoid circular dependency with the "network" package.
	ProvisionedNetworkToNetworkInverseTable = "networks"
	// ProvisionedNetworkToNetworkColumn is the table column denoting the ProvisionedNetworkToNetwork relation/edge.
	ProvisionedNetworkToNetworkColumn = "provisioned_network_provisioned_network_to_network"
	// ProvisionedNetworkToBuildTable is the table that holds the ProvisionedNetworkToBuild relation/edge.
	ProvisionedNetworkToBuildTable = "provisioned_networks"
	// ProvisionedNetworkToBuildInverseTable is the table name for the Build entity.
	// It exists in this package in order to avoid circular dependency with the "build" package.
	ProvisionedNetworkToBuildInverseTable = "builds"
	// ProvisionedNetworkToBuildColumn is the table column denoting the ProvisionedNetworkToBuild relation/edge.
	ProvisionedNetworkToBuildColumn = "provisioned_network_provisioned_network_to_build"
	// ProvisionedNetworkToTeamTable is the table that holds the ProvisionedNetworkToTeam relation/edge.
	ProvisionedNetworkToTeamTable = "provisioned_networks"
	// ProvisionedNetworkToTeamInverseTable is the table name for the Team entity.
	// It exists in this package in order to avoid circular dependency with the "team" package.
	ProvisionedNetworkToTeamInverseTable = "teams"
	// ProvisionedNetworkToTeamColumn is the table column denoting the ProvisionedNetworkToTeam relation/edge.
	ProvisionedNetworkToTeamColumn = "provisioned_network_provisioned_network_to_team"
	// ProvisionedNetworkToProvisionedHostTable is the table that holds the ProvisionedNetworkToProvisionedHost relation/edge.
	ProvisionedNetworkToProvisionedHostTable = "provisioned_hosts"
	// ProvisionedNetworkToProvisionedHostInverseTable is the table name for the ProvisionedHost entity.
	// It exists in this package in order to avoid circular dependency with the "provisionedhost" package.
	ProvisionedNetworkToProvisionedHostInverseTable = "provisioned_hosts"
	// ProvisionedNetworkToProvisionedHostColumn is the table column denoting the ProvisionedNetworkToProvisionedHost relation/edge.
	ProvisionedNetworkToProvisionedHostColumn = "provisioned_host_provisioned_host_to_provisioned_network"
	// ProvisionedNetworkToPlanTable is the table that holds the ProvisionedNetworkToPlan relation/edge.
	ProvisionedNetworkToPlanTable = "provisioned_networks"
	// ProvisionedNetworkToPlanInverseTable is the table name for the Plan entity.
	// It exists in this package in order to avoid circular dependency with the "plan" package.
	ProvisionedNetworkToPlanInverseTable = "plans"
	// ProvisionedNetworkToPlanColumn is the table column denoting the ProvisionedNetworkToPlan relation/edge.
	ProvisionedNetworkToPlanColumn = "plan_plan_to_provisioned_network"
)

// Columns holds all SQL columns for provisionednetwork fields.
var Columns = []string{
	FieldID,
	FieldName,
	FieldCidr,
	FieldVars,
}

// ForeignKeys holds the SQL foreign-keys that are owned by the "provisioned_networks"
// table and are not defined as standalone fields in the schema.
var ForeignKeys = []string{
	"plan_plan_to_provisioned_network",
	"provisioned_network_provisioned_network_to_network",
	"provisioned_network_provisioned_network_to_build",
	"provisioned_network_provisioned_network_to_team",
}

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

// OrderOption defines the ordering options for the ProvisionedNetwork queries.
type OrderOption func(*sql.Selector)

// ByID orders the results by the id field.
func ByID(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldID, opts...).ToFunc()
}

// ByName orders the results by the name field.
func ByName(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldName, opts...).ToFunc()
}

// ByCidr orders the results by the cidr field.
func ByCidr(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldCidr, opts...).ToFunc()
}

// ByProvisionedNetworkToStatusField orders the results by ProvisionedNetworkToStatus field.
func ByProvisionedNetworkToStatusField(field string, opts ...sql.OrderTermOption) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborTerms(s, newProvisionedNetworkToStatusStep(), sql.OrderByField(field, opts...))
	}
}

// ByProvisionedNetworkToNetworkField orders the results by ProvisionedNetworkToNetwork field.
func ByProvisionedNetworkToNetworkField(field string, opts ...sql.OrderTermOption) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborTerms(s, newProvisionedNetworkToNetworkStep(), sql.OrderByField(field, opts...))
	}
}

// ByProvisionedNetworkToBuildField orders the results by ProvisionedNetworkToBuild field.
func ByProvisionedNetworkToBuildField(field string, opts ...sql.OrderTermOption) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborTerms(s, newProvisionedNetworkToBuildStep(), sql.OrderByField(field, opts...))
	}
}

// ByProvisionedNetworkToTeamField orders the results by ProvisionedNetworkToTeam field.
func ByProvisionedNetworkToTeamField(field string, opts ...sql.OrderTermOption) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborTerms(s, newProvisionedNetworkToTeamStep(), sql.OrderByField(field, opts...))
	}
}

// ByProvisionedNetworkToProvisionedHostCount orders the results by ProvisionedNetworkToProvisionedHost count.
func ByProvisionedNetworkToProvisionedHostCount(opts ...sql.OrderTermOption) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborsCount(s, newProvisionedNetworkToProvisionedHostStep(), opts...)
	}
}

// ByProvisionedNetworkToProvisionedHost orders the results by ProvisionedNetworkToProvisionedHost terms.
func ByProvisionedNetworkToProvisionedHost(term sql.OrderTerm, terms ...sql.OrderTerm) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborTerms(s, newProvisionedNetworkToProvisionedHostStep(), append([]sql.OrderTerm{term}, terms...)...)
	}
}

// ByProvisionedNetworkToPlanField orders the results by ProvisionedNetworkToPlan field.
func ByProvisionedNetworkToPlanField(field string, opts ...sql.OrderTermOption) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborTerms(s, newProvisionedNetworkToPlanStep(), sql.OrderByField(field, opts...))
	}
}
func newProvisionedNetworkToStatusStep() *sqlgraph.Step {
	return sqlgraph.NewStep(
		sqlgraph.From(Table, FieldID),
		sqlgraph.To(ProvisionedNetworkToStatusInverseTable, FieldID),
		sqlgraph.Edge(sqlgraph.O2O, false, ProvisionedNetworkToStatusTable, ProvisionedNetworkToStatusColumn),
	)
}
func newProvisionedNetworkToNetworkStep() *sqlgraph.Step {
	return sqlgraph.NewStep(
		sqlgraph.From(Table, FieldID),
		sqlgraph.To(ProvisionedNetworkToNetworkInverseTable, FieldID),
		sqlgraph.Edge(sqlgraph.M2O, false, ProvisionedNetworkToNetworkTable, ProvisionedNetworkToNetworkColumn),
	)
}
func newProvisionedNetworkToBuildStep() *sqlgraph.Step {
	return sqlgraph.NewStep(
		sqlgraph.From(Table, FieldID),
		sqlgraph.To(ProvisionedNetworkToBuildInverseTable, FieldID),
		sqlgraph.Edge(sqlgraph.M2O, false, ProvisionedNetworkToBuildTable, ProvisionedNetworkToBuildColumn),
	)
}
func newProvisionedNetworkToTeamStep() *sqlgraph.Step {
	return sqlgraph.NewStep(
		sqlgraph.From(Table, FieldID),
		sqlgraph.To(ProvisionedNetworkToTeamInverseTable, FieldID),
		sqlgraph.Edge(sqlgraph.M2O, false, ProvisionedNetworkToTeamTable, ProvisionedNetworkToTeamColumn),
	)
}
func newProvisionedNetworkToProvisionedHostStep() *sqlgraph.Step {
	return sqlgraph.NewStep(
		sqlgraph.From(Table, FieldID),
		sqlgraph.To(ProvisionedNetworkToProvisionedHostInverseTable, FieldID),
		sqlgraph.Edge(sqlgraph.O2M, true, ProvisionedNetworkToProvisionedHostTable, ProvisionedNetworkToProvisionedHostColumn),
	)
}
func newProvisionedNetworkToPlanStep() *sqlgraph.Step {
	return sqlgraph.NewStep(
		sqlgraph.From(Table, FieldID),
		sqlgraph.To(ProvisionedNetworkToPlanInverseTable, FieldID),
		sqlgraph.Edge(sqlgraph.O2O, true, ProvisionedNetworkToPlanTable, ProvisionedNetworkToPlanColumn),
	)
}
