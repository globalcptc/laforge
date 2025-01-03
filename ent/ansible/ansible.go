// Code generated by ent, DO NOT EDIT.

package ansible

import (
	"fmt"
	"io"
	"strconv"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"github.com/google/uuid"
)

const (
	// Label holds the string label denoting the ansible type in the database.
	Label = "ansible"
	// FieldID holds the string denoting the id field in the database.
	FieldID = "id"
	// FieldName holds the string denoting the name field in the database.
	FieldName = "name"
	// FieldHCLID holds the string denoting the hcl_id field in the database.
	FieldHCLID = "hcl_id"
	// FieldDescription holds the string denoting the description field in the database.
	FieldDescription = "description"
	// FieldSource holds the string denoting the source field in the database.
	FieldSource = "source"
	// FieldPlaybookName holds the string denoting the playbook_name field in the database.
	FieldPlaybookName = "playbook_name"
	// FieldMethod holds the string denoting the method field in the database.
	FieldMethod = "method"
	// FieldInventory holds the string denoting the inventory field in the database.
	FieldInventory = "inventory"
	// FieldAbsPath holds the string denoting the abs_path field in the database.
	FieldAbsPath = "abs_path"
	// FieldTags holds the string denoting the tags field in the database.
	FieldTags = "tags"
	// EdgeUsers holds the string denoting the users edge name in mutations.
	EdgeUsers = "Users"
	// EdgeEnvironment holds the string denoting the environment edge name in mutations.
	EdgeEnvironment = "Environment"
	// Table holds the table name of the ansible in the database.
	Table = "ansibles"
	// UsersTable is the table that holds the Users relation/edge.
	UsersTable = "users"
	// UsersInverseTable is the table name for the User entity.
	// It exists in this package in order to avoid circular dependency with the "user" package.
	UsersInverseTable = "users"
	// UsersColumn is the table column denoting the Users relation/edge.
	UsersColumn = "ansible_users"
	// EnvironmentTable is the table that holds the Environment relation/edge.
	EnvironmentTable = "ansibles"
	// EnvironmentInverseTable is the table name for the Environment entity.
	// It exists in this package in order to avoid circular dependency with the "environment" package.
	EnvironmentInverseTable = "environments"
	// EnvironmentColumn is the table column denoting the Environment relation/edge.
	EnvironmentColumn = "environment_ansibles"
)

// Columns holds all SQL columns for ansible fields.
var Columns = []string{
	FieldID,
	FieldName,
	FieldHCLID,
	FieldDescription,
	FieldSource,
	FieldPlaybookName,
	FieldMethod,
	FieldInventory,
	FieldAbsPath,
	FieldTags,
}

// ForeignKeys holds the SQL foreign-keys that are owned by the "ansibles"
// table and are not defined as standalone fields in the schema.
var ForeignKeys = []string{
	"environment_ansibles",
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

// Method defines the type for the "method" enum field.
type Method string

// Method values.
const (
	MethodLOCAL Method = "LOCAL"
)

func (m Method) String() string {
	return string(m)
}

// MethodValidator is a validator for the "method" field enum values. It is called by the builders before save.
func MethodValidator(m Method) error {
	switch m {
	case MethodLOCAL:
		return nil
	default:
		return fmt.Errorf("ansible: invalid enum value for method field: %q", m)
	}
}

// OrderOption defines the ordering options for the Ansible queries.
type OrderOption func(*sql.Selector)

// ByID orders the results by the id field.
func ByID(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldID, opts...).ToFunc()
}

// ByName orders the results by the name field.
func ByName(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldName, opts...).ToFunc()
}

// ByHCLID orders the results by the hcl_id field.
func ByHCLID(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldHCLID, opts...).ToFunc()
}

// ByDescription orders the results by the description field.
func ByDescription(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldDescription, opts...).ToFunc()
}

// BySource orders the results by the source field.
func BySource(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldSource, opts...).ToFunc()
}

// ByPlaybookName orders the results by the playbook_name field.
func ByPlaybookName(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldPlaybookName, opts...).ToFunc()
}

// ByMethod orders the results by the method field.
func ByMethod(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldMethod, opts...).ToFunc()
}

// ByInventory orders the results by the inventory field.
func ByInventory(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldInventory, opts...).ToFunc()
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
func newEnvironmentStep() *sqlgraph.Step {
	return sqlgraph.NewStep(
		sqlgraph.From(Table, FieldID),
		sqlgraph.To(EnvironmentInverseTable, FieldID),
		sqlgraph.Edge(sqlgraph.M2O, true, EnvironmentTable, EnvironmentColumn),
	)
}

// MarshalGQL implements graphql.Marshaler interface.
func (e Method) MarshalGQL(w io.Writer) {
	io.WriteString(w, strconv.Quote(e.String()))
}

// UnmarshalGQL implements graphql.Unmarshaler interface.
func (e *Method) UnmarshalGQL(val interface{}) error {
	str, ok := val.(string)
	if !ok {
		return fmt.Errorf("enum %T must be a string", val)
	}
	*e = Method(str)
	if err := MethodValidator(*e); err != nil {
		return fmt.Errorf("%s is not a valid Method", str)
	}
	return nil
}
