// The VS Code extension: "the logic lives once in Go beside the
// validator... The VS Code extension becomes a thin wrapper".
// Diagnostics, completion, hover, and go to
// definition are all standard LSP capabilities -- vscode-languageclient
// wires those up automatically the moment the client starts, with no
// custom code here at all, and none of them need a host or team picked
// first: schema completion/hover come from the JSON Schema alone,
// cross-referencing (unknown script/host/network names) comes from the
// same loader.Load validation `laforge check` runs, and template
// completion/hover in a script offers the union of every host that
// script could run on, deliberately independent of any one host. The
// only real code in this file is the two genuinely optional preview
// features: live render preview and "show everything available," each
// needing ONE example host picked from the status bar to resolve
// against -- never a team, since "every team gets exactly the same
// network... nothing is ever templated per team" means team 1 is always
// exactly as representative as any other.
import * as vscode from 'vscode'
import { LanguageClient, LanguageClientOptions, ServerOptions, TransportKind } from 'vscode-languageclient/node'
import * as fs from 'fs'
import * as os from 'os'
import * as path from 'path'

let client: LanguageClient
let statusBarItem: vscode.StatusBarItem
let previewPanel: vscode.WebviewPanel | undefined
let fieldReferencePanel: vscode.WebviewPanel | undefined
let pinned: PinnedContext | undefined

interface PinnedContext {
  environment: string
  host: string
  team: number
}

interface PickerOptions {
  environments: { name: string; teams: number; hosts: string[] }[]
}

interface SchemaFieldOption {
  name: string
  description?: string
  example?: string
}

interface SchemaFieldInfo {
  name: string
  type: string
  enum?: string[]
  description?: string
  example?: string
  options?: SchemaFieldOption[]
}

interface TypeReferenceResult {
  kind: string
  description?: string
  fields: SchemaFieldInfo[]
}

// findOnPath is a minimal, dependency-free "which": vscode-languageclient
// just spawns the bare command and lets the OS resolve it, which is what
// produced a raw ENOENT instead of a useful message when it wasn't found.
function findOnPath(cmd: string): string | undefined {
  const dirs = (process.env.PATH || '').split(path.delimiter)
  const exts = process.platform === 'win32' ? (process.env.PATHEXT || '.EXE').split(path.delimiter) : ['']
  for (const dir of dirs) {
    for (const ext of exts) {
      const candidate = path.join(dir, cmd + ext)
      if (fs.existsSync(candidate)) return candidate
    }
  }
  return undefined
}

// resolveServerPath mirrors what `go install ./cmd/laforge-lsp` (the
// README's own build step) actually produces: a binary in `go env
// GOPATH`'s bin dir, which is very often NOT on PATH for a GUI-launched
// VS Code even once it's on PATH in a terminal. A configured path
// containing a separator is an explicit choice and is trusted as-is;
// only the bare default name gets this fallback search.
function resolveServerPath(configured: string): string | undefined {
  if (configured.includes('/') || configured.includes(path.sep)) {
    return fs.existsSync(configured) ? configured : undefined
  }
  const onPath = findOnPath(configured)
  if (onPath) return onPath
  const exe = process.platform === 'win32' ? configured + '.exe' : configured
  const gopathBin = path.join(process.env.GOPATH || path.join(os.homedir(), 'go'), 'bin', exe)
  return fs.existsSync(gopathBin) ? gopathBin : undefined
}

export async function activate(context: vscode.ExtensionContext): Promise<void> {
  const config = vscode.workspace.getConfiguration('laforge')
  const configuredServerPath = config.get<string>('serverPath', 'laforge-lsp')
  const serverPath = resolveServerPath(configuredServerPath)

  if (!serverPath) {
    // The generic vscode-languageclient failure here is a raw "spawn
    // laforge-lsp ENOENT" -- true but useless. This is the one real
    // setup step every other LSP extension hides (Go's own extension
    // offers an "Install gopls" button for the same reason), so name
    // the exact command and run it in the right directory instead of
    // making the volunteer figure out go install + GOPATH themselves.
    const install = 'Install Now (go install)'
    const choice = await vscode.window.showErrorMessage(
      `LaForge language server ("${configuredServerPath}") isn't installed or on PATH.`,
      install,
      'Open Settings',
    )
    if (choice === install) {
      const repoRoot = path.join(context.extensionPath, '..', '..')
      const term = vscode.window.createTerminal({ name: 'LaForge: install laforge-lsp', cwd: repoRoot })
      term.show()
      term.sendText('go install ./cmd/laforge-lsp && echo "Installed. Run \\"Developer: Reload Window\\" to connect."')
    } else if (choice === 'Open Settings') {
      await vscode.commands.executeCommand('workbench.action.openSettings', 'laforge.serverPath')
    }
    return
  }

  const serverOptions: ServerOptions = {
    run: { command: serverPath, transport: TransportKind.stdio },
    debug: { command: serverPath, transport: TransportKind.stdio },
  }
  const clientOptions: LanguageClientOptions = {
    documentSelector: [
      { scheme: 'file', pattern: '**/*.{yaml,yml}' },
      { scheme: 'file', pattern: '**/*.{sh,ps1,bat,cmd}' },
    ],
  }
  client = new LanguageClient('laforge', 'LaForge Language Server', serverOptions, clientOptions)
  context.subscriptions.push(client)
  await client.start()

  // Deliberately low-key: this only feeds the two optional preview
  // features below (render preview, "show everything available"), never
  // completion, hover, or diagnostics -- those already work from schema
  // and content alone, no host or team needed (see pickContext's own
  // doc comment). The wording says "example" on purpose, so it doesn't
  // read as a required setup step.
  statusBarItem = vscode.window.createStatusBarItem(vscode.StatusBarAlignment.Right, 100)
  statusBarItem.command = 'laforge.pickContext'
  statusBarItem.text = '$(eye) LaForge: example render'
  statusBarItem.tooltip = 'Optional: pick a host to preview rendered scripts against. Completion, hover, and errors already work without this.'
  statusBarItem.show()
  context.subscriptions.push(statusBarItem)

  context.subscriptions.push(vscode.commands.registerCommand('laforge.pickContext', pickContext))
  context.subscriptions.push(vscode.commands.registerCommand('laforge.showContext', showContext))
  context.subscriptions.push(vscode.commands.registerCommand('laforge.toggleRenderPreview', toggleRenderPreview))
  context.subscriptions.push(
    vscode.commands.registerCommand('laforge.showFieldSuggestions', () => {
      void vscode.commands.executeCommand('editor.action.triggerSuggest')
    }),
  )
  context.subscriptions.push(vscode.commands.registerCommand('laforge.toggleFieldReference', toggleFieldReference))

  context.subscriptions.push(vscode.workspace.onDidChangeTextDocument(onDocChange))
  context.subscriptions.push(vscode.window.onDidChangeActiveTextEditor(() => void updatePreview()))

  // "Show all of the configuration options that are available," as a
  // real markdown-style reference panel beside the editor -- not a
  // cramped Explorer-sidebar list. Toggled via laforge.toggleFieldReference
  // (same pattern as toggleRenderPreview), and kept live as the cursor
  // moves or the document changes.
  context.subscriptions.push(vscode.window.onDidChangeActiveTextEditor(() => void updateFieldReference()))
  context.subscriptions.push(vscode.window.onDidChangeTextEditorSelection((e) => void updateFieldReference(e.textEditor)))
  context.subscriptions.push(
    vscode.workspace.onDidChangeTextDocument((e) => {
      if (vscode.window.activeTextEditor && e.document === vscode.window.activeTextEditor.document) {
        void updateFieldReference()
      }
    }),
  )
}

export async function deactivate(): Promise<void> {
  if (client) {
    await client.stop()
  }
}

// pickContext is the status bar's own command, for the two optional
// preview features (render preview, "show everything available") --
// nothing else needs it. Populates its choices from the real, current
// workspace topology (laforge/pickerOptions, internal/lsp/custom.go's
// LaForgePickerOptions), not a hardcoded list, so it always reflects
// whatever content is actually open.
//
// Always previews team 1, never asks. "Every team gets exactly the same
// network... nothing is ever templated per team"
// means team 1's resolved vars are exactly as representative as team
// 30's -- there is no real choice here, so asking for one is just a
// pointless extra click, not a meaningful option.
async function pickContext(): Promise<void> {
  const opts = await client.sendRequest<PickerOptions>('laforge/pickerOptions', {})
  if (!opts.environments || opts.environments.length === 0) {
    void vscode.window.showInformationMessage('LaForge: no environments found in this workspace yet.')
    return
  }
  const envName =
    opts.environments.length === 1
      ? opts.environments[0].name
      : await vscode.window.showQuickPick(
          opts.environments.map((e) => e.name),
          { placeHolder: 'Environment' },
        )
  if (!envName) return
  const env = opts.environments.find((e) => e.name === envName)
  if (!env) return

  const host = await vscode.window.showQuickPick(env.hosts, {
    placeHolder: 'Host to preview (as name) -- every team is identical, so this always previews team 1',
  })
  if (!host) return

  pinned = { environment: envName, host, team: 1 }
  statusBarItem.text = `$(eye) example: ${pinned.host}`
  void updatePreview()
}

// showContext is "'Show everything available': a command that opens the
// full context for the pinned host, the same output as `laforge
// context`" -- the real laforge/context request,
// byte-identical to the CLI's own output (internal/render.NewContextView
// is the one shared implementation both use).
async function showContext(): Promise<void> {
  if (!pinned) {
    await pickContext()
    if (!pinned) return
  }
  const result = await client.sendRequest('laforge/context', pinned)
  const doc = await vscode.workspace.openTextDocument({
    content: JSON.stringify(result, null, 2),
    language: 'json',
  })
  await vscode.window.showTextDocument(doc, { preview: true })
}

// toggleRenderPreview is "Live preview: the rendered script beside the
// source, updating as you type" -- a side-by-side webview that
// re-requests laforge/renderPreview on every edit to the active script.
async function toggleRenderPreview(): Promise<void> {
  if (previewPanel) {
    previewPanel.dispose()
    previewPanel = undefined
    return
  }
  previewPanel = vscode.window.createWebviewPanel(
    'laforgeRenderPreview',
    'LaForge: Rendered script',
    vscode.ViewColumn.Beside,
    {},
  )
  previewPanel.onDidDispose(() => {
    previewPanel = undefined
  })
  await updatePreview()
}

// toggleFieldReference is "a sidebar that opens / can be opened to show
// all of the configuration options that are available" -- moved beside
// the editor, not into the Explorer panel, per real feedback that a
// bare field-name list off to the side with no room to read was "more
// confusing," not less. Same toggle pattern as toggleRenderPreview.
async function toggleFieldReference(): Promise<void> {
  if (fieldReferencePanel) {
    fieldReferencePanel.dispose()
    fieldReferencePanel = undefined
    return
  }
  // Captured before the panel exists: createWebviewPanel would otherwise
  // steal focus first (its showOptions default preserveFocus to false),
  // which would make activeTextEditor already undefined by the time the
  // first render reads it.
  const editorAtOpen = vscode.window.activeTextEditor
  fieldReferencePanel = vscode.window.createWebviewPanel(
    'laforgeFieldReference',
    'LaForge: Field Reference',
    { viewColumn: vscode.ViewColumn.Beside, preserveFocus: true },
    {},
  )
  fieldReferencePanel.onDidDispose(() => {
    fieldReferencePanel = undefined
  })
  await updateFieldReference(editorAtOpen)
}

// updateFieldReference deliberately does nothing when there's no active
// text editor at all -- clicking into the panel itself (a webview is
// never a text editor), the sidebar, or a terminal all fire
// onDidChangeActiveTextEditor with undefined, and the panel should keep
// showing whatever it last showed rather than blanking out. It only
// replaces the content for a genuine "wrong kind of file" case, where
// there IS an active editor but it isn't a LaForge YAML file.
async function updateFieldReference(editor: vscode.TextEditor | undefined = vscode.window.activeTextEditor): Promise<void> {
  if (!fieldReferencePanel) return
  if (!editor) return
  if (!isLaForgeYAML(editor.document)) {
    fieldReferencePanel.webview.html = renderReferenceHTML(null, 'Open a LaForge host, container, network, script, or environment file to see its fields here.')
    return
  }
  try {
    const result = await client.sendRequest<TypeReferenceResult>('laforge/typeReference', {
      uri: editor.document.uri.toString(),
      position: editor.selection.active,
    })
    fieldReferencePanel.webview.html = renderReferenceHTML(result, null)
  } catch (err) {
    const message = err instanceof Error ? err.message : String(err)
    fieldReferencePanel.webview.html = renderReferenceHTML(null, message)
  }
}

function onDocChange(e: vscode.TextDocumentChangeEvent): void {
  if (vscode.window.activeTextEditor && e.document === vscode.window.activeTextEditor.document) {
    void updatePreview()
  }
  maybeAutoTriggerSuggest(e)
}

// maybeAutoTriggerSuggest is "when the cursor creates a new line in a
// type object, show all of the available options" made automatic, not
// just a keypress (laforge.showFieldSuggestions, bound to a real
// keybinding, is still there as a reliable fallback -- see
// package.json). VS Code's own quick-suggestions only fire once a
// character is typed; landing on a brand new blank line via Enter alone
// shows nothing without this. Fires only for a genuine newline
// insertion (not every edit) that lands the cursor on a now-blank line
// in a real LaForge YAML file, so it never interrupts typing elsewhere
// (mid-word, inside a string, etc.).
function maybeAutoTriggerSuggest(e: vscode.TextDocumentChangeEvent): void {
  if (!isLaForgeYAML(e.document)) return
  const change = e.contentChanges[e.contentChanges.length - 1]
  if (!change || !change.text.startsWith('\n')) return
  const editor = vscode.window.activeTextEditor
  if (!editor || editor.document !== e.document) return
  const line = editor.document.lineAt(editor.selection.active.line)
  if (line.text.trim() !== '') return
  void vscode.commands.executeCommand('editor.action.triggerSuggest')
}

function isLaForgeYAML(doc: vscode.TextDocument): boolean {
  return doc.languageId === 'yaml' || /\.ya?ml$/i.test(doc.fileName)
}

// md is a minimal inline-markdown-to-HTML pass -- enough for the schema
// descriptions actually written (backtick code spans, **bold**), not a
// general renderer. Escapes HTML first so schema text can never inject
// markup.
function md(s: string): string {
  const esc = s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
  return esc.replace(/`([^`]+)`/g, '<code>$1</code>').replace(/\*\*([^*]+)\*\*/g, '<strong>$1</strong>')
}

// renderReferenceHTML is "something like markdown that appears
// (preferably on the right column)" made real: real prose per field
// (name, type, enum, and now a genuine description -- see
// internal/schema/schemas/*.schema.json's own "description" keywords),
// laid out with headings and spacing that read like documentation, in
// the same beside-the-editor webview pattern as the render preview.
function renderReferenceHTML(result: TypeReferenceResult | null, message: string | null): string {
  const esc = (s: string) => s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
  const style = `
    <style>
      body {
        font-family: var(--vscode-font-family); font-size: var(--vscode-font-size);
        line-height: 1.5; padding: 24px 32px; max-width: 720px;
      }
      h1 { font-size: 1.3em; text-transform: capitalize; margin: 0 0 4px; }
      .subtitle { color: var(--vscode-descriptionForeground); margin-bottom: 24px; font-size: 0.95em; }
      .field { margin-bottom: 20px; padding-bottom: 16px; border-bottom: 1px solid var(--vscode-widget-border, transparent); }
      .field:last-child { border-bottom: none; }
      .field-head { display: flex; align-items: baseline; gap: 8px; flex-wrap: wrap; }
      .name { font-weight: 600; font-family: var(--vscode-editor-font-family); font-size: 1.05em; }
      .badge {
        color: var(--vscode-descriptionForeground); font-size: 0.8em;
        border: 1px solid var(--vscode-widget-border, currentColor); border-radius: 3px;
        padding: 0px 6px; opacity: 0.85;
      }
      .desc { margin-top: 6px; }
      .desc code { font-family: var(--vscode-editor-font-family); background: var(--vscode-textCodeBlock-background); padding: 1px 4px; border-radius: 3px; }
      .enum { margin-top: 8px; font-size: 0.92em; color: var(--vscode-descriptionForeground); }
      .enum ul { margin: 4px 0 0; padding-left: 20px; }
      .example { margin-top: 10px; background: var(--vscode-textCodeBlock-background); border-radius: 4px; padding: 10px 12px; overflow-x: auto; }
      .example pre { margin: 0; font-family: var(--vscode-editor-font-family); font-size: 0.92em; white-space: pre; }
      .options { margin-top: 12px; border-left: 2px solid var(--vscode-widget-border, currentColor); padding-left: 14px; }
      .option { margin-bottom: 16px; }
      .option:last-child { margin-bottom: 0; }
      .option-name { font-weight: 600; font-family: var(--vscode-editor-font-family); }
      .option-desc { margin-top: 3px; font-size: 0.95em; }
      .option-example { margin-top: 6px; background: var(--vscode-textCodeBlock-background); border-radius: 4px; padding: 8px 10px; overflow-x: auto; }
      .option-example pre { margin: 0; font-family: var(--vscode-editor-font-family); font-size: 0.9em; white-space: pre; }
      .msg { color: var(--vscode-descriptionForeground); font-style: italic; padding: 24px 32px; }
    </style>
  `
  if (message) {
    return `${style}<div class="msg">${esc(message)}</div>`
  }
  if (!result) {
    return `${style}<div class="msg">Nothing to show.</div>`
  }
  const fields = result.fields
    .map((f) => {
      const typeBadge = f.type ? `<span class="badge">${esc(f.type)}</span>` : ''
      const enumBlock =
        f.enum && f.enum.length > 0
          ? `<div class="enum">One of:<ul>${f.enum.map((v) => `<li><code>${esc(v)}</code></li>`).join('')}</ul></div>`
          : ''
      const descLine = f.description ? `<div class="desc">${md(f.description)}</div>` : `<div class="desc msg">No description yet.</div>`
      const exampleBlock = f.example ? `<div class="example"><pre>${esc(f.example)}</pre></div>` : ''
      const optionsBlock =
        f.options && f.options.length > 0
          ? `<div class="options">${f.options
              .map((o) => {
                const oDesc = o.description ? `<div class="option-desc">${md(o.description)}</div>` : ''
                const oExample = o.example ? `<div class="option-example"><pre>${esc(o.example)}</pre></div>` : ''
                return `<div class="option"><div class="option-name">${esc(o.name)}</div>${oDesc}${oExample}</div>`
              })
              .join('')}</div>`
          : ''
      return `<div class="field"><div class="field-head"><span class="name">${esc(f.name)}</span>${typeBadge}</div>${descLine}${enumBlock}${exampleBlock}${optionsBlock}</div>`
    })
    .join('')
  const intro = result.description ? md(result.description) : `Every field a ${esc(result.kind)} can have, from its schema.`
  return `${style}<h1>${esc(result.kind)}</h1><div class="subtitle">${intro} Follows your cursor -- move it into a different file or document to see that type's own fields.</div>${fields}`
}

async function updatePreview(): Promise<void> {
  if (!previewPanel || !pinned) return
  const editor = vscode.window.activeTextEditor
  if (!editor) return

  try {
    const result = await client.sendRequest<{ rendered: string }>('laforge/renderPreview', {
      uri: editor.document.uri.toString(),
      environment: pinned.environment,
      host: pinned.host,
      team: pinned.team,
    })
    previewPanel.webview.html = renderHTML(result.rendered, null)
  } catch (err) {
    const message = err instanceof Error ? err.message : String(err)
    previewPanel.webview.html = renderHTML('', message)
  }
}

function renderHTML(rendered: string, error: string | null): string {
  const esc = (s: string) => s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
  if (error) {
    return `<pre style="color:#f66;white-space:pre-wrap;">${esc(error)}</pre>`
  }
  return `<pre style="white-space:pre-wrap;font-family:monospace;">${esc(rendered)}</pre>`
}
