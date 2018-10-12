package core

import (
	"encoding/binary"
	"fmt"
	"net"
	"strings"

	"github.com/pkg/errors"
)

// Host defines a configurable type for customizing host parameters within the infrastructure.
type Host struct {
	ID               string                 `hcl:",label" json:"id,omitempty"`
	Hostname         string                 `hcl:"hostname,attr" cty:"hostname" json:"hostname,omitempty"`
	Description      string                 `hcl:"description,attr" json:"description,omitempty"`
	OS               string                 `hcl:"os,attr" json:"os,omitempty"`
	AMI              string                 `hcl:"ami,attr" json:"ami,omitempty"`
	LastOctet        int                    `hcl:"last_octet,attr" json:"last_octet,omitempty"`
	InstanceSize     string                 `hcl:"instance_size,attr" json:"instance_size,omitempty"`
	Disk             Disk                   `hcl:"disk,block" json:"disk,omitempty"`
	ProvisionSteps   []string               `hcl:"provision_steps,attr" json:"provision_steps,omitempty"`
	Provisioners     []Provisioner          `json:"-"`
	ExposedTCPPorts  []string               `hcl:"exposed_tcp_ports,attr" json:"exposed_tcp_ports,omitempty"`
	ExposedUDPPorts  []string               `hcl:"exposed_udp_ports,attr" json:"exposed_udp_ports,omitempty"`
	OverridePassword string                 `hcl:"override_password,attr" json:"override_password,omitempty"`
	UserGroups       []string               `hcl:"user_groups,attr" json:"user_groups,omitempty"`
	Dependencies     []*HostDependency      `hcl:"depends_on,block" json:"dependencies,omitempty"`
	IO               IO                     `hcl:"io,block" json:"io,omitempty"`
	Vars             map[string]string      `hcl:"vars,attr" json:"vars,omitempty"`
	Tags             map[string]string      `hcl:"tags,attr" json:"tags,omitempty"`
	Maintainer       *User                  `hcl:"maintainer,block" json:"maintainer,omitempty"`
	OnConflict       OnConflict             `hcl:"on_conflict,block" json:"on_conflict,omitempty"`
	Caller           Caller                 `json:"-"`
	Scripts          map[string]*Script     `json:"-"`
	Commands         map[string]*Command    `json:"-"`
	Files            map[string]*RemoteFile `json:"-"`
	DNSRecords       map[string]*DNSRecord  `json:"-"`
}

// Disk is a configurable type for setting the root volume's disk size in GB
type Disk struct {
	Size int `hcl:"size,attr" json:"size,omitempty"`
}

// HostDependency is a configurable type for defining host or network dependencies to allow a dependency graph to be honored during deployment
type HostDependency struct {
	HostID     string     `hcl:"host,attr" json:"host,omitempty"`
	NetworkID  string     `hcl:"network,attr" json:"network,omitempty"`
	Host       *Host      `json:"-"`
	Network    *Network   `json:"-"`
	Step       string     `hcl:"step,attr" json:"step,omitempty"`
	StepID     int        `json:"step_id,omitempty"`
	OnConflict OnConflict `hcl:"on_conflict,block" json:"on_conflict,omitempty"`
}

// HasTag is a template helper function to return true/false if the host contains a tag of a specific key
func (h *Host) HasTag(tag string) bool {
	_, t := h.Tags[tag]
	return t
}

// TagEquals is a template helper function to return true/false if the host contains a tag key of a specific value
func (h *Host) TagEquals(tag, value string) bool {
	v, t := h.Tags[tag]
	if !t {
		return false
	}
	if v == value {
		return true
	}
	return false
}

// FinalStepID gets the final step ID for a host's offset
func (h *Host) FinalStepID() int {
	if len(h.Provisioners) == 0 {
		return -1
	}
	return len(h.Provisioners) - 1
}

// GetCaller implements the Mergeable interface
func (h *Host) GetCaller() Caller {
	return h.Caller
}

// GetID implements the Mergeable interface
func (h *Host) GetID() string {
	return h.ID
}

// GetOnConflict implements the Mergeable interface
func (h *Host) GetOnConflict() OnConflict {
	return h.OnConflict
}

// SetCaller implements the Mergeable interface
func (h *Host) SetCaller(c Caller) {
	h.Caller = c
}

// SetOnConflict implements the Mergeable interface
func (h *Host) SetOnConflict(o OnConflict) {
	h.OnConflict = o
}

// Swap implements the Mergeable interface
func (h *Host) Swap(m Mergeable) error {
	rawVal, ok := m.(*Host)
	if !ok {
		return errors.Wrapf(ErrSwapTypeMismatch, "expected %T, got %T", h, m)
	}
	*h = *rawVal
	return nil
}

// CalcIP is used to calculate the IP of a host within a given subnet
func (h *Host) CalcIP(subnet string) string {
	ip, _, err := net.ParseCIDR(subnet)
	if err != nil {
		return fmt.Sprintf("ERR_INVALID_SUBNET_%s_FOR_HOST_%s", subnet, h.ID)
	}
	offset32 := uint32(h.LastOctet)
	ip32 := IPv42Int(ip)
	newIP := Int2IPv4(ip32 + offset32)
	return newIP.To4().String()
}

// IsWindows is a template helper function to determine if the underlying operating system is windows
func (h *Host) IsWindows() bool {
	switch strings.ToLower(h.OS) {
	case "w2k3":
		return true
	case "w2k8":
		return true
	case "w2k12":
		return true
	case "w2k16":
		return true
	case "windows":
		return true
	default:
		return false
	}
}

// Index attempts to index all children dependencies of this type
func (h *Host) Index(base *Laforge) error {
	h.Scripts = map[string]*Script{}
	h.Commands = map[string]*Command{}
	h.Files = map[string]*RemoteFile{}
	h.DNSRecords = map[string]*DNSRecord{}
	iprov := map[string]string{}
	h.Provisioners = []Provisioner{}

	for _, s := range h.ProvisionSteps {
		Logger.Debugf("indexing provision step %s for host %s", s, h.ID)
		iprov[s] = "included"
	}
	for name, script := range base.Scripts {
		status, found := iprov[name]
		if !found {
			continue
		}
		if status == "included" {
			h.Scripts[name] = script
			iprov[name] = "script"
			Logger.Debugf("Resolved %T dependency %s for %s", script, script.ID, h.ID)
		}
	}
	for name, command := range base.Commands {
		status, found := iprov[name]
		if !found {
			continue
		}
		if status == "included" {
			h.Commands[name] = command
			iprov[name] = "command"
			Logger.Debugf("Resolved %T dependency %s for %s", command, command.ID, h.ID)
		}
	}
	for name, file := range base.Files {
		status, found := iprov[name]
		if !found {
			continue
		}
		if status == "included" {
			h.Files[name] = file
			iprov[name] = "remote_file"
			Logger.Debugf("Resolved %T dependency %s for %s", file, file.ID, h.ID)
		}
	}
	for name, record := range base.DNSRecords {
		status, found := iprov[name]
		if !found {
			continue
		}
		if status == "included" {
			h.DNSRecords[name] = record
			iprov[name] = "dns_record"
			Logger.Debugf("Resolved %T dependency %s for %s", record, record.ID, h.ID)
		}
	}
	for x, status := range iprov {
		if status == "included" {
			return fmt.Errorf("unmet provision_step dependency %s for host %s\n%s", x, h.ID, h.Caller.Error())
		}
	}
	for _, s := range h.ProvisionSteps {
		switch iprov[s] {
		case "script":
			h.Provisioners = append(h.Provisioners, h.Scripts[s])
		case "command":
			h.Provisioners = append(h.Provisioners, h.Commands[s])
		case "remote_file":
			h.Provisioners = append(h.Provisioners, h.Files[s])
		case "dns_record":
			h.Provisioners = append(h.Provisioners, h.DNSRecords[s])
		default:
			return fmt.Errorf("unmet provision_step dependency %s for host %s\n%s", s, h.ID, h.Caller.Error())
		}
	}
	return nil
}

// IPv42Int converts net.IP address objects to their uint32 representation
func IPv42Int(ip net.IP) uint32 {
	if len(ip) == 16 {
		return binary.BigEndian.Uint32(ip[12:16])
	}
	return binary.BigEndian.Uint32(ip)
}

// Int2IPv4 converts uint32s to their net.IP object
func Int2IPv4(nn uint32) net.IP {
	ip := make(net.IP, 4)
	binary.BigEndian.PutUint32(ip, nn)
	return ip
}
