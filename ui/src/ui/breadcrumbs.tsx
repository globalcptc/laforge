import { Link } from '@tanstack/react-router';
import { ChevronRight } from 'lucide-react';
import type { ReactNode } from 'react';
import { cn } from './cn.ts';

export type Crumb = { label: string; href?: string };

/**
 * Layout 2 hides breadcrumbs by default. This app nests too deeply for that to work — workflow to
 * version to run to step, volunteer to season to team — so they render inside the page header and
 * double as the back path.
 */
export function Breadcrumbs({ items, className }: { items: Crumb[]; className?: string }) {
  if (items.length === 0) return null;

  return (
    <nav aria-label="Breadcrumb" className={cn('mb-1 flex items-center gap-1 text-xs', className)}>
      {items.map((item, index) => {
        const last = index === items.length - 1;
        return (
          <span key={`${item.label}-${index}`} className="flex items-center gap-1">
            {item.href && !last ? (
              <Link to={item.href} className="text-fg-muted hover:text-accent hover:underline">
                {item.label}
              </Link>
            ) : (
              <span className={last ? 'font-medium text-fg' : 'text-fg-muted'}>{item.label}</span>
            )}
            {last ? null : <ChevronRight className="size-3 text-fg-subtle" />}
          </span>
        );
      })}
    </nav>
  );
}

export function PageHeader({
  title,
  description,
  breadcrumbs,
  actions,
}: {
  title: ReactNode;
  description?: ReactNode;
  breadcrumbs?: Crumb[];
  actions?: ReactNode;
}) {
  return (
    <header className="page-header">
      {breadcrumbs ? <Breadcrumbs items={breadcrumbs} /> : null}
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="min-w-0">
          <h1 className="text-xl font-semibold tracking-tight text-fg">{title}</h1>
          {description ? <p className="mt-1 text-sm text-fg-muted">{description}</p> : null}
        </div>
        {actions ? <div className="flex flex-wrap items-center gap-2 sm:shrink-0">{actions}</div> : null}
      </div>
    </header>
  );
}
