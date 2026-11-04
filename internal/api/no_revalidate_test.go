package api

import (
	"encoding/json"
	"testing"

	"github.com/globalcptc/laforge/internal/db"
)

// Results are matched to their payload checks by kind and arguments, whatever
// the key order, and each mark is used once, so of two identical checks only
// the marked one is flagged.
func TestMarkNoRevalidate(t *testing.T) {
	payload := []byte(`{"checks":[
		{"kind":"file_exists","args":{"value":"C:\\setup.exe"},"no_revalidate":true},
		{"kind":"port_listening","args":{"port":80,"proto":"tcp"}},
		{"kind":"port_listening","args":{"proto":"tcp","port":443},"no_revalidate":true},
		{"kind":"service_running","args":{"value":"sshd"}}
	]}`)
	results := []db.ValidatorResult{
		{Kind: "file_exists", Args: json.RawMessage(`{"value": "C:\\setup.exe"}`)},
		{Kind: "port_listening", Args: json.RawMessage(`{"port":80,"proto":"tcp"}`)},
		{Kind: "port_listening", Args: json.RawMessage(`{"port":443,"proto":"tcp"}`)},
		{Kind: "service_running", Args: json.RawMessage(`{"value":"sshd"}`)},
		{Kind: "file_exists", Args: json.RawMessage(`{"value":"C:\\setup.exe"}`)}, // a second identical result: mark already used
	}
	got := markNoRevalidate(results, noRevalidateChecks(payload))
	want := []bool{true, false, true, false, false}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("result %d (%s): marked %v, want %v", i, results[i].Kind, got[i], want[i])
		}
	}
	if m := noRevalidateChecks([]byte(`{"checks":[{"kind":"a","args":{}}]}`)); m != nil {
		t.Errorf("payload with no marks: got %v, want nil", m)
	}
}
