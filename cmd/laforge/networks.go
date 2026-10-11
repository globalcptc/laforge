package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"sort"

	"github.com/globalcptc/laforge/internal/loader"
	"github.com/globalcptc/laforge/internal/render"
)

// runNetworks prints an environment's full networking layout as a console tree:
// the environment's overall cidr, each network it places (with that network's
// cidr), and under each network every placed host/container copy with its last
// octet and resolved address. Everything is derived from the content, no builder
// or account needed -- it's the companion to `laforge stats`, answering "what
// address does each box get, and does the layout fit the declared space" at a
// glance.
func runNetworks(args []string) error {
	fs := flag.NewFlagSet("networks", flag.ExitOnError)
	repo := repoFlag(fs)
	envFlag := fs.String("environment", "", "environment to show; default shows every environment")
	fs.Parse(args)

	// Lenient load, same as stats: a repo with stray non-content YAML (or even
	// content errors) can still have its networking shown for what it defines.
	c, err := loader.Load(*repo)
	if err != nil {
		return err
	}
	if len(c.Environments) == 0 {
		if len(c.Errors) > 0 {
			return fmt.Errorf("no environment found in %s -- laforge networks needs a content repo with at least one `environment:` document (%d file(s) there weren't LaForge content; run `laforge check` for detail)", *repo, len(c.Errors))
		}
		return fmt.Errorf("no environment found in %s -- laforge networks needs a content repo with at least one `environment:` document", *repo)
	}
	if len(c.Errors) > 0 {
		fmt.Fprintf(os.Stderr, "note: skipped %d YAML file(s) that aren't LaForge content; run `laforge check` for detail\n", len(c.Errors))
	}

	envs := c.Environments
	if *envFlag != "" {
		name, err := resolveEnvironmentName(c, *envFlag)
		if err != nil {
			return err
		}
		envs = nil
		for _, e := range c.Environments {
			if e.Name == name {
				envs = append(envs, e)
			}
		}
		if len(envs) == 0 {
			avail := make([]string, 0, len(c.Environments))
			for _, e := range c.Environments {
				avail = append(avail, e.Name)
			}
			sort.Strings(avail)
			return fmt.Errorf("no environment named %q in %s (available: %v)", name, *repo, avail)
		}
	}

	hostByName := make(map[string]*loader.Host, len(c.Hosts))
	for i := range c.Hosts {
		hostByName[c.Hosts[i].Name] = &c.Hosts[i]
	}
	containerByName := make(map[string]*loader.Container, len(c.Containers))
	for i := range c.Containers {
		containerByName[c.Containers[i].Name] = &c.Containers[i]
	}
	networkByName := make(map[string]*loader.Network, len(c.Networks))
	for i := range c.Networks {
		networkByName[c.Networks[i].Name] = &c.Networks[i]
	}

	for i, e := range envs {
		if i > 0 {
			fmt.Println()
		}
		printEnvNetworks(e, hostByName, containerByName, networkByName)
	}
	return nil
}

// netLeaf is one placed host/container copy under a network.
type netLeaf struct {
	as     string
	octet  int
	addr   string // resolved cidr+octet, or "" when the network has no usable cidr
	detail string // "host ubuntu22", "container nginx:alpine", "container (compose)"
}

func printEnvNetworks(e loader.Environment, hosts map[string]*loader.Host, containers map[string]*loader.Container, nets map[string]*loader.Network) {
	fmt.Printf("Environment: %s  (%d team%s)\n", e.Name, e.Teams, plural(e.Teams))
	if e.CIDR != "" {
		fmt.Printf("CIDR: %s\n", e.CIDR)
	} else {
		fmt.Println("CIDR: (none declared)")
	}

	netNames := make([]string, 0, len(e.Networks))
	for name := range e.Networks {
		netNames = append(netNames, name)
	}
	sort.Strings(netNames)
	if len(netNames) == 0 {
		fmt.Println("(no networks placed)")
		return
	}

	// Parse the env cidr once so each network can be flagged if it falls outside
	// the declared space (the same rule `laforge check` enforces, surfaced here
	// visually).
	var envNet *net.IPNet
	if e.CIDR != "" {
		if _, n, err := net.ParseCIDR(e.CIDR); err == nil {
			envNet = n
		}
	}

	// Gather leaves per network and size the as/octet columns across the whole
	// environment so every network's rows align.
	leaves := make(map[string][]netLeaf, len(netNames))
	maxAs, maxOctet := 0, 0
	for _, netName := range netNames {
		netCIDR := ""
		if n := nets[netName]; n != nil {
			netCIDR = n.CIDR
		}
		var ls []netLeaf
		for objName, copies := range e.Networks[netName] {
			detail := objectDetail(objName, hosts, containers)
			for _, cp := range copies {
				as := cp.As
				if as == "" {
					as = objName
				}
				addr := ""
				if netCIDR != "" {
					if a, err := render.Address(netCIDR, cp.LastOctet); err == nil {
						addr = a
					}
				}
				ls = append(ls, netLeaf{as: as, octet: cp.LastOctet, addr: addr, detail: detail})
				if len(as) > maxAs {
					maxAs = len(as)
				}
				if w := len(fmt.Sprintf(".%d", cp.LastOctet)); w > maxOctet {
					maxOctet = w
				}
			}
		}
		// Lowest octet first (reads like the address plan), ties by name.
		sort.Slice(ls, func(i, j int) bool {
			if ls[i].octet != ls[j].octet {
				return ls[i].octet < ls[j].octet
			}
			return ls[i].as < ls[j].as
		})
		leaves[netName] = ls
	}

	for i, netName := range netNames {
		branch, childPrefix := "├── ", "│   "
		if i == len(netNames)-1 {
			branch, childPrefix = "└── ", "    "
		}
		netCIDR := ""
		if n := nets[netName]; n != nil {
			netCIDR = n.CIDR
		}
		header := netName + "  "
		if netCIDR != "" {
			header += netCIDR
		} else {
			header += "(no cidr)"
		}
		if envNet != nil && netCIDR != "" && !cidrFits(envNet, netCIDR) {
			header += "  ⚠ outside environment cidr"
		}
		fmt.Printf("%s%s\n", branch, header)

		ls := leaves[netName]
		if len(ls) == 0 {
			fmt.Printf("%s└── (no hosts placed)\n", childPrefix)
			continue
		}
		for j, l := range ls {
			leafBranch := "├── "
			if j == len(ls)-1 {
				leafBranch = "└── "
			}
			addr := l.addr
			if addr == "" {
				addr = "(n/a)"
			}
			fmt.Printf("%s%s%-*s  %*s  → %-15s  %s\n",
				childPrefix, leafBranch,
				maxAs, l.as,
				maxOctet, fmt.Sprintf(".%d", l.octet),
				addr, l.detail)
		}
	}
}

// objectDetail describes a placed object: its kind and the one fact that matters
// for a networking view (a host's os image, a container's image or that it's a
// compose project). An object the environment places but no file defines is
// marked so it isn't silently blank (`laforge check` reports it properly).
func objectDetail(name string, hosts map[string]*loader.Host, containers map[string]*loader.Container) string {
	if h := hosts[name]; h != nil {
		return "host " + orUnset(h.OS)
	}
	if ct := containers[name]; ct != nil {
		if ct.Compose != "" {
			return "container (compose)"
		}
		return "container " + orUnset(ct.Image)
	}
	return "(undefined object)"
}

// cidrFits reports whether the network cidr `inner` sits entirely within outer
// (same or more specific). Mirrors loader.cidrContains; kept local so the CLI
// doesn't depend on a loader internal.
func cidrFits(outer *net.IPNet, inner string) bool {
	ip, n, err := net.ParseCIDR(inner)
	if err != nil || !outer.Contains(ip) {
		return false
	}
	oo, _ := outer.Mask.Size()
	io, _ := n.Mask.Size()
	return io >= oo
}
