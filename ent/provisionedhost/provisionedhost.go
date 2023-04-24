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
	// EdgeStatus holds the string denoting the status edge name in mutations.
	EdgeStatus = "Status"
	// EdgeProvisionedNetwork holds the string denoting the provisionednetwork edge name in mutations.
	EdgeProvisionedNetwork = "ProvisionedNetwork"
	// EdgeHost holds the string denoting the host edge name in mutations.
	EdgeHost = "Host"
	// EdgeEndStepPlan holds the string denoting the endstepplan edge name in mutations.
	EdgeEndStepPlan = "EndStepPlan"
	// EdgeBuild holds the string denoting the build edge name in mutations.
	EdgeBuild = "Build"
	// EdgeProvisioningSteps holds the string denoting the provisioningsteps edge name in mutations.
	EdgeProvisioningSteps = "ProvisioningSteps"
	// EdgeProvisioningScheduledSteps holds the string denoting the provisioningscheduledsteps edge name in mutations.
	EdgeProvisioningScheduledSteps = "ProvisioningScheduledSteps"
	// EdgeAgentStatuses holds the string denoting the agentstatuses edge name in mutations.
	EdgeAgentStatuses = "AgentStatuses"
	// EdgeAgentTasks holds the string denoting the agenttasks edge name in mutations.
	EdgeAgentTasks = "AgentTasks"
	// EdgePlan holds the string denoting the plan edge name in mutations.
	EdgePlan = "Plan"
	// EdgeGinFileMiddleware holds the string denoting the ginfilemiddleware edge name in mutations.
	EdgeGinFileMiddleware = "GinFileMiddleware"
	// Table holds the table name of the provisionedhost in the database.
	Table = "provisioned_hosts"
	// StatusTable is the table that holds the Status relation/edge.
	StatusTable = "status"
	// StatusInverseTable is the table name for the Status entity.
	// It exists in this package in order to avoid circular dependency with the "status" package.
	StatusInverseTable = "status"
	// StatusColumn is the table column denoting the Status relation/edge.
	StatusColumn = "provisioned_host_status"
	// ProvisionedNetworkTable is the table that holds the ProvisionedNetwork relation/edge.
	ProvisionedNetworkTable = "provisioned_hosts"
	// ProvisionedNetworkInverseTable is the table name for the ProvisionedNetwork entity.
	// It exists in this package in order to avoid circular dependency with the "provisionednetwork" package.
	ProvisionedNetworkInverseTable = "provisioned_networks"
	// ProvisionedNetworkColumn is the table column denoting the ProvisionedNetwork relation/edge.
	ProvisionedNetworkColumn = "provisioned_host_provisioned_network"
	// HostTable is the table that holds the Host relation/edge.
	HostTable = "provisioned_hosts"
	// HostInverseTable is the table name for the Host entity.
	// It exists in this package in order to avoid circular dependency with the "host" package.
	HostInverseTable = "hosts"
	// HostColumn is the table column denoting the Host relation/edge.
	HostColumn = "provisioned_host_host"
	// EndStepPlanTable is the table that holds the EndStepPlan relation/edge.
	EndStepPlanTable = "provisioned_hosts"
	// EndStepPlanInverseTable is the table name for the Plan entity.
	// It exists in this package in order to avoid circular dependency with the "plan" package.
	EndStepPlanInverseTable = "plans"
	// EndStepPlanColumn is the table column denoting the EndStepPlan relation/edge.
	EndStepPlanColumn = "provisioned_host_end_step_plan"
	// BuildTable is the table that holds the Build relation/edge.
	BuildTable = "provisioned_hosts"
	// BuildInverseTable is the table name for the Build entity.
	// It exists in this package in order to avoid circular dependency with the "build" package.
	BuildInverseTable = "builds"
	// BuildColumn is the table column denoting the Build relation/edge.
	BuildColumn = "provisioned_host_build"
	// ProvisioningStepsTable is the table that holds the ProvisioningSteps relation/edge.
	ProvisioningStepsTable = "provisioning_steps"
	// ProvisioningStepsInverseTable is the table name for the ProvisioningStep entity.
	// It exists in this package in order to avoid circular dependency with the "provisioningstep" package.
	ProvisioningStepsInverseTable = "provisioning_steps"
	// ProvisioningStepsColumn is the table column denoting the ProvisioningSteps relation/edge.
	ProvisioningStepsColumn = "provisioning_step_provisioning_step_to_provisioned_host"
	// ProvisioningScheduledStepsTable is the table that holds the ProvisioningScheduledSteps relation/edge.
	ProvisioningScheduledStepsTable = "provisioning_scheduled_steps"
	// ProvisioningScheduledStepsInverseTable is the table name for the ProvisioningScheduledStep entity.
	// It exists in this package in order to avoid circular dependency with the "provisioningscheduledstep" package.
	ProvisioningScheduledStepsInverseTable = "provisioning_scheduled_steps"
	// ProvisioningScheduledStepsColumn is the table column denoting the ProvisioningScheduledSteps relation/edge.
	ProvisioningScheduledStepsColumn = "provisioning_scheduled_step_provisioned_host"
	// AgentStatusesTable is the table that holds the AgentStatuses relation/edge.
	AgentStatusesTable = "agent_status"
	// AgentStatusesInverseTable is the table name for the AgentStatus entity.
	// It exists in this package in order to avoid circular dependency with the "agentstatus" package.
	AgentStatusesInverseTable = "agent_status"
	// AgentStatusesColumn is the table column denoting the AgentStatuses relation/edge.
	AgentStatusesColumn = "agent_status_provisioned_host"
	// AgentTasksTable is the table that holds the AgentTasks relation/edge.
	AgentTasksTable = "agent_tasks"
	// AgentTasksInverseTable is the table name for the AgentTask entity.
	// It exists in this package in order to avoid circular dependency with the "agenttask" package.
	AgentTasksInverseTable = "agent_tasks"
	// AgentTasksColumn is the table column denoting the AgentTasks relation/edge.
	AgentTasksColumn = "agent_task_provisioned_host"
	// PlanTable is the table that holds the Plan relation/edge.
	PlanTable = "provisioned_hosts"
	// PlanInverseTable is the table name for the Plan entity.
	// It exists in this package in order to avoid circular dependency with the "plan" package.
	PlanInverseTable = "plans"
	// PlanColumn is the table column denoting the Plan relation/edge.
	PlanColumn = "plan_provisioned_host"
	// GinFileMiddlewareTable is the table that holds the GinFileMiddleware relation/edge.
	GinFileMiddlewareTable = "provisioned_hosts"
	// GinFileMiddlewareInverseTable is the table name for the GinFileMiddleware entity.
	// It exists in this package in order to avoid circular dependency with the "ginfilemiddleware" package.
	GinFileMiddlewareInverseTable = "gin_file_middlewares"
	// GinFileMiddlewareColumn is the table column denoting the GinFileMiddleware relation/edge.
	GinFileMiddlewareColumn = "gin_file_middleware_provisioned_host"
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
	"gin_file_middleware_provisioned_host",
	"plan_provisioned_host",
	"provisioned_host_provisioned_network",
	"provisioned_host_host",
	"provisioned_host_end_step_plan",
	"provisioned_host_build",
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
