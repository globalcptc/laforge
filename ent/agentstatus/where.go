// Code generated by ent, DO NOT EDIT.

package agentstatus

import (
	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"github.com/gen0cide/laforge/ent/predicate"
	"github.com/google/uuid"
)

// ID filters vertices based on their ID field.
func ID(id uuid.UUID) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldEQ(FieldID, id))
}

// IDEQ applies the EQ predicate on the ID field.
func IDEQ(id uuid.UUID) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldEQ(FieldID, id))
}

// IDNEQ applies the NEQ predicate on the ID field.
func IDNEQ(id uuid.UUID) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldNEQ(FieldID, id))
}

// IDIn applies the In predicate on the ID field.
func IDIn(ids ...uuid.UUID) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldIn(FieldID, ids...))
}

// IDNotIn applies the NotIn predicate on the ID field.
func IDNotIn(ids ...uuid.UUID) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldNotIn(FieldID, ids...))
}

// IDGT applies the GT predicate on the ID field.
func IDGT(id uuid.UUID) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldGT(FieldID, id))
}

// IDGTE applies the GTE predicate on the ID field.
func IDGTE(id uuid.UUID) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldGTE(FieldID, id))
}

// IDLT applies the LT predicate on the ID field.
func IDLT(id uuid.UUID) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldLT(FieldID, id))
}

// IDLTE applies the LTE predicate on the ID field.
func IDLTE(id uuid.UUID) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldLTE(FieldID, id))
}

// ClientID applies equality check predicate on the "ClientID" field. It's identical to ClientIDEQ.
func ClientID(v string) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldEQ(FieldClientID, v))
}

// Hostname applies equality check predicate on the "Hostname" field. It's identical to HostnameEQ.
func Hostname(v string) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldEQ(FieldHostname, v))
}

// UpTime applies equality check predicate on the "UpTime" field. It's identical to UpTimeEQ.
func UpTime(v int64) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldEQ(FieldUpTime, v))
}

// BootTime applies equality check predicate on the "BootTime" field. It's identical to BootTimeEQ.
func BootTime(v int64) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldEQ(FieldBootTime, v))
}

// NumProcs applies equality check predicate on the "NumProcs" field. It's identical to NumProcsEQ.
func NumProcs(v int64) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldEQ(FieldNumProcs, v))
}

// Os applies equality check predicate on the "Os" field. It's identical to OsEQ.
func Os(v string) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldEQ(FieldOs, v))
}

// HostID applies equality check predicate on the "HostID" field. It's identical to HostIDEQ.
func HostID(v string) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldEQ(FieldHostID, v))
}

// Load1 applies equality check predicate on the "Load1" field. It's identical to Load1EQ.
func Load1(v float64) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldEQ(FieldLoad1, v))
}

// Load5 applies equality check predicate on the "Load5" field. It's identical to Load5EQ.
func Load5(v float64) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldEQ(FieldLoad5, v))
}

// Load15 applies equality check predicate on the "Load15" field. It's identical to Load15EQ.
func Load15(v float64) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldEQ(FieldLoad15, v))
}

// TotalMem applies equality check predicate on the "TotalMem" field. It's identical to TotalMemEQ.
func TotalMem(v int64) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldEQ(FieldTotalMem, v))
}

// FreeMem applies equality check predicate on the "FreeMem" field. It's identical to FreeMemEQ.
func FreeMem(v int64) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldEQ(FieldFreeMem, v))
}

// UsedMem applies equality check predicate on the "UsedMem" field. It's identical to UsedMemEQ.
func UsedMem(v int64) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldEQ(FieldUsedMem, v))
}

// Timestamp applies equality check predicate on the "Timestamp" field. It's identical to TimestampEQ.
func Timestamp(v int64) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldEQ(FieldTimestamp, v))
}

// ClientIDEQ applies the EQ predicate on the "ClientID" field.
func ClientIDEQ(v string) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldEQ(FieldClientID, v))
}

// ClientIDNEQ applies the NEQ predicate on the "ClientID" field.
func ClientIDNEQ(v string) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldNEQ(FieldClientID, v))
}

// ClientIDIn applies the In predicate on the "ClientID" field.
func ClientIDIn(vs ...string) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldIn(FieldClientID, vs...))
}

// ClientIDNotIn applies the NotIn predicate on the "ClientID" field.
func ClientIDNotIn(vs ...string) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldNotIn(FieldClientID, vs...))
}

// ClientIDGT applies the GT predicate on the "ClientID" field.
func ClientIDGT(v string) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldGT(FieldClientID, v))
}

// ClientIDGTE applies the GTE predicate on the "ClientID" field.
func ClientIDGTE(v string) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldGTE(FieldClientID, v))
}

// ClientIDLT applies the LT predicate on the "ClientID" field.
func ClientIDLT(v string) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldLT(FieldClientID, v))
}

// ClientIDLTE applies the LTE predicate on the "ClientID" field.
func ClientIDLTE(v string) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldLTE(FieldClientID, v))
}

// ClientIDContains applies the Contains predicate on the "ClientID" field.
func ClientIDContains(v string) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldContains(FieldClientID, v))
}

// ClientIDHasPrefix applies the HasPrefix predicate on the "ClientID" field.
func ClientIDHasPrefix(v string) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldHasPrefix(FieldClientID, v))
}

// ClientIDHasSuffix applies the HasSuffix predicate on the "ClientID" field.
func ClientIDHasSuffix(v string) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldHasSuffix(FieldClientID, v))
}

// ClientIDEqualFold applies the EqualFold predicate on the "ClientID" field.
func ClientIDEqualFold(v string) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldEqualFold(FieldClientID, v))
}

// ClientIDContainsFold applies the ContainsFold predicate on the "ClientID" field.
func ClientIDContainsFold(v string) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldContainsFold(FieldClientID, v))
}

// HostnameEQ applies the EQ predicate on the "Hostname" field.
func HostnameEQ(v string) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldEQ(FieldHostname, v))
}

// HostnameNEQ applies the NEQ predicate on the "Hostname" field.
func HostnameNEQ(v string) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldNEQ(FieldHostname, v))
}

// HostnameIn applies the In predicate on the "Hostname" field.
func HostnameIn(vs ...string) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldIn(FieldHostname, vs...))
}

// HostnameNotIn applies the NotIn predicate on the "Hostname" field.
func HostnameNotIn(vs ...string) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldNotIn(FieldHostname, vs...))
}

// HostnameGT applies the GT predicate on the "Hostname" field.
func HostnameGT(v string) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldGT(FieldHostname, v))
}

// HostnameGTE applies the GTE predicate on the "Hostname" field.
func HostnameGTE(v string) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldGTE(FieldHostname, v))
}

// HostnameLT applies the LT predicate on the "Hostname" field.
func HostnameLT(v string) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldLT(FieldHostname, v))
}

// HostnameLTE applies the LTE predicate on the "Hostname" field.
func HostnameLTE(v string) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldLTE(FieldHostname, v))
}

// HostnameContains applies the Contains predicate on the "Hostname" field.
func HostnameContains(v string) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldContains(FieldHostname, v))
}

// HostnameHasPrefix applies the HasPrefix predicate on the "Hostname" field.
func HostnameHasPrefix(v string) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldHasPrefix(FieldHostname, v))
}

// HostnameHasSuffix applies the HasSuffix predicate on the "Hostname" field.
func HostnameHasSuffix(v string) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldHasSuffix(FieldHostname, v))
}

// HostnameEqualFold applies the EqualFold predicate on the "Hostname" field.
func HostnameEqualFold(v string) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldEqualFold(FieldHostname, v))
}

// HostnameContainsFold applies the ContainsFold predicate on the "Hostname" field.
func HostnameContainsFold(v string) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldContainsFold(FieldHostname, v))
}

// UpTimeEQ applies the EQ predicate on the "UpTime" field.
func UpTimeEQ(v int64) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldEQ(FieldUpTime, v))
}

// UpTimeNEQ applies the NEQ predicate on the "UpTime" field.
func UpTimeNEQ(v int64) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldNEQ(FieldUpTime, v))
}

// UpTimeIn applies the In predicate on the "UpTime" field.
func UpTimeIn(vs ...int64) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldIn(FieldUpTime, vs...))
}

// UpTimeNotIn applies the NotIn predicate on the "UpTime" field.
func UpTimeNotIn(vs ...int64) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldNotIn(FieldUpTime, vs...))
}

// UpTimeGT applies the GT predicate on the "UpTime" field.
func UpTimeGT(v int64) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldGT(FieldUpTime, v))
}

// UpTimeGTE applies the GTE predicate on the "UpTime" field.
func UpTimeGTE(v int64) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldGTE(FieldUpTime, v))
}

// UpTimeLT applies the LT predicate on the "UpTime" field.
func UpTimeLT(v int64) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldLT(FieldUpTime, v))
}

// UpTimeLTE applies the LTE predicate on the "UpTime" field.
func UpTimeLTE(v int64) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldLTE(FieldUpTime, v))
}

// BootTimeEQ applies the EQ predicate on the "BootTime" field.
func BootTimeEQ(v int64) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldEQ(FieldBootTime, v))
}

// BootTimeNEQ applies the NEQ predicate on the "BootTime" field.
func BootTimeNEQ(v int64) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldNEQ(FieldBootTime, v))
}

// BootTimeIn applies the In predicate on the "BootTime" field.
func BootTimeIn(vs ...int64) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldIn(FieldBootTime, vs...))
}

// BootTimeNotIn applies the NotIn predicate on the "BootTime" field.
func BootTimeNotIn(vs ...int64) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldNotIn(FieldBootTime, vs...))
}

// BootTimeGT applies the GT predicate on the "BootTime" field.
func BootTimeGT(v int64) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldGT(FieldBootTime, v))
}

// BootTimeGTE applies the GTE predicate on the "BootTime" field.
func BootTimeGTE(v int64) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldGTE(FieldBootTime, v))
}

// BootTimeLT applies the LT predicate on the "BootTime" field.
func BootTimeLT(v int64) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldLT(FieldBootTime, v))
}

// BootTimeLTE applies the LTE predicate on the "BootTime" field.
func BootTimeLTE(v int64) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldLTE(FieldBootTime, v))
}

// NumProcsEQ applies the EQ predicate on the "NumProcs" field.
func NumProcsEQ(v int64) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldEQ(FieldNumProcs, v))
}

// NumProcsNEQ applies the NEQ predicate on the "NumProcs" field.
func NumProcsNEQ(v int64) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldNEQ(FieldNumProcs, v))
}

// NumProcsIn applies the In predicate on the "NumProcs" field.
func NumProcsIn(vs ...int64) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldIn(FieldNumProcs, vs...))
}

// NumProcsNotIn applies the NotIn predicate on the "NumProcs" field.
func NumProcsNotIn(vs ...int64) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldNotIn(FieldNumProcs, vs...))
}

// NumProcsGT applies the GT predicate on the "NumProcs" field.
func NumProcsGT(v int64) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldGT(FieldNumProcs, v))
}

// NumProcsGTE applies the GTE predicate on the "NumProcs" field.
func NumProcsGTE(v int64) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldGTE(FieldNumProcs, v))
}

// NumProcsLT applies the LT predicate on the "NumProcs" field.
func NumProcsLT(v int64) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldLT(FieldNumProcs, v))
}

// NumProcsLTE applies the LTE predicate on the "NumProcs" field.
func NumProcsLTE(v int64) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldLTE(FieldNumProcs, v))
}

// OsEQ applies the EQ predicate on the "Os" field.
func OsEQ(v string) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldEQ(FieldOs, v))
}

// OsNEQ applies the NEQ predicate on the "Os" field.
func OsNEQ(v string) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldNEQ(FieldOs, v))
}

// OsIn applies the In predicate on the "Os" field.
func OsIn(vs ...string) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldIn(FieldOs, vs...))
}

// OsNotIn applies the NotIn predicate on the "Os" field.
func OsNotIn(vs ...string) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldNotIn(FieldOs, vs...))
}

// OsGT applies the GT predicate on the "Os" field.
func OsGT(v string) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldGT(FieldOs, v))
}

// OsGTE applies the GTE predicate on the "Os" field.
func OsGTE(v string) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldGTE(FieldOs, v))
}

// OsLT applies the LT predicate on the "Os" field.
func OsLT(v string) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldLT(FieldOs, v))
}

// OsLTE applies the LTE predicate on the "Os" field.
func OsLTE(v string) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldLTE(FieldOs, v))
}

// OsContains applies the Contains predicate on the "Os" field.
func OsContains(v string) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldContains(FieldOs, v))
}

// OsHasPrefix applies the HasPrefix predicate on the "Os" field.
func OsHasPrefix(v string) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldHasPrefix(FieldOs, v))
}

// OsHasSuffix applies the HasSuffix predicate on the "Os" field.
func OsHasSuffix(v string) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldHasSuffix(FieldOs, v))
}

// OsEqualFold applies the EqualFold predicate on the "Os" field.
func OsEqualFold(v string) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldEqualFold(FieldOs, v))
}

// OsContainsFold applies the ContainsFold predicate on the "Os" field.
func OsContainsFold(v string) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldContainsFold(FieldOs, v))
}

// HostIDEQ applies the EQ predicate on the "HostID" field.
func HostIDEQ(v string) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldEQ(FieldHostID, v))
}

// HostIDNEQ applies the NEQ predicate on the "HostID" field.
func HostIDNEQ(v string) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldNEQ(FieldHostID, v))
}

// HostIDIn applies the In predicate on the "HostID" field.
func HostIDIn(vs ...string) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldIn(FieldHostID, vs...))
}

// HostIDNotIn applies the NotIn predicate on the "HostID" field.
func HostIDNotIn(vs ...string) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldNotIn(FieldHostID, vs...))
}

// HostIDGT applies the GT predicate on the "HostID" field.
func HostIDGT(v string) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldGT(FieldHostID, v))
}

// HostIDGTE applies the GTE predicate on the "HostID" field.
func HostIDGTE(v string) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldGTE(FieldHostID, v))
}

// HostIDLT applies the LT predicate on the "HostID" field.
func HostIDLT(v string) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldLT(FieldHostID, v))
}

// HostIDLTE applies the LTE predicate on the "HostID" field.
func HostIDLTE(v string) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldLTE(FieldHostID, v))
}

// HostIDContains applies the Contains predicate on the "HostID" field.
func HostIDContains(v string) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldContains(FieldHostID, v))
}

// HostIDHasPrefix applies the HasPrefix predicate on the "HostID" field.
func HostIDHasPrefix(v string) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldHasPrefix(FieldHostID, v))
}

// HostIDHasSuffix applies the HasSuffix predicate on the "HostID" field.
func HostIDHasSuffix(v string) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldHasSuffix(FieldHostID, v))
}

// HostIDEqualFold applies the EqualFold predicate on the "HostID" field.
func HostIDEqualFold(v string) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldEqualFold(FieldHostID, v))
}

// HostIDContainsFold applies the ContainsFold predicate on the "HostID" field.
func HostIDContainsFold(v string) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldContainsFold(FieldHostID, v))
}

// Load1EQ applies the EQ predicate on the "Load1" field.
func Load1EQ(v float64) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldEQ(FieldLoad1, v))
}

// Load1NEQ applies the NEQ predicate on the "Load1" field.
func Load1NEQ(v float64) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldNEQ(FieldLoad1, v))
}

// Load1In applies the In predicate on the "Load1" field.
func Load1In(vs ...float64) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldIn(FieldLoad1, vs...))
}

// Load1NotIn applies the NotIn predicate on the "Load1" field.
func Load1NotIn(vs ...float64) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldNotIn(FieldLoad1, vs...))
}

// Load1GT applies the GT predicate on the "Load1" field.
func Load1GT(v float64) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldGT(FieldLoad1, v))
}

// Load1GTE applies the GTE predicate on the "Load1" field.
func Load1GTE(v float64) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldGTE(FieldLoad1, v))
}

// Load1LT applies the LT predicate on the "Load1" field.
func Load1LT(v float64) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldLT(FieldLoad1, v))
}

// Load1LTE applies the LTE predicate on the "Load1" field.
func Load1LTE(v float64) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldLTE(FieldLoad1, v))
}

// Load5EQ applies the EQ predicate on the "Load5" field.
func Load5EQ(v float64) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldEQ(FieldLoad5, v))
}

// Load5NEQ applies the NEQ predicate on the "Load5" field.
func Load5NEQ(v float64) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldNEQ(FieldLoad5, v))
}

// Load5In applies the In predicate on the "Load5" field.
func Load5In(vs ...float64) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldIn(FieldLoad5, vs...))
}

// Load5NotIn applies the NotIn predicate on the "Load5" field.
func Load5NotIn(vs ...float64) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldNotIn(FieldLoad5, vs...))
}

// Load5GT applies the GT predicate on the "Load5" field.
func Load5GT(v float64) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldGT(FieldLoad5, v))
}

// Load5GTE applies the GTE predicate on the "Load5" field.
func Load5GTE(v float64) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldGTE(FieldLoad5, v))
}

// Load5LT applies the LT predicate on the "Load5" field.
func Load5LT(v float64) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldLT(FieldLoad5, v))
}

// Load5LTE applies the LTE predicate on the "Load5" field.
func Load5LTE(v float64) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldLTE(FieldLoad5, v))
}

// Load15EQ applies the EQ predicate on the "Load15" field.
func Load15EQ(v float64) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldEQ(FieldLoad15, v))
}

// Load15NEQ applies the NEQ predicate on the "Load15" field.
func Load15NEQ(v float64) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldNEQ(FieldLoad15, v))
}

// Load15In applies the In predicate on the "Load15" field.
func Load15In(vs ...float64) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldIn(FieldLoad15, vs...))
}

// Load15NotIn applies the NotIn predicate on the "Load15" field.
func Load15NotIn(vs ...float64) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldNotIn(FieldLoad15, vs...))
}

// Load15GT applies the GT predicate on the "Load15" field.
func Load15GT(v float64) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldGT(FieldLoad15, v))
}

// Load15GTE applies the GTE predicate on the "Load15" field.
func Load15GTE(v float64) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldGTE(FieldLoad15, v))
}

// Load15LT applies the LT predicate on the "Load15" field.
func Load15LT(v float64) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldLT(FieldLoad15, v))
}

// Load15LTE applies the LTE predicate on the "Load15" field.
func Load15LTE(v float64) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldLTE(FieldLoad15, v))
}

// TotalMemEQ applies the EQ predicate on the "TotalMem" field.
func TotalMemEQ(v int64) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldEQ(FieldTotalMem, v))
}

// TotalMemNEQ applies the NEQ predicate on the "TotalMem" field.
func TotalMemNEQ(v int64) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldNEQ(FieldTotalMem, v))
}

// TotalMemIn applies the In predicate on the "TotalMem" field.
func TotalMemIn(vs ...int64) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldIn(FieldTotalMem, vs...))
}

// TotalMemNotIn applies the NotIn predicate on the "TotalMem" field.
func TotalMemNotIn(vs ...int64) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldNotIn(FieldTotalMem, vs...))
}

// TotalMemGT applies the GT predicate on the "TotalMem" field.
func TotalMemGT(v int64) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldGT(FieldTotalMem, v))
}

// TotalMemGTE applies the GTE predicate on the "TotalMem" field.
func TotalMemGTE(v int64) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldGTE(FieldTotalMem, v))
}

// TotalMemLT applies the LT predicate on the "TotalMem" field.
func TotalMemLT(v int64) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldLT(FieldTotalMem, v))
}

// TotalMemLTE applies the LTE predicate on the "TotalMem" field.
func TotalMemLTE(v int64) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldLTE(FieldTotalMem, v))
}

// FreeMemEQ applies the EQ predicate on the "FreeMem" field.
func FreeMemEQ(v int64) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldEQ(FieldFreeMem, v))
}

// FreeMemNEQ applies the NEQ predicate on the "FreeMem" field.
func FreeMemNEQ(v int64) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldNEQ(FieldFreeMem, v))
}

// FreeMemIn applies the In predicate on the "FreeMem" field.
func FreeMemIn(vs ...int64) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldIn(FieldFreeMem, vs...))
}

// FreeMemNotIn applies the NotIn predicate on the "FreeMem" field.
func FreeMemNotIn(vs ...int64) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldNotIn(FieldFreeMem, vs...))
}

// FreeMemGT applies the GT predicate on the "FreeMem" field.
func FreeMemGT(v int64) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldGT(FieldFreeMem, v))
}

// FreeMemGTE applies the GTE predicate on the "FreeMem" field.
func FreeMemGTE(v int64) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldGTE(FieldFreeMem, v))
}

// FreeMemLT applies the LT predicate on the "FreeMem" field.
func FreeMemLT(v int64) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldLT(FieldFreeMem, v))
}

// FreeMemLTE applies the LTE predicate on the "FreeMem" field.
func FreeMemLTE(v int64) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldLTE(FieldFreeMem, v))
}

// UsedMemEQ applies the EQ predicate on the "UsedMem" field.
func UsedMemEQ(v int64) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldEQ(FieldUsedMem, v))
}

// UsedMemNEQ applies the NEQ predicate on the "UsedMem" field.
func UsedMemNEQ(v int64) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldNEQ(FieldUsedMem, v))
}

// UsedMemIn applies the In predicate on the "UsedMem" field.
func UsedMemIn(vs ...int64) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldIn(FieldUsedMem, vs...))
}

// UsedMemNotIn applies the NotIn predicate on the "UsedMem" field.
func UsedMemNotIn(vs ...int64) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldNotIn(FieldUsedMem, vs...))
}

// UsedMemGT applies the GT predicate on the "UsedMem" field.
func UsedMemGT(v int64) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldGT(FieldUsedMem, v))
}

// UsedMemGTE applies the GTE predicate on the "UsedMem" field.
func UsedMemGTE(v int64) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldGTE(FieldUsedMem, v))
}

// UsedMemLT applies the LT predicate on the "UsedMem" field.
func UsedMemLT(v int64) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldLT(FieldUsedMem, v))
}

// UsedMemLTE applies the LTE predicate on the "UsedMem" field.
func UsedMemLTE(v int64) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldLTE(FieldUsedMem, v))
}

// TimestampEQ applies the EQ predicate on the "Timestamp" field.
func TimestampEQ(v int64) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldEQ(FieldTimestamp, v))
}

// TimestampNEQ applies the NEQ predicate on the "Timestamp" field.
func TimestampNEQ(v int64) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldNEQ(FieldTimestamp, v))
}

// TimestampIn applies the In predicate on the "Timestamp" field.
func TimestampIn(vs ...int64) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldIn(FieldTimestamp, vs...))
}

// TimestampNotIn applies the NotIn predicate on the "Timestamp" field.
func TimestampNotIn(vs ...int64) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldNotIn(FieldTimestamp, vs...))
}

// TimestampGT applies the GT predicate on the "Timestamp" field.
func TimestampGT(v int64) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldGT(FieldTimestamp, v))
}

// TimestampGTE applies the GTE predicate on the "Timestamp" field.
func TimestampGTE(v int64) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldGTE(FieldTimestamp, v))
}

// TimestampLT applies the LT predicate on the "Timestamp" field.
func TimestampLT(v int64) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldLT(FieldTimestamp, v))
}

// TimestampLTE applies the LTE predicate on the "Timestamp" field.
func TimestampLTE(v int64) predicate.AgentStatus {
	return predicate.AgentStatus(sql.FieldLTE(FieldTimestamp, v))
}

// HasAgentStatusToProvisionedHost applies the HasEdge predicate on the "AgentStatusToProvisionedHost" edge.
func HasAgentStatusToProvisionedHost() predicate.AgentStatus {
	return predicate.AgentStatus(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.Edge(sqlgraph.M2O, false, AgentStatusToProvisionedHostTable, AgentStatusToProvisionedHostColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasAgentStatusToProvisionedHostWith applies the HasEdge predicate on the "AgentStatusToProvisionedHost" edge with a given conditions (other predicates).
func HasAgentStatusToProvisionedHostWith(preds ...predicate.ProvisionedHost) predicate.AgentStatus {
	return predicate.AgentStatus(func(s *sql.Selector) {
		step := newAgentStatusToProvisionedHostStep()
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// HasAgentStatusToProvisionedNetwork applies the HasEdge predicate on the "AgentStatusToProvisionedNetwork" edge.
func HasAgentStatusToProvisionedNetwork() predicate.AgentStatus {
	return predicate.AgentStatus(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.Edge(sqlgraph.M2O, false, AgentStatusToProvisionedNetworkTable, AgentStatusToProvisionedNetworkColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasAgentStatusToProvisionedNetworkWith applies the HasEdge predicate on the "AgentStatusToProvisionedNetwork" edge with a given conditions (other predicates).
func HasAgentStatusToProvisionedNetworkWith(preds ...predicate.ProvisionedNetwork) predicate.AgentStatus {
	return predicate.AgentStatus(func(s *sql.Selector) {
		step := newAgentStatusToProvisionedNetworkStep()
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// HasAgentStatusToBuild applies the HasEdge predicate on the "AgentStatusToBuild" edge.
func HasAgentStatusToBuild() predicate.AgentStatus {
	return predicate.AgentStatus(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.Edge(sqlgraph.M2O, false, AgentStatusToBuildTable, AgentStatusToBuildColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasAgentStatusToBuildWith applies the HasEdge predicate on the "AgentStatusToBuild" edge with a given conditions (other predicates).
func HasAgentStatusToBuildWith(preds ...predicate.Build) predicate.AgentStatus {
	return predicate.AgentStatus(func(s *sql.Selector) {
		step := newAgentStatusToBuildStep()
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// And groups predicates with the AND operator between them.
func And(predicates ...predicate.AgentStatus) predicate.AgentStatus {
	return predicate.AgentStatus(sql.AndPredicates(predicates...))
}

// Or groups predicates with the OR operator between them.
func Or(predicates ...predicate.AgentStatus) predicate.AgentStatus {
	return predicate.AgentStatus(sql.OrPredicates(predicates...))
}

// Not applies the not operator on the given predicate.
func Not(p predicate.AgentStatus) predicate.AgentStatus {
	return predicate.AgentStatus(sql.NotPredicates(p))
}
