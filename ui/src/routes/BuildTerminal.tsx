import { useEffect, useRef, useState } from 'react'
import { Link, useParams } from '@tanstack/react-router'
import { Terminal as TerminalIcon, ArrowLeft } from 'lucide-react'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import '@xterm/xterm/css/xterm.css'
import { API_BASE } from '../api/client'
import { useDeployedObjects } from '../api/hooks'
import type { ObjectStatus } from '../api/types'
import { StatusBadge } from '../components/StatusBadge'
import { Badge, buttonClass, EmptyState, Spinner } from '../ui'

type Status = 'connecting' | 'open' | 'closed'

// A shell exists only on a host/container that is actually up with a live agent.
// This MUST match the api's shellReachable (internal/api/terminal.go): the
// handshake is refused with 409 for any other status, which a browser surfaces
// as a bare "can't establish a connection" error. We check the status up front
// so the page can explain *why* instead of dialing a socket that will be
// refused.
const SHELL_REACHABLE: ReadonlySet<ObjectStatus> = new Set<ObjectStatus>(['running', 'building', 'finished'])

// Why a shell can't be opened, in plain language, per non-reachable status.
function shellUnavailableReason(status: ObjectStatus): string {
  switch (status) {
    case 'pending':
    case 'deploying':
      return 'This host isn’t running yet — a terminal needs a live agent. Try again once it’s up.'
    case 'deploy_failed':
      return 'This host failed to deploy, so there’s no agent to open a shell on.'
    case 'build_failed':
      return 'This host’s build failed, so it has no live agent to open a shell on.'
    case 'invalid':
      return 'This host is in an invalid state and can’t open a shell.'
    case 'destroying':
    case 'destroyed':
      return 'This host has been torn down, so there’s no shell to open.'
    default:
      return 'This host isn’t in a state that can open a shell — a terminal needs a running host with a live agent.'
  }
}

// BuildTerminal is a live, interactive root/admin shell on one host, opened
// through its agent and relayed over a WebSocket to the api (GET
// /builds/{id}/objects/{objectId}/terminal). xterm.js renders a real terminal;
// bytes flow both ways as binary WebSocket messages, and a small JSON control
// message carries resizes. The session cookie authenticates the handshake (same
// origin, or SameSite=None cross-origin), so no token handling is needed here.
export function BuildTerminal() {
  const { repoId, buildId, objectId } = useParams({ from: '/repos/$repoId/builds/$buildId/hosts/$objectId/terminal' })
  const mountRef = useRef<HTMLDivElement>(null)
  const [status, setStatus] = useState<Status>('connecting')

  // The host's own status decides whether a shell is even possible. undefined
  // while the object list is still loading (or the object is gone); a boolean
  // once we know. We gate the socket on this so we never dial a doomed handshake.
  const objects = useDeployedObjects(buildId)
  const obj = objects.data?.find((o) => o.id === objectId)
  const reachable = obj ? SHELL_REACHABLE.has(obj.status) : undefined

  useEffect(() => {
    // Only open the socket when we know the host can take a shell. While the
    // status is still loading (undefined) or not reachable (false) we render an
    // explanation instead, so there's no mount div to attach to anyway.
    if (reachable !== true) return
    const el = mountRef.current
    if (!el) return

    const term = new Terminal({
      cursorBlink: true,
      fontSize: 13,
      // A concrete font stack, NOT a CSS var: xterm measures the character cell by
      // setting this on a canvas ctx.font, and canvas can't parse var(...) -- it
      // then measured with a fallback font whose metrics didn't match the rendered
      // glyphs, squashing the rows. "JetBrains Mono Variable" is the family the
      // app's @fontsource-variable/jetbrains-mono actually registers.
      fontFamily: '"JetBrains Mono Variable", "JetBrains Mono", ui-monospace, "SF Mono", Menlo, Consolas, monospace',
      theme: { background: '#0b0618' },
    })
    const fit = new FitAddon()
    term.loadAddon(fit)
    term.open(el)
    fit.fit()
    term.focus()

    // Re-fit once the web font is actually loaded: xterm measures the character
    // cell from the font, so if it measured before @fontsource JetBrains Mono was
    // ready, the rows would be sized to a fallback font (wrong ratio). Guarded so
    // it never runs against a disposed terminal.
    let disposed = false
    document.fonts?.ready.then(() => {
      if (!disposed) {
        try {
          fit.fit()
        } catch {
          /* terminal already gone */
        }
      }
    })

    // http(s)://host → ws(s)://host, same origin as the api.
    const base = new URL(API_BASE, window.location.href)
    base.protocol = base.protocol === 'https:' ? 'wss:' : 'ws:'
    const url = `${base.origin}${base.pathname.replace(/\/$/, '')}/builds/${buildId}/objects/${objectId}/terminal`

    const ws = new WebSocket(url)
    ws.binaryType = 'arraybuffer'
    const enc = new TextEncoder()

    const sendResize = () => {
      if (ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ type: 'resize', cols: term.cols, rows: term.rows }))
      }
    }

    ws.onopen = () => {
      setStatus('open')
      sendResize()
      term.focus()
    }
    ws.onmessage = (ev) => {
      if (ev.data instanceof ArrayBuffer) term.write(new Uint8Array(ev.data))
      else if (typeof ev.data === 'string') term.write(ev.data)
    }
    ws.onclose = () => {
      setStatus('closed')
      term.write('\r\n\x1b[2m[ session closed ]\x1b[0m\r\n')
    }
    ws.onerror = () => setStatus('closed')

    const onData = term.onData((d) => {
      if (ws.readyState === WebSocket.OPEN) ws.send(enc.encode(d))
    })
    const onResize = term.onResize(() => sendResize())
    const onWindowResize = () => fit.fit()
    window.addEventListener('resize', onWindowResize)

    return () => {
      disposed = true
      window.removeEventListener('resize', onWindowResize)
      onData.dispose()
      onResize.dispose()
      ws.close()
      term.dispose()
    }
  }, [buildId, objectId, reachable])

  const tone = status === 'open' ? 'success' : status === 'connecting' ? 'warning' : 'neutral'
  const label = status === 'open' ? 'Connected' : status === 'connecting' ? 'Connecting…' : 'Closed'

  return (
    <div className="flex h-full flex-col gap-3">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2 text-sm font-semibold text-fg">
          <TerminalIcon size={15} className="text-fg-muted" /> Terminal
          {reachable === true && (
            <Badge tone={tone} className="ml-1">
              {label}
            </Badge>
          )}
        </div>
        <Link to="/repos/$repoId/builds/$buildId/hosts" params={{ repoId, buildId }} className={buttonClass({ variant: 'ghost', size: 'sm' })}>
          <ArrowLeft size={14} /> Back to hosts
        </Link>
      </div>
      <p className="text-xs text-fg-muted">
        A live root/admin shell on this host through its agent, encrypted end to end. Closing this page ends the session.
      </p>
      {reachable === true ? (
        <div ref={mountRef} className="min-h-0 flex-1 overflow-hidden rounded-token border border-border bg-[#0b0618] p-2" />
      ) : (
        <div className="flex min-h-0 flex-1 items-center justify-center rounded-token border border-border bg-surface-sunken">
          {objects.isLoading || (reachable === undefined && !objects.isError) ? (
            <Spinner />
          ) : !obj ? (
            <EmptyState title="Host not found" description="This host is no longer part of the build." />
          ) : (
            <EmptyState
              title="No shell available"
              description={
                <span className="flex flex-col items-center gap-2">
                  <StatusBadge status={obj.status} />
                  <span>{shellUnavailableReason(obj.status)}</span>
                </span>
              }
            />
          )}
        </div>
      )}
    </div>
  )
}
