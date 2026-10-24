package orchestrator

import (
	"github.com/globalcptc/laforge/internal/db"
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

func TestExternalAccessFingerprintTracksReadyPublicTargets(t *testing.T) {
	c := contentWithPublic(&loader.Ports{TCP: []string{"3389"}})
	fp := externalAccessFingerprint(c)
	objs := []db.DeployedObject{
		{Kind: "host", ObjectName: "private", Status: "running", ExternalRef: db.StrPtr("private-ref")},
		{Kind: "host", ObjectName: "jumpbox", Status: "creating", ExternalRef: db.StrPtr("public-ref")},
	}
	if _, ready := externalAccessTeamFingerprint(c, objs, fp, false); ready {
		t.Fatal("private VM or unfinished create consumed public task")
	}
	objs[1].Status = "running"
	first, ready := externalAccessTeamFingerprint(c, objs, fp, false)
	if !ready {
		t.Fatal("ready public VM did not queue forwarding")
	}
	objs[1].Status = "finished"
	stable, _ := externalAccessTeamFingerprint(c, objs, fp, false)
	if stable != first {
		t.Fatal("guest step progress changed fingerprint")
	}
	objs = append(objs, db.DeployedObject{Kind: "host", ObjectName: "jumpbox", Status: "building", ExternalRef: db.StrPtr("second-public-ref")})
	second, ready := externalAccessTeamFingerprint(c, objs, fp, false)
	if !ready || second == first {
		t.Fatal("later public VM did not trigger forwarding")
	}
	objs[1].ExternalRef = db.StrPtr("rebuilt-public-ref")
	rebuilt, _ := externalAccessTeamFingerprint(c, objs, fp, false)
	if rebuilt == second {
		t.Fatal("replacement VM did not trigger forwarding")
	}
}
