package api

import (
	"net/http"

	"github.com/globalcptc/laforge/internal/db"
)

// The read side of "where does this host actually live, and how do I get into
// it by hand?" (direct feedback). The placement facts are builder-specific --
// which ones exist varies by builder kind -- so they're returned as a plain
// label/value list the UI renders generically rather than a fixed schema.

type infraFact struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

type objectInfra struct {
	Placement []infraFact `json:"placement"`
	// Password is the environment-wide root/administrator password, exposed so
	// an operator can log in by hand. It is environment-level, not per-host
	// (LaForge stores no per-host password), and only ever returned to a
	// caller already authorized to read this build.
	Password string `json:"password,omitempty"`
}

// handleObjectInfra resolves an object's builder placement -- the details
// needed to find it directly on the builder (cluster, project/pool, instance
// name) -- plus the environment's root password, for hand access.
func (s *Server) handleObjectInfra(w http.ResponseWriter, r *http.Request) {
	build, repo, err := s.buildAndOwningRepository(r)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	if _, err := s.requireLevel(r.Context(), r, repo, levelRead); err != nil {
		writeAuthError(w, err)
		return
	}
	objectID, err := s.objectInBuild(r, build.ID)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	obj, err := s.Queries.GetDeployedObject(r.Context(), objectID)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}

	facts := make([]infraFact, 0, 6)
	add := func(label, value string) {
		if value != "" {
			facts = append(facts, infraFact{Label: label, Value: value})
		}
	}

	// Builder placement, resolved best-effort through configured_build ->
	// builder_config -> builder_credential. Each hop that fails is simply
	// skipped, so a partially-resolvable chain still shows what it can.
	if cb, err := s.Queries.GetConfiguredBuild(r.Context(), build.ConfiguredBuildID); err == nil {
		if bc, err := s.Queries.GetBuilderConfigByName(r.Context(), cb.BuilderConfigName); err == nil {
			add("Builder", bc.Kind+" ("+bc.Name+")")
			apiURL := ""
			if bc.IncusApiUrl != nil {
				apiURL = *bc.IncusApiUrl
			}
			if bc.IncusCredentialID.Valid {
				if cred, err := s.Queries.GetBuilderCredential(r.Context(), bc.IncusCredentialID); err == nil {
					add("Cluster", cred.ServerName)
					if apiURL == "" {
						apiURL = cred.ApiUrl
					}
				}
			}
			add("API endpoint", apiURL)
			if bc.IncusStoragePool != nil {
				add("Storage pool", *bc.IncusStoragePool)
			}
		}
	}
	add("Instance", db.StrOrEmpty(obj.ExternalRef))
	add("Network", db.StrOrEmpty(obj.NetworkName))

	infra := objectInfra{Placement: facts}
	if env, err := s.Queries.GetEnvironmentByRevisionAndName(r.Context(), db.GetEnvironmentByRevisionAndNameParams{
		ContentRevisionID: build.ContentRevisionID,
		Name:              build.EnvironmentName,
	}); err == nil {
		infra.Password = db.StrOrEmpty(env.RootPassword)
	}

	writeJSON(w, http.StatusOK, infra)
}
