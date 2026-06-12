import { CommonModule } from '@angular/common';
import { Component, inject } from '@angular/core';
import { FormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { catchError, combineLatest, map, of, startWith, Subject, switchMap } from 'rxjs';

import { ApiClient } from '../../api/client';
import { ApiError, Repository } from '../../api/models';
import { EmptyStateComponent } from '../../shared/empty-state.component';
import { ErrorMessageComponent } from '../../shared/error-message.component';
import { PageHeaderComponent } from '../../shared/page-header.component';
import { StatusBadgeComponent } from '../../shared/status-badge.component';

@Component({
  selector: 'app-repositories-page',
  standalone: true,
  imports: [CommonModule, ReactiveFormsModule, PageHeaderComponent, StatusBadgeComponent, EmptyStateComponent, ErrorMessageComponent],
  template: `
    <app-page-header title="Repositories" description="Source repositories watched by webhooks or manual triggers." />
    <app-error-message [error]="error" />
    <section class="grid gap-6 lg:grid-cols-[1fr_24rem]">
      <div class="rounded-md border border-slate-200 bg-white">
        <div class="border-b border-slate-200 px-4 py-3 font-medium">Repository List</div>
        <div *ngIf="repositories$ | async as repositories">
          <div *ngIf="repositories.length; else empty" class="divide-y divide-slate-100">
            <div *ngFor="let repository of repositories" class="px-4 py-3">
              <div class="flex items-center justify-between gap-3">
                <div>
                  <p class="text-sm font-medium">{{ repository.name }}</p>
                  <p class="break-all text-xs text-slate-500">{{ repository.spec.url }}</p>
                </div>
                <app-status-badge [status]="repository.status?.phase || 'Pending'" />
              </div>
              <p class="mt-2 break-all rounded-md bg-slate-50 px-3 py-2 text-xs text-slate-600">{{ webhookUrl(repository) }}</p>
            </div>
          </div>
          <ng-template #empty><app-empty-state title="No repositories" message="Create a repository to receive Git events." /></ng-template>
        </div>
      </div>
      <form [formGroup]="form" (ngSubmit)="create()" class="rounded-md border border-slate-200 bg-white p-4">
        <h2 class="text-sm font-semibold">Create Repository</h2>
        <label class="mt-4 block text-sm">Name<input class="mt-1 w-full rounded-md border border-slate-300 px-3 py-2" formControlName="name" /></label>
        <label class="mt-3 block text-sm">Project<input class="mt-1 w-full rounded-md border border-slate-300 px-3 py-2" formControlName="projectRef" /></label>
        <label class="mt-3 block text-sm">Provider
          <select class="mt-1 w-full rounded-md border border-slate-300 px-3 py-2" formControlName="provider">
            <option value="github">github</option>
            <option value="gitlab">gitlab</option>
            <option value="gitea">gitea</option>
            <option value="generic">generic</option>
          </select>
        </label>
        <label class="mt-3 block text-sm">URL<input class="mt-1 w-full rounded-md border border-slate-300 px-3 py-2" formControlName="url" /></label>
        <label class="mt-3 block text-sm">Default branch<input class="mt-1 w-full rounded-md border border-slate-300 px-3 py-2" formControlName="defaultBranch" /></label>
        <label class="mt-3 block text-sm">Pipeline template<input class="mt-1 w-full rounded-md border border-slate-300 px-3 py-2" formControlName="pipelineTemplateRef" /></label>
        <button class="mt-4 rounded-md bg-emerald-700 px-4 py-2 text-sm font-medium text-white disabled:opacity-50" [disabled]="form.invalid">Create</button>
      </form>
    </section>
  `
})
export class RepositoriesPageComponent {
  private readonly api = inject(ApiClient);
  private readonly fb = inject(FormBuilder);
  private readonly refresh$ = new Subject<void>();
  error: ApiError | null = null;
  apiBase = '';
  readonly repositories$ = combineLatest([
    this.refresh$.pipe(startWith(undefined), switchMap(() => this.api.repositories().pipe(catchError((error: ApiError) => { this.error = error; return of([]); })))),
    this.api.webhookUrl('github', '')
  ]).pipe(map(([repositories, base]) => { this.apiBase = base.replace(/\/github\/$/, ''); return repositories; }));
  readonly form = this.fb.nonNullable.group({
    name: ['', Validators.required],
    projectRef: ['', Validators.required],
    provider: ['github' as Repository['spec']['provider'], Validators.required],
    url: ['', Validators.required],
    defaultBranch: ['main', Validators.required],
    pipelineTemplateRef: ['', Validators.required]
  });

  webhookUrl(repository: Repository): string {
    return `${this.apiBase}/${repository.spec.provider}/${repository.name}`;
  }

  create(): void {
    if (this.form.invalid) return;
    const value = this.form.getRawValue();
    this.api.createRepository({
      name: value.name,
      spec: {
        projectRef: value.projectRef,
        provider: value.provider,
        url: value.url,
        defaultBranch: value.defaultBranch,
        pipelineTemplateRef: value.pipelineTemplateRef,
        webhook: { enabled: true, events: ['push'] }
      }
    }).subscribe({ next: () => { this.form.reset({ provider: 'github', defaultBranch: 'main' }); this.refresh$.next(); }, error: (error: ApiError) => (this.error = error) });
  }
}
