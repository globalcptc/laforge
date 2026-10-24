// Public instance config: values the UI needs before a person has even
// signed in, safe to hand to anyone -- see Server.AppSlug's own doc
// comment ("cosmetic, never used for an auth decision"). Deliberately
// its own small handler rather than folded into an existing one: this is
// the one place "what does this LaForge instance call itself on GitHub"
// belongs, and it has to be unauthenticated, since the sign-in screen
// and the "Add Repository" flow both need it before there's a session.
package api

import "net/http"

type publicConfig struct {
	// GithubAppSlug builds "Install on GitHub" links
	// (github.com/apps/<slug>/installations/new) in the UI. Empty when
	// this instance has no GitHub App configured at all (AppID unset) --
	// the UI falls back to its own direct-registration path in that
	// case, the same way every App-dependent server behavior already
	// does (see Server.AppPrivateKey's own doc comment).
	GithubAppSlug string `json:"github_app_slug"`
}

func (s *Server) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, publicConfig{GithubAppSlug: s.AppSlug})
}
