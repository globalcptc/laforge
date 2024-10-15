// Code generated by ent, DO NOT EDIT.

package agentstatus

import (
	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"github.com/google/uuid"
)

const (
	// Label holds the string label denoting the agentstatus type in the database.
	Label = "agent_status"
	// FieldID holds the string denoting the id field in the database.
	FieldID = "id"
	// FieldClientID holds the string denoting the clientid field in the database.
	FieldClientID = "client_id"
	// FieldHostname holds the string denoting the hostname field in the database.
	FieldHostname = "hostname"
	// FieldUpTime holds the string denoting the uptime field in the database.
	FieldUpTime = "up_time"
	// FieldBootTime holds the string denoting the boottime field in the database.
	FieldBootTime = "boot_time"
	// FieldNumProcs holds the string denoting the numprocs field in the database.
	FieldNumProcs = "num_procs"
	// FieldOs holds the string denoting the os field in the database.
	FieldOs = "os"
	// FieldHostID holds the string denoting the hostid field in the database.
	FieldHostID = "host_id"
	// FieldLoad1 holds the string denoting the load1 field in the database.
	FieldLoad1 = "load1"
	// FieldLoad5 holds the string denoting the load5 field in the database.
	FieldLoad5 = "load5"
	// FieldLoad15 holds the string denoting the load15 field in the database.
	FieldLoad15 = "load15"
	// FieldTotalMem holds the string denoting the totalmem field in the database.
	FieldTotalMem = "total_mem"
	// FieldFreeMem holds the string denoting the freemem field in the database.
	FieldFreeMem = "free_mem"
	// FieldUsedMem holds the string denoting the usedmem field in the database.
	FieldUsedMem = "used_mem"
	// FieldTimestamp holds the string denoting the timestamp field in the database.
	FieldTimestamp = "timestamp"
	// EdgeProvisionedHost holds the string denoting the provisionedhost edge name in mutations.
	EdgeProvisionedHost = "ProvisionedHost"
	// EdgeProvisionedNetwork holds the string denoting the provisionednetwork edge name in mutations.
	EdgeProvisionedNetwork = "ProvisionedNetwork"
	// EdgeBuild holds the string denoting the build edge name in mutations.
	EdgeBuild = "Build"
	// Table holds the table name of the agentstatus in the database.
	Table = "agent_status"
	// ProvisionedHostTable is the table that holds the ProvisionedHost relation/edge.
	ProvisionedHostTable = "agent_status"
	// ProvisionedHostInverseTable is the table name for the ProvisionedHost entity.
	// It exists in this package in order to avoid circular dependency with the "provisionedhost" package.
	ProvisionedHostInverseTable = "provisioned_hosts"
	// ProvisionedHostColumn is the table column denoting the ProvisionedHost relation/edge.
	ProvisionedHostColumn = "agent_status_provisioned_host"
	// ProvisionedNetworkTable is the table that holds the ProvisionedNetwork relation/edge.
	ProvisionedNetworkTable = "agent_status"
	// ProvisionedNetworkInverseTable is the table name for the ProvisionedNetwork entity.
	// It exists in this package in order to avoid circular dependency with the "provisionednetwork" package.
	ProvisionedNetworkInverseTable = "provisioned_networks"
	// ProvisionedNetworkColumn is the table column denoting the ProvisionedNetwork relation/edge.
	ProvisionedNetworkColumn = "agent_status_provisioned_network"
	// BuildTable is the table that holds the Build relation/edge.
	BuildTable = "agent_status"
	// BuildInverseTable is the table name for the Build entity.
	// It exists in this package in order to avoid circular dependency with the "build" package.
	BuildInverseTable = "builds"
	// BuildColumn is the table column denoting the Build relation/edge.
	BuildColumn = "agent_status_build"
)

// Columns holds all SQL columns for agentstatus fields.
var Columns = []string{
	FieldID,
	FieldClientID,
	FieldHostname,
	FieldUpTime,
	FieldBootTime,
	FieldNumProcs,
	FieldOs,
	FieldHostID,
	FieldLoad1,
	FieldLoad5,
	FieldLoad15,
	FieldTotalMem,
	FieldFreeMem,
	FieldUsedMem,
	FieldTimestamp,
}

// ForeignKeys holds the SQL foreign-keys that are owned by the "agent_status"
// table and are not defined as standalone fields in the schema.
var ForeignKeys = []string{
	"agent_status_provisioned_host",
	"agent_status_provisioned_network",
	"agent_status_build",
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

// OrderOption defines the ordering options for the AgentStatus queries.
type OrderOption func(*sql.Selector)

// ByID orders the results by the id field.
func ByID(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldID, opts...).ToFunc()
}

// ByClientID orders the results by the ClientID field.
func ByClientID(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldClientID, opts...).ToFunc()
}

// ByHostname orders the results by the Hostname field.
func ByHostname(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldHostname, opts...).ToFunc()
}

// ByUpTime orders the results by the UpTime field.
func ByUpTime(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldUpTime, opts...).ToFunc()
}

// ByBootTime orders the results by the BootTime field.
func ByBootTime(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldBootTime, opts...).ToFunc()
}

// ByNumProcs orders the results by the NumProcs field.
func ByNumProcs(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldNumProcs, opts...).ToFunc()
}

// ByOs orders the results by the Os field.
func ByOs(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldOs, opts...).ToFunc()
}

// ByHostID orders the results by the HostID field.
func ByHostID(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldHostID, opts...).ToFunc()
}

// ByLoad1 orders the results by the Load1 field.
func ByLoad1(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldLoad1, opts...).ToFunc()
}

// ByLoad5 orders the results by the Load5 field.
func ByLoad5(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldLoad5, opts...).ToFunc()
}

// ByLoad15 orders the results by the Load15 field.
func ByLoad15(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldLoad15, opts...).ToFunc()
}

// ByTotalMem orders the results by the TotalMem field.
func ByTotalMem(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldTotalMem, opts...).ToFunc()
}

// ByFreeMem orders the results by the FreeMem field.
func ByFreeMem(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldFreeMem, opts...).ToFunc()
}

// ByUsedMem orders the results by the UsedMem field.
func ByUsedMem(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldUsedMem, opts...).ToFunc()
}

// ByTimestamp orders the results by the Timestamp field.
func ByTimestamp(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldTimestamp, opts...).ToFunc()
}

// ByProvisionedHostField orders the results by ProvisionedHost field.
func ByProvisionedHostField(field string, opts ...sql.OrderTermOption) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborTerms(s, newProvisionedHostStep(), sql.OrderByField(field, opts...))
	}
}

// ByProvisionedNetworkField orders the results by ProvisionedNetwork field.
func ByProvisionedNetworkField(field string, opts ...sql.OrderTermOption) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborTerms(s, newProvisionedNetworkStep(), sql.OrderByField(field, opts...))
	}
}

// ByBuildField orders the results by Build field.
func ByBuildField(field string, opts ...sql.OrderTermOption) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborTerms(s, newBuildStep(), sql.OrderByField(field, opts...))
	}
}
func newProvisionedHostStep() *sqlgraph.Step {
	return sqlgraph.NewStep(
		sqlgraph.From(Table, FieldID),
		sqlgraph.To(ProvisionedHostInverseTable, FieldID),
		sqlgraph.Edge(sqlgraph.M2O, false, ProvisionedHostTable, ProvisionedHostColumn),
	)
}
func newProvisionedNetworkStep() *sqlgraph.Step {
	return sqlgraph.NewStep(
		sqlgraph.From(Table, FieldID),
		sqlgraph.To(ProvisionedNetworkInverseTable, FieldID),
		sqlgraph.Edge(sqlgraph.M2O, false, ProvisionedNetworkTable, ProvisionedNetworkColumn),
	)
}
func newBuildStep() *sqlgraph.Step {
	return sqlgraph.NewStep(
		sqlgraph.From(Table, FieldID),
		sqlgraph.To(BuildInverseTable, FieldID),
		sqlgraph.Edge(sqlgraph.M2O, false, BuildTable, BuildColumn),
	)
}
