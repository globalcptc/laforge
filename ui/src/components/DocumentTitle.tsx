import { useRouterState } from '@tanstack/react-router'
import { useEffect } from 'react'
import { useBuild, useRepository } from '../api/hooks'

// The browser tab title for every page, kept short: "LaForge", then at most one
// item of context, then the page -- "LaForge · cptc12-central · Hosts" on a build
// (its environment), "LaForge · cptc12 · Builds" on a repository, "LaForge ·
// Infra" elsewhere. Names come from the same queries the pages themselves load,
// so it costs no extra requests. The standalone terminal window sets its own
// title (BuildTerminal), so it's left alone here.
const BUILD_SECTIONS: Record<string, string> = {
  '': 'Overview',
  hosts: 'Hosts',
  access: 'Access',
  'external-access': 'External Access',
  logs: 'Logs',
  findings: 'Findings',
  schedule: 'Schedule',
  topology: 'Topology',
  artifacts: 'Artifacts',
  admin: 'Admin',
}

const REPO_SECTIONS: Record<string, string> = {
  '': 'Builds',
  people: 'People',
  access: 'Access',
}

const PAGES: [RegExp, string | ((m: RegExpMatchArray) => string)][] = [
  [/^\/$/, 'Home'],
  [/^\/sign-in$/, 'Sign In'],
  [/^\/repos$/, 'Repos'],
  [/^\/admin\/installations$/, 'Installations'],
  [/^\/admin\/admins$/, 'Admins'],
  [/^\/admin\/infrastructure$/, 'Infra'],
  [/^\/admin\/infrastructure\/new$/, 'New Builder'],
  [/^\/admin\/infrastructure\/([^/]+)\/edit$/, (m) => `Edit ${decodeURIComponent(m[1])}`],
  [/^\/admin\/infrastructure\/([^/]+)$/, (m) => decodeURIComponent(m[1])],
  [/^\/settings\/api-keys$/, 'API Keys'],
  [/^\/admin\/api-keys$/, 'All API Keys'],
]

export function DocumentTitle() {
  const pathname = useRouterState({ select: (s) => s.location.pathname }).replace(/\/+$/, '') || '/'
  const repoMatch = pathname.match(/^\/repos\/([^/]+)(?:\/(.*))?$/)
  const repoId = repoMatch?.[1]
  const rest = repoMatch?.[2] ?? ''
  const buildMatch = rest.match(/^builds\/([^/]+)(?:\/(.*))?$/)
  const buildId = buildMatch?.[1]
  const section = (buildMatch?.[2] ?? '').split('/')[0]
  const terminal = /\/terminal$/.test(pathname)

  const { data: repo } = useRepository(repoId)
  const { data: build } = useBuild(terminal ? undefined : buildId)

  useEffect(() => {
    if (terminal) return
    const parts = ['LaForge']
    if (repoId) {
      if (buildId) {
        if (build?.environment_name) parts.push(build.environment_name)
        parts.push(BUILD_SECTIONS[section] ?? 'Build')
      } else {
        if (repo) parts.push(repo.github_repo)
        parts.push(REPO_SECTIONS[rest] ?? 'Repository')
      }
    } else {
      for (const [re, label] of PAGES) {
        const m = pathname.match(re)
        if (m) {
          parts.push(typeof label === 'string' ? label : label(m))
          break
        }
      }
    }
    document.title = parts.join(' · ')
  }, [pathname, terminal, repoId, buildId, section, rest, repo, build?.environment_name])

  return null
}
