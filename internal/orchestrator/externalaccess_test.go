package orchestrator

import (
	"testing"

	"github.com/globalcptc/laforge/internal/loader"
)

func contentWithPublic(public *loader.Ports) *loader.Content {
	return &loader.Content{Hosts: []loader.Host{{Name: "jumpbox", Ports: loader.Ports{TCP: []string{"3389", "3390"}}, Public: public}}}
}

func TestExternalAccessFingerprint(t *testing.T) {
	// No host declares public: -> empty (the reconciler then does nothing).
	if fp := externalAccessFingerprint(&loader.Content{}); fp != "" {
		t.Errorf("no public ports: fingerprint = %q, want empty", fp)
	}

	a := externalAccessFingerprint(contentWithPublic(&loader.Ports{TCP: []string{"3389"}}))
	if a == "" {
		t.Fatal("a public port should produce a non-empty fingerprint")
	}

	// Same public set -> same fingerprint (idempotent re-runs don't re-fire).
	if b := externalAccessFingerprint(contentWithPublic(&loader.Ports{TCP: []string{"3389"}})); a != b {
		t.Errorf("same public set gave different fingerprints: %q vs %q", a, b)
	}

	// A changed port -> a different fingerprint (a content edit re-fires).
	if c := externalAccessFingerprint(contentWithPublic(&loader.Ports{TCP: []string{"3390"}})); a == c {
		t.Error("changing the public port should change the fingerprint")
	}
}
