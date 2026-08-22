import { Component, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterLink, RouterLinkActive, RouterOutlet } from '@angular/router';
import { catchError, of } from 'rxjs';

import { ApiClient } from './api/client';

@Component({
  selector: 'app-root',
  standalone: true,
  imports: [CommonModule, RouterLink, RouterLinkActive, RouterOutlet],
  templateUrl: './app.component.html'
})
export class AppComponent {
  private readonly api = inject(ApiClient);
  readonly currentUser$ = this.api.currentUser().pipe(catchError(() => of(null)));

  readonly nav = [
    { label: 'Dashboard', path: '/dashboard', icon: 'observe' },
    { label: 'First Run', path: '/first-run', icon: 'pipeline' },
    { label: 'Projects', path: '/projects', icon: 'platform' },
    { label: 'Repositories', path: '/repositories', icon: 'source' },
    { label: 'Pipeline Templates', path: '/pipeline-templates', icon: 'pipeline' },
    { label: 'Providers', path: '/providers', icon: 'platform' },
    { label: 'Build Runs', path: '/build-runs', icon: 'build' },
    { label: 'Environments', path: '/environments', icon: 'deploy' },
    { label: 'Releases', path: '/releases', icon: 'deploy' }
  ];
}
