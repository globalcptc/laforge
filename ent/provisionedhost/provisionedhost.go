// Code generated by ent, DO NOT EDIT.

package provisionedhost

import (
	"fmt"
	"io"
	"strconv"

	"github.com/google/uuid"
)

const (
	// Label holds the string label denoting the provisionedhost type in the database.
	Label = "provisioned_host"
	// FieldID holds the string denoting the id field in the database.
	FieldID = "id"
	// FieldSubnetIP holds the string denoting the subnet_ip field in the database.
	FieldSubnetIP = "subnet_ip"
	// FieldAddonType holds the string denoting the addon_type field in the database.
	FieldAddonType = "addon_type"
	// FieldVars holds the string denoting the vars field in the database.
	FieldVars = "vars"
	// EdgeProvisionedHostToStatus holds the string denoting the provisionedhosttostatus edge name in mutations.
	EdgeProvisionedHostToStatus = "ProvisionedHostToStatus"
	// EdgeProvisionedHostToProvisionedNetwork holds the string denoting the provisionedhosttoprovisionednetwork edge name in mutations.
	EdgeProvisionedHostToProvisionedNetwork = "ProvisionedHostToProvisionedNetwork"
	// EdgeProvisionedHostToHost holds the string denoting the provisionedhosttohost edge name in mutations.
	EdgeProvisionedHostToHost = "ProvisionedHostToHost"
	// EdgeProvisionedHostToEndStepPlan holds the string denoting the provisionedhosttoendstepplan edge name in mutations.
	EdgeProvisionedHostToEndStepPlan = "ProvisionedHostToEndStepPlan"
	// EdgeProvisionedHostToBuild holds the string denoting the provisionedhosttobuild edge name in mutations.
	EdgeProvisionedHostToBuild = "ProvisionedHostToBuild"
	// EdgeProvisionedHostToProvisioningStep holds the string denoting the provisionedhosttoprovisioningstep edge name in mutations.
	EdgeProvisionedHostToProvisioningStep = "ProvisionedHostToProvisioningStep"
	// EdgeProvisionedHostToProvisioningScheduledStep holds the string denoting the provisionedhosttoprovisioningscheduledstep edge name in mutations.
	EdgeProvisionedHostToProvisioningScheduledStep = "ProvisionedHostToProvisioningScheduledStep"
	// EdgeProvisionedHostToAgentStatus holds the string denoting the provisionedhosttoagentstatus edge name in mutations.
	EdgeProvisionedHostToAgentStatus = "ProvisionedHostToAgentStatus"
	// EdgeProvisionedHostToAgentTask holds the string denoting the provisionedhosttoagenttask edge name in mutations.
	EdgeProvisionedHostToAgentTask = "ProvisionedHostToAgentTask"
	// EdgeProvisionedHostToPlan holds the string denoting the provisionedhosttoplan edge name in mutations.
	EdgeProvisionedHostToPlan = "ProvisionedHostToPlan"
	// EdgeProvisionedHostToGinFileMiddleware holds the string denoting the provisionedhosttoginfilemiddleware edge name in mutations.
	EdgeProvisionedHostToGinFileMiddleware = "ProvisionedHostToGinFileMiddleware"
	// Table holds the table name of the provisionedhost in the database.
	Table = "provisioned_hosts"
	// ProvisionedHostToStatusTable is the table that holds the ProvisionedHostToStatus relation/edge.
	ProvisionedHostToStatusTable = "status"
	// ProvisionedHostToStatusInverseTable is the table name for the Status entity.
	// It exists in this package in order to avoid circular dependency with the "status" package.
	ProvisionedHostToStatusInverseTable = "status"
	// ProvisionedHostToStatusColumn is the table column denoting the ProvisionedHostToStatus relation/edge.
	ProvisionedHostToStatusColumn = "provisioned_host_provisioned_host_to_status"
	// ProvisionedHostToProvisionedNetworkTable is the table that holds the ProvisionedHostToProvisionedNetwork relation/edge.
	ProvisionedHostToProvisionedNetworkTable = "provisioned_hosts"
	// ProvisionedHostToProvisionedNetworkInverseTable is the table name for the ProvisionedNetwork entity.
	// It exists in this package in order to avoid circular dependency with the "provisionednetwork" package.
	ProvisionedHostToProvisionedNetworkInverseTable = "provisioned_networks"
	// ProvisionedHostToProvisionedNetworkColumn is the table column denoting the ProvisionedHostToProvisionedNetwork relation/edge.
	ProvisionedHostToProvisionedNetworkColumn = "provisioned_host_provisioned_host_to_provisioned_network"
	// ProvisionedHostToHostTable is the table that holds the ProvisionedHostToHost relation/edge.
	ProvisionedHostToHostTable = "provisioned_hosts"
	// ProvisionedHostToHostInverseTable is the table name for the Host entity.
	// It exists in this package in order to avoid circular dependency with the "host" package.
	ProvisionedHostToHostInverseTable = "hosts"
	// ProvisionedHostToHostColumn is the table column denoting the ProvisionedHostToHost relation/edge.
	ProvisionedHostToHostColumn = "provisioned_host_provisioned_host_to_host"
	// ProvisionedHostToEndStepPlanTable is the table that holds the ProvisionedHostToEndStepPlan relation/edge.
	ProvisionedHostToEndStepPlanTable = "provisioned_hosts"
	// ProvisionedHostToEndStepPlanInverseTable is the table name for the Plan entity.
	// It exists in this package in order to avoid circular dependency with the "plan" package.
	ProvisionedHostToEndStepPlanInverseTable = "plans"
	// ProvisionedHostToEndStepPlanColumn is the table column denoting the ProvisionedHostToEndStepPlan relation/edge.
	ProvisionedHostToEndStepPlanColumn = "provisioned_host_provisioned_host_to_end_step_plan"
	// ProvisionedHostToBuildTable is the table that holds the ProvisionedHostToBuild relation/edge.
	ProvisionedHostToBuildTable = "provisioned_hosts"
	// ProvisionedHostToBuildInverseTable is the table name for the Build entity.
	// It exists in this package in order to avoid circular dependency with the "build" package.
	ProvisionedHostToBuildInverseTable = "builds"
	// ProvisionedHostToBuildColumn is the table column denoting the ProvisionedHostToBuild relation/edge.
	ProvisionedHostToBuildColumn = "provisioned_host_provisioned_host_to_build"
	// ProvisionedHostToProvisioningStepTable is the table that holds the ProvisionedHostToProvisioningStep relation/edge.
	ProvisionedHostToProvisioningStepTable = "provisioning_steps"
	// ProvisionedHostToProvisioningStepInverseTable is the table name for the ProvisioningStep entity.
	// It exists in this package in order to avoid circular dependency with the "provisioningstep" package.
	ProvisionedHostToProvisioningStepInverseTable = "provisioning_steps"
	// ProvisionedHostToProvisioningStepColumn is the table column denoting the ProvisionedHostToProvisioningStep relation/edge.
	ProvisionedHostToProvisioningStepColumn = "provisioning_step_provisioning_step_to_provisioned_host"
	// ProvisionedHostToProvisioningScheduledStepTable is the table that holds the ProvisionedHostToProvisioningScheduledStep relation/edge.
	ProvisionedHostToProvisioningScheduledStepTable = "provisioning_scheduled_steps"
	// ProvisionedHostToProvisioningScheduledStepInverseTable is the table name for the ProvisioningScheduledStep entity.
	// It exists in this package in order to avoid circular dependency with the "provisioningscheduledstep" package.
	ProvisionedHostToProvisioningScheduledStepInverseTable = "provisioning_scheduled_steps"
	// ProvisionedHostToProvisioningScheduledStepColumn is the table column denoting the ProvisionedHostToProvisioningScheduledStep relation/edge.
	ProvisionedHostToProvisioningScheduledStepColumn = "provisioning_scheduled_step_provisioned_host"
	// ProvisionedHostToAgentStatusTable is the table that holds the ProvisionedHostToAgentStatus relation/edge.
	ProvisionedHostToAgentStatusTable = "agent_status"
	// ProvisionedHostToAgentStatusInverseTable is the table name for the AgentStatus entity.
	// It exists in this package in order to avoid circular dependency with the "agentstatus" package.
	ProvisionedHostToAgentStatusInverseTable = "agent_status"
	// ProvisionedHostToAgentStatusColumn is the table column denoting the ProvisionedHostToAgentStatus relation/edge.
	ProvisionedHostToAgentStatusColumn = "agent_status_provisioned_host"
	// ProvisionedHostToAgentTaskTable is the table that holds the ProvisionedHostToAgentTask relation/edge.
	ProvisionedHostToAgentTaskTable = "agent_tasks"
	// ProvisionedHostToAgentTaskInverseTable is the table name for the AgentTask entity.
	// It exists in this package in order to avoid circular dependency with the "agenttask" package.
	ProvisionedHostToAgentTaskInverseTable = "agent_tasks"
	// ProvisionedHostToAgentTaskColumn is the table column denoting the ProvisionedHostToAgentTask relation/edge.
	ProvisionedHostToAgentTaskColumn = "agent_task_provisioned_host"
	// ProvisionedHostToPlanTable is the table that holds the ProvisionedHostToPlan relation/edge.
	ProvisionedHostToPlanTable = "provisioned_hosts"
	// ProvisionedHostToPlanInverseTable is the table name for the Plan entity.
	// It exists in this package in order to avoid circular dependency with the "plan" package.
	ProvisionedHostToPlanInverseTable = "plans"
	// ProvisionedHostToPlanColumn is the table column denoting the ProvisionedHostToPlan relation/edge.
	ProvisionedHostToPlanColumn = "plan_plan_to_provisioned_host"
	// ProvisionedHostToGinFileMiddlewareTable is the table that holds the ProvisionedHostToGinFileMiddleware relation/edge.
	ProvisionedHostToGinFileMiddlewareTable = "provisioned_hosts"
	// ProvisionedHostToGinFileMiddlewareInverseTable is the table name for the GinFileMiddleware entity.
	// It exists in this package in order to avoid circular dependency with the "ginfilemiddleware" package.
	ProvisionedHostToGinFileMiddlewareInverseTable = "gin_file_middlewares"
	// ProvisionedHostToGinFileMiddlewareColumn is the table column denoting the ProvisionedHostToGinFileMiddleware relation/edge.
	ProvisionedHostToGinFileMiddlewareColumn = "gin_file_middleware_gin_file_middleware_to_provisioned_host"
)

// Columns holds all SQL columns for provisionedhost fields.
var Columns = []string{
	FieldID,
	FieldSubnetIP,
	FieldAddonType,
	FieldVars,
}

// ForeignKeys holds the SQL foreign-keys that are owned by the "provisioned_hosts"
// table and are not defined as standalone fields in the schema.
var ForeignKeys = []string{
	"gin_file_middleware_gin_file_middleware_to_provisioned_host",
	"plan_plan_to_provisioned_host",
	"provisioned_host_provisioned_host_to_provisioned_network",
	"provisioned_host_provisioned_host_to_host",
	"provisioned_host_provisioned_host_to_end_step_plan",
	"provisioned_host_provisioned_host_to_build",
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

// AddonType defines the type for the "addon_type" enum field.
type AddonType string

// AddonType values.
const (
	AddonTypeDNS AddonType = "DNS"
)

func (at AddonType) String() string {
	return string(at)
}

// AddonTypeValidator is a validator for the "addon_type" field enum values. It is called by the builders before save.
func AddonTypeValidator(at AddonType) error {
	switch at {
	case AddonTypeDNS:
		return nil
	default:
		return fmt.Errorf("provisionedhost: invalid enum value for addon_type field: %q", at)
	}
}

// MarshalGQL implements graphql.Marshaler interface.
func (at AddonType) MarshalGQL(w io.Writer) {
	io.WriteString(w, strconv.Quote(at.String()))
}

// UnmarshalGQL implements graphql.Unmarshaler interface.
func (at *AddonType) UnmarshalGQL(val interface{}) error {
	str, ok := val.(string)
	if !ok {
		return fmt.Errorf("enum %T must be a string", val)
	}
	*at = AddonType(str)
	if err := AddonTypeValidator(*at); err != nil {
		return fmt.Errorf("%s is not a valid AddonType", str)
	}
	return nil
}
