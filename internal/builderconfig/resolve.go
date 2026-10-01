// Package builderconfig is the real answer to "what about the MicroCloud
// builder" -- and, since (2026-09-25) confirmed directly, to "we want to
// support both as real with completely separate builder types": Incus and
// MicroCloud are genuinely different infrastructure shapes, not one
// client wearing two names. See migration 00014's own doc comment for the
// full split:
//
//   - kind "microcloud" is one real MicroCloud cluster (Ceph + OVN + LXD,
//     any member answers for the whole cluster) -- a single
//     internal/builder/microcloud.Builder pointed at that one endpoint. It
//     is its own LXD-based builder, separate from Incus (they share only the
//     builder.Builder interface).
//   - kind "incus" is a POOL of independent, non-clustered Incus hosts --
//     internal/builder/incuspool.Pool wrapping one internal/builder/incus.Builder
//     per host, routing every call to a team's assigned host
//     deterministically (see Pool's own doc comment for "round-robin
//     style... spread the load" made concrete).
//
// Resolve is the lookup that turns a real db.BuilderConfig row (migration
// 00012/00014) into the real builder.Builder it describes.
// internal/runner's own per-task Builders resolver (Runner.Builders,
// mirroring Checkouts' per-task checkout resolution) is the real caller; internal/api's
// builder-config CRUD endpoints are the other.
package builderconfig

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/globalcptc/laforge/internal/builder"
	awsbuilder "github.com/globalcptc/laforge/internal/builder/aws"
	"github.com/globalcptc/laforge/internal/builder/fake"
	"github.com/globalcptc/laforge/internal/builder/incus"
	"github.com/globalcptc/laforge/internal/builder/incuspool"
	"github.com/globalcptc/laforge/internal/builder/microcloud"
	openstackbuilder "github.com/globalcptc/laforge/internal/builder/openstack"
	"github.com/globalcptc/laforge/internal/db"
)

// Resolve constructs the real builder.Builder row describes. For kind
// "microcloud"/"incus", each connection comes from an enrolled
// builder_credential row (migration 00016) when the config names one, or
// from the older cert/key file paths otherwise -- a missing credential or
// unreadable file is a clear error here, not a stub. Pool is needed for
// kind "fake" (internal/builder/fake.New records its simulated work in
// Postgres) and to load enrolled credentials.
func Resolve(pool *pgxpool.Pool, row db.BuilderConfig) (builder.Builder, error) {
	switch row.Kind {
	case "fake":
		return fake.New(pool), nil
	case "microcloud":
		return resolveMicrocloud(pool, row)
	case "incus":
		return resolveIncusPool(pool, row)
	case "aws":
		return resolveAWS(row)
	case "openstack":
		return resolveOpenStack(row)
	default:
		return nil, fmt.Errorf("builder config %q: unknown kind %q", row.Name, row.Kind)
	}
}

// cloudImages / cloudSizes read the shared image/size columns for a cloud
// builder. The columns hold the same Incus ImageRef/SizeSpec shape every
// builder kind stores, so a cloud image id (an AMI, a Glance image) is carried
// in the ImageRef's Fingerprint (its Alias as a fallback), and a cloud
// instance type/flavor in the SizeSpec's Type -- the fields the wizard fills
// for a cloud builder. A size with no Type is skipped rather than mapped to an
// empty instance type.
func cloudImages(raw []byte) map[string]string {
	refs := map[string]incus.ImageRef{}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &refs)
	}
	out := make(map[string]string, len(refs))
	for name, ref := range refs {
		if id := ref.Fingerprint; id != "" {
			out[name] = id
		} else if ref.Alias != "" {
			out[name] = ref.Alias
		}
	}
	return out
}

func cloudSizes(raw []byte) map[string]string {
	specs := map[string]incus.SizeSpec{}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &specs)
	}
	out := make(map[string]string, len(specs))
	for name, s := range specs {
		if s.Type != "" {
			out[name] = s.Type
		}
	}
	return out
}

// resolveAWS / resolveOpenStack construct the cloud builders. Credentials come
// from each SDK's standard environment chain (AWS_*, OS_*) and region from the
// matching env var (set on the runner), so nothing secret is stored in the DB;
// the image/size maps come from the builder config the wizard writes.
func resolveAWS(row db.BuilderConfig) (builder.Builder, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	b, err := awsbuilder.New(ctx, awsbuilder.Config{
		Region: os.Getenv("AWS_REGION"),
		Images: cloudImages(row.IncusImages),
		Sizes:  cloudSizes(row.IncusSizes),
	})
	if err != nil {
		return nil, fmt.Errorf("builder config %q: %w", row.Name, err)
	}
	return b, nil
}

func resolveOpenStack(row db.BuilderConfig) (builder.Builder, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	b, err := openstackbuilder.New(ctx, openstackbuilder.Config{
		Region: os.Getenv("OS_REGION_NAME"),
		Images: cloudImages(row.IncusImages),
		Sizes:  cloudSizes(row.IncusSizes),
	})
	if err != nil {
		return nil, fmt.Errorf("builder config %q: %w", row.Name, err)
	}
	return b, nil
}

// ResolveMicrocloudClient returns the low-level *microcloud.Client for a
// builder -- what the per-builder image-build job needs (it drives
// exec/import/publish directly, below the builder.Builder interface). Only the
// MicroCloud kind builds a docker base image: LXD has no OCI runtime. Kind
// "incus" runs containers as native OCI (Incus 6.3+ pulls straight from the
// registry at deploy time), so it never uses a base image; this reports that
// rather than building one -- and nothing should ask for it (usesDockerBase is
// microcloud-only).
func ResolveMicrocloudClient(pool *pgxpool.Pool, row db.BuilderConfig) (*microcloud.Client, error) {
	switch row.Kind {
	case "microcloud":
		var ep endpoint
		var err error
		if row.IncusCredentialID.Valid {
			ep, err = loadCredential(pool, row.IncusCredentialID)
		} else {
			ep, err = pathEndpoint(db.StrOrEmpty(row.IncusApiUrl), db.StrOrEmpty(row.IncusClientCertPath),
				db.StrOrEmpty(row.IncusClientKeyPath), db.StrOrEmpty(row.IncusServerCertPem))
		}
		if err != nil {
			return nil, fmt.Errorf("builder config %q: %w", row.Name, err)
		}
		client, err := ep.microcloudClient()
		if err != nil {
			return nil, fmt.Errorf("builder config %q: constructing client: %w", row.Name, err)
		}
		return client, nil
	case "incus":
		return nil, fmt.Errorf("builder config %q: kind %q runs containers as native OCI and needs no docker base image", row.Name, row.Kind)
	default:
		return nil, fmt.Errorf("builder config %q: kind %q has no MicroCloud client", row.Name, row.Kind)
	}
}

// endpoint is one Incus/MicroCloud connection's material, however it was
// stored.
type endpoint struct {
	apiURL                         string
	certPEM, keyPEM, serverCertPEM []byte
}

func (e endpoint) client() (*incus.Client, error) {
	return incus.NewClient(e.apiURL, e.certPEM, e.keyPEM, e.serverCertPEM, "")
}

func loadCredential(pool *pgxpool.Pool, id pgtype.UUID) (endpoint, error) {
	if pool == nil {
		return endpoint{}, errors.New("loading an enrolled credential needs a database connection")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cred, err := db.New(pool).GetBuilderCredential(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return endpoint{}, fmt.Errorf("no such builder credential %s", id.String())
		}
		return endpoint{}, fmt.Errorf("loading builder credential %s: %w", id.String(), err)
	}
	return endpoint{
		apiURL: cred.ApiUrl, certPEM: []byte(cred.ClientCertPem),
		keyPEM: []byte(cred.ClientKeyPem), serverCertPEM: []byte(cred.ServerCertPem),
	}, nil
}

func pathEndpoint(apiURL, certPath, keyPath, serverCertPEM string) (endpoint, error) {
	if apiURL == "" {
		return endpoint{}, errors.New("api_url is required (or connect with a trust token)")
	}
	if certPath == "" || keyPath == "" {
		return endpoint{}, errors.New("client cert and key paths are both required (or connect with a trust token)")
	}
	if serverCertPEM == "" {
		return endpoint{}, errors.New("the server certificate is required (or connect with a trust token)")
	}
	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		return endpoint{}, fmt.Errorf("reading client cert: %w", err)
	}
	keyPEM, err := os.ReadFile(keyPath)
	if err != nil {
		return endpoint{}, fmt.Errorf("reading client key: %w", err)
	}
	return endpoint{apiURL: apiURL, certPEM: certPEM, keyPEM: keyPEM, serverCertPEM: []byte(serverCertPEM)}, nil
}

func decodeImagesAndSizes(row db.BuilderConfig) (map[string]incus.ImageRef, map[string]incus.SizeSpec, error) {
	images := map[string]incus.ImageRef{}
	if len(row.IncusImages) > 0 {
		if err := json.Unmarshal(row.IncusImages, &images); err != nil {
			return nil, nil, fmt.Errorf("builder config %q: decoding incus_images: %w", row.Name, err)
		}
	}
	sizes := map[string]incus.SizeSpec{}
	if len(row.IncusSizes) > 0 {
		if err := json.Unmarshal(row.IncusSizes, &sizes); err != nil {
			return nil, nil, fmt.Errorf("builder config %q: decoding incus_sizes: %w", row.Name, err)
		}
	}
	return images, sizes, nil
}

// resolveMicrocloud builds the MicroCloud builder -- one real, single-endpoint
// cluster (Ceph + OVN + LXD). MicroCloud is its OWN builder package
// (internal/builder/microcloud), separate from Incus: they are distinct
// products (MicroCloud runs LXD, Incus is the LinuxContainers fork) and do not
// share a builder implementation, only the builder.Builder interface. The
// incus_* column names are legacy storage (migration 00012/00014) shared by
// both kinds' connection material, not a claim that microcloud is Incus.
func resolveMicrocloud(pool *pgxpool.Pool, row db.BuilderConfig) (builder.Builder, error) {
	var ep endpoint
	var err error
	if row.IncusCredentialID.Valid {
		ep, err = loadCredential(pool, row.IncusCredentialID)
	} else {
		ep, err = pathEndpoint(db.StrOrEmpty(row.IncusApiUrl), db.StrOrEmpty(row.IncusClientCertPath),
			db.StrOrEmpty(row.IncusClientKeyPath), db.StrOrEmpty(row.IncusServerCertPem))
	}
	if err != nil {
		return nil, fmt.Errorf("builder config %q: %w", row.Name, err)
	}
	client, err := ep.microcloudClient()
	if err != nil {
		return nil, fmt.Errorf("builder config %q: constructing client: %w", row.Name, err)
	}
	if row.IncusOperationTimeoutSeconds != nil && *row.IncusOperationTimeoutSeconds > 0 {
		client.OperationTimeout = time.Duration(*row.IncusOperationTimeoutSeconds) * time.Second
	}

	images, sizes, err := decodeMicrocloudImagesAndSizes(row)
	if err != nil {
		return nil, err
	}

	return microcloud.New(client, microcloud.Config{
		Images: images, Sizes: sizes,
		OVNUplinkNetwork:      db.StrOrEmpty(row.IncusOvnUplinkNetwork),
		StoragePool:           db.StrOrEmpty(row.IncusStoragePool),
		ExternalAccessIP:      db.StrOrEmpty(row.ExternalAccessIp),
		ExternalPortMin:       int(db.Int32OrZero(row.ExternalPortMin)),
		ExternalPortMax:       int(db.Int32OrZero(row.ExternalPortMax)),
		DockerBaseFingerprint: row.DockerBaseFingerprint,
	}), nil
}

// decodeMicrocloudImagesAndSizes decodes the same incus_images/incus_sizes JSON
// the config stores into MicroCloud's own value types. The JSON shape is
// identical across builder kinds (shared columns), so this is a straight
// unmarshal, not a claim of shared types.
func decodeMicrocloudImagesAndSizes(row db.BuilderConfig) (map[string]microcloud.ImageRef, map[string]microcloud.SizeSpec, error) {
	images := map[string]microcloud.ImageRef{}
	if len(row.IncusImages) > 0 {
		if err := json.Unmarshal(row.IncusImages, &images); err != nil {
			return nil, nil, fmt.Errorf("builder config %q: decoding incus_images: %w", row.Name, err)
		}
	}
	sizes := map[string]microcloud.SizeSpec{}
	if len(row.IncusSizes) > 0 {
		if err := json.Unmarshal(row.IncusSizes, &sizes); err != nil {
			return nil, nil, fmt.Errorf("builder config %q: decoding incus_sizes: %w", row.Name, err)
		}
	}
	return images, sizes, nil
}

// microcloudClient builds a MicroCloud (LXD) REST client from this endpoint's
// connection material -- the LXD-based sibling of client() (which serves the
// Incus pool).
func (e endpoint) microcloudClient() (*microcloud.Client, error) {
	return microcloud.NewClient(e.apiURL, e.certPEM, e.keyPEM, e.serverCertPEM, "")
}

// resolveIncusPool builds one internal/builder/incus.Builder per entry in
// the row's own incus_hosts JSONB array (migration 00014) and wraps them
// in an internal/builder/incuspool.Pool. Every host needs a connection --
// an enrolled credential_id, or the older api_url/cert paths/server cert
// -- validated per host, with the host's own index in the error so a
// broken entry in a pool of many is easy to find. At least one host is
// required: a pool with none configured can never resolve a team to
// anything.
func resolveIncusPool(pool *pgxpool.Pool, row db.BuilderConfig) (builder.Builder, error) {
	var hostConfigs []incus.HostConfig
	if len(row.IncusHosts) > 0 {
		if err := json.Unmarshal(row.IncusHosts, &hostConfigs); err != nil {
			return nil, fmt.Errorf("builder config %q: decoding incus_hosts: %w", row.Name, err)
		}
	}
	if len(hostConfigs) == 0 {
		return nil, fmt.Errorf("builder config %q: incus_hosts must list at least one host for kind \"incus\"", row.Name)
	}

	images, sizes, err := decodeImagesAndSizes(row)
	if err != nil {
		return nil, err
	}

	hosts := make([]*incus.Builder, 0, len(hostConfigs))
	for i, hc := range hostConfigs {
		var ep endpoint
		if hc.CredentialID != "" {
			var id pgtype.UUID
			if err := id.Scan(hc.CredentialID); err != nil {
				return nil, fmt.Errorf("builder config %q: incus_hosts[%d]: invalid credential_id: %w", row.Name, i, err)
			}
			ep, err = loadCredential(pool, id)
		} else {
			ep, err = pathEndpoint(hc.APIURL, hc.ClientCertPath, hc.ClientKeyPath, hc.ServerCertPEM)
		}
		if err != nil {
			return nil, fmt.Errorf("builder config %q: incus_hosts[%d]: %w", row.Name, i, err)
		}
		client, err := ep.client()
		if err != nil {
			return nil, fmt.Errorf("builder config %q: incus_hosts[%d]: constructing client: %w", row.Name, i, err)
		}
		if hc.OperationTimeoutSeconds > 0 {
			client.OperationTimeout = time.Duration(hc.OperationTimeoutSeconds) * time.Second
		}
		hosts = append(hosts, incus.New(client, incus.Config{
			Images: images, Sizes: sizes,
			OVNUplinkNetwork: hc.OVNUplinkNetwork, StoragePool: hc.StoragePool, DockerBaseFingerprint: row.DockerBaseFingerprint,
			ExternalAccessIP: db.StrOrEmpty(row.ExternalAccessIp),
			ExternalPortMin:  int(db.Int32OrZero(row.ExternalPortMin)),
			ExternalPortMax:  int(db.Int32OrZero(row.ExternalPortMax)),
		}))
	}

	return incuspool.New(hosts, images), nil
}
