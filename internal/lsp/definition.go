package lsp

import (
	"path/filepath"
	"regexp"

	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

var identRE = regexp.MustCompile(`[A-Za-z0-9_-]+`)

// Definition is "Go to definition: a script name jumps to the script, a
// host name to the host file, a people source to the CSV"
// -- driven purely by
// which real, bare name is under the cursor, so it works the same way
// whether that name appears in a `steps:` script reference, a
// `depends_on` list, an environment's networks topology, or a script's
// own `people:` field, without needing to know which of those it's
// looking at.
func Definition(w *Workspace, relPath, text string, pos protocol.Position) []protocol.Location {
	content, _ := w.Snapshot()
	if content == nil {
		return nil
	}
	line := lineAt(text, pos.Line)
	name, ok := tokenAt(line, pos.Character, identRE)
	if !ok {
		return nil
	}

	// Hosts, containers, and networks share one bare-name namespace
	// ("Names are bare and must not collide across networks, hosts, and
	// containers"), so at most one of these three can
	// ever match.
	for i := range content.Hosts {
		if content.Hosts[i].Name == name {
			return locationAt(w.RepoRoot, content.Hosts[i].SourceFile)
		}
	}
	for i := range content.Containers {
		if content.Containers[i].Name == name {
			return locationAt(w.RepoRoot, content.Containers[i].SourceFile)
		}
	}
	for i := range content.Networks {
		if content.Networks[i].Name == name {
			return locationAt(w.RepoRoot, content.Networks[i].SourceFile)
		}
	}
	for i := range content.Scripts {
		if content.Scripts[i].Name == name {
			return locationAt(w.RepoRoot, content.Scripts[i].SourceFile)
		}
	}
	for i := range content.People {
		if content.People[i].Name == name {
			return locationAt(w.RepoRoot, content.People[i].SourceFile)
		}
	}
	return nil
}

func locationAt(repoRoot, rel string) []protocol.Location {
	if rel == "" {
		return nil
	}
	return []protocol.Location{{
		URI:   uri.File(filepath.Join(repoRoot, rel)),
		Range: protocol.Range{Start: protocol.Position{}, End: protocol.Position{Line: 0, Character: 1}},
	}}
}
