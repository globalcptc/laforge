// Mirrors of the real Go JSON shapes (internal/db/models.go and the
// handler-specific response types in internal/api). Kept as a plain
// hand-written file rather than codegen -- the backend has no OpenAPI/
// protobuf schema yet (the plan's long-term answer is Connect-RPC, not
// built yet), so this is the
// one place drift between Go and TS could happen; kept deliberately
// small and close to the wire shape to make that easy to spot.

// PublicConfig mirrors internal/api/config.go's own publicConfig --
// unauthenticated, safe to fetch before there's a session, since it's
// only ever used to build "Install on GitHub" links.
export interface PublicConfig {
  github_app_slug: string
}

export interface Repository {
  id: string
  github_owner: string
  github_repo: string
  created_at: string
}

export interface UnapprovedInstalledRepository {
  id: string
  installation_id: string
  github_owner: string
  github_repo: string
  github_repo_id: number
  created_at: string
  account_login: string
  account_type: string
}

// InstalledRepo/Installation mirror internal/api/installations.go's own
// installedRepoView/installationView -- every installation this instance
// knows about, each with every repo it covers and that repo's real
// approved/pending state (GET /installations), not just the ones still
// pending.
export interface InstalledRepo {
  github_owner: string
  github_repo: string
  github_repo_id: number
  approved: boolean
  repository_id?: string // LaForge's id, once approved
}

export interface Installation {
  id: string
  installation_id: number
  account_login: string
  account_type: string // "Organization" | "User"
  suspended: boolean
  created_at: string
  updated_at: string
  repos: InstalledRepo[]
}

export interface ConfiguredBuild {
  id: string
  repository_id: string
  branch: string
  environment_path: string
  builder_config_name: string
  auto_deploy_enabled: boolean
  competition_started: boolean
  current_content_revision_id: string | null
  created_at: string
}

export interface Build {
  id: string
  configured_build_id: string | null
  content_revision_id: string
  environment_name: string
  status: 'planned' | 'deploying' | 'building' | 'finished' | 'failed' | 'tearing_down' | 'torn_down' | 'purged'
  created_at: string
  // Set when the orchestrator's last reconcile of this build failed for a
  // reason that isn't a content error -- most often a build/builder
  // incompatibility (the chosen builder has no image for an os the environment
  // uses). Null/absent when the last reconcile succeeded.
  reconcile_error?: string | null
  // Commit facts + auto/manual marker are populated by the builds listing
  // (GET /repos/{id}/builds); other endpoints returning a bare build omit them.
  commit_sha?: string
  commit_message?: string | null
  committed_at?: string | null
  auto_built?: boolean
}

export interface Team {
  id: string
  build_id: string
  team_number: number
  access_state: 'open' | 'closed'
  access_override_until: string | null
  // A manual override the schedule reconciler honors: '' = follow the
  // environment's access: windows; 'open'/'closed' = hold that state until
  // access_override_until (then the schedule resumes).
  access_override_state: '' | 'open' | 'closed'
}

export type ObjectKind = 'network' | 'host' | 'container'
export type ObjectStatus =
  | 'pending'
  | 'deploying'
  | 'running'
  | 'building'
  | 'finished'
  | 'deploy_failed'
  | 'build_failed'
  | 'invalid'
  | 'destroying'
  | 'destroyed'
export type AgentStatus = 'booting' | 'healthy' | 'late' | 'missing'

export interface AgentHealth {
  last_heartbeat_at?: string
  agent_status: AgentStatus
}

/** Live power state at the hoster (builder truth), independent of the agent. '' = not yet polled. */
export type PowerState = '' | 'running' | 'stopped' | 'other' | 'missing'

export interface DeployedObject {
  id: string
  team_id: string
  kind: ObjectKind
  object_name: string
  as_name: string | null
  network_name: string | null
  fingerprint: string
  status: ObjectStatus
  external_ref: string | null
  last_error: string | null
  updated_at: string
  power_state?: PowerState
  power_state_checked_at?: string | null
  agent?: AgentHealth
  // Dependencies this object's steps are waiting on (its `depends_on` that
  // haven't finished yet). Set while its steps are materialized but blocked;
  // null/empty once released. Drives the "Dependency Blocked" state.
  blocked_on?: string[] | null
}

export interface TeamSummary extends Team {
  objects: DeployedObject[]
}

export interface AccessWindow {
  open: string
  close: string
}

export interface BuildDetail extends Build {
  teams: TeamSummary[]
  access?: AccessWindow[]
  // Commit + builder for the build header (resolved best-effort server-side).
  builder_config_name?: string
  builder_kind?: string
  // Branch the configured build tracks; empty for an ad-hoc build.
  branch?: string
}

export interface LFEvent {
  id: string
  build_id: string
  task_id: string | null
  deployed_object_id?: string | null
  kind: string
  message: string
  payload: unknown
  created_at: string
  /** Resolved host this event is about (as-name or content name); empty for build-level events. */
  host?: string
  team_number?: number | null
}

export interface Task {
  id: string
  build_id: string
  deployed_object_id: string | null
  kind: string
  payload: unknown
  status: 'pending' | 'leased' | 'done' | 'failed'
  attempts: number
  lease_owner: string | null
  lease_expires_at: string | null
  last_error: string | null
  created_at: string
  updated_at: string
}

export interface AgentTask {
  id: string
  deployed_object_id: string
  step_index: number
  command: string
  payload: unknown
  status: 'pending' | 'leased' | 'done' | 'failed'
  lease_expires_at: string | null
  attempts: number
  output: string | null
  last_error: string | null
  created_at: string
  updated_at: string
}

export interface Me {
  github_login: string
  avatar_url?: string
  is_instance_admin: boolean
  /** IANA timezone the UI renders timestamps in; empty means follow the browser. */
  timezone: string
}

export type AccessLevel = 'read' | 'build' | 'manage' | 'admin'

export interface AdHocTarget {
  ids?: string[]
  team?: number
  kind?: 'host' | 'container'
  search?: string
  network?: string
  // Operate by tag: an object matches only if it carries every one of these
  // tags (empty value = "has this key"). Unlike `ids`, a tag target is
  // re-resolved at fire time, so a scheduled task by tag picks up hosts added
  // after it was scheduled.
  tags?: Record<string, string>
}

export interface AdHocResult {
  matched: number
  objects?: DeployedObject[]
  created?: AgentTask[]
}

export interface FindingInstance {
  team_number?: number // absent for an environment-level finding
  object_kind: 'environment' | 'network' | 'host' | 'container'
  object_name: string
  as_name?: string
  source: string // "direct" or "script:<name>"
  severity: number
  difficulty: number
  description: string
}

// Build -> Topology (GET /builds/{id}/topology): the deployed copies grouped
// into a per-team tree of networks and their members, enriched with content
// facts (CIDR, OS/image, size, dependencies).
export interface TopologyMember {
  object_id: string
  kind: 'host' | 'container'
  object_name: string
  as_name?: string
  status: ObjectStatus
  power_state?: PowerState
  os?: string
  image?: string
  size?: string
  depends_on?: string[]
  tags?: Record<string, string>
}
export interface TopologyNetwork {
  object_id?: string
  name: string
  cidr?: string
  status?: ObjectStatus
  members: TopologyMember[]
}
export interface TopologyTeam {
  team_number: number
  networks: TopologyNetwork[]
}
export interface TopologyResponse {
  teams: TopologyTeam[]
}

// Build -> Artifacts (GET /builds/{id}/artifacts): storage by class with sizes.
export interface ArtifactClass {
  class: string
  count: number
  bytes: number
  purgeable?: boolean
  on_demand?: boolean
  description: string
}
export interface ArtifactsResponse {
  classes: ArtifactClass[]
  total_bytes: number
}

export interface Person {
  id: string
  people_source_name: string
  username: string
  attributes: Record<string, string>
}

export type TeamHealth = 'completed' | 'failed_infra' | 'failed_steps' | 'failed_checkin' | 'in_progress'

export interface TeamStateCounts {
  team_number: number
  by_status: Record<string, number>
  by_health: Record<TeamHealth, number>
}

export interface FailureGroup {
  message: string
  count: number
  objects: string[]
}

export interface ActivityBucket {
  at: string
  active: number
}

export interface DashboardData {
  provisioning_by_status: Record<string, number>
  by_team: TeamStateCounts[]
  failures_by_cause: FailureGroup[]
  agent_activity: ActivityBucket[]
  resource_usage: ResourceBucket[]
}

// Build-wide average of each host metric over one time bucket (across every
// heartbeat in it). cpu/mem/disk are percentages (0-100); net_rx/net_tx are
// bytes/sec (download / upload).
export interface ResourceBucket {
  at: string
  cpu: number
  mem: number
  disk: number
  net_rx: number
  net_tx: number
}

export interface UpcomingChange {
  team: number
  kind: ObjectKind
  object_name: string
  as_name?: string
  network_name?: string
  change: 'new' | 'changed' | 'removed'
}

export interface UpcomingChanges {
  pending: boolean
  changes: UpcomingChange[]
}

export interface StepValidator {
  kind: string
  passed: boolean
  message?: string
}

export interface StepStatus {
  step_index: number
  command: string
  status: 'pending' | 'leased' | 'done' | 'failed'
  attempts: number
  last_error?: string
  output?: string
  // Which authored step this expanded command belongs to, so the sub-commands
  // of one step (write_file + execute + validate) group under one heading.
  group_index: number
  group_label?: string
  validators: StepValidator[]
}

// Builder-specific placement facts (label/value, varies by builder) plus the
// environment root password, for finding/logging into a host by hand.
export interface ObjectInfra {
  placement: { label: string; value: string }[]
  password?: string
}

export interface RenderedStep {
  index: number
  action: string
  script_name?: string
  rendered?: string
  raw: Record<string, unknown>
}

// ScheduledTask mirrors db.ScheduledTask (internal/db/scheduled_task.sql.go)
// -- a content-authored `schedule:` entry or a UI-created ad-hoc one,
// both dispatched by internal/orchestrator.DispatchDueScheduledTasks
// through the same real CreateAgentTask path immediate ad-hoc dispatch
// uses. The when_expr grammar is internal/schedule.
export interface PowerResult {
  object_id: string
  name: string
  error?: string
}

export interface PowerActionResponse {
  results: PowerResult[]
  failed: number
}

// One object a rebuild will tear down and recreate -- the target host plus,
// when cascading, everything in its team that depends on it.
export interface RebuildAffected {
  id: string
  object_name: string
  as_name: string
  kind: string
  team_number: number
}

export interface RebuildResponse {
  affected: RebuildAffected[]
  count: number
  dry_run: boolean
}

// Agent trust-anchor expiry (GET /cert-status): the mTLS CA that signs every
// per-host agent cert, and the gateway's server cert. No key material, dates only.
export interface CertInfo {
  configured: boolean
  subject?: string
  not_after?: string
  days_remaining: number
  expiring_soon: boolean
  expired: boolean
  error?: string
}

export interface CertStatus {
  warn_threshold_days: number
  ca: CertInfo
  server: CertInfo
}

// One realized external endpoint (GET /builds/{id}/external-access): the public
// address:port to connect to for a host's `public:` port.
export interface ExternalAccessEntry {
  deployed_object_id: string
  team_number: number
  object_name: string
  as_name: string | null
  kind: string
  protocol: string
  internal_port: string
  external_port: string
  public_address: string
}

// ObjectConfig (GET /builds/{id}/objects/{objectId}/config): one host/container's
// full authored config from the build's content revision -- what the host info
// panel shows beyond runtime state.
export interface ConfigFinding {
  severity: number
  difficulty: number
  description: string
}

export interface ObjectConfig {
  kind: string
  name: string
  os?: string
  image?: string
  size?: string
  disk?: number
  command?: string[]
  tcp_ports?: string[]
  udp_ports?: string[]
  env?: Record<string, string>
  vars?: Record<string, string>
  tags?: Record<string, string>
  depends_on?: string[]
  findings?: ConfigFinding[]
}

export interface SchedulePreview {
  valid: boolean
  error?: string
  fires_once: boolean
  anchor?: string
  occurrences: string[]
}

export interface ScheduledTask {
  id: string
  build_id: string
  source: 'content' | 'adhoc'
  deployed_object_id: string | null
  schedule_index: number | null
  target: AdHocTarget | null
  when_expr: string
  command: string
  payload: unknown
  anchor: string
  fires_once: boolean
  next_fire_at: string | null
  status: 'pending' | 'fired' | 'canceled'
  created_by: string | null
  created_at: string
  updated_at: string
}

export interface AgentHeartbeat {
  id: string
  deployed_object_id: string
  cert_fingerprint: string
  remote_addr: string | null
  next_poll_ms: number | null
  created_at: string
  // Basic host metrics sampled by the agent on this check-in (null from an
  // older agent or a metric it couldn't read). cpu/mem/disk are percentages;
  // net_rx/tx are bytes per second since the previous heartbeat.
  cpu_pct: number | null
  mem_pct: number | null
  disk_pct: number | null
  net_rx_bps: number | null
  net_tx_bps: number | null
}

// IncusImageRef/IncusSizeSpec mirror internal/builder/incus.ImageRef/
// SizeSpec's own JSON shape exactly (their json tags, added specifically
// so a builder_config row's incus_images/incus_sizes JSONB round-trips
// with no separate wire-format struct needed on the Go side either).
export interface IncusImageRef {
  fingerprint?: string
  alias: string
  server?: string
  protocol?: string
  vm: boolean
}

export interface IncusSizeSpec {
  cpu: string
  memory: string
  // Cloud instance type / flavor (AWS EC2 type, OpenStack flavor) for the
  // non-Incus builders; ignored by Incus, which uses cpu/memory.
  type?: string
}

// IncusHostConfig mirrors internal/builder/incus.HostConfig's own JSON
// shape exactly -- one pool member, for kind="incus"'s own incus_hosts
// array (migration 00014). kind="microcloud" doesn't use this at all; it
// uses the single incus_api_url/... fields below directly, since a real
// MicroCloud cluster has exactly one endpoint no matter how many members
// it has (any one answers for the whole cluster).
export interface IncusHostConfig {
  credential_id?: string
  api_url: string
  client_cert_path: string
  client_key_path: string
  server_cert_pem: string
  ovn_uplink_network?: string
  storage_pool?: string
  operation_timeout_seconds?: number
}

// BuilderConfig mirrors internal/db.BuilderConfig -- the
// "what about the MicroCloud builder" gap: the real, named registry
// configured_build.builder_config_name looks up at deploy time
// (internal/runner.Runner.ResolveBuilderFromDB). Split (2026-09-25) into
// two genuinely separate real infrastructure shapes: "microcloud" is one
// real cluster (Ceph + OVN + LXD, the single incus_* fields below), and
// "incus" is a POOL of independent, non-clustered hosts (incus_hosts,
// one entry per host, team-assigned round-robin -- see
// internal/builder/incuspool's own doc comment).
export interface InstanceAdmin {
  github_login: string
  /** Who added them; null when seeded from LAFORGE_ADMIN_LOGINS on first startup. */
  added_by: string | null
  created_at: string
  /** Only known once they have signed in. */
  avatar_url: string | null
}

export interface RegistryCredential {
  id: string
  registry_host: string
  username: string
  created_at: string
}

export interface BuilderImageBuild {
  id: string
  builder_config_id: string
  kind: string
  status: 'pending' | 'running' | 'succeeded' | 'failed'
  image_fingerprint: string
  error: string
  created_at: string
  started_at: string | null
  finished_at: string | null
}

export interface ImageBuildLogResponse {
  id: string
  status: 'pending' | 'running' | 'succeeded' | 'failed'
  image_fingerprint: string
  error: string
  done: boolean
  lines: { seq: number; line: string }[]
}

export interface BuilderConfig {
  id: string
  name: string
  kind: 'fake' | 'incus' | 'microcloud' | 'aws' | 'openstack'
  container_base_server: string
  container_base_alias: string
  docker_base_fingerprint: string
  incus_api_url: string | null
  incus_client_cert_path: string | null
  incus_client_key_path: string | null
  incus_server_cert_pem: string | null
  incus_ovn_uplink_network: string | null
  incus_storage_pool: string | null
  /** MicroCloud only: the LXD project everything is created in; null/empty is `default`. */
  incus_project?: string | null
  incus_operation_timeout_seconds: number | null
  incus_images: Record<string, IncusImageRef>
  incus_sizes: Record<string, IncusSizeSpec>
  incus_hosts: IncusHostConfig[]
  incus_credential_id: string | null
  external_access_ip: string | null
  external_port_min: number | null
  external_port_max: number | null
  created_at: string
  updated_at: string
}

// BuilderConfigRequest is the create/update request body -- mirrors
// internal/api/builder_configs.go's own builderConfigRequest.
export interface BuilderConfigRequest {
  kind: 'fake' | 'incus' | 'microcloud' | 'aws' | 'openstack'
  incus_api_url?: string
  incus_client_cert_path?: string
  incus_client_key_path?: string
  incus_server_cert_pem?: string
  incus_ovn_uplink_network?: string
  incus_storage_pool?: string
  incus_project?: string
  incus_operation_timeout_seconds?: number
  incus_images?: Record<string, IncusImageRef>
  incus_sizes?: Record<string, IncusSizeSpec>
  incus_hosts?: IncusHostConfig[]
  incus_credential_id?: string
  external_access_ip?: string
  external_port_min?: number
  external_port_max?: number
}

// StoragePoolInfo/NetworkInfo/ImageInfo mirror internal/builder/incus's
// own discover.go types exactly -- real, live answers from the server
// itself (POST /builder-configs/probe/discover), not typed-in strings.
export interface StoragePoolInfo {
  name: string
  driver: string
  status: string
}

export interface NetworkInfo {
  name: string
  type: string // "physical" | "bridge" | "ovn" | ...
  managed: boolean
  status: string
}

export interface ImageInfo {
  fingerprint: string
  aliases: string[]
  architecture: string
  type: string // "container" | "virtual-machine"
  properties: {
    os: string
    release: string
    description: string
  }
}

export interface Discovery {
  storage_pools: StoragePoolInfo[]
  networks: NetworkInfo[]
  images: ImageInfo[]
  /** MicroCloud only: the projects the server lets LaForge see. */
  projects?: ProjectInfo[]
}

// ProjectInfo is one Incus/LXD project and which resources it keeps separate
// from `default` -- mirrors internal/builder.ProjectInfo.
export interface ProjectInfo {
  name: string
  description: string
  features_networks: boolean
  features_images: boolean
  features_profiles: boolean
  features_storage_volumes: boolean
  restricted: boolean
}

// BuilderConnection mirrors internal/api/builder_probe.go's connectionView:
// an enrolled Incus/MicroCloud server (never its key material) plus what it
// really has right now.
export interface BuilderCredentialSummary {
  id: string
  api_url: string
  server_name: string
  server_fingerprint: string
  created_at: string
}

export interface BuilderConnection {
  credential: BuilderCredentialSummary
  discovery: Discovery
}


// BranchInfo/EnvironmentFileInfo mirror internal/api/repo_browse.go --
// real dropdown sources for a configured build's branch and environment
// file, instead of a blind text field.
export interface BranchInfo {
  name: string
}

export interface ListBranchesResponse {
  default_branch: string
  branches: BranchInfo[]
}

export interface EnvironmentFileInfo {
  path: string
  name: string
}

export interface ListEnvironmentFilesResponse {
  files: EnvironmentFileInfo[]
  has_errors: boolean
  yaml_files: string[] // every .yaml/.yml on the branch, for the file picker
}

// BuilderSummary mirrors GET /builders: what anyone configuring a build
// needs to pick a builder.
export interface BuilderSummary {
  name: string
  kind: string
}

// DriftResource mirrors internal/builder.Resource -- one thing
// Builder.Inspect found already existing at the hoster.
export interface DriftResource {
  external_ref: string
  kind: string
}

// DriftReport mirrors internal/api/drift.go's own driftReportResponse --
// a leftover, finally wired to a real
// caller: Builder.Inspect's live answer, diffed against this build's own
// deployed_object rows.
export interface DriftReport {
  orphaned: DriftResource[]
  missing: DeployedObject[]
  tracked: number
}

// RepoAccess mirrors internal/api/builds_ui.go's repoAccessView: one
// repository's people -- GitHub's collaborators on it plus anyone an admin
// has set a level for. A person's level starts from their GitHub role; an
// admin's setting (level) replaces it, and 'none' takes access away.
export type RepoLevel = AccessLevel | 'none'

export interface RepoAccessPerson {
  login: string
  avatar_url: string
  github_role: string // '' when not a collaborator on GitHub
  github_level: RepoLevel
  level: RepoLevel | '' // '' = follows GitHub
  effective: RepoLevel
}

export interface RepoAccess {
  repository: Repository
  people: RepoAccessPerson[]
  github_error?: string
}

export interface MyRepoAccess {
  level: RepoLevel
  can_manage_access: boolean
}

// HomeData mirrors internal/api/home.go's homeView: what is live right now
// across every repository this person can access.
export interface HomeCounts {
  hosts: number
  containers: number
  networks: number
  objects_failed: number
  agents_healthy: number
  agents_late: number
  agents_missing: number
  agents_booting: number
  tasks_outstanding: number
  tasks_failed: number
}

export interface HomeBuild {
  id: string
  environment_name: string
  status: string
  created_at: string
  commit_sha: string
  ref: string
  builder_config_name: string
  teams: number
  teams_open: number
  counts: HomeCounts
  access?: AccessWindow[]
}

export interface HomeData {
  active_builds: number
  totals: HomeCounts
  repositories: { id: string; github_owner: string; github_repo: string; builds: HomeBuild[] }[]
  builders: { name: string; kind: string; active_builds: number; counts: HomeCounts }[]
  attention: AttentionItem[]
}

export interface AttentionItem {
  repository_id: string
  repository: string
  build_id: string
  environment_name: string
  category: string
  reason: string
}
