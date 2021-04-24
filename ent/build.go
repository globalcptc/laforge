// Code generated by entc, DO NOT EDIT.

package ent

import (
	"fmt"
	"strings"

	"entgo.io/ent/dialect/sql"
	"github.com/gen0cide/laforge/ent/build"
	"github.com/gen0cide/laforge/ent/competition"
	"github.com/gen0cide/laforge/ent/environment"
	"github.com/gen0cide/laforge/ent/status"
)

// Build is the model entity for the Build schema.
type Build struct {
	config `json:"-"`
	// ID of the ent.
	ID int `json:"id,omitempty"`
	// Revision holds the value of the "revision" field.
	Revision int `json:"revision,omitempty"`
	// Edges holds the relations/edges for other nodes in the graph.
	// The values are being populated by the BuildQuery when eager-loading is set.
	Edges BuildEdges `json:"edges"`

	// Edges put into the main struct to be loaded via hcl
	// BuildToStatus holds the value of the BuildToStatus edge.
	HCLBuildToStatus *Status `json:"BuildToStatus,omitempty"`
	// BuildToEnvironment holds the value of the BuildToEnvironment edge.
	HCLBuildToEnvironment *Environment `json:"BuildToEnvironment,omitempty"`
	// BuildToCompetition holds the value of the BuildToCompetition edge.
	HCLBuildToCompetition *Competition `json:"BuildToCompetition,omitempty"`
	// BuildToProvisionedNetwork holds the value of the BuildToProvisionedNetwork edge.
	HCLBuildToProvisionedNetwork []*ProvisionedNetwork `json:"BuildToProvisionedNetwork,omitempty"`
	// BuildToTeam holds the value of the BuildToTeam edge.
	HCLBuildToTeam []*Team `json:"BuildToTeam,omitempty"`
	// BuildToPlan holds the value of the BuildToPlan edge.
	HCLBuildToPlan []*Plan `json:"BuildToPlan,omitempty"`
	//
	build_build_to_environment *int
	build_build_to_competition *int
}

// BuildEdges holds the relations/edges for other nodes in the graph.
type BuildEdges struct {
	// BuildToStatus holds the value of the BuildToStatus edge.
	BuildToStatus *Status `json:"BuildToStatus,omitempty"`
	// BuildToEnvironment holds the value of the BuildToEnvironment edge.
	BuildToEnvironment *Environment `json:"BuildToEnvironment,omitempty"`
	// BuildToCompetition holds the value of the BuildToCompetition edge.
	BuildToCompetition *Competition `json:"BuildToCompetition,omitempty"`
	// BuildToProvisionedNetwork holds the value of the BuildToProvisionedNetwork edge.
	BuildToProvisionedNetwork []*ProvisionedNetwork `json:"BuildToProvisionedNetwork,omitempty"`
	// BuildToTeam holds the value of the BuildToTeam edge.
	BuildToTeam []*Team `json:"BuildToTeam,omitempty"`
	// BuildToPlan holds the value of the BuildToPlan edge.
	BuildToPlan []*Plan `json:"BuildToPlan,omitempty"`
	// loadedTypes holds the information for reporting if a
	// type was loaded (or requested) in eager-loading or not.
	loadedTypes [6]bool
}

// BuildToStatusOrErr returns the BuildToStatus value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e BuildEdges) BuildToStatusOrErr() (*Status, error) {
	if e.loadedTypes[0] {
		if e.BuildToStatus == nil {
			// The edge BuildToStatus was loaded in eager-loading,
			// but was not found.
			return nil, &NotFoundError{label: status.Label}
		}
		return e.BuildToStatus, nil
	}
	return nil, &NotLoadedError{edge: "BuildToStatus"}
}

// BuildToEnvironmentOrErr returns the BuildToEnvironment value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e BuildEdges) BuildToEnvironmentOrErr() (*Environment, error) {
	if e.loadedTypes[1] {
		if e.BuildToEnvironment == nil {
			// The edge BuildToEnvironment was loaded in eager-loading,
			// but was not found.
			return nil, &NotFoundError{label: environment.Label}
		}
		return e.BuildToEnvironment, nil
	}
	return nil, &NotLoadedError{edge: "BuildToEnvironment"}
}

// BuildToCompetitionOrErr returns the BuildToCompetition value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e BuildEdges) BuildToCompetitionOrErr() (*Competition, error) {
	if e.loadedTypes[2] {
		if e.BuildToCompetition == nil {
			// The edge BuildToCompetition was loaded in eager-loading,
			// but was not found.
			return nil, &NotFoundError{label: competition.Label}
		}
		return e.BuildToCompetition, nil
	}
	return nil, &NotLoadedError{edge: "BuildToCompetition"}
}

// BuildToProvisionedNetworkOrErr returns the BuildToProvisionedNetwork value or an error if the edge
// was not loaded in eager-loading.
func (e BuildEdges) BuildToProvisionedNetworkOrErr() ([]*ProvisionedNetwork, error) {
	if e.loadedTypes[3] {
		return e.BuildToProvisionedNetwork, nil
	}
	return nil, &NotLoadedError{edge: "BuildToProvisionedNetwork"}
}

// BuildToTeamOrErr returns the BuildToTeam value or an error if the edge
// was not loaded in eager-loading.
func (e BuildEdges) BuildToTeamOrErr() ([]*Team, error) {
	if e.loadedTypes[4] {
		return e.BuildToTeam, nil
	}
	return nil, &NotLoadedError{edge: "BuildToTeam"}
}

// BuildToPlanOrErr returns the BuildToPlan value or an error if the edge
// was not loaded in eager-loading.
func (e BuildEdges) BuildToPlanOrErr() ([]*Plan, error) {
	if e.loadedTypes[5] {
		return e.BuildToPlan, nil
	}
	return nil, &NotLoadedError{edge: "BuildToPlan"}
}

// scanValues returns the types for scanning values from sql.Rows.
func (*Build) scanValues(columns []string) ([]interface{}, error) {
	values := make([]interface{}, len(columns))
	for i := range columns {
		switch columns[i] {
		case build.FieldID, build.FieldRevision:
			values[i] = &sql.NullInt64{}
		case build.ForeignKeys[0]: // build_build_to_environment
			values[i] = &sql.NullInt64{}
		case build.ForeignKeys[1]: // build_build_to_competition
			values[i] = &sql.NullInt64{}
		default:
			return nil, fmt.Errorf("unexpected column %q for type Build", columns[i])
		}
	}
	return values, nil
}

// assignValues assigns the values that were returned from sql.Rows (after scanning)
// to the Build fields.
func (b *Build) assignValues(columns []string, values []interface{}) error {
	if m, n := len(values), len(columns); m < n {
		return fmt.Errorf("mismatch number of scan values: %d != %d", m, n)
	}
	for i := range columns {
		switch columns[i] {
		case build.FieldID:
			value, ok := values[i].(*sql.NullInt64)
			if !ok {
				return fmt.Errorf("unexpected type %T for field id", value)
			}
			b.ID = int(value.Int64)
		case build.FieldRevision:
			if value, ok := values[i].(*sql.NullInt64); !ok {
				return fmt.Errorf("unexpected type %T for field revision", values[i])
			} else if value.Valid {
				b.Revision = int(value.Int64)
			}
		case build.ForeignKeys[0]:
			if value, ok := values[i].(*sql.NullInt64); !ok {
				return fmt.Errorf("unexpected type %T for edge-field build_build_to_environment", value)
			} else if value.Valid {
				b.build_build_to_environment = new(int)
				*b.build_build_to_environment = int(value.Int64)
			}
		case build.ForeignKeys[1]:
			if value, ok := values[i].(*sql.NullInt64); !ok {
				return fmt.Errorf("unexpected type %T for edge-field build_build_to_competition", value)
			} else if value.Valid {
				b.build_build_to_competition = new(int)
				*b.build_build_to_competition = int(value.Int64)
			}
		}
	}
	return nil
}

// QueryBuildToStatus queries the "BuildToStatus" edge of the Build entity.
func (b *Build) QueryBuildToStatus() *StatusQuery {
	return (&BuildClient{config: b.config}).QueryBuildToStatus(b)
}

// QueryBuildToEnvironment queries the "BuildToEnvironment" edge of the Build entity.
func (b *Build) QueryBuildToEnvironment() *EnvironmentQuery {
	return (&BuildClient{config: b.config}).QueryBuildToEnvironment(b)
}

// QueryBuildToCompetition queries the "BuildToCompetition" edge of the Build entity.
func (b *Build) QueryBuildToCompetition() *CompetitionQuery {
	return (&BuildClient{config: b.config}).QueryBuildToCompetition(b)
}

// QueryBuildToProvisionedNetwork queries the "BuildToProvisionedNetwork" edge of the Build entity.
func (b *Build) QueryBuildToProvisionedNetwork() *ProvisionedNetworkQuery {
	return (&BuildClient{config: b.config}).QueryBuildToProvisionedNetwork(b)
}

// QueryBuildToTeam queries the "BuildToTeam" edge of the Build entity.
func (b *Build) QueryBuildToTeam() *TeamQuery {
	return (&BuildClient{config: b.config}).QueryBuildToTeam(b)
}

// QueryBuildToPlan queries the "BuildToPlan" edge of the Build entity.
func (b *Build) QueryBuildToPlan() *PlanQuery {
	return (&BuildClient{config: b.config}).QueryBuildToPlan(b)
}

// Update returns a builder for updating this Build.
// Note that you need to call Build.Unwrap() before calling this method if this Build
// was returned from a transaction, and the transaction was committed or rolled back.
func (b *Build) Update() *BuildUpdateOne {
	return (&BuildClient{config: b.config}).UpdateOne(b)
}

// Unwrap unwraps the Build entity that was returned from a transaction after it was closed,
// so that all future queries will be executed through the driver which created the transaction.
func (b *Build) Unwrap() *Build {
	tx, ok := b.config.driver.(*txDriver)
	if !ok {
		panic("ent: Build is not a transactional entity")
	}
	b.config.driver = tx.drv
	return b
}

// String implements the fmt.Stringer.
func (b *Build) String() string {
	var builder strings.Builder
	builder.WriteString("Build(")
	builder.WriteString(fmt.Sprintf("id=%v", b.ID))
	builder.WriteString(", revision=")
	builder.WriteString(fmt.Sprintf("%v", b.Revision))
	builder.WriteByte(')')
	return builder.String()
}

// Builds is a parsable slice of Build.
type Builds []*Build

func (b Builds) config(cfg config) {
	for _i := range b {
		b[_i].config = cfg
	}
}
