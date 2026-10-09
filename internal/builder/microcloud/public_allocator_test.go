package microcloud

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

func testPublicConfig() PublicAccessConfig {
	return PublicAccessConfig{Type: "nic", Network: "GUEST_GUAC_WAN", CIDR: "10.250.0.0/16", Ranges: "10.250.3.100-10.250.3.199", DNS: []string{"10.250.0.1"}, MTU: 1500}
}

func TestPublicConfigValidation(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*PublicAccessConfig)
	}{
		{"empty network", func(c *PublicAccessConfig) { c.Network = "" }},
		{"unknown type", func(c *PublicAccessConfig) { c.Type = "bridge" }},
		{"subnet host bits", func(c *PublicAccessConfig) { c.CIDR = "10.250.3.1/16" }},
		{"outside subnet", func(c *PublicAccessConfig) { c.Ranges = "192.0.2.10" }},
		{"network address", func(c *PublicAccessConfig) { c.Ranges = "10.250.0.0" }},
		{"broadcast address", func(c *PublicAccessConfig) { c.Ranges = "10.250.255.255" }},
		{"gateway in pool", func(c *PublicAccessConfig) { c.Gateway = "10.250.3.100" }},
		{"DNS in pool", func(c *PublicAccessConfig) { c.DNS = []string{"10.250.3.101"} }},
		{"route gateway in pool", func(c *PublicAccessConfig) { c.Routes = []PublicRoute{{To: "192.0.2.0/24", Via: "10.250.3.101"}} }},
		{"off-link gateway", func(c *PublicAccessConfig) { c.Gateway = "192.0.2.1" }},
		{"bad DNS", func(c *PublicAccessConfig) { c.DNS = []string{"bad"} }},
		{"bad MTU", func(c *PublicAccessConfig) { c.MTU = 100 }},
		{"empty pool", func(c *PublicAccessConfig) { c.Ranges = " , " }},
		{"proxy with NIC settings", func(c *PublicAccessConfig) { c.Type = "proxy" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := testPublicConfig()
			tc.change(&cfg)
			if err := cfg.Validate(); err == nil {
				t.Fatal("accepted invalid config")
			}
		})
	}
	cfg := testPublicConfig()
	cfg.Routes = []PublicRoute{{To: "192.0.2.0/24", Via: "10.250.0.1"}}
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
	if err := (PublicAccessConfig{}).Validate(); err != nil {
		t.Fatal(err)
	}
	if ips, err := parsePublicIPs("10.250.3.0-10.250.10.255"); err != nil || len(ips) != 2048 {
		t.Fatalf("large pool: %d, %v", len(ips), err)
	}
}

func TestPublicAllocationConcurrentAndDurable(t *testing.T) {
	dsn := os.Getenv("LAFORGE_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set LAFORGE_TEST_DATABASE_URL to a migrated test database")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	var id pgtype.UUID
	if err := pool.QueryRow(ctx, `INSERT INTO builder_config (name,kind) VALUES ($1,'microcloud') RETURNING id`, fmt.Sprintf("public-nic-test-%d", os.Getpid())).Scan(&id); err != nil {
		t.Fatal(err)
	}
	defer func() {
		pool.Exec(ctx, `DELETE FROM microcloud_public_address WHERE builder_id=$1`, id)
		pool.Exec(ctx, `DELETE FROM builder_config WHERE id=$1`, id)
	}()
	cfg := testPublicConfig()
	cfg.Ranges = "10.250.3.100-10.250.3.119"
	const count = 20
	addresses := make([]string, count)
	errs := make([]error, count)
	var wg sync.WaitGroup
	for i := range addresses {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			a := SQLPublicAddressAllocator{Pool: pool, BuilderID: id}
			addresses[i], errs[i] = a.Reserve(ctx, "project", fmt.Sprintf("build-copy-%d", i), fmt.Sprintf("owner-%d", i), cfg, false)
		}(i)
	}
	wg.Wait()
	seen := map[string]bool{}
	for i, ip := range addresses {
		if errs[i] != nil {
			t.Fatal(errs[i])
		}
		if seen[ip] {
			t.Fatalf("duplicate address %s", ip)
		}
		seen[ip] = true
	}
	a := SQLPublicAddressAllocator{Pool: pool, BuilderID: id}
	if _, err := a.Reserve(ctx, "project", "overflow", "overflow-owner", cfg, false); err == nil {
		t.Fatal("exhausted pool accepted another box")
	}
	for i, ip := range addresses {
		again, err := a.Reserve(ctx, "project", fmt.Sprintf("build-copy-%d", i), fmt.Sprintf("owner-%d", i), cfg, false)
		if err != nil || again != ip {
			t.Fatalf("retry moved address: %s -> %s (%v)", ip, again, err)
		}
	}
	if _, err := a.Reserve(ctx, "project", "build-copy-0", "other-build-same-name", cfg, false); err == nil {
		t.Fatal("a colliding hoster name shared another build's IP")
	}
	if err := a.Release(ctx, "project", "build-copy-0", "other-build-same-name"); err != nil {
		t.Fatal(err)
	}
	if _, err := a.Reserve(ctx, "project", "overflow", "overflow-owner", cfg, false); err == nil {
		t.Fatal("a colliding deployment released another build's IP")
	}
	// A config edit cannot silently replace an attached address.
	changed := cfg
	changed.Gateway = "10.250.0.1"
	if _, err := a.Reserve(ctx, "project", "build-copy-0", "owner-0", changed, false); err == nil {
		t.Fatal("settings change accepted without rebuild")
	}
	if ip, err := a.Reserve(ctx, "project", "build-copy-0", "owner-0", changed, true); err != nil || ip != addresses[0] {
		t.Fatalf("rebuild settings edit changed address: %s %v", ip, err)
	}
	changed.Ranges = "10.250.3.200"
	if _, err := a.Reserve(ctx, "project", "build-copy-0", "owner-0", changed, true); err == nil {
		t.Fatal("rebuild silently replaced retained address outside new pool")
	}
	if err := a.Release(ctx, "project", "build-copy-0"); err != nil {
		t.Fatal(err)
	}
	next, err := a.Reserve(ctx, "other-project", "new-deployment", "new-owner", cfg, false)
	if err != nil || next != addresses[0] {
		t.Fatalf("reuse after release: %s, %v", next, err)
	}
	// Multiple retries of the SAME box share one durable reservation.
	var retry sync.WaitGroup
	for range 10 {
		retry.Add(1)
		go func() {
			defer retry.Done()
			ip, err := a.Reserve(ctx, "other-project", "new-deployment", "new-owner", cfg, false)
			if err != nil || ip != next {
				t.Errorf("concurrent retry: %s %v", ip, err)
			}
		}()
	}
	retry.Wait()
}

func TestPublicAllocationGloballyUniqueAcrossBuildersAndBuilds(t *testing.T) {
	dsn := os.Getenv("LAFORGE_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set LAFORGE_TEST_DATABASE_URL to a migrated test database")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	var ids [2]pgtype.UUID
	for i := range ids {
		if err := pool.QueryRow(ctx, `INSERT INTO builder_config(name,kind) VALUES ($1,'microcloud') RETURNING id`, fmt.Sprintf("public-global-%d-%d", os.Getpid(), i)).Scan(&ids[i]); err != nil {
			t.Fatal(err)
		}
		defer func(id pgtype.UUID) {
			pool.Exec(ctx, `DELETE FROM microcloud_public_address WHERE builder_id=$1`, id)
			pool.Exec(ctx, `DELETE FROM builder_config WHERE id=$1`, id)
		}(ids[i])
	}
	const count = 24
	addresses := make([]string, count)
	errs := make([]error, count)
	var wg sync.WaitGroup
	for i := range addresses {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			cfg := testPublicConfig()
			cfg.Network = fmt.Sprintf("public-network-%d", i%2)
			cfg.Ranges = "10.250.4.100-10.250.4.123"
			a := SQLPublicAddressAllocator{Pool: pool, BuilderID: ids[i%2]}
			addresses[i], errs[i] = a.Reserve(ctx, fmt.Sprintf("project-%d", i%2), fmt.Sprintf("build-%d-team-%d-copy-%d", i/6, i/2, i), fmt.Sprintf("owner-%d", i), cfg, false)
		}(i)
	}
	wg.Wait()
	seen := map[string]bool{}
	for i, ip := range addresses {
		if errs[i] != nil {
			t.Fatal(errs[i])
		}
		if seen[ip] {
			t.Fatalf("duplicate global IP %s", ip)
		}
		seen[ip] = true
	}
	// The database rejects a collision even if a writer bypasses the allocator.
	_, err = pool.Exec(ctx, `INSERT INTO microcloud_public_address(builder_id,project,instance_name,external_name,network,address,settings) VALUES($1,'different-project','bypass','bypass-owner','different-network',$2,'{}')`, ids[1], addresses[0])
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23505" {
		t.Fatalf("unique constraint not enforced: %v", err)
	}

	// Permanent destruction makes an IP available to an entirely different build
	// and builder config. Rebuild never calls Release (covered by lifecycle tests).
	a := SQLPublicAddressAllocator{Pool: pool, BuilderID: ids[0]}
	if err := a.Release(ctx, "project-0", "build-0-team-0-copy-0"); err != nil {
		t.Fatal(err)
	}
	cfg := testPublicConfig()
	cfg.Ranges = "10.250.4.100-10.250.4.123"
	a.BuilderID = ids[1]
	ip, err := a.Reserve(ctx, "new-project", "new-build", "new-owner", cfg, false)
	if err != nil || ip != addresses[0] {
		t.Fatalf("destroyed box address not reused: %s %v", ip, err)
	}
}
