// Code generated by ent, DO NOT EDIT.

package validation

import (
	"fmt"
	"io"
	"strconv"

	"github.com/google/uuid"
)

const (
	// Label holds the string label denoting the validation type in the database.
	Label = "validation"
	// FieldID holds the string denoting the id field in the database.
	FieldID = "id"
	// FieldHclID holds the string denoting the hcl_id field in the database.
	FieldHclID = "hcl_id"
	// FieldValidationType holds the string denoting the validation_type field in the database.
	FieldValidationType = "validation_type"
	// FieldOutput holds the string denoting the output field in the database.
	FieldOutput = "output"
	// FieldState holds the string denoting the state field in the database.
	FieldState = "state"
	// FieldErrorMessage holds the string denoting the error_message field in the database.
	FieldErrorMessage = "error_message"
	// FieldHash holds the string denoting the hash field in the database.
	FieldHash = "hash"
	// FieldRegex holds the string denoting the regex field in the database.
	FieldRegex = "regex"
	// FieldIP holds the string denoting the ip field in the database.
	FieldIP = "ip"
	// FieldPort holds the string denoting the port field in the database.
	FieldPort = "port"
	// FieldHostname holds the string denoting the hostname field in the database.
	FieldHostname = "hostname"
	// FieldNameservers holds the string denoting the nameservers field in the database.
	FieldNameservers = "nameservers"
	// FieldPackageName holds the string denoting the package_name field in the database.
	FieldPackageName = "package_name"
	// FieldUsername holds the string denoting the username field in the database.
	FieldUsername = "username"
	// FieldGroupName holds the string denoting the group_name field in the database.
	FieldGroupName = "group_name"
	// FieldFilePath holds the string denoting the file_path field in the database.
	FieldFilePath = "file_path"
	// FieldSearchString holds the string denoting the search_string field in the database.
	FieldSearchString = "search_string"
	// FieldServiceName holds the string denoting the service_name field in the database.
	FieldServiceName = "service_name"
	// FieldServiceStatus holds the string denoting the service_status field in the database.
	FieldServiceStatus = "service_status"
	// FieldProcessName holds the string denoting the process_name field in the database.
	FieldProcessName = "process_name"
	// EdgeValidationToAgentTask holds the string denoting the validationtoagenttask edge name in mutations.
	EdgeValidationToAgentTask = "ValidationToAgentTask"
	// EdgeValidationToScript holds the string denoting the validationtoscript edge name in mutations.
	EdgeValidationToScript = "ValidationToScript"
	// EdgeValidationToEnvironment holds the string denoting the validationtoenvironment edge name in mutations.
	EdgeValidationToEnvironment = "ValidationToEnvironment"
	// Table holds the table name of the validation in the database.
	Table = "validations"
	// ValidationToAgentTaskTable is the table that holds the ValidationToAgentTask relation/edge.
	ValidationToAgentTaskTable = "validations"
	// ValidationToAgentTaskInverseTable is the table name for the AgentTask entity.
	// It exists in this package in order to avoid circular dependency with the "agenttask" package.
	ValidationToAgentTaskInverseTable = "agent_tasks"
	// ValidationToAgentTaskColumn is the table column denoting the ValidationToAgentTask relation/edge.
	ValidationToAgentTaskColumn = "agent_task_agent_task_to_validation"
	// ValidationToScriptTable is the table that holds the ValidationToScript relation/edge.
	ValidationToScriptTable = "scripts"
	// ValidationToScriptInverseTable is the table name for the Script entity.
	// It exists in this package in order to avoid circular dependency with the "script" package.
	ValidationToScriptInverseTable = "scripts"
	// ValidationToScriptColumn is the table column denoting the ValidationToScript relation/edge.
	ValidationToScriptColumn = "script_script_to_validation"
	// ValidationToEnvironmentTable is the table that holds the ValidationToEnvironment relation/edge. The primary key declared below.
	ValidationToEnvironmentTable = "environment_EnvironmentToValidation"
	// ValidationToEnvironmentInverseTable is the table name for the Environment entity.
	// It exists in this package in order to avoid circular dependency with the "environment" package.
	ValidationToEnvironmentInverseTable = "environments"
)

// Columns holds all SQL columns for validation fields.
var Columns = []string{
	FieldID,
	FieldHclID,
	FieldValidationType,
	FieldOutput,
	FieldState,
	FieldErrorMessage,
	FieldHash,
	FieldRegex,
	FieldIP,
	FieldPort,
	FieldHostname,
	FieldNameservers,
	FieldPackageName,
	FieldUsername,
	FieldGroupName,
	FieldFilePath,
	FieldSearchString,
	FieldServiceName,
	FieldServiceStatus,
	FieldProcessName,
}

// ForeignKeys holds the SQL foreign-keys that are owned by the "validations"
// table and are not defined as standalone fields in the schema.
var ForeignKeys = []string{
	"agent_task_agent_task_to_validation",
}

var (
	// ValidationToEnvironmentPrimaryKey and ValidationToEnvironmentColumn2 are the table columns denoting the
	// primary key for the ValidationToEnvironment relation (M2M).
	ValidationToEnvironmentPrimaryKey = []string{"environment_id", "validation_id"}
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
	// DefaultValidationType holds the default value on creation for the "validation_type" field.
	DefaultValidationType string
	// DefaultOutput holds the default value on creation for the "output" field.
	DefaultOutput string
	// DefaultErrorMessage holds the default value on creation for the "error_message" field.
	DefaultErrorMessage string
	// DefaultID holds the default value on creation for the "id" field.
	DefaultID func() uuid.UUID
)

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
		return fmt.Errorf("validation: invalid enum value for state field: %q", s)
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
