import { useEffect, useMemo, useRef, useState } from 'react'
import { Check, ChevronDown, LogOut, Monitor, Moon, Sun } from 'lucide-react'
import { AnchoredPopover, Badge, Select, cn, type ThemePreference } from '../ui'
import { DARK_QUERY, THEME_STORAGE_KEY } from '../ui/theme-script'
import { useSetTimezone } from '../api/hooks'
import type { Me } from '../api/types'

// Every IANA zone the browser knows, for the timezone picker; a short
// fallback covers the rare engine without Intl.supportedValuesOf.
function allTimeZones(): string[] {
  try {
    const fn = (Intl as unknown as { supportedValuesOf?: (k: string) => string[] }).supportedValuesOf
    if (fn) return fn('timeZone')
  } catch {
    // fall through
  }
  return ['UTC', 'America/New_York', 'America/Chicago', 'America/Denver', 'America/Los_Angeles', 'Europe/London', 'Europe/Berlin', 'Asia/Tokyo']
}

const THEMES: { value: ThemePreference; label: string; icon: typeof Monitor }[] = [
  { value: 'system', label: 'System', icon: Monitor },
  { value: 'light', label: 'Light', icon: Sun },
  { value: 'dark', label: 'Dark', icon: Moon },
]

// Same storage and rules as the pre-paint script and ThemeToggle: an
// explicit choice is stored, no entry means follow the OS.
function readTheme(): ThemePreference {
  try {
    const stored = localStorage.getItem(THEME_STORAGE_KEY)
    return stored === 'light' || stored === 'dark' ? stored : 'system'
  } catch {
    return 'system'
  }
}

function applyTheme(pref: ThemePreference) {
  const dark = pref === 'dark' || (pref === 'system' && window.matchMedia(DARK_QUERY).matches)
  if (dark) document.documentElement.dataset.theme = 'dark'
  else delete document.documentElement.dataset.theme
}

function Avatar({ url, size }: { url?: string; size: string }) {
  return url ? <img src={url} alt="" className={cn(size, 'rounded-full')} /> : <span className={cn(size, 'rounded-full bg-surface-sunken')} />
}

// The signed-in person's menu in the navbar: who they are, the theme, and
// signing out -- one control instead of a row of separate buttons.
export function UserMenu({ me, onSignOut }: { me: Me; onSignOut: () => void }) {
  const [open, setOpen] = useState(false)
  const [theme, setTheme] = useState<ThemePreference>(readTheme)
  const anchorRef = useRef<HTMLButtonElement>(null)
  const panelRef = useRef<HTMLDivElement>(null)
  const setTimezone = useSetTimezone()
  const zones = useMemo(allTimeZones, [])

  // While following the OS, track it live.
  useEffect(() => {
    if (theme !== 'system') return
    const media = window.matchMedia(DARK_QUERY)
    const onChange = () => applyTheme('system')
    media.addEventListener('change', onChange)
    return () => media.removeEventListener('change', onChange)
  }, [theme])

  useEffect(() => {
    if (!open) return
    const onPointerDown = (e: PointerEvent) => {
      const target = e.target as Node
      if (anchorRef.current?.contains(target) || panelRef.current?.contains(target)) return
      setOpen(false)
    }
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') setOpen(false)
    }
    window.addEventListener('pointerdown', onPointerDown)
    window.addEventListener('keydown', onKey)
    return () => {
      window.removeEventListener('pointerdown', onPointerDown)
      window.removeEventListener('keydown', onKey)
    }
  }, [open])

  function chooseTheme(next: ThemePreference) {
    try {
      if (next === 'system') localStorage.removeItem(THEME_STORAGE_KEY)
      else localStorage.setItem(THEME_STORAGE_KEY, next)
    } catch {
      // Storage blocked: the choice still applies for this page view.
    }
    applyTheme(next)
    setTheme(next)
  }

  return (
    <>
      <button
        ref={anchorRef}
        type="button"
        onClick={() => setOpen((v) => !v)}
        aria-haspopup="menu"
        aria-expanded={open}
        className="flex items-center gap-2 rounded-token px-1.5 py-1 text-sm text-fg-muted hover:bg-surface-hover hover:text-fg"
      >
        <Avatar url={me.avatar_url} size="size-7" />
        <span className="hidden sm:inline">@{me.github_login}</span>
        <ChevronDown size={14} className="text-fg-subtle" />
      </button>
      {open && (
        <AnchoredPopover anchorRef={anchorRef} panelRef={panelRef} minWidth={240} maxHeight={360} role="menu">
          <div className="flex items-center gap-3 border-b border-border px-3 py-3">
            <Avatar url={me.avatar_url} size="size-9" />
            <div className="min-w-0">
              <div className="truncate text-sm font-medium text-fg">@{me.github_login}</div>
              <div className="text-xs text-fg-muted">Signed in with GitHub</div>
              {me.is_instance_admin && (
                <Badge tone="info" className="mt-1">
                  Instance admin
                </Badge>
              )}
            </div>
          </div>
          <div className="border-b border-border p-1">
            <div className="px-2.5 pb-1 pt-1.5 text-[11px] font-semibold uppercase tracking-wide text-fg-subtle">Theme</div>
            {THEMES.map((t) => (
              <button
                key={t.value}
                type="button"
                role="menuitemradio"
                aria-checked={theme === t.value}
                onClick={() => chooseTheme(t.value)}
                className={cn(
                  'flex w-full items-center gap-2 rounded-token-sm px-2.5 py-1.5 text-left text-sm text-fg hover:bg-surface-hover',
                  theme === t.value && 'text-accent',
                )}
              >
                <t.icon className="size-3.5" aria-hidden />
                <span className="flex-1">{t.label}</span>
                {theme === t.value && <Check className="size-3.5" aria-hidden />}
              </button>
            ))}
          </div>
          <div className="border-b border-border p-1">
            <div className="px-2.5 pb-1 pt-1.5 text-[11px] font-semibold uppercase tracking-wide text-fg-subtle">Timezone</div>
            <div className="px-1.5 pb-1.5">
              <Select
                value={me.timezone ?? ''}
                onChange={(e) => setTimezone.mutate(e.target.value)}
                disabled={setTimezone.isPending}
                className="w-full"
                aria-label="Timezone"
              >
                <option value="">System default ({Intl.DateTimeFormat().resolvedOptions().timeZone})</option>
                {zones.map((z) => (
                  <option key={z} value={z}>
                    {z}
                  </option>
                ))}
              </Select>
              <div className="mt-1 px-0.5 text-[11px] text-fg-subtle">Timestamps across the app render in this zone.</div>
            </div>
          </div>
          <div className="p-1">
            <button
              type="button"
              role="menuitem"
              onClick={() => {
                setOpen(false)
                onSignOut()
              }}
              className="flex w-full items-center gap-2 rounded-token-sm px-2.5 py-1.5 text-left text-sm text-fg hover:bg-surface-hover"
            >
              <LogOut className="size-3.5" aria-hidden />
              Sign Out
            </button>
          </div>
        </AnchoredPopover>
      )}
    </>
  )
}
