import { async, ComponentFixture, TestBed } from '@angular/core/testing';

import { NukeDbModalComponent } from './nuke-db-modal.component';

describe('NukeDbModalComponent', () => {
  let component: NukeDbModalComponent;
  let fixture: ComponentFixture<NukeDbModalComponent>;

  beforeEach(async(() => {
    TestBed.configureTestingModule({
      declarations: [NukeDbModalComponent]
    }).compileComponents();
  }));

  beforeEach(() => {
    fixture = TestBed.createComponent(NukeDbModalComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
