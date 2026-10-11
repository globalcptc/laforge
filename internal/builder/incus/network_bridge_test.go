package incus

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/globalcptc/laforge/internal/builder"
)

func bridgeTestClient(t *testing.T, handle func(http.ResponseWriter, *http.Request)) *Client {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(handle))
	t.Cleanup(server.Close)
	return &Client{BaseURL: server.URL, HTTPClient: server.Client()}
}

func bridgeReply(w http.ResponseWriter, metadata any) {
	json.NewEncoder(w).Encode(map[string]any{"type": "sync", "status_code": 200, "metadata": metadata})
}

func TestDeployNativeBridge(t *testing.T) {
	for _, tc := range []struct{ name, extension, firewall, existing, wantErr string }{
		{name: "supported", extension: "network_bridge_acl_devices", firewall: "nftables", existing: "[]"},
		{name: "missing NIC ACL support", firewall: "nftables", existing: "[]", wantErr: "requires Incus"},
		{name: "wrong firewall", extension: "network_bridge_acl_devices", firewall: "xtables", existing: "[]", wantErr: "nftables"},
		{name: "overlapping bridge", extension: "network_bridge_acl_devices", firewall: "nftables", existing: `[{"name":"other-build","type":"bridge","managed":true,"config":{"ipv4.address":"10.60.0.1/16"}}]`, wantErr: "overlaps"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			creates := 0
			client := bridgeTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				switch {
				case r.Method == "GET" && r.URL.Path == "/1.0":
					bridgeReply(w, map[string]any{"api_extensions": []string{tc.extension}, "environment": map[string]string{"firewall": tc.firewall}})
				case r.Method == "GET" && r.URL.RequestURI() == "/1.0/networks?recursion=1":
					bridgeReply(w, json.RawMessage(tc.existing))
				case r.Method == "POST" && r.URL.Path == "/1.0/networks":
					creates++
					var request struct {
						Type   string
						Config map[string]string
					}
					if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
						t.Error(err)
					}
					if request.Type != "bridge" || request.Config["bridge.driver"] != "native" || request.Config["ipv4.address"] != "10.60.1.1/24" || request.Config["ipv4.nat"] != "true" || request.Config["ipv4.dhcp"] != "true" || request.Config["network"] != "" {
						t.Errorf("unexpected native bridge request: %+v", request)
					}
					bridgeReply(w, nil)
				default:
					t.Errorf("unexpected request %s %s", r.Method, r.URL)
					http.Error(w, "unexpected", 500)
				}
			})
			_, err := New(client, Config{NetworkType: "bridge"}).DeployNetwork(context.Background(), builder.NetworkSpec{ExternalName: "bridge-test", CIDR: "10.60.1.0/24"})
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("error = %v, want %s", err, tc.wantErr)
				}
				if creates != 0 {
					t.Fatal("invalid network reached create")
				}
			} else if err != nil || creates != 1 {
				t.Fatalf("create count %d, error %v", creates, err)
			}
		})
	}
}

func TestBridgeNetworkAccessDoesNotPeer(t *testing.T) {
	patches := 0
	client := bridgeTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PATCH" || !strings.HasPrefix(r.URL.Path, "/1.0/networks/") || strings.Contains(r.URL.Path, "peers") {
			t.Errorf("unexpected call %s %s", r.Method, r.URL)
		}
		patches++
		bridgeReply(w, nil)
	})
	err := New(client, Config{NetworkType: "bridge"}).ConfigureNetworkAccess(context.Background(), "1", []builder.NetworkAccess{
		{ExternalName: "a", CIDR: "10.60.1.0/24"}, {ExternalName: "b", CIDR: "10.60.2.0/24", VisibleFrom: []string{"a"}},
	})
	if err != nil || patches != 2 {
		t.Fatalf("patches %d, error %v", patches, err)
	}
}

func TestExternalAccessWaitsForEveryVMBeforeMutating(t *testing.T) {
	mutations := 0
	client := bridgeTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			mutations++
		}
		status := "Running"
		if strings.Contains(r.URL.Path, "second") {
			status = "Stopped"
		}
		bridgeReply(w, map[string]string{"status": status})
	})
	b := New(client, Config{NetworkType: "bridge", ExternalAccessIP: "203.0.113.10"})
	hosts := []builder.ExternalHost{{ExternalRef: "first", Address: "10.0.0.10", TCPPorts: []string{"3389"}}, {ExternalRef: "second", Address: "10.0.0.11", TCPPorts: []string{"22"}}}
	endpoints, err := b.ConfigureExternalAccess(context.Background(), "1", hosts)
	if !errors.Is(err, builder.ErrExternalAccessNotReady) || len(endpoints) != 0 || mutations != 0 {
		t.Fatalf("endpoints %v, mutations %d, error %v", endpoints, mutations, err)
	}
}

func TestRunningBridgePublicPortsPreservePrivateACL(t *testing.T) {
	ref := "lf-vm"
	nic := map[string]string{"type": "nic", "network": "lf-lan", "ipv4.address": "10.60.1.10", "security.acls": hostACLName(ref)}
	conflicts, reads, proxies := 0, 0, 0
	client := bridgeTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/1.0/instances/lf-vm/state":
			bridgeReply(w, map[string]string{"status": "Running"})
		case r.Method == "POST" && r.URL.Path == "/1.0/network-acls":
			var acl struct{ Ingress []map[string]string }
			json.NewDecoder(r.Body).Decode(&acl)
			if len(acl.Ingress) != 1 || acl.Ingress[0]["source"] != "0.0.0.0/0" || acl.Ingress[0]["destination_port"] != "3389" {
				t.Errorf("public rules = %+v", acl)
			}
			bridgeReply(w, nil)
		case r.Method == "GET" && r.URL.Path == "/1.0/instances/lf-vm":
			reads++
			w.Header().Set("ETag", `"revision"`)
			bridgeReply(w, map[string]any{"devices": map[string]any{"eth0": nic}})
		case r.Method == "PATCH" && r.URL.Path == "/1.0/instances/lf-vm":
			var patch struct{ Devices map[string]map[string]string }
			json.NewDecoder(r.Body).Decode(&patch)
			if updated, ok := patch.Devices["eth0"]; ok {
				if r.Header.Get("If-Match") != `"revision"` {
					t.Error("missing ETag precondition")
				}
				if conflicts == 0 {
					conflicts++
					nic["security.acls"] += ",concurrent-acl"
					w.WriteHeader(412)
					json.NewEncoder(w).Encode(map[string]any{"type": "error", "error_code": 412, "error": "ETag mismatch"})
					return
				}
				nic = updated
			} else {
				proxies++
				proxy := patch.Devices["lfx-tcp-3389"]
				if proxy["nat"] != "true" || proxy["connect"] != "tcp:10.60.1.10:3389" || proxy["listen"] != "tcp:203.0.113.10:40000" {
					t.Errorf("proxy = %+v", proxy)
				}
			}
			bridgeReply(w, nil)
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL)
			http.Error(w, "unexpected", 500)
		}
	})
	b := New(client, Config{NetworkType: "bridge", ExternalAccessIP: "203.0.113.10"})
	endpoints, err := b.ConfigureExternalAccess(context.Background(), "1", []builder.ExternalHost{{ExternalRef: ref, Address: "10.60.1.10", TCPPorts: []string{"3389"}}})
	if err != nil {
		t.Fatal(err)
	}
	if reads != 2 || proxies != 1 || len(endpoints) != 1 || endpoints[0].ExternalPort != "40000" {
		t.Fatalf("reads %d, proxies %d, endpoints %+v", reads, proxies, endpoints)
	}
	for _, name := range []string{hostACLName(ref), publicACLName(ref), "concurrent-acl"} {
		if !strings.Contains(nic["security.acls"], name) {
			t.Errorf("lost ACL %s in %+v", name, nic)
		}
	}
	if nic["ipv4.address"] != "10.60.1.10" || nic["network"] != "lf-lan" || nic["security.acls.default.egress.action"] != "allow" {
		t.Errorf("NIC corrupted: %+v", nic)
	}
}

func TestNetworkModeCompatibility(t *testing.T) {
	for _, tc := range []struct {
		config Config
		want   string
	}{
		{Config{}, "bridge"}, {Config{OVNUplinkNetwork: "UPLINK"}, "ovn"},
		{Config{NetworkType: "bridge", OVNUplinkNetwork: "stale-uplink"}, "bridge"},
		{Config{NetworkType: "ovn", OVNUplinkNetwork: "UPLINK"}, "ovn"},
	} {
		if got := tc.config.NetworkMode(); got != tc.want {
			t.Errorf("%+v -> %s, want %s", tc.config, got, tc.want)
		}
	}
}

func TestBridgeAdoptionValidatesExistingNetwork(t *testing.T) {
	for _, existingType := range []string{"bridge", "ovn"} {
		t.Run(existingType, func(t *testing.T) {
			client := bridgeTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				switch {
				case r.URL.Path == "/1.0":
					bridgeReply(w, map[string]any{"api_extensions": []string{"network_bridge_acl_devices"}, "environment": map[string]string{"firewall": "nftables"}})
				case r.Method == "GET" && r.URL.Path == "/1.0/networks":
					bridgeReply(w, []any{})
				case r.Method == "POST":
					w.WriteHeader(409)
					json.NewEncoder(w).Encode(map[string]any{"type": "error", "error_code": 409, "error": "Network already exists"})
				default:
					bridgeReply(w, map[string]any{"type": existingType, "config": map[string]string{"ipv4.address": "10.60.1.1/24", "bridge.driver": "native"}})
				}
			})
			_, err := New(client, Config{NetworkType: "bridge"}).DeployNetwork(context.Background(), builder.NetworkSpec{ExternalName: "adopt", CIDR: "10.60.1.0/24"})
			if existingType == "bridge" && err != nil {
				t.Fatal(err)
			}
			if existingType == "ovn" && (err == nil || !strings.Contains(err.Error(), "rebuild")) {
				t.Fatalf("accepted old OVN network: %v", err)
			}
		})
	}
}
