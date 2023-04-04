// Code generated by ent, DO NOT EDIT.

package filedelete

import (
	"github.com/google/uuid"
)

const (
	// Label holds the string label denoting the filedelete type in the database.
	Label = "file_delete"
	// FieldID holds the string denoting the id field in the database.
	FieldID = "id"
	// FieldHclID holds the string denoting the hcl_id field in the database.
	FieldHclID = "hcl_id"
	// FieldPath holds the string denoting the path field in the database.
	FieldPath = "path"
	// FieldTags holds the string denoting the tags field in the database.
	FieldTags = "tags"
	// EdgeFileDeleteToEnvironment holds the string denoting the filedeletetoenvironment edge name in mutations.
	EdgeFileDeleteToEnvironment = "FileDeleteToEnvironment"
	// Table holds the table name of the filedelete in the database.
	Table = "file_deletes"
	// FileDeleteToEnvironmentTable is the table that holds the FileDeleteToEnvironment relation/edge.
	FileDeleteToEnvironmentTable = "file_deletes"
	// FileDeleteToEnvironmentInverseTable is the table name for the Environment entity.
	// It exists in this package in order to avoid circular dependency with the "environment" package.
	FileDeleteToEnvironmentInverseTable = "environments"
	// FileDeleteToEnvironmentColumn is the table column denoting the FileDeleteToEnvironment relation/edge.
	FileDeleteToEnvironmentColumn = "environment_environment_to_file_delete"
)

// Columns holds all SQL columns for filedelete fields.
var Columns = []string{
	FieldID,
	FieldHclID,
	FieldPath,
	FieldTags,
}

// ForeignKeys holds the SQL foreign-keys that are owned by the "file_deletes"
// table and are not defined as standalone fields in the schema.
var ForeignKeys = []string{
	"environment_environment_to_file_delete",
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
