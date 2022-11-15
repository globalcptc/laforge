import { Injectable, NgModule } from '@angular/core';
import { BrowserModule, Title } from '@angular/platform-browser';
import { RouterStateSnapshot, TitleStrategy } from '@angular/router';

import { AppRoutingModule } from './app-routing.module';
import { AppComponent } from './app.component';
import { GraphQLModule } from './graphql.module';
import { PagesModule } from './pages/pages.module';
import { BrowserAnimationsModule } from '@angular/platform-browser/animations';
import { ComponentsModule } from '@components/components.module';

@Injectable({ providedIn: 'root' })
export class LaForgePageTitleStrategy extends TitleStrategy {
  constructor(private readonly title: Title) {
    super();
  }

  override updateTitle(routerState: RouterStateSnapshot) {
    const title = this.buildTitle(routerState);
    if (title !== undefined) {
      this.title.setTitle(`LaForge | ${title}`);
    } else {
      this.title.setTitle('LaForge');
    }
  }
}

@NgModule({
  declarations: [AppComponent],
  imports: [BrowserModule, AppRoutingModule, GraphQLModule, ComponentsModule, PagesModule, BrowserAnimationsModule],
  bootstrap: [AppComponent],
  providers: [{ provide: TitleStrategy, useClass: LaForgePageTitleStrategy }]
})
export class AppModule {}
