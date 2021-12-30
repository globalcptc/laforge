import { ChangeDetectorRef, Component, OnInit, OnDestroy } from '@angular/core';
import { MatSnackBar } from '@angular/material/snack-bar';
import { ActivatedRoute } from '@angular/router';
import { LaForgeGetBuildTreeQuery, LaForgeSubscribeUpdatedAgentStatusGQL } from '@graphql';
import { ApiService } from '@services/api/api.service';
import { StatusService } from '@services/status/status.service';
import { BehaviorSubject, Subscription } from 'rxjs';
import { SubheaderService } from 'src/app/_metronic/partials/layout/subheader/_services/subheader.service';
import { EnvironmentService } from 'src/app/services/environment/environment.service';

import { LaForgeExecuteBuildGQL } from '../../../generated/graphql';

@Component({
  selector: 'app-build',
  templateUrl: './build.component.html',
  styleUrls: ['./build.component.scss']
})
export class BuildComponent implements OnInit, OnDestroy {
  private unsubscribe: Subscription[] = [];
  private buildId: string;
  build: BehaviorSubject<LaForgeGetBuildTreeQuery['build']>;
  // statuses: BehaviorSubject<LaForgeSubscribeUpdatedStatusSubscription['updatedStatus'][]>;
  // agentStatuses: BehaviorSubject<LaForgeSubscribeUpdatedAgentStatusSubscription['updatedAgentStatus'][]>;
  executeBuildLoading = false;
  planStatusesLoading = false;
  agentStatusesLoading = false;
  // viewTeams: BehaviorSubject<LaForgeGetBuildTreeQuery['build']['buildToTeam']>;

  constructor(
    private subheader: SubheaderService,
    public envService: EnvironmentService,
    private cdRef: ChangeDetectorRef,
    private executeBuild: LaForgeExecuteBuildGQL,
    private snackbar: MatSnackBar,
    private api: ApiService,
    private route: ActivatedRoute,
    private updatedAgentStatusGql: LaForgeSubscribeUpdatedAgentStatusGQL,
    private status: StatusService
  ) {
    this.subheader.setTitle('Build');
    this.subheader.setDescription('Monitor the progress of a given build');
    this.subheader.setShowEnvDropdown(false);

    this.build = new BehaviorSubject(null);
    // this.statuses = new BehaviorSubject([]);
    // this.agentStatuses = new BehaviorSubject([]);
    // this.viewTeams = new BehaviorSubject([]);
  }

  ngOnInit(): void {
    this.route.params.subscribe((params) => {
      this.buildId = params.id;
      this.initBuildTree();
      this.status.loadStatusCacheFromBuild(this.buildId);
      this.status.loadAgentStatusCacheFromBuild(this.buildId);
      // this.initStatusMap();
      // this.initAgentStatusMap();

      // const sub1 = this.envService.getStatusSubscription().subscribe(({ data: { updatedStatus }, errors }) => {
      //   // console.log(`Updated Status ${updatedStatus.id}`);
      //   const currentStatuses = this.statuses.value;
      //   const statusIndex = currentStatuses.findIndex((s) => s.id === updatedStatus.id);
      //   if (statusIndex === -1) return this.statuses.next([...currentStatuses, updatedStatus]);
      //   const updatedStatuses = [...currentStatuses];
      //   updatedStatuses[statusIndex] = { ...updatedStatus };
      //   this.statuses.next(updatedStatuses);
      // });
      // this.unsubscribe.push(sub1);
      // const sub2 = this.envService.getAgentStatusSubscription().subscribe(({ data: { updatedAgentStatus }, errors }) => {
      //   // console.log(`Updated Agent Status ${updatedAgentStatus.clientId}`);
      //   const currentAgentStatuses = this.agentStatuses.value;
      //   const agentStatusIndex = currentAgentStatuses.findIndex((as) => as.clientId === updatedAgentStatus.clientId);
      //   if (agentStatusIndex === -1) return this.agentStatuses.next([...currentAgentStatuses, updatedAgentStatus]);
      //   const updatedAgentStatuses = [...currentAgentStatuses];
      //   updatedAgentStatuses[agentStatusIndex] = { ...updatedAgentStatus };
      //   this.agentStatuses.next(updatedAgentStatuses);
      // });
      // this.unsubscribe.push(sub2);
    });
  }

  ngOnDestroy(): void {
    this.unsubscribe.forEach((sub) => sub.unsubscribe());
  }

  checkBuildStatus(): void {}

  envIsSelected(): boolean {
    return this.envService.getEnvironmentInfo().getValue() != null;
  }

  initBuildTree(): void {
    this.api.getBuildTree(this.buildId).then((b) => {
      this.build.next(b);
      // this.viewTeams.next([b.buildToTeam[0]]);
    });
  }

  // initStatusMap(): void {
  //   this.api.listBuildStatuses(this.buildId).then((s) => {
  //     this.statuses.next(s);
  //   });
  // }

  // initAgentStatusMap(): void {
  //   this.api.listBuildAgentStatuses(this.buildId).then((as) => {
  //     this.agentStatuses.next(as);
  //   });
  // }
}
