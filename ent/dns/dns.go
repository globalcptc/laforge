// Code generated by ent, DO NOT EDIT.

package dns

import (
	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"github.com/google/uuid"
)

const (
	// Label holds the string label denoting the dns type in the database.
	Label = "dns"
	// FieldID holds the string denoting the id field in the database.
	FieldID = "id"
	// FieldHCLID holds the string denoting the hcl_id field in the database.
	FieldHCLID = "hcl_id"
	// FieldType holds the string denoting the type field in the database.
	FieldType = "type"
	// FieldRootDomain holds the string denoting the root_domain field in the database.
	FieldRootDomain = "root_domain"
	// FieldDNSServers holds the string denoting the dns_servers field in the database.
	FieldDNSServers = "dns_servers"
	// FieldNtpServers holds the string denoting the ntp_servers field in the database.
	FieldNtpServers = "ntp_servers"
	// FieldConfig holds the string denoting the config field in the database.
	FieldConfig = "config"
	// EdgeEnvironments holds the string denoting the environments edge name in mutations.
	EdgeEnvironments = "Environments"
	// EdgeCompetitions holds the string denoting the competitions edge name in mutations.
	EdgeCompetitions = "Competitions"
	// Table holds the table name of the dns in the database.
	Table = "dn_ss"
	// EnvironmentsTable is the table that holds the Environments relation/edge. The primary key declared below.
	EnvironmentsTable = "environment_DNS"
	// EnvironmentsInverseTable is the table name for the Environment entity.
	// It exists in this package in order to avoid circular dependency with the "environment" package.
	EnvironmentsInverseTable = "environments"
	// CompetitionsTable is the table that holds the Competitions relation/edge. The primary key declared below.
	CompetitionsTable = "competition_DNS"
	// CompetitionsInverseTable is the table name for the Competition entity.
	// It exists in this package in order to avoid circular dependency with the "competition" package.
	CompetitionsInverseTable = "competitions"
)

// Columns holds all SQL columns for dns fields.
var Columns = []string{
	FieldID,
	FieldHCLID,
	FieldType,
	FieldRootDomain,
	FieldDNSServers,
	FieldNtpServers,
	FieldConfig,
}

var (
	// EnvironmentsPrimaryKey and EnvironmentsColumn2 are the table columns denoting the
	// primary key for the Environments relation (M2M).
	EnvironmentsPrimaryKey = []string{"environment_id", "dns_id"}
	// CompetitionsPrimaryKey and CompetitionsColumn2 are the table columns denoting the
	// primary key for the Competitions relation (M2M).
	CompetitionsPrimaryKey = []string{"competition_id", "dns_id"}
)

// ValidColumn reports if the column name is valid (part of the table columns).
func ValidColumn(column string) bool {
	for i := range Columns {
		if column == Columns[i] {
			return true
		}
	}
	return false
}

var (
	// DefaultID holds the default value on creation for the "id" field.
	DefaultID func() uuid.UUID
)

// OrderOption defines the ordering options for the DNS queries.
type OrderOption func(*sql.Selector)

// ByID orders the results by the id field.
func ByID(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldID, opts...).ToFunc()
}

// ByHCLID orders the results by the hcl_id field.
func ByHCLID(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldHCLID, opts...).ToFunc()
}

// ByType orders the results by the type field.
func ByType(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldType, opts...).ToFunc()
}

// ByRootDomain orders the results by the root_domain field.
func ByRootDomain(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldRootDomain, opts...).ToFunc()
}

// ByEnvironmentsCount orders the results by Environments count.
func ByEnvironmentsCount(opts ...sql.OrderTermOption) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborsCount(s, newEnvironmentsStep(), opts...)
	}
}

// ByEnvironments orders the results by Environments terms.
func ByEnvironments(term sql.OrderTerm, terms ...sql.OrderTerm) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborTerms(s, newEnvironmentsStep(), append([]sql.OrderTerm{term}, terms...)...)
	}
}

// ByCompetitionsCount orders the results by Competitions count.
func ByCompetitionsCount(opts ...sql.OrderTermOption) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborsCount(s, newCompetitionsStep(), opts...)
	}
}

// ByCompetitions orders the results by Competitions terms.
func ByCompetitions(term sql.OrderTerm, terms ...sql.OrderTerm) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborTerms(s, newCompetitionsStep(), append([]sql.OrderTerm{term}, terms...)...)
	}
}
func newEnvironmentsStep() *sqlgraph.Step {
	return sqlgraph.NewStep(
		sqlgraph.From(Table, FieldID),
		sqlgraph.To(EnvironmentsInverseTable, FieldID),
		sqlgraph.Edge(sqlgraph.M2M, true, EnvironmentsTable, EnvironmentsPrimaryKey...),
	)
}
func newCompetitionsStep() *sqlgraph.Step {
	return sqlgraph.NewStep(
		sqlgraph.From(Table, FieldID),
		sqlgraph.To(CompetitionsInverseTable, FieldID),
		sqlgraph.Edge(sqlgraph.M2M, true, CompetitionsTable, CompetitionsPrimaryKey...),
	)
}
