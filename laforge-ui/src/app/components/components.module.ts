import { NgModule } from '@angular/core';

import { LayoutComponent } from './layout/layout.component';
import { RouterOutlet } from '@angular/router';
import { CommonModule } from '@angular/common';

@NgModule({
  declarations: [LayoutComponent],
  imports: [RouterOutlet, CommonModule],
  exports: []
})
export class ComponentsModule {}
