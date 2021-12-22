import { ChangeDetectorRef, Component, OnInit } from '@angular/core';
import { MatDialog } from '@angular/material/dialog';
import { MatSnackBar } from '@angular/material/snack-bar';

import { LaForgeBuildCommitState, LaForgeBuildCommitType, LaForgeListBuildCommitsQuery, LaForgeListEnvironmentsQuery } from '@graphql';
import { ApiService } from '@services/api/api.service';
import { EnvironmentService } from '@services/environment/environment.service';
import { BehaviorSubject, Observable } from 'rxjs';
import { SubheaderService } from 'src/app/_metronic/partials/layout/subheader/_services/subheader.service';

import { ImportRepoModalComponent } from '@components/import-repo-modal/import-repo-modal.component';

@Component({
  selector: 'app-dashboard',
  templateUrl: './home.component.html',
  styleUrls: ['./home.component.scss']
})
export class HomeComponent implements OnInit {
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
    private dialog: MatDialog
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
  }

  getLatestCommitHash(environment: LaForgeListEnvironmentsQuery['environments'][0]): string {
    return environment.EnvironmentToRepository[0].RepositoryToRepoCommit.sort((a, b) => a.revision - b.revision)[0].hash.substring(0, 7);
  }

  getLatestCommitLink(environment: LaForgeListEnvironmentsQuery['environments'][0]): string {
    const commitHash = environment.EnvironmentToRepository[0].RepositoryToRepoCommit.sort((a, b) => a.revision - b.revision)[0].hash;
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
    return environment.EnvironmentToRepository[0].RepositoryToRepoCommit.sort((a, b) => a.revision - b.revision)[0].author.replace(
      / <.*>/g,
      ''
    );
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
        bc[environment.id] = [...buildCommits].sort((a, b) => a.revision - b.revision);
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

  buildCommitIsCancellable(buildCommit: LaForgeListBuildCommitsQuery['getBuildCommits'][0]): boolean {
    if (buildCommit.state === LaForgeBuildCommitState.Cancelled) return false;
    if (buildCommit.state === LaForgeBuildCommitState.Inprogress) return false;
    return true;
  }

  buildCommitIsPlanable(buildCommit: LaForgeListBuildCommitsQuery['getBuildCommits'][0]): boolean {
    if (buildCommit.state === LaForgeBuildCommitState.Planning) return true;
    return false;
  }

  buildCommitIsManagable(buildCommit: LaForgeListBuildCommitsQuery['getBuildCommits'][0]): boolean {
    // if (buildCommit.state === LaForgeBuildCommitState.Cancelled) return false;
    // if (buildCommit.state === LaForgeBuildCommitState.Planning) return false;
    return true;
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
          this.snackbar
            .open('Build created. Please wait for files to render.', 'Okay', {
              panelClass: ['bg-success', 'text-white']
            })
            .afterDismissed()
            .subscribe(() => window.location.reload());
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

  updateEnvironmentFromGit(environment: LaForgeListEnvironmentsQuery['environments'][0]) {
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
          this.snackbar
            .open('Build cancelled.', null, {
              duration: 1000,
              panelClass: ['bg-success', 'text-white']
            })
            .afterDismissed()
            .subscribe(() => window.location.reload());
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
}
