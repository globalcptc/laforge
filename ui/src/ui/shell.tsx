'use client';

import { Link } from '@tanstack/react-router';
import { Menu, PanelLeftClose, PanelLeftOpen } from 'lucide-react';
import { useEffect, useState, type ReactNode } from 'react';
import { cn } from './cn.ts';
import { Button } from './primitives.tsx';

export type NavItem = {
  href: string;
  label: string;
  icon?: ReactNode;
  badge?: ReactNode;
};

export type NavSection = {
  title?: string;
  items: NavItem[];
};

/**
 * The fixed shell. Regions scroll independently, which is what the React Flow canvas and the run
 * timeline need. Below 1024px the sidebar becomes an overlay and `layout-static` releases the fixed
 * height, because fixed panes fight mobile browser chrome.
 */
export function AppShell({
  navbar,
  sidebarTop,
  sections,
  activePath,
  children,
}: {
  navbar: ReactNode;
  /** Rendered above the nav sections, e.g. a repository switcher -- opts in per app, costs nothing when omitted. */
  sidebarTop?: ReactNode;
  sections: NavSection[];
  activePath: string;
  children: ReactNode;
}) {
  const [collapsed, setCollapsed] = useState(false);
  const [mobileOpen, setMobileOpen] = useState(false);

  useEffect(() => {
    const root = document.documentElement;
    const apply = () => {
      const fixed = window.matchMedia('(min-width: 1024px)').matches;
      root.classList.toggle('layout-fixed', fixed);
      root.classList.toggle('layout-static', !fixed);
    };
    apply();
    window.addEventListener('resize', apply);
    return () => {
      window.removeEventListener('resize', apply);
      root.classList.remove('layout-fixed', 'layout-static');
    };
  }, []);

  useEffect(() => setMobileOpen(false), [activePath]);

  /* While the drawer is open the page behind it must not scroll, and Escape closes it. */
  useEffect(() => {
    if (!mobileOpen) return;
    const onKey = (event: KeyboardEvent) => {
      if (event.key === 'Escape') setMobileOpen(false);
    };
    window.addEventListener('keydown', onKey);
    document.documentElement.classList.add('nav-drawer-open');
    return () => {
      window.removeEventListener('keydown', onKey);
      document.documentElement.classList.remove('nav-drawer-open');
    };
  }, [mobileOpen]);

  const allHrefs = sections.flatMap((section) => section.items.map((item) => item.href));

  return (
    <div className="app-shell">
      <header className="app-navbar">
        <Button
          variant="ghost"
          size="icon"
          className="lg:hidden"
          aria-label="Toggle navigation"
          aria-expanded={mobileOpen}
          onClick={() => setMobileOpen((open) => !open)}
        >
          <Menu className="size-4" />
        </Button>
        <Button
          variant="ghost"
          size="icon"
          className="hidden lg:inline-flex"
          aria-label={collapsed ? 'Expand sidebar' : 'Collapse sidebar'}
          onClick={() => setCollapsed((value) => !value)}
        >
          {collapsed ? <PanelLeftOpen className="size-4" /> : <PanelLeftClose className="size-4" />}
        </Button>
        {navbar}
      </header>

      <div className="app-body">
        {mobileOpen ? (
          <button
            type="button"
            className="app-scrim lg:hidden"
            aria-label="Close navigation"
            onClick={() => setMobileOpen(false)}
          />
        ) : null}
        <nav className="app-sidebar" data-collapsed={collapsed} data-open={mobileOpen}>
          {sidebarTop && !collapsed ? <div className="border-b border-border p-2">{sidebarTop}</div> : null}
          <div className="py-2">
            {sections.map((section, index) => (
              <div key={section.title ?? index}>
                {section.title && !collapsed ? (
                  <div className="nav-section">{section.title}</div>
                ) : null}
                {section.items.map((item) => (
                  <Link
                    key={item.href}
                    to={item.href}
                    className="nav-link"
                    data-active={isActive(activePath, item.href, allHrefs)}
                    title={collapsed ? item.label : undefined}
                  >
                    {item.icon ? <span className="shrink-0">{item.icon}</span> : null}
                    {collapsed ? null : <span className="flex-1 truncate">{item.label}</span>}
                    {!collapsed && item.badge ? item.badge : null}
                  </Link>
                ))}
              </div>
            ))}
          </div>
        </nav>

        <main className="app-content">{children}</main>
      </div>
    </div>
  );
}

/** Longest matching nav href wins, so `/admin` does not stay lit on `/admin/seasons`. */
function isActive(current: string, href: string, hrefs: string[]): boolean {
  const matches = hrefs.filter(
    (candidate) =>
      current === candidate || (candidate !== '/' && current.startsWith(`${candidate}/`)),
  );
  if (matches.length === 0) return false;
  const best = matches.reduce((a, b) => (a.length >= b.length ? a : b));
  return best === href;
}

/**
 * Content column plus an optional docked inspector. Kept separate from AppShell so a page can opt
 * into the third column without every page paying for it.
 */
export function ContentWithInspector({
  children,
  inspector,
  className,
}: {
  children: ReactNode;
  inspector?: ReactNode;
  className?: string;
}) {
  return (
    <div className={cn('flex min-h-0 flex-1', className)}>
      <div className="flex min-w-0 flex-1 flex-col overflow-y-auto">{children}</div>
      {inspector}
    </div>
  );
}
