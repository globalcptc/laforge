--
-- PostgreSQL database dump
--
-- Dumped from database version 16.15
-- Dumped by pg_dump version 16.15 (Homebrew)
--
-- Name: account; Type: TABLE; Schema: public; Owner: -
--
CREATE TABLE public.account (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    github_id bigint NOT NULL,
    github_login text NOT NULL,
    avatar_url text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    timezone text DEFAULT ''::text NOT NULL
);
--
-- Name: agent_heartbeat; Type: TABLE; Schema: public; Owner: -
--
CREATE TABLE public.agent_heartbeat (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    deployed_object_id uuid NOT NULL,
    cert_fingerprint text NOT NULL,
    remote_addr text,
    next_poll_ms integer,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    cpu_pct double precision,
    mem_pct double precision,
    disk_pct double precision,
    net_rx_bps double precision,
    net_tx_bps double precision
);
--
-- Name: external_access; Type: TABLE; Schema: public; Owner: -
--
CREATE TABLE public.external_access (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    deployed_object_id uuid NOT NULL,
    protocol text NOT NULL,
    internal_port text NOT NULL,
    public_address text NOT NULL,
    external_port text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT external_access_pkey PRIMARY KEY (id),
    CONSTRAINT external_access_object_proto_port_key UNIQUE (deployed_object_id, protocol, internal_port)
);
--
-- Name: agent_artifact; Type: TABLE; Schema: public; Owner: -
--
CREATE TABLE public.agent_artifact (
    deployed_object_id uuid NOT NULL,
    platform text NOT NULL,
    token text NOT NULL,
    agent_binary bytea NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT agent_artifact_pkey PRIMARY KEY (deployed_object_id),
    CONSTRAINT agent_artifact_token_key UNIQUE (token)
);
--
-- Name: agent_session; Type: TABLE; Schema: public; Owner: -
--
CREATE TABLE public.agent_session (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    deployed_object_id uuid NOT NULL,
    cert_fingerprint text NOT NULL,
    first_seen_at timestamp with time zone DEFAULT now() NOT NULL,
    last_heartbeat_at timestamp with time zone DEFAULT now() NOT NULL
);
--
-- Name: agent_task; Type: TABLE; Schema: public; Owner: -
--
CREATE TABLE public.agent_task (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    deployed_object_id uuid NOT NULL,
    step_index integer NOT NULL,
    command text NOT NULL,
    payload jsonb DEFAULT '{}'::jsonb NOT NULL,
    status text DEFAULT 'pending'::text NOT NULL,
    lease_expires_at timestamp with time zone,
    attempts integer DEFAULT 0 NOT NULL,
    output text,
    last_error text,
    ignore_errors boolean DEFAULT false NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    ad_hoc boolean DEFAULT false NOT NULL,
    CONSTRAINT agent_task_status_check CHECK ((status = ANY (ARRAY['pending'::text, 'leased'::text, 'done'::text, 'failed'::text, 'ignored'::text, 'blocked'::text])))
);
--
-- Name: build; Type: TABLE; Schema: public; Owner: -
--
CREATE TABLE public.build (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    configured_build_id uuid,
    content_revision_id uuid NOT NULL,
    environment_name text NOT NULL,
    status text DEFAULT 'planned'::text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    auto_built boolean DEFAULT false NOT NULL,
    reconcile_error text,
    created_by_account_id uuid,
    CONSTRAINT build_status_check CHECK ((status = ANY (ARRAY['planned'::text, 'deploying'::text, 'building'::text, 'finished'::text, 'failed'::text, 'tearing_down'::text, 'torn_down'::text, 'purged'::text])))
);
--
-- Name: attention_dismissal; Type: TABLE; Schema: public; Owner: -
--
CREATE TABLE public.attention_dismissal (
    account_id uuid NOT NULL,
    build_id uuid NOT NULL,
    category text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT attention_dismissal_pkey PRIMARY KEY (account_id, build_id, category)
);
--
-- Name: shell_session; Type: TABLE; Schema: public; Owner: -
--
CREATE TABLE public.shell_session (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    deployed_object_id uuid NOT NULL,
    opened_by_account_id uuid,
    status text DEFAULT 'pending'::text NOT NULL,
    client_addr text,
    started_at timestamp with time zone DEFAULT now() NOT NULL,
    ended_at timestamp with time zone,
    CONSTRAINT shell_session_pkey PRIMARY KEY (id),
    CONSTRAINT shell_session_status_check CHECK ((status = ANY (ARRAY['pending'::text, 'active'::text, 'closed'::text])))
);
--
-- Name: builder_config; Type: TABLE; Schema: public; Owner: -
--
CREATE TABLE public.builder_config (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    name text NOT NULL,
    kind text NOT NULL,
    incus_api_url text,
    incus_client_cert_path text,
    incus_client_key_path text,
    incus_server_cert_pem text,
    incus_ovn_uplink_network text,
    incus_storage_pool text,
    incus_operation_timeout_seconds integer,
    incus_images jsonb DEFAULT '{}'::jsonb NOT NULL,
    incus_sizes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    incus_hosts jsonb DEFAULT '[]'::jsonb NOT NULL,
    incus_credential_id uuid,
    container_base_server text DEFAULT 'https://cloud-images.ubuntu.com/releases'::text NOT NULL,
    container_base_alias text DEFAULT '22.04'::text NOT NULL,
    docker_base_fingerprint text DEFAULT ''::text NOT NULL,
    external_access_ip text,
    external_port_min integer,
    external_port_max integer,
    CONSTRAINT builder_config_kind_check CHECK ((kind = ANY (ARRAY['fake'::text, 'incus'::text, 'microcloud'::text, 'aws'::text, 'openstack'::text])))
);

CREATE TABLE public.builder_image_build (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    builder_config_id uuid NOT NULL,
    kind text DEFAULT 'docker_base'::text NOT NULL,
    status text DEFAULT 'pending'::text NOT NULL,
    image_fingerprint text DEFAULT ''::text NOT NULL,
    error text DEFAULT ''::text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    started_at timestamp with time zone,
    finished_at timestamp with time zone,
    CONSTRAINT builder_image_build_pkey PRIMARY KEY (id),
    CONSTRAINT builder_image_build_status_check CHECK ((status = ANY (ARRAY['pending'::text, 'running'::text, 'succeeded'::text, 'failed'::text]))),
    CONSTRAINT builder_image_build_builder_config_id_fkey FOREIGN KEY (builder_config_id) REFERENCES public.builder_config(id) ON DELETE CASCADE
);

CREATE INDEX builder_image_build_pending_idx ON public.builder_image_build USING btree (created_at) WHERE (status = 'pending'::text);
CREATE INDEX builder_image_build_by_builder_idx ON public.builder_image_build USING btree (builder_config_id, created_at DESC);

CREATE TABLE public.builder_image_build_log (
    id bigint GENERATED BY DEFAULT AS IDENTITY NOT NULL,
    build_id uuid NOT NULL,
    seq integer NOT NULL,
    line text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT builder_image_build_log_pkey PRIMARY KEY (id),
    CONSTRAINT builder_image_build_log_build_id_fkey FOREIGN KEY (build_id) REFERENCES public.builder_image_build(id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX builder_image_build_log_seq_idx ON public.builder_image_build_log USING btree (build_id, seq);

CREATE TABLE public.registry_credential (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    registry_host text NOT NULL,
    username text NOT NULL,
    secret text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT registry_credential_pkey PRIMARY KEY (id),
    CONSTRAINT registry_credential_registry_host_key UNIQUE (registry_host)
);
--
-- Name: instance_admin; Type: TABLE; Schema: public; Owner: -
--
CREATE TABLE public.instance_admin (
    github_login text NOT NULL,
    added_by text,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);

CREATE UNIQUE INDEX instance_admin_login_idx ON public.instance_admin USING btree (lower(github_login));
--
-- Name: builder_credential; Type: TABLE; Schema: public; Owner: -
--
CREATE TABLE public.builder_credential (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    api_url text NOT NULL,
    server_name text DEFAULT ''::text NOT NULL,
    server_fingerprint text NOT NULL,
    server_cert_pem text NOT NULL,
    client_cert_pem text NOT NULL,
    client_key_pem text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);
--
-- Name: configured_build; Type: TABLE; Schema: public; Owner: -
--
CREATE TABLE public.configured_build (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    repository_id uuid NOT NULL,
    branch text NOT NULL,
    environment_path text NOT NULL,
    builder_config_name text NOT NULL,
    competition_started boolean DEFAULT false NOT NULL,
    auto_deploy_enabled boolean DEFAULT true NOT NULL,
    current_content_revision_id uuid,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);
--
-- Name: container; Type: TABLE; Schema: public; Owner: -
--
CREATE TABLE public.container (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    content_revision_id uuid NOT NULL,
    path text NOT NULL,
    name text NOT NULL,
    image text,
    size text,
    ports jsonb DEFAULT '{}'::jsonb NOT NULL,
    depends_on jsonb DEFAULT '[]'::jsonb NOT NULL,
    steps jsonb DEFAULT '[]'::jsonb NOT NULL,
    schedule jsonb DEFAULT '[]'::jsonb NOT NULL,
    vars jsonb DEFAULT '{}'::jsonb NOT NULL,
    tags jsonb DEFAULT '{}'::jsonb NOT NULL,
    findings jsonb DEFAULT '[]'::jsonb NOT NULL,
    people jsonb DEFAULT '[]'::jsonb NOT NULL,
    extends text
);
--
-- Name: content_revision; Type: TABLE; Schema: public; Owner: -
--
CREATE TABLE public.content_revision (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    repository_id uuid NOT NULL,
    commit_sha text NOT NULL,
    ref text,
    valid boolean DEFAULT false NOT NULL,
    validation_errors jsonb DEFAULT '[]'::jsonb NOT NULL,
    validated_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    commit_message text,
    commit_author text,
    committed_at timestamp with time zone
);
--
-- Name: deployed_object; Type: TABLE; Schema: public; Owner: -
--
CREATE TABLE public.deployed_object (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    team_id uuid NOT NULL,
    kind text NOT NULL,
    object_name text NOT NULL,
    as_name text,
    network_name text,
    fingerprint text DEFAULT ''::text NOT NULL,
    status text DEFAULT 'pending'::text NOT NULL,
    external_ref text,
    last_error text,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    power_state text DEFAULT ''::text NOT NULL,
    power_state_checked_at timestamp with time zone,
    tags jsonb DEFAULT '{}'::jsonb NOT NULL,
    steps_materialized_at timestamp with time zone,
    blocked_on jsonb,
    CONSTRAINT deployed_object_kind_check CHECK ((kind = ANY (ARRAY['network'::text, 'host'::text, 'container'::text, 'dns'::text]))),
    CONSTRAINT deployed_object_status_check CHECK ((status = ANY (ARRAY['pending'::text, 'deploying'::text, 'running'::text, 'building'::text, 'finished'::text, 'deploy_failed'::text, 'build_failed'::text, 'invalid'::text, 'destroying'::text, 'destroyed'::text])))
);
--
-- Name: environment; Type: TABLE; Schema: public; Owner: -
--
CREATE TABLE public.environment (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    content_revision_id uuid NOT NULL,
    path text NOT NULL,
    name text NOT NULL,
    schema_version integer,
    description text,
    teams integer NOT NULL,
    root_password text,
    start_at timestamp with time zone,
    stop_at timestamp with time zone,
    dns jsonb,
    access jsonb DEFAULT '[]'::jsonb NOT NULL,
    vars jsonb DEFAULT '{}'::jsonb NOT NULL,
    tags jsonb DEFAULT '{}'::jsonb NOT NULL,
    findings jsonb DEFAULT '[]'::jsonb NOT NULL,
    extends text,
    agent_debug boolean DEFAULT false NOT NULL,
    container_logs jsonb
);
--
-- Name: event; Type: TABLE; Schema: public; Owner: -
--
CREATE TABLE public.event (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    build_id uuid NOT NULL,
    task_id uuid,
    kind text NOT NULL,
    message text NOT NULL,
    payload jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    deployed_object_id uuid
);
--
-- Name: fake_hoster_resource; Type: TABLE; Schema: public; Owner: -
--
CREATE TABLE public.fake_hoster_resource (
    external_ref text NOT NULL,
    kind text NOT NULL,
    ensure_count integer DEFAULT 1 NOT NULL,
    destroyed boolean DEFAULT false NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT fake_hoster_resource_kind_check CHECK ((kind = ANY (ARRAY['network'::text, 'host'::text, 'container'::text])))
);
--
-- Name: fake_dns_record; Type: TABLE; Schema: public; Owner: -
--
CREATE TABLE public.fake_dns_record (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    team text NOT NULL,
    name text NOT NULL,
    type text NOT NULL,
    value text NOT NULL,
    priority integer DEFAULT 0 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);
--
-- Name: github_installation; Type: TABLE; Schema: public; Owner: -
--
CREATE TABLE public.github_installation (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    installation_id bigint NOT NULL,
    account_login text NOT NULL,
    account_type text NOT NULL,
    suspended boolean DEFAULT false NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);
--
-- Name: goose_db_version; Type: TABLE; Schema: public; Owner: -
--
CREATE TABLE public.goose_db_version (
    id integer NOT NULL,
    version_id bigint NOT NULL,
    is_applied boolean NOT NULL,
    tstamp timestamp without time zone DEFAULT now() NOT NULL
);
--
-- Name: goose_db_version_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--
ALTER TABLE public.goose_db_version ALTER COLUMN id ADD GENERATED BY DEFAULT AS IDENTITY (
    SEQUENCE NAME public.goose_db_version_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);
--
-- Name: host; Type: TABLE; Schema: public; Owner: -
--
CREATE TABLE public.host (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    content_revision_id uuid NOT NULL,
    path text NOT NULL,
    name text NOT NULL,
    os text,
    size text,
    disk_gb integer,
    ports jsonb DEFAULT '{}'::jsonb NOT NULL,
    depends_on jsonb DEFAULT '[]'::jsonb NOT NULL,
    steps jsonb DEFAULT '[]'::jsonb NOT NULL,
    schedule jsonb DEFAULT '[]'::jsonb NOT NULL,
    vars jsonb DEFAULT '{}'::jsonb NOT NULL,
    tags jsonb DEFAULT '{}'::jsonb NOT NULL,
    findings jsonb DEFAULT '[]'::jsonb NOT NULL,
    people jsonb DEFAULT '[]'::jsonb NOT NULL,
    extends text
);
--
-- Name: installation_repository; Type: TABLE; Schema: public; Owner: -
--
CREATE TABLE public.installation_repository (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    installation_id uuid NOT NULL,
    github_owner text NOT NULL,
    github_repo text NOT NULL,
    github_repo_id bigint NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);
--
-- Name: network; Type: TABLE; Schema: public; Owner: -
--
CREATE TABLE public.network (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    content_revision_id uuid NOT NULL,
    path text NOT NULL,
    name text NOT NULL,
    cidr text NOT NULL,
    visible_from jsonb DEFAULT '[]'::jsonb NOT NULL,
    vars jsonb DEFAULT '{}'::jsonb NOT NULL,
    tags jsonb DEFAULT '{}'::jsonb NOT NULL,
    findings jsonb DEFAULT '[]'::jsonb NOT NULL,
    extends text
);
--
-- Name: people_source; Type: TABLE; Schema: public; Owner: -
--
CREATE TABLE public.people_source (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    content_revision_id uuid NOT NULL,
    path text NOT NULL,
    name text NOT NULL
);
--
-- Name: person; Type: TABLE; Schema: public; Owner: -
--
CREATE TABLE public.person (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    people_source_id uuid NOT NULL,
    username text NOT NULL,
    attributes jsonb DEFAULT '{}'::jsonb NOT NULL
);
--
-- Name: placement; Type: TABLE; Schema: public; Owner: -
--
CREATE TABLE public.placement (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    environment_id uuid NOT NULL,
    network_name text NOT NULL,
    object_kind text NOT NULL,
    object_name text NOT NULL,
    as_name text NOT NULL,
    last_octet integer NOT NULL,
    public jsonb,
    CONSTRAINT placement_object_kind_check CHECK ((object_kind = ANY (ARRAY['host'::text, 'container'::text])))
);
--
-- Name: repository; Type: TABLE; Schema: public; Owner: -
--
CREATE TABLE public.repository (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    github_owner text NOT NULL,
    github_repo text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    installation_id uuid
);
--
-- Name: repository_access; Type: TABLE; Schema: public; Owner: -
--
CREATE TABLE public.repository_access (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    repository_id uuid NOT NULL,
    account_id uuid NOT NULL,
    level text NOT NULL,
    granted_by uuid,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT repository_access_level_check CHECK ((level = ANY (ARRAY['none'::text, 'read'::text, 'build'::text, 'manage'::text, 'admin'::text])))
);
--
-- Name: scheduled_task; Type: TABLE; Schema: public; Owner: -
--
CREATE TABLE public.scheduled_task (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    build_id uuid NOT NULL,
    source text NOT NULL,
    deployed_object_id uuid,
    schedule_index integer,
    target jsonb,
    when_expr text NOT NULL,
    command text NOT NULL,
    payload jsonb DEFAULT '{}'::jsonb NOT NULL,
    anchor text DEFAULT ''::text NOT NULL,
    fires_once boolean NOT NULL,
    next_fire_at timestamp with time zone,
    status text DEFAULT 'pending'::text NOT NULL,
    created_by text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT scheduled_task_check CHECK ((((source = 'content'::text) AND (deployed_object_id IS NOT NULL) AND (schedule_index IS NOT NULL) AND (target IS NULL)) OR ((source = 'adhoc'::text) AND (deployed_object_id IS NULL) AND (schedule_index IS NULL) AND (target IS NOT NULL)))),
    CONSTRAINT scheduled_task_source_check CHECK ((source = ANY (ARRAY['content'::text, 'adhoc'::text]))),
    CONSTRAINT scheduled_task_status_check CHECK ((status = ANY (ARRAY['pending'::text, 'fired'::text, 'canceled'::text])))
);
--
-- Name: script; Type: TABLE; Schema: public; Owner: -
--
CREATE TABLE public.script (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    content_revision_id uuid NOT NULL,
    path text NOT NULL,
    name text NOT NULL,
    description text,
    language text,
    source_path text,
    timeout_seconds integer,
    args jsonb DEFAULT '[]'::jsonb NOT NULL,
    ignore_errors boolean DEFAULT false NOT NULL,
    tags jsonb DEFAULT '{}'::jsonb NOT NULL,
    findings jsonb DEFAULT '[]'::jsonb NOT NULL,
    people jsonb DEFAULT '[]'::jsonb NOT NULL,
    validate jsonb DEFAULT '[]'::jsonb NOT NULL
);
--
-- Name: session; Type: TABLE; Schema: public; Owner: -
--
CREATE TABLE public.session (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    account_id uuid NOT NULL,
    token_hash text NOT NULL,
    github_token text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    github_token_expires_at timestamp with time zone,
    github_refresh_token text DEFAULT ''::text NOT NULL,
    github_refresh_expires_at timestamp with time zone
);
--
-- Name: task; Type: TABLE; Schema: public; Owner: -
--
CREATE TABLE public.task (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    build_id uuid NOT NULL,
    deployed_object_id uuid,
    kind text NOT NULL,
    payload jsonb DEFAULT '{}'::jsonb NOT NULL,
    status text DEFAULT 'pending'::text NOT NULL,
    attempts integer DEFAULT 0 NOT NULL,
    lease_owner text,
    lease_expires_at timestamp with time zone,
    last_error text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT task_status_check CHECK ((status = ANY (ARRAY['pending'::text, 'leased'::text, 'done'::text, 'failed'::text])))
);
--
-- Name: team; Type: TABLE; Schema: public; Owner: -
--
CREATE TABLE public.team (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    build_id uuid NOT NULL,
    team_number integer NOT NULL,
    access_state text DEFAULT 'closed'::text NOT NULL,
    access_override_until timestamp with time zone,
    access_override_state text DEFAULT ''::text NOT NULL,
    CONSTRAINT team_access_state_check CHECK ((access_state = ANY (ARRAY['open'::text, 'closed'::text]))),
    CONSTRAINT team_access_override_state_check CHECK ((access_override_state = ANY (ARRAY[''::text, 'open'::text, 'closed'::text])))
);
--
-- Name: validator_result; Type: TABLE; Schema: public; Owner: -
--
CREATE TABLE public.validator_result (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    agent_task_id uuid NOT NULL,
    kind text NOT NULL,
    args jsonb DEFAULT '{}'::jsonb NOT NULL,
    passed boolean NOT NULL,
    message text,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);
--
-- Name: account account_github_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.account
    ADD CONSTRAINT account_github_id_key UNIQUE (github_id);
--
-- Name: account account_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.account
    ADD CONSTRAINT account_pkey PRIMARY KEY (id);
--
-- Name: agent_heartbeat agent_heartbeat_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.agent_heartbeat
    ADD CONSTRAINT agent_heartbeat_pkey PRIMARY KEY (id);
--
-- Name: agent_session agent_session_deployed_object_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.agent_session
    ADD CONSTRAINT agent_session_deployed_object_id_key UNIQUE (deployed_object_id);
--
-- Name: agent_session agent_session_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.agent_session
    ADD CONSTRAINT agent_session_pkey PRIMARY KEY (id);
--
-- Name: agent_task agent_task_deployed_object_id_step_index_key; Type: CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.agent_task
    ADD CONSTRAINT agent_task_deployed_object_id_step_index_key UNIQUE (deployed_object_id, step_index);
--
-- Name: agent_task agent_task_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.agent_task
    ADD CONSTRAINT agent_task_pkey PRIMARY KEY (id);
--
-- Name: build build_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.build
    ADD CONSTRAINT build_pkey PRIMARY KEY (id);
--
-- Name: builder_config builder_config_name_key; Type: CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.builder_config
    ADD CONSTRAINT builder_config_name_key UNIQUE (name);
--
-- Name: builder_config builder_config_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.builder_config
    ADD CONSTRAINT builder_config_pkey PRIMARY KEY (id);
--
-- Name: builder_credential builder_credential_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.builder_credential
    ADD CONSTRAINT builder_credential_pkey PRIMARY KEY (id);
--
-- Name: builder_config builder_config_incus_credential_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.builder_config
    ADD CONSTRAINT builder_config_incus_credential_id_fkey FOREIGN KEY (incus_credential_id) REFERENCES public.builder_credential(id);
--
-- Name: configured_build configured_build_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.configured_build
    ADD CONSTRAINT configured_build_pkey PRIMARY KEY (id);
--
-- Name: configured_build configured_build_repository_id_branch_environment_path_key; Type: CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.configured_build
    ADD CONSTRAINT configured_build_repository_id_branch_environment_path_key UNIQUE (repository_id, branch, environment_path);
--
-- Name: container container_content_revision_id_name_key; Type: CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.container
    ADD CONSTRAINT container_content_revision_id_name_key UNIQUE (content_revision_id, name);
--
-- Name: container container_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.container
    ADD CONSTRAINT container_pkey PRIMARY KEY (id);
--
-- Name: content_revision content_revision_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.content_revision
    ADD CONSTRAINT content_revision_pkey PRIMARY KEY (id);
--
-- Name: content_revision content_revision_repository_id_commit_sha_key; Type: CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.content_revision
    ADD CONSTRAINT content_revision_repository_id_commit_sha_key UNIQUE (repository_id, commit_sha);
--
-- Name: deployed_object deployed_object_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.deployed_object
    ADD CONSTRAINT deployed_object_pkey PRIMARY KEY (id);
--
-- Name: environment environment_content_revision_id_name_key; Type: CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.environment
    ADD CONSTRAINT environment_content_revision_id_name_key UNIQUE (content_revision_id, name);
--
-- Name: environment environment_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.environment
    ADD CONSTRAINT environment_pkey PRIMARY KEY (id);
--
-- Name: event event_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.event
    ADD CONSTRAINT event_pkey PRIMARY KEY (id);
--
-- Name: fake_dns_record fake_dns_record_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.fake_dns_record
    ADD CONSTRAINT fake_dns_record_pkey PRIMARY KEY (id);
--
-- Name: fake_hoster_resource fake_hoster_resource_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.fake_hoster_resource
    ADD CONSTRAINT fake_hoster_resource_pkey PRIMARY KEY (external_ref);
--
-- Name: github_installation github_installation_installation_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.github_installation
    ADD CONSTRAINT github_installation_installation_id_key UNIQUE (installation_id);
--
-- Name: github_installation github_installation_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.github_installation
    ADD CONSTRAINT github_installation_pkey PRIMARY KEY (id);
--
-- Name: goose_db_version goose_db_version_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.goose_db_version
    ADD CONSTRAINT goose_db_version_pkey PRIMARY KEY (id);
--
-- Name: host host_content_revision_id_name_key; Type: CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.host
    ADD CONSTRAINT host_content_revision_id_name_key UNIQUE (content_revision_id, name);
--
-- Name: host host_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.host
    ADD CONSTRAINT host_pkey PRIMARY KEY (id);
--
-- Name: installation_repository installation_repository_installation_id_github_repo_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.installation_repository
    ADD CONSTRAINT installation_repository_installation_id_github_repo_id_key UNIQUE (installation_id, github_repo_id);
--
-- Name: installation_repository installation_repository_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.installation_repository
    ADD CONSTRAINT installation_repository_pkey PRIMARY KEY (id);
--
-- Name: network network_content_revision_id_name_key; Type: CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.network
    ADD CONSTRAINT network_content_revision_id_name_key UNIQUE (content_revision_id, name);
--
-- Name: network network_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.network
    ADD CONSTRAINT network_pkey PRIMARY KEY (id);
--
-- Name: people_source people_source_content_revision_id_path_key; Type: CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.people_source
    ADD CONSTRAINT people_source_content_revision_id_path_key UNIQUE (content_revision_id, path);
--
-- Name: people_source people_source_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.people_source
    ADD CONSTRAINT people_source_pkey PRIMARY KEY (id);
--
-- Name: person person_people_source_id_username_key; Type: CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.person
    ADD CONSTRAINT person_people_source_id_username_key UNIQUE (people_source_id, username);
--
-- Name: person person_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.person
    ADD CONSTRAINT person_pkey PRIMARY KEY (id);
--
-- Name: placement placement_environment_id_as_name_key; Type: CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.placement
    ADD CONSTRAINT placement_environment_id_as_name_key UNIQUE (environment_id, as_name);
--
-- Name: placement placement_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.placement
    ADD CONSTRAINT placement_pkey PRIMARY KEY (id);
--
-- Name: repository_access repository_access_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.repository_access
    ADD CONSTRAINT repository_access_pkey PRIMARY KEY (id);
--
-- Name: repository_access repository_access_repository_id_account_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.repository_access
    ADD CONSTRAINT repository_access_repository_id_account_id_key UNIQUE (repository_id, account_id);
--
-- Name: repository repository_github_owner_github_repo_key; Type: CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.repository
    ADD CONSTRAINT repository_github_owner_github_repo_key UNIQUE (github_owner, github_repo);
--
-- Name: repository repository_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.repository
    ADD CONSTRAINT repository_pkey PRIMARY KEY (id);
--
-- Name: scheduled_task scheduled_task_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.scheduled_task
    ADD CONSTRAINT scheduled_task_pkey PRIMARY KEY (id);
--
-- Name: script script_content_revision_id_name_key; Type: CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.script
    ADD CONSTRAINT script_content_revision_id_name_key UNIQUE (content_revision_id, name);
--
-- Name: script script_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.script
    ADD CONSTRAINT script_pkey PRIMARY KEY (id);
--
-- Name: session session_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.session
    ADD CONSTRAINT session_pkey PRIMARY KEY (id);
--
-- Name: session session_token_hash_key; Type: CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.session
    ADD CONSTRAINT session_token_hash_key UNIQUE (token_hash);
--
-- Name: task task_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.task
    ADD CONSTRAINT task_pkey PRIMARY KEY (id);
--
-- Name: team team_build_id_team_number_key; Type: CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.team
    ADD CONSTRAINT team_build_id_team_number_key UNIQUE (build_id, team_number);
--
-- Name: team team_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.team
    ADD CONSTRAINT team_pkey PRIMARY KEY (id);
--
-- Name: validator_result validator_result_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.validator_result
    ADD CONSTRAINT validator_result_pkey PRIMARY KEY (id);
--
-- Name: agent_heartbeat_object_time_idx; Type: INDEX; Schema: public; Owner: -
--
CREATE INDEX agent_heartbeat_object_time_idx ON public.agent_heartbeat USING btree (deployed_object_id, created_at);
--
-- Name: agent_session_fingerprint_idx; Type: INDEX; Schema: public; Owner: -
--
CREATE INDEX agent_session_fingerprint_idx ON public.agent_session USING btree (cert_fingerprint);
--
-- Name: agent_task_leasable_idx; Type: INDEX; Schema: public; Owner: -
--
CREATE INDEX agent_task_leasable_idx ON public.agent_task USING btree (deployed_object_id, status, step_index);
--
-- Name: deployed_object_identity_idx; Type: INDEX; Schema: public; Owner: -
--
CREATE UNIQUE INDEX deployed_object_identity_idx ON public.deployed_object USING btree (team_id, kind, COALESCE(as_name, object_name));
--
-- Name: event_build_idx; Type: INDEX; Schema: public; Owner: -
--
CREATE INDEX event_build_idx ON public.event USING btree (build_id, created_at);
--
-- Name: event_deployed_object_id_idx; Type: INDEX; Schema: public; Owner: -
--
CREATE INDEX event_deployed_object_id_idx ON public.event USING btree (deployed_object_id) WHERE (deployed_object_id IS NOT NULL);
--
-- Name: fake_dns_record_team_idx; Type: INDEX; Schema: public; Owner: -
--
CREATE INDEX fake_dns_record_team_idx ON public.fake_dns_record USING btree (team);
--
-- Name: scheduled_task_build_idx; Type: INDEX; Schema: public; Owner: -
--
CREATE INDEX scheduled_task_build_idx ON public.scheduled_task USING btree (build_id, status);
--
-- Name: scheduled_task_content_once; Type: INDEX; Schema: public; Owner: -
--
CREATE UNIQUE INDEX scheduled_task_content_once ON public.scheduled_task USING btree (deployed_object_id, schedule_index) WHERE (source = 'content'::text);
--
-- Name: scheduled_task_due_idx; Type: INDEX; Schema: public; Owner: -
--
CREATE INDEX scheduled_task_due_idx ON public.scheduled_task USING btree (next_fire_at) WHERE (status = 'pending'::text);
--
-- Name: session_expires_idx; Type: INDEX; Schema: public; Owner: -
--
CREATE INDEX session_expires_idx ON public.session USING btree (expires_at);
--
-- Name: task_leasable_idx; Type: INDEX; Schema: public; Owner: -
--
CREATE INDEX task_leasable_idx ON public.task USING btree (status, lease_expires_at);
--
-- Name: task_one_open_per_object; Type: INDEX; Schema: public; Owner: -
--
CREATE UNIQUE INDEX task_one_open_per_object ON public.task USING btree (deployed_object_id) WHERE ((status = ANY (ARRAY['pending'::text, 'leased'::text])) AND (deployed_object_id IS NOT NULL));
--
-- Name: agent_heartbeat agent_heartbeat_deployed_object_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.agent_heartbeat
    ADD CONSTRAINT agent_heartbeat_deployed_object_id_fkey FOREIGN KEY (deployed_object_id) REFERENCES public.deployed_object(id) ON DELETE CASCADE;
--
-- Name: agent_session agent_session_deployed_object_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.agent_session
    ADD CONSTRAINT agent_session_deployed_object_id_fkey FOREIGN KEY (deployed_object_id) REFERENCES public.deployed_object(id) ON DELETE CASCADE;
--
-- Name: agent_task agent_task_deployed_object_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.agent_task
    ADD CONSTRAINT agent_task_deployed_object_id_fkey FOREIGN KEY (deployed_object_id) REFERENCES public.deployed_object(id) ON DELETE CASCADE;
--
-- Name: build build_configured_build_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.build
    ADD CONSTRAINT build_configured_build_id_fkey FOREIGN KEY (configured_build_id) REFERENCES public.configured_build(id) ON DELETE SET NULL;
--
-- Name: build build_content_revision_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.build
    ADD CONSTRAINT build_content_revision_id_fkey FOREIGN KEY (content_revision_id) REFERENCES public.content_revision(id) ON DELETE CASCADE;
--
-- Name: configured_build configured_build_current_content_revision_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.configured_build
    ADD CONSTRAINT configured_build_current_content_revision_id_fkey FOREIGN KEY (current_content_revision_id) REFERENCES public.content_revision(id);
--
-- Name: configured_build configured_build_repository_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.configured_build
    ADD CONSTRAINT configured_build_repository_id_fkey FOREIGN KEY (repository_id) REFERENCES public.repository(id) ON DELETE CASCADE;
--
-- Name: container container_content_revision_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.container
    ADD CONSTRAINT container_content_revision_id_fkey FOREIGN KEY (content_revision_id) REFERENCES public.content_revision(id) ON DELETE CASCADE;
--
-- Name: content_revision content_revision_repository_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.content_revision
    ADD CONSTRAINT content_revision_repository_id_fkey FOREIGN KEY (repository_id) REFERENCES public.repository(id) ON DELETE CASCADE;
--
-- Name: deployed_object deployed_object_team_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.deployed_object
    ADD CONSTRAINT deployed_object_team_id_fkey FOREIGN KEY (team_id) REFERENCES public.team(id) ON DELETE CASCADE;
--
-- Name: environment environment_content_revision_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.environment
    ADD CONSTRAINT environment_content_revision_id_fkey FOREIGN KEY (content_revision_id) REFERENCES public.content_revision(id) ON DELETE CASCADE;
--
-- Name: event event_build_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.event
    ADD CONSTRAINT event_build_id_fkey FOREIGN KEY (build_id) REFERENCES public.build(id) ON DELETE CASCADE;
--
-- Name: event event_deployed_object_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.event
    ADD CONSTRAINT event_deployed_object_id_fkey FOREIGN KEY (deployed_object_id) REFERENCES public.deployed_object(id) ON DELETE SET NULL;
--
-- Name: event event_task_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.event
    ADD CONSTRAINT event_task_id_fkey FOREIGN KEY (task_id) REFERENCES public.task(id) ON DELETE SET NULL;
--
-- Name: host host_content_revision_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.host
    ADD CONSTRAINT host_content_revision_id_fkey FOREIGN KEY (content_revision_id) REFERENCES public.content_revision(id) ON DELETE CASCADE;
--
-- Name: installation_repository installation_repository_installation_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.installation_repository
    ADD CONSTRAINT installation_repository_installation_id_fkey FOREIGN KEY (installation_id) REFERENCES public.github_installation(id) ON DELETE CASCADE;
--
-- Name: network network_content_revision_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.network
    ADD CONSTRAINT network_content_revision_id_fkey FOREIGN KEY (content_revision_id) REFERENCES public.content_revision(id) ON DELETE CASCADE;
--
-- Name: people_source people_source_content_revision_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.people_source
    ADD CONSTRAINT people_source_content_revision_id_fkey FOREIGN KEY (content_revision_id) REFERENCES public.content_revision(id) ON DELETE CASCADE;
--
-- Name: person person_people_source_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.person
    ADD CONSTRAINT person_people_source_id_fkey FOREIGN KEY (people_source_id) REFERENCES public.people_source(id) ON DELETE CASCADE;
--
-- Name: placement placement_environment_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.placement
    ADD CONSTRAINT placement_environment_id_fkey FOREIGN KEY (environment_id) REFERENCES public.environment(id) ON DELETE CASCADE;
--
-- Name: repository_access repository_access_account_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.repository_access
    ADD CONSTRAINT repository_access_account_id_fkey FOREIGN KEY (account_id) REFERENCES public.account(id) ON DELETE CASCADE;
--
-- Name: repository_access repository_access_granted_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.repository_access
    ADD CONSTRAINT repository_access_granted_by_fkey FOREIGN KEY (granted_by) REFERENCES public.account(id);
--
-- Name: repository_access repository_access_repository_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.repository_access
    ADD CONSTRAINT repository_access_repository_id_fkey FOREIGN KEY (repository_id) REFERENCES public.repository(id) ON DELETE CASCADE;
--
-- Name: repository repository_installation_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.repository
    ADD CONSTRAINT repository_installation_id_fkey FOREIGN KEY (installation_id) REFERENCES public.github_installation(id) ON DELETE SET NULL;
--
-- Name: scheduled_task scheduled_task_build_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.scheduled_task
    ADD CONSTRAINT scheduled_task_build_id_fkey FOREIGN KEY (build_id) REFERENCES public.build(id) ON DELETE CASCADE;
--
-- Name: scheduled_task scheduled_task_deployed_object_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.scheduled_task
    ADD CONSTRAINT scheduled_task_deployed_object_id_fkey FOREIGN KEY (deployed_object_id) REFERENCES public.deployed_object(id) ON DELETE CASCADE;
--
-- Name: script script_content_revision_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.script
    ADD CONSTRAINT script_content_revision_id_fkey FOREIGN KEY (content_revision_id) REFERENCES public.content_revision(id) ON DELETE CASCADE;
--
-- Name: session session_account_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.session
    ADD CONSTRAINT session_account_id_fkey FOREIGN KEY (account_id) REFERENCES public.account(id) ON DELETE CASCADE;
--
-- Name: task task_build_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.task
    ADD CONSTRAINT task_build_id_fkey FOREIGN KEY (build_id) REFERENCES public.build(id) ON DELETE CASCADE;
--
-- Name: task task_deployed_object_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.task
    ADD CONSTRAINT task_deployed_object_id_fkey FOREIGN KEY (deployed_object_id) REFERENCES public.deployed_object(id) ON DELETE CASCADE;
--
-- Name: team team_build_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.team
    ADD CONSTRAINT team_build_id_fkey FOREIGN KEY (build_id) REFERENCES public.build(id) ON DELETE CASCADE;
--
-- Name: validator_result validator_result_agent_task_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--
ALTER TABLE ONLY public.validator_result
    ADD CONSTRAINT validator_result_agent_task_id_fkey FOREIGN KEY (agent_task_id) REFERENCES public.agent_task(id) ON DELETE CASCADE;
--
-- PostgreSQL database dump complete
--
