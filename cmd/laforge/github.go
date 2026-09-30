// This file is the CLI's additions: `laforge login` (GitHub
// device-flow sign-in), `laforge repo`, and `laforge build`, talking to
// the laforge-api service over HTTP via apiclient.go.
package main

import (
	"context"
	"flag"
	"fmt"
	"strings"

	"github.com/globalcptc/laforge/internal/db"
	"github.com/globalcptc/laforge/internal/ghclient"
)

// runLogin fetches the App's client id from laforge-api itself (GET
// /auth/client-id -- not a secret, see that handler's own doc comment)
// rather than requiring it set locally: the CLI already needs to know
// the API's URL for every other command (LAFORGE_API_URL), and that's
// the only thing a person running `laforge login` should ever need to
// know, not an internal GitHub App identifier.
func runLogin(args []string) error {
	c := newAPIClient()
	var clientIDResp struct {
		ClientID string `json:"client_id"`
	}
	if err := c.do("GET", "/auth/client-id", nil, &clientIDResp); err != nil {
		return fmt.Errorf("fetching the GitHub App's client id from %s: %w\n\n(is laforge-api running and reachable? see LAFORGE_API_URL. If it's running but this still fails, no GitHub App is configured on that server yet)", c.baseURL, err)
	}
	clientID := clientIDResp.ClientID

	gh := ghclient.New()
	ctx := context.Background()
	dc, err := gh.RequestDeviceCode(ctx, clientID, []string{"repo"})
	if err != nil {
		return fmt.Errorf("requesting device code: %w", err)
	}

	fmt.Printf("First, go to %s\n", dc.VerificationURI)
	fmt.Printf("Then enter this code: %s\n\n", dc.UserCode)
	fmt.Println("Waiting for you to authorize (this will wait up to a few minutes)...")

	tok, err := gh.PollForToken(ctx, clientID, dc.DeviceCode, dc.Interval, dc.ExpiresIn)
	if err != nil {
		return fmt.Errorf("waiting for authorization: %w", err)
	}
	if err := writeStoredToken(tok); err != nil {
		return fmt.Errorf("saving token: %w", err)
	}

	if user, err := gh.GetAuthenticatedUser(ctx, tok); err == nil {
		fmt.Printf("Logged in as %s.\n", user.Login)
	} else {
		fmt.Println("Logged in.")
	}
	return nil
}

func runRepo(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: laforge repo <add|list> ...")
	}
	c := newAPIClient()
	switch args[0] {
	case "add":
		if len(args) != 2 {
			return fmt.Errorf("usage: laforge repo add <owner>/<repo>")
		}
		owner, name, ok := strings.Cut(args[1], "/")
		if !ok {
			return fmt.Errorf("expected <owner>/<repo>, got %q", args[1])
		}
		var repo db.Repository
		if err := c.do("POST", "/repos", map[string]string{"owner": owner, "repo": name}, &repo); err != nil {
			return err
		}
		fmt.Printf("registered %s/%s (id %s)\n", repo.GithubOwner, repo.GithubRepo, repo.ID.String())
		return nil
	case "list":
		var repos []db.Repository
		if err := c.do("GET", "/repos", nil, &repos); err != nil {
			return err
		}
		if len(repos) == 0 {
			fmt.Println("no repositories registered")
			return nil
		}
		for _, r := range repos {
			fmt.Printf("%s  %s/%s\n", r.ID.String(), r.GithubOwner, r.GithubRepo)
		}
		return nil
	default:
		return fmt.Errorf("unknown repo subcommand %q (want add or list)", args[0])
	}
}

// resolveRepoID finds a registered repository's id by "<owner>/<repo>",
// since the API only takes ids on configured-build routes -- the CLI is
// where the friendlier owner/repo form lives.
func resolveRepoID(c *apiClient, ownerRepo string) (db.Repository, error) {
	owner, name, ok := strings.Cut(ownerRepo, "/")
	if !ok {
		return db.Repository{}, fmt.Errorf("expected <owner>/<repo>, got %q", ownerRepo)
	}
	var repos []db.Repository
	if err := c.do("GET", "/repos", nil, &repos); err != nil {
		return db.Repository{}, err
	}
	for _, r := range repos {
		if r.GithubOwner == owner && r.GithubRepo == name {
			return r, nil
		}
	}
	return db.Repository{}, fmt.Errorf("no registered repository %q -- run `laforge repo add %s` first", ownerRepo, ownerRepo)
}

func runBuild(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: laforge build <configure|list|status|auto-deploy|lock> ...")
	}
	c := newAPIClient()
	switch args[0] {
	case "configure":
		return runBuildConfigure(c, args[1:])
	case "list":
		return runBuildList(c, args[1:])
	case "status":
		return runBuildStatus(c, args[1:])
	case "auto-deploy":
		return runBuildSetAutoDeploy(c, args[1:])
	case "lock":
		return runBuildSetLock(c, args[1:])
	default:
		return fmt.Errorf("unknown build subcommand %q (want configure, list, status, auto-deploy, or lock)", args[0])
	}
}

func runBuildConfigure(c *apiClient, args []string) error {
	fs := flag.NewFlagSet("build configure", flag.ExitOnError)
	branch := fs.String("branch", "", "branch to track (required)")
	envPath := fs.String("env", "", "environment file path within the repo (required)")
	builder := fs.String("builder", "", "builder config name (required)")
	positionals, flagArgs := splitArgs(args, map[string]bool{"branch": true, "env": true, "builder": true})
	fs.Parse(flagArgs)
	if len(positionals) != 1 {
		return fmt.Errorf("usage: laforge build configure <owner>/<repo> --branch <b> --env <path> --builder <name>")
	}
	if *branch == "" || *envPath == "" || *builder == "" {
		return fmt.Errorf("--branch, --env, and --builder are all required")
	}

	repo, err := resolveRepoID(c, positionals[0])
	if err != nil {
		return err
	}
	var cb db.ConfiguredBuild
	err = c.do("POST", "/repos/"+repo.ID.String()+"/configured-builds", map[string]string{
		"branch": *branch, "environment_path": *envPath, "builder_config_name": *builder,
	}, &cb)
	if err != nil {
		return err
	}
	fmt.Printf("configured build %s: %s/%s@%s -> %s (builder %s), auto_deploy=%v\n",
		cb.ID.String(), repo.GithubOwner, repo.GithubRepo, cb.Branch, cb.EnvironmentPath, cb.BuilderConfigName, cb.AutoDeployEnabled)
	return nil
}

func runBuildList(c *apiClient, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: laforge build list <owner>/<repo>")
	}
	repo, err := resolveRepoID(c, args[0])
	if err != nil {
		return err
	}
	var builds []db.ConfiguredBuild
	if err := c.do("GET", "/repos/"+repo.ID.String()+"/configured-builds", nil, &builds); err != nil {
		return err
	}
	if len(builds) == 0 {
		fmt.Println("no configured builds for this repository")
		return nil
	}
	for _, cb := range builds {
		printBuildStatus(cb)
	}
	return nil
}

func runBuildStatus(c *apiClient, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: laforge build status <configured-build-id>")
	}
	var cb db.ConfiguredBuild
	if err := c.do("GET", "/configured-builds/"+args[0], nil, &cb); err != nil {
		return err
	}
	printBuildStatus(cb)
	if cb.CurrentContentRevisionID.Valid {
		var envs []db.Environment
		if err := c.do("GET", "/content-revisions/"+cb.CurrentContentRevisionID.String()+"/environments", nil, &envs); err == nil {
			for _, e := range envs {
				fmt.Printf("  environment: %s (%d teams)\n", e.Name, e.Teams)
			}
		}
	}
	return nil
}

func printBuildStatus(cb db.ConfiguredBuild) {
	rev := "none yet"
	if cb.CurrentContentRevisionID.Valid {
		rev = cb.CurrentContentRevisionID.String()
	}
	fmt.Printf("%s  branch=%s  env=%s  builder=%s  auto_deploy=%v  competition_started=%v  current_revision=%s\n",
		cb.ID.String(), cb.Branch, cb.EnvironmentPath, cb.BuilderConfigName, cb.AutoDeployEnabled, cb.CompetitionStarted, rev)
}

func runBuildSetAutoDeploy(c *apiClient, args []string) error {
	id, on, err := parseOnOffArgs("build auto-deploy", args)
	if err != nil {
		return err
	}
	var cb db.ConfiguredBuild
	if err := c.do("POST", "/configured-builds/"+id+"/auto-deploy", map[string]bool{"enabled": on}, &cb); err != nil {
		return err
	}
	printBuildStatus(cb)
	return nil
}

func runBuildSetLock(c *apiClient, args []string) error {
	id, on, err := parseOnOffArgs("build lock", args)
	if err != nil {
		return err
	}
	var cb db.ConfiguredBuild
	if err := c.do("POST", "/configured-builds/"+id+"/lock", map[string]bool{"started": on}, &cb); err != nil {
		return err
	}
	printBuildStatus(cb)
	return nil
}

func parseOnOffArgs(cmd string, args []string) (id string, on bool, err error) {
	if len(args) != 2 {
		return "", false, fmt.Errorf("usage: laforge %s <configured-build-id> <on|off>", cmd)
	}
	switch args[1] {
	case "on":
		on = true
	case "off":
		on = false
	default:
		return "", false, fmt.Errorf("expected \"on\" or \"off\", got %q", args[1])
	}
	return args[0], on, nil
}
