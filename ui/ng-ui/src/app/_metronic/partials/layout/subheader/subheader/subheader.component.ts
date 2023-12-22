import { ChangeDetectorRef, Component, OnInit } from '@angular/core';
import { Observable } from 'rxjs';
import { BreadcrumbItemModel } from '../_models/breadcrumb-item.model';
import { LayoutService } from '../../../../core';
import { SubheaderService } from '../_services/subheader.service';

import { PlanService } from '@services/plan/plan.service';
import { MatSelectChange } from '@angular/material/select';
import { EnvironmentService } from 'src/app/services/environment/environment.service';
import { LaForgeGetBuildTreeQuery, LaForgeGetEnvironmentGQL, LaForgeGetEnvironmentInfoQuery, LaForgeGetEnvironmentsQuery } from '@graphql';

interface Branch {
  name: string;
  hash: string;
}

@Component({
  selector: 'app-subheader',
  templateUrl: './subheader.component.html',
  styleUrls: ['./subheader.component.scss']
})
export class SubheaderComponent implements OnInit {
  subheaderCSSClasses = '';
  subheaderContainerCSSClasses = '';
  subheaderMobileToggle = false;
  subheaderDisplayDesc = false;
  subheaderDisplayDaterangepicker = false;
  title$: Observable<string>;
  breadcrumbs$: Observable<BreadcrumbItemModel[]>;
  description$: Observable<string>;
  showEnvDropdown$: Observable<boolean>;
  branches: Branch[] = [
    { name: 'Bradley', hash: '98y3if' },
    { name: 'Lucas', hash: '32a7fh' }
  ];
  environments: Observable<LaForgeGetEnvironmentsQuery['environments']>;
  environment: LaForgeGetEnvironmentInfoQuery['environment'];
  build: LaForgeGetBuildTreeQuery['build'];
  envIsLoading: Observable<boolean>;
  buildIsLoading: Observable<boolean>;

  constructor(
    private layout: LayoutService,
    private subheader: SubheaderService,
    private planService: PlanService,
    public envService: EnvironmentService,
    private cdRef: ChangeDetectorRef
  ) {
    this.title$ = this.subheader.titleSubject.asObservable();
    this.breadcrumbs$ = this.subheader.breadCrumbsSubject.asObservable();
    this.description$ = this.subheader.descriptionSubject.asObservable();
    this.showEnvDropdown$ = this.subheader.showEnvironmentDropdown.asObservable();

    // this.envService.getEnvironments().subscribe(() => {
    //   this.cdRef.markForCheck();
    // });

    this.environments = this.envService.getEnvironments().asObservable();

    this.envService
      .getEnvironmentInfo()
      .asObservable()
      .subscribe((env) => (this.environment = env));
    this.envService
      .getBuildTree()
      .asObservable()
      .subscribe((env) => (this.build = env));
    this.envIsLoading = this.envService.envIsLoading.asObservable();
    this.buildIsLoading = this.envService.buildIsLoading.asObservable();
  }

  ngOnInit() {
    this.subheaderCSSClasses = this.layout.getStringCSSClasses('subheader');
    this.subheaderContainerCSSClasses = this.layout.getStringCSSClasses('subheader_container');
    this.subheaderMobileToggle = this.layout.getProp('subheader.mobileToggle');
    this.subheaderDisplayDesc = this.layout.getProp('subheader.displayDesc');
    this.subheaderDisplayDaterangepicker = this.layout.getProp('subheader.displayDaterangepicker');
  }

  onBranchSelect(changeEvent: MatSelectChange) {
    // console.log(changeEvent);
    this.planService.getPlan(changeEvent.value);
  }

  compareEnvDropdown(
    option: {
      env: string;
      build: string;
    },
    value: {
      env: string;
      build: string;
    }
  ) {
    return option.env === value.env && option.build === value.build;
  }

  selectEnvironment(event: MatSelectChange) {
    const [envId, buildId] = (event.value as string).split('|');
    this.envService.setCurrentEnv(envId, buildId);
    this.cdRef.detectChanges();
  }
}
