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
    { label: 'Dashboard', path: '/dashboard' },
    { label: 'Projects', path: '/projects' },
    { label: 'Repositories', path: '/repositories' },
    { label: 'Pipeline Templates', path: '/pipeline-templates' },
    { label: 'Build Runs', path: '/build-runs' },
    { label: 'Environments', path: '/environments' },
    { label: 'Releases', path: '/releases' }
  ];
}
