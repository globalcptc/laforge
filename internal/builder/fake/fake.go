// Package fake is the fake builder: "Orchestrator +
// runner contract with a fake builder." Its "hoster" is a real Postgres
// table (fake_hoster_resource, migrations/00003), not in-process memory --
// deliberately, so that killing a runner mid-deploy can never lose what
// the hoster already has, the same way killing a process talking to a
// real cloud API wouldn't. That's what makes the chaos tests in
// internal/chaos a genuine proof of the resilience design rather than a
// tautology (a fake that forgets everything when its process dies would
// make convergence trivially unprovable).
package fake

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/globalcptc/laforge/internal/builder"
	"github.com/globalcptc/laforge/internal/db"
)

type Builder struct {
	Pool *pgxpool.Pool

	// WorkDelay simulates a real hoster API call taking real wall-clock
	// time -- long enough that a chaos test can reliably SIGKILL a runner
	// while a deploy/destroy call is genuinely in flight, not before it
	// even started.
	WorkDelay time.Duration
}

func New(pool *pgxpool.Pool) *Builder {
	return &Builder{Pool: pool}
}

func (b *Builder) wait(ctx context.Context) error {
	if b.WorkDelay <= 0 {
		return nil
	}
	select {
	case <-time.After(b.WorkDelay):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// ensure is every Deploy* call: wait (simulating real API latency), then
// INSERT ... ON CONFLICT DO UPDATE against the real table -- "create if
// missing, adopt if already there." Called twice for the same
// externalName (a retried task after a crashed runner) increments
// ensure_count but never creates a second row; internal/chaos asserts on
// exactly that.
func (b *Builder) ensure(ctx context.Context, externalName, kind string) (string, error) {
	if err := b.wait(ctx); err != nil {
		return "", err
	}
	q := db.New(b.Pool)
	row, err := q.EnsureFakeHosterResource(ctx, db.EnsureFakeHosterResourceParams{
		ExternalRef: externalName, Kind: kind,
	})
	if err != nil {
		return "", err
	}
	return row.ExternalRef, nil
}

func (b *Builder) destroy(ctx context.Context, externalRef string) error {
	if err := b.wait(ctx); err != nil {
		return err
	}
	q := db.New(b.Pool)
	// Destroying an already-destroyed (or never-existed) resource is a
	// no-op success, not an error -- matching "safe to call more than
	// once." A resource that was never deployed has no row at all, so
	// pgx.ErrNoRows is the expected, non-error outcome here.
	_, err := q.DestroyFakeHosterResource(ctx, externalRef)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	return nil
}

func (b *Builder) DeployNetwork(ctx context.Context, spec builder.NetworkSpec) (string, error) {
	return b.ensure(ctx, spec.ExternalName, "network")
}

func (b *Builder) DeployHost(ctx context.Context, spec builder.HostSpec) (string, error) {
	return b.ensure(ctx, spec.ExternalName, "host")
}

func (b *Builder) DeployContainer(ctx context.Context, spec builder.ContainerSpec) (string, error) {
	return b.ensure(ctx, spec.ExternalName, "container")
}

func (b *Builder) DestroyNetwork(ctx context.Context, team, externalRef string) error {
	return b.destroy(ctx, externalRef)
}

func (b *Builder) DestroyHost(ctx context.Context, team, externalRef string) error {
	return b.destroy(ctx, externalRef)
}

func (b *Builder) DestroyContainer(ctx context.Context, team, externalRef string) error {
	return b.destroy(ctx, externalRef)
}

// PowerAction is a no-op success on the fake hoster: there's no real guest
// to start, stop, or reboot, so the fake just reports the operation
// "succeeded" (after its usual simulated latency), which is enough for the
// UI/dispatch path to be exercised end to end locally.
func (b *Builder) PowerAction(ctx context.Context, team, externalRef, action string, force bool) error {
	return b.wait(ctx)
}

func (b *Builder) Inspect(ctx context.Context) ([]builder.Resource, error) {
	q := db.New(b.Pool)
	rows, err := q.ListFakeHosterResources(ctx)
	if err != nil {
		return nil, err
	}
	var out []builder.Resource
	for _, r := range rows {
		if r.Destroyed {
			continue
		}
		// The fake hoster has no real power state; anything not destroyed is
		// "running" (a network reports "" like the real builder does).
		state := builder.PowerStateRunning
		if r.Kind == "network" {
			state = ""
		}
		out = append(out, builder.Resource{ExternalRef: r.ExternalRef, Kind: r.Kind, State: state})
	}
	return out, nil
}

// OpenAccess and CloseAccess are no-ops here: the fake builder has no
// real network to open or close a team's ingress on. Present only to
// satisfy the interface -- a real builder's implementation is where
// "closing a team... terminate established connections" actually lives.
func (b *Builder) OpenAccess(ctx context.Context, team string) error  { return nil }
func (b *Builder) CloseAccess(ctx context.Context, team string) error { return nil }

// ConfigureNetworkAccess is a no-op for the in-memory fake: there is no real
// network fabric to peer or firewall.
func (b *Builder) ConfigureNetworkAccess(ctx context.Context, team string, networks []builder.NetworkAccess) error {
	return nil
}

// ConfigureExternalAccess fabricates endpoints without touching anything real:
// a deterministic synthetic public address per (host, port) so the whole
// pipeline (reconcile -> record -> surface) can be exercised end to end in
// tests. Models the shared-IP builders' port-NAT (one address, a remapped
// external port per entry) rather than the public-IP ones.
func (b *Builder) ConfigureExternalAccess(ctx context.Context, team string, hosts []builder.ExternalHost) ([]builder.ExternalEndpoint, error) {
	const fakeIP = "198.51.100.1" // TEST-NET-2, never a real address
	var out []builder.ExternalEndpoint
	port := 40000
	for _, h := range hosts {
		add := func(proto string, ports []string) {
			for _, p := range ports {
				out = append(out, builder.ExternalEndpoint{
					ExternalRef: h.ExternalRef, Protocol: proto, InternalPort: p,
					ExternalPort: strconv.Itoa(port), PublicAddress: fakeIP + ":" + strconv.Itoa(port),
				})
				port++
			}
		}
		add("tcp", h.TCPPorts)
		add("udp", h.UDPPorts)
	}
	return out, nil
}
