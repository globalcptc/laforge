// Code generated by ent, DO NOT EDIT.

package script

import (
	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"github.com/google/uuid"
)

const (
	// Label holds the string label denoting the script type in the database.
	Label = "script"
	// FieldID holds the string denoting the id field in the database.
	FieldID = "id"
	// FieldHCLID holds the string denoting the hcl_id field in the database.
	FieldHCLID = "hcl_id"
	// FieldName holds the string denoting the name field in the database.
	FieldName = "name"
	// FieldLanguage holds the string denoting the language field in the database.
	FieldLanguage = "language"
	// FieldDescription holds the string denoting the description field in the database.
	FieldDescription = "description"
	// FieldSource holds the string denoting the source field in the database.
	FieldSource = "source"
	// FieldSourceType holds the string denoting the source_type field in the database.
	FieldSourceType = "source_type"
	// FieldCooldown holds the string denoting the cooldown field in the database.
	FieldCooldown = "cooldown"
	// FieldTimeout holds the string denoting the timeout field in the database.
	FieldTimeout = "timeout"
	// FieldIgnoreErrors holds the string denoting the ignore_errors field in the database.
	FieldIgnoreErrors = "ignore_errors"
	// FieldArgs holds the string denoting the args field in the database.
	FieldArgs = "args"
	// FieldDisabled holds the string denoting the disabled field in the database.
	FieldDisabled = "disabled"
	// FieldVars holds the string denoting the vars field in the database.
	FieldVars = "vars"
	// FieldAbsPath holds the string denoting the abs_path field in the database.
	FieldAbsPath = "abs_path"
	// FieldTags holds the string denoting the tags field in the database.
	FieldTags = "tags"
	// EdgeUsers holds the string denoting the users edge name in mutations.
	EdgeUsers = "Users"
	// EdgeFindings holds the string denoting the findings edge name in mutations.
	EdgeFindings = "Findings"
	// EdgeEnvironment holds the string denoting the environment edge name in mutations.
	EdgeEnvironment = "Environment"
	// Table holds the table name of the script in the database.
	Table = "scripts"
	// UsersTable is the table that holds the Users relation/edge.
	UsersTable = "users"
	// UsersInverseTable is the table name for the User entity.
	// It exists in this package in order to avoid circular dependency with the "user" package.
	UsersInverseTable = "users"
	// UsersColumn is the table column denoting the Users relation/edge.
	UsersColumn = "script_users"
	// FindingsTable is the table that holds the Findings relation/edge.
	FindingsTable = "findings"
	// FindingsInverseTable is the table name for the Finding entity.
	// It exists in this package in order to avoid circular dependency with the "finding" package.
	FindingsInverseTable = "findings"
	// FindingsColumn is the table column denoting the Findings relation/edge.
	FindingsColumn = "script_findings"
	// EnvironmentTable is the table that holds the Environment relation/edge.
	EnvironmentTable = "scripts"
	// EnvironmentInverseTable is the table name for the Environment entity.
	// It exists in this package in order to avoid circular dependency with the "environment" package.
	EnvironmentInverseTable = "environments"
	// EnvironmentColumn is the table column denoting the Environment relation/edge.
	EnvironmentColumn = "environment_scripts"
)

// Columns holds all SQL columns for script fields.
var Columns = []string{
	FieldID,
	FieldHCLID,
	FieldName,
	FieldLanguage,
	FieldDescription,
	FieldSource,
	FieldSourceType,
	FieldCooldown,
	FieldTimeout,
	FieldIgnoreErrors,
	FieldArgs,
	FieldDisabled,
	FieldVars,
	FieldAbsPath,
	FieldTags,
}

// ForeignKeys holds the SQL foreign-keys that are owned by the "scripts"
// table and are not defined as standalone fields in the schema.
var ForeignKeys = []string{
	"environment_scripts",
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

// OrderOption defines the ordering options for the Script queries.
type OrderOption func(*sql.Selector)

// ByID orders the results by the id field.
func ByID(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldID, opts...).ToFunc()
}

// ByHCLID orders the results by the hcl_id field.
func ByHCLID(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldHCLID, opts...).ToFunc()
}

// ByName orders the results by the name field.
func ByName(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldName, opts...).ToFunc()
}

// ByLanguage orders the results by the language field.
func ByLanguage(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldLanguage, opts...).ToFunc()
}

// ByDescription orders the results by the description field.
func ByDescription(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldDescription, opts...).ToFunc()
}

// BySource orders the results by the source field.
func BySource(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldSource, opts...).ToFunc()
}

// BySourceType orders the results by the source_type field.
func BySourceType(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldSourceType, opts...).ToFunc()
}

// ByCooldown orders the results by the cooldown field.
func ByCooldown(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldCooldown, opts...).ToFunc()
}

// ByTimeout orders the results by the timeout field.
func ByTimeout(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldTimeout, opts...).ToFunc()
}

// ByIgnoreErrors orders the results by the ignore_errors field.
func ByIgnoreErrors(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldIgnoreErrors, opts...).ToFunc()
}

// ByDisabled orders the results by the disabled field.
func ByDisabled(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldDisabled, opts...).ToFunc()
}

// ByAbsPath orders the results by the abs_path field.
func ByAbsPath(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldAbsPath, opts...).ToFunc()
}

// ByUsersCount orders the results by Users count.
func ByUsersCount(opts ...sql.OrderTermOption) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborsCount(s, newUsersStep(), opts...)
	}
}

// ByUsers orders the results by Users terms.
func ByUsers(term sql.OrderTerm, terms ...sql.OrderTerm) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborTerms(s, newUsersStep(), append([]sql.OrderTerm{term}, terms...)...)
	}
}

// ByFindingsCount orders the results by Findings count.
func ByFindingsCount(opts ...sql.OrderTermOption) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborsCount(s, newFindingsStep(), opts...)
	}
}

// ByFindings orders the results by Findings terms.
func ByFindings(term sql.OrderTerm, terms ...sql.OrderTerm) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborTerms(s, newFindingsStep(), append([]sql.OrderTerm{term}, terms...)...)
	}
}

// ByEnvironmentField orders the results by Environment field.
func ByEnvironmentField(field string, opts ...sql.OrderTermOption) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborTerms(s, newEnvironmentStep(), sql.OrderByField(field, opts...))
	}
}
func newUsersStep() *sqlgraph.Step {
	return sqlgraph.NewStep(
		sqlgraph.From(Table, FieldID),
		sqlgraph.To(UsersInverseTable, FieldID),
		sqlgraph.Edge(sqlgraph.O2M, false, UsersTable, UsersColumn),
	)
}
func newFindingsStep() *sqlgraph.Step {
	return sqlgraph.NewStep(
		sqlgraph.From(Table, FieldID),
		sqlgraph.To(FindingsInverseTable, FieldID),
		sqlgraph.Edge(sqlgraph.O2M, false, FindingsTable, FindingsColumn),
	)
}
func newEnvironmentStep() *sqlgraph.Step {
	return sqlgraph.NewStep(
		sqlgraph.From(Table, FieldID),
		sqlgraph.To(EnvironmentInverseTable, FieldID),
		sqlgraph.Edge(sqlgraph.M2O, true, EnvironmentTable, EnvironmentColumn),
	)
}
