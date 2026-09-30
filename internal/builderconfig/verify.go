package builderconfig

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/globalcptc/laforge/internal/builder/incus"
	"github.com/globalcptc/laforge/internal/db"
)

// VerifyImages asks every host of an Incus/MicroCloud builder config, through
// its own API, whether it really has each image the config offers -- by
// fingerprint when the config names one (the same image has the same
// fingerprint on every host), else by alias. A pool is only as good as its
// least-stocked host: a team placed on a host missing an image can't deploy
// it, so any gap is an error naming the image and the host. Images pulled
// from a remote server on first use (Server set) aren't checked.
func VerifyImages(ctx context.Context, pool *pgxpool.Pool, row db.BuilderConfig) error {
	if row.Kind != "microcloud" && row.Kind != "incus" {
		return nil
	}
	images := map[string]incus.ImageRef{}
	if len(row.IncusImages) > 0 {
		if err := json.Unmarshal(row.IncusImages, &images); err != nil {
			return fmt.Errorf("decoding incus_images: %w", err)
		}
	}
	var local []string
	for name, ref := range images {
		if ref.Server == "" {
			local = append(local, name)
		}
	}
	if len(local) == 0 {
		return nil
	}
	sort.Strings(local)

	hosts, err := hostEndpoints(pool, row)
	if err != nil {
		return err
	}
	var problems []string
	for _, h := range hosts {
		client, err := h.ep.client()
		if err != nil {
			return fmt.Errorf("%s: %w", h.label, err)
		}
		have, err := client.ListImages(ctx)
		if err != nil {
			return fmt.Errorf("listing images on %s: %w", h.label, err)
		}
		fingerprints := map[string]bool{}
		aliases := map[string]bool{}
		for _, img := range have {
			fingerprints[img.Fingerprint] = true
			for _, a := range img.Aliases {
				aliases[a] = true
			}
		}
		for _, name := range local {
			ref := images[name]
			if ref.Fingerprint != "" && !fingerprints[ref.Fingerprint] {
				problems = append(problems, fmt.Sprintf("%q is not on %s", name, h.label))
			} else if ref.Fingerprint == "" && !aliases[ref.Alias] {
				problems = append(problems, fmt.Sprintf("%q (alias %s) is not on %s", name, ref.Alias, h.label))
			}
		}
	}
	if len(problems) > 0 {
		return fmt.Errorf("every host needs every image this builder offers: %s", strings.Join(problems, "; "))
	}
	return nil
}

type labeledEndpoint struct {
	label string
	ep    endpoint
}

func hostEndpoints(pool *pgxpool.Pool, row db.BuilderConfig) ([]labeledEndpoint, error) {
	if row.Kind == "microcloud" {
		var ep endpoint
		var err error
		if row.IncusCredentialID.Valid {
			ep, err = loadCredential(pool, row.IncusCredentialID)
		} else {
			ep, err = pathEndpoint(db.StrOrEmpty(row.IncusApiUrl), db.StrOrEmpty(row.IncusClientCertPath),
				db.StrOrEmpty(row.IncusClientKeyPath), db.StrOrEmpty(row.IncusServerCertPem))
		}
		if err != nil {
			return nil, err
		}
		return []labeledEndpoint{{label: ep.apiURL, ep: ep}}, nil
	}
	var hostConfigs []incus.HostConfig
	if err := json.Unmarshal(row.IncusHosts, &hostConfigs); err != nil {
		return nil, fmt.Errorf("decoding incus_hosts: %w", err)
	}
	out := make([]labeledEndpoint, 0, len(hostConfigs))
	for i, hc := range hostConfigs {
		var ep endpoint
		var err error
		if hc.CredentialID != "" {
			var id pgtype.UUID
			if err := id.Scan(hc.CredentialID); err != nil {
				return nil, fmt.Errorf("host %d: invalid credential_id: %w", i+1, err)
			}
			ep, err = loadCredential(pool, id)
		} else {
			ep, err = pathEndpoint(hc.APIURL, hc.ClientCertPath, hc.ClientKeyPath, hc.ServerCertPEM)
		}
		if err != nil {
			return nil, fmt.Errorf("host %d: %w", i+1, err)
		}
		out = append(out, labeledEndpoint{label: fmt.Sprintf("host %d (%s)", i+1, ep.apiURL), ep: ep})
	}
	return out, nil
}
