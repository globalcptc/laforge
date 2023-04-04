// Code generated by ent, DO NOT EDIT.

package ent

import (
	"fmt"
	"strings"

	"entgo.io/ent/dialect/sql"
	"github.com/gen0cide/laforge/ent/build"
	"github.com/gen0cide/laforge/ent/plan"
	"github.com/gen0cide/laforge/ent/provisionedhost"
	"github.com/gen0cide/laforge/ent/provisionednetwork"
	"github.com/gen0cide/laforge/ent/provisioningscheduledstep"
	"github.com/gen0cide/laforge/ent/provisioningstep"
	"github.com/gen0cide/laforge/ent/status"
	"github.com/gen0cide/laforge/ent/team"
	"github.com/google/uuid"
)

// Plan is the model entity for the Plan schema.
type Plan struct {
	config `json:"-"`
	// ID of the ent.
	ID uuid.UUID `json:"id,omitempty"`
	// StepNumber holds the value of the "step_number" field.
	StepNumber int `json:"step_number,omitempty"`
	// Type holds the value of the "type" field.
	Type plan.Type `json:"type,omitempty"`
	// BuildID holds the value of the "build_id" field.
	BuildID string `json:"build_id,omitempty"`
	// Edges holds the relations/edges for other nodes in the graph.
	// The values are being populated by the PlanQuery when eager-loading is set.
	Edges PlanEdges `json:"edges"`

	// Edges put into the main struct to be loaded via hcl
	// PrevPlan holds the value of the PrevPlan edge.
	HCLPrevPlan []*Plan `json:"PrevPlan,omitempty"`
	// NextPlan holds the value of the NextPlan edge.
	HCLNextPlan []*Plan `json:"NextPlan,omitempty"`
	// PlanToBuild holds the value of the PlanToBuild edge.
	HCLPlanToBuild *Build `json:"PlanToBuild,omitempty"`
	// PlanToTeam holds the value of the PlanToTeam edge.
	HCLPlanToTeam *Team `json:"PlanToTeam,omitempty"`
	// PlanToProvisionedNetwork holds the value of the PlanToProvisionedNetwork edge.
	HCLPlanToProvisionedNetwork *ProvisionedNetwork `json:"PlanToProvisionedNetwork,omitempty"`
	// PlanToProvisionedHost holds the value of the PlanToProvisionedHost edge.
	HCLPlanToProvisionedHost *ProvisionedHost `json:"PlanToProvisionedHost,omitempty"`
	// PlanToProvisioningStep holds the value of the PlanToProvisioningStep edge.
	HCLPlanToProvisioningStep *ProvisioningStep `json:"PlanToProvisioningStep,omitempty"`
	// PlanToProvisioningScheduledStep holds the value of the PlanToProvisioningScheduledStep edge.
	HCLPlanToProvisioningScheduledStep *ProvisioningScheduledStep `json:"PlanToProvisioningScheduledStep,omitempty"`
	// PlanToStatus holds the value of the PlanToStatus edge.
	HCLPlanToStatus *Status `json:"PlanToStatus,omitempty"`
	// PlanToPlanDiffs holds the value of the PlanToPlanDiffs edge.
	HCLPlanToPlanDiffs []*PlanDiff `json:"PlanToPlanDiffs,omitempty"`
	//
	plan_plan_to_build *uuid.UUID
}

// PlanEdges holds the relations/edges for other nodes in the graph.
type PlanEdges struct {
	// PrevPlan holds the value of the PrevPlan edge.
	PrevPlan []*Plan `json:"PrevPlan,omitempty"`
	// NextPlan holds the value of the NextPlan edge.
	NextPlan []*Plan `json:"NextPlan,omitempty"`
	// PlanToBuild holds the value of the PlanToBuild edge.
	PlanToBuild *Build `json:"PlanToBuild,omitempty"`
	// PlanToTeam holds the value of the PlanToTeam edge.
	PlanToTeam *Team `json:"PlanToTeam,omitempty"`
	// PlanToProvisionedNetwork holds the value of the PlanToProvisionedNetwork edge.
	PlanToProvisionedNetwork *ProvisionedNetwork `json:"PlanToProvisionedNetwork,omitempty"`
	// PlanToProvisionedHost holds the value of the PlanToProvisionedHost edge.
	PlanToProvisionedHost *ProvisionedHost `json:"PlanToProvisionedHost,omitempty"`
	// PlanToProvisioningStep holds the value of the PlanToProvisioningStep edge.
	PlanToProvisioningStep *ProvisioningStep `json:"PlanToProvisioningStep,omitempty"`
	// PlanToProvisioningScheduledStep holds the value of the PlanToProvisioningScheduledStep edge.
	PlanToProvisioningScheduledStep *ProvisioningScheduledStep `json:"PlanToProvisioningScheduledStep,omitempty"`
	// PlanToStatus holds the value of the PlanToStatus edge.
	PlanToStatus *Status `json:"PlanToStatus,omitempty"`
	// PlanToPlanDiffs holds the value of the PlanToPlanDiffs edge.
	PlanToPlanDiffs []*PlanDiff `json:"PlanToPlanDiffs,omitempty"`
	// loadedTypes holds the information for reporting if a
	// type was loaded (or requested) in eager-loading or not.
	loadedTypes [10]bool
}

// PrevPlanOrErr returns the PrevPlan value or an error if the edge
// was not loaded in eager-loading.
func (e PlanEdges) PrevPlanOrErr() ([]*Plan, error) {
	if e.loadedTypes[0] {
		return e.PrevPlan, nil
	}
	return nil, &NotLoadedError{edge: "PrevPlan"}
}

// NextPlanOrErr returns the NextPlan value or an error if the edge
// was not loaded in eager-loading.
func (e PlanEdges) NextPlanOrErr() ([]*Plan, error) {
	if e.loadedTypes[1] {
		return e.NextPlan, nil
	}
	return nil, &NotLoadedError{edge: "NextPlan"}
}

// PlanToBuildOrErr returns the PlanToBuild value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e PlanEdges) PlanToBuildOrErr() (*Build, error) {
	if e.loadedTypes[2] {
		if e.PlanToBuild == nil {
			// The edge PlanToBuild was loaded in eager-loading,
			// but was not found.
			return nil, &NotFoundError{label: build.Label}
		}
		return e.PlanToBuild, nil
	}
	return nil, &NotLoadedError{edge: "PlanToBuild"}
}

// PlanToTeamOrErr returns the PlanToTeam value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e PlanEdges) PlanToTeamOrErr() (*Team, error) {
	if e.loadedTypes[3] {
		if e.PlanToTeam == nil {
			// The edge PlanToTeam was loaded in eager-loading,
			// but was not found.
			return nil, &NotFoundError{label: team.Label}
		}
		return e.PlanToTeam, nil
	}
	return nil, &NotLoadedError{edge: "PlanToTeam"}
}

// PlanToProvisionedNetworkOrErr returns the PlanToProvisionedNetwork value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e PlanEdges) PlanToProvisionedNetworkOrErr() (*ProvisionedNetwork, error) {
	if e.loadedTypes[4] {
		if e.PlanToProvisionedNetwork == nil {
			// The edge PlanToProvisionedNetwork was loaded in eager-loading,
			// but was not found.
			return nil, &NotFoundError{label: provisionednetwork.Label}
		}
		return e.PlanToProvisionedNetwork, nil
	}
	return nil, &NotLoadedError{edge: "PlanToProvisionedNetwork"}
}

// PlanToProvisionedHostOrErr returns the PlanToProvisionedHost value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e PlanEdges) PlanToProvisionedHostOrErr() (*ProvisionedHost, error) {
	if e.loadedTypes[5] {
		if e.PlanToProvisionedHost == nil {
			// The edge PlanToProvisionedHost was loaded in eager-loading,
			// but was not found.
			return nil, &NotFoundError{label: provisionedhost.Label}
		}
		return e.PlanToProvisionedHost, nil
	}
	return nil, &NotLoadedError{edge: "PlanToProvisionedHost"}
}

// PlanToProvisioningStepOrErr returns the PlanToProvisioningStep value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e PlanEdges) PlanToProvisioningStepOrErr() (*ProvisioningStep, error) {
	if e.loadedTypes[6] {
		if e.PlanToProvisioningStep == nil {
			// The edge PlanToProvisioningStep was loaded in eager-loading,
			// but was not found.
			return nil, &NotFoundError{label: provisioningstep.Label}
		}
		return e.PlanToProvisioningStep, nil
	}
	return nil, &NotLoadedError{edge: "PlanToProvisioningStep"}
}

// PlanToProvisioningScheduledStepOrErr returns the PlanToProvisioningScheduledStep value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e PlanEdges) PlanToProvisioningScheduledStepOrErr() (*ProvisioningScheduledStep, error) {
	if e.loadedTypes[7] {
		if e.PlanToProvisioningScheduledStep == nil {
			// The edge PlanToProvisioningScheduledStep was loaded in eager-loading,
			// but was not found.
			return nil, &NotFoundError{label: provisioningscheduledstep.Label}
		}
		return e.PlanToProvisioningScheduledStep, nil
	}
	return nil, &NotLoadedError{edge: "PlanToProvisioningScheduledStep"}
}

// PlanToStatusOrErr returns the PlanToStatus value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e PlanEdges) PlanToStatusOrErr() (*Status, error) {
	if e.loadedTypes[8] {
		if e.PlanToStatus == nil {
			// The edge PlanToStatus was loaded in eager-loading,
			// but was not found.
			return nil, &NotFoundError{label: status.Label}
		}
		return e.PlanToStatus, nil
	}
	return nil, &NotLoadedError{edge: "PlanToStatus"}
}

// PlanToPlanDiffsOrErr returns the PlanToPlanDiffs value or an error if the edge
// was not loaded in eager-loading.
func (e PlanEdges) PlanToPlanDiffsOrErr() ([]*PlanDiff, error) {
	if e.loadedTypes[9] {
		return e.PlanToPlanDiffs, nil
	}
	return nil, &NotLoadedError{edge: "PlanToPlanDiffs"}
}

// scanValues returns the types for scanning values from sql.Rows.
func (*Plan) scanValues(columns []string) ([]interface{}, error) {
	values := make([]interface{}, len(columns))
	for i := range columns {
		switch columns[i] {
		case plan.FieldStepNumber:
			values[i] = new(sql.NullInt64)
		case plan.FieldType, plan.FieldBuildID:
			values[i] = new(sql.NullString)
		case plan.FieldID:
			values[i] = new(uuid.UUID)
		case plan.ForeignKeys[0]: // plan_plan_to_build
			values[i] = &sql.NullScanner{S: new(uuid.UUID)}
		default:
			return nil, fmt.Errorf("unexpected column %q for type Plan", columns[i])
		}
	}
	return values, nil
}

// assignValues assigns the values that were returned from sql.Rows (after scanning)
// to the Plan fields.
func (pl *Plan) assignValues(columns []string, values []interface{}) error {
	if m, n := len(values), len(columns); m < n {
		return fmt.Errorf("mismatch number of scan values: %d != %d", m, n)
	}
	for i := range columns {
		switch columns[i] {
		case plan.FieldID:
			if value, ok := values[i].(*uuid.UUID); !ok {
				return fmt.Errorf("unexpected type %T for field id", values[i])
			} else if value != nil {
				pl.ID = *value
			}
		case plan.FieldStepNumber:
			if value, ok := values[i].(*sql.NullInt64); !ok {
				return fmt.Errorf("unexpected type %T for field step_number", values[i])
			} else if value.Valid {
				pl.StepNumber = int(value.Int64)
			}
		case plan.FieldType:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field type", values[i])
			} else if value.Valid {
				pl.Type = plan.Type(value.String)
			}
		case plan.FieldBuildID:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field build_id", values[i])
			} else if value.Valid {
				pl.BuildID = value.String
			}
		case plan.ForeignKeys[0]:
			if value, ok := values[i].(*sql.NullScanner); !ok {
				return fmt.Errorf("unexpected type %T for field plan_plan_to_build", values[i])
			} else if value.Valid {
				pl.plan_plan_to_build = new(uuid.UUID)
				*pl.plan_plan_to_build = *value.S.(*uuid.UUID)
			}
		}
	}
	return nil
}

// QueryPrevPlan queries the "PrevPlan" edge of the Plan entity.
func (pl *Plan) QueryPrevPlan() *PlanQuery {
	return (&PlanClient{config: pl.config}).QueryPrevPlan(pl)
}

// QueryNextPlan queries the "NextPlan" edge of the Plan entity.
func (pl *Plan) QueryNextPlan() *PlanQuery {
	return (&PlanClient{config: pl.config}).QueryNextPlan(pl)
}

// QueryPlanToBuild queries the "PlanToBuild" edge of the Plan entity.
func (pl *Plan) QueryPlanToBuild() *BuildQuery {
	return (&PlanClient{config: pl.config}).QueryPlanToBuild(pl)
}

// QueryPlanToTeam queries the "PlanToTeam" edge of the Plan entity.
func (pl *Plan) QueryPlanToTeam() *TeamQuery {
	return (&PlanClient{config: pl.config}).QueryPlanToTeam(pl)
}

// QueryPlanToProvisionedNetwork queries the "PlanToProvisionedNetwork" edge of the Plan entity.
func (pl *Plan) QueryPlanToProvisionedNetwork() *ProvisionedNetworkQuery {
	return (&PlanClient{config: pl.config}).QueryPlanToProvisionedNetwork(pl)
}

// QueryPlanToProvisionedHost queries the "PlanToProvisionedHost" edge of the Plan entity.
func (pl *Plan) QueryPlanToProvisionedHost() *ProvisionedHostQuery {
	return (&PlanClient{config: pl.config}).QueryPlanToProvisionedHost(pl)
}

// QueryPlanToProvisioningStep queries the "PlanToProvisioningStep" edge of the Plan entity.
func (pl *Plan) QueryPlanToProvisioningStep() *ProvisioningStepQuery {
	return (&PlanClient{config: pl.config}).QueryPlanToProvisioningStep(pl)
}

// QueryPlanToProvisioningScheduledStep queries the "PlanToProvisioningScheduledStep" edge of the Plan entity.
func (pl *Plan) QueryPlanToProvisioningScheduledStep() *ProvisioningScheduledStepQuery {
	return (&PlanClient{config: pl.config}).QueryPlanToProvisioningScheduledStep(pl)
}

// QueryPlanToStatus queries the "PlanToStatus" edge of the Plan entity.
func (pl *Plan) QueryPlanToStatus() *StatusQuery {
	return (&PlanClient{config: pl.config}).QueryPlanToStatus(pl)
}

// QueryPlanToPlanDiffs queries the "PlanToPlanDiffs" edge of the Plan entity.
func (pl *Plan) QueryPlanToPlanDiffs() *PlanDiffQuery {
	return (&PlanClient{config: pl.config}).QueryPlanToPlanDiffs(pl)
}

// Update returns a builder for updating this Plan.
// Note that you need to call Plan.Unwrap() before calling this method if this Plan
// was returned from a transaction, and the transaction was committed or rolled back.
func (pl *Plan) Update() *PlanUpdateOne {
	return (&PlanClient{config: pl.config}).UpdateOne(pl)
}

// Unwrap unwraps the Plan entity that was returned from a transaction after it was closed,
// so that all future queries will be executed through the driver which created the transaction.
func (pl *Plan) Unwrap() *Plan {
	tx, ok := pl.config.driver.(*txDriver)
	if !ok {
		panic("ent: Plan is not a transactional entity")
	}
	pl.config.driver = tx.drv
	return pl
}

// String implements the fmt.Stringer.
func (pl *Plan) String() string {
	var builder strings.Builder
	builder.WriteString("Plan(")
	builder.WriteString(fmt.Sprintf("id=%v", pl.ID))
	builder.WriteString(", step_number=")
	builder.WriteString(fmt.Sprintf("%v", pl.StepNumber))
	builder.WriteString(", type=")
	builder.WriteString(fmt.Sprintf("%v", pl.Type))
	builder.WriteString(", build_id=")
	builder.WriteString(pl.BuildID)
	builder.WriteByte(')')
	return builder.String()
}

// Plans is a parsable slice of Plan.
type Plans []*Plan

func (pl Plans) config(cfg config) {
	for _i := range pl {
		pl[_i].config = cfg
	}
}
