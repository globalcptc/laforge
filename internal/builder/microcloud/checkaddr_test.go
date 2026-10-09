package microcloud

import (
	"context"
	"strings"
	"testing"
)

// CheckAddresses reports every address -- the right server, an impostor, and
// one nothing answers on -- in order, and redeeming nothing leaves the token
// usable for a real Enroll afterwards.
func TestCheckAddressesReportsEachAddressWithoutRedeeming(t *testing.T) {
	f := newFakeEnrollServer(t)
	other := newFakeEnrollServer(t)
	tok, err := ParseTrustToken(f.token)
	if err != nil {
		t.Fatal(err)
	}
	good := strings.TrimPrefix(f.srv.URL, "https://")
	impostor := strings.TrimPrefix(other.srv.URL, "https://")
	got := CheckAddresses(context.Background(), []string{"127.0.0.1:1", impostor, good}, tok.Fingerprint)
	if len(got) != 3 || got[0].Address != "127.0.0.1:1" || got[1].Address != impostor || got[2].Address != good {
		t.Fatalf("results not in input order: %+v", got)
	}
	if got[0].Reachable || got[0].Error == "" {
		t.Errorf("unreachable address: %+v", got[0])
	}
	if !got[1].Reachable || got[1].FingerprintMatches || !strings.Contains(got[1].Error, "certificate") {
		t.Errorf("impostor: %+v", got[1])
	}
	if !got[2].Reachable || !got[2].FingerprintMatches || got[2].Error != "" {
		t.Errorf("real server: %+v", got[2])
	}
	if _, err := Enroll(context.Background(), f.token, "laforge-test", good); err != nil {
		t.Fatalf("Enroll after a check should still work (the token wasn't spent): %v", err)
	}
}

func TestTokenAddressesOverrideReplacesTheList(t *testing.T) {
	tok := TrustToken{Addresses: []string{"10.0.0.1:8443", "10.0.0.2:8443"}}
	if got := TokenAddresses(tok, ""); len(got) != 2 {
		t.Errorf("no override: %v", got)
	}
	if got := TokenAddresses(tok, "203.0.113.5"); len(got) != 1 || got[0] != "203.0.113.5:8443" {
		t.Errorf("override: %v", got)
	}
}

// When nothing is reachable, the error names each address and why.
func TestEnrollUnreachableNamesEachAddress(t *testing.T) {
	f := newFakeEnrollServer(t)
	_, err := Enroll(context.Background(), f.token, "laforge-test", "127.0.0.1:1")
	if err == nil || !strings.Contains(err.Error(), "127.0.0.1:1: ") {
		t.Fatalf("err = %v, want the address and its reason", err)
	}
}
