import { useEffect, useState } from 'react'
import { Check, X } from 'lucide-react'
import { useDecideOAuthRequest, useMe, useOAuthRequest } from '../api/hooks'
import { API_BASE, ApiError } from '../api/client'
import { AuthBackdrop, Button, Card, CardBody, Spinner } from '../ui'

// Where ChatGPT or Claude.ai send a person to connect to LaForge's MCP server
// (internal/api/oauth_server.go). It signs them in if needed, says what the
// app will be able to do, and on Approve sends the browser back to the app
// with a one-time code the app trades for an API key.
export function OAuthConsent() {
  const requestId = new URLSearchParams(window.location.search).get('request') ?? ''
  const { data: me, isLoading: meLoading, isError: signedOut } = useMe()
  const { data: req, error: reqError, isLoading: reqLoading } = useOAuthRequest(requestId, !!me)
  const decide = useDecideOAuthRequest()
  const [error, setError] = useState<string | null>(null)
  const [leaving, setLeaving] = useState(false)

  // Not signed in: GitHub sign-in, then straight back here.
  useEffect(() => {
    if (!meLoading && signedOut && requestId) {
      const back = `/oauth/consent?request=${encodeURIComponent(requestId)}`
      window.location.href = `${API_BASE}/auth/github/login?return_to=${encodeURIComponent(back)}`
    }
  }, [meLoading, signedOut, requestId])

  async function onDecide(approve: boolean) {
    setError(null)
    try {
      const { redirect_to } = await decide.mutateAsync({ id: requestId, approve })
      setLeaving(true)
      window.location.href = redirect_to
    } catch (e) {
      setError(e instanceof ApiError ? e.message : 'Something went wrong. Try again from the app.')
    }
  }

  let body: React.ReactNode
  if (!requestId) {
    body = <p className="text-sm text-fg-muted">This page needs a sign-in request. Start again from the app you're connecting.</p>
  } else if (meLoading || signedOut || reqLoading || leaving) {
    body = (
      <div className="flex items-center justify-center gap-2 text-sm text-fg-muted">
        <Spinner /> {leaving ? 'Returning to the app…' : 'Loading…'}
      </div>
    )
  } else if (reqError || !req) {
    body = (
      <p className="text-sm text-fg-muted">
        {reqError instanceof ApiError ? reqError.message : 'This sign-in link expired.'} Start again from the app you're connecting.
      </p>
    )
  } else {
    body = (
      <>
        <div className="text-center">
          <h1 className="text-lg font-semibold tracking-tight text-fg">Connect {req.client_name} To LaForge</h1>
          <div className="mt-1 font-mono text-xs text-fg-muted">{req.redirect_host}</div>
        </div>
        <div className="space-y-2 text-sm text-fg">
          <p>
            {req.client_name} will read everything you can read in LaForge, including host passwords and rendered scripts. It can power hosts on, off
            or through a reboot in repositories where you have manage access.
          </p>
          <p className="text-fg-muted">
            It can't rebuild, deploy, tear down or run commands. Access lasts {req.expires_in_days} days, and you can revoke it any time under Connected
            Apps on the API Keys page.
          </p>
        </div>
        {me && (
          <div className="flex items-center gap-2 text-xs text-fg-muted">
            {me.avatar_url && <img src={me.avatar_url} alt="" className="size-5 rounded-full" />}
            Signed in as <span className="font-mono text-fg">@{me.github_login}</span>
          </div>
        )}
        {error && <div className="rounded-token border border-danger/30 bg-danger-soft p-2 text-xs text-danger">{error}</div>}
        <div className="flex gap-2">
          <Button variant="danger" className="flex-1" onClick={() => onDecide(false)} disabled={decide.isPending}>
            <X size={14} /> Deny
          </Button>
          <Button variant="primary" className="flex-1" onClick={() => onDecide(true)} disabled={decide.isPending}>
            {decide.isPending ? <Spinner /> : <Check size={14} />} Approve
          </Button>
        </div>
      </>
    )
  }

  return (
    <main className="relative grid min-h-dvh place-items-center px-4">
      <AuthBackdrop />
      <div className="relative z-10 w-full max-w-md">
        <Card className="border-white/60 bg-surface-raised/95 shadow-panel backdrop-blur-xl">
          <CardBody className="flex flex-col gap-5 py-8">
            <img src="/cptc-icon.png" alt="" width={44} height={44} className="mx-auto" />
            {body}
          </CardBody>
        </Card>
      </div>
    </main>
  )
}
