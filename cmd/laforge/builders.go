package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/globalcptc/laforge/internal/db"
)

// builderImage is one os -> image mapping inside a builder config's
// incus_images, mirroring the shape the server stores (see
// internal/builder/incus/config.go).
type builderImage struct {
	VM          bool   `json:"vm"`
	Alias       string `json:"alias"`
	Server      string `json:"server"`
	Protocol    string `json:"protocol"`
	Fingerprint string `json:"fingerprint"`
}

func decodeImages(raw json.RawMessage) map[string]builderImage {
	imgs := map[string]builderImage{}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &imgs)
	}
	return imgs
}

// runBuilders lists every configured builder: name, kind, the hoster it talks
// to, and how many os images it has mapped.
func runBuilders(args []string) error {
	c := newAPIClient()
	var cfgs []db.BuilderConfig
	if err := c.do("GET", "/builder-configs", nil, &cfgs); err != nil {
		return err
	}
	if len(cfgs) == 0 {
		fmt.Println("no builders configured")
		return nil
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tKIND\tAPI URL\tIMAGES")
	for _, cfg := range cfgs {
		apiURL := ""
		if cfg.IncusApiUrl != nil {
			apiURL = *cfg.IncusApiUrl
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%d\n", cfg.Name, cfg.Kind, apiURL, len(decodeImages(cfg.IncusImages)))
	}
	return w.Flush()
}

// runImages lists the os -> image mappings a builder provides -- the same names
// content's `os:` must match. Handy for diagnosing a "builder has no image for
// X" deploy block: it shows exactly which os names this builder actually maps.
func runImages(args []string) error {
	available := false
	name := ""
	for _, a := range args {
		switch {
		case a == "--available" || a == "-a":
			available = true
		case !strings.HasPrefix(a, "-"):
			name = a
		}
	}
	if name == "" {
		return fmt.Errorf("usage: laforge images <builder> [--available]")
	}
	c := newAPIClient()
	if available {
		return listAvailableImages(c, name)
	}

	var cfg db.BuilderConfig
	if err := c.do("GET", "/builder-configs/"+url.PathEscape(name), nil, &cfg); err != nil {
		return err
	}
	imgs := decodeImages(cfg.IncusImages)
	if len(imgs) == 0 {
		fmt.Printf("builder %q has no images configured\n", cfg.Name)
		return nil
	}
	names := make([]string, 0, len(imgs))
	for name := range imgs {
		names = append(names, name)
	}
	sort.Strings(names)

	w := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
	fmt.Fprintf(w, "OS\tTYPE\tIMAGE\n")
	for _, name := range names {
		im := imgs[name]
		typ := "container"
		if im.VM {
			typ = "vm"
		}
		ref := im.Alias
		if ref == "" {
			ref = im.Fingerprint
		}
		if im.Server != "" {
			ref = im.Server + "/" + ref
		}
		fmt.Fprintf(w, "%s\t%s\t%s\n", name, typ, ref)
	}
	return w.Flush()
}

// listAvailableImages shows what the builder's hoster actually holds -- the
// images you could map an os to -- by asking the server to re-discover them
// live over the builder's stored connection.
func listAvailableImages(c *apiClient, name string) error {
	var disc struct {
		Images []struct {
			Fingerprint  string   `json:"fingerprint"`
			Aliases      []string `json:"aliases"`
			Architecture string   `json:"architecture"`
			Type         string   `json:"type"`
			Properties   struct {
				Description string `json:"description"`
			} `json:"properties"`
		} `json:"images"`
	}
	if err := c.do("GET", "/builder-configs/"+url.PathEscape(name)+"/available-images", nil, &disc); err != nil {
		return err
	}
	if len(disc.Images) == 0 {
		fmt.Println("no images found on the hoster")
		return nil
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
	fmt.Fprintln(w, "ALIAS\tTYPE\tARCH\tFINGERPRINT\tDESCRIPTION")
	for _, im := range disc.Images {
		alias := "(none)"
		if len(im.Aliases) > 0 {
			alias = strings.Join(im.Aliases, ",")
		}
		typ := "container"
		if im.Type == "virtual-machine" {
			typ = "vm"
		}
		fp := im.Fingerprint
		if len(fp) > 12 {
			fp = fp[:12]
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", alias, typ, im.Architecture, fp, im.Properties.Description)
	}
	return w.Flush()
}

// runRegistries lists the stored private-registry credentials (host + user;
// the secret is never returned by the server).
func runRegistries(args []string) error {
	c := newAPIClient()
	var creds []struct {
		RegistryHost string `json:"registry_host"`
		Username     string `json:"username"`
	}
	if err := c.do("GET", "/registry-credentials", nil, &creds); err != nil {
		return err
	}
	if len(creds) == 0 {
		fmt.Println("no registry credentials configured")
		return nil
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
	fmt.Fprintln(w, "REGISTRY HOST\tUSERNAME")
	for _, cr := range creds {
		fmt.Fprintf(w, "%s\t%s\n", cr.RegistryHost, cr.Username)
	}
	return w.Flush()
}
