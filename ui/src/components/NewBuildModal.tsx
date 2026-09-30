import { useEffect, useMemo, useRef, useState, type ReactNode } from 'react'
import { Link } from '@tanstack/react-router'
import { Check, ChevronDown, ChevronRight, ChevronsUpDown, FileCode2, FileText, Folder, FolderOpen, GitBranch, Search, ServerCog } from 'lucide-react'
import { useBranches, useBuilders, useEnvironmentFiles, useMe } from '../api/hooks'
import { ApiError } from '../api/client'
import type { EnvironmentFileInfo } from '../api/types'
import { AnchoredPopover, Badge, Button, Input, Modal, Spinner, cn } from '../ui'
import { KIND_LABEL, type Kind } from './builder-wizard/model'

type CreateRequest = { branch: string; environment_path: string; builder_config_name: string }

// New Build: three picks, each from real data -- a branch from GitHub, an
// environment file on that branch (quick picks for the ones at the root, or
// browse the branch's files), and a builder.
export function NewBuildModal({
  repoId,
  open,
  onClose,
  onCreate,
}: {
  repoId: string
  open: boolean
  onClose: () => void
  onCreate: (req: CreateRequest) => Promise<void>
}) {
  const { data: branchList, error: branchError, isLoading: branchesLoading } = useBranches(repoId)
  const [chosenBranch, setChosenBranch] = useState('')
  // Until someone picks, the repository's real default branch.
  const branch = chosenBranch || branchList?.default_branch || ''
  const { data: envFiles, isLoading: envLoading, error: envError } = useEnvironmentFiles(repoId, branch)
  const { data: builders, isLoading: buildersLoading } = useBuilders()
  const { data: me } = useMe()
  const [chosenPath, setEnvironmentPath] = useState('')
  const [chosenBuilder, setBuilderName] = useState('')
  const [browsing, setBrowsing] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)

  // Only one environment file, or one builder: it's the pick.
  const environmentPath = chosenPath || (envFiles?.files.length === 1 ? envFiles.files[0].path : '')
  const builderName = chosenBuilder || (builders?.length === 1 ? builders[0].name : '')

  function pickBranch(b: string) {
    if (b === branch) return
    setChosenBranch(b)
    setEnvironmentPath('')
    setBrowsing(false)
  }

  async function submit() {
    setError(null)
    setSubmitting(true)
    try {
      await onCreate({ branch, environment_path: environmentPath, builder_config_name: builderName })
      setEnvironmentPath('')
      setBuilderName('')
    } catch (e) {
      setError(e instanceof ApiError ? e.message : 'Creating the build failed')
    } finally {
      setSubmitting(false)
    }
  }

  const envByPath = new Map((envFiles?.files ?? []).map((f) => [f.path, f]))
  const rootEnvs = (envFiles?.files ?? []).filter((f) => !f.path.includes('/'))
  const pickedNested = environmentPath && !rootEnvs.some((f) => f.path === environmentPath) ? envByPath.get(environmentPath) : undefined
  const ready = !!branch && !!environmentPath && !!builderName

  return (
    <Modal
      open={open}
      onClose={onClose}
      title="New Build"
      description="Pick a branch, an environment file on it, and a builder. Every CI-passing push to the branch then tracks here, ready to build."
      className="w-[min(94vw,46rem)]"
      footer={
        <>
          {error && <span className="mr-auto max-w-md text-xs text-danger">{error}</span>}
          <Button variant="ghost" onClick={onClose}>
            Cancel
          </Button>
          <Button variant="primary" onClick={submit} disabled={!ready || submitting}>
            {submitting ? <Spinner /> : <Check size={12} />} Create Build
          </Button>
        </>
      }
    >
      <div className="flex flex-col gap-5 py-1">
        <Section n={1} title="Branch">
          {branchError ? (
            <div className="flex flex-col gap-1">
              <Input value={branch} onChange={(e) => pickBranch(e.target.value)} placeholder="main" className="h-9 max-w-xs" />
              <span className="text-xs text-fg-muted">
                Branches couldn't be read from GitHub ({branchError instanceof ApiError ? branchError.message : 'request failed'}); type one instead.
              </span>
            </div>
          ) : branchesLoading ? (
            <Loading text="Reading branches from GitHub…" />
          ) : (
            <BranchPicker branches={(branchList?.branches ?? []).map((b) => b.name)} defaultBranch={branchList?.default_branch} value={branch} onChange={pickBranch} />
          )}
        </Section>

        <Section n={2} title="Environment File">
          {!branch ? (
            <span className="text-sm text-fg-muted">Pick a branch first.</span>
          ) : envLoading ? (
            <Loading text={`Reading ${branch}…`} />
          ) : envError ? (
            <span className="text-sm text-danger">Couldn't read {branch}: {envError instanceof ApiError ? envError.message : 'request failed'}</span>
          ) : (
            <div className="flex flex-col gap-2">
              {rootEnvs.length > 0 ? (
                <div className="grid gap-2 sm:grid-cols-2">
                  {rootEnvs.map((f) => (
                    <EnvCard key={f.path} file={f} selected={environmentPath === f.path} onSelect={() => setEnvironmentPath(f.path)} />
                  ))}
                </div>
              ) : (
                <span className="text-sm text-fg-muted">No environment files at the root of {branch}. Browse the branch to find one.</span>
              )}
              {pickedNested && <EnvCard file={pickedNested} selected onSelect={() => undefined} />}
              <div>
                <button
                  type="button"
                  onClick={() => setBrowsing((v) => !v)}
                  className="flex items-center gap-1 text-xs font-medium text-accent hover:underline"
                >
                  {browsing ? <ChevronDown size={12} /> : <ChevronRight size={12} />} Browse files on {branch}
                </button>
                {browsing && (
                  <FileBrowser
                    files={envFiles?.yaml_files ?? []}
                    environments={envByPath}
                    selected={environmentPath}
                    onSelect={(p) => setEnvironmentPath(p)}
                  />
                )}
              </div>
              {envFiles?.has_errors && (
                <span className="text-xs text-warning">Some content on this branch doesn't validate, so an environment file may be missing from the list.</span>
              )}
            </div>
          )}
        </Section>

        <Section n={3} title="Builder">
          {buildersLoading ? (
            <Loading text="Loading builders…" />
          ) : !builders || builders.length === 0 ? (
            <span className="text-sm text-fg-muted">
              No builders yet.{' '}
              {me?.is_instance_admin ? (
                <Link to="/admin/infrastructure/new" className="text-accent hover:underline">
                  Add one
                </Link>
              ) : (
                'Ask an instance admin to add one.'
              )}
            </span>
          ) : (
            <div className="grid gap-2 sm:grid-cols-2">
              {builders.map((b) => (
                <PickCard key={b.name} selected={builderName === b.name} onSelect={() => setBuilderName(b.name)}>
                  <ServerCog size={16} className="mt-0.5 shrink-0 text-fg-muted" />
                  <div className="min-w-0">
                    <div className="truncate font-medium text-fg">{b.name}</div>
                    <div className="text-xs text-fg-muted">{KIND_LABEL[b.kind as Kind] ?? b.kind}</div>
                  </div>
                </PickCard>
              ))}
            </div>
          )}
        </Section>
      </div>
    </Modal>
  )
}

function Section({ n, title, children }: { n: number; title: string; children: ReactNode }) {
  return (
    <section className="flex gap-3">
      <span className="flex size-6 shrink-0 items-center justify-center rounded-full bg-accent-soft text-xs font-semibold text-accent-fg">{n}</span>
      <div className="min-w-0 flex-1">
        <div className="mb-2 text-sm font-medium text-fg">{title}</div>
        {children}
      </div>
    </section>
  )
}

function Loading({ text }: { text: string }) {
  return (
    <div className="flex items-center gap-2 text-sm text-fg-muted">
      <Spinner /> {text}
    </div>
  )
}

function PickCard({ selected, onSelect, children }: { selected: boolean; onSelect: () => void; children: ReactNode }) {
  return (
    <button
      type="button"
      onClick={onSelect}
      aria-pressed={selected}
      className={cn(
        'relative flex items-start gap-2.5 rounded-token border px-3 py-2.5 text-left text-sm transition-colors',
        selected ? 'border-accent bg-accent-soft ring-1 ring-accent' : 'border-border bg-surface-raised hover:border-border-strong hover:bg-surface-hover',
      )}
    >
      {children}
      {selected && <Check size={14} className="absolute right-2 top-2 text-accent" />}
    </button>
  )
}

function EnvCard({ file, selected, onSelect }: { file: EnvironmentFileInfo; selected: boolean; onSelect: () => void }) {
  return (
    <PickCard selected={selected} onSelect={onSelect}>
      <FileCode2 size={16} className="mt-0.5 shrink-0 text-fg-muted" />
      <div className="min-w-0 pr-4">
        <div className="truncate font-medium text-fg">{file.name}</div>
        <div className="truncate font-mono text-xs text-fg-muted">{file.path}</div>
      </div>
    </PickCard>
  )
}

function BranchPicker({
  branches,
  defaultBranch,
  value,
  onChange,
}: {
  branches: string[]
  defaultBranch?: string
  value: string
  onChange: (b: string) => void
}) {
  const [open, setOpen] = useState(false)
  const [search, setSearch] = useState('')
  const anchorRef = useRef<HTMLButtonElement>(null)
  const panelRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!open) return
    const onPointerDown = (e: PointerEvent) => {
      const target = e.target as Node
      if (anchorRef.current?.contains(target) || panelRef.current?.contains(target)) return
      setOpen(false)
    }
    window.addEventListener('pointerdown', onPointerDown)
    return () => window.removeEventListener('pointerdown', onPointerDown)
  }, [open])

  const q = search.trim().toLowerCase()
  const shown = branches.filter((b) => !q || b.toLowerCase().includes(q))
  // The default branch first, then the rest alphabetically.
  shown.sort((a, b) => (a === defaultBranch ? -1 : b === defaultBranch ? 1 : a.localeCompare(b)))

  function choose(b: string) {
    onChange(b)
    setOpen(false)
    setSearch('')
  }

  return (
    <>
      <button
        ref={anchorRef}
        type="button"
        onClick={() => setOpen((v) => !v)}
        aria-haspopup="listbox"
        aria-expanded={open}
        className="flex h-9 w-full max-w-sm items-center gap-2 rounded-token border border-border bg-surface-raised px-3 text-left text-sm hover:border-border-strong"
      >
        <GitBranch size={14} className="shrink-0 text-fg-muted" />
        <span className={cn('min-w-0 flex-1 truncate', value ? 'text-fg' : 'text-fg-subtle')}>{value || 'Choose a branch'}</span>
        {value && value === defaultBranch && <Badge tone="neutral">default</Badge>}
        <ChevronsUpDown size={14} className="shrink-0 text-fg-subtle" />
      </button>
      {open && (
        <AnchoredPopover anchorRef={anchorRef} panelRef={panelRef} minWidth={300} maxHeight={320} role="listbox">
          <div className="flex items-center gap-2 border-b border-border px-3 py-2">
            <Search size={12} className="text-fg-muted" />
            <input
              autoFocus
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter' && shown[0]) choose(shown[0])
                if (e.key === 'Escape') setOpen(false)
              }}
              placeholder={`Search ${branches.length} branches…`}
              className="w-full bg-transparent text-sm text-fg outline-none placeholder:text-fg-subtle"
            />
          </div>
          <div className="p-1">
            {shown.length === 0 && <div className="px-2.5 py-2 text-xs text-fg-muted">No branches match</div>}
            {shown.map((b) => (
              <button
                key={b}
                type="button"
                role="option"
                aria-selected={b === value}
                onClick={() => choose(b)}
                className={cn(
                  'flex w-full items-center gap-2 rounded-token-sm px-2.5 py-1.5 text-left text-sm hover:bg-surface-hover',
                  b === value ? 'text-accent' : 'text-fg',
                )}
              >
                <GitBranch size={12} className="shrink-0 text-fg-subtle" />
                <span className="min-w-0 flex-1 truncate">{b}</span>
                {b === defaultBranch && <Badge tone="neutral">default</Badge>}
                {b === value && <Check size={14} />}
              </button>
            ))}
          </div>
        </AnchoredPopover>
      )}
    </>
  )
}

interface TreeNode {
  name: string
  path: string
  children: TreeNode[]
  file: boolean
}

function buildTree(paths: string[]): TreeNode {
  const root: TreeNode = { name: '', path: '', children: [], file: false }
  for (const p of paths) {
    let node = root
    const parts = p.split('/')
    parts.forEach((part, i) => {
      const isFile = i === parts.length - 1
      const path = parts.slice(0, i + 1).join('/')
      let child = node.children.find((c) => c.name === part && c.file === isFile)
      if (!child) {
        child = { name: part, path, children: [], file: isFile }
        node.children.push(child)
      }
      node = child
    })
  }
  const sort = (n: TreeNode) => {
    n.children.sort((a, b) => (a.file === b.file ? a.name.localeCompare(b.name) : a.file ? 1 : -1))
    n.children.forEach(sort)
  }
  sort(root)
  return root
}

// The branch's YAML files as a folder tree. Environment files are the ones
// that can be picked; everything else is shown for orientation.
function FileBrowser({
  files,
  environments,
  selected,
  onSelect,
}: {
  files: string[]
  environments: Map<string, EnvironmentFileInfo>
  selected: string
  onSelect: (path: string) => void
}) {
  const [filter, setFilter] = useState('')
  const [onlyEnvs, setOnlyEnvs] = useState(false)
  const q = filter.trim().toLowerCase()
  const shown = files.filter((f) => (!q || f.toLowerCase().includes(q)) && (!onlyEnvs || environments.has(f)))
  const shownKey = shown.join('\n')
  const tree = useMemo(() => buildTree(shownKey ? shownKey.split('\n') : []), [shownKey])
  const [openDirs, setOpenDirs] = useState<Set<string>>(() => {
    // Open the folders on the way to every environment file.
    const s = new Set<string>()
    for (const p of environments.keys()) {
      const parts = p.split('/')
      for (let i = 1; i < parts.length; i++) s.add(parts.slice(0, i).join('/'))
    }
    return s
  })
  const toggle = (p: string) =>
    setOpenDirs((s) => {
      const next = new Set(s)
      if (next.has(p)) next.delete(p)
      else next.add(p)
      return next
    })

  return (
    <div className="mt-2 overflow-hidden rounded-token border border-border">
      <div className="flex items-center gap-3 border-b border-border bg-surface-sunken px-3 py-1.5">
        <Search size={12} className="text-fg-muted" />
        <input
          value={filter}
          onChange={(e) => setFilter(e.target.value)}
          placeholder="Filter files"
          className="min-w-0 flex-1 bg-transparent text-xs text-fg outline-none placeholder:text-fg-subtle"
        />
        <label className="flex shrink-0 items-center gap-1.5 text-xs text-fg-muted">
          <input type="checkbox" checked={onlyEnvs} onChange={(e) => setOnlyEnvs(e.target.checked)} />
          Environment files only
        </label>
      </div>
      <div className="max-h-64 overflow-auto py-1">
        {shown.length === 0 ? (
          <div className="px-3 py-2 text-xs text-fg-muted">No files match.</div>
        ) : (
          tree.children.map((n) => (
            <TreeRow key={n.path} node={n} depth={0} openDirs={openDirs} forceOpen={!!q} toggle={toggle} environments={environments} selected={selected} onSelect={onSelect} />
          ))
        )}
      </div>
    </div>
  )
}

function TreeRow({
  node,
  depth,
  openDirs,
  forceOpen,
  toggle,
  environments,
  selected,
  onSelect,
}: {
  node: TreeNode
  depth: number
  openDirs: Set<string>
  forceOpen: boolean
  toggle: (p: string) => void
  environments: Map<string, EnvironmentFileInfo>
  selected: string
  onSelect: (p: string) => void
}) {
  const indent = { paddingLeft: `${12 + depth * 16}px` }
  if (!node.file) {
    const isOpen = forceOpen || openDirs.has(node.path)
    return (
      <>
        <button type="button" onClick={() => toggle(node.path)} style={indent} className="flex w-full items-center gap-1.5 py-1 pr-3 text-left text-xs text-fg hover:bg-surface-hover">
          {isOpen ? <ChevronDown size={12} className="text-fg-subtle" /> : <ChevronRight size={12} className="text-fg-subtle" />}
          {isOpen ? <FolderOpen size={13} className="text-fg-muted" /> : <Folder size={13} className="text-fg-muted" />}
          {node.name}
        </button>
        {isOpen &&
          node.children.map((c) => (
            <TreeRow key={c.path} node={c} depth={depth + 1} openDirs={openDirs} forceOpen={forceOpen} toggle={toggle} environments={environments} selected={selected} onSelect={onSelect} />
          ))}
      </>
    )
  }
  const env = environments.get(node.path)
  const isSelected = selected === node.path
  return (
    <button
      type="button"
      disabled={!env}
      onClick={() => env && onSelect(node.path)}
      style={{ paddingLeft: `${12 + depth * 16 + 18}px` }}
      title={env ? `Environment "${env.name}"` : 'Not an environment file'}
      className={cn(
        'flex w-full items-center gap-1.5 py-1 pr-3 text-left text-xs',
        env ? 'text-fg hover:bg-surface-hover' : 'cursor-default text-fg-subtle',
        isSelected && 'bg-accent-soft text-accent-fg',
      )}
    >
      {env ? <FileCode2 size={13} className="text-accent" /> : <FileText size={13} />}
      <span className="truncate font-mono">{node.name}</span>
      {env && <Badge tone="info">environment · {env.name}</Badge>}
      {isSelected && <Check size={12} className="ml-auto" />}
    </button>
  )
}
