// Code generated by entc, DO NOT EDIT.

package ent

import (
	"encoding/json"
	"fmt"
	"strings"

	"entgo.io/ent/dialect/sql"
	"github.com/gen0cide/laforge/ent/environment"
	"github.com/gen0cide/laforge/ent/network"
	"github.com/google/uuid"
)

// Network is the model entity for the Network schema.
type Network struct {
	config ` json:"-"`
	// ID of the ent.
	ID uuid.UUID `json:"id,omitempty"`
	// HclID holds the value of the "hcl_id" field.
	HclID string `json:"hcl_id,omitempty" hcl:"id,label"`
	// Name holds the value of the "name" field.
	Name string `json:"name,omitempty" hcl:"name,attr"`
	// Cidr holds the value of the "cidr" field.
	Cidr string `json:"cidr,omitempty" hcl:"cidr,attr"`
	// VdiVisible holds the value of the "vdi_visible" field.
	VdiVisible bool `json:"vdi_visible,omitempty" hcl:"vdi_visible,optional"`
	// Vars holds the value of the "vars" field.
	Vars map[string]string `json:"vars,omitempty" hcl:"vars,optional"`
	// Tags holds the value of the "tags" field.
	Tags map[string]string `json:"tags,omitempty" hcl:"tags,optional"`
	// Edges holds the relations/edges for other nodes in the graph.
	// The values are being populated by the NetworkQuery when eager-loading is set.
	Edges NetworkEdges `json:"edges"`

	// Edges put into the main struct to be loaded via hcl
	// NetworkToEnvironment holds the value of the NetworkToEnvironment edge.
	HCLNetworkToEnvironment *Environment `json:"NetworkToEnvironment,omitempty"`
	// NetworkToHostDependency holds the value of the NetworkToHostDependency edge.
	HCLNetworkToHostDependency []*HostDependency `json:"NetworkToHostDependency,omitempty"`
	// NetworkToIncludedNetwork holds the value of the NetworkToIncludedNetwork edge.
	HCLNetworkToIncludedNetwork []*IncludedNetwork `json:"NetworkToIncludedNetwork,omitempty"`
	//
	environment_environment_to_network *uuid.UUID
}

// NetworkEdges holds the relations/edges for other nodes in the graph.
type NetworkEdges struct {
	// NetworkToEnvironment holds the value of the NetworkToEnvironment edge.
	NetworkToEnvironment *Environment `json:"NetworkToEnvironment,omitempty"`
	// NetworkToHostDependency holds the value of the NetworkToHostDependency edge.
	NetworkToHostDependency []*HostDependency `json:"NetworkToHostDependency,omitempty"`
	// NetworkToIncludedNetwork holds the value of the NetworkToIncludedNetwork edge.
	NetworkToIncludedNetwork []*IncludedNetwork `json:"NetworkToIncludedNetwork,omitempty"`
	// loadedTypes holds the information for reporting if a
	// type was loaded (or requested) in eager-loading or not.
	loadedTypes [3]bool
}

// NetworkToEnvironmentOrErr returns the NetworkToEnvironment value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e NetworkEdges) NetworkToEnvironmentOrErr() (*Environment, error) {
	if e.loadedTypes[0] {
		if e.NetworkToEnvironment == nil {
			// The edge NetworkToEnvironment was loaded in eager-loading,
			// but was not found.
			return nil, &NotFoundError{label: environment.Label}
		}
		return e.NetworkToEnvironment, nil
	}
	return nil, &NotLoadedError{edge: "NetworkToEnvironment"}
}

// NetworkToHostDependencyOrErr returns the NetworkToHostDependency value or an error if the edge
// was not loaded in eager-loading.
func (e NetworkEdges) NetworkToHostDependencyOrErr() ([]*HostDependency, error) {
	if e.loadedTypes[1] {
		return e.NetworkToHostDependency, nil
	}
	return nil, &NotLoadedError{edge: "NetworkToHostDependency"}
}

// NetworkToIncludedNetworkOrErr returns the NetworkToIncludedNetwork value or an error if the edge
// was not loaded in eager-loading.
func (e NetworkEdges) NetworkToIncludedNetworkOrErr() ([]*IncludedNetwork, error) {
	if e.loadedTypes[2] {
		return e.NetworkToIncludedNetwork, nil
	}
	return nil, &NotLoadedError{edge: "NetworkToIncludedNetwork"}
}

// scanValues returns the types for scanning values from sql.Rows.
func (*Network) scanValues(columns []string) ([]interface{}, error) {
	values := make([]interface{}, len(columns))
	for i := range columns {
		switch columns[i] {
		case network.FieldVars, network.FieldTags:
			values[i] = new([]byte)
		case network.FieldVdiVisible:
			values[i] = new(sql.NullBool)
		case network.FieldHclID, network.FieldName, network.FieldCidr:
			values[i] = new(sql.NullString)
		case network.FieldID:
			values[i] = new(uuid.UUID)
		case network.ForeignKeys[0]: // environment_environment_to_network
			values[i] = &sql.NullScanner{S: new(uuid.UUID)}
		default:
			return nil, fmt.Errorf("unexpected column %q for type Network", columns[i])
		}
	}
	return values, nil
}

// assignValues assigns the values that were returned from sql.Rows (after scanning)
// to the Network fields.
func (n *Network) assignValues(columns []string, values []interface{}) error {
	if m, n := len(values), len(columns); m < n {
		return fmt.Errorf("mismatch number of scan values: %d != %d", m, n)
	}
	for i := range columns {
		switch columns[i] {
		case network.FieldID:
			if value, ok := values[i].(*uuid.UUID); !ok {
				return fmt.Errorf("unexpected type %T for field id", values[i])
			} else if value != nil {
				n.ID = *value
			}
		case network.FieldHclID:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field hcl_id", values[i])
			} else if value.Valid {
				n.HclID = value.String
			}
		case network.FieldName:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field name", values[i])
			} else if value.Valid {
				n.Name = value.String
			}
		case network.FieldCidr:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field cidr", values[i])
			} else if value.Valid {
				n.Cidr = value.String
			}
		case network.FieldVdiVisible:
			if value, ok := values[i].(*sql.NullBool); !ok {
				return fmt.Errorf("unexpected type %T for field vdi_visible", values[i])
			} else if value.Valid {
				n.VdiVisible = value.Bool
			}
		case network.FieldVars:
			if value, ok := values[i].(*[]byte); !ok {
				return fmt.Errorf("unexpected type %T for field vars", values[i])
			} else if value != nil && len(*value) > 0 {
				if err := json.Unmarshal(*value, &n.Vars); err != nil {
					return fmt.Errorf("unmarshal field vars: %w", err)
				}
			}
		case network.FieldTags:
			if value, ok := values[i].(*[]byte); !ok {
				return fmt.Errorf("unexpected type %T for field tags", values[i])
			} else if value != nil && len(*value) > 0 {
				if err := json.Unmarshal(*value, &n.Tags); err != nil {
					return fmt.Errorf("unmarshal field tags: %w", err)
				}
			}
		case network.ForeignKeys[0]:
			if value, ok := values[i].(*sql.NullScanner); !ok {
				return fmt.Errorf("unexpected type %T for field environment_environment_to_network", values[i])
			} else if value.Valid {
				n.environment_environment_to_network = new(uuid.UUID)
				*n.environment_environment_to_network = *value.S.(*uuid.UUID)
			}
		}
	}
	return nil
}

// QueryNetworkToEnvironment queries the "NetworkToEnvironment" edge of the Network entity.
func (n *Network) QueryNetworkToEnvironment() *EnvironmentQuery {
	return (&NetworkClient{config: n.config}).QueryNetworkToEnvironment(n)
}

// QueryNetworkToHostDependency queries the "NetworkToHostDependency" edge of the Network entity.
func (n *Network) QueryNetworkToHostDependency() *HostDependencyQuery {
	return (&NetworkClient{config: n.config}).QueryNetworkToHostDependency(n)
}

// QueryNetworkToIncludedNetwork queries the "NetworkToIncludedNetwork" edge of the Network entity.
func (n *Network) QueryNetworkToIncludedNetwork() *IncludedNetworkQuery {
	return (&NetworkClient{config: n.config}).QueryNetworkToIncludedNetwork(n)
}

// Update returns a builder for updating this Network.
// Note that you need to call Network.Unwrap() before calling this method if this Network
// was returned from a transaction, and the transaction was committed or rolled back.
func (n *Network) Update() *NetworkUpdateOne {
	return (&NetworkClient{config: n.config}).UpdateOne(n)
}

// Unwrap unwraps the Network entity that was returned from a transaction after it was closed,
// so that all future queries will be executed through the driver which created the transaction.
func (n *Network) Unwrap() *Network {
	tx, ok := n.config.driver.(*txDriver)
	if !ok {
		panic("ent: Network is not a transactional entity")
	}
	n.config.driver = tx.drv
	return n
}

// String implements the fmt.Stringer.
func (n *Network) String() string {
	var builder strings.Builder
	builder.WriteString("Network(")
	builder.WriteString(fmt.Sprintf("id=%v", n.ID))
	builder.WriteString(", hcl_id=")
	builder.WriteString(n.HclID)
	builder.WriteString(", name=")
	builder.WriteString(n.Name)
	builder.WriteString(", cidr=")
	builder.WriteString(n.Cidr)
	builder.WriteString(", vdi_visible=")
	builder.WriteString(fmt.Sprintf("%v", n.VdiVisible))
	builder.WriteString(", vars=")
	builder.WriteString(fmt.Sprintf("%v", n.Vars))
	builder.WriteString(", tags=")
	builder.WriteString(fmt.Sprintf("%v", n.Tags))
	builder.WriteByte(')')
	return builder.String()
}

// Networks is a parsable slice of Network.
type Networks []*Network

func (n Networks) config(cfg config) {
	for _i := range n {
		n[_i].config = cfg
	}
}
