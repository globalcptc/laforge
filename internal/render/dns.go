package render

import (
	"fmt"
	"sort"

	"github.com/globalcptc/laforge/internal/loader"
)

// DNSRecord is one resolved DNS record -- either one of the automatic A
// records every copy gets, or one of the environment's own dns.records
// entries. Mirrors builder.DNSRecordSpec's shape (see internal/builder)
// but is defined here, not imported: this package computes DNS records
// from content the same way it computes everything else (Address, Steps,
// Vars) with no dependency on internal/builder, and both
// internal/orchestrator and internal/runner already copy these four
// fields into their own builder.DNSRecordSpec at the call site rather
// than share a type across that layering boundary.
type DNSRecord struct {
	Name     string
	Type     string
	Value    string
	Priority int
}

// DNSRecords produces the automatic DNS records: "A records for every host are
// generated automatically, with custom records added in the environment's
// dns block." Team-invariant -- a copy's address is the network's CIDR
// plus its own last_octet (see Address), which never depends on team
// number (only which of that team's own isolated networks it lands on,
// which every team has an identical copy of -- "The CIDR is fixed and
// identical for every team"). So this is computed once per environment,
// not once per team, and the caller (internal/orchestrator's reconcileDNS,
// internal/runner's executeDeploy) applies the identical record set to
// each team's own DNS write, the same way DeployNetwork is called once per
// team with the one CIDR every team's network definition carries.
func DNSRecords(c *loader.Content, env *loader.Environment) ([]DNSRecord, error) {
	var records []DNSRecord
	var networkNames []string
	for n := range env.Networks {
		networkNames = append(networkNames, n)
	}
	sort.Strings(networkNames)
	for _, networkName := range networkNames {
		network := findNetwork(c, networkName)
		if network == nil {
			return nil, fmt.Errorf("environment %q references network %q, which does not exist", env.Name, networkName)
		}
		var objNames []string
		for o := range env.Networks[networkName] {
			objNames = append(objNames, o)
		}
		sort.Strings(objNames)
		for _, objName := range objNames {
			for _, cp := range env.Networks[networkName][objName] {
				addr, err := Address(network.CIDR, cp.LastOctet)
				if err != nil {
					return nil, fmt.Errorf("computing address for %q: %w", cp.As, err)
				}
				records = append(records, DNSRecord{Name: cp.As, Type: "A", Value: addr})
			}
		}
	}
	if env.DNS != nil {
		for _, r := range env.DNS.Records {
			records = append(records, DNSRecord{Name: r.Name, Type: r.Type, Value: r.Target, Priority: r.Priority})
		}
	}
	return records, nil
}
