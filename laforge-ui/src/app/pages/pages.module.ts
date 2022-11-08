import { NgModule } from '@angular/core';

import { DashboardComponent } from './dashboard/dashboard.component';
import { ErrorComponent } from './error/error.component';
import { AuthComponent } from './auth/auth.component';
import { RouterOutlet } from '@angular/router';

@NgModule({
  declarations: [DashboardComponent, ErrorComponent, AuthComponent],
  imports: [RouterOutlet],
  exports: []
})
export class PagesModule {}
