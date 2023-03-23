import { async, ComponentFixture, TestBed } from '@angular/core/testing';

import { ViewLogsModalComponent } from './view-logs-modal.component';

describe('DeleteBuildModalComponent', () => {
  let component: ViewLogsModalComponent;
  let fixture: ComponentFixture<ViewLogsModalComponent>;

  beforeEach(async(() => {
    TestBed.configureTestingModule({
      declarations: [ViewLogsModalComponent]
    }).compileComponents();
  }));

  beforeEach(() => {
    fixture = TestBed.createComponent(ViewLogsModalComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
