// laforge is the CLI foundation:
// "laforge context, laforge render, and laforge check against a checkout,
// since authors need them from the first content they write, and check is
// the same code the server runs per commit."
//
// This is deliberately still small. The rest of the real CLI (build,
// deploy, the GitHub-backed commands) comes with the services, which
// don't exist yet -- these three commands work entirely
// against a local checkout, no server involved, matching "in a second,
// with no build and no infrastructure."
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/globalcptc/laforge/internal/hclconvert"
	"github.com/globalcptc/laforge/internal/loader"
	"github.com/globalcptc/laforge/internal/render"
)

// version is the CLI's build version, stamped in at release time via
// -ldflags "-X main.version=<tag>" (see the Makefile). "dev" for a plain
// `go build`/`go install`.
var version = "dev"

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "version", "-v", "--version":
		fmt.Println("laforge", version)
		return
	case "context":
		err = runContext(os.Args[2:])
	case "render":
		err = runRender(os.Args[2:])
	case "check":
		err = runCheck(os.Args[2:])
	case "convert":
		err = runConvert(os.Args[2:])
	case "login":
		err = runLogin(os.Args[2:])
	case "repo":
		err = runRepo(os.Args[2:])
	case "build":
		err = runBuild(os.Args[2:])
	case "builders":
		err = runBuilders(os.Args[2:])
	case "images":
		err = runImages(os.Args[2:])
	case "registries":
		err = runRegistries(os.Args[2:])
	case "-h", "--help", "help":
		usage()
		return
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n", os.Args[1])
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `laforge -- CLI foundations

  laforge context <environment> --host <as> --team <n> [--repo <path>]
      Print the full template context for one host/container copy, as YAML.

  laforge render <script-source-path> --host <as> --team <n> [--environment <name>] [--repo <path>]
      Render one script's real source file against a resolved context and print it.

  laforge check [--repo <path>]
      Validate and render everything: every file against its schema, every
      cross-file reference, and every script against every host in every
      team. Exit code is non-zero if anything failed.

  laforge convert <old-hcl-repo> <new-repo-out>
      One-shot: convert an existing HCL content repository into the new
      YAML format. Prints every judgment call and everything it could not
      translate, then exits non-zero if anything needs manual review.

Server-backed commands: talk to a running laforge-api (LAFORGE_API_URL, default
http://localhost:8080).

  laforge login
      Sign in via GitHub device flow. Saves the token to ~/.laforge/token
      for the commands below.

  laforge repo add <owner>/<repo>
  laforge repo list
      Register a repository, or list registered ones.

  laforge build configure <owner>/<repo> --branch <b> --env <path> --builder <name>
  laforge build list <owner>/<repo>
  laforge build status <configured-build-id>
  laforge build follow <configured-build-id> <on|off>
  laforge build lock <configured-build-id> <on|off>
      Configure a build (branch + environment file + builder config),
      inspect it, and toggle follow mode / the competition-started lock.

  laforge builders
      List every configured builder: name, kind, hoster URL, and image count.

  laforge images <builder> [--available]
      List the os -> image mappings a builder provides (the names content's
      os: field must match) -- use it to diagnose a "builder has no image for
      X" deploy block. With --available, instead list every image the builder's
      hoster actually holds, discovered live: what you could map an os to.

  laforge registries
      List the stored private-registry credentials (host + username).

  laforge version
      Print the build version.`)
}

func repoFlag(fs *flag.FlagSet) *string {
	return fs.String("repo", ".", "path to a content repository checkout")
}

// splitArgs separates positional arguments from flags, because Go's
// stdlib flag package stops parsing at the first non-flag token -- it
// cannot handle the documented invocation form,
// "laforge context lm-test --host web01 --team 3", where the positional
// environment name comes *before* the flags. Every flag used by these
// three commands takes a value (none are booleans), so each recognized
// flag token consumes the next token as its value; everything else is
// positional.
func splitArgs(args []string, flagNames map[string]bool) (positionals, flagArgs []string) {
	for i := 0; i < len(args); i++ {
		a := args[i]
		name := strings.TrimLeft(a, "-")
		if eq := strings.IndexByte(name, '='); eq >= 0 {
			name = name[:eq]
		}
		if strings.HasPrefix(a, "-") && flagNames[name] {
			flagArgs = append(flagArgs, a)
			if !strings.Contains(a, "=") && i+1 < len(args) {
				i++
				flagArgs = append(flagArgs, args[i])
			}
			continue
		}
		positionals = append(positionals, a)
	}
	return positionals, flagArgs
}

func runContext(args []string) error {
	fs := flag.NewFlagSet("context", flag.ExitOnError)
	repo := repoFlag(fs)
	host := fs.String("host", "", "the `as` name of the host/container copy (required)")
	team := fs.Int("team", 0, "team number, 1-indexed (required)")
	positionals, flagArgs := splitArgs(args, map[string]bool{"repo": true, "host": true, "team": true})
	fs.Parse(flagArgs)

	if len(positionals) != 1 {
		return fmt.Errorf("usage: laforge context <environment> --host <as> --team <n>")
	}
	if *host == "" || *team == 0 {
		return fmt.Errorf("--host and --team are required")
	}
	envName := positionals[0]

	c, err := loadOrExplain(*repo)
	if err != nil {
		return err
	}
	ctx, err := render.Resolve(c, envName, *host, *team)
	if err != nil {
		return err
	}

	out := render.NewContextView(ctx)
	enc := yaml.NewEncoder(os.Stdout)
	enc.SetIndent(2)
	defer enc.Close()
	return enc.Encode(out)
}

func runRender(args []string) error {
	fs := flag.NewFlagSet("render", flag.ExitOnError)
	repo := repoFlag(fs)
	host := fs.String("host", "", "the `as` name of the host/container copy (required)")
	team := fs.Int("team", 0, "team number, 1-indexed (required)")
	envFlag := fs.String("environment", "", "environment name (required if the repo has more than one)")
	positionals, flagArgs := splitArgs(args, map[string]bool{"repo": true, "host": true, "team": true, "environment": true})
	fs.Parse(flagArgs)

	if len(positionals) != 1 {
		return fmt.Errorf("usage: laforge render <script-source-path> --host <as> --team <n>")
	}
	if *host == "" || *team == 0 {
		return fmt.Errorf("--host and --team are required")
	}
	scriptPath := positionals[0]

	c, err := loadOrExplain(*repo)
	if err != nil {
		return err
	}
	envName, err := resolveEnvironmentName(c, *envFlag)
	if err != nil {
		return err
	}

	script, err := findScriptByPathOrName(*repo, c, scriptPath)
	if err != nil {
		return err
	}

	ctx, err := render.Resolve(c, envName, *host, *team)
	if err != nil {
		return err
	}
	out, err := render.RenderScript(*repo, script, ctx, c)
	if err != nil {
		return err
	}
	fmt.Print(out)
	return nil
}

// findScriptByPathOrName supports the exact form the documented
// example uses ("laforge render scripts/create-linux-users.sh ...", a path
// to the real source file), with a fallback to the script's logical name
// (its `script:` header value) for convenience -- the example doesn't specify
// which form the CLI should accept, so both are supported rather than
// guessing wrong.
func findScriptByPathOrName(repo string, c *loader.Content, arg string) (*loader.Script, error) {
	for i := range c.Scripts {
		s := &c.Scripts[i]
		full := filepath.Join(filepath.Dir(s.SourceFile), s.Source)
		if full == arg || filepath.Base(full) == arg {
			return s, nil
		}
	}
	for i := range c.Scripts {
		if c.Scripts[i].Name == arg {
			return &c.Scripts[i], nil
		}
	}
	return nil, fmt.Errorf("no script matches %q (tried matching it as a source file path and as a script name)", arg)
}

func resolveEnvironmentName(c *loader.Content, flagVal string) (string, error) {
	if flagVal != "" {
		return flagVal, nil
	}
	if len(c.Environments) == 1 {
		return c.Environments[0].Name, nil
	}
	var names []string
	for _, e := range c.Environments {
		names = append(names, e.Name)
	}
	sort.Strings(names)
	return "", fmt.Errorf("this repository has %d environments (%v); pass --environment to pick one", len(c.Environments), names)
}

func runCheck(args []string) error {
	fs := flag.NewFlagSet("check", flag.ExitOnError)
	repo := repoFlag(fs)
	fs.Parse(args)

	c, err := loader.Load(*repo)
	if err != nil {
		return err
	}
	fmt.Println(c.Summary())
	for _, e := range c.Errors {
		if e.Line > 0 {
			fmt.Printf("%s:%d: %s\n", e.File, e.Line, e.Message)
		} else {
			fmt.Printf("%s: %s\n", e.File, e.Message)
		}
	}
	if len(c.Errors) > 0 {
		return fmt.Errorf("%d schema/reference error(s), not rendering until these are fixed", len(c.Errors))
	}

	renderErrs := render.CheckAll(*repo, c)
	for _, e := range renderErrs {
		fmt.Println(e.String())
	}
	if len(renderErrs) > 0 {
		return fmt.Errorf("%d render error(s)", len(renderErrs))
	}
	fmt.Println("check passed: every script rendered clean for every host in every team")
	return nil
}

func runConvert(args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("usage: laforge convert <old-hcl-repo> <new-repo-out>")
	}
	src, dst := args[0], args[1]

	result, err := hclconvert.Convert(src)
	if err != nil {
		return err
	}
	if err := hclconvert.Write(result, dst); err != nil {
		return err
	}

	warn, info := 0, 0
	for _, n := range result.Notes {
		fmt.Printf("[%s] %s: %s\n", n.Severity, n.File, n.Message)
		if n.Severity == "warn" {
			warn++
		} else {
			info++
		}
	}
	fmt.Printf("\nconverted: %d networks, %d hosts, %d scripts, %d environments, %d people sources\n",
		len(result.Networks), len(result.Hosts), len(result.Scripts), len(result.Environments), len(result.People))
	fmt.Printf("%d note(s): %d for review (warn), %d informational\n", len(result.Notes), warn, info)

	if warn > 0 {
		return fmt.Errorf("%d item(s) need manual review, see [warn] lines above", warn)
	}
	return nil
}

func loadOrExplain(repo string) (*loader.Content, error) {
	c, err := loader.Load(repo)
	if err != nil {
		return nil, err
	}
	if len(c.Errors) > 0 {
		for _, e := range c.Errors {
			fmt.Fprintf(os.Stderr, "%s: %s\n", e.File, e.Message)
		}
		return nil, fmt.Errorf("%d schema/reference error(s) in %s -- run `laforge check` for details", len(c.Errors), repo)
	}
	return c, nil
}
