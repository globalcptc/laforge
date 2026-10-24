# Builder monitoring & stats — design + build plan

Status: in progress. Builder pages today are just the wizard with clickable tabs;
this replaces that with a real info page per builder (health charts + status +
public endpoints + edit actions) and a per-builder quota page.

## Goal

Watch the health of the builder **stacks themselves** (the Incus servers, the
MicroCloud cluster, the AWS account) — host/hoster-level, NOT the deployed
guests' agent metrics. The data comes from each hoster's own API.

## Self-describing stats contract (builders publish, UI renders dynamically)

A builder optionally implements a `MetricProvider` (probed like the other
optional builder interfaces, `b.(interface{ Stats(...) })`):

```go
type Metric struct {
    Key   string   // stable id: "cpu", "mem", "disk_ops", "net", "api_latency", "quota:vpcs"
    Label string   // display: "CPU", "Memory", "Disk R/W ops", "API response time", "VPCs"
    Node  string   // "" = builder-wide; a server/member name for per-node (Incus)
    Kind  string   // "timeseries" | "gauge"
    Unit  string   // "%", "ops/s", "ms", "count", "bytes/s", ...
    Value float64
    Max   *float64 // set only for gauges (the quota limit); nil = unbounded series
}
type Stats struct {
    Metrics  []Metric
    Warnings []string
}
type MetricProvider interface { Stats(ctx context.Context) (Stats, error) }
```

The UI renders whatever comes back: `timeseries` → a line chart (from stored
history), `gauge` → a usage bar (`Value/Max`, warning band near the limit). Add a
metric in a builder and it shows up automatically — no hardcoded panels. A
builder that implements nothing (OpenStack, or an un-upgraded builder) simply has
no stats UI.

## Collection pipeline

- An orchestrator loop polls each **distinct** builder (not per build) every
  **120s** (configurable flag; chosen over 60s to halve cluster load).
- One gather per builder per cycle, **batched and shared**: the single `Inspect`
  pass already lists all managed resources, so it feeds (a) power-state
  reconciliation (today `PollInstanceStates` calls `Inspect` once *per build* —
  consolidate to once per builder), (b) quota usage counts, and (c) `Stats()`.
- Samples stored append-only in a new `builder_metric_sample` table
  (builder_config_id, key, node, ts, value, unit, kind, max), bounded retention
  reusing the `agent_heartbeat` cleanup pattern (one config knob).
- Gauges render latest `Value/Max`; time-series charts query the history window.

### Per-builder internal batching
- **AWS** → one `GetMetricData` carrying all metric queries; quota **limits
  cached hourly** (Service Quotas), usage from the shared inventory. ~1 CloudWatch
  call/cycle regardless of instance count.
- **Incus / MicroCloud** → one `GET /1.0/cluster/members?recursion=2` (all members
  + state in one call); standalone (non-clustered) host falls back to
  `/1.0/resources` + the single implicit member.
- Net: ~1–2 hoster calls per builder per 120s, flat.

## Metrics per builder (availability is uneven — by design)

| | CPU | Memory | Disk | Network | API latency | Quotas |
|---|---|---|---|---|---|---|
| **AWS** (CloudWatch + Service Quotas) | CPUUtilization % | — (needs CW agent; omit) | **Read/Write ops** (EBS Volume R/W ops) | NetworkIn/Out | timed Describe call | **real**: VPCs/region, CIDR/VPC, EIPs, vCPUs, SGs/VPC, ECS tasks |
| **Incus / MicroCloud** (LXD API) | load avg (per member) | total/free → % | pool used/total → % | omit (weak) | timed `GET /1.0` | N/A (no quotas) |
| **OpenStack** | (not implemented — opt out) | | | | | |

Semantics differ (load avg vs % vs ops) — each metric carries its own `Unit`, and
the page labels its source ("LXD" / "CloudWatch") so it's never misread.

## Quota page — per builder (they are vastly different)

- **AWS**: prominent — usage-vs-limit gauges with a warning band for the limits
  that actually break a mid-competition deploy (one VPC per team vs 5/region,
  5 CIDR/VPC, one EIP per public host vs 5, running vCPUs, SGs/VPC, ECS tasks).
- **OpenStack**: minor/informational (project quota-set cores/ram/instances/
  floating-IPs); we're admins, so framed as "can be raised", not an alarm.
- **Incus / MicroCloud**: no quota section (no quota concept).

Falls out of the contract for free: quota gauges are just that builder's
`Kind:gauge` metrics, on its own page.

## Pre-deploy readiness check (AWS)

Reuse the quota usage/limit data to compute an environment's demand up front
(teams × networks → VPCs/CIDRs, `public:` hosts → EIPs, hosts×size → vCPUs) and
warn before a build that will exceed a quota — the sibling of the IAM pre-flight
check. AWS-only, since it's the only hoster where a quota wall silently breaks a
deploy.

## UI

- **Builder info page** (replaces the tabbed-wizard view): header (name, kind,
  region/account or cluster summary, + **Edit config / Images / Sizes** buttons
  that deep-link into the existing wizard edit flow); dynamic time-series charts
  (incl. API latency); status panel (active builds, resource counts per kind);
  public-accessible endpoints list (`deployed_object.public_address`).
- **Edit buttons** also appear as row actions in the builder list.
- **Quota page** per builder (gauges), present only where the builder publishes
  gauges.

## Build order (increments)

1. **Contract + pipeline spine** — `builder.Metric/Stats/MetricProvider`;
   `builder_metric_sample` table (migration + sqlc: insert, list-by-builder-window,
   delete-old); orchestrator 120s poll loop over distinct builders storing samples;
   retention. Testable against a disposable DB with the fake builder.
2. **Consolidate `PollInstanceStates`** to one `Inspect` per builder, shared with
   the stats gather. ✅ DONE — `PollInstanceStatesForBuilds` caches Inspect by
   builder_config name (dedup across builds sharing a builder); the single-build
   `PollInstanceStates` now delegates to the extracted `inspectStateMap` +
   `applyInstanceStates`. Orchestrator poll loop batches all builds in one pass.
3. **Incus / MicroCloud `Stats()`** (cluster members recursion=2 → cpu/mem/disk +
   api_latency).
4. **AWS `Stats()`** (CloudWatch batched + Service Quotas cached + inventory;
   disk = R/W ops; quota gauges). New IAM: `cloudwatch:GetMetricData`,
   `servicequotas:GetServiceQuota` (+ ListServiceQuotas) — add to policy/preflight/docs.
5. **UI** — builder info page (status + public list + edit buttons first, then the
   dynamic charts), per-builder quota page.
6. **Pre-deploy readiness check** (AWS quota demand vs limits). ✅ DONE —
   optional `builder.ReadinessChecker` contract (`Demand`/`Readiness`); AWS
   `CheckReadiness` compares teams→VPCs, public hosts→EIPs, sizes→instance
   vCPUs against current usage + Service Quotas (VPC L-F678F1CE, EIP L-0263D0A3,
   vCPU L-1216C47A), advisory warnings only. `GET /builds/{id}/readiness` computes
   demand from the build's content (`computeDemand`, pure) and runs the check;
   non-AWS builders return `checked:false`. Pure `evalReadiness`/`computeDemand`
   unit-tested. No new IAM (reuses Describe* + servicequotas already in preflight).

## Decided
- Host/hoster source, not agents.
- Time-series for health metrics; gauges for quotas.
- 120s cadence (configurable); retention reuses the heartbeat window.
- Quota page per-builder. OpenStack opts out of all of this.
- Builders self-describe; UI is generic.
