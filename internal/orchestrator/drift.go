// Drift detection: the last leftover gap.
// `Builder.Inspect` ("list what exists, for adoption and drift") has
// been real and correct on both real builders since it was written, but
// nothing outside its own tests ever called it -- unlike the other four
// gaps that same audit found and fixed, this one had no obvious, narrow
// caller waiting, so it was deliberately left as "worth a real design
// decision before wiring either in." The design, made here: an
// admin-triggered "detect drift for this build" action, not a
// background sweep -- Inspect hits the real hoster API for every
// resource it has, which isn't something to run automatically on a
// timer for every build without an operator asking for it. Wired to a
// real API endpoint (internal/api/drift.go) and a real UI action, not
// just this package.
package orchestrator

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/globalcptc/laforge/internal/builder"
	"github.com/globalcptc/laforge/internal/builderconfig"
	"github.com/globalcptc/laforge/internal/db"
)

// DriftReport is Inspect's real, live answer compared against this
// build's own deployed_object rows, keyed the same way a Deploy*/Destroy*
// call already does (ExternalRef) -- exactly what Resource's own doc
// comment says Inspect exists for.
type DriftReport struct {
	// Orphaned is every resource the hoster actually has that this build
	// has no record of at all (never deployed by LaForge, or deployed and
	// then somehow lost from deployed_object) -- a real "adopt or clean
	// this up" candidate for an operator.
	Orphaned []builder.Resource
	// Missing is every deployed_object this build believes is live
	// (status "deployed") whose own ExternalRef the hoster no longer has
	// -- destroyed or lost outside LaForge's own control.
	Missing []db.DeployedObject
	// Tracked is every deployed_object whose ExternalRef the hoster
	// confirms it still has -- the "no drift" case, included so a report
	// with zero Orphaned/Missing is distinguishable from one that
	// silently inspected nothing.
	Tracked int
}

// DetectDrift resolves buildID's real builder through the same
// build -> configured_build -> builder_config chain
// runner.ResolveBuilderFromDB already uses for real deploys, calls its
// real Inspect, and diffs the result against this build's own
// deployed_object rows. Returns a clear error (not a fallback to some
// other builder) for a build with no configured_build at all -- there's
// no builder_config to resolve against, and guessing one would make a
// drift report meaningless.
func DetectDrift(ctx context.Context, pool *pgxpool.Pool, buildID pgtype.UUID) (DriftReport, error) {
	q := db.New(pool)
	build, err := q.GetBuild(ctx, buildID)
	if err != nil {
		return DriftReport{}, fmt.Errorf("loading build: %w", err)
	}
	if !build.ConfiguredBuildID.Valid {
		return DriftReport{}, errors.New("this build has no configured_build, so there's no builder_config to detect drift against")
	}
	cb, err := q.GetConfiguredBuild(ctx, build.ConfiguredBuildID)
	if err != nil {
		return DriftReport{}, fmt.Errorf("loading configured build: %w", err)
	}
	row, err := q.GetBuilderConfigByName(ctx, cb.BuilderConfigName)
	if err != nil {
		return DriftReport{}, fmt.Errorf("loading builder config %q: %w", cb.BuilderConfigName, err)
	}
	bldr, err := builderconfig.Resolve(pool, row)
	if err != nil {
		return DriftReport{}, fmt.Errorf("resolving builder %q: %w", cb.BuilderConfigName, err)
	}

	allResources, err := bldr.Inspect(ctx)
	if err != nil {
		return DriftReport{}, fmt.Errorf("inspecting %q: %w", cb.BuilderConfigName, err)
	}
	objects, err := q.ListDeployedObjectsByBuild(ctx, buildID)
	if err != nil {
		return DriftReport{}, fmt.Errorf("loading deployed objects: %w", err)
	}

	// A real builder's Inspect lists EVERY resource it has, across every
	// build that shares it (a real cluster, or even this package's own
	// shared fake_hoster_resource table, is never scoped to one build) --
	// found live by this package's own first drift test run: without
	// this filter, every other build that ever used the same builder
	// showed up as "orphaned" in a report about a completely different
	// build. runner.deterministicExternalName always prefixes a
	// resource's own name with "build-<this build's id>-", the one real
	// signal for "this resource is even in scope for this build's own
	// report" -- a resource with a different prefix belongs to some
	// other build's own drift report, not this one; it's not evaluated
	// as orphaned OR tracked here at all.
	prefix := "build-" + buildID.String() + "-"
	resources := make([]builder.Resource, 0, len(allResources))
	for _, res := range allResources {
		if strings.HasPrefix(res.ExternalRef, prefix) {
			resources = append(resources, res)
		}
	}

	// Inspect's own contract (Resource.Kind's doc comment) only ever
	// covers "network | host | container" -- "dns" is a real
	// deployed_object kind (item 27) with its own status/external_ref,
	// but no builder's Inspect implementation was ever going to list it
	// (the fake builder's own DNS records live in a separate table
	// entirely, fake_dns_record, not fake_hoster_resource). Found live
	// by this package's own first drift test run: every dns object came
	// back as spuriously "missing" until objects outside Inspect's real
	// scope were excluded here -- the same class of bug DiffUpcoming's
	// own dns fix (item 27) already hit once.
	inspectable := func(kind string) bool { return kind == "network" || kind == "host" || kind == "container" }

	tracked := make(map[string]bool, len(objects))
	for _, o := range objects {
		if inspectable(o.Kind) && o.ExternalRef != nil && *o.ExternalRef != "" {
			tracked[*o.ExternalRef] = true
		}
	}

	live := make(map[string]bool, len(resources))
	var report DriftReport
	for _, res := range resources {
		live[res.ExternalRef] = true
		if !tracked[res.ExternalRef] {
			report.Orphaned = append(report.Orphaned, res)
		}
	}
	for _, o := range objects {
		if !inspectable(o.Kind) || !infraUp(o.Status) || o.ExternalRef == nil || *o.ExternalRef == "" {
			continue
		}
		if live[*o.ExternalRef] {
			report.Tracked++
		} else {
			report.Missing = append(report.Missing, o)
		}
	}
	return report, nil
}
