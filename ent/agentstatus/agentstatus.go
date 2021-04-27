// Code generated by entc, DO NOT EDIT.

package agentstatus

import (
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
	// EdgeAgentStatusToProvisionedHost holds the string denoting the agentstatustoprovisionedhost edge name in mutations.
	EdgeAgentStatusToProvisionedHost = "AgentStatusToProvisionedHost"
	// Table holds the table name of the agentstatus in the database.
	Table = "agent_status"
	// AgentStatusToProvisionedHostTable is the table the holds the AgentStatusToProvisionedHost relation/edge. The primary key declared below.
	AgentStatusToProvisionedHostTable = "agent_status_AgentStatusToProvisionedHost"
	// AgentStatusToProvisionedHostInverseTable is the table name for the ProvisionedHost entity.
	// It exists in this package in order to avoid circular dependency with the "provisionedhost" package.
	AgentStatusToProvisionedHostInverseTable = "provisioned_hosts"
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

var (
	// AgentStatusToProvisionedHostPrimaryKey and AgentStatusToProvisionedHostColumn2 are the table columns denoting the
	// primary key for the AgentStatusToProvisionedHost relation (M2M).
	AgentStatusToProvisionedHostPrimaryKey = []string{"agent_status_id", "provisioned_host_id"}
)

// ValidColumn reports if the column name is valid (part of the table columns).
func ValidColumn(column string) bool {
	for i := range Columns {
		if column == Columns[i] {
			return true
		}
	}
	return false
}

var (
	// DefaultID holds the default value on creation for the "id" field.
	DefaultID func() uuid.UUID
)
