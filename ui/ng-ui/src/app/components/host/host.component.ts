import { Component, Input, OnInit, OnDestroy, ChangeDetectorRef, ChangeDetectionStrategy } from '@angular/core';
import { MatDialog } from '@angular/material/dialog';
import {
  LaForgeGetBuildCommitQuery,
  LaForgeGetBuildTreeQuery,
  LaForgeProvisionedHost,
  LaForgeProvisionStatus,
  LaForgeSubscribeUpdatedAgentStatusSubscription,
  LaForgeSubscribeUpdatedStatusSubscription
} from '@graphql';
import { StatusService } from '@services/status/status.service';

import { BehaviorSubject, Subscription } from 'rxjs';

import { RebuildService } from '../../services/rebuild/rebuild.service';
import { HostModalComponent } from '../host-modal/host-modal.component';

// eslint-disable-next-line max-len
type BuildCommitProvisionedHost = LaForgeGetBuildCommitQuery['getBuildCommit']['BuildCommitToBuild']['buildToTeam'][0]['TeamToProvisionedNetwork'][0]['ProvisionedNetworkToProvisionedHost'][0];
// eslint-disable-next-line max-len
type BuildTreeProvisionedHost = LaForgeGetBuildTreeQuery['build']['buildToTeam'][0]['TeamToProvisionedNetwork'][0]['ProvisionedNetworkToProvisionedHost'][0];

@Component({
  selector: 'app-host',
  templateUrl: './host.component.html',
  styleUrls: ['./host.component.scss'],
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class HostComponent implements OnInit, OnDestroy {
  // @Input() status: Status;
  // @Input()
  // eslint-disable-next-line max-len
  // provisionedHost: LaForgeGetBuildTreeQuery['build']['buildToTeam'][0]['TeamToProvisionedNetwork'][0]['ProvisionedNetworkToProvisionedHost'][0];
  @Input()
  // eslint-disable-next-line max-len
  provisionedHost: BuildCommitProvisionedHost | BuildTreeProvisionedHost;
  @Input() planDiffs: LaForgeGetBuildCommitQuery['getBuildCommit']['BuildCommitToPlanDiffs'] | undefined;
  // @Input() buildStatusMap: LaForgeSubscribeUpdatedStatusSubscription['updatedStatus'][] | undefined;
  // @Input() buildAgentStatusMap: LaForgeSubscribeUpdatedAgentStatusSubscription['updatedAgentStatus'][] | undefined;
  @Input() style: 'compact' | 'collapsed' | 'expanded';
  @Input() selectable: boolean;
  @Input() parentSelected: boolean;
  @Input() hasAgent: boolean;
  @Input() mode: 'plan' | 'build' | 'manage';
  unsubscribe: Subscription[] = [];
  isSelectedState = false;
  expandOverride = false;
  shouldHideLoading = false;
  shouldHide = false;
  planStatus: BehaviorSubject<LaForgeSubscribeUpdatedStatusSubscription['updatedStatus']>;
  provisionStatus: BehaviorSubject<LaForgeSubscribeUpdatedStatusSubscription['updatedStatus']>;
  agentStatus: BehaviorSubject<LaForgeSubscribeUpdatedAgentStatusSubscription['updatedAgentStatus']>;

  constructor(public dialog: MatDialog, private rebuild: RebuildService, private status: StatusService, private cdRef: ChangeDetectorRef) {
    if (!this.mode) this.mode = 'manage';
    if (!this.style) this.style = 'compact';
    if (!this.selectable) this.selectable = false;
    if (!this.parentSelected) this.parentSelected = false;
    if (!this.hasAgent) this.hasAgent = false;
  }

  ngOnInit() {
    this.planStatus = this.status.getStatusSubject(this.provisionedHost.ProvisionedHostToPlan.PlanToStatus.id);
    const sub1 = this.planStatus.subscribe(() => this.cdRef.markForCheck());
    this.unsubscribe.push(sub1);
    if (this.mode !== 'plan') {
      this.provisionStatus = this.status.getStatusSubject((this.provisionedHost as BuildTreeProvisionedHost).ProvisionedHostToStatus.id);
      const sub = this.provisionStatus.subscribe(() => this.cdRef.markForCheck());
      this.unsubscribe.push(sub);
    }
    this.agentStatus = this.status.getAgentStatusSubject(this.provisionedHost.id);
    const sub2 = this.agentStatus.subscribe(() => this.cdRef.markForCheck());
    this.unsubscribe.push(sub2);
  }

  ngOnDestroy() {}

  viewDetails(): void {
    this.dialog.open(HostModalComponent, {
      width: '50%',
      height: '80%',
      data: {
        provisionedHost: this.provisionedHost,
        planStatus: this.getStatus(),
        agentStatus: this.getAgentStatus()
      }
    });
  }

  isAgentStale(): boolean {
    const agentStatus = this.getAgentStatus();
    if (!agentStatus) return true;
    return Date.now() / 1000 - agentStatus.timestamp > 120;
  }

  getPlanDiff(): LaForgeGetBuildCommitQuery['getBuildCommit']['BuildCommitToPlanDiffs'][0] | undefined {
    return this.planDiffs?.filter((pd) => pd.PlanDiffToPlan.id === this.provisionedHost.ProvisionedHostToPlan.id)[0] ?? undefined;
  }

  getStatus(): LaForgeSubscribeUpdatedStatusSubscription['updatedStatus'] | undefined {
    // return this.buildStatusMap?.filter((s) => s.id === this.provisionedHost.ProvisionedHostToPlan.PlanToStatus.id)[0] ?? undefined;
    return this.planStatus.getValue();
  }

  getAgentStatus(): LaForgeSubscribeUpdatedAgentStatusSubscription['updatedAgentStatus'] | undefined {
    // return this.buildAgentStatusMap?.filter((as) => as.clientId === this.provisionedHost.id)[0] ?? undefined;
    return this.agentStatus.getValue();
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
          return 'fas fa-computer-classic';
      }
    }
    const status = this.getStatus();
    if (!status) return 'fa fa-spinner fa-spin';

    if (status?.state) {
      switch (status.state) {
        case LaForgeProvisionStatus.Todelete:
          return 'fas fa-recycle';
        case LaForgeProvisionStatus.Deleteinprogress:
          return 'fas fa-trash-restore';
        case LaForgeProvisionStatus.Deleted:
          return 'fad fa-trash';
      }
    }
    if (this.getAgentStatus()) {
      if (this.isAgentStale()) return 'fas fa-exclamation-circle';
      if (this.childrenCompleted()) return 'fas fa-check-circle';
      else return 'fas fa-satellite-dish';
    } else {
      if (!status?.state) {
        return 'fas fa-minus-circle';
      }
      switch (status.state) {
        case LaForgeProvisionStatus.Complete:
          return 'fas fa-box-check';
        case LaForgeProvisionStatus.Failed:
          return 'fas fa-ban';
        case LaForgeProvisionStatus.Inprogress:
          return 'fas fa-play-circle';
        case LaForgeProvisionStatus.Awaiting:
          return 'fas fa-spinner fa-spin';
        case LaForgeProvisionStatus.Planning:
          return 'fas fa-ruler-triangle';
        default:
          return 'fas fa-computer-classic';
      }
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

    if (status?.state) {
      switch (status.state) {
        case LaForgeProvisionStatus.Todelete:
          return 'primary';
        case LaForgeProvisionStatus.Deleteinprogress:
          return 'info';
        case LaForgeProvisionStatus.Deleted:
          return 'dark';
      }
    }
    if (this.getAgentStatus()) {
      if (this.isAgentStale()) return 'warning';
      else return 'success';
    } else {
      if (!status?.state) {
        return 'minus-circle';
      }
      switch (status.state) {
        case LaForgeProvisionStatus.Complete:
          return 'success';
        case LaForgeProvisionStatus.Failed:
          return 'danger';
        case LaForgeProvisionStatus.Inprogress:
          return 'info';
        case LaForgeProvisionStatus.Planning:
          return 'primary';
        default:
          return 'dark';
      }
    }
  }

  onSelect(): void {
    let success = false;
    if (!this.isSelected()) {
      success = this.rebuild.addHost(this.provisionedHost as LaForgeProvisionedHost);
    } else {
      success = this.rebuild.removeHost(this.provisionedHost as LaForgeProvisionedHost);
    }
    // console.log(success);
    if (success) this.isSelectedState = !this.isSelectedState;
  }

  onIndeterminateChange(isIndeterminate: boolean): void {
    if (!isIndeterminate && this.isSelectedState)
      setTimeout(() => this.rebuild.addHost(this.provisionedHost as LaForgeProvisionedHost), 500);
  }

  isSelected(): boolean {
    return this.rebuild.hasHost(this.provisionedHost as LaForgeProvisionedHost);
  }

  getChildStatus(
    // eslint-disable-next-line max-len
    provisioningStep: LaForgeGetBuildCommitQuery['getBuildCommit']['BuildCommitToBuild']['buildToTeam'][0]['TeamToProvisionedNetwork'][0]['ProvisionedNetworkToProvisionedHost'][0]['ProvisionedHostToProvisioningStep'][0]
  ): LaForgeSubscribeUpdatedStatusSubscription['updatedStatus'] {
    // return this.buildStatusMap?.filter((s) => s.id === provisioningStep.ProvisioningStepToPlan.PlanToStatus.id)[0] ?? undefined;
    return undefined;
  }

  childrenCompleted(): boolean {
    if (this.mode === 'plan') return true;
    if (this.mode === 'manage') return true;
    if (
      this.planStatus.getValue().state === LaForgeProvisionStatus.Complete &&
      this.provisionStatus.getValue().state === LaForgeProvisionStatus.Complete
    )
      return true;
    else return false;
  }

  checkShouldHide() {
    // if (this.mode === 'plan') {
    //   const planDiff = this.getPlanDiff()
    //   if (!planDiff) return (this.shouldHide = false);
    //   const latestCommit = this.envService.getLatestCommit();
    //   if (!latestCommit) return (this.shouldHide = false);
    //   const phostPlan = this.envService.getPlan(this.provisionedHost.ProvisionedHostToPlan.id);
    //   if (phostPlan?.PlanToPlanDiffs.length > 0) {
    //     // expand if latest diff is a part of the latest commit
    //     if (latestCommit && latestCommit.BuildCommitToPlanDiffs.filter((diff) => diff.id === planDiff.id).length > 0) {
    //       this.shouldHideLoading = false;
    //       this.shouldHide = false;
    //       return;
    //     }
    //   }
    //   this.shouldHideLoading = false;
    //   this.shouldHide = true;
    //   return;
    // }
    this.shouldHide = false;
  }

  shouldCollapse(): boolean {
    if (this.mode === 'plan') {
      // const plan = this.envService.getPlan(this.provisionedHost.ProvisionedHostToPlan.id);
      // if (plan?.PlanToPlanDiffs.length > 0) {
      //   const latestCommitRevision = this.envService.getBuildTree().getValue()?.BuildToLatestBuildCommit.revision;
      //   const latestDiff = [...plan.PlanToPlanDiffs].sort((a, b) => b.revision - a.revision)[0];
      //   // collapse if latest diff isn't a part of the latest commit
      //   if (latestCommitRevision && latestCommitRevision != latestDiff.revision) return true;
      // return false;
      // }
      return true;
    }
    const status = this.getStatus();
    if (status && status.state === LaForgeProvisionStatus.Deleted) return true;
    if (status && status.state === LaForgeProvisionStatus.Awaiting) return true;
    if (status && status.state === LaForgeProvisionStatus.Parentawaiting) return true;
    // return hostChildrenCompleted(this.provisionedHost as LaForgeProvisionedHost, this.envService.getStatus);
    return this.childrenCompleted();
  }

  toggleCollapse(): void {
    this.expandOverride = !this.expandOverride;
  }
}
