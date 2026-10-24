import type { HTMLAttributes, ReactNode, TdHTMLAttributes, ThHTMLAttributes } from 'react';
import { cn } from './cn.ts';

/**
 * Sticky-header table. The bounded-height shell means the header sticks with no calc() height
 * plumbing — the scroll container is the wrapper, not the window.
 *
 * `contain` (default) is for pages where this wrapper *is* the scrollport. Turn it off on
 * document-scrolled pages: `overflow-auto` + `overscroll-contain` eats the wheel even when the
 * table does not overflow, so hovering the table freezes the page.
 */
export function TableScroller({
  contain = true,
  className,
  ...props
}: HTMLAttributes<HTMLDivElement> & { contain?: boolean }) {
  return (
    <div
      className={cn(
        contain ? 'min-h-0 flex-1 overflow-auto overscroll-contain' : 'overflow-x-auto',
        className,
      )}
      {...props}
    />
  );
}

export function Table({
  density = 'default',
  className,
  ...props
}: HTMLAttributes<HTMLTableElement> & { density?: 'default' | 'compact' }) {
  return <table className={cn('data-table', className)} data-density={density} {...props} />;
}

export function Th({ className, ...props }: ThHTMLAttributes<HTMLTableCellElement>) {
  return <th scope="col" className={className} {...props} />;
}

export function Td({ className, ...props }: TdHTMLAttributes<HTMLTableCellElement>) {
  return <td className={className} {...props} />;
}

export function NestedRow({ colSpan, children }: { colSpan: number; children: ReactNode }) {
  return (
    <tr data-nested="true">
      <td colSpan={colSpan} className="px-0 py-0">
        <div className="border-l-2 border-accent/40 px-4 py-3">{children}</div>
      </td>
    </tr>
  );
}
