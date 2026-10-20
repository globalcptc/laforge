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
	userData, format := agentdelivery.DeliveredUserData(ctx.OS, kind, agentURLPlaceholder)

	view := cloudInitView{OS: ctx.OS, Format: string(format), UserData: userData}
	switch format {
	case agentdelivery.FormatCloudInit:
		view.Delivery = "cloud-init user-data (read from the LXD socket datasource on Incus/MicroCloud, or the hoster's native user-data channel)"
	case agentdelivery.FormatCloudbase:
		if kind == "incus" || kind == "microcloud" {
			view.Delivery = "NoCloud config-drive ISO (volume label CIDATA) -- cloudbase-init can't read the LXD socket, so meta-data + user-data ride an attached disk"
			view.MetaData = fmt.Sprintf("instance-id: <unique-per-deploy>\nlocal-hostname: %s\n", *obj.AsName)
		} else {
			view.Delivery = "cloudbase-init user-data (the hoster's native user-data channel)"
		}
	case agentdelivery.FormatEC2Launch:
		view.Delivery = "EC2 user-data (base64), executed by EC2Launch"
	}
	if agentdelivery.PlatformFor(ctx.OS) == agentdelivery.Windows {
		view.Note = "This is the first-boot user-data LaForge delivers. It installs the LaForge agent and clears the built-in Administrator's \"password must change at next logon\" flag -- it does NOT set the Windows password. The Administrator password is set by cloudbase-init's own password plugin, which is configured on the image (random by default) and runs before this user-data. So on a host where the agent hasn't run yet, the login password came from the image's cloudbase-init config, not from LaForge -- and the image's baked-in password no longer applies once cloudbase-init has reset it. A LaForge set_password step, run by the agent, is what sets a password you'd know."
	}
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
