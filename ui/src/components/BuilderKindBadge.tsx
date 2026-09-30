import { Boxes, Cloud, FlaskConical, Server, type LucideIcon } from 'lucide-react'
import { Badge, cn } from '../ui'

// Builder kind -> icon + display name. One generic cloud/host icon per family;
// the label is what distinguishes them. Shared by the Configured Builds header
// pill and the build page's "infrastructure" pill.
const BUILDER_KIND: Record<string, { icon: LucideIcon; label: string }> = {
  microcloud: { icon: Cloud, label: 'MicroCloud' },
  incus: { icon: Boxes, label: 'Incus' },
  aws: { icon: Cloud, label: 'AWS' },
  openstack: { icon: Server, label: 'OpenStack' },
  fake: { icon: FlaskConical, label: 'Fake' },
}

// `label` overrides the visible text (used on the build header to show the
// configured builder's name instead of its type); the type stays in the
// hover title either way. `name` is the configured builder's name, for the
// tooltip.
export function BuilderKindBadge({ kind, name, label, className }: { kind?: string; name?: string; label?: string; className?: string }) {
  if (!kind) return null
  const m = BUILDER_KIND[kind] ?? { icon: Server, label: kind }
  const Icon = m.icon
  const title = name ? `Builder: ${name} (${m.label})` : `Builder type: ${m.label}`
  return (
    <Badge tone="neutral" className={cn('shrink-0 rounded-full py-0.5', className)} title={title}>
      <Icon size={11} /> {label ?? m.label}
    </Badge>
  )
}
