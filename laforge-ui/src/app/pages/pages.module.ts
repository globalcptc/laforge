import { CommonModule } from '@angular/common';
import { NgModule } from '@angular/core';

import { RouterOutlet } from '@angular/router';

import { AuthComponent } from './auth/auth.component';
import { DashboardComponent } from './dashboard/dashboard.component';
import { ErrorComponent } from './error/error.component';

@NgModule({
  declarations: [DashboardComponent, ErrorComponent, AuthComponent],
  imports: [RouterOutlet, CommonModule],
  exports: []
})
export class PagesModule {}
