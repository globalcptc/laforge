package aws

import (
	"strings"
	"testing"
)

// TestEvalReadiness checks the pure quota comparison: a dimension warns only when
// its limit is known (>0), the build needs some of it, and current+demand would
// exceed it; an unknown limit (0) is silently skipped so a missing
// servicequotas permission never false-warns.
func TestEvalReadiness(t *testing.T) {
	cases := []struct {
		name                       string
		wantVPCs, wantEIPs, wantVC int
		usage                      awsReadinessUsage
		limits                     awsReadinessLimits
		wantContains               []string
		wantNone                   bool
	}{
		{
			name:     "all within limits",
			wantVPCs: 3, wantEIPs: 2, wantVC: 8,
			usage:    awsReadinessUsage{vpcs: 1, eips: 1, vcpus: 10},
			limits:   awsReadinessLimits{vpcs: 5, eips: 5, vcpus: 64},
			wantNone: true,
		},
		{
			name:     "vpcs exceed",
			wantVPCs: 8,
			usage:    awsReadinessUsage{vpcs: 0},
			limits:   awsReadinessLimits{vpcs: 5},
			wantContains: []string{"VPCs (one per team)"},
		},
		{
			name:     "eips exceed with existing usage",
			wantEIPs: 3,
			usage:    awsReadinessUsage{eips: 3},
			limits:   awsReadinessLimits{eips: 5},
			wantContains: []string{"Elastic IPs"},
		},
		{
			name:   "vcpus exceed",
			wantVC: 40,
			usage:  awsReadinessUsage{vcpus: 30},
			limits: awsReadinessLimits{vcpus: 64},
			wantContains: []string{"instance vCPUs"},
		},
		{
			name:     "unknown limit is skipped",
			wantVPCs: 100,
			limits:   awsReadinessLimits{vpcs: 0},
			wantNone: true,
		},
		{
			name:     "zero demand is skipped",
			wantVPCs: 0,
			usage:    awsReadinessUsage{vpcs: 10},
			limits:   awsReadinessLimits{vpcs: 5},
			wantNone: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := evalReadiness(tc.wantVPCs, tc.wantEIPs, tc.wantVC, tc.usage, tc.limits)
			if tc.wantNone {
				if len(got) != 0 {
					t.Fatalf("want no warnings, got %v", got)
				}
				return
			}
			if len(got) != len(tc.wantContains) {
				t.Fatalf("want %d warning(s), got %d: %v", len(tc.wantContains), len(got), got)
			}
			for i, sub := range tc.wantContains {
				if !strings.Contains(got[i], sub) {
					t.Errorf("warning %d = %q, want it to contain %q", i, got[i], sub)
				}
			}
		})
	}
}
