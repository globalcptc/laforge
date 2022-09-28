package planner

import (
	"bytes"
	"net"
	"path"
	"strconv"
	"strings"
	"text/template"
	"unicode/utf8"

	"github.com/bradfitz/iter"
	"github.com/gen0cide/laforge/ent"
	"github.com/iancoleman/strcase"
	"github.com/sirupsen/logrus"
)

type TempleteContext struct {
	Build              *ent.Build              `yaml:"build,omitempty"`
	Competition        *ent.Competition        `yaml:"competition,omitempty"`
	Environment        *ent.Environment        `yaml:"environment,omitempty"`
	Host               *ent.Host               `yaml:"host,omitempty"`
	DNS                *ent.DNS                `yaml:"dns,omitempty"`
	DNSRecords         []*ent.DNSRecord        `yaml:"dns_records,omitempty"`
	IncludedNetworks   []*ent.IncludedNetwork  `yaml:"included_networks,omitempty"`
	Network            *ent.Network            `yaml:"network,omitempty"`
	Script             *ent.Script             `yaml:"script,omitempty"`
	Team               *ent.Team               `yaml:"team,omitempty"`
	Identities         []*ent.Identity         `yaml:"identities,omitempty"`
	ProvisionedNetwork *ent.ProvisionedNetwork `yaml:"provisioned_network,omitempty"`
	ProvisionedHost    *ent.ProvisionedHost    `yaml:"provisioned_host,omitempty"`
	ProvisioningStep   *ent.ProvisioningStep   `yaml:"provisioning_step,omitempty"`
	AgentSlug          string                  `yaml:"agent_slug,omitempty"`
	Ansible            *ent.Ansible            `yaml:"ansible,omitempty"`
}

// TemplateFuncLib is a standard template library of functions
var TemplateFuncLib = template.FuncMap{
	"hclstring":            QuotedHCLString,
	"N":                    iter.N,
	"UnsafeAtoi":           UnsafeStringAsInt,
	"Decr":                 Decr,
	"ToUpper":              strings.ToUpper,
	"Contains":             strings.Contains,
	"HasPrefix":            strings.HasPrefix,
	"HasSuffix":            strings.HasSuffix,
	"Join":                 strings.Join,
	"Replace":              strings.Replace,
	"Repeat":               strings.Repeat,
	"Split":                strings.Split,
	"Title":                strings.Title,
	"ToLower":              strings.ToLower,
	"ToSnake":              strcase.ToSnake,
	"ToScreamingSnake":     strcase.ToScreamingSnake,
	"ToKebab":              strcase.ToKebab,
	"ToScreamingKebab":     strcase.ToScreamingKebab,
	"ToDelimited":          strcase.ToDelimited,
	"ToScreamingDelimited": strcase.ToScreamingDelimited,
	"ToCamel":              strcase.ToCamel,
	"ToLowerCamel":         strcase.ToLowerCamel,
	"Incr":                 Incr,
	"CalcIP":               CalcIP,
	"TagEquals":            TagEquals,
	"Octet":                Octet,
	"Base":                 path.Base,
}

// Octet is a template helper function to get a network's octet at a specified offset
func Octet(n *ent.Network, octet int) string {
	// Check if we've got the CIDR itself before we make errors
	if n.Cidr == "" {
		return "NO_CIDR"
	}

	// Split the IP.IP.IP.IP/MASK on the dots
	octets := strings.Split(n.Cidr, ".")

	// We should have 4 things, if not, it's wrong
	if len(octets) <= 3 {
		return "INVALID_CIDR"
	}

	// If we are pulling the last octet, we'll also have the CIDR range, which is a problem.
	// If not, then we can just return the result
	if octet == 4 {
		last := strings.Split(octets[octet-1], "/")
		return last[0]
	} else {
		return octets[octet-1]
	}
}

// CIDR is a template helper function to get the subnet off a CIDR range
func CIDR(n *ent.Network) string {
	// If we don't even have the property, there's a problem
	if n.Cidr == "" {
		return "NO_CIDR"
	}

	// Let's split on the / to get the mask
	cidr := strings.Split(n.Cidr, "/")

	// We should have two things, if we don't, there's a problem
	if len(octets) <= 2 {
		return "INVALID_CIDR"
	}

	// Now we can return it.
	return cidr[len(cidr)-1]
}

func TagEquals(h *ent.Host, tag, value string) bool {
	v, t := h.Tags[tag]
	if !t {
		return false
	}
	if v == value {
		return true
	}
	return false
}

// CalcIP is used to calculate the IP of a host within a given subnet
func CalcIP(subnet string, lastOctect int) (string, error) {
	ip, _, err := net.ParseCIDR(subnet)
	if err != nil {
		logrus.Errorf("Invalid Subnet %v. Err: %v", subnet, err)
		return "", err
	}
	offset32 := uint32(lastOctect)
	ip32 := IPv42Int(ip)
	newIP := Int2IPv4(ip32 + offset32)
	return newIP.To4().String(), nil
}

// UnsafeStringAsInt is a template helper function that will return -1 if it cannot convert the string to an integer.
func UnsafeStringAsInt(s string) int {
	i, err := strconv.Atoi(s)
	if err != nil {
		return -1
	}
	return i
}

// Decr is a template helper function to non-destructively decrement an integer
func Decr(i int) int {
	return i - 1
}

// Incr is a template helper function to non-destructively increment an integer
func Incr(i int) int {
	return i + 1
}

// QuotedHCLString is a template function to render safe HCLv2 strings
func QuotedHCLString(s string) string {
	e := new(bytes.Buffer)

	//nolint:gosec,errcheck
	e.WriteByte('"')

	start := 0
	for i := 0; i < len(s); {
		if b := s[i]; b < utf8.RuneSelf {
			if htmlSafeSet[b] {
				i++
				continue
			}
			if start < i {

				//nolint:gosec,errcheck
				e.WriteString(s[start:i])

			}
			switch b {
			case '\\', '"':

				//nolint:gosec,errcheck
				e.WriteByte('\\')

				//nolint:gosec,errcheck
				e.WriteByte(b)

			case '\n':

				//nolint:gosec,errcheck
				e.WriteByte('\\')

				//nolint:gosec,errcheck
				e.WriteByte('n')

			case '\r':

				//nolint:gosec,errcheck
				e.WriteByte('\\')

				//nolint:gosec,errcheck
				e.WriteByte('r')

			case '\t':

				//nolint:gosec,errcheck
				e.WriteByte('\\')

				//nolint:gosec,errcheck
				e.WriteByte('t')
			default:
				// This encodes bytes < 0x20 except for \t, \n and \r.
				// If escapeHTML is set, it also escapes <, >, and &
				// because they can lead to security holes when
				// user-controlled strings are rendered into JSON
				// and served to some browsers.
				//nolint:gosec,errcheck
				e.WriteString(`\u00`)

				//nolint:gosec,errcheck
				e.WriteByte(hex[b>>4])

				//nolint:gosec,errcheck
				e.WriteByte(hex[b&0xF])
			}
			i++
			start = i
			continue
		}
		c, size := utf8.DecodeRuneInString(s[i:])
		if c == utf8.RuneError && size == 1 {
			if start < i {
				//nolint:gosec,errcheck
				e.WriteString(s[start:i])
			}
			//nolint:gosec,errcheck
			e.WriteString(`\ufffd`)
			i += size
			start = i
			continue
		}

		if c == '\u2028' || c == '\u2029' {
			if start < i {
				//nolint:gosec,errcheck
				e.WriteString(s[start:i])
			}
			//nolint:gosec,errcheck
			e.WriteString(`\u202`)
			//nolint:gosec,errcheck
			e.WriteByte(hex[c&0xF])
			i += size
			start = i
			continue
		}
		i += size
	}
	if start < len(s) {
		//nolint:gosec,errcheck
		e.WriteString(s[start:])
	}
	//nolint:gosec,errcheck
	e.WriteByte('"')
	return e.String()
}

var hex = "0123456789abcdef"

var htmlSafeSet = [utf8.RuneSelf]bool{
	' ':  true,
	'!':  true,
	'"':  false,
	'#':  true,
	'$':  true,
	'%':  true,
	'&':  false,
	'\'': true,
	'(':  true,
	')':  true,
	'*':  true,
	'+':  true,
	',':  true,
	'-':  true,
	'.':  true,
	'/':  true,
	'0':  true,
	'1':  true,
	'2':  true,
	'3':  true,
	'4':  true,
	'5':  true,
	'6':  true,
	'7':  true,
	'8':  true,
	'9':  true,
	':':  true,
	';':  true,
	'<':  false,
	'=':  true,
	'>':  false,
	'?':  true,
	'@':  true,
	'A':  true,
	'B':  true,
	'C':  true,
	'D':  true,
	'E':  true,
	'F':  true,
	'G':  true,
	'H':  true,
	'I':  true,
	'J':  true,
	'K':  true,
	'L':  true,
	'M':  true,
	'N':  true,
	'O':  true,
	'P':  true,
	'Q':  true,
	'R':  true,
	'S':  true,
	'T':  true,
	'U':  true,
	'V':  true,
	'W':  true,
	'X':  true,
	'Y':  true,
	'Z':  true,
	'[':  true,
	'\\': false,
	']':  true,
	'^':  true,
	'_':  true,
	'`':  true,
	'a':  true,
	'b':  true,
	'c':  true,
	'd':  true,
	'e':  true,
	'f':  true,
	'g':  true,
	'h':  true,
	'i':  true,
	'j':  true,
	'k':  true,
	'l':  true,
	'm':  true,
	'n':  true,
	'o':  true,
	'p':  true,
	'q':  true,
	'r':  true,
	's':  true,
	't':  true,
	'u':  true,
	'v':  true,
	'w':  true,
	'x':  true,
	'y':  true,
	'z':  true,
	'{':  true,
	'|':  true,
	'}':  true,
	'~':  true,
}
