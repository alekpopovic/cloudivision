import { CommonModule } from '@angular/common';
import { Component, inject } from '@angular/core';
import { FormArray, FormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { catchError, of, startWith, Subject, switchMap } from 'rxjs';

import { ApiClient } from '../../api/client';
import { ApiError } from '../../api/models';
import { EmptyStateComponent } from '../../shared/empty-state.component';
import { ErrorMessageComponent } from '../../shared/error-message.component';
import { PageHeaderComponent } from '../../shared/page-header.component';
import { StatusBadgeComponent } from '../../shared/status-badge.component';

@Component({
  selector: 'app-pipeline-templates-page',
  standalone: true,
  imports: [CommonModule, ReactiveFormsModule, PageHeaderComponent, StatusBadgeComponent, EmptyStateComponent, ErrorMessageComponent],
  template: `
    <app-page-header title="Pipeline Templates" description="Reusable build definitions and image settings." />
    <app-error-message [error]="error" />
    <section class="grid gap-6 xl:grid-cols-[1fr_30rem]">
      <div class="rounded-md border border-slate-200 bg-white">
        <div class="border-b border-slate-200 px-4 py-3 font-medium">Templates</div>
        <div *ngIf="templates$ | async as templates">
          <div *ngIf="templates.length; else empty" class="divide-y divide-slate-100">
            <div *ngFor="let template of templates" class="px-4 py-3">
              <div class="flex items-center justify-between">
                <p class="text-sm font-medium">{{ template.name }}</p>
                <app-status-badge [status]="template.status?.phase || 'Ready'" />
              </div>
              <p class="mt-1 text-xs text-slate-500">{{ template.spec.steps?.length || 0 }} steps / {{ template.spec.build?.builder || 'none' }}</p>
            </div>
          </div>
          <ng-template #empty><app-empty-state title="No templates" message="Create a template with steps and build settings." /></ng-template>
        </div>
      </div>
      <form [formGroup]="form" (ngSubmit)="create()" class="rounded-md border border-slate-200 bg-white p-4">
        <h2 class="text-sm font-semibold">Create Template</h2>
        <label class="mt-4 block text-sm">Name<input class="mt-1 w-full rounded-md border border-slate-300 px-3 py-2" formControlName="name" /></label>
        <label class="mt-3 block text-sm">Project<input class="mt-1 w-full rounded-md border border-slate-300 px-3 py-2" formControlName="projectRef" /></label>
        <div class="mt-4">
          <div class="flex items-center justify-between">
            <h3 class="text-sm font-semibold">Steps</h3>
            <button type="button" class="rounded-md border border-slate-300 px-3 py-1 text-xs" (click)="addStep()">Add step</button>
          </div>
          <div formArrayName="steps" class="mt-3 space-y-3">
            <div *ngFor="let step of steps.controls; let i = index" [formGroupName]="i" class="rounded-md border border-slate-200 p-3">
              <input class="w-full rounded-md border border-slate-300 px-3 py-2 text-sm" placeholder="step name" formControlName="name" />
              <input class="mt-2 w-full rounded-md border border-slate-300 px-3 py-2 text-sm" placeholder="image" formControlName="image" />
              <input class="mt-2 w-full rounded-md border border-slate-300 px-3 py-2 text-sm" placeholder="command, e.g. npm test" formControlName="command" />
            </div>
          </div>
        </div>
        <h3 class="mt-4 text-sm font-semibold">Build Settings</h3>
        <label class="mt-3 flex items-center gap-2 text-sm"><input type="checkbox" formControlName="buildEnabled" /> Build image</label>
        <label class="mt-3 block text-sm">Builder
          <select class="mt-1 w-full rounded-md border border-slate-300 px-3 py-2" formControlName="builder">
            <option value="buildkit">buildkit</option>
            <option value="buildah">buildah</option>
            <option value="none">none</option>
          </select>
        </label>
        <label class="mt-3 block text-sm">Dockerfile<input class="mt-1 w-full rounded-md border border-slate-300 px-3 py-2" formControlName="dockerfile" /></label>
        <label class="mt-3 block text-sm">Context dir<input class="mt-1 w-full rounded-md border border-slate-300 px-3 py-2" formControlName="contextDir" /></label>
        <button class="mt-4 rounded-md bg-emerald-700 px-4 py-2 text-sm font-medium text-white disabled:opacity-50" [disabled]="form.invalid">Create</button>
      </form>
    </section>
  `
})
export class PipelineTemplatesPageComponent {
  private readonly api = inject(ApiClient);
  private readonly fb = inject(FormBuilder);
  private readonly refresh$ = new Subject<void>();
  error: ApiError | null = null;
  readonly templates$ = this.refresh$.pipe(
    startWith(undefined),
    switchMap(() => this.api.pipelineTemplates().pipe(catchError((error: ApiError) => { this.error = error; return of([]); })))
  );
  readonly form = this.fb.nonNullable.group({
    name: ['', Validators.required],
    projectRef: [''],
    steps: this.fb.array([this.stepGroup()]),
    buildEnabled: [true],
    builder: ['buildkit' as 'buildkit' | 'buildah' | 'none'],
    dockerfile: ['Dockerfile'],
    contextDir: ['.']
  });

  get steps(): FormArray {
    return this.form.controls.steps;
  }

  addStep(): void {
    this.steps.push(this.stepGroup());
  }

  create(): void {
    if (this.form.invalid) return;
    const value = this.form.getRawValue();
    this.api.createPipelineTemplate({
      name: value.name,
      spec: {
        projectRef: value.projectRef || undefined,
        steps: value.steps.map((step) => ({ name: step.name, image: step.image, command: step.command.split(' ').filter(Boolean) })),
        build: { enabled: value.buildEnabled, builder: value.builder, dockerfile: value.dockerfile, contextDir: value.contextDir, push: true },
        security: { runAsNonRoot: true, allowPrivileged: false, readOnlyRootFilesystem: false }
      }
    }).subscribe({ next: () => { this.form.reset({ buildEnabled: true, builder: 'buildkit', dockerfile: 'Dockerfile', contextDir: '.' }); this.refresh$.next(); }, error: (error: ApiError) => (this.error = error) });
  }

  private stepGroup() {
    return this.fb.nonNullable.group({
      name: ['', Validators.required],
      image: ['', Validators.required],
      command: ['', Validators.required]
    });
  }
}
