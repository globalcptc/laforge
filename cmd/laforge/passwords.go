package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"
)

// passwordRow mirrors internal/api.buildCredentialRow: one account on one
// host, with its current password as LaForge set it.
type passwordRow struct {
	Team     int32  `json:"team"`
	Host     string `json:"host"`
	Username string `json:"username"`
	Password string `json:"password"`
	Applied  bool   `json:"applied"`
	Admin    bool   `json:"admin"`
	Source   string `json:"source"`
}

// runPasswords lists every account LaForge has set a password for across a
// build's hosts -- the automatic admin step and every set_password /
// create_user, generated or written -- filtered by team and/or host. Pending
// marks a password whose step hasn't run on the box yet.
func runPasswords(args []string) error {
	fs := flag.NewFlagSet("passwords", flag.ExitOnError)
	build := fs.String("build", "", "build id (required)")
	team := fs.Int("team", 0, "only this team")
	host := fs.String("host", "", "only this host (its \"as\" name)")
	asCSV := fs.Bool("csv", false, "output CSV")
	asJSON := fs.Bool("json", false, "output JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *build == "" {
		return fmt.Errorf("usage: laforge passwords --build <id> [--team <n>] [--host <as>] [--csv | --json]")
	}
	if *asCSV && *asJSON {
		return fmt.Errorf("choose one of --csv or --json, not both")
	}
	var all []passwordRow
	if err := newAPIClient().do("GET", "/builds/"+url.PathEscape(*build)+"/credentials", nil, &all); err != nil {
		return err
	}
	rows := all[:0]
	for _, r := range all {
		if (*team == 0 || int(r.Team) == *team) && (*host == "" || strings.EqualFold(r.Host, *host)) {
			rows = append(rows, r)
		}
	}
	switch {
	case *asJSON:
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(rows)
	case *asCSV:
		w := csv.NewWriter(os.Stdout)
		_ = w.Write([]string{"team", "host", "username", "password", "applied", "admin"})
		for _, r := range rows {
			_ = w.Write([]string{strconv.Itoa(int(r.Team)), r.Host, r.Username, r.Password, strconv.FormatBool(r.Applied), strconv.FormatBool(r.Admin)})
		}
		w.Flush()
		return w.Error()
	}
	if len(rows) == 0 {
		fmt.Println("no passwords recorded for the matching hosts")
		return nil
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
	fmt.Fprintln(w, "TEAM\tHOST\tUSER\tPASSWORD\tSTATE")
	for _, r := range rows {
		state := "applied"
		if !r.Applied {
			state = "pending"
		}
		user := r.Username
		if r.Admin {
			user += " (admin)"
		}
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\n", r.Team, r.Host, user, r.Password, state)
	}
	return w.Flush()
}
