package main

import (
	"fmt"
	"os"
	"text/tabwriter"
)

// cliCertInfo mirrors internal/api.certInfo (GET /cert-status): expiry metadata
// only, no key material.
type cliCertInfo struct {
	Configured    bool    `json:"configured"`
	Subject       string  `json:"subject"`
	NotAfter      *string `json:"not_after"`
	DaysRemaining int     `json:"days_remaining"`
	ExpiringSoon  bool    `json:"expiring_soon"`
	Expired       bool    `json:"expired"`
	Error         string  `json:"error"`
}

// runCertStatus prints the agent mTLS CA and gateway certificate expiry so an
// operator (or a monitoring cron) can see the agent trust anchor coming up for
// renewal. Exits non-zero if either cert is expired or within the warning
// window, so it drops straight into a check script.
func runCertStatus(args []string) error {
	var st struct {
		WarnThresholdDays int         `json:"warn_threshold_days"`
		CA                cliCertInfo `json:"ca"`
		Server            cliCertInfo `json:"server"`
	}
	if err := newAPIClient().do("GET", "/cert-status", nil, &st); err != nil {
		return err
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
	fmt.Fprintln(w, "CERT\tSUBJECT\tEXPIRES\tDAYS LEFT\tSTATUS")
	printCert(w, "agent CA", st.CA)
	printCert(w, "gateway", st.Server)
	w.Flush()

	if st.CA.Expired || st.Server.Expired {
		return fmt.Errorf("a certificate has expired -- renew it (scripts/gen-certs.sh); agents cannot connect until you do")
	}
	if st.CA.ExpiringSoon || st.Server.ExpiringSoon {
		return fmt.Errorf("a certificate expires within %d days -- renew it before then", st.WarnThresholdDays)
	}
	return nil
}

func printCert(w *tabwriter.Writer, label string, ci cliCertInfo) {
	switch {
	case !ci.Configured:
		fmt.Fprintf(w, "%s\t—\t—\t—\tnot configured\n", label)
	case ci.Error != "":
		fmt.Fprintf(w, "%s\t—\t—\t—\terror: %s\n", label, ci.Error)
	default:
		status := "ok"
		if ci.Expired {
			status = "EXPIRED"
		} else if ci.ExpiringSoon {
			status = "EXPIRING SOON"
		}
		exp := "—"
		if ci.NotAfter != nil {
			exp = *ci.NotAfter
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\n", label, ci.Subject, exp, ci.DaysRemaining, status)
	}
}
