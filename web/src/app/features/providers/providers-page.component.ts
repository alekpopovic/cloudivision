import { CommonModule } from '@angular/common';
import { Component, inject } from '@angular/core';
import { catchError, of, timer, switchMap } from 'rxjs';

import { ApiClient } from '../../api/client';
import { ApiError } from '../../api/models';
import { ErrorMessageComponent } from '../../shared/error-message.component';
import { PageHeaderComponent } from '../../shared/page-header.component';

@Component({
  selector: 'app-providers-page',
  standalone: true,
  imports: [CommonModule, PageHeaderComponent, ErrorMessageComponent],
  template: `
    <app-page-header title="Providers" description="Registered integration capabilities and live adapter health." />
    <app-error-message [error]="error" />
    <div *ngIf="providers$ | async as providers" class="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
      <article *ngFor="let provider of providers" class="rounded-md border border-slate-200 bg-white p-4">
        <div class="flex items-start justify-between gap-3">
          <div><p class="text-xs font-semibold uppercase tracking-wide text-slate-500">{{ provider.type }}</p><h2 class="mt-1 font-semibold">{{ provider.name }}</h2></div>
          <span class="rounded-full px-2 py-1 text-xs font-medium" [class.bg-emerald-50]="provider.health.healthy" [class.text-emerald-700]="provider.health.healthy" [class.bg-rose-50]="!provider.health.healthy" [class.text-rose-700]="!provider.health.healthy">{{ provider.health.healthy ? 'Healthy' : 'Unavailable' }}</span>
        </div>
        <p class="mt-3 text-sm text-slate-600">{{ provider.health.message }}</p>
        <p class="mt-1 text-xs text-slate-400">Checked {{ provider.health.checkedAt | date:'medium' }}</p>
        <div class="mt-4 flex flex-wrap gap-2">
          <span *ngFor="let capability of provider.capabilities" class="rounded bg-slate-100 px-2 py-1 text-xs text-slate-700" [title]="capability.description">{{ capability.name }}</span>
        </div>
      </article>
      <p *ngIf="!providers.length" class="text-sm text-slate-500">No providers are registered.</p>
    </div>
  `
})
export class ProvidersPageComponent {
  private readonly api = inject(ApiClient);
  error: ApiError | null = null;
  readonly providers$ = timer(0, 30000).pipe(switchMap(() => this.api.providerHealth().pipe(catchError((error: ApiError) => { this.error = error; return of([]); }))));
}
