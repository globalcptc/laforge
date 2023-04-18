// Code generated by ent, DO NOT EDIT.

package scheduledstep

import (
	"fmt"
	"io"
	"strconv"

	"github.com/google/uuid"
)

const (
	// Label holds the string label denoting the scheduledstep type in the database.
	Label = "scheduled_step"
	// FieldID holds the string denoting the id field in the database.
	FieldID = "id"
	// FieldHclID holds the string denoting the hcl_id field in the database.
	FieldHclID = "hcl_id"
	// FieldName holds the string denoting the name field in the database.
	FieldName = "name"
	// FieldDescription holds the string denoting the description field in the database.
	FieldDescription = "description"
	// FieldStep holds the string denoting the step field in the database.
	FieldStep = "step"
	// FieldType holds the string denoting the type field in the database.
	FieldType = "type"
	// FieldSchedule holds the string denoting the schedule field in the database.
	FieldSchedule = "schedule"
	// FieldRunAt holds the string denoting the run_at field in the database.
	FieldRunAt = "run_at"
	// EdgeScheduledStepToEnvironment holds the string denoting the scheduledsteptoenvironment edge name in mutations.
	EdgeScheduledStepToEnvironment = "ScheduledStepToEnvironment"
	// Table holds the table name of the scheduledstep in the database.
	Table = "scheduled_steps"
	// ScheduledStepToEnvironmentTable is the table that holds the ScheduledStepToEnvironment relation/edge.
	ScheduledStepToEnvironmentTable = "scheduled_steps"
	// ScheduledStepToEnvironmentInverseTable is the table name for the Environment entity.
	// It exists in this package in order to avoid circular dependency with the "environment" package.
	ScheduledStepToEnvironmentInverseTable = "environments"
	// ScheduledStepToEnvironmentColumn is the table column denoting the ScheduledStepToEnvironment relation/edge.
	ScheduledStepToEnvironmentColumn = "environment_environment_to_scheduled_step"
)

// Columns holds all SQL columns for scheduledstep fields.
var Columns = []string{
	FieldID,
	FieldHclID,
	FieldName,
	FieldDescription,
	FieldStep,
	FieldType,
	FieldSchedule,
	FieldRunAt,
}

// ForeignKeys holds the SQL foreign-keys that are owned by the "scheduled_steps"
// table and are not defined as standalone fields in the schema.
var ForeignKeys = []string{
	"environment_environment_to_scheduled_step",
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

// Type defines the type for the "type" enum field.
type Type string

// Type values.
const (
	TypeCRON    Type = "CRON"
	TypeRUNONCE Type = "RUNONCE"
)

func (_type Type) String() string {
	return string(_type)
}

// TypeValidator is a validator for the "type" field enum values. It is called by the builders before save.
func TypeValidator(_type Type) error {
	switch _type {
	case TypeCRON, TypeRUNONCE:
		return nil
	default:
		return fmt.Errorf("scheduledstep: invalid enum value for type field: %q", _type)
	}
}

// MarshalGQL implements graphql.Marshaler interface.
func (_type Type) MarshalGQL(w io.Writer) {
	io.WriteString(w, strconv.Quote(_type.String()))
}

// UnmarshalGQL implements graphql.Unmarshaler interface.
func (_type *Type) UnmarshalGQL(val interface{}) error {
	str, ok := val.(string)
	if !ok {
		return fmt.Errorf("enum %T must be a string", val)
	}
	*_type = Type(str)
	if err := TypeValidator(*_type); err != nil {
		return fmt.Errorf("%s is not a valid Type", str)
	}
	return nil
}