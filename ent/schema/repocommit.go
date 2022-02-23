package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/google/uuid"
)

// RepoCommit holds the schema definition for the RepoCommit entity.
type RepoCommit struct {
	ent.Schema
}

// Fields of the RepoCommit.
func (RepoCommit) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New),
		field.Int("revision"),
		field.String("hash"),
		field.JSON("author", object.Signature{}),
		field.JSON("committer", object.Signature{}),
		field.String("pgp_signature"),
		field.String("message"),
		field.String("tree_hash"),
		field.Strings("parent_hashes"),
	}
}

// Edges of the RepoCommit.
func (RepoCommit) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("RepoCommitToRepository", Repository.Type).Ref("RepositoryToRepoCommit").Unique(),
	}
}
