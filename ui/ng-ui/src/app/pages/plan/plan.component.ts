import { ChangeDetectorRef, Component, OnDestroy, OnInit } from '@angular/core';
import { MatSnackBar } from '@angular/material/snack-bar';
import { ActivatedRoute, Router } from '@angular/router';
import { LaForgeApproveBuildCommitGQL, LaForgeBuildCommitState, LaForgeCancelBuildCommitGQL, LaForgeGetBuildCommitQuery } from '@graphql';
import { BehaviorSubject } from 'rxjs';
import { SubheaderService } from 'src/app/_metronic/partials/layout/subheader/_services/subheader.service';
import { ApiService } from 'src/app/services/api/api.service';
import { EnvironmentService } from 'src/app/services/environment/environment.service';

@Component({
  selector: 'app-plan',
  templateUrl: './plan.component.html',
  styleUrls: ['./plan.component.scss']
})
export class PlanComponent implements OnInit, OnDestroy {
  approveDenyCommitLoading: BehaviorSubject<boolean>;
  // planLoading = true;
  buildCommitId: string;
  buildCommit: BehaviorSubject<LaForgeGetBuildCommitQuery['getBuildCommit']>;

  constructor(
    private api: ApiService,
    private cdRef: ChangeDetectorRef,
    private subheader: SubheaderService,
    public envService: EnvironmentService,
    private approveBuildCommit: LaForgeApproveBuildCommitGQL,
    private cancelBuildCommit: LaForgeCancelBuildCommitGQL,
    private snackBar: MatSnackBar,
    private router: Router,
    private route: ActivatedRoute
  ) {
    this.subheader.setTitle('Plan');
    this.subheader.setDescription('Plan an environment to build');
    this.subheader.setShowEnvDropdown(false);

    this.approveDenyCommitLoading = new BehaviorSubject(false);
    this.buildCommit = new BehaviorSubject(null);

    this.route.params.subscribe((params) => {
      this.buildCommitId = params.id;
      this.getBuildCommit();
    });
  }

  ngOnInit(): void {
    // this.planLoading = true;
  }

  ngOnDestroy(): void {}

  getBuildCommit(): void {
    this.api.getBuildCommit(this.buildCommitId).then((getBuildCommit) => {
      this.buildCommit.next(getBuildCommit);
    });
  }

  getCommitStateColor(): string {
    switch (this.buildCommit.value.state) {
      case LaForgeBuildCommitState.Approved:
        return 'accent';
      case LaForgeBuildCommitState.Cancelled:
        return 'warn';
      case LaForgeBuildCommitState.Inprogress:
        return 'accent';
      case LaForgeBuildCommitState.Planning:
        return 'primary';
      case LaForgeBuildCommitState.Applied:
        return 'link';
      default:
        return 'dark';
    }
  }

  getCommitStateText(): string {
    switch (this.buildCommit.value.state) {
      case LaForgeBuildCommitState.Approved:
        return 'Approved';
      case LaForgeBuildCommitState.Cancelled:
        return 'Cancelled';
      case LaForgeBuildCommitState.Inprogress:
        return 'In Progress';
      case LaForgeBuildCommitState.Planning:
        return 'Planning';
      case LaForgeBuildCommitState.Applied:
        return 'Applied';
      default:
        return '';
    }
  }

  approveCommit(): void {
    this.approveBuildCommit
      .mutate({
        buildCommitId: this.buildCommitId
      })
      .toPromise()
      .then(
        ({ data, errors }) => {
          if (errors) {
            this.snackBar.open('Error while approving commit. See logs for more info.', 'Okay', {
              duration: 3000,
              panelClass: 'bg-danger text-white'
            });
          } else if (data.approveCommit) {
            this.snackBar.open('Commit approved', 'Nice!', {
              // duration: 3000,
              panelClass: ['bg-success', 'text-white']
            });
            this.router.navigate(['build', this.buildCommit.value.Build.id]);
          }
        },
        (err) => {
          console.error(err);
          this.snackBar.open('Error while approving commit. See console for more info.', 'Okay', {
            duration: 3000,
            panelClass: 'bg-danger text-white'
          });
        }
      );
  }

  cancelCommit(): void {
    this.cancelBuildCommit
      .mutate({
        buildCommitId: this.buildCommitId
      })
      .toPromise()
      .then(
        ({ data, errors }) => {
          if (errors) {
            this.snackBar.open('Error while cancelling commit. See logs for more info.', 'Okay', {
              duration: 3000,
              panelClass: 'bg-danger'
            });
          } else if (data.cancelCommit) {
            const ref = this.snackBar.open('Commit cancelled', 'Okay', {
              duration: 3000
            });
            ref.afterDismissed().subscribe(() => this.router.navigate(['home']));
          }
        },
        (err) => {
          console.error(err);
          this.snackBar.open('Error while cancelling commit. See console for more info.', 'Okay', {
            duration: 3000,
            panelClass: 'bg-danger'
          });
        }
      );
  }

  canApproveDenyCommit(): boolean {
    // const latestCommit = this.envService.getLatestCommit();
    // if (!latestCommit) return false;
    // if (latestCommit.state === LaForgeBuildCommitState.Planning) return true;
    if (!this.buildCommit.value) return false;
    if (this.buildCommit.value.state === LaForgeBuildCommitState.Planning) return true;
    return false;
  }
}
