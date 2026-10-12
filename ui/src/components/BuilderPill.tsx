import { Boxes, Cloud, FlaskConical, Server, type LucideIcon } from 'lucide-react'
import { cn } from '../ui'

// Builder kind -> icon, so a colored pill can still carry the family glyph.
// Mirrors BuilderKindBadge's map; kept here so a pill works standalone.
const KIND_ICON: Record<string, LucideIcon> = {
  microcloud: Cloud,
  incus: Boxes,
  aws: Cloud,
  openstack: Server,
  fake: FlaskConical,
}

// contrastText picks black or white for text sitting on the given "#rrggbb"
// background, by relative luminance (WCAG-ish). Keeps a user's chosen pill
// color readable whatever they pick.
function contrastText(hex: string): string {
  const h = hex.replace('#', '')
  if (h.length !== 6) return '#000'
  const r = parseInt(h.slice(0, 2), 16) / 255
  const g = parseInt(h.slice(2, 4), 16) / 255
  const b = parseInt(h.slice(4, 6), 16) / 255
  const lin = (c: number) => (c <= 0.03928 ? c / 12.92 : Math.pow((c + 0.055) / 1.055, 2.4))
  const lum = 0.2126 * lin(r) + 0.7152 * lin(g) + 0.0722 * lin(b)
  return lum > 0.5 ? '#111' : '#fff'
}

// BuilderPill renders a builder's name as a colored pill using the builder's
// user-chosen color. With no color it falls back to a neutral token pill, so
// it is safe to use everywhere a builder name appears regardless of whether a
// color was set. `icon` adds the kind glyph; `title` overrides the hover text.
export function BuilderPill({
  name,
  color,
  kind,
  icon = true,
  className,
  title,
}: {
  name?: string | null
  color?: string | null
  kind?: string
  icon?: boolean
  className?: string
  title?: string
}) {
  if (!name) return null
  const Icon = kind ? KIND_ICON[kind] : undefined
  const colored = !!color && /^#[0-9a-fA-F]{6}$/.test(color)
  const style = colored ? { backgroundColor: color!, color: contrastText(color!), borderColor: color! } : undefined
  return (
    <span
      className={cn(
        'inline-flex shrink-0 items-center gap-1 rounded-full border px-2 py-0.5 text-xs font-medium',
        !colored && 'border-border bg-surface-sunken text-fg',
        className,
      )}
      style={style}
      title={title ?? name}
    >
      {icon && Icon && <Icon size={11} />}
      {name}
    </span>
  )
}
