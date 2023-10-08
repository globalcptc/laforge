import { CommonModule } from '@angular/common';
import { NgModule } from '@angular/core';

import { RouterOutlet } from '@angular/router';

import { LayoutComponent } from './layout/layout.component';

@NgModule({
  declarations: [LayoutComponent],
  imports: [RouterOutlet, CommonModule],
  exports: []
})
export class ComponentsModule {}
