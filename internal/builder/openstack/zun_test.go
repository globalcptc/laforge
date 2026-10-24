package openstack

import "testing"

func TestZunResources(t *testing.T) {
	if cpu, mem, ok := zunResources("1/1024"); !ok || cpu != 1 || mem != 1024 {
		t.Errorf("zunResources(cpu/mem) = %v/%d ok=%v, want 1/1024 true", cpu, mem, ok)
	}
	if cpu, mem, ok := zunResources("2.5/2048"); !ok || cpu != 2.5 || mem != 2048 {
		t.Errorf("zunResources(fractional cpu) = %v/%d ok=%v, want 2.5/2048 true", cpu, mem, ok)
	}
	// A Nova flavor id (not cpu/mem) yields ok=false -- let Zun default.
	if _, _, ok := zunResources("m1.small"); ok {
		t.Errorf("zunResources(flavor id) ok=true, want false")
	}
	if _, _, ok := zunResources(""); ok {
		t.Errorf("zunResources(empty) ok=true, want false")
	}
	if _, _, ok := zunResources("1/notanumber"); ok {
		t.Errorf("zunResources(bad memory) ok=true, want false")
	}
}

func TestCapsulePorts(t *testing.T) {
	ports := capsulePorts([]string{"80", "443", "bad"}, []string{"53"})
	if len(ports) != 3 {
		t.Fatalf("capsulePorts len = %d, want 3 (bad port dropped)", len(ports))
	}
	if ports[0]["containerPort"] != 80 || ports[0]["protocol"] != "TCP" {
		t.Errorf("first mapping = %v/%v, want 80/TCP", ports[0]["containerPort"], ports[0]["protocol"])
	}
	if ports[2]["containerPort"] != 53 || ports[2]["protocol"] != "UDP" {
		t.Errorf("udp mapping = %v/%v, want 53/UDP", ports[2]["containerPort"], ports[2]["protocol"])
	}
}
