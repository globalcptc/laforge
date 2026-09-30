'use client';

import {
  useEffect,
  useLayoutEffect,
  useState,
  type HTMLAttributes,
  type ReactNode,
  type RefObject,
} from 'react';
import { createPortal } from 'react-dom';
import { cn } from './cn.ts';

/** Breathing room from the viewport edge, and between the anchor and the panel. */
const EDGE = 8;
const GAP = 4;
/** Below this much room underneath, opening downwards is not worth it and the panel flips up. */
const MIN_ROOM = 140;

type Position = { left: number; width: number; maxHeight: number } & (
  { top: number; bottom?: never } | { bottom: number; top?: never }
);

function measure(anchor: HTMLElement, minWidth: number, maxHeight: number): Position {
  const rect = anchor.getBoundingClientRect();
  const width = Math.max(rect.width, minWidth);
  const left = Math.min(
    Math.max(rect.left, EDGE),
    Math.max(window.innerWidth - width - EDGE, EDGE),
  );

  const below = window.innerHeight - rect.bottom - GAP - EDGE;
  const above = rect.top - GAP - EDGE;
  const flip = below < Math.min(maxHeight, MIN_ROOM) && above > below;

  return flip
    ? {
        left,
        width,
        maxHeight: Math.min(maxHeight, Math.max(above, MIN_ROOM)),
        bottom: window.innerHeight - rect.top + GAP,
      }
    : {
        left,
        width,
        maxHeight: Math.min(maxHeight, Math.max(below, MIN_ROOM)),
        top: rect.bottom + GAP,
      };
}

/**
 * A panel pinned to an input but rendered outside of it, so no ancestor can clip it or paint over
 * it. Both happen constantly here: `InspectorSection` is `overflow-hidden`, the modal body scrolls,
 * and a `<dialog>` lives in the browser's top layer where no z-index from the page can reach.
 *
 * That last one is why the portal targets the nearest open `<dialog>` rather than always the body:
 * a body portal would render *behind* a modal that had promoted itself to the top layer.
 */
export function AnchoredPopover({
  anchorRef,
  panelRef,
  minWidth = 240,
  maxHeight = 256,
  className,
  children,
  ...rest
}: {
  anchorRef: RefObject<HTMLElement | null>;
  /** Attached to the panel so callers can tell a click inside it from a click away. */
  panelRef?: RefObject<HTMLDivElement | null>;
  minWidth?: number;
  maxHeight?: number;
  className?: string;
  children: ReactNode;
} & Omit<HTMLAttributes<HTMLDivElement>, 'children' | 'className' | 'style'>) {
  const [position, setPosition] = useState<Position | null>(null);
  // A ref is not reactive, so re-read it once on mount for the case where the anchor attaches in the
  // same commit that opens the panel.
  const [anchor, setAnchor] = useState<HTMLElement | null>(anchorRef.current);
  useEffect(() => {
    setAnchor(anchorRef.current);
  }, [anchorRef]);

  useLayoutEffect(() => {
    if (!anchor) return;

    const reposition = () => setPosition(measure(anchor, minWidth, maxHeight));
    reposition();

    // Capture, so scrolling any container between the anchor and the root keeps the panel attached
    // rather than leaving it stranded mid-page.
    window.addEventListener('scroll', reposition, true);
    window.addEventListener('resize', reposition);
    return () => {
      window.removeEventListener('scroll', reposition, true);
      window.removeEventListener('resize', reposition);
    };
  }, [anchor, minWidth, maxHeight]);

  // The portal target comes from the anchor's own position in the tree, so a picker inside a modal
  // stays inside that modal's top layer. A non-null anchor means we are on the client.
  const target = anchor ? (anchor.closest('dialog') ?? document.body) : null;

  if (!target || !position) return null;

  return createPortal(
    <div
      ref={panelRef}
      className={cn(
        'fixed z-[100] overflow-y-auto overscroll-contain rounded-token border border-border bg-surface-raised shadow-md',
        className,
      )}
      style={{
        left: position.left,
        width: position.width,
        maxHeight: position.maxHeight,
        ...(position.top === undefined ? { bottom: position.bottom } : { top: position.top }),
      }}
      {...rest}
    >
      {children}
    </div>,
    target,
  );
}
