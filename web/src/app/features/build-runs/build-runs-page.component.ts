import { CommonModule } from '@angular/common';
import { Component, inject } from '@angular/core';
import { FormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { RouterLink } from '@angular/router';
import { BehaviorSubject, catchError, combineLatest, debounceTime, map, of, startWith, switchMap, timer } from 'rxjs';

import { ApiClient } from '../../api/client';
import { ApiError, BuildRun } from '../../api/models';
import { EmptyStateComponent } from '../../shared/empty-state.component';
import { ErrorMessageComponent } from '../../shared/error-message.component';
import { PageHeaderComponent } from '../../shared/page-header.component';
import { StatusBadgeComponent } from '../../shared/status-badge.component';

@Component({
  selector: 'app-build-runs-page',
  standalone: true,
  imports: [CommonModule, ReactiveFormsModule, RouterLink, PageHeaderComponent, StatusBadgeComponent, EmptyStateComponent, ErrorMessageComponent],
  template: `
    <app-page-header title="Build Runs" description="CI executions, status and manual triggers." />
    <app-error-message [error]="error" />
    <section class="grid gap-6 xl:grid-cols-[1fr_26rem]">
      <div>
        <form [formGroup]="filters" class="mb-4 grid gap-3 rounded-md border border-slate-200 bg-white p-4 md:grid-cols-3">
          <input class="rounded-md border border-slate-300 px-3 py-2 text-sm" placeholder="phase" formControlName="phase" />
          <input class="rounded-md border border-slate-300 px-3 py-2 text-sm" placeholder="project" formControlName="project" />
          <input class="rounded-md border border-slate-300 px-3 py-2 text-sm" placeholder="repository" formControlName="repository" />
        </form>
        <div class="rounded-md border border-slate-200 bg-white">
          <div class="flex justify-between border-b border-slate-200 px-4 py-3"><span class="font-medium">BuildRun List</span><span class="text-xs text-slate-500">50 per page</span></div>
          <div *ngIf="page$ | async as page">
            <ng-container *ngIf="page.items as runs">
            <div *ngIf="runs.length; else empty" class="divide-y divide-slate-100">
              <a *ngFor="let run of runs" [routerLink]="['/build-runs', run.namespace, run.name]" class="flex items-center justify-between px-4 py-3 hover:bg-slate-50">
                <span>
                  <span class="block text-sm font-medium">{{ run.name }}</span>
                  <span class="text-xs text-slate-500">{{ run.spec.projectRef }} / {{ run.spec.repositoryRef }} / {{ run.spec.revision }}</span>
                </span>
                <app-status-badge [status]="run.status?.phase || 'Pending'" />
              </a>
            </div>
            <ng-template #empty><app-empty-state title="No BuildRuns" message="Run filters may be empty, or no builds have run yet." /></ng-template>
            <div class="flex items-center justify-between border-t border-slate-200 px-4 py-3 text-xs text-slate-600">
              <span>{{ page.totalCount }} matching BuildRuns</span>
              <span class="flex gap-2">
                <button type="button" class="rounded border border-slate-300 px-3 py-1 disabled:opacity-40" [disabled]="pageIndex === 0" (click)="previousPage()">Previous</button>
                <button type="button" class="rounded border border-slate-300 px-3 py-1 disabled:opacity-40" [disabled]="!page.nextPageToken" (click)="nextPage(page.nextPageToken)">Next</button>
              </span>
            </div>
            </ng-container>
          </div>
        </div>
      </div>
      <form [formGroup]="triggerForm" (ngSubmit)="trigger()" class="rounded-md border border-slate-200 bg-white p-4">
        <h2 class="text-sm font-semibold">Manual Trigger</h2>
        <label class="mt-4 block text-sm">Name<input class="mt-1 w-full rounded-md border border-slate-300 px-3 py-2" formControlName="name" /></label>
        <label class="mt-3 block text-sm">Namespace<input class="mt-1 w-full rounded-md border border-slate-300 px-3 py-2" formControlName="namespace" /></label>
        <label class="mt-3 block text-sm">Project<input class="mt-1 w-full rounded-md border border-slate-300 px-3 py-2" formControlName="projectRef" /></label>
        <label class="mt-3 block text-sm">Repository<input class="mt-1 w-full rounded-md border border-slate-300 px-3 py-2" formControlName="repositoryRef" /></label>
        <label class="mt-3 block text-sm">Pipeline template<input class="mt-1 w-full rounded-md border border-slate-300 px-3 py-2" formControlName="pipelineTemplateRef" /></label>
        <label class="mt-3 block text-sm">Revision<input class="mt-1 w-full rounded-md border border-slate-300 px-3 py-2" formControlName="revision" /></label>
        <label class="mt-3 block text-sm">Image repository<input class="mt-1 w-full rounded-md border border-slate-300 px-3 py-2" formControlName="imageRepository" /></label>
        <label class="mt-3 block text-sm">Image tag<input class="mt-1 w-full rounded-md border border-slate-300 px-3 py-2" formControlName="imageTag" /></label>
        <button class="mt-4 rounded-md bg-emerald-700 px-4 py-2 text-sm font-medium text-white disabled:opacity-50" [disabled]="triggerForm.invalid">Trigger</button>
      </form>
    </section>
  `
})
export class BuildRunsPageComponent {
  private readonly api = inject(ApiClient);
  private readonly fb = inject(FormBuilder);
  error: ApiError | null = null;
  pageIndex = 0;
  private pageTokens = [''];
  private readonly pageToken$ = new BehaviorSubject('');
  readonly filters = this.fb.nonNullable.group({ phase: [''], project: [''], repository: [''] });
  readonly page$ = this.filters.valueChanges.pipe(
    startWith(this.filters.getRawValue()),
    debounceTime(200),
    switchMap((filters) => {
      this.pageIndex = 0;
      this.pageTokens = [''];
      this.pageToken$.next('');
      return combineLatest([timer(0, 5000), this.pageToken$]).pipe(
      switchMap(([, pageToken]) => this.api.buildRunPage({
				limit: '50',
        ...(pageToken ? { pageToken } : {}),
        ...(filters.phase ? { phase: filters.phase } : {}),
        ...(filters.project ? { project: filters.project } : {}),
        ...(filters.repository ? { repository: filters.repository } : {})
				}).pipe(catchError((error: ApiError) => { this.error = error; return of({ items: [] as BuildRun[], nextPageToken: '', totalCount: 0, limit: 50 }); })))
    );})
  );
  readonly filteredRuns$ = this.page$.pipe(map((page) => page.items));
  readonly triggerForm = this.fb.nonNullable.group({
    name: ['', Validators.required],
    namespace: ['default', Validators.required],
    projectRef: ['', Validators.required],
    repositoryRef: ['', Validators.required],
    pipelineTemplateRef: ['', Validators.required],
    revision: ['main', Validators.required],
    imageRepository: ['', Validators.required],
    imageTag: ['manual']
  });

  trigger(): void {
    if (this.triggerForm.invalid) return;
    const value = this.triggerForm.getRawValue();
    this.api.createBuildRun({
      name: value.name,
      namespace: value.namespace,
      spec: {
        projectRef: value.projectRef,
        repositoryRef: value.repositoryRef,
        pipelineTemplateRef: value.pipelineTemplateRef,
        revision: value.revision,
        triggeredBy: { type: 'manual', actor: 'web' },
        image: { repository: value.imageRepository, tag: value.imageTag },
        executor: 'job'
      }
    }).subscribe({ next: () => this.triggerForm.reset({ namespace: 'default', revision: 'main', imageTag: 'manual' }), error: (error: ApiError) => (this.error = error) });
  }

  nextPage(token: string | undefined): void {
    if (!token) return;
    this.pageIndex++;
    this.pageTokens[this.pageIndex] = token;
    this.pageToken$.next(token);
  }

  previousPage(): void {
    if (this.pageIndex === 0) return;
    this.pageIndex--;
    this.pageToken$.next(this.pageTokens[this.pageIndex]);
  }
}
