// Code generated by entc, DO NOT EDIT.

package ent

import (
	"fmt"
	"strings"

	"entgo.io/ent/dialect/sql"
	"github.com/gen0cide/laforge/ent/agentstatus"
	"github.com/gen0cide/laforge/ent/build"
	"github.com/gen0cide/laforge/ent/provisionedhost"
	"github.com/gen0cide/laforge/ent/provisionednetwork"
	"github.com/google/uuid"
)

// AgentStatus is the model entity for the AgentStatus schema.
type AgentStatus struct {
	config `json:"-"`
	// ID of the ent.
	ID uuid.UUID `json:"id,omitempty"`
	// ClientID holds the value of the "ClientID" field.
	ClientID string `json:"ClientID,omitempty"`
	// Hostname holds the value of the "Hostname" field.
	Hostname string `json:"Hostname,omitempty"`
	// UpTime holds the value of the "UpTime" field.
	UpTime int64 `json:"UpTime,omitempty"`
	// BootTime holds the value of the "BootTime" field.
	BootTime int64 `json:"BootTime,omitempty"`
	// NumProcs holds the value of the "NumProcs" field.
	NumProcs int64 `json:"NumProcs,omitempty"`
	// Os holds the value of the "Os" field.
	Os string `json:"Os,omitempty"`
	// HostID holds the value of the "HostID" field.
	HostID string `json:"HostID,omitempty"`
	// Load1 holds the value of the "Load1" field.
	Load1 float64 `json:"Load1,omitempty"`
	// Load5 holds the value of the "Load5" field.
	Load5 float64 `json:"Load5,omitempty"`
	// Load15 holds the value of the "Load15" field.
	Load15 float64 `json:"Load15,omitempty"`
	// TotalMem holds the value of the "TotalMem" field.
	TotalMem int64 `json:"TotalMem,omitempty"`
	// FreeMem holds the value of the "FreeMem" field.
	FreeMem int64 `json:"FreeMem,omitempty"`
	// UsedMem holds the value of the "UsedMem" field.
	UsedMem int64 `json:"UsedMem,omitempty"`
	// Timestamp holds the value of the "Timestamp" field.
	Timestamp int64 `json:"Timestamp,omitempty"`
	// Edges holds the relations/edges for other nodes in the graph.
	// The values are being populated by the AgentStatusQuery when eager-loading is set.
	Edges AgentStatusEdges `json:"edges"`

	// Edges put into the main struct to be loaded via hcl
	// AgentStatusToProvisionedHost holds the value of the AgentStatusToProvisionedHost edge.
	HCLAgentStatusToProvisionedHost *ProvisionedHost `json:"AgentStatusToProvisionedHost,omitempty"`
	// AgentStatusToProvisionedNetwork holds the value of the AgentStatusToProvisionedNetwork edge.
	HCLAgentStatusToProvisionedNetwork *ProvisionedNetwork `json:"AgentStatusToProvisionedNetwork,omitempty"`
	// AgentStatusToBuild holds the value of the AgentStatusToBuild edge.
	HCLAgentStatusToBuild *Build `json:"AgentStatusToBuild,omitempty"`
	//
	agent_status_agent_status_to_provisioned_host    *uuid.UUID
	agent_status_agent_status_to_provisioned_network *uuid.UUID
	agent_status_agent_status_to_build               *uuid.UUID
}

// AgentStatusEdges holds the relations/edges for other nodes in the graph.
type AgentStatusEdges struct {
	// AgentStatusToProvisionedHost holds the value of the AgentStatusToProvisionedHost edge.
	AgentStatusToProvisionedHost *ProvisionedHost `json:"AgentStatusToProvisionedHost,omitempty"`
	// AgentStatusToProvisionedNetwork holds the value of the AgentStatusToProvisionedNetwork edge.
	AgentStatusToProvisionedNetwork *ProvisionedNetwork `json:"AgentStatusToProvisionedNetwork,omitempty"`
	// AgentStatusToBuild holds the value of the AgentStatusToBuild edge.
	AgentStatusToBuild *Build `json:"AgentStatusToBuild,omitempty"`
	// loadedTypes holds the information for reporting if a
	// type was loaded (or requested) in eager-loading or not.
	loadedTypes [3]bool
}

// AgentStatusToProvisionedHostOrErr returns the AgentStatusToProvisionedHost value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e AgentStatusEdges) AgentStatusToProvisionedHostOrErr() (*ProvisionedHost, error) {
	if e.loadedTypes[0] {
		if e.AgentStatusToProvisionedHost == nil {
			// The edge AgentStatusToProvisionedHost was loaded in eager-loading,
			// but was not found.
			return nil, &NotFoundError{label: provisionedhost.Label}
		}
		return e.AgentStatusToProvisionedHost, nil
	}
	return nil, &NotLoadedError{edge: "AgentStatusToProvisionedHost"}
}

// AgentStatusToProvisionedNetworkOrErr returns the AgentStatusToProvisionedNetwork value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e AgentStatusEdges) AgentStatusToProvisionedNetworkOrErr() (*ProvisionedNetwork, error) {
	if e.loadedTypes[1] {
		if e.AgentStatusToProvisionedNetwork == nil {
			// The edge AgentStatusToProvisionedNetwork was loaded in eager-loading,
			// but was not found.
			return nil, &NotFoundError{label: provisionednetwork.Label}
		}
		return e.AgentStatusToProvisionedNetwork, nil
	}
	return nil, &NotLoadedError{edge: "AgentStatusToProvisionedNetwork"}
}

// AgentStatusToBuildOrErr returns the AgentStatusToBuild value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e AgentStatusEdges) AgentStatusToBuildOrErr() (*Build, error) {
	if e.loadedTypes[2] {
		if e.AgentStatusToBuild == nil {
			// The edge AgentStatusToBuild was loaded in eager-loading,
			// but was not found.
			return nil, &NotFoundError{label: build.Label}
		}
		return e.AgentStatusToBuild, nil
	}
	return nil, &NotLoadedError{edge: "AgentStatusToBuild"}
}

// scanValues returns the types for scanning values from sql.Rows.
func (*AgentStatus) scanValues(columns []string) ([]interface{}, error) {
	values := make([]interface{}, len(columns))
	for i := range columns {
		switch columns[i] {
		case agentstatus.FieldLoad1, agentstatus.FieldLoad5, agentstatus.FieldLoad15:
			values[i] = new(sql.NullFloat64)
		case agentstatus.FieldUpTime, agentstatus.FieldBootTime, agentstatus.FieldNumProcs, agentstatus.FieldTotalMem, agentstatus.FieldFreeMem, agentstatus.FieldUsedMem, agentstatus.FieldTimestamp:
			values[i] = new(sql.NullInt64)
		case agentstatus.FieldClientID, agentstatus.FieldHostname, agentstatus.FieldOs, agentstatus.FieldHostID:
			values[i] = new(sql.NullString)
		case agentstatus.FieldID:
			values[i] = new(uuid.UUID)
		case agentstatus.ForeignKeys[0]: // agent_status_agent_status_to_provisioned_host
			values[i] = &sql.NullScanner{S: new(uuid.UUID)}
		case agentstatus.ForeignKeys[1]: // agent_status_agent_status_to_provisioned_network
			values[i] = &sql.NullScanner{S: new(uuid.UUID)}
		case agentstatus.ForeignKeys[2]: // agent_status_agent_status_to_build
			values[i] = &sql.NullScanner{S: new(uuid.UUID)}
		default:
			return nil, fmt.Errorf("unexpected column %q for type AgentStatus", columns[i])
		}
	}
	return values, nil
}

// assignValues assigns the values that were returned from sql.Rows (after scanning)
// to the AgentStatus fields.
func (as *AgentStatus) assignValues(columns []string, values []interface{}) error {
	if m, n := len(values), len(columns); m < n {
		return fmt.Errorf("mismatch number of scan values: %d != %d", m, n)
	}
	for i := range columns {
		switch columns[i] {
		case agentstatus.FieldID:
			if value, ok := values[i].(*uuid.UUID); !ok {
				return fmt.Errorf("unexpected type %T for field id", values[i])
			} else if value != nil {
				as.ID = *value
			}
		case agentstatus.FieldClientID:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field ClientID", values[i])
			} else if value.Valid {
				as.ClientID = value.String
			}
		case agentstatus.FieldHostname:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field Hostname", values[i])
			} else if value.Valid {
				as.Hostname = value.String
			}
		case agentstatus.FieldUpTime:
			if value, ok := values[i].(*sql.NullInt64); !ok {
				return fmt.Errorf("unexpected type %T for field UpTime", values[i])
			} else if value.Valid {
				as.UpTime = value.Int64
			}
		case agentstatus.FieldBootTime:
			if value, ok := values[i].(*sql.NullInt64); !ok {
				return fmt.Errorf("unexpected type %T for field BootTime", values[i])
			} else if value.Valid {
				as.BootTime = value.Int64
			}
		case agentstatus.FieldNumProcs:
			if value, ok := values[i].(*sql.NullInt64); !ok {
				return fmt.Errorf("unexpected type %T for field NumProcs", values[i])
			} else if value.Valid {
				as.NumProcs = value.Int64
			}
		case agentstatus.FieldOs:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field Os", values[i])
			} else if value.Valid {
				as.Os = value.String
			}
		case agentstatus.FieldHostID:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field HostID", values[i])
			} else if value.Valid {
				as.HostID = value.String
			}
		case agentstatus.FieldLoad1:
			if value, ok := values[i].(*sql.NullFloat64); !ok {
				return fmt.Errorf("unexpected type %T for field Load1", values[i])
			} else if value.Valid {
				as.Load1 = value.Float64
			}
		case agentstatus.FieldLoad5:
			if value, ok := values[i].(*sql.NullFloat64); !ok {
				return fmt.Errorf("unexpected type %T for field Load5", values[i])
			} else if value.Valid {
				as.Load5 = value.Float64
			}
		case agentstatus.FieldLoad15:
			if value, ok := values[i].(*sql.NullFloat64); !ok {
				return fmt.Errorf("unexpected type %T for field Load15", values[i])
			} else if value.Valid {
				as.Load15 = value.Float64
			}
		case agentstatus.FieldTotalMem:
			if value, ok := values[i].(*sql.NullInt64); !ok {
				return fmt.Errorf("unexpected type %T for field TotalMem", values[i])
			} else if value.Valid {
				as.TotalMem = value.Int64
			}
		case agentstatus.FieldFreeMem:
			if value, ok := values[i].(*sql.NullInt64); !ok {
				return fmt.Errorf("unexpected type %T for field FreeMem", values[i])
			} else if value.Valid {
				as.FreeMem = value.Int64
			}
		case agentstatus.FieldUsedMem:
			if value, ok := values[i].(*sql.NullInt64); !ok {
				return fmt.Errorf("unexpected type %T for field UsedMem", values[i])
			} else if value.Valid {
				as.UsedMem = value.Int64
			}
		case agentstatus.FieldTimestamp:
			if value, ok := values[i].(*sql.NullInt64); !ok {
				return fmt.Errorf("unexpected type %T for field Timestamp", values[i])
			} else if value.Valid {
				as.Timestamp = value.Int64
			}
		case agentstatus.ForeignKeys[0]:
			if value, ok := values[i].(*sql.NullScanner); !ok {
				return fmt.Errorf("unexpected type %T for field agent_status_agent_status_to_provisioned_host", values[i])
			} else if value.Valid {
				as.agent_status_agent_status_to_provisioned_host = new(uuid.UUID)
				*as.agent_status_agent_status_to_provisioned_host = *value.S.(*uuid.UUID)
			}
		case agentstatus.ForeignKeys[1]:
			if value, ok := values[i].(*sql.NullScanner); !ok {
				return fmt.Errorf("unexpected type %T for field agent_status_agent_status_to_provisioned_network", values[i])
			} else if value.Valid {
				as.agent_status_agent_status_to_provisioned_network = new(uuid.UUID)
				*as.agent_status_agent_status_to_provisioned_network = *value.S.(*uuid.UUID)
			}
		case agentstatus.ForeignKeys[2]:
			if value, ok := values[i].(*sql.NullScanner); !ok {
				return fmt.Errorf("unexpected type %T for field agent_status_agent_status_to_build", values[i])
			} else if value.Valid {
				as.agent_status_agent_status_to_build = new(uuid.UUID)
				*as.agent_status_agent_status_to_build = *value.S.(*uuid.UUID)
			}
		}
	}
	return nil
}

// QueryAgentStatusToProvisionedHost queries the "AgentStatusToProvisionedHost" edge of the AgentStatus entity.
func (as *AgentStatus) QueryAgentStatusToProvisionedHost() *ProvisionedHostQuery {
	return (&AgentStatusClient{config: as.config}).QueryAgentStatusToProvisionedHost(as)
}

// QueryAgentStatusToProvisionedNetwork queries the "AgentStatusToProvisionedNetwork" edge of the AgentStatus entity.
func (as *AgentStatus) QueryAgentStatusToProvisionedNetwork() *ProvisionedNetworkQuery {
	return (&AgentStatusClient{config: as.config}).QueryAgentStatusToProvisionedNetwork(as)
}

// QueryAgentStatusToBuild queries the "AgentStatusToBuild" edge of the AgentStatus entity.
func (as *AgentStatus) QueryAgentStatusToBuild() *BuildQuery {
	return (&AgentStatusClient{config: as.config}).QueryAgentStatusToBuild(as)
}

// Update returns a builder for updating this AgentStatus.
// Note that you need to call AgentStatus.Unwrap() before calling this method if this AgentStatus
// was returned from a transaction, and the transaction was committed or rolled back.
func (as *AgentStatus) Update() *AgentStatusUpdateOne {
	return (&AgentStatusClient{config: as.config}).UpdateOne(as)
}

// Unwrap unwraps the AgentStatus entity that was returned from a transaction after it was closed,
// so that all future queries will be executed through the driver which created the transaction.
func (as *AgentStatus) Unwrap() *AgentStatus {
	tx, ok := as.config.driver.(*txDriver)
	if !ok {
		panic("ent: AgentStatus is not a transactional entity")
	}
	as.config.driver = tx.drv
	return as
}

// String implements the fmt.Stringer.
func (as *AgentStatus) String() string {
	var builder strings.Builder
	builder.WriteString("AgentStatus(")
	builder.WriteString(fmt.Sprintf("id=%v", as.ID))
	builder.WriteString(", ClientID=")
	builder.WriteString(as.ClientID)
	builder.WriteString(", Hostname=")
	builder.WriteString(as.Hostname)
	builder.WriteString(", UpTime=")
	builder.WriteString(fmt.Sprintf("%v", as.UpTime))
	builder.WriteString(", BootTime=")
	builder.WriteString(fmt.Sprintf("%v", as.BootTime))
	builder.WriteString(", NumProcs=")
	builder.WriteString(fmt.Sprintf("%v", as.NumProcs))
	builder.WriteString(", Os=")
	builder.WriteString(as.Os)
	builder.WriteString(", HostID=")
	builder.WriteString(as.HostID)
	builder.WriteString(", Load1=")
	builder.WriteString(fmt.Sprintf("%v", as.Load1))
	builder.WriteString(", Load5=")
	builder.WriteString(fmt.Sprintf("%v", as.Load5))
	builder.WriteString(", Load15=")
	builder.WriteString(fmt.Sprintf("%v", as.Load15))
	builder.WriteString(", TotalMem=")
	builder.WriteString(fmt.Sprintf("%v", as.TotalMem))
	builder.WriteString(", FreeMem=")
	builder.WriteString(fmt.Sprintf("%v", as.FreeMem))
	builder.WriteString(", UsedMem=")
	builder.WriteString(fmt.Sprintf("%v", as.UsedMem))
	builder.WriteString(", Timestamp=")
	builder.WriteString(fmt.Sprintf("%v", as.Timestamp))
	builder.WriteByte(')')
	return builder.String()
}

// AgentStatusSlice is a parsable slice of AgentStatus.
type AgentStatusSlice []*AgentStatus

func (as AgentStatusSlice) config(cfg config) {
	for _i := range as {
		as[_i].config = cfg
	}
}
