import { async, ComponentFixture, TestBed } from '@angular/core/testing';

import { ScheduledStepsModalComponent } from './scheduled-steps-modal.component';

describe('ScheduledStepsModalComponent', () => {
  let component: ScheduledStepsModalComponent;
  let fixture: ComponentFixture<ScheduledStepsModalComponent>;

  beforeEach(async(() => {
    TestBed.configureTestingModule({
      declarations: [ScheduledStepsModalComponent]
    }).compileComponents();
  }));

  beforeEach(() => {
    fixture = TestBed.createComponent(ScheduledStepsModalComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
