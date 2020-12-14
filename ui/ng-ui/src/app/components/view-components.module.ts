import { NgModule } from '@angular/core';
import { CommonModule } from '@angular/common';
import { NetworkComponent } from './network/network.component';
import { HostComponent } from './host/host.component';
import { TeamComponent } from './team/team.component';
import { HostModalComponent } from './host-modal/host-modal.component';
import { MatDialogModule } from '@angular/material/dialog';
import { MatTableModule } from '@angular/material/table';
import { MatButtonModule } from '@angular/material/button';
import { NetworkModalComponent } from './network-modal/network-modal.component';

@NgModule({
  declarations: [NetworkComponent, HostComponent, TeamComponent, HostModalComponent, NetworkModalComponent],
  imports: [CommonModule, MatDialogModule, MatTableModule, MatButtonModule],
  exports: [NetworkComponent, HostComponent, TeamComponent]
})
export class ViewComponentsModule {}