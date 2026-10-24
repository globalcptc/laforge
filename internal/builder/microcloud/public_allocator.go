package microcloud

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PublicAddressAllocator must durably reserve before any hoster mutation. A
// failed create keeps its reservation: the hoster may have completed anyway.
type PublicAddressAllocator interface {
	Reserve(context.Context, string, string, string, PublicAccessConfig, bool) (string, error)
	Release(context.Context, string, string, ...string) error
}

type SQLPublicAddressAllocator struct {
	Pool      *pgxpool.Pool
	BuilderID pgtype.UUID
}

func (a SQLPublicAddressAllocator) Reserve(ctx context.Context, project, name, externalName string, cfg PublicAccessConfig, allowUpdate bool) (string, error) {
	if a.Pool == nil || !a.BuilderID.Valid {
		return "", fmt.Errorf("public NIC allocation requires a saved MicroCloud builder and database")
	}
	if externalName == "" {
		return "", fmt.Errorf("public NIC allocation requires a stable deployment identity")
	}
	if err := cfg.Validate(); err != nil {
		return "", err
	}
	ips, err := parsePublicIPs(cfg.Ranges)
	if err != nil {
		return "", err
	}
	settings, err := json.Marshal(cfg)
	if err != nil {
		return "", err
	}
	tx, err := a.Pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(context.Background())
	// All MicroCloud configs share one address namespace across builds and
	// runner processes. The unique address constraint is the final authority.
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended('microcloud-public-address', 0))`); err != nil {
		return "", err
	}
	var address, network, owner string
	var sameSettings bool
	err = tx.QueryRow(ctx, `SELECT host(address), network, settings = $4::jsonb, external_name FROM microcloud_public_address WHERE builder_id=$1 AND project=$2 AND instance_name=$3`, a.BuilderID, project, name, settings).Scan(&address, &network, &sameSettings, &owner)
	if err == nil {
		// Readable LXD names include a shortened hash. Never let a collision
		// in that hash adopt another build's box or share its reservation.
		if owner != externalName {
			return "", fmt.Errorf("instance name %s belongs to a different deployment; refusing to share its public IP", name)
		}
		if !sameSettings || network != cfg.Network {
			if !allowUpdate {
				return "", fmt.Errorf("public NIC settings changed for %s; rebuild the host to apply them", name)
			}
			inPool := false
			for _, ip := range ips {
				if ip == address {
					inPool = true
					break
				}
			}
			if network != cfg.Network || !inPool {
				return "", fmt.Errorf("retained public IP %s for %s is incompatible with the new network/pool; keep its network and include it in the pool", address, name)
			}
			if _, err := tx.Exec(ctx, `UPDATE microcloud_public_address SET settings=$4::jsonb WHERE builder_id=$1 AND project=$2 AND instance_name=$3`, a.BuilderID, project, name, settings); err != nil {
				return "", err
			}
		}
		if err := tx.Commit(ctx); err != nil {
			return "", err
		}
		slog.InfoContext(ctx, "MicroCloud public IP reservation reused", "builder_id", a.BuilderID.String(), "project", project, "instance", name, "network", cfg.Network, "public_ip", address, "settings_updated", !sameSettings)
		return address, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return "", err
	}
	// One statement chooses the first free address, rather than one round trip
	// per address. The transaction lock prevents two deploys choosing it.
	err = tx.QueryRow(ctx, `INSERT INTO microcloud_public_address (builder_id, project, instance_name, network, address, settings, external_name)
		SELECT $1, $2, $3, $4, candidate::inet, $5::jsonb, $7
		FROM unnest($6::text[]) WITH ORDINALITY AS pool(candidate, ordinal)
		WHERE NOT EXISTS (SELECT 1 FROM microcloud_public_address a WHERE a.address=pool.candidate::inet)
		ORDER BY ordinal LIMIT 1 RETURNING host(address)`, a.BuilderID, project, name, cfg.Network, settings, ips, externalName).Scan(&address)
	if errors.Is(err, pgx.ErrNoRows) {
		slog.WarnContext(ctx, "MicroCloud public IP pool exhausted", "builder_id", a.BuilderID.String(), "network", cfg.Network, "instance", name)
		return "", fmt.Errorf("public NIC address pool exhausted on network %s", cfg.Network)
	}
	if err != nil {
		return "", err
	}
	if err := tx.Commit(ctx); err != nil {
		return "", err
	}
	slog.InfoContext(ctx, "MicroCloud public IP reserved", "builder_id", a.BuilderID.String(), "project", project, "instance", name, "network", cfg.Network, "public_ip", address)
	return address, nil
}

func (a SQLPublicAddressAllocator) Release(ctx context.Context, project, name string, owner ...string) error {
	if a.Pool == nil || !a.BuilderID.Valid {
		return fmt.Errorf("public NIC release requires a saved MicroCloud builder and database")
	}
	var address string
	identity := ""
	if len(owner) > 0 {
		identity = owner[0]
	}
	err := a.Pool.QueryRow(ctx, `DELETE FROM microcloud_public_address WHERE builder_id=$1 AND project=$2 AND instance_name=$3 AND ($4='' OR external_name=$4) RETURNING host(address)`, a.BuilderID, project, name, identity).Scan(&address)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	slog.InfoContext(ctx, "MicroCloud public IP released after confirmed deletion", "builder_id", a.BuilderID.String(), "project", project, "instance", name, "public_ip", address)
	return nil
}
