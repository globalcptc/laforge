import { ChangeDetectorRef, Component, OnDestroy, OnInit } from '@angular/core';
import { MatDialog } from '@angular/material/dialog';
import { MatSnackBar } from '@angular/material/snack-bar';

import {
  LaForgeBuildCommitState,
  LaForgeBuildCommitType,
  LaForgeListBuildCommitsQuery,
  LaForgeListEnvironmentsQuery,
  LaForgeProvisionStatus,
  LaForgeSubscribeUpdatedBuildCommitGQL
} from '@graphql';
import { ApiService } from '@services/api/api.service';
import { EnvironmentService } from '@services/environment/environment.service';
import { BehaviorSubject, Observable, Subscription } from 'rxjs';
import { SubheaderService } from 'src/app/_metronic/partials/layout/subheader/_services/subheader.service';

import { DeleteBuildModalComponent } from '@components/delete-build-modal/delete-build-modal.component';
import { ImportRepoModalComponent } from '@components/import-repo-modal/import-repo-modal.component';
import { ViewLogsModalComponent } from '@components/view-logs-modal/view-logs-modal.component';

@Component({
  selector: 'app-dashboard',
  templateUrl: './home.component.html',
  styleUrls: ['./home.component.scss']
})
export class HomeComponent implements OnInit, OnDestroy {
  private unsubscribe: Subscription[] = [];
  getEnvironmentsLoading: Observable<boolean>;
  // environmentsCols: string[] = ['name', 'competition_id', 'build_count', 'revision', 'pull-actions', 'actions'];
  // environments: Observable<LaForgeGetEnvironmentsQuery['environments']>;
  environments: Promise<LaForgeListEnvironmentsQuery['environments']>;
  expandedEnvironments: { [key: string]: boolean };
  buildCommits: BehaviorSubject<{
    [key: string]: LaForgeListBuildCommitsQuery['getBuildCommits'];
  }>;
  showIconsOnly = false;

  constructor(
    private cdRef: ChangeDetectorRef,
    private subheader: SubheaderService,
    public envService: EnvironmentService,
    private api: ApiService,
    private snackbar: MatSnackBar,
    private dialog: MatDialog,
    private updatedBuildCommitGQL: LaForgeSubscribeUpdatedBuildCommitGQL
  ) {
    this.subheader.setTitle('Home');
    this.subheader.setDescription('Overview of all environments and builds');
    this.subheader.setShowEnvDropdown(false);

    this.getEnvironmentsLoading = this.envService.envIsLoading.asObservable();
    this.expandedEnvironments = {};
    this.buildCommits = new BehaviorSubject({});
    // this.buildCommits = {};
    // this.environments = this.envService.getEnvironments().asObservable();
  }

  ngOnInit(): void {
    // this.envService.getEnvironments().subscribe(() => {
    //   this.cdRef.markForCheck();
    // });
    this.environments = this.api.listEnvironments();
    const buildCommitSubscription = this.updatedBuildCommitGQL.subscribe().subscribe(({ data: { updatedCommit }, errors }) => {
      if (errors) {
        this.snackbar.open('Error updating build commit list. See console for more info.', null, {
          duration: 3000,
          panelClass: ['bg-danger', 'text-white']
        });
        return console.error(errors);
      }
      const buildCommitMap = { ...this.buildCommits.getValue() };
      if (buildCommitMap[updatedCommit.BuildCommitToBuild.buildToEnvironment.id]) {
        const envBuildCommits = [...buildCommitMap[updatedCommit.BuildCommitToBuild.buildToEnvironment.id]];
        const index = envBuildCommits.findIndex((bc) => bc.id === updatedCommit.id);
        if (index === -1) {
          envBuildCommits.push(updatedCommit);
          this.snackbar.open('Build commit created', null, {
            duration: 1000,
            panelClass: ['bg-success', 'text-white']
          });
        } else envBuildCommits.splice(index, 1, updatedCommit);
        buildCommitMap[updatedCommit.BuildCommitToBuild.buildToEnvironment.id] = envBuildCommits;
      } else {
        buildCommitMap[updatedCommit.BuildCommitToBuild.buildToEnvironment.id] = [updatedCommit];
      }
      this.buildCommits.next(buildCommitMap);
    });
    this.unsubscribe.push(buildCommitSubscription);
  }

  ngOnDestroy(): void {
    this.unsubscribe.forEach((s) => s.unsubscribe());
  }

  getLatestRepoCommit(environment: LaForgeListEnvironmentsQuery['environments'][0]) {
    return [...environment.EnvironmentToRepository[0].RepositoryToRepoCommit].sort((a, b) => b.revision - a.revision)[0] || undefined;
  }

  getLatestCommitHash(environment: LaForgeListEnvironmentsQuery['environments'][0]): string {
    return this.getLatestRepoCommit(environment)?.hash.substring(0, 7) ?? 'Unknown';
  }

  getLatestCommitLink(environment: LaForgeListEnvironmentsQuery['environments'][0]): string {
    const commitHash = this.getLatestRepoCommit(environment)?.hash ?? null;
    if (!commitHash) return '#';
    const repoUrl = environment.EnvironmentToRepository[0].repo_url;
    const provider = repoUrl.split(':')[0].split('@')[1];
    const repoPath = repoUrl.split(':')[1].replace('.git', '');
    switch (provider) {
      case 'github.com':
        return `https://github.com/${repoPath}/commit/${commitHash}`;
      default:
        return '#';
    }
  }

  getLastCommitter(environment: LaForgeListEnvironmentsQuery['environments'][0]): string {
    return this.getLatestRepoCommit(environment)?.author.replace(/ <.*>/g, '') ?? 'Unknown';
  }

  async toggleExpandEnvironment(environment: LaForgeListEnvironmentsQuery['environments'][0]): Promise<void> {
    if (this.expandedEnvironments[environment.id]) delete this.expandedEnvironments[environment.id];
    else {
      this.expandedEnvironments[environment.id] = true;
      await this.pullBuildCommits(environment);
      this.cdRef.detectChanges();
    }
    const buildsElement = document.getElementById(`builds-${environment.id}`);
    buildsElement.classList.toggle('collapsed');
    buildsElement.classList.toggle('expanded');
    if (buildsElement.style.maxHeight !== '0px') buildsElement.style.maxHeight = '0px';
    else buildsElement.style.maxHeight = buildsElement.scrollHeight + 'px';
  }

  environmentIsExpanded(environment: LaForgeListEnvironmentsQuery['environments'][0]): boolean {
    return this.expandedEnvironments[environment.id];
  }

  pullBuildCommits(environment: LaForgeListEnvironmentsQuery['environments'][0]): Promise<void> {
    return this.api.listBuildCommits(environment.id).then(
      (buildCommits) => {
        const bc = this.buildCommits.value;
        bc[environment.id] = [...buildCommits].sort((a, b) => {
          const start1 = a.BuildCommitToServerTask ? a.BuildCommitToServerTask[0]?.start_time ?? 0 : 0;
          const start2 = b.BuildCommitToServerTask ? b.BuildCommitToServerTask[0]?.start_time ?? 0 : 0;
          return new Date(start2).getTime() - new Date(start1).getTime();
        });
        this.buildCommits.next(bc);
      },
      (err) => {
        console.error(err);
        this.snackbar.open('Error while pulling build commits. See console/logs for details.', 'Okay', {
          panelClass: ['bg-danger', 'text-white']
        });
      }
    );
  }

  groupBuildCommits(buildCommits: LaForgeListBuildCommitsQuery['getBuildCommits']): LaForgeListBuildCommitsQuery['getBuildCommits'][] {
    if (!buildCommits) return [];
    const buildToBuildCommits: { [key: number]: LaForgeListBuildCommitsQuery['getBuildCommits'] } = {};
    for (const buildCommit of buildCommits) {
      if (!buildToBuildCommits[buildCommit.BuildCommitToBuild.revision])
        buildToBuildCommits[buildCommit.BuildCommitToBuild.revision] = [buildCommit];
      else buildToBuildCommits[buildCommit.BuildCommitToBuild.revision].push(buildCommit);
    }
    const groupedBuildCommits: LaForgeListBuildCommitsQuery['getBuildCommits'][] = [];
    for (const revision of Object.keys(buildToBuildCommits)
      .map((key) => parseInt(key))
      .sort((a, b) => b - a)) {
      groupedBuildCommits.push(buildToBuildCommits[revision]);
    }
    return groupedBuildCommits;
  }

  getBuildStateColor(state: string): string {
    switch (state) {
      case LaForgeProvisionStatus.Complete:
        return 'success';
      // case LaForgeProvisionStatus.Cancelled:
      //   return 'danger';
      // case LaForgeProvisionStatus.Failed:
      //   return 'danger';
      // case LaForgeProvisionStatus.Inprogress:
      //   return 'info';
      // case LaForgeProvisionStatus.Planning:
      //   return 'primary';
      // case LaForgeProvisionStatus.Deleteinprogress:
      //   return 'info';
      default:
        return 'dark';
    }
  }

  getBuildStateText(state: string): string {
    if (state === LaForgeProvisionStatus.Complete) return 'ACTIVE';
    else return 'INACTIVE';
  }

  getBuildCommitHash(buildCommit: LaForgeListBuildCommitsQuery['getBuildCommits'][0]): string {
    return buildCommit.BuildCommitToBuild.BuildToRepoCommit.hash.substring(0, 7);
  }

  getBuildCommitLink(buildCommit: LaForgeListBuildCommitsQuery['getBuildCommits'][0]): string {
    const commitHash = buildCommit.BuildCommitToBuild.BuildToRepoCommit.hash;
    const repoUrl = buildCommit.BuildCommitToBuild.BuildToRepoCommit.RepoCommitToRepository.repo_url;
    const provider = repoUrl.split(':')[0].split('@')[1];
    const repoPath = repoUrl.split(':')[1].replace('.git', '');
    switch (provider) {
      case 'github.com':
        return `https://github.com/${repoPath}/commit/${commitHash}`;
      default:
        return '#';
    }
  }

  getBuildCommitCommitter(buildCommit: LaForgeListBuildCommitsQuery['getBuildCommits'][0]): string {
    return buildCommit.BuildCommitToBuild.BuildToRepoCommit.author.replace(/ <.*>/g, '');
  }

  getBuildCommitStateColor(state: string): string {
    switch (state) {
      case LaForgeBuildCommitState.Applied:
        return 'dark';
      case LaForgeBuildCommitState.Approved:
        return 'success';
      case LaForgeBuildCommitState.Cancelled:
        return 'danger';
      case LaForgeBuildCommitState.Inprogress:
        return 'info';
      case LaForgeBuildCommitState.Planning:
        return 'primary';
      default:
        return 'dark';
    }
  }

  getBuildCommitTypeColor(state: string): string {
    switch (state) {
      case LaForgeBuildCommitType.Delete:
        return 'danger';
      case LaForgeBuildCommitType.Rebuild:
        return 'info';
      case LaForgeBuildCommitType.Root:
        return 'dark';
      default:
        return 'dark';
    }
  }

  buildIsCancellable(buildCommit: LaForgeListBuildCommitsQuery['getBuildCommits'][0]): boolean {
    return buildCommit.state === LaForgeBuildCommitState.Inprogress;
  }

  buildCommitIsCancellable(buildCommit: LaForgeListBuildCommitsQuery['getBuildCommits'][0]): boolean {
    if (buildCommit.state === LaForgeBuildCommitState.Cancelled) return false;
    if (buildCommit.state === LaForgeBuildCommitState.Inprogress) return false;
    if (buildCommit.state === LaForgeBuildCommitState.Applied) return false;
    return true;
  }

  buildCommitIsPlanable(buildCommit: LaForgeListBuildCommitsQuery['getBuildCommits'][0]): boolean {
    if (buildCommit.state === LaForgeBuildCommitState.Planning) return true;
    return false;
  }

  buildCommitIsViewable(buildCommit: LaForgeListBuildCommitsQuery['getBuildCommits'][0]): boolean {
    if (buildCommit.state === LaForgeBuildCommitState.Inprogress) return true;
    return false;
  }

  buildIsManagable(build: LaForgeListBuildCommitsQuery['getBuildCommits'][0]['BuildCommitToBuild']): boolean {
    if (build.buildToStatus.state === LaForgeProvisionStatus.Deleted) return false;
    if (build.buildToStatus.state === LaForgeProvisionStatus.Planning) return false;
    if (build.buildToStatus.state === LaForgeProvisionStatus.Cancelled) return false;
    return true;
  }

  buildIsDestroyable(buildCommitGroup: LaForgeListBuildCommitsQuery['getBuildCommits']): boolean {
    if (
      buildCommitGroup.filter((bc) => bc.state === LaForgeBuildCommitState.Inprogress || bc.state === LaForgeBuildCommitState.Planning)
        .length > 0
    )
      return false;
    return true;
  }

  toggleDeleteBuildModal(
    build: LaForgeListBuildCommitsQuery['getBuildCommits'][0]['BuildCommitToBuild'],
    env: LaForgeListEnvironmentsQuery['environments'][0]
  ): void {
    this.dialog.open(DeleteBuildModalComponent, {
      width: '50%',
      data: {
        buildName: `${env.name} v${build.revision}`,
        buildId: build.id
      }
    });
  }

  openGitDialog() {
    this.dialog
      .open(ImportRepoModalComponent, {
        width: '50%',
        height: '75%',
        autoFocus: true
      })
      .afterClosed()
      .subscribe(() => console.log('TODO: Refresh env list after import')); // TODO: Refresh env list after import
  }

  createBuild(environment: LaForgeListEnvironmentsQuery['environments'][0]) {
    this.snackbar.open('Building environment...', null, {
      panelClass: ['bg-info', 'text-white']
    });
    this.api.createBuild(environment.id).then(
      (build) => {
        if (build.id) {
          // this.snackbar.open('Build created. Please wait for files to render.', 'Okay', {
          //   panelClass: ['bg-success', 'text-white']
          // });
          // .afterDismissed()
          // .subscribe(() => window.location.reload());
        }
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
        setTimeout(() => window.location.reload(), 1000);
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

  updateEnvironmentFromGit(environment: LaForgeListEnvironmentsQuery['environments'][0]) {
    this.snackbar.open('Pulling environment from git...', null, {
      panelClass: ['bg-info', 'text-white']
    });
    this.api.updateEnvFromGit(environment.id).then(
      (env) => {
        if (env.length > 0) {
          this.snackbar
            .open('Environment successfully loaded. Refreshing page...', null, {
              duration: 1000,
              panelClass: ['bg-success', 'text-white']
            })
            .afterDismissed()
            .subscribe(() => window.location.reload());
        }
      },
      (err) => {
        console.error(err);
        this.snackbar.open('Error while pulling repo from git. See console/logs for details.', 'Okay', {
          panelClass: ['bg-danger', 'text-white']
        });
      }
    );
  }

  cancelBuildCommit(buildCommit: LaForgeListBuildCommitsQuery['getBuildCommits'][0]): void {
    this.snackbar.open('Cancelling build...', 'Okay', {
      panelClass: ['bg-info', 'text-white']
    });
    this.api.cancelBuildCommit(buildCommit.id).then(
      (success) => {
        if (success) {
          this.snackbar.open('Build cancelled.', null, {
            duration: 1000,
            panelClass: ['bg-success', 'text-white']
          });
          // .afterDismissed()
          // .subscribe(() => window.location.reload());
        } else {
          this.snackbar.open('Unknown error ocurred. Check server logs.', 'Okay', {
            panelClass: ['bg-danger', 'text-white']
          });
        }
      },
      (err) => {
        console.error(err);
        this.snackbar.open('Error cancelling build. See console/logs for details.', 'Okay', {
          panelClass: ['bg-danger', 'text-white']
        });
      }
    );
  }

  canViewBuildCommitLogs(buildCommit: LaForgeListBuildCommitsQuery['getBuildCommits'][0]): boolean {
    return buildCommit.BuildCommitToServerTask.length > 0;
  }

  viewEnvironmentLogs(environment: LaForgeListEnvironmentsQuery['environments'][0]): void {
    const taskUUIDs = environment.EnvironmentToServerTask.map((s) => s.id);
    this.dialog.open(ViewLogsModalComponent, {
      width: '75%',
      height: '95%',
      data: {
        taskUUIDs
      }
    });
  }

  viewBuildCommitLogs(buildCommit: LaForgeListBuildCommitsQuery['getBuildCommits'][0]): void {
    const taskUUIDs = buildCommit.BuildCommitToServerTask.map((s) => s.id);
    this.dialog.open(ViewLogsModalComponent, {
      width: '75%',
      height: '95%',
      data: {
        taskUUIDs
      }
    });
  }
}
