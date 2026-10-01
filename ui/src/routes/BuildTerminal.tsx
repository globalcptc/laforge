import { useEffect, useRef, useState } from 'react'
import { Link, useParams } from '@tanstack/react-router'
import { Terminal as TerminalIcon, ArrowLeft } from 'lucide-react'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import '@xterm/xterm/css/xterm.css'
import { API_BASE } from '../api/client'
import { Badge, buttonClass } from '../ui'

type Status = 'connecting' | 'open' | 'closed'

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

  useEffect(() => {
    const el = mountRef.current
    if (!el) return

    const term = new Terminal({
      cursorBlink: true,
      fontSize: 13,
      fontFamily: 'var(--vscode-editor-font-family, ui-monospace, "JetBrains Mono", monospace)',
      theme: { background: '#0b0618' },
    })
    const fit = new FitAddon()
    term.loadAddon(fit)
    term.open(el)
    fit.fit()
    term.focus()

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
      window.removeEventListener('resize', onWindowResize)
      onData.dispose()
      onResize.dispose()
      ws.close()
      term.dispose()
    }
  }, [buildId, objectId])

  const tone = status === 'open' ? 'success' : status === 'connecting' ? 'warning' : 'neutral'
  const label = status === 'open' ? 'Connected' : status === 'connecting' ? 'Connecting…' : 'Closed'

  return (
    <div className="flex h-full flex-col gap-3">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2 text-sm font-semibold text-fg">
          <TerminalIcon size={15} className="text-fg-muted" /> Terminal
          <Badge tone={tone} className="ml-1">
            {label}
          </Badge>
        </div>
        <Link to="/repos/$repoId/builds/$buildId/hosts" params={{ repoId, buildId }} className={buttonClass({ variant: 'ghost', size: 'sm' })}>
          <ArrowLeft size={14} /> Back to hosts
        </Link>
      </div>
      <p className="text-xs text-fg-muted">
        A live root/admin shell on this host through its agent, encrypted end to end. Closing this page ends the session.
      </p>
      <div ref={mountRef} className="min-h-0 flex-1 overflow-hidden rounded-token border border-border bg-[#0b0618] p-2" />
    </div>
  )
}
