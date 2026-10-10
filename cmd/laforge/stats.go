package main

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"text/tabwriter"

	"github.com/globalcptc/laforge/internal/loader"
)

// runStats renders a quick, at-a-glance profile of what an environment deploys:
// how many hosts/containers, broken down by os image, by size, by network, and
// how many are publicly exposed -- the numbers you want before you provision (an
// AMI/instance-type to map for every os/size, a sense of scale and blast radius).
// Everything is derived from the content, no builder or account needed.
func runStats(args []string) error {
	fs := flag.NewFlagSet("stats", flag.ExitOnError)
	repo := repoFlag(fs)
	envFlag := fs.String("environment", "", "environment to profile; default profiles every environment")
	fs.Parse(args)

	c, err := loadOrExplain(*repo)
	if err != nil {
		return err
	}
	if len(c.Environments) == 0 {
		return fmt.Errorf("no environments defined in %s", *repo)
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
	}

	hostByName := make(map[string]*loader.Host, len(c.Hosts))
	for i := range c.Hosts {
		hostByName[c.Hosts[i].Name] = &c.Hosts[i]
	}
	containerByName := make(map[string]*loader.Container, len(c.Containers))
	for i := range c.Containers {
		containerByName[c.Containers[i].Name] = &c.Containers[i]
	}

	for i, e := range envs {
		if i > 0 {
			fmt.Println()
		}
		printEnvStats(e, hostByName, containerByName)
	}
	return nil
}

// envStats is the per-team tally for one environment; totals are per-team × teams.
type envStats struct {
	teams                   int
	networks                int
	hosts, containers       int
	composeContainers       int
	publicHosts, publicCtrs int
	osCount                 map[string]int // host os -> per-team count
	imageCount              map[string]int // container image -> per-team count
	sizeCount               map[string]int // size -> per-team count
	perNetwork              map[string]int // network -> per-team instance count
}

func printEnvStats(e loader.Environment, hosts map[string]*loader.Host, containers map[string]*loader.Container) {
	s := envStats{
		teams: e.Teams, networks: len(e.Networks),
		osCount: map[string]int{}, imageCount: map[string]int{}, sizeCount: map[string]int{}, perNetwork: map[string]int{},
	}
	for netName, objs := range e.Networks {
		for objName, copies := range objs {
			n := len(copies) // instances of this object per team
			s.perNetwork[netName] += n
			if h := hosts[objName]; h != nil {
				s.hosts += n
				s.osCount[orUnset(h.OS)] += n
				s.sizeCount[orUnset(h.Size)] += n
				if hasPorts(h.Public) {
					s.publicHosts += n
				}
				continue
			}
			if ct := containers[objName]; ct != nil {
				s.containers += n
				if ct.Compose != "" {
					s.composeContainers += n
					s.imageCount["(compose project)"] += n
				} else {
					s.imageCount[orUnset(ct.Image)] += n
				}
				if ct.Size != "" {
					s.sizeCount[ct.Size] += n
				}
				if hasPorts(ct.Public) {
					s.publicCtrs += n
				}
			}
		}
	}

	teams := s.teams
	if teams < 1 {
		teams = 1
	}
	fmt.Printf("Environment: %s   (%d team%s)\n", e.Name, s.teams, plural(s.teams))

	w := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
	line := func(label string, perTeam int) {
		fmt.Fprintf(w, "  %s\t%d/team\t%d total\n", label, perTeam, perTeam*teams)
	}
	fmt.Fprintf(w, "  Networks\t%d\t\n", s.networks)
	line("Hosts", s.hosts)
	if s.containers > 0 {
		label := "Containers"
		if s.composeContainers > 0 {
			label = fmt.Sprintf("Containers (%d compose)", s.composeContainers)
		}
		line(label, s.containers)
	}
	line("Public hosts", s.publicHosts)
	if s.publicCtrs > 0 {
		line("Public containers", s.publicCtrs)
	}
	w.Flush()

	printBreakdown("Images (host os)", s.osCount, teams)
	if len(s.imageCount) > 0 {
		printBreakdown("Container images", s.imageCount, teams)
	}
	printBreakdown("Sizes", s.sizeCount, teams)
	printBreakdown("Per network", s.perNetwork, teams)
}

// printBreakdown prints a sorted count map with per-team and total columns.
func printBreakdown(title string, counts map[string]int, teams int) {
	if len(counts) == 0 {
		return
	}
	keys := make([]string, 0, len(counts))
	for k := range counts {
		keys = append(keys, k)
	}
	// Most-used first, ties alphabetical.
	sort.Slice(keys, func(i, j int) bool {
		if counts[keys[i]] != counts[keys[j]] {
			return counts[keys[i]] > counts[keys[j]]
		}
		return keys[i] < keys[j]
	})
	fmt.Printf("  %s:\n", title)
	w := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
	for _, k := range keys {
		fmt.Fprintf(w, "    %s\t%d/team\t%d total\n", k, counts[k], counts[k]*teams)
	}
	w.Flush()
}

func hasPorts(p *loader.Ports) bool {
	return p != nil && (len(p.TCP) > 0 || len(p.UDP) > 0)
}

func orUnset(s string) string {
	if s == "" {
		return "(unset)"
	}
	return s
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}
