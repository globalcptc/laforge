// Code generated by ent, DO NOT EDIT.

package agenttask

import (
	"fmt"
	"io"
	"strconv"

	"github.com/google/uuid"
)

const (
	// Label holds the string label denoting the agenttask type in the database.
	Label = "agent_task"
	// FieldID holds the string denoting the id field in the database.
	FieldID = "id"
	// FieldCommand holds the string denoting the command field in the database.
	FieldCommand = "command"
	// FieldArgs holds the string denoting the args field in the database.
	FieldArgs = "args"
	// FieldNumber holds the string denoting the number field in the database.
	FieldNumber = "number"
	// FieldOutput holds the string denoting the output field in the database.
	FieldOutput = "output"
	// FieldState holds the string denoting the state field in the database.
	FieldState = "state"
	// FieldErrorMessage holds the string denoting the error_message field in the database.
	FieldErrorMessage = "error_message"
	// EdgeProvisioningStep holds the string denoting the provisioningstep edge name in mutations.
	EdgeProvisioningStep = "ProvisioningStep"
	// EdgeProvisioningScheduledStep holds the string denoting the provisioningscheduledstep edge name in mutations.
	EdgeProvisioningScheduledStep = "ProvisioningScheduledStep"
	// EdgeProvisionedHost holds the string denoting the provisionedhost edge name in mutations.
	EdgeProvisionedHost = "ProvisionedHost"
	// EdgeAdhocPlan holds the string denoting the adhocplan edge name in mutations.
	EdgeAdhocPlan = "AdhocPlan"
	// Table holds the table name of the agenttask in the database.
	Table = "agent_tasks"
	// ProvisioningStepTable is the table that holds the ProvisioningStep relation/edge.
	ProvisioningStepTable = "agent_tasks"
	// ProvisioningStepInverseTable is the table name for the ProvisioningStep entity.
	// It exists in this package in order to avoid circular dependency with the "provisioningstep" package.
	ProvisioningStepInverseTable = "provisioning_steps"
	// ProvisioningStepColumn is the table column denoting the ProvisioningStep relation/edge.
	ProvisioningStepColumn = "agent_task_provisioning_step"
	// ProvisioningScheduledStepTable is the table that holds the ProvisioningScheduledStep relation/edge.
	ProvisioningScheduledStepTable = "provisioning_scheduled_steps"
	// ProvisioningScheduledStepInverseTable is the table name for the ProvisioningScheduledStep entity.
	// It exists in this package in order to avoid circular dependency with the "provisioningscheduledstep" package.
	ProvisioningScheduledStepInverseTable = "provisioning_scheduled_steps"
	// ProvisioningScheduledStepColumn is the table column denoting the ProvisioningScheduledStep relation/edge.
	ProvisioningScheduledStepColumn = "agent_task_provisioning_scheduled_step"
	// ProvisionedHostTable is the table that holds the ProvisionedHost relation/edge.
	ProvisionedHostTable = "agent_tasks"
	// ProvisionedHostInverseTable is the table name for the ProvisionedHost entity.
	// It exists in this package in order to avoid circular dependency with the "provisionedhost" package.
	ProvisionedHostInverseTable = "provisioned_hosts"
	// ProvisionedHostColumn is the table column denoting the ProvisionedHost relation/edge.
	ProvisionedHostColumn = "agent_task_provisioned_host"
	// AdhocPlanTable is the table that holds the AdhocPlan relation/edge.
	AdhocPlanTable = "adhoc_plans"
	// AdhocPlanInverseTable is the table name for the AdhocPlan entity.
	// It exists in this package in order to avoid circular dependency with the "adhocplan" package.
	AdhocPlanInverseTable = "adhoc_plans"
	// AdhocPlanColumn is the table column denoting the AdhocPlan relation/edge.
	AdhocPlanColumn = "adhoc_plan_agent_task"
)

// Columns holds all SQL columns for agenttask fields.
var Columns = []string{
	FieldID,
	FieldCommand,
	FieldArgs,
	FieldNumber,
	FieldOutput,
	FieldState,
	FieldErrorMessage,
}

// ForeignKeys holds the SQL foreign-keys that are owned by the "agent_tasks"
// table and are not defined as standalone fields in the schema.
var ForeignKeys = []string{
	"agent_task_provisioning_step",
	"agent_task_provisioned_host",
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
	// DefaultOutput holds the default value on creation for the "output" field.
	DefaultOutput string
	// DefaultErrorMessage holds the default value on creation for the "error_message" field.
	DefaultErrorMessage string
	// DefaultID holds the default value on creation for the "id" field.
	DefaultID func() uuid.UUID
)

// Command defines the type for the "command" enum field.
type Command string

// Command values.
const (
	CommandDEFAULT        Command = "DEFAULT"
	CommandDELETE         Command = "DELETE"
	CommandREBOOT         Command = "REBOOT"
	CommandEXTRACT        Command = "EXTRACT"
	CommandDOWNLOAD       Command = "DOWNLOAD"
	CommandCREATEUSER     Command = "CREATEUSER"
	CommandCREATEUSERPASS Command = "CREATEUSERPASS"
	CommandADDTOGROUP     Command = "ADDTOGROUP"
	CommandEXECUTE        Command = "EXECUTE"
	CommandVALIDATE       Command = "VALIDATE"
	CommandCHANGEPERMS    Command = "CHANGEPERMS"
	CommandAPPENDFILE     Command = "APPENDFILE"
	CommandANSIBLE        Command = "ANSIBLE"
)

func (c Command) String() string {
	return string(c)
}

// CommandValidator is a validator for the "command" field enum values. It is called by the builders before save.
func CommandValidator(c Command) error {
	switch c {
	case CommandDEFAULT, CommandDELETE, CommandREBOOT, CommandEXTRACT, CommandDOWNLOAD, CommandCREATEUSER, CommandCREATEUSERPASS, CommandADDTOGROUP, CommandEXECUTE, CommandVALIDATE, CommandCHANGEPERMS, CommandAPPENDFILE, CommandANSIBLE:
		return nil
	default:
		return fmt.Errorf("agenttask: invalid enum value for command field: %q", c)
	}
}

// State defines the type for the "state" enum field.
type State string

// State values.
const (
	StateAWAITING   State = "AWAITING"
	StateINPROGRESS State = "INPROGRESS"
	StateFAILED     State = "FAILED"
	StateCOMPLETE   State = "COMPLETE"
)

func (s State) String() string {
	return string(s)
}

// StateValidator is a validator for the "state" field enum values. It is called by the builders before save.
func StateValidator(s State) error {
	switch s {
	case StateAWAITING, StateINPROGRESS, StateFAILED, StateCOMPLETE:
		return nil
	default:
		return fmt.Errorf("agenttask: invalid enum value for state field: %q", s)
	}
}

// MarshalGQL implements graphql.Marshaler interface.
func (c Command) MarshalGQL(w io.Writer) {
	io.WriteString(w, strconv.Quote(c.String()))
}

// UnmarshalGQL implements graphql.Unmarshaler interface.
func (c *Command) UnmarshalGQL(val interface{}) error {
	str, ok := val.(string)
	if !ok {
		return fmt.Errorf("enum %T must be a string", val)
	}
	*c = Command(str)
	if err := CommandValidator(*c); err != nil {
		return fmt.Errorf("%s is not a valid Command", str)
	}
	return nil
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
