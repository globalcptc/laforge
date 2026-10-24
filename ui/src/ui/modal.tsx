'use client';

import { X } from 'lucide-react';
import { useEffect, useRef, type ReactNode } from 'react';
import { cn } from './cn.ts';
import { Button } from './primitives.tsx';

/**
 * Built on native `<dialog>` so focus trapping, the backdrop, escape handling, and inert background
 * content come from the platform rather than a hand-rolled implementation that gets one of them
 * subtly wrong.
 */
export function Modal({
  open,
  onClose,
  title,
  description,
  footer,
  children,
  className,
}: {
  open: boolean;
  onClose: () => void;
  title: ReactNode;
  description?: ReactNode;
  footer?: ReactNode;
  children: ReactNode;
  className?: string;
}) {
  const ref = useRef<HTMLDialogElement>(null);
  // A dialog.close() WE call (unmount, or React StrictMode's mount→cleanup→
  // remount in dev) fires the native 'close' event just like an Escape or a
  // backdrop click does. Without this guard that programmatic close calls
  // onClose, which nulls the parent state that renders the modal -- so under
  // StrictMode the modal opens and is torn straight back down ("flashes for
  // half a second"). This flag marks a close as ours so onClose fires only
  // for real user dismissals.
  const programmaticClose = useRef(false);

  useEffect(() => {
    const dialog = ref.current;
    if (!dialog) return;
    if (!dialog.open) dialog.showModal();
    return () => {
      if (dialog.open) {
        programmaticClose.current = true;
        dialog.close();
      }
    };
  }, [open]);

  function handleNativeClose() {
    if (programmaticClose.current) {
      programmaticClose.current = false;
      return;
    }
    onClose();
  }

  // Stay out of the DOM when closed. A closed <dialog> is display:none in the UA stylesheet, but
  // any preflight that resets `dialog { display: block }` would leave an invisible box sitting on
  // top of the page and eat every click — which is exactly "buttons dead, links in the sidebar
  // still work."
  if (!open) return null;

  return (
    <dialog
      ref={ref}
      onClose={handleNativeClose}
      onClick={(event) => {
        // A click that lands on the dialog element itself is a click on the backdrop, because the
        // padded content below covers the rest of it.
        if (event.target === ref.current) onClose();
      }}
      className={cn(
        'm-auto w-[min(92vw,34rem)] rounded-token border border-border bg-surface-raised p-0 text-fg shadow-xl backdrop:bg-fg/30',
        className,
      )}
    >
      <header className="flex items-start gap-2 border-b border-border px-4 py-3">
        <div className="min-w-0 flex-1">
          <h2 className="text-sm font-semibold">{title}</h2>
          {description ? <p className="mt-0.5 text-xs text-fg-muted">{description}</p> : null}
        </div>
        <Button type="button" variant="ghost" size="icon" onClick={onClose} aria-label="Close">
          <X className="size-4" />
        </Button>
      </header>

      <div className="max-h-[70vh] overflow-auto px-4 py-3">{children}</div>

      {footer ? (
        <footer className="flex items-center justify-end gap-2 border-t border-border px-4 py-3">
          {footer}
        </footer>
      ) : null}
    </dialog>
  );
}
