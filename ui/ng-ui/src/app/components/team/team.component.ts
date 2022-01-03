import { Component, Input, OnInit, OnDestroy } from '@angular/core';
import {
  LaForgeProvisionStatus,
  LaForgeSubscribeUpdatedStatusSubscription,
  LaForgeTeam,
  LaForgePlanFieldsFragment,
  LaForgeGetBuildCommitQuery
} from '@graphql';
import { EnvironmentService } from '@services/environment/environment.service';
import { StatusService } from '@services/status/status.service';
import { BehaviorSubject } from 'rxjs';

import { RebuildService } from '../../services/rebuild/rebuild.service';

@Component({
  selector: 'app-team',
  templateUrl: './team.component.html',
  styleUrls: ['./team.component.scss']
})
export class TeamComponent implements OnInit, OnDestroy {
  @Input() title: string;
  // @Input() team: LaForgeGetBuildTreeQuery['build']['buildToTeam'][0];
  @Input() team: LaForgeGetBuildCommitQuery['getBuildCommit']['BuildCommitToBuild']['buildToTeam'][0];
  // @Input() planStatuses: LaForgeGetBuildCommitQuery['getBuildCommit']['BuildCommitToPlanDiffs'] | undefined;
  @Input() planDiffs: LaForgeGetBuildCommitQuery['getBuildCommit']['BuildCommitToPlanDiffs'] | undefined;
  // @Input() buildStatusMap: LaForgeSubscribeUpdatedStatusSubscription['updatedStatus'][] | undefined;
  // @Input() buildAgentStatusMap: LaForgeSubscribeUpdatedAgentStatusSubscription['updatedAgentStatus'][] | undefined;
  @Input() style: 'compact' | 'collapsed' | 'expanded';
  @Input() selectable: boolean;
  @Input() mode: 'plan' | 'build' | 'manage';
  isSelectedState = false;
  // planStatus: LaForgeSubscribeUpdatedStatusSubscription['updatedStatus'];
  expandOverride = false;
  shouldHideLoading = false;
  shouldHide: BehaviorSubject<boolean>;
  latestDiff: LaForgePlanFieldsFragment['PlanToPlanDiffs'][0];
  planStatus: BehaviorSubject<LaForgeSubscribeUpdatedStatusSubscription['updatedStatus']>;

  constructor(private rebuild: RebuildService, private envService: EnvironmentService, private status: StatusService) {
    if (!this.mode) this.mode = 'manage';
    if (!this.style) this.style = 'compact';
    if (!this.selectable) this.selectable = false;

    this.shouldHide = new BehaviorSubject(false);
  }

  ngOnInit(): void {
    if (this.mode === 'plan') {
      if (!this.getPlanDiff()) this.shouldHide.next(true);
    }
    this.planStatus = this.status.getStatusSubject(this.team.TeamToPlan.PlanToStatus.id);
  }

  ngOnDestroy() {}

  // checkPlanStatus(): void {
  // this.planStatus = this.envService.getStatus(this.team.TeamToPlan.PlanToStatus.id) || this.planStatus;
  // }

  // checkLatestPlanDiff(): void {
  //   if (this.latestDiff) return;
  //   const teamPlan = this.envService.getPlan(this.team.TeamToPlan.id);
  //   if (!teamPlan) return;
  //   this.latestDiff = [...teamPlan.PlanToPlanDiffs].sort((a, b) => b.revision - a.revision)[0];
  // }

  // getStatus(): ProvisionStatus {
  //   // let status: ProvisionStatus = ProvisionStatus.ProvStatusUndefined;
  //   let numWithAgentData = 0;
  //   let totalAgents = 0;
  //   for (const network of this.team.TeamToProvisionedNetwork) {
  //     for (const host of network.ProvisionedNetworkToProvisionedHost) {
  //       totalAgents++;
  //       if (host.ProvisionedHostToAgentStatus?.clientId) numWithAgentData++;
  //     }
  //   }
  //   if (numWithAgentData === totalAgents) return ProvisionStatus.COMPLETE;
  //   else if (numWithAgentData === 0) return ProvisionStatus.FAILED;
  //   else return ProvisionStatus.INPROGRESS;
  // }

  allChildrenResponding(): boolean {
    if (this.mode === 'plan') return true;
    // TODO: Add in build stuff
    // let numWithAgentData = 0;
    // let numWithCompletedSteps = 0;
    // let totalHosts = 0;
    // for (const pnet of this.team.TeamToProvisionedNetwork) {
    //   for (const host of pnet.ProvisionedNetworkToProvisionedHost) {
    //     totalHosts++;
    //     if (host.ProvisionedHostToAgentStatus?.clientId) numWithAgentData++;
    //     let totalSteps = 0;
    //     let totalCompletedSteps = 0;
    //     for (const step of host.ProvisionedHostToProvisioningStep) {
    //       if (step.step_number === 0) continue;
    //       totalSteps++;
    //       if (
    //         step.ProvisioningStepToStatus.id &&
    //         this.envService.getStatus(step.ProvisioningStepToPlan.PlanToStatus.id)?.state === LaForgeProvisionStatus.Complete
    //       )
    //         totalCompletedSteps++;
    //     }
    //     if (totalSteps === totalCompletedSteps) numWithCompletedSteps++;
    //   }
    // }
    // return numWithAgentData === totalHosts && numWithCompletedSteps === totalHosts;
  }

  getPlanDiff(): LaForgeGetBuildCommitQuery['getBuildCommit']['BuildCommitToPlanDiffs'][0] | undefined {
    return this.planDiffs?.filter((pd) => pd.PlanDiffToPlan.id === this.team.TeamToPlan.id)[0] ?? undefined;
  }

  getStatus(): LaForgeSubscribeUpdatedStatusSubscription['updatedStatus'] | undefined {
    // return this.buildStatusMap?.filter((s) => s.id === this.team.TeamToPlan.PlanToStatus.id)[0] ?? undefined;
    return this.planStatus.getValue();
  }

  getStatusIcon(): string {
    if (this.mode === 'plan') {
      const planDiff = this.getPlanDiff();
      if (!planDiff) return 'fas fa-spinner fa-spin';
      switch (planDiff.new_state) {
        case LaForgeProvisionStatus.Torebuild:
          return 'fas fa-sync-alt';
        case LaForgeProvisionStatus.Todelete:
          return 'fad fa-trash';
        case LaForgeProvisionStatus.Planning:
          return 'fas fa-ruler-triangle';
        default:
          return 'fal fa-users';
      }
    }
    const status = this.getStatus();
    if (!status) return 'fas fa-minus-circle';

    switch (status.state) {
      case LaForgeProvisionStatus.Planning:
        return 'fas fa-ruler-triangle';
      case LaForgeProvisionStatus.Todelete:
        return 'fas fa-recycle fas';
      case LaForgeProvisionStatus.Deleteinprogress:
        return 'fas fa-trash-restore';
      case LaForgeProvisionStatus.Deleted:
        return 'fas fa-trash fas';
      case LaForgeProvisionStatus.Failed:
        return 'fas fa-ban';
      default:
        return 'fal fa-users';
    }
  }

  getStatusColor(): string {
    if (this.mode === 'plan') {
      const planDiff = this.getPlanDiff();
      if (!planDiff) return 'dark';
      switch (planDiff.new_state) {
        case LaForgeProvisionStatus.Torebuild:
          return 'warning';
        case LaForgeProvisionStatus.Todelete:
          return 'danger';
        case LaForgeProvisionStatus.Planning:
          return 'primary';
        default:
          return 'dark';
      }
    }
    const status = this.getStatus();
    if (!status) return 'dark';

    switch (status.state) {
      case LaForgeProvisionStatus.Complete:
        if (this.allChildrenResponding()) {
          return 'success';
        } else {
          return 'warning';
        }
      case LaForgeProvisionStatus.Inprogress:
        return 'info';
      case LaForgeProvisionStatus.Tainted:
        return 'danger';
      case LaForgeProvisionStatus.Failed:
        return 'danger';
      case LaForgeProvisionStatus.Todelete:
        return 'primary';
      case LaForgeProvisionStatus.Deleteinprogress:
        return 'info';
      case LaForgeProvisionStatus.Planning:
        return 'primary';
      default:
        return 'dark';
    }
  }

  onSelect(): void {
    if (this.mode === 'plan') return;
    let success = false;
    if (!this.isSelected()) {
      success = this.rebuild.addTeam(this.team as LaForgeTeam);
    } else {
      success = this.rebuild.removeTeam(this.team as LaForgeTeam);
    }
    if (success) this.isSelectedState = !this.isSelectedState;
  }

  isSelected(): boolean {
    return this.rebuild.hasTeam(this.team as LaForgeTeam);
  }

  checkShouldHide() {
    if (this.mode === 'plan') {
      if (!this.latestDiff) return this.shouldHide.next(false);
      const latestCommit = this.envService.getLatestCommit();
      if (!latestCommit) return false;
      const teamPlan = this.envService.getPlan(this.team.TeamToPlan.id);
      if (teamPlan?.PlanToPlanDiffs.length > 0) {
        // expand if latest diff is a part of the latest commit
        if (latestCommit && latestCommit.BuildCommitToPlanDiffs.filter((diff) => diff.id === this.latestDiff.id).length > 0) {
          this.shouldHideLoading = false;
          this.shouldHide.next(false);
          return;
        }
      }
      this.shouldHideLoading = false;
      this.shouldHide.next(true);
      return;
    }
    this.shouldHide.next(false);
  }

  shouldCollapse(): boolean {
    if (this.mode === 'plan') {
      //   const latestCommit = this.envService.getBuildTree().getValue()?.BuildToLatestBuildCommit;
      //   const teamPlan = this.envService.getPlan(this.team.TeamToPlan.id);
      //   if (teamPlan?.PlanToPlanDiffs.length > 0) {
      //     const latestDiff = [...teamPlan.PlanToPlanDiffs].sort((a, b) => b.revision - a.revision)[0];
      //     // expand if latest diff is a part of the latest commit
      //     if (latestCommit && latestCommit.BuildCommitToPlanDiffs.filter((diff) => diff.id === latestDiff.id).length > 0) {
      //       // this.expandOverride = true;
      return false;
      //     }
      //   }
      //   // this.expandOverride = false;
      //   return true;
    }
    const status = this.getStatus();
    return (
      status &&
      (status.state === LaForgeProvisionStatus.Deleted ||
        (status.state === LaForgeProvisionStatus.Complete && this.allChildrenResponding()))
    );
  }

  canOverrideExpand(): boolean {
    const status = this.getStatus();
    return status && (status.state === LaForgeProvisionStatus.Complete || status.state === LaForgeProvisionStatus.Deleted);
  }

  toggleCollapse(): void {
    this.expandOverride = !this.expandOverride;
  }
}
