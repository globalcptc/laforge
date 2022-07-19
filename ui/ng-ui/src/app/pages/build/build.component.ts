import { Component, OnInit, OnDestroy } from '@angular/core';
import { MatDialog } from '@angular/material/dialog';
import { MatSnackBar } from '@angular/material/snack-bar';
import { ActivatedRoute, Router } from '@angular/router';
import { LaForgeGetBuildTreeQuery, LaForgeGetPlanStatusCountsQuery, LaForgeProvisionStatus } from '@graphql';
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
  planStatusCounts: BehaviorSubject<LaForgeGetPlanStatusCountsQuery['getPlanStatusCounts']>;
  executeBuildLoading = false;
  planStatusesLoading = false;
  agentStatusesLoading = false;
  // viewTeams: BehaviorSubject<LaForgeGetBuildTreeQuery['build']['buildToTeam']>;
  statusCountInterval: NodeJS.Timeout;

  constructor(
    private subheader: SubheaderService,
    public envService: EnvironmentService,
    private snackbar: MatSnackBar,
    private api: ApiService,
    private route: ActivatedRoute,
    private router: Router,
    private status: StatusService,
    private dialog: MatDialog
  ) {
    this.subheader.setTitle('Build');
    this.subheader.setDescription('Monitor the progress of a given build');
    this.subheader.setShowEnvDropdown(false);

    this.build = new BehaviorSubject(null);
    this.planStatusCounts = new BehaviorSubject({
      planning: 0,
      awaiting: 0,
      parentAwaiting: 0,
      inProgress: 0,
      failed: 0,
      complete: 0,
      tainted: 0,
      toDelete: 0,
      deleteInProgress: 0,
      deleted: 0,
      toRebuild: 0,
      cancelled: 0
    });
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
      this.statusCountInterval = setInterval(
        () => this.api.getPlanStatusCountCache(this.buildId).then((counts) => this.planStatusCounts.next(counts)),
        2000
      );
    });
  }

  ngOnDestroy(): void {
    this.unsubscribe.forEach((sub) => sub.unsubscribe());
    if (this.statusCountInterval) clearInterval(this.statusCountInterval);
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

  buildIsCancellable(): boolean {
    const _build = this.build.getValue();
    if (!_build) return false;
    const _buildStatus = this.status.getStatusValue(_build.buildToStatus.id);
    if (!_buildStatus) return false;
    if (_buildStatus.state === LaForgeProvisionStatus.Deleted) return false;
    if (_buildStatus.state === LaForgeProvisionStatus.Deleteinprogress) return false;
    if (_buildStatus.state === LaForgeProvisionStatus.Todelete) return false;
    return true;
  }

  cancelBuild(buildId: string) {
    this.snackbar.open('Build is being cancelled...', null, {
      duration: 1000,
      panelClass: ['bg-info', 'text-white']
    });
    this.api.cancelBuild(buildId).then(
      () => {
        this.snackbar.open('Build cancelled successfully.', null, {
          duration: 1000,
          panelClass: ['bg-success', 'text-white']
        });
        setTimeout(() => this.router.navigate(['manage', this.build.getValue().id]), 1000);
      },
      (err) => {
        console.error(err);
        this.snackbar.open('Error while creating build. Please check logs for details.', 'Okay', {
          duration: 3000,
          panelClass: ['bg-danger', 'text-white']
        });
      }
    );
  }
}
