import { useState } from 'react'
import { Cpu, MemoryStick, Plus, Trash2 } from 'lucide-react'
import { Button, Input, Select } from '../../ui'
import { CPU_OPTIONS, MEMORY_OPTIONS, defaultSize, type SizeDraft } from './model'

export function StepSizes({ sizes, onChange }: { sizes: SizeDraft[]; onChange: (sizes: SizeDraft[]) => void }) {
  const update = (i: number, patch: Partial<SizeDraft>) => onChange(sizes.map((s, j) => (j === i ? { ...s, ...patch } : s)))
  const [newName, setNewName] = useState('')

  function add() {
    const name = newName.trim()
    if (!name || sizes.some((s) => s.name === name)) return
    onChange([...sizes, { name, ...defaultSize(name) }])
    setNewName('')
  }

  return (
    <div className="flex flex-col gap-4">
      <p className="text-sm text-fg-muted">
        Content asks for a machine by size name. Set the CPUs and memory each size gets on this builder.
      </p>
      <div className="card divide-y divide-border">
        <div className="hidden grid-cols-[1fr_8rem_8rem_2rem] gap-3 px-3 py-2 text-xs font-medium uppercase tracking-wide text-fg-muted sm:grid">
          <span>Size</span>
          <span className="flex items-center gap-1">
            <Cpu size={11} /> CPUs
          </span>
          <span className="flex items-center gap-1">
            <MemoryStick size={11} /> Memory
          </span>
          <span />
        </div>
        {sizes.map((s, i) => (
          <div key={s.name} className="grid grid-cols-2 items-center gap-3 px-3 py-2 sm:grid-cols-[1fr_8rem_8rem_2rem]">
            <div className="col-span-2 font-mono text-xs text-fg sm:col-span-1">{s.name}</div>
            <Select value={s.cpu} onChange={(e) => update(i, { cpu: e.target.value })} className="h-8 text-xs" aria-label={`${s.name} CPUs`}>
              {[...new Set([...CPU_OPTIONS, s.cpu])].map((c) => (
                <option key={c} value={c}>
                  {c} CPU{c === '1' ? '' : 's'}
                </option>
              ))}
            </Select>
            <Select value={s.memory} onChange={(e) => update(i, { memory: e.target.value })} className="h-8 text-xs" aria-label={`${s.name} memory`}>
              {[...new Set([...MEMORY_OPTIONS, s.memory])].map((m) => (
                <option key={m} value={m}>
                  {m.replace('MiB', ' MB').replace('GiB', ' GB')}
                </option>
              ))}
            </Select>
            <Button variant="ghost" size="icon" onClick={() => onChange(sizes.filter((_, j) => j !== i))} title="Remove" className="text-fg-muted hover:bg-danger-soft hover:text-danger">
              <Trash2 size={12} />
            </Button>
          </div>
        ))}
      </div>
      <div className="flex items-center gap-2">
        <Input
          value={newName}
          onChange={(e) => setNewName(e.target.value)}
          onKeyDown={(e) => e.key === 'Enter' && add()}
          placeholder="Another size name, e.g. xx-large"
          className="h-8 max-w-xs text-xs"
        />
        <Button variant="secondary" size="sm" onClick={add} disabled={!newName.trim()}>
          <Plus size={12} /> Add Size
        </Button>
      </div>
    </div>
  )
}
