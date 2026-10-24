import type { LucideIcon } from 'lucide-react'

// "Every list has a real one that explains what would populate it and
// how." The design system's
// own EmptyState (src/ui/primitives.tsx) has no icon slot -- this app's
// own copy keeps one, since every real usage here leads with a Lucide
// icon that names what's missing at a glance.
export function EmptyState({
  icon: Icon,
  title,
  hint,
  action,
}: {
  icon: LucideIcon
  title: string
  hint?: string
  action?: React.ReactNode
}) {
  return (
    <div className="flex flex-col items-center justify-center gap-2 rounded-token-lg border border-dashed border-border px-6 py-16 text-center">
      <Icon size={28} className="text-fg-subtle" />
      <div className="text-sm font-medium text-fg">{title}</div>
      {hint && <div className="max-w-sm text-xs text-fg-muted">{hint}</div>}
      {action}
    </div>
  )
}
