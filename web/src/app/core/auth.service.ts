import { Injectable } from '@angular/core';

import { Principal, Role } from '../api/models';

@Injectable({ providedIn: 'root' })
export class AuthService {
  token(): string | null {
    return localStorage.getItem('cloudivision.authToken');
  }

  can(principal: Principal | null | undefined, action: 'read' | 'trigger-build' | 'manage-projects' | 'approve-release'): boolean {
    if (!principal?.roles?.length) {
      return false;
    }
    if (principal.roles.includes('admin')) {
      return true;
    }
    if (action === 'read') {
      return this.hasAnyRole(principal, ['viewer', 'developer', 'project-admin']);
    }
    if (action === 'trigger-build' || action === 'approve-release') {
      return this.hasAnyRole(principal, ['developer', 'project-admin']);
    }
    if (action === 'manage-projects') {
      return this.hasAnyRole(principal, ['project-admin']);
    }
    return false;
  }

  private hasAnyRole(principal: Principal, roles: Role[]): boolean {
    return roles.some((role) => principal.roles?.includes(role));
  }
}
