import { useConfig } from '../api/hooks'
import { buttonClass, cn } from '../ui'
import type { ButtonProps } from '../ui'

// "Install on GitHub, approve in LaForge" -- the
// real install link, github.com/apps/<slug>/installations/new, built
// from this instance's own App slug (GET /config,
// internal/api/config.go) rather than hard-coded, since a self-hosted
// instance can register its own App under any name. Shared by Home.tsx
// (the primary "add a repository" entry point) and
// InstalledRepositories.tsx (the approval screen this link feeds), so
// the two screens never drift.
//
// Renders nothing at all once config has loaded and there's no App
// configured (github_app_slug empty) -- a deployment can genuinely run
// with no App at all (Server.AppPrivateKey's own doc comment), and there
// is no install link to offer in that case.
export function InstallGitHubAppLink({ variant = 'primary', className }: { variant?: ButtonProps['variant']; className?: string }) {
  const { data: config, isLoading } = useConfig()

  if (!isLoading && config && !config.github_app_slug) return null

  return (
    <a
      href={config ? `https://github.com/apps/${config.github_app_slug}/installations/new` : undefined}
      target="_blank"
      rel="noreferrer"
      className={cn(buttonClass({ variant, size: 'sm' }), !config && 'pointer-events-none opacity-50', className)}
      aria-disabled={!config}
    >
      Install GitHub App
    </a>
  )
}
