package builderconfig

import "testing"

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
