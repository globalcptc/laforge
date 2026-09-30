import { useState } from 'react'
import { Link } from '@tanstack/react-router'
import { ExternalLink, GitBranchPlus, ShieldCheck } from 'lucide-react'
import { useInstallations, useApproveInstalledRepository, useMe } from '../api/hooks'
import { ApiError } from '../api/client'
import { EmptyState } from '../components/EmptyState'
import { InstallGitHubAppLink } from '../components/InstallGitHubAppLink'
import type { Installation, InstalledRepo } from '../api/types'
import { Badge, Button, Card, CardHeader, PageHeader, Spinner, useToast } from '../ui'

// "Install on GitHub, approve in LaForge" -- and, per
// direct product feedback, the real
// answer to "Installations should list everywhere we have the app
// installed that we know about and allow us to manage it, even if it's
// just us linking to GitHub pages": every real installation
// (GET /installations), not just the ones with something still pending
// -- the old screen only ever listed unapproved repos, so an
// installation with everything already approved simply vanished from the
// one place meant to make every installation manageable. The one place
// "Install GitHub App" lives now (see Home.tsx, which used to also offer
// it -- moved here so there's exactly one entry point, not two).
export function InstalledRepositories() {
  const { data: installations, isLoading, error } = useInstallations()
  const { data: me } = useMe()
  const isInstanceAdmin = !!me?.is_instance_admin

  if (error) {
    const forbidden = error instanceof ApiError && error.status === 403
    return (
      <div className="mx-auto max-w-2xl p-6">
        <Card className="p-4 text-sm text-fg">
          {forbidden
            ? "You don't administer any installations, so there's nothing to manage here."
            : error instanceof ApiError
              ? error.message
              : 'Failed to load installations.'}
        </Card>
      </div>
    )
  }

  return (
    <>
      <PageHeader
        title="GitHub Connections"
        description="Every GitHub organization or account the LaForge app is installed on, and the repositories it covers."
        actions={isInstanceAdmin ? <InstallGitHubAppLink /> : undefined}
      />
      <div className="page-body flex flex-col gap-4">
        {isLoading && (
          <div className="flex items-center gap-2 text-sm text-fg-muted">
            <Spinner /> Loading…
          </div>
        )}

        {!isLoading && (!installations || installations.length === 0) && (
          <EmptyState icon={GitBranchPlus} title="No GitHub connections yet" hint="Install the GitHub App on an organization or account and it'll show up here." />
        )}

        {installations?.map((inst) => (
          <InstallationCard key={inst.id} installation={inst} canApprove={isInstanceAdmin} />
        ))}
      </div>
    </>
  )
}

// githubManageURL is "even if it's just us linking to GitHub pages" made
// real: GitHub's own installation settings page, where add/remove
// repository and uninstall actually live -- an organization and a
// personal account use different URL shapes for the same real page.
function githubManageURL(installation: Installation): string {
  if (installation.account_type === 'Organization') {
    return `https://github.com/organizations/${installation.account_login}/settings/installations/${installation.installation_id}`
  }
  return `https://github.com/settings/installations/${installation.installation_id}`
}

function InstallationCard({ installation, canApprove }: { installation: Installation; canApprove: boolean }) {
  return (
    <Card>
      <CardHeader
        title={
          <span className="flex items-center gap-2">
            {installation.account_login}
            <Badge tone="neutral">{installation.account_type}</Badge>
            {installation.suspended && <Badge tone="warning">Suspended</Badge>}
          </span>
        }
        description={`${installation.repos.length} repositor${installation.repos.length === 1 ? 'y' : 'ies'}`}
        actions={
          <a href={githubManageURL(installation)} target="_blank" rel="noreferrer" className="flex items-center gap-1 text-xs text-fg-muted hover:text-fg">
            Manage on GitHub <ExternalLink size={12} />
          </a>
        }
      />
      {installation.repos.length === 0 ? (
        <div className="px-4 py-3 text-sm text-fg-muted">No repositories selected for this installation yet.</div>
      ) : (
        <div className="divide-y divide-border">
          {installation.repos.map((repo) => (
            <RepoRow key={repo.github_repo_id} repo={repo} installationId={installation.id} canApprove={canApprove} />
          ))}
        </div>
      )}
    </Card>
  )
}

function RepoRow({ repo, installationId, canApprove }: { repo: InstalledRepo; installationId: string; canApprove: boolean }) {
  const approve = useApproveInstalledRepository()
  const [error, setError] = useState<string | null>(null)
  const toast = useToast()

  async function onApprove() {
    setError(null)
    try {
      await approve.mutateAsync({ owner: repo.github_owner, repo: repo.github_repo, installation_id: installationId })
      toast({ title: 'Repository approved', description: `${repo.github_owner}/${repo.github_repo}`, tone: 'success' })
    } catch (e) {
      const message = e instanceof ApiError ? e.message : 'Approve failed'
      setError(message)
      toast({ title: 'Approve failed', description: message, tone: 'danger', duration: 0 })
    }
  }

  return (
    <div className="flex items-center justify-between px-4 py-2">
      <span className="font-medium text-fg">
        {repo.github_owner}/{repo.github_repo}
      </span>
      {repo.approved ? (
        <div className="flex items-center gap-3">
          {repo.repository_id && (
            <Link
              to="/repos/$repoId/access"
              params={{ repoId: repo.repository_id }}
              className="flex items-center gap-1 text-xs font-medium text-accent hover:underline"
            >
              <ShieldCheck size={12} /> Access
            </Link>
          )}
          <Badge tone="success">Approved</Badge>
        </div>
      ) : !canApprove ? (
        <Badge tone="warning">Awaiting approval</Badge>
      ) : (
        <div className="flex items-center gap-2">
          {error && <span className="text-xs text-danger">{error}</span>}
          <Button variant="primary" size="sm" onClick={onApprove} disabled={approve.isPending}>
            {approve.isPending ? 'Approving…' : 'Approve'}
          </Button>
        </div>
      )}
    </div>
  )
}
