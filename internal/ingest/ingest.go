package ingest

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/globalcptc/laforge/internal/db"
	"github.com/globalcptc/laforge/internal/loader"
	"github.com/globalcptc/laforge/internal/render"
)

// ValidationIssue is one problem found while validating a commit, whether
// from schema/cross-file checks (internal/loader) or from actually
// rendering every script against every host in every team
// (internal/render.CheckAll) -- collapsed into one flat list because "the
// api ingests the commit, runs full schema validation" per the plan means
// the whole `laforge check` pass, not just the schema half of it.
type ValidationIssue struct {
	File        string `json:"file,omitempty"`
	Line        int    `json:"line,omitempty"`
	Environment string `json:"environment,omitempty"`
	Team        int    `json:"team,omitempty"`
	As          string `json:"as,omitempty"`
	Message     string `json:"message"`
}

type Result struct {
	Revision db.ContentRevision
	Valid    bool
	Issues   []ValidationIssue
}

// ValidateAndStore runs the full `laforge check` pipeline against workDir
// and records the outcome as a new content_revision under repoID, tied to
// commitSHA and ref. On success it also persists every object the commit
// resolved to -- environments, networks, hosts, containers, scripts,
// people, and each environment's placements -- so the database reflects
// exactly what that commit's content is, the same way `laforge check`
// would describe it from a checkout.
//
// It never returns an error for bad *content* -- a schema violation, a
// broken template reference, anything `laforge check` itself would report
// -- that's precisely what content_revision.valid and Result.Issues are
// for. An error return means something infrastructural went wrong (the
// database, an I/O failure reading workDir) and the caller should treat
// the whole ingest attempt as failed, distinct from "the commit itself is
// invalid."
//
// Everything written to Postgres -- the content_revision row, its
// validation result, and (when valid) every child row persistContent
// writes -- happens inside one transaction. A commit with a dozen hosts
// and a few hundred scripts is a lot of individual INSERTs; without a
// transaction, a failure partway through (a dropped connection, a
// context timeout) would leave a content_revision marked valid with only
// some of its content actually persisted -- silently wrong data, not a
// visible error. Taking the pool rather than an already-open *db.Queries
// is what makes that possible: this function owns the transaction's
// whole lifetime.
func ValidateAndStore(ctx context.Context, pool *pgxpool.Pool, repoID pgtype.UUID, commitSHA, ref, workDir string) (*Result, error) {
	c, err := loader.Load(workDir)
	if err != nil {
		return nil, fmt.Errorf("loading content: %w", err)
	}

	var issues []ValidationIssue
	for _, e := range c.Errors {
		issues = append(issues, ValidationIssue{File: e.File, Line: e.Line, Message: e.Message})
	}
	// Only attempt to render on top of content that's at least
	// schema-valid, matching the CLI's own `runCheck`: rendering over
	// known-broken data would just produce noise on top of the real
	// problem.
	if len(c.Errors) == 0 {
		for _, e := range render.CheckAll(workDir, c) {
			issues = append(issues, ValidationIssue{Environment: e.Environment, Team: e.Team, As: e.As, Message: e.Message})
		}
	}
	valid := len(issues) == 0

	// Idempotent adopt: the same commit ingested twice -- a webhook re-delivery,
	// or an on-demand ingest (configured-build create/sync, internal/api) followed
	// by the push webhook -- must return the EXISTING revision rather than fail on
	// the (repository_id, commit_sha) unique key. A SHA uniquely identifies its
	// content, so the existing row is authoritative; its stored `valid` reflects
	// whether content was actually persisted for it.
	if existing, err := db.New(pool).GetContentRevisionByRepoAndSHA(ctx, db.GetContentRevisionByRepoAndSHAParams{
		RepositoryID: repoID, CommitSha: commitSHA,
	}); err == nil {
		return &Result{Revision: existing, Valid: existing.Valid, Issues: issues}, nil
	}

	issuesJSON, err := jsonField(issues, "[]")
	if err != nil {
		return nil, fmt.Errorf("encoding validation issues: %w", err)
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback(ctx) // no-op once Commit succeeds below
	q := db.New(tx)

	message, author, committedAt := commitMeta(ctx, workDir)
	rev, err := q.CreateContentRevision(ctx, db.CreateContentRevisionParams{
		RepositoryID:  repoID,
		CommitSha:     commitSHA,
		Ref:           strPtrOrNil(ref),
		CommitMessage: strPtrOrNil(message),
		CommitAuthor:  strPtrOrNil(author),
		CommittedAt:   committedAt,
	})
	if err != nil {
		return nil, fmt.Errorf("recording content revision: %w", err)
	}
	rev, err = q.SetContentRevisionValidation(ctx, db.SetContentRevisionValidationParams{
		ID: rev.ID, Valid: valid, ValidationErrors: issuesJSON,
	})
	if err != nil {
		return nil, fmt.Errorf("recording validation result: %w", err)
	}

	if valid {
		if err := persistContent(ctx, q, rev.ID, c); err != nil {
			return nil, fmt.Errorf("persisting content: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("committing transaction: %w", err)
	}

	return &Result{Revision: rev, Valid: valid, Issues: issues}, nil
}

func persistContent(ctx context.Context, q *db.Queries, revID pgtype.UUID, c *loader.Content) error {
	for _, n := range c.Networks {
		vars, tags, findings, err := commonJSON(n.Vars, n.Tags, n.Findings)
		if err != nil {
			return fmt.Errorf("network %q: %w", n.Name, err)
		}
		visibleFrom, err := jsonField(n.VisibleFrom, "[]")
		if err != nil {
			return fmt.Errorf("network %q visible_from: %w", n.Name, err)
		}
		if _, err := q.CreateNetwork(ctx, db.CreateNetworkParams{
			ContentRevisionID: revID, Path: n.SourceFile, Name: n.Name, Cidr: n.CIDR, VisibleFrom: visibleFrom,
			Vars: vars, Tags: tags, Findings: findings, Extends: strPtrOrNil(n.Extends),
		}); err != nil {
			return fmt.Errorf("network %q: %w", n.Name, err)
		}
	}

	for _, h := range c.Hosts {
		vars, tags, findings, err := commonJSON(h.Vars, h.Tags, h.Findings)
		if err != nil {
			return fmt.Errorf("host %q: %w", h.Name, err)
		}
		ports, err := jsonField(h.Ports, "{}")
		if err != nil {
			return fmt.Errorf("host %q: %w", h.Name, err)
		}
		dependsOn, err := jsonField(h.DependsOn, "[]")
		if err != nil {
			return fmt.Errorf("host %q: %w", h.Name, err)
		}
		steps, err := jsonField(h.Steps, "[]")
		if err != nil {
			return fmt.Errorf("host %q: %w", h.Name, err)
		}
		sched, err := jsonField(h.Schedule, "[]")
		if err != nil {
			return fmt.Errorf("host %q: %w", h.Name, err)
		}
		people, err := jsonField(h.People, "[]")
		if err != nil {
			return fmt.Errorf("host %q: %w", h.Name, err)
		}
		if _, err := q.CreateHost(ctx, db.CreateHostParams{
			ContentRevisionID: revID, Path: h.SourceFile, Name: h.Name, Os: strPtrOrNil(h.OS), Size: strPtrOrNil(h.Size),
			DiskGb: int32PtrOrNil(h.Disk), Ports: ports, DependsOn: dependsOn, Steps: steps, Schedule: sched,
			Vars: vars, Tags: tags, Findings: findings, People: people, Extends: strPtrOrNil(h.Extends),
		}); err != nil {
			return fmt.Errorf("host %q: %w", h.Name, err)
		}
	}

	for _, ct := range c.Containers {
		vars, tags, findings, err := commonJSON(ct.Vars, ct.Tags, ct.Findings)
		if err != nil {
			return fmt.Errorf("container %q: %w", ct.Name, err)
		}
		ports, err := jsonField(ct.Ports, "{}")
		if err != nil {
			return fmt.Errorf("container %q: %w", ct.Name, err)
		}
		dependsOn, err := jsonField(ct.DependsOn, "[]")
		if err != nil {
			return fmt.Errorf("container %q: %w", ct.Name, err)
		}
		steps, err := jsonField(ct.Steps, "[]")
		if err != nil {
			return fmt.Errorf("container %q: %w", ct.Name, err)
		}
		sched, err := jsonField(ct.Schedule, "[]")
		if err != nil {
			return fmt.Errorf("container %q: %w", ct.Name, err)
		}
		people, err := jsonField(ct.People, "[]")
		if err != nil {
			return fmt.Errorf("container %q: %w", ct.Name, err)
		}
		if _, err := q.CreateContainer(ctx, db.CreateContainerParams{
			ContentRevisionID: revID, Path: ct.SourceFile, Name: ct.Name, Image: strPtrOrNil(ct.Image), Size: strPtrOrNil(ct.Size),
			Ports: ports, DependsOn: dependsOn, Steps: steps, Schedule: sched,
			Vars: vars, Tags: tags, Findings: findings, People: people, Extends: strPtrOrNil(ct.Extends),
		}); err != nil {
			return fmt.Errorf("container %q: %w", ct.Name, err)
		}
	}

	for _, s := range c.Scripts {
		_, tags, findings, err := commonJSON(nil, s.Tags, s.Findings)
		if err != nil {
			return fmt.Errorf("script %q: %w", s.Name, err)
		}
		args, err := jsonField(s.Args, "[]")
		if err != nil {
			return fmt.Errorf("script %q: %w", s.Name, err)
		}
		people, err := jsonField(s.People, "[]")
		if err != nil {
			return fmt.Errorf("script %q: %w", s.Name, err)
		}
		validate, err := jsonField(s.Validate, "[]")
		if err != nil {
			return fmt.Errorf("script %q: %w", s.Name, err)
		}
		if _, err := q.CreateScript(ctx, db.CreateScriptParams{
			ContentRevisionID: revID, Path: s.SourceFile, Name: s.Name, Description: strPtrOrNil(s.Description),
			Language: strPtrOrNil(s.Language), SourcePath: strPtrOrNil(s.Source), TimeoutSeconds: int32PtrOrNil(s.Timeout),
			Args: args, IgnoreErrors: s.IgnoreErrors, Tags: tags, Findings: findings, People: people, Validate: validate,
		}); err != nil {
			return fmt.Errorf("script %q: %w", s.Name, err)
		}
	}

	for _, ps := range c.People {
		psRow, err := q.CreatePeopleSource(ctx, db.CreatePeopleSourceParams{
			ContentRevisionID: revID, Path: ps.SourceFile, Name: ps.Name,
		})
		if err != nil {
			return fmt.Errorf("people source %q: %w", ps.Name, err)
		}
		for _, p := range ps.People {
			attrs, err := jsonField(p.Attributes, "{}")
			if err != nil {
				return fmt.Errorf("people source %q, person %q: %w", ps.Name, p.Username, err)
			}
			if _, err := q.CreatePerson(ctx, db.CreatePersonParams{
				PeopleSourceID: psRow.ID, Username: p.Username, Attributes: attrs,
			}); err != nil {
				return fmt.Errorf("people source %q, person %q: %w", ps.Name, p.Username, err)
			}
		}
	}

	containerNames := make(map[string]bool, len(c.Containers))
	for _, ct := range c.Containers {
		containerNames[ct.Name] = true
	}

	for _, e := range c.Environments {
		vars, tags, findings, err := commonJSON(e.Vars, e.Tags, e.Findings)
		if err != nil {
			return fmt.Errorf("environment %q: %w", e.Name, err)
		}
		access, err := jsonField(e.Access, "[]")
		if err != nil {
			return fmt.Errorf("environment %q: %w", e.Name, err)
		}
		var dnsJSON []byte
		if e.DNS != nil {
			dnsJSON, err = json.Marshal(e.DNS)
			if err != nil {
				return fmt.Errorf("environment %q: %w", e.Name, err)
			}
		}

		envRow, err := q.CreateEnvironment(ctx, db.CreateEnvironmentParams{
			ContentRevisionID: revID, Path: e.SourceFile, Name: e.Name,
			SchemaVersion: int32PtrOrNil(e.Schema), Description: strPtrOrNil(e.Description), Teams: int32(e.Teams),
			RootPassword: strPtrOrNil(e.RootPass),
			StartAt:      timestamptzOrZero(e.Start), StopAt: timestamptzOrZero(e.Stop),
			Dns: dnsJSON, Access: access, Vars: vars, Tags: tags, Findings: findings, Extends: strPtrOrNil(e.Extends),
			AgentDebug: e.AgentDebug,
		})
		if err != nil {
			return fmt.Errorf("environment %q: %w", e.Name, err)
		}

		for networkName, objs := range e.Networks {
			for objName, copies := range objs {
				kind := "host"
				if containerNames[objName] {
					kind = "container"
				}
				for _, cp := range copies {
					// `public:` moved onto the host/container definition, so it's
					// no longer a per-placement value; external access reads it from
					// content (loader.Load), not this row. The placement.public
					// column is left unused (nothing reads it).
					if _, err := q.CreatePlacement(ctx, db.CreatePlacementParams{
						EnvironmentID: envRow.ID, NetworkName: networkName, ObjectKind: kind, ObjectName: objName,
						AsName: cp.As, LastOctet: int32(cp.LastOctet), Public: nil,
					}); err != nil {
						return fmt.Errorf("environment %q, placement %s/%s: %w", e.Name, networkName, cp.As, err)
					}
				}
			}
		}
	}

	return nil
}

// commonJSON encodes the three fields (vars, tags, findings) that appear,
// identically shaped, on nearly every content type -- a small convenience
// so persistContent's per-type blocks above don't repeat the same three
// jsonField calls five times over. A nil vars map is fine (scripts have no
// vars field at all; callers pass nil and get back "{}" like everything
// else).
func commonJSON(vars, tags map[string]string, findings []loader.Finding) (varsJSON, tagsJSON, findingsJSON []byte, err error) {
	if varsJSON, err = jsonField(vars, "{}"); err != nil {
		return nil, nil, nil, err
	}
	if tagsJSON, err = jsonField(tags, "{}"); err != nil {
		return nil, nil, nil, err
	}
	if findingsJSON, err = jsonField(findings, "[]"); err != nil {
		return nil, nil, nil, err
	}
	return varsJSON, tagsJSON, findingsJSON, nil
}

// jsonField marshals v, substituting emptyLiteral ("[]" or "{}", matching
// the destination jsonb column's own DEFAULT) for a nil slice/map, which
// json.Marshal would otherwise render as the JSON `null` rather than an
// empty array/object.
func jsonField(v interface{}, emptyLiteral string) ([]byte, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	if string(b) == "null" {
		return []byte(emptyLiteral), nil
	}
	return b, nil
}

func strPtrOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func int32PtrOrNil(v int) *int32 {
	if v == 0 {
		return nil
	}
	x := int32(v)
	return &x
}

// timestamptzOrZero parses an RFC3339 string (as internal/loader stores
// `start`/`stop`/etc, already normalized by sanitizeForSchema) into a
// pgtype.Timestamptz. An empty or unparseable string becomes SQL NULL
// rather than an error -- schema validation already rejects a malformed
// date long before content reaches this package, so a parse failure here
// would mean the schema's own format check regressed, not that this
// commit is bad; storing NULL is more honest than pretending it has no
// value has a real cause here.
func timestamptzOrZero(s string) pgtype.Timestamptz {
	if s == "" {
		return pgtype.Timestamptz{}
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: t, Valid: true}
}

// commitMeta reads the checked-out commit's subject, author name, and commit
// time from the work tree, for the builds tables to show "what commit is this"
// beyond the bare SHA. Best-effort: any failure (not a git checkout, git
// missing) returns empty/NULL values, and the revision still records its SHA.
// %s is the subject line, %an the author name, %cI the strict-ISO commit date.
func commitMeta(ctx context.Context, workDir string) (message, author string, committedAt pgtype.Timestamptz) {
	out, err := exec.CommandContext(ctx, "git", "-C", workDir, "log", "-1", "--format=%s%x00%an%x00%cI").Output()
	if err != nil {
		return "", "", pgtype.Timestamptz{}
	}
	parts := strings.Split(strings.TrimRight(string(out), "\n"), "\x00")
	if len(parts) != 3 {
		return "", "", pgtype.Timestamptz{}
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]), timestamptzOrZero(strings.TrimSpace(parts[2]))
}
