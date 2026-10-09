package builder

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
)

// ErrInvalidOnboardRequest marks an onboarding failure caused by bad operator
// input -- a malformed trust token, a token given to a kind that doesn't take
// one -- rather than a failure to reach or enroll the hoster. An Onboarder wraps
// such errors with it (fmt.Errorf("%w: ...", ErrInvalidOnboardRequest, ...)) so
// the API can answer 400 for these and 502 for genuine connect failures.
var ErrInvalidOnboardRequest = errors.New("invalid onboarding request")

// Onboarding is the other half of the builder contract, kept separate from the
// Builder interface on purpose. Builder is an already-configured, running
// hoster (you call DeployHost on it). Onboarding is what happens BEFORE that:
// connecting a brand-new hoster of some kind and reading what it has, producing
// the stored connection material that internal/builderconfig.Resolve later turns
// into a Builder. It differs fundamentally by type -- MicroCloud and Incus
// onboard by enrolling a trust token (a certificate exchange); AWS and OpenStack
// "onboard" by validating environment credentials -- so it cannot be a method on
// Builder, and is its own per-type contract here.

// StoragePoolInfo, NetworkInfo, and ImageInfo are the resources an operator
// picks from when configuring a hoster, discovered by onboarding. They live in
// this neutral package (not a specific builder's) because the onboarding
// contract and the API layer both speak them, independent of which builder kind
// produced them. The LXD/Incus builders alias these; a cloud builder fills the
// analogous fields (an AMI is an ImageInfo, an instance type a "network"/pool is
// whatever that cloud calls it) or leaves them empty when it has no interactive
// discovery.
type StoragePoolInfo struct {
	Name   string `json:"name"`
	Driver string `json:"driver"` // "zfs", "ceph", "dir", ...
	Status string `json:"status"`
}

type NetworkInfo struct {
	Name    string `json:"name"`
	Type    string `json:"type"` // "physical", "bridge", "ovn", ...
	Managed bool   `json:"managed"`
	Status  string `json:"status"`
}

type ImageInfo struct {
	Fingerprint  string   `json:"fingerprint"`
	Aliases      []string `json:"aliases"`
	Architecture string   `json:"architecture"`
	Type         string   `json:"type"` // "container" or "virtual-machine"
	Properties   struct {
		OS          string `json:"os"`
		Release     string `json:"release"`
		Description string `json:"description"`
	} `json:"properties"`
}

// Discovery is what a hoster reports it has -- the pickable resources. Never nil
// slices in a value the API returns, so the wire is always a JSON array.
type Discovery struct {
	StoragePools []StoragePoolInfo `json:"storage_pools"`
	// Networks are the server's own (the `default` project's), since uplink
	// networks always live there whatever project instances go in.
	Networks []NetworkInfo `json:"networks"`
	// Images are the chosen project's (Connection.Project), which differ from
	// `default`'s when that project has features.images on.
	Images []ImageInfo `json:"images"`
	// Projects lists the server's projects, for a kind that lets an operator
	// choose one (MicroCloud); empty for every other kind.
	Projects []ProjectInfo `json:"projects"`
}

// ProjectInfo is one Incus/LXD project and the features that decide what it
// keeps separate from `default`.
type ProjectInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	// Each is true when the project has its own (rather than sharing
	// `default`'s): networks and ACLs, images, profiles, storage volumes.
	Networks       bool `json:"features_networks"`
	Images         bool `json:"features_images"`
	Profiles       bool `json:"features_profiles"`
	StorageVolumes bool `json:"features_storage_volumes"`
	// Restricted is the project's own `restricted` setting.
	Restricted bool `json:"restricted"`
}

// OnboardRequest is the operator's input for connecting a new hoster. Which
// fields matter depends on the kind: the LXD/Incus family reads Token (and the
// optional Address override); cloud kinds authenticate from the environment and
// read neither.
type OnboardRequest struct {
	Token   string
	Address string
}

// Connection is the credential material an onboarding produced, to be persisted
// (builder_credential). Empty for kinds that authenticate from the environment
// (AWS/OpenStack), which store no per-hoster secret.
type Connection struct {
	APIURL            string
	ServerName        string
	ServerFingerprint string
	ServerCertPEM     []byte
	ClientCertPEM     []byte
	ClientKeyPEM      []byte
	// Project scopes Rediscover's project-specific results (images) to an
	// Incus/LXD project; "" is `default`. Not part of the stored credential.
	Project string
}

// HasCredential reports whether this onboarding produced stored credential
// material (true for trust-token kinds, false for env-credential clouds).
func (c Connection) HasCredential() bool { return c.APIURL != "" }

// Onboarding is a completed connect: what to store, plus what was discovered.
type Onboarding struct {
	Connection Connection
	Discovery  Discovery
}

// Onboarder is a builder type's onboarding contract. Each builder package
// implements it and registers under its kind, so the connect endpoint dispatches
// by kind instead of hard-coding one family's flow.
type Onboarder interface {
	// Onboard connects to and validates a new hoster of this kind, returning the
	// connection material to store and the resources discovered on it.
	Onboard(ctx context.Context, req OnboardRequest) (*Onboarding, error)
	// Rediscover re-reads a hoster's resources from already-stored connection
	// material -- editing an existing builder config, so its choices are still
	// picks from the hoster's current state. For env-credential kinds the
	// Connection is empty and discovery runs against the environment.
	Rediscover(ctx context.Context, conn Connection) (*Discovery, error)
}

var (
	onboardersMu sync.RWMutex
	onboarders   = map[string]Onboarder{}
)

// RegisterOnboarder records a kind's Onboarder. Each builder package calls this
// from an init(), so importing the package (directly, or via builderconfig which
// imports them all) makes its kind onboardable. Registering the same kind twice
// panics -- a programming error, caught at startup, not silently last-wins.
func RegisterOnboarder(kind string, o Onboarder) {
	onboardersMu.Lock()
	defer onboardersMu.Unlock()
	if _, dup := onboarders[kind]; dup {
		panic(fmt.Sprintf("builder: onboarder for kind %q registered twice", kind))
	}
	onboarders[kind] = o
}

// OnboarderFor returns the Onboarder registered for kind, or an error naming the
// kinds that are registered so the caller can report it cleanly.
func OnboarderFor(kind string) (Onboarder, error) {
	onboardersMu.RLock()
	defer onboardersMu.RUnlock()
	o, ok := onboarders[kind]
	if !ok {
		return nil, fmt.Errorf("no onboarder for builder kind %q (known: %v)", kind, sortedKinds())
	}
	return o, nil
}

func sortedKinds() []string {
	ks := make([]string, 0, len(onboarders))
	for k := range onboarders {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	return ks
}
