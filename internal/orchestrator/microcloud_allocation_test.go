package orchestrator

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/globalcptc/laforge/internal/builder"
	"github.com/globalcptc/laforge/internal/builder/microcloud"
	"github.com/globalcptc/laforge/internal/db"
	"github.com/globalcptc/laforge/internal/runner"
)

// Exercise the real MicroCloud -> runner -> reconciliation path: a duplicate
// must fail the network task on its first attempt, retain the teardown reference,
// and prevent the dependent guests from being queued or bootstrapped.
func TestMicrocloudDuplicateAllocationBlocksGuests(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	q := db.New(pool)
	build := newTestBuild(t, pool, "microcloud-duplicate-allocation")
	networks := map[string]map[string]any{}
	instanceRequests := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reply := func(value any) { _ = json.NewEncoder(w).Encode(map[string]any{"type": "sync", "metadata": value}) }
		switch {
		case r.Method == "POST" && r.URL.Path == "/1.0/networks":
			var n map[string]any
			if err := json.NewDecoder(r.Body).Decode(&n); err != nil {
				t.Error(err)
				return
			}
			n["config"].(map[string]any)["volatile.network.ipv4.address"] = "10.10.44.35"
			n["status"], n["managed"], n["project"] = "Created", true, "default"
			networks[n["name"].(string)] = n
			reply(nil)
		case r.Method == "GET" && strings.HasPrefix(r.URL.Path, "/1.0/networks/"):
			if n, ok := networks[strings.TrimPrefix(r.URL.Path, "/1.0/networks/")]; ok {
				reply(n)
				return
			}
			w.WriteHeader(404)
			_ = json.NewEncoder(w).Encode(map[string]any{"type": "error", "error_code": 404, "error": "not found"})
		case r.URL.Path == "/1.0/networks":
			all := []map[string]any{{"name": "manual-network", "project": "default", "type": "ovn", "managed": true, "status": "Created", "config": map[string]string{"network": "UPLINK", "volatile.network.ipv4.address": "10.10.44.35"}}}
			for _, n := range networks {
				all = append(all, n)
			}
			reply(all)
		default:
			instanceRequests++
			t.Errorf("unexpected guest/bootstrap request: %s %s", r.Method, r.URL)
			reply(nil)
		}
	}))
	defer srv.Close()
	b := microcloud.New(&microcloud.Client{BaseURL: srv.URL, HTTPClient: srv.Client(), NetworkLock: microcloud.SQLNetworkLocker{Pool: pool}}, microcloud.Config{OVNUplinkNetwork: "UPLINK"})
	r := &runner.Runner{Pool: pool, Builder: b, RepoRoot: "../../examples/lm-test", ID: "duplicate-allocation-runner", BuildID: &build.ID, LeaseDuration: time.Minute, HeartbeatInterval: time.Second}
	if err := Reconcile(ctx, pool, r.RepoRoot, build.ID); err != nil {
		t.Fatal(err)
	}
	for i := 0; ; i++ {
		if i > 100 {
			t.Fatal("runner did not drain")
		}
		worked, err := r.LeaseAndExecuteOne(ctx)
		if err != nil && !errors.Is(err, builder.ErrPermanentConfig) {
			t.Fatal(err)
		}
		if !worked {
			break
		}
	}
	if err := Reconcile(ctx, pool, r.RepoRoot, build.ID); err != nil {
		t.Fatal(err)
	}
	objects, err := q.ListDeployedObjectsByBuild(ctx, build.ID)
	if err != nil {
		t.Fatal(err)
	}
	networkCount, guestCount := 0, 0
	for _, obj := range objects {
		if obj.Status != "deploy_failed" {
			t.Errorf("%s/%s status=%s", obj.Kind, obj.ObjectName, obj.Status)
		}
		if obj.Kind == "network" {
			networkCount++
			if db.StrOrEmpty(obj.ExternalRef) == "" {
				t.Error("failed network lost teardown reference")
			}
			if !strings.Contains(db.StrOrEmpty(obj.LastError), "10.10.44.35") {
				t.Errorf("missing conflict diagnostic: %v", obj.LastError)
			}
		} else {
			guestCount++
		}
	}
	if networkCount == 0 || guestCount == 0 || instanceRequests != 0 {
		t.Fatalf("networks=%d guests=%d instance requests=%d", networkCount, guestCount, instanceRequests)
	}
	var guestTasks, retriedNetworks int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM task WHERE build_id=$1 AND kind IN ('deploy_host','deploy_container')`, build.ID).Scan(&guestTasks); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM task WHERE build_id=$1 AND kind='deploy_network' AND (attempts<>1 OR status<>'failed')`, build.ID).Scan(&retriedNetworks); err != nil {
		t.Fatal(err)
	}
	if guestTasks != 0 || retriedNetworks != 0 {
		t.Fatalf("guest tasks=%d tasks not failed on first attempt=%d", guestTasks, retriedNetworks)
	}
	if settled, err := AdvanceBuildStatus(ctx, pool, build.ID); err != nil || !settled {
		t.Fatalf("failed build did not settle: settled=%v err=%v", settled, err)
	}
}
