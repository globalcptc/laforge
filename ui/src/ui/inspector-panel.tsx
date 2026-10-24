'use client';

import { ChevronDown, X } from 'lucide-react';
import { useState, type ReactNode } from 'react';
import { cn } from './cn.ts';
import { Button } from './primitives.tsx';

export type InspectorPanelProps = {
  open: boolean;
  onClose: () => void;
  title: ReactNode;
  subtitle?: ReactNode;
  /** Sticky action row pinned to the bottom of the panel, e.g. Save / Discard. */
  footer?: ReactNode;
  children: ReactNode;
  className?: string;
  width?: string;
  /** Optional control(s) pinned in the header, to the left of the Close button (e.g. an expand/collapse toggle). */
  headerAction?: ReactNode;
  /**
   * Force the fixed overlay drawer at every width instead of docking flush on wide screens.
   * The docked variant is a static flex child and only lays out correctly inside a horizontal
   * inspector grid (ContentWithInspector); on a plain full-width page it otherwise flows to the
   * bottom of the content. Use this whenever the panel is a transient pop-open on a full-width page.
   */
  overlay?: boolean;
};

/**
 * One right-docked panel primitive, shared by the workflow node inspector, the volunteer
 * quick-view, and the report schema browser, so the three do not each invent their own. Flush and
 * docked on wide screens; an overlay drawer below 1280px, where a permanent third column would
 * squeeze the content area past usefulness.
 */
export function InspectorPanel({
  open,
  onClose,
  title,
  subtitle,
  footer,
  children,
  className,
  width,
  headerAction,
  overlay,
}: InspectorPanelProps) {
  if (!open) return null;

  return (
    <>
      <div aria-hidden onClick={onClose} className={cn('fixed inset-0 z-30 bg-fg/20', !overlay && 'xl:hidden')} />
      <aside
        className={cn(
          'inspector',
          overlay && 'fixed right-0 z-40 shadow-overlay [inset-block:var(--spacing-navbar)_0]',
          className,
        )}
        style={width ? { width } : undefined}
        aria-label={typeof title === 'string' ? title : 'Inspector'}
      >
        <header className="sticky top-0 z-10 flex items-start gap-2 border-b border-border bg-surface-raised px-4 py-3">
          <div className="min-w-0 flex-1">
            <h2 className="truncate text-sm font-semibold text-fg">{title}</h2>
            {subtitle ? <p className="mt-0.5 truncate text-xs text-fg-muted">{subtitle}</p> : null}
          </div>
          {headerAction ? <div className="flex shrink-0 items-center">{headerAction}</div> : null}
          <Button variant="ghost" size="icon" onClick={onClose} aria-label="Close panel">
            <X className="size-4" />
          </Button>
        </header>

        <div className="flex-1 px-4 py-3">{children}</div>

        {footer ? (
          <footer className="sticky bottom-0 border-t border-border bg-surface-raised px-4 py-3">
            {footer}
          </footer>
        ) : null}
      </aside>
    </>
  );
}

/** Labelled row for the read-only halves of an inspector. */
export function InspectorRow({ label, children }: { label: ReactNode; children: ReactNode }) {
  return (
    <div className="flex items-baseline justify-between gap-3 border-b border-border py-2 last:border-b-0">
      <span className="shrink-0 text-xs font-medium text-fg-muted">{label}</span>
      <span className="min-w-0 text-right text-sm text-fg">{children}</span>
    </div>
  );
}

/** Colour is structure here, not decoration: it tells one kind of section from the next at a glance. */
export type InspectorSectionTone = 'neutral' | 'config' | 'people' | 'routing' | 'delivery';

const sectionHeaders: Record<InspectorSectionTone, string> = {
  neutral: 'bg-surface text-fg-muted',
  config: 'bg-accent-soft text-accent-fg',
  people: 'bg-brand-soft text-brand-fg',
  routing: 'bg-warning-soft text-warning',
  delivery: 'bg-surface-sunken text-fg-muted',
};

/**
 * One titled card in an inspector. Sections stack, so a flat list of headings turns into a wall of
 * controls fast — the card edge plus a toned header is what makes "who approves" scannable next to
 * "delivery". Noisy sections can start collapsed rather than being cut.
 */
export function InspectorSection({
  title,
  description,
  tone = 'neutral',
  actions,
  collapsible,
  defaultCollapsed,
  children,
  className,
}: {
  title: ReactNode;
  description?: ReactNode;
  tone?: InspectorSectionTone;
  actions?: ReactNode;
  collapsible?: boolean;
  defaultCollapsed?: boolean;
  children: ReactNode;
  className?: string;
}) {
  const [collapsed, setCollapsed] = useState(Boolean(collapsible && defaultCollapsed));
  const open = !collapsible || !collapsed;

  const heading = (
    <>
      <span className="min-w-0 flex-1 truncate text-xs font-semibold uppercase tracking-wide">
        {title}
      </span>
      {collapsible ? (
        <ChevronDown
          className={cn('size-3.5 shrink-0 transition-transform', collapsed && '-rotate-90')}
        />
      ) : null}
    </>
  );

  return (
    <section
      className={cn(
        'mb-3 overflow-hidden rounded-token border border-border bg-surface-raised',
        className,
      )}
    >
      <div className={cn('flex items-center gap-2 border-b border-border', sectionHeaders[tone])}>
        {collapsible ? (
          <button
            type="button"
            aria-expanded={open}
            className="flex min-w-0 flex-1 items-center gap-2 px-3 py-2 text-left"
            onClick={() => setCollapsed((current) => !current)}
          >
            {heading}
          </button>
        ) : (
          <div className="flex min-w-0 flex-1 items-center gap-2 px-3 py-2">{heading}</div>
        )}
        {actions ? <div className="shrink-0 pr-2">{actions}</div> : null}
      </div>
      {open ? (
        <div className="px-3 py-2.5 [&>*:last-child]:mb-0">
          {description ? <p className="mb-2 text-xs text-fg-muted">{description}</p> : null}
          {children}
        </div>
      ) : null}
    </section>
  );
}
