package builderconfig

import (
	"testing"

	"github.com/globalcptc/laforge/internal/db"
)

// The wizard stores a cloud builder's maps in the shared Incus image/size
// columns: the image id in ImageRef.fingerprint, the instance type in
// SizeSpec.type. These prove the cloud resolver extracts them (the exact
// impedance the draft builders lacked before).
func TestCloudImages(t *testing.T) {
	got := cloudImages([]byte(`{"ubuntu22":{"fingerprint":"ami-0abc123","alias":"","vm":false},"win2019":{"fingerprint":"","alias":"ami-win"}}`))
	if got["ubuntu22"] != "ami-0abc123" {
		t.Fatalf("ubuntu22 = %q, want ami-0abc123", got["ubuntu22"])
	}
	if got["win2019"] != "ami-win" { // falls back to alias when fingerprint empty
		t.Fatalf("win2019 = %q, want ami-win (alias fallback)", got["win2019"])
	}
}

func TestCloudSizes(t *testing.T) {
	got := cloudSizes([]byte(`{"small":{"cpu":"","memory":"","type":"t3.medium"},"tiny":{"cpu":"1","memory":"1GiB"}}`))
	if got["small"] != "t3.medium" {
		t.Fatalf("small = %q, want t3.medium", got["small"])
	}
	if _, ok := got["tiny"]; ok { // no type -> skipped, not mapped to ""
		t.Fatalf("tiny should be skipped (no instance type), got %q", got["tiny"])
	}
}

// resolveAWS reads the per-builder aws_config JSONB; a valid blob resolves and a
// malformed one is a clear error (not a silent env-only fallback).
func TestResolveAWSConfig(t *testing.T) {
	row := db.BuilderConfig{Name: "aws1", Kind: "aws", AwsConfig: []byte(`{"region":"us-west-2","ecs_cluster":"lab"}`)}
	if _, err := resolveAWS(nil, row); err != nil {
		t.Fatalf("resolveAWS with valid aws_config: %v", err)
	}
	bad := db.BuilderConfig{Name: "aws2", Kind: "aws", AwsConfig: []byte(`{not json`)}
	if _, err := resolveAWS(nil, bad); err == nil {
		t.Fatal("resolveAWS should error on malformed aws_config")
	}
	// Empty aws_config is fine (env fallbacks apply).
	if _, err := resolveAWS(nil, db.BuilderConfig{Name: "aws3", Kind: "aws"}); err != nil {
		t.Fatalf("resolveAWS with no aws_config: %v", err)
	}
}
