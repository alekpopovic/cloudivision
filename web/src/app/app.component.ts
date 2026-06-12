import { Component } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterLink, RouterLinkActive, RouterOutlet } from '@angular/router';

@Component({
  selector: 'app-root',
  standalone: true,
  imports: [CommonModule, RouterLink, RouterLinkActive, RouterOutlet],
  templateUrl: './app.component.html'
})
export class AppComponent {
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
