'use client';

import { Paperclip, Upload, X } from 'lucide-react';
import { useRef, useState, type ReactNode } from 'react';
import { cn } from './cn.ts';
import { Button, Spinner } from './primitives.tsx';

/**
 * The native file input renders as unstyleable browser chrome, so it is kept visually hidden and
 * driven by a real button. Drag and drop is a bonus path, never the only one.
 */
export function FileDrop({
  accept,
  label = 'Choose File',
  hint = 'or drag one here',
  busy = false,
  disabled = false,
  selected = null,
  onFile,
  onRemove,
  className,
}: {
  accept?: string;
  label?: string;
  hint?: ReactNode;
  busy?: boolean;
  disabled?: boolean;
  selected?: { name: string; meta?: ReactNode; preview?: ReactNode } | null;
  onFile: (file: File) => void;
  onRemove?: () => void;
  className?: string;
}) {
  const inputRef = useRef<HTMLInputElement>(null);
  const [dragging, setDragging] = useState(false);
  const locked = disabled || busy;

  const take = (files: FileList | null) => {
    const file = files?.[0];
    if (file) onFile(file);
  };

  return (
    <div className={className}>
      <input
        ref={inputRef}
        type="file"
        accept={accept}
        disabled={locked}
        tabIndex={-1}
        className="sr-only"
        onChange={(event) => {
          take(event.target.files);
          // Clear so picking the same file twice in a row (say, after a failed upload) still fires.
          event.target.value = '';
        }}
      />

      {selected ? (
        <div className="flex items-center gap-2 rounded-token border border-border bg-surface-raised px-3 py-2">
          {selected.preview ?? <Paperclip className="size-4 shrink-0 text-fg-subtle" />}
          <div className="min-w-0 flex-1">
            <p className="truncate text-sm text-fg">{selected.name}</p>
            {selected.meta ? (
              <p className="truncate text-xs text-fg-subtle">{selected.meta}</p>
            ) : null}
          </div>
          {busy ? <Spinner className="shrink-0" /> : null}
          <Button
            type="button"
            size="sm"
            onClick={() => inputRef.current?.click()}
            disabled={locked}
          >
            Replace
          </Button>
          {onRemove ? (
            <Button
              type="button"
              variant="danger"
              size="icon"
              aria-label="Remove file"
              onClick={onRemove}
              disabled={locked}
            >
              <X className="size-4" />
            </Button>
          ) : null}
        </div>
      ) : (
        <div
          onDragOver={(event) => {
            event.preventDefault();
            if (!locked) setDragging(true);
          }}
          onDragLeave={() => setDragging(false)}
          onDrop={(event) => {
            event.preventDefault();
            setDragging(false);
            if (!locked) take(event.dataTransfer.files);
          }}
          className={cn(
            'flex flex-col items-center gap-2 rounded-token border border-dashed px-4 py-5 text-center transition-colors',
            dragging ? 'border-accent bg-accent-soft' : 'border-border bg-surface-sunken',
            locked && 'opacity-60',
          )}
        >
          <Upload className="size-5 text-fg-subtle" />
          <Button
            type="button"
            size="sm"
            onClick={() => inputRef.current?.click()}
            disabled={locked}
          >
            {busy ? <Spinner /> : null}
            {label}
          </Button>
          <p className="text-xs text-fg-subtle">{busy ? 'Uploading…' : hint}</p>
        </div>
      )}
    </div>
  );
}
