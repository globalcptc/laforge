import { useState } from 'react'
import { Link, useParams, useRouterState } from '@tanstack/react-router'
import { Boxes, ChevronDown, ChevronRight, FolderGit2, IdCard, Search, ShieldCheck } from 'lucide-react'
import { useMyRepoAccess, useRepositories } from '../api/hooks'
import type { Repository } from '../api/types'
import { Spinner, cn } from '../ui'

const STORAGE_KEY = 'laforge.nav.repos'

function readCollapsed(): Record<string, boolean> {
  try {
    return JSON.parse(localStorage.getItem(STORAGE_KEY) ?? '{}')
  } catch {
    return {}
  }
}

function writeCollapsed(v: Record<string, boolean>) {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(v))
  } catch {
    // Storage blocked -- the list just won't remember what was collapsed.
  }
}

// The sidebar's Repositories section: an expandable list of every
// repository this person can access, grouped under its GitHub organization.
// The repository being viewed opens to show its own pages. What's collapsed
// is remembered per browser.
export function RepoNav() {
  const { data: repos, isLoading } = useRepositories()
  const params = useParams({ strict: false }) as { repoId?: string }
  const pathname = useRouterState({ select: (s) => s.location.pathname })
  const [collapsed, setCollapsed] = useState<Record<string, boolean>>(readCollapsed)
  const [search, setSearch] = useState('')

  const toggle = (key: string) =>
    setCollapsed((c) => {
      const next = { ...c, [key]: !c[key] }
      writeCollapsed(next)
      return next
    })

  const q = search.trim().toLowerCase()
  const shown = (repos ?? []).filter((r) => !q || `${r.github_owner}/${r.github_repo}`.toLowerCase().includes(q))
  const byOwner = new Map<string, Repository[]>()
  for (const r of shown) byOwner.set(r.github_owner, [...(byOwner.get(r.github_owner) ?? []), r])
  const owners = [...byOwner.keys()].sort((a, b) => a.localeCompare(b))
  const open = !collapsed['__all']

  return (
    <div className="flex flex-col">
      {/* Icons line up with the nav links below (12px in); the chevron
          sits in the gutter to the left of the folder icon. */}
      <div className="flex items-center rounded-token text-sm hover:bg-surface-hover">
        <button
          type="button"
          onClick={() => toggle('__all')}
          aria-expanded={open}
          aria-label={open ? 'Collapse repositories' : 'Expand repositories'}
          className="flex h-9 w-3 shrink-0 items-center justify-center text-fg-subtle hover:text-fg"
        >
          {open ? <ChevronDown size={12} /> : <ChevronRight size={12} />}
        </button>
        <Link
          to="/repos"
          className={cn('flex min-w-0 flex-1 items-center gap-2.5 py-2 pr-3 font-medium', pathname === '/repos' ? 'text-accent-fg' : 'text-fg')}
        >
          <FolderGit2 className="size-4 shrink-0 text-fg-muted" />
          <span className="truncate">Repositories</span>
          {repos && <span className="ml-auto text-xs font-normal text-fg-subtle">{repos.length}</span>}
        </Link>
      </div>

      {open && (
        <div className="mt-1 flex flex-col gap-0.5">
          {(repos?.length ?? 0) > 8 && (
            <div className="mx-1 mb-1 flex items-center gap-1.5 rounded-token border border-border bg-surface px-2 py-1">
              <Search size={12} className="text-fg-subtle" />
              <input
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                placeholder="Filter"
                className="w-full bg-transparent text-xs text-fg outline-none placeholder:text-fg-subtle"
              />
            </div>
          )}
          {isLoading && (
            <div className="flex items-center gap-2 px-3 py-1.5 text-xs text-fg-muted">
              <Spinner /> Loading…
            </div>
          )}
          {!isLoading && shown.length === 0 && (
            <div className="px-3 py-1.5 text-xs text-fg-muted">{q ? 'No matches' : 'No repositories yet'}</div>
          )}
          {owners.map((owner) => {
            const ownerOpen = !collapsed[owner] || !!q
            return (
              <div key={owner}>
                <button
                  type="button"
                  onClick={() => toggle(owner)}
                  aria-expanded={ownerOpen}
                  className="flex w-full items-center gap-1 rounded-token py-1 pl-1 pr-3 text-left text-xs font-medium text-fg-muted hover:bg-surface-hover hover:text-fg"
                >
                  {ownerOpen ? <ChevronDown size={12} /> : <ChevronRight size={12} />}
                  <span className="truncate">{owner}</span>
                </button>
                {ownerOpen &&
                  byOwner.get(owner)!.map((r) => (
                    <RepoItem key={r.id} repo={r} current={r.id === params.repoId} pathname={pathname} />
                  ))}
              </div>
            )
          })}
        </div>
      )}
    </div>
  )
}

function RepoItem({ repo, current, pathname }: { repo: Repository; current: boolean; pathname: string }) {
  const base = `/repos/${repo.id}`
  return (
    <div>
      <Link
        to="/repos/$repoId"
        params={{ repoId: repo.id }}
        className={cn(
          'flex items-center gap-2 rounded-token py-1 pl-8 pr-3 text-sm hover:bg-surface-hover',
          current ? 'font-medium text-fg' : 'text-fg-muted hover:text-fg',
        )}
      >
        <span className="truncate">{repo.github_repo}</span>
      </Link>
      {current && <RepoPages repoId={repo.id} base={base} pathname={pathname} />}
    </div>
  )
}

function RepoPages({ repoId, base, pathname }: { repoId: string; base: string; pathname: string }) {
  const { data: myAccess } = useMyRepoAccess(repoId)
  const pages = [
    { to: '/repos/$repoId', label: 'Builds', icon: Boxes, active: pathname === base || pathname.startsWith(`${base}/builds`) },
    { to: '/repos/$repoId/people', label: 'Identities', icon: IdCard, active: pathname === `${base}/people` },
    ...(myAccess?.can_manage_access
      ? [{ to: '/repos/$repoId/access', label: 'Access', icon: ShieldCheck, active: pathname === `${base}/access` }]
      : []),
  ] as const
  return (
    <div className="ml-9 flex flex-col gap-0.5 border-l border-border py-0.5 pl-2">
      {pages.map((p) => (
        <Link
          key={p.label}
          to={p.to}
          params={{ repoId }}
          className={cn(
            'flex items-center gap-2 rounded-token px-2 py-1 text-xs',
            p.active ? 'bg-accent-soft text-accent-fg' : 'text-fg-muted hover:bg-surface-hover hover:text-fg',
          )}
        >
          <p.icon size={12} className="shrink-0" />
          {p.label}
        </Link>
      ))}
    </div>
  )
}
