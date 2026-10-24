package builder

import (
	"context"
	"strings"
	"testing"
)

type stubOnboarder struct{}

func (stubOnboarder) Onboard(context.Context, OnboardRequest) (*Onboarding, error) { return &Onboarding{}, nil }
func (stubOnboarder) Rediscover(context.Context, Connection) (*Discovery, error)   { return &Discovery{}, nil }

func TestOnboarderRegistry(t *testing.T) {
	RegisterOnboarder("test-kind-registry", stubOnboarder{})

	if _, err := OnboarderFor("test-kind-registry"); err != nil {
		t.Fatalf("OnboarderFor(registered) = %v, want it found", err)
	}

	_, err := OnboarderFor("no-such-kind")
	if err == nil {
		t.Fatal("OnboarderFor(unknown) = nil error, want an error")
	}
	if !strings.Contains(err.Error(), "test-kind-registry") {
		t.Errorf("unknown-kind error should list known kinds, got: %v", err)
	}
}

func TestRegisterOnboarderRejectsDuplicate(t *testing.T) {
	RegisterOnboarder("test-kind-dup", stubOnboarder{})
	defer func() {
		if recover() == nil {
			t.Fatal("registering the same kind twice should panic")
		}
	}()
	RegisterOnboarder("test-kind-dup", stubOnboarder{})
}

func TestConnectionHasCredential(t *testing.T) {
	if (Connection{}).HasCredential() {
		t.Error("empty connection should have no credential (env-cred kinds)")
	}
	if !(Connection{APIURL: "https://host:8443"}).HasCredential() {
		t.Error("a connection with an API URL should report a credential")
	}
}
