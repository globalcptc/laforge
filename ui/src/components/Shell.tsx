import { useQueryClient } from '@tanstack/react-query'
import { Link, Outlet, useRouterState } from '@tanstack/react-router'
import { GitBranchPlus, House, ServerCog, ShieldCheck } from 'lucide-react'
import { AppShell, BrandMark, cn, type NavSection } from '../ui'
import { RepoNav } from './RepoNav'
import { UserMenu } from './UserMenu'
import { GlobalAlerts } from './GlobalAlerts'
import { CertExpiryBanner } from './CertExpiryBanner'
import { api } from '../api/client'
import { useMe } from '../api/hooks'

// Access is per repository (a repository's own Access page, reached from
// the Repositories list above these sections), not app-wide.
const sections: NavSection[] = [
  {
    title: 'Admin',
    items: [
      { href: '/admin/installations', label: 'GitHub Connections', icon: <GitBranchPlus className="size-4" /> },
      { href: '/admin/infrastructure', label: 'Infrastructure', icon: <ServerCog className="size-4" /> },
      { href: '/admin/admins', label: 'Admins', icon: <ShieldCheck className="size-4" /> },
    ],
  },
]

// The global shell -- scoped down from the full version
// (the context switcher, attention/message centre badges).
// What IS real: session-aware sign-in state, and sign-out
// actually ending the server-side session (not just forgetting the
// cookie client-side). Sign-in itself is gated at the router level
// (RequireAuth in router.tsx), so this shell only ever renders once a
// real session exists, except for /sign-in's own route which bypasses
// it entirely.
export function Shell() {
  const { data: me } = useMe()
  const qc = useQueryClient()
  const pathname = useRouterState({ select: (s) => s.location.pathname })

  async function signOut() {
    await api.post('/auth/logout')
    qc.setQueryData(['me'], undefined)
    qc.invalidateQueries()
  }

  if (pathname === '/sign-in' || !me) {
    return <Outlet />
  }

  return (
    <AppShell
      activePath={pathname}
      sidebarTop={
        <div className="flex flex-col gap-1">
          <Link
            to="/"
            className={cn(
              'flex items-center gap-2.5 rounded-token px-3 py-2 text-sm font-medium',
              pathname === '/' ? 'bg-accent-soft text-accent-fg' : 'text-fg hover:bg-surface-hover',
            )}
          >
            <House className="size-4 shrink-0" />
            Home
          </Link>
          <RepoNav />
        </div>
      }
      sections={sections}
      navbar={
        <div className="flex min-w-0 flex-1 items-center justify-between gap-3">
          <BrandMark src="/cptc-mark.webp" size={28} product="LaForge" />
          <div className="flex items-center gap-1">
            <GlobalAlerts />
            <UserMenu me={me} onSignOut={signOut} />
          </div>
        </div>
      }
    >
      <CertExpiryBanner />
      <Outlet />
    </AppShell>
  )
}
