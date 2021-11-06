// Code generated by entc, DO NOT EDIT.

package ent

import (
	"fmt"
	"strings"

	"entgo.io/ent/dialect/sql"
	"github.com/gen0cide/laforge/ent/agenttask"
	"github.com/gen0cide/laforge/ent/provisionedhost"
	"github.com/gen0cide/laforge/ent/provisioningstep"
	"github.com/google/uuid"
)

// AgentTask is the model entity for the AgentTask schema.
type AgentTask struct {
	config `json:"-"`
	// ID of the ent.
	ID uuid.UUID `json:"id,omitempty"`
	// Command holds the value of the "command" field.
	Command agenttask.Command `json:"command,omitempty"`
	// Args holds the value of the "args" field.
	Args string `json:"args,omitempty"`
	// Number holds the value of the "number" field.
	Number int `json:"number,omitempty"`
	// Output holds the value of the "output" field.
	Output string `json:"output,omitempty"`
	// State holds the value of the "state" field.
	State agenttask.State `json:"state,omitempty"`
	// ErrorMessage holds the value of the "error_message" field.
	ErrorMessage string `json:"error_message,omitempty"`
	// Edges holds the relations/edges for other nodes in the graph.
	// The values are being populated by the AgentTaskQuery when eager-loading is set.
	Edges AgentTaskEdges `json:"edges"`

	// Edges put into the main struct to be loaded via hcl
	// AgentTaskToProvisioningStep holds the value of the AgentTaskToProvisioningStep edge.
	HCLAgentTaskToProvisioningStep *ProvisioningStep `json:"AgentTaskToProvisioningStep,omitempty"`
	// AgentTaskToProvisionedHost holds the value of the AgentTaskToProvisionedHost edge.
	HCLAgentTaskToProvisionedHost *ProvisionedHost `json:"AgentTaskToProvisionedHost,omitempty"`
	// AgentTaskToAdhocPlan holds the value of the AgentTaskToAdhocPlan edge.
	HCLAgentTaskToAdhocPlan []*AdhocPlan `json:"AgentTaskToAdhocPlan,omitempty"`
	//
	agent_task_agent_task_to_provisioning_step *uuid.UUID
	agent_task_agent_task_to_provisioned_host  *uuid.UUID
}

// AgentTaskEdges holds the relations/edges for other nodes in the graph.
type AgentTaskEdges struct {
	// AgentTaskToProvisioningStep holds the value of the AgentTaskToProvisioningStep edge.
	AgentTaskToProvisioningStep *ProvisioningStep `json:"AgentTaskToProvisioningStep,omitempty"`
	// AgentTaskToProvisionedHost holds the value of the AgentTaskToProvisionedHost edge.
	AgentTaskToProvisionedHost *ProvisionedHost `json:"AgentTaskToProvisionedHost,omitempty"`
	// AgentTaskToAdhocPlan holds the value of the AgentTaskToAdhocPlan edge.
	AgentTaskToAdhocPlan []*AdhocPlan `json:"AgentTaskToAdhocPlan,omitempty"`
	// loadedTypes holds the information for reporting if a
	// type was loaded (or requested) in eager-loading or not.
	loadedTypes [3]bool
}

// AgentTaskToProvisioningStepOrErr returns the AgentTaskToProvisioningStep value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e AgentTaskEdges) AgentTaskToProvisioningStepOrErr() (*ProvisioningStep, error) {
	if e.loadedTypes[0] {
		if e.AgentTaskToProvisioningStep == nil {
			// The edge AgentTaskToProvisioningStep was loaded in eager-loading,
			// but was not found.
			return nil, &NotFoundError{label: provisioningstep.Label}
		}
		return e.AgentTaskToProvisioningStep, nil
	}
	return nil, &NotLoadedError{edge: "AgentTaskToProvisioningStep"}
}

// AgentTaskToProvisionedHostOrErr returns the AgentTaskToProvisionedHost value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e AgentTaskEdges) AgentTaskToProvisionedHostOrErr() (*ProvisionedHost, error) {
	if e.loadedTypes[1] {
		if e.AgentTaskToProvisionedHost == nil {
			// The edge AgentTaskToProvisionedHost was loaded in eager-loading,
			// but was not found.
			return nil, &NotFoundError{label: provisionedhost.Label}
		}
		return e.AgentTaskToProvisionedHost, nil
	}
	return nil, &NotLoadedError{edge: "AgentTaskToProvisionedHost"}
}

// AgentTaskToAdhocPlanOrErr returns the AgentTaskToAdhocPlan value or an error if the edge
// was not loaded in eager-loading.
func (e AgentTaskEdges) AgentTaskToAdhocPlanOrErr() ([]*AdhocPlan, error) {
	if e.loadedTypes[2] {
		return e.AgentTaskToAdhocPlan, nil
	}
	return nil, &NotLoadedError{edge: "AgentTaskToAdhocPlan"}
}

// scanValues returns the types for scanning values from sql.Rows.
func (*AgentTask) scanValues(columns []string) ([]interface{}, error) {
	values := make([]interface{}, len(columns))
	for i := range columns {
		switch columns[i] {
		case agenttask.FieldNumber:
			values[i] = new(sql.NullInt64)
		case agenttask.FieldCommand, agenttask.FieldArgs, agenttask.FieldOutput, agenttask.FieldState, agenttask.FieldErrorMessage:
			values[i] = new(sql.NullString)
		case agenttask.FieldID:
			values[i] = new(uuid.UUID)
		case agenttask.ForeignKeys[0]: // agent_task_agent_task_to_provisioning_step
			values[i] = &sql.NullScanner{S: new(uuid.UUID)}
		case agenttask.ForeignKeys[1]: // agent_task_agent_task_to_provisioned_host
			values[i] = &sql.NullScanner{S: new(uuid.UUID)}
		default:
			return nil, fmt.Errorf("unexpected column %q for type AgentTask", columns[i])
		}
	}
	return values, nil
}

// assignValues assigns the values that were returned from sql.Rows (after scanning)
// to the AgentTask fields.
func (at *AgentTask) assignValues(columns []string, values []interface{}) error {
	if m, n := len(values), len(columns); m < n {
		return fmt.Errorf("mismatch number of scan values: %d != %d", m, n)
	}
	for i := range columns {
		switch columns[i] {
		case agenttask.FieldID:
			if value, ok := values[i].(*uuid.UUID); !ok {
				return fmt.Errorf("unexpected type %T for field id", values[i])
			} else if value != nil {
				at.ID = *value
			}
		case agenttask.FieldCommand:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field command", values[i])
			} else if value.Valid {
				at.Command = agenttask.Command(value.String)
			}
		case agenttask.FieldArgs:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field args", values[i])
			} else if value.Valid {
				at.Args = value.String
			}
		case agenttask.FieldNumber:
			if value, ok := values[i].(*sql.NullInt64); !ok {
				return fmt.Errorf("unexpected type %T for field number", values[i])
			} else if value.Valid {
				at.Number = int(value.Int64)
			}
		case agenttask.FieldOutput:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field output", values[i])
			} else if value.Valid {
				at.Output = value.String
			}
		case agenttask.FieldState:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field state", values[i])
			} else if value.Valid {
				at.State = agenttask.State(value.String)
			}
		case agenttask.FieldErrorMessage:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field error_message", values[i])
			} else if value.Valid {
				at.ErrorMessage = value.String
			}
		case agenttask.ForeignKeys[0]:
			if value, ok := values[i].(*sql.NullScanner); !ok {
				return fmt.Errorf("unexpected type %T for field agent_task_agent_task_to_provisioning_step", values[i])
			} else if value.Valid {
				at.agent_task_agent_task_to_provisioning_step = new(uuid.UUID)
				*at.agent_task_agent_task_to_provisioning_step = *value.S.(*uuid.UUID)
			}
		case agenttask.ForeignKeys[1]:
			if value, ok := values[i].(*sql.NullScanner); !ok {
				return fmt.Errorf("unexpected type %T for field agent_task_agent_task_to_provisioned_host", values[i])
			} else if value.Valid {
				at.agent_task_agent_task_to_provisioned_host = new(uuid.UUID)
				*at.agent_task_agent_task_to_provisioned_host = *value.S.(*uuid.UUID)
			}
		}
	}
	return nil
}

// QueryAgentTaskToProvisioningStep queries the "AgentTaskToProvisioningStep" edge of the AgentTask entity.
func (at *AgentTask) QueryAgentTaskToProvisioningStep() *ProvisioningStepQuery {
	return (&AgentTaskClient{config: at.config}).QueryAgentTaskToProvisioningStep(at)
}

// QueryAgentTaskToProvisionedHost queries the "AgentTaskToProvisionedHost" edge of the AgentTask entity.
func (at *AgentTask) QueryAgentTaskToProvisionedHost() *ProvisionedHostQuery {
	return (&AgentTaskClient{config: at.config}).QueryAgentTaskToProvisionedHost(at)
}

// QueryAgentTaskToAdhocPlan queries the "AgentTaskToAdhocPlan" edge of the AgentTask entity.
func (at *AgentTask) QueryAgentTaskToAdhocPlan() *AdhocPlanQuery {
	return (&AgentTaskClient{config: at.config}).QueryAgentTaskToAdhocPlan(at)
}

// Update returns a builder for updating this AgentTask.
// Note that you need to call AgentTask.Unwrap() before calling this method if this AgentTask
// was returned from a transaction, and the transaction was committed or rolled back.
func (at *AgentTask) Update() *AgentTaskUpdateOne {
	return (&AgentTaskClient{config: at.config}).UpdateOne(at)
}

// Unwrap unwraps the AgentTask entity that was returned from a transaction after it was closed,
// so that all future queries will be executed through the driver which created the transaction.
func (at *AgentTask) Unwrap() *AgentTask {
	tx, ok := at.config.driver.(*txDriver)
	if !ok {
		panic("ent: AgentTask is not a transactional entity")
	}
	at.config.driver = tx.drv
	return at
}

// String implements the fmt.Stringer.
func (at *AgentTask) String() string {
	var builder strings.Builder
	builder.WriteString("AgentTask(")
	builder.WriteString(fmt.Sprintf("id=%v", at.ID))
	builder.WriteString(", command=")
	builder.WriteString(fmt.Sprintf("%v", at.Command))
	builder.WriteString(", args=")
	builder.WriteString(at.Args)
	builder.WriteString(", number=")
	builder.WriteString(fmt.Sprintf("%v", at.Number))
	builder.WriteString(", output=")
	builder.WriteString(at.Output)
	builder.WriteString(", state=")
	builder.WriteString(fmt.Sprintf("%v", at.State))
	builder.WriteString(", error_message=")
	builder.WriteString(at.ErrorMessage)
	builder.WriteByte(')')
	return builder.String()
}

// AgentTasks is a parsable slice of AgentTask.
type AgentTasks []*AgentTask

func (at AgentTasks) config(cfg config) {
	for _i := range at {
		at[_i].config = cfg
	}
}
