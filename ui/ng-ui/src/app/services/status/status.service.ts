import { Injectable, OnDestroy } from '@angular/core';
import {
  LaForgeSubscribeUpdatedAgentStatusGQL,
  LaForgeSubscribeUpdatedAgentStatusSubscription,
  LaForgeSubscribeUpdatedStatusGQL,
  LaForgeSubscribeUpdatedStatusSubscription
} from '@graphql';
import { ApiService } from '@services/api/api.service';
import { BehaviorSubject, Subscription } from 'rxjs';

@Injectable({
  providedIn: 'root'
})
export class StatusService implements OnDestroy {
  private unsubscribe: Subscription[];
  // Status
  private statusSubjectMap: { [key: string]: BehaviorSubject<LaForgeSubscribeUpdatedStatusSubscription['updatedStatus']> };
  // Agent Status
  private agentStatusSubjectMap: { [key: string]: BehaviorSubject<LaForgeSubscribeUpdatedAgentStatusSubscription['updatedAgentStatus']> };

  constructor(
    private api: ApiService,
    private updatedStatusGql: LaForgeSubscribeUpdatedStatusGQL,
    private updatedAgentStatusGql: LaForgeSubscribeUpdatedAgentStatusGQL
  ) {
    this.unsubscribe = [];
    this.statusSubjectMap = {};
    this.agentStatusSubjectMap = {};

    this.initStatusSubscription();
    this.initAgentStatusSubscription();
  }

  ngOnDestroy() {
    this.unsubscribe.forEach((s) => s.unsubscribe());
  }

  private initStatusSubscription() {
    const sub = this.updatedStatusGql.subscribe().subscribe(({ data: { updatedStatus }, errors }) => {
      if (errors) return;
      this.updateStatus(updatedStatus);
    });
    this.unsubscribe.push(sub);
  }

  private initAgentStatusSubscription() {
    const sub = this.updatedAgentStatusGql.subscribe().subscribe(({ data: { updatedAgentStatus }, errors }) => {
      if (errors) return;
      this.updateAgentStatus(updatedAgentStatus);
    });
    this.unsubscribe.push(sub);
  }

  private updateStatus(updatedStatus: LaForgeSubscribeUpdatedStatusSubscription['updatedStatus']) {
    if (!this.statusSubjectMap[updatedStatus.id]) this.statusSubjectMap[updatedStatus.id] = new BehaviorSubject(null);
    this.statusSubjectMap[updatedStatus.id].next(updatedStatus);
  }

  private updateAgentStatus(updatedAgentStatus: LaForgeSubscribeUpdatedAgentStatusSubscription['updatedAgentStatus']) {
    if (!this.agentStatusSubjectMap[updatedAgentStatus.clientId])
      this.agentStatusSubjectMap[updatedAgentStatus.clientId] = new BehaviorSubject(null);
    this.agentStatusSubjectMap[updatedAgentStatus.clientId].next(updatedAgentStatus);
  }

  public loadStatusCacheFromBuild(buildUUID: string) {
    this.api.listBuildStatuses(buildUUID).then((statuses) => {
      statuses.forEach((s) => this.updateStatus(s));
    });
  }

  public loadAgentStatusCacheFromBuild(buildUUID: string) {
    this.api.listBuildAgentStatuses(buildUUID).then((agentStatuses) => {
      agentStatuses.forEach((as) => this.updateAgentStatus(as));
    });
  }

  public getStatusSubject(statusId: string): BehaviorSubject<LaForgeSubscribeUpdatedStatusSubscription['updatedStatus']> {
    if (!this.statusSubjectMap[statusId]) this.statusSubjectMap[statusId] = new BehaviorSubject(null);
    return this.statusSubjectMap[statusId];
  }

  public getAgentStatusSubject(clientId: string): BehaviorSubject<LaForgeSubscribeUpdatedAgentStatusSubscription['updatedAgentStatus']> {
    if (!this.agentStatusSubjectMap[clientId]) this.agentStatusSubjectMap[clientId] = new BehaviorSubject(null);
    return this.agentStatusSubjectMap[clientId];
  }
}
