import { Component, Inject, OnDestroy, OnInit } from '@angular/core';
import { MatDialogRef, MAT_DIALOG_DATA } from '@angular/material/dialog';
import { LaForgeGetProvisionedHostGQL, LaForgeGetProvisionedHostQuery } from '@graphql';

import { BehaviorSubject, Subscription } from 'rxjs';

@Component({
  selector: 'app-host-modal',
  templateUrl: './scheduled-steps-modal.component.html',
  styleUrls: ['./scheduled-steps-modal.component.scss']
})
class ScheduledStepsModalComponent implements OnInit, OnDestroy {
  subscription: Subscription;
  provisionedHost: BehaviorSubject<LaForgeGetProvisionedHostQuery['provisionedHost'] | null>;

  constructor(
    public dialogRef: MatDialogRef<ScheduledStepsModalComponent>,
    @Inject(MAT_DIALOG_DATA)
    public data: {
      provisionedHostId: string;
    },
    private getProvisionedHostGQL: LaForgeGetProvisionedHostGQL
  ) {
    this.provisionedHost = new BehaviorSubject(null);
  }

  ngOnInit(): void {
    this.subscription = this.getProvisionedHostGQL
      .watch({ id: this.data.provisionedHostId }, { pollInterval: 2000 })
      .valueChanges.subscribe(({ data, error, errors }) => {
        if (error) console.error(error);
        if (errors) errors.forEach(console.error);
        if (data.provisionedHost) this.provisionedHost.next(data.provisionedHost);
      });
  }

  ngOnDestroy(): void {
    this.subscription.unsubscribe();
  }

  onClose(): void {
    this.dialogRef.close();
  }

  isAgentStale(): boolean {
    if (!this.provisionedHost.getValue()?.AgentStatuses[0]) return true;
    return Date.now() / 1000 - this.provisionedHost.getValue()?.AgentStatuses[0].timestamp > 120;
  }

  getStatusIcon(): string {
    if (this.isAgentStale()) return 'exclamation-circle';
    else return 'check-circle';
  }

  getStatusColor(): string {
    if (this.isAgentStale()) return 'warning';
    else return 'success';
  }
}

export { ScheduledStepsModalComponent };
