package microcloud

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

const leakedPort = "lxd-net6167-instance-90dd46b4-47ac-4816-8084-b025305fbf43-eth0"

// fakeLeftovers answers creates with OVN's duplicate-address refusal and
// lists instances with the given volatile.uuid owners.
func fakeLeftovers(t *testing.T, owners map[string]string, deleteErr string) (*Client, *int) {
	t.Helper()
	creates := 0
	reply := func(w http.ResponseWriter, md interface{}) {
		json.NewEncoder(w).Encode(map[string]interface{}{"type": "sync", "status_code": 200, "metadata": md})
	}
	fail := func(w http.ResponseWriter, code int, msg string) {
		w.WriteHeader(code)
		json.NewEncoder(w).Encode(map[string]interface{}{"type": "error", "error_code": code, "error": msg})
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/1.0/instances":
			creates++
			fail(w, 500, `Failed creating instance record: Failed initialising instance: Failed to add device "eth0": Failed setting up OVN port: Failed to run: ovn-nbctl lsp-add lxd-net6167-ls-int lxd-net6167-instance-8f244a62-4c1b-43d8-9b6c-18cf5ea0dbc0-eth0: exit status 1 (ovn-nbctl: Error on switch lxd-net6167-ls-int: duplicate IPv4 address '10.0.1.12' found on logical switch port '`+leakedPort+`')`)
		case r.Method == http.MethodGet && r.URL.Path == "/1.0/instances":
			var list []map[string]interface{}
			for name, uuid := range owners {
				list = append(list, map[string]interface{}{"name": name, "config": map[string]string{"volatile.uuid": uuid}})
			}
			reply(w, list)
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/1.0/instances/"):
			if deleteErr == "" {
				fail(w, 404, "Instance not found")
				return
			}
			name := strings.TrimPrefix(r.URL.Path, "/1.0/instances/")
			reply(w, map[string]interface{}{"name": name, "config": map[string]string{}, "devices": map[string]interface{}{}})
		case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/1.0/instances/") && deleteErr != "":
			fail(w, 404, deleteErr)
		default:
			reply(w, map[string]interface{}{})
		}
	}))
	t.Cleanup(srv.Close)
	return &Client{BaseURL: srv.URL, HTTPClient: srv.Client(), OperationTimeout: 5, Project: "CPTC-Comp-Official"}, &creates
}

// A leaked OVN port fails the deploy permanently, once, with the command that
// removes it -- not a retry loop that can never succeed.
func TestLeakedOVNPortIsPermanentWithCleanup(t *testing.T) {
	c, creates := fakeLeftovers(t, nil, "")
	_, err := staleTestBuilder(c).DeployHost(context.Background(), staleSpec)
	if !errors.Is(err, builder.ErrPermanentConfig) {
		t.Fatalf("err = %v, want a permanent error", err)
	}
	for _, want := range []string{"microovn.ovn-nbctl lsp-del " + leakedPort, "90dd46b4-47ac-4816-8084-b025305fbf43", "10.0.1.12"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error doesn't mention %q: %v", want, err)
		}
	}
	if *creates != 1 {
		t.Errorf("creates = %d, want 1", *creates)
	}
}

// When a live instance still owns the port, the address is really taken:
// name that instance instead of suggesting the port be deleted.
func TestDuplicateAddressNamesTheOwningInstance(t *testing.T) {
	c, _ := fakeLeftovers(t, map[string]string{"lf-old-minetrain-123abc": "90dd46b4-47ac-4816-8084-b025305fbf43"}, "")
	_, err := staleTestBuilder(c).DeployHost(context.Background(), staleSpec)
	if !errors.Is(err, builder.ErrPermanentConfig) || !strings.Contains(err.Error(), "lf-old-minetrain-123abc") {
		t.Fatalf("err = %v, want a permanent error naming the owning instance", err)
	}
	if strings.Contains(err.Error(), "lsp-del") {
		t.Errorf("suggested deleting a port a live instance owns: %v", err)
	}
}

// A record LXD can't delete ("Instance storage pool not found") is not
// "already gone": the destroy fails permanently and says how to remove it.
func TestDestroyReportsBrokenInstanceRecord(t *testing.T) {
	c, _ := fakeLeftovers(t, nil, "Failed getting instance pool: Instance storage pool not found")
	err := staleTestBuilder(c).destroyInstance(context.Background(), "lf-t5-vdi-kali-02-a7744f")
	if !errors.Is(err, builder.ErrPermanentConfig) {
		t.Fatalf("err = %v, want a permanent error", err)
	}
	for _, want := range []string{"lxd sql global", "name='lf-t5-vdi-kali-02-a7744f'", "custom/cidata-lf-t5-vdi-kali-02-a7744f"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error doesn't mention %q: %v", want, err)
		}
	}
}

// A volume collision under a name LXD still has an instance for points at
// that record, not at Ceph.
func TestKnownInstanceCollisionDoesNotBlameCeph(t *testing.T) {
	f, c := newFakeStaleCeph(t, "")
	base := instanceName(staleSpec.DisplayName, staleSpec.ExternalName, "")
	f.known[base] = true
	_, err := staleTestBuilder(c).withFreshNameOnStaleStorage(context.Background(), staleSpec.DisplayName, staleSpec.ExternalName, func(ctx context.Context) (string, error) {
		return base, errors.New(`Cannot create volume "cidata-x", volume already exists on storage pool "p"`)
	})
	if err == nil || strings.Contains(err.Error(), "rbd rm") || !strings.Contains(err.Error(), "instance record "+base) {
		t.Fatalf("err = %v, want it to point at the LXD record, not Ceph", err)
	}
}
