// Code generated by ent, DO NOT EDIT.

package repocommit

import (
	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"github.com/google/uuid"
)

const (
	// Label holds the string label denoting the repocommit type in the database.
	Label = "repo_commit"
	// FieldID holds the string denoting the id field in the database.
	FieldID = "id"
	// FieldRevision holds the string denoting the revision field in the database.
	FieldRevision = "revision"
	// FieldHash holds the string denoting the hash field in the database.
	FieldHash = "hash"
	// FieldAuthor holds the string denoting the author field in the database.
	FieldAuthor = "author"
	// FieldCommitter holds the string denoting the committer field in the database.
	FieldCommitter = "committer"
	// FieldPgpSignature holds the string denoting the pgp_signature field in the database.
	FieldPgpSignature = "pgp_signature"
	// FieldMessage holds the string denoting the message field in the database.
	FieldMessage = "message"
	// FieldTreeHash holds the string denoting the tree_hash field in the database.
	FieldTreeHash = "tree_hash"
	// FieldParentHashes holds the string denoting the parent_hashes field in the database.
	FieldParentHashes = "parent_hashes"
	// EdgeRepository holds the string denoting the repository edge name in mutations.
	EdgeRepository = "Repository"
	// Table holds the table name of the repocommit in the database.
	Table = "repo_commits"
	// RepositoryTable is the table that holds the Repository relation/edge.
	RepositoryTable = "repo_commits"
	// RepositoryInverseTable is the table name for the Repository entity.
	// It exists in this package in order to avoid circular dependency with the "repository" package.
	RepositoryInverseTable = "repositories"
	// RepositoryColumn is the table column denoting the Repository relation/edge.
	RepositoryColumn = "repository_repo_commits"
)

// Columns holds all SQL columns for repocommit fields.
var Columns = []string{
	FieldID,
	FieldRevision,
	FieldHash,
	FieldAuthor,
	FieldCommitter,
	FieldPgpSignature,
	FieldMessage,
	FieldTreeHash,
	FieldParentHashes,
}

// ForeignKeys holds the SQL foreign-keys that are owned by the "repo_commits"
// table and are not defined as standalone fields in the schema.
var ForeignKeys = []string{
	"repository_repo_commits",
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

// OrderOption defines the ordering options for the RepoCommit queries.
type OrderOption func(*sql.Selector)

// ByID orders the results by the id field.
func ByID(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldID, opts...).ToFunc()
}

// ByRevision orders the results by the revision field.
func ByRevision(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldRevision, opts...).ToFunc()
}

// ByHash orders the results by the hash field.
func ByHash(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldHash, opts...).ToFunc()
}

// ByPgpSignature orders the results by the pgp_signature field.
func ByPgpSignature(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldPgpSignature, opts...).ToFunc()
}

// ByMessage orders the results by the message field.
func ByMessage(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldMessage, opts...).ToFunc()
}

// ByTreeHash orders the results by the tree_hash field.
func ByTreeHash(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldTreeHash, opts...).ToFunc()
}

// ByRepositoryField orders the results by Repository field.
func ByRepositoryField(field string, opts ...sql.OrderTermOption) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborTerms(s, newRepositoryStep(), sql.OrderByField(field, opts...))
	}
}
func newRepositoryStep() *sqlgraph.Step {
	return sqlgraph.NewStep(
		sqlgraph.From(Table, FieldID),
		sqlgraph.To(RepositoryInverseTable, FieldID),
		sqlgraph.Edge(sqlgraph.M2O, true, RepositoryTable, RepositoryColumn),
	)
}
