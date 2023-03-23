// Code generated by entc, DO NOT EDIT.

package ent

import (
	"encoding/json"
	"fmt"
	"strings"

	"entgo.io/ent/dialect/sql"
	"github.com/gen0cide/laforge/ent/environment"
	"github.com/google/uuid"
)

// Environment is the model entity for the Environment schema.
type Environment struct {
	config ` json:"-"`
	// ID of the ent.
	ID uuid.UUID `json:"id,omitempty"`
	// HclID holds the value of the "hcl_id" field.
	HclID string `json:"hcl_id,omitempty" hcl:"id,label"`
	// CompetitionID holds the value of the "competition_id" field.
	CompetitionID string `json:"competition_id,omitempty" hcl:"competition_id,attr"`
	// Name holds the value of the "name" field.
	Name string `json:"name,omitempty" hcl:"name,attr"`
	// Description holds the value of the "description" field.
	Description string `json:"description,omitempty" hcl:"description,attr"`
	// Builder holds the value of the "builder" field.
	Builder string `json:"builder,omitempty" hcl:"builder,attr"`
	// TeamCount holds the value of the "team_count" field.
	TeamCount int `json:"team_count,omitempty" hcl:"team_count,attr"`
	// Revision holds the value of the "revision" field.
	Revision int `json:"revision,omitempty" hcl:"revision,optional"`
	// AdminCidrs holds the value of the "admin_cidrs" field.
	AdminCidrs []string `json:"admin_cidrs,omitempty" hcl:"admin_ranges,attr"`
	// ExposedVdiPorts holds the value of the "exposed_vdi_ports" field.
	ExposedVdiPorts []string `json:"exposed_vdi_ports,omitempty" hcl:"vdi_allowed_tcp_ports"`
	// Config holds the value of the "config" field.
	Config map[string]string `json:"config,omitempty" hcl:"config,optional"`
	// Tags holds the value of the "tags" field.
	Tags map[string]string `json:"tags,omitempty" hcl:"tags,optional"`
	// Edges holds the relations/edges for other nodes in the graph.
	// The values are being populated by the EnvironmentQuery when eager-loading is set.
	Edges EnvironmentEdges `json:"edges"`

	// Edges put into the main struct to be loaded via hcl
	// EnvironmentToUser holds the value of the EnvironmentToUser edge.
	HCLEnvironmentToUser []*User `json:"EnvironmentToUser,omitempty" hcl:"maintainer,block"`
	// EnvironmentToHost holds the value of the EnvironmentToHost edge.
	HCLEnvironmentToHost []*Host `json:"EnvironmentToHost,omitempty"`
	// EnvironmentToCompetition holds the value of the EnvironmentToCompetition edge.
	HCLEnvironmentToCompetition []*Competition `json:"EnvironmentToCompetition,omitempty"`
	// EnvironmentToIdentity holds the value of the EnvironmentToIdentity edge.
	HCLEnvironmentToIdentity []*Identity `json:"EnvironmentToIdentity,omitempty"`
	// EnvironmentToCommand holds the value of the EnvironmentToCommand edge.
	HCLEnvironmentToCommand []*Command `json:"EnvironmentToCommand,omitempty"`
	// EnvironmentToScript holds the value of the EnvironmentToScript edge.
	HCLEnvironmentToScript []*Script `json:"EnvironmentToScript,omitempty"`
	// EnvironmentToFileDownload holds the value of the EnvironmentToFileDownload edge.
	HCLEnvironmentToFileDownload []*FileDownload `json:"EnvironmentToFileDownload,omitempty"`
	// EnvironmentToFileDelete holds the value of the EnvironmentToFileDelete edge.
	HCLEnvironmentToFileDelete []*FileDelete `json:"EnvironmentToFileDelete,omitempty"`
	// EnvironmentToFileExtract holds the value of the EnvironmentToFileExtract edge.
	HCLEnvironmentToFileExtract []*FileExtract `json:"EnvironmentToFileExtract,omitempty"`
	// EnvironmentToIncludedNetwork holds the value of the EnvironmentToIncludedNetwork edge.
	HCLEnvironmentToIncludedNetwork []*IncludedNetwork `json:"EnvironmentToIncludedNetwork,omitempty" hcl:"included_network,block"`
	// EnvironmentToFinding holds the value of the EnvironmentToFinding edge.
	HCLEnvironmentToFinding []*Finding `json:"EnvironmentToFinding,omitempty"`
	// EnvironmentToDNSRecord holds the value of the EnvironmentToDNSRecord edge.
	HCLEnvironmentToDNSRecord []*DNSRecord `json:"EnvironmentToDNSRecord,omitempty"`
	// EnvironmentToDNS holds the value of the EnvironmentToDNS edge.
	HCLEnvironmentToDNS []*DNS `json:"EnvironmentToDNS,omitempty"`
	// EnvironmentToNetwork holds the value of the EnvironmentToNetwork edge.
	HCLEnvironmentToNetwork []*Network `json:"EnvironmentToNetwork,omitempty"`
	// EnvironmentToHostDependency holds the value of the EnvironmentToHostDependency edge.
	HCLEnvironmentToHostDependency []*HostDependency `json:"EnvironmentToHostDependency,omitempty"`
	// EnvironmentToAnsible holds the value of the EnvironmentToAnsible edge.
	HCLEnvironmentToAnsible []*Ansible `json:"EnvironmentToAnsible,omitempty"`
	// EnvironmentToBuild holds the value of the EnvironmentToBuild edge.
	HCLEnvironmentToBuild []*Build `json:"EnvironmentToBuild,omitempty"`
	// EnvironmentToRepository holds the value of the EnvironmentToRepository edge.
	HCLEnvironmentToRepository []*Repository `json:"EnvironmentToRepository,omitempty"`
	// EnvironmentToServerTask holds the value of the EnvironmentToServerTask edge.
	HCLEnvironmentToServerTask []*ServerTask `json:"EnvironmentToServerTask,omitempty"`
	// EnvironmentToValidation holds the value of the EnvironmentToValidation edge.
	HCLEnvironmentToValidation []*Validation `json:"EnvironmentToValidation,omitempty"`
	//

}

// EnvironmentEdges holds the relations/edges for other nodes in the graph.
type EnvironmentEdges struct {
	// EnvironmentToUser holds the value of the EnvironmentToUser edge.
	EnvironmentToUser []*User `json:"EnvironmentToUser,omitempty" hcl:"maintainer,block"`
	// EnvironmentToHost holds the value of the EnvironmentToHost edge.
	EnvironmentToHost []*Host `json:"EnvironmentToHost,omitempty"`
	// EnvironmentToCompetition holds the value of the EnvironmentToCompetition edge.
	EnvironmentToCompetition []*Competition `json:"EnvironmentToCompetition,omitempty"`
	// EnvironmentToIdentity holds the value of the EnvironmentToIdentity edge.
	EnvironmentToIdentity []*Identity `json:"EnvironmentToIdentity,omitempty"`
	// EnvironmentToCommand holds the value of the EnvironmentToCommand edge.
	EnvironmentToCommand []*Command `json:"EnvironmentToCommand,omitempty"`
	// EnvironmentToScript holds the value of the EnvironmentToScript edge.
	EnvironmentToScript []*Script `json:"EnvironmentToScript,omitempty"`
	// EnvironmentToFileDownload holds the value of the EnvironmentToFileDownload edge.
	EnvironmentToFileDownload []*FileDownload `json:"EnvironmentToFileDownload,omitempty"`
	// EnvironmentToFileDelete holds the value of the EnvironmentToFileDelete edge.
	EnvironmentToFileDelete []*FileDelete `json:"EnvironmentToFileDelete,omitempty"`
	// EnvironmentToFileExtract holds the value of the EnvironmentToFileExtract edge.
	EnvironmentToFileExtract []*FileExtract `json:"EnvironmentToFileExtract,omitempty"`
	// EnvironmentToIncludedNetwork holds the value of the EnvironmentToIncludedNetwork edge.
	EnvironmentToIncludedNetwork []*IncludedNetwork `json:"EnvironmentToIncludedNetwork,omitempty" hcl:"included_network,block"`
	// EnvironmentToFinding holds the value of the EnvironmentToFinding edge.
	EnvironmentToFinding []*Finding `json:"EnvironmentToFinding,omitempty"`
	// EnvironmentToDNSRecord holds the value of the EnvironmentToDNSRecord edge.
	EnvironmentToDNSRecord []*DNSRecord `json:"EnvironmentToDNSRecord,omitempty"`
	// EnvironmentToDNS holds the value of the EnvironmentToDNS edge.
	EnvironmentToDNS []*DNS `json:"EnvironmentToDNS,omitempty"`
	// EnvironmentToNetwork holds the value of the EnvironmentToNetwork edge.
	EnvironmentToNetwork []*Network `json:"EnvironmentToNetwork,omitempty"`
	// EnvironmentToHostDependency holds the value of the EnvironmentToHostDependency edge.
	EnvironmentToHostDependency []*HostDependency `json:"EnvironmentToHostDependency,omitempty"`
	// EnvironmentToAnsible holds the value of the EnvironmentToAnsible edge.
	EnvironmentToAnsible []*Ansible `json:"EnvironmentToAnsible,omitempty"`
	// EnvironmentToBuild holds the value of the EnvironmentToBuild edge.
	EnvironmentToBuild []*Build `json:"EnvironmentToBuild,omitempty"`
	// EnvironmentToRepository holds the value of the EnvironmentToRepository edge.
	EnvironmentToRepository []*Repository `json:"EnvironmentToRepository,omitempty"`
	// EnvironmentToServerTask holds the value of the EnvironmentToServerTask edge.
	EnvironmentToServerTask []*ServerTask `json:"EnvironmentToServerTask,omitempty"`
	// EnvironmentToValidation holds the value of the EnvironmentToValidation edge.
	EnvironmentToValidation []*Validation `json:"EnvironmentToValidation,omitempty"`
	// loadedTypes holds the information for reporting if a
	// type was loaded (or requested) in eager-loading or not.
	loadedTypes [20]bool
}

// EnvironmentToUserOrErr returns the EnvironmentToUser value or an error if the edge
// was not loaded in eager-loading.
func (e EnvironmentEdges) EnvironmentToUserOrErr() ([]*User, error) {
	if e.loadedTypes[0] {
		return e.EnvironmentToUser, nil
	}
	return nil, &NotLoadedError{edge: "EnvironmentToUser"}
}

// EnvironmentToHostOrErr returns the EnvironmentToHost value or an error if the edge
// was not loaded in eager-loading.
func (e EnvironmentEdges) EnvironmentToHostOrErr() ([]*Host, error) {
	if e.loadedTypes[1] {
		return e.EnvironmentToHost, nil
	}
	return nil, &NotLoadedError{edge: "EnvironmentToHost"}
}

// EnvironmentToCompetitionOrErr returns the EnvironmentToCompetition value or an error if the edge
// was not loaded in eager-loading.
func (e EnvironmentEdges) EnvironmentToCompetitionOrErr() ([]*Competition, error) {
	if e.loadedTypes[2] {
		return e.EnvironmentToCompetition, nil
	}
	return nil, &NotLoadedError{edge: "EnvironmentToCompetition"}
}

// EnvironmentToIdentityOrErr returns the EnvironmentToIdentity value or an error if the edge
// was not loaded in eager-loading.
func (e EnvironmentEdges) EnvironmentToIdentityOrErr() ([]*Identity, error) {
	if e.loadedTypes[3] {
		return e.EnvironmentToIdentity, nil
	}
	return nil, &NotLoadedError{edge: "EnvironmentToIdentity"}
}

// EnvironmentToCommandOrErr returns the EnvironmentToCommand value or an error if the edge
// was not loaded in eager-loading.
func (e EnvironmentEdges) EnvironmentToCommandOrErr() ([]*Command, error) {
	if e.loadedTypes[4] {
		return e.EnvironmentToCommand, nil
	}
	return nil, &NotLoadedError{edge: "EnvironmentToCommand"}
}

// EnvironmentToScriptOrErr returns the EnvironmentToScript value or an error if the edge
// was not loaded in eager-loading.
func (e EnvironmentEdges) EnvironmentToScriptOrErr() ([]*Script, error) {
	if e.loadedTypes[5] {
		return e.EnvironmentToScript, nil
	}
	return nil, &NotLoadedError{edge: "EnvironmentToScript"}
}

// EnvironmentToFileDownloadOrErr returns the EnvironmentToFileDownload value or an error if the edge
// was not loaded in eager-loading.
func (e EnvironmentEdges) EnvironmentToFileDownloadOrErr() ([]*FileDownload, error) {
	if e.loadedTypes[6] {
		return e.EnvironmentToFileDownload, nil
	}
	return nil, &NotLoadedError{edge: "EnvironmentToFileDownload"}
}

// EnvironmentToFileDeleteOrErr returns the EnvironmentToFileDelete value or an error if the edge
// was not loaded in eager-loading.
func (e EnvironmentEdges) EnvironmentToFileDeleteOrErr() ([]*FileDelete, error) {
	if e.loadedTypes[7] {
		return e.EnvironmentToFileDelete, nil
	}
	return nil, &NotLoadedError{edge: "EnvironmentToFileDelete"}
}

// EnvironmentToFileExtractOrErr returns the EnvironmentToFileExtract value or an error if the edge
// was not loaded in eager-loading.
func (e EnvironmentEdges) EnvironmentToFileExtractOrErr() ([]*FileExtract, error) {
	if e.loadedTypes[8] {
		return e.EnvironmentToFileExtract, nil
	}
	return nil, &NotLoadedError{edge: "EnvironmentToFileExtract"}
}

// EnvironmentToIncludedNetworkOrErr returns the EnvironmentToIncludedNetwork value or an error if the edge
// was not loaded in eager-loading.
func (e EnvironmentEdges) EnvironmentToIncludedNetworkOrErr() ([]*IncludedNetwork, error) {
	if e.loadedTypes[9] {
		return e.EnvironmentToIncludedNetwork, nil
	}
	return nil, &NotLoadedError{edge: "EnvironmentToIncludedNetwork"}
}

// EnvironmentToFindingOrErr returns the EnvironmentToFinding value or an error if the edge
// was not loaded in eager-loading.
func (e EnvironmentEdges) EnvironmentToFindingOrErr() ([]*Finding, error) {
	if e.loadedTypes[10] {
		return e.EnvironmentToFinding, nil
	}
	return nil, &NotLoadedError{edge: "EnvironmentToFinding"}
}

// EnvironmentToDNSRecordOrErr returns the EnvironmentToDNSRecord value or an error if the edge
// was not loaded in eager-loading.
func (e EnvironmentEdges) EnvironmentToDNSRecordOrErr() ([]*DNSRecord, error) {
	if e.loadedTypes[11] {
		return e.EnvironmentToDNSRecord, nil
	}
	return nil, &NotLoadedError{edge: "EnvironmentToDNSRecord"}
}

// EnvironmentToDNSOrErr returns the EnvironmentToDNS value or an error if the edge
// was not loaded in eager-loading.
func (e EnvironmentEdges) EnvironmentToDNSOrErr() ([]*DNS, error) {
	if e.loadedTypes[12] {
		return e.EnvironmentToDNS, nil
	}
	return nil, &NotLoadedError{edge: "EnvironmentToDNS"}
}

// EnvironmentToNetworkOrErr returns the EnvironmentToNetwork value or an error if the edge
// was not loaded in eager-loading.
func (e EnvironmentEdges) EnvironmentToNetworkOrErr() ([]*Network, error) {
	if e.loadedTypes[13] {
		return e.EnvironmentToNetwork, nil
	}
	return nil, &NotLoadedError{edge: "EnvironmentToNetwork"}
}

// EnvironmentToHostDependencyOrErr returns the EnvironmentToHostDependency value or an error if the edge
// was not loaded in eager-loading.
func (e EnvironmentEdges) EnvironmentToHostDependencyOrErr() ([]*HostDependency, error) {
	if e.loadedTypes[14] {
		return e.EnvironmentToHostDependency, nil
	}
	return nil, &NotLoadedError{edge: "EnvironmentToHostDependency"}
}

// EnvironmentToAnsibleOrErr returns the EnvironmentToAnsible value or an error if the edge
// was not loaded in eager-loading.
func (e EnvironmentEdges) EnvironmentToAnsibleOrErr() ([]*Ansible, error) {
	if e.loadedTypes[15] {
		return e.EnvironmentToAnsible, nil
	}
	return nil, &NotLoadedError{edge: "EnvironmentToAnsible"}
}

// EnvironmentToBuildOrErr returns the EnvironmentToBuild value or an error if the edge
// was not loaded in eager-loading.
func (e EnvironmentEdges) EnvironmentToBuildOrErr() ([]*Build, error) {
	if e.loadedTypes[16] {
		return e.EnvironmentToBuild, nil
	}
	return nil, &NotLoadedError{edge: "EnvironmentToBuild"}
}

// EnvironmentToRepositoryOrErr returns the EnvironmentToRepository value or an error if the edge
// was not loaded in eager-loading.
func (e EnvironmentEdges) EnvironmentToRepositoryOrErr() ([]*Repository, error) {
	if e.loadedTypes[17] {
		return e.EnvironmentToRepository, nil
	}
	return nil, &NotLoadedError{edge: "EnvironmentToRepository"}
}

// EnvironmentToServerTaskOrErr returns the EnvironmentToServerTask value or an error if the edge
// was not loaded in eager-loading.
func (e EnvironmentEdges) EnvironmentToServerTaskOrErr() ([]*ServerTask, error) {
	if e.loadedTypes[18] {
		return e.EnvironmentToServerTask, nil
	}
	return nil, &NotLoadedError{edge: "EnvironmentToServerTask"}
}

// EnvironmentToValidationOrErr returns the EnvironmentToValidation value or an error if the edge
// was not loaded in eager-loading.
func (e EnvironmentEdges) EnvironmentToValidationOrErr() ([]*Validation, error) {
	if e.loadedTypes[19] {
		return e.EnvironmentToValidation, nil
	}
	return nil, &NotLoadedError{edge: "EnvironmentToValidation"}
}

// scanValues returns the types for scanning values from sql.Rows.
func (*Environment) scanValues(columns []string) ([]interface{}, error) {
	values := make([]interface{}, len(columns))
	for i := range columns {
		switch columns[i] {
		case environment.FieldAdminCidrs, environment.FieldExposedVdiPorts, environment.FieldConfig, environment.FieldTags:
			values[i] = new([]byte)
		case environment.FieldTeamCount, environment.FieldRevision:
			values[i] = new(sql.NullInt64)
		case environment.FieldHclID, environment.FieldCompetitionID, environment.FieldName, environment.FieldDescription, environment.FieldBuilder:
			values[i] = new(sql.NullString)
		case environment.FieldID:
			values[i] = new(uuid.UUID)
		default:
			return nil, fmt.Errorf("unexpected column %q for type Environment", columns[i])
		}
	}
	return values, nil
}

// assignValues assigns the values that were returned from sql.Rows (after scanning)
// to the Environment fields.
func (e *Environment) assignValues(columns []string, values []interface{}) error {
	if m, n := len(values), len(columns); m < n {
		return fmt.Errorf("mismatch number of scan values: %d != %d", m, n)
	}
	for i := range columns {
		switch columns[i] {
		case environment.FieldID:
			if value, ok := values[i].(*uuid.UUID); !ok {
				return fmt.Errorf("unexpected type %T for field id", values[i])
			} else if value != nil {
				e.ID = *value
			}
		case environment.FieldHclID:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field hcl_id", values[i])
			} else if value.Valid {
				e.HclID = value.String
			}
		case environment.FieldCompetitionID:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field competition_id", values[i])
			} else if value.Valid {
				e.CompetitionID = value.String
			}
		case environment.FieldName:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field name", values[i])
			} else if value.Valid {
				e.Name = value.String
			}
		case environment.FieldDescription:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field description", values[i])
			} else if value.Valid {
				e.Description = value.String
			}
		case environment.FieldBuilder:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field builder", values[i])
			} else if value.Valid {
				e.Builder = value.String
			}
		case environment.FieldTeamCount:
			if value, ok := values[i].(*sql.NullInt64); !ok {
				return fmt.Errorf("unexpected type %T for field team_count", values[i])
			} else if value.Valid {
				e.TeamCount = int(value.Int64)
			}
		case environment.FieldRevision:
			if value, ok := values[i].(*sql.NullInt64); !ok {
				return fmt.Errorf("unexpected type %T for field revision", values[i])
			} else if value.Valid {
				e.Revision = int(value.Int64)
			}
		case environment.FieldAdminCidrs:
			if value, ok := values[i].(*[]byte); !ok {
				return fmt.Errorf("unexpected type %T for field admin_cidrs", values[i])
			} else if value != nil && len(*value) > 0 {
				if err := json.Unmarshal(*value, &e.AdminCidrs); err != nil {
					return fmt.Errorf("unmarshal field admin_cidrs: %w", err)
				}
			}
		case environment.FieldExposedVdiPorts:
			if value, ok := values[i].(*[]byte); !ok {
				return fmt.Errorf("unexpected type %T for field exposed_vdi_ports", values[i])
			} else if value != nil && len(*value) > 0 {
				if err := json.Unmarshal(*value, &e.ExposedVdiPorts); err != nil {
					return fmt.Errorf("unmarshal field exposed_vdi_ports: %w", err)
				}
			}
		case environment.FieldConfig:
			if value, ok := values[i].(*[]byte); !ok {
				return fmt.Errorf("unexpected type %T for field config", values[i])
			} else if value != nil && len(*value) > 0 {
				if err := json.Unmarshal(*value, &e.Config); err != nil {
					return fmt.Errorf("unmarshal field config: %w", err)
				}
			}
		case environment.FieldTags:
			if value, ok := values[i].(*[]byte); !ok {
				return fmt.Errorf("unexpected type %T for field tags", values[i])
			} else if value != nil && len(*value) > 0 {
				if err := json.Unmarshal(*value, &e.Tags); err != nil {
					return fmt.Errorf("unmarshal field tags: %w", err)
				}
			}
		}
	}
	return nil
}

// QueryEnvironmentToUser queries the "EnvironmentToUser" edge of the Environment entity.
func (e *Environment) QueryEnvironmentToUser() *UserQuery {
	return (&EnvironmentClient{config: e.config}).QueryEnvironmentToUser(e)
}

// QueryEnvironmentToHost queries the "EnvironmentToHost" edge of the Environment entity.
func (e *Environment) QueryEnvironmentToHost() *HostQuery {
	return (&EnvironmentClient{config: e.config}).QueryEnvironmentToHost(e)
}

// QueryEnvironmentToCompetition queries the "EnvironmentToCompetition" edge of the Environment entity.
func (e *Environment) QueryEnvironmentToCompetition() *CompetitionQuery {
	return (&EnvironmentClient{config: e.config}).QueryEnvironmentToCompetition(e)
}

// QueryEnvironmentToIdentity queries the "EnvironmentToIdentity" edge of the Environment entity.
func (e *Environment) QueryEnvironmentToIdentity() *IdentityQuery {
	return (&EnvironmentClient{config: e.config}).QueryEnvironmentToIdentity(e)
}

// QueryEnvironmentToCommand queries the "EnvironmentToCommand" edge of the Environment entity.
func (e *Environment) QueryEnvironmentToCommand() *CommandQuery {
	return (&EnvironmentClient{config: e.config}).QueryEnvironmentToCommand(e)
}

// QueryEnvironmentToScript queries the "EnvironmentToScript" edge of the Environment entity.
func (e *Environment) QueryEnvironmentToScript() *ScriptQuery {
	return (&EnvironmentClient{config: e.config}).QueryEnvironmentToScript(e)
}

// QueryEnvironmentToFileDownload queries the "EnvironmentToFileDownload" edge of the Environment entity.
func (e *Environment) QueryEnvironmentToFileDownload() *FileDownloadQuery {
	return (&EnvironmentClient{config: e.config}).QueryEnvironmentToFileDownload(e)
}

// QueryEnvironmentToFileDelete queries the "EnvironmentToFileDelete" edge of the Environment entity.
func (e *Environment) QueryEnvironmentToFileDelete() *FileDeleteQuery {
	return (&EnvironmentClient{config: e.config}).QueryEnvironmentToFileDelete(e)
}

// QueryEnvironmentToFileExtract queries the "EnvironmentToFileExtract" edge of the Environment entity.
func (e *Environment) QueryEnvironmentToFileExtract() *FileExtractQuery {
	return (&EnvironmentClient{config: e.config}).QueryEnvironmentToFileExtract(e)
}

// QueryEnvironmentToIncludedNetwork queries the "EnvironmentToIncludedNetwork" edge of the Environment entity.
func (e *Environment) QueryEnvironmentToIncludedNetwork() *IncludedNetworkQuery {
	return (&EnvironmentClient{config: e.config}).QueryEnvironmentToIncludedNetwork(e)
}

// QueryEnvironmentToFinding queries the "EnvironmentToFinding" edge of the Environment entity.
func (e *Environment) QueryEnvironmentToFinding() *FindingQuery {
	return (&EnvironmentClient{config: e.config}).QueryEnvironmentToFinding(e)
}

// QueryEnvironmentToDNSRecord queries the "EnvironmentToDNSRecord" edge of the Environment entity.
func (e *Environment) QueryEnvironmentToDNSRecord() *DNSRecordQuery {
	return (&EnvironmentClient{config: e.config}).QueryEnvironmentToDNSRecord(e)
}

// QueryEnvironmentToDNS queries the "EnvironmentToDNS" edge of the Environment entity.
func (e *Environment) QueryEnvironmentToDNS() *DNSQuery {
	return (&EnvironmentClient{config: e.config}).QueryEnvironmentToDNS(e)
}

// QueryEnvironmentToNetwork queries the "EnvironmentToNetwork" edge of the Environment entity.
func (e *Environment) QueryEnvironmentToNetwork() *NetworkQuery {
	return (&EnvironmentClient{config: e.config}).QueryEnvironmentToNetwork(e)
}

// QueryEnvironmentToHostDependency queries the "EnvironmentToHostDependency" edge of the Environment entity.
func (e *Environment) QueryEnvironmentToHostDependency() *HostDependencyQuery {
	return (&EnvironmentClient{config: e.config}).QueryEnvironmentToHostDependency(e)
}

// QueryEnvironmentToAnsible queries the "EnvironmentToAnsible" edge of the Environment entity.
func (e *Environment) QueryEnvironmentToAnsible() *AnsibleQuery {
	return (&EnvironmentClient{config: e.config}).QueryEnvironmentToAnsible(e)
}

// QueryEnvironmentToBuild queries the "EnvironmentToBuild" edge of the Environment entity.
func (e *Environment) QueryEnvironmentToBuild() *BuildQuery {
	return (&EnvironmentClient{config: e.config}).QueryEnvironmentToBuild(e)
}

// QueryEnvironmentToRepository queries the "EnvironmentToRepository" edge of the Environment entity.
func (e *Environment) QueryEnvironmentToRepository() *RepositoryQuery {
	return (&EnvironmentClient{config: e.config}).QueryEnvironmentToRepository(e)
}

// QueryEnvironmentToServerTask queries the "EnvironmentToServerTask" edge of the Environment entity.
func (e *Environment) QueryEnvironmentToServerTask() *ServerTaskQuery {
	return (&EnvironmentClient{config: e.config}).QueryEnvironmentToServerTask(e)
}

// QueryEnvironmentToValidation queries the "EnvironmentToValidation" edge of the Environment entity.
func (e *Environment) QueryEnvironmentToValidation() *ValidationQuery {
	return (&EnvironmentClient{config: e.config}).QueryEnvironmentToValidation(e)
}

// Update returns a builder for updating this Environment.
// Note that you need to call Environment.Unwrap() before calling this method if this Environment
// was returned from a transaction, and the transaction was committed or rolled back.
func (e *Environment) Update() *EnvironmentUpdateOne {
	return (&EnvironmentClient{config: e.config}).UpdateOne(e)
}

// Unwrap unwraps the Environment entity that was returned from a transaction after it was closed,
// so that all future queries will be executed through the driver which created the transaction.
func (e *Environment) Unwrap() *Environment {
	tx, ok := e.config.driver.(*txDriver)
	if !ok {
		panic("ent: Environment is not a transactional entity")
	}
	e.config.driver = tx.drv
	return e
}

// String implements the fmt.Stringer.
func (e *Environment) String() string {
	var builder strings.Builder
	builder.WriteString("Environment(")
	builder.WriteString(fmt.Sprintf("id=%v", e.ID))
	builder.WriteString(", hcl_id=")
	builder.WriteString(e.HclID)
	builder.WriteString(", competition_id=")
	builder.WriteString(e.CompetitionID)
	builder.WriteString(", name=")
	builder.WriteString(e.Name)
	builder.WriteString(", description=")
	builder.WriteString(e.Description)
	builder.WriteString(", builder=")
	builder.WriteString(e.Builder)
	builder.WriteString(", team_count=")
	builder.WriteString(fmt.Sprintf("%v", e.TeamCount))
	builder.WriteString(", revision=")
	builder.WriteString(fmt.Sprintf("%v", e.Revision))
	builder.WriteString(", admin_cidrs=")
	builder.WriteString(fmt.Sprintf("%v", e.AdminCidrs))
	builder.WriteString(", exposed_vdi_ports=")
	builder.WriteString(fmt.Sprintf("%v", e.ExposedVdiPorts))
	builder.WriteString(", config=")
	builder.WriteString(fmt.Sprintf("%v", e.Config))
	builder.WriteString(", tags=")
	builder.WriteString(fmt.Sprintf("%v", e.Tags))
	builder.WriteByte(')')
	return builder.String()
}

// Environments is a parsable slice of Environment.
type Environments []*Environment

func (e Environments) config(cfg config) {
	for _i := range e {
		e[_i].config = cfg
	}
}
