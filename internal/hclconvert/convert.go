package hclconvert

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/globalcptc/laforge/internal/loader"
)

// Result is everything a conversion run produced: the new-model objects,
// ready to write out, and every Note explaining a judgment call or a thing
// that could not be carried over.
type Result struct {
	Networks     []loader.Network
	Hosts        []loader.Host
	Scripts      []loader.Script // Source still points at the OLD relative filename; Write copies the real file alongside.
	Environments []loader.Environment
	People       map[string][]loader.Person // group name ("employees", "rooms", ...) -> rows

	// scriptSourceOnDisk maps a converted script's new name to the OLD
	// absolute path of its real source file, so Write can copy the actual
	// bytes across untouched -- "scripts carry over untouched."
	scriptSourceOnDisk map[string]string

	Notes []Note
}

// legacyReboot / legacyRun record the pre-agent "command" blocks (per the
// user: "from a pre-agent version of laforge (1.0), probably doesn't
// really matter anymore" -- kept anyway, since inlining them costs little
// and is strictly more faithful than silently dropping a real, currently
// live part of 20+ hosts' provisioning sequences).
type legacyCommand struct {
	program string
	args    []string
}

// Convert runs the whole old-repo -> new-model mapping. It never returns
// an error for bad *content* -- every problem becomes a Note instead, per
// "flags anything it cannot translate rather than guessing." An error
// return is reserved for the repo being unreadable at all.
func Convert(repoRoot string) (*Result, error) {
	blocks, parseNotes, err := ParseRepo(repoRoot)
	if err != nil {
		return nil, err
	}

	r := &Result{
		People:             make(map[string][]loader.Person),
		scriptSourceOnDisk: make(map[string]string),
		Notes:              parseNotes,
	}

	// Index everything by its old absolute ID first, since environments,
	// hosts, and provision_steps all reference other blocks by ID and
	// conversion has to resolve those regardless of file order.
	byID := make(map[string]*Block) // "host"/"network"/"script" blocks, keyed by their HCL label
	commands := make(map[string]legacyCommand)
	for _, b := range blocks {
		switch b.Type {
		case "host", "network", "script":
			byID[b.label()] = b
		case "command":
			commands[b.label()] = legacyCommand{program: b.str("program"), args: b.strList("args")}
		case "file_download":
			r.note(b.SourceFile, "info", fmt.Sprintf("skipped file_download %q: this is the old agent's bootstrap-download mechanism, obsolete by design -- the new agent is compiled in with its identity baked in, never downloaded as a provisioning step", b.label()))
		}
	}

	names := newNameRegistry()

	for _, b := range blocks {
		switch b.Type {
		case "network":
			r.convertNetwork(b, names)
		case "identity":
			r.convertIdentity(b)
		}
	}
	for _, b := range blocks {
		if b.Type == "script" {
			r.convertScript(repoRoot, b, names)
		}
	}
	for _, b := range blocks {
		if b.Type == "host" {
			r.convertHost(b, names, commands, byID)
		}
	}
	for _, b := range blocks {
		if b.Type == "environment" {
			r.convertEnvironment(b, blocks, byID, names)
		}
	}

	sort.Slice(r.Notes, func(i, j int) bool { return r.Notes[i].File < r.Notes[j].File })
	return r, nil
}

func (r *Result) note(file, severity, msg string) {
	r.Notes = append(r.Notes, Note{File: file, Severity: severity, Message: msg})
}

// --- naming ---

var slugRe = regexp.MustCompile(`[^a-z0-9-]+`)

// slug turns an old HCL ID like "/hosts/prod/APT-DC-01" into a new bare
// name. Hosts and containers share one namespace in the new model, same
// as networks and scripts each have their own -- slugRegistry (below)
// enforces that and flags any real collision rather than silently
// overwriting one object with another.
func slug(id string) string {
	base := id
	if i := strings.LastIndex(id, "/"); i >= 0 {
		base = id[i+1:]
	}
	s := strings.ToLower(base)
	s = slugRe.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if s == "" {
		s = "x"
	}
	return s
}

// nameRegistry assigns a unique new-model name per namespace (hosts and
// containers share one; networks and scripts each have their own) and
// remembers old-ID -> new-name so later blocks (environments, provision
// steps) can resolve references made by old ID.
type nameRegistry struct {
	byNamespace map[string]map[string]bool // namespace -> name -> taken
	oldToNew    map[string]string          // old ID -> new name (globally unique keys, since old IDs are absolute paths)
}

func newNameRegistry() *nameRegistry {
	return &nameRegistry{byNamespace: make(map[string]map[string]bool), oldToNew: make(map[string]string)}
}

// assign is called exactly once per definition block (never as a lookup --
// see the `lookup` method for that), so a repeated oldID here is never a
// legitimate re-reference: it means two different blocks in the old repo
// were declared under the identical old ID (a real, if rare, copy-paste bug
// in the source content -- e.g. a script file cloned from another without
// updating its internal label). Silently reusing the cached name would
// silently drop the second definition's content when it's written to the
// same output filename, which is exactly the kind of guess this converter
// isn't supposed to make -- so a duplicate oldID is treated as an ordinary
// name collision and both survive under distinct names. Only the FIRST
// oldID -> new-name mapping is kept for later reference resolution
// (provision_steps, environment topology), which matches the old system's
// own ambiguity: nothing could have unambiguously referenced the second
// definition by that ID either.
func (n *nameRegistry) assign(r *Result, namespace, oldID string) string {
	if n.byNamespace[namespace] == nil {
		n.byNamespace[namespace] = make(map[string]bool)
	}
	base := slug(oldID)
	name := base
	for i := 2; n.byNamespace[namespace][name]; i++ {
		name = fmt.Sprintf("%s-%d", base, i)
		r.note(oldID, "warn", fmt.Sprintf("name collision converting to the %q namespace: %q was already taken, renamed to %q -- check whether these were actually meant to be the same object", namespace, base, name))
	}
	n.byNamespace[namespace][name] = true
	if _, exists := n.oldToNew[oldID]; !exists {
		n.oldToNew[oldID] = name
	} else {
		r.note(oldID, "warn", fmt.Sprintf("old ID %q was used as the label for more than one block in the source repo -- this is itself a bug in the old content (likely a copy-pasted file whose label was never updated); the duplicate was kept as %q but anything that referenced %q by ID resolves to whichever definition was seen first", oldID, name, oldID))
	}
	return name
}

func (n *nameRegistry) lookup(oldID string) (string, bool) {
	v, ok := n.oldToNew[oldID]
	return v, ok
}

// --- network ---

func (r *Result) convertNetwork(b *Block, names *nameRegistry) {
	name := names.assign(r, "network", b.label())
	net := loader.Network{
		Name: name,
		CIDR: b.str("cidr"),
		Vars: b.strMap("vars"),
	}
	// Old HCL's boolean vdi_visible=true meant "reachable from the VDI network";
	// carry that forward as visible_from: [vdi], the closest equivalent in the
	// new allowlist model.
	if b.boolean("vdi_visible") {
		net.VisibleFrom = []string{"vdi"}
	}
	r.Networks = append(r.Networks, net)
}

// --- identity -> a row in a people/<group>.csv ---

func (r *Result) convertIdentity(b *Block) {
	// identities/<group>/<name>.laforge -> group "employees"/"rooms"/"customers"
	parts := strings.Split(strings.Trim(b.SourceFile, "/"), "/")
	group := "people"
	if len(parts) >= 2 && parts[0] == "identities" {
		group = parts[1]
	}

	vars, _ := b.Attrs["vars"].(map[string]interface{})
	get := func(k string) string {
		if v, ok := vars[k]; ok {
			return fmt.Sprintf("%v", v)
		}
		return ""
	}
	username := get("username")
	if username == "" {
		// Every real identity block has vars.username, but nothing
		// guarantees a future one will -- fall back rather than emit a
		// row with no username, which the new loader would reject outright.
		username = slug(b.label())
		r.note(b.SourceFile, "warn", fmt.Sprintf("identity %q has no vars.username, derived %q from its ID instead -- verify this is a sensible login name", b.label(), username))
	}

	attrs := map[string]string{
		"title":        get("title"),
		"department":   get("ou"),
		"domain_admin": fmt.Sprintf("%v", vars["is_ad_dom"]),
		"sudo":         fmt.Sprintf("%v", vars["is_nix_sudo"]),
		"address":      get("address"),
		"comment":      get("comment"),
		"full_name":    get("fullname"),
		"description":  b.str("description"),
		"enabled":      "true",
	}

	r.People[group] = append(r.People[group], loader.Person{
		Username: username,
		Attributes: mergeStr(attrs, map[string]string{
			"first_name": b.str("firstname"),
			"last_name":  b.str("lastname"),
			"email":      b.str("email"),
			"password":   b.str("password"),
		}),
	})
}

func mergeStr(a, b map[string]string) map[string]string {
	out := make(map[string]string, len(a)+len(b))
	for k, v := range a {
		out[k] = v
	}
	for k, v := range b {
		out[k] = v
	}
	return out
}

// --- script ---

var languageMap = map[string]string{
	"shell":      "bash",
	"bash":       "bash",
	"powershell": "powershell",
	"batch":      "batch",
}

func (r *Result) convertScript(repoRoot string, b *Block, names *nameRegistry) {
	name := names.assign(r, "script", b.label())

	lang := languageMap[strings.ToLower(b.str("language"))]
	if lang == "" {
		r.note(b.SourceFile, "warn", fmt.Sprintf("script %q has unrecognized language %q, left blank -- set it by hand", b.label(), b.str("language")))
	}

	if st := b.str("source_type"); st != "" && st != "local" {
		r.note(b.SourceFile, "warn", fmt.Sprintf("script %q has source_type %q, only \"local\" is supported by the converter (and the only value seen in real content) -- source file was not copied, fix by hand", b.label(), st))
	} else {
		// Resolve the real source file next to this script's .laforge, so
		// Write can copy it byte-for-byte -- "scripts carry over untouched."
		r.scriptSourceOnDisk[name] = joinRel(repoRoot, b.SourceFile, b.str("source"))
	}

	tags := b.strMap("tags")
	if maints := b.childrenOfType("maintainer"); len(maints) > 0 {
		tags = withTag(tags, "maintainer", maints[0].label())
	}
	if cd, ok := b.number("cooldown"); ok && cd != 0 {
		tags = withTag(tags, "cooldown_seconds", fmt.Sprintf("%v", cd))
		r.note(b.SourceFile, "info", fmt.Sprintf("script %q had a cooldown, no equivalent field in the new model -- kept as tag cooldown_seconds", b.label()))
	}
	if b.boolean("disabled") {
		tags = withTag(tags, "disabled", "true")
		r.note(b.SourceFile, "info", fmt.Sprintf("script %q was disabled=true in the old model -- converted anyway with tag disabled=true, since the new model has no disabled field; remove it from any steps that use it if it should stay unused", b.label()))
	}

	timeout := 0
	if v, ok := b.number("timeout"); ok {
		timeout = int(v)
	}

	r.Scripts = append(r.Scripts, loader.Script{
		Name:         name,
		Description:  b.str("description"),
		Language:     lang,
		Source:       b.str("source"),
		Timeout:      timeout,
		IgnoreErrors: b.boolean("ignore_errors"),
		Tags:         tags,
	})
}

func withTag(tags map[string]string, k, v string) map[string]string {
	if tags == nil {
		tags = map[string]string{}
	}
	tags[k] = v
	return tags
}

func joinRel(repoRoot, hclFileRel, sourceRel string) string {
	dir := hclFileRel
	if i := strings.LastIndex(dir, "/"); i >= 0 {
		dir = dir[:i]
	} else {
		dir = ""
	}
	return repoRoot + "/" + dir + "/" + strings.TrimPrefix(sourceRel, "./")
}

// --- host ---

// hostMeta is the addressing information the OLD model bakes directly into
// a host block, which the NEW model deliberately moved out of the host
// definition and into the environment's placement. Kept alongside the registry so
// convertEnvironment can look it up per included host.
type hostMeta struct {
	hostname  string
	lastOctet int
}

var hostMetaByOldID = map[string]hostMeta{}

func (r *Result) convertHost(b *Block, names *nameRegistry, commands map[string]legacyCommand, byID map[string]*Block) {
	name := names.assign(r, "host_or_container", b.label())

	octet := 0
	if v, ok := b.number("last_octet"); ok {
		octet = int(v)
	}
	hostMetaByOldID[b.label()] = hostMeta{hostname: b.str("hostname"), lastOctet: octet}

	var diskGB int
	for _, d := range b.childrenOfType("disk") {
		if v, ok := d.number("size"); ok {
			diskGB = int(v)
		}
	}

	tags := b.strMap("tags")
	if maints := b.childrenOfType("maintainer"); len(maints) > 0 {
		tags = withTag(tags, "maintainer", maints[0].label())
	}
	if desc := b.str("description"); desc != "" {
		tags = withTag(tags, "description", desc)
	}

	vars := b.strMap("vars")
	if op := b.str("override_password"); op != "" {
		vars = withTag(vars, "override_password", op)
		r.note(b.SourceFile, "info", fmt.Sprintf("host %q had override_password, no dedicated field in the new model -- kept as vars.override_password", b.label()))
	}

	var ports loader.Ports
	ports.TCP = b.strList("exposed_tcp_ports")
	ports.UDP = b.strList("exposed_udp_ports")

	steps := r.convertProvisionSteps(b, commands, byID, names)

	r.Hosts = append(r.Hosts, loader.Host{
		Name:  name,
		OS:    b.str("os"),
		Size:  b.str("instance_size"),
		Disk:  diskGB,
		Ports: ports,
		Steps: steps,
		Vars:  vars,
		Tags:  tags,
	})
}

var rebootProgramRe = regexp.MustCompile(`(?i)^(shutdown|reboot)$`)

func (r *Result) convertProvisionSteps(b *Block, commands map[string]legacyCommand, byID map[string]*Block, names *nameRegistry) []loader.Step {
	var steps []loader.Step
	for _, ref := range b.strList("provision_steps") {
		switch {
		case strings.HasPrefix(ref, "/scripts/"):
			if scriptBlock, ok := byID[ref]; ok {
				scriptName, _ := names.lookup(scriptBlock.label())
				steps = append(steps, loader.Step{"script": scriptName})
			} else {
				r.note(b.SourceFile, "warn", fmt.Sprintf("host %q references script %q, which was never defined -- step dropped", b.label(), ref))
			}
		case strings.HasPrefix(ref, "/commands/"):
			cmd, ok := commands[ref]
			if !ok {
				r.note(b.SourceFile, "warn", fmt.Sprintf("host %q references command %q, which was never defined -- step dropped", b.label(), ref))
				continue
			}
			if rebootProgramRe.MatchString(cmd.program) {
				steps = append(steps, loader.Step{"reboot": map[string]interface{}{}})
			} else {
				steps = append(steps, loader.Step{"run": shellJoin(cmd.program, cmd.args)})
			}
			r.note(b.SourceFile, "info", fmt.Sprintf("host %q: inlined legacy command %q as a step (pre-agent v1.0 construct)", b.label(), ref))
		default:
			r.note(b.SourceFile, "warn", fmt.Sprintf("host %q has a provision_steps entry %q that isn't a script or command reference -- step dropped", b.label(), ref))
		}
	}
	return steps
}

func shellJoin(program string, args []string) string {
	parts := append([]string{program}, args...)
	for i, p := range parts {
		if strings.ContainsAny(p, " \t\"'") {
			parts[i] = fmt.Sprintf("%q", p)
		}
	}
	return strings.Join(parts, " ")
}

// --- competition + environment -> environment ---

func (r *Result) convertEnvironment(envBlock *Block, allBlocks []*Block, byID map[string]*Block, names *nameRegistry) {
	name := envBlock.str("name")
	if name == "" {
		name = slug(envBlock.label())
	}

	var competition *Block
	compID := envBlock.str("competition_id")
	for _, b := range allBlocks {
		if b.Type == "competition" && b.label() == compID {
			competition = b
			break
		}
	}

	var rootPassword string
	var dns *loader.DNS
	if competition != nil {
		rootPassword = competition.str("root_password")
		if dnsBlocks := competition.childrenOfType("dns"); len(dnsBlocks) > 0 {
			d := dnsBlocks[0]
			dns = &loader.DNS{
				Type:       d.str("type"),
				RootDomain: d.str("root_domain"),
				DNSServers: d.strList("dns_servers"),
				NTPServers: d.strList("ntp_servers"),
			}
		}
	} else {
		r.note(envBlock.SourceFile, "warn", fmt.Sprintf("environment %q references competition %q, which was never found -- root_password and dns left blank", envBlock.label(), compID))
	}

	vars := envBlock.strMap("config")
	if len(vars) > 0 {
		r.note(envBlock.SourceFile, "info", fmt.Sprintf("environment %q's old `config` map became `vars` -- some of these (vpc_cidr, public_cidr, DNS server IPs, laforge_server_url, ...) look like per-hoster settings that may belong in a server-side builder config instead of environment vars; review them", envBlock.label()))
	}

	if builder := envBlock.str("builder"); builder != "" {
		r.note(envBlock.SourceFile, "info", fmt.Sprintf("environment %q named builder %q -- dropped: environment files never name a builder in the new model, it's chosen at build time", envBlock.label(), builder))
	}

	tags := map[string]string{}
	if maints := envBlock.childrenOfType("maintainer"); len(maints) > 0 {
		tags["maintainer"] = maints[0].label()
	}

	teams := 1
	if v, ok := envBlock.number("team_count"); ok {
		teams = int(v)
	}

	topology := make(map[string]map[string][]loader.Copy)
	for _, inc := range envBlock.childrenOfType("included_network") {
		netOldID := inc.label()
		netName, ok := names.lookup(netOldID)
		if !ok {
			r.note(envBlock.SourceFile, "warn", fmt.Sprintf("environment %q includes network %q, which was never defined -- skipped", envBlock.label(), netOldID))
			continue
		}
		if topology[netName] == nil {
			topology[netName] = make(map[string][]loader.Copy)
		}
		for _, hostOldID := range inc.strList("included_hosts") {
			objName, ok := names.lookup(hostOldID)
			if !ok {
				r.note(envBlock.SourceFile, "warn", fmt.Sprintf("environment %q includes host %q, which was never defined -- skipped", envBlock.label(), hostOldID))
				continue
			}
			meta := hostMetaByOldID[hostOldID]
			as := meta.hostname
			if as == "" {
				as = objName
			}
			topology[netName][objName] = append(topology[netName][objName], loader.Copy{
				As:        as,
				LastOctet: meta.lastOctet,
			})
		}
	}

	r.Environments = append(r.Environments, loader.Environment{
		Name:     name,
		Schema:   1,
		Teams:    teams,
		RootPass: rootPassword,
		DNS:      dns,
		Vars:     vars,
		Tags:     tags,
		Networks: topology,
	})
}
