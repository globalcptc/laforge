# LaForge for VS Code

A thin client over `laforge-lsp` (`../../cmd/laforge-lsp`, `../../internal/lsp`) -- diagnostics,
completion, hover, and go to definition are all standard LSP capabilities the extension gets for
free from `vscode-languageclient`. The real code in this extension is: three optional preview
panels beside the editor (live render preview, "show everything available", and a field reference
with real descriptions for every field a type has) and an auto-trigger + keybinding so suggestions
show up without having to already know what to type. See `docs/rewrite-plan.md`'s "Editor support"
section and `docs/milestone-12-status.md`.

## Running it

```bash
# From the repo root: build and install the language server onto PATH.
go install ./cmd/laforge-lsp

cd editors/vscode
npm install
npm run compile
```

Then: **open this folder (`editors/vscode`) itself as a VS Code workspace** -- not the repo
root, and not `examples/lm-test`. `.vscode/launch.json` (real, checked in) only means anything
to VS Code's Run and Debug panel when the folder holding it is the open workspace; opening any
other folder means F5 has no launch configuration to find, and nothing happens with no error
either -- exactly what "the module isn't loading, and there's nothing in the console" looks
like from the outside.

With `editors/vscode` open, **Run and Debug → Launch Extension** (or `F5`) compiles and opens a
second, real "Extension Development Host" window, with the extension loaded and
`examples/lm-test` opened automatically as its content workspace. Confirm you're really in that
second window (its title bar says "Extension Development Host") before assuming something's
broken -- edits in the *first* window, or a stale build, are the other two real ways this looks
like "nothing happened."

If `laforge-lsp` isn't on `PATH`, set `laforge.serverPath` in VS Code settings to its full path.

## What's real here

- **Diagnostics, completion, hover, go to definition**: standard LSP, wired up by
  `vscode-languageclient`'s normal client registration. No custom code in this extension at all --
  everything real is server-side (`internal/lsp`). None of these need a host picked first:
  diagnostics reuse the same validation `laforge check` runs (including unknown script/host/
  network references), schema completion/hover come from the JSON Schema alone, and completion/
  hover inside a script offer the union of every host that script could run on. Because the
  validation is the loader's, a content repo's `.laforgeignore` applies here too: YAML under an
  ignored path (a container's Docker Compose project, another tool's configuration) gets no
  LaForge diagnostics.
- **LaForge: example render** (status bar item, optional): populates its choices from the real,
  current workspace topology (`laforge/pickerOptions`), not a hardcoded list. Always previews
  team 1 -- every team is identical by design, so there's no real choice to make there.
- **LaForge: Show everything available**: the real `laforge/context` request, opened as JSON --
  byte-identical to what `laforge context` prints, since both call the same
  `internal/render.NewContextView`.
- **LaForge: Toggle live render preview**: a side panel that re-requests `laforge/renderPreview`
  on every edit to the active script file, against the example host you picked -- the exact real
  output `render.RenderScript` produces, live against the editor's own unsaved buffer.
- **Suggestions on a new line, not just after typing a character**: real operator feedback --
  VS Code's own quick-suggestions only fire once you've typed something, so landing on a brand new
  blank line inside a `host:`/`script:`/... block via Enter showed nothing. The extension now
  watches for exactly that (a newline that lands the cursor on a blank line in a LaForge file) and
  triggers suggestions itself. `Ctrl+Alt+Space` (`Cmd+Alt+Space` on Mac) does the same thing on
  demand, as a reliable fallback -- **LaForge: Show available fields here** in the command palette.
- **LaForge: Show Field Reference** (`Ctrl+Alt+R` / `Cmd+Alt+R`, or the command palette): a
  documentation panel beside the editor, not a cramped sidebar list -- every field the type under
  your cursor has, from the schema, with a real prose description for each one (every schema under
  `internal/schema/schemas` now uses JSON Schema's own `description` keyword), its type, and its
  enum values where it has them. Follows the cursor as you move it, including across a
  `---`-separated multi-document file, where the second document's own type governs, not the
  first one's. Same `laforge/typeReference` data completion itself uses, so the panel can never
  disagree with what completion would actually offer.
- **Diagnostics that point at the real broken line, not just the top of the file**: another real
  report -- a step referencing a script that doesn't exist (or any other cross-file reference
  error: `depends_on`, `extends`, name collisions) used to always land on line 1, regardless of how
  deep the actual problem was. `internal/lsp/diagnostics.go` now does a best-effort text search for
  the value the error names and resolves to the real line -- not a full YAML-position rewrite, but
  a substantial, tested improvement over always pointing at the header line.

## What's verified, and how

This extension type-checks (`npm run compile`, clean) and the language server it talks to has real
test coverage at every layer: pure-function tests for each feature, a real end-to-end test that
speaks the actual LSP wire protocol (`internal/lsp/server_test.go`, including every custom request
this extension calls), and a real subprocess smoke test over the compiled binary's own stdio
framing (see `docs/milestone-12-status.md`). It has also been launched for real, inside an actual
VS Code Extension Host on this machine -- confirmed live that the language server process spawns,
stays alive, and the extension's own status bar item renders, which only happens once `activate()`
has genuinely run and the language client has connected. Interactive behavior (diagnostics
rendering, completion offering the right entries, the sidebar's own content) has been proven at the
server/protocol level as above, but not clicked through pixel-by-pixel in a live, screenshotted
session -- if something's subtly off, the actual editor integration is the most likely place, not
the logic underneath it.
