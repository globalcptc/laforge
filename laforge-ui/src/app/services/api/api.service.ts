import { Injectable } from '@angular/core';
import { LaForgeAuthUser } from '@graphql';

@Injectable({
  providedIn: 'root'
})
export class ApiService {
  constructor() {}

  public Me(): LaForgeAuthUser | null {
    // TODO: Return the me graphql query
    return null;
  }
}
