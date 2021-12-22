import { ChangeDetectorRef, Component, OnInit, OnDestroy } from '@angular/core';
import { MatSnackBar } from '@angular/material/snack-bar';
import { ActivatedRoute } from '@angular/router';
import { LaForgeGetBuildTreeQuery, LaForgeSubscribeUpdatedStatusSubscription } from '@graphql';
import { ApiService } from '@services/api/api.service';
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
  statuses: BehaviorSubject<LaForgeSubscribeUpdatedStatusSubscription['updatedStatus'][]>;
  executeBuildLoading = false;
  planStatusesLoading = false;
  agentStatusesLoading = false;

  constructor(
    private subheader: SubheaderService,
    public envService: EnvironmentService,
    private cdRef: ChangeDetectorRef,
    private executeBuild: LaForgeExecuteBuildGQL,
    private snackbar: MatSnackBar,
    private api: ApiService,
    private route: ActivatedRoute
  ) {
    this.subheader.setTitle('Build');
    this.subheader.setDescription('Monitor the progress of a given build');
    this.subheader.setShowEnvDropdown(false);

    this.build = new BehaviorSubject(null);
    this.statuses = new BehaviorSubject([]);
  }

  ngOnInit(): void {
    this.route.params.subscribe((params) => {
      this.buildId = params.id;
      this.initBuildTree();
      this.initStatusMap();

      const sub1 = this.envService.getStatusSubscription().subscribe(({ data: { updatedStatus }, errors }) => {
        console.log(updatedStatus.id);
      });
      this.unsubscribe.push(sub1);
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
    });
  }

  initStatusMap(): void {
    this.api.listBuildStatuses(this.buildId).then((s) => {
      this.statuses.next(s);
    });
  }
}
