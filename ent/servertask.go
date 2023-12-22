// Code generated by ent, DO NOT EDIT.

package ent

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/sql"
	"github.com/gen0cide/laforge/ent/authuser"
	"github.com/gen0cide/laforge/ent/build"
	"github.com/gen0cide/laforge/ent/buildcommit"
	"github.com/gen0cide/laforge/ent/environment"
	"github.com/gen0cide/laforge/ent/servertask"
	"github.com/gen0cide/laforge/ent/status"
	"github.com/google/uuid"
)

// ServerTask is the model entity for the ServerTask schema.
type ServerTask struct {
	config `json:"-"`
	// ID of the ent.
	ID uuid.UUID `json:"id,omitempty"`
	// Type holds the value of the "type" field.
	Type servertask.Type `json:"type,omitempty"`
	// StartTime holds the value of the "start_time" field.
	StartTime time.Time `json:"start_time,omitempty"`
	// EndTime holds the value of the "end_time" field.
	EndTime time.Time `json:"end_time,omitempty"`
	// Errors holds the value of the "errors" field.
	Errors []string `json:"errors,omitempty"`
	// LogFilePath holds the value of the "log_file_path" field.
	LogFilePath string `json:"log_file_path,omitempty"`
	// Edges holds the relations/edges for other nodes in the graph.
	// The values are being populated by the ServerTaskQuery when eager-loading is set.
	Edges ServerTaskEdges `json:"edges"`

	// vvvvvvvvvvvv CUSTOM vvvvvvvvvvvv
	// Edges put into the main struct to be loaded via hcl
	// ServerTaskToAuthUser holds the value of the ServerTaskToAuthUser edge.
	HCLServerTaskToAuthUser *AuthUser `json:"ServerTaskToAuthUser,omitempty"`
	// ServerTaskToStatus holds the value of the ServerTaskToStatus edge.
	HCLServerTaskToStatus *Status `json:"ServerTaskToStatus,omitempty"`
	// ServerTaskToEnvironment holds the value of the ServerTaskToEnvironment edge.
	HCLServerTaskToEnvironment *Environment `json:"ServerTaskToEnvironment,omitempty"`
	// ServerTaskToBuild holds the value of the ServerTaskToBuild edge.
	HCLServerTaskToBuild *Build `json:"ServerTaskToBuild,omitempty"`
	// ServerTaskToBuildCommit holds the value of the ServerTaskToBuildCommit edge.
	HCLServerTaskToBuildCommit *BuildCommit `json:"ServerTaskToBuildCommit,omitempty"`
	// ServerTaskToGinFileMiddleware holds the value of the ServerTaskToGinFileMiddleware edge.
	HCLServerTaskToGinFileMiddleware []*GinFileMiddleware `json:"ServerTaskToGinFileMiddleware,omitempty"`
	// ^^^^^^^^^^^^ CUSTOM ^^^^^^^^^^^^^
	server_task_server_task_to_auth_user    *uuid.UUID
	server_task_server_task_to_environment  *uuid.UUID
	server_task_server_task_to_build        *uuid.UUID
	server_task_server_task_to_build_commit *uuid.UUID
	selectValues                            sql.SelectValues
}

// ServerTaskEdges holds the relations/edges for other nodes in the graph.
type ServerTaskEdges struct {
	// ServerTaskToAuthUser holds the value of the ServerTaskToAuthUser edge.
	ServerTaskToAuthUser *AuthUser `json:"ServerTaskToAuthUser,omitempty"`
	// ServerTaskToStatus holds the value of the ServerTaskToStatus edge.
	ServerTaskToStatus *Status `json:"ServerTaskToStatus,omitempty"`
	// ServerTaskToEnvironment holds the value of the ServerTaskToEnvironment edge.
	ServerTaskToEnvironment *Environment `json:"ServerTaskToEnvironment,omitempty"`
	// ServerTaskToBuild holds the value of the ServerTaskToBuild edge.
	ServerTaskToBuild *Build `json:"ServerTaskToBuild,omitempty"`
	// ServerTaskToBuildCommit holds the value of the ServerTaskToBuildCommit edge.
	ServerTaskToBuildCommit *BuildCommit `json:"ServerTaskToBuildCommit,omitempty"`
	// ServerTaskToGinFileMiddleware holds the value of the ServerTaskToGinFileMiddleware edge.
	ServerTaskToGinFileMiddleware []*GinFileMiddleware `json:"ServerTaskToGinFileMiddleware,omitempty"`
	// loadedTypes holds the information for reporting if a
	// type was loaded (or requested) in eager-loading or not.
	loadedTypes [6]bool
	// totalCount holds the count of the edges above.
	totalCount [6]map[string]int

	namedServerTaskToGinFileMiddleware map[string][]*GinFileMiddleware
}

// ServerTaskToAuthUserOrErr returns the ServerTaskToAuthUser value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e ServerTaskEdges) ServerTaskToAuthUserOrErr() (*AuthUser, error) {
	if e.loadedTypes[0] {
		if e.ServerTaskToAuthUser == nil {
			// Edge was loaded but was not found.
			return nil, &NotFoundError{label: authuser.Label}
		}
		return e.ServerTaskToAuthUser, nil
	}
	return nil, &NotLoadedError{edge: "ServerTaskToAuthUser"}
}

// ServerTaskToStatusOrErr returns the ServerTaskToStatus value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e ServerTaskEdges) ServerTaskToStatusOrErr() (*Status, error) {
	if e.loadedTypes[1] {
		if e.ServerTaskToStatus == nil {
			// Edge was loaded but was not found.
			return nil, &NotFoundError{label: status.Label}
		}
		return e.ServerTaskToStatus, nil
	}
	return nil, &NotLoadedError{edge: "ServerTaskToStatus"}
}

// ServerTaskToEnvironmentOrErr returns the ServerTaskToEnvironment value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e ServerTaskEdges) ServerTaskToEnvironmentOrErr() (*Environment, error) {
	if e.loadedTypes[2] {
		if e.ServerTaskToEnvironment == nil {
			// Edge was loaded but was not found.
			return nil, &NotFoundError{label: environment.Label}
		}
		return e.ServerTaskToEnvironment, nil
	}
	return nil, &NotLoadedError{edge: "ServerTaskToEnvironment"}
}

// ServerTaskToBuildOrErr returns the ServerTaskToBuild value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e ServerTaskEdges) ServerTaskToBuildOrErr() (*Build, error) {
	if e.loadedTypes[3] {
		if e.ServerTaskToBuild == nil {
			// Edge was loaded but was not found.
			return nil, &NotFoundError{label: build.Label}
		}
		return e.ServerTaskToBuild, nil
	}
	return nil, &NotLoadedError{edge: "ServerTaskToBuild"}
}

// ServerTaskToBuildCommitOrErr returns the ServerTaskToBuildCommit value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e ServerTaskEdges) ServerTaskToBuildCommitOrErr() (*BuildCommit, error) {
	if e.loadedTypes[4] {
		if e.ServerTaskToBuildCommit == nil {
			// Edge was loaded but was not found.
			return nil, &NotFoundError{label: buildcommit.Label}
		}
		return e.ServerTaskToBuildCommit, nil
	}
	return nil, &NotLoadedError{edge: "ServerTaskToBuildCommit"}
}

// ServerTaskToGinFileMiddlewareOrErr returns the ServerTaskToGinFileMiddleware value or an error if the edge
// was not loaded in eager-loading.
func (e ServerTaskEdges) ServerTaskToGinFileMiddlewareOrErr() ([]*GinFileMiddleware, error) {
	if e.loadedTypes[5] {
		return e.ServerTaskToGinFileMiddleware, nil
	}
	return nil, &NotLoadedError{edge: "ServerTaskToGinFileMiddleware"}
}

// scanValues returns the types for scanning values from sql.Rows.
func (*ServerTask) scanValues(columns []string) ([]any, error) {
	values := make([]any, len(columns))
	for i := range columns {
		switch columns[i] {
		case servertask.FieldErrors:
			values[i] = new([]byte)
		case servertask.FieldType, servertask.FieldLogFilePath:
			values[i] = new(sql.NullString)
		case servertask.FieldStartTime, servertask.FieldEndTime:
			values[i] = new(sql.NullTime)
		case servertask.FieldID:
			values[i] = new(uuid.UUID)
		case servertask.ForeignKeys[0]: // server_task_server_task_to_auth_user
			values[i] = &sql.NullScanner{S: new(uuid.UUID)}
		case servertask.ForeignKeys[1]: // server_task_server_task_to_environment
			values[i] = &sql.NullScanner{S: new(uuid.UUID)}
		case servertask.ForeignKeys[2]: // server_task_server_task_to_build
			values[i] = &sql.NullScanner{S: new(uuid.UUID)}
		case servertask.ForeignKeys[3]: // server_task_server_task_to_build_commit
			values[i] = &sql.NullScanner{S: new(uuid.UUID)}
		default:
			values[i] = new(sql.UnknownType)
		}
	}
	return values, nil
}

// assignValues assigns the values that were returned from sql.Rows (after scanning)
// to the ServerTask fields.
func (st *ServerTask) assignValues(columns []string, values []any) error {
	if m, n := len(values), len(columns); m < n {
		return fmt.Errorf("mismatch number of scan values: %d != %d", m, n)
	}
	for i := range columns {
		switch columns[i] {
		case servertask.FieldID:
			if value, ok := values[i].(*uuid.UUID); !ok {
				return fmt.Errorf("unexpected type %T for field id", values[i])
			} else if value != nil {
				st.ID = *value
			}
		case servertask.FieldType:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field type", values[i])
			} else if value.Valid {
				st.Type = servertask.Type(value.String)
			}
		case servertask.FieldStartTime:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field start_time", values[i])
			} else if value.Valid {
				st.StartTime = value.Time
			}
		case servertask.FieldEndTime:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field end_time", values[i])
			} else if value.Valid {
				st.EndTime = value.Time
			}
		case servertask.FieldErrors:
			if value, ok := values[i].(*[]byte); !ok {
				return fmt.Errorf("unexpected type %T for field errors", values[i])
			} else if value != nil && len(*value) > 0 {
				if err := json.Unmarshal(*value, &st.Errors); err != nil {
					return fmt.Errorf("unmarshal field errors: %w", err)
				}
			}
		case servertask.FieldLogFilePath:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field log_file_path", values[i])
			} else if value.Valid {
				st.LogFilePath = value.String
			}
		case servertask.ForeignKeys[0]:
			if value, ok := values[i].(*sql.NullScanner); !ok {
				return fmt.Errorf("unexpected type %T for field server_task_server_task_to_auth_user", values[i])
			} else if value.Valid {
				st.server_task_server_task_to_auth_user = new(uuid.UUID)
				*st.server_task_server_task_to_auth_user = *value.S.(*uuid.UUID)
			}
		case servertask.ForeignKeys[1]:
			if value, ok := values[i].(*sql.NullScanner); !ok {
				return fmt.Errorf("unexpected type %T for field server_task_server_task_to_environment", values[i])
			} else if value.Valid {
				st.server_task_server_task_to_environment = new(uuid.UUID)
				*st.server_task_server_task_to_environment = *value.S.(*uuid.UUID)
			}
		case servertask.ForeignKeys[2]:
			if value, ok := values[i].(*sql.NullScanner); !ok {
				return fmt.Errorf("unexpected type %T for field server_task_server_task_to_build", values[i])
			} else if value.Valid {
				st.server_task_server_task_to_build = new(uuid.UUID)
				*st.server_task_server_task_to_build = *value.S.(*uuid.UUID)
			}
		case servertask.ForeignKeys[3]:
			if value, ok := values[i].(*sql.NullScanner); !ok {
				return fmt.Errorf("unexpected type %T for field server_task_server_task_to_build_commit", values[i])
			} else if value.Valid {
				st.server_task_server_task_to_build_commit = new(uuid.UUID)
				*st.server_task_server_task_to_build_commit = *value.S.(*uuid.UUID)
			}
		default:
			st.selectValues.Set(columns[i], values[i])
		}
	}
	return nil
}

// Value returns the ent.Value that was dynamically selected and assigned to the ServerTask.
// This includes values selected through modifiers, order, etc.
func (st *ServerTask) Value(name string) (ent.Value, error) {
	return st.selectValues.Get(name)
}

// QueryServerTaskToAuthUser queries the "ServerTaskToAuthUser" edge of the ServerTask entity.
func (st *ServerTask) QueryServerTaskToAuthUser() *AuthUserQuery {
	return NewServerTaskClient(st.config).QueryServerTaskToAuthUser(st)
}

// QueryServerTaskToStatus queries the "ServerTaskToStatus" edge of the ServerTask entity.
func (st *ServerTask) QueryServerTaskToStatus() *StatusQuery {
	return NewServerTaskClient(st.config).QueryServerTaskToStatus(st)
}

// QueryServerTaskToEnvironment queries the "ServerTaskToEnvironment" edge of the ServerTask entity.
func (st *ServerTask) QueryServerTaskToEnvironment() *EnvironmentQuery {
	return NewServerTaskClient(st.config).QueryServerTaskToEnvironment(st)
}

// QueryServerTaskToBuild queries the "ServerTaskToBuild" edge of the ServerTask entity.
func (st *ServerTask) QueryServerTaskToBuild() *BuildQuery {
	return NewServerTaskClient(st.config).QueryServerTaskToBuild(st)
}

// QueryServerTaskToBuildCommit queries the "ServerTaskToBuildCommit" edge of the ServerTask entity.
func (st *ServerTask) QueryServerTaskToBuildCommit() *BuildCommitQuery {
	return NewServerTaskClient(st.config).QueryServerTaskToBuildCommit(st)
}

// QueryServerTaskToGinFileMiddleware queries the "ServerTaskToGinFileMiddleware" edge of the ServerTask entity.
func (st *ServerTask) QueryServerTaskToGinFileMiddleware() *GinFileMiddlewareQuery {
	return NewServerTaskClient(st.config).QueryServerTaskToGinFileMiddleware(st)
}

// Update returns a builder for updating this ServerTask.
// Note that you need to call ServerTask.Unwrap() before calling this method if this ServerTask
// was returned from a transaction, and the transaction was committed or rolled back.
func (st *ServerTask) Update() *ServerTaskUpdateOne {
	return NewServerTaskClient(st.config).UpdateOne(st)
}

// Unwrap unwraps the ServerTask entity that was returned from a transaction after it was closed,
// so that all future queries will be executed through the driver which created the transaction.
func (st *ServerTask) Unwrap() *ServerTask {
	_tx, ok := st.config.driver.(*txDriver)
	if !ok {
		panic("ent: ServerTask is not a transactional entity")
	}
	st.config.driver = _tx.drv
	return st
}

// String implements the fmt.Stringer.
func (st *ServerTask) String() string {
	var builder strings.Builder
	builder.WriteString("ServerTask(")
	builder.WriteString(fmt.Sprintf("id=%v, ", st.ID))
	builder.WriteString("type=")
	builder.WriteString(fmt.Sprintf("%v", st.Type))
	builder.WriteString(", ")
	builder.WriteString("start_time=")
	builder.WriteString(st.StartTime.Format(time.ANSIC))
	builder.WriteString(", ")
	builder.WriteString("end_time=")
	builder.WriteString(st.EndTime.Format(time.ANSIC))
	builder.WriteString(", ")
	builder.WriteString("errors=")
	builder.WriteString(fmt.Sprintf("%v", st.Errors))
	builder.WriteString(", ")
	builder.WriteString("log_file_path=")
	builder.WriteString(st.LogFilePath)
	builder.WriteByte(')')
	return builder.String()
}

// NamedServerTaskToGinFileMiddleware returns the ServerTaskToGinFileMiddleware named value or an error if the edge was not
// loaded in eager-loading with this name.
func (st *ServerTask) NamedServerTaskToGinFileMiddleware(name string) ([]*GinFileMiddleware, error) {
	if st.Edges.namedServerTaskToGinFileMiddleware == nil {
		return nil, &NotLoadedError{edge: name}
	}
	nodes, ok := st.Edges.namedServerTaskToGinFileMiddleware[name]
	if !ok {
		return nil, &NotLoadedError{edge: name}
	}
	return nodes, nil
}

func (st *ServerTask) appendNamedServerTaskToGinFileMiddleware(name string, edges ...*GinFileMiddleware) {
	if st.Edges.namedServerTaskToGinFileMiddleware == nil {
		st.Edges.namedServerTaskToGinFileMiddleware = make(map[string][]*GinFileMiddleware)
	}
	if len(edges) == 0 {
		st.Edges.namedServerTaskToGinFileMiddleware[name] = []*GinFileMiddleware{}
	} else {
		st.Edges.namedServerTaskToGinFileMiddleware[name] = append(st.Edges.namedServerTaskToGinFileMiddleware[name], edges...)
	}
}

// ServerTasks is a parsable slice of ServerTask.
type ServerTasks []*ServerTask
