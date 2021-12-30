import { Injectable } from '@angular/core';
import {
  LaForgeCreateBuildGQL,
  LaForgeCreateBuildMutation,
  LaForgeCreateEnvironmentFromGitGQL,
  LaForgeCreateEnvironmentFromGitMutation,
  LaForgeCreateUserGQL,
  LaForgeCreateUserMutation,
  LaForgeGetAllAgentStatusesGQL,
  LaForgeGetAllAgentStatusesQuery,
  LaForgeGetAllPlanStatusesGQL,
  LaForgeGetAllPlanStatusesQuery,
  LaForgeGetBuildCommitsGQL,
  LaForgeGetBuildCommitsQuery,
  LaForgeGetBuildPlansGQL,
  LaForgeGetBuildPlansQuery,
  LaForgeGetBuildTreeGQL,
  LaForgeGetBuildTreeQuery,
  LaForgeGetEnvironmentInfoGQL,
  LaForgeGetEnvironmentInfoQuery,
  LaForgeGetEnvironmentsGQL,
  LaForgeGetEnvironmentsQuery,
  LaForgeGetUserListGQL,
  LaForgeGetUserListQuery,
  LaForgeModifyCurrentUserGQL,
  LaForgeModifyCurrentUserMutation,
  LaForgeProviderType,
  LaForgeRoleLevel,
  LaForgeUpdateUserGQL,
  LaForgeUpdateUserMutation,
  LaForgeListEnvironmentsQuery,
  LaForgeListEnvironmentsGQL,
  LaForgeListBuildCommitsGQL,
  LaForgeListBuildCommitsQuery,
  LaForgeCancelBuildCommitGQL,
  LaForgeApproveBuildCommitGQL,
  LaForgeUpdateEnvironmentViaPullGQL,
  LaForgeUpdateEnvironmentViaPullMutation,
  LaForgeGetBuildCommitQuery,
  LaForgeGetBuildCommitGQL,
  LaForgeGetBuildStatusesGQL,
  LaForgeGetBuildStatusesQuery,
  LaForgeListAgentStatusesGQL,
  LaForgeListAgentStatusesQuery
} from '@graphql';

@Injectable({
  providedIn: 'root'
})
export class ApiService {
  constructor(
    private getEnvironments: LaForgeGetEnvironmentsGQL,
    // private pullPlanStatuses: LaForgePullPlanStatusesGQL,
    // private pullAgentStatuses: LaForgePullAgentStatusesGQL,
    private getAllAgentStatuses: LaForgeGetAllAgentStatusesGQL,
    private getAllPlanStatuses: LaForgeGetAllPlanStatusesGQL,
    private getEnvironmentInfoGQL: LaForgeGetEnvironmentInfoGQL,
    private getBuildTreeGQL: LaForgeGetBuildTreeGQL,
    private getBuildPlansGQL: LaForgeGetBuildPlansGQL,
    private getBuildCommitsGQL: LaForgeGetBuildCommitsGQL,
    private createBuildGQL: LaForgeCreateBuildGQL,
    private modifyCurrentUserGQL: LaForgeModifyCurrentUserGQL,
    private createEnvironmentFromGitGQL: LaForgeCreateEnvironmentFromGitGQL,
    private getUserListGQL: LaForgeGetUserListGQL,
    private updateUserGQL: LaForgeUpdateUserGQL,
    private createUserGQL: LaForgeCreateUserGQL,
    private updateEnvironmentViaPullGQL: LaForgeUpdateEnvironmentViaPullGQL,
    private listEnvironmentsGQL: LaForgeListEnvironmentsGQL,
    private listBuildCommitsGQL: LaForgeListBuildCommitsGQL,
    private cancelBuildCommitGQL: LaForgeCancelBuildCommitGQL,
    private approveBuildCommitGQL: LaForgeApproveBuildCommitGQL,
    private getBuildCommitGQL: LaForgeGetBuildCommitGQL,
    private getBuildStatuses: LaForgeGetBuildStatusesGQL,
    private listAgentStatuses: LaForgeListAgentStatusesGQL
  ) {}

  /**
   * Pulls status objects for all plans on a build
   * @param buildId The build ID that contains plans
   * @returns All plan objects relating to a build
   */
  public pullAllPlanStatuses(buildId: string, count: number, offset: number): Promise<LaForgeGetAllPlanStatusesQuery['getAllPlanStatus']> {
    return new Promise((resolve, reject) => {
      this.getAllPlanStatuses
        .fetch({
          buildId,
          count,
          offset
        })
        .toPromise()
        .then(({ data, error, errors }) => {
          if (error) {
            return reject(error);
          } else if (errors) {
            return reject(errors);
          }
          resolve(data.getAllPlanStatus);
        });
    });
  }

  /**
   * Pulls the build tree (with all branches only having ID's) and its contained agent statuses
   * @param buildId The build ID that agents relate to
   * @returns The build tree with only agents as full objects
   */
  public pullAllAgentStatuses(
    buildId: string,
    count: number,
    offset: number
  ): Promise<LaForgeGetAllAgentStatusesQuery['getAllAgentStatus']> {
    return new Promise((resolve, reject) => {
      this.getAllAgentStatuses
        .fetch({
          buildId,
          count,
          offset
        })
        .toPromise()
        .then(({ data, error, errors }) => {
          if (error) {
            return reject(error);
          } else if (errors) {
            return reject(errors);
          }
          resolve(data.getAllAgentStatus);
        });
    });
  }

  /**
   * Lists basic info about environments from the API once, without exposing a subscription or observable
   */
  public async listEnvironments(): Promise<LaForgeListEnvironmentsQuery['environments']> {
    return new Promise((resolve, reject) => {
      this.listEnvironmentsGQL
        .fetch()
        .toPromise()
        .then(({ data, error, errors }) => {
          if (error) {
            return reject(error);
          } else if (errors) {
            return reject(errors);
          }
          resolve(data.environments);
        });
    });
  }

  /**
   * Lists all build commits under an environment from the API once, without exposing a subscription or observable
   */
  public async listBuildCommits(envUUID: string): Promise<LaForgeListBuildCommitsQuery['getBuildCommits']> {
    return new Promise((resolve, reject) => {
      this.listBuildCommitsGQL
        .fetch({
          envUUID
        })
        .toPromise()
        .then(({ data, error, errors }) => {
          if (error) {
            return reject(error);
          } else if (errors) {
            return reject(errors);
          }
          resolve(data.getBuildCommits);
        }, reject);
    });
  }

  /**
   * Lists all statuses under an build from the API once, without exposing a subscription or observable
   */
  public async listBuildStatuses(buildUUID: string): Promise<LaForgeGetBuildStatusesQuery['build']['buildToPlan'][0]['PlanToStatus'][]> {
    return new Promise((resolve, reject) => {
      this.getBuildStatuses
        .fetch({
          buildUUID
        })
        .toPromise()
        .then(({ data, error, errors }) => {
          if (error) {
            return reject(error);
          } else if (errors) {
            return reject(errors);
          }
          resolve(data.build.buildToPlan.map((p) => p.PlanToStatus));
        }, reject);
    });
  }

  /**
   * Lists all statuses under an build from the API once, without exposing a subscription or observable
   */
  public async listBuildAgentStatuses(buildUUID: string): Promise<LaForgeListAgentStatusesQuery['listAgentStatuses']> {
    return new Promise((resolve, reject) => {
      this.listAgentStatuses
        .fetch({
          buildUUID
        })
        .toPromise()
        .then(({ data, error, errors }) => {
          if (error) {
            return reject(error);
          } else if (errors) {
            return reject(errors);
          }
          resolve(data.listAgentStatuses);
        }, reject);
    });
  }

  public async getBuildCommit(buildCommitUUID: string): Promise<LaForgeGetBuildCommitQuery['getBuildCommit']> {
    return new Promise((resolve, reject) => {
      this.getBuildCommitGQL
        .fetch({
          buildCommitUUID
        })
        .toPromise()
        .then(({ data, error, errors }) => {
          if (error) {
            return reject(error);
          } else if (errors) {
            return reject(errors);
          }
          resolve(data.getBuildCommit);
        }, reject);
    });
  }

  public async cancelBuildCommit(buildCommitId: string): Promise<boolean> {
    return new Promise((resolve, reject) => {
      this.cancelBuildCommitGQL
        .mutate({
          buildCommitId
        })
        .toPromise()
        .then(({ data, errors }) => {
          if (errors) {
            return reject(errors);
          }
          resolve(data.cancelCommit);
        });
    });
  }

  public async approveBuildCommit(buildCommitId: string): Promise<boolean> {
    return new Promise((resolve, reject) => {
      this.approveBuildCommitGQL
        .mutate({
          buildCommitId
        })
        .toPromise()
        .then(({ data, errors }) => {
          if (errors) {
            return reject(errors);
          }
          resolve(data.approveCommit);
        });
    });
  }

  /**
   * Pulls an environment from the API once, without exposing a subscription or observable
   * @param id The Environment ID of the environment
   */
  public async pullEnvironments(): Promise<LaForgeGetEnvironmentsQuery['environments']> {
    return new Promise((resolve, reject) => {
      this.getEnvironments
        .fetch()
        .toPromise()
        .then(({ data, error, errors }) => {
          if (error) {
            return reject(error);
          } else if (errors) {
            return reject(errors);
          }
          resolve(data.environments);
        });
    });
  }

  public async pullEnvironmentInfo(envId: string): Promise<LaForgeGetEnvironmentInfoQuery['environment']> {
    return new Promise((resolve, reject) => {
      this.getEnvironmentInfoGQL
        .fetch({
          envId: envId
        })
        .toPromise()
        .then(({ data, error, errors }) => {
          if (error) {
            return reject(error);
          } else if (errors) {
            return reject(errors);
          }
          resolve(data.environment);
        }, reject);
    });
  }

  public async getBuildTree(buildId: string): Promise<LaForgeGetBuildTreeQuery['build']> {
    return new Promise((resolve, reject) => {
      this.getBuildTreeGQL
        .fetch({
          buildId: buildId
        })
        .toPromise()
        .then(({ data, error, errors }) => {
          if (error) {
            return reject(error);
          } else if (errors) {
            return reject(errors);
          }
          resolve(data.build);
        }, reject);
    });
  }

  public async pullBuildPlans(buildId: string): Promise<LaForgeGetBuildPlansQuery['build']> {
    return new Promise((resolve, reject) => {
      this.getBuildPlansGQL
        .fetch({
          buildId
        })
        .toPromise()
        .then(({ data, errors, error }) => {
          if (error) {
            return reject(error);
          } else if (errors) {
            return reject(errors);
          }
          resolve(data.build);
        }, reject);
    });
  }

  public async pullBuildCommits(buildId: string): Promise<LaForgeGetBuildCommitsQuery['build']['BuildToBuildCommits']> {
    return new Promise((resolve, reject) => {
      this.getBuildCommitsGQL
        .fetch({
          buildId
        })
        .toPromise()
        .then(({ data, errors, error }) => {
          if (error) {
            return reject(error);
          } else if (errors) {
            return reject(errors);
          }
          resolve(data.build.BuildToBuildCommits);
        }, reject);
    });
  }

  public async createBuild(envId: string): Promise<LaForgeCreateBuildMutation['createBuild']> {
    return new Promise((resolve, reject) => {
      this.createBuildGQL
        .mutate({
          envId
        })
        .toPromise()
        .then(({ data, errors }) => {
          if (errors) {
            return reject(errors);
          } else if (data.createBuild) {
            return resolve(data.createBuild);
          }
          reject(new Error('unknown error occurred while creating build'));
        }, reject);
    });
  }

  public async updateEnvFromGit(envId: string): Promise<LaForgeUpdateEnvironmentViaPullMutation['updateEnviromentViaPull']> {
    return new Promise((resolve, reject) => {
      this.updateEnvironmentViaPullGQL
        .mutate({
          envId
        })
        .toPromise()
        .then(({ data, errors }) => {
          if (errors) {
            return reject(errors);
          } else if (data) {
            return resolve(data.updateEnviromentViaPull);
          }
          reject(new Error('unknown error occurred while updating enviroment'));
        }, reject);
    });
  }

  public async updateAuthUser(updateAuthUserInput: {
    firstName?: string;
    lastName?: string;
    email?: string;
    phone?: string;
    company?: string;
    occupation?: string;
  }): Promise<LaForgeModifyCurrentUserMutation['modifySelfUserInfo']> {
    return new Promise((resolve, reject) => {
      this.modifyCurrentUserGQL
        .mutate({
          ...updateAuthUserInput
        })
        .toPromise()
        .then(({ data, errors }) => {
          if (errors) {
            return reject(errors);
          } else if (data.modifySelfUserInfo) {
            return resolve(data.modifySelfUserInfo);
          }
          reject(new Error('unknown error occurred while updating current user'));
        }, reject);
    });
  }

  public async createEnvFromGit(createEnvFromGitInput: {
    repoURL: string;
    branchName: string;
    envFilePath: string;
  }): Promise<LaForgeCreateEnvironmentFromGitMutation['createEnviromentFromRepo']> {
    return new Promise((resolve, reject) => {
      this.createEnvironmentFromGitGQL
        .mutate({
          ...createEnvFromGitInput
        })
        .toPromise()
        .then(({ data, errors }) => {
          if (errors) {
            return reject(errors);
          } else if (data.createEnviromentFromRepo) {
            return resolve(data.createEnviromentFromRepo);
          }
          reject(new Error('unknown error occurred while cloning env from git'));
        }, reject);
    });
  }

  public async getAllUsers(): Promise<LaForgeGetUserListQuery['getUserList']> {
    return new Promise((resolve, reject) => {
      this.getUserListGQL
        .fetch()
        .toPromise()
        .then(({ data, errors }) => {
          if (errors) {
            return reject(errors);
          } else if (data.getUserList) {
            return resolve(data.getUserList);
          }
          reject(new Error('unknown error occurred while getting user list'));
        }, reject);
    });
  }

  public async modifyUser(
    userId: string,
    input: {
      firstName?: string;
      lastName?: string;
      email: string;
      phone?: string;
      company?: string;
      occupation?: string;
      role: LaForgeRoleLevel;
      provider: LaForgeProviderType;
    }
  ): Promise<LaForgeUpdateUserMutation['modifyAdminUserInfo']> {
    return new Promise((resolve, reject) => {
      this.updateUserGQL
        .mutate({
          userId,
          ...input
        })
        .toPromise()
        .then(({ data, errors }) => {
          if (errors) {
            return reject(errors);
          } else if (data.modifyAdminUserInfo) {
            return resolve(data.modifyAdminUserInfo);
          }
          reject(new Error('unknown error occurred while updating user'));
        }, reject);
    });
  }

  public async createUser(input: {
    username: string;
    password: string;
    role: LaForgeRoleLevel;
    provider: LaForgeProviderType;
  }): Promise<LaForgeCreateUserMutation['createUser']> {
    return new Promise((resolve, reject) => {
      this.createUserGQL
        .mutate({
          ...input
        })
        .toPromise()
        .then(({ data, errors }) => {
          if (errors) {
            return reject(errors);
          } else if (data.createUser) {
            return resolve(data.createUser);
          }
          reject(new Error('unknown error occurred while creating user'));
        }, reject);
    });
  }
}
