import type { CSSProperties } from 'react';

const FACETS: {
  top: string;
  left: string;
  size: number;
  color: string;
  rot: number;
  delay: string;
  dur: string;
}[] = [
  { top: '9%', left: '10%', size: 96, color: 'var(--cptc-red)', rot: -18, delay: '0s', dur: '9s' },
  {
    top: '16%',
    left: '82%',
    size: 56,
    color: 'var(--cptc-blue)',
    rot: 12,
    delay: '0.8s',
    dur: '11s',
  },
  {
    top: '58%',
    left: '6%',
    size: 68,
    color: 'var(--cptc-coral)',
    rot: 8,
    delay: '0.4s',
    dur: '10s',
  },
  {
    top: '74%',
    left: '72%',
    size: 108,
    color: 'var(--cptc-blue)',
    rot: -10,
    delay: '1.2s',
    dur: '12s',
  },
  {
    top: '38%',
    left: '46%',
    size: 46,
    color: 'var(--cptc-red)',
    rot: 20,
    delay: '1.6s',
    dur: '8s',
  },
  {
    top: '84%',
    left: '22%',
    size: 60,
    color: 'var(--cptc-coral)',
    rot: -14,
    delay: '0.6s',
    dur: '11s',
  },
];

export type AuthBackdropPalette = 'ui' | 'mark';

/**
 * `ui`: the screen-tuned UI red/blue (volunteers). `mark`: the logo's own inks (registration).
 * Coral and violet are shared. Either is on-brand; keep one per app.
 */
const PALETTES: Record<AuthBackdropPalette, { vars: Record<string, string>; blueWash: string; redWash: string }> = {
  ui: {
    vars: { '--cptc-red': '#e9050c', '--cptc-coral': '#fd9493', '--cptc-blue': '#004ae7', '--cptc-purple': '#301956' },
    blueWash: 'rgba(0,74,231,0.75)',
    redWash: 'rgba(233,5,12,0.45)',
  },
  mark: {
    vars: { '--cptc-red': '#ed2330', '--cptc-coral': '#fd9493', '--cptc-blue': '#1d4e9f', '--cptc-purple': '#301956' },
    blueWash: 'rgba(29,78,159,0.8)',
    redWash: 'rgba(237,35,48,0.45)',
  },
};

/**
 * Full-bleed animated CPTC backdrop for sign-in and public landing pages.
 * Purple wash, drifting brand glows, faceted hex outlines — CSS only, respects reduced motion.
 * Needs the `cptc-*` keyframes from @cptc/ui/tailwind.css (or styles.css).
 */
export function AuthBackdrop({ palette = 'ui' }: { palette?: AuthBackdropPalette } = {}) {
  const colors = PALETTES[palette];
  return (
    <div
      className="cptc-bg-anim pointer-events-none fixed inset-0 z-0 overflow-hidden bg-[#0b0618]"
      style={colors.vars as CSSProperties}
    >
      <div
        data-anim
        className="absolute inset-[-25%]"
        style={{
          backgroundImage: [
            'radial-gradient(ellipse 80% 60% at 18% -10%, rgba(48,25,86,1), transparent 62%)',
            `radial-gradient(ellipse 70% 60% at 105% 8%, ${colors.blueWash}, transparent 55%)`,
            `radial-gradient(ellipse 65% 55% at 50% 115%, ${colors.redWash}, transparent 60%)`,
          ].join(', '),
          animation: 'cptc-wash-shift 18s ease-in-out infinite',
        }}
      />

      <div
        data-anim
        className="absolute -left-40 top-[-12%] h-[520px] w-[520px] rounded-full blur-[95px]"
        style={{
          backgroundColor: 'color-mix(in srgb, var(--cptc-blue) 60%, transparent)',
          animation: 'cptc-drift-a 12s ease-in-out infinite',
        }}
      />
      <div
        data-anim
        className="absolute right-[-12%] top-1/4 h-[460px] w-[460px] rounded-full blur-[100px]"
        style={{
          backgroundColor: 'color-mix(in srgb, var(--cptc-red) 50%, transparent)',
          animation: 'cptc-drift-b 14s ease-in-out infinite',
        }}
      />
      <div
        data-anim
        className="absolute bottom-[-18%] left-1/3 h-[480px] w-[480px] rounded-full blur-[110px]"
        style={{
          backgroundColor: 'color-mix(in srgb, var(--cptc-coral) 40%, transparent)',
          animation: 'cptc-drift-c 16s ease-in-out infinite',
        }}
      />
      <div
        data-anim
        className="absolute left-[20%] top-[35%] h-[380px] w-[380px] rounded-full blur-[90px]"
        style={{
          backgroundColor: 'color-mix(in srgb, var(--cptc-purple) 55%, transparent)',
          animation: 'cptc-drift-d 13s ease-in-out infinite',
        }}
      />

      {FACETS.map((f, i) => (
        <svg
          key={i}
          data-anim
          viewBox="0 0 100 100"
          className="absolute"
          style={
            {
              top: f.top,
              left: f.left,
              width: f.size,
              height: f.size,
              '--rot': `${f.rot}deg`,
              animation: `cptc-facet-float ${f.dur} ease-in-out ${f.delay} infinite`,
            } as CSSProperties
          }
        >
          <polygon
            points="50,3 95,27 95,73 50,97 5,73 5,27"
            fill="none"
            stroke={f.color}
            strokeWidth="3"
          />
        </svg>
      ))}

      <div
        data-anim
        className="absolute inset-0 opacity-[0.08]"
        style={{
          backgroundImage:
            'linear-gradient(rgba(255,255,255,0.7) 1px, transparent 1px), linear-gradient(90deg, rgba(255,255,255,0.7) 1px, transparent 1px)',
          backgroundSize: '56px 56px',
          animation: 'cptc-grid-pan 28s linear infinite',
        }}
      />

      <div
        className="absolute inset-0"
        style={{
          backgroundImage:
            'linear-gradient(to top, #0b0618, transparent, color-mix(in srgb, #0b0618 50%, transparent))',
        }}
      />
    </div>
  );
}
