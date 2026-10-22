// The companion to render.go's rendered-steps view: handleObjectCloudInit
// shows the FIRST-BOOT configuration a deployed host received -- the
// cloud-init / cloudbase-init / EC2Launch user-data (and, on Incus/MicroCloud
// Windows, the config-drive meta-data) that installs the agent before any
// authored step runs. The Render tab shows the steps the agent runs; this is
// everything that happened on the box before the agent even existed, which is
// exactly what you need when the agent HASN'T run and you're trying to work out
// why -- e.g. why a Windows login password isn't what you expect (it's set by
// cloudbase-init's own image config, not by LaForge; see the note below).
package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/globalcptc/laforge/internal/agentdelivery"
	"github.com/globalcptc/laforge/internal/db"
	"github.com/globalcptc/laforge/internal/loader"
	"github.com/globalcptc/laforge/internal/render"
)

// agentURLPlaceholder stands in for the real per-host agent download URL. That
// URL carries a one-time token issued at deploy and consumed on first boot, so
// it can't be reproduced from a read-only request -- and it's irrelevant to the
// question this view answers (what first-boot config the host received).
const agentURLPlaceholder = "https://<gateway>/agent/<this-host>   # one-time download token, issued at deploy"

type cloudInitView struct {
	OS       string `json:"os"`
	Format   string `json:"format"`   // cloud-init | cloudbase-init | ec2launch
	Delivery string `json:"delivery"` // how it reaches the guest
	UserData string `json:"user_data"`
	MetaData string `json:"meta_data,omitempty"` // config-drive meta-data (Incus/MicroCloud Windows)
	Note     string `json:"note,omitempty"`
	// TempPassword is this host's first-boot temporary admin password (set by the
	// user-data below), so an operator can log in before the agent runs the real
	// set_password step. Admin user is Administrator on Windows, root on Linux.
	TempPassword string `json:"temp_password,omitempty"`
	TempUsername string `json:"temp_username,omitempty"`
}

// handleObjectCloudInit resolves this host's OS (via the same content checkout
// render.go uses) and its build's builder kind, then renders the exact
// agent-install user-data LaForge delivers -- byte-identical to delivery except
// the one-time download URL, which is shown as a placeholder.
func (s *Server) handleObjectCloudInit(w http.ResponseWriter, r *http.Request) {
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
	if obj.Kind == "network" || obj.AsName == nil {
		writeError(w, http.StatusBadRequest, errors.New("a network has no first-boot configuration"))
		return
	}
	team, err := s.Queries.GetTeam(r.Context(), obj.TeamID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	repoRoot := s.RepoRoot
	if s.Checkouts != nil {
		dir, err := s.Checkouts.ForBuild(r.Context(), build.ID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		repoRoot = dir
	}
	c, err := loader.Load(repoRoot)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if len(c.Errors) > 0 {
		writeError(w, http.StatusInternalServerError, errors.New("content is not currently valid -- cannot resolve this host's OS"))
		return
	}
	ctx, err := render.Resolve(c, build.EnvironmentName, *obj.AsName, int(team.TeamNumber))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	kind := s.builderKindForBuild(r.Context(), build)
	// The host's first-boot temporary admin password, derived from the build
	// secret the same way delivery derives it, so what the UI shows matches what
	// the box was given. Best-effort: if the secret can't be read, the view just
	// omits it (and renders the user-data without a password block).
	tempPass := ""
	if secret, serr := db.RenderSecretForBuild(r.Context(), s.Queries, build.ID); serr == nil && len(secret) > 0 {
		tempPass = agentdelivery.TempAdminPassword(string(secret), objectID.String())
	}
	userData, format := agentdelivery.DeliveredUserData(ctx.OS, kind, agentURLPlaceholder, tempPass)

	view := cloudInitView{OS: ctx.OS, Format: string(format), UserData: userData, TempPassword: tempPass}
	if tempPass != "" {
		view.TempUsername = "root"
		if agentdelivery.PlatformFor(ctx.OS) == agentdelivery.Windows {
			view.TempUsername = "Administrator"
		}
	}
	embeds := kind == "incus" || kind == "microcloud"
	switch format {
	case agentdelivery.FormatCloudInit:
		if embeds {
			view.Delivery = "NoCloud config-drive ISO (volume label CIDATA) when the agent is embedded, carrying meta-data, user-data, and the agent binary; otherwise cloud-init's LXD socket datasource"
			view.MetaData = fmt.Sprintf("instance-id: <unique-per-deploy>\nlocal-hostname: %s\n", *obj.AsName)
		} else {
			view.Delivery = "cloud-init user-data (the hoster's native user-data channel)"
		}
	case agentdelivery.FormatCloudbase:
		if embeds {
			view.Delivery = "NoCloud config-drive ISO (volume label CIDATA) -- cloudbase-init can't read the LXD socket, so meta-data, user-data, and the embedded agent binary ride an attached disk"
			view.MetaData = fmt.Sprintf("instance-id: <unique-per-deploy>\nlocal-hostname: %s\n", *obj.AsName)
		} else {
			view.Delivery = "cloudbase-init user-data (the hoster's native user-data channel)"
		}
	case agentdelivery.FormatEC2Launch:
		view.Delivery = "EC2 user-data (base64), executed by EC2Launch"
	}
	note := "This is the first-boot configuration LaForge delivers. The agent is embed-first: on Incus/MicroCloud the patched binary is written onto the config drive and the user-data copies it locally, falling back to the one-time download URL if it isn't there (AWS and the MicroCloud public-NIC path are download-only)."
	if tempPass != "" {
		note += fmt.Sprintf(" It also sets a TEMPORARY %s password (shown above) at first boot so you can log in before the agent runs -- the agent's first set_password step then replaces it with the environment's real password over its secure channel, so the real password never rides user-data. The temporary password is only valid until that step runs.", view.TempUsername)
	}
	if agentdelivery.PlatformFor(ctx.OS) == agentdelivery.Windows {
		note += " The user-data also clears the built-in Administrator's \"password must change at next logon\" flag that cloudbase-init's image config sets."
	}
	view.Note = note
	writeJSON(w, http.StatusOK, view)
}

// builderKindForBuild resolves a build's builder_config.kind (incus, microcloud,
// aws, openstack, fake) via the same build -> configured_build -> builder_config
// chain the runner uses, returning "" when the build has no configured builder
// (an ad-hoc build) or it can't be read -- callers treat "" as "assume the
// default (cloudbase for Windows)".
func (s *Server) builderKindForBuild(ctx context.Context, build db.Build) string {
	if !build.ConfiguredBuildID.Valid {
		return ""
	}
	cb, err := s.Queries.GetConfiguredBuild(ctx, build.ConfiguredBuildID)
	if err != nil {
		return ""
	}
	row, err := s.Queries.GetBuilderConfigByName(ctx, cb.BuilderConfigName)
	if err != nil {
		return ""
	}
	return row.Kind
}
