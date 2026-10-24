import { Link } from '@tanstack/react-router'
import { ChevronRight, FolderGit2, GitBranchPlus } from 'lucide-react'
import { useRepositories, useUnapprovedInstalledRepositories } from '../api/hooks'
import { EmptyState } from '../components/EmptyState'
import { PageHeader, Spinner } from '../ui'

// Every repository this person can access (GET /repos, already scoped to
// them). Registering a repository always goes through "install on GitHub,
// approve in LaForge" (Admin → GitHub Connections); what this page does is
// surface that an approval is waiting, so an admin doesn't have to already
// know to go check.
export function Repositories() {
  const { data: repos, isLoading } = useRepositories()
  const { data: pending } = useUnapprovedInstalledRepositories()

  return (
    <>
      <PageHeader title="Repositories" description="Every repository registered with LaForge through a GitHub connection." />
      <div className="page-body">
        {pending && pending.length > 0 && (
          <Link
            to="/admin/installations"
            className="mb-4 flex items-center gap-2 rounded-token border border-warning/30 bg-warning-soft px-4 py-2 text-sm text-warning hover:border-warning/50"
          >
            <GitBranchPlus size={14} />
            {pending.length} repositor{pending.length === 1 ? 'y is' : 'ies are'} installed but not yet approved
            <ChevronRight size={14} className="ml-auto" />
          </Link>
        )}

        {isLoading && (
          <div className="flex items-center gap-2 text-sm text-fg-muted">
            <Spinner /> Loading…
          </div>
        )}

        {!isLoading && (!repos || repos.length === 0) && (
          <EmptyState icon={FolderGit2} title="No repositories registered yet" hint="Install the GitHub App on a repository, then approve it from Admin → GitHub Connections." />
        )}

        {repos && repos.length > 0 && (
          <div className="mx-auto flex max-w-3xl flex-col gap-2">
            {repos.map((r) => (
              <Link
                key={r.id}
                to="/repos/$repoId"
                params={{ repoId: r.id }}
                className="card flex items-center gap-3 px-4 py-3 hover:border-border-strong"
              >
                <FolderGit2 size={18} className="text-fg-muted" />
                <span className="font-medium text-fg">
                  {r.github_owner}/{r.github_repo}
                </span>
                <ChevronRight size={16} className="ml-auto text-fg-subtle" />
              </Link>
            ))}
          </div>
        )}
      </div>
    </>
  )
}
