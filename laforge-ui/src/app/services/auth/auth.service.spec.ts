import { TestBed } from '@angular/core/testing';
import { ApolloTestingModule } from 'apollo-angular/testing';
import { LaForgeAuthUser } from '@graphql';
import { MockAuthUser } from '@mock-graphql';
import { GraphQLModule } from 'src/app/graphql.module';

import { AuthService } from './auth.service';

describe('AuthService', () => {
  let service: AuthService;
  let currentUserSpy: jasmine.Spy<() => LaForgeAuthUser | null>;

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [GraphQLModule, ApolloTestingModule]
    });
    service = TestBed.inject(AuthService);
    let test = spyOn(service, 'CurrentUser').and.callFake(() => {
      // Generate a fake auth user
      return MockAuthUser();
    });
  });

  it('should be created', () => {
    expect(service).toBeTruthy();
  });
});
