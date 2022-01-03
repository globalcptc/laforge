import { ChangeDetectorRef, Component, OnInit, OnDestroy } from '@angular/core';
import { MatDialog } from '@angular/material/dialog';
import { MatSnackBar } from '@angular/material/snack-bar';
import { ActivatedRoute, Router } from '@angular/router';
import { LaForgeGetBuildTreeQuery, LaForgeProvisionStatus, LaForgeSubscribeUpdatedStatusSubscription } from '@graphql';
import { ApiService } from '@services/api/api.service';
import { RebuildService } from '@services/rebuild/rebuild.service';
import { StatusService } from '@services/status/status.service';
import { GraphQLError } from 'graphql';
import { BehaviorSubject, Subscription } from 'rxjs';

import { SubheaderService } from '../../_metronic/partials/layout/subheader/_services/subheader.service';

import { DeleteBuildModalComponent } from '@components/delete-build-modal/delete-build-modal.component';

@Component({
  selector: 'app-manage',
  templateUrl: './manage.component.html',
  styleUrls: ['./manage.component.scss']
})
export class ManageComponent implements OnInit, OnDestroy {
  private unsubscribe: Subscription[] = [];
  private buildId: string;
  build: BehaviorSubject<LaForgeGetBuildTreeQuery['build']>;
  buildStatus: BehaviorSubject<LaForgeSubscribeUpdatedStatusSubscription['updatedStatus']>;
  environmentDetailsCols: string[] = ['TeamCount', 'AdminCIDRs', 'ExposedVDIPorts'];
  selectionMode: BehaviorSubject<boolean>;
  buildIsLoading: BehaviorSubject<boolean>;
  isRebuildLoading: BehaviorSubject<boolean>;
  rebuildErrors: (GraphQLError | Error)[] = [];
  confirmDeleteBuild = false;

  constructor(
    private dialog: MatDialog,
    private cdRef: ChangeDetectorRef,
    private subheader: SubheaderService,
    private rebuild: RebuildService,
    private router: Router,
    private snackbar: MatSnackBar,
    private status: StatusService,
    private route: ActivatedRoute,
    private api: ApiService
  ) {
    this.subheader.setTitle('Environment');
    this.subheader.setDescription('Manage your currently running environment');
    this.subheader.setShowEnvDropdown(false);

    this.build = new BehaviorSubject(null);
    this.buildStatus = new BehaviorSubject(null);
    this.selectionMode = new BehaviorSubject(false);
    this.buildIsLoading = new BehaviorSubject(false);
    this.isRebuildLoading = new BehaviorSubject(false);
  }

  ngOnInit(): void {
    this.route.params.subscribe((params) => {
      this.buildId = params.id;
      this.initBuildTree();
      this.status.loadStatusCacheFromBuild(this.buildId);
      this.status.loadAgentStatusCacheFromBuild(this.buildId);
    });
  }

  ngOnDestroy(): void {
    this.unsubscribe.forEach((sub) => sub.unsubscribe());
  }

  initBuildTree(): void {
    this.api.getBuildTree(this.buildId).then((b) => {
      this.build.next(b);
      this.buildStatus = this.status.getStatusSubject(b.buildToStatus.id);
    });
  }

  rebuildEnv(): void {
    this.isRebuildLoading.next(true);
    this.snackbar.open('Generating rebuild...', null, {
      panelClass: ['bg-info', 'text-white']
    });
    this.rebuild
      .executeRebuild()
      .then(
        (success) => {
          if (success) {
            this.router.navigate(['home']);
            this.snackbar.dismiss();
          } else {
            this.rebuildErrors = [Error('Rebuild was unsuccessfull, please check server logs for failure point.')];
            this.snackbar.open('Error generating rebuild. Please see server logs for more info.', 'Okay', {
              panelClass: ['bg-danger', 'text-white']
            });
          }
        },
        (errs) => {
          this.rebuildErrors = errs;
          console.error(errs);
          this.snackbar.open('Error generating rebuild. Please see console for more info.', 'Okay', {
            panelClass: ['bg-danger', 'text-white']
          });
        }
      )
      .finally(() => this.isRebuildLoading.next(false));
  }

  toggleSelectionMode(): void {
    this.selectionMode.next(!this.selectionMode.getValue());
  }

  toggleDeleteBuildModal(): void {
    this.dialog.open(DeleteBuildModalComponent, {
      width: '50%',
      data: {
        buildName: `${this.build.getValue().buildToEnvironment.name} v${this.build.getValue().revision}`,
        buildId: this.buildId
      }
    });
  }

  getStatus(): LaForgeSubscribeUpdatedStatusSubscription['updatedStatus'] | undefined {
    return this.buildStatus.getValue();
  }

  canDeleteBuild(): boolean {
    const status = this.getStatus();
    return (
      status &&
      (status.state === LaForgeProvisionStatus.Complete ||
        status.state === LaForgeProvisionStatus.Failed ||
        status.state === LaForgeProvisionStatus.Tainted)
    );
  }

  canSelect(): boolean {
    const status = this.getStatus();
    return (
      status &&
      status.state !== LaForgeProvisionStatus.Planning &&
      status.state !== LaForgeProvisionStatus.Deleted &&
      status.state !== LaForgeProvisionStatus.Todelete &&
      status.state !== LaForgeProvisionStatus.Deleteinprogress &&
      status.state !== LaForgeProvisionStatus.Inprogress
    );
  }

  canRebuildBuild(): boolean {
    return this.canSelect() && this.rebuild.rootPlans.length > 0;
  }
}
