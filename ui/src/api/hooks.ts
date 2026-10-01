import { useEffect, useRef, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { API_BASE, api } from './client'
import type {
  AdHocResult,
  CertStatus,
  PowerActionResponse,
  RebuildResponse,
  AdHocTarget,
  AgentHeartbeat,
  Build,
  BuildDetail,
  BuilderConfig,
  BuilderConfigRequest,
  BuilderImageBuild,
  ImageBuildLogResponse,
  RegistryCredential,
  BuilderConnection,
  BuilderSummary,
  ConfiguredBuild,
  DashboardData,
  DeployedObject,
  DriftReport,
  FindingInstance,
  TopologyResponse,
  ArtifactsResponse,
  HomeData,
  Installation,
  MyRepoAccess,
  RepoAccess,
  LFEvent,
  ListBranchesResponse,
  ListEnvironmentFilesResponse,
  Me,
  Person,
  PublicConfig,
  RenderedStep,
  StepStatus,
  ObjectConfig,
  ObjectInfra,
  Repository,
  ScheduledTask,
  SchedulePreview,
  Task,
  Team,
  UnapprovedInstalledRepository,
  UpcomingChanges,
} from './types'

export function useMe() {
  return useQuery<Me>({
    queryKey: ['me'],
    queryFn: () => api.get('/auth/me'),
    retry: false,
  })
}

// useSetTimezone persists the signed-in account's IANA timezone (server
// validates it), then refreshes `me` so every timestamp in the UI
// re-renders in the new zone immediately.
export function useSetTimezone() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (timezone: string) => api.post('/auth/me/timezone', { timezone }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['me'] }),
  })
}

// useConfig is this instance's own public config (GET /config,
// internal/api/config.go) -- unauthenticated, so it's safe to call
// before there's a session at all. github_app_slug is empty on a
// deployment with no GitHub App configured (Server.AppPrivateKey's own
// doc comment: "a deployment can run with no App configured at all"),
// which is exactly what the "Add Repository" flow needs to know before
// deciding whether to offer the GitHub install link or go straight to
// direct registration.
export function useConfig() {
  return useQuery<PublicConfig>({ queryKey: ['config'], queryFn: () => api.get('/config'), staleTime: Infinity })
}

// useHome is Home's server-side aggregate (internal/api/home.go), kept
// fresh while the page is open.
export function useHome() {
  return useQuery<HomeData>({ queryKey: ['home'], queryFn: () => api.get('/home'), refetchInterval: 15_000 })
}

// useCertStatus reports the agent CA / gateway cert expiry so the UI can warn
// before the trust anchor lapses. Certs expire slowly, so this only needs an
// hourly refresh to notice the <3-month crossing, not aggressive polling.
export function useCertStatus() {
  return useQuery<CertStatus>({ queryKey: ['cert-status'], queryFn: () => api.get('/cert-status'), refetchInterval: 3_600_000 })
}

export function useRepositories() {
  return useQuery<Repository[]>({ queryKey: ['repositories'], queryFn: () => api.get('/repos') })
}

export function useRepository(id: string | undefined) {
  return useQuery<Repository>({
    queryKey: ['repository', id],
    queryFn: () => api.get(`/repos/${id}`),
    enabled: !!id,
  })
}

export function useBuilds(repoId: string | undefined) {
  return useQuery<Build[]>({
    queryKey: ['builds', repoId],
    queryFn: () => api.get(`/repos/${repoId}/builds`),
    enabled: !!repoId,
    refetchInterval: 10_000,
  })
}

// useConfiguredBuilds is "Repository → configured builds" -- the
// {branch, environment file, builder config} triples a repo's builds get
// created from ("a build is always configured
// explicitly"), which had an API already but no UI surface
// until now.
export function useConfiguredBuilds(repoId: string | undefined) {
  return useQuery<ConfiguredBuild[]>({
    queryKey: ['configured-builds', repoId],
    queryFn: () => api.get(`/repos/${repoId}/configured-builds`),
    enabled: !!repoId,
  })
}

export function useCreateConfiguredBuild(repoId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (req: { branch: string; environment_path: string; builder_config_name: string }) =>
      api.post<ConfiguredBuild>(`/repos/${repoId}/configured-builds`, req),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['configured-builds', repoId] }),
  })
}

// useSetConfiguredBuildAutoDeploy toggles per-branch auto-deploy on a
// configured build (auto_deploy_enabled -- internal/api/builds.go's
// handleSetConfiguredBuildAutoDeploy). On, a new CI-passing commit builds
// AND deploys; off, it builds only (validated, free) and waits for a
// manual deploy. Auto-build itself is always on.
export function useSetConfiguredBuildAutoDeploy(repoId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ configuredBuildId, enabled }: { configuredBuildId: string; enabled: boolean }) =>
      api.post<ConfiguredBuild>(`/configured-builds/${configuredBuildId}/auto-deploy`, { enabled }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['configured-builds', repoId] }),
  })
}

// useSetLock toggles "marking the competition started" (competition_started
// -- internal/api/builds.go's handleSetLock), which locks automatic
// deploys while leaving manual deploys, single-host rebuilds, and ad-hoc
// tasks available ("marking the competition
// started locks automatic deploys").
export function useSetLock(repoId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ configuredBuildId, started }: { configuredBuildId: string; started: boolean }) =>
      api.post<ConfiguredBuild>(`/configured-builds/${configuredBuildId}/lock`, { started }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['configured-builds', repoId] }),
  })
}

// useTriggerBuild is the "Build" verb (Build and
// deploy are separate verbs): resolve the configured build's currently
// tracked, CI-passed commit into a real `planned` build. Touches no
// hoster -- internal/api/build_triggers.go's handleTriggerBuild.
export function useTriggerBuild(repoId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (configuredBuildId: string) => api.post<Build>(`/configured-builds/${configuredBuildId}/builds`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['builds', repoId] }),
  })
}

// useSyncConfiguredBuild is the "Sync latest commit" action: ingest the branch
// HEAD on demand so a configured build whose content is committed but not yet
// ingested (a new build, or a commit no push webhook has processed) gets a
// tracked revision and becomes buildable -- no throwaway push needed. Returns
// the updated configured build (with current_content_revision_id set when the
// commit is buildable). Invalidates the configured-builds list so the UI's
// "Build Now" enables immediately.
export function useSyncConfiguredBuild(repoId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (configuredBuildId: string) => api.post<ConfiguredBuild>(`/configured-builds/${configuredBuildId}/sync`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['configured-builds', repoId] }),
  })
}

// useDeleteConfiguredBuild removes a configured build. The server refuses (409)
// while it has live builds (deploying/building/finished/tearing down); existing
// builds survive as detached history.
export function useDeleteConfiguredBuild(repoId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (configuredBuildId: string) => api.delete<void>(`/configured-builds/${configuredBuildId}`),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['configured-builds', repoId] })
      qc.invalidateQueries({ queryKey: ['builds', repoId] })
    },
  })
}

// useDeployBuild is the "Deploy" verb: flip a `planned` build to
// `deploying`, which is what internal/orchestrator's poll loop watches
// for before it creates a single real task.
export function useDeployBuild(buildId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: () => api.post<Build>(`/builds/${buildId}/deploy`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['build', buildId] }),
  })
}

// useTeardownBuild destroys
// every real object this build has deployed (internal/orchestrator.Teardown,
// driven by cmd/laforge-orchestrator's own poll loop -- same async shape
// as Deploy above), landing on a real 'torn_down' status once every
// object genuinely is. Gated at levelManage on the backend, not
// levelBuild -- this acts on live infrastructure, not the build's
// existence.
export function useTeardownBuild(buildId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: () => api.post<Build>(`/builds/${buildId}/teardown`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['build', buildId] }),
  })
}

export function useBuild(buildId: string | undefined) {
  return useQuery<BuildDetail>({
    queryKey: ['build', buildId],
    queryFn: () => api.get(`/builds/${buildId}`),
    enabled: !!buildId,
    refetchInterval: 5_000,
  })
}

// useDashboard is the real server-side aggregation (internal/api/
// dashboard.go) behind the health band's charts -- "all fed by
// server-side aggregates rather than counted in the browser"
// not client-side counting over useBuild's payload.
export function useDashboard(buildId: string | undefined) {
  return useQuery<DashboardData>({
    queryKey: ['dashboard', buildId],
    queryFn: () => api.get(`/builds/${buildId}/dashboard`),
    enabled: !!buildId,
    refetchInterval: 5_000,
  })
}

// useUpcomingChanges is "if a newer commit has built, what deploying it
// would alter, at environment, team, network, and host level"
// -- real output of
// internal/orchestrator.DiffUpcoming (internal/api/upcoming.go), not a
// client-side guess. `pending: false` means there's nothing newer than
// what this build was created from to compare against.
export function useUpcomingChanges(buildId: string | undefined) {
  return useQuery<UpcomingChanges>({
    queryKey: ['upcoming', buildId],
    queryFn: () => api.get(`/builds/${buildId}/upcoming`),
    enabled: !!buildId,
    refetchInterval: 30_000,
  })
}

// useApplyUpcoming is the deliberate action useUpcomingChanges's preview
// exists to inform: "a new commit is applied to the existing build, not
// a teardown and rebuild from scratch" --
// internal/api/build_triggers.go's handleApplyUpcoming updates this
// build's own row in place (team/deployed_object rows never move), so
// the already-running orchestrator poll loop picks up the content change
// on its next pass and destroys/redeploys only what actually changed.
export function useApplyUpcoming(buildId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: () => api.post<Build>(`/builds/${buildId}/apply-upcoming`),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['upcoming', buildId] })
      qc.invalidateQueries({ queryKey: ['build', buildId] })
    },
  })
}

// useDetectDrift is a leftover,
// finally given a real caller: an admin-triggered action (GET
// /builds/{id}/drift, internal/orchestrator.DetectDrift), not a
// background sweep -- it makes a real, live call to the build's own
// builder, so it's a deliberate click, not something polled.
export function useDetectDrift(buildId: string) {
  return useMutation({
    mutationFn: () => api.get<DriftReport>(`/builds/${buildId}/drift`),
  })
}

export function useDeployedObjects(buildId: string | undefined) {
  return useQuery<DeployedObject[]>({
    queryKey: ['objects', buildId],
    queryFn: () => api.get(`/builds/${buildId}/objects`),
    enabled: !!buildId,
    refetchInterval: 5_000,
  })
}

export function useEvents(buildId: string | undefined) {
  return useQuery<LFEvent[]>({
    queryKey: ['events', buildId],
    queryFn: () => api.get(`/builds/${buildId}/events`),
    enabled: !!buildId,
  })
}

// useObjectEvents is "per-object logs" -- one
// host or container's own slice of the journal (internal/api's
// ListEventsByDeployedObject), scoped to real deploy/destroy/access
// lifecycle events -- there's an adjacent
// gap (agent step execution doesn't write events yet).
export function useObjectEvents(buildId: string | undefined, objectId: string | undefined) {
  return useQuery<LFEvent[]>({
    queryKey: ['object-events', buildId, objectId],
    queryFn: () => api.get(`/builds/${buildId}/objects/${objectId}/events`),
    enabled: !!buildId && !!objectId,
  })
}

// useObjectHeartbeats is the "live troubleshooting" / "rules checking"
// data (migrations/00007's real append-only agent_heartbeat log) for one
// specific object -- newest first, including the real remote address
// each check-in came from.
export function useObjectHeartbeats(buildId: string | undefined, objectId: string | undefined) {
  return useQuery<AgentHeartbeat[]>({
    queryKey: ['object-heartbeats', buildId, objectId],
    queryFn: () => api.get(`/builds/${buildId}/objects/${objectId}/heartbeats`),
    enabled: !!buildId && !!objectId,
  })
}

// useObjectRender is "the exact script delivered to a given host in a
// given team can be retrieved from the UI and matches what the agent
// ran" -- real output of internal/render.Resolve
// + RenderScript (internal/api/render.go), not a client-side guess.
// Disabled until the panel actually opens on the "render" tab, since
// unlike events/heartbeats this reads a live git checkout on the server,
// not just Postgres -- no reason to pay for it on every object row.
export function useObjectRender(buildId: string | undefined, objectId: string | undefined, enabled: boolean) {
  return useQuery<RenderedStep[]>({
    queryKey: ['object-render', buildId, objectId],
    queryFn: () => api.get(`/builds/${buildId}/objects/${objectId}/render`),
    enabled: !!buildId && !!objectId && enabled,
  })
}

// useObjectSteps is one host/container's materialized build steps and their
// live status (the real agent_task rows + validator results), so the detail
// panel can show where it is in its build. Polls while open so a running
// step's status updates without a manual refresh.
export function useObjectSteps(buildId: string | undefined, objectId: string | undefined, enabled: boolean) {
  return useQuery<StepStatus[]>({
    queryKey: ['object-steps', buildId, objectId],
    queryFn: () => api.get(`/builds/${buildId}/objects/${objectId}/steps`),
    enabled: !!buildId && !!objectId && enabled,
    refetchInterval: enabled ? 5_000 : false,
  })
}

// useObjectInfra is one object's builder placement (where to find it on the
// builder) plus the environment root password, for hand access. Reads the
// builder-config chain server-side (internal/api/object_infra.go), so like
// render it's disabled until the Info tab that shows it is actually open.
export function useObjectConfig(buildId: string | undefined, objectId: string | undefined, enabled: boolean) {
  return useQuery<ObjectConfig>({
    queryKey: ['object-config', buildId, objectId],
    queryFn: () => api.get(`/builds/${buildId}/objects/${objectId}/config`),
    enabled: !!buildId && !!objectId && enabled,
  })
}

export function useObjectInfra(buildId: string | undefined, objectId: string | undefined, enabled: boolean) {
  return useQuery<ObjectInfra>({
    queryKey: ['object-infra', buildId, objectId],
    queryFn: () => api.get(`/builds/${buildId}/objects/${objectId}/infra`),
    enabled: !!buildId && !!objectId && enabled,
  })
}

// useInstallations is "list everywhere we have the app installed that we
// know about" -- every real installation (GET /installations,
// internal/api/installations.go's handleListInstallations), each with
// every repo it covers and that repo's real approved/pending state, not
// just the ones still pending. 403s for anyone not in the server's
// LAFORGE_ADMIN_LOGINS -- same reasoning as useUnapprovedInstalledRepositories
// below, which this one is meant to fully replace on the Installations
// screen (kept, still used by Home.tsx's own pending-approval banner).
export function useInstallations() {
  return useQuery<Installation[]>({
    queryKey: ['installations'],
    queryFn: () => api.get('/installations'),
    refetchInterval: 15_000,
    retry: false,
  })
}

// useUnapprovedInstalledRepositories is "install on GitHub, approve in
// LaForge"'s own list:
// every repository some installation currently covers that LaForge
// isn't tracking yet. 403s for anyone not in the server's
// LAFORGE_ADMIN_LOGINS -- surfaced by the page itself, not hidden, same
// as every other access error in this app.
export function useUnapprovedInstalledRepositories() {
  return useQuery<UnapprovedInstalledRepository[]>({
    queryKey: ['installations', 'unapproved'],
    queryFn: () => api.get('/installations/repositories'),
    refetchInterval: 15_000,
    // A 403 here means "not an instance admin," not a transient
    // failure -- retrying won't fix it, it only delays the real message
    // by several seconds of default exponential backoff. Same reasoning
    // useMe() already uses.
    retry: false,
  })
}

export function useApproveInstalledRepository() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (req: { owner: string; repo: string; installation_id: string }) =>
      api.post<Repository>('/installations/repositories/approve', req),
    onSuccess: () => {
      // Broad key so BOTH the Installations page (['installations']) and the
      // pending list (['installations','unapproved']) refetch -- invalidating
      // the longer key alone never matches the shorter one, which is why the
      // Approve button needed a manual reload to reflect.
      qc.invalidateQueries({ queryKey: ['installations'] })
      qc.invalidateQueries({ queryKey: ['repositories'] })
    },
  })
}

// useBuilderConfigs is "what about the MicroCloud builder" made real at
// the UI layer: every registered builder_config row
// (internal/runner.Runner.ResolveBuilderFromDB is what actually resolves
// one of these by name at deploy time). 403s for anyone not an instance
// admin, same reasoning as useUnapprovedInstalledRepositories.
export function useBuilderConfigs() {
  return useQuery<BuilderConfig[]>({
    queryKey: ['builder-configs'],
    queryFn: () => api.get('/builder-configs'),
    retry: false,
  })
}

export function useCreateBuilderConfig() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ name, ...req }: { name: string } & BuilderConfigRequest) =>
      api.post<BuilderConfig>(`/builder-configs/${encodeURIComponent(name)}`, req),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['builder-configs'] }),
  })
}

export function useUpdateBuilderConfig() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ name, ...req }: { name: string } & BuilderConfigRequest) =>
      api.put<BuilderConfig>(`/builder-configs/${encodeURIComponent(name)}`, req),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['builder-configs'] }),
  })
}

export function useDeleteBuilderConfig() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (name: string) => api.delete(`/builder-configs/${encodeURIComponent(name)}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['builder-configs'] }),
  })
}

// Per-builder docker base image build: list recent builds, trigger a rebuild,
// and tail one build's live log (polling while it runs).
export function useBuilderImageBuilds(name: string | undefined) {
  return useQuery<BuilderImageBuild[]>({
    queryKey: ['builder-image-builds', name],
    queryFn: () => api.get(`/builder-configs/${encodeURIComponent(name!)}/image-builds`),
    enabled: !!name,
    refetchInterval: 5000, // catch the auto-build that starts on builder create
  })
}

export function useRebuildBuilderImage(name: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: () => api.post<BuilderImageBuild>(`/builder-configs/${encodeURIComponent(name)}/image-builds`, {}),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['builder-image-builds', name] })
      qc.invalidateQueries({ queryKey: ['builder-configs'] })
    },
  })
}

// Private Docker registry credentials (Admin → Infrastructure). The secret is
// write-only: sent on upsert, never returned.
export function useRegistryCredentials() {
  return useQuery<RegistryCredential[]>({
    queryKey: ['registry-credentials'],
    queryFn: () => api.get('/registry-credentials'),
  })
}

export function useUpsertRegistryCredential() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (body: { registry_host: string; username: string; secret: string }) =>
      api.put('/registry-credentials', body),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['registry-credentials'] }),
  })
}

export function useDeleteRegistryCredential() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (host: string) => api.delete(`/registry-credentials/${encodeURIComponent(host)}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['registry-credentials'] }),
  })
}

// Confirm a stored registry credential actually authenticates. A failed test
// is a normal result (ok:false with a reason), not a thrown error.
export function useTestRegistryCredential() {
  return useMutation<{ ok: boolean; message: string }, unknown, string>({
    mutationFn: (host: string) => api.post(`/registry-credentials/${encodeURIComponent(host)}/test`),
  })
}

export interface RegistryImage {
  repository: string
  tags: string[]
  error?: string
}
export interface RegistryImagesView {
  supported: boolean
  message?: string
  truncated: boolean
  images: RegistryImage[]
}

// Browse the images a registry holds (repositories + tags). `host` null keeps
// it idle until the operator opens the browser for a specific registry.
export function useRegistryImages(host: string | null) {
  return useQuery<RegistryImagesView>({
    queryKey: ['registry-images', host],
    queryFn: () => api.get(`/registry-credentials/${encodeURIComponent(host as string)}/images`),
    enabled: !!host,
  })
}

export function useImageBuildLog(buildId: string | undefined) {
  return useQuery<ImageBuildLogResponse>({
    queryKey: ['image-build-log', buildId],
    queryFn: () => api.get(`/image-builds/${buildId}/log`),
    enabled: !!buildId,
    // Tail live while the build is running; stop once it's done.
    refetchInterval: (query) => (query.state.data?.done ? false : 1200),
  })
}

// useBranches/useEnvironmentFiles back the configured-build form's real
// dropdowns (UI feedback: "branch, environment
// file, and builder on the new build form should all be select or
// dropdowns as we should have all of that information") -- real GitHub
// branches and a real parse of whatever environment files exist on the
// selected branch, not typed-in strings. See internal/api/repo_browse.go.
export function useBranches(repoId: string | undefined) {
  return useQuery<ListBranchesResponse>({
    queryKey: ['branches', repoId],
    queryFn: () => api.get(`/repos/${repoId}/branches`),
    enabled: !!repoId,
    retry: false,
  })
}

export function useBuilders() {
  return useQuery<BuilderSummary[]>({ queryKey: ['builders'], queryFn: () => api.get('/builders') })
}

export function useEnvironmentFiles(repoId: string | undefined, branch: string) {
  return useQuery<ListEnvironmentFilesResponse>({
    queryKey: ['environment-files', repoId, branch],
    queryFn: () => api.get(`/repos/${repoId}/environment-files?branch=${encodeURIComponent(branch)}`),
    enabled: !!repoId && !!branch,
    retry: false,
  })
}

// useConnectBuilder/useBuilderConnection back the
// builder config workflow (internal/api/builder_probe.go): connecting is one
// pasted Incus trust token -- LaForge verifies the server against the
// token's fingerprint, enrolls its own client certificate, and keeps the key
// server-side -- and every later choice is a pick from what the server
// actually has.
export function useConnectBuilder() {
  return useMutation({
    // kind selects the builder type's onboarding (the backend dispatches through
    // the builder.Onboarder registry); trust-token kinds send a token.
    mutationFn: (req: { kind: string; token: string; address?: string }) => api.post<BuilderConnection>('/builder-connections', req),
  })
}

export function useBuilderConnection(credentialId: string | undefined, kind?: string) {
  return useQuery<BuilderConnection>({
    queryKey: ['builder-connection', credentialId, kind],
    queryFn: () => api.get(`/builder-connections/${credentialId}${kind ? `?kind=${encodeURIComponent(kind)}` : ''}`),
    enabled: !!credentialId,
    retry: false,
    staleTime: 60_000,
  })
}

// useRepoAccess and friends back one repository's access page: everyone
// GitHub lists on the repository, each starting from their GitHub role, and
// an admin's per-person setting that replaces it.
export function useRepoAccess(repoId: string | undefined) {
  return useQuery<RepoAccess>({
    queryKey: ['repo-access', repoId],
    queryFn: () => api.get(`/repos/${repoId}/access`),
    enabled: !!repoId,
    retry: false,
  })
}

export function useMyRepoAccess(repoId: string | undefined) {
  return useQuery<MyRepoAccess>({
    queryKey: ['my-repo-access', repoId],
    queryFn: () => api.get(`/repos/${repoId}/my-access`),
    enabled: !!repoId,
    retry: false,
  })
}

export function usePeople(repoId: string | undefined, search: string) {
  return useQuery<Person[]>({
    queryKey: ['people', repoId, search],
    queryFn: () => api.get(`/repos/${repoId}/people${search ? `?q=${encodeURIComponent(search)}` : ''}`),
    enabled: !!repoId,
  })
}

export function useFindings(buildId: string | undefined) {
  return useQuery<FindingInstance[]>({
    queryKey: ['findings', buildId],
    queryFn: () => api.get(`/builds/${buildId}/findings`),
    enabled: !!buildId,
  })
}

export function useTopology(buildId: string | undefined) {
  return useQuery<TopologyResponse>({
    queryKey: ['topology', buildId],
    queryFn: () => api.get(`/builds/${buildId}/topology`),
    enabled: !!buildId,
  })
}

export function useArtifacts(buildId: string | undefined) {
  return useQuery<ArtifactsResponse>({
    queryKey: ['artifacts', buildId],
    queryFn: () => api.get(`/builds/${buildId}/artifacts`),
    enabled: !!buildId,
  })
}

export function usePurgeArtifacts(buildId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: () => api.post(`/builds/${buildId}/artifacts/purge`, {}),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['artifacts', buildId] }),
  })
}

export function useSetRepositoryAccess(repoId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ login, level }: { login: string; level: string }) =>
      api.put(`/repos/${repoId}/access/${encodeURIComponent(login)}`, { level }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['repo-access', repoId] })
      qc.invalidateQueries({ queryKey: ['my-repo-access', repoId] })
    },
  })
}

// useDeleteRepositoryAccess is the other half of "grant and revoke
// levels" -- DELETE /repos/{id}/access/{login}
// (internal/api/builds_ui.go's handleDeleteRepositoryAccess), found
// missing entirely by direct audit: the grant endpoint could only ever
// create or downgrade a level, never remove one.
export function useDeleteRepositoryAccess(repoId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (login: string) => api.delete(`/repos/${repoId}/access/${encodeURIComponent(login)}`),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['repo-access', repoId] })
      qc.invalidateQueries({ queryKey: ['my-repo-access', repoId] })
    },
  })
}

export function useAdHocTask(buildId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (req: { target: AdHocTarget; command: string; payload?: unknown; dry_run?: boolean }) =>
      api.post<AdHocResult>(`/builds/${buildId}/tasks`, req),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['objects', buildId] }),
  })
}

// usePowerAction runs an infrastructure power action (start/stop/reboot,
// hard or graceful) against the target hosts at the hoster -- not an agent
// task, so it works when the guest agent is dead. Refreshes the build so
// changed instance states show up.
export function usePowerAction(buildId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (req: { target: AdHocTarget; action: 'start' | 'stop' | 'reboot'; force: boolean }) =>
      api.post<PowerActionResponse>(`/builds/${buildId}/power`, req),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['objects', buildId] })
      qc.invalidateQueries({ queryKey: ['build', buildId] })
    },
  })
}

// useRebuild forces a tear-down-and-recreate of the target host(s) and, when
// include_dependents is set, everything in their team that depends on them.
// dry_run resolves the affected set without changing anything (the confirm
// dialog's blast-radius preview); a real rebuild refreshes the build so the
// objects' flip back through pending/deploying shows up.
export function useRebuild(buildId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (req: { target: AdHocTarget; include_dependents: boolean; dry_run?: boolean }) =>
      api.post<RebuildResponse>(`/builds/${buildId}/rebuild`, req),
    onSuccess: (_res, req) => {
      if (req.dry_run) return // a preview changed nothing
      qc.invalidateQueries({ queryKey: ['objects', buildId] })
      qc.invalidateQueries({ queryKey: ['build', buildId] })
    },
  })
}

// useScheduledTasks/useCreateScheduledTask/useCancelScheduledTask back
// "Build → Schedule: injects, upcoming and past... manage access can
// cancel or reschedule", plus ad-hoc scheduling
// ("schedule this for later" alongside useAdHocTask's "run it now") --
// same real dispatch path, see internal/api/scheduled_tasks.go.
export function useScheduledTasks(buildId: string | undefined) {
  return useQuery<ScheduledTask[]>({
    queryKey: ['scheduled-tasks', buildId],
    queryFn: () => api.get(`/builds/${buildId}/scheduled-tasks`),
    enabled: !!buildId,
    refetchInterval: 10_000,
  })
}

export function useCreateScheduledTask(buildId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (req: { target: AdHocTarget; when: string; command: string; payload?: unknown }) =>
      api.post<ScheduledTask>(`/builds/${buildId}/scheduled-tasks`, req),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['scheduled-tasks', buildId] }),
  })
}

// usePreviewSchedule turns a `when:` expression into the next few real fire
// times (server-side, the same NextFireAfter the dispatch loop uses), so the
// UI can show and confirm them before committing. Keyed on the expression so
// each distinct one is cached; disabled while empty.
export function usePreviewSchedule(buildId: string, when: string) {
  return useQuery<SchedulePreview>({
    queryKey: ['schedule-preview', buildId, when],
    queryFn: () => api.post(`/builds/${buildId}/scheduled-tasks/preview`, { when }),
    enabled: when.trim() !== '',
    staleTime: 30_000,
  })
}

export function useCancelScheduledTask(buildId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (taskId: string) => api.post<ScheduledTask>(`/builds/${buildId}/scheduled-tasks/${taskId}/cancel`, {}),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['scheduled-tasks', buildId] }),
  })
}

export function useSetTeamAccess(buildId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ team, action, extendMinutes, reduceMinutes }: { team: number; action: 'open' | 'close' | 'extend' | 'reduce'; extendMinutes?: number; reduceMinutes?: number }) =>
      api.post<Task | Team>(`/builds/${buildId}/teams/${team}/access`, { action, extend_minutes: extendMinutes, reduce_minutes: reduceMinutes }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['build', buildId] }),
  })
}

// useLiveEvents opens the real SSE stream (internal/api/live.go) and
// keeps a rolling buffer of what it's seen, reconnecting on drop --
// "Gateway unreachable from the browser: live updates degrade to
// polling with a visible banner" is approximated here
// by exposing `connected` so a caller can show that banner; the actual
// polling fallback is handleBuild's own refetchInterval, already running
// underneath regardless.
export function useLiveEvents(buildId: string | undefined, onEvent?: (ev: LFEvent) => void) {
  const [connected, setConnected] = useState(false)
  const onEventRef = useRef(onEvent)
  onEventRef.current = onEvent

  useEffect(() => {
    if (!buildId) return
    const source = new EventSource(`${API_BASE}/builds/${buildId}/live`, { withCredentials: true })
    source.onopen = () => setConnected(true)
    source.onerror = () => setConnected(false)
    // One fixed frame name ("laforge-event") for every real row -- see
    // live.go's own doc comment for why: EventSource has no wildcard
    // listener, so the server names every frame the same thing and puts
    // the actual distinguishing kind inside the JSON payload instead.
    source.addEventListener('laforge-event', (msg) => {
      try {
        onEventRef.current?.(JSON.parse((msg as MessageEvent).data) as LFEvent)
      } catch {
        /* malformed frame -- ignore rather than crash the stream handler */
      }
    })
    return () => {
      source.close()
      setConnected(false)
    }
  }, [buildId])

  return { connected }
}
