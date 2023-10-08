import { TestBed } from '@angular/core/testing';
import { MockAuthUser, GetCurrentUserDocument } from '@graphql';
import { ApolloTestingController, ApolloTestingModule } from 'apollo-angular/testing';

import { AuthService } from './auth.service';

describe('AuthService', () => {
  let controller: ApolloTestingController;
  let service: AuthService;
  // let currentUserSpy: jasmine.Spy<() => Observable<LaForgeGetCurrentUserQuery['currentUser']>>;

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [ApolloTestingModule]
    });
    service = TestBed.inject(AuthService);
    // currentUserSpy = spyOn(service, 'CurrentUser').and.returnValue(of(MockAuthUser()));
    // Inject Apollo testing controller
    controller = TestBed.inject(ApolloTestingController);
  });

  afterEach(() => {
    controller.verify();
  });

  it('should be created', () => {
    expect(service).toBeTruthy();
  });

  it('should return current user', () => {
    service.CurrentUser().subscribe((currentUser) => {
      console.log(currentUser);
      expect(currentUser).toEqual(MockAuthUser());
    });

    const op = controller.expectOne(GetCurrentUserDocument);
    expect(op.operation.operationName).toBe('GetCurrentUser');

    console.log(MockAuthUser());

    op.flush({
      data: {
        currentUser: MockAuthUser()
      }
    });
  });
});
