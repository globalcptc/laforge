package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"text/tabwriter"
)

// accessExportRow is one line of the external-access export (where to connect +
// how to log in), mirroring internal/api.externalAccessExportRow.
type accessExportRow struct {
	Team     int32  `json:"team"`
	Host     string `json:"host"`
	IP       string `json:"ip"`
	Port     string `json:"port"`
	Protocol string `json:"protocol"`
	Username string `json:"username"`
	Password string `json:"password"`
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
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *build == "" {
		return fmt.Errorf("usage: laforge access --build <id> [--csv | --json]")
	}
	if *asCSV && *asJSON {
		return fmt.Errorf("choose one of --csv or --json, not both")
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
