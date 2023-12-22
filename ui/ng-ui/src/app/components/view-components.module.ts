import { CommonModule } from '@angular/common';
import { NgModule } from '@angular/core';
import { ReactiveFormsModule } from '@angular/forms';

import { MatButtonModule } from '@angular/material/button';
import { MatCheckboxModule } from '@angular/material/checkbox';
import { MatDialogModule } from '@angular/material/dialog';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatSelectModule } from '@angular/material/select';
import { MatTableModule } from '@angular/material/table';

import { MomentModule } from 'ngx-moment';

import { LaforgePipesModule } from '../pipes/pipes.module';

import { DeleteBuildModalComponent } from './delete-build-modal/delete-build-modal.component';
import { EditUserModalComponent } from './edit-user-modal/edit-user-modal.component';
import { HostModalComponent } from './host-modal/host-modal.component';
import { HostComponent } from './host/host.component';
import { ImportRepoModalComponent } from './import-repo-modal/import-repo-modal.component';
import { NetworkModalComponent } from './network-modal/network-modal.component';
import { NetworkComponent } from './network/network.component';
import { NukeDbModalComponent } from './nuke-db-modal/nuke-db-modal.component';
import { ScheduledStepsModalComponent } from './scheduled-steps-modal/scheduled-steps-modal.component';
import { StepModalComponent } from './step-modal/step-modal.component';
import { StepComponent } from './step/step.component';
import { TaskListComponent } from './task-list/task-list.component';
import { TaskComponent } from './task/task.component';
import { TeamComponent } from './team/team.component';
import { ViewLogsModalComponent } from './view-logs-modal/view-logs-modal.component';

@NgModule({
  declarations: [
    NetworkComponent,
    HostComponent,
    TeamComponent,
    HostModalComponent,
    NetworkModalComponent,
    DeleteBuildModalComponent,
    StepComponent,
    TaskListComponent,
    TaskComponent,
    StepModalComponent,
    EditUserModalComponent,
    ImportRepoModalComponent,
    ViewLogsModalComponent,
    NukeDbModalComponent,
    ScheduledStepsModalComponent
  ],
  imports: [
    CommonModule,
    MatDialogModule,
    MatTableModule,
    MatButtonModule,
    MatCheckboxModule,
    MomentModule,
    LaforgePipesModule,
    MatInputModule,
    MatFormFieldModule,
    ReactiveFormsModule,
    MatSelectModule
  ],
  exports: [NetworkComponent, HostComponent, TeamComponent, TaskListComponent, TaskComponent]
})
export class ViewComponentsModule {}
