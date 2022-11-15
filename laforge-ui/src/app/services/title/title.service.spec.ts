import { waitForAsync, TestBed } from '@angular/core/testing';

import { TitleService } from './title.service';

describe('TitleService', () => {
  let service: TitleService;

  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [TitleService]
    });
    service = TestBed.inject(TitleService);
  });

  it('should be created', () => {
    expect(service).toBeTruthy();
  });

  it('should notify subscribers', waitForAsync(() => {
    service.setTitle('new title');

    service.getTitle().subscribe((title) => {
      expect(title).toBe('new title');
    });
  }));
});
