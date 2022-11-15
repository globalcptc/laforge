import { gql } from 'apollo-angular';
import { Injectable } from '@angular/core';
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

export type LaForgeGetUserListQueryVariables = Exact<{ [key: string]: never; }>;


export type LaForgeGetUserListQuery = { __typename?: 'Query', getUserList?: Array<{ __typename?: 'AuthUser', id: string, first_name: string, last_name: string, username: string, provider: LaForgeProviderType, role: LaForgeRoleLevel, email: string, phone: string, company: string, occupation: string } | null> | null };

export type LaForgeGetCurrentUserQueryVariables = Exact<{ [key: string]: never; }>;


export type LaForgeGetCurrentUserQuery = { __typename?: 'Query', currentUser?: { __typename?: 'AuthUser', id: string, first_name: string, last_name: string, username: string, provider: LaForgeProviderType, role: LaForgeRoleLevel, email: string, phone: string, company: string, occupation: string } | null };

export type LaForgeAuthUserFieldsFragment = { __typename?: 'AuthUser', id: string, first_name: string, last_name: string, username: string, provider: LaForgeProviderType, role: LaForgeRoleLevel, email: string, phone: string, company: string, occupation: string };

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
    ${AuthUserFieldsFragmentDoc}`;

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
    ${AuthUserFieldsFragmentDoc}`;

  @Injectable({
    providedIn: 'root'
  })
  export class LaForgeGetCurrentUserGQL extends Apollo.Query<LaForgeGetCurrentUserQuery, LaForgeGetCurrentUserQueryVariables> {
    document = GetCurrentUserDocument;
    
    constructor(apollo: Apollo.Apollo) {
      super(apollo);
    }
  }