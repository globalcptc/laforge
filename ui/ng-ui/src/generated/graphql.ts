import { Injectable } from '@angular/core';
import { gql } from 'apollo-angular';
import * as Apollo from 'apollo-angular';
export type Maybe<T> = T | null;
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
  Map: any;
  Time: any;
};

export type LaForgeAdhocPlan = {
  __typename?: 'AdhocPlan';
  id: Scalars['ID'];
  NextAdhocPlans?: Maybe<Array<Maybe<LaForgeAdhocPlan>>>;
  PrevAdhocPlans?: Maybe<Array<Maybe<LaForgeAdhocPlan>>>;
  Build: LaForgeBuild;
  Status: LaForgeStatus;
  AgentTask: LaForgeAgentTask;
};

export enum LaForgeAgentCommand {
  Default = 'DEFAULT',
  Delete = 'DELETE',
  Reboot = 'REBOOT',
  Extract = 'EXTRACT',
  Download = 'DOWNLOAD',
  Createuser = 'CREATEUSER',
  Createuserpass = 'CREATEUSERPASS',
  Addtogroup = 'ADDTOGROUP',
  Execute = 'EXECUTE',
  Validate = 'VALIDATE',
  Changeperms = 'CHANGEPERMS',
  Appendfile = 'APPENDFILE'
}

export type LaForgeAgentConfig = {
  __typename?: 'AgentConfig';
  GrpcServerUri: Scalars['String'];
  ApiDownloadUrl: Scalars['String'];
};

export type LaForgeAgentStatus = {
  __typename?: 'AgentStatus';
  clientId: Scalars['String'];
  hostname: Scalars['String'];
  upTime: Scalars['Int'];
  bootTime: Scalars['Int'];
  numProcs: Scalars['Int'];
  OS: Scalars['String'];
  hostID: Scalars['String'];
  load1?: Maybe<Scalars['Float']>;
  load5?: Maybe<Scalars['Float']>;
  load15?: Maybe<Scalars['Float']>;
  totalMem: Scalars['Int'];
  freeMem: Scalars['Int'];
  usedMem: Scalars['Int'];
  timestamp: Scalars['Int'];
  ProvisionedHost?: Maybe<LaForgeProvisionedHost>;
  ProvisionedNetwork?: Maybe<LaForgeProvisionedNetwork>;
  Build?: Maybe<LaForgeBuild>;
};

export type LaForgeAgentStatusBatch = {
  __typename?: 'AgentStatusBatch';
  agentStatuses: Array<Maybe<LaForgeAgentStatus>>;
  pageInfo: LaForgeLaForgePageInfo;
};

export type LaForgeAgentTask = {
  __typename?: 'AgentTask';
  id: Scalars['ID'];
  args?: Maybe<Scalars['String']>;
  command: LaForgeAgentCommand;
  number: Scalars['Int'];
  output?: Maybe<Scalars['String']>;
  state: LaForgeAgentTaskState;
  errorMessage?: Maybe<Scalars['String']>;
  ProvisioningStep?: Maybe<LaForgeProvisioningStep>;
  ProvisioningScheduledStep?: Maybe<LaForgeProvisioningScheduledStep>;
  ProvisionedHost: LaForgeProvisionedHost;
  AdhocPlans?: Maybe<Array<Maybe<LaForgeAdhocPlan>>>;
};

export enum LaForgeAgentTaskState {
  Awaiting = 'AWAITING',
  Inprogress = 'INPROGRESS',
  Failed = 'FAILED',
  Complete = 'COMPLETE'
}

export type LaForgeAnsible = {
  __typename?: 'Ansible';
  id: Scalars['ID'];
  hclId: Scalars['String'];
  name: Scalars['String'];
  description: Scalars['String'];
  source: Scalars['String'];
  playbookName: Scalars['String'];
  method: LaForgeAnsibleMethod;
  inventory: Scalars['String'];
  absPath: Scalars['String'];
  tags?: Maybe<Array<Maybe<LaForgeTagMap>>>;
  Users?: Maybe<Array<Maybe<LaForgeUser>>>;
  Environment?: Maybe<LaForgeEnvironment>;
};

export enum LaForgeAnsibleMethod {
  Local = 'LOCAL'
}

export type LaForgeAuthConfig = {
  __typename?: 'AuthConfig';
  GithubId: Scalars['String'];
  CookieTimeout: Scalars['Int'];
};

export type LaForgeAuthUser = {
  __typename?: 'AuthUser';
  id: Scalars['ID'];
  username: Scalars['String'];
  firstName: Scalars['String'];
  lastName: Scalars['String'];
  email: Scalars['String'];
  phone: Scalars['String'];
  company: Scalars['String'];
  occupation: Scalars['String'];
  publicKey: Scalars['String'];
  role: LaForgeRoleLevel;
  provider: LaForgeProviderType;
  ServerTasks?: Maybe<Array<Maybe<LaForgeServerTask>>>;
};

export type LaForgeBuild = {
  __typename?: 'Build';
  id: Scalars['ID'];
  revision: Scalars['Int'];
  environmentRevision: Scalars['Int'];
  completedPlan: Scalars['Boolean'];
  Status: LaForgeStatus;
  Environment: LaForgeEnvironment;
  Competition: LaForgeCompetition;
  LatestBuildCommit?: Maybe<LaForgeBuildCommit>;
  RepoCommit: LaForgeRepoCommit;
  ProvisionedNetworks: Array<Maybe<LaForgeProvisionedNetwork>>;
  Teams: Array<Maybe<LaForgeTeam>>;
  Plans: Array<Maybe<LaForgePlan>>;
  BuildCommits: Array<Maybe<LaForgeBuildCommit>>;
  AdhocPlans: Array<Maybe<LaForgeAdhocPlan>>;
  AgentStatuses: Array<Maybe<LaForgeAgentStatus>>;
  ServerTasks: Array<Maybe<LaForgeServerTask>>;
};

export type LaForgeBuildCommit = {
  __typename?: 'BuildCommit';
  id: Scalars['ID'];
  type: LaForgeBuildCommitType;
  revision: Scalars['Int'];
  state: LaForgeBuildCommitState;
  createdAt: Scalars['Time'];
  Build: LaForgeBuild;
  ServerTasks: Array<Maybe<LaForgeServerTask>>;
  PlanDiffs: Array<Maybe<LaForgePlanDiff>>;
};

export enum LaForgeBuildCommitState {
  Planning = 'PLANNING',
  Inprogress = 'INPROGRESS',
  Applied = 'APPLIED',
  Cancelled = 'CANCELLED',
  Approved = 'APPROVED'
}

export enum LaForgeBuildCommitType {
  Root = 'ROOT',
  Rebuild = 'REBUILD',
  Delete = 'DELETE'
}

export type LaForgeBuilderConfig = {
  __typename?: 'BuilderConfig';
  Builder: Scalars['String'];
  ConfigFile: Scalars['String'];
};

export type LaForgeCommand = {
  __typename?: 'Command';
  id: Scalars['ID'];
  hclId: Scalars['String'];
  name: Scalars['String'];
  description: Scalars['String'];
  program: Scalars['String'];
  args: Array<Maybe<Scalars['String']>>;
  ignoreErrors: Scalars['Boolean'];
  disabled: Scalars['Boolean'];
  cooldown: Scalars['Int'];
  timeout: Scalars['Int'];
  vars?: Maybe<Array<Maybe<LaForgeVarsMap>>>;
  tags?: Maybe<Array<Maybe<LaForgeTagMap>>>;
  Users: Array<Maybe<LaForgeUser>>;
  Environment: LaForgeEnvironment;
};

export type LaForgeCompetition = {
  __typename?: 'Competition';
  id: Scalars['ID'];
  hclId: Scalars['String'];
  rootPassword: Scalars['String'];
  startTime?: Maybe<Scalars['Int']>;
  stopTime?: Maybe<Scalars['Int']>;
  config?: Maybe<Array<Maybe<LaForgeConfigMap>>>;
  tags?: Maybe<Array<Maybe<LaForgeTagMap>>>;
  DNS: Array<Maybe<LaForgeDns>>;
  Environment: LaForgeEnvironment;
  Builds: Array<Maybe<LaForgeBuild>>;
};

export type LaForgeDns = {
  __typename?: 'DNS';
  id: Scalars['ID'];
  hclId: Scalars['String'];
  type: Scalars['String'];
  rootDomain: Scalars['String'];
  dnsServers: Array<Maybe<Scalars['String']>>;
  ntpServers: Array<Maybe<Scalars['String']>>;
  config?: Maybe<Array<Maybe<LaForgeConfigMap>>>;
  Environments: Array<Maybe<LaForgeEnvironment>>;
  Competitions: Array<Maybe<LaForgeCompetition>>;
};

export type LaForgeDnsRecord = {
  __typename?: 'DNSRecord';
  id: Scalars['ID'];
  hclId: Scalars['String'];
  name: Scalars['String'];
  values: Array<Maybe<Scalars['String']>>;
  type: Scalars['String'];
  zone: Scalars['String'];
  vars: Array<Maybe<LaForgeVarsMap>>;
  disabled: Scalars['Boolean'];
  tags: Array<Maybe<LaForgeTagMap>>;
  Environment: LaForgeEnvironment;
};

export type LaForgeDatabaseConfig = {
  __typename?: 'DatabaseConfig';
  PostgresUri: Scalars['String'];
  AdminUser: Scalars['String'];
};

export type LaForgeDisk = {
  __typename?: 'Disk';
  id: Scalars['ID'];
  size: Scalars['Int'];
  Host: LaForgeHost;
};

export type LaForgeEnvironment = {
  __typename?: 'Environment';
  id: Scalars['ID'];
  hclId: Scalars['String'];
  competitionId: Scalars['String'];
  name: Scalars['String'];
  description: Scalars['String'];
  builder: Scalars['String'];
  teamCount: Scalars['Int'];
  revision: Scalars['Int'];
  adminCidrs: Array<Maybe<Scalars['String']>>;
  exposedVdiPorts: Array<Maybe<Scalars['String']>>;
  config?: Maybe<Array<Maybe<LaForgeConfigMap>>>;
  tags?: Maybe<Array<Maybe<LaForgeTagMap>>>;
  Users: Array<Maybe<LaForgeUser>>;
  Hosts: Array<Maybe<LaForgeHost>>;
  Competitions: Array<Maybe<LaForgeCompetition>>;
  Identities: Array<Maybe<LaForgeIdentity>>;
  Commands: Array<Maybe<LaForgeCommand>>;
  Scripts: Array<Maybe<LaForgeScript>>;
  FileDownloads: Array<Maybe<LaForgeFileDownload>>;
  FileDeletes: Array<Maybe<LaForgeFileDelete>>;
  FileExtracts: Array<Maybe<LaForgeFileExtract>>;
  IncludedNetworks: Array<Maybe<LaForgeIncludedNetwork>>;
  Findings: Array<Maybe<LaForgeFinding>>;
  DNSRecords: Array<Maybe<LaForgeDnsRecord>>;
  DNS: Array<Maybe<LaForgeDns>>;
  Networks: Array<Maybe<LaForgeNetwork>>;
  HostDependencies: Array<Maybe<LaForgeHostDependency>>;
  Ansibles: Array<Maybe<LaForgeAnsible>>;
  ScheduledSteps: Array<Maybe<LaForgeScheduledStep>>;
  Builds: Array<Maybe<LaForgeBuild>>;
  Repositories: Array<Maybe<LaForgeRepository>>;
  ServerTasks: Array<Maybe<LaForgeServerTask>>;
};

export type LaForgeFileDelete = {
  __typename?: 'FileDelete';
  id: Scalars['ID'];
  hclId: Scalars['String'];
  path: Scalars['String'];
  tags: Array<Maybe<LaForgeTagMap>>;
  Environment: LaForgeEnvironment;
};

export type LaForgeFileDownload = {
  __typename?: 'FileDownload';
  id: Scalars['ID'];
  hclId: Scalars['String'];
  sourceType: Scalars['String'];
  source: Scalars['String'];
  destination: Scalars['String'];
  template: Scalars['Boolean'];
  perms: Scalars['String'];
  disabled: Scalars['Boolean'];
  md5: Scalars['String'];
  absPath: Scalars['String'];
  tags: Array<Maybe<LaForgeTagMap>>;
  Environment: LaForgeEnvironment;
};

export type LaForgeFileExtract = {
  __typename?: 'FileExtract';
  id: Scalars['ID'];
  hclId: Scalars['String'];
  source: Scalars['String'];
  destination: Scalars['String'];
  type: Scalars['String'];
  tags: Array<Maybe<LaForgeTagMap>>;
  Environment: LaForgeEnvironment;
};

export type LaForgeFinding = {
  __typename?: 'Finding';
  id: Scalars['ID'];
  name: Scalars['String'];
  description: Scalars['String'];
  severity: LaForgeFindingSeverity;
  difficulty: LaForgeFindingDifficulty;
  tags: Array<Maybe<LaForgeTagMap>>;
  Users: Array<Maybe<LaForgeUser>>;
  Host?: Maybe<LaForgeHost>;
  Script?: Maybe<LaForgeScript>;
  Environment?: Maybe<LaForgeEnvironment>;
};

export enum LaForgeFindingDifficulty {
  ZeroDifficulty = 'ZeroDifficulty',
  NoviceDifficulty = 'NoviceDifficulty',
  AdvancedDifficulty = 'AdvancedDifficulty',
  ExpertDifficulty = 'ExpertDifficulty',
  NullDifficulty = 'NullDifficulty'
}

export enum LaForgeFindingSeverity {
  ZeroSeverity = 'ZeroSeverity',
  LowSeverity = 'LowSeverity',
  MediumSeverity = 'MediumSeverity',
  HighSeverity = 'HighSeverity',
  CriticalSeverity = 'CriticalSeverity',
  NullSeverity = 'NullSeverity'
}

export type LaForgeGinFileMiddleware = {
  __typename?: 'GinFileMiddleware';
  id: Scalars['ID'];
  urlId: Scalars['String'];
  filePath: Scalars['String'];
  accessed: Scalars['Boolean'];
  ProvisionedHost?: Maybe<LaForgeProvisionedHost>;
  ProvisioningStep?: Maybe<LaForgeProvisioningStep>;
  ProvisioningScheduledStep?: Maybe<LaForgeProvisioningScheduledStep>;
};

export type LaForgeGraphqlConfig = {
  __typename?: 'GraphqlConfig';
  Hostname: Scalars['String'];
  RedisServerUri: Scalars['String'];
};

export type LaForgeHost = {
  __typename?: 'Host';
  id: Scalars['ID'];
  hclId: Scalars['String'];
  hostname: Scalars['String'];
  description: Scalars['String'];
  OS: Scalars['String'];
  lastOctet: Scalars['Int'];
  instanceSize: Scalars['String'];
  allowMacChanges: Scalars['Boolean'];
  exposedTcpPorts: Array<Maybe<Scalars['String']>>;
  exposedUdpPorts: Array<Maybe<Scalars['String']>>;
  overridePassword: Scalars['String'];
  vars?: Maybe<Array<Maybe<LaForgeVarsMap>>>;
  userGroups: Array<Maybe<Scalars['String']>>;
  provisionSteps: Array<Maybe<Scalars['String']>>;
  tags: Array<Maybe<LaForgeTagMap>>;
  Disk: LaForgeDisk;
  Users: Array<Maybe<LaForgeUser>>;
  Environment: LaForgeEnvironment;
  IncludedNetworks: Array<Maybe<LaForgeIncludedNetwork>>;
  DependOnHostDependencies: Array<Maybe<LaForgeHostDependency>>;
  RequiredByHostDependencies: Array<Maybe<LaForgeHostDependency>>;
};

export type LaForgeHostDependency = {
  __typename?: 'HostDependency';
  id: Scalars['ID'];
  hostId: Scalars['String'];
  networkId: Scalars['String'];
  RequiredBy?: Maybe<LaForgeHost>;
  DependOnHost?: Maybe<LaForgeHost>;
  DependOnNetwork?: Maybe<LaForgeNetwork>;
  Environment?: Maybe<LaForgeEnvironment>;
};

export type LaForgeIdentity = {
  __typename?: 'Identity';
  id: Scalars['ID'];
  hclid: Scalars['String'];
  firstName: Scalars['String'];
  lastName: Scalars['String'];
  email: Scalars['String'];
  password: Scalars['String'];
  description: Scalars['String'];
  avatarFile: Scalars['String'];
  vars: Array<Maybe<LaForgeVarsMap>>;
  tags: Array<Maybe<LaForgeTagMap>>;
  Environment: LaForgeEnvironment;
};

export type LaForgeIncludedNetwork = {
  __typename?: 'IncludedNetwork';
  id: Scalars['ID'];
  name: Scalars['String'];
  includedHosts: Array<Maybe<Scalars['String']>>;
  Tags: Array<Maybe<LaForgeTag>>;
  Hosts: Array<Maybe<LaForgeHost>>;
  Network?: Maybe<LaForgeNetwork>;
  Environments: Array<Maybe<LaForgeEnvironment>>;
};

export type LaForgeLaForgePageInfo = {
  __typename?: 'LaForgePageInfo';
  total: Scalars['Int'];
  nextOffset: Scalars['Int'];
};

export type LaForgeMutation = {
  __typename?: 'Mutation';
  loadEnvironment?: Maybe<Array<Maybe<LaForgeEnvironment>>>;
  createBuild?: Maybe<LaForgeBuild>;
  deleteUser: Scalars['Boolean'];
  executePlan?: Maybe<LaForgeBuild>;
  deleteBuild: Scalars['String'];
  createTask: Scalars['Boolean'];
  dumpBuild: Scalars['String'];
  rebuild: Scalars['Boolean'];
  approveCommit: Scalars['Boolean'];
  cancelCommit: Scalars['Boolean'];
  createAgentTasks: Array<Maybe<LaForgeAgentTask>>;
  createBatchAgentTasks: Array<Maybe<LaForgeAgentTask>>;
  createEnviromentFromRepo: Array<Maybe<LaForgeEnvironment>>;
  updateEnviromentViaPull: Array<Maybe<LaForgeEnvironment>>;
  cancelBuild: Scalars['Boolean'];
  modifySelfPassword: Scalars['Boolean'];
  modifySelfUserInfo?: Maybe<LaForgeAuthUser>;
  createUser?: Maybe<LaForgeAuthUser>;
  modifyAdminUserInfo?: Maybe<LaForgeAuthUser>;
  modifyAdminPassword: Scalars['Boolean'];
  nukeBackend: Array<Maybe<LaForgeIntMap>>;
};

export type LaForgeMutationLoadEnvironmentArgs = {
  envFilePath: Scalars['String'];
};

export type LaForgeMutationCreateBuildArgs = {
  envUUID: Scalars['String'];
  renderFiles?: Scalars['Boolean'];
};

export type LaForgeMutationDeleteUserArgs = {
  userUUID: Scalars['String'];
};

export type LaForgeMutationExecutePlanArgs = {
  buildUUID: Scalars['String'];
};

export type LaForgeMutationDeleteBuildArgs = {
  buildUUID: Scalars['String'];
};

export type LaForgeMutationCreateTaskArgs = {
  proHostUUID: Scalars['String'];
  command: LaForgeAgentCommand;
  args: Scalars['String'];
};

export type LaForgeMutationDumpBuildArgs = {
  buildUUID: Scalars['String'];
};

export type LaForgeMutationRebuildArgs = {
  rootPlans: Array<Maybe<Scalars['String']>>;
};

export type LaForgeMutationApproveCommitArgs = {
  commitUUID: Scalars['String'];
};

export type LaForgeMutationCancelCommitArgs = {
  commitUUID: Scalars['String'];
};

export type LaForgeMutationCreateAgentTasksArgs = {
  hostHCLID: Scalars['String'];
  command: LaForgeAgentCommand;
  buildUUID: Scalars['String'];
  args: Array<Scalars['String']>;
  teams: Array<Scalars['Int']>;
};

export type LaForgeMutationCreateBatchAgentTasksArgs = {
  proHostUUIDs: Array<Scalars['String']>;
  command: LaForgeAgentCommand;
  args: Array<Scalars['String']>;
};

export type LaForgeMutationCreateEnviromentFromRepoArgs = {
  repoURL: Scalars['String'];
  branchName?: Scalars['String'];
  envFilePath: Scalars['String'];
};

export type LaForgeMutationUpdateEnviromentViaPullArgs = {
  envUUID: Scalars['String'];
};

export type LaForgeMutationCancelBuildArgs = {
  buildUUID: Scalars['String'];
};

export type LaForgeMutationModifySelfPasswordArgs = {
  currentPassword: Scalars['String'];
  newPassword: Scalars['String'];
};

export type LaForgeMutationModifySelfUserInfoArgs = {
  firstName?: Maybe<Scalars['String']>;
  lastName?: Maybe<Scalars['String']>;
  email?: Maybe<Scalars['String']>;
  phone?: Maybe<Scalars['String']>;
  company?: Maybe<Scalars['String']>;
  occupation?: Maybe<Scalars['String']>;
};

export type LaForgeMutationCreateUserArgs = {
  username: Scalars['String'];
  password: Scalars['String'];
  role: LaForgeRoleLevel;
  provider: LaForgeProviderType;
};

export type LaForgeMutationModifyAdminUserInfoArgs = {
  userID: Scalars['String'];
  username?: Maybe<Scalars['String']>;
  firstName?: Maybe<Scalars['String']>;
  lastName?: Maybe<Scalars['String']>;
  email?: Maybe<Scalars['String']>;
  phone?: Maybe<Scalars['String']>;
  company?: Maybe<Scalars['String']>;
  occupation?: Maybe<Scalars['String']>;
  role?: Maybe<LaForgeRoleLevel>;
  provider?: Maybe<LaForgeProviderType>;
};

export type LaForgeMutationModifyAdminPasswordArgs = {
  userID: Scalars['String'];
  newPassword: Scalars['String'];
};

export type LaForgeNetwork = {
  __typename?: 'Network';
  id: Scalars['ID'];
  hclId: Scalars['String'];
  name: Scalars['String'];
  cidr: Scalars['String'];
  vdiVisible: Scalars['Boolean'];
  vars?: Maybe<Array<Maybe<LaForgeVarsMap>>>;
  tags: Array<Maybe<LaForgeTagMap>>;
  Environment: LaForgeEnvironment;
  HostDependencies: Array<Maybe<LaForgeHostDependency>>;
  IncludedNetworks: Array<Maybe<LaForgeIncludedNetwork>>;
};

export type LaForgePlan = {
  __typename?: 'Plan';
  id: Scalars['ID'];
  stepNumber: Scalars['Int'];
  type: LaForgePlanType;
  NextPlans: Array<Maybe<LaForgePlan>>;
  PrevPlans: Array<Maybe<LaForgePlan>>;
  Build?: Maybe<LaForgeBuild>;
  Team?: Maybe<LaForgeTeam>;
  ProvisionedNetwork?: Maybe<LaForgeProvisionedNetwork>;
  ProvisionedHost?: Maybe<LaForgeProvisionedHost>;
  ProvisioningStep?: Maybe<LaForgeProvisioningStep>;
  ProvisioningScheduledStep?: Maybe<LaForgeProvisioningScheduledStep>;
  Status: LaForgeStatus;
  PlanDiffs: Array<Maybe<LaForgePlanDiff>>;
};

export type LaForgePlanCounts = {
  __typename?: 'PlanCounts';
  planning: Scalars['Int'];
  awaiting: Scalars['Int'];
  parentAwaiting: Scalars['Int'];
  inProgress: Scalars['Int'];
  failed: Scalars['Int'];
  complete: Scalars['Int'];
  tainted: Scalars['Int'];
  undefined: Scalars['Int'];
  toDelete: Scalars['Int'];
  deleteInProgress: Scalars['Int'];
  deleted: Scalars['Int'];
  toRebuild: Scalars['Int'];
  cancelled: Scalars['Int'];
};

export type LaForgePlanDiff = {
  __typename?: 'PlanDiff';
  id: Scalars['ID'];
  revision: Scalars['Int'];
  newState: LaForgeProvisionStatus;
  BuildCommit: LaForgeBuildCommit;
  Plan: LaForgePlan;
};

export enum LaForgePlanType {
  StartBuild = 'start_build',
  StartTeam = 'start_team',
  ProvisionNetwork = 'provision_network',
  ProvisionHost = 'provision_host',
  ExecuteStep = 'execute_step',
  Undefined = 'undefined'
}

export enum LaForgeProviderType {
  Local = 'LOCAL',
  Github = 'GITHUB',
  Openid = 'OPENID',
  Undefined = 'UNDEFINED'
}

export enum LaForgeProvisionStatus {
  Planning = 'PLANNING',
  Awaiting = 'AWAITING',
  Parentawaiting = 'PARENTAWAITING',
  Inprogress = 'INPROGRESS',
  Failed = 'FAILED',
  Complete = 'COMPLETE',
  Tainted = 'TAINTED',
  Undefined = 'UNDEFINED',
  Todelete = 'TODELETE',
  Deleteinprogress = 'DELETEINPROGRESS',
  Deleted = 'DELETED',
  Torebuild = 'TOREBUILD',
  Cancelled = 'CANCELLED'
}

export enum LaForgeProvisionStatusFor {
  Build = 'Build',
  Team = 'Team',
  Plan = 'Plan',
  ProvisionedNetwork = 'ProvisionedNetwork',
  ProvisionedHost = 'ProvisionedHost',
  ProvisioningStep = 'ProvisioningStep',
  Undefined = 'Undefined'
}

export type LaForgeProvisionedHost = {
  __typename?: 'ProvisionedHost';
  id: Scalars['ID'];
  subnetIp: Scalars['String'];
  addonType?: Maybe<LaForgeProvisionedHostAddonType>;
  vars?: Maybe<Array<Maybe<LaForgeVarsMap>>>;
  Status: LaForgeStatus;
  ProvisionedNetwork: LaForgeProvisionedNetwork;
  Host: LaForgeHost;
  EndStepPlan?: Maybe<LaForgePlan>;
  Build: LaForgeBuild;
  ProvisioningSteps: Array<Maybe<LaForgeProvisioningStep>>;
  ProvisioningScheduledSteps: Array<Maybe<LaForgeProvisioningScheduledStep>>;
  AgentStatuses: Array<Maybe<LaForgeAgentStatus>>;
  AgentTasks: Array<Maybe<LaForgeAgentTask>>;
  Plan: LaForgePlan;
  GinFileMiddleware?: Maybe<LaForgeGinFileMiddleware>;
};

export enum LaForgeProvisionedHostAddonType {
  Dns = 'DNS'
}

export type LaForgeProvisionedNetwork = {
  __typename?: 'ProvisionedNetwork';
  id: Scalars['ID'];
  name: Scalars['String'];
  cidr: Scalars['String'];
  vars?: Maybe<Array<Maybe<LaForgeVarsMap>>>;
  Status?: Maybe<LaForgeStatus>;
  Network?: Maybe<LaForgeNetwork>;
  Build?: Maybe<LaForgeBuild>;
  Team?: Maybe<LaForgeTeam>;
  ProvisionedHosts: Array<Maybe<LaForgeProvisionedHost>>;
  Plan?: Maybe<LaForgePlan>;
};

export type LaForgeProvisioningScheduledStep = {
  __typename?: 'ProvisioningScheduledStep';
  id: Scalars['ID'];
  type: LaForgeProvisioningScheduledStepType;
  runTime: Scalars['Time'];
  Status?: Maybe<LaForgeStatus>;
  ScheduledStep: LaForgeScheduledStep;
  ProvisionedHost: LaForgeProvisionedHost;
  Script?: Maybe<LaForgeScript>;
  Command?: Maybe<LaForgeCommand>;
  DNSRecord?: Maybe<LaForgeDnsRecord>;
  FileDelete?: Maybe<LaForgeFileDelete>;
  FileDownload?: Maybe<LaForgeFileDownload>;
  FileExtract?: Maybe<LaForgeFileExtract>;
  Ansible?: Maybe<LaForgeAnsible>;
  AgentTasks: Array<Maybe<LaForgeAgentTask>>;
  Plan?: Maybe<LaForgePlan>;
  GinFileMiddleware?: Maybe<LaForgeGinFileMiddleware>;
};

export enum LaForgeProvisioningScheduledStepType {
  Ansible = 'Ansible',
  Script = 'Script',
  Command = 'Command',
  DnsRecord = 'DNSRecord',
  FileDelete = 'FileDelete',
  FileDownload = 'FileDownload',
  FileExtract = 'FileExtract',
  Undefined = 'Undefined'
}

export type LaForgeProvisioningStep = {
  __typename?: 'ProvisioningStep';
  id: Scalars['ID'];
  type: LaForgeProvisioningStepType;
  stepNumber: Scalars['Int'];
  Status?: Maybe<LaForgeStatus>;
  ProvisionedHost: LaForgeProvisionedHost;
  Script?: Maybe<LaForgeScript>;
  Command?: Maybe<LaForgeCommand>;
  DNSRecord?: Maybe<LaForgeDnsRecord>;
  FileDelete?: Maybe<LaForgeFileDelete>;
  FileDownload?: Maybe<LaForgeFileDownload>;
  FileExtract?: Maybe<LaForgeFileExtract>;
  Ansible?: Maybe<LaForgeAnsible>;
  Plan?: Maybe<LaForgePlan>;
  AgentTasks: Array<Maybe<LaForgeAgentTask>>;
  GinFileMiddleware?: Maybe<LaForgeGinFileMiddleware>;
};

export enum LaForgeProvisioningStepType {
  Ansible = 'Ansible',
  Script = 'Script',
  Command = 'Command',
  DnsRecord = 'DNSRecord',
  FileDelete = 'FileDelete',
  FileDownload = 'FileDownload',
  FileExtract = 'FileExtract',
  Undefined = 'Undefined'
}

export type LaForgeQuery = {
  __typename?: 'Query';
  environments?: Maybe<Array<Maybe<LaForgeEnvironment>>>;
  environment?: Maybe<LaForgeEnvironment>;
  provisionedHost?: Maybe<LaForgeProvisionedHost>;
  provisionedNetwork?: Maybe<LaForgeProvisionedNetwork>;
  provisionedStep?: Maybe<LaForgeProvisioningStep>;
  plan?: Maybe<LaForgePlan>;
  getBuilds?: Maybe<Array<Maybe<LaForgeBuild>>>;
  build?: Maybe<LaForgeBuild>;
  getBuildCommits?: Maybe<Array<Maybe<LaForgeBuildCommit>>>;
  getBuildCommit?: Maybe<LaForgeBuildCommit>;
  status?: Maybe<LaForgeStatus>;
  agentStatus?: Maybe<LaForgeAgentStatus>;
  getServerTasks?: Maybe<Array<Maybe<LaForgeServerTask>>>;
  currentUser?: Maybe<LaForgeAuthUser>;
  getUserList?: Maybe<Array<Maybe<LaForgeAuthUser>>>;
  getCurrentUserTasks?: Maybe<Array<Maybe<LaForgeServerTask>>>;
  getAgentTasks?: Maybe<Array<Maybe<LaForgeAgentTask>>>;
  listAgentStatuses?: Maybe<Array<Maybe<LaForgeAgentStatus>>>;
  listBuildStatuses?: Maybe<Array<Maybe<LaForgeStatus>>>;
  getAllAgentStatus?: Maybe<LaForgeAgentStatusBatch>;
  getAllPlanStatus?: Maybe<LaForgeStatusBatch>;
  getPlanStatusCounts: LaForgePlanCounts;
  viewServerTaskLogs: Scalars['String'];
  viewAgentTask: LaForgeAgentTask;
  serverTasks?: Maybe<Array<Maybe<LaForgeServerTask>>>;
  serverConfig?: Maybe<LaForgeServerConfig>;
};

export type LaForgeQueryEnvironmentArgs = {
  envUUID: Scalars['String'];
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

export type LaForgeQueryPlanArgs = {
  planUUID: Scalars['String'];
};

export type LaForgeQueryBuildArgs = {
  buildUUID: Scalars['String'];
};

export type LaForgeQueryGetBuildCommitsArgs = {
  envUUID: Scalars['String'];
};

export type LaForgeQueryGetBuildCommitArgs = {
  buildCommitUUID: Scalars['String'];
};

export type LaForgeQueryStatusArgs = {
  statusUUID: Scalars['String'];
};

export type LaForgeQueryAgentStatusArgs = {
  clientId: Scalars['String'];
};

export type LaForgeQueryGetAgentTasksArgs = {
  proStepUUID: Scalars['String'];
};

export type LaForgeQueryListAgentStatusesArgs = {
  buildUUID: Scalars['String'];
};

export type LaForgeQueryListBuildStatusesArgs = {
  buildUUID: Scalars['String'];
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

export type LaForgeQueryGetPlanStatusCountsArgs = {
  buildUUID: Scalars['String'];
};

export type LaForgeQueryViewServerTaskLogsArgs = {
  taskID: Scalars['String'];
};

export type LaForgeQueryViewAgentTaskArgs = {
  taskID: Scalars['String'];
};

export type LaForgeQueryServerTasksArgs = {
  taskUUIDs: Array<Maybe<Scalars['String']>>;
};

export type LaForgeRepoCommit = {
  __typename?: 'RepoCommit';
  id: Scalars['ID'];
  revision: Scalars['Int'];
  hash: Scalars['String'];
  author: Scalars['String'];
  committer: Scalars['String'];
  pgpSignature: Scalars['String'];
  message: Scalars['String'];
  treeHash: Scalars['String'];
  parentHashes: Array<Maybe<Scalars['String']>>;
  Repository: LaForgeRepository;
};

export type LaForgeRepository = {
  __typename?: 'Repository';
  id: Scalars['ID'];
  repoUrl: Scalars['String'];
  branchName: Scalars['String'];
  environmentFilepath: Scalars['String'];
  Environments: Array<Maybe<LaForgeEnvironment>>;
  RepoCommits: Array<Maybe<LaForgeRepoCommit>>;
};

export enum LaForgeRoleLevel {
  Admin = 'ADMIN',
  User = 'USER',
  Undefined = 'UNDEFINED'
}

export type LaForgeScheduledStep = {
  __typename?: 'ScheduledStep';
  id: Scalars['ID'];
  hclId: Scalars['String'];
  name: Scalars['String'];
  description: Scalars['String'];
  step: Scalars['String'];
  type: LaForgeScheduledStepType;
  schedule?: Maybe<Scalars['String']>;
  runAt?: Maybe<Scalars['Int']>;
  Environment?: Maybe<LaForgeEnvironment>;
};

export enum LaForgeScheduledStepType {
  Cron = 'CRON',
  Runonce = 'RUNONCE'
}

export type LaForgeScript = {
  __typename?: 'Script';
  id: Scalars['ID'];
  hclId: Scalars['String'];
  name: Scalars['String'];
  language: Scalars['String'];
  description: Scalars['String'];
  source: Scalars['String'];
  sourceType: Scalars['String'];
  cooldown: Scalars['Int'];
  timeout: Scalars['Int'];
  ignoreErrors: Scalars['Boolean'];
  args: Array<Maybe<Scalars['String']>>;
  disabled: Scalars['Boolean'];
  vars?: Maybe<Array<Maybe<LaForgeVarsMap>>>;
  absPath: Scalars['String'];
  tags?: Maybe<Array<Maybe<LaForgeTagMap>>>;
  Users: Array<Maybe<LaForgeUser>>;
  Findings: Array<Maybe<LaForgeFinding>>;
  Environment: LaForgeEnvironment;
};

export type LaForgeServerConfig = {
  __typename?: 'ServerConfig';
  ConfigFile: Scalars['String'];
  Debug?: Maybe<Scalars['Boolean']>;
  LogFolder: Scalars['String'];
  GinMode: Scalars['String'];
  Builders: Scalars['Map'];
  Database?: Maybe<LaForgeDatabaseConfig>;
  Auth?: Maybe<LaForgeAuthConfig>;
  UI?: Maybe<LaForgeUiConfig>;
  Agent?: Maybe<LaForgeAgentConfig>;
  Graphql?: Maybe<LaForgeGraphqlConfig>;
};

export type LaForgeServerTask = {
  __typename?: 'ServerTask';
  id: Scalars['ID'];
  type: LaForgeServerTaskType;
  startTime?: Maybe<Scalars['Time']>;
  endTime?: Maybe<Scalars['Time']>;
  errors?: Maybe<Array<Maybe<Scalars['String']>>>;
  logFilePath?: Maybe<Scalars['String']>;
  AuthUser: LaForgeAuthUser;
  Status: LaForgeStatus;
  Environment?: Maybe<LaForgeEnvironment>;
  Build?: Maybe<LaForgeBuild>;
  BuildCommit?: Maybe<LaForgeBuildCommit>;
  GinFileMiddleware: Array<Maybe<LaForgeGinFileMiddleware>>;
};

export enum LaForgeServerTaskType {
  Loadenv = 'LOADENV',
  Createbuild = 'CREATEBUILD',
  Renderfiles = 'RENDERFILES',
  Deletebuild = 'DELETEBUILD',
  Rebuild = 'REBUILD',
  Executebuild = 'EXECUTEBUILD'
}

export type LaForgeStatus = {
  __typename?: 'Status';
  id: Scalars['ID'];
  state: LaForgeProvisionStatus;
  statusFor: LaForgeProvisionStatusFor;
  startedAt: Scalars['String'];
  endedAt: Scalars['String'];
  failed: Scalars['Boolean'];
  completed: Scalars['Boolean'];
  error?: Maybe<Scalars['String']>;
  Build?: Maybe<LaForgeBuild>;
  ProvisionedNetwork?: Maybe<LaForgeProvisionedNetwork>;
  ProvisionedHost?: Maybe<LaForgeProvisionedHost>;
  ProvisioningStep?: Maybe<LaForgeProvisioningStep>;
  Team?: Maybe<LaForgeTeam>;
  Plan?: Maybe<LaForgePlan>;
  ServerTask?: Maybe<LaForgeServerTask>;
  AdhocPlan?: Maybe<LaForgeAdhocPlan>;
  ProvisioningScheduledStep?: Maybe<LaForgeProvisioningScheduledStep>;
};

export type LaForgeStatusBatch = {
  __typename?: 'StatusBatch';
  statuses: Array<Maybe<LaForgeStatus>>;
  pageInfo: LaForgeLaForgePageInfo;
};

export type LaForgeSubscription = {
  __typename?: 'Subscription';
  updatedAgentStatus: LaForgeAgentStatus;
  updatedStatus: LaForgeStatus;
  updatedServerTask: LaForgeServerTask;
  updatedBuild: LaForgeBuild;
  updatedCommit: LaForgeBuildCommit;
  updatedAgentTask: LaForgeAgentTask;
  streamServerTaskLog: Scalars['String'];
};

export type LaForgeSubscriptionStreamServerTaskLogArgs = {
  taskID: Scalars['String'];
};

export type LaForgeTag = {
  __typename?: 'Tag';
  id: Scalars['ID'];
  uuid: Scalars['ID'];
  name: Scalars['String'];
  description: Array<Maybe<LaForgeTagMap>>;
};

export type LaForgeTeam = {
  __typename?: 'Team';
  id: Scalars['ID'];
  teamNumber: Scalars['Int'];
  Build: LaForgeBuild;
  Status?: Maybe<LaForgeStatus>;
  ProvisionedNetworks: Array<Maybe<LaForgeProvisionedNetwork>>;
  Plan: LaForgePlan;
};

export type LaForgeUiConfig = {
  __typename?: 'UIConfig';
  HttpsEnabled: Scalars['Boolean'];
  AllowedOrigins: Array<Maybe<Scalars['String']>>;
};

export type LaForgeUser = {
  __typename?: 'User';
  id: Scalars['ID'];
  hclId: Scalars['ID'];
  name: Scalars['String'];
  uuid: Scalars['String'];
  email: Scalars['String'];
  Tag: Array<Maybe<LaForgeTag>>;
  Environments: Array<Maybe<LaForgeEnvironment>>;
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

export type LaForgeGetUserListQuery = { __typename?: 'Query' } & {
  getUserList?: Maybe<Array<Maybe<{ __typename?: 'AuthUser' } & LaForgeUserListFieldsFragment>>>;
};

export type LaForgeUpdateUserMutationVariables = Exact<{
  userId: Scalars['String'];
  firstName?: Maybe<Scalars['String']>;
  lastName?: Maybe<Scalars['String']>;
  email: Scalars['String'];
  phone?: Maybe<Scalars['String']>;
  company?: Maybe<Scalars['String']>;
  occupation?: Maybe<Scalars['String']>;
  role: LaForgeRoleLevel;
  provider: LaForgeProviderType;
}>;

export type LaForgeUpdateUserMutation = { __typename?: 'Mutation' } & {
  modifyAdminUserInfo?: Maybe<{ __typename?: 'AuthUser' } & LaForgeUserListFieldsFragment>;
};

export type LaForgeCreateUserMutationVariables = Exact<{
  username: Scalars['String'];
  password: Scalars['String'];
  provider: LaForgeProviderType;
  role: LaForgeRoleLevel;
}>;

export type LaForgeCreateUserMutation = { __typename?: 'Mutation' } & {
  createUser?: Maybe<{ __typename?: 'AuthUser' } & LaForgeUserListFieldsFragment>;
};

export type LaForgeNukeBackendMutationVariables = Exact<{ [key: string]: never }>;

export type LaForgeNukeBackendMutation = { __typename?: 'Mutation' } & {
  nukeBackend: Array<Maybe<{ __typename?: 'intMap' } & Pick<LaForgeIntMap, 'key' | 'value'>>>;
};

export type LaForgeGetAgentTasksQueryVariables = Exact<{
  proStepId: Scalars['String'];
}>;

export type LaForgeGetAgentTasksQuery = { __typename?: 'Query' } & {
  getAgentTasks?: Maybe<Array<Maybe<{ __typename?: 'AgentTask' } & LaForgeAgentTaskFieldsFragment>>>;
};

export type LaForgeListAgentStatusesQueryVariables = Exact<{
  buildUUID: Scalars['String'];
}>;

export type LaForgeListAgentStatusesQuery = { __typename?: 'Query' } & {
  listAgentStatuses?: Maybe<Array<Maybe<{ __typename?: 'AgentStatus' } & LaForgeAgentStatusFieldsFragment>>>;
};

export type LaForgeGetCurrentUserQueryVariables = Exact<{ [key: string]: never }>;

export type LaForgeGetCurrentUserQuery = { __typename?: 'Query' } & {
  currentUser?: Maybe<{ __typename?: 'AuthUser' } & LaForgeAuthUserFieldsFragment>;
};

export type LaForgeGetBuildTreeQueryVariables = Exact<{
  buildId: Scalars['String'];
}>;

export type LaForgeGetBuildTreeQuery = { __typename?: 'Query' } & {
  build?: Maybe<
    { __typename?: 'Build' } & Pick<LaForgeBuild, 'id' | 'revision'> & {
        Environment: { __typename?: 'Environment' } & Pick<
          LaForgeEnvironment,
          'id' | 'name' | 'description' | 'teamCount' | 'adminCidrs' | 'exposedVdiPorts'
        >;
        RepoCommit: { __typename?: 'RepoCommit' } & Pick<LaForgeRepoCommit, 'id' | 'hash' | 'committer'> & {
            Repository: { __typename?: 'Repository' } & Pick<LaForgeRepository, 'id' | 'repoUrl'>;
          };
        Status: { __typename?: 'Status' } & Pick<LaForgeStatus, 'id'>;
        Teams: Array<
          Maybe<
            { __typename?: 'Team' } & Pick<LaForgeTeam, 'id' | 'teamNumber'> & {
                Status?: Maybe<{ __typename?: 'Status' } & Pick<LaForgeStatus, 'id'>>;
                Plan: { __typename?: 'Plan' } & Pick<LaForgePlan, 'id'> & { Status: { __typename?: 'Status' } & Pick<LaForgeStatus, 'id'> };
                ProvisionedNetworks: Array<
                  Maybe<
                    { __typename?: 'ProvisionedNetwork' } & Pick<LaForgeProvisionedNetwork, 'id' | 'name' | 'cidr'> & {
                        Network?: Maybe<
                          { __typename?: 'Network' } & Pick<LaForgeNetwork, 'id' | 'vdiVisible'> & {
                              vars?: Maybe<Array<Maybe<{ __typename?: 'varsMap' } & Pick<LaForgeVarsMap, 'key' | 'value'>>>>;
                              tags: Array<Maybe<{ __typename?: 'tagMap' } & Pick<LaForgeTagMap, 'key' | 'value'>>>;
                            }
                        >;
                        Status?: Maybe<{ __typename?: 'Status' } & Pick<LaForgeStatus, 'id'>>;
                        Plan?: Maybe<
                          { __typename?: 'Plan' } & Pick<LaForgePlan, 'id'> & {
                              Status: { __typename?: 'Status' } & Pick<LaForgeStatus, 'id'>;
                            }
                        >;
                        ProvisionedHosts: Array<
                          Maybe<
                            { __typename?: 'ProvisionedHost' } & Pick<LaForgeProvisionedHost, 'id' | 'subnetIp'> & {
                                Host: { __typename?: 'Host' } & Pick<
                                  LaForgeHost,
                                  | 'id'
                                  | 'hostname'
                                  | 'description'
                                  | 'OS'
                                  | 'allowMacChanges'
                                  | 'exposedTcpPorts'
                                  | 'exposedUdpPorts'
                                  | 'userGroups'
                                  | 'overridePassword'
                                > & {
                                    vars?: Maybe<Array<Maybe<{ __typename?: 'varsMap' } & Pick<LaForgeVarsMap, 'key' | 'value'>>>>;
                                    tags: Array<Maybe<{ __typename?: 'tagMap' } & Pick<LaForgeTagMap, 'key' | 'value'>>>;
                                  };
                                Status: { __typename?: 'Status' } & Pick<LaForgeStatus, 'id'>;
                                Plan: { __typename?: 'Plan' } & Pick<LaForgePlan, 'id'> & {
                                    Status: { __typename?: 'Status' } & Pick<LaForgeStatus, 'id'>;
                                  };
                                ProvisioningSteps: Array<
                                  Maybe<
                                    { __typename?: 'ProvisioningStep' } & Pick<LaForgeProvisioningStep, 'id' | 'type' | 'stepNumber'> & {
                                        Script?: Maybe<
                                          { __typename?: 'Script' } & Pick<
                                            LaForgeScript,
                                            'id' | 'name' | 'language' | 'description' | 'source' | 'sourceType' | 'disabled' | 'args'
                                          > & {
                                              vars?: Maybe<
                                                Array<Maybe<{ __typename?: 'varsMap' } & Pick<LaForgeVarsMap, 'key' | 'value'>>>
                                              >;
                                              tags?: Maybe<Array<Maybe<{ __typename?: 'tagMap' } & Pick<LaForgeTagMap, 'key' | 'value'>>>>;
                                            }
                                        >;
                                        Command?: Maybe<
                                          { __typename?: 'Command' } & Pick<
                                            LaForgeCommand,
                                            'id' | 'name' | 'description' | 'program' | 'args' | 'disabled'
                                          > & {
                                              vars?: Maybe<
                                                Array<Maybe<{ __typename?: 'varsMap' } & Pick<LaForgeVarsMap, 'key' | 'value'>>>
                                              >;
                                              tags?: Maybe<Array<Maybe<{ __typename?: 'tagMap' } & Pick<LaForgeTagMap, 'key' | 'value'>>>>;
                                            }
                                        >;
                                        DNSRecord?: Maybe<
                                          { __typename?: 'DNSRecord' } & Pick<
                                            LaForgeDnsRecord,
                                            'id' | 'name' | 'values' | 'type' | 'zone' | 'disabled'
                                          > & {
                                              vars: Array<Maybe<{ __typename?: 'varsMap' } & Pick<LaForgeVarsMap, 'key' | 'value'>>>;
                                              tags: Array<Maybe<{ __typename?: 'tagMap' } & Pick<LaForgeTagMap, 'key' | 'value'>>>;
                                            }
                                        >;
                                        FileDownload?: Maybe<
                                          { __typename?: 'FileDownload' } & Pick<
                                            LaForgeFileDownload,
                                            'id' | 'source' | 'sourceType' | 'destination' | 'disabled'
                                          > & { tags: Array<Maybe<{ __typename?: 'tagMap' } & Pick<LaForgeTagMap, 'key' | 'value'>>> }
                                        >;
                                        FileDelete?: Maybe<
                                          { __typename?: 'FileDelete' } & Pick<LaForgeFileDelete, 'id' | 'path'> & {
                                              tags: Array<Maybe<{ __typename?: 'tagMap' } & Pick<LaForgeTagMap, 'key' | 'value'>>>;
                                            }
                                        >;
                                        FileExtract?: Maybe<
                                          { __typename?: 'FileExtract' } & Pick<
                                            LaForgeFileExtract,
                                            'id' | 'source' | 'destination' | 'type'
                                          > & { tags: Array<Maybe<{ __typename?: 'tagMap' } & Pick<LaForgeTagMap, 'key' | 'value'>>> }
                                        >;
                                        Status?: Maybe<{ __typename?: 'Status' } & Pick<LaForgeStatus, 'id'>>;
                                        Plan?: Maybe<
                                          { __typename?: 'Plan' } & Pick<LaForgePlan, 'id'> & {
                                              Status: { __typename?: 'Status' } & Pick<LaForgeStatus, 'id'>;
                                            }
                                        >;
                                      }
                                  >
                                >;
                                AgentStatuses: Array<Maybe<{ __typename?: 'AgentStatus' } & Pick<LaForgeAgentStatus, 'clientId'>>>;
                              }
                          >
                        >;
                      }
                  >
                >;
              }
          >
        >;
        ServerTasks: Array<Maybe<{ __typename?: 'ServerTask' } & Pick<LaForgeServerTask, 'id'>>>;
      }
  >;
};

export type LaForgeGetBuildPlansQueryVariables = Exact<{
  buildId: Scalars['String'];
}>;

export type LaForgeGetBuildPlansQuery = { __typename?: 'Query' } & {
  build?: Maybe<
    { __typename?: 'Build' } & Pick<LaForgeBuild, 'id'> & { Plans: Array<Maybe<{ __typename?: 'Plan' } & LaForgePlanFieldsFragment>> }
  >;
};

export type LaForgeGetBuildStatusesQueryVariables = Exact<{
  buildUUID: Scalars['String'];
}>;

export type LaForgeGetBuildStatusesQuery = { __typename?: 'Query' } & {
  build?: Maybe<
    { __typename?: 'Build' } & Pick<LaForgeBuild, 'id'> & {
        Plans: Array<
          Maybe<{ __typename?: 'Plan' } & Pick<LaForgePlan, 'id'> & { Status: { __typename?: 'Status' } & LaForgeStatusFieldsFragment }>
        >;
        Teams: Array<
          Maybe<
            { __typename?: 'Team' } & Pick<LaForgeTeam, 'id'> & {
                Status?: Maybe<{ __typename?: 'Status' } & LaForgeStatusFieldsFragment>;
                ProvisionedNetworks: Array<
                  Maybe<
                    { __typename?: 'ProvisionedNetwork' } & Pick<LaForgeProvisionedNetwork, 'id'> & {
                        Status?: Maybe<{ __typename?: 'Status' } & LaForgeStatusFieldsFragment>;
                        ProvisionedHosts: Array<
                          Maybe<
                            { __typename?: 'ProvisionedHost' } & Pick<LaForgeProvisionedHost, 'id'> & {
                                Status: { __typename?: 'Status' } & LaForgeStatusFieldsFragment;
                              }
                          >
                        >;
                      }
                  >
                >;
              }
          >
        >;
      }
  >;
};

export type LaForgeGetBuildCommitsQueryVariables = Exact<{
  buildId: Scalars['String'];
}>;

export type LaForgeGetBuildCommitsQuery = { __typename?: 'Query' } & {
  build?: Maybe<
    { __typename?: 'Build' } & Pick<LaForgeBuild, 'id'> & {
        BuildCommits: Array<Maybe<{ __typename?: 'BuildCommit' } & LaForgeBuildCommitFieldsFragment>>;
      }
  >;
};

export type LaForgeGetPlanStatusCountsQueryVariables = Exact<{
  buildId: Scalars['String'];
}>;

export type LaForgeGetPlanStatusCountsQuery = { __typename?: 'Query' } & {
  getPlanStatusCounts: { __typename?: 'PlanCounts' } & Pick<
    LaForgePlanCounts,
    | 'planning'
    | 'awaiting'
    | 'parentAwaiting'
    | 'inProgress'
    | 'failed'
    | 'complete'
    | 'tainted'
    | 'toDelete'
    | 'deleteInProgress'
    | 'deleted'
    | 'toRebuild'
    | 'cancelled'
  >;
};

export type LaForgeListBuildCommitsQueryVariables = Exact<{
  envUUID: Scalars['String'];
}>;

export type LaForgeListBuildCommitsQuery = { __typename?: 'Query' } & {
  getBuildCommits?: Maybe<Array<Maybe<{ __typename?: 'BuildCommit' } & LaForgeBuildCommitFieldsFragment>>>;
};

export type LaForgeGetBuildCommitQueryVariables = Exact<{
  buildCommitUUID: Scalars['String'];
}>;

export type LaForgeGetBuildCommitQuery = { __typename?: 'Query' } & {
  getBuildCommit?: Maybe<
    { __typename?: 'BuildCommit' } & Pick<LaForgeBuildCommit, 'id' | 'revision' | 'state' | 'type'> & {
        Build: { __typename?: 'Build' } & Pick<LaForgeBuild, 'id' | 'revision'> & {
            RepoCommit: { __typename?: 'RepoCommit' } & Pick<LaForgeRepoCommit, 'id' | 'hash' | 'author'> & {
                Repository: { __typename?: 'Repository' } & Pick<LaForgeRepository, 'id' | 'repoUrl'>;
              };
            Environment: { __typename?: 'Environment' } & Pick<LaForgeEnvironment, 'id' | 'name'>;
            Teams: Array<
              Maybe<
                { __typename?: 'Team' } & Pick<LaForgeTeam, 'id' | 'teamNumber'> & {
                    Plan: { __typename?: 'Plan' } & Pick<LaForgePlan, 'id'> & {
                        Status: { __typename?: 'Status' } & Pick<LaForgeStatus, 'id' | 'state'>;
                      };
                    ProvisionedNetworks: Array<
                      Maybe<
                        { __typename?: 'ProvisionedNetwork' } & Pick<LaForgeProvisionedNetwork, 'id' | 'name' | 'cidr'> & {
                            Plan?: Maybe<
                              { __typename?: 'Plan' } & Pick<LaForgePlan, 'id'> & {
                                  Status: { __typename?: 'Status' } & Pick<LaForgeStatus, 'id' | 'state'>;
                                }
                            >;
                            Network?: Maybe<{ __typename?: 'Network' } & Pick<LaForgeNetwork, 'id' | 'vdiVisible'>>;
                            ProvisionedHosts: Array<
                              Maybe<
                                { __typename?: 'ProvisionedHost' } & Pick<LaForgeProvisionedHost, 'id' | 'subnetIp'> & {
                                    Plan: { __typename?: 'Plan' } & Pick<LaForgePlan, 'id'> & {
                                        Status: { __typename?: 'Status' } & Pick<LaForgeStatus, 'id' | 'state'>;
                                      };
                                    Host: { __typename?: 'Host' } & Pick<LaForgeHost, 'id' | 'hostname'>;
                                    ProvisioningSteps: Array<
                                      Maybe<
                                        { __typename?: 'ProvisioningStep' } & Pick<LaForgeProvisioningStep, 'id' | 'stepNumber'> & {
                                            Plan?: Maybe<
                                              { __typename?: 'Plan' } & Pick<LaForgePlan, 'id'> & {
                                                  Status: { __typename?: 'Status' } & Pick<LaForgeStatus, 'id' | 'state'>;
                                                }
                                            >;
                                          }
                                      >
                                    >;
                                  }
                              >
                            >;
                          }
                      >
                    >;
                  }
              >
            >;
          };
        PlanDiffs: Array<
          Maybe<
            { __typename?: 'PlanDiff' } & Pick<LaForgePlanDiff, 'id' | 'newState'> & {
                Plan: { __typename?: 'Plan' } & Pick<LaForgePlan, 'id'>;
              }
          >
        >;
      }
  >;
};

export type LaForgeGetEnvironmentQueryVariables = Exact<{
  envId: Scalars['String'];
}>;

export type LaForgeGetEnvironmentQuery = { __typename?: 'Query' } & {
  environment?: Maybe<
    { __typename?: 'Environment' } & Pick<
      LaForgeEnvironment,
      'id' | 'competitionId' | 'name' | 'description' | 'builder' | 'teamCount' | 'revision' | 'adminCidrs' | 'exposedVdiPorts'
    > & {
        tags?: Maybe<Array<Maybe<{ __typename?: 'tagMap' } & Pick<LaForgeTagMap, 'key' | 'value'>>>>;
        config?: Maybe<Array<Maybe<{ __typename?: 'configMap' } & Pick<LaForgeConfigMap, 'key' | 'value'>>>>;
        Users: Array<Maybe<{ __typename?: 'User' } & Pick<LaForgeUser, 'id' | 'name' | 'uuid' | 'email'>>>;
        Repositories: Array<Maybe<{ __typename?: 'Repository' } & Pick<LaForgeRepository, 'id' | 'repoUrl' | 'branchName'>>>;
        Builds: Array<
          Maybe<
            { __typename?: 'Build' } & Pick<LaForgeBuild, 'id' | 'revision'> & {
                Status: { __typename?: 'Status' } & LaForgeStatusFieldsFragment;
                Teams: Array<
                  Maybe<
                    { __typename?: 'Team' } & Pick<LaForgeTeam, 'id' | 'teamNumber'> & {
                        Status?: Maybe<{ __typename?: 'Status' } & LaForgeStatusFieldsFragment>;
                        ProvisionedNetworks: Array<
                          Maybe<
                            { __typename?: 'ProvisionedNetwork' } & Pick<LaForgeProvisionedNetwork, 'id' | 'name' | 'cidr'> & {
                                Status?: Maybe<{ __typename?: 'Status' } & LaForgeStatusFieldsFragment>;
                                Network?: Maybe<
                                  { __typename?: 'Network' } & Pick<LaForgeNetwork, 'id' | 'vdiVisible'> & {
                                      vars?: Maybe<Array<Maybe<{ __typename?: 'varsMap' } & Pick<LaForgeVarsMap, 'key' | 'value'>>>>;
                                      tags: Array<Maybe<{ __typename?: 'tagMap' } & Pick<LaForgeTagMap, 'key' | 'value'>>>;
                                    }
                                >;
                                ProvisionedHosts: Array<
                                  Maybe<
                                    { __typename?: 'ProvisionedHost' } & Pick<LaForgeProvisionedHost, 'id' | 'subnetIp'> & {
                                        Status: { __typename?: 'Status' } & LaForgeStatusFieldsFragment;
                                        Host: { __typename?: 'Host' } & Pick<
                                          LaForgeHost,
                                          | 'id'
                                          | 'hostname'
                                          | 'description'
                                          | 'OS'
                                          | 'allowMacChanges'
                                          | 'exposedTcpPorts'
                                          | 'exposedUdpPorts'
                                          | 'userGroups'
                                          | 'overridePassword'
                                        > & {
                                            vars?: Maybe<Array<Maybe<{ __typename?: 'varsMap' } & Pick<LaForgeVarsMap, 'key' | 'value'>>>>;
                                            tags: Array<Maybe<{ __typename?: 'tagMap' } & Pick<LaForgeTagMap, 'key' | 'value'>>>;
                                          };
                                        ProvisioningSteps: Array<
                                          Maybe<
                                            { __typename?: 'ProvisioningStep' } & Pick<LaForgeProvisioningStep, 'id' | 'type'> & {
                                                Status?: Maybe<{ __typename?: 'Status' } & LaForgeStatusFieldsFragment>;
                                                Script?: Maybe<
                                                  { __typename?: 'Script' } & Pick<
                                                    LaForgeScript,
                                                    | 'id'
                                                    | 'name'
                                                    | 'language'
                                                    | 'description'
                                                    | 'source'
                                                    | 'sourceType'
                                                    | 'disabled'
                                                    | 'args'
                                                  > & {
                                                      vars?: Maybe<
                                                        Array<Maybe<{ __typename?: 'varsMap' } & Pick<LaForgeVarsMap, 'key' | 'value'>>>
                                                      >;
                                                      tags?: Maybe<
                                                        Array<Maybe<{ __typename?: 'tagMap' } & Pick<LaForgeTagMap, 'key' | 'value'>>>
                                                      >;
                                                    }
                                                >;
                                                Command?: Maybe<
                                                  { __typename?: 'Command' } & Pick<
                                                    LaForgeCommand,
                                                    'id' | 'name' | 'description' | 'program' | 'args' | 'disabled'
                                                  > & {
                                                      vars?: Maybe<
                                                        Array<Maybe<{ __typename?: 'varsMap' } & Pick<LaForgeVarsMap, 'key' | 'value'>>>
                                                      >;
                                                      tags?: Maybe<
                                                        Array<Maybe<{ __typename?: 'tagMap' } & Pick<LaForgeTagMap, 'key' | 'value'>>>
                                                      >;
                                                    }
                                                >;
                                                DNSRecord?: Maybe<
                                                  { __typename?: 'DNSRecord' } & Pick<
                                                    LaForgeDnsRecord,
                                                    'id' | 'name' | 'values' | 'type' | 'zone' | 'disabled'
                                                  > & {
                                                      vars: Array<
                                                        Maybe<{ __typename?: 'varsMap' } & Pick<LaForgeVarsMap, 'key' | 'value'>>
                                                      >;
                                                      tags: Array<Maybe<{ __typename?: 'tagMap' } & Pick<LaForgeTagMap, 'key' | 'value'>>>;
                                                    }
                                                >;
                                                FileDownload?: Maybe<
                                                  { __typename?: 'FileDownload' } & Pick<
                                                    LaForgeFileDownload,
                                                    'id' | 'source' | 'sourceType' | 'destination' | 'disabled'
                                                  > & {
                                                      tags: Array<Maybe<{ __typename?: 'tagMap' } & Pick<LaForgeTagMap, 'key' | 'value'>>>;
                                                    }
                                                >;
                                                FileDelete?: Maybe<
                                                  { __typename?: 'FileDelete' } & Pick<LaForgeFileDelete, 'id' | 'path'> & {
                                                      tags: Array<Maybe<{ __typename?: 'tagMap' } & Pick<LaForgeTagMap, 'key' | 'value'>>>;
                                                    }
                                                >;
                                                FileExtract?: Maybe<
                                                  { __typename?: 'FileExtract' } & Pick<
                                                    LaForgeFileExtract,
                                                    'id' | 'source' | 'destination' | 'type'
                                                  > & {
                                                      tags: Array<Maybe<{ __typename?: 'tagMap' } & Pick<LaForgeTagMap, 'key' | 'value'>>>;
                                                    }
                                                >;
                                              }
                                          >
                                        >;
                                      }
                                  >
                                >;
                              }
                          >
                        >;
                      }
                  >
                >;
              }
          >
        >;
      }
  >;
};

export type LaForgeGetEnvironmentsQueryVariables = Exact<{ [key: string]: never }>;

export type LaForgeGetEnvironmentsQuery = { __typename?: 'Query' } & {
  environments?: Maybe<
    Array<
      Maybe<
        { __typename?: 'Environment' } & Pick<LaForgeEnvironment, 'id' | 'name' | 'competitionId' | 'revision'> & {
            Builds: Array<Maybe<{ __typename?: 'Build' } & Pick<LaForgeBuild, 'id' | 'revision'>>>;
          }
      >
    >
  >;
};

export type LaForgeListEnvironmentsQueryVariables = Exact<{ [key: string]: never }>;

export type LaForgeListEnvironmentsQuery = { __typename?: 'Query' } & {
  environments?: Maybe<
    Array<
      Maybe<
        { __typename?: 'Environment' } & Pick<LaForgeEnvironment, 'id' | 'name' | 'teamCount'> & {
            Repositories: Array<
              Maybe<
                { __typename?: 'Repository' } & Pick<LaForgeRepository, 'id' | 'repoUrl' | 'branchName' | 'environmentFilepath'> & {
                    RepoCommits: Array<
                      Maybe<{ __typename?: 'RepoCommit' } & Pick<LaForgeRepoCommit, 'id' | 'revision' | 'author' | 'hash'>>
                    >;
                  }
              >
            >;
            Networks: Array<Maybe<{ __typename?: 'Network' } & Pick<LaForgeNetwork, 'id'>>>;
            Hosts: Array<Maybe<{ __typename?: 'Host' } & Pick<LaForgeHost, 'id'>>>;
            ServerTasks: Array<Maybe<{ __typename?: 'ServerTask' } & Pick<LaForgeServerTask, 'id'>>>;
          }
      >
    >
  >;
};

export type LaForgeGetEnvironmentInfoQueryVariables = Exact<{
  envId: Scalars['String'];
}>;

export type LaForgeGetEnvironmentInfoQuery = { __typename?: 'Query' } & {
  environment?: Maybe<
    { __typename?: 'Environment' } & Pick<
      LaForgeEnvironment,
      'id' | 'competitionId' | 'name' | 'description' | 'builder' | 'teamCount' | 'revision' | 'adminCidrs' | 'exposedVdiPorts'
    > & {
        tags?: Maybe<Array<Maybe<{ __typename?: 'tagMap' } & Pick<LaForgeTagMap, 'key' | 'value'>>>>;
        config?: Maybe<Array<Maybe<{ __typename?: 'configMap' } & Pick<LaForgeConfigMap, 'key' | 'value'>>>>;
        Users: Array<Maybe<{ __typename?: 'User' } & Pick<LaForgeUser, 'id' | 'name' | 'uuid' | 'email'>>>;
        Builds: Array<
          Maybe<
            { __typename?: 'Build' } & Pick<LaForgeBuild, 'id' | 'revision'> & {
                Status: { __typename?: 'Status' } & LaForgeStatusFieldsFragment;
              }
          >
        >;
      }
  >;
};

export type LaForgeStatusFieldsFragment = { __typename?: 'Status' } & Pick<
  LaForgeStatus,
  'id' | 'state' | 'startedAt' | 'endedAt' | 'failed' | 'completed' | 'error'
>;

export type LaForgeAgentStatusFieldsFragment = { __typename?: 'AgentStatus' } & Pick<
  LaForgeAgentStatus,
  | 'clientId'
  | 'hostname'
  | 'upTime'
  | 'bootTime'
  | 'numProcs'
  | 'OS'
  | 'hostID'
  | 'load1'
  | 'load5'
  | 'load15'
  | 'totalMem'
  | 'freeMem'
  | 'usedMem'
  | 'timestamp'
>;

export type LaForgePlanFieldsFragment = { __typename?: 'Plan' } & Pick<LaForgePlan, 'id' | 'stepNumber' | 'type'> & {
    Status: { __typename?: 'Status' } & LaForgeStatusFieldsFragment;
    PlanDiffs: Array<Maybe<{ __typename?: 'PlanDiff' } & LaForgePlanDiffFieldsFragment>>;
  };

export type LaForgePlanDiffFieldsFragment = { __typename?: 'PlanDiff' } & Pick<LaForgePlanDiff, 'id' | 'revision' | 'newState'>;

export type LaForgeBuildCommitFieldsFragment = { __typename?: 'BuildCommit' } & Pick<
  LaForgeBuildCommit,
  'id' | 'revision' | 'state' | 'type'
> & {
    Build: { __typename?: 'Build' } & Pick<LaForgeBuild, 'id' | 'revision'> & {
        RepoCommit: { __typename?: 'RepoCommit' } & Pick<LaForgeRepoCommit, 'id' | 'hash' | 'author'> & {
            Repository: { __typename?: 'Repository' } & Pick<LaForgeRepository, 'id' | 'repoUrl'>;
          };
        Status: { __typename?: 'Status' } & Pick<LaForgeStatus, 'id' | 'state'>;
        Environment: { __typename?: 'Environment' } & Pick<LaForgeEnvironment, 'id'>;
      };
    ServerTasks: Array<Maybe<{ __typename?: 'ServerTask' } & Pick<LaForgeServerTask, 'id' | 'startTime' | 'endTime'>>>;
  };

export type LaForgeAuthUserFieldsFragment = { __typename?: 'AuthUser' } & Pick<
  LaForgeAuthUser,
  'id' | 'username' | 'role' | 'provider' | 'firstName' | 'lastName' | 'email' | 'phone' | 'company' | 'occupation' | 'publicKey'
>;

export type LaForgeAgentTaskFieldsFragment = { __typename?: 'AgentTask' } & Pick<
  LaForgeAgentTask,
  'id' | 'state' | 'command' | 'args' | 'number' | 'output' | 'errorMessage'
>;

export type LaForgePageInfoFieldsFragment = { __typename?: 'LaForgePageInfo' } & Pick<LaForgeLaForgePageInfo, 'total' | 'nextOffset'>;

export type LaForgeUserListFieldsFragment = { __typename?: 'AuthUser' } & Pick<
  LaForgeAuthUser,
  'id' | 'firstName' | 'lastName' | 'username' | 'provider' | 'role' | 'email' | 'phone' | 'company' | 'occupation'
>;

export type LaForgeRebuildMutationVariables = Exact<{
  rootPlans: Array<Maybe<Scalars['String']>> | Maybe<Scalars['String']>;
}>;

export type LaForgeRebuildMutation = { __typename?: 'Mutation' } & Pick<LaForgeMutation, 'rebuild'>;

export type LaForgeDeleteBuildMutationVariables = Exact<{
  buildId: Scalars['String'];
}>;

export type LaForgeDeleteBuildMutation = { __typename?: 'Mutation' } & Pick<LaForgeMutation, 'deleteBuild'>;

export type LaForgeExecuteBuildMutationVariables = Exact<{
  buildId: Scalars['String'];
}>;

export type LaForgeExecuteBuildMutation = { __typename?: 'Mutation' } & {
  executePlan?: Maybe<{ __typename?: 'Build' } & Pick<LaForgeBuild, 'id'>>;
};

export type LaForgeCancelBuildMutationVariables = Exact<{
  buildId: Scalars['String'];
}>;

export type LaForgeCancelBuildMutation = { __typename?: 'Mutation' } & Pick<LaForgeMutation, 'cancelBuild'>;

export type LaForgeCreateBuildMutationVariables = Exact<{
  envId: Scalars['String'];
}>;

export type LaForgeCreateBuildMutation = { __typename?: 'Mutation' } & {
  createBuild?: Maybe<{ __typename?: 'Build' } & Pick<LaForgeBuild, 'id'>>;
};

export type LaForgeModifyCurrentUserMutationVariables = Exact<{
  firstName?: Maybe<Scalars['String']>;
  lastName?: Maybe<Scalars['String']>;
  email?: Maybe<Scalars['String']>;
  phone?: Maybe<Scalars['String']>;
  company?: Maybe<Scalars['String']>;
  occupation?: Maybe<Scalars['String']>;
}>;

export type LaForgeModifyCurrentUserMutation = { __typename?: 'Mutation' } & {
  modifySelfUserInfo?: Maybe<{ __typename?: 'AuthUser' } & LaForgeAuthUserFieldsFragment>;
};

export type LaForgeCreateEnvironmentFromGitMutationVariables = Exact<{
  repoURL: Scalars['String'];
  branchName: Scalars['String'];
  envFilePath: Scalars['String'];
}>;

export type LaForgeCreateEnvironmentFromGitMutation = { __typename?: 'Mutation' } & {
  createEnviromentFromRepo: Array<Maybe<{ __typename?: 'Environment' } & Pick<LaForgeEnvironment, 'id'>>>;
};

export type LaForgeUpdateEnvironmentViaPullMutationVariables = Exact<{
  envId: Scalars['String'];
}>;

export type LaForgeUpdateEnvironmentViaPullMutation = { __typename?: 'Mutation' } & {
  updateEnviromentViaPull: Array<Maybe<{ __typename?: 'Environment' } & Pick<LaForgeEnvironment, 'id'>>>;
};

export type LaForgeApproveBuildCommitMutationVariables = Exact<{
  buildCommitId: Scalars['String'];
}>;

export type LaForgeApproveBuildCommitMutation = { __typename?: 'Mutation' } & Pick<LaForgeMutation, 'approveCommit'>;

export type LaForgeCancelBuildCommitMutationVariables = Exact<{
  buildCommitId: Scalars['String'];
}>;

export type LaForgeCancelBuildCommitMutation = { __typename?: 'Mutation' } & Pick<LaForgeMutation, 'cancelCommit'>;

export type LaForgeGetStatusQueryVariables = Exact<{
  statusId: Scalars['String'];
}>;

export type LaForgeGetStatusQuery = { __typename?: 'Query' } & { status?: Maybe<{ __typename?: 'Status' } & LaForgeStatusFieldsFragment> };

export type LaForgeGetAgentStatusQueryVariables = Exact<{
  clientId: Scalars['String'];
}>;

export type LaForgeGetAgentStatusQuery = { __typename?: 'Query' } & {
  agentStatus?: Maybe<{ __typename?: 'AgentStatus' } & LaForgeAgentStatusFieldsFragment>;
};

export type LaForgeGetAllPlanStatusesQueryVariables = Exact<{
  buildId: Scalars['String'];
  count: Scalars['Int'];
  offset: Scalars['Int'];
}>;

export type LaForgeGetAllPlanStatusesQuery = { __typename?: 'Query' } & {
  getAllPlanStatus?: Maybe<
    { __typename?: 'StatusBatch' } & {
      statuses: Array<Maybe<{ __typename?: 'Status' } & LaForgeStatusFieldsFragment>>;
      pageInfo: { __typename?: 'LaForgePageInfo' } & LaForgePageInfoFieldsFragment;
    }
  >;
};

export type LaForgeGetAllAgentStatusesQueryVariables = Exact<{
  buildId: Scalars['String'];
  count: Scalars['Int'];
  offset: Scalars['Int'];
}>;

export type LaForgeGetAllAgentStatusesQuery = { __typename?: 'Query' } & {
  getAllAgentStatus?: Maybe<
    { __typename?: 'AgentStatusBatch' } & {
      agentStatuses: Array<Maybe<{ __typename?: 'AgentStatus' } & LaForgeAgentStatusFieldsFragment>>;
      pageInfo: { __typename?: 'LaForgePageInfo' } & LaForgePageInfoFieldsFragment;
    }
  >;
};

export type LaForgeListBuildStatusesQueryVariables = Exact<{
  buildUUID: Scalars['String'];
}>;

export type LaForgeListBuildStatusesQuery = { __typename?: 'Query' } & {
  listBuildStatuses?: Maybe<Array<Maybe<{ __typename?: 'Status' } & LaForgeStatusFieldsFragment>>>;
};

export type LaForgeSubscribeUpdatedStatusSubscriptionVariables = Exact<{ [key: string]: never }>;

export type LaForgeSubscribeUpdatedStatusSubscription = { __typename?: 'Subscription' } & {
  updatedStatus: { __typename?: 'Status' } & LaForgeStatusFieldsFragment;
};

export type LaForgeSubscribeUpdatedAgentStatusSubscriptionVariables = Exact<{ [key: string]: never }>;

export type LaForgeSubscribeUpdatedAgentStatusSubscription = { __typename?: 'Subscription' } & {
  updatedAgentStatus: { __typename?: 'AgentStatus' } & LaForgeAgentStatusFieldsFragment;
};

export type LaForgeSubscribeUpdatedServerTaskSubscriptionVariables = Exact<{ [key: string]: never }>;

export type LaForgeSubscribeUpdatedServerTaskSubscription = { __typename?: 'Subscription' } & {
  updatedServerTask: { __typename?: 'ServerTask' } & LaForgeServerTaskFieldsFragment;
};

export type LaForgeSubscribeUpdatedBuildSubscriptionVariables = Exact<{ [key: string]: never }>;

export type LaForgeSubscribeUpdatedBuildSubscription = { __typename?: 'Subscription' } & {
  updatedBuild: { __typename?: 'Build' } & Pick<LaForgeBuild, 'id'> & {
      LatestBuildCommit?: Maybe<{ __typename?: 'BuildCommit' } & Pick<LaForgeBuildCommit, 'id'>>;
    };
};

export type LaForgeSubscribeUpdatedBuildCommitSubscriptionVariables = Exact<{ [key: string]: never }>;

export type LaForgeSubscribeUpdatedBuildCommitSubscription = { __typename?: 'Subscription' } & {
  updatedCommit: { __typename?: 'BuildCommit' } & LaForgeBuildCommitFieldsFragment;
};

export type LaForgeSubscribeUpdatedAgentTaskSubscriptionVariables = Exact<{ [key: string]: never }>;

export type LaForgeSubscribeUpdatedAgentTaskSubscription = { __typename?: 'Subscription' } & {
  updatedAgentTask: { __typename?: 'AgentTask' } & LaForgeAgentTaskFieldsFragment;
};

export type LaForgeServerTaskFieldsFragment = { __typename?: 'ServerTask' } & Pick<
  LaForgeServerTask,
  'id' | 'type' | 'startTime' | 'endTime' | 'errors' | 'logFilePath'
> & {
    Status: { __typename?: 'Status' } & LaForgeStatusFieldsFragment;
    Environment?: Maybe<{ __typename?: 'Environment' } & Pick<LaForgeEnvironment, 'id' | 'name'>>;
    Build?: Maybe<{ __typename?: 'Build' } & Pick<LaForgeBuild, 'id' | 'revision'>>;
  };

export type LaForgeGetCurrentUserTasksQueryVariables = Exact<{ [key: string]: never }>;

export type LaForgeGetCurrentUserTasksQuery = { __typename?: 'Query' } & {
  getCurrentUserTasks?: Maybe<Array<Maybe<{ __typename?: 'ServerTask' } & LaForgeServerTaskFieldsFragment>>>;
};

export type LaForgeGetServerTaskLogsQueryVariables = Exact<{
  taskUUID: Scalars['String'];
}>;

export type LaForgeGetServerTaskLogsQuery = { __typename?: 'Query' } & Pick<LaForgeQuery, 'viewServerTaskLogs'>;

export type LaForgeGetServerTasksQueryVariables = Exact<{
  taskUUIDs: Array<Maybe<Scalars['String']>> | Maybe<Scalars['String']>;
}>;

export type LaForgeGetServerTasksQuery = { __typename?: 'Query' } & {
  serverTasks?: Maybe<Array<Maybe<{ __typename?: 'ServerTask' } & LaForgeServerTaskFieldsFragment>>>;
};

export type LaForgeStreamServerTaskLogSubscriptionVariables = Exact<{
  taskUUID: Scalars['String'];
}>;

export type LaForgeStreamServerTaskLogSubscription = { __typename?: 'Subscription' } & Pick<LaForgeSubscription, 'streamServerTaskLog'>;

export const AgentStatusFieldsFragmentDoc = gql`
  fragment AgentStatusFields on AgentStatus {
    clientId
    hostname
    upTime
    bootTime
    numProcs
    OS
    hostID
    load1
    load5
    load15
    totalMem
    freeMem
    usedMem
    timestamp
  }
`;
export const StatusFieldsFragmentDoc = gql`
  fragment StatusFields on Status {
    id
    state
    startedAt
    endedAt
    failed
    completed
    error
  }
`;
export const PlanDiffFieldsFragmentDoc = gql`
  fragment PlanDiffFields on PlanDiff {
    id
    revision
    newState
  }
`;
export const PlanFieldsFragmentDoc = gql`
  fragment PlanFields on Plan {
    id
    stepNumber
    type
    Status {
      ...StatusFields
    }
    PlanDiffs {
      ...PlanDiffFields
    }
  }
  ${StatusFieldsFragmentDoc}
  ${PlanDiffFieldsFragmentDoc}
`;
export const BuildCommitFieldsFragmentDoc = gql`
  fragment BuildCommitFields on BuildCommit {
    id
    revision
    Build {
      id
      revision
      RepoCommit {
        id
        hash
        author
        Repository {
          id
          repoUrl
        }
      }
      Status {
        id
        state
      }
      Environment {
        id
      }
    }
    ServerTasks {
      id
      startTime
      endTime
    }
    state
    type
  }
`;
export const AuthUserFieldsFragmentDoc = gql`
  fragment AuthUserFields on AuthUser {
    id
    username
    role
    provider
    firstName
    lastName
    email
    phone
    company
    occupation
    publicKey
  }
`;
export const AgentTaskFieldsFragmentDoc = gql`
  fragment AgentTaskFields on AgentTask {
    id
    state
    command
    args
    number
    output
    errorMessage
  }
`;
export const PageInfoFieldsFragmentDoc = gql`
  fragment PageInfoFields on LaForgePageInfo {
    total
    nextOffset
  }
`;
export const UserListFieldsFragmentDoc = gql`
  fragment UserListFields on AuthUser {
    id
    firstName
    lastName
    username
    provider
    role
    email
    phone
    company
    occupation
  }
`;
export const ServerTaskFieldsFragmentDoc = gql`
  fragment ServerTaskFields on ServerTask {
    id
    type
    startTime
    endTime
    errors
    logFilePath
    Status {
      ...StatusFields
    }
    Environment {
      id
      name
    }
    Build {
      id
      revision
    }
  }
  ${StatusFieldsFragmentDoc}
`;
export const GetUserListDocument = gql`
  query GetUserList {
    getUserList {
      ...UserListFields
    }
  }
  ${UserListFieldsFragmentDoc}
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
export const UpdateUserDocument = gql`
  mutation UpdateUser(
    $userId: String!
    $firstName: String
    $lastName: String
    $email: String!
    $phone: String
    $company: String
    $occupation: String
    $role: RoleLevel!
    $provider: ProviderType!
  ) {
    modifyAdminUserInfo(
      userID: $userId
      firstName: $firstName
      lastName: $lastName
      email: $email
      phone: $phone
      company: $company
      occupation: $occupation
      role: $role
      provider: $provider
    ) {
      ...UserListFields
    }
  }
  ${UserListFieldsFragmentDoc}
`;

@Injectable({
  providedIn: 'root'
})
export class LaForgeUpdateUserGQL extends Apollo.Mutation<LaForgeUpdateUserMutation, LaForgeUpdateUserMutationVariables> {
  document = UpdateUserDocument;

  constructor(apollo: Apollo.Apollo) {
    super(apollo);
  }
}
export const CreateUserDocument = gql`
  mutation CreateUser($username: String!, $password: String!, $provider: ProviderType!, $role: RoleLevel!) {
    createUser(username: $username, password: $password, provider: $provider, role: $role) {
      ...UserListFields
    }
  }
  ${UserListFieldsFragmentDoc}
`;

@Injectable({
  providedIn: 'root'
})
export class LaForgeCreateUserGQL extends Apollo.Mutation<LaForgeCreateUserMutation, LaForgeCreateUserMutationVariables> {
  document = CreateUserDocument;

  constructor(apollo: Apollo.Apollo) {
    super(apollo);
  }
}
export const NukeBackendDocument = gql`
  mutation NukeBackend {
    nukeBackend {
      key
      value
    }
  }
`;

@Injectable({
  providedIn: 'root'
})
export class LaForgeNukeBackendGQL extends Apollo.Mutation<LaForgeNukeBackendMutation, LaForgeNukeBackendMutationVariables> {
  document = NukeBackendDocument;

  constructor(apollo: Apollo.Apollo) {
    super(apollo);
  }
}
export const GetAgentTasksDocument = gql`
  query GetAgentTasks($proStepId: String!) {
    getAgentTasks(proStepUUID: $proStepId) {
      ...AgentTaskFields
    }
  }
  ${AgentTaskFieldsFragmentDoc}
`;

@Injectable({
  providedIn: 'root'
})
export class LaForgeGetAgentTasksGQL extends Apollo.Query<LaForgeGetAgentTasksQuery, LaForgeGetAgentTasksQueryVariables> {
  document = GetAgentTasksDocument;

  constructor(apollo: Apollo.Apollo) {
    super(apollo);
  }
}
export const ListAgentStatusesDocument = gql`
  query ListAgentStatuses($buildUUID: String!) {
    listAgentStatuses(buildUUID: $buildUUID) {
      ...AgentStatusFields
    }
  }
  ${AgentStatusFieldsFragmentDoc}
`;

@Injectable({
  providedIn: 'root'
})
export class LaForgeListAgentStatusesGQL extends Apollo.Query<LaForgeListAgentStatusesQuery, LaForgeListAgentStatusesQueryVariables> {
  document = ListAgentStatusesDocument;

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
export const GetBuildTreeDocument = gql`
  query GetBuildTree($buildId: String!) {
    build(buildUUID: $buildId) {
      id
      revision
      Environment {
        id
        name
        description
        teamCount
        adminCidrs
        exposedVdiPorts
      }
      RepoCommit {
        id
        hash
        committer
        Repository {
          id
          repoUrl
        }
      }
      Status {
        id
      }
      Teams {
        id
        teamNumber
        Status {
          id
        }
        Plan {
          id
          Status {
            id
          }
        }
        ProvisionedNetworks {
          id
          name
          cidr
          Network {
            id
            vdiVisible
            vars {
              key
              value
            }
            tags {
              key
              value
            }
          }
          Status {
            id
          }
          Plan {
            id
            Status {
              id
            }
          }
          ProvisionedHosts {
            id
            subnetIp
            Host {
              id
              hostname
              description
              OS
              allowMacChanges
              exposedTcpPorts
              exposedUdpPorts
              userGroups
              overridePassword
              vars {
                key
                value
              }
              tags {
                key
                value
              }
            }
            Status {
              id
            }
            Plan {
              id
              Status {
                id
              }
            }
            ProvisioningSteps {
              id
              type
              stepNumber
              Script {
                id
                name
                language
                description
                source
                sourceType
                disabled
                args
                vars {
                  key
                  value
                }
                tags {
                  key
                  value
                }
              }
              Command {
                id
                name
                description
                program
                args
                disabled
                vars {
                  key
                  value
                }
                tags {
                  key
                  value
                }
              }
              DNSRecord {
                id
                name
                values
                type
                zone
                disabled
                vars {
                  key
                  value
                }
                tags {
                  key
                  value
                }
              }
              FileDownload {
                id
                source
                sourceType
                destination
                disabled
                tags {
                  key
                  value
                }
              }
              FileDelete {
                id
                path
                tags {
                  key
                  value
                }
              }
              FileExtract {
                id
                source
                destination
                type
                tags {
                  key
                  value
                }
              }
              Status {
                id
              }
              Plan {
                id
                Status {
                  id
                }
              }
            }
            AgentStatuses {
              clientId
            }
          }
        }
      }
      ServerTasks {
        id
      }
    }
  }
`;

@Injectable({
  providedIn: 'root'
})
export class LaForgeGetBuildTreeGQL extends Apollo.Query<LaForgeGetBuildTreeQuery, LaForgeGetBuildTreeQueryVariables> {
  document = GetBuildTreeDocument;

  constructor(apollo: Apollo.Apollo) {
    super(apollo);
  }
}
export const GetBuildPlansDocument = gql`
  query GetBuildPlans($buildId: String!) {
    build(buildUUID: $buildId) {
      id
      Plans {
        ...PlanFields
      }
    }
  }
  ${PlanFieldsFragmentDoc}
`;

@Injectable({
  providedIn: 'root'
})
export class LaForgeGetBuildPlansGQL extends Apollo.Query<LaForgeGetBuildPlansQuery, LaForgeGetBuildPlansQueryVariables> {
  document = GetBuildPlansDocument;

  constructor(apollo: Apollo.Apollo) {
    super(apollo);
  }
}
export const GetBuildStatusesDocument = gql`
  query GetBuildStatuses($buildUUID: String!) {
    build(buildUUID: $buildUUID) {
      id
      Plans {
        id
        Status {
          ...StatusFields
        }
      }
      Teams {
        id
        Status {
          ...StatusFields
        }
        ProvisionedNetworks {
          id
          Status {
            ...StatusFields
          }
          ProvisionedHosts {
            id
            Status {
              ...StatusFields
            }
          }
        }
      }
    }
  }
  ${StatusFieldsFragmentDoc}
`;

@Injectable({
  providedIn: 'root'
})
export class LaForgeGetBuildStatusesGQL extends Apollo.Query<LaForgeGetBuildStatusesQuery, LaForgeGetBuildStatusesQueryVariables> {
  document = GetBuildStatusesDocument;

  constructor(apollo: Apollo.Apollo) {
    super(apollo);
  }
}
export const GetBuildCommitsDocument = gql`
  query GetBuildCommits($buildId: String!) {
    build(buildUUID: $buildId) {
      id
      BuildCommits {
        ...BuildCommitFields
      }
    }
  }
  ${BuildCommitFieldsFragmentDoc}
`;

@Injectable({
  providedIn: 'root'
})
export class LaForgeGetBuildCommitsGQL extends Apollo.Query<LaForgeGetBuildCommitsQuery, LaForgeGetBuildCommitsQueryVariables> {
  document = GetBuildCommitsDocument;

  constructor(apollo: Apollo.Apollo) {
    super(apollo);
  }
}
export const GetPlanStatusCountsDocument = gql`
  query GetPlanStatusCounts($buildId: String!) {
    getPlanStatusCounts(buildUUID: $buildId) {
      planning
      awaiting
      parentAwaiting
      inProgress
      failed
      complete
      tainted
      toDelete
      deleteInProgress
      deleted
      toRebuild
      cancelled
    }
  }
`;

@Injectable({
  providedIn: 'root'
})
export class LaForgeGetPlanStatusCountsGQL extends Apollo.Query<LaForgeGetPlanStatusCountsQuery, LaForgeGetPlanStatusCountsQueryVariables> {
  document = GetPlanStatusCountsDocument;

  constructor(apollo: Apollo.Apollo) {
    super(apollo);
  }
}
export const ListBuildCommitsDocument = gql`
  query ListBuildCommits($envUUID: String!) {
    getBuildCommits(envUUID: $envUUID) {
      ...BuildCommitFields
    }
  }
  ${BuildCommitFieldsFragmentDoc}
`;

@Injectable({
  providedIn: 'root'
})
export class LaForgeListBuildCommitsGQL extends Apollo.Query<LaForgeListBuildCommitsQuery, LaForgeListBuildCommitsQueryVariables> {
  document = ListBuildCommitsDocument;

  constructor(apollo: Apollo.Apollo) {
    super(apollo);
  }
}
export const GetBuildCommitDocument = gql`
  query GetBuildCommit($buildCommitUUID: String!) {
    getBuildCommit(buildCommitUUID: $buildCommitUUID) {
      id
      revision
      state
      type
      Build {
        id
        revision
        RepoCommit {
          id
          hash
          author
          Repository {
            id
            repoUrl
          }
        }
        Environment {
          id
          name
        }
        Teams {
          id
          Plan {
            id
            Status {
              id
              state
            }
          }
          teamNumber
          ProvisionedNetworks {
            id
            Plan {
              id
              Status {
                id
                state
              }
            }
            name
            cidr
            Network {
              id
              vdiVisible
            }
            ProvisionedHosts {
              id
              Plan {
                id
                Status {
                  id
                  state
                }
              }
              subnetIp
              Host {
                id
                hostname
              }
              ProvisioningSteps {
                id
                stepNumber
                Plan {
                  id
                  Status {
                    id
                    state
                  }
                }
              }
            }
          }
        }
      }
      PlanDiffs {
        id
        newState
        Plan {
          id
        }
      }
    }
  }
`;

@Injectable({
  providedIn: 'root'
})
export class LaForgeGetBuildCommitGQL extends Apollo.Query<LaForgeGetBuildCommitQuery, LaForgeGetBuildCommitQueryVariables> {
  document = GetBuildCommitDocument;

  constructor(apollo: Apollo.Apollo) {
    super(apollo);
  }
}
export const GetEnvironmentDocument = gql`
  query GetEnvironment($envId: String!) {
    environment(envUUID: $envId) {
      id
      competitionId
      name
      description
      builder
      teamCount
      revision
      adminCidrs
      exposedVdiPorts
      tags {
        key
        value
      }
      config {
        key
        value
      }
      Users {
        id
        name
        uuid
        email
      }
      Repositories {
        id
        repoUrl
        branchName
      }
      Builds {
        id
        revision
        Status {
          ...StatusFields
        }
        Teams {
          id
          teamNumber
          Status {
            ...StatusFields
          }
          ProvisionedNetworks {
            id
            name
            cidr
            Status {
              ...StatusFields
            }
            Network {
              id
              vdiVisible
              vars {
                key
                value
              }
              tags {
                key
                value
              }
            }
            ProvisionedHosts {
              id
              subnetIp
              Status {
                ...StatusFields
              }
              Host {
                id
                hostname
                description
                OS
                allowMacChanges
                exposedTcpPorts
                exposedUdpPorts
                userGroups
                overridePassword
                vars {
                  key
                  value
                }
                tags {
                  key
                  value
                }
              }
              ProvisioningSteps {
                id
                type
                Status {
                  ...StatusFields
                }
                Script {
                  id
                  name
                  language
                  description
                  source
                  sourceType
                  disabled
                  args
                  vars {
                    key
                    value
                  }
                  tags {
                    key
                    value
                  }
                }
                Command {
                  id
                  name
                  description
                  program
                  args
                  disabled
                  vars {
                    key
                    value
                  }
                  tags {
                    key
                    value
                  }
                }
                DNSRecord {
                  id
                  name
                  values
                  type
                  zone
                  disabled
                  vars {
                    key
                    value
                  }
                  tags {
                    key
                    value
                  }
                }
                FileDownload {
                  id
                  source
                  sourceType
                  destination
                  disabled
                  tags {
                    key
                    value
                  }
                }
                FileDelete {
                  id
                  path
                  tags {
                    key
                    value
                  }
                }
                FileExtract {
                  id
                  source
                  destination
                  type
                  tags {
                    key
                    value
                  }
                }
              }
            }
          }
        }
      }
    }
  }
  ${StatusFieldsFragmentDoc}
`;

@Injectable({
  providedIn: 'root'
})
export class LaForgeGetEnvironmentGQL extends Apollo.Query<LaForgeGetEnvironmentQuery, LaForgeGetEnvironmentQueryVariables> {
  document = GetEnvironmentDocument;

  constructor(apollo: Apollo.Apollo) {
    super(apollo);
  }
}
export const GetEnvironmentsDocument = gql`
  query GetEnvironments {
    environments {
      id
      name
      competitionId
      revision
      Builds {
        id
        revision
      }
    }
  }
`;

@Injectable({
  providedIn: 'root'
})
export class LaForgeGetEnvironmentsGQL extends Apollo.Query<LaForgeGetEnvironmentsQuery, LaForgeGetEnvironmentsQueryVariables> {
  document = GetEnvironmentsDocument;

  constructor(apollo: Apollo.Apollo) {
    super(apollo);
  }
}
export const ListEnvironmentsDocument = gql`
  query ListEnvironments {
    environments {
      id
      name
      Repositories {
        id
        repoUrl
        branchName
        environmentFilepath
        RepoCommits {
          id
          revision
          author
          hash
        }
      }
      teamCount
      Networks {
        id
      }
      Hosts {
        id
      }
      ServerTasks {
        id
      }
    }
  }
`;

@Injectable({
  providedIn: 'root'
})
export class LaForgeListEnvironmentsGQL extends Apollo.Query<LaForgeListEnvironmentsQuery, LaForgeListEnvironmentsQueryVariables> {
  document = ListEnvironmentsDocument;

  constructor(apollo: Apollo.Apollo) {
    super(apollo);
  }
}
export const GetEnvironmentInfoDocument = gql`
  query GetEnvironmentInfo($envId: String!) {
    environment(envUUID: $envId) {
      id
      competitionId
      name
      description
      builder
      teamCount
      revision
      adminCidrs
      exposedVdiPorts
      tags {
        key
        value
      }
      config {
        key
        value
      }
      Users {
        id
        name
        uuid
        email
      }
      Builds {
        id
        revision
        Status {
          ...StatusFields
        }
      }
    }
  }
  ${StatusFieldsFragmentDoc}
`;

@Injectable({
  providedIn: 'root'
})
export class LaForgeGetEnvironmentInfoGQL extends Apollo.Query<LaForgeGetEnvironmentInfoQuery, LaForgeGetEnvironmentInfoQueryVariables> {
  document = GetEnvironmentInfoDocument;

  constructor(apollo: Apollo.Apollo) {
    super(apollo);
  }
}
export const RebuildDocument = gql`
  mutation Rebuild($rootPlans: [String]!) {
    rebuild(rootPlans: $rootPlans)
  }
`;

@Injectable({
  providedIn: 'root'
})
export class LaForgeRebuildGQL extends Apollo.Mutation<LaForgeRebuildMutation, LaForgeRebuildMutationVariables> {
  document = RebuildDocument;

  constructor(apollo: Apollo.Apollo) {
    super(apollo);
  }
}
export const DeleteBuildDocument = gql`
  mutation DeleteBuild($buildId: String!) {
    deleteBuild(buildUUID: $buildId)
  }
`;

@Injectable({
  providedIn: 'root'
})
export class LaForgeDeleteBuildGQL extends Apollo.Mutation<LaForgeDeleteBuildMutation, LaForgeDeleteBuildMutationVariables> {
  document = DeleteBuildDocument;

  constructor(apollo: Apollo.Apollo) {
    super(apollo);
  }
}
export const ExecuteBuildDocument = gql`
  mutation ExecuteBuild($buildId: String!) {
    executePlan(buildUUID: $buildId) {
      id
    }
  }
`;

@Injectable({
  providedIn: 'root'
})
export class LaForgeExecuteBuildGQL extends Apollo.Mutation<LaForgeExecuteBuildMutation, LaForgeExecuteBuildMutationVariables> {
  document = ExecuteBuildDocument;

  constructor(apollo: Apollo.Apollo) {
    super(apollo);
  }
}
export const CancelBuildDocument = gql`
  mutation CancelBuild($buildId: String!) {
    cancelBuild(buildUUID: $buildId)
  }
`;

@Injectable({
  providedIn: 'root'
})
export class LaForgeCancelBuildGQL extends Apollo.Mutation<LaForgeCancelBuildMutation, LaForgeCancelBuildMutationVariables> {
  document = CancelBuildDocument;

  constructor(apollo: Apollo.Apollo) {
    super(apollo);
  }
}
export const CreateBuildDocument = gql`
  mutation CreateBuild($envId: String!) {
    createBuild(envUUID: $envId, renderFiles: true) {
      id
    }
  }
`;

@Injectable({
  providedIn: 'root'
})
export class LaForgeCreateBuildGQL extends Apollo.Mutation<LaForgeCreateBuildMutation, LaForgeCreateBuildMutationVariables> {
  document = CreateBuildDocument;

  constructor(apollo: Apollo.Apollo) {
    super(apollo);
  }
}
export const ModifyCurrentUserDocument = gql`
  mutation ModifyCurrentUser($firstName: String, $lastName: String, $email: String, $phone: String, $company: String, $occupation: String) {
    modifySelfUserInfo(
      firstName: $firstName
      lastName: $lastName
      email: $email
      phone: $phone
      company: $company
      occupation: $occupation
    ) {
      ...AuthUserFields
    }
  }
  ${AuthUserFieldsFragmentDoc}
`;

@Injectable({
  providedIn: 'root'
})
export class LaForgeModifyCurrentUserGQL extends Apollo.Mutation<
  LaForgeModifyCurrentUserMutation,
  LaForgeModifyCurrentUserMutationVariables
> {
  document = ModifyCurrentUserDocument;

  constructor(apollo: Apollo.Apollo) {
    super(apollo);
  }
}
export const CreateEnvironmentFromGitDocument = gql`
  mutation CreateEnvironmentFromGit($repoURL: String!, $branchName: String!, $envFilePath: String!) {
    createEnviromentFromRepo(repoURL: $repoURL, branchName: $branchName, envFilePath: $envFilePath) {
      id
    }
  }
`;

@Injectable({
  providedIn: 'root'
})
export class LaForgeCreateEnvironmentFromGitGQL extends Apollo.Mutation<
  LaForgeCreateEnvironmentFromGitMutation,
  LaForgeCreateEnvironmentFromGitMutationVariables
> {
  document = CreateEnvironmentFromGitDocument;

  constructor(apollo: Apollo.Apollo) {
    super(apollo);
  }
}
export const UpdateEnvironmentViaPullDocument = gql`
  mutation UpdateEnvironmentViaPull($envId: String!) {
    updateEnviromentViaPull(envUUID: $envId) {
      id
    }
  }
`;

@Injectable({
  providedIn: 'root'
})
export class LaForgeUpdateEnvironmentViaPullGQL extends Apollo.Mutation<
  LaForgeUpdateEnvironmentViaPullMutation,
  LaForgeUpdateEnvironmentViaPullMutationVariables
> {
  document = UpdateEnvironmentViaPullDocument;

  constructor(apollo: Apollo.Apollo) {
    super(apollo);
  }
}
export const ApproveBuildCommitDocument = gql`
  mutation ApproveBuildCommit($buildCommitId: String!) {
    approveCommit(commitUUID: $buildCommitId)
  }
`;

@Injectable({
  providedIn: 'root'
})
export class LaForgeApproveBuildCommitGQL extends Apollo.Mutation<
  LaForgeApproveBuildCommitMutation,
  LaForgeApproveBuildCommitMutationVariables
> {
  document = ApproveBuildCommitDocument;

  constructor(apollo: Apollo.Apollo) {
    super(apollo);
  }
}
export const CancelBuildCommitDocument = gql`
  mutation CancelBuildCommit($buildCommitId: String!) {
    cancelCommit(commitUUID: $buildCommitId)
  }
`;

@Injectable({
  providedIn: 'root'
})
export class LaForgeCancelBuildCommitGQL extends Apollo.Mutation<
  LaForgeCancelBuildCommitMutation,
  LaForgeCancelBuildCommitMutationVariables
> {
  document = CancelBuildCommitDocument;

  constructor(apollo: Apollo.Apollo) {
    super(apollo);
  }
}
export const GetStatusDocument = gql`
  query GetStatus($statusId: String!) {
    status(statusUUID: $statusId) {
      ...StatusFields
    }
  }
  ${StatusFieldsFragmentDoc}
`;

@Injectable({
  providedIn: 'root'
})
export class LaForgeGetStatusGQL extends Apollo.Query<LaForgeGetStatusQuery, LaForgeGetStatusQueryVariables> {
  document = GetStatusDocument;

  constructor(apollo: Apollo.Apollo) {
    super(apollo);
  }
}
export const GetAgentStatusDocument = gql`
  query GetAgentStatus($clientId: String!) {
    agentStatus(clientId: $clientId) {
      ...AgentStatusFields
    }
  }
  ${AgentStatusFieldsFragmentDoc}
`;

@Injectable({
  providedIn: 'root'
})
export class LaForgeGetAgentStatusGQL extends Apollo.Query<LaForgeGetAgentStatusQuery, LaForgeGetAgentStatusQueryVariables> {
  document = GetAgentStatusDocument;

  constructor(apollo: Apollo.Apollo) {
    super(apollo);
  }
}
export const GetAllPlanStatusesDocument = gql`
  query GetAllPlanStatuses($buildId: String!, $count: Int!, $offset: Int!) {
    getAllPlanStatus(buildUUID: $buildId, count: $count, offset: $offset) {
      statuses {
        ...StatusFields
      }
      pageInfo {
        ...PageInfoFields
      }
    }
  }
  ${StatusFieldsFragmentDoc}
  ${PageInfoFieldsFragmentDoc}
`;

@Injectable({
  providedIn: 'root'
})
export class LaForgeGetAllPlanStatusesGQL extends Apollo.Query<LaForgeGetAllPlanStatusesQuery, LaForgeGetAllPlanStatusesQueryVariables> {
  document = GetAllPlanStatusesDocument;

  constructor(apollo: Apollo.Apollo) {
    super(apollo);
  }
}
export const GetAllAgentStatusesDocument = gql`
  query GetAllAgentStatuses($buildId: String!, $count: Int!, $offset: Int!) {
    getAllAgentStatus(buildUUID: $buildId, count: $count, offset: $offset) {
      agentStatuses {
        ...AgentStatusFields
      }
      pageInfo {
        ...PageInfoFields
      }
    }
  }
  ${AgentStatusFieldsFragmentDoc}
  ${PageInfoFieldsFragmentDoc}
`;

@Injectable({
  providedIn: 'root'
})
export class LaForgeGetAllAgentStatusesGQL extends Apollo.Query<LaForgeGetAllAgentStatusesQuery, LaForgeGetAllAgentStatusesQueryVariables> {
  document = GetAllAgentStatusesDocument;

  constructor(apollo: Apollo.Apollo) {
    super(apollo);
  }
}
export const ListBuildStatusesDocument = gql`
  query ListBuildStatuses($buildUUID: String!) {
    listBuildStatuses(buildUUID: $buildUUID) {
      ...StatusFields
    }
  }
  ${StatusFieldsFragmentDoc}
`;

@Injectable({
  providedIn: 'root'
})
export class LaForgeListBuildStatusesGQL extends Apollo.Query<LaForgeListBuildStatusesQuery, LaForgeListBuildStatusesQueryVariables> {
  document = ListBuildStatusesDocument;

  constructor(apollo: Apollo.Apollo) {
    super(apollo);
  }
}
export const SubscribeUpdatedStatusDocument = gql`
  subscription SubscribeUpdatedStatus {
    updatedStatus {
      ...StatusFields
    }
  }
  ${StatusFieldsFragmentDoc}
`;

@Injectable({
  providedIn: 'root'
})
export class LaForgeSubscribeUpdatedStatusGQL extends Apollo.Subscription<
  LaForgeSubscribeUpdatedStatusSubscription,
  LaForgeSubscribeUpdatedStatusSubscriptionVariables
> {
  document = SubscribeUpdatedStatusDocument;

  constructor(apollo: Apollo.Apollo) {
    super(apollo);
  }
}
export const SubscribeUpdatedAgentStatusDocument = gql`
  subscription SubscribeUpdatedAgentStatus {
    updatedAgentStatus {
      ...AgentStatusFields
    }
  }
  ${AgentStatusFieldsFragmentDoc}
`;

@Injectable({
  providedIn: 'root'
})
export class LaForgeSubscribeUpdatedAgentStatusGQL extends Apollo.Subscription<
  LaForgeSubscribeUpdatedAgentStatusSubscription,
  LaForgeSubscribeUpdatedAgentStatusSubscriptionVariables
> {
  document = SubscribeUpdatedAgentStatusDocument;

  constructor(apollo: Apollo.Apollo) {
    super(apollo);
  }
}
export const SubscribeUpdatedServerTaskDocument = gql`
  subscription SubscribeUpdatedServerTask {
    updatedServerTask {
      ...ServerTaskFields
    }
  }
  ${ServerTaskFieldsFragmentDoc}
`;

@Injectable({
  providedIn: 'root'
})
export class LaForgeSubscribeUpdatedServerTaskGQL extends Apollo.Subscription<
  LaForgeSubscribeUpdatedServerTaskSubscription,
  LaForgeSubscribeUpdatedServerTaskSubscriptionVariables
> {
  document = SubscribeUpdatedServerTaskDocument;

  constructor(apollo: Apollo.Apollo) {
    super(apollo);
  }
}
export const SubscribeUpdatedBuildDocument = gql`
  subscription SubscribeUpdatedBuild {
    updatedBuild {
      id
      LatestBuildCommit {
        id
      }
    }
  }
`;

@Injectable({
  providedIn: 'root'
})
export class LaForgeSubscribeUpdatedBuildGQL extends Apollo.Subscription<
  LaForgeSubscribeUpdatedBuildSubscription,
  LaForgeSubscribeUpdatedBuildSubscriptionVariables
> {
  document = SubscribeUpdatedBuildDocument;

  constructor(apollo: Apollo.Apollo) {
    super(apollo);
  }
}
export const SubscribeUpdatedBuildCommitDocument = gql`
  subscription SubscribeUpdatedBuildCommit {
    updatedCommit {
      ...BuildCommitFields
    }
  }
  ${BuildCommitFieldsFragmentDoc}
`;

@Injectable({
  providedIn: 'root'
})
export class LaForgeSubscribeUpdatedBuildCommitGQL extends Apollo.Subscription<
  LaForgeSubscribeUpdatedBuildCommitSubscription,
  LaForgeSubscribeUpdatedBuildCommitSubscriptionVariables
> {
  document = SubscribeUpdatedBuildCommitDocument;

  constructor(apollo: Apollo.Apollo) {
    super(apollo);
  }
}
export const SubscribeUpdatedAgentTaskDocument = gql`
  subscription SubscribeUpdatedAgentTask {
    updatedAgentTask {
      ...AgentTaskFields
    }
  }
  ${AgentTaskFieldsFragmentDoc}
`;

@Injectable({
  providedIn: 'root'
})
export class LaForgeSubscribeUpdatedAgentTaskGQL extends Apollo.Subscription<
  LaForgeSubscribeUpdatedAgentTaskSubscription,
  LaForgeSubscribeUpdatedAgentTaskSubscriptionVariables
> {
  document = SubscribeUpdatedAgentTaskDocument;

  constructor(apollo: Apollo.Apollo) {
    super(apollo);
  }
}
export const GetCurrentUserTasksDocument = gql`
  query GetCurrentUserTasks {
    getCurrentUserTasks {
      ...ServerTaskFields
    }
  }
  ${ServerTaskFieldsFragmentDoc}
`;

@Injectable({
  providedIn: 'root'
})
export class LaForgeGetCurrentUserTasksGQL extends Apollo.Query<LaForgeGetCurrentUserTasksQuery, LaForgeGetCurrentUserTasksQueryVariables> {
  document = GetCurrentUserTasksDocument;

  constructor(apollo: Apollo.Apollo) {
    super(apollo);
  }
}
export const GetServerTaskLogsDocument = gql`
  query GetServerTaskLogs($taskUUID: String!) {
    viewServerTaskLogs(taskID: $taskUUID)
  }
`;

@Injectable({
  providedIn: 'root'
})
export class LaForgeGetServerTaskLogsGQL extends Apollo.Query<LaForgeGetServerTaskLogsQuery, LaForgeGetServerTaskLogsQueryVariables> {
  document = GetServerTaskLogsDocument;

  constructor(apollo: Apollo.Apollo) {
    super(apollo);
  }
}
export const GetServerTasksDocument = gql`
  query GetServerTasks($taskUUIDs: [String]!) {
    serverTasks(taskUUIDs: $taskUUIDs) {
      ...ServerTaskFields
    }
  }
  ${ServerTaskFieldsFragmentDoc}
`;

@Injectable({
  providedIn: 'root'
})
export class LaForgeGetServerTasksGQL extends Apollo.Query<LaForgeGetServerTasksQuery, LaForgeGetServerTasksQueryVariables> {
  document = GetServerTasksDocument;

  constructor(apollo: Apollo.Apollo) {
    super(apollo);
  }
}
export const StreamServerTaskLogDocument = gql`
  subscription StreamServerTaskLog($taskUUID: String!) {
    streamServerTaskLog(taskID: $taskUUID)
  }
`;

@Injectable({
  providedIn: 'root'
})
export class LaForgeStreamServerTaskLogGQL extends Apollo.Subscription<
  LaForgeStreamServerTaskLogSubscription,
  LaForgeStreamServerTaskLogSubscriptionVariables
> {
  document = StreamServerTaskLogDocument;

  constructor(apollo: Apollo.Apollo) {
    super(apollo);
  }
}
