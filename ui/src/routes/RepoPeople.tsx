import { useParams } from '@tanstack/react-router'
import { useMemo, useState } from 'react'
import { IdCard, Search } from 'lucide-react'
import { usePeople } from '../api/hooks'
import { EmptyState } from '../components/EmptyState'
import { PageHeader, Spinner, Table, TableScroller, Td, Th } from '../ui'

// "People: the user database from this repo's CSVs. Searchable by name,
// username, department, title. Shows passwords, because during an event
// that is the point." Columns are computed from
// whatever attributes the real CSV actually has (people are free-form --
// see internal/api/people.go), not a fixed schema, so a repo with
// different CSV columns gets a different table for free.
export function RepoPeople() {
  const { repoId } = useParams({ from: '/repos/$repoId/people' })
  const [search, setSearch] = useState('')
  const { data: people, isLoading } = usePeople(repoId, search)

  const columns = useMemo(() => {
    const seen = new Set<string>()
    for (const p of people ?? []) {
      for (const key of Object.keys(p.attributes)) seen.add(key)
    }
    // A stable, sensible order for the common CSV columns this project's
    // own example content uses, then whatever else shows up.
    const preferred = ['first_name', 'last_name', 'title', 'department', 'password', 'domain_admin', 'sudo']
    return [...preferred.filter((c) => seen.has(c)), ...[...seen].filter((c) => !preferred.includes(c))]
  }, [people])

  return (
    <>
      <PageHeader title="Identities" breadcrumbs={[{ label: 'Repositories', href: '/repos' }, { label: 'Repository', href: `/repos/${repoId}` }, { label: 'Identities' }]} />
      <div className="page-body">
        <div className="mb-4 flex items-center gap-2 rounded-token border border-border bg-surface-raised px-3 py-1.5">
          <Search size={14} className="text-fg-muted" />
          <input
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Search name, username, department, title…"
            className="w-full bg-transparent text-sm text-fg outline-none placeholder:text-fg-subtle"
          />
        </div>

        {isLoading && (
          <div className="flex items-center gap-2 text-sm text-fg-muted">
            <Spinner /> Loading…
          </div>
        )}

        {!isLoading && (!people || people.length === 0) && (
          <EmptyState
            icon={IdCard}
            title={search ? 'No matches' : 'No identities yet'}
            hint={search ? 'Try a different search term.' : 'Add a people/*.csv to this repository’s content to populate this list.'}
          />
        )}

        {people && people.length > 0 && (
          <TableScroller contain={false}>
            <Table>
              <thead>
                <tr>
                  <Th>Username</Th>
                  {columns.map((c) => (
                    <Th key={c}>{c.replace(/_/g, ' ')}</Th>
                  ))}
                  <Th>Source</Th>
                </tr>
              </thead>
              <tbody>
                {people.map((p) => (
                  <tr key={p.id}>
                    <Td className="font-medium text-fg">{p.username}</Td>
                    {columns.map((c) => (
                      <Td key={c} className="text-fg-muted">
                        {p.attributes[c] ?? '—'}
                      </Td>
                    ))}
                    <Td className="text-xs text-fg-muted">{p.people_source_name}</Td>
                  </tr>
                ))}
              </tbody>
            </Table>
          </TableScroller>
        )}
      </div>
    </>
  )
}
