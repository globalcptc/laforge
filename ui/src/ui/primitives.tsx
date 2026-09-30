import { cva, type VariantProps } from 'class-variance-authority';
import type {
  ButtonHTMLAttributes,
  ComponentProps,
  HTMLAttributes,
  ReactNode,
  SelectHTMLAttributes,
} from 'react';
import { cn } from './cn.ts';

const buttonStyles = cva(
  'inline-flex items-center justify-center gap-1.5 rounded-token font-medium whitespace-nowrap transition-colors disabled:pointer-events-none disabled:opacity-50 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-accent',
  {
    variants: {
      variant: {
        primary: 'bg-accent text-fg-inverted hover:bg-accent-hover',
        secondary:
          'bg-surface-raised text-fg border border-border hover:bg-surface-hover hover:border-border-strong',
        ghost: 'text-fg-muted hover:bg-surface-hover hover:text-fg',
        danger: 'bg-danger text-fg-inverted hover:brightness-95',
        brand: 'bg-brand text-fg-inverted hover:bg-brand-hover',
        // CPTC coral, a light fill -- fixed dark ink for text, not white.
        coral: 'bg-cptc-coral text-cptc-coral-fg hover:brightness-95',
        link: 'text-accent hover:text-accent-hover underline-offset-4 hover:underline',
      },
      size: {
        sm: 'h-7 px-2.5 text-xs',
        md: 'h-9 px-3.5 text-sm',
        lg: 'h-10 px-4 text-sm',
        icon: 'size-8 p-0',
      },
    },
    compoundVariants: [
      // Icon rows stay quiet: tint the glyph, soft fill on hover — not a solid chip per row.
      {
        variant: 'danger',
        size: 'icon',
        class: 'bg-transparent text-danger hover:bg-danger-soft hover:text-danger',
      },
      {
        variant: 'brand',
        size: 'icon',
        class: 'bg-transparent text-brand hover:bg-brand-soft hover:text-brand',
      },
    ],
    defaultVariants: { variant: 'secondary', size: 'md' },
  },
);

export type ButtonProps = ButtonHTMLAttributes<HTMLButtonElement> &
  VariantProps<typeof buttonStyles>;

export function Button({ className, variant, size, ...props }: ButtonProps) {
  return <button className={cn(buttonStyles({ variant, size }), className)} {...props} />;
}

export function buttonClass(options: VariantProps<typeof buttonStyles> = {}): string {
  return buttonStyles(options);
}

export function Card({ className, ...props }: HTMLAttributes<HTMLDivElement>) {
  return <div className={cn('card', className)} {...props} />;
}

export function CardHeader({
  title,
  description,
  actions,
  className,
}: {
  title: ReactNode;
  description?: ReactNode;
  actions?: ReactNode;
  className?: string;
}) {
  return (
    <div className={cn('card-header', className)}>
      <div className="min-w-0">
        <div className="card-title truncate">{title}</div>
        {description ? <p className="mt-0.5 text-xs text-fg-muted">{description}</p> : null}
      </div>
      {actions ? <div className="ml-auto flex shrink-0 items-center gap-2">{actions}</div> : null}
    </div>
  );
}

export function CardBody({ className, ...props }: HTMLAttributes<HTMLDivElement>) {
  return <div className={cn('p-4', className)} {...props} />;
}

const badgeStyles = cva(
  'inline-flex items-center gap-1 rounded-token-sm border border-transparent px-1.5 py-0.5 text-xs/none font-medium',
  {
    variants: {
      tone: {
        neutral: 'bg-surface-sunken text-fg-muted',
        accent: 'bg-accent-soft text-accent-fg',
        brand: 'bg-brand-soft text-brand-fg',
        success: 'bg-success-soft text-success',
        warning: 'bg-warning-soft text-warning',
        danger: 'bg-danger-soft text-danger',
        info: 'bg-info-soft text-info',
        orange: 'bg-orange-soft text-orange',
        ink: 'bg-ink-soft text-ink',
        indigo: 'bg-indigo-soft text-indigo',
        sky: 'bg-sky-soft text-sky',
        purple: 'bg-purple-soft text-purple',
        pink: 'bg-pink-soft text-pink',
      },
    },
    defaultVariants: { tone: 'neutral' },
  },
);

export type BadgeProps = HTMLAttributes<HTMLSpanElement> & VariantProps<typeof badgeStyles>;

export function Badge({ className, tone, ...props }: BadgeProps) {
  return <span className={cn(badgeStyles({ tone }), className)} {...props} />;
}

/**
 * Chips on the Volunteers table double as filter controls — clicking one adds it to the filter
 * bar — so the interactive and static forms have to look identical.
 */
export function Chip({
  className,
  tone,
  onRemove,
  children,
  ...props
}: BadgeProps & { onRemove?: () => void }) {
  const interactive = Boolean(props.onClick);
  return (
    <span
      className={cn(
        badgeStyles({ tone }),
        interactive && 'cursor-pointer hover:border-border-strong',
        className,
      )}
      {...props}
    >
      {children}
      {onRemove ? (
        <button
          type="button"
          aria-label="Remove"
          className="ml-0.5 text-fg-subtle hover:text-fg"
          onClick={(event) => {
            event.stopPropagation();
            onRemove();
          }}
        >
          &times;
        </button>
      ) : null}
    </span>
  );
}

const fieldBase =
  'w-full rounded-token border border-border bg-surface-raised px-2.5 text-sm text-fg placeholder:text-fg-subtle focus-visible:border-accent focus-visible:outline-2 focus-visible:outline-offset-0 focus-visible:outline-accent-soft disabled:bg-surface-sunken disabled:text-fg-muted';

export function Input({ className, ...props }: ComponentProps<'input'>) {
  return <input className={cn(fieldBase, 'h-9', className)} {...props} />;
}

export function Textarea({ className, ...props }: ComponentProps<'textarea'>) {
  return <textarea className={cn(fieldBase, 'min-h-20 py-2', className)} {...props} />;
}

export function Select({ className, ...props }: SelectHTMLAttributes<HTMLSelectElement>) {
  return <select className={cn(fieldBase, 'h-9 pr-8', className)} {...props} />;
}

export function Label({ className, ...props }: HTMLAttributes<HTMLLabelElement>) {
  return (
    <label className={cn('mb-1 block text-xs font-medium text-fg-muted', className)} {...props} />
  );
}

export function Field({
  label,
  hint,
  error,
  children,
  className,
}: {
  label: ReactNode;
  hint?: ReactNode;
  error?: ReactNode;
  children: ReactNode;
  className?: string;
}) {
  return (
    <div className={cn('mb-3', className)}>
      <Label>{label}</Label>
      {children}
      {hint && !error ? <p className="mt-1 text-xs text-fg-subtle">{hint}</p> : null}
      {error ? <p className="mt-1 text-xs text-danger">{error}</p> : null}
    </div>
  );
}

export function EmptyState({
  title,
  description,
  action,
}: {
  title: ReactNode;
  description?: ReactNode;
  action?: ReactNode;
}) {
  return (
    <div className="flex flex-col items-center justify-center gap-2 px-6 py-14 text-center">
      <p className="text-sm font-medium text-fg">{title}</p>
      {description ? <p className="max-w-md text-sm text-fg-muted">{description}</p> : null}
      {action ? <div className="mt-2">{action}</div> : null}
    </div>
  );
}

export function Spinner({ className }: { className?: string }) {
  return (
    <span
      role="status"
      aria-label="Loading"
      className={cn(
        'inline-block size-4 animate-spin rounded-full border-2 border-border border-t-accent',
        className,
      )}
    />
  );
}
