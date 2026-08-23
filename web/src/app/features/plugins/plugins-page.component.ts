import { CommonModule } from '@angular/common';
import { Component, inject } from '@angular/core';
import { catchError, of, switchMap, timer } from 'rxjs';

import { ApiClient } from '../../api/client';
import { ApiError } from '../../api/models';
import { ErrorMessageComponent } from '../../shared/error-message.component';
import { PageHeaderComponent } from '../../shared/page-header.component';

@Component({
  selector: 'app-plugins-page',
  standalone: true,
  imports: [CommonModule, PageHeaderComponent, ErrorMessageComponent],
  template: `
    <app-page-header title="Plugins" description="Statically registered platform extensions, capabilities, configuration and health." />
    <app-error-message [error]="error" />
    <div *ngIf="plugins$ | async as plugins" class="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
      <article *ngFor="let plugin of plugins" class="rounded-md border border-slate-200 bg-white p-4">
        <div class="flex items-start justify-between gap-3">
          <div>
            <p class="text-xs font-semibold uppercase tracking-wide text-slate-500">{{ plugin.metadata.type }}</p>
            <h2 class="mt-1 font-semibold">{{ plugin.metadata.name }}</h2>
            <p class="text-xs text-slate-400">Version {{ plugin.metadata.version }}</p>
          </div>
          <span class="rounded-full px-2 py-1 text-xs font-medium"
            [class.bg-emerald-50]="plugin.health.healthy" [class.text-emerald-700]="plugin.health.healthy"
            [class.bg-rose-50]="!plugin.health.healthy" [class.text-rose-700]="!plugin.health.healthy">
            {{ plugin.health.healthy ? 'Healthy' : 'Unavailable' }}
          </span>
        </div>
        <p class="mt-3 text-sm text-slate-600">{{ plugin.health.message }}</p>
        <p class="mt-1 text-xs" [class.text-emerald-700]="plugin.metadata.configured" [class.text-amber-700]="!plugin.metadata.configured">
          {{ plugin.metadata.configured ? 'Configured' : 'Not configured' }} · {{ plugin.metadata.configurationStatus }}
        </p>
        <div class="mt-4 flex flex-wrap gap-2">
          <span *ngFor="let capability of plugin.metadata.capabilities" class="rounded bg-slate-100 px-2 py-1 text-xs text-slate-700" [title]="capability.description">{{ capability.name }}</span>
        </div>
        <a *ngIf="plugin.metadata.docsUrl" class="mt-4 inline-block text-sm text-sky-700 hover:underline" [href]="plugin.metadata.docsUrl" target="_blank" rel="noreferrer">Documentation</a>
      </article>
      <p *ngIf="!plugins.length" class="text-sm text-slate-500">No plugins are registered.</p>
    </div>
  `
})
export class PluginsPageComponent {
  private readonly api = inject(ApiClient);
  error: ApiError | null = null;
  readonly plugins$ = timer(0, 30000).pipe(
    switchMap(() => this.api.pluginHealth().pipe(catchError((error: ApiError) => {
      this.error = error;
      return of([]);
    })))
  );
}
