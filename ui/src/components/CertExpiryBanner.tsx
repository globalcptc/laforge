import { AlertTriangle } from 'lucide-react'
import { useCertStatus } from '../api/hooks'
import { useTimeFormat } from '../lib/time'
import type { CertInfo } from '../api/types'
import { cn } from '../ui'

// An app-wide banner that warns before the agent trust anchor lapses. The agent
// mTLS CA signs every per-host agent certificate; if it (or the gateway's own
// cert) expires, every agent connection fails at once. GET /cert-status reports
// each cert's expiry, and this renders nothing until one is within the warning
// window (3 months) or already expired -- then a clear, unmissable banner.
export function CertExpiryBanner() {
  const { data } = useCertStatus()
  const fmt = useTimeFormat()
  if (!data) return null

  const problems: { label: string; info: CertInfo }[] = [
    { label: 'agent CA certificate', info: data.ca },
    { label: 'gateway certificate', info: data.server },
  ].filter((p) => p.info.configured && (p.info.expiring_soon || p.info.expired))
  if (problems.length === 0) return null

  const anyExpired = problems.some((p) => p.info.expired)
  return (
    <div
      role="alert"
      className={cn(
        'mx-4 mt-3 flex items-start gap-2 rounded-token border px-3 py-2 text-sm',
        anyExpired ? 'border-danger bg-danger-soft text-danger' : 'border-warning bg-warning-soft text-warning',
      )}
    >
      <AlertTriangle size={16} className="mt-0.5 shrink-0" />
      <div className="flex flex-col gap-0.5">
        {problems.map((p) => {
          const when = p.info.not_after ? fmt.dateTime(p.info.not_after) : 'an unknown date'
          return (
            <div key={p.label}>
              {p.info.expired ? (
                <>
                  The {p.label} <strong>expired</strong> on {when} — agents cannot connect until it is renewed.
                </>
              ) : (
                <>
                  The {p.label} expires in{' '}
                  <strong>
                    {p.info.days_remaining} day{p.info.days_remaining === 1 ? '' : 's'}
                  </strong>{' '}
                  (on {when}). Renew it before then, or agents will stop connecting.
                </>
              )}
            </div>
          )
        })}
      </div>
    </div>
  )
}
