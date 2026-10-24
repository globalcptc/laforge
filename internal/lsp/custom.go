package lsp

import (
	"fmt"
	"strings"

	"go.lsp.dev/protocol"

	"github.com/globalcptc/laforge/internal/render"
)

// ContextResult and RenderPreviewResult are the two "LaForge-specific"
// requests from the Editor support table that
// have no standard LSP method: "Live preview: the rendered script beside
// the source, updating as you type" and "'Show everything available':
// a command that opens the full context for the pinned host, the same
// output as `laforge context`." Both are driven by the editor's own
// "pinned context" (host and team, chosen in a status bar) rather than
// cursor position, since a script can run on several hosts at once (see
// scriptscope.go's own reasoning).

// LaForgeContext is "Show everything available" -- byte-for-byte the
// same data `laforge context` prints (internal/render.NewContextView),
// over the custom "laforge/context" request.
func LaForgeContext(w *Workspace, envName, host string, team int) (*render.ContextView, error) {
	content, _ := w.Snapshot()
	if content == nil {
		return nil, fmt.Errorf("no content loaded yet")
	}
	ctx, err := render.Resolve(content, envName, host, team)
	if err != nil {
		return nil, err
	}
	v := render.NewContextView(ctx)
	return &v, nil
}

// LaForgeRenderPreview is "Live preview: the rendered script beside the
// source, updating as you type" -- the exact real output
// render.RenderScript would produce (byte-identical to what an agent
// receives, same as internal/api/render.go's own post-hoc rendering
// endpoint), against the workspace's live scratch mirror so an unsaved
// edit shows up immediately, over the custom "laforge/renderPreview"
// request.
func LaForgeRenderPreview(w *Workspace, relScriptPath, envName, host string, team int) (string, error) {
	content, _ := w.Snapshot()
	if content == nil {
		return "", fmt.Errorf("no content loaded yet")
	}
	script := scriptForSourceFile(content, relScriptPath)
	if script == nil {
		return "", fmt.Errorf("%s is not a known script's source file", relScriptPath)
	}
	ctx, err := render.Resolve(content, envName, host, team)
	if err != nil {
		return "", err
	}
	return render.RenderScript(w.scratch, script, ctx, content)
}

// PickerOptions is what a "pinned host and team" status bar picker
// needs to populate itself: every real environment name, and for each,
// every real `as` name in its topology and its real team count.
type PickerOptions struct {
	Environments []EnvironmentPicker `json:"environments"`
}

type EnvironmentPicker struct {
	Name  string   `json:"name"`
	Teams int      `json:"teams"`
	Hosts []string `json:"hosts"`
}

func LaForgePickerOptions(w *Workspace) PickerOptions {
	content, _ := w.Snapshot()
	if content == nil {
		return PickerOptions{}
	}
	var out PickerOptions
	for _, env := range content.Environments {
		ep := EnvironmentPicker{Name: env.Name, Teams: env.Teams}
		seen := make(map[string]bool)
		for _, objs := range env.Networks {
			for _, copies := range objs {
				for _, cp := range copies {
					if !seen[cp.As] {
						seen[cp.As] = true
						ep.Hosts = append(ep.Hosts, cp.As)
					}
				}
			}
		}
		out.Environments = append(out.Environments, ep)
	}
	return out
}

// TypeReferenceResult is the sidebar's own real data: every field a
// config document's type has, straight from the schema -- "we should
// also include a sidebar... to show all of the configuration options
// that are available" (a real, direct ask, not a nice-to-have). Reuses
// exactly the same documentAt/headerKind/schemaFieldsForKind chain
// configCompletion already runs, over the custom "laforge/typeReference"
// request, so the sidebar can never show a different field list than
// what completion would actually offer at that same cursor position --
// one source of truth, not two independently-maintained ones.
type TypeReferenceResult struct {
	Kind        string        `json:"kind"`
	Description string        `json:"description,omitempty"`
	Fields      []schemaField `json:"fields"`
}

// LaForgeTypeReference resolves relPath/pos to the type governing the
// document under the cursor (not necessarily the file's first document
// -- see documentAt's own doc comment) and returns every field its
// schema defines. Errors (not a YAML file, no recognized header yet, an
// unknown kind) are real, user-facing states the sidebar shows as
// messages rather than an empty panel -- "no header yet" is the normal
// state for a document that's still just `host:` with nothing typed
// after it, not a bug.
func LaForgeTypeReference(relPath, text string, pos protocol.Position) (*TypeReferenceResult, error) {
	if !strings.HasSuffix(relPath, ".yaml") && !strings.HasSuffix(relPath, ".yml") {
		return nil, fmt.Errorf("not a LaForge content file")
	}
	doc, _ := documentAt(text, pos.Line)
	kind, ok := headerKind(doc)
	if !ok {
		return nil, fmt.Errorf("no recognized type header (environment/network/host/container/script) in this document yet")
	}
	fields, err := schemaFieldsForKind(kind)
	if err != nil {
		return nil, err
	}
	desc, err := kindDescription(kind)
	if err != nil {
		return nil, err
	}
	return &TypeReferenceResult{Kind: string(kind), Description: desc, Fields: fields}, nil
}
