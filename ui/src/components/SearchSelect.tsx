import { useEffect, useRef, useState } from 'react'
import { Check, ChevronsUpDown, Search } from 'lucide-react'
import { AnchoredPopover, Spinner, cn } from '../ui'

export interface SearchOption {
  value: string
  label: string
  hint?: string
}

// SearchSelect is the shared searchable combobox (the pattern BranchPicker /
// FilePicker hand-roll): a trigger button + an AnchoredPopover with a filter
// input and option rows, keyboard-navigable. Two modes:
//   - client filter (default): filters `options` by the typed text.
//   - server search (`onSearch` set): debounces the typed text and calls
//     onSearch(q) so the owner can fetch more options (used for AMI name search);
//     `options` is still shown as-is, and short queries are ignored.
export function SearchSelect({
  value,
  options,
  onChange,
  onSearch,
  placeholder = 'Choose…',
  searchPlaceholder,
  loading = false,
  emptyLabel = 'No matches',
  minSearchChars = 2,
}: {
  value: string
  options: SearchOption[]
  onChange: (value: string) => void
  onSearch?: (query: string) => void
  placeholder?: string
  searchPlaceholder?: string
  loading?: boolean
  emptyLabel?: string
  minSearchChars?: number
}) {
  const [open, setOpen] = useState(false)
  const [search, setSearch] = useState('')
  const anchorRef = useRef<HTMLButtonElement>(null)
  const panelRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!open) return
    const onPointerDown = (e: PointerEvent) => {
      const target = e.target as Node
      if (anchorRef.current?.contains(target) || panelRef.current?.contains(target)) return
      setOpen(false)
    }
    window.addEventListener('pointerdown', onPointerDown)
    return () => window.removeEventListener('pointerdown', onPointerDown)
  }, [open])

  // Server search: debounce the query (and ignore too-short ones) so typing
  // doesn't fire a request per keystroke. onSearch is called through a ref so a
  // caller that passes a fresh function each render (the common case) doesn't
  // re-trigger the debounce on every re-render -- which otherwise loops: fetch →
  // state update → re-render → new onSearch identity → effect re-fires → fetch.
  // The effect depends ONLY on the typed text.
  const onSearchRef = useRef(onSearch)
  useEffect(() => {
    onSearchRef.current = onSearch
  }, [onSearch])
  const hasSearch = !!onSearch
  useEffect(() => {
    if (!hasSearch) return
    const q = search.trim()
    if (q.length > 0 && q.length < minSearchChars) return
    const t = setTimeout(() => onSearchRef.current?.(q), 350)
    return () => clearTimeout(t)
  }, [search, hasSearch, minSearchChars])

  const q = search.trim().toLowerCase()
  // With a server search the list is already scoped; client-filter only in the
  // default (no onSearch) mode.
  const shown = onSearch ? options : options.filter((o) => !q || o.label.toLowerCase().includes(q) || o.value.toLowerCase().includes(q) || (o.hint ?? '').toLowerCase().includes(q))

  // Remember every label we've seen so the trigger keeps showing the chosen
  // option's name even after a later server search drops it from `options`
  // (otherwise the button flips between the name and the raw value as results
  // churn). Ref mutation during render is fine -- it's a cache, not state.
  const labelCache = useRef<Record<string, string>>({})
  for (const o of options) labelCache.current[o.value] = o.label
  const displayLabel = value ? labelCache.current[value] ?? value : placeholder

  function choose(v: string) {
    onChange(v)
    setOpen(false)
    setSearch('')
  }

  return (
    <>
      <button
        ref={anchorRef}
        type="button"
        onClick={() => setOpen((v) => !v)}
        aria-haspopup="listbox"
        aria-expanded={open}
        className="flex h-9 w-full items-center gap-2 rounded-token border border-border bg-surface-raised px-3 text-left text-sm hover:border-border-strong"
      >
        <span className={cn('min-w-0 flex-1 truncate', value ? 'font-mono text-fg' : 'text-fg-subtle')}>
          {displayLabel}
        </span>
        <ChevronsUpDown size={14} className="shrink-0 text-fg-subtle" />
      </button>
      {open && (
        <AnchoredPopover anchorRef={anchorRef} panelRef={panelRef} minWidth={320} maxHeight={340} role="listbox">
          <div className="flex items-center gap-2 border-b border-border px-3 py-2">
            <Search size={12} className="text-fg-muted" />
            <input
              autoFocus
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter' && shown[0]) choose(shown[0].value)
                if (e.key === 'Escape') setOpen(false)
              }}
              placeholder={searchPlaceholder ?? 'Search…'}
              className="w-full bg-transparent text-sm text-fg outline-none placeholder:text-fg-subtle"
            />
            {loading && <Spinner />}
          </div>
          <div className="p-1">
            {shown.length === 0 && (
              <div className="px-2.5 py-2 text-xs text-fg-muted">
                {onSearch && q.length > 0 && q.length < minSearchChars ? `Type at least ${minSearchChars} characters` : emptyLabel}
              </div>
            )}
            {shown.map((o) => (
              <button
                key={o.value}
                type="button"
                role="option"
                aria-selected={o.value === value}
                onClick={() => choose(o.value)}
                className={cn(
                  'flex w-full items-center gap-2 rounded-token-sm px-2.5 py-1.5 text-left text-sm hover:bg-surface-hover',
                  o.value === value ? 'text-accent' : 'text-fg',
                )}
              >
                <span className="flex min-w-0 flex-1 flex-col">
                  <span className="truncate">{o.label}</span>
                  {o.hint && <span className="truncate font-mono text-xs text-fg-subtle">{o.hint}</span>}
                </span>
                {o.value === value && <Check size={14} className="shrink-0" />}
              </button>
            ))}
          </div>
        </AnchoredPopover>
      )}
    </>
  )
}
