// Code generated by entc, DO NOT EDIT.

package ent

import (
	"encoding/json"
	"fmt"
	"strings"

	"entgo.io/ent/dialect/sql"
	"github.com/gen0cide/laforge/ent/tag"
	"github.com/google/uuid"
)

// Tag is the model entity for the Tag schema.
type Tag struct {
	config `json:"-"`
	// ID of the ent.
	ID uuid.UUID `json:"id,omitempty"`
	// UUID holds the value of the "uuid" field.
	UUID uuid.UUID `json:"uuid,omitempty"`
	// Name holds the value of the "name" field.
	Name string `json:"name,omitempty"`
	// Description holds the value of the "description" field.
	Description                              map[string]string `json:"description,omitempty"`
	included_network_included_network_to_tag *uuid.UUID
	user_user_to_tag                         *uuid.UUID
}

// scanValues returns the types for scanning values from sql.Rows.
func (*Tag) scanValues(columns []string) ([]interface{}, error) {
	values := make([]interface{}, len(columns))
	for i := range columns {
		switch columns[i] {
		case tag.FieldDescription:
			values[i] = new([]byte)
		case tag.FieldName:
			values[i] = new(sql.NullString)
		case tag.FieldID, tag.FieldUUID:
			values[i] = new(uuid.UUID)
		case tag.ForeignKeys[0]: // included_network_included_network_to_tag
			values[i] = &sql.NullScanner{S: new(uuid.UUID)}
		case tag.ForeignKeys[1]: // user_user_to_tag
			values[i] = &sql.NullScanner{S: new(uuid.UUID)}
		default:
			return nil, fmt.Errorf("unexpected column %q for type Tag", columns[i])
		}
	}
	return values, nil
}

// assignValues assigns the values that were returned from sql.Rows (after scanning)
// to the Tag fields.
func (t *Tag) assignValues(columns []string, values []interface{}) error {
	if m, n := len(values), len(columns); m < n {
		return fmt.Errorf("mismatch number of scan values: %d != %d", m, n)
	}
	for i := range columns {
		switch columns[i] {
		case tag.FieldID:
			if value, ok := values[i].(*uuid.UUID); !ok {
				return fmt.Errorf("unexpected type %T for field id", values[i])
			} else if value != nil {
				t.ID = *value
			}
		case tag.FieldUUID:
			if value, ok := values[i].(*uuid.UUID); !ok {
				return fmt.Errorf("unexpected type %T for field uuid", values[i])
			} else if value != nil {
				t.UUID = *value
			}
		case tag.FieldName:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field name", values[i])
			} else if value.Valid {
				t.Name = value.String
			}
		case tag.FieldDescription:
			if value, ok := values[i].(*[]byte); !ok {
				return fmt.Errorf("unexpected type %T for field description", values[i])
			} else if value != nil && len(*value) > 0 {
				if err := json.Unmarshal(*value, &t.Description); err != nil {
					return fmt.Errorf("unmarshal field description: %w", err)
				}
			}
		case tag.ForeignKeys[0]:
			if value, ok := values[i].(*sql.NullScanner); !ok {
				return fmt.Errorf("unexpected type %T for field included_network_included_network_to_tag", values[i])
			} else if value.Valid {
				t.included_network_included_network_to_tag = new(uuid.UUID)
				*t.included_network_included_network_to_tag = *value.S.(*uuid.UUID)
			}
		case tag.ForeignKeys[1]:
			if value, ok := values[i].(*sql.NullScanner); !ok {
				return fmt.Errorf("unexpected type %T for field user_user_to_tag", values[i])
			} else if value.Valid {
				t.user_user_to_tag = new(uuid.UUID)
				*t.user_user_to_tag = *value.S.(*uuid.UUID)
			}
		}
	}
	return nil
}

// Update returns a builder for updating this Tag.
// Note that you need to call Tag.Unwrap() before calling this method if this Tag
// was returned from a transaction, and the transaction was committed or rolled back.
func (t *Tag) Update() *TagUpdateOne {
	return (&TagClient{config: t.config}).UpdateOne(t)
}

// Unwrap unwraps the Tag entity that was returned from a transaction after it was closed,
// so that all future queries will be executed through the driver which created the transaction.
func (t *Tag) Unwrap() *Tag {
	tx, ok := t.config.driver.(*txDriver)
	if !ok {
		panic("ent: Tag is not a transactional entity")
	}
	t.config.driver = tx.drv
	return t
}

// String implements the fmt.Stringer.
func (t *Tag) String() string {
	var builder strings.Builder
	builder.WriteString("Tag(")
	builder.WriteString(fmt.Sprintf("id=%v", t.ID))
	builder.WriteString(", uuid=")
	builder.WriteString(fmt.Sprintf("%v", t.UUID))
	builder.WriteString(", name=")
	builder.WriteString(t.Name)
	builder.WriteString(", description=")
	builder.WriteString(fmt.Sprintf("%v", t.Description))
	builder.WriteByte(')')
	return builder.String()
}

// Tags is a parsable slice of Tag.
type Tags []*Tag

func (t Tags) config(cfg config) {
	for _i := range t {
		t[_i].config = cfg
	}
}
