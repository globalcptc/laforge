package db

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// Interval converts a Go duration into the pgtype.Interval sqlc generates
// for a `$n::interval` parameter (LeaseTask, HeartbeatTask). Only
// Microseconds is set -- fine for the lease durations this package
// actually deals with (seconds to minutes), no need for the Days/Months
// fields pgtype.Interval also carries.
func Interval(d time.Duration) pgtype.Interval {
	return pgtype.Interval{Microseconds: d.Microseconds(), Valid: true}
}

// StrPtr is the same "empty string means SQL NULL" convention
// internal/ingest already uses, exported here since internal/orchestrator
// and internal/runner both need it for the same nullable text columns
// (external_ref, last_error, as_name, network_name, ...).
func StrPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// StrOrEmpty is StrPtr's inverse, for reading a nullable column back as a
// plain string when "" and NULL are equally fine to treat the same way.
func StrOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// Int32OrZero reads a nullable int4 column back as a plain int32, NULL as 0.
func Int32OrZero(n *int32) int32 {
	if n == nil {
		return 0
	}
	return *n
}
