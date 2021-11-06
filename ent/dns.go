// Code generated by entc, DO NOT EDIT.

package ent

import (
	"encoding/json"
	"fmt"
	"strings"

	"entgo.io/ent/dialect/sql"
	"github.com/gen0cide/laforge/ent/dns"
	"github.com/google/uuid"
)

// DNS is the model entity for the DNS schema.
type DNS struct {
	config ` json:"-"`
	// ID of the ent.
	ID uuid.UUID `json:"id,omitempty"`
	// HclID holds the value of the "hcl_id" field.
	HclID string `json:"hcl_id,omitempty" hcl:"id,label"`
	// Type holds the value of the "type" field.
	Type string `json:"type,omitempty" hcl:"type,attr"`
	// RootDomain holds the value of the "root_domain" field.
	RootDomain string `json:"root_domain,omitempty" hcl:"root_domain,attr" `
	// DNSServers holds the value of the "dns_servers" field.
	DNSServers []string `json:"dns_servers,omitempty" hcl:"dns_servers,attr"`
	// NtpServers holds the value of the "ntp_servers" field.
	NtpServers []string `json:"ntp_servers,omitempty" hcl:"ntp_servers,optional"`
	// Config holds the value of the "config" field.
	Config map[string]string `json:"config,omitempty" hcl:"config,optional"`
	// Edges holds the relations/edges for other nodes in the graph.
	// The values are being populated by the DNSQuery when eager-loading is set.
	Edges DNSEdges `json:"edges"`

	// Edges put into the main struct to be loaded via hcl
	// DNSToEnvironment holds the value of the DNSToEnvironment edge.
	HCLDNSToEnvironment []*Environment `json:"DNSToEnvironment,omitempty"`
	// DNSToCompetition holds the value of the DNSToCompetition edge.
	HCLDNSToCompetition []*Competition `json:"DNSToCompetition,omitempty"`
	//

}

// DNSEdges holds the relations/edges for other nodes in the graph.
type DNSEdges struct {
	// DNSToEnvironment holds the value of the DNSToEnvironment edge.
	DNSToEnvironment []*Environment `json:"DNSToEnvironment,omitempty"`
	// DNSToCompetition holds the value of the DNSToCompetition edge.
	DNSToCompetition []*Competition `json:"DNSToCompetition,omitempty"`
	// loadedTypes holds the information for reporting if a
	// type was loaded (or requested) in eager-loading or not.
	loadedTypes [2]bool
}

// DNSToEnvironmentOrErr returns the DNSToEnvironment value or an error if the edge
// was not loaded in eager-loading.
func (e DNSEdges) DNSToEnvironmentOrErr() ([]*Environment, error) {
	if e.loadedTypes[0] {
		return e.DNSToEnvironment, nil
	}
	return nil, &NotLoadedError{edge: "DNSToEnvironment"}
}

// DNSToCompetitionOrErr returns the DNSToCompetition value or an error if the edge
// was not loaded in eager-loading.
func (e DNSEdges) DNSToCompetitionOrErr() ([]*Competition, error) {
	if e.loadedTypes[1] {
		return e.DNSToCompetition, nil
	}
	return nil, &NotLoadedError{edge: "DNSToCompetition"}
}

// scanValues returns the types for scanning values from sql.Rows.
func (*DNS) scanValues(columns []string) ([]interface{}, error) {
	values := make([]interface{}, len(columns))
	for i := range columns {
		switch columns[i] {
		case dns.FieldDNSServers, dns.FieldNtpServers, dns.FieldConfig:
			values[i] = new([]byte)
		case dns.FieldHclID, dns.FieldType, dns.FieldRootDomain:
			values[i] = new(sql.NullString)
		case dns.FieldID:
			values[i] = new(uuid.UUID)
		default:
			return nil, fmt.Errorf("unexpected column %q for type DNS", columns[i])
		}
	}
	return values, nil
}

// assignValues assigns the values that were returned from sql.Rows (after scanning)
// to the DNS fields.
func (d *DNS) assignValues(columns []string, values []interface{}) error {
	if m, n := len(values), len(columns); m < n {
		return fmt.Errorf("mismatch number of scan values: %d != %d", m, n)
	}
	for i := range columns {
		switch columns[i] {
		case dns.FieldID:
			if value, ok := values[i].(*uuid.UUID); !ok {
				return fmt.Errorf("unexpected type %T for field id", values[i])
			} else if value != nil {
				d.ID = *value
			}
		case dns.FieldHclID:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field hcl_id", values[i])
			} else if value.Valid {
				d.HclID = value.String
			}
		case dns.FieldType:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field type", values[i])
			} else if value.Valid {
				d.Type = value.String
			}
		case dns.FieldRootDomain:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field root_domain", values[i])
			} else if value.Valid {
				d.RootDomain = value.String
			}
		case dns.FieldDNSServers:
			if value, ok := values[i].(*[]byte); !ok {
				return fmt.Errorf("unexpected type %T for field dns_servers", values[i])
			} else if value != nil && len(*value) > 0 {
				if err := json.Unmarshal(*value, &d.DNSServers); err != nil {
					return fmt.Errorf("unmarshal field dns_servers: %w", err)
				}
			}
		case dns.FieldNtpServers:
			if value, ok := values[i].(*[]byte); !ok {
				return fmt.Errorf("unexpected type %T for field ntp_servers", values[i])
			} else if value != nil && len(*value) > 0 {
				if err := json.Unmarshal(*value, &d.NtpServers); err != nil {
					return fmt.Errorf("unmarshal field ntp_servers: %w", err)
				}
			}
		case dns.FieldConfig:
			if value, ok := values[i].(*[]byte); !ok {
				return fmt.Errorf("unexpected type %T for field config", values[i])
			} else if value != nil && len(*value) > 0 {
				if err := json.Unmarshal(*value, &d.Config); err != nil {
					return fmt.Errorf("unmarshal field config: %w", err)
				}
			}
		}
	}
	return nil
}

// QueryDNSToEnvironment queries the "DNSToEnvironment" edge of the DNS entity.
func (d *DNS) QueryDNSToEnvironment() *EnvironmentQuery {
	return (&DNSClient{config: d.config}).QueryDNSToEnvironment(d)
}

// QueryDNSToCompetition queries the "DNSToCompetition" edge of the DNS entity.
func (d *DNS) QueryDNSToCompetition() *CompetitionQuery {
	return (&DNSClient{config: d.config}).QueryDNSToCompetition(d)
}

// Update returns a builder for updating this DNS.
// Note that you need to call DNS.Unwrap() before calling this method if this DNS
// was returned from a transaction, and the transaction was committed or rolled back.
func (d *DNS) Update() *DNSUpdateOne {
	return (&DNSClient{config: d.config}).UpdateOne(d)
}

// Unwrap unwraps the DNS entity that was returned from a transaction after it was closed,
// so that all future queries will be executed through the driver which created the transaction.
func (d *DNS) Unwrap() *DNS {
	tx, ok := d.config.driver.(*txDriver)
	if !ok {
		panic("ent: DNS is not a transactional entity")
	}
	d.config.driver = tx.drv
	return d
}

// String implements the fmt.Stringer.
func (d *DNS) String() string {
	var builder strings.Builder
	builder.WriteString("DNS(")
	builder.WriteString(fmt.Sprintf("id=%v", d.ID))
	builder.WriteString(", hcl_id=")
	builder.WriteString(d.HclID)
	builder.WriteString(", type=")
	builder.WriteString(d.Type)
	builder.WriteString(", root_domain=")
	builder.WriteString(d.RootDomain)
	builder.WriteString(", dns_servers=")
	builder.WriteString(fmt.Sprintf("%v", d.DNSServers))
	builder.WriteString(", ntp_servers=")
	builder.WriteString(fmt.Sprintf("%v", d.NtpServers))
	builder.WriteString(", config=")
	builder.WriteString(fmt.Sprintf("%v", d.Config))
	builder.WriteByte(')')
	return builder.String()
}

// DNSs is a parsable slice of DNS.
type DNSs []*DNS

func (d DNSs) config(cfg config) {
	for _i := range d {
		d[_i].config = cfg
	}
}
