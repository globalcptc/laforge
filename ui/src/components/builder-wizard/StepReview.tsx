import { Badge } from '../../ui'
import { KIND_LABEL, isCloud, shortFingerprint, type Draft } from './model'

export function StepReview({ draft }: { draft: Draft }) {
  const cloud = isCloud(draft.kind)
  return (
    <div className="flex flex-col gap-4 text-sm">
      <Section title="Builder">
        <Row label="Name">
          <span className="font-mono">{draft.name}</span>
        </Row>
        <Row label="Type">{KIND_LABEL[draft.kind]}</Row>
      </Section>

      {cloud && (
        <>
          <Section title="Credentials">
            <Row label="Source">Runner environment ({draft.kind === 'aws' ? 'AWS_* / AWS_REGION' : 'OS_* / OS_REGION_NAME'})</Row>
          </Section>
          <Section title="Images">
            {draft.images.filter((i) => i.name && i.fingerprint).length === 0 && <div className="px-3 py-2 text-fg-muted">None mapped.</div>}
            {draft.images.filter((i) => i.name && i.fingerprint).map((i) => (
              <Row key={i.name} label={<span className="font-mono">{i.name}</span>}>
                <span className="font-mono">{i.fingerprint}</span>
              </Row>
            ))}
          </Section>
          <Section title="Sizes">
            {draft.sizes.filter((s) => s.name && s.type).length === 0 && <div className="px-3 py-2 text-fg-muted">None mapped.</div>}
            {draft.sizes.filter((s) => s.name && s.type).map((s) => (
              <Row key={s.name} label={<span className="font-mono">{s.name}</span>}>
                <span className="font-mono">{s.type}</span>
              </Row>
            ))}
          </Section>
        </>
      )}

      {draft.kind !== 'fake' && !cloud && (
        <>
          <Section title={draft.kind === 'incus' ? `Hosts (${draft.hosts.length})` : 'Cluster'}>
            {draft.hosts.map((h, i) => (
              <Row key={h.key} label={draft.kind === 'incus' ? `Host ${i + 1}` : 'Server'}>
                <div>
                  {h.connection?.credential.server_name || h.connection?.credential.api_url || h.legacy?.api_url || 'Saved connection'}
                </div>
                <div className="text-xs text-fg-muted">
                  pool <span className="font-mono">{h.storagePool || '—'}</span> · uplink <span className="font-mono">{h.uplink || '—'}</span>
                </div>
              </Row>
            ))}
          </Section>

          <Section title="Images">
            {draft.images.length === 0 && <div className="px-3 py-2 text-fg-muted">None offered.</div>}
            {draft.images.map((i) => (
              <Row key={i.name} label={<span className="font-mono">{i.name}</span>}>
                <span className="font-mono">{i.alias || shortFingerprint(i.fingerprint)}</span>{' '}
                <Badge tone={i.vm ? 'info' : 'neutral'}>{i.vm ? 'VM' : 'Container'}</Badge>
                {i.server && <span className="ml-1 text-xs text-fg-muted">downloaded on first use</span>}
              </Row>
            ))}
          </Section>

          <Section title="Sizes">
            {draft.sizes.map((s) => (
              <Row key={s.name} label={<span className="font-mono">{s.name}</span>}>
                {s.cpu} CPU{s.cpu === '1' ? '' : 's'} · {s.memory.replace('MiB', ' MB').replace('GiB', ' GB')}
              </Row>
            ))}
          </Section>

        </>
      )}
    </div>
  )
}

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div>
      <div className="mb-1.5 text-xs font-semibold uppercase tracking-wide text-fg-muted">{title}</div>
      <div className="card divide-y divide-border">{children}</div>
    </div>
  )
}

function Row({ label, children }: { label: React.ReactNode; children: React.ReactNode }) {
  return (
    <div className="flex gap-4 px-3 py-2">
      <div className="w-40 shrink-0 break-all text-fg-muted">{label}</div>
      <div className="min-w-0 flex-1 text-fg">{children}</div>
    </div>
  )
}
