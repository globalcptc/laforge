import { ComponentFixture, TestBed, waitForAsync } from '@angular/core/testing';
import { By } from '@angular/platform-browser';
import { RouterTestingModule } from '@angular/router/testing';
import { TitleService } from '@services/title/title.service';

import { LayoutComponent } from './layout.component';

describe('LayoutComponent', () => {
  let component: LayoutComponent;
  let fixture: ComponentFixture<LayoutComponent>;
  let titleService: TitleService;
  let getRandomIntSpy: jasmine.Spy<() => number>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [RouterTestingModule],
      declarations: [LayoutComponent],
      providers: [TitleService]
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
    let title = fixture.debugElement.query(By.css('.page-title'));
    expect(title.nativeElement.textContent.trim()).toBe('LaForge Test');
  }));

  it('should dynamically render title', waitForAsync(() => {
    let title = fixture.debugElement.query(By.css('.page-title'));
    titleService.setTitle('LaForge Test 1');
    fixture.detectChanges();
    expect(title.nativeElement.textContent.trim()).toBe('LaForge Test 1');
    titleService.setTitle('LaForge Test 2');
    fixture.detectChanges();
    expect(title.nativeElement.textContent.trim()).toBe('LaForge Test 2');
  }));
});
