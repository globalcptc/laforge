// Code generated by ent, DO NOT EDIT.

package ent

import (
	"fmt"
	"strings"
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/sql"
	"github.com/gen0cide/laforge/ent/build"
	"github.com/gen0cide/laforge/ent/buildcommit"
	"github.com/google/uuid"
)

// BuildCommit is the model entity for the BuildCommit schema.
type BuildCommit struct {
	config `json:"-"`
	// ID of the ent.
	ID uuid.UUID `json:"id,omitempty"`
	// Type holds the value of the "type" field.
	Type buildcommit.Type `json:"type,omitempty"`
	// Revision holds the value of the "revision" field.
	Revision int `json:"revision,omitempty"`
	// State holds the value of the "state" field.
	State buildcommit.State `json:"state,omitempty"`
	// CreatedAt holds the value of the "created_at" field.
	CreatedAt time.Time `json:"created_at,omitempty"`
	// Edges holds the relations/edges for other nodes in the graph.
	// The values are being populated by the BuildCommitQuery when eager-loading is set.
	Edges BuildCommitEdges `json:"edges"`

	// vvvvvvvvvvvv CUSTOM vvvvvvvvvvvv
	// Edges put into the main struct to be loaded via hcl
	// Build holds the value of the Build edge.
	HCLBuild *Build `json:"Build,omitempty"`
	// ServerTasks holds the value of the ServerTasks edge.
	HCLServerTasks []*ServerTask `json:"ServerTasks,omitempty"`
	// PlanDiffs holds the value of the PlanDiffs edge.
	HCLPlanDiffs []*PlanDiff `json:"PlanDiffs,omitempty"`
	// ^^^^^^^^^^^^ CUSTOM ^^^^^^^^^^^^^
	build_commit_build *uuid.UUID
	selectValues       sql.SelectValues
}

// BuildCommitEdges holds the relations/edges for other nodes in the graph.
type BuildCommitEdges struct {
	// Build holds the value of the Build edge.
	Build *Build `json:"Build,omitempty"`
	// ServerTasks holds the value of the ServerTasks edge.
	ServerTasks []*ServerTask `json:"ServerTasks,omitempty"`
	// PlanDiffs holds the value of the PlanDiffs edge.
	PlanDiffs []*PlanDiff `json:"PlanDiffs,omitempty"`
	// loadedTypes holds the information for reporting if a
	// type was loaded (or requested) in eager-loading or not.
	loadedTypes [3]bool
	// totalCount holds the count of the edges above.
	totalCount [3]map[string]int

	namedServerTasks map[string][]*ServerTask
	namedPlanDiffs   map[string][]*PlanDiff
}

// BuildOrErr returns the Build value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e BuildCommitEdges) BuildOrErr() (*Build, error) {
	if e.loadedTypes[0] {
		if e.Build == nil {
			// Edge was loaded but was not found.
			return nil, &NotFoundError{label: build.Label}
		}
		return e.Build, nil
	}
	return nil, &NotLoadedError{edge: "Build"}
}

// ServerTasksOrErr returns the ServerTasks value or an error if the edge
// was not loaded in eager-loading.
func (e BuildCommitEdges) ServerTasksOrErr() ([]*ServerTask, error) {
	if e.loadedTypes[1] {
		return e.ServerTasks, nil
	}
	return nil, &NotLoadedError{edge: "ServerTasks"}
}

// PlanDiffsOrErr returns the PlanDiffs value or an error if the edge
// was not loaded in eager-loading.
func (e BuildCommitEdges) PlanDiffsOrErr() ([]*PlanDiff, error) {
	if e.loadedTypes[2] {
		return e.PlanDiffs, nil
	}
	return nil, &NotLoadedError{edge: "PlanDiffs"}
}

// scanValues returns the types for scanning values from sql.Rows.
func (*BuildCommit) scanValues(columns []string) ([]any, error) {
	values := make([]any, len(columns))
	for i := range columns {
		switch columns[i] {
		case buildcommit.FieldRevision:
			values[i] = new(sql.NullInt64)
		case buildcommit.FieldType, buildcommit.FieldState:
			values[i] = new(sql.NullString)
		case buildcommit.FieldCreatedAt:
			values[i] = new(sql.NullTime)
		case buildcommit.FieldID:
			values[i] = new(uuid.UUID)
		case buildcommit.ForeignKeys[0]: // build_commit_build
			values[i] = &sql.NullScanner{S: new(uuid.UUID)}
		default:
			values[i] = new(sql.UnknownType)
		}
	}
	return values, nil
}

// assignValues assigns the values that were returned from sql.Rows (after scanning)
// to the BuildCommit fields.
func (bc *BuildCommit) assignValues(columns []string, values []any) error {
	if m, n := len(values), len(columns); m < n {
		return fmt.Errorf("mismatch number of scan values: %d != %d", m, n)
	}
	for i := range columns {
		switch columns[i] {
		case buildcommit.FieldID:
			if value, ok := values[i].(*uuid.UUID); !ok {
				return fmt.Errorf("unexpected type %T for field id", values[i])
			} else if value != nil {
				bc.ID = *value
			}
		case buildcommit.FieldType:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field type", values[i])
			} else if value.Valid {
				bc.Type = buildcommit.Type(value.String)
			}
		case buildcommit.FieldRevision:
			if value, ok := values[i].(*sql.NullInt64); !ok {
				return fmt.Errorf("unexpected type %T for field revision", values[i])
			} else if value.Valid {
				bc.Revision = int(value.Int64)
			}
		case buildcommit.FieldState:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field state", values[i])
			} else if value.Valid {
				bc.State = buildcommit.State(value.String)
			}
		case buildcommit.FieldCreatedAt:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field created_at", values[i])
			} else if value.Valid {
				bc.CreatedAt = value.Time
			}
		case buildcommit.ForeignKeys[0]:
			if value, ok := values[i].(*sql.NullScanner); !ok {
				return fmt.Errorf("unexpected type %T for field build_commit_build", values[i])
			} else if value.Valid {
				bc.build_commit_build = new(uuid.UUID)
				*bc.build_commit_build = *value.S.(*uuid.UUID)
			}
		default:
			bc.selectValues.Set(columns[i], values[i])
		}
	}
	return nil
}

// Value returns the ent.Value that was dynamically selected and assigned to the BuildCommit.
// This includes values selected through modifiers, order, etc.
func (bc *BuildCommit) Value(name string) (ent.Value, error) {
	return bc.selectValues.Get(name)
}

// QueryBuild queries the "Build" edge of the BuildCommit entity.
func (bc *BuildCommit) QueryBuild() *BuildQuery {
	return NewBuildCommitClient(bc.config).QueryBuild(bc)
}

// QueryServerTasks queries the "ServerTasks" edge of the BuildCommit entity.
func (bc *BuildCommit) QueryServerTasks() *ServerTaskQuery {
	return NewBuildCommitClient(bc.config).QueryServerTasks(bc)
}

// QueryPlanDiffs queries the "PlanDiffs" edge of the BuildCommit entity.
func (bc *BuildCommit) QueryPlanDiffs() *PlanDiffQuery {
	return NewBuildCommitClient(bc.config).QueryPlanDiffs(bc)
}

// Update returns a builder for updating this BuildCommit.
// Note that you need to call BuildCommit.Unwrap() before calling this method if this BuildCommit
// was returned from a transaction, and the transaction was committed or rolled back.
func (bc *BuildCommit) Update() *BuildCommitUpdateOne {
	return NewBuildCommitClient(bc.config).UpdateOne(bc)
}

// Unwrap unwraps the BuildCommit entity that was returned from a transaction after it was closed,
// so that all future queries will be executed through the driver which created the transaction.
func (bc *BuildCommit) Unwrap() *BuildCommit {
	_tx, ok := bc.config.driver.(*txDriver)
	if !ok {
		panic("ent: BuildCommit is not a transactional entity")
	}
	bc.config.driver = _tx.drv
	return bc
}

// String implements the fmt.Stringer.
func (bc *BuildCommit) String() string {
	var builder strings.Builder
	builder.WriteString("BuildCommit(")
	builder.WriteString(fmt.Sprintf("id=%v, ", bc.ID))
	builder.WriteString("type=")
	builder.WriteString(fmt.Sprintf("%v", bc.Type))
	builder.WriteString(", ")
	builder.WriteString("revision=")
	builder.WriteString(fmt.Sprintf("%v", bc.Revision))
	builder.WriteString(", ")
	builder.WriteString("state=")
	builder.WriteString(fmt.Sprintf("%v", bc.State))
	builder.WriteString(", ")
	builder.WriteString("created_at=")
	builder.WriteString(bc.CreatedAt.Format(time.ANSIC))
	builder.WriteByte(')')
	return builder.String()
}

// NamedServerTasks returns the ServerTasks named value or an error if the edge was not
// loaded in eager-loading with this name.
func (bc *BuildCommit) NamedServerTasks(name string) ([]*ServerTask, error) {
	if bc.Edges.namedServerTasks == nil {
		return nil, &NotLoadedError{edge: name}
	}
	nodes, ok := bc.Edges.namedServerTasks[name]
	if !ok {
		return nil, &NotLoadedError{edge: name}
	}
	return nodes, nil
}

func (bc *BuildCommit) appendNamedServerTasks(name string, edges ...*ServerTask) {
	if bc.Edges.namedServerTasks == nil {
		bc.Edges.namedServerTasks = make(map[string][]*ServerTask)
	}
	if len(edges) == 0 {
		bc.Edges.namedServerTasks[name] = []*ServerTask{}
	} else {
		bc.Edges.namedServerTasks[name] = append(bc.Edges.namedServerTasks[name], edges...)
	}
}

// NamedPlanDiffs returns the PlanDiffs named value or an error if the edge was not
// loaded in eager-loading with this name.
func (bc *BuildCommit) NamedPlanDiffs(name string) ([]*PlanDiff, error) {
	if bc.Edges.namedPlanDiffs == nil {
		return nil, &NotLoadedError{edge: name}
	}
	nodes, ok := bc.Edges.namedPlanDiffs[name]
	if !ok {
		return nil, &NotLoadedError{edge: name}
	}
	return nodes, nil
}

func (bc *BuildCommit) appendNamedPlanDiffs(name string, edges ...*PlanDiff) {
	if bc.Edges.namedPlanDiffs == nil {
		bc.Edges.namedPlanDiffs = make(map[string][]*PlanDiff)
	}
	if len(edges) == 0 {
		bc.Edges.namedPlanDiffs[name] = []*PlanDiff{}
	} else {
		bc.Edges.namedPlanDiffs[name] = append(bc.Edges.namedPlanDiffs[name], edges...)
	}
}

// BuildCommits is a parsable slice of BuildCommit.
type BuildCommits []*BuildCommit
