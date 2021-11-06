// Code generated by entc, DO NOT EDIT.

package ent

import (
	"fmt"
	"strings"

	"entgo.io/ent/dialect/sql"
	"github.com/gen0cide/laforge/ent/build"
	"github.com/gen0cide/laforge/ent/network"
	"github.com/gen0cide/laforge/ent/plan"
	"github.com/gen0cide/laforge/ent/provisionednetwork"
	"github.com/gen0cide/laforge/ent/status"
	"github.com/gen0cide/laforge/ent/team"
	"github.com/google/uuid"
)

// ProvisionedNetwork is the model entity for the ProvisionedNetwork schema.
type ProvisionedNetwork struct {
	config `json:"-"`
	// ID of the ent.
	ID uuid.UUID `json:"id,omitempty"`
	// Name holds the value of the "name" field.
	Name string `json:"name,omitempty"`
	// Cidr holds the value of the "cidr" field.
	Cidr string `json:"cidr,omitempty"`
	// Edges holds the relations/edges for other nodes in the graph.
	// The values are being populated by the ProvisionedNetworkQuery when eager-loading is set.
	Edges ProvisionedNetworkEdges `json:"edges"`

	// Edges put into the main struct to be loaded via hcl
	// ProvisionedNetworkToStatus holds the value of the ProvisionedNetworkToStatus edge.
	HCLProvisionedNetworkToStatus *Status `json:"ProvisionedNetworkToStatus,omitempty"`
	// ProvisionedNetworkToNetwork holds the value of the ProvisionedNetworkToNetwork edge.
	HCLProvisionedNetworkToNetwork *Network `json:"ProvisionedNetworkToNetwork,omitempty"`
	// ProvisionedNetworkToBuild holds the value of the ProvisionedNetworkToBuild edge.
	HCLProvisionedNetworkToBuild *Build `json:"ProvisionedNetworkToBuild,omitempty"`
	// ProvisionedNetworkToTeam holds the value of the ProvisionedNetworkToTeam edge.
	HCLProvisionedNetworkToTeam *Team `json:"ProvisionedNetworkToTeam,omitempty"`
	// ProvisionedNetworkToProvisionedHost holds the value of the ProvisionedNetworkToProvisionedHost edge.
	HCLProvisionedNetworkToProvisionedHost []*ProvisionedHost `json:"ProvisionedNetworkToProvisionedHost,omitempty"`
	// ProvisionedNetworkToPlan holds the value of the ProvisionedNetworkToPlan edge.
	HCLProvisionedNetworkToPlan *Plan `json:"ProvisionedNetworkToPlan,omitempty"`
	//
	plan_plan_to_provisioned_network                   *uuid.UUID
	provisioned_network_provisioned_network_to_network *uuid.UUID
	provisioned_network_provisioned_network_to_build   *uuid.UUID
	provisioned_network_provisioned_network_to_team    *uuid.UUID
}

// ProvisionedNetworkEdges holds the relations/edges for other nodes in the graph.
type ProvisionedNetworkEdges struct {
	// ProvisionedNetworkToStatus holds the value of the ProvisionedNetworkToStatus edge.
	ProvisionedNetworkToStatus *Status `json:"ProvisionedNetworkToStatus,omitempty"`
	// ProvisionedNetworkToNetwork holds the value of the ProvisionedNetworkToNetwork edge.
	ProvisionedNetworkToNetwork *Network `json:"ProvisionedNetworkToNetwork,omitempty"`
	// ProvisionedNetworkToBuild holds the value of the ProvisionedNetworkToBuild edge.
	ProvisionedNetworkToBuild *Build `json:"ProvisionedNetworkToBuild,omitempty"`
	// ProvisionedNetworkToTeam holds the value of the ProvisionedNetworkToTeam edge.
	ProvisionedNetworkToTeam *Team `json:"ProvisionedNetworkToTeam,omitempty"`
	// ProvisionedNetworkToProvisionedHost holds the value of the ProvisionedNetworkToProvisionedHost edge.
	ProvisionedNetworkToProvisionedHost []*ProvisionedHost `json:"ProvisionedNetworkToProvisionedHost,omitempty"`
	// ProvisionedNetworkToPlan holds the value of the ProvisionedNetworkToPlan edge.
	ProvisionedNetworkToPlan *Plan `json:"ProvisionedNetworkToPlan,omitempty"`
	// loadedTypes holds the information for reporting if a
	// type was loaded (or requested) in eager-loading or not.
	loadedTypes [6]bool
}

// ProvisionedNetworkToStatusOrErr returns the ProvisionedNetworkToStatus value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e ProvisionedNetworkEdges) ProvisionedNetworkToStatusOrErr() (*Status, error) {
	if e.loadedTypes[0] {
		if e.ProvisionedNetworkToStatus == nil {
			// The edge ProvisionedNetworkToStatus was loaded in eager-loading,
			// but was not found.
			return nil, &NotFoundError{label: status.Label}
		}
		return e.ProvisionedNetworkToStatus, nil
	}
	return nil, &NotLoadedError{edge: "ProvisionedNetworkToStatus"}
}

// ProvisionedNetworkToNetworkOrErr returns the ProvisionedNetworkToNetwork value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e ProvisionedNetworkEdges) ProvisionedNetworkToNetworkOrErr() (*Network, error) {
	if e.loadedTypes[1] {
		if e.ProvisionedNetworkToNetwork == nil {
			// The edge ProvisionedNetworkToNetwork was loaded in eager-loading,
			// but was not found.
			return nil, &NotFoundError{label: network.Label}
		}
		return e.ProvisionedNetworkToNetwork, nil
	}
	return nil, &NotLoadedError{edge: "ProvisionedNetworkToNetwork"}
}

// ProvisionedNetworkToBuildOrErr returns the ProvisionedNetworkToBuild value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e ProvisionedNetworkEdges) ProvisionedNetworkToBuildOrErr() (*Build, error) {
	if e.loadedTypes[2] {
		if e.ProvisionedNetworkToBuild == nil {
			// The edge ProvisionedNetworkToBuild was loaded in eager-loading,
			// but was not found.
			return nil, &NotFoundError{label: build.Label}
		}
		return e.ProvisionedNetworkToBuild, nil
	}
	return nil, &NotLoadedError{edge: "ProvisionedNetworkToBuild"}
}

// ProvisionedNetworkToTeamOrErr returns the ProvisionedNetworkToTeam value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e ProvisionedNetworkEdges) ProvisionedNetworkToTeamOrErr() (*Team, error) {
	if e.loadedTypes[3] {
		if e.ProvisionedNetworkToTeam == nil {
			// The edge ProvisionedNetworkToTeam was loaded in eager-loading,
			// but was not found.
			return nil, &NotFoundError{label: team.Label}
		}
		return e.ProvisionedNetworkToTeam, nil
	}
	return nil, &NotLoadedError{edge: "ProvisionedNetworkToTeam"}
}

// ProvisionedNetworkToProvisionedHostOrErr returns the ProvisionedNetworkToProvisionedHost value or an error if the edge
// was not loaded in eager-loading.
func (e ProvisionedNetworkEdges) ProvisionedNetworkToProvisionedHostOrErr() ([]*ProvisionedHost, error) {
	if e.loadedTypes[4] {
		return e.ProvisionedNetworkToProvisionedHost, nil
	}
	return nil, &NotLoadedError{edge: "ProvisionedNetworkToProvisionedHost"}
}

// ProvisionedNetworkToPlanOrErr returns the ProvisionedNetworkToPlan value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e ProvisionedNetworkEdges) ProvisionedNetworkToPlanOrErr() (*Plan, error) {
	if e.loadedTypes[5] {
		if e.ProvisionedNetworkToPlan == nil {
			// The edge ProvisionedNetworkToPlan was loaded in eager-loading,
			// but was not found.
			return nil, &NotFoundError{label: plan.Label}
		}
		return e.ProvisionedNetworkToPlan, nil
	}
	return nil, &NotLoadedError{edge: "ProvisionedNetworkToPlan"}
}

// scanValues returns the types for scanning values from sql.Rows.
func (*ProvisionedNetwork) scanValues(columns []string) ([]interface{}, error) {
	values := make([]interface{}, len(columns))
	for i := range columns {
		switch columns[i] {
		case provisionednetwork.FieldName, provisionednetwork.FieldCidr:
			values[i] = new(sql.NullString)
		case provisionednetwork.FieldID:
			values[i] = new(uuid.UUID)
		case provisionednetwork.ForeignKeys[0]: // plan_plan_to_provisioned_network
			values[i] = &sql.NullScanner{S: new(uuid.UUID)}
		case provisionednetwork.ForeignKeys[1]: // provisioned_network_provisioned_network_to_network
			values[i] = &sql.NullScanner{S: new(uuid.UUID)}
		case provisionednetwork.ForeignKeys[2]: // provisioned_network_provisioned_network_to_build
			values[i] = &sql.NullScanner{S: new(uuid.UUID)}
		case provisionednetwork.ForeignKeys[3]: // provisioned_network_provisioned_network_to_team
			values[i] = &sql.NullScanner{S: new(uuid.UUID)}
		default:
			return nil, fmt.Errorf("unexpected column %q for type ProvisionedNetwork", columns[i])
		}
	}
	return values, nil
}

// assignValues assigns the values that were returned from sql.Rows (after scanning)
// to the ProvisionedNetwork fields.
func (pn *ProvisionedNetwork) assignValues(columns []string, values []interface{}) error {
	if m, n := len(values), len(columns); m < n {
		return fmt.Errorf("mismatch number of scan values: %d != %d", m, n)
	}
	for i := range columns {
		switch columns[i] {
		case provisionednetwork.FieldID:
			if value, ok := values[i].(*uuid.UUID); !ok {
				return fmt.Errorf("unexpected type %T for field id", values[i])
			} else if value != nil {
				pn.ID = *value
			}
		case provisionednetwork.FieldName:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field name", values[i])
			} else if value.Valid {
				pn.Name = value.String
			}
		case provisionednetwork.FieldCidr:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field cidr", values[i])
			} else if value.Valid {
				pn.Cidr = value.String
			}
		case provisionednetwork.ForeignKeys[0]:
			if value, ok := values[i].(*sql.NullScanner); !ok {
				return fmt.Errorf("unexpected type %T for field plan_plan_to_provisioned_network", values[i])
			} else if value.Valid {
				pn.plan_plan_to_provisioned_network = new(uuid.UUID)
				*pn.plan_plan_to_provisioned_network = *value.S.(*uuid.UUID)
			}
		case provisionednetwork.ForeignKeys[1]:
			if value, ok := values[i].(*sql.NullScanner); !ok {
				return fmt.Errorf("unexpected type %T for field provisioned_network_provisioned_network_to_network", values[i])
			} else if value.Valid {
				pn.provisioned_network_provisioned_network_to_network = new(uuid.UUID)
				*pn.provisioned_network_provisioned_network_to_network = *value.S.(*uuid.UUID)
			}
		case provisionednetwork.ForeignKeys[2]:
			if value, ok := values[i].(*sql.NullScanner); !ok {
				return fmt.Errorf("unexpected type %T for field provisioned_network_provisioned_network_to_build", values[i])
			} else if value.Valid {
				pn.provisioned_network_provisioned_network_to_build = new(uuid.UUID)
				*pn.provisioned_network_provisioned_network_to_build = *value.S.(*uuid.UUID)
			}
		case provisionednetwork.ForeignKeys[3]:
			if value, ok := values[i].(*sql.NullScanner); !ok {
				return fmt.Errorf("unexpected type %T for field provisioned_network_provisioned_network_to_team", values[i])
			} else if value.Valid {
				pn.provisioned_network_provisioned_network_to_team = new(uuid.UUID)
				*pn.provisioned_network_provisioned_network_to_team = *value.S.(*uuid.UUID)
			}
		}
	}
	return nil
}

// QueryProvisionedNetworkToStatus queries the "ProvisionedNetworkToStatus" edge of the ProvisionedNetwork entity.
func (pn *ProvisionedNetwork) QueryProvisionedNetworkToStatus() *StatusQuery {
	return (&ProvisionedNetworkClient{config: pn.config}).QueryProvisionedNetworkToStatus(pn)
}

// QueryProvisionedNetworkToNetwork queries the "ProvisionedNetworkToNetwork" edge of the ProvisionedNetwork entity.
func (pn *ProvisionedNetwork) QueryProvisionedNetworkToNetwork() *NetworkQuery {
	return (&ProvisionedNetworkClient{config: pn.config}).QueryProvisionedNetworkToNetwork(pn)
}

// QueryProvisionedNetworkToBuild queries the "ProvisionedNetworkToBuild" edge of the ProvisionedNetwork entity.
func (pn *ProvisionedNetwork) QueryProvisionedNetworkToBuild() *BuildQuery {
	return (&ProvisionedNetworkClient{config: pn.config}).QueryProvisionedNetworkToBuild(pn)
}

// QueryProvisionedNetworkToTeam queries the "ProvisionedNetworkToTeam" edge of the ProvisionedNetwork entity.
func (pn *ProvisionedNetwork) QueryProvisionedNetworkToTeam() *TeamQuery {
	return (&ProvisionedNetworkClient{config: pn.config}).QueryProvisionedNetworkToTeam(pn)
}

// QueryProvisionedNetworkToProvisionedHost queries the "ProvisionedNetworkToProvisionedHost" edge of the ProvisionedNetwork entity.
func (pn *ProvisionedNetwork) QueryProvisionedNetworkToProvisionedHost() *ProvisionedHostQuery {
	return (&ProvisionedNetworkClient{config: pn.config}).QueryProvisionedNetworkToProvisionedHost(pn)
}

// QueryProvisionedNetworkToPlan queries the "ProvisionedNetworkToPlan" edge of the ProvisionedNetwork entity.
func (pn *ProvisionedNetwork) QueryProvisionedNetworkToPlan() *PlanQuery {
	return (&ProvisionedNetworkClient{config: pn.config}).QueryProvisionedNetworkToPlan(pn)
}

// Update returns a builder for updating this ProvisionedNetwork.
// Note that you need to call ProvisionedNetwork.Unwrap() before calling this method if this ProvisionedNetwork
// was returned from a transaction, and the transaction was committed or rolled back.
func (pn *ProvisionedNetwork) Update() *ProvisionedNetworkUpdateOne {
	return (&ProvisionedNetworkClient{config: pn.config}).UpdateOne(pn)
}

// Unwrap unwraps the ProvisionedNetwork entity that was returned from a transaction after it was closed,
// so that all future queries will be executed through the driver which created the transaction.
func (pn *ProvisionedNetwork) Unwrap() *ProvisionedNetwork {
	tx, ok := pn.config.driver.(*txDriver)
	if !ok {
		panic("ent: ProvisionedNetwork is not a transactional entity")
	}
	pn.config.driver = tx.drv
	return pn
}

// String implements the fmt.Stringer.
func (pn *ProvisionedNetwork) String() string {
	var builder strings.Builder
	builder.WriteString("ProvisionedNetwork(")
	builder.WriteString(fmt.Sprintf("id=%v", pn.ID))
	builder.WriteString(", name=")
	builder.WriteString(pn.Name)
	builder.WriteString(", cidr=")
	builder.WriteString(pn.Cidr)
	builder.WriteByte(')')
	return builder.String()
}

// ProvisionedNetworks is a parsable slice of ProvisionedNetwork.
type ProvisionedNetworks []*ProvisionedNetwork

func (pn ProvisionedNetworks) config(cfg config) {
	for _i := range pn {
		pn[_i].config = cfg
	}
}
