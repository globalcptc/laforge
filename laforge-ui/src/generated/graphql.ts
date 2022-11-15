import { Injectable } from '@angular/core';
import { gql } from 'apollo-angular';
import * as Apollo from 'apollo-angular';
export type Maybe<T> = T | null;
export type InputMaybe<T> = Maybe<T>;
export type Exact<T extends { [key: string]: unknown }> = { [K in keyof T]: T[K] };
export type MakeOptional<T, K extends keyof T> = Omit<T, K> & { [SubKey in K]?: Maybe<T[SubKey]> };
export type MakeMaybe<T, K extends keyof T> = Omit<T, K> & { [SubKey in K]: Maybe<T[SubKey]> };
/** All built-in and custom scalars, mapped to their actual values */
export type Scalars = {
  ID: string;
  String: string;
  Boolean: boolean;
  Int: number;
  Float: number;
  Time: any;
};

export enum LaForgeAgentCommand {
  Addtogroup = 'ADDTOGROUP',
  Appendfile = 'APPENDFILE',
  Changeperms = 'CHANGEPERMS',
  Createuser = 'CREATEUSER',
  Createuserpass = 'CREATEUSERPASS',
  Default = 'DEFAULT',
  Delete = 'DELETE',
  Download = 'DOWNLOAD',
  Execute = 'EXECUTE',
  Extract = 'EXTRACT',
  Reboot = 'REBOOT',
  Validate = 'VALIDATE'
}

export type LaForgeAgentStatus = {
  __typename?: 'AgentStatus';
  OS: Scalars['String'];
  bootTime: Scalars['Int'];
  clientId: Scalars['String'];
  freeMem: Scalars['Int'];
  hostID: Scalars['String'];
  hostname: Scalars['String'];
  load1?: Maybe<Scalars['Float']>;
  load5?: Maybe<Scalars['Float']>;
  load15?: Maybe<Scalars['Float']>;
  numProcs: Scalars['Int'];
  timestamp: Scalars['Int'];
  totalMem: Scalars['Int'];
  upTime: Scalars['Int'];
  usedMem: Scalars['Int'];
};

export type LaForgeAgentStatusBatch = {
  __typename?: 'AgentStatusBatch';
  agentStatuses: Array<Maybe<LaForgeAgentStatus>>;
  pageInfo: LaForgeLaForgePageInfo;
};

export type LaForgeAgentTask = {
  __typename?: 'AgentTask';
  args?: Maybe<Scalars['String']>;
  command: LaForgeAgentCommand;
  error_message?: Maybe<Scalars['String']>;
  id: Scalars['ID'];
  number: Scalars['Int'];
  output?: Maybe<Scalars['String']>;
  state: LaForgeAgentTaskState;
};

export enum LaForgeAgentTaskState {
  Awaiting = 'AWAITING',
  Complete = 'COMPLETE',
  Failed = 'FAILED',
  Inprogress = 'INPROGRESS'
}

export type LaForgeAuthUser = {
  __typename?: 'AuthUser';
  company: Scalars['String'];
  email: Scalars['String'];
  first_name: Scalars['String'];
  id: Scalars['ID'];
  last_name: Scalars['String'];
  occupation: Scalars['String'];
  phone: Scalars['String'];
  provider: LaForgeProviderType;
  publicKey: Scalars['String'];
  role: LaForgeRoleLevel;
  username: Scalars['String'];
};

export type LaForgeBuild = {
  __typename?: 'Build';
  BuildToBuildCommits: Array<Maybe<LaForgeBuildCommit>>;
  BuildToLatestBuildCommit?: Maybe<LaForgeBuildCommit>;
  BuildToRepoCommit: LaForgeRepoCommit;
  BuildToServerTasks: Array<Maybe<LaForgeServerTask>>;
  buildToCompetition: LaForgeCompetition;
  buildToEnvironment: LaForgeEnvironment;
  buildToPlan: Array<Maybe<LaForgePlan>>;
  buildToProvisionedNetwork: Array<Maybe<LaForgeProvisionedNetwork>>;
  buildToStatus: LaForgeStatus;
  buildToTeam: Array<Maybe<LaForgeTeam>>;
  completed_plan: Scalars['Boolean'];
  environment_revision: Scalars['Int'];
  id: Scalars['ID'];
  revision: Scalars['Int'];
};

export type LaForgeBuildCommit = {
  __typename?: 'BuildCommit';
  BuildCommitToBuild: LaForgeBuild;
  BuildCommitToPlanDiffs: Array<Maybe<LaForgePlanDiff>>;
  BuildCommitToServerTask: Array<Maybe<LaForgeServerTask>>;
  createdAt: Scalars['Time'];
  id: Scalars['ID'];
  revision: Scalars['Int'];
  state: LaForgeBuildCommitState;
  type: LaForgeBuildCommitType;
};

export enum LaForgeBuildCommitState {
  Applied = 'APPLIED',
  Approved = 'APPROVED',
  Cancelled = 'CANCELLED',
  Inprogress = 'INPROGRESS',
  Planning = 'PLANNING'
}

export enum LaForgeBuildCommitType {
  Delete = 'DELETE',
  Rebuild = 'REBUILD',
  Root = 'ROOT'
}

export type LaForgeCommand = {
  __typename?: 'Command';
  CommandToEnvironment: LaForgeEnvironment;
  args: Array<Maybe<Scalars['String']>>;
  cooldown: Scalars['Int'];
  description: Scalars['String'];
  disabled: Scalars['Boolean'];
  hcl_id: Scalars['String'];
  id: Scalars['ID'];
  ignoreErrors: Scalars['Boolean'];
  name: Scalars['String'];
  program: Scalars['String'];
  tags?: Maybe<Array<Maybe<LaForgeTagMap>>>;
  timeout: Scalars['Int'];
  vars?: Maybe<Array<Maybe<LaForgeVarsMap>>>;
};

export type LaForgeCompetition = {
  __typename?: 'Competition';
  CompetitionToBuild: Array<Maybe<LaForgeBuild>>;
  CompetitionToEnvironment: LaForgeEnvironment;
  competitionToDNS: Array<Maybe<LaForgeDns>>;
  config?: Maybe<Array<Maybe<LaForgeConfigMap>>>;
  hcl_id: Scalars['String'];
  id: Scalars['ID'];
  root_password: Scalars['String'];
  tags?: Maybe<Array<Maybe<LaForgeTagMap>>>;
};

export type LaForgeDns = {
  __typename?: 'DNS';
  DNSToCompetition: Array<Maybe<LaForgeCompetition>>;
  DNSToEnvironment: Array<Maybe<LaForgeEnvironment>>;
  config?: Maybe<Array<Maybe<LaForgeConfigMap>>>;
  dns_servers: Array<Maybe<Scalars['String']>>;
  hcl_id: Scalars['String'];
  id: Scalars['ID'];
  ntp_servers: Array<Maybe<Scalars['String']>>;
  root_domain: Scalars['String'];
  type: Scalars['String'];
};

export type LaForgeDnsRecord = {
  __typename?: 'DNSRecord';
  DNSRecordToEnvironment: LaForgeEnvironment;
  disabled: Scalars['Boolean'];
  hcl_id: Scalars['String'];
  id: Scalars['ID'];
  name: Scalars['String'];
  tags: Array<Maybe<LaForgeTagMap>>;
  type: Scalars['String'];
  values: Array<Maybe<Scalars['String']>>;
  vars: Array<Maybe<LaForgeVarsMap>>;
  zone: Scalars['String'];
};

export type LaForgeDisk = {
  __typename?: 'Disk';
  DiskToHost: LaForgeHost;
  size: Scalars['Int'];
};

export type LaForgeEnvironment = {
  __typename?: 'Environment';
  EnvironmentToBuild: Array<Maybe<LaForgeBuild>>;
  EnvironmentToCommand: Array<Maybe<LaForgeCommand>>;
  EnvironmentToCompetition: Array<Maybe<LaForgeCompetition>>;
  EnvironmentToDNS: Array<Maybe<LaForgeDns>>;
  EnvironmentToDNSRecord: Array<Maybe<LaForgeDnsRecord>>;
  EnvironmentToFileDelete: Array<Maybe<LaForgeFileDelete>>;
  EnvironmentToFileDownload: Array<Maybe<LaForgeFileDownload>>;
  EnvironmentToFileExtract: Array<Maybe<LaForgeFileExtract>>;
  EnvironmentToHost: Array<Maybe<LaForgeHost>>;
  EnvironmentToIdentity: Array<Maybe<LaForgeIdentity>>;
  EnvironmentToNetwork: Array<Maybe<LaForgeNetwork>>;
  EnvironmentToRepository: Array<Maybe<LaForgeRepository>>;
  EnvironmentToScript: Array<Maybe<LaForgeScript>>;
  EnvironmentToServerTask: Array<Maybe<LaForgeServerTask>>;
  EnvironmentToUser: Array<Maybe<LaForgeUser>>;
  admin_cidrs: Array<Maybe<Scalars['String']>>;
  builder: Scalars['String'];
  competition_id: Scalars['String'];
  config?: Maybe<Array<Maybe<LaForgeConfigMap>>>;
  description: Scalars['String'];
  exposed_vdi_ports: Array<Maybe<Scalars['String']>>;
  hcl_id: Scalars['String'];
  id: Scalars['ID'];
  name: Scalars['String'];
  revision: Scalars['Int'];
  tags?: Maybe<Array<Maybe<LaForgeTagMap>>>;
  team_count: Scalars['Int'];
};

export type LaForgeFileDelete = {
  __typename?: 'FileDelete';
  FileDeleteToEnvironment: LaForgeEnvironment;
  hcl_id: Scalars['String'];
  id: Scalars['ID'];
  path: Scalars['String'];
  tags: Array<Maybe<LaForgeTagMap>>;
};

export type LaForgeFileDownload = {
  __typename?: 'FileDownload';
  FileDownloadToEnvironment: LaForgeEnvironment;
  absPath: Scalars['String'];
  destination: Scalars['String'];
  disabled: Scalars['Boolean'];
  hcl_id: Scalars['String'];
  id: Scalars['ID'];
  md5: Scalars['String'];
  perms: Scalars['String'];
  source: Scalars['String'];
  sourceType: Scalars['String'];
  tags: Array<Maybe<LaForgeTagMap>>;
  template: Scalars['Boolean'];
};

export type LaForgeFileExtract = {
  __typename?: 'FileExtract';
  FileExtractToEnvironment: LaForgeEnvironment;
  destination: Scalars['String'];
  hcl_id: Scalars['String'];
  id: Scalars['ID'];
  source: Scalars['String'];
  tags: Array<Maybe<LaForgeTagMap>>;
  type: Scalars['String'];
};

export type LaForgeFinding = {
  __typename?: 'Finding';
  FindingToEnvironment: LaForgeEnvironment;
  FindingToScript: LaForgeScript;
  FindingToUser: Array<Maybe<LaForgeUser>>;
  description: Scalars['String'];
  difficulty: LaForgeFindingDifficulty;
  name: Scalars['String'];
  severity: LaForgeFindingSeverity;
  tags: Array<Maybe<LaForgeTagMap>>;
};

export enum LaForgeFindingDifficulty {
  AdvancedDifficulty = 'AdvancedDifficulty',
  ExpertDifficulty = 'ExpertDifficulty',
  NoviceDifficulty = 'NoviceDifficulty',
  NullDifficulty = 'NullDifficulty',
  ZeroDifficulty = 'ZeroDifficulty'
}

export enum LaForgeFindingSeverity {
  CriticalSeverity = 'CriticalSeverity',
  HighSeverity = 'HighSeverity',
  LowSeverity = 'LowSeverity',
  MediumSeverity = 'MediumSeverity',
  NullSeverity = 'NullSeverity',
  ZeroSeverity = 'ZeroSeverity'
}

export type LaForgeHost = {
  __typename?: 'Host';
  HostToDisk: LaForgeDisk;
  HostToEnvironment: LaForgeEnvironment;
  OS: Scalars['String'];
  allow_mac_changes: Scalars['Boolean'];
  description: Scalars['String'];
  exposed_tcp_ports: Array<Maybe<Scalars['String']>>;
  exposed_udp_ports: Array<Maybe<Scalars['String']>>;
  hcl_id: Scalars['String'];
  hostname: Scalars['String'];
  id: Scalars['ID'];
  instance_size: Scalars['String'];
  last_octet: Scalars['Int'];
  override_password: Scalars['String'];
  provision_steps: Array<Maybe<Scalars['String']>>;
  tags: Array<Maybe<LaForgeTagMap>>;
  user_groups: Array<Maybe<Scalars['String']>>;
  vars?: Maybe<Array<Maybe<LaForgeVarsMap>>>;
};

export type LaForgeIdentity = {
  __typename?: 'Identity';
  IdentityToEnvironment: LaForgeEnvironment;
  avatar_file: Scalars['String'];
  description: Scalars['String'];
  email: Scalars['String'];
  first_name: Scalars['String'];
  hcl_id: Scalars['String'];
  id: Scalars['ID'];
  last_name: Scalars['String'];
  password: Scalars['String'];
  tags: Array<Maybe<LaForgeTagMap>>;
  vars: Array<Maybe<LaForgeVarsMap>>;
};

export type LaForgeLaForgePageInfo = {
  __typename?: 'LaForgePageInfo';
  nextOffset: Scalars['Int'];
  total: Scalars['Int'];
};

export type LaForgeMutation = {
  __typename?: 'Mutation';
  approveCommit: Scalars['Boolean'];
  cancelBuild: Scalars['Boolean'];
  cancelCommit: Scalars['Boolean'];
  createAgentTasks: Array<Maybe<LaForgeAgentTask>>;
  createBatchAgentTasks: Array<Maybe<LaForgeAgentTask>>;
  createBuild?: Maybe<LaForgeBuild>;
  createEnviromentFromRepo: Array<Maybe<LaForgeEnvironment>>;
  createTask: Scalars['Boolean'];
  createUser?: Maybe<LaForgeAuthUser>;
  deleteBuild: Scalars['String'];
  deleteUser: Scalars['Boolean'];
  dumpBuild: Scalars['String'];
  executePlan?: Maybe<LaForgeBuild>;
  loadEnvironment?: Maybe<Array<Maybe<LaForgeEnvironment>>>;
  modifyAdminPassword: Scalars['Boolean'];
  modifyAdminUserInfo?: Maybe<LaForgeAuthUser>;
  modifySelfPassword: Scalars['Boolean'];
  modifySelfUserInfo?: Maybe<LaForgeAuthUser>;
  nukeBackend: Array<Maybe<LaForgeIntMap>>;
  rebuild: Scalars['Boolean'];
  updateEnviromentViaPull: Array<Maybe<LaForgeEnvironment>>;
};

export type LaForgeMutationApproveCommitArgs = {
  commitUUID: Scalars['String'];
};

export type LaForgeMutationCancelBuildArgs = {
  buildUUID: Scalars['String'];
};

export type LaForgeMutationCancelCommitArgs = {
  commitUUID: Scalars['String'];
};

export type LaForgeMutationCreateAgentTasksArgs = {
  args: Array<Scalars['String']>;
  buildUUID: Scalars['String'];
  command: LaForgeAgentCommand;
  hostHCLID: Scalars['String'];
  teams: Array<Scalars['Int']>;
};

export type LaForgeMutationCreateBatchAgentTasksArgs = {
  args: Array<Scalars['String']>;
  command: LaForgeAgentCommand;
  proHostUUIDs: Array<Scalars['String']>;
};

export type LaForgeMutationCreateBuildArgs = {
  envUUID: Scalars['String'];
  renderFiles?: Scalars['Boolean'];
};

export type LaForgeMutationCreateEnviromentFromRepoArgs = {
  branchName?: Scalars['String'];
  envFilePath: Scalars['String'];
  repoURL: Scalars['String'];
};

export type LaForgeMutationCreateTaskArgs = {
  args: Scalars['String'];
  command: LaForgeAgentCommand;
  proHostUUID: Scalars['String'];
};

export type LaForgeMutationCreateUserArgs = {
  password: Scalars['String'];
  provider: LaForgeProviderType;
  role: LaForgeRoleLevel;
  username: Scalars['String'];
};

export type LaForgeMutationDeleteBuildArgs = {
  buildUUID: Scalars['String'];
};

export type LaForgeMutationDeleteUserArgs = {
  userUUID: Scalars['String'];
};

export type LaForgeMutationDumpBuildArgs = {
  buildUUID: Scalars['String'];
};

export type LaForgeMutationExecutePlanArgs = {
  buildUUID: Scalars['String'];
};

export type LaForgeMutationLoadEnvironmentArgs = {
  envFilePath: Scalars['String'];
};

export type LaForgeMutationModifyAdminPasswordArgs = {
  newPassword: Scalars['String'];
  userID: Scalars['String'];
};

export type LaForgeMutationModifyAdminUserInfoArgs = {
  company?: InputMaybe<Scalars['String']>;
  email?: InputMaybe<Scalars['String']>;
  firstName?: InputMaybe<Scalars['String']>;
  lastName?: InputMaybe<Scalars['String']>;
  occupation?: InputMaybe<Scalars['String']>;
  phone?: InputMaybe<Scalars['String']>;
  provider?: InputMaybe<LaForgeProviderType>;
  role?: InputMaybe<LaForgeRoleLevel>;
  userID: Scalars['String'];
  username?: InputMaybe<Scalars['String']>;
};

export type LaForgeMutationModifySelfPasswordArgs = {
  currentPassword: Scalars['String'];
  newPassword: Scalars['String'];
};

export type LaForgeMutationModifySelfUserInfoArgs = {
  company?: InputMaybe<Scalars['String']>;
  email?: InputMaybe<Scalars['String']>;
  firstName?: InputMaybe<Scalars['String']>;
  lastName?: InputMaybe<Scalars['String']>;
  occupation?: InputMaybe<Scalars['String']>;
  phone?: InputMaybe<Scalars['String']>;
};

export type LaForgeMutationRebuildArgs = {
  rootPlans: Array<InputMaybe<Scalars['String']>>;
};

export type LaForgeMutationUpdateEnviromentViaPullArgs = {
  envUUID: Scalars['String'];
};

export type LaForgeNetwork = {
  __typename?: 'Network';
  NetworkToEnvironment: LaForgeEnvironment;
  cidr: Scalars['String'];
  hcl_id: Scalars['String'];
  id: Scalars['ID'];
  name: Scalars['String'];
  tags: Array<Maybe<LaForgeTagMap>>;
  vars?: Maybe<Array<Maybe<LaForgeVarsMap>>>;
  vdi_visible: Scalars['Boolean'];
};

export type LaForgePlan = {
  __typename?: 'Plan';
  NextPlan: Array<Maybe<LaForgePlan>>;
  PlanToBuild: LaForgeBuild;
  PlanToPlanDiffs: Array<Maybe<LaForgePlanDiff>>;
  PlanToProvisionedHost: LaForgeProvisionedHost;
  PlanToProvisionedNetwork: LaForgeProvisionedNetwork;
  PlanToProvisioningStep: LaForgeProvisioningStep;
  PlanToStatus: LaForgeStatus;
  PlanToTeam: LaForgeTeam;
  PrevPlan: Array<Maybe<LaForgePlan>>;
  build_id: Scalars['String'];
  id: Scalars['ID'];
  step_number: Scalars['Int'];
  type: LaForgePlanType;
};

export type LaForgePlanCounts = {
  __typename?: 'PlanCounts';
  awaiting: Scalars['Int'];
  cancelled: Scalars['Int'];
  complete: Scalars['Int'];
  deleteInProgress: Scalars['Int'];
  deleted: Scalars['Int'];
  failed: Scalars['Int'];
  inProgress: Scalars['Int'];
  parentAwaiting: Scalars['Int'];
  planning: Scalars['Int'];
  tainted: Scalars['Int'];
  toDelete: Scalars['Int'];
  toRebuild: Scalars['Int'];
  undefined: Scalars['Int'];
};

export type LaForgePlanDiff = {
  __typename?: 'PlanDiff';
  PlanDiffToBuildCommit: LaForgeBuildCommit;
  PlanDiffToPlan: LaForgePlan;
  id: Scalars['ID'];
  new_state: LaForgeProvisionStatus;
  revision: Scalars['Int'];
};

export enum LaForgePlanType {
  ExecuteStep = 'execute_step',
  ProvisionHost = 'provision_host',
  ProvisionNetwork = 'provision_network',
  StartBuild = 'start_build',
  StartTeam = 'start_team',
  Undefined = 'undefined'
}

export enum LaForgeProviderType {
  Github = 'GITHUB',
  Local = 'LOCAL',
  Openid = 'OPENID',
  Undefined = 'UNDEFINED'
}

export enum LaForgeProvisionStatus {
  Awaiting = 'AWAITING',
  Cancelled = 'CANCELLED',
  Complete = 'COMPLETE',
  Deleted = 'DELETED',
  Deleteinprogress = 'DELETEINPROGRESS',
  Failed = 'FAILED',
  Inprogress = 'INPROGRESS',
  Parentawaiting = 'PARENTAWAITING',
  Planning = 'PLANNING',
  Tainted = 'TAINTED',
  Todelete = 'TODELETE',
  Torebuild = 'TOREBUILD',
  Undefined = 'UNDEFINED'
}

export enum LaForgeProvisionStatusFor {
  Build = 'Build',
  Plan = 'Plan',
  ProvisionedHost = 'ProvisionedHost',
  ProvisionedNetwork = 'ProvisionedNetwork',
  ProvisioningStep = 'ProvisioningStep',
  Team = 'Team',
  Undefined = 'Undefined'
}

export type LaForgeProvisionedHost = {
  __typename?: 'ProvisionedHost';
  ProvisionedHostToAgentStatus?: Maybe<LaForgeAgentStatus>;
  ProvisionedHostToHost: LaForgeHost;
  ProvisionedHostToPlan: LaForgePlan;
  ProvisionedHostToProvisionedNetwork: LaForgeProvisionedNetwork;
  ProvisionedHostToProvisioningStep: Array<Maybe<LaForgeProvisioningStep>>;
  ProvisionedHostToStatus: LaForgeStatus;
  id: Scalars['ID'];
  subnet_ip: Scalars['String'];
};

export type LaForgeProvisionedNetwork = {
  __typename?: 'ProvisionedNetwork';
  ProvisionedNetworkToBuild: LaForgeBuild;
  ProvisionedNetworkToNetwork: LaForgeNetwork;
  ProvisionedNetworkToPlan: LaForgePlan;
  ProvisionedNetworkToProvisionedHost: Array<Maybe<LaForgeProvisionedHost>>;
  ProvisionedNetworkToStatus: LaForgeStatus;
  ProvisionedNetworkToTeam: LaForgeTeam;
  cidr: Scalars['String'];
  id: Scalars['ID'];
  name: Scalars['String'];
};

export type LaForgeProvisioningStep = {
  __typename?: 'ProvisioningStep';
  ProvisioningStepToCommand?: Maybe<LaForgeCommand>;
  ProvisioningStepToDNSRecord?: Maybe<LaForgeDnsRecord>;
  ProvisioningStepToFileDelete?: Maybe<LaForgeFileDelete>;
  ProvisioningStepToFileDownload?: Maybe<LaForgeFileDownload>;
  ProvisioningStepToFileExtract?: Maybe<LaForgeFileExtract>;
  ProvisioningStepToPlan?: Maybe<LaForgePlan>;
  ProvisioningStepToProvisionedHost: LaForgeProvisionedHost;
  ProvisioningStepToScript?: Maybe<LaForgeScript>;
  ProvisioningStepToStatus: LaForgeStatus;
  id: Scalars['ID'];
  step_number: Scalars['Int'];
  type: LaForgeProvisioningStepType;
};

export enum LaForgeProvisioningStepType {
  Ansible = 'Ansible',
  Command = 'Command',
  DnsRecord = 'DNSRecord',
  FileDelete = 'FileDelete',
  FileDownload = 'FileDownload',
  FileExtract = 'FileExtract',
  Script = 'Script',
  Undefined = 'Undefined'
}

export type LaForgeQuery = {
  __typename?: 'Query';
  agentStatus?: Maybe<LaForgeAgentStatus>;
  build?: Maybe<LaForgeBuild>;
  currentUser?: Maybe<LaForgeAuthUser>;
  environment?: Maybe<LaForgeEnvironment>;
  environments?: Maybe<Array<Maybe<LaForgeEnvironment>>>;
  getAgentTasks?: Maybe<Array<Maybe<LaForgeAgentTask>>>;
  getAllAgentStatus?: Maybe<LaForgeAgentStatusBatch>;
  getAllPlanStatus?: Maybe<LaForgeStatusBatch>;
  getBuildCommit?: Maybe<LaForgeBuildCommit>;
  getBuildCommits?: Maybe<Array<Maybe<LaForgeBuildCommit>>>;
  getBuilds?: Maybe<Array<Maybe<LaForgeBuild>>>;
  getCurrentUserTasks?: Maybe<Array<Maybe<LaForgeServerTask>>>;
  getPlanStatusCounts: LaForgePlanCounts;
  getServerTasks?: Maybe<Array<Maybe<LaForgeServerTask>>>;
  getUserList?: Maybe<Array<Maybe<LaForgeAuthUser>>>;
  listAgentStatuses?: Maybe<Array<Maybe<LaForgeAgentStatus>>>;
  listBuildStatuses?: Maybe<Array<Maybe<LaForgeStatus>>>;
  plan?: Maybe<LaForgePlan>;
  provisionedHost?: Maybe<LaForgeProvisionedHost>;
  provisionedNetwork?: Maybe<LaForgeProvisionedNetwork>;
  provisionedStep?: Maybe<LaForgeProvisioningStep>;
  serverTasks?: Maybe<Array<Maybe<LaForgeServerTask>>>;
  status?: Maybe<LaForgeStatus>;
  viewAgentTask: LaForgeAgentTask;
  viewServerTaskLogs: Scalars['String'];
};

export type LaForgeQueryAgentStatusArgs = {
  clientId: Scalars['String'];
};

export type LaForgeQueryBuildArgs = {
  buildUUID: Scalars['String'];
};

export type LaForgeQueryEnvironmentArgs = {
  envUUID: Scalars['String'];
};

export type LaForgeQueryGetAgentTasksArgs = {
  proStepUUID: Scalars['String'];
};

export type LaForgeQueryGetAllAgentStatusArgs = {
  buildUUID: Scalars['String'];
  count: Scalars['Int'];
  offset: Scalars['Int'];
};

export type LaForgeQueryGetAllPlanStatusArgs = {
  buildUUID: Scalars['String'];
  count: Scalars['Int'];
  offset: Scalars['Int'];
};

export type LaForgeQueryGetBuildCommitArgs = {
  buildCommitUUID: Scalars['String'];
};

export type LaForgeQueryGetBuildCommitsArgs = {
  envUUID: Scalars['String'];
};

export type LaForgeQueryGetPlanStatusCountsArgs = {
  buildUUID: Scalars['String'];
};

export type LaForgeQueryListAgentStatusesArgs = {
  buildUUID: Scalars['String'];
};

export type LaForgeQueryListBuildStatusesArgs = {
  buildUUID: Scalars['String'];
};

export type LaForgeQueryPlanArgs = {
  planUUID: Scalars['String'];
};

export type LaForgeQueryProvisionedHostArgs = {
  proHostUUID: Scalars['String'];
};

export type LaForgeQueryProvisionedNetworkArgs = {
  proNetUUID: Scalars['String'];
};

export type LaForgeQueryProvisionedStepArgs = {
  proStepUUID: Scalars['String'];
};

export type LaForgeQueryServerTasksArgs = {
  taskUUIDs: Array<InputMaybe<Scalars['String']>>;
};

export type LaForgeQueryStatusArgs = {
  statusUUID: Scalars['String'];
};

export type LaForgeQueryViewAgentTaskArgs = {
  taskID: Scalars['String'];
};

export type LaForgeQueryViewServerTaskLogsArgs = {
  taskID: Scalars['String'];
};

export type LaForgeRepoCommit = {
  __typename?: 'RepoCommit';
  RepoCommitToRepository: LaForgeRepository;
  author: Scalars['String'];
  committer: Scalars['String'];
  hash: Scalars['String'];
  id: Scalars['ID'];
  message: Scalars['String'];
  parent_hashes: Array<Maybe<Scalars['String']>>;
  pgp_signature: Scalars['String'];
  revision: Scalars['Int'];
  tree_hash: Scalars['String'];
};

export type LaForgeRepository = {
  __typename?: 'Repository';
  RepositoryToRepoCommit: Array<Maybe<LaForgeRepoCommit>>;
  branch_name: Scalars['String'];
  environment_filepath: Scalars['String'];
  id: Scalars['ID'];
  repo_url: Scalars['String'];
};

export enum LaForgeRoleLevel {
  Admin = 'ADMIN',
  Undefined = 'UNDEFINED',
  User = 'USER'
}

export type LaForgeScript = {
  __typename?: 'Script';
  ScriptToEnvironment: LaForgeEnvironment;
  absPath: Scalars['String'];
  args: Array<Maybe<Scalars['String']>>;
  cooldown: Scalars['Int'];
  description: Scalars['String'];
  disabled: Scalars['Boolean'];
  hcl_id: Scalars['String'];
  id: Scalars['ID'];
  ignore_errors: Scalars['Boolean'];
  language: Scalars['String'];
  name: Scalars['String'];
  scriptToFinding: Array<Maybe<LaForgeFinding>>;
  source: Scalars['String'];
  source_type: Scalars['String'];
  tags?: Maybe<Array<Maybe<LaForgeTagMap>>>;
  timeout: Scalars['Int'];
  vars?: Maybe<Array<Maybe<LaForgeVarsMap>>>;
};

export type LaForgeServerTask = {
  __typename?: 'ServerTask';
  ServerTaskToAuthUser: LaForgeAuthUser;
  ServerTaskToBuild?: Maybe<LaForgeBuild>;
  ServerTaskToBuildCommit?: Maybe<LaForgeBuildCommit>;
  ServerTaskToEnvironment?: Maybe<LaForgeEnvironment>;
  ServerTaskToStatus: LaForgeStatus;
  end_time?: Maybe<Scalars['Time']>;
  errors?: Maybe<Array<Maybe<Scalars['String']>>>;
  id: Scalars['ID'];
  log_file_path?: Maybe<Scalars['String']>;
  start_time?: Maybe<Scalars['Time']>;
  type: LaForgeServerTaskType;
};

export enum LaForgeServerTaskType {
  Createbuild = 'CREATEBUILD',
  Deletebuild = 'DELETEBUILD',
  Executebuild = 'EXECUTEBUILD',
  Loadenv = 'LOADENV',
  Rebuild = 'REBUILD',
  Renderfiles = 'RENDERFILES'
}

export type LaForgeStatus = {
  __typename?: 'Status';
  completed: Scalars['Boolean'];
  ended_at: Scalars['String'];
  error?: Maybe<Scalars['String']>;
  failed: Scalars['Boolean'];
  id: Scalars['ID'];
  started_at: Scalars['String'];
  state: LaForgeProvisionStatus;
  status_for: LaForgeProvisionStatusFor;
};

export type LaForgeStatusBatch = {
  __typename?: 'StatusBatch';
  pageInfo: LaForgeLaForgePageInfo;
  statuses: Array<Maybe<LaForgeStatus>>;
};

export type LaForgeSubscription = {
  __typename?: 'Subscription';
  streamServerTaskLog: Scalars['String'];
  updatedAgentStatus: LaForgeAgentStatus;
  updatedAgentTask: LaForgeAgentTask;
  updatedBuild: LaForgeBuild;
  updatedCommit: LaForgeBuildCommit;
  updatedServerTask: LaForgeServerTask;
  updatedStatus: LaForgeStatus;
};

export type LaForgeSubscriptionStreamServerTaskLogArgs = {
  taskID: Scalars['String'];
};

export type LaForgeTeam = {
  __typename?: 'Team';
  TeamToBuild: LaForgeBuild;
  TeamToPlan: LaForgePlan;
  TeamToProvisionedNetwork: Array<Maybe<LaForgeProvisionedNetwork>>;
  TeamToStatus: LaForgeStatus;
  id: Scalars['ID'];
  team_number: Scalars['Int'];
};

export type LaForgeUser = {
  __typename?: 'User';
  email: Scalars['String'];
  id: Scalars['ID'];
  name: Scalars['String'];
  uuid: Scalars['String'];
};

export type LaForgeConfigMap = {
  __typename?: 'configMap';
  key: Scalars['String'];
  value: Scalars['String'];
};

export type LaForgeIntMap = {
  __typename?: 'intMap';
  key: Scalars['String'];
  value: Scalars['Int'];
};

export type LaForgeTagMap = {
  __typename?: 'tagMap';
  key: Scalars['String'];
  value: Scalars['String'];
};

export type LaForgeVarsMap = {
  __typename?: 'varsMap';
  key: Scalars['String'];
  value: Scalars['String'];
};

export type LaForgeGetUserListQueryVariables = Exact<{ [key: string]: never }>;

export type LaForgeGetUserListQuery = {
  __typename?: 'Query';
  getUserList?: Array<{
    __typename?: 'AuthUser';
    id: string;
    first_name: string;
    last_name: string;
    username: string;
    provider: LaForgeProviderType;
    role: LaForgeRoleLevel;
    email: string;
    phone: string;
    company: string;
    occupation: string;
  } | null> | null;
};

export type LaForgeGetCurrentUserQueryVariables = Exact<{ [key: string]: never }>;

export type LaForgeGetCurrentUserQuery = {
  __typename?: 'Query';
  currentUser?: {
    __typename?: 'AuthUser';
    id: string;
    first_name: string;
    last_name: string;
    username: string;
    provider: LaForgeProviderType;
    role: LaForgeRoleLevel;
    email: string;
    phone: string;
    company: string;
    occupation: string;
  } | null;
};

export type LaForgeAuthUserFieldsFragment = {
  __typename?: 'AuthUser';
  id: string;
  first_name: string;
  last_name: string;
  username: string;
  provider: LaForgeProviderType;
  role: LaForgeRoleLevel;
  email: string;
  phone: string;
  company: string;
  occupation: string;
};

export const AuthUserFieldsFragmentDoc = gql`
  fragment AuthUserFields on AuthUser {
    id
    first_name
    last_name
    username
    provider
    role
    email
    phone
    company
    occupation
  }
`;
export const GetUserListDocument = gql`
  query GetUserList {
    getUserList {
      ...AuthUserFields
    }
  }
  ${AuthUserFieldsFragmentDoc}
`;

@Injectable({
  providedIn: 'root'
})
export class LaForgeGetUserListGQL extends Apollo.Query<LaForgeGetUserListQuery, LaForgeGetUserListQueryVariables> {
  document = GetUserListDocument;

  constructor(apollo: Apollo.Apollo) {
    super(apollo);
  }
}
export const GetCurrentUserDocument = gql`
  query GetCurrentUser {
    currentUser {
      ...AuthUserFields
    }
  }
  ${AuthUserFieldsFragmentDoc}
`;

@Injectable({
  providedIn: 'root'
})
export class LaForgeGetCurrentUserGQL extends Apollo.Query<LaForgeGetCurrentUserQuery, LaForgeGetCurrentUserQueryVariables> {
  document = GetCurrentUserDocument;

  constructor(apollo: Apollo.Apollo) {
    super(apollo);
  }
}

export const MockAgentStatus = (overrides?: Partial<LaForgeAgentStatus>): { __typename: 'AgentStatus' } & LaForgeAgentStatus => {
  return {
    __typename: 'AgentStatus',
    OS: overrides && overrides.hasOwnProperty('OS') ? overrides.OS! : 'ut',
    bootTime: overrides && overrides.hasOwnProperty('bootTime') ? overrides.bootTime! : 4263,
    clientId: overrides && overrides.hasOwnProperty('clientId') ? overrides.clientId! : 'quisquam',
    freeMem: overrides && overrides.hasOwnProperty('freeMem') ? overrides.freeMem! : 2366,
    hostID: overrides && overrides.hasOwnProperty('hostID') ? overrides.hostID! : 'quae',
    hostname: overrides && overrides.hasOwnProperty('hostname') ? overrides.hostname! : 'labore',
    load1: overrides && overrides.hasOwnProperty('load1') ? overrides.load1! : 9.23,
    load5: overrides && overrides.hasOwnProperty('load5') ? overrides.load5! : 6.96,
    load15: overrides && overrides.hasOwnProperty('load15') ? overrides.load15! : 4.9,
    numProcs: overrides && overrides.hasOwnProperty('numProcs') ? overrides.numProcs! : 3258,
    timestamp: overrides && overrides.hasOwnProperty('timestamp') ? overrides.timestamp! : 718,
    totalMem: overrides && overrides.hasOwnProperty('totalMem') ? overrides.totalMem! : 9460,
    upTime: overrides && overrides.hasOwnProperty('upTime') ? overrides.upTime! : 4442,
    usedMem: overrides && overrides.hasOwnProperty('usedMem') ? overrides.usedMem! : 3697
  };
};

export const MockAgentStatusBatch = (
  overrides?: Partial<LaForgeAgentStatusBatch>
): { __typename: 'AgentStatusBatch' } & LaForgeAgentStatusBatch => {
  return {
    __typename: 'AgentStatusBatch',
    agentStatuses: overrides && overrides.hasOwnProperty('agentStatuses') ? overrides.agentStatuses! : [MockAgentStatus()],
    pageInfo: overrides && overrides.hasOwnProperty('pageInfo') ? overrides.pageInfo! : MockLaForgePageInfo()
  };
};

export const MockAgentTask = (overrides?: Partial<LaForgeAgentTask>): { __typename: 'AgentTask' } & LaForgeAgentTask => {
  return {
    __typename: 'AgentTask',
    args: overrides && overrides.hasOwnProperty('args') ? overrides.args! : 'assumenda',
    command: overrides && overrides.hasOwnProperty('command') ? overrides.command! : LaForgeAgentCommand.Addtogroup,
    error_message: overrides && overrides.hasOwnProperty('error_message') ? overrides.error_message! : 'magni',
    id: overrides && overrides.hasOwnProperty('id') ? overrides.id! : '49473e47-7e7f-4b73-9cc6-cbbd0fedd13f',
    number: overrides && overrides.hasOwnProperty('number') ? overrides.number! : 2536,
    output: overrides && overrides.hasOwnProperty('output') ? overrides.output! : 'occaecati',
    state: overrides && overrides.hasOwnProperty('state') ? overrides.state! : LaForgeAgentTaskState.Awaiting
  };
};

export const MockAuthUser = (overrides?: Partial<LaForgeAuthUser>): { __typename: 'AuthUser' } & LaForgeAuthUser => {
  return {
    __typename: 'AuthUser',
    company: overrides && overrides.hasOwnProperty('company') ? overrides.company! : 'rem',
    email: overrides && overrides.hasOwnProperty('email') ? overrides.email! : 'animi',
    first_name: overrides && overrides.hasOwnProperty('first_name') ? overrides.first_name! : 'impedit',
    id: overrides && overrides.hasOwnProperty('id') ? overrides.id! : 'e8ab6459-2393-4a0a-90fe-c768dbd70086',
    last_name: overrides && overrides.hasOwnProperty('last_name') ? overrides.last_name! : 'dolor',
    occupation: overrides && overrides.hasOwnProperty('occupation') ? overrides.occupation! : 'dolorem',
    phone: overrides && overrides.hasOwnProperty('phone') ? overrides.phone! : 'autem',
    provider: overrides && overrides.hasOwnProperty('provider') ? overrides.provider! : LaForgeProviderType.Github,
    publicKey: overrides && overrides.hasOwnProperty('publicKey') ? overrides.publicKey! : 'qui',
    role: overrides && overrides.hasOwnProperty('role') ? overrides.role! : LaForgeRoleLevel.Admin,
    username: overrides && overrides.hasOwnProperty('username') ? overrides.username! : 'vel'
  };
};

export const MockBuild = (overrides?: Partial<LaForgeBuild>): { __typename: 'Build' } & LaForgeBuild => {
  return {
    __typename: 'Build',
    BuildToBuildCommits:
      overrides && overrides.hasOwnProperty('BuildToBuildCommits') ? overrides.BuildToBuildCommits! : [MockBuildCommit()],
    BuildToLatestBuildCommit:
      overrides && overrides.hasOwnProperty('BuildToLatestBuildCommit') ? overrides.BuildToLatestBuildCommit! : MockBuildCommit(),
    BuildToRepoCommit: overrides && overrides.hasOwnProperty('BuildToRepoCommit') ? overrides.BuildToRepoCommit! : MockRepoCommit(),
    BuildToServerTasks: overrides && overrides.hasOwnProperty('BuildToServerTasks') ? overrides.BuildToServerTasks! : [MockServerTask()],
    buildToCompetition: overrides && overrides.hasOwnProperty('buildToCompetition') ? overrides.buildToCompetition! : MockCompetition(),
    buildToEnvironment: overrides && overrides.hasOwnProperty('buildToEnvironment') ? overrides.buildToEnvironment! : MockEnvironment(),
    buildToPlan: overrides && overrides.hasOwnProperty('buildToPlan') ? overrides.buildToPlan! : [MockPlan()],
    buildToProvisionedNetwork:
      overrides && overrides.hasOwnProperty('buildToProvisionedNetwork')
        ? overrides.buildToProvisionedNetwork!
        : [MockProvisionedNetwork()],
    buildToStatus: overrides && overrides.hasOwnProperty('buildToStatus') ? overrides.buildToStatus! : MockStatus(),
    buildToTeam: overrides && overrides.hasOwnProperty('buildToTeam') ? overrides.buildToTeam! : [MockTeam()],
    completed_plan: overrides && overrides.hasOwnProperty('completed_plan') ? overrides.completed_plan! : false,
    environment_revision: overrides && overrides.hasOwnProperty('environment_revision') ? overrides.environment_revision! : 8218,
    id: overrides && overrides.hasOwnProperty('id') ? overrides.id! : '894b64ef-c970-422f-a7c6-dae16c23beb4',
    revision: overrides && overrides.hasOwnProperty('revision') ? overrides.revision! : 442
  };
};

export const MockBuildCommit = (overrides?: Partial<LaForgeBuildCommit>): { __typename: 'BuildCommit' } & LaForgeBuildCommit => {
  return {
    __typename: 'BuildCommit',
    BuildCommitToBuild: overrides && overrides.hasOwnProperty('BuildCommitToBuild') ? overrides.BuildCommitToBuild! : MockBuild(),
    BuildCommitToPlanDiffs:
      overrides && overrides.hasOwnProperty('BuildCommitToPlanDiffs') ? overrides.BuildCommitToPlanDiffs! : [MockPlanDiff()],
    BuildCommitToServerTask:
      overrides && overrides.hasOwnProperty('BuildCommitToServerTask') ? overrides.BuildCommitToServerTask! : [MockServerTask()],
    createdAt: overrides && overrides.hasOwnProperty('createdAt') ? overrides.createdAt! : 'ipsam',
    id: overrides && overrides.hasOwnProperty('id') ? overrides.id! : '70817adc-926d-4c0b-8848-35b422609037',
    revision: overrides && overrides.hasOwnProperty('revision') ? overrides.revision! : 4675,
    state: overrides && overrides.hasOwnProperty('state') ? overrides.state! : LaForgeBuildCommitState.Applied,
    type: overrides && overrides.hasOwnProperty('type') ? overrides.type! : LaForgeBuildCommitType.Delete
  };
};

export const MockCommand = (overrides?: Partial<LaForgeCommand>): { __typename: 'Command' } & LaForgeCommand => {
  return {
    __typename: 'Command',
    CommandToEnvironment:
      overrides && overrides.hasOwnProperty('CommandToEnvironment') ? overrides.CommandToEnvironment! : MockEnvironment(),
    args: overrides && overrides.hasOwnProperty('args') ? overrides.args! : ['rerum'],
    cooldown: overrides && overrides.hasOwnProperty('cooldown') ? overrides.cooldown! : 9449,
    description: overrides && overrides.hasOwnProperty('description') ? overrides.description! : 'non',
    disabled: overrides && overrides.hasOwnProperty('disabled') ? overrides.disabled! : false,
    hcl_id: overrides && overrides.hasOwnProperty('hcl_id') ? overrides.hcl_id! : 'error',
    id: overrides && overrides.hasOwnProperty('id') ? overrides.id! : '382d58ee-6625-4a44-b765-d3c6cd3b70b7',
    ignoreErrors: overrides && overrides.hasOwnProperty('ignoreErrors') ? overrides.ignoreErrors! : false,
    name: overrides && overrides.hasOwnProperty('name') ? overrides.name! : 'eum',
    program: overrides && overrides.hasOwnProperty('program') ? overrides.program! : 'qui',
    tags: overrides && overrides.hasOwnProperty('tags') ? overrides.tags! : [MockTagMap()],
    timeout: overrides && overrides.hasOwnProperty('timeout') ? overrides.timeout! : 3759,
    vars: overrides && overrides.hasOwnProperty('vars') ? overrides.vars! : [MockVarsMap()]
  };
};

export const MockCompetition = (overrides?: Partial<LaForgeCompetition>): { __typename: 'Competition' } & LaForgeCompetition => {
  return {
    __typename: 'Competition',
    CompetitionToBuild: overrides && overrides.hasOwnProperty('CompetitionToBuild') ? overrides.CompetitionToBuild! : [MockBuild()],
    CompetitionToEnvironment:
      overrides && overrides.hasOwnProperty('CompetitionToEnvironment') ? overrides.CompetitionToEnvironment! : MockEnvironment(),
    competitionToDNS: overrides && overrides.hasOwnProperty('competitionToDNS') ? overrides.competitionToDNS! : [MockDns()],
    config: overrides && overrides.hasOwnProperty('config') ? overrides.config! : [MockConfigMap()],
    hcl_id: overrides && overrides.hasOwnProperty('hcl_id') ? overrides.hcl_id! : 'fugit',
    id: overrides && overrides.hasOwnProperty('id') ? overrides.id! : 'a33c2381-2e53-40e3-a7b3-b7bb83ed3464',
    root_password: overrides && overrides.hasOwnProperty('root_password') ? overrides.root_password! : 'beatae',
    tags: overrides && overrides.hasOwnProperty('tags') ? overrides.tags! : [MockTagMap()]
  };
};

export const MockDns = (overrides?: Partial<LaForgeDns>): { __typename: 'DNS' } & LaForgeDns => {
  return {
    __typename: 'DNS',
    DNSToCompetition: overrides && overrides.hasOwnProperty('DNSToCompetition') ? overrides.DNSToCompetition! : [MockCompetition()],
    DNSToEnvironment: overrides && overrides.hasOwnProperty('DNSToEnvironment') ? overrides.DNSToEnvironment! : [MockEnvironment()],
    config: overrides && overrides.hasOwnProperty('config') ? overrides.config! : [MockConfigMap()],
    dns_servers: overrides && overrides.hasOwnProperty('dns_servers') ? overrides.dns_servers! : ['tempore'],
    hcl_id: overrides && overrides.hasOwnProperty('hcl_id') ? overrides.hcl_id! : 'est',
    id: overrides && overrides.hasOwnProperty('id') ? overrides.id! : 'a4246799-9224-4123-80f6-80d4297bd007',
    ntp_servers: overrides && overrides.hasOwnProperty('ntp_servers') ? overrides.ntp_servers! : ['iure'],
    root_domain: overrides && overrides.hasOwnProperty('root_domain') ? overrides.root_domain! : 'itaque',
    type: overrides && overrides.hasOwnProperty('type') ? overrides.type! : 'quia'
  };
};

export const MockDnsRecord = (overrides?: Partial<LaForgeDnsRecord>): { __typename: 'DNSRecord' } & LaForgeDnsRecord => {
  return {
    __typename: 'DNSRecord',
    DNSRecordToEnvironment:
      overrides && overrides.hasOwnProperty('DNSRecordToEnvironment') ? overrides.DNSRecordToEnvironment! : MockEnvironment(),
    disabled: overrides && overrides.hasOwnProperty('disabled') ? overrides.disabled! : false,
    hcl_id: overrides && overrides.hasOwnProperty('hcl_id') ? overrides.hcl_id! : 'tempore',
    id: overrides && overrides.hasOwnProperty('id') ? overrides.id! : '50b9a2f6-a83a-4fbd-aab8-fc59a56bd547',
    name: overrides && overrides.hasOwnProperty('name') ? overrides.name! : 'beatae',
    tags: overrides && overrides.hasOwnProperty('tags') ? overrides.tags! : [MockTagMap()],
    type: overrides && overrides.hasOwnProperty('type') ? overrides.type! : 'fugiat',
    values: overrides && overrides.hasOwnProperty('values') ? overrides.values! : ['aliquam'],
    vars: overrides && overrides.hasOwnProperty('vars') ? overrides.vars! : [MockVarsMap()],
    zone: overrides && overrides.hasOwnProperty('zone') ? overrides.zone! : 'quo'
  };
};

export const MockDisk = (overrides?: Partial<LaForgeDisk>): { __typename: 'Disk' } & LaForgeDisk => {
  return {
    __typename: 'Disk',
    DiskToHost: overrides && overrides.hasOwnProperty('DiskToHost') ? overrides.DiskToHost! : MockHost(),
    size: overrides && overrides.hasOwnProperty('size') ? overrides.size! : 7704
  };
};

export const MockEnvironment = (overrides?: Partial<LaForgeEnvironment>): { __typename: 'Environment' } & LaForgeEnvironment => {
  return {
    __typename: 'Environment',
    EnvironmentToBuild: overrides && overrides.hasOwnProperty('EnvironmentToBuild') ? overrides.EnvironmentToBuild! : [MockBuild()],
    EnvironmentToCommand: overrides && overrides.hasOwnProperty('EnvironmentToCommand') ? overrides.EnvironmentToCommand! : [MockCommand()],
    EnvironmentToCompetition:
      overrides && overrides.hasOwnProperty('EnvironmentToCompetition') ? overrides.EnvironmentToCompetition! : [MockCompetition()],
    EnvironmentToDNS: overrides && overrides.hasOwnProperty('EnvironmentToDNS') ? overrides.EnvironmentToDNS! : [MockDns()],
    EnvironmentToDNSRecord:
      overrides && overrides.hasOwnProperty('EnvironmentToDNSRecord') ? overrides.EnvironmentToDNSRecord! : [MockDnsRecord()],
    EnvironmentToFileDelete:
      overrides && overrides.hasOwnProperty('EnvironmentToFileDelete') ? overrides.EnvironmentToFileDelete! : [MockFileDelete()],
    EnvironmentToFileDownload:
      overrides && overrides.hasOwnProperty('EnvironmentToFileDownload') ? overrides.EnvironmentToFileDownload! : [MockFileDownload()],
    EnvironmentToFileExtract:
      overrides && overrides.hasOwnProperty('EnvironmentToFileExtract') ? overrides.EnvironmentToFileExtract! : [MockFileExtract()],
    EnvironmentToHost: overrides && overrides.hasOwnProperty('EnvironmentToHost') ? overrides.EnvironmentToHost! : [MockHost()],
    EnvironmentToIdentity:
      overrides && overrides.hasOwnProperty('EnvironmentToIdentity') ? overrides.EnvironmentToIdentity! : [MockIdentity()],
    EnvironmentToNetwork: overrides && overrides.hasOwnProperty('EnvironmentToNetwork') ? overrides.EnvironmentToNetwork! : [MockNetwork()],
    EnvironmentToRepository:
      overrides && overrides.hasOwnProperty('EnvironmentToRepository') ? overrides.EnvironmentToRepository! : [MockRepository()],
    EnvironmentToScript: overrides && overrides.hasOwnProperty('EnvironmentToScript') ? overrides.EnvironmentToScript! : [MockScript()],
    EnvironmentToServerTask:
      overrides && overrides.hasOwnProperty('EnvironmentToServerTask') ? overrides.EnvironmentToServerTask! : [MockServerTask()],
    EnvironmentToUser: overrides && overrides.hasOwnProperty('EnvironmentToUser') ? overrides.EnvironmentToUser! : [MockUser()],
    admin_cidrs: overrides && overrides.hasOwnProperty('admin_cidrs') ? overrides.admin_cidrs! : ['molestiae'],
    builder: overrides && overrides.hasOwnProperty('builder') ? overrides.builder! : 'aut',
    competition_id: overrides && overrides.hasOwnProperty('competition_id') ? overrides.competition_id! : 'asperiores',
    config: overrides && overrides.hasOwnProperty('config') ? overrides.config! : [MockConfigMap()],
    description: overrides && overrides.hasOwnProperty('description') ? overrides.description! : 'omnis',
    exposed_vdi_ports: overrides && overrides.hasOwnProperty('exposed_vdi_ports') ? overrides.exposed_vdi_ports! : ['animi'],
    hcl_id: overrides && overrides.hasOwnProperty('hcl_id') ? overrides.hcl_id! : 'molestiae',
    id: overrides && overrides.hasOwnProperty('id') ? overrides.id! : 'dff32e77-b6cb-47ad-a902-36f6bfdfe374',
    name: overrides && overrides.hasOwnProperty('name') ? overrides.name! : 'a',
    revision: overrides && overrides.hasOwnProperty('revision') ? overrides.revision! : 2495,
    tags: overrides && overrides.hasOwnProperty('tags') ? overrides.tags! : [MockTagMap()],
    team_count: overrides && overrides.hasOwnProperty('team_count') ? overrides.team_count! : 4849
  };
};

export const MockFileDelete = (overrides?: Partial<LaForgeFileDelete>): { __typename: 'FileDelete' } & LaForgeFileDelete => {
  return {
    __typename: 'FileDelete',
    FileDeleteToEnvironment:
      overrides && overrides.hasOwnProperty('FileDeleteToEnvironment') ? overrides.FileDeleteToEnvironment! : MockEnvironment(),
    hcl_id: overrides && overrides.hasOwnProperty('hcl_id') ? overrides.hcl_id! : 'facilis',
    id: overrides && overrides.hasOwnProperty('id') ? overrides.id! : '5b6e6359-4479-43db-88b2-596effdd69dd',
    path: overrides && overrides.hasOwnProperty('path') ? overrides.path! : 'laboriosam',
    tags: overrides && overrides.hasOwnProperty('tags') ? overrides.tags! : [MockTagMap()]
  };
};

export const MockFileDownload = (overrides?: Partial<LaForgeFileDownload>): { __typename: 'FileDownload' } & LaForgeFileDownload => {
  return {
    __typename: 'FileDownload',
    FileDownloadToEnvironment:
      overrides && overrides.hasOwnProperty('FileDownloadToEnvironment') ? overrides.FileDownloadToEnvironment! : MockEnvironment(),
    absPath: overrides && overrides.hasOwnProperty('absPath') ? overrides.absPath! : 'quaerat',
    destination: overrides && overrides.hasOwnProperty('destination') ? overrides.destination! : 'quia',
    disabled: overrides && overrides.hasOwnProperty('disabled') ? overrides.disabled! : false,
    hcl_id: overrides && overrides.hasOwnProperty('hcl_id') ? overrides.hcl_id! : 'tempora',
    id: overrides && overrides.hasOwnProperty('id') ? overrides.id! : '6d0b00b4-fa51-4158-b73c-99732bcf288a',
    md5: overrides && overrides.hasOwnProperty('md5') ? overrides.md5! : 'iste',
    perms: overrides && overrides.hasOwnProperty('perms') ? overrides.perms! : 'odit',
    source: overrides && overrides.hasOwnProperty('source') ? overrides.source! : 'consequatur',
    sourceType: overrides && overrides.hasOwnProperty('sourceType') ? overrides.sourceType! : 'quo',
    tags: overrides && overrides.hasOwnProperty('tags') ? overrides.tags! : [MockTagMap()],
    template: overrides && overrides.hasOwnProperty('template') ? overrides.template! : true
  };
};

export const MockFileExtract = (overrides?: Partial<LaForgeFileExtract>): { __typename: 'FileExtract' } & LaForgeFileExtract => {
  return {
    __typename: 'FileExtract',
    FileExtractToEnvironment:
      overrides && overrides.hasOwnProperty('FileExtractToEnvironment') ? overrides.FileExtractToEnvironment! : MockEnvironment(),
    destination: overrides && overrides.hasOwnProperty('destination') ? overrides.destination! : 'ab',
    hcl_id: overrides && overrides.hasOwnProperty('hcl_id') ? overrides.hcl_id! : 'eos',
    id: overrides && overrides.hasOwnProperty('id') ? overrides.id! : '132c556e-acf6-462e-aa4c-7feaec2fb40b',
    source: overrides && overrides.hasOwnProperty('source') ? overrides.source! : 'libero',
    tags: overrides && overrides.hasOwnProperty('tags') ? overrides.tags! : [MockTagMap()],
    type: overrides && overrides.hasOwnProperty('type') ? overrides.type! : 'quis'
  };
};

export const MockFinding = (overrides?: Partial<LaForgeFinding>): { __typename: 'Finding' } & LaForgeFinding => {
  return {
    __typename: 'Finding',
    FindingToEnvironment:
      overrides && overrides.hasOwnProperty('FindingToEnvironment') ? overrides.FindingToEnvironment! : MockEnvironment(),
    FindingToScript: overrides && overrides.hasOwnProperty('FindingToScript') ? overrides.FindingToScript! : MockScript(),
    FindingToUser: overrides && overrides.hasOwnProperty('FindingToUser') ? overrides.FindingToUser! : [MockUser()],
    description: overrides && overrides.hasOwnProperty('description') ? overrides.description! : 'quaerat',
    difficulty: overrides && overrides.hasOwnProperty('difficulty') ? overrides.difficulty! : LaForgeFindingDifficulty.AdvancedDifficulty,
    name: overrides && overrides.hasOwnProperty('name') ? overrides.name! : 'ea',
    severity: overrides && overrides.hasOwnProperty('severity') ? overrides.severity! : LaForgeFindingSeverity.CriticalSeverity,
    tags: overrides && overrides.hasOwnProperty('tags') ? overrides.tags! : [MockTagMap()]
  };
};

export const MockHost = (overrides?: Partial<LaForgeHost>): { __typename: 'Host' } & LaForgeHost => {
  return {
    __typename: 'Host',
    HostToDisk: overrides && overrides.hasOwnProperty('HostToDisk') ? overrides.HostToDisk! : MockDisk(),
    HostToEnvironment: overrides && overrides.hasOwnProperty('HostToEnvironment') ? overrides.HostToEnvironment! : MockEnvironment(),
    OS: overrides && overrides.hasOwnProperty('OS') ? overrides.OS! : 'ab',
    allow_mac_changes: overrides && overrides.hasOwnProperty('allow_mac_changes') ? overrides.allow_mac_changes! : false,
    description: overrides && overrides.hasOwnProperty('description') ? overrides.description! : 'autem',
    exposed_tcp_ports: overrides && overrides.hasOwnProperty('exposed_tcp_ports') ? overrides.exposed_tcp_ports! : ['accusantium'],
    exposed_udp_ports: overrides && overrides.hasOwnProperty('exposed_udp_ports') ? overrides.exposed_udp_ports! : ['autem'],
    hcl_id: overrides && overrides.hasOwnProperty('hcl_id') ? overrides.hcl_id! : 'reiciendis',
    hostname: overrides && overrides.hasOwnProperty('hostname') ? overrides.hostname! : 'blanditiis',
    id: overrides && overrides.hasOwnProperty('id') ? overrides.id! : '0e069d27-59c9-4206-8cfb-2ce08e75bcf1',
    instance_size: overrides && overrides.hasOwnProperty('instance_size') ? overrides.instance_size! : 'aliquid',
    last_octet: overrides && overrides.hasOwnProperty('last_octet') ? overrides.last_octet! : 4400,
    override_password: overrides && overrides.hasOwnProperty('override_password') ? overrides.override_password! : 'qui',
    provision_steps: overrides && overrides.hasOwnProperty('provision_steps') ? overrides.provision_steps! : ['ea'],
    tags: overrides && overrides.hasOwnProperty('tags') ? overrides.tags! : [MockTagMap()],
    user_groups: overrides && overrides.hasOwnProperty('user_groups') ? overrides.user_groups! : ['omnis'],
    vars: overrides && overrides.hasOwnProperty('vars') ? overrides.vars! : [MockVarsMap()]
  };
};

export const MockIdentity = (overrides?: Partial<LaForgeIdentity>): { __typename: 'Identity' } & LaForgeIdentity => {
  return {
    __typename: 'Identity',
    IdentityToEnvironment:
      overrides && overrides.hasOwnProperty('IdentityToEnvironment') ? overrides.IdentityToEnvironment! : MockEnvironment(),
    avatar_file: overrides && overrides.hasOwnProperty('avatar_file') ? overrides.avatar_file! : 'aut',
    description: overrides && overrides.hasOwnProperty('description') ? overrides.description! : 'fuga',
    email: overrides && overrides.hasOwnProperty('email') ? overrides.email! : 'molestiae',
    first_name: overrides && overrides.hasOwnProperty('first_name') ? overrides.first_name! : 'temporibus',
    hcl_id: overrides && overrides.hasOwnProperty('hcl_id') ? overrides.hcl_id! : 'culpa',
    id: overrides && overrides.hasOwnProperty('id') ? overrides.id! : '83a0b63c-bc71-4d88-93fb-cad0a826c603',
    last_name: overrides && overrides.hasOwnProperty('last_name') ? overrides.last_name! : 'atque',
    password: overrides && overrides.hasOwnProperty('password') ? overrides.password! : 'tempora',
    tags: overrides && overrides.hasOwnProperty('tags') ? overrides.tags! : [MockTagMap()],
    vars: overrides && overrides.hasOwnProperty('vars') ? overrides.vars! : [MockVarsMap()]
  };
};

export const MockLaForgePageInfo = (
  overrides?: Partial<LaForgeLaForgePageInfo>
): { __typename: 'LaForgePageInfo' } & LaForgeLaForgePageInfo => {
  return {
    __typename: 'LaForgePageInfo',
    nextOffset: overrides && overrides.hasOwnProperty('nextOffset') ? overrides.nextOffset! : 5848,
    total: overrides && overrides.hasOwnProperty('total') ? overrides.total! : 9631
  };
};

export const MockMutation = (overrides?: Partial<LaForgeMutation>): { __typename: 'Mutation' } & LaForgeMutation => {
  return {
    __typename: 'Mutation',
    approveCommit: overrides && overrides.hasOwnProperty('approveCommit') ? overrides.approveCommit! : true,
    cancelBuild: overrides && overrides.hasOwnProperty('cancelBuild') ? overrides.cancelBuild! : false,
    cancelCommit: overrides && overrides.hasOwnProperty('cancelCommit') ? overrides.cancelCommit! : true,
    createAgentTasks: overrides && overrides.hasOwnProperty('createAgentTasks') ? overrides.createAgentTasks! : [MockAgentTask()],
    createBatchAgentTasks:
      overrides && overrides.hasOwnProperty('createBatchAgentTasks') ? overrides.createBatchAgentTasks! : [MockAgentTask()],
    createBuild: overrides && overrides.hasOwnProperty('createBuild') ? overrides.createBuild! : MockBuild(),
    createEnviromentFromRepo:
      overrides && overrides.hasOwnProperty('createEnviromentFromRepo') ? overrides.createEnviromentFromRepo! : [MockEnvironment()],
    createTask: overrides && overrides.hasOwnProperty('createTask') ? overrides.createTask! : false,
    createUser: overrides && overrides.hasOwnProperty('createUser') ? overrides.createUser! : MockAuthUser(),
    deleteBuild: overrides && overrides.hasOwnProperty('deleteBuild') ? overrides.deleteBuild! : 'possimus',
    deleteUser: overrides && overrides.hasOwnProperty('deleteUser') ? overrides.deleteUser! : true,
    dumpBuild: overrides && overrides.hasOwnProperty('dumpBuild') ? overrides.dumpBuild! : 'adipisci',
    executePlan: overrides && overrides.hasOwnProperty('executePlan') ? overrides.executePlan! : MockBuild(),
    loadEnvironment: overrides && overrides.hasOwnProperty('loadEnvironment') ? overrides.loadEnvironment! : [MockEnvironment()],
    modifyAdminPassword: overrides && overrides.hasOwnProperty('modifyAdminPassword') ? overrides.modifyAdminPassword! : false,
    modifyAdminUserInfo: overrides && overrides.hasOwnProperty('modifyAdminUserInfo') ? overrides.modifyAdminUserInfo! : MockAuthUser(),
    modifySelfPassword: overrides && overrides.hasOwnProperty('modifySelfPassword') ? overrides.modifySelfPassword! : true,
    modifySelfUserInfo: overrides && overrides.hasOwnProperty('modifySelfUserInfo') ? overrides.modifySelfUserInfo! : MockAuthUser(),
    nukeBackend: overrides && overrides.hasOwnProperty('nukeBackend') ? overrides.nukeBackend! : [MockIntMap()],
    rebuild: overrides && overrides.hasOwnProperty('rebuild') ? overrides.rebuild! : false,
    updateEnviromentViaPull:
      overrides && overrides.hasOwnProperty('updateEnviromentViaPull') ? overrides.updateEnviromentViaPull! : [MockEnvironment()]
  };
};

export const MockNetwork = (overrides?: Partial<LaForgeNetwork>): { __typename: 'Network' } & LaForgeNetwork => {
  return {
    __typename: 'Network',
    NetworkToEnvironment:
      overrides && overrides.hasOwnProperty('NetworkToEnvironment') ? overrides.NetworkToEnvironment! : MockEnvironment(),
    cidr: overrides && overrides.hasOwnProperty('cidr') ? overrides.cidr! : 'voluptas',
    hcl_id: overrides && overrides.hasOwnProperty('hcl_id') ? overrides.hcl_id! : 'et',
    id: overrides && overrides.hasOwnProperty('id') ? overrides.id! : '134bbe71-7a72-4e7b-9571-3848ba54167d',
    name: overrides && overrides.hasOwnProperty('name') ? overrides.name! : 'non',
    tags: overrides && overrides.hasOwnProperty('tags') ? overrides.tags! : [MockTagMap()],
    vars: overrides && overrides.hasOwnProperty('vars') ? overrides.vars! : [MockVarsMap()],
    vdi_visible: overrides && overrides.hasOwnProperty('vdi_visible') ? overrides.vdi_visible! : false
  };
};

export const MockPlan = (overrides?: Partial<LaForgePlan>): { __typename: 'Plan' } & LaForgePlan => {
  return {
    __typename: 'Plan',
    NextPlan: overrides && overrides.hasOwnProperty('NextPlan') ? overrides.NextPlan! : [MockPlan()],
    PlanToBuild: overrides && overrides.hasOwnProperty('PlanToBuild') ? overrides.PlanToBuild! : MockBuild(),
    PlanToPlanDiffs: overrides && overrides.hasOwnProperty('PlanToPlanDiffs') ? overrides.PlanToPlanDiffs! : [MockPlanDiff()],
    PlanToProvisionedHost:
      overrides && overrides.hasOwnProperty('PlanToProvisionedHost') ? overrides.PlanToProvisionedHost! : MockProvisionedHost(),
    PlanToProvisionedNetwork:
      overrides && overrides.hasOwnProperty('PlanToProvisionedNetwork') ? overrides.PlanToProvisionedNetwork! : MockProvisionedNetwork(),
    PlanToProvisioningStep:
      overrides && overrides.hasOwnProperty('PlanToProvisioningStep') ? overrides.PlanToProvisioningStep! : MockProvisioningStep(),
    PlanToStatus: overrides && overrides.hasOwnProperty('PlanToStatus') ? overrides.PlanToStatus! : MockStatus(),
    PlanToTeam: overrides && overrides.hasOwnProperty('PlanToTeam') ? overrides.PlanToTeam! : MockTeam(),
    PrevPlan: overrides && overrides.hasOwnProperty('PrevPlan') ? overrides.PrevPlan! : [MockPlan()],
    build_id: overrides && overrides.hasOwnProperty('build_id') ? overrides.build_id! : 'et',
    id: overrides && overrides.hasOwnProperty('id') ? overrides.id! : 'bbc1ddc8-da1c-42e5-8672-bbeb671cba0c',
    step_number: overrides && overrides.hasOwnProperty('step_number') ? overrides.step_number! : 1954,
    type: overrides && overrides.hasOwnProperty('type') ? overrides.type! : LaForgePlanType.ExecuteStep
  };
};

export const MockPlanCounts = (overrides?: Partial<LaForgePlanCounts>): { __typename: 'PlanCounts' } & LaForgePlanCounts => {
  return {
    __typename: 'PlanCounts',
    awaiting: overrides && overrides.hasOwnProperty('awaiting') ? overrides.awaiting! : 6373,
    cancelled: overrides && overrides.hasOwnProperty('cancelled') ? overrides.cancelled! : 502,
    complete: overrides && overrides.hasOwnProperty('complete') ? overrides.complete! : 4180,
    deleteInProgress: overrides && overrides.hasOwnProperty('deleteInProgress') ? overrides.deleteInProgress! : 9982,
    deleted: overrides && overrides.hasOwnProperty('deleted') ? overrides.deleted! : 386,
    failed: overrides && overrides.hasOwnProperty('failed') ? overrides.failed! : 7159,
    inProgress: overrides && overrides.hasOwnProperty('inProgress') ? overrides.inProgress! : 8256,
    parentAwaiting: overrides && overrides.hasOwnProperty('parentAwaiting') ? overrides.parentAwaiting! : 4248,
    planning: overrides && overrides.hasOwnProperty('planning') ? overrides.planning! : 638,
    tainted: overrides && overrides.hasOwnProperty('tainted') ? overrides.tainted! : 9561,
    toDelete: overrides && overrides.hasOwnProperty('toDelete') ? overrides.toDelete! : 4560,
    toRebuild: overrides && overrides.hasOwnProperty('toRebuild') ? overrides.toRebuild! : 3733,
    undefined: overrides && overrides.hasOwnProperty('undefined') ? overrides.undefined! : 3927
  };
};

export const MockPlanDiff = (overrides?: Partial<LaForgePlanDiff>): { __typename: 'PlanDiff' } & LaForgePlanDiff => {
  return {
    __typename: 'PlanDiff',
    PlanDiffToBuildCommit:
      overrides && overrides.hasOwnProperty('PlanDiffToBuildCommit') ? overrides.PlanDiffToBuildCommit! : MockBuildCommit(),
    PlanDiffToPlan: overrides && overrides.hasOwnProperty('PlanDiffToPlan') ? overrides.PlanDiffToPlan! : MockPlan(),
    id: overrides && overrides.hasOwnProperty('id') ? overrides.id! : '5ba1966b-68b7-4ead-a2b4-057a7f25dfe8',
    new_state: overrides && overrides.hasOwnProperty('new_state') ? overrides.new_state! : LaForgeProvisionStatus.Awaiting,
    revision: overrides && overrides.hasOwnProperty('revision') ? overrides.revision! : 3658
  };
};

export const MockProvisionedHost = (
  overrides?: Partial<LaForgeProvisionedHost>
): { __typename: 'ProvisionedHost' } & LaForgeProvisionedHost => {
  return {
    __typename: 'ProvisionedHost',
    ProvisionedHostToAgentStatus:
      overrides && overrides.hasOwnProperty('ProvisionedHostToAgentStatus') ? overrides.ProvisionedHostToAgentStatus! : MockAgentStatus(),
    ProvisionedHostToHost: overrides && overrides.hasOwnProperty('ProvisionedHostToHost') ? overrides.ProvisionedHostToHost! : MockHost(),
    ProvisionedHostToPlan: overrides && overrides.hasOwnProperty('ProvisionedHostToPlan') ? overrides.ProvisionedHostToPlan! : MockPlan(),
    ProvisionedHostToProvisionedNetwork:
      overrides && overrides.hasOwnProperty('ProvisionedHostToProvisionedNetwork')
        ? overrides.ProvisionedHostToProvisionedNetwork!
        : MockProvisionedNetwork(),
    ProvisionedHostToProvisioningStep:
      overrides && overrides.hasOwnProperty('ProvisionedHostToProvisioningStep')
        ? overrides.ProvisionedHostToProvisioningStep!
        : [MockProvisioningStep()],
    ProvisionedHostToStatus:
      overrides && overrides.hasOwnProperty('ProvisionedHostToStatus') ? overrides.ProvisionedHostToStatus! : MockStatus(),
    id: overrides && overrides.hasOwnProperty('id') ? overrides.id! : '0875d8cc-2520-49dd-b981-698c65454240',
    subnet_ip: overrides && overrides.hasOwnProperty('subnet_ip') ? overrides.subnet_ip! : 'porro'
  };
};

export const MockProvisionedNetwork = (
  overrides?: Partial<LaForgeProvisionedNetwork>
): { __typename: 'ProvisionedNetwork' } & LaForgeProvisionedNetwork => {
  return {
    __typename: 'ProvisionedNetwork',
    ProvisionedNetworkToBuild:
      overrides && overrides.hasOwnProperty('ProvisionedNetworkToBuild') ? overrides.ProvisionedNetworkToBuild! : MockBuild(),
    ProvisionedNetworkToNetwork:
      overrides && overrides.hasOwnProperty('ProvisionedNetworkToNetwork') ? overrides.ProvisionedNetworkToNetwork! : MockNetwork(),
    ProvisionedNetworkToPlan:
      overrides && overrides.hasOwnProperty('ProvisionedNetworkToPlan') ? overrides.ProvisionedNetworkToPlan! : MockPlan(),
    ProvisionedNetworkToProvisionedHost:
      overrides && overrides.hasOwnProperty('ProvisionedNetworkToProvisionedHost')
        ? overrides.ProvisionedNetworkToProvisionedHost!
        : [MockProvisionedHost()],
    ProvisionedNetworkToStatus:
      overrides && overrides.hasOwnProperty('ProvisionedNetworkToStatus') ? overrides.ProvisionedNetworkToStatus! : MockStatus(),
    ProvisionedNetworkToTeam:
      overrides && overrides.hasOwnProperty('ProvisionedNetworkToTeam') ? overrides.ProvisionedNetworkToTeam! : MockTeam(),
    cidr: overrides && overrides.hasOwnProperty('cidr') ? overrides.cidr! : 'officiis',
    id: overrides && overrides.hasOwnProperty('id') ? overrides.id! : 'd49c2598-c864-4eb7-beb2-b41ba38e6b71',
    name: overrides && overrides.hasOwnProperty('name') ? overrides.name! : 'consequatur'
  };
};

export const MockProvisioningStep = (
  overrides?: Partial<LaForgeProvisioningStep>
): { __typename: 'ProvisioningStep' } & LaForgeProvisioningStep => {
  return {
    __typename: 'ProvisioningStep',
    ProvisioningStepToCommand:
      overrides && overrides.hasOwnProperty('ProvisioningStepToCommand') ? overrides.ProvisioningStepToCommand! : MockCommand(),
    ProvisioningStepToDNSRecord:
      overrides && overrides.hasOwnProperty('ProvisioningStepToDNSRecord') ? overrides.ProvisioningStepToDNSRecord! : MockDnsRecord(),
    ProvisioningStepToFileDelete:
      overrides && overrides.hasOwnProperty('ProvisioningStepToFileDelete') ? overrides.ProvisioningStepToFileDelete! : MockFileDelete(),
    ProvisioningStepToFileDownload:
      overrides && overrides.hasOwnProperty('ProvisioningStepToFileDownload')
        ? overrides.ProvisioningStepToFileDownload!
        : MockFileDownload(),
    ProvisioningStepToFileExtract:
      overrides && overrides.hasOwnProperty('ProvisioningStepToFileExtract') ? overrides.ProvisioningStepToFileExtract! : MockFileExtract(),
    ProvisioningStepToPlan:
      overrides && overrides.hasOwnProperty('ProvisioningStepToPlan') ? overrides.ProvisioningStepToPlan! : MockPlan(),
    ProvisioningStepToProvisionedHost:
      overrides && overrides.hasOwnProperty('ProvisioningStepToProvisionedHost')
        ? overrides.ProvisioningStepToProvisionedHost!
        : MockProvisionedHost(),
    ProvisioningStepToScript:
      overrides && overrides.hasOwnProperty('ProvisioningStepToScript') ? overrides.ProvisioningStepToScript! : MockScript(),
    ProvisioningStepToStatus:
      overrides && overrides.hasOwnProperty('ProvisioningStepToStatus') ? overrides.ProvisioningStepToStatus! : MockStatus(),
    id: overrides && overrides.hasOwnProperty('id') ? overrides.id! : 'a03a1be3-c4f9-4e1a-9090-ac866dd0815c',
    step_number: overrides && overrides.hasOwnProperty('step_number') ? overrides.step_number! : 466,
    type: overrides && overrides.hasOwnProperty('type') ? overrides.type! : LaForgeProvisioningStepType.Ansible
  };
};

export const MockQuery = (overrides?: Partial<LaForgeQuery>): { __typename: 'Query' } & LaForgeQuery => {
  return {
    __typename: 'Query',
    agentStatus: overrides && overrides.hasOwnProperty('agentStatus') ? overrides.agentStatus! : MockAgentStatus(),
    build: overrides && overrides.hasOwnProperty('build') ? overrides.build! : MockBuild(),
    currentUser: overrides && overrides.hasOwnProperty('currentUser') ? overrides.currentUser! : MockAuthUser(),
    environment: overrides && overrides.hasOwnProperty('environment') ? overrides.environment! : MockEnvironment(),
    environments: overrides && overrides.hasOwnProperty('environments') ? overrides.environments! : [MockEnvironment()],
    getAgentTasks: overrides && overrides.hasOwnProperty('getAgentTasks') ? overrides.getAgentTasks! : [MockAgentTask()],
    getAllAgentStatus: overrides && overrides.hasOwnProperty('getAllAgentStatus') ? overrides.getAllAgentStatus! : MockAgentStatusBatch(),
    getAllPlanStatus: overrides && overrides.hasOwnProperty('getAllPlanStatus') ? overrides.getAllPlanStatus! : MockStatusBatch(),
    getBuildCommit: overrides && overrides.hasOwnProperty('getBuildCommit') ? overrides.getBuildCommit! : MockBuildCommit(),
    getBuildCommits: overrides && overrides.hasOwnProperty('getBuildCommits') ? overrides.getBuildCommits! : [MockBuildCommit()],
    getBuilds: overrides && overrides.hasOwnProperty('getBuilds') ? overrides.getBuilds! : [MockBuild()],
    getCurrentUserTasks: overrides && overrides.hasOwnProperty('getCurrentUserTasks') ? overrides.getCurrentUserTasks! : [MockServerTask()],
    getPlanStatusCounts: overrides && overrides.hasOwnProperty('getPlanStatusCounts') ? overrides.getPlanStatusCounts! : MockPlanCounts(),
    getServerTasks: overrides && overrides.hasOwnProperty('getServerTasks') ? overrides.getServerTasks! : [MockServerTask()],
    getUserList: overrides && overrides.hasOwnProperty('getUserList') ? overrides.getUserList! : [MockAuthUser()],
    listAgentStatuses: overrides && overrides.hasOwnProperty('listAgentStatuses') ? overrides.listAgentStatuses! : [MockAgentStatus()],
    listBuildStatuses: overrides && overrides.hasOwnProperty('listBuildStatuses') ? overrides.listBuildStatuses! : [MockStatus()],
    plan: overrides && overrides.hasOwnProperty('plan') ? overrides.plan! : MockPlan(),
    provisionedHost: overrides && overrides.hasOwnProperty('provisionedHost') ? overrides.provisionedHost! : MockProvisionedHost(),
    provisionedNetwork:
      overrides && overrides.hasOwnProperty('provisionedNetwork') ? overrides.provisionedNetwork! : MockProvisionedNetwork(),
    provisionedStep: overrides && overrides.hasOwnProperty('provisionedStep') ? overrides.provisionedStep! : MockProvisioningStep(),
    serverTasks: overrides && overrides.hasOwnProperty('serverTasks') ? overrides.serverTasks! : [MockServerTask()],
    status: overrides && overrides.hasOwnProperty('status') ? overrides.status! : MockStatus(),
    viewAgentTask: overrides && overrides.hasOwnProperty('viewAgentTask') ? overrides.viewAgentTask! : MockAgentTask(),
    viewServerTaskLogs: overrides && overrides.hasOwnProperty('viewServerTaskLogs') ? overrides.viewServerTaskLogs! : 'quasi'
  };
};

export const MockRepoCommit = (overrides?: Partial<LaForgeRepoCommit>): { __typename: 'RepoCommit' } & LaForgeRepoCommit => {
  return {
    __typename: 'RepoCommit',
    RepoCommitToRepository:
      overrides && overrides.hasOwnProperty('RepoCommitToRepository') ? overrides.RepoCommitToRepository! : MockRepository(),
    author: overrides && overrides.hasOwnProperty('author') ? overrides.author! : 'quos',
    committer: overrides && overrides.hasOwnProperty('committer') ? overrides.committer! : 'qui',
    hash: overrides && overrides.hasOwnProperty('hash') ? overrides.hash! : 'repellendus',
    id: overrides && overrides.hasOwnProperty('id') ? overrides.id! : 'ed9703c1-4946-4d78-aed0-6ad9d4d5de0d',
    message: overrides && overrides.hasOwnProperty('message') ? overrides.message! : 'ducimus',
    parent_hashes: overrides && overrides.hasOwnProperty('parent_hashes') ? overrides.parent_hashes! : ['dolor'],
    pgp_signature: overrides && overrides.hasOwnProperty('pgp_signature') ? overrides.pgp_signature! : 'maxime',
    revision: overrides && overrides.hasOwnProperty('revision') ? overrides.revision! : 7281,
    tree_hash: overrides && overrides.hasOwnProperty('tree_hash') ? overrides.tree_hash! : 'cumque'
  };
};

export const MockRepository = (overrides?: Partial<LaForgeRepository>): { __typename: 'Repository' } & LaForgeRepository => {
  return {
    __typename: 'Repository',
    RepositoryToRepoCommit:
      overrides && overrides.hasOwnProperty('RepositoryToRepoCommit') ? overrides.RepositoryToRepoCommit! : [MockRepoCommit()],
    branch_name: overrides && overrides.hasOwnProperty('branch_name') ? overrides.branch_name! : 'tempora',
    environment_filepath: overrides && overrides.hasOwnProperty('environment_filepath') ? overrides.environment_filepath! : 'dicta',
    id: overrides && overrides.hasOwnProperty('id') ? overrides.id! : 'e97f8841-e61d-451b-93f6-99aacfac2fad',
    repo_url: overrides && overrides.hasOwnProperty('repo_url') ? overrides.repo_url! : 'dolor'
  };
};

export const MockScript = (overrides?: Partial<LaForgeScript>): { __typename: 'Script' } & LaForgeScript => {
  return {
    __typename: 'Script',
    ScriptToEnvironment: overrides && overrides.hasOwnProperty('ScriptToEnvironment') ? overrides.ScriptToEnvironment! : MockEnvironment(),
    absPath: overrides && overrides.hasOwnProperty('absPath') ? overrides.absPath! : 'accusamus',
    args: overrides && overrides.hasOwnProperty('args') ? overrides.args! : ['quia'],
    cooldown: overrides && overrides.hasOwnProperty('cooldown') ? overrides.cooldown! : 1741,
    description: overrides && overrides.hasOwnProperty('description') ? overrides.description! : 'similique',
    disabled: overrides && overrides.hasOwnProperty('disabled') ? overrides.disabled! : true,
    hcl_id: overrides && overrides.hasOwnProperty('hcl_id') ? overrides.hcl_id! : 'quaerat',
    id: overrides && overrides.hasOwnProperty('id') ? overrides.id! : '320910d0-eaf9-4a15-bd19-dfc2127fd47b',
    ignore_errors: overrides && overrides.hasOwnProperty('ignore_errors') ? overrides.ignore_errors! : false,
    language: overrides && overrides.hasOwnProperty('language') ? overrides.language! : 'quae',
    name: overrides && overrides.hasOwnProperty('name') ? overrides.name! : 'blanditiis',
    scriptToFinding: overrides && overrides.hasOwnProperty('scriptToFinding') ? overrides.scriptToFinding! : [MockFinding()],
    source: overrides && overrides.hasOwnProperty('source') ? overrides.source! : 'fuga',
    source_type: overrides && overrides.hasOwnProperty('source_type') ? overrides.source_type! : 'maiores',
    tags: overrides && overrides.hasOwnProperty('tags') ? overrides.tags! : [MockTagMap()],
    timeout: overrides && overrides.hasOwnProperty('timeout') ? overrides.timeout! : 3708,
    vars: overrides && overrides.hasOwnProperty('vars') ? overrides.vars! : [MockVarsMap()]
  };
};

export const MockServerTask = (overrides?: Partial<LaForgeServerTask>): { __typename: 'ServerTask' } & LaForgeServerTask => {
  return {
    __typename: 'ServerTask',
    ServerTaskToAuthUser: overrides && overrides.hasOwnProperty('ServerTaskToAuthUser') ? overrides.ServerTaskToAuthUser! : MockAuthUser(),
    ServerTaskToBuild: overrides && overrides.hasOwnProperty('ServerTaskToBuild') ? overrides.ServerTaskToBuild! : MockBuild(),
    ServerTaskToBuildCommit:
      overrides && overrides.hasOwnProperty('ServerTaskToBuildCommit') ? overrides.ServerTaskToBuildCommit! : MockBuildCommit(),
    ServerTaskToEnvironment:
      overrides && overrides.hasOwnProperty('ServerTaskToEnvironment') ? overrides.ServerTaskToEnvironment! : MockEnvironment(),
    ServerTaskToStatus: overrides && overrides.hasOwnProperty('ServerTaskToStatus') ? overrides.ServerTaskToStatus! : MockStatus(),
    end_time: overrides && overrides.hasOwnProperty('end_time') ? overrides.end_time! : 'dolor',
    errors: overrides && overrides.hasOwnProperty('errors') ? overrides.errors! : ['molestiae'],
    id: overrides && overrides.hasOwnProperty('id') ? overrides.id! : '8e7720ea-08ff-460d-90c8-38a22c6402f9',
    log_file_path: overrides && overrides.hasOwnProperty('log_file_path') ? overrides.log_file_path! : 'expedita',
    start_time: overrides && overrides.hasOwnProperty('start_time') ? overrides.start_time! : 'inventore',
    type: overrides && overrides.hasOwnProperty('type') ? overrides.type! : LaForgeServerTaskType.Createbuild
  };
};

export const MockStatus = (overrides?: Partial<LaForgeStatus>): { __typename: 'Status' } & LaForgeStatus => {
  return {
    __typename: 'Status',
    completed: overrides && overrides.hasOwnProperty('completed') ? overrides.completed! : true,
    ended_at: overrides && overrides.hasOwnProperty('ended_at') ? overrides.ended_at! : 'nobis',
    error: overrides && overrides.hasOwnProperty('error') ? overrides.error! : 'ut',
    failed: overrides && overrides.hasOwnProperty('failed') ? overrides.failed! : true,
    id: overrides && overrides.hasOwnProperty('id') ? overrides.id! : '127e850f-f5bb-40c1-aced-35142fa0348c',
    started_at: overrides && overrides.hasOwnProperty('started_at') ? overrides.started_at! : 'perferendis',
    state: overrides && overrides.hasOwnProperty('state') ? overrides.state! : LaForgeProvisionStatus.Awaiting,
    status_for: overrides && overrides.hasOwnProperty('status_for') ? overrides.status_for! : LaForgeProvisionStatusFor.Build
  };
};

export const MockStatusBatch = (overrides?: Partial<LaForgeStatusBatch>): { __typename: 'StatusBatch' } & LaForgeStatusBatch => {
  return {
    __typename: 'StatusBatch',
    pageInfo: overrides && overrides.hasOwnProperty('pageInfo') ? overrides.pageInfo! : MockLaForgePageInfo(),
    statuses: overrides && overrides.hasOwnProperty('statuses') ? overrides.statuses! : [MockStatus()]
  };
};

export const MockSubscription = (overrides?: Partial<LaForgeSubscription>): { __typename: 'Subscription' } & LaForgeSubscription => {
  return {
    __typename: 'Subscription',
    streamServerTaskLog: overrides && overrides.hasOwnProperty('streamServerTaskLog') ? overrides.streamServerTaskLog! : 'recusandae',
    updatedAgentStatus: overrides && overrides.hasOwnProperty('updatedAgentStatus') ? overrides.updatedAgentStatus! : MockAgentStatus(),
    updatedAgentTask: overrides && overrides.hasOwnProperty('updatedAgentTask') ? overrides.updatedAgentTask! : MockAgentTask(),
    updatedBuild: overrides && overrides.hasOwnProperty('updatedBuild') ? overrides.updatedBuild! : MockBuild(),
    updatedCommit: overrides && overrides.hasOwnProperty('updatedCommit') ? overrides.updatedCommit! : MockBuildCommit(),
    updatedServerTask: overrides && overrides.hasOwnProperty('updatedServerTask') ? overrides.updatedServerTask! : MockServerTask(),
    updatedStatus: overrides && overrides.hasOwnProperty('updatedStatus') ? overrides.updatedStatus! : MockStatus()
  };
};

export const MockTeam = (overrides?: Partial<LaForgeTeam>): { __typename: 'Team' } & LaForgeTeam => {
  return {
    __typename: 'Team',
    TeamToBuild: overrides && overrides.hasOwnProperty('TeamToBuild') ? overrides.TeamToBuild! : MockBuild(),
    TeamToPlan: overrides && overrides.hasOwnProperty('TeamToPlan') ? overrides.TeamToPlan! : MockPlan(),
    TeamToProvisionedNetwork:
      overrides && overrides.hasOwnProperty('TeamToProvisionedNetwork') ? overrides.TeamToProvisionedNetwork! : [MockProvisionedNetwork()],
    TeamToStatus: overrides && overrides.hasOwnProperty('TeamToStatus') ? overrides.TeamToStatus! : MockStatus(),
    id: overrides && overrides.hasOwnProperty('id') ? overrides.id! : '8384ad9a-55ae-4711-989f-feb546bd5b4e',
    team_number: overrides && overrides.hasOwnProperty('team_number') ? overrides.team_number! : 2589
  };
};

export const MockUser = (overrides?: Partial<LaForgeUser>): { __typename: 'User' } & LaForgeUser => {
  return {
    __typename: 'User',
    email: overrides && overrides.hasOwnProperty('email') ? overrides.email! : 'sunt',
    id: overrides && overrides.hasOwnProperty('id') ? overrides.id! : 'a5756f00-41a6-422a-8a7d-d13ee6a63750',
    name: overrides && overrides.hasOwnProperty('name') ? overrides.name! : 'porro',
    uuid: overrides && overrides.hasOwnProperty('uuid') ? overrides.uuid! : 'nobis'
  };
};

export const MockConfigMap = (overrides?: Partial<LaForgeConfigMap>): { __typename: 'configMap' } & LaForgeConfigMap => {
  return {
    __typename: 'configMap',
    key: overrides && overrides.hasOwnProperty('key') ? overrides.key! : 'nostrum',
    value: overrides && overrides.hasOwnProperty('value') ? overrides.value! : 'voluptates'
  };
};

export const MockIntMap = (overrides?: Partial<LaForgeIntMap>): { __typename: 'intMap' } & LaForgeIntMap => {
  return {
    __typename: 'intMap',
    key: overrides && overrides.hasOwnProperty('key') ? overrides.key! : 'similique',
    value: overrides && overrides.hasOwnProperty('value') ? overrides.value! : 1161
  };
};

export const MockTagMap = (overrides?: Partial<LaForgeTagMap>): { __typename: 'tagMap' } & LaForgeTagMap => {
  return {
    __typename: 'tagMap',
    key: overrides && overrides.hasOwnProperty('key') ? overrides.key! : 'ratione',
    value: overrides && overrides.hasOwnProperty('value') ? overrides.value! : 'et'
  };
};

export const MockVarsMap = (overrides?: Partial<LaForgeVarsMap>): { __typename: 'varsMap' } & LaForgeVarsMap => {
  return {
    __typename: 'varsMap',
    key: overrides && overrides.hasOwnProperty('key') ? overrides.key! : 'aut',
    value: overrides && overrides.hasOwnProperty('value') ? overrides.value! : 'aut'
  };
};
