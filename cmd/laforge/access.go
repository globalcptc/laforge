package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"
)

// accessExportRow is one line of the external-access export (where to connect +
// how to log in), mirroring internal/api.externalAccessExportRow.
type accessExportRow struct {
	Team             int32  `json:"team"`
	Host             string `json:"host"`
	IP               string `json:"ip"`
	Port             string `json:"port"`
	Protocol         string `json:"protocol"`
	Username         string `json:"username"`
	Password         string `json:"password"`
	DeployedObjectID string `json:"deployed_object_id"`
	Kind             string `json:"kind"`
	InternalPort     string `json:"internal_port"`
	PublicAddress    string `json:"public_address"`
	PasswordApplied  bool   `json:"password_applied"`
}

// runAccess lists the external endpoints a build exposes -- for each team/host,
// the public address:port to connect to for its `public:` ports (RDP and the
// like). On Incus/MicroCloud that's a shared IP with per-team ports; on AWS a
// public IP per host. With --csv it exports the full connection detail --
// team, host, IP, port, and the login username/password -- for handing to teams.
func runAccess(args []string) error {
	fs := flag.NewFlagSet("access", flag.ExitOnError)
	build := fs.String("build", "", "build id (required)")
	asCSV := fs.Bool("csv", false, "export as CSV including login username/password (for distribution)")
	asJSON := fs.Bool("json", false, "export as JSON including login username/password (for distribution)")
	scoring := fs.Bool("scoring", false, "export as the scoring app's Team Connections import JSON (needs --season and --competition)")
	season := fs.String("season", "", "scoring season shortname, e.g. cptc12 (with --scoring)")
	competition := fs.String("competition", "", "scoring competition shortname, e.g. us-socal (with --scoring)")
	teamOffset := fs.Int("team-offset", 0, "added to each LaForge team number for the scoring team number (with --scoring)")
	exclude := fs.String("exclude", "", "comma-separated host names to leave out of every team, e.g. OutOfScope (with --scoring)")
	instructionsFile := fs.String("instructions-file", "", "markdown file of captain instructions for every team; omit to keep scoring's existing ones (with --scoring)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *build == "" {
		return fmt.Errorf("usage: laforge access --build <id> [--csv | --json | --scoring --season <s> --competition <c>]")
	}
	if n := countTrue(*asCSV, *asJSON, *scoring); n > 1 {
		return fmt.Errorf("choose one of --csv, --json or --scoring")
	}
	if *scoring {
		if *season == "" || *competition == "" {
			return fmt.Errorf("--scoring needs --season and --competition (the scoring app's shortnames)")
		}
		instructions := ""
		if *instructionsFile != "" {
			b, err := os.ReadFile(*instructionsFile)
			if err != nil {
				return err
			}
			instructions = string(b)
		}
		rows, err := fetchAccessExport(*build)
		if err != nil {
			return err
		}
		entries := scoringEntries(rows, *season, *competition, *teamOffset, splitList(*exclude), instructions)
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(entries)
	}
	if *asCSV || *asJSON {
		rows, err := fetchAccessExport(*build)
		if err != nil {
			return err
		}
		if *asJSON {
			return writeAccessJSON(rows)
		}
		return writeAccessCSV(rows)
	}

	var rows []struct {
		TeamNumber    int32   `json:"team_number"`
		ObjectName    string  `json:"object_name"`
		AsName        *string `json:"as_name"`
		Kind          string  `json:"kind"`
		Protocol      string  `json:"protocol"`
		InternalPort  string  `json:"internal_port"`
		PublicAddress string  `json:"public_address"`
	}
	if err := newAPIClient().do("GET", "/builds/"+url.PathEscape(*build)+"/external-access", nil, &rows); err != nil {
		return err
	}
	if len(rows) == 0 {
		fmt.Println("no external access configured for this build")
		return nil
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
	fmt.Fprintln(w, "TEAM\tHOST\tPROTO\tPORT\tCONNECT TO")
	for _, r := range rows {
		host := r.ObjectName
		if r.AsName != nil && *r.AsName != "" {
			host = *r.AsName
		}
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\n", r.TeamNumber, host, r.Protocol, r.InternalPort, r.PublicAddress)
	}
	return w.Flush()
}

// fetchAccessExport pulls the full external-access export (with credentials).
func fetchAccessExport(build string) ([]accessExportRow, error) {
	var rows []accessExportRow
	if err := newAPIClient().do("GET", "/builds/"+url.PathEscape(build)+"/external-access/export", nil, &rows); err != nil {
		return nil, err
	}
	return rows, nil
}

// writeAccessCSV writes the export to stdout as CSV -- pipe it to a file to hand
// each team its own connection sheet.
func writeAccessCSV(rows []accessExportRow) error {
	w := csv.NewWriter(os.Stdout)
	if err := w.Write([]string{"team", "host", "ip", "port", "protocol", "username", "password"}); err != nil {
		return err
	}
	for _, r := range rows {
		if err := w.Write([]string{strconv.Itoa(int(r.Team)), r.Host, r.IP, r.Port, r.Protocol, r.Username, r.Password}); err != nil {
			return err
		}
	}
	w.Flush()
	return w.Error()
}

// writeAccessJSON writes the export to stdout as indented JSON -- the same rows,
// for feeding another tool.
func writeAccessJSON(rows []accessExportRow) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(rows)
}

// scoringHost / scoringEntry are the scoring app's Team Connections bulk-import
// shape (scoring repo docs/examples/README.md) -- the same JSON the External
// Access page's Export for Scoring wizard produces.
type scoringHost struct {
	Host     string `json:"host"`
	URL      string `json:"url"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type scoringEntry struct {
	Season       string        `json:"season"`
	Competition  string        `json:"competition"`
	TeamNumber   int           `json:"team_number"`
	Instructions string        `json:"instructions,omitempty"`
	Hosts        []scoringHost `json:"hosts"`
}

// scoringEntries shapes the export rows into one scoring entry per team: url is
// the bare ip:port, the login is the host's current admin account. Containers
// (no login) and excluded host names are left out; a host with several public
// ports gets the port in its label; a team left with no hosts is dropped
// (scoring rejects an empty one). A selected endpoint whose password hasn't
// been applied on the box yet is warned about on stderr.
func scoringEntries(rows []accessExportRow, season, competition string, offset int, exclude []string, instructions string) []scoringEntry {
	skip := map[string]bool{}
	for _, h := range exclude {
		skip[strings.ToLower(h)] = true
	}
	ports := map[string]int{}
	for _, r := range rows {
		ports[r.DeployedObjectID]++
	}
	byTeam := map[int32][]scoringHost{}
	var teams []int32
	for _, r := range rows {
		if r.Kind != "host" || r.Username == "" || r.Password == "" || skip[strings.ToLower(r.Host)] {
			continue
		}
		if !r.PasswordApplied {
			fmt.Fprintf(os.Stderr, "warning: team %d %s: password not yet applied on the box (its step hasn't run)\n", r.Team, r.Host)
		}
		label := r.Host
		if ports[r.DeployedObjectID] > 1 {
			label = fmt.Sprintf("%s (%s/%s)", r.Host, r.Protocol, r.InternalPort)
		}
		if _, ok := byTeam[r.Team]; !ok {
			teams = append(teams, r.Team)
		}
		byTeam[r.Team] = append(byTeam[r.Team], scoringHost{Host: label, URL: r.PublicAddress, Username: r.Username, Password: r.Password})
	}
	sort.Slice(teams, func(i, j int) bool { return teams[i] < teams[j] })
	out := make([]scoringEntry, 0, len(teams))
	for _, t := range teams {
		out = append(out, scoringEntry{
			Season: season, Competition: competition, TeamNumber: int(t) + offset,
			Instructions: strings.TrimSpace(instructions), Hosts: byTeam[t],
		})
	}
	return out
}

func splitList(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func countTrue(bs ...bool) int {
	n := 0
	for _, b := range bs {
		if b {
			n++
		}
	}
	return n
}
