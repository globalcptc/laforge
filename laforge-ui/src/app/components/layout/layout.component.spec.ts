import { ComponentFixture, TestBed } from '@angular/core/testing';
import { RouterTestingModule } from '@angular/router/testing';

import { LayoutComponent } from './layout.component';

describe('LayoutComponent', () => {
  let component: LayoutComponent;
  let fixture: ComponentFixture<LayoutComponent>;
  let getRandomIntSpy: jasmine.Spy<() => number>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [RouterTestingModule],
      declarations: [LayoutComponent]
    }).compileComponents();

    fixture = TestBed.createComponent(LayoutComponent);
    getRandomIntSpy = spyOn(fixture.componentInstance, 'getRandomInt').and.returnValue(0);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
    expect(getRandomIntSpy).toHaveBeenCalled();
  });
});
