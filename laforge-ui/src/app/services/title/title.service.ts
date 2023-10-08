import { Injectable } from '@angular/core';
import { BehaviorSubject } from 'rxjs';

@Injectable({
  providedIn: 'root'
})
export class TitleService {
  private title: BehaviorSubject<string>;

  constructor() {
    this.title = new BehaviorSubject<string>('LaForge');
  }

  public getTitle(): BehaviorSubject<string> {
    return this.title;
  }

  public setTitle(title: string): void {
    this.title.next(title);
  }
}
