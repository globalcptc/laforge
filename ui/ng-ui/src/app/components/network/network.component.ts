import { Component, Input, OnInit, OnDestroy } from '@angular/core';
import { MatDialog } from '@angular/material/dialog';
import {
  LaForgeGetBuildCommitQuery,
  LaForgePlanFieldsFragment,
  LaForgeProvisionStatus,
  LaForgeSubscribeUpdatedStatusSubscription
} from '@graphql';
import { EnvironmentService } from '@services/environment/environment.service';
import { StatusService } from '@services/status/status.service';
import { BehaviorSubject, Subscription } from 'rxjs';
import { RebuildService } from 'src/app/services/rebuild/rebuild.service';

import { NetworkModalComponent } from '../network-modal/network-modal.component';

@Component({
  selector: 'app-network',
  templateUrl: './network.component.html',
  styleUrls: ['./network.component.scss']
})
export class NetworkComponent implements OnInit, OnDestroy {
  private unsubscribe: Subscription[] = [];
  // @Input() provisionedNetwork: LaForgeProvisionedNetwork;
  // @Input() status: Status;
  @Input()
  provisionedNetwork: LaForgeGetBuildCommitQuery['getBuildCommit']['BuildCommitToBuild']['buildToTeam'][0]['TeamToProvisionedNetwork'][0];
  @Input() planDiffs: LaForgeGetBuildCommitQuery['getBuildCommit']['BuildCommitToPlanDiffs'] | undefined;
  // @Input() buildStatusMap: LaForgeSubscribeUpdatedStatusSubscription['updatedStatus'][] | undefined;
  // @Input() buildAgentStatusMap: LaForgeSubscribeUpdatedAgentStatusSubscription['updatedAgentStatus'][] | undefined;
  @Input() style: 'compact' | 'collapsed' | 'expanded';
  @Input() selectable: boolean;
  @Input() parentSelected: boolean;
  @Input() mode: 'plan' | 'build' | 'manage';
  isSelectedState = false;
  // planStatus: LaForgeSubscribeUpdatedStatusSubscription['updatedStatus'];
  expandOverride = false;
  shouldHideLoading = false;
  shouldHide = false;
  latestDiff: LaForgePlanFieldsFragment['PlanToPlanDiffs'][0];
  planStatus: BehaviorSubject<LaForgeSubscribeUpdatedStatusSubscription['updatedStatus']>;

  constructor(
    public dialog: MatDialog,
    private rebuild: RebuildService,
    private envService: EnvironmentService,
    private status: StatusService
  ) {
    if (!this.mode) this.mode = 'manage';
    if (!this.style) this.style = 'compact';
    if (!this.selectable) this.selectable = false;
    if (!this.parentSelected) this.parentSelected = false;
  }

  ngOnInit(): void {
    this.planStatus = this.status.getStatusSubject(this.provisionedNetwork.ProvisionedNetworkToPlan.PlanToStatus.id);
  }

  ngOnDestroy() {}

  viewDetails(): void {
    this.dialog.open(NetworkModalComponent, {
      width: '50%',
      height: '80%',
      data: { provisionedNetwork: this.provisionedNetwork, planStatus: this.planStatus }
    });
  }

  allChildrenResponding(): boolean {
    if (this.mode === 'plan') return true;
    // TODO: rewrite build/manage page stuff
    // let numWithAgentData = 0;
    // let numWithCompletedSteps = 0;
    // let totalHosts = 0;
    // for (const host of this.provisionedNetwork.ProvisionedNetworkToProvisionedHost) {
    //   totalHosts++;
    //   if (host.ProvisionedHostToAgentStatus?.clientId) numWithAgentData++;
    //   let totalSteps = 0;
    //   let totalCompletedSteps = 0;
    //   for (const step of host.ProvisionedHostToProvisioningStep) {
    //     if (step.step_number === 0) continue;
    //     totalSteps++;
    //     if (
    //       step.ProvisioningStepToStatus.id &&
    //       this.envService.getStatus(step.ProvisioningStepToPlan.PlanToStatus.id)?.state === LaForgeProvisionStatus.Complete
    //     )
    //       totalCompletedSteps++;
    //   }
    //   if (totalSteps === totalCompletedSteps) numWithCompletedSteps++;
    // }
    // return numWithAgentData === totalHosts && numWithCompletedSteps === totalHosts;
  }

  getPlanDiff(): LaForgeGetBuildCommitQuery['getBuildCommit']['BuildCommitToPlanDiffs'][0] | undefined {
    return this.planDiffs?.filter((pd) => pd.PlanDiffToPlan.id === this.provisionedNetwork.ProvisionedNetworkToPlan.id)[0] ?? undefined;
  }

  getStatus(): LaForgeSubscribeUpdatedStatusSubscription['updatedStatus'] | undefined {
    // return this.buildStatusMap?.filter((s) => s.id === this.provisionedNetwork.ProvisionedNetworkToPlan.PlanToStatus.id)[0] ?? undefined;
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
          return 'fal fa-network-wired';
      }
    }
    const status = this.getStatus();
    if (!status) return 'fas fa-minus-circle';

    switch (status.state) {
      case LaForgeProvisionStatus.Planning:
        return 'fas fa-ruler-triangle';
      case LaForgeProvisionStatus.Todelete:
        return 'fas fa-recycle';
      case LaForgeProvisionStatus.Deleteinprogress:
        return 'fas fa-trash-restore';
      case LaForgeProvisionStatus.Deleted:
        return 'fas fa-trash';
      case LaForgeProvisionStatus.Failed:
        return 'fas fa-ban';
      default:
        return 'fal fa-network-wired';
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
    let success = false;
    if (!this.isSelected()) {
      success = this.rebuild.addNetwork(this.provisionedNetwork);
    } else {
      success = this.rebuild.removeNetwork(this.provisionedNetwork);
    }
    if (success) this.isSelectedState = !this.isSelectedState;
  }

  onIndeterminateChange(isIndeterminate: boolean): void {
    if (!isIndeterminate && this.isSelectedState) setTimeout(() => this.rebuild.addNetwork(this.provisionedNetwork), 500);
  }

  isSelected(): boolean {
    return this.rebuild.hasNetwork(this.provisionedNetwork);
  }

  checkShouldHide() {
    if (this.mode === 'plan') {
      if (!this.latestDiff) return (this.shouldHide = false);
      const latestCommit = this.envService.getLatestCommit();
      if (!latestCommit) return (this.shouldHide = false);
      const pnetPlan = this.envService.getPlan(this.provisionedNetwork.ProvisionedNetworkToPlan.id);
      if (pnetPlan?.PlanToPlanDiffs.length > 0) {
        // expand if latest diff is a part of the latest commit
        if (latestCommit && latestCommit.BuildCommitToPlanDiffs.filter((diff) => diff.id === this.latestDiff.id).length > 0) {
          this.shouldHideLoading = false;
          this.shouldHide = false;
          return;
        }
      }
      this.shouldHideLoading = false;
      this.shouldHide = true;
      return;
    }
    this.shouldHide = false;
  }

  shouldCollapse(): boolean {
    if (this.mode === 'plan') {
      // const pnetPlan = this.envService.getPlan(this.provisionedNetwork.ProvisionedNetworkToPlan.id);
      // const latestCommitRevision = this.envService.getBuildTree().getValue()?.BuildToLatestBuildCommit.revision;
      // if (pnetPlan?.PlanToPlanDiffs.length > 0) {
      //   const latestDiff = [...pnetPlan.PlanToPlanDiffs].sort((a, b) => b.revision - a.revision)[0];
      //   // collapse if latest diff isn't a part of the latest commit
      //   if (latestCommitRevision && latestCommitRevision === latestDiff.revision) {
      //     this.shouldCollapseLoading = false;
      return false;
      //   }
      // }
      // this.shouldCollapseLoading = false;
      // return true;
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
