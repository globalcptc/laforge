import { ChangeDetectorRef, Component, OnInit } from '@angular/core';
import { MatDialog } from '@angular/material/dialog';
import { MatSnackBar } from '@angular/material/snack-bar';

import { LaForgeGetEnvironmentsQuery } from '@graphql';
import { ApiService } from '@services/api/api.service';
import { EnvironmentService } from '@services/environment/environment.service';
import { Observable } from 'rxjs';
import { SubheaderService } from 'src/app/_metronic/partials/layout/subheader/_services/subheader.service';

import { ImportRepoModalComponent } from '@components/import-repo-modal/import-repo-modal.component';

@Component({
  selector: 'app-dashboard',
  templateUrl: './home.component.html',
  styleUrls: ['./home.component.scss']
})
export class HomeComponent implements OnInit {
  getEnvironmentsLoading: Observable<boolean>;
  environmentsCols: string[] = ['name', 'competition_id', 'build_count', 'revision', 'pull-actions', 'actions'];
  environments: Observable<LaForgeGetEnvironmentsQuery['environments']>;
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
    this.environments = this.envService.getEnvironments().asObservable();
  }

  ngOnInit(): void {
    // this.envService.getEnvironments().subscribe(() => {
    //   this.cdRef.markForCheck();
    // });
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

  createBuild(envId: string) {
    this.snackbar.open('Build is being created.', 'Okay', {
      duration: 3000,
      panelClass: ['bg-info', 'text-white']
    });
    this.api.createBuild(envId).then(
      (build) => {
        if (build.id) {
          this.snackbar.open('Build created. Please wait for files to render.', 'Okay', {
            duration: 3000,
            panelClass: ['bg-success', 'text-white']
          });
          this.envService.initEnvironments();
          this.envService.setCurrentEnv(envId, build.id);
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

  updateEnvironmentFromGit(envId: string) {
    this.api.updateEnvFromGit(envId).then(
      (env) => {
        if (env.length > 0) {
          this.snackbar.open('Environment successfully loaded. Refreshing page...', null, {
            panelClass: ['bg-success', 'text-white']
          });
          window.location.reload();
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
}
