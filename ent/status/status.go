// Code generated by ent, DO NOT EDIT.

package status

import (
	"fmt"
	"io"
	"strconv"

	"github.com/google/uuid"
)

const (
	// Label holds the string label denoting the status type in the database.
	Label = "status"
	// FieldID holds the string denoting the id field in the database.
	FieldID = "id"
	// FieldState holds the string denoting the state field in the database.
	FieldState = "state"
	// FieldStatusFor holds the string denoting the status_for field in the database.
	FieldStatusFor = "status_for"
	// FieldStartedAt holds the string denoting the started_at field in the database.
	FieldStartedAt = "started_at"
	// FieldEndedAt holds the string denoting the ended_at field in the database.
	FieldEndedAt = "ended_at"
	// FieldFailed holds the string denoting the failed field in the database.
	FieldFailed = "failed"
	// FieldCompleted holds the string denoting the completed field in the database.
	FieldCompleted = "completed"
	// FieldError holds the string denoting the error field in the database.
	FieldError = "error"
	// EdgeStatusToBuild holds the string denoting the statustobuild edge name in mutations.
	EdgeStatusToBuild = "StatusToBuild"
	// EdgeStatusToProvisionedNetwork holds the string denoting the statustoprovisionednetwork edge name in mutations.
	EdgeStatusToProvisionedNetwork = "StatusToProvisionedNetwork"
	// EdgeStatusToProvisionedHost holds the string denoting the statustoprovisionedhost edge name in mutations.
	EdgeStatusToProvisionedHost = "StatusToProvisionedHost"
	// EdgeStatusToProvisioningStep holds the string denoting the statustoprovisioningstep edge name in mutations.
	EdgeStatusToProvisioningStep = "StatusToProvisioningStep"
	// EdgeStatusToTeam holds the string denoting the statustoteam edge name in mutations.
	EdgeStatusToTeam = "StatusToTeam"
	// EdgeStatusToPlan holds the string denoting the statustoplan edge name in mutations.
	EdgeStatusToPlan = "StatusToPlan"
	// EdgeStatusToServerTask holds the string denoting the statustoservertask edge name in mutations.
	EdgeStatusToServerTask = "StatusToServerTask"
	// EdgeStatusToAdhocPlan holds the string denoting the statustoadhocplan edge name in mutations.
	EdgeStatusToAdhocPlan = "StatusToAdhocPlan"
	// EdgeStatusToProvisioningScheduledStep holds the string denoting the statustoprovisioningscheduledstep edge name in mutations.
	EdgeStatusToProvisioningScheduledStep = "StatusToProvisioningScheduledStep"
	// Table holds the table name of the status in the database.
	Table = "status"
	// StatusToBuildTable is the table that holds the StatusToBuild relation/edge.
	StatusToBuildTable = "status"
	// StatusToBuildInverseTable is the table name for the Build entity.
	// It exists in this package in order to avoid circular dependency with the "build" package.
	StatusToBuildInverseTable = "builds"
	// StatusToBuildColumn is the table column denoting the StatusToBuild relation/edge.
	StatusToBuildColumn = "build_build_to_status"
	// StatusToProvisionedNetworkTable is the table that holds the StatusToProvisionedNetwork relation/edge.
	StatusToProvisionedNetworkTable = "status"
	// StatusToProvisionedNetworkInverseTable is the table name for the ProvisionedNetwork entity.
	// It exists in this package in order to avoid circular dependency with the "provisionednetwork" package.
	StatusToProvisionedNetworkInverseTable = "provisioned_networks"
	// StatusToProvisionedNetworkColumn is the table column denoting the StatusToProvisionedNetwork relation/edge.
	StatusToProvisionedNetworkColumn = "provisioned_network_provisioned_network_to_status"
	// StatusToProvisionedHostTable is the table that holds the StatusToProvisionedHost relation/edge.
	StatusToProvisionedHostTable = "status"
	// StatusToProvisionedHostInverseTable is the table name for the ProvisionedHost entity.
	// It exists in this package in order to avoid circular dependency with the "provisionedhost" package.
	StatusToProvisionedHostInverseTable = "provisioned_hosts"
	// StatusToProvisionedHostColumn is the table column denoting the StatusToProvisionedHost relation/edge.
	StatusToProvisionedHostColumn = "provisioned_host_provisioned_host_to_status"
	// StatusToProvisioningStepTable is the table that holds the StatusToProvisioningStep relation/edge.
	StatusToProvisioningStepTable = "status"
	// StatusToProvisioningStepInverseTable is the table name for the ProvisioningStep entity.
	// It exists in this package in order to avoid circular dependency with the "provisioningstep" package.
	StatusToProvisioningStepInverseTable = "provisioning_steps"
	// StatusToProvisioningStepColumn is the table column denoting the StatusToProvisioningStep relation/edge.
	StatusToProvisioningStepColumn = "provisioning_step_provisioning_step_to_status"
	// StatusToTeamTable is the table that holds the StatusToTeam relation/edge.
	StatusToTeamTable = "status"
	// StatusToTeamInverseTable is the table name for the Team entity.
	// It exists in this package in order to avoid circular dependency with the "team" package.
	StatusToTeamInverseTable = "teams"
	// StatusToTeamColumn is the table column denoting the StatusToTeam relation/edge.
	StatusToTeamColumn = "team_team_to_status"
	// StatusToPlanTable is the table that holds the StatusToPlan relation/edge.
	StatusToPlanTable = "status"
	// StatusToPlanInverseTable is the table name for the Plan entity.
	// It exists in this package in order to avoid circular dependency with the "plan" package.
	StatusToPlanInverseTable = "plans"
	// StatusToPlanColumn is the table column denoting the StatusToPlan relation/edge.
	StatusToPlanColumn = "plan_plan_to_status"
	// StatusToServerTaskTable is the table that holds the StatusToServerTask relation/edge.
	StatusToServerTaskTable = "status"
	// StatusToServerTaskInverseTable is the table name for the ServerTask entity.
	// It exists in this package in order to avoid circular dependency with the "servertask" package.
	StatusToServerTaskInverseTable = "server_tasks"
	// StatusToServerTaskColumn is the table column denoting the StatusToServerTask relation/edge.
	StatusToServerTaskColumn = "server_task_server_task_to_status"
	// StatusToAdhocPlanTable is the table that holds the StatusToAdhocPlan relation/edge.
	StatusToAdhocPlanTable = "status"
	// StatusToAdhocPlanInverseTable is the table name for the AdhocPlan entity.
	// It exists in this package in order to avoid circular dependency with the "adhocplan" package.
	StatusToAdhocPlanInverseTable = "adhoc_plans"
	// StatusToAdhocPlanColumn is the table column denoting the StatusToAdhocPlan relation/edge.
	StatusToAdhocPlanColumn = "adhoc_plan_adhoc_plan_to_status"
	// StatusToProvisioningScheduledStepTable is the table that holds the StatusToProvisioningScheduledStep relation/edge.
	StatusToProvisioningScheduledStepTable = "status"
	// StatusToProvisioningScheduledStepInverseTable is the table name for the ProvisioningScheduledStep entity.
	// It exists in this package in order to avoid circular dependency with the "provisioningscheduledstep" package.
	StatusToProvisioningScheduledStepInverseTable = "provisioning_scheduled_steps"
	// StatusToProvisioningScheduledStepColumn is the table column denoting the StatusToProvisioningScheduledStep relation/edge.
	StatusToProvisioningScheduledStepColumn = "provisioning_scheduled_step_provisioning_scheduled_step_to_status"
)

// Columns holds all SQL columns for status fields.
var Columns = []string{
	FieldID,
	FieldState,
	FieldStatusFor,
	FieldStartedAt,
	FieldEndedAt,
	FieldFailed,
	FieldCompleted,
	FieldError,
}

// ForeignKeys holds the SQL foreign-keys that are owned by the "status"
// table and are not defined as standalone fields in the schema.
var ForeignKeys = []string{
	"adhoc_plan_adhoc_plan_to_status",
	"build_build_to_status",
	"plan_plan_to_status",
	"provisioned_host_provisioned_host_to_status",
	"provisioned_network_provisioned_network_to_status",
	"provisioning_scheduled_step_provisioning_scheduled_step_to_status",
	"provisioning_step_provisioning_step_to_status",
	"server_task_server_task_to_status",
	"team_team_to_status",
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
	// DefaultFailed holds the default value on creation for the "failed" field.
	DefaultFailed bool
	// DefaultCompleted holds the default value on creation for the "completed" field.
	DefaultCompleted bool
	// DefaultID holds the default value on creation for the "id" field.
	DefaultID func() uuid.UUID
)

// State defines the type for the "state" enum field.
type State string

// State values.
const (
	StatePLANNING         State = "PLANNING"
	StateAWAITING         State = "AWAITING"
	StatePARENTAWAITING   State = "PARENTAWAITING"
	StateINPROGRESS       State = "INPROGRESS"
	StateFAILED           State = "FAILED"
	StateCOMPLETE         State = "COMPLETE"
	StateTAINTED          State = "TAINTED"
	StateTODELETE         State = "TODELETE"
	StateDELETEINPROGRESS State = "DELETEINPROGRESS"
	StateDELETED          State = "DELETED"
	StateTOREBUILD        State = "TOREBUILD"
	StateCANCELLED        State = "CANCELLED"
)

func (s State) String() string {
	return string(s)
}

// StateValidator is a validator for the "state" field enum values. It is called by the builders before save.
func StateValidator(s State) error {
	switch s {
	case StatePLANNING, StateAWAITING, StatePARENTAWAITING, StateINPROGRESS, StateFAILED, StateCOMPLETE, StateTAINTED, StateTODELETE, StateDELETEINPROGRESS, StateDELETED, StateTOREBUILD, StateCANCELLED:
		return nil
	default:
		return fmt.Errorf("status: invalid enum value for state field: %q", s)
	}
}

// StatusFor defines the type for the "status_for" enum field.
type StatusFor string

// StatusFor values.
const (
	StatusForBuild                     StatusFor = "Build"
	StatusForTeam                      StatusFor = "Team"
	StatusForPlan                      StatusFor = "Plan"
	StatusForProvisionedNetwork        StatusFor = "ProvisionedNetwork"
	StatusForProvisionedHost           StatusFor = "ProvisionedHost"
	StatusForProvisioningStep          StatusFor = "ProvisioningStep"
	StatusForProvisioningScheduledStep StatusFor = "ProvisioningScheduledStep"
	StatusForServerTask                StatusFor = "ServerTask"
)

func (sf StatusFor) String() string {
	return string(sf)
}

// StatusForValidator is a validator for the "status_for" field enum values. It is called by the builders before save.
func StatusForValidator(sf StatusFor) error {
	switch sf {
	case StatusForBuild, StatusForTeam, StatusForPlan, StatusForProvisionedNetwork, StatusForProvisionedHost, StatusForProvisioningStep, StatusForProvisioningScheduledStep, StatusForServerTask:
		return nil
	default:
		return fmt.Errorf("status: invalid enum value for status_for field: %q", sf)
	}
}

// MarshalGQL implements graphql.Marshaler interface.
func (s State) MarshalGQL(w io.Writer) {
	io.WriteString(w, strconv.Quote(s.String()))
}

// UnmarshalGQL implements graphql.Unmarshaler interface.
func (s *State) UnmarshalGQL(val interface{}) error {
	str, ok := val.(string)
	if !ok {
		return fmt.Errorf("enum %T must be a string", val)
	}
	*s = State(str)
	if err := StateValidator(*s); err != nil {
		return fmt.Errorf("%s is not a valid State", str)
	}
	return nil
}

// MarshalGQL implements graphql.Marshaler interface.
func (sf StatusFor) MarshalGQL(w io.Writer) {
	io.WriteString(w, strconv.Quote(sf.String()))
}

// UnmarshalGQL implements graphql.Unmarshaler interface.
func (sf *StatusFor) UnmarshalGQL(val interface{}) error {
	str, ok := val.(string)
	if !ok {
		return fmt.Errorf("enum %T must be a string", val)
	}
	*sf = StatusFor(str)
	if err := StatusForValidator(*sf); err != nil {
		return fmt.Errorf("%s is not a valid StatusFor", str)
	}
	return nil
}
