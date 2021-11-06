// Code generated by entc, DO NOT EDIT.

package ent

import (
	"encoding/json"
	"fmt"
	"strings"

	"entgo.io/ent/dialect/sql"
	"github.com/gen0cide/laforge/ent/environment"
	"github.com/gen0cide/laforge/ent/identity"
	"github.com/google/uuid"
)

// Identity is the model entity for the Identity schema.
type Identity struct {
	config ` json:"-"`
	// ID of the ent.
	ID uuid.UUID `json:"id,omitempty"`
	// HclID holds the value of the "hcl_id" field.
	HclID string `json:"hcl_id,omitempty" hcl:"id,label"`
	// FirstName holds the value of the "first_name" field.
	FirstName string `json:"first_name,omitempty" hcl:"firstname,attr"`
	// LastName holds the value of the "last_name" field.
	LastName string `json:"last_name,omitempty" hcl:"lastname,attr" `
	// Email holds the value of the "email" field.
	Email string `json:"email,omitempty" hcl:"email,attr" `
	// Password holds the value of the "password" field.
	Password string `json:"password,omitempty" hcl:"password,attr" `
	// Description holds the value of the "description" field.
	Description string `json:"description,omitempty" hcl:"description,optional" `
	// AvatarFile holds the value of the "avatar_file" field.
	AvatarFile string `json:"avatar_file,omitempty" hcl:"avatar_file,optional" `
	// Vars holds the value of the "vars" field.
	Vars map[string]string `json:"vars,omitempty" hcl:"vars,optional"`
	// Tags holds the value of the "tags" field.
	Tags map[string]string `json:"tags,omitempty" hcl:"tags,optional"`
	// Edges holds the relations/edges for other nodes in the graph.
	// The values are being populated by the IdentityQuery when eager-loading is set.
	Edges IdentityEdges `json:"edges"`

	// Edges put into the main struct to be loaded via hcl
	// IdentityToEnvironment holds the value of the IdentityToEnvironment edge.
	HCLIdentityToEnvironment *Environment `json:"IdentityToEnvironment,omitempty"`
	//
	environment_environment_to_identity *uuid.UUID
}

// IdentityEdges holds the relations/edges for other nodes in the graph.
type IdentityEdges struct {
	// IdentityToEnvironment holds the value of the IdentityToEnvironment edge.
	IdentityToEnvironment *Environment `json:"IdentityToEnvironment,omitempty"`
	// loadedTypes holds the information for reporting if a
	// type was loaded (or requested) in eager-loading or not.
	loadedTypes [1]bool
}

// IdentityToEnvironmentOrErr returns the IdentityToEnvironment value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e IdentityEdges) IdentityToEnvironmentOrErr() (*Environment, error) {
	if e.loadedTypes[0] {
		if e.IdentityToEnvironment == nil {
			// The edge IdentityToEnvironment was loaded in eager-loading,
			// but was not found.
			return nil, &NotFoundError{label: environment.Label}
		}
		return e.IdentityToEnvironment, nil
	}
	return nil, &NotLoadedError{edge: "IdentityToEnvironment"}
}

// scanValues returns the types for scanning values from sql.Rows.
func (*Identity) scanValues(columns []string) ([]interface{}, error) {
	values := make([]interface{}, len(columns))
	for i := range columns {
		switch columns[i] {
		case identity.FieldVars, identity.FieldTags:
			values[i] = new([]byte)
		case identity.FieldHclID, identity.FieldFirstName, identity.FieldLastName, identity.FieldEmail, identity.FieldPassword, identity.FieldDescription, identity.FieldAvatarFile:
			values[i] = new(sql.NullString)
		case identity.FieldID:
			values[i] = new(uuid.UUID)
		case identity.ForeignKeys[0]: // environment_environment_to_identity
			values[i] = &sql.NullScanner{S: new(uuid.UUID)}
		default:
			return nil, fmt.Errorf("unexpected column %q for type Identity", columns[i])
		}
	}
	return values, nil
}

// assignValues assigns the values that were returned from sql.Rows (after scanning)
// to the Identity fields.
func (i *Identity) assignValues(columns []string, values []interface{}) error {
	if m, n := len(values), len(columns); m < n {
		return fmt.Errorf("mismatch number of scan values: %d != %d", m, n)
	}
	for j := range columns {
		switch columns[j] {
		case identity.FieldID:
			if value, ok := values[j].(*uuid.UUID); !ok {
				return fmt.Errorf("unexpected type %T for field id", values[j])
			} else if value != nil {
				i.ID = *value
			}
		case identity.FieldHclID:
			if value, ok := values[j].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field hcl_id", values[j])
			} else if value.Valid {
				i.HclID = value.String
			}
		case identity.FieldFirstName:
			if value, ok := values[j].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field first_name", values[j])
			} else if value.Valid {
				i.FirstName = value.String
			}
		case identity.FieldLastName:
			if value, ok := values[j].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field last_name", values[j])
			} else if value.Valid {
				i.LastName = value.String
			}
		case identity.FieldEmail:
			if value, ok := values[j].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field email", values[j])
			} else if value.Valid {
				i.Email = value.String
			}
		case identity.FieldPassword:
			if value, ok := values[j].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field password", values[j])
			} else if value.Valid {
				i.Password = value.String
			}
		case identity.FieldDescription:
			if value, ok := values[j].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field description", values[j])
			} else if value.Valid {
				i.Description = value.String
			}
		case identity.FieldAvatarFile:
			if value, ok := values[j].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field avatar_file", values[j])
			} else if value.Valid {
				i.AvatarFile = value.String
			}
		case identity.FieldVars:
			if value, ok := values[j].(*[]byte); !ok {
				return fmt.Errorf("unexpected type %T for field vars", values[j])
			} else if value != nil && len(*value) > 0 {
				if err := json.Unmarshal(*value, &i.Vars); err != nil {
					return fmt.Errorf("unmarshal field vars: %w", err)
				}
			}
		case identity.FieldTags:
			if value, ok := values[j].(*[]byte); !ok {
				return fmt.Errorf("unexpected type %T for field tags", values[j])
			} else if value != nil && len(*value) > 0 {
				if err := json.Unmarshal(*value, &i.Tags); err != nil {
					return fmt.Errorf("unmarshal field tags: %w", err)
				}
			}
		case identity.ForeignKeys[0]:
			if value, ok := values[j].(*sql.NullScanner); !ok {
				return fmt.Errorf("unexpected type %T for field environment_environment_to_identity", values[j])
			} else if value.Valid {
				i.environment_environment_to_identity = new(uuid.UUID)
				*i.environment_environment_to_identity = *value.S.(*uuid.UUID)
			}
		}
	}
	return nil
}

// QueryIdentityToEnvironment queries the "IdentityToEnvironment" edge of the Identity entity.
func (i *Identity) QueryIdentityToEnvironment() *EnvironmentQuery {
	return (&IdentityClient{config: i.config}).QueryIdentityToEnvironment(i)
}

// Update returns a builder for updating this Identity.
// Note that you need to call Identity.Unwrap() before calling this method if this Identity
// was returned from a transaction, and the transaction was committed or rolled back.
func (i *Identity) Update() *IdentityUpdateOne {
	return (&IdentityClient{config: i.config}).UpdateOne(i)
}

// Unwrap unwraps the Identity entity that was returned from a transaction after it was closed,
// so that all future queries will be executed through the driver which created the transaction.
func (i *Identity) Unwrap() *Identity {
	tx, ok := i.config.driver.(*txDriver)
	if !ok {
		panic("ent: Identity is not a transactional entity")
	}
	i.config.driver = tx.drv
	return i
}

// String implements the fmt.Stringer.
func (i *Identity) String() string {
	var builder strings.Builder
	builder.WriteString("Identity(")
	builder.WriteString(fmt.Sprintf("id=%v", i.ID))
	builder.WriteString(", hcl_id=")
	builder.WriteString(i.HclID)
	builder.WriteString(", first_name=")
	builder.WriteString(i.FirstName)
	builder.WriteString(", last_name=")
	builder.WriteString(i.LastName)
	builder.WriteString(", email=")
	builder.WriteString(i.Email)
	builder.WriteString(", password=")
	builder.WriteString(i.Password)
	builder.WriteString(", description=")
	builder.WriteString(i.Description)
	builder.WriteString(", avatar_file=")
	builder.WriteString(i.AvatarFile)
	builder.WriteString(", vars=")
	builder.WriteString(fmt.Sprintf("%v", i.Vars))
	builder.WriteString(", tags=")
	builder.WriteString(fmt.Sprintf("%v", i.Tags))
	builder.WriteByte(')')
	return builder.String()
}

// Identities is a parsable slice of Identity.
type Identities []*Identity

func (i Identities) config(cfg config) {
	for _i := range i {
		i[_i].config = cfg
	}
}
