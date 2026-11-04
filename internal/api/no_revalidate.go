package api

import (
	"encoding/json"

	"github.com/globalcptc/laforge/internal/db"
)

// noRevalidateChecks counts a validate task's checks marked `revalidate: false`
// (no_revalidate in the rendered payload), keyed by checkKey, so the Steps tab
// can mark the matching results.
func noRevalidateChecks(payload []byte) map[string]int {
	var p struct {
		Checks []struct {
			Kind         string          `json:"kind"`
			Args         json.RawMessage `json:"args"`
			NoRevalidate bool            `json:"no_revalidate"`
		} `json:"checks"`
	}
	if json.Unmarshal(payload, &p) != nil {
		return nil
	}
	var out map[string]int
	for _, c := range p.Checks {
		if c.NoRevalidate {
			if out == nil {
				out = map[string]int{}
			}
			out[checkKey(c.Kind, c.Args)]++
		}
	}
	return out
}

// checkKey identifies a check by its kind and arguments, with the arguments
// re-encoded so key order and spacing don't matter (the agent echoes a check's
// arguments back in its result).
func checkKey(kind string, args json.RawMessage) string {
	var v interface{}
	if len(args) > 0 && json.Unmarshal(args, &v) == nil {
		if b, err := json.Marshal(v); err == nil {
			return kind + "\x00" + string(b)
		}
	}
	return kind + "\x00{}"
}

// markNoRevalidate flags each result whose check is marked revalidate: false,
// consuming one mark per match so two identical checks are told apart.
func markNoRevalidate(results []db.ValidatorResult, marks map[string]int) []bool {
	out := make([]bool, len(results))
	for i, r := range results {
		k := checkKey(r.Kind, r.Args)
		if marks[k] > 0 {
			marks[k]--
			out[i] = true
		}
	}
	return out
}
