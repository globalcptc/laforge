// Pre-deploy readiness: before a build is deployed, estimate what it will need
// from its builder (teams -> isolated networks/VPCs, public: hosts -> external
// addresses/EIPs, host sizes -> instance vCPUs) and ask the builder whether its
// backend's quotas can take it. AWS is the only builder that answers today (it's
// the one hoster where a quota wall silently breaks a deploy); every other kind
// has no ReadinessChecker and returns an empty, clear verdict. Read-only and
// side-effect-free -- it's advisory, surfaced in the UI next to the Deploy
// action, never a gate. See docs/plans/builder-monitoring.md.
package api

import (
	"errors"
	"net/http"

	"github.com/globalcptc/laforge/internal/builder"
	"github.com/globalcptc/laforge/internal/builderconfig"
	"github.com/globalcptc/laforge/internal/loader"
)

type readinessResponse struct {
	// Checked is false when the builder publishes no readiness check (every kind
	// but AWS), so the UI can distinguish "clear" from "not applicable".
	Checked  bool     `json:"checked"`
	Warnings []string `json:"warnings"`
}

// handleGetBuildReadiness computes this build's demand from its own content
// revision and runs it past the builder's quota check. Gated at levelBuild, the
// same level that creates and deploys builds.
func (s *Server) handleGetBuildReadiness(w http.ResponseWriter, r *http.Request) {
	build, repo, err := s.buildAndOwningRepository(r)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	if _, err := s.requireLevel(r.Context(), r, repo, levelBuild); err != nil {
		writeAuthError(w, err)
		return
	}
	if !build.ConfiguredBuildID.Valid {
		writeError(w, http.StatusBadRequest, errors.New("this build has no configured build behind it -- no builder to check readiness against"))
		return
	}
	cb, err := s.Queries.GetConfiguredBuild(r.Context(), build.ConfiguredBuildID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	row, err := s.Queries.GetBuilderConfigByName(r.Context(), cb.BuilderConfigName)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	bldr, err := builderconfig.Resolve(s.Pool, row)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	checker, ok := bldr.(builder.ReadinessChecker)
	if !ok {
		// Warnings is [], never nil: it serializes to a JSON array, not null, so
		// the UI can read .length without guarding every path.
		writeJSON(w, http.StatusOK, readinessResponse{Checked: false, Warnings: []string{}}) // this builder has no quota wall to warn about
		return
	}

	content, err := s.loadBuildContent(r, build)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	env := findEnvironment(content, build.EnvironmentName)
	if env == nil {
		writeError(w, http.StatusBadRequest, errEnvironmentNotFound)
		return
	}
	verdict, err := checker.CheckReadiness(r.Context(), computeDemand(content, *env))
	if err != nil {
		writeError(w, http.StatusBadGateway, err)
		return
	}
	// Never emit null for warnings: a checker that found nothing returns a nil
	// slice, which would JSON-marshal to null and break the UI's .length read
	// (observed on a planned AWS build: "u.data.warnings is null").
	warnings := verdict.Warnings
	if warnings == nil {
		warnings = []string{}
	}
	writeJSON(w, http.StatusOK, readinessResponse{Checked: true, Warnings: warnings})
}

func findEnvironment(c *loader.Content, name string) *loader.Environment {
	for i := range c.Environments {
		if c.Environments[i].Name == name {
			return &c.Environments[i]
		}
	}
	return nil
}

// computeDemand estimates what one environment needs from its builder across all
// teams, from the resolved content. It's an upper-bound estimate for an advisory
// check, counting one instance per (network, object, copy):
//   - Teams          -> one isolated network (a VPC on AWS) per team
//   - PublicHosts    -> hosts with a public: block, each needing an external
//     address (an EIP on AWS), summed across teams
//   - InstanceCounts -> size name -> host-instance count across all teams
//
// Containers are left out: on AWS they're Fargate tasks against a separate quota,
// not EC2 instances, so counting them toward VPC/EIP/vCPU demand would mislead.
// Pure, so it's unit-testable without any builder.
func computeDemand(c *loader.Content, env loader.Environment) builder.Demand {
	hostByName := make(map[string]loader.Host, len(c.Hosts))
	for _, h := range c.Hosts {
		hostByName[h.Name] = h
	}
	d := builder.Demand{Teams: env.Teams, InstanceCounts: map[string]int{}}
	teams := env.Teams
	if teams < 1 {
		teams = 1 // an unspecified team count still deploys one team's worth
	}
	perTeamPublic := 0
	for _, objs := range env.Networks {
		for name, copies := range objs {
			h, isHost := hostByName[name]
			if !isHost {
				continue // a container (or a stale reference) -- not an EC2 instance
			}
			n := len(copies)
			if n == 0 {
				n = 1 // placed with no explicit copies == one instance
			}
			d.InstanceCounts[h.Size] += n * teams
			if h.Public != nil {
				perTeamPublic += n
			}
		}
	}
	d.PublicHosts = perTeamPublic * teams
	return d
}
