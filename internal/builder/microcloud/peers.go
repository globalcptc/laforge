package microcloud

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
)

// A team's networks are joined by OVN peers: one half-peer on each side of
// every pair, named for its target (peerName). LXD links the two halves when
// the second is created and only then routes between them; until then each
// half reads "Pending".
//
// A network rebuild can leave a stale half behind. Deleting a network removes
// its own halves, but its siblings' halves pointing at it survive, still tied to
// the deleted network. When the network comes back under the same name, creating
// the sibling half again returns "already exists", and the new network's own
// half never links: it sits Pending and no traffic crosses (cptc12-se team 2,
// 2026-11-04). ensurePeerMesh checks every pair after creating it and rebuilds
// any that didn't link; deleteReversePeers stops teardown leaving those stale
// halves in the first place.

type networkPeer struct {
	Name          string `json:"name"`
	TargetNetwork string `json:"target_network"`
	Status        string `json:"status"`
}

// ensurePeerMesh full-mesh peers the given OVN networks and checks every pair
// actually linked, rebuilding a pair that didn't. Returns an error naming any
// pair still unlinked after the rebuild, so a broken mesh fails the task
// instead of reporting it applied.
func (b *Builder) ensurePeerMesh(ctx context.Context, nets []string) error {
	for _, a := range nets {
		for _, c := range nets {
			if a == c {
				continue
			}
			if err := b.ensurePeer(ctx, a, c); err != nil {
				return fmt.Errorf("peering %s -> %s: %w", a, c, err)
			}
		}
	}
	for i, a := range nets {
		for _, c := range nets[i+1:] {
			if ok, err := b.peerPairLinked(ctx, a, c); err != nil {
				return err
			} else if ok {
				continue
			}
			slog.Warn("MicroCloud network peer pair is not linked; recreating both halves", "a", a, "b", c)
			b.deletePeer(ctx, a, c)
			b.deletePeer(ctx, c, a)
			if err := b.ensurePeer(ctx, a, c); err != nil {
				return fmt.Errorf("re-peering %s -> %s: %w", a, c, err)
			}
			if err := b.ensurePeer(ctx, c, a); err != nil {
				return fmt.Errorf("re-peering %s -> %s: %w", c, a, err)
			}
			ok, err := b.peerPairLinked(ctx, a, c)
			if err != nil {
				return err
			}
			if !ok {
				return fmt.Errorf("network peers %s <-> %s did not link after recreating both halves", a, c)
			}
		}
	}
	return nil
}

// peerPairLinked reports whether both halves of the a<->b pair exist, target
// each other, and are Created.
func (b *Builder) peerPairLinked(ctx context.Context, a, c string) (bool, error) {
	for _, p := range [][2]string{{a, c}, {c, a}} {
		peer, err := b.getPeer(ctx, p[0], p[1])
		if err != nil {
			return false, fmt.Errorf("reading peer %s -> %s: %w", p[0], p[1], err)
		}
		if peer == nil || peer.TargetNetwork != p[1] || !strings.EqualFold(peer.Status, "Created") {
			return false, nil
		}
	}
	return true, nil
}

// getPeer reads the half-peer on `from` that targets `to`; nil if it's absent.
func (b *Builder) getPeer(ctx context.Context, from, to string) (*networkPeer, error) {
	raw, err := b.Client.get(ctx, "/1.0/networks/"+from+"/peers/"+peerName(to))
	if err != nil {
		var apiErr *APIError
		if errors.As(err, &apiErr) && apiErr.NotFound() {
			return nil, nil
		}
		return nil, err
	}
	var p networkPeer
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

// deletePeer removes the half-peer on `from` that targets `to`. Best-effort.
func (b *Builder) deletePeer(ctx context.Context, from, to string) {
	b.Client.delete(ctx, "/1.0/networks/"+from+"/peers/"+peerName(to))
}

// deleteReversePeers removes the siblings' halves pointing at a network about
// to be deleted, found through the network's own halves. Run before those own
// halves are removed. Best-effort.
func (b *Builder) deleteReversePeers(ctx context.Context, net string) {
	raw, err := b.Client.get(ctx, "/1.0/networks/"+net+"/peers?recursion=1")
	if err != nil {
		return
	}
	var peers []networkPeer
	if json.Unmarshal(raw, &peers) != nil {
		return
	}
	for _, p := range peers {
		if p.TargetNetwork != "" {
			b.deletePeer(ctx, p.TargetNetwork, net)
		}
	}
}
