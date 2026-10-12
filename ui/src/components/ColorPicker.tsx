import { Check, X } from 'lucide-react'
import { cn } from '../ui'
import { BuilderPill } from './BuilderPill'

// A curated palette for builder pills: the CPTC logo inks plus a spread of
// distinct, legible hues. Picked so two builders are easy to tell apart at a
// glance and every swatch reads well behind white or black pill text.
const PRESETS = [
  '#4c63b0', // cptc blue-mid
  '#1d4e9f', // cptc blue
  '#301956', // cptc purple
  '#ed2330', // cptc red
  '#0891b2', // cyan
  '#0f766e', // teal
  '#15803d', // green
  '#b45309', // amber
  '#be185d', // pink
  '#7c3aed', // violet
  '#475569', // slate
  '#1e293b', // ink
]

// ColorPicker: preset swatches + a native custom-color input + a clear button,
// with a live pill preview. `value` is "" (none) or a "#rrggbb" hex. Used by the
// builder editor so an operator can give each builder a recognizable color.
export function ColorPicker({
  value,
  onChange,
  previewName,
  previewKind,
}: {
  value: string
  onChange: (hex: string) => void
  previewName?: string
  previewKind?: string
}) {
  const normalized = value.toLowerCase()
  const isPreset = PRESETS.map((p) => p.toLowerCase()).includes(normalized)
  const custom = value && !isPreset ? value : '#4c63b0'
  return (
    <div className="flex flex-col gap-3">
      <div className="flex flex-wrap items-center gap-2">
        {PRESETS.map((hex) => {
          const selected = normalized === hex.toLowerCase()
          return (
            <button
              key={hex}
              type="button"
              onClick={() => onChange(hex)}
              title={hex}
              aria-label={`Use ${hex}`}
              className={cn(
                'grid h-7 w-7 place-items-center rounded-full border transition-transform hover:scale-110',
                selected ? 'border-fg ring-2 ring-accent-soft' : 'border-border',
              )}
              style={{ backgroundColor: hex }}
            >
              {selected && <Check size={13} color="#fff" />}
            </button>
          )
        })}

        {/* Custom color via the browser's native picker. */}
        <label
          className={cn(
            'relative grid h-7 w-7 cursor-pointer place-items-center overflow-hidden rounded-full border text-[9px] font-semibold uppercase',
            value && !isPreset ? 'border-fg ring-2 ring-accent-soft text-fg-inverted' : 'border-border border-dashed text-fg-muted',
          )}
          title="Custom color"
          style={value && !isPreset ? { backgroundColor: value } : undefined}
        >
          {!(value && !isPreset) && '+'}
          <input
            type="color"
            value={custom}
            onChange={(e) => onChange(e.target.value)}
            className="absolute inset-0 cursor-pointer opacity-0"
          />
        </label>

        {value && (
          <button
            type="button"
            onClick={() => onChange('')}
            className="inline-flex items-center gap-1 rounded-token border border-border px-2 py-1 text-xs text-fg-muted hover:border-border-strong hover:text-fg"
          >
            <X size={12} /> None
          </button>
        )}
      </div>

      <div className="flex items-center gap-2 text-xs text-fg-subtle">
        Preview
        <BuilderPill name={previewName || 'builder-name'} color={value} kind={previewKind} />
      </div>
    </div>
  )
}
