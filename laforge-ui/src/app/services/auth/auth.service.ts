import { Injectable } from '@angular/core';
import { LaForgeAuthUser } from 'src/generated/graphql';

@Injectable({
  providedIn: 'root'
})
export class AuthService {
  constructor() {}

  public CurrentUser(): LaForgeAuthUser | null {
    return null;
  }

  public Login(): void {}

  public Logout(): void {}
}
