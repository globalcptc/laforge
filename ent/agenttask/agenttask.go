// Code generated by entc, DO NOT EDIT.

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
	// EdgeAgentTaskToProvisioningStep holds the string denoting the agenttasktoprovisioningstep edge name in mutations.
	EdgeAgentTaskToProvisioningStep = "AgentTaskToProvisioningStep"
	// EdgeAgentTaskToProvisionedHost holds the string denoting the agenttasktoprovisionedhost edge name in mutations.
	EdgeAgentTaskToProvisionedHost = "AgentTaskToProvisionedHost"
	// EdgeAgentTaskToAdhocPlan holds the string denoting the agenttasktoadhocplan edge name in mutations.
	EdgeAgentTaskToAdhocPlan = "AgentTaskToAdhocPlan"
	// EdgeAgentTaskToValidation holds the string denoting the agenttasktovalidation edge name in mutations.
	EdgeAgentTaskToValidation = "AgentTaskToValidation"
	// Table holds the table name of the agenttask in the database.
	Table = "agent_tasks"
	// AgentTaskToProvisioningStepTable is the table that holds the AgentTaskToProvisioningStep relation/edge.
	AgentTaskToProvisioningStepTable = "agent_tasks"
	// AgentTaskToProvisioningStepInverseTable is the table name for the ProvisioningStep entity.
	// It exists in this package in order to avoid circular dependency with the "provisioningstep" package.
	AgentTaskToProvisioningStepInverseTable = "provisioning_steps"
	// AgentTaskToProvisioningStepColumn is the table column denoting the AgentTaskToProvisioningStep relation/edge.
	AgentTaskToProvisioningStepColumn = "agent_task_agent_task_to_provisioning_step"
	// AgentTaskToProvisionedHostTable is the table that holds the AgentTaskToProvisionedHost relation/edge.
	AgentTaskToProvisionedHostTable = "agent_tasks"
	// AgentTaskToProvisionedHostInverseTable is the table name for the ProvisionedHost entity.
	// It exists in this package in order to avoid circular dependency with the "provisionedhost" package.
	AgentTaskToProvisionedHostInverseTable = "provisioned_hosts"
	// AgentTaskToProvisionedHostColumn is the table column denoting the AgentTaskToProvisionedHost relation/edge.
	AgentTaskToProvisionedHostColumn = "agent_task_agent_task_to_provisioned_host"
	// AgentTaskToAdhocPlanTable is the table that holds the AgentTaskToAdhocPlan relation/edge.
	AgentTaskToAdhocPlanTable = "adhoc_plans"
	// AgentTaskToAdhocPlanInverseTable is the table name for the AdhocPlan entity.
	// It exists in this package in order to avoid circular dependency with the "adhocplan" package.
	AgentTaskToAdhocPlanInverseTable = "adhoc_plans"
	// AgentTaskToAdhocPlanColumn is the table column denoting the AgentTaskToAdhocPlan relation/edge.
	AgentTaskToAdhocPlanColumn = "adhoc_plan_adhoc_plan_to_agent_task"
	// AgentTaskToValidationTable is the table that holds the AgentTaskToValidation relation/edge.
	AgentTaskToValidationTable = "validations"
	// AgentTaskToValidationInverseTable is the table name for the Validation entity.
	// It exists in this package in order to avoid circular dependency with the "validation" package.
	AgentTaskToValidationInverseTable = "validations"
	// AgentTaskToValidationColumn is the table column denoting the AgentTaskToValidation relation/edge.
	AgentTaskToValidationColumn = "agent_task_agent_task_to_validation"
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
	"agent_task_agent_task_to_provisioning_step",
	"agent_task_agent_task_to_provisioned_host",
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
	CommandVALIDATOR      Command = "VALIDATOR"
)

func (c Command) String() string {
	return string(c)
}

// CommandValidator is a validator for the "command" field enum values. It is called by the builders before save.
func CommandValidator(c Command) error {
	switch c {
	case CommandDEFAULT, CommandDELETE, CommandREBOOT, CommandEXTRACT, CommandDOWNLOAD, CommandCREATEUSER, CommandCREATEUSERPASS, CommandADDTOGROUP, CommandEXECUTE, CommandVALIDATE, CommandCHANGEPERMS, CommandAPPENDFILE, CommandVALIDATOR:
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
