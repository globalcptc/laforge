package aws

import "testing"

func TestFargateSize(t *testing.T) {
	if cpu, mem := fargateSize("512/1024"); cpu != "512" || mem != "1024" {
		t.Errorf("fargateSize(cpu/mem) = %q/%q, want 512/1024", cpu, mem)
	}
	if cpu, mem := fargateSize("t3.medium"); cpu != "256" || mem != "512" {
		t.Errorf("fargateSize(instance type) = %q/%q, want default 256/512", cpu, mem)
	}
	if cpu, mem := fargateSize(""); cpu != "256" || mem != "512" {
		t.Errorf("fargateSize(empty) = %q/%q, want default 256/512", cpu, mem)
	}
}

func TestSanitizeECSName(t *testing.T) {
	if got := sanitizeECSName("web01.team-3"); got != "web01-team-3" {
		t.Errorf("sanitizeECSName dotted = %q", got)
	}
	if got := sanitizeECSName(""); got != "container" {
		t.Errorf("sanitizeECSName empty = %q, want container", got)
	}
	if got := sanitizeECSName("Ab-9_z"); got != "Ab-9_z" {
		t.Errorf("sanitizeECSName already-valid = %q", got)
	}
}

func TestPortMappings(t *testing.T) {
	pm := portMappings([]string{"80", "443", "bad"}, []string{"53"})
	if len(pm) != 3 {
		t.Fatalf("portMappings len = %d, want 3 (bad port dropped)", len(pm))
	}
	if *pm[0].ContainerPort != 80 || pm[0].Protocol != "tcp" {
		t.Errorf("first mapping = %d/%s, want 80/tcp", *pm[0].ContainerPort, pm[0].Protocol)
	}
	if *pm[2].ContainerPort != 53 || pm[2].Protocol != "udp" {
		t.Errorf("udp mapping = %d/%s, want 53/udp", *pm[2].ContainerPort, pm[2].Protocol)
	}
}
