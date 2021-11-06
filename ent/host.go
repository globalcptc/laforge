// Code generated by entc, DO NOT EDIT.

package ent

import (
	"encoding/json"
	"fmt"
	"strings"

	"entgo.io/ent/dialect/sql"
	"github.com/gen0cide/laforge/ent/disk"
	"github.com/gen0cide/laforge/ent/environment"
	"github.com/gen0cide/laforge/ent/host"
	"github.com/google/uuid"
)

// Host is the model entity for the Host schema.
type Host struct {
	config ` json:"-"`
	// ID of the ent.
	ID uuid.UUID `json:"id,omitempty"`
	// HclID holds the value of the "hcl_id" field.
	HclID string `json:"hcl_id,omitempty" hcl:"id,label"`
	// Hostname holds the value of the "hostname" field.
	Hostname string `json:"hostname,omitempty" hcl:"hostname,attr"`
	// Description holds the value of the "description" field.
	Description string `json:"description,omitempty" hcl:"description,optional" `
	// OS holds the value of the "OS" field.
	OS string `json:"OS,omitempty" hcl:"os,attr"`
	// LastOctet holds the value of the "last_octet" field.
	LastOctet int `json:"last_octet,omitempty" hcl:"last_octet,attr"`
	// InstanceSize holds the value of the "instance_size" field.
	InstanceSize string `json:"instance_size,omitempty" hcl:"instance_size,attr"`
	// AllowMACChanges holds the value of the "allow_mac_changes" field.
	AllowMACChanges bool `json:"allow_mac_changes,omitempty" hcl:"allow_mac_changes,optional"`
	// ExposedTCPPorts holds the value of the "exposed_tcp_ports" field.
	ExposedTCPPorts []string `json:"exposed_tcp_ports,omitempty" hcl:"exposed_tcp_ports,optional"`
	// ExposedUDPPorts holds the value of the "exposed_udp_ports" field.
	ExposedUDPPorts []string `json:"exposed_udp_ports,omitempty" hcl:"exposed_udp_ports,optional"`
	// OverridePassword holds the value of the "override_password" field.
	OverridePassword string `json:"override_password,omitempty" hcl:"override_password,optional"`
	// Vars holds the value of the "vars" field.
	Vars map[string]string `json:"vars,omitempty" hcl:"vars,optional"`
	// UserGroups holds the value of the "user_groups" field.
	UserGroups []string `json:"user_groups,omitempty" hcl:"user_groups,optional"`
	// ProvisionSteps holds the value of the "provision_steps" field.
	ProvisionSteps []string `json:"provision_steps,omitempty" hcl:"provision_steps,optional"`
	// Tags holds the value of the "tags" field.
	Tags map[string]string `json:"tags,omitempty" hcl:"tags,optional"`
	// Edges holds the relations/edges for other nodes in the graph.
	// The values are being populated by the HostQuery when eager-loading is set.
	Edges HostEdges `json:"edges"`

	// Edges put into the main struct to be loaded via hcl
	// HostToDisk holds the value of the HostToDisk edge.
	HCLHostToDisk *Disk `json:"HostToDisk,omitempty" hcl:"disk,block"`
	// HostToUser holds the value of the HostToUser edge.
	HCLHostToUser []*User `json:"HostToUser,omitempty" hcl:"maintainer,block"`
	// HostToEnvironment holds the value of the HostToEnvironment edge.
	HCLHostToEnvironment *Environment `json:"HostToEnvironment,omitempty"`
	// HostToIncludedNetwork holds the value of the HostToIncludedNetwork edge.
	HCLHostToIncludedNetwork []*IncludedNetwork `json:"HostToIncludedNetwork,omitempty"`
	// DependOnHostToHostDependency holds the value of the DependOnHostToHostDependency edge.
	HCLDependOnHostToHostDependency []*HostDependency `json:"DependOnHostToHostDependency,omitempty" hcl:"depends_on,block"`
	// DependByHostToHostDependency holds the value of the DependByHostToHostDependency edge.
	HCLDependByHostToHostDependency []*HostDependency `json:"DependByHostToHostDependency,omitempty"`
	//
	environment_environment_to_host *uuid.UUID
}

// HostEdges holds the relations/edges for other nodes in the graph.
type HostEdges struct {
	// HostToDisk holds the value of the HostToDisk edge.
	HostToDisk *Disk `json:"HostToDisk,omitempty" hcl:"disk,block"`
	// HostToUser holds the value of the HostToUser edge.
	HostToUser []*User `json:"HostToUser,omitempty" hcl:"maintainer,block"`
	// HostToEnvironment holds the value of the HostToEnvironment edge.
	HostToEnvironment *Environment `json:"HostToEnvironment,omitempty"`
	// HostToIncludedNetwork holds the value of the HostToIncludedNetwork edge.
	HostToIncludedNetwork []*IncludedNetwork `json:"HostToIncludedNetwork,omitempty"`
	// DependOnHostToHostDependency holds the value of the DependOnHostToHostDependency edge.
	DependOnHostToHostDependency []*HostDependency `json:"DependOnHostToHostDependency,omitempty" hcl:"depends_on,block"`
	// DependByHostToHostDependency holds the value of the DependByHostToHostDependency edge.
	DependByHostToHostDependency []*HostDependency `json:"DependByHostToHostDependency,omitempty"`
	// loadedTypes holds the information for reporting if a
	// type was loaded (or requested) in eager-loading or not.
	loadedTypes [6]bool
}

// HostToDiskOrErr returns the HostToDisk value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e HostEdges) HostToDiskOrErr() (*Disk, error) {
	if e.loadedTypes[0] {
		if e.HostToDisk == nil {
			// The edge HostToDisk was loaded in eager-loading,
			// but was not found.
			return nil, &NotFoundError{label: disk.Label}
		}
		return e.HostToDisk, nil
	}
	return nil, &NotLoadedError{edge: "HostToDisk"}
}

// HostToUserOrErr returns the HostToUser value or an error if the edge
// was not loaded in eager-loading.
func (e HostEdges) HostToUserOrErr() ([]*User, error) {
	if e.loadedTypes[1] {
		return e.HostToUser, nil
	}
	return nil, &NotLoadedError{edge: "HostToUser"}
}

// HostToEnvironmentOrErr returns the HostToEnvironment value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e HostEdges) HostToEnvironmentOrErr() (*Environment, error) {
	if e.loadedTypes[2] {
		if e.HostToEnvironment == nil {
			// The edge HostToEnvironment was loaded in eager-loading,
			// but was not found.
			return nil, &NotFoundError{label: environment.Label}
		}
		return e.HostToEnvironment, nil
	}
	return nil, &NotLoadedError{edge: "HostToEnvironment"}
}

// HostToIncludedNetworkOrErr returns the HostToIncludedNetwork value or an error if the edge
// was not loaded in eager-loading.
func (e HostEdges) HostToIncludedNetworkOrErr() ([]*IncludedNetwork, error) {
	if e.loadedTypes[3] {
		return e.HostToIncludedNetwork, nil
	}
	return nil, &NotLoadedError{edge: "HostToIncludedNetwork"}
}

// DependOnHostToHostDependencyOrErr returns the DependOnHostToHostDependency value or an error if the edge
// was not loaded in eager-loading.
func (e HostEdges) DependOnHostToHostDependencyOrErr() ([]*HostDependency, error) {
	if e.loadedTypes[4] {
		return e.DependOnHostToHostDependency, nil
	}
	return nil, &NotLoadedError{edge: "DependOnHostToHostDependency"}
}

// DependByHostToHostDependencyOrErr returns the DependByHostToHostDependency value or an error if the edge
// was not loaded in eager-loading.
func (e HostEdges) DependByHostToHostDependencyOrErr() ([]*HostDependency, error) {
	if e.loadedTypes[5] {
		return e.DependByHostToHostDependency, nil
	}
	return nil, &NotLoadedError{edge: "DependByHostToHostDependency"}
}

// scanValues returns the types for scanning values from sql.Rows.
func (*Host) scanValues(columns []string) ([]interface{}, error) {
	values := make([]interface{}, len(columns))
	for i := range columns {
		switch columns[i] {
		case host.FieldExposedTCPPorts, host.FieldExposedUDPPorts, host.FieldVars, host.FieldUserGroups, host.FieldProvisionSteps, host.FieldTags:
			values[i] = new([]byte)
		case host.FieldAllowMACChanges:
			values[i] = new(sql.NullBool)
		case host.FieldLastOctet:
			values[i] = new(sql.NullInt64)
		case host.FieldHclID, host.FieldHostname, host.FieldDescription, host.FieldOS, host.FieldInstanceSize, host.FieldOverridePassword:
			values[i] = new(sql.NullString)
		case host.FieldID:
			values[i] = new(uuid.UUID)
		case host.ForeignKeys[0]: // environment_environment_to_host
			values[i] = &sql.NullScanner{S: new(uuid.UUID)}
		default:
			return nil, fmt.Errorf("unexpected column %q for type Host", columns[i])
		}
	}
	return values, nil
}

// assignValues assigns the values that were returned from sql.Rows (after scanning)
// to the Host fields.
func (h *Host) assignValues(columns []string, values []interface{}) error {
	if m, n := len(values), len(columns); m < n {
		return fmt.Errorf("mismatch number of scan values: %d != %d", m, n)
	}
	for i := range columns {
		switch columns[i] {
		case host.FieldID:
			if value, ok := values[i].(*uuid.UUID); !ok {
				return fmt.Errorf("unexpected type %T for field id", values[i])
			} else if value != nil {
				h.ID = *value
			}
		case host.FieldHclID:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field hcl_id", values[i])
			} else if value.Valid {
				h.HclID = value.String
			}
		case host.FieldHostname:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field hostname", values[i])
			} else if value.Valid {
				h.Hostname = value.String
			}
		case host.FieldDescription:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field description", values[i])
			} else if value.Valid {
				h.Description = value.String
			}
		case host.FieldOS:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field OS", values[i])
			} else if value.Valid {
				h.OS = value.String
			}
		case host.FieldLastOctet:
			if value, ok := values[i].(*sql.NullInt64); !ok {
				return fmt.Errorf("unexpected type %T for field last_octet", values[i])
			} else if value.Valid {
				h.LastOctet = int(value.Int64)
			}
		case host.FieldInstanceSize:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field instance_size", values[i])
			} else if value.Valid {
				h.InstanceSize = value.String
			}
		case host.FieldAllowMACChanges:
			if value, ok := values[i].(*sql.NullBool); !ok {
				return fmt.Errorf("unexpected type %T for field allow_mac_changes", values[i])
			} else if value.Valid {
				h.AllowMACChanges = value.Bool
			}
		case host.FieldExposedTCPPorts:
			if value, ok := values[i].(*[]byte); !ok {
				return fmt.Errorf("unexpected type %T for field exposed_tcp_ports", values[i])
			} else if value != nil && len(*value) > 0 {
				if err := json.Unmarshal(*value, &h.ExposedTCPPorts); err != nil {
					return fmt.Errorf("unmarshal field exposed_tcp_ports: %w", err)
				}
			}
		case host.FieldExposedUDPPorts:
			if value, ok := values[i].(*[]byte); !ok {
				return fmt.Errorf("unexpected type %T for field exposed_udp_ports", values[i])
			} else if value != nil && len(*value) > 0 {
				if err := json.Unmarshal(*value, &h.ExposedUDPPorts); err != nil {
					return fmt.Errorf("unmarshal field exposed_udp_ports: %w", err)
				}
			}
		case host.FieldOverridePassword:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field override_password", values[i])
			} else if value.Valid {
				h.OverridePassword = value.String
			}
		case host.FieldVars:
			if value, ok := values[i].(*[]byte); !ok {
				return fmt.Errorf("unexpected type %T for field vars", values[i])
			} else if value != nil && len(*value) > 0 {
				if err := json.Unmarshal(*value, &h.Vars); err != nil {
					return fmt.Errorf("unmarshal field vars: %w", err)
				}
			}
		case host.FieldUserGroups:
			if value, ok := values[i].(*[]byte); !ok {
				return fmt.Errorf("unexpected type %T for field user_groups", values[i])
			} else if value != nil && len(*value) > 0 {
				if err := json.Unmarshal(*value, &h.UserGroups); err != nil {
					return fmt.Errorf("unmarshal field user_groups: %w", err)
				}
			}
		case host.FieldProvisionSteps:
			if value, ok := values[i].(*[]byte); !ok {
				return fmt.Errorf("unexpected type %T for field provision_steps", values[i])
			} else if value != nil && len(*value) > 0 {
				if err := json.Unmarshal(*value, &h.ProvisionSteps); err != nil {
					return fmt.Errorf("unmarshal field provision_steps: %w", err)
				}
			}
		case host.FieldTags:
			if value, ok := values[i].(*[]byte); !ok {
				return fmt.Errorf("unexpected type %T for field tags", values[i])
			} else if value != nil && len(*value) > 0 {
				if err := json.Unmarshal(*value, &h.Tags); err != nil {
					return fmt.Errorf("unmarshal field tags: %w", err)
				}
			}
		case host.ForeignKeys[0]:
			if value, ok := values[i].(*sql.NullScanner); !ok {
				return fmt.Errorf("unexpected type %T for field environment_environment_to_host", values[i])
			} else if value.Valid {
				h.environment_environment_to_host = new(uuid.UUID)
				*h.environment_environment_to_host = *value.S.(*uuid.UUID)
			}
		}
	}
	return nil
}

// QueryHostToDisk queries the "HostToDisk" edge of the Host entity.
func (h *Host) QueryHostToDisk() *DiskQuery {
	return (&HostClient{config: h.config}).QueryHostToDisk(h)
}

// QueryHostToUser queries the "HostToUser" edge of the Host entity.
func (h *Host) QueryHostToUser() *UserQuery {
	return (&HostClient{config: h.config}).QueryHostToUser(h)
}

// QueryHostToEnvironment queries the "HostToEnvironment" edge of the Host entity.
func (h *Host) QueryHostToEnvironment() *EnvironmentQuery {
	return (&HostClient{config: h.config}).QueryHostToEnvironment(h)
}

// QueryHostToIncludedNetwork queries the "HostToIncludedNetwork" edge of the Host entity.
func (h *Host) QueryHostToIncludedNetwork() *IncludedNetworkQuery {
	return (&HostClient{config: h.config}).QueryHostToIncludedNetwork(h)
}

// QueryDependOnHostToHostDependency queries the "DependOnHostToHostDependency" edge of the Host entity.
func (h *Host) QueryDependOnHostToHostDependency() *HostDependencyQuery {
	return (&HostClient{config: h.config}).QueryDependOnHostToHostDependency(h)
}

// QueryDependByHostToHostDependency queries the "DependByHostToHostDependency" edge of the Host entity.
func (h *Host) QueryDependByHostToHostDependency() *HostDependencyQuery {
	return (&HostClient{config: h.config}).QueryDependByHostToHostDependency(h)
}

// Update returns a builder for updating this Host.
// Note that you need to call Host.Unwrap() before calling this method if this Host
// was returned from a transaction, and the transaction was committed or rolled back.
func (h *Host) Update() *HostUpdateOne {
	return (&HostClient{config: h.config}).UpdateOne(h)
}

// Unwrap unwraps the Host entity that was returned from a transaction after it was closed,
// so that all future queries will be executed through the driver which created the transaction.
func (h *Host) Unwrap() *Host {
	tx, ok := h.config.driver.(*txDriver)
	if !ok {
		panic("ent: Host is not a transactional entity")
	}
	h.config.driver = tx.drv
	return h
}

// String implements the fmt.Stringer.
func (h *Host) String() string {
	var builder strings.Builder
	builder.WriteString("Host(")
	builder.WriteString(fmt.Sprintf("id=%v", h.ID))
	builder.WriteString(", hcl_id=")
	builder.WriteString(h.HclID)
	builder.WriteString(", hostname=")
	builder.WriteString(h.Hostname)
	builder.WriteString(", description=")
	builder.WriteString(h.Description)
	builder.WriteString(", OS=")
	builder.WriteString(h.OS)
	builder.WriteString(", last_octet=")
	builder.WriteString(fmt.Sprintf("%v", h.LastOctet))
	builder.WriteString(", instance_size=")
	builder.WriteString(h.InstanceSize)
	builder.WriteString(", allow_mac_changes=")
	builder.WriteString(fmt.Sprintf("%v", h.AllowMACChanges))
	builder.WriteString(", exposed_tcp_ports=")
	builder.WriteString(fmt.Sprintf("%v", h.ExposedTCPPorts))
	builder.WriteString(", exposed_udp_ports=")
	builder.WriteString(fmt.Sprintf("%v", h.ExposedUDPPorts))
	builder.WriteString(", override_password=")
	builder.WriteString(h.OverridePassword)
	builder.WriteString(", vars=")
	builder.WriteString(fmt.Sprintf("%v", h.Vars))
	builder.WriteString(", user_groups=")
	builder.WriteString(fmt.Sprintf("%v", h.UserGroups))
	builder.WriteString(", provision_steps=")
	builder.WriteString(fmt.Sprintf("%v", h.ProvisionSteps))
	builder.WriteString(", tags=")
	builder.WriteString(fmt.Sprintf("%v", h.Tags))
	builder.WriteByte(')')
	return builder.String()
}

// Hosts is a parsable slice of Host.
type Hosts []*Host

func (h Hosts) config(cfg config) {
	for _i := range h {
		h[_i].config = cfg
	}
}
