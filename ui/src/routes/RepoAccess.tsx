import { useState } from 'react'
import { useParams } from '@tanstack/react-router'
import { Check, Minus, RefreshCw, RotateCcw, Search, ShieldCheck } from 'lucide-react'
import { useDeleteRepositoryAccess, useMe, useRepoAccess, useSetRepositoryAccess } from '../api/hooks'
import { ApiError } from '../api/client'
import { EmptyState } from '../components/EmptyState'
import type { RepoAccessPerson, RepoLevel } from '../api/types'
import { Badge, Button, Input, PageHeader, Select, Spinner, Table, TableScroller, Td, Th, cn, useToast } from '../ui'

const LEVELS: RepoLevel[] = ['none', 'read', 'build', 'manage', 'admin']

const LEVEL_LABEL: Record<RepoLevel, string> = {
  none: 'No access',
  read: 'Read',
  build: 'Build',
  manage: 'Manage',
  admin: 'Admin',
}

// What each level allows, as the API enforces it (internal/api: the level each
// handler's requireLevel asks for). min is the lowest level that can do it;
// every level above it can too.
const GUIDE_LEVELS: { level: RepoLevel; github: string }[] = [
  { level: 'read', github: 'Read, Triage' },
  { level: 'build', github: 'Write, Maintain' },
  { level: 'manage', github: 'Set here only' },
  { level: 'admin', github: 'Admin' },
]

const ACTIONS: { action: string; min: RepoLevel }[] = [
  { action: 'See builds, hosts and logs', min: 'read' },
  { action: 'See host passwords', min: 'read' },
  { action: 'See external access endpoints', min: 'read' },
  { action: 'Start a build', min: 'build' },
  { action: 'Deploy a build', min: 'build' },
  { action: 'Apply content changes', min: 'build' },
  { action: 'Change configured build settings', min: 'build' },
  { action: 'Rebuild a host', min: 'manage' },
  { action: 'Restart a host’s steps', min: 'manage' },
  { action: 'Power hosts on or off', min: 'manage' },
  { action: 'Open a terminal', min: 'manage' },
  { action: 'Run ad-hoc and scheduled tasks', min: 'manage' },
  { action: 'Change team access windows', min: 'manage' },
  { action: 'Check drift, purge artifacts', min: 'manage' },
  { action: 'Destroy a build', min: 'manage' },
  { action: 'Change access on this page', min: 'admin' },
]

const LEVEL_RANK: Record<RepoLevel, number> = { none: 0, read: 1, build: 2, manage: 3, admin: 4 }

// One repository's access. GitHub decides who is listed (everyone with
// access to the repository there) and where each person starts (their
// GitHub role); an admin can set anyone to a different level, on this
// repository only.
export function RepoAccess() {
  const { repoId } = useParams({ from: '/repos/$repoId/access' })
  const { data, isLoading, error, refetch, isFetching } = useRepoAccess(repoId)
  const { data: me } = useMe()
  const [filter, setFilter] = useState('')

  const repo = data?.repository
  const title = repo ? `${repo.github_owner}/${repo.github_repo}` : '…'

  if (error) {
    const forbidden = error instanceof ApiError && error.status === 403
    return (
      <div className="mx-auto max-w-2xl p-6">
        <div className="card p-4 text-sm text-fg">
          {forbidden ? "Only this repository's admins can see or change its access." : error instanceof ApiError ? error.message : 'Failed to load access.'}
        </div>
      </div>
    )
  }

  const people = data?.people ?? []
  const q = filter.trim().toLowerCase()
  const rows = q ? people.filter((p) => p.login.toLowerCase().includes(q)) : people
  const changed = people.filter((p) => p.level !== '').length

  return (
    <>
      <PageHeader
        title="Access"
        description={`Who can do what on ${title}. Everyone with access on GitHub is listed at the level their GitHub role gives them; set a different level for anyone. This repository only.`}
        breadcrumbs={[{ label: 'Repositories', href: '/repos' }, { label: repo?.github_repo ?? '…', href: `/repos/${repoId}` }, { label: 'Access' }]}
      />
      <div className="page-body flex flex-col gap-3">
        {isLoading || !data ? (
          <div className="flex items-center gap-2 text-sm text-fg-muted">
            <Spinner /> Reading collaborators from GitHub…
          </div>
        ) : (
          <>
            <div className="flex flex-wrap items-center gap-3">
              <div className="relative">
                <Search size={14} className="pointer-events-none absolute left-2.5 top-1/2 -translate-y-1/2 text-fg-subtle" />
                <Input value={filter} onChange={(e) => setFilter(e.target.value)} placeholder="Filter people" className="h-8 w-56 pl-8 text-sm" />
              </div>
              <span className="text-xs text-fg-muted">
                {people.length} {people.length === 1 ? 'person' : 'people'} · {changed} changed from GitHub
              </span>
              <Button variant="ghost" size="sm" onClick={() => refetch()} disabled={isFetching} className="ml-auto">
                {isFetching ? <Spinner /> : <RefreshCw size={12} />} Refresh
              </Button>
            </div>

            {data.github_error && (
              <div className="rounded-token border border-warning/30 bg-warning-soft px-3 py-2 text-sm text-warning">
                Couldn't read this repository's collaborators from GitHub, so only people with a level set here are listed. ({data.github_error})
              </div>
            )}

            {people.length === 0 ? (
              <EmptyState
                icon={ShieldCheck}
                title="No one found"
                hint="No one has access to this repository on GitHub. Add collaborators on GitHub, then refresh."
              />
            ) : (
              <TableScroller contain={false}>
                <Table>
                  <thead>
                    <tr>
                      <Th>Person</Th>
                      <Th>GitHub role</Th>
                      <Th>LaForge access</Th>
                      <Th className="w-24" />
                    </tr>
                  </thead>
                  <tbody>
                    {rows.map((p) => (
                      <PersonRow key={p.login} repoId={repoId} person={p} isMe={p.login.toLowerCase() === me?.github_login.toLowerCase()} />
                    ))}
                    {rows.length === 0 && (
                      <tr>
                        <Td colSpan={4} className="text-fg-muted">
                          No one matches "{filter}".
                        </Td>
                      </tr>
                    )}
                  </tbody>
                </Table>
              </TableScroller>
            )}

            <LevelGuide />
          </>
        )}
      </div>
    </>
  )
}

function PersonRow({ repoId, person, isMe }: { repoId: string; person: RepoAccessPerson; isMe: boolean }) {
  const set = useSetRepositoryAccess(repoId)
  const reset = useDeleteRepositoryAccess(repoId)
  const toast = useToast()
  const busy = set.isPending || reset.isPending
  const onGithub = person.github_role !== ''
  const isChanged = person.level !== ''

  async function onLevel(next: string) {
    try {
      if (next === '') await reset.mutateAsync(person.login)
      else await set.mutateAsync({ login: person.login, level: next })
      toast({
        title: 'Access updated',
        description: `${person.login}: ${next === '' ? `back to GitHub (${LEVEL_LABEL[person.github_level]})` : LEVEL_LABEL[next as RepoLevel]}`,
        tone: 'success',
      })
    } catch (err) {
      toast({ title: 'Access change failed', description: err instanceof ApiError ? err.message : undefined, tone: 'danger', duration: 0 })
    }
  }

  return (
    <tr>
      <Td>
        <span className="flex items-center gap-2 font-medium text-fg">
          {person.avatar_url ? (
            <img src={person.avatar_url} alt="" className="size-5 rounded-full" />
          ) : (
            <span className="size-5 rounded-full bg-surface-sunken" />
          )}
          {person.login}
          {isMe && <span className="text-xs font-normal text-fg-subtle">(you)</span>}
        </span>
      </Td>
      <Td>
        {onGithub ? (
          <span className="text-fg">{person.github_role}</span>
        ) : (
          <Badge tone="neutral">Not on GitHub</Badge>
        )}
      </Td>
      <Td>
        <div className="flex items-center gap-2">
          <Select
            value={person.level}
            disabled={busy}
            onChange={(e) => onLevel(e.target.value)}
            className={cn('h-8 w-52 text-xs', isChanged ? 'border-accent/50 text-accent' : 'text-fg')}
            aria-label={`${person.login} access`}
          >
            <option value="">{onGithub ? `Inherit from GitHub (${person.github_role})` : 'No access (not on GitHub)'}</option>
            {LEVELS.map((l) => (
              <option key={l} value={l}>
                {LEVEL_LABEL[l]}
              </option>
            ))}
          </Select>
          {isChanged && <Badge tone="info">Changed</Badge>}
        </div>
      </Td>
      <Td className="text-right">
        {isChanged && (
          <Button variant="ghost" size="sm" onClick={() => onLevel('')} disabled={busy} title="Go back to their GitHub role">
            <RotateCcw size={12} /> Reset
          </Button>
        )}
      </Td>
    </tr>
  )
}

// LevelGuide is the levels as a grid: one row per action, one column per level,
// a check where that level can do it. The first row names the GitHub role that
// starts someone at each level.
function LevelGuide() {
  return (
    <section className="flex flex-col gap-2">
      <h2 className="text-sm font-semibold text-fg">What each level can do</h2>
      <TableScroller contain={false}>
        <Table>
          <thead>
            <tr>
              <Th>Action</Th>
              {GUIDE_LEVELS.map((g) => (
                <Th key={g.level} className="w-28 text-center">
                  {LEVEL_LABEL[g.level]}
                </Th>
              ))}
            </tr>
          </thead>
          <tbody>
            {/* Where people start: the level each GitHub role gives them. */}
            <tr className="bg-info-soft text-xs text-info">
              <Td className="text-[11px] font-medium italic text-info">GitHub Default</Td>
              {GUIDE_LEVELS.map((g) => (
                <Td key={g.level} className="text-center text-xs text-info">
                  {g.github}
                </Td>
              ))}
            </tr>
            {ACTIONS.map((a, i) => (
              <tr key={a.action} className={i % 2 === 1 ? 'bg-surface-sunken' : undefined}>
                <Td className="text-fg">{a.action}</Td>
                {GUIDE_LEVELS.map((g) => (
                  <Td key={g.level} className="text-center">
                    {LEVEL_RANK[g.level] >= LEVEL_RANK[a.min] ? (
                      <Check size={16} className="inline text-brand" aria-label="Yes" />
                    ) : (
                      <Minus size={14} className="inline text-fg-subtle" aria-label="No" />
                    )}
                  </Td>
                ))}
              </tr>
            ))}
          </tbody>
        </Table>
      </TableScroller>
      <p className="text-xs text-fg-muted">
        No access can’t see the repository. Instance admins have Admin on every repository.
      </p>
    </section>
  )
}
