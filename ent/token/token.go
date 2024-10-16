// Code generated by ent, DO NOT EDIT.

package token

import (
	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"github.com/google/uuid"
)

const (
	// Label holds the string label denoting the token type in the database.
	Label = "token"
	// FieldID holds the string denoting the id field in the database.
	FieldID = "id"
	// FieldToken holds the string denoting the token field in the database.
	FieldToken = "token"
	// FieldExpireAt holds the string denoting the expire_at field in the database.
	FieldExpireAt = "expire_at"
	// EdgeAuthUser holds the string denoting the authuser edge name in mutations.
	EdgeAuthUser = "AuthUser"
	// Table holds the table name of the token in the database.
	Table = "tokens"
	// AuthUserTable is the table that holds the AuthUser relation/edge.
	AuthUserTable = "tokens"
	// AuthUserInverseTable is the table name for the AuthUser entity.
	// It exists in this package in order to avoid circular dependency with the "authuser" package.
	AuthUserInverseTable = "auth_users"
	// AuthUserColumn is the table column denoting the AuthUser relation/edge.
	AuthUserColumn = "auth_user_tokens"
)

// Columns holds all SQL columns for token fields.
var Columns = []string{
	FieldID,
	FieldToken,
	FieldExpireAt,
}

// ForeignKeys holds the SQL foreign-keys that are owned by the "tokens"
// table and are not defined as standalone fields in the schema.
var ForeignKeys = []string{
	"auth_user_tokens",
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

// OrderOption defines the ordering options for the Token queries.
type OrderOption func(*sql.Selector)

// ByID orders the results by the id field.
func ByID(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldID, opts...).ToFunc()
}

// ByToken orders the results by the token field.
func ByToken(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldToken, opts...).ToFunc()
}

// ByExpireAt orders the results by the expire_at field.
func ByExpireAt(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldExpireAt, opts...).ToFunc()
}

// ByAuthUserField orders the results by AuthUser field.
func ByAuthUserField(field string, opts ...sql.OrderTermOption) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborTerms(s, newAuthUserStep(), sql.OrderByField(field, opts...))
	}
}
func newAuthUserStep() *sqlgraph.Step {
	return sqlgraph.NewStep(
		sqlgraph.From(Table, FieldID),
		sqlgraph.To(AuthUserInverseTable, FieldID),
		sqlgraph.Edge(sqlgraph.M2O, true, AuthUserTable, AuthUserColumn),
	)
}
