import { Component, OnInit, OnDestroy } from '@angular/core';
import { MatDialog } from '@angular/material/dialog';
import { MatSnackBar } from '@angular/material/snack-bar';
import { ActivatedRoute } from '@angular/router';
import { LaForgeGetBuildTreeQuery } from '@graphql';
import { ApiService } from '@services/api/api.service';
import { StatusService } from '@services/status/status.service';
import { BehaviorSubject, Subscription } from 'rxjs';
import { SubheaderService } from 'src/app/_metronic/partials/layout/subheader/_services/subheader.service';
import { EnvironmentService } from 'src/app/services/environment/environment.service';

import { ViewLogsModalComponent } from '@components/view-logs-modal/view-logs-modal.component';

@Component({
  selector: 'app-build',
  templateUrl: './build.component.html',
  styleUrls: ['./build.component.scss']
})
export class BuildComponent implements OnInit, OnDestroy {
  private unsubscribe: Subscription[] = [];
  private buildId: string;
  build: BehaviorSubject<LaForgeGetBuildTreeQuery['build']>;
  executeBuildLoading = false;
  planStatusesLoading = false;
  agentStatusesLoading = false;
  // viewTeams: BehaviorSubject<LaForgeGetBuildTreeQuery['build']['buildToTeam']>;

  constructor(
    private subheader: SubheaderService,
    public envService: EnvironmentService,
    private snackbar: MatSnackBar,
    private api: ApiService,
    private route: ActivatedRoute,
    private status: StatusService,
    private dialog: MatDialog
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

  viewBuildLogs() {
    const taskUUIDs = this.build.getValue().BuildToServerTasks.map((s) => s.id);
    this.dialog.open(ViewLogsModalComponent, {
      width: '75%',
      height: '90%',
      data: {
        taskUUIDs
      }
    });
  }
}
