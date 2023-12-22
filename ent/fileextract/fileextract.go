// Code generated by ent, DO NOT EDIT.

package fileextract

import (
	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"github.com/google/uuid"
)

const (
	// Label holds the string label denoting the fileextract type in the database.
	Label = "file_extract"
	// FieldID holds the string denoting the id field in the database.
	FieldID = "id"
	// FieldHCLID holds the string denoting the hcl_id field in the database.
	FieldHCLID = "hcl_id"
	// FieldSource holds the string denoting the source field in the database.
	FieldSource = "source"
	// FieldDestination holds the string denoting the destination field in the database.
	FieldDestination = "destination"
	// FieldType holds the string denoting the type field in the database.
	FieldType = "type"
	// FieldTags holds the string denoting the tags field in the database.
	FieldTags = "tags"
	// EdgeEnvironment holds the string denoting the environment edge name in mutations.
	EdgeEnvironment = "Environment"
	// Table holds the table name of the fileextract in the database.
	Table = "file_extracts"
	// EnvironmentTable is the table that holds the Environment relation/edge.
	EnvironmentTable = "file_extracts"
	// EnvironmentInverseTable is the table name for the Environment entity.
	// It exists in this package in order to avoid circular dependency with the "environment" package.
	EnvironmentInverseTable = "environments"
	// EnvironmentColumn is the table column denoting the Environment relation/edge.
	EnvironmentColumn = "environment_file_extracts"
)

// Columns holds all SQL columns for fileextract fields.
var Columns = []string{
	FieldID,
	FieldHCLID,
	FieldSource,
	FieldDestination,
	FieldType,
	FieldTags,
}

// ForeignKeys holds the SQL foreign-keys that are owned by the "file_extracts"
// table and are not defined as standalone fields in the schema.
var ForeignKeys = []string{
	"environment_file_extracts",
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

// OrderOption defines the ordering options for the FileExtract queries.
type OrderOption func(*sql.Selector)

// ByID orders the results by the id field.
func ByID(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldID, opts...).ToFunc()
}

// ByHCLID orders the results by the hcl_id field.
func ByHCLID(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldHCLID, opts...).ToFunc()
}

// BySource orders the results by the source field.
func BySource(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldSource, opts...).ToFunc()
}

// ByDestination orders the results by the destination field.
func ByDestination(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldDestination, opts...).ToFunc()
}

// ByType orders the results by the type field.
func ByType(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldType, opts...).ToFunc()
}

// ByEnvironmentField orders the results by Environment field.
func ByEnvironmentField(field string, opts ...sql.OrderTermOption) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborTerms(s, newEnvironmentStep(), sql.OrderByField(field, opts...))
	}
}
func newEnvironmentStep() *sqlgraph.Step {
	return sqlgraph.NewStep(
		sqlgraph.From(Table, FieldID),
		sqlgraph.To(EnvironmentInverseTable, FieldID),
		sqlgraph.Edge(sqlgraph.M2O, true, EnvironmentTable, EnvironmentColumn),
	)
}
