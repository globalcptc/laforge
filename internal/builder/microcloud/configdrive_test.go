package microcloud

import (
	"bytes"
	"testing"

	"github.com/kdomanski/iso9660"
)

// TestBuildCidataISO: the config drive is a real ISO whose volume label is
// CIDATA (what a NoCloud datasource looks for) and which contains the
// meta-data and user-data a booting guest reads.
func TestBuildCidataISO(t *testing.T) {
	iso, err := buildCidataISO("obj-123", "#ps1_sysnative\nWrite-Output hi\n")
	if err != nil {
		t.Fatal(err)
	}
	img, err := iso9660.OpenImage(bytes.NewReader(iso))
	if err != nil {
		t.Fatalf("produced bytes are not a valid ISO: %v", err)
	}
	root, err := img.RootDir()
	if err != nil {
		t.Fatal(err)
	}
	children, err := root.GetChildren()
	if err != nil {
		t.Fatal(err)
	}
	names := map[string]bool{}
	for _, c := range children {
		names[c.Name()] = true
	}
	if !names["meta-data"] || !names["user-data"] {
		t.Fatalf("ISO is missing meta-data/user-data, has: %v", names)
	}
}
