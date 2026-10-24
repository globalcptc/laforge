package api

import (
	"net/http"
	"strings"

	"github.com/globalcptc/laforge/internal/db"
	"github.com/jackc/pgx/v5/pgtype"
)

// handleListExternalAccess returns every realized external endpoint in a build:
// for each team/host, the public address:port a competitor or operator connects
// to for a `public:` port (one shared uplink IP with per-team ports on
// Incus/MicroCloud, a public IP per host on AWS). levelRead -- it's the "how do
// I reach these hosts" directory, surfaced from the external_access table the
// runner records, never exposing how the NAT was set up.
func (s *Server) handleListExternalAccess(w http.ResponseWriter, r *http.Request) {
	build, repo, err := s.buildAndOwningRepository(r)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	if _, err := s.requireLevel(r.Context(), r, repo, levelRead); err != nil {
		writeAuthError(w, err)
		return
	}
	rows, err := s.Queries.ListExternalAccessByBuild(r.Context(), build.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if rows == nil {
		rows = []db.ListExternalAccessByBuildRow{}
	}
	writeJSON(w, http.StatusOK, rows)
}

// externalAccessExportRow is one line of the full external-access export: where
// to connect AND how to log in. The password is the host's CURRENT admin
// password (see currentCredentials -- the last admin set_password the agent
// applied, else the environment root password); the username is that built-in
// admin account, derived from the host's OS. Containers get no login (no admin
// account over a public port), so username/password are blank for them.
type externalAccessExportRow struct {
	Team     int32  `json:"team"`
	Host     string `json:"host"`
	IP       string `json:"ip"`
	Port     string `json:"port"`
	Protocol string `json:"protocol"`
	Username string `json:"username"`
	Password string `json:"password"`
	// The fields below let a client select and label endpoints (the scoring
	// export wizard): which object and internal port this endpoint maps, and
	// whether the password is confirmed on the box yet.
	DeployedObjectID pgtype.UUID `json:"deployed_object_id"`
	ObjectName       string      `json:"object_name"`
	Kind             string      `json:"kind"`
	InternalPort     string      `json:"internal_port"`
	PublicAddress    string      `json:"public_address"`
	PasswordApplied  bool        `json:"password_applied"`
	PasswordSource   string      `json:"password_source,omitempty"`
}

// handleExportExternalAccess returns the external endpoints enriched with the
// login credentials, for distributing to teams (e.g. as a CSV from the CLI).
// levelRead -- the same floor handleObjectInfra uses to expose the root password
// per host; this is that data aggregated across the build.
func (s *Server) handleExportExternalAccess(w http.ResponseWriter, r *http.Request) {
	build, repo, err := s.buildAndOwningRepository(r)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	if _, err := s.requireLevel(r.Context(), r, repo, levelRead); err != nil {
		writeAuthError(w, err)
		return
	}
	rows, err := s.Queries.ListExternalAccessByBuild(r.Context(), build.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	// Per-host OS (for the login username) + the environment root password.
	osByName := map[string]string{}
	if content, err := s.loadBuildContent(r, build); err == nil {
		for _, h := range content.Hosts {
			osByName[h.Name] = h.OS
		}
	}
	rootPassword := ""
	if env, err := s.Queries.GetEnvironmentByRevisionAndName(r.Context(), db.GetEnvironmentByRevisionAndNameParams{
		ContentRevisionID: build.ContentRevisionID, Name: build.EnvironmentName,
	}); err == nil {
		rootPassword = db.StrOrEmpty(env.RootPassword)
	}
	passwords, err := currentCredentials(r.Context(), s.Queries, build.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	out := make([]externalAccessExportRow, 0, len(rows))
	for _, e := range rows {
		host := e.ObjectName
		if e.AsName != nil && *e.AsName != "" {
			host = *e.AsName
		}
		ip := e.PublicAddress
		if i := strings.LastIndex(e.PublicAddress, ":"); i >= 0 {
			ip = e.PublicAddress[:i]
		}
		row := externalAccessExportRow{
			Team: e.TeamNumber, Host: host, IP: ip, Port: e.ExternalPort, Protocol: e.Protocol,
			DeployedObjectID: e.DeployedObjectID, ObjectName: e.ObjectName, Kind: e.Kind,
			InternalPort: e.InternalPort, PublicAddress: e.PublicAddress,
		}
		// Only hosts have an admin login; a container exposes a service port,
		// not a desktop/shell to sign into.
		if e.Kind == "host" {
			hp := passwordFor(passwords, e.DeployedObjectID, rootPassword)
			row.Username = adminUsernameForOS(osByName[e.ObjectName])
			row.Password = hp.Password
			row.PasswordApplied = hp.Applied
			row.PasswordSource = hp.Source
		}
		out = append(out, row)
	}
	writeJSON(w, http.StatusOK, out)
}

// adminUsernameForOS is the built-in admin account a team logs in as -- the same
// coarse "anything windows-ish is Windows" split agent delivery already uses.
func adminUsernameForOS(os string) string {
	if strings.Contains(strings.ToLower(os), "win") {
		return "Administrator"
	}
	return "root"
}
