import { Component, Input, OnInit, OnDestroy, ChangeDetectorRef, ChangeDetectionStrategy } from '@angular/core';
import { MatDialog } from '@angular/material/dialog';
import {
  LaForgeGetBuildTreeQuery,
  LaForgeProvisioningStepType,
  LaForgeSubscribeUpdatedStatusSubscription,
  LaForgeProvisionStatus,
  LaForgePlanFieldsFragment,
  LaForgeGetBuildCommitQuery,
  LaForgeProvisioningScheduledStepType
} from '@graphql';
import { StatusService } from '@services/status/status.service';
import { BehaviorSubject, Subscription } from 'rxjs';
import { ApiService } from 'src/app/services/api/api.service';
import { EnvironmentService } from 'src/app/services/environment/environment.service';

import { StepModalComponent } from '@components/step-modal/step-modal.component';

// eslint-disable-next-line max-len
type BuildTreeProvisioningStep = LaForgeGetBuildTreeQuery['build']['Teams'][0]['ProvisionedNetworks'][0]['ProvisionedHosts'][0]['ProvisioningSteps'][0];
// eslint-disable-next-line max-len
type BuildTreeProvisioningScheduledStep = LaForgeGetBuildTreeQuery['build']['Teams'][0]['ProvisionedNetworks'][0]['ProvisionedHosts'][0]['ProvisioningScheduledSteps'][0];

@Component({
  selector: 'app-step',
  templateUrl: './step.component.html',
  styleUrls: ['./step.component.scss'],
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class StepComponent implements OnInit, OnDestroy {
  private unsubscribe: Subscription[] = [];
  @Input() stepNumber: number;
  @Input() provisioningStep: BuildTreeProvisioningStep | null;
  @Input() provisioningScheduledStep: BuildTreeProvisioningScheduledStep | null;
  @Input() planDiffs: LaForgeGetBuildCommitQuery['getBuildCommit']['PlanDiffs'] | undefined;
  // @Input() buildStatusMap: LaForgeSubscribeUpdatedStatusSubscription['updatedStatus'][] | undefined;
  @Input() showDetail: boolean;
  @Input() style: 'compact' | 'expanded';
  @Input() mode: 'plan' | 'build' | 'manage';
  // planStatus: LaForgeSubscribeUpdatedStatusSubscription['updatedStatus'];
  provisioningStepStatus: LaForgeSubscribeUpdatedStatusSubscription['updatedStatus'];
  latestDiff: LaForgePlanFieldsFragment['PlanDiffs'][0];
  planStatus: BehaviorSubject<LaForgeSubscribeUpdatedStatusSubscription['updatedStatus']>;

  constructor(
    private api: ApiService,
    private cdRef: ChangeDetectorRef,
    private envService: EnvironmentService,
    private dialog: MatDialog,
    private status: StatusService
  ) {
    if (!this.mode) this.mode = 'manage';
  }

  ngOnInit() {
    const step: BuildTreeProvisioningStep | BuildTreeProvisioningScheduledStep = this.provisioningStep || this.provisioningScheduledStep;
    if (step.Plan?.Status?.id) {
      this.planStatus = this.status.getStatusSubject(step.Plan.Status.id);
      const sub = this.planStatus.subscribe(() => this.cdRef.markForCheck());
      this.unsubscribe.push(sub);
    }
    // const sub = this.envService.statusUpdate.asObservable().subscribe(() => {
    //   this.checkPlanStatus();
    //   this.checkprovisioningStepStatus();
    // });
    // this.unsubscribe.push(sub);
    // if (this.mode === 'plan') {
    //   const sub2 = this.envService.planUpdate.asObservable().subscribe(() => {
    //     this.checkLatestPlanDiff();
    //     this.cdRef.markForCheck();
    //   });
    //   this.unsubscribe.push(sub2);
    // }
  }

  ngOnDestroy() {
    this.unsubscribe.forEach((sub) => sub.unsubscribe());
  }

  viewDetails(): void {
    this.dialog.open(StepModalComponent, {
      width: '50%',
      height: '80%',
      data: {
        provisioningStep: this.provisioningStep,
        provisioningScheduledStep: this.provisioningScheduledStep,
        planStatus: this.planStatus
      }
    });
  }

  getPlanDiff(): LaForgeGetBuildCommitQuery['getBuildCommit']['PlanDiffs'][0] | undefined {
    const step: BuildTreeProvisioningStep | BuildTreeProvisioningScheduledStep = this.provisioningStep || this.provisioningScheduledStep;
    // UserData scripts have no plan associated
    if (!step.Plan) return undefined;
    return this.planDiffs?.filter((pd) => pd.Plan.id === step.Plan.id)[0] ?? undefined;
  }

  getStatus(): LaForgeSubscribeUpdatedStatusSubscription['updatedStatus'] | undefined {
    // return (
    //   this.buildStatusMap?.filter((s) => s.id === this.provisioningStep.ProvisioningStepToPlan?.Status.id ?? null)[0] ?? undefined
    // );
    return this.planStatus?.getValue() ?? undefined;
  }

  getStatusIcon(): string {
    if (this.provisioningStep) {
      switch (this.provisioningStep.type) {
        case LaForgeProvisioningStepType.Script:
          return 'file-code';
        case LaForgeProvisioningStepType.Command:
          return 'terminal';
        case LaForgeProvisioningStepType.DnsRecord:
          return 'globe';
        case LaForgeProvisioningStepType.FileDownload:
          return 'download';
        case LaForgeProvisioningStepType.FileDelete:
          return 'trash';
        case LaForgeProvisioningStepType.FileExtract:
          return 'file-archive';
        case LaForgeProvisioningStepType.Ansible:
          return 'archive';
      }
    } else if (this.provisioningScheduledStep) {
      switch (this.provisioningScheduledStep.type) {
        case LaForgeProvisioningScheduledStepType.Script:
          return 'file-code';
        case LaForgeProvisioningScheduledStepType.Command:
          return 'terminal';
        case LaForgeProvisioningScheduledStepType.DnsRecord:
          return 'globe';
        case LaForgeProvisioningScheduledStepType.FileDownload:
          return 'download';
        case LaForgeProvisioningScheduledStepType.FileDelete:
          return 'trash';
        case LaForgeProvisioningScheduledStepType.FileExtract:
          return 'file-archive';
        case LaForgeProvisioningScheduledStepType.Ansible:
          return 'archive';
      }
    }
    return 'minus-circle';
  }

  getStatusColor(): string {
    if (this.mode === 'plan') {
      const planDiff = this.getPlanDiff();
      if (!planDiff) return 'dark';
      switch (planDiff.newState) {
        case LaForgeProvisionStatus.Torebuild:
          return 'warning';
        case LaForgeProvisionStatus.Planning:
          return 'primary';
        default:
          return 'dark';
      }
    }
    const status = this.getStatus();
    if (!status) return 'black';

    switch (status.state) {
      case LaForgeProvisionStatus.Complete:
        return 'success';
      case LaForgeProvisionStatus.Todelete:
        return 'warning';
      case LaForgeProvisionStatus.Deleteinprogress:
        return 'warning';
      case LaForgeProvisionStatus.Deleted:
        return 'dark';
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

  getText(): string {
    const step: BuildTreeProvisioningStep | BuildTreeProvisioningScheduledStep = this.provisioningStep || this.provisioningScheduledStep;
    console.log(this.provisioningStep);
    console.log(this.provisioningScheduledStep);
    console.log(step);
    switch (step.type) {
      case LaForgeProvisioningStepType.Script || LaForgeProvisioningScheduledStepType.Script:
        return `${step.Script.source} ${step.Script.args.join(' ')}`;
      case LaForgeProvisioningStepType.Command:
        return `${step.Command.program} ${step.Command.args.join(' ')}`;
      case LaForgeProvisioningStepType.DnsRecord:
        return 'DNSRecord';
      case LaForgeProvisioningStepType.FileDownload:
        // eslint-disable-next-line max-len
        return `${step.FileDownload.source} -> ${step.FileDownload.destination}`;
      case LaForgeProvisioningStepType.FileDelete:
        return 'FileDelete';
      case LaForgeProvisioningStepType.FileExtract:
        return 'FileExtract';
      case LaForgeProvisioningStepType.Ansible:
        return 'Ansible';
      default:
        return 'Step';
    }
  }
}
