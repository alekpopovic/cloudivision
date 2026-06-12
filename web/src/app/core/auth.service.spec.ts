import { TestBed } from '@angular/core/testing';

import { AuthService } from './auth.service';

describe('AuthService', () => {
  let service: AuthService;

  beforeEach(() => {
    TestBed.configureTestingModule({});
    service = TestBed.inject(AuthService);
  });

  it('allows developers to trigger builds but not manage projects', () => {
    const principal = { subject: 'user-1', roles: ['developer' as const] };

    expect(service.can(principal, 'trigger-build')).toBeTrue();
    expect(service.can(principal, 'manage-projects')).toBeFalse();
  });

  it('allows admin to do everything', () => {
    const principal = { subject: 'admin', roles: ['admin' as const] };

    expect(service.can(principal, 'manage-projects')).toBeTrue();
    expect(service.can(principal, 'approve-release')).toBeTrue();
  });
});
