'use client';

import { Check, Monitor, Moon, Sun } from 'lucide-react';
import { useEffect, useRef, useState } from 'react';
import { AnchoredPopover } from './anchored-popover.tsx';
import { cn } from './cn.ts';
import { buttonClass } from './primitives.tsx';
import { DARK_QUERY, THEME_STORAGE_KEY as STORAGE_KEY } from './theme-script.ts';

export type ThemePreference = 'system' | 'light' | 'dark';

function readPreference(): ThemePreference {
  try {
    const stored = localStorage.getItem(STORAGE_KEY);
    return stored === 'light' || stored === 'dark' ? stored : 'system';
  } catch {
    return 'system';
  }
}

function apply(preference: ThemePreference) {
  const dark =
    preference === 'dark' || (preference === 'system' && window.matchMedia(DARK_QUERY).matches);
  const root = document.documentElement;
  if (dark) root.dataset.theme = 'dark';
  else delete root.dataset.theme;
}

const LABEL: Record<ThemePreference, string> = {
  system: 'System',
  light: 'Light',
  dark: 'Dark',
};

const ICON: Record<ThemePreference, typeof Monitor> = {
  system: Monitor,
  light: Sun,
  dark: Moon,
};

const OPTIONS: ThemePreference[] = ['system', 'light', 'dark'];

/**
 * Navbar icon button opening a popup with all three options explicit
 * (System/Light/Dark) -- direct product feedback: cycling through them
 * one click at a time made it hard to tell what you'd land on, or to get
 * back to "system" once past it. The icon shown on the trigger still
 * reflects the current preference, not the resolved theme, so "system"
 * stays visually distinguishable from an explicit light/dark choice.
 */
export function ThemeToggle({ className }: { className?: string }) {
  // Rendered as 'system' on the server; the real preference is only readable after mount.
  const [preference, setPreference] = useState<ThemePreference>('system');
  const [open, setOpen] = useState(false);
  const anchorRef = useRef<HTMLButtonElement>(null);
  const panelRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    setPreference(readPreference());
  }, []);

  // While following the OS, track it live — someone flipping macOS to dark mid-session gets it.
  useEffect(() => {
    if (preference !== 'system') return;
    const media = window.matchMedia(DARK_QUERY);
    const onChange = () => apply('system');
    media.addEventListener('change', onChange);
    return () => media.removeEventListener('change', onChange);
  }, [preference]);

  useEffect(() => {
    if (!open) return;
    const onPointerDown = (e: PointerEvent) => {
      const target = e.target as Node;
      if (anchorRef.current?.contains(target) || panelRef.current?.contains(target)) return;
      setOpen(false);
    };
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') setOpen(false);
    };
    window.addEventListener('pointerdown', onPointerDown);
    window.addEventListener('keydown', onKey);
    return () => {
      window.removeEventListener('pointerdown', onPointerDown);
      window.removeEventListener('keydown', onKey);
    };
  }, [open]);

  function choose(next: ThemePreference) {
    try {
      if (next === 'system') localStorage.removeItem(STORAGE_KEY);
      else localStorage.setItem(STORAGE_KEY, next);
    } catch {
      // Storage blocked (private mode): the choice still applies for this page view.
    }
    apply(next);
    setPreference(next);
    setOpen(false);
  }

  const Icon = ICON[preference];

  return (
    <>
      <button
        ref={anchorRef}
        type="button"
        className={cn(buttonClass({ variant: 'ghost', size: 'icon' }), className)}
        aria-label={`Theme: ${LABEL[preference]}. Choose a theme.`}
        aria-haspopup="menu"
        aria-expanded={open}
        title={`Theme: ${LABEL[preference]}`}
        onClick={() => setOpen((v) => !v)}
      >
        <Icon className="size-4" aria-hidden />
      </button>
      {open && (
        <AnchoredPopover anchorRef={anchorRef} panelRef={panelRef} minWidth={160} maxHeight={200} role="menu">
          <div className="p-1">
            {OPTIONS.map((opt) => {
              const OptIcon = ICON[opt];
              return (
                <button
                  key={opt}
                  type="button"
                  role="menuitemradio"
                  aria-checked={preference === opt}
                  onClick={() => choose(opt)}
                  className={cn(
                    'flex w-full items-center gap-2 rounded-token-sm px-2.5 py-1.5 text-left text-sm text-fg hover:bg-surface-hover',
                    preference === opt && 'text-accent',
                  )}
                >
                  <OptIcon className="size-3.5" aria-hidden />
                  <span className="flex-1">{LABEL[opt]}</span>
                  {preference === opt && <Check className="size-3.5" aria-hidden />}
                </button>
              )
            })}
          </div>
        </AnchoredPopover>
      )}
    </>
  );
}
