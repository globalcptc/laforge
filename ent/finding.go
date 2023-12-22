// Code generated by ent, DO NOT EDIT.

package ent

import (
	"encoding/json"
	"fmt"
	"strings"

	"entgo.io/ent"
	"entgo.io/ent/dialect/sql"
	"github.com/gen0cide/laforge/ent/environment"
	"github.com/gen0cide/laforge/ent/finding"
	"github.com/gen0cide/laforge/ent/host"
	"github.com/gen0cide/laforge/ent/script"
	"github.com/google/uuid"
)

// Finding is the model entity for the Finding schema.
type Finding struct {
	config ` json:"-"`
	// ID of the ent.
	ID uuid.UUID `json:"id,omitempty"`
	// Name holds the value of the "name" field.
	Name string `json:"name,omitempty" hcl:"name,attr"`
	// Description holds the value of the "description" field.
	Description string `json:"description,omitempty" hcl:"description,optional"`
	// Severity holds the value of the "severity" field.
	Severity finding.Severity `json:"severity,omitempty" hcl:"severity,attr"`
	// Difficulty holds the value of the "difficulty" field.
	Difficulty finding.Difficulty `json:"difficulty,omitempty" hcl:"difficulty,attr"`
	// Tags holds the value of the "tags" field.
	Tags map[string]string `json:"tags,omitempty" hcl:"tags,optional"`
	// Edges holds the relations/edges for other nodes in the graph.
	// The values are being populated by the FindingQuery when eager-loading is set.
	Edges FindingEdges `json:"edges"`

	// vvvvvvvvvvvv CUSTOM vvvvvvvvvvvv
	// Edges put into the main struct to be loaded via hcl
	// FindingToUser holds the value of the FindingToUser edge.
	HCLFindingToUser []*User `json:"FindingToUser,omitempty" hcl:"maintainer,block"`
	// FindingToHost holds the value of the FindingToHost edge.
	HCLFindingToHost *Host `json:"FindingToHost,omitempty"`
	// FindingToScript holds the value of the FindingToScript edge.
	HCLFindingToScript *Script `json:"FindingToScript,omitempty"`
	// FindingToEnvironment holds the value of the FindingToEnvironment edge.
	HCLFindingToEnvironment *Environment `json:"FindingToEnvironment,omitempty"`
	// ^^^^^^^^^^^^ CUSTOM ^^^^^^^^^^^^^
	environment_environment_to_finding *uuid.UUID
	finding_finding_to_host            *uuid.UUID
	script_script_to_finding           *uuid.UUID
	selectValues                       sql.SelectValues
}

// FindingEdges holds the relations/edges for other nodes in the graph.
type FindingEdges struct {
	// FindingToUser holds the value of the FindingToUser edge.
	FindingToUser []*User `json:"FindingToUser,omitempty" hcl:"maintainer,block"`
	// FindingToHost holds the value of the FindingToHost edge.
	FindingToHost *Host `json:"FindingToHost,omitempty"`
	// FindingToScript holds the value of the FindingToScript edge.
	FindingToScript *Script `json:"FindingToScript,omitempty"`
	// FindingToEnvironment holds the value of the FindingToEnvironment edge.
	FindingToEnvironment *Environment `json:"FindingToEnvironment,omitempty"`
	// loadedTypes holds the information for reporting if a
	// type was loaded (or requested) in eager-loading or not.
	loadedTypes [4]bool
	// totalCount holds the count of the edges above.
	totalCount [4]map[string]int

	namedFindingToUser map[string][]*User
}

// FindingToUserOrErr returns the FindingToUser value or an error if the edge
// was not loaded in eager-loading.
func (e FindingEdges) FindingToUserOrErr() ([]*User, error) {
	if e.loadedTypes[0] {
		return e.FindingToUser, nil
	}
	return nil, &NotLoadedError{edge: "FindingToUser"}
}

// FindingToHostOrErr returns the FindingToHost value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e FindingEdges) FindingToHostOrErr() (*Host, error) {
	if e.loadedTypes[1] {
		if e.FindingToHost == nil {
			// Edge was loaded but was not found.
			return nil, &NotFoundError{label: host.Label}
		}
		return e.FindingToHost, nil
	}
	return nil, &NotLoadedError{edge: "FindingToHost"}
}

// FindingToScriptOrErr returns the FindingToScript value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e FindingEdges) FindingToScriptOrErr() (*Script, error) {
	if e.loadedTypes[2] {
		if e.FindingToScript == nil {
			// Edge was loaded but was not found.
			return nil, &NotFoundError{label: script.Label}
		}
		return e.FindingToScript, nil
	}
	return nil, &NotLoadedError{edge: "FindingToScript"}
}

// FindingToEnvironmentOrErr returns the FindingToEnvironment value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e FindingEdges) FindingToEnvironmentOrErr() (*Environment, error) {
	if e.loadedTypes[3] {
		if e.FindingToEnvironment == nil {
			// Edge was loaded but was not found.
			return nil, &NotFoundError{label: environment.Label}
		}
		return e.FindingToEnvironment, nil
	}
	return nil, &NotLoadedError{edge: "FindingToEnvironment"}
}

// scanValues returns the types for scanning values from sql.Rows.
func (*Finding) scanValues(columns []string) ([]any, error) {
	values := make([]any, len(columns))
	for i := range columns {
		switch columns[i] {
		case finding.FieldTags:
			values[i] = new([]byte)
		case finding.FieldName, finding.FieldDescription, finding.FieldSeverity, finding.FieldDifficulty:
			values[i] = new(sql.NullString)
		case finding.FieldID:
			values[i] = new(uuid.UUID)
		case finding.ForeignKeys[0]: // environment_environment_to_finding
			values[i] = &sql.NullScanner{S: new(uuid.UUID)}
		case finding.ForeignKeys[1]: // finding_finding_to_host
			values[i] = &sql.NullScanner{S: new(uuid.UUID)}
		case finding.ForeignKeys[2]: // script_script_to_finding
			values[i] = &sql.NullScanner{S: new(uuid.UUID)}
		default:
			values[i] = new(sql.UnknownType)
		}
	}
	return values, nil
}

// assignValues assigns the values that were returned from sql.Rows (after scanning)
// to the Finding fields.
func (f *Finding) assignValues(columns []string, values []any) error {
	if m, n := len(values), len(columns); m < n {
		return fmt.Errorf("mismatch number of scan values: %d != %d", m, n)
	}
	for i := range columns {
		switch columns[i] {
		case finding.FieldID:
			if value, ok := values[i].(*uuid.UUID); !ok {
				return fmt.Errorf("unexpected type %T for field id", values[i])
			} else if value != nil {
				f.ID = *value
			}
		case finding.FieldName:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field name", values[i])
			} else if value.Valid {
				f.Name = value.String
			}
		case finding.FieldDescription:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field description", values[i])
			} else if value.Valid {
				f.Description = value.String
			}
		case finding.FieldSeverity:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field severity", values[i])
			} else if value.Valid {
				f.Severity = finding.Severity(value.String)
			}
		case finding.FieldDifficulty:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field difficulty", values[i])
			} else if value.Valid {
				f.Difficulty = finding.Difficulty(value.String)
			}
		case finding.FieldTags:
			if value, ok := values[i].(*[]byte); !ok {
				return fmt.Errorf("unexpected type %T for field tags", values[i])
			} else if value != nil && len(*value) > 0 {
				if err := json.Unmarshal(*value, &f.Tags); err != nil {
					return fmt.Errorf("unmarshal field tags: %w", err)
				}
			}
		case finding.ForeignKeys[0]:
			if value, ok := values[i].(*sql.NullScanner); !ok {
				return fmt.Errorf("unexpected type %T for field environment_environment_to_finding", values[i])
			} else if value.Valid {
				f.environment_environment_to_finding = new(uuid.UUID)
				*f.environment_environment_to_finding = *value.S.(*uuid.UUID)
			}
		case finding.ForeignKeys[1]:
			if value, ok := values[i].(*sql.NullScanner); !ok {
				return fmt.Errorf("unexpected type %T for field finding_finding_to_host", values[i])
			} else if value.Valid {
				f.finding_finding_to_host = new(uuid.UUID)
				*f.finding_finding_to_host = *value.S.(*uuid.UUID)
			}
		case finding.ForeignKeys[2]:
			if value, ok := values[i].(*sql.NullScanner); !ok {
				return fmt.Errorf("unexpected type %T for field script_script_to_finding", values[i])
			} else if value.Valid {
				f.script_script_to_finding = new(uuid.UUID)
				*f.script_script_to_finding = *value.S.(*uuid.UUID)
			}
		default:
			f.selectValues.Set(columns[i], values[i])
		}
	}
	return nil
}

// Value returns the ent.Value that was dynamically selected and assigned to the Finding.
// This includes values selected through modifiers, order, etc.
func (f *Finding) Value(name string) (ent.Value, error) {
	return f.selectValues.Get(name)
}

// QueryFindingToUser queries the "FindingToUser" edge of the Finding entity.
func (f *Finding) QueryFindingToUser() *UserQuery {
	return NewFindingClient(f.config).QueryFindingToUser(f)
}

// QueryFindingToHost queries the "FindingToHost" edge of the Finding entity.
func (f *Finding) QueryFindingToHost() *HostQuery {
	return NewFindingClient(f.config).QueryFindingToHost(f)
}

// QueryFindingToScript queries the "FindingToScript" edge of the Finding entity.
func (f *Finding) QueryFindingToScript() *ScriptQuery {
	return NewFindingClient(f.config).QueryFindingToScript(f)
}

// QueryFindingToEnvironment queries the "FindingToEnvironment" edge of the Finding entity.
func (f *Finding) QueryFindingToEnvironment() *EnvironmentQuery {
	return NewFindingClient(f.config).QueryFindingToEnvironment(f)
}

// Update returns a builder for updating this Finding.
// Note that you need to call Finding.Unwrap() before calling this method if this Finding
// was returned from a transaction, and the transaction was committed or rolled back.
func (f *Finding) Update() *FindingUpdateOne {
	return NewFindingClient(f.config).UpdateOne(f)
}

// Unwrap unwraps the Finding entity that was returned from a transaction after it was closed,
// so that all future queries will be executed through the driver which created the transaction.
func (f *Finding) Unwrap() *Finding {
	_tx, ok := f.config.driver.(*txDriver)
	if !ok {
		panic("ent: Finding is not a transactional entity")
	}
	f.config.driver = _tx.drv
	return f
}

// String implements the fmt.Stringer.
func (f *Finding) String() string {
	var builder strings.Builder
	builder.WriteString("Finding(")
	builder.WriteString(fmt.Sprintf("id=%v, ", f.ID))
	builder.WriteString("name=")
	builder.WriteString(f.Name)
	builder.WriteString(", ")
	builder.WriteString("description=")
	builder.WriteString(f.Description)
	builder.WriteString(", ")
	builder.WriteString("severity=")
	builder.WriteString(fmt.Sprintf("%v", f.Severity))
	builder.WriteString(", ")
	builder.WriteString("difficulty=")
	builder.WriteString(fmt.Sprintf("%v", f.Difficulty))
	builder.WriteString(", ")
	builder.WriteString("tags=")
	builder.WriteString(fmt.Sprintf("%v", f.Tags))
	builder.WriteByte(')')
	return builder.String()
}

// NamedFindingToUser returns the FindingToUser named value or an error if the edge was not
// loaded in eager-loading with this name.
func (f *Finding) NamedFindingToUser(name string) ([]*User, error) {
	if f.Edges.namedFindingToUser == nil {
		return nil, &NotLoadedError{edge: name}
	}
	nodes, ok := f.Edges.namedFindingToUser[name]
	if !ok {
		return nil, &NotLoadedError{edge: name}
	}
	return nodes, nil
}

func (f *Finding) appendNamedFindingToUser(name string, edges ...*User) {
	if f.Edges.namedFindingToUser == nil {
		f.Edges.namedFindingToUser = make(map[string][]*User)
	}
	if len(edges) == 0 {
		f.Edges.namedFindingToUser[name] = []*User{}
	} else {
		f.Edges.namedFindingToUser[name] = append(f.Edges.namedFindingToUser[name], edges...)
	}
}

// Findings is a parsable slice of Finding.
type Findings []*Finding
