package incus

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// fakePeers models LXD's OVN peering: a half-peer is Created once its partner
// half exists and links to it, Pending otherwise. A stale half (left on a
// sibling after its target was deleted) never links, like the real one.
type fakePeers struct {
	mu    sync.Mutex
	peers map[string]map[string]*fakePeer // network -> peer name -> half
}

type fakePeer struct {
	target string
	stale  bool
}

func (f *fakePeers) status(net, name string) string {
	p := f.peers[net][name]
	if p.stale {
		return "Created" // what a stale half reports: it looks fine from its own side
	}
	back := f.peers[p.target][peerName(net)]
	if back != nil && !back.stale && back.target == net {
		return "Created"
	}
	return "Pending"
}

func (f *fakePeers) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/1.0/networks/"), "/")
	net := parts[0]
	if len(parts) < 2 || parts[1] != "peers" {
		networkReply(w, nil) // PATCHes of network config
		return
	}
	if f.peers[net] == nil {
		f.peers[net] = map[string]*fakePeer{}
	}
	switch {
	case r.Method == "POST":
		var body struct {
			Name          string `json:"name"`
			TargetNetwork string `json:"target_network"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		if f.peers[net][body.Name] != nil {
			networkFailure(w, http.StatusConflict, "Failed creating peer: A peer for that name already exists")
			return
		}
		f.peers[net][body.Name] = &fakePeer{target: body.TargetNetwork}
		networkReply(w, nil)
	case r.Method == "DELETE" && len(parts) == 3:
		delete(f.peers[net], parts[2])
		networkReply(w, nil)
	case r.Method == "GET" && len(parts) == 3:
		p := f.peers[net][parts[2]]
		if p == nil {
			networkFailure(w, http.StatusNotFound, "Network peer not found")
			return
		}
		networkReply(w, networkPeer{Name: parts[2], TargetNetwork: p.target, Status: f.status(net, parts[2])})
	case r.Method == "GET" && r.URL.Query().Get("recursion") == "1":
		var out []networkPeer
		for name, p := range f.peers[net] {
			out = append(out, networkPeer{Name: name, TargetNetwork: p.target, Status: f.status(net, name)})
		}
		networkReply(w, out)
	case r.Method == "GET":
		var urls []string
		for name := range f.peers[net] {
			urls = append(urls, "/1.0/networks/"+net+"/peers/"+name)
		}
		networkReply(w, urls)
	default:
		networkFailure(w, http.StatusBadRequest, "unexpected "+r.Method+" "+r.URL.Path)
	}
}

func (f *fakePeers) linked(t *testing.T, nets ...string) {
	t.Helper()
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, a := range nets {
		for _, c := range nets {
			if a == c {
				continue
			}
			p := f.peers[a][peerName(c)]
			if p == nil || p.stale || f.status(a, peerName(c)) != "Created" {
				t.Errorf("peer %s -> %s is not linked: %+v", a, c, p)
			}
		}
	}
}

func newPeerBuilder(t *testing.T) (*Builder, *fakePeers) {
	f := &fakePeers{peers: map[string]map[string]*fakePeer{}}
	srv := httptest.NewServer(f)
	t.Cleanup(srv.Close)
	return &Builder{Client: &Client{BaseURL: srv.URL, HTTPClient: srv.Client()}, Config: Config{NetworkType: "ovn"}}, f
}

func networkReply(w http.ResponseWriter, metadata any) {
	_ = json.NewEncoder(w).Encode(map[string]any{"type": "sync", "metadata": metadata})
}

func networkFailure(w http.ResponseWriter, code int, message string) {
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]any{"type": "error", "error_code": code, "error": message})
}

func TestEnsurePeerMeshLinksFreshNetworks(t *testing.T) {
	b, f := newPeerBuilder(t)
	if err := b.ensurePeerMesh(context.Background(), []string{"corp", "prod", "vdi"}); err != nil {
		t.Fatal(err)
	}
	f.linked(t, "corp", "prod", "vdi")
}

// The cptc12-se team 2 case: vdi was deleted and recreated under the same name,
// and corp and prod still held their old halves pointing at it. Creating those
// halves again returns "already exists"; the mesh must notice the pair didn't
// link and rebuild it.
func TestEnsurePeerMeshRepairsStaleHalfAfterRebuild(t *testing.T) {
	b, f := newPeerBuilder(t)
	f.peers["corp"] = map[string]*fakePeer{peerName("prod"): {target: "prod"}, peerName("vdi"): {target: "vdi", stale: true}}
	f.peers["prod"] = map[string]*fakePeer{peerName("corp"): {target: "corp"}, peerName("vdi"): {target: "vdi", stale: true}}
	if err := b.ensurePeerMesh(context.Background(), []string{"corp", "prod", "vdi"}); err != nil {
		t.Fatal(err)
	}
	f.linked(t, "corp", "prod", "vdi")
}

// Teardown must take the siblings' halves pointing at the dying network with
// it, so a later network of the same name links cleanly.
func TestTeardownRemovesReversePeers(t *testing.T) {
	b, f := newPeerBuilder(t)
	ctx := context.Background()
	if err := b.ensurePeerMesh(ctx, []string{"corp", "prod", "vdi"}); err != nil {
		t.Fatal(err)
	}
	b.teardownNetworkAccess(ctx, "vdi")
	for _, sib := range []string{"corp", "prod"} {
		if p := f.peers[sib][peerName("vdi")]; p != nil {
			t.Errorf("%s still has a half-peer to the torn-down vdi: %+v", sib, p)
		}
	}
	if len(f.peers["vdi"]) != 0 {
		t.Errorf("vdi still has its own halves: %v", f.peers["vdi"])
	}
	f.linked(t, "corp", "prod")
}
