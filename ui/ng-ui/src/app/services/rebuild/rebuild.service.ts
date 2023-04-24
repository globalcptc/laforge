import { Injectable } from '@angular/core';
import { LaForgeGetBuildCommitQuery, LaForgeProvisionedHost, LaForgeProvisionedNetwork, LaForgeRebuildGQL, LaForgeTeam } from '@graphql';

type ProvisionedNetwork =
  | LaForgeProvisionedNetwork
  | LaForgeGetBuildCommitQuery['getBuildCommit']['Build']['Teams'][0]['ProvisionedNetworks'][0];

@Injectable({
  providedIn: 'root'
})
export class RebuildService {
  rootPlans: string[];

  constructor(private rebuild: LaForgeRebuildGQL) {
    this.rootPlans = [];
  }

  /**
   * Executes the current rebuild based on the selected plans
   * @returns promise if the execution was a success (promise rejects with query errors)
   */
  executeRebuild = (): Promise<boolean> => {
    return new Promise<boolean>((resolve, reject) => {
      this.rebuild
        .mutate({
          rootPlans: this.rootPlans
        })
        .toPromise()
        .then(({ data, errors }) => {
          if (errors) {
            return reject(errors);
          }
          return resolve(data.rebuild);
        }, reject);
    });
  };

  addPlan = (planId: string): void => {
    console.log(`add plan: ${planId}`);
    if (this.rootPlans.filter((id: string) => id === planId).length === 0) this.rootPlans.push(planId);
  };

  removePlan = (planId: string): void => {
    console.log(`rem plan: ${planId}`);
    if (this.rootPlans.indexOf(planId) >= 0) this.rootPlans.splice(this.rootPlans.indexOf(planId), 1);
  };

  /**
   * addTeam selects teams to rebuild
   * @param team team to rebuild
   * @returns successfully added to list to rebuild
   */
  addTeam = (team: LaForgeTeam): boolean => {
    const planId = team.Plan?.id ?? null;
    if (planId === null) return false;
    this.addPlan(planId);
    return true;
  };

  /**
   * removeTeam removes a team from the rebuild list
   * @param team team to remove from rebuild list
   * @returns successfully removed from the list to rebuild
   */
  removeTeam = (team: LaForgeTeam): boolean => {
    const planId = team.Plan?.id ?? null;
    if (planId === null) return false;
    this.removePlan(planId);
    return true;
  };

  /**
   * hasTeam checks if a team is in the list to rebuild
   * @param team team to check
   * @returns if team is in rebuild list
   */
  hasTeam = (team: LaForgeTeam): boolean => {
    const planId = team.Plan?.id ?? null;
    if (planId === null) return false;
    return this.rootPlans.indexOf(planId) >= 0;
  };

  /**
   * addNetwork selects networks to rebuild
   * @param network network to rebuild
   * @returns successfully added to list to rebuild
   */
  addNetwork = (network: ProvisionedNetwork): boolean => {
    const planId = network.Plan?.id ?? null;
    if (planId === null) return false;
    this.addPlan(planId);
    return true;
  };

  /**
   * removeNetwork removes a network from the rebuild list
   * @param network network to remove from rebuild list
   * @returns successfully removed from the list to rebuild
   */
  removeNetwork = (network: ProvisionedNetwork): boolean => {
    const planId = network.Plan?.id ?? null;
    if (planId === null) return false;
    this.removePlan(planId);
    return true;
  };

  /**
   * hasNetwork checks if a network is in the list to rebuild
   * @param network network to check
   * @returns if network is in rebuild list
   */
  hasNetwork = (network: ProvisionedNetwork): boolean => {
    const planId = network.Plan?.id ?? null;
    if (planId === null) return false;
    return this.rootPlans.indexOf(planId) >= 0;
  };

  /**
   * addHost selects hosts to rebuild
   * @param host host to rebuild
   * @returns successfully added to list to rebuild
   */
  addHost = (host: LaForgeProvisionedHost): boolean => {
    const planId = host.Plan?.id ?? null;
    if (planId === null) return false;
    this.addPlan(planId);
    return true;
  };

  /**
   * removeHost removes a host from the rebuild list
   * @param host host to remove from rebuild list
   * @returns successfully removed from the list to rebuild
   */
  removeHost = (host: LaForgeProvisionedHost): boolean => {
    const planId = host.Plan?.id ?? null;
    if (planId === null) return false;
    this.removePlan(planId);
    return true;
  };

  /**
   * hasHost checks if a host is in the list to rebuild
   * @param host host to check
   * @returns if host is in rebuild list
   */
  hasHost = (host: LaForgeProvisionedHost): boolean => {
    const planId = host.Plan?.id ?? null;
    if (planId === null) return false;
    return this.rootPlans.indexOf(planId) >= 0;
  };
}
