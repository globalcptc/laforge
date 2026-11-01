package mcp

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// Tool inputs shared by the page tools.

type noArgsIn struct {
	fieldChars
}

type repoPageIn struct {
	Repository string `json:"repository" jsonschema:"the repository, as owner/name or its id"`
	fieldChars
}

type buildPageIn struct {
	BuildID string `json:"build_id" jsonschema:"the build's id, or a unique prefix of it such as 9531b5b4"`
	fieldChars
}

type objectPageIn struct {
	BuildID string `json:"build_id" jsonschema:"the build's id, or a unique prefix of it such as 9531b5b4"`
	Object  string `json:"object" jsonschema:"the object's id (or id prefix), or its name such as dc01"`
	Team    int    `json:"team,omitempty" jsonschema:"the team number, when the name exists in several teams"`
	fieldChars
}

type builderPageIn struct {
	Builder string `json:"builder" jsonschema:"the builder (infrastructure) name, as list_builders shows it"`
	fieldChars
}

// buildPath checks the build and returns /builds/{id}.
func (c *call) buildPath(ctx context.Context, ref string) (string, error) {
	id, err := c.build(ctx, ref)
	if err != nil {
		return "", err
	}
	return "/builds/" + id.String(), nil
}

// objectPath checks the build and object and returns /builds/{id}/objects/{objectId}.
func (c *call) objectPath(ctx context.Context, in objectPageIn) (string, error) {
	id, err := c.build(ctx, in.BuildID)
	if err != nil {
		return "", err
	}
	obj, err := c.object(ctx, id, in.Object, in.Team)
	if err != nil {
		return "", err
	}
	return "/builds/" + id.String() + "/objects/" + obj.String(), nil
}

func buildPage(suffix string) func(context.Context, *call, buildPageIn) (string, url.Values, int, error) {
	return func(ctx context.Context, c *call, in buildPageIn) (string, url.Values, int, error) {
		p, err := c.buildPath(ctx, in.BuildID)
		return p + suffix, nil, in.limit(), err
	}
}

func objectPage(suffix string) func(context.Context, *call, objectPageIn) (string, url.Values, int, error) {
	return func(ctx context.Context, c *call, in objectPageIn) (string, url.Values, int, error) {
		p, err := c.objectPath(ctx, in)
		return p + suffix, nil, in.limit(), err
	}
}

func repoPage(suffix string) func(context.Context, *call, repoPageIn) (string, url.Values, int, error) {
	return func(ctx context.Context, c *call, in repoPageIn) (string, url.Values, int, error) {
		id, err := c.repoID(ctx, in.Repository)
		return "/repos/" + id + suffix, nil, in.limit(), err
	}
}

func instancePage(path string) func(context.Context, *call, noArgsIn) (string, url.Values, int, error) {
	return func(_ context.Context, _ *call, in noArgsIn) (string, url.Values, int, error) {
		return path, nil, in.limit(), nil
	}
}

func builderPage(suffix string) func(context.Context, *call, builderPageIn) (string, url.Values, int, error) {
	return func(_ context.Context, _ *call, in builderPageIn) (string, url.Values, int, error) {
		if strings.TrimSpace(in.Builder) == "" {
			return "", nil, 0, errors.New("builder is required")
		}
		return "/builder-configs/" + seg(in.Builder) + suffix, nil, in.limit(), nil
	}
}

// --- inputs with extra options ---

type buildEventsIn struct {
	BuildID string `json:"build_id" jsonschema:"the build's id, or a unique prefix of it such as 9531b5b4"`
	Limit   int    `json:"limit,omitempty" jsonschema:"how many events, newest first (default 200, at most 2000)"`
	Before  string `json:"before,omitempty" jsonschema:"only events before this time (RFC 3339), to page back through older events"`
	fieldChars
}

type reportIn struct {
	BuildID  string `json:"build_id" jsonschema:"the build's id, or a unique prefix of it such as 9531b5b4"`
	ReportID string `json:"report_id" jsonschema:"the validation report's id, from list_validation_reports"`
	fieldChars
}

type scheduledRunsIn struct {
	BuildID string `json:"build_id" jsonschema:"the build's id, or a unique prefix of it such as 9531b5b4"`
	TaskID  string `json:"scheduled_task_id" jsonschema:"the scheduled task's id, from list_scheduled_tasks"`
	fieldChars
}

type peopleIn struct {
	Repository string `json:"repository" jsonschema:"the repository, as owner/name or its id"`
	Query      string `json:"query,omitempty" jsonschema:"only people matching this text"`
	RevisionID string `json:"revision_id,omitempty" jsonschema:"a content revision id; default is the latest"`
	fieldChars
}

type revisionIn struct {
	RevisionID string `json:"revision_id" jsonschema:"a content revision's id, from list_content_revisions or get_build"`
	fieldChars
}

type configuredBuildIn struct {
	ConfiguredBuildID string `json:"configured_build_id" jsonschema:"the configured build's id, from list_configured_builds"`
	fieldChars
}

type builderStatsIn struct {
	Builder string `json:"builder" jsonschema:"the builder (infrastructure) name, as list_builders shows it"`
	Window  string `json:"window,omitempty" jsonschema:"how far back, as a Go duration such as 30m, 6h or 24h (default 6h)"`
	fieldChars
}

type imageBuildLogIn struct {
	ImageBuildID string `json:"image_build_id" jsonschema:"the image build's id, from list_image_builds"`
	Since        int    `json:"since,omitempty" jsonschema:"only log lines after this sequence number"`
	fieldChars
}

func parseID(s, what string) (pgtype.UUID, error) {
	var id pgtype.UUID
	if err := id.Scan(strings.TrimSpace(s)); err != nil {
		return id, fmt.Errorf("%s %q isn't a valid id", what, s)
	}
	return id, nil
}

func (s *Server) registerPageTools() {
	// --- overview and repositories ---
	addPage(s, "home", "Home",
		"Your LaForge home page: active builds across your repositories, with what needs attention (failures, agents lost, stuck deploys).",
		instancePage("/home"))
	addPage(s, "list_configured_builds", "List Configured Builds",
		"A repository's configured builds: branch, environment file, builder, follow mode, auto-deploy, the competition-started lock, and the commit each tracks.",
		repoPage("/configured-builds"))
	addPage(s, "get_configured_build", "Get Configured Build",
		"One configured build's settings and the commit it currently tracks.",
		func(ctx context.Context, c *call, in configuredBuildIn) (string, url.Values, int, error) {
			id, err := parseID(in.ConfiguredBuildID, "configured_build_id")
			if err != nil {
				return "", nil, 0, err
			}
			if err := c.checkRepo(c.q().McpGetConfiguredBuildRepository(ctx, id)); err != nil {
				return "", nil, 0, err
			}
			return "/configured-builds/" + id.String(), nil, in.limit(), nil
		})
	addPage(s, "list_revision_environments", "List Environments In Revision",
		"The environments a content revision (commit) defines: name, file, team count, schedule.",
		func(ctx context.Context, c *call, in revisionIn) (string, url.Values, int, error) {
			id, err := parseID(in.RevisionID, "revision_id")
			if err != nil {
				return "", nil, 0, err
			}
			if err := c.checkRepo(c.q().McpGetRevisionRepository(ctx, id)); err != nil {
				return "", nil, 0, err
			}
			return "/content-revisions/" + id.String() + "/environments", nil, in.limit(), nil
		})
	addPage(s, "list_people", "List People",
		"The people (from the repository's people CSV files) a repository's content can reference, with all their attributes.",
		func(ctx context.Context, c *call, in peopleIn) (string, url.Values, int, error) {
			id, err := c.repoID(ctx, in.Repository)
			if err != nil {
				return "", nil, 0, err
			}
			q := url.Values{}
			if in.Query != "" {
				q.Set("q", in.Query)
			}
			if in.RevisionID != "" {
				q.Set("revision", in.RevisionID)
			}
			return "/repos/" + id + "/people", q, in.limit(), nil
		})
	addPage(s, "list_repo_access", "List Repository Access",
		"Who has access to a repository and at what level (GitHub role and any LaForge override). Needs instance admin.",
		repoPage("/access"))
	addPage(s, "my_repo_access", "My Repository Access",
		"Your own access level on a repository and where it comes from.",
		repoPage("/my-access"))

	// --- one build ---
	addPage(s, "get_build_detail", "Get Build Detail",
		"The build page's full data: the build, its environment, teams and every deployed object with agent health. (get_build is the short summary.)",
		buildPage(""))
	addPage(s, "get_dashboard", "Get Build Dashboard",
		"The build dashboard: status matrix by team and host, failures, and the agent check-in time series.",
		buildPage("/dashboard"))
	addPage(s, "get_topology", "Get Topology",
		"The build's network topology per team: networks with their CIDRs and the hosts and containers on each (OS, image, size, dependencies, tags, status).",
		buildPage("/topology"))
	addPage(s, "list_build_events", "List Build Events (Logs)",
		"The build's event log, newest first: deploys, builder requests and their results, step starts and finishes, failures, operator actions. Page back with before.",
		func(ctx context.Context, c *call, in buildEventsIn) (string, url.Values, int, error) {
			p, err := c.buildPath(ctx, in.BuildID)
			q := url.Values{"limit": {fmt.Sprint(clamp(in.Limit, 200, 2000))}}
			if in.Before != "" {
				q.Set("before", in.Before)
			}
			return p + "/events", q, in.limit(), err
		})
	addPage(s, "events_summary", "Events Summary",
		"How many events the build has.",
		buildPage("/events/summary"))
	addPage(s, "list_findings", "List Findings",
		"The build's findings (the intended vulnerabilities): where each is, its severity, difficulty and description.",
		buildPage("/findings"))
	addPage(s, "list_external_access", "List External Access",
		"Every public endpoint in the build: team, host, public IP and port, protocol, and the internal port it maps to.",
		buildPage("/external-access"))
	addPage(s, "export_external_access", "Export External Access With Logins",
		"The public endpoints with each host's login (username and current password, and whether the password is applied yet), as handed to teams.",
		buildPage("/external-access/export"))
	addPage(s, "list_credentials", "List Credentials",
		"Every account LaForge has set a password for on every host in the build: host, team, username, password, and whether it's applied.",
		buildPage("/credentials"))
	addPage(s, "list_validation_reports", "List Validation Reports",
		"The build's revalidation reports: when, how many hosts, passed, failed, still running.",
		buildPage("/validation-reports"))
	addPage(s, "get_validation_report", "Get Validation Report",
		"One revalidation report: each host's result and every validator's outcome and message.",
		func(ctx context.Context, c *call, in reportIn) (string, url.Values, int, error) {
			p, err := c.buildPath(ctx, in.BuildID)
			return p + "/validation-reports/" + seg(in.ReportID), nil, in.limit(), err
		})
	addPage(s, "list_scheduled_tasks", "List Scheduled Tasks",
		"The build's scheduled tasks (authored schedule entries and operator-scheduled commands): what, where, when, next fire, status, payload, and their run counts.",
		buildPage("/scheduled-tasks"))
	addPage(s, "list_scheduled_task_runs", "List Scheduled Task Runs",
		"Every run of one scheduled task: status, command, payload, output and error.",
		func(ctx context.Context, c *call, in scheduledRunsIn) (string, url.Values, int, error) {
			p, err := c.buildPath(ctx, in.BuildID)
			return p + "/scheduled-tasks/" + seg(in.TaskID) + "/runs", nil, in.limit(), err
		})
	addPage(s, "get_artifacts", "Get Artifact Storage",
		"How much storage the build's agent binaries, step output and events take.",
		buildPage("/artifacts"))
	addPage(s, "get_upcoming", "Get Upcoming Changes",
		"What would change if the build were brought up to its configured build's latest commit: hosts added, removed, rebuilt, or with changed steps.",
		buildPage("/upcoming"))

	// --- one object ---
	addPage(s, "list_steps", "List Steps",
		"A host's or container's steps in order: the authored step each belongs to, command, status, attempts, timing, wait_for and milestone, output, error, and validator results.",
		objectPage("/steps"))
	addPage(s, "list_object_events", "List Object Events",
		"Everything that happened to one object: builder requests with their full detail and results, step starts and finishes, power changes, errors.",
		objectPage("/events"))
	addPage(s, "list_object_heartbeats", "List Object Heartbeats",
		"An agent's most recent check-ins: time, address, CPU, memory, disk and network use, boot and agent start times.",
		objectPage("/heartbeats"))
	addPage(s, "get_object_infra", "Get Object Infrastructure",
		"Where an object runs and how to log in: builder, instance name, the current root/Administrator password and every account LaForge set a password for.",
		objectPage("/infra"))
	addPage(s, "get_object_config", "Get Object Config",
		"An object's authored configuration as resolved for this build: OS or image, size, ports, dependencies, vars, env, tags, people.",
		objectPage("/config"))
	addPage(s, "render_object", "Render Object",
		"An object's steps rendered for this build and team, exactly as the agent runs them: every script with its templating filled in, and the raw authored steps.",
		objectPage("/render"))
	addPage(s, "get_cloud_init", "Get Cloud-Init",
		"The first-boot cloud-init (user data) the builder gives an object, with its temporary login.",
		objectPage("/cloud-init"))
	addPage(s, "list_object_validation_runs", "List Object Validation Runs",
		"An object's recent revalidation runs and their results.",
		objectPage("/validation-runs"))

	// --- builders and the instance ---
	addPage(s, "list_builders", "List Builders",
		"The infrastructure builders LaForge deploys to: name, kind and color.",
		instancePage("/builders"))
	addPage(s, "list_builder_configs", "List Builder Configs",
		"Every builder's configuration (connection details for instance admins): hoster URL, project, storage, networks, images, sizes, concurrency.",
		instancePage("/builder-configs"))
	addPage(s, "get_builder_config", "Get Builder Config",
		"One builder's configuration.",
		builderPage(""))
	addPage(s, "get_builder_stats", "Get Builder Stats",
		"A builder's health over time from its stored samples: CPU, memory, storage and network per cluster member or server, and quota gauges.",
		func(_ context.Context, _ *call, in builderStatsIn) (string, url.Values, int, error) {
			if strings.TrimSpace(in.Builder) == "" {
				return "", nil, 0, errors.New("builder is required")
			}
			q := url.Values{}
			if in.Window != "" {
				d, err := time.ParseDuration(in.Window)
				if err != nil || d <= 0 || d > 24*time.Hour {
					return "", nil, 0, fmt.Errorf("window %q must be a duration up to 24h, such as 30m or 6h", in.Window)
				}
				q.Set("window", in.Window)
			}
			return "/builder-configs/" + seg(in.Builder) + "/stats", q, in.limit(), nil
		})
	addPage(s, "get_builder_usage", "Get Builder Usage",
		"What's running on a builder: active builds, object counts by kind and status, and public endpoints.",
		builderPage("/usage"))
	addPage(s, "list_image_builds", "List Image Builds",
		"A builder's recent base-image builds and their status. Needs instance admin.",
		builderPage("/image-builds"))
	addPage(s, "get_image_build_log", "Get Image Build Log",
		"An image build's console log. Needs instance admin.",
		func(ctx context.Context, c *call, in imageBuildLogIn) (string, url.Values, int, error) {
			id, err := parseID(in.ImageBuildID, "image_build_id")
			if err != nil {
				return "", nil, 0, err
			}
			q := url.Values{}
			if in.Since > 0 {
				q.Set("since", fmt.Sprint(in.Since))
			}
			return "/image-builds/" + id.String() + "/log", q, in.limit(), nil
		})
	addPage(s, "list_instance_admins", "List Instance Admins",
		"Who has instance-wide admin. Needs instance admin.",
		instancePage("/instance-admins"))
	addPage(s, "list_installations", "List GitHub Installations",
		"The GitHub App installations and the repositories each covers. Needs instance admin.",
		instancePage("/installations"))
	addPage(s, "list_unapproved_repositories", "List Unapproved Repositories",
		"Repositories the GitHub App can see that aren't approved in LaForge yet. Needs instance admin.",
		instancePage("/installations/repositories"))
	addPage(s, "list_registry_credentials", "List Registry Credentials",
		"Container registries LaForge has credentials for (host and username). Needs instance admin.",
		instancePage("/registry-credentials"))
	addPage(s, "get_cert_status", "Get Certificate Status",
		"When the agent CA and gateway certificates expire.",
		instancePage("/cert-status"))
}
