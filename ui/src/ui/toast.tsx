'use client';

import { AlertCircle, CheckCircle2, Info, X } from 'lucide-react';
import { createContext, useCallback, useContext, useState, type ReactNode } from 'react';
import { cn } from './cn.ts';

// "Toasts are a courtesy; the message centre is the record". This is the
// courtesy half only -- ephemeral,
// client-side, dismiss-and-forget. The record itself is the real,
// persistent `event` table every build already writes to and
// GET /builds/{id}/events already reads from; the build's Logs tab
// (BuildLogs.tsx) is the UI for that record, and app-wide alerts live in
// GlobalAlerts.tsx -- not this file. A toast never invents its own
// backend state -- every toast this app fires is raised at a real call
// site right when a real mutation succeeds or fails, or when a real
// live SSE event arrives (useLiveEvents), never a client-side guess.
export type ToastTone = 'success' | 'danger' | 'info';

export interface ToastInput {
  title: string;
  description?: string;
  tone?: ToastTone;
  /** ms before auto-dismiss; 0 disables auto-dismiss (stays until closed). */
  duration?: number;
}

type ToastRecord = ToastInput & { id: number };

const ToastContext = createContext<((toast: ToastInput) => void) | null>(null);

export function useToast(): (toast: ToastInput) => void {
  const fn = useContext(ToastContext);
  if (!fn) throw new Error('useToast must be used within a ToastProvider');
  return fn;
}

const ICON: Record<ToastTone, typeof Info> = { success: CheckCircle2, danger: AlertCircle, info: Info };
const TONE_CLASS: Record<ToastTone, string> = {
  success: 'border-success/30 bg-success-soft text-success',
  danger: 'border-danger/30 bg-danger-soft text-danger',
  info: 'border-border bg-surface-raised text-fg',
};

let nextId = 1;

export function ToastProvider({ children }: { children: ReactNode }) {
  const [toasts, setToasts] = useState<ToastRecord[]>([]);

  const dismiss = useCallback((id: number) => {
    setToasts((cur) => cur.filter((t) => t.id !== id));
  }, []);

  const push = useCallback(
    (input: ToastInput) => {
      const id = nextId++;
      setToasts((cur) => [...cur, { id, tone: 'info', duration: 5000, ...input }]);
      const duration = input.duration ?? 5000;
      if (duration > 0) {
        setTimeout(() => dismiss(id), duration);
      }
    },
    [dismiss],
  );

  return (
    <ToastContext.Provider value={push}>
      {children}
      <div className="pointer-events-none fixed bottom-4 right-4 z-[200] flex w-80 flex-col gap-2">
        {toasts.map((t) => (
          <ToastCard key={t.id} toast={t} onDismiss={() => dismiss(t.id)} />
        ))}
      </div>
    </ToastContext.Provider>
  );
}

function ToastCard({ toast, onDismiss }: { toast: ToastRecord; onDismiss: () => void }) {
  const tone = toast.tone ?? 'info';
  const Icon = ICON[tone];

  return (
    <div
      role="status"
      className={cn(
        'pointer-events-auto flex items-start gap-2 rounded-token border px-3 py-2.5 text-sm shadow-panel',
        TONE_CLASS[tone],
      )}
    >
      <Icon size={16} className="mt-0.5 shrink-0" />
      <div className="min-w-0 flex-1">
        <div className="font-medium">{toast.title}</div>
        {toast.description && <div className="mt-0.5 text-xs opacity-80">{toast.description}</div>}
      </div>
      <button type="button" onClick={onDismiss} aria-label="Dismiss" className="shrink-0 opacity-60 hover:opacity-100">
        <X size={14} />
      </button>
    </div>
  );
}
