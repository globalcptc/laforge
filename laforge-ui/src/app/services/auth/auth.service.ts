import { Injectable } from '@angular/core';
import { Router } from '@angular/router';
import { LaForgeGetCurrentUserGQL, LaForgeGetCurrentUserQuery } from '@graphql';
import { map, Observable } from 'rxjs';

@Injectable({
  providedIn: 'root'
})
export class AuthService {
  private _currentUser: Observable<LaForgeGetCurrentUserQuery['currentUser']>;

  constructor(private router: Router, private getCurrentUserGQL: LaForgeGetCurrentUserGQL) {
    this._currentUser = this.getCurrentUserGQL.watch().valueChanges.pipe(map((result) => result.data.currentUser));
  }

  public CurrentUser(): Observable<LaForgeGetCurrentUserQuery['currentUser']> {
    return this._currentUser;
  }

  public Login(): void {}

  public Logout(): void {}
}
