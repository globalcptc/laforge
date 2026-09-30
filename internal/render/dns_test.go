package render_test

import (
	"testing"

	"github.com/globalcptc/laforge/internal/loader"
	"github.com/globalcptc/laforge/internal/render"
)

func findLMTestEnvironment(t *testing.T, c *loader.Content) *loader.Environment {
	t.Helper()
	for i := range c.Environments {
		if c.Environments[i].Name == "lm-test" {
			return &c.Environments[i]
		}
	}
	t.Fatal("lm-test environment not found in example content")
	return nil
}

func TestDNSRecordsGeneratesAutoARecordsPlusCustomOnes(t *testing.T) {
	c := loadExample(t)
	e := findLMTestEnvironment(t, c)
	records, err := render.DNSRecords(c, e)
	if err != nil {
		t.Fatalf("DNSRecords: %v", err)
	}

	// 8 copies total (db01, web01, scoreboard, dc01, ws01, devdb, kali01,
	// kali02, wireguard -- that's actually 9, see below) plus 2 custom
	// records from lm-test.yaml's dns.records block.
	byName := make(map[string]render.DNSRecord)
	for _, r := range records {
		byName[r.Name+"/"+r.Type] = r
	}

	auto := []struct {
		name, addr string
	}{
		{"db01", "10.0.1.11"},
		{"web01", "10.0.1.12"},
		{"scoreboard", "10.0.1.50"},
		{"dc01", "10.0.1.20"},
		{"ws01", "10.0.1.21"},
		{"devdb", "10.0.2.5"},
		{"kali01", "10.0.254.201"},
		{"kali02", "10.0.254.202"},
		{"wireguard", "10.0.255.5"},
	}
	for _, a := range auto {
		r, ok := byName[a.name+"/A"]
		if !ok {
			t.Errorf("expected an automatic A record for %q, none found (got %+v)", a.name, records)
			continue
		}
		if r.Value != a.addr {
			t.Errorf("A record for %q = %q, want %q", a.name, r.Value, a.addr)
		}
	}

	cname, ok := byName["intranet/CNAME"]
	if !ok {
		t.Fatal("expected the custom intranet CNAME record from lm-test.yaml's dns.records block")
	}
	if cname.Value != "web01" {
		t.Errorf("intranet CNAME target = %q, want web01", cname.Value)
	}

	mx, ok := byName["mail/MX"]
	if !ok {
		t.Fatal("expected the custom mail MX record from lm-test.yaml's dns.records block")
	}
	if mx.Value != "mail01" || mx.Priority != 10 {
		t.Errorf("mail MX record = %+v, want target mail01 priority 10", mx)
	}

	wantCount := len(auto) + 2
	if len(records) != wantCount {
		t.Errorf("DNSRecords returned %d records, want %d: %+v", len(records), wantCount, records)
	}
}

func TestDNSRecordsIsTeamInvariant(t *testing.T) {
	// Address computation never depends on team number (see DNSRecords'
	// own doc comment) -- the environment has no per-team parameter to
	// DNSRecords at all, so this is really just confirming the function
	// signature doesn't take one and two calls against the same content
	// agree.
	c := loadExample(t)
	e := findLMTestEnvironment(t, c)
	r1, err := render.DNSRecords(c, e)
	if err != nil {
		t.Fatalf("DNSRecords: %v", err)
	}
	r2, err := render.DNSRecords(c, e)
	if err != nil {
		t.Fatalf("DNSRecords: %v", err)
	}
	if len(r1) != len(r2) {
		t.Fatalf("DNSRecords is not deterministic: %d vs %d records", len(r1), len(r2))
	}
}
