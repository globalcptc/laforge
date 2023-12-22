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
	// EdgeDNSToEnvironment holds the string denoting the dnstoenvironment edge name in mutations.
	EdgeDNSToEnvironment = "DNSToEnvironment"
	// EdgeDNSToCompetition holds the string denoting the dnstocompetition edge name in mutations.
	EdgeDNSToCompetition = "DNSToCompetition"
	// Table holds the table name of the dns in the database.
	Table = "dn_ss"
	// DNSToEnvironmentTable is the table that holds the DNSToEnvironment relation/edge. The primary key declared below.
	DNSToEnvironmentTable = "environment_EnvironmentToDNS"
	// DNSToEnvironmentInverseTable is the table name for the Environment entity.
	// It exists in this package in order to avoid circular dependency with the "environment" package.
	DNSToEnvironmentInverseTable = "environments"
	// DNSToCompetitionTable is the table that holds the DNSToCompetition relation/edge. The primary key declared below.
	DNSToCompetitionTable = "competition_CompetitionToDNS"
	// DNSToCompetitionInverseTable is the table name for the Competition entity.
	// It exists in this package in order to avoid circular dependency with the "competition" package.
	DNSToCompetitionInverseTable = "competitions"
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
	// DNSToEnvironmentPrimaryKey and DNSToEnvironmentColumn2 are the table columns denoting the
	// primary key for the DNSToEnvironment relation (M2M).
	DNSToEnvironmentPrimaryKey = []string{"environment_id", "dns_id"}
	// DNSToCompetitionPrimaryKey and DNSToCompetitionColumn2 are the table columns denoting the
	// primary key for the DNSToCompetition relation (M2M).
	DNSToCompetitionPrimaryKey = []string{"competition_id", "dns_id"}
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

// ByDNSToEnvironmentCount orders the results by DNSToEnvironment count.
func ByDNSToEnvironmentCount(opts ...sql.OrderTermOption) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborsCount(s, newDNSToEnvironmentStep(), opts...)
	}
}

// ByDNSToEnvironment orders the results by DNSToEnvironment terms.
func ByDNSToEnvironment(term sql.OrderTerm, terms ...sql.OrderTerm) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborTerms(s, newDNSToEnvironmentStep(), append([]sql.OrderTerm{term}, terms...)...)
	}
}

// ByDNSToCompetitionCount orders the results by DNSToCompetition count.
func ByDNSToCompetitionCount(opts ...sql.OrderTermOption) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborsCount(s, newDNSToCompetitionStep(), opts...)
	}
}

// ByDNSToCompetition orders the results by DNSToCompetition terms.
func ByDNSToCompetition(term sql.OrderTerm, terms ...sql.OrderTerm) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborTerms(s, newDNSToCompetitionStep(), append([]sql.OrderTerm{term}, terms...)...)
	}
}
func newDNSToEnvironmentStep() *sqlgraph.Step {
	return sqlgraph.NewStep(
		sqlgraph.From(Table, FieldID),
		sqlgraph.To(DNSToEnvironmentInverseTable, FieldID),
		sqlgraph.Edge(sqlgraph.M2M, true, DNSToEnvironmentTable, DNSToEnvironmentPrimaryKey...),
	)
}
func newDNSToCompetitionStep() *sqlgraph.Step {
	return sqlgraph.NewStep(
		sqlgraph.From(Table, FieldID),
		sqlgraph.To(DNSToCompetitionInverseTable, FieldID),
		sqlgraph.Edge(sqlgraph.M2M, true, DNSToCompetitionTable, DNSToCompetitionPrimaryKey...),
	)
}
