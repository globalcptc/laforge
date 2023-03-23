import { async, ComponentFixture, TestBed } from '@angular/core/testing';

import { ImportRepoModalComponent } from './import-repo-modal.component';

describe('EditUserModalComponent', () => {
  let component: ImportRepoModalComponent;
  let fixture: ComponentFixture<ImportRepoModalComponent>;

  beforeEach(async(() => {
    TestBed.configureTestingModule({
      declarations: [ImportRepoModalComponent]
    }).compileComponents();
  }));

  beforeEach(() => {
    fixture = TestBed.createComponent(ImportRepoModalComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
