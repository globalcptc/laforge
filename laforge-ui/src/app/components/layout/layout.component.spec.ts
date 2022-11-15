import { Component } from '@angular/core';
import { ComponentFixture, TestBed, waitForAsync } from '@angular/core/testing';
import { By } from '@angular/platform-browser';
import { TitleService } from '@services/title/title.service';

import { LayoutComponent } from './layout.component';

// eslint-disable-next-line @angular-eslint/component-selector
@Component({ selector: 'router-outlet', template: '' })
class RouterOutletStubComponent {}

describe('LayoutComponent', () => {
  let component: LayoutComponent;
  let fixture: ComponentFixture<LayoutComponent>;
  let titleService: TitleService;
  let getRandomIntSpy: jasmine.Spy<() => number>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      declarations: [LayoutComponent, RouterOutletStubComponent]
    }).compileComponents();

    fixture = TestBed.createComponent(LayoutComponent);
    component = fixture.componentInstance;
    getRandomIntSpy = spyOn(component, 'getRandomInt').and.returnValue(0);
    titleService = TestBed.inject(TitleService);
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
    expect(getRandomIntSpy).toHaveBeenCalled();
  });

  it('should render title', waitForAsync(() => {
    titleService.setTitle('LaForge Test');
    fixture.detectChanges();
    const title = fixture.debugElement.query(By.css('.page-title'));
    expect(title.nativeElement.textContent.trim()).toBe('LaForge Test');
  }));

  it('should dynamically render title', waitForAsync(() => {
    const title = fixture.debugElement.query(By.css('.page-title'));
    titleService.setTitle('LaForge Test 1');
    fixture.detectChanges();
    expect(title.nativeElement.textContent.trim()).toBe('LaForge Test 1');
    titleService.setTitle('LaForge Test 2');
    fixture.detectChanges();
    expect(title.nativeElement.textContent.trim()).toBe('LaForge Test 2');
  }));
});
