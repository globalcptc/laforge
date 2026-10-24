import { Link, useParams } from '@tanstack/react-router'
import { IdCard, ShieldCheck } from 'lucide-react'
import { useMyRepoAccess, useRepository } from '../api/hooks'
import { ConfiguredBuilds } from '../components/ConfiguredBuilds'
import { PageHeader } from '../ui'

// "Repository → Builds: Every build of this repository."
// Builds are grouped under the configured build that produced them, inside
// ConfiguredBuilds, rather than a second disconnected flat list.
export function RepoBuilds() {
  const { repoId } = useParams({ from: '/repos/$repoId' })
  const { data: repo } = useRepository(repoId)
  const { data: myAccess } = useMyRepoAccess(repoId)

  return (
    <>
      <PageHeader
        title={repo ? `${repo.github_owner}/${repo.github_repo}` : '…'}
        breadcrumbs={[{ label: 'Repositories', href: '/repos' }, { label: repo ? repo.github_repo : '…' }]}
        actions={
          repo && (
            <div className="flex items-center gap-4">
              {/* Identities (formerly People) demoted to a small secondary link
                 (direct product feedback: "it's getting first order level of
                 access when it's probably the least important part of the build"). */}
              <Link to="/repos/$repoId/people" params={{ repoId }} className="flex items-center gap-1 text-xs text-fg-subtle hover:text-fg-muted">
                <IdCard size={11} /> Identities
              </Link>
              {myAccess?.can_manage_access && (
                <Link to="/repos/$repoId/access" params={{ repoId }} className="flex items-center gap-1 text-xs font-medium text-accent hover:underline">
                  <ShieldCheck size={12} /> Access
                </Link>
              )}
            </div>
          )
        }
      />
      <div className="page-body">
        <ConfiguredBuilds repoId={repoId} />
      </div>
    </>
  )
}
