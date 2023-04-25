// Code generated by ent, DO NOT EDIT.

package command

import (
	"github.com/google/uuid"
)

const (
	// Label holds the string label denoting the command type in the database.
	Label = "command"
	// FieldID holds the string denoting the id field in the database.
	FieldID = "id"
	// FieldHclID holds the string denoting the hcl_id field in the database.
	FieldHclID = "hcl_id"
	// FieldName holds the string denoting the name field in the database.
	FieldName = "name"
	// FieldDescription holds the string denoting the description field in the database.
	FieldDescription = "description"
	// FieldProgram holds the string denoting the program field in the database.
	FieldProgram = "program"
	// FieldArgs holds the string denoting the args field in the database.
	FieldArgs = "args"
	// FieldIgnoreErrors holds the string denoting the ignore_errors field in the database.
	FieldIgnoreErrors = "ignore_errors"
	// FieldDisabled holds the string denoting the disabled field in the database.
	FieldDisabled = "disabled"
	// FieldCooldown holds the string denoting the cooldown field in the database.
	FieldCooldown = "cooldown"
	// FieldTimeout holds the string denoting the timeout field in the database.
	FieldTimeout = "timeout"
	// FieldVars holds the string denoting the vars field in the database.
	FieldVars = "vars"
	// FieldTags holds the string denoting the tags field in the database.
	FieldTags = "tags"
	// EdgeUsers holds the string denoting the users edge name in mutations.
	EdgeUsers = "Users"
	// EdgeEnvironment holds the string denoting the environment edge name in mutations.
	EdgeEnvironment = "Environment"
	// Table holds the table name of the command in the database.
	Table = "commands"
	// UsersTable is the table that holds the Users relation/edge.
	UsersTable = "users"
	// UsersInverseTable is the table name for the User entity.
	// It exists in this package in order to avoid circular dependency with the "user" package.
	UsersInverseTable = "users"
	// UsersColumn is the table column denoting the Users relation/edge.
	UsersColumn = "command_users"
	// EnvironmentTable is the table that holds the Environment relation/edge.
	EnvironmentTable = "commands"
	// EnvironmentInverseTable is the table name for the Environment entity.
	// It exists in this package in order to avoid circular dependency with the "environment" package.
	EnvironmentInverseTable = "environments"
	// EnvironmentColumn is the table column denoting the Environment relation/edge.
	EnvironmentColumn = "environment_commands"
)

// Columns holds all SQL columns for command fields.
var Columns = []string{
	FieldID,
	FieldHclID,
	FieldName,
	FieldDescription,
	FieldProgram,
	FieldArgs,
	FieldIgnoreErrors,
	FieldDisabled,
	FieldCooldown,
	FieldTimeout,
	FieldVars,
	FieldTags,
}

// ForeignKeys holds the SQL foreign-keys that are owned by the "commands"
// table and are not defined as standalone fields in the schema.
var ForeignKeys = []string{
	"environment_commands",
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
	// CooldownValidator is a validator for the "cooldown" field. It is called by the builders before save.
	CooldownValidator func(int) error
	// TimeoutValidator is a validator for the "timeout" field. It is called by the builders before save.
	TimeoutValidator func(int) error
	// DefaultID holds the default value on creation for the "id" field.
	DefaultID func() uuid.UUID
)
