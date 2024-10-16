// Code generated by ent, DO NOT EDIT.

package ent

import (
	"fmt"
	"strings"
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/sql"
	"github.com/gen0cide/laforge/ent/ansible"
	"github.com/gen0cide/laforge/ent/command"
	"github.com/gen0cide/laforge/ent/dnsrecord"
	"github.com/gen0cide/laforge/ent/filedelete"
	"github.com/gen0cide/laforge/ent/filedownload"
	"github.com/gen0cide/laforge/ent/fileextract"
	"github.com/gen0cide/laforge/ent/ginfilemiddleware"
	"github.com/gen0cide/laforge/ent/plan"
	"github.com/gen0cide/laforge/ent/provisionedhost"
	"github.com/gen0cide/laforge/ent/provisioningscheduledstep"
	"github.com/gen0cide/laforge/ent/scheduledstep"
	"github.com/gen0cide/laforge/ent/script"
	"github.com/gen0cide/laforge/ent/status"
	"github.com/google/uuid"
)

// ProvisioningScheduledStep is the model entity for the ProvisioningScheduledStep schema.
type ProvisioningScheduledStep struct {
	config `json:"-"`
	// ID of the ent.
	ID uuid.UUID `json:"id,omitempty"`
	// Type holds the value of the "type" field.
	Type provisioningscheduledstep.Type `json:"type,omitempty"`
	// RunTime holds the value of the "run_time" field.
	RunTime time.Time `json:"run_time,omitempty"`
	// Edges holds the relations/edges for other nodes in the graph.
	// The values are being populated by the ProvisioningScheduledStepQuery when eager-loading is set.
	Edges ProvisioningScheduledStepEdges `json:"edges"`

	// vvvvvvvvvvvv CUSTOM vvvvvvvvvvvv
	// Edges put into the main struct to be loaded via hcl
	// Status holds the value of the Status edge.
	HCLStatus *Status `json:"Status,omitempty"`
	// ScheduledStep holds the value of the ScheduledStep edge.
	HCLScheduledStep *ScheduledStep `json:"ScheduledStep,omitempty"`
	// ProvisionedHost holds the value of the ProvisionedHost edge.
	HCLProvisionedHost *ProvisionedHost `json:"ProvisionedHost,omitempty"`
	// Script holds the value of the Script edge.
	HCLScript *Script `json:"Script,omitempty"`
	// Command holds the value of the Command edge.
	HCLCommand *Command `json:"Command,omitempty"`
	// DNSRecord holds the value of the DNSRecord edge.
	HCLDNSRecord *DNSRecord `json:"DNSRecord,omitempty"`
	// FileDelete holds the value of the FileDelete edge.
	HCLFileDelete *FileDelete `json:"FileDelete,omitempty"`
	// FileDownload holds the value of the FileDownload edge.
	HCLFileDownload *FileDownload `json:"FileDownload,omitempty"`
	// FileExtract holds the value of the FileExtract edge.
	HCLFileExtract *FileExtract `json:"FileExtract,omitempty"`
	// Ansible holds the value of the Ansible edge.
	HCLAnsible *Ansible `json:"Ansible,omitempty"`
	// AgentTasks holds the value of the AgentTasks edge.
	HCLAgentTasks []*AgentTask `json:"AgentTasks,omitempty"`
	// Plan holds the value of the Plan edge.
	HCLPlan *Plan `json:"Plan,omitempty"`
	// GinFileMiddleware holds the value of the GinFileMiddleware edge.
	HCLGinFileMiddleware *GinFileMiddleware `json:"GinFileMiddleware,omitempty"`
	// ^^^^^^^^^^^^ CUSTOM ^^^^^^^^^^^^^
	gin_file_middleware_provisioning_scheduled_step *uuid.UUID
	plan_provisioning_scheduled_step                *uuid.UUID
	provisioning_scheduled_step_scheduled_step      *uuid.UUID
	provisioning_scheduled_step_provisioned_host    *uuid.UUID
	provisioning_scheduled_step_script              *uuid.UUID
	provisioning_scheduled_step_command             *uuid.UUID
	provisioning_scheduled_step_dns_record          *uuid.UUID
	provisioning_scheduled_step_file_delete         *uuid.UUID
	provisioning_scheduled_step_file_download       *uuid.UUID
	provisioning_scheduled_step_file_extract        *uuid.UUID
	provisioning_scheduled_step_ansible             *uuid.UUID
	selectValues                                    sql.SelectValues
}

// ProvisioningScheduledStepEdges holds the relations/edges for other nodes in the graph.
type ProvisioningScheduledStepEdges struct {
	// Status holds the value of the Status edge.
	Status *Status `json:"Status,omitempty"`
	// ScheduledStep holds the value of the ScheduledStep edge.
	ScheduledStep *ScheduledStep `json:"ScheduledStep,omitempty"`
	// ProvisionedHost holds the value of the ProvisionedHost edge.
	ProvisionedHost *ProvisionedHost `json:"ProvisionedHost,omitempty"`
	// Script holds the value of the Script edge.
	Script *Script `json:"Script,omitempty"`
	// Command holds the value of the Command edge.
	Command *Command `json:"Command,omitempty"`
	// DNSRecord holds the value of the DNSRecord edge.
	DNSRecord *DNSRecord `json:"DNSRecord,omitempty"`
	// FileDelete holds the value of the FileDelete edge.
	FileDelete *FileDelete `json:"FileDelete,omitempty"`
	// FileDownload holds the value of the FileDownload edge.
	FileDownload *FileDownload `json:"FileDownload,omitempty"`
	// FileExtract holds the value of the FileExtract edge.
	FileExtract *FileExtract `json:"FileExtract,omitempty"`
	// Ansible holds the value of the Ansible edge.
	Ansible *Ansible `json:"Ansible,omitempty"`
	// AgentTasks holds the value of the AgentTasks edge.
	AgentTasks []*AgentTask `json:"AgentTasks,omitempty"`
	// Plan holds the value of the Plan edge.
	Plan *Plan `json:"Plan,omitempty"`
	// GinFileMiddleware holds the value of the GinFileMiddleware edge.
	GinFileMiddleware *GinFileMiddleware `json:"GinFileMiddleware,omitempty"`
	// loadedTypes holds the information for reporting if a
	// type was loaded (or requested) in eager-loading or not.
	loadedTypes [13]bool
	// totalCount holds the count of the edges above.
	totalCount [13]map[string]int

	namedAgentTasks map[string][]*AgentTask
}

// StatusOrErr returns the Status value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e ProvisioningScheduledStepEdges) StatusOrErr() (*Status, error) {
	if e.loadedTypes[0] {
		if e.Status == nil {
			// Edge was loaded but was not found.
			return nil, &NotFoundError{label: status.Label}
		}
		return e.Status, nil
	}
	return nil, &NotLoadedError{edge: "Status"}
}

// ScheduledStepOrErr returns the ScheduledStep value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e ProvisioningScheduledStepEdges) ScheduledStepOrErr() (*ScheduledStep, error) {
	if e.loadedTypes[1] {
		if e.ScheduledStep == nil {
			// Edge was loaded but was not found.
			return nil, &NotFoundError{label: scheduledstep.Label}
		}
		return e.ScheduledStep, nil
	}
	return nil, &NotLoadedError{edge: "ScheduledStep"}
}

// ProvisionedHostOrErr returns the ProvisionedHost value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e ProvisioningScheduledStepEdges) ProvisionedHostOrErr() (*ProvisionedHost, error) {
	if e.loadedTypes[2] {
		if e.ProvisionedHost == nil {
			// Edge was loaded but was not found.
			return nil, &NotFoundError{label: provisionedhost.Label}
		}
		return e.ProvisionedHost, nil
	}
	return nil, &NotLoadedError{edge: "ProvisionedHost"}
}

// ScriptOrErr returns the Script value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e ProvisioningScheduledStepEdges) ScriptOrErr() (*Script, error) {
	if e.loadedTypes[3] {
		if e.Script == nil {
			// Edge was loaded but was not found.
			return nil, &NotFoundError{label: script.Label}
		}
		return e.Script, nil
	}
	return nil, &NotLoadedError{edge: "Script"}
}

// CommandOrErr returns the Command value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e ProvisioningScheduledStepEdges) CommandOrErr() (*Command, error) {
	if e.loadedTypes[4] {
		if e.Command == nil {
			// Edge was loaded but was not found.
			return nil, &NotFoundError{label: command.Label}
		}
		return e.Command, nil
	}
	return nil, &NotLoadedError{edge: "Command"}
}

// DNSRecordOrErr returns the DNSRecord value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e ProvisioningScheduledStepEdges) DNSRecordOrErr() (*DNSRecord, error) {
	if e.loadedTypes[5] {
		if e.DNSRecord == nil {
			// Edge was loaded but was not found.
			return nil, &NotFoundError{label: dnsrecord.Label}
		}
		return e.DNSRecord, nil
	}
	return nil, &NotLoadedError{edge: "DNSRecord"}
}

// FileDeleteOrErr returns the FileDelete value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e ProvisioningScheduledStepEdges) FileDeleteOrErr() (*FileDelete, error) {
	if e.loadedTypes[6] {
		if e.FileDelete == nil {
			// Edge was loaded but was not found.
			return nil, &NotFoundError{label: filedelete.Label}
		}
		return e.FileDelete, nil
	}
	return nil, &NotLoadedError{edge: "FileDelete"}
}

// FileDownloadOrErr returns the FileDownload value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e ProvisioningScheduledStepEdges) FileDownloadOrErr() (*FileDownload, error) {
	if e.loadedTypes[7] {
		if e.FileDownload == nil {
			// Edge was loaded but was not found.
			return nil, &NotFoundError{label: filedownload.Label}
		}
		return e.FileDownload, nil
	}
	return nil, &NotLoadedError{edge: "FileDownload"}
}

// FileExtractOrErr returns the FileExtract value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e ProvisioningScheduledStepEdges) FileExtractOrErr() (*FileExtract, error) {
	if e.loadedTypes[8] {
		if e.FileExtract == nil {
			// Edge was loaded but was not found.
			return nil, &NotFoundError{label: fileextract.Label}
		}
		return e.FileExtract, nil
	}
	return nil, &NotLoadedError{edge: "FileExtract"}
}

// AnsibleOrErr returns the Ansible value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e ProvisioningScheduledStepEdges) AnsibleOrErr() (*Ansible, error) {
	if e.loadedTypes[9] {
		if e.Ansible == nil {
			// Edge was loaded but was not found.
			return nil, &NotFoundError{label: ansible.Label}
		}
		return e.Ansible, nil
	}
	return nil, &NotLoadedError{edge: "Ansible"}
}

// AgentTasksOrErr returns the AgentTasks value or an error if the edge
// was not loaded in eager-loading.
func (e ProvisioningScheduledStepEdges) AgentTasksOrErr() ([]*AgentTask, error) {
	if e.loadedTypes[10] {
		return e.AgentTasks, nil
	}
	return nil, &NotLoadedError{edge: "AgentTasks"}
}

// PlanOrErr returns the Plan value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e ProvisioningScheduledStepEdges) PlanOrErr() (*Plan, error) {
	if e.loadedTypes[11] {
		if e.Plan == nil {
			// Edge was loaded but was not found.
			return nil, &NotFoundError{label: plan.Label}
		}
		return e.Plan, nil
	}
	return nil, &NotLoadedError{edge: "Plan"}
}

// GinFileMiddlewareOrErr returns the GinFileMiddleware value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e ProvisioningScheduledStepEdges) GinFileMiddlewareOrErr() (*GinFileMiddleware, error) {
	if e.loadedTypes[12] {
		if e.GinFileMiddleware == nil {
			// Edge was loaded but was not found.
			return nil, &NotFoundError{label: ginfilemiddleware.Label}
		}
		return e.GinFileMiddleware, nil
	}
	return nil, &NotLoadedError{edge: "GinFileMiddleware"}
}

// scanValues returns the types for scanning values from sql.Rows.
func (*ProvisioningScheduledStep) scanValues(columns []string) ([]any, error) {
	values := make([]any, len(columns))
	for i := range columns {
		switch columns[i] {
		case provisioningscheduledstep.FieldType:
			values[i] = new(sql.NullString)
		case provisioningscheduledstep.FieldRunTime:
			values[i] = new(sql.NullTime)
		case provisioningscheduledstep.FieldID:
			values[i] = new(uuid.UUID)
		case provisioningscheduledstep.ForeignKeys[0]: // gin_file_middleware_provisioning_scheduled_step
			values[i] = &sql.NullScanner{S: new(uuid.UUID)}
		case provisioningscheduledstep.ForeignKeys[1]: // plan_provisioning_scheduled_step
			values[i] = &sql.NullScanner{S: new(uuid.UUID)}
		case provisioningscheduledstep.ForeignKeys[2]: // provisioning_scheduled_step_scheduled_step
			values[i] = &sql.NullScanner{S: new(uuid.UUID)}
		case provisioningscheduledstep.ForeignKeys[3]: // provisioning_scheduled_step_provisioned_host
			values[i] = &sql.NullScanner{S: new(uuid.UUID)}
		case provisioningscheduledstep.ForeignKeys[4]: // provisioning_scheduled_step_script
			values[i] = &sql.NullScanner{S: new(uuid.UUID)}
		case provisioningscheduledstep.ForeignKeys[5]: // provisioning_scheduled_step_command
			values[i] = &sql.NullScanner{S: new(uuid.UUID)}
		case provisioningscheduledstep.ForeignKeys[6]: // provisioning_scheduled_step_dns_record
			values[i] = &sql.NullScanner{S: new(uuid.UUID)}
		case provisioningscheduledstep.ForeignKeys[7]: // provisioning_scheduled_step_file_delete
			values[i] = &sql.NullScanner{S: new(uuid.UUID)}
		case provisioningscheduledstep.ForeignKeys[8]: // provisioning_scheduled_step_file_download
			values[i] = &sql.NullScanner{S: new(uuid.UUID)}
		case provisioningscheduledstep.ForeignKeys[9]: // provisioning_scheduled_step_file_extract
			values[i] = &sql.NullScanner{S: new(uuid.UUID)}
		case provisioningscheduledstep.ForeignKeys[10]: // provisioning_scheduled_step_ansible
			values[i] = &sql.NullScanner{S: new(uuid.UUID)}
		default:
			values[i] = new(sql.UnknownType)
		}
	}
	return values, nil
}

// assignValues assigns the values that were returned from sql.Rows (after scanning)
// to the ProvisioningScheduledStep fields.
func (pss *ProvisioningScheduledStep) assignValues(columns []string, values []any) error {
	if m, n := len(values), len(columns); m < n {
		return fmt.Errorf("mismatch number of scan values: %d != %d", m, n)
	}
	for i := range columns {
		switch columns[i] {
		case provisioningscheduledstep.FieldID:
			if value, ok := values[i].(*uuid.UUID); !ok {
				return fmt.Errorf("unexpected type %T for field id", values[i])
			} else if value != nil {
				pss.ID = *value
			}
		case provisioningscheduledstep.FieldType:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field type", values[i])
			} else if value.Valid {
				pss.Type = provisioningscheduledstep.Type(value.String)
			}
		case provisioningscheduledstep.FieldRunTime:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field run_time", values[i])
			} else if value.Valid {
				pss.RunTime = value.Time
			}
		case provisioningscheduledstep.ForeignKeys[0]:
			if value, ok := values[i].(*sql.NullScanner); !ok {
				return fmt.Errorf("unexpected type %T for field gin_file_middleware_provisioning_scheduled_step", values[i])
			} else if value.Valid {
				pss.gin_file_middleware_provisioning_scheduled_step = new(uuid.UUID)
				*pss.gin_file_middleware_provisioning_scheduled_step = *value.S.(*uuid.UUID)
			}
		case provisioningscheduledstep.ForeignKeys[1]:
			if value, ok := values[i].(*sql.NullScanner); !ok {
				return fmt.Errorf("unexpected type %T for field plan_provisioning_scheduled_step", values[i])
			} else if value.Valid {
				pss.plan_provisioning_scheduled_step = new(uuid.UUID)
				*pss.plan_provisioning_scheduled_step = *value.S.(*uuid.UUID)
			}
		case provisioningscheduledstep.ForeignKeys[2]:
			if value, ok := values[i].(*sql.NullScanner); !ok {
				return fmt.Errorf("unexpected type %T for field provisioning_scheduled_step_scheduled_step", values[i])
			} else if value.Valid {
				pss.provisioning_scheduled_step_scheduled_step = new(uuid.UUID)
				*pss.provisioning_scheduled_step_scheduled_step = *value.S.(*uuid.UUID)
			}
		case provisioningscheduledstep.ForeignKeys[3]:
			if value, ok := values[i].(*sql.NullScanner); !ok {
				return fmt.Errorf("unexpected type %T for field provisioning_scheduled_step_provisioned_host", values[i])
			} else if value.Valid {
				pss.provisioning_scheduled_step_provisioned_host = new(uuid.UUID)
				*pss.provisioning_scheduled_step_provisioned_host = *value.S.(*uuid.UUID)
			}
		case provisioningscheduledstep.ForeignKeys[4]:
			if value, ok := values[i].(*sql.NullScanner); !ok {
				return fmt.Errorf("unexpected type %T for field provisioning_scheduled_step_script", values[i])
			} else if value.Valid {
				pss.provisioning_scheduled_step_script = new(uuid.UUID)
				*pss.provisioning_scheduled_step_script = *value.S.(*uuid.UUID)
			}
		case provisioningscheduledstep.ForeignKeys[5]:
			if value, ok := values[i].(*sql.NullScanner); !ok {
				return fmt.Errorf("unexpected type %T for field provisioning_scheduled_step_command", values[i])
			} else if value.Valid {
				pss.provisioning_scheduled_step_command = new(uuid.UUID)
				*pss.provisioning_scheduled_step_command = *value.S.(*uuid.UUID)
			}
		case provisioningscheduledstep.ForeignKeys[6]:
			if value, ok := values[i].(*sql.NullScanner); !ok {
				return fmt.Errorf("unexpected type %T for field provisioning_scheduled_step_dns_record", values[i])
			} else if value.Valid {
				pss.provisioning_scheduled_step_dns_record = new(uuid.UUID)
				*pss.provisioning_scheduled_step_dns_record = *value.S.(*uuid.UUID)
			}
		case provisioningscheduledstep.ForeignKeys[7]:
			if value, ok := values[i].(*sql.NullScanner); !ok {
				return fmt.Errorf("unexpected type %T for field provisioning_scheduled_step_file_delete", values[i])
			} else if value.Valid {
				pss.provisioning_scheduled_step_file_delete = new(uuid.UUID)
				*pss.provisioning_scheduled_step_file_delete = *value.S.(*uuid.UUID)
			}
		case provisioningscheduledstep.ForeignKeys[8]:
			if value, ok := values[i].(*sql.NullScanner); !ok {
				return fmt.Errorf("unexpected type %T for field provisioning_scheduled_step_file_download", values[i])
			} else if value.Valid {
				pss.provisioning_scheduled_step_file_download = new(uuid.UUID)
				*pss.provisioning_scheduled_step_file_download = *value.S.(*uuid.UUID)
			}
		case provisioningscheduledstep.ForeignKeys[9]:
			if value, ok := values[i].(*sql.NullScanner); !ok {
				return fmt.Errorf("unexpected type %T for field provisioning_scheduled_step_file_extract", values[i])
			} else if value.Valid {
				pss.provisioning_scheduled_step_file_extract = new(uuid.UUID)
				*pss.provisioning_scheduled_step_file_extract = *value.S.(*uuid.UUID)
			}
		case provisioningscheduledstep.ForeignKeys[10]:
			if value, ok := values[i].(*sql.NullScanner); !ok {
				return fmt.Errorf("unexpected type %T for field provisioning_scheduled_step_ansible", values[i])
			} else if value.Valid {
				pss.provisioning_scheduled_step_ansible = new(uuid.UUID)
				*pss.provisioning_scheduled_step_ansible = *value.S.(*uuid.UUID)
			}
		default:
			pss.selectValues.Set(columns[i], values[i])
		}
	}
	return nil
}

// Value returns the ent.Value that was dynamically selected and assigned to the ProvisioningScheduledStep.
// This includes values selected through modifiers, order, etc.
func (pss *ProvisioningScheduledStep) Value(name string) (ent.Value, error) {
	return pss.selectValues.Get(name)
}

// QueryStatus queries the "Status" edge of the ProvisioningScheduledStep entity.
func (pss *ProvisioningScheduledStep) QueryStatus() *StatusQuery {
	return NewProvisioningScheduledStepClient(pss.config).QueryStatus(pss)
}

// QueryScheduledStep queries the "ScheduledStep" edge of the ProvisioningScheduledStep entity.
func (pss *ProvisioningScheduledStep) QueryScheduledStep() *ScheduledStepQuery {
	return NewProvisioningScheduledStepClient(pss.config).QueryScheduledStep(pss)
}

// QueryProvisionedHost queries the "ProvisionedHost" edge of the ProvisioningScheduledStep entity.
func (pss *ProvisioningScheduledStep) QueryProvisionedHost() *ProvisionedHostQuery {
	return NewProvisioningScheduledStepClient(pss.config).QueryProvisionedHost(pss)
}

// QueryScript queries the "Script" edge of the ProvisioningScheduledStep entity.
func (pss *ProvisioningScheduledStep) QueryScript() *ScriptQuery {
	return NewProvisioningScheduledStepClient(pss.config).QueryScript(pss)
}

// QueryCommand queries the "Command" edge of the ProvisioningScheduledStep entity.
func (pss *ProvisioningScheduledStep) QueryCommand() *CommandQuery {
	return NewProvisioningScheduledStepClient(pss.config).QueryCommand(pss)
}

// QueryDNSRecord queries the "DNSRecord" edge of the ProvisioningScheduledStep entity.
func (pss *ProvisioningScheduledStep) QueryDNSRecord() *DNSRecordQuery {
	return NewProvisioningScheduledStepClient(pss.config).QueryDNSRecord(pss)
}

// QueryFileDelete queries the "FileDelete" edge of the ProvisioningScheduledStep entity.
func (pss *ProvisioningScheduledStep) QueryFileDelete() *FileDeleteQuery {
	return NewProvisioningScheduledStepClient(pss.config).QueryFileDelete(pss)
}

// QueryFileDownload queries the "FileDownload" edge of the ProvisioningScheduledStep entity.
func (pss *ProvisioningScheduledStep) QueryFileDownload() *FileDownloadQuery {
	return NewProvisioningScheduledStepClient(pss.config).QueryFileDownload(pss)
}

// QueryFileExtract queries the "FileExtract" edge of the ProvisioningScheduledStep entity.
func (pss *ProvisioningScheduledStep) QueryFileExtract() *FileExtractQuery {
	return NewProvisioningScheduledStepClient(pss.config).QueryFileExtract(pss)
}

// QueryAnsible queries the "Ansible" edge of the ProvisioningScheduledStep entity.
func (pss *ProvisioningScheduledStep) QueryAnsible() *AnsibleQuery {
	return NewProvisioningScheduledStepClient(pss.config).QueryAnsible(pss)
}

// QueryAgentTasks queries the "AgentTasks" edge of the ProvisioningScheduledStep entity.
func (pss *ProvisioningScheduledStep) QueryAgentTasks() *AgentTaskQuery {
	return NewProvisioningScheduledStepClient(pss.config).QueryAgentTasks(pss)
}

// QueryPlan queries the "Plan" edge of the ProvisioningScheduledStep entity.
func (pss *ProvisioningScheduledStep) QueryPlan() *PlanQuery {
	return NewProvisioningScheduledStepClient(pss.config).QueryPlan(pss)
}

// QueryGinFileMiddleware queries the "GinFileMiddleware" edge of the ProvisioningScheduledStep entity.
func (pss *ProvisioningScheduledStep) QueryGinFileMiddleware() *GinFileMiddlewareQuery {
	return NewProvisioningScheduledStepClient(pss.config).QueryGinFileMiddleware(pss)
}

// Update returns a builder for updating this ProvisioningScheduledStep.
// Note that you need to call ProvisioningScheduledStep.Unwrap() before calling this method if this ProvisioningScheduledStep
// was returned from a transaction, and the transaction was committed or rolled back.
func (pss *ProvisioningScheduledStep) Update() *ProvisioningScheduledStepUpdateOne {
	return NewProvisioningScheduledStepClient(pss.config).UpdateOne(pss)
}

// Unwrap unwraps the ProvisioningScheduledStep entity that was returned from a transaction after it was closed,
// so that all future queries will be executed through the driver which created the transaction.
func (pss *ProvisioningScheduledStep) Unwrap() *ProvisioningScheduledStep {
	_tx, ok := pss.config.driver.(*txDriver)
	if !ok {
		panic("ent: ProvisioningScheduledStep is not a transactional entity")
	}
	pss.config.driver = _tx.drv
	return pss
}

// String implements the fmt.Stringer.
func (pss *ProvisioningScheduledStep) String() string {
	var builder strings.Builder
	builder.WriteString("ProvisioningScheduledStep(")
	builder.WriteString(fmt.Sprintf("id=%v, ", pss.ID))
	builder.WriteString("type=")
	builder.WriteString(fmt.Sprintf("%v", pss.Type))
	builder.WriteString(", ")
	builder.WriteString("run_time=")
	builder.WriteString(pss.RunTime.Format(time.ANSIC))
	builder.WriteByte(')')
	return builder.String()
}

// NamedAgentTasks returns the AgentTasks named value or an error if the edge was not
// loaded in eager-loading with this name.
func (pss *ProvisioningScheduledStep) NamedAgentTasks(name string) ([]*AgentTask, error) {
	if pss.Edges.namedAgentTasks == nil {
		return nil, &NotLoadedError{edge: name}
	}
	nodes, ok := pss.Edges.namedAgentTasks[name]
	if !ok {
		return nil, &NotLoadedError{edge: name}
	}
	return nodes, nil
}

func (pss *ProvisioningScheduledStep) appendNamedAgentTasks(name string, edges ...*AgentTask) {
	if pss.Edges.namedAgentTasks == nil {
		pss.Edges.namedAgentTasks = make(map[string][]*AgentTask)
	}
	if len(edges) == 0 {
		pss.Edges.namedAgentTasks[name] = []*AgentTask{}
	} else {
		pss.Edges.namedAgentTasks[name] = append(pss.Edges.namedAgentTasks[name], edges...)
	}
}

// ProvisioningScheduledSteps is a parsable slice of ProvisioningScheduledStep.
type ProvisioningScheduledSteps []*ProvisioningScheduledStep
