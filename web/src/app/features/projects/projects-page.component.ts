import { CommonModule } from '@angular/common';
import { Component, inject } from '@angular/core';
import { FormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { RouterLink } from '@angular/router';
import { catchError, of, startWith, switchMap, Subject } from 'rxjs';

import { ApiClient } from '../../api/client';
import { ApiError } from '../../api/models';
import { EmptyStateComponent } from '../../shared/empty-state.component';
import { ErrorMessageComponent } from '../../shared/error-message.component';
import { PageHeaderComponent } from '../../shared/page-header.component';
import { StatusBadgeComponent } from '../../shared/status-badge.component';

@Component({
  selector: 'app-projects-page',
  standalone: true,
  imports: [CommonModule, ReactiveFormsModule, RouterLink, PageHeaderComponent, StatusBadgeComponent, EmptyStateComponent, ErrorMessageComponent],
  template: `
    <app-page-header title="Projects" description="Tenant boundaries for repositories, builds and environments." />
    <app-error-message [error]="error" />
    <section class="grid gap-6 lg:grid-cols-[1fr_24rem]">
      <div class="rounded-md border border-slate-200 bg-white">
        <div class="border-b border-slate-200 px-4 py-3 font-medium">Project List</div>
        <div *ngIf="projects$ | async as projects">
          <div *ngIf="projects.length; else empty" class="divide-y divide-slate-100">
            <a *ngFor="let project of projects" [routerLink]="['/projects', project.namespace, project.name]" class="flex items-center justify-between px-4 py-3 hover:bg-slate-50">
              <span>
                <span class="block text-sm font-medium">{{ project.spec.displayName || project.name }}</span>
                <span class="text-xs text-slate-500">{{ project.spec.namespace }} / {{ project.spec.ownerTeam }}</span>
              </span>
              <app-status-badge [status]="project.status?.phase || 'Pending'" />
            </a>
          </div>
          <ng-template #empty><app-empty-state title="No projects" message="Create a project to start isolating CI/CD work." /></ng-template>
        </div>
      </div>
      <form [formGroup]="form" (ngSubmit)="create()" class="rounded-md border border-slate-200 bg-white p-4">
        <h2 class="text-sm font-semibold">Create Project</h2>
        <label class="mt-4 block text-sm">Name<input class="mt-1 w-full rounded-md border border-slate-300 px-3 py-2" formControlName="name" /></label>
        <label class="mt-3 block text-sm">Display name<input class="mt-1 w-full rounded-md border border-slate-300 px-3 py-2" formControlName="displayName" /></label>
        <label class="mt-3 block text-sm">Owner team<input class="mt-1 w-full rounded-md border border-slate-300 px-3 py-2" formControlName="ownerTeam" /></label>
        <label class="mt-3 block text-sm">Namespace<input class="mt-1 w-full rounded-md border border-slate-300 px-3 py-2" formControlName="namespace" /></label>
        <label class="mt-3 block text-sm">Default registry<input class="mt-1 w-full rounded-md border border-slate-300 px-3 py-2" formControlName="defaultRegistry" /></label>
        <label class="mt-3 block text-sm">Pod security
          <select class="mt-1 w-full rounded-md border border-slate-300 px-3 py-2" formControlName="podSecurityLevel">
            <option value="restricted">restricted</option>
            <option value="baseline">baseline</option>
          </select>
        </label>
        <button class="mt-4 rounded-md bg-emerald-700 px-4 py-2 text-sm font-medium text-white disabled:opacity-50" [disabled]="form.invalid">Create</button>
      </form>
    </section>
  `
})
export class ProjectsPageComponent {
  private readonly api = inject(ApiClient);
  private readonly fb = inject(FormBuilder);
  private readonly refresh$ = new Subject<void>();
  error: ApiError | null = null;
  readonly projects$ = this.refresh$.pipe(
    startWith(undefined),
    switchMap(() => this.api.projects().pipe(catchError((error: ApiError) => { this.error = error; return of([]); })))
  );
  readonly form = this.fb.nonNullable.group({
    name: ['', Validators.required],
    displayName: ['', Validators.required],
    ownerTeam: ['', Validators.required],
    namespace: ['', Validators.required],
    defaultRegistry: ['', Validators.required],
    podSecurityLevel: ['restricted', Validators.required]
  });

  create(): void {
    if (this.form.invalid) return;
    const value = this.form.getRawValue();
    this.api.createProject({
      name: value.name,
      spec: {
        displayName: value.displayName,
        ownerTeam: value.ownerTeam,
        namespace: value.namespace,
        defaultRegistry: value.defaultRegistry,
        defaultBranch: 'main',
        isolation: { createNamespace: true, podSecurityLevel: value.podSecurityLevel as 'baseline' | 'restricted', networkPolicyMode: 'defaultDeny' }
      }
    }).subscribe({ next: () => { this.form.reset({ podSecurityLevel: 'restricted' }); this.refresh$.next(); }, error: (error: ApiError) => (this.error = error) });
  }
}
