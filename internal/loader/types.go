// Package loader walks a content repository, classifies each document by its
// `<type>: <name>` header, validates it against internal/schema, decodes it
// into these typed structs, and runs the cross-file checks a single
// document's JSON Schema can't express on its own (name collisions,
// depends_on targets existing, public ports being a subset of a host's own
// ports).
package loader

// Finding is deliberately three fields and nothing else — see
// "Findings are just an inline list... No names, no IDs, no separate files."
type Finding struct {
	Severity    int    `yaml:"severity" json:"severity"`
	Difficulty  int    `yaml:"difficulty" json:"difficulty"`
	Description string `yaml:"description" json:"description"`
}

type Ports struct {
	TCP []string `yaml:"tcp,omitempty" json:"tcp,omitempty"`
	UDP []string `yaml:"udp,omitempty" json:"udp,omitempty"`
}

// Step is intentionally loose (map[string]interface{}) rather than a tagged
// union in Go: the schema already enforces "exactly one action key," and a
// generic map is what the future template/render engine and
// the agent-command translation (later) both need to inspect without this
// package having to know every action shape twice.
type Step map[string]interface{}

// ActionKey returns the step's one action key (e.g. "script", "download"),
// ignoring "validate" and "when" -- "validate" may ride alongside any
// step, and "when" is a schedule entry's own required timing field,
// never an action itself.
// The same Step type represents both a steps: entry and a schedule:
// entry (identical shape, one more key), which is exactly why "when"
// has to be excluded here too, not just where a schedule entry gets
// parsed -- map iteration order is random, so silently returning "when"
// as if it were the real action only shows up intermittently, not on
// every run. Schema validation guarantees exactly one real action key
// exists by the time a Step reaches here.
func (s Step) ActionKey() string {
	for k := range s {
		if k != "validate" && k != "when" {
			return k
		}
	}
	return ""
}

// Every SourceFile below is set by the loader after decode (see loader.go)
// and is never present in the YAML/JSON itself, in either direction:
// `yaml:"-"`/`json:"-"` keep it out of output too, which matters once
// something (the converter in internal/hclconvert, a future `laforge fmt`,
// internal/ingest persisting to Postgres, internal/api serving JSON) reuses
// these types for output rather than only reading them.
//
// omitempty throughout is the same idea applied to every genuinely optional
// field: a hand-authored file typically omits what it doesn't set, and
// generated output should read the same way, not full of `findings: []` /
// `tags: {}` / `extends: ""` noise on every object that doesn't use them.
//
// json tags mirror the yaml tags key-for-key rather than falling back to Go's
// default (capitalized field name) JSON encoding, so a Postgres jsonb column
// or an API response holding one of these looks exactly like the YAML a
// person would have hand-authored, not a differently-cased shadow of it.

type Network struct {
	SourceFile string `yaml:"-" json:"-"`
	Name       string `yaml:"name" json:"name"`
	CIDR       string `yaml:"cidr" json:"cidr"`
	// VisibleFrom is the ACL allowlist for this network: the other networks
	// (by name) that are allowed to reach it. Enforced as ACLs on the OVN
	// routing host by the builder. Replaces the old boolean `vdi_visible`,
	// which could only say "the VDI network reaches this" -- an array lets any
	// network's reachability be defined explicitly (`visible_from: [vdi, client]`).
	VisibleFrom []string          `yaml:"visible_from,omitempty" json:"visible_from,omitempty"`
	Vars        map[string]string `yaml:"vars,omitempty" json:"vars,omitempty"`
	Tags        map[string]string `yaml:"tags,omitempty" json:"tags,omitempty"`
	Findings    []Finding         `yaml:"findings,omitempty" json:"findings,omitempty"`
	Extends     string            `yaml:"extends,omitempty" json:"extends,omitempty"`
}

type Host struct {
	SourceFile string   `yaml:"-" json:"-"`
	Name       string   `yaml:"name" json:"name"`
	OS         string   `yaml:"os" json:"os"`
	Size       string   `yaml:"size" json:"size"`
	Disk       int      `yaml:"disk" json:"disk"`
	Ports      Ports    `yaml:"ports,omitempty" json:"ports,omitempty"`
	DependsOn  []string `yaml:"depends_on,omitempty" json:"depends_on,omitempty"`
	Steps      []Step   `yaml:"steps,omitempty" json:"steps,omitempty"`
	// Schedule is deliberately a separate list from Steps, not a step
	// kind within it -- "steps run once, in order; schedule entries fire
	// independently, on their own clock" (a real design mismatch in the
	// old shape, where `schedule:` lived inside `steps:` despite never
	// behaving like one). Each entry is the same loose Step shape plus a
	// required `when:` natural-language expression (internal/schedule).
	Schedule []Step            `yaml:"schedule,omitempty" json:"schedule,omitempty"`
	Vars     map[string]string `yaml:"vars,omitempty" json:"vars,omitempty"`
	Tags     map[string]string `yaml:"tags,omitempty" json:"tags,omitempty"`
	Findings []Finding         `yaml:"findings,omitempty" json:"findings,omitempty"`
	People   []PeopleRef       `yaml:"people,omitempty" json:"people,omitempty"`
	Extends  string            `yaml:"extends,omitempty" json:"extends,omitempty"`
}

type Container struct {
	SourceFile string   `yaml:"-" json:"-"`
	Name string `yaml:"name" json:"name"`
	// Image is the OCI image ref the container runs, e.g. "nginx:alpine" or
	// "registry.internal/team/app:1.2". LaForge runs it as a Docker container
	// (inside a thin nesting LXD instance on Incus) -- see the builder.
	Image string `yaml:"image" json:"image"`
	Size  string `yaml:"size" json:"size"`
	// Env is passed to the container as environment variables (docker -e).
	Env map[string]string `yaml:"env,omitempty" json:"env,omitempty"`
	// Command overrides the image's default command/entrypoint arguments.
	Command   []string `yaml:"command,omitempty" json:"command,omitempty"`
	Ports     Ports    `yaml:"ports,omitempty" json:"ports,omitempty"`
	DependsOn []string `yaml:"depends_on,omitempty" json:"depends_on,omitempty"`
	Steps     []Step   `yaml:"steps,omitempty" json:"steps,omitempty"`
	// Schedule: see Host.Schedule's own comment -- same shape, same reasoning.
	Schedule []Step            `yaml:"schedule,omitempty" json:"schedule,omitempty"`
	Vars     map[string]string `yaml:"vars,omitempty" json:"vars,omitempty"`
	Tags     map[string]string `yaml:"tags,omitempty" json:"tags,omitempty"`
	Findings []Finding         `yaml:"findings,omitempty" json:"findings,omitempty"`
	People   []PeopleRef       `yaml:"people,omitempty" json:"people,omitempty"`
	Extends  string            `yaml:"extends,omitempty" json:"extends,omitempty"`
}

type Script struct {
	SourceFile   string                   `yaml:"-" json:"-"`
	Name         string                   `yaml:"name" json:"name"`
	Description  string                   `yaml:"description,omitempty" json:"description,omitempty"`
	Language     string                   `yaml:"language" json:"language"`
	Source       string                   `yaml:"source" json:"source"`
	Timeout      int                      `yaml:"timeout,omitempty" json:"timeout,omitempty"`
	Args         []string                 `yaml:"args,omitempty" json:"args,omitempty"`
	IgnoreErrors bool                     `yaml:"ignore_errors,omitempty" json:"ignore_errors,omitempty"`
	Tags         map[string]string        `yaml:"tags,omitempty" json:"tags,omitempty"`
	Findings     []Finding                `yaml:"findings,omitempty" json:"findings,omitempty"`
	People       []PeopleRef              `yaml:"people,omitempty" json:"people,omitempty"`
	Validate     []map[string]interface{} `yaml:"validate,omitempty" json:"validate,omitempty"`
}

type DNSRecord struct {
	Name     string `yaml:"name" json:"name"`
	Type     string `yaml:"type" json:"type"`
	Target   string `yaml:"target" json:"target"`
	Priority int    `yaml:"priority,omitempty" json:"priority,omitempty"`
}

type DNS struct {
	Type       string      `yaml:"type,omitempty" json:"type,omitempty"`
	RootDomain string      `yaml:"root_domain,omitempty" json:"root_domain,omitempty"`
	DNSServers []string    `yaml:"dns_servers,omitempty" json:"dns_servers,omitempty"`
	NTPServers []string    `yaml:"ntp_servers,omitempty" json:"ntp_servers,omitempty"`
	Records    []DNSRecord `yaml:"records,omitempty" json:"records,omitempty"`
}

type AccessWindow struct {
	Open  string `yaml:"open" json:"open"`
	Close string `yaml:"close" json:"close"`
}

// Public is nil if the field was absent, a non-nil *Ports if it was an
// object, per "Leave `public` out, or set it to `false`, and nothing is
// made public" -- both "absent" and "false" collapse to the same nil here.
type Copy struct {
	As        string `yaml:"as" json:"as"`
	LastOctet int    `yaml:"last_octet" json:"last_octet"`
	Public    *Ports `yaml:"public,omitempty" json:"public,omitempty"`
}

type Environment struct {
	SourceFile  string            `yaml:"-" json:"-"`
	Name        string            `yaml:"name" json:"name"`
	Schema      int               `yaml:"schema,omitempty" json:"schema,omitempty"`
	Description string            `yaml:"description,omitempty" json:"description,omitempty"`
	Teams       int               `yaml:"teams" json:"teams"`
	RootPass    string            `yaml:"root_password,omitempty" json:"root_password,omitempty"`
	Start       string            `yaml:"start,omitempty" json:"start,omitempty"`
	Stop        string            `yaml:"stop,omitempty" json:"stop,omitempty"`
	DNS         *DNS              `yaml:"dns,omitempty" json:"dns,omitempty"`
	Access      []AccessWindow    `yaml:"access,omitempty" json:"access,omitempty"`
	Vars        map[string]string `yaml:"vars,omitempty" json:"vars,omitempty"`
	Tags        map[string]string `yaml:"tags,omitempty" json:"tags,omitempty"`
	Findings    []Finding         `yaml:"findings,omitempty" json:"findings,omitempty"`
	// Networks: network name -> object name (host or container) -> copies
	Networks map[string]map[string][]Copy `yaml:"networks,omitempty" json:"networks,omitempty"`
	Extends  string                       `yaml:"extends,omitempty" json:"extends,omitempty"`
}

// Person is one row of a people/*.csv file. Columns beyond username are
// intentionally a free-form bag -- see "A people CSV is data, not
// configuration" -- so a repo can add whatever columns its scripts need
// without touching this package.
type Person struct {
	SourceFile string            `yaml:"-" json:"-"`
	Username   string            `yaml:"username" json:"username"`
	Attributes map[string]string `yaml:"attributes,omitempty" json:"attributes,omitempty"`
}

type PeopleSource struct {
	SourceFile string   `yaml:"-" json:"-"`
	Name       string   `yaml:"name" json:"name"` // "employees" from people/employees.csv
	People     []Person `yaml:"people,omitempty" json:"people,omitempty"`
}

// PeopleRef is one entry in an object's `people:` list: a whole people/*.csv
// file (by name, no extension) whose rows this object concerns, optionally
// narrowed to a subset of usernames. An unfiltered ref means "everyone in that
// file"; Filter, when set, means "just these usernames from that file." The
// resolved set is what a script sees as `{{ .people }}` (see internal/render).
type PeopleRef struct {
	File   string   `yaml:"file" json:"file"`
	Filter []string `yaml:"filter,omitempty" json:"filter,omitempty"`
}
