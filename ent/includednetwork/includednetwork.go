// Code generated by entc, DO NOT EDIT.

package includednetwork

import (
	"github.com/google/uuid"
)

const (
	// Label holds the string label denoting the includednetwork type in the database.
	Label = "included_network"
	// FieldID holds the string denoting the id field in the database.
	FieldID = "id"
	// FieldName holds the string denoting the name field in the database.
	FieldName = "name"
	// FieldHosts holds the string denoting the hosts field in the database.
	FieldHosts = "hosts"
	// EdgeIncludedNetworkToTag holds the string denoting the includednetworktotag edge name in mutations.
	EdgeIncludedNetworkToTag = "IncludedNetworkToTag"
	// EdgeIncludedNetworkToHost holds the string denoting the includednetworktohost edge name in mutations.
	EdgeIncludedNetworkToHost = "IncludedNetworkToHost"
	// EdgeIncludedNetworkToNetwork holds the string denoting the includednetworktonetwork edge name in mutations.
	EdgeIncludedNetworkToNetwork = "IncludedNetworkToNetwork"
	// EdgeIncludedNetworkToEnvironment holds the string denoting the includednetworktoenvironment edge name in mutations.
	EdgeIncludedNetworkToEnvironment = "IncludedNetworkToEnvironment"
	// Table holds the table name of the includednetwork in the database.
	Table = "included_networks"
	// IncludedNetworkToTagTable is the table that holds the IncludedNetworkToTag relation/edge.
	IncludedNetworkToTagTable = "tags"
	// IncludedNetworkToTagInverseTable is the table name for the Tag entity.
	// It exists in this package in order to avoid circular dependency with the "tag" package.
	IncludedNetworkToTagInverseTable = "tags"
	// IncludedNetworkToTagColumn is the table column denoting the IncludedNetworkToTag relation/edge.
	IncludedNetworkToTagColumn = "included_network_included_network_to_tag"
	// IncludedNetworkToHostTable is the table that holds the IncludedNetworkToHost relation/edge. The primary key declared below.
	IncludedNetworkToHostTable = "included_network_IncludedNetworkToHost"
	// IncludedNetworkToHostInverseTable is the table name for the Host entity.
	// It exists in this package in order to avoid circular dependency with the "host" package.
	IncludedNetworkToHostInverseTable = "hosts"
	// IncludedNetworkToNetworkTable is the table that holds the IncludedNetworkToNetwork relation/edge.
	IncludedNetworkToNetworkTable = "included_networks"
	// IncludedNetworkToNetworkInverseTable is the table name for the Network entity.
	// It exists in this package in order to avoid circular dependency with the "network" package.
	IncludedNetworkToNetworkInverseTable = "networks"
	// IncludedNetworkToNetworkColumn is the table column denoting the IncludedNetworkToNetwork relation/edge.
	IncludedNetworkToNetworkColumn = "included_network_included_network_to_network"
	// IncludedNetworkToEnvironmentTable is the table that holds the IncludedNetworkToEnvironment relation/edge. The primary key declared below.
	IncludedNetworkToEnvironmentTable = "environment_EnvironmentToIncludedNetwork"
	// IncludedNetworkToEnvironmentInverseTable is the table name for the Environment entity.
	// It exists in this package in order to avoid circular dependency with the "environment" package.
	IncludedNetworkToEnvironmentInverseTable = "environments"
)

// Columns holds all SQL columns for includednetwork fields.
var Columns = []string{
	FieldID,
	FieldName,
	FieldHosts,
}

// ForeignKeys holds the SQL foreign-keys that are owned by the "included_networks"
// table and are not defined as standalone fields in the schema.
var ForeignKeys = []string{
	"included_network_included_network_to_network",
}

var (
	// IncludedNetworkToHostPrimaryKey and IncludedNetworkToHostColumn2 are the table columns denoting the
	// primary key for the IncludedNetworkToHost relation (M2M).
	IncludedNetworkToHostPrimaryKey = []string{"included_network_id", "host_id"}
	// IncludedNetworkToEnvironmentPrimaryKey and IncludedNetworkToEnvironmentColumn2 are the table columns denoting the
	// primary key for the IncludedNetworkToEnvironment relation (M2M).
	IncludedNetworkToEnvironmentPrimaryKey = []string{"environment_id", "included_network_id"}
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
