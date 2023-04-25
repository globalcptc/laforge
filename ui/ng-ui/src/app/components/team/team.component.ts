import { Component, Input, OnInit, OnDestroy, ChangeDetectionStrategy, ChangeDetectorRef } from '@angular/core';
import {
  LaForgeProvisionStatus,
  LaForgeSubscribeUpdatedStatusSubscription,
  LaForgeTeam,
  LaForgePlanFieldsFragment,
  LaForgeGetBuildCommitQuery,
  LaForgeGetBuildTreeQuery
} from '@graphql';
import { EnvironmentService } from '@services/environment/environment.service';
import { StatusService } from '@services/status/status.service';
import { BehaviorSubject, Subscription } from 'rxjs';

import { RebuildService } from '../../services/rebuild/rebuild.service';

// eslint-disable-next-line max-len
type BuildCommitTeam = LaForgeGetBuildCommitQuery['getBuildCommit']['Build']['Teams'][0];
// eslint-disable-next-line max-len
type BuildTreeTeam = LaForgeGetBuildTreeQuery['build']['Teams'][0];
@Component({
  selector: 'app-team',
  templateUrl: './team.component.html',
  styleUrls: ['./team.component.scss'],
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class TeamComponent implements OnInit, OnDestroy {
  private unsubscribe: Subscription[] = [];
  @Input() title: string;
  // @Input() team: LaForgeGetBuildTreeQuery['build']['buildToTeam'][0];
  @Input() team: BuildCommitTeam | BuildTreeTeam;
  // @Input() planStatuses: LaForgeGetBuildCommitQuery['getBuildCommit']['PlanDiffs'] | undefined;
  @Input() planDiffs: LaForgeGetBuildCommitQuery['getBuildCommit']['PlanDiffs'] | undefined;
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
  latestDiff: LaForgePlanFieldsFragment['PlanDiffs'][0];
  planStatus: BehaviorSubject<LaForgeSubscribeUpdatedStatusSubscription['updatedStatus']>;
  provisionStatus: BehaviorSubject<LaForgeSubscribeUpdatedStatusSubscription['updatedStatus']>;

  constructor(
    private rebuild: RebuildService,
    private envService: EnvironmentService,
    private status: StatusService,
    private cdRef: ChangeDetectorRef
  ) {
    if (!this.mode) this.mode = 'manage';
    if (!this.style) this.style = 'compact';
    if (!this.selectable) this.selectable = false;

    this.shouldHide = new BehaviorSubject(false);
  }

  ngOnInit(): void {
    if (this.mode === 'plan') {
      if (!this.getPlanDiff()) this.shouldHide.next(true);
    }
    this.planStatus = this.status.getStatusSubject(this.team.Plan.Status.id);
    const sub1 = this.planStatus.subscribe(() => this.cdRef.markForCheck());
    this.unsubscribe.push(sub1);
    if (this.mode !== 'plan') {
      this.provisionStatus = this.status.getStatusSubject((this.team as BuildTreeTeam).Status.id);
      const sub = this.provisionStatus.subscribe(() => this.cdRef.markForCheck());
      this.unsubscribe.push(sub);
    }
  }

  ngOnDestroy() {
    this.unsubscribe.forEach((s) => s.unsubscribe());
  }

  allChildrenResponding(): boolean {
    if (this.mode === 'plan') return true;
    if (!this.planStatus.getValue() || !this.provisionStatus.getValue()) return false;
    return (
      this.planStatus.getValue().state === LaForgeProvisionStatus.Complete &&
      this.provisionStatus.getValue().state === LaForgeProvisionStatus.Complete
    );
  }

  getPlanDiff(): LaForgeGetBuildCommitQuery['getBuildCommit']['PlanDiffs'][0] | undefined {
    return this.planDiffs?.filter((pd) => pd.Plan.id === this.team.Plan.id)[0] ?? undefined;
  }

  getStatus(): LaForgeSubscribeUpdatedStatusSubscription['updatedStatus'] | undefined {
    // return this.buildStatusMap?.filter((s) => s.id === this.team.TeamToPlan.Status.id)[0] ?? undefined;
    return this.planStatus.getValue();
  }

  getStatusIcon(): string {
    if (this.mode === 'plan') {
      const planDiff = this.getPlanDiff();
      if (!planDiff) return 'fas fa-spinner fa-spin';
      switch (planDiff.newState) {
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
      switch (planDiff.newState) {
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

  shouldCollapse(): boolean {
    if (this.mode === 'plan') {
      //   const latestCommit = this.envService.getBuildTree().getValue()?.BuildToLatestBuildCommit;
      //   const teamPlan = this.envService.getPlan(this.team.TeamToPlan.id);
      //   if (teamPlan?.PlanDiffs.length > 0) {
      //     const latestDiff = [...teamPlan.PlanDiffs].sort((a, b) => b.revision - a.revision)[0];
      //     // expand if latest diff is a part of the latest commit
      //     if (latestCommit && latestCommit.PlanDiffs.filter((diff) => diff.id === latestDiff.id).length > 0) {
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
