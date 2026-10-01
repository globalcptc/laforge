import { createRootRoute, createRoute, createRouter, Navigate } from '@tanstack/react-router'
import { Shell } from './components/Shell'
import { SignIn } from './routes/SignIn'
import { Home } from './routes/Home'
import { Repositories } from './routes/Repositories'
import { RepoBuilds } from './routes/RepoBuilds'
import { BuildLayout } from './routes/BuildLayout'
import { BuildOverview } from './routes/BuildOverview'
import { BuildHosts } from './routes/BuildHosts'
import { BuildTerminal } from './routes/BuildTerminal'
import { BuildAccess } from './routes/BuildAccess'
import { BuildExternalAccess } from './routes/BuildExternalAccess'
import { BuildLogs } from './routes/BuildLogs'
import { BuildFindings } from './routes/BuildFindings'
import { BuildSchedule } from './routes/BuildSchedule'
import { BuildArtifacts } from './routes/BuildArtifacts'
import { RepoPeople } from './routes/RepoPeople'
import { RepoAccess } from './routes/RepoAccess'
import { InstalledRepositories } from './routes/InstalledRepositories'
import { BuilderConfigs } from './routes/BuilderConfigs'
import { BuilderWizard } from './routes/BuilderWizard'
import { useMe } from './api/hooks'
import { Spinner } from './ui'
import type { ReactNode } from 'react'

// RequireAuth is the auth gate -- "Sections the user cannot use are
// hidden... a deep link they lack access to explains itself" per-repo is
// handled server-side (401/403 from the API); this is the coarser
// "not signed in at all" gate that sends an anonymous visitor to
// /sign-in rather than showing them a wall of loading spinners and
// failed requests.
function RequireAuth({ children }: { children: ReactNode }) {
  const { data: me, isLoading, isError } = useMe()
  if (isLoading)
    return (
      <div className="flex items-center gap-2 p-6 text-sm text-fg-muted">
        <Spinner /> Loading…
      </div>
    )
  if (isError || !me) return <Navigate to="/sign-in" />
  return <>{children}</>
}

const rootRoute = createRootRoute({ component: Shell })

const signInRoute = createRoute({ getParentRoute: () => rootRoute, path: '/sign-in', component: SignIn })

const indexRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/',
  component: () => (
    <RequireAuth>
      <Home />
    </RequireAuth>
  ),
})

const reposRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/repos',
  component: () => (
    <RequireAuth>
      <Repositories />
    </RequireAuth>
  ),
})

const repoRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/repos/$repoId',
  component: () => (
    <RequireAuth>
      <RepoBuilds />
    </RequireAuth>
  ),
})

const buildLayoutRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/repos/$repoId/builds/$buildId',
  component: () => (
    <RequireAuth>
      <BuildLayout />
    </RequireAuth>
  ),
})

const buildIndexRoute = createRoute({ getParentRoute: () => buildLayoutRoute, path: '/', component: BuildOverview })
const buildHostsRoute = createRoute({ getParentRoute: () => buildLayoutRoute, path: 'hosts', component: BuildHosts })
const buildAccessRoute = createRoute({ getParentRoute: () => buildLayoutRoute, path: 'access', component: BuildAccess })
const buildExternalAccessRoute = createRoute({ getParentRoute: () => buildLayoutRoute, path: 'external-access', component: BuildExternalAccess })
const buildLogsRoute = createRoute({ getParentRoute: () => buildLayoutRoute, path: 'logs', component: BuildLogs })
const buildFindingsRoute = createRoute({ getParentRoute: () => buildLayoutRoute, path: 'findings', component: BuildFindings })
const buildScheduleRoute = createRoute({ getParentRoute: () => buildLayoutRoute, path: 'schedule', component: BuildSchedule })
// The old standalone Topology view was folded into Hosts (its content facts --
// CIDR, OS/image, size, dependencies -- now render on the host lines). Keep the
// path as a redirect so existing links/bookmarks still land somewhere sensible.
const buildTopologyRoute = createRoute({
  getParentRoute: () => buildLayoutRoute,
  path: 'topology',
  component: () => {
    const { repoId, buildId } = buildLayoutRoute.useParams()
    return <Navigate to="/repos/$repoId/builds/$buildId/hosts" params={{ repoId, buildId }} replace />
  },
})
const buildArtifactsRoute = createRoute({ getParentRoute: () => buildLayoutRoute, path: 'artifacts', component: BuildArtifacts })
// Per-host interactive shell -- reached from a host's "Open terminal" action,
// not a build-level tab, since a shell is always about one specific host.
const buildTerminalRoute = createRoute({ getParentRoute: () => buildLayoutRoute, path: 'hosts/$objectId/terminal', component: BuildTerminal })

const repoPeopleRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/repos/$repoId/people',
  component: () => (
    <RequireAuth>
      <RepoPeople />
    </RequireAuth>
  ),
})

const installedRepositoriesRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/admin/installations',
  component: () => (
    <RequireAuth>
      <InstalledRepositories />
    </RequireAuth>
  ),
})

const builderConfigsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/admin/infrastructure',
  component: () => (
    <RequireAuth>
      <BuilderConfigs />
    </RequireAuth>
  ),
})

const newBuilderRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/admin/infrastructure/new',
  component: () => (
    <RequireAuth>
      <BuilderWizard />
    </RequireAuth>
  ),
})

const editBuilderRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/admin/infrastructure/$name',
  component: () => (
    <RequireAuth>
      <BuilderWizard />
    </RequireAuth>
  ),
})

const repoAccessRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/repos/$repoId/access',
  component: () => (
    <RequireAuth>
      <RepoAccess />
    </RequireAuth>
  ),
})

const routeTree = rootRoute.addChildren([
  indexRoute,
  signInRoute,
  reposRoute,
  repoRoute,
  repoPeopleRoute,
  installedRepositoriesRoute,
  builderConfigsRoute,
  newBuilderRoute,
  editBuilderRoute,
  repoAccessRoute,
  buildLayoutRoute.addChildren([buildIndexRoute, buildHostsRoute, buildAccessRoute, buildExternalAccessRoute, buildLogsRoute, buildFindingsRoute, buildScheduleRoute, buildTopologyRoute, buildArtifactsRoute, buildTerminalRoute]),
])

export const router = createRouter({ routeTree })

declare module '@tanstack/react-router' {
  interface Register {
    router: typeof router
  }
}
