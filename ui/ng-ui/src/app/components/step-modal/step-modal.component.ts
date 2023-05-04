import { ChangeDetectorRef, Component, Inject, OnDestroy, OnInit } from '@angular/core';
import { MatDialogRef, MAT_DIALOG_DATA } from '@angular/material/dialog';
import {
  LaForgeGetAgentTasksQuery,
  LaForgeGetBuildTreeQuery,
  LaForgeProvisioningScheduledStep,
  LaForgeProvisioningStep,
  LaForgeProvisioningStepType,
  LaForgeProvisionStatus,
  LaForgeStatus,
  LaForgeSubscribeUpdatedStatusSubscription
} from '@graphql';
import { EnvironmentService } from '@services/environment/environment.service';
import { BehaviorSubject, Subscription } from 'rxjs';

import { LaForgeGetAgentTasksGQL } from '../../../generated/graphql';

// eslint-disable-next-line max-len
type BuildTreeProvisioningStep = LaForgeGetBuildTreeQuery['build']['Teams'][0]['ProvisionedNetworks'][0]['ProvisionedHosts'][0]['ProvisioningSteps'][0];
// eslint-disable-next-line max-len
type BuildTreeProvisioningScheduledStep = LaForgeGetBuildTreeQuery['build']['Teams'][0]['ProvisionedNetworks'][0]['ProvisionedHosts'][0]['ProvisioningScheduledSteps'][0];

@Component({
  selector: 'app-step-modal',
  templateUrl: './step-modal.component.html',
  styleUrls: ['./step-modal.component.scss']
})
export class StepModalComponent implements OnInit, OnDestroy {
  taskColumns: string[] = ['args', 'state'];
  failedChildren = false;
  agentTasks: BehaviorSubject<LaForgeGetAgentTasksQuery['getAgentTasks']>;
  subscription: Subscription;

  constructor(
    public dialogRef: MatDialogRef<StepModalComponent>,
    @Inject(MAT_DIALOG_DATA)
    public data: {
      provisioningStep: BuildTreeProvisioningStep | null;
      provisioningScheduledStep: BuildTreeProvisioningScheduledStep | null;
      planStatus: BehaviorSubject<LaForgeSubscribeUpdatedStatusSubscription['updatedStatus']>;
    },
    private getAgentTasks: LaForgeGetAgentTasksGQL,
    private cdRef: ChangeDetectorRef,
    private envService: EnvironmentService
  ) {
    this.agentTasks = new BehaviorSubject([]);
  }

  ngOnInit(): void {
    this.subscription = this.getAgentTasks
      .watch(
        {
          proStepId: this.data.provisioningStep?.id ?? undefined,
          proSchedStepId: this.data.provisioningScheduledStep?.id ?? undefined
        },
        {
          pollInterval: 5000
        }
      )
      .valueChanges.subscribe(({ data, error, errors }) => {
        if (error) {
          return this.agentTasks.error(error);
        } else if (errors) {
          return this.agentTasks.error(errors);
        }
        const tasks = [...data.getAgentTasks];
        for (let i = 0; i < tasks.length; i++) {
          const updatedTask = this.envService.getAgentTask(tasks[i].id);
          if (updatedTask) tasks[i] = { ...updatedTask };
        }
        this.agentTasks.next(tasks.sort((a, b) => a.number - b.number));
      }, this.agentTasks.error);
  }

  ngOnDestroy(): void {
    this.subscription.unsubscribe();
  }

  onClose(): void {
    this.dialogRef.close();
  }

  // getStatus(): ProvisionStatus {
  // let numWithAgentData = 0;
  // let totalAgents = 0;
  // for (const host of this.data.provisioningStep.ProvisionedHosts) {
  //   totalAgents++;
  //   if (host.AgentStatuses[0]?.clientId) numWithAgentData++;
  // }
  // if (numWithAgentData === totalAgents) {
  //   this.failedChildren = false;
  //   return ProvisionStatus.COMPLETE;
  // } else if (numWithAgentData === 0) {
  //   return ProvisionStatus.FAILED;
  // } else {
  //   this.failedChildren = true;
  //   return ProvisionStatus.INPROGRESS;
  // }
  // }

  getStatusText(): string {
    if (!this.data.planStatus) return 'Unknown';
    switch (this.data.planStatus.getValue().state) {
      case LaForgeProvisionStatus.Complete:
        return 'Complete';
      case LaForgeProvisionStatus.Failed:
        return 'Failed';
      case LaForgeProvisionStatus.Inprogress:
        return 'In Progress';
      case LaForgeProvisionStatus.Tainted:
        return 'Tainted';
      default:
        return 'Unknown';
    }
  }

  getStatusIcon(): string {
    if (!this.data.planStatus) return 'minus-circle';
    switch (this.data.planStatus.getValue().state) {
      case LaForgeProvisionStatus.Complete:
        return 'check-circle';
      case LaForgeProvisionStatus.Failed:
        return 'times-circle';
      case LaForgeProvisionStatus.Inprogress:
        return 'play-circle';
      case LaForgeProvisionStatus.Tainted:
        return 'skull';
      default:
        return 'minus-circle';
    }
  }

  getStatusColor(): string {
    if (!this.data.planStatus) return 'dark';
    switch (this.data.planStatus.getValue().state) {
      case LaForgeProvisionStatus.Complete:
        return 'success';
      case LaForgeProvisionStatus.Failed:
        return 'danger';
      case LaForgeProvisionStatus.Inprogress:
        return 'info';
      case LaForgeProvisionStatus.Tainted:
        return 'danger';
      default:
        return 'minus-circle';
    }
  }

  getText(): string {
    const step = this.data.provisioningStep || this.data.provisioningScheduledStep;
    switch (step.type) {
      case LaForgeProvisioningStepType.Script:
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
      default:
        return 'Step';
    }
  }
}
