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
              <p class="mt-1 text-xs" [class.text-emerald-700]="template.spec.cache?.enabled" [class.text-slate-500]="!template.spec.cache?.enabled">Cache: {{ template.spec.cache?.enabled ? (template.spec.cache?.mode || 'configured') : 'disabled' }}</p>
            </div>
          </div>
          <ng-template #empty><app-empty-state title="No templates" message="Create a template with steps and build settings." /></ng-template>
        </div>
      </div>
      <form [formGroup]="form" (ngSubmit)="create()" class="rounded-md border border-slate-200 bg-white p-4">
				<div class="flex items-center justify-between"><h2 class="text-sm font-semibold">PipelineTemplate editor</h2><button type="button" class="rounded border border-slate-300 px-2 py-1 text-xs" (click)="yamlMode = !yamlMode">{{ yamlMode ? 'Visual mode' : 'YAML mode' }}</button></div>
				<div *ngIf="yamlMode" class="mt-4">
					<label class="text-sm font-medium">YAML preview</label>
					<textarea class="mt-1 h-72 w-full rounded border border-slate-300 bg-slate-950 p-3 font-mono text-xs text-slate-100" readonly [value]="yamlPreview"></textarea>
					<p class="mt-1 text-xs text-slate-500">Apply edited YAML through kubectl; browser-side YAML parsing is not configured.</p>
				</div>
				<ng-container *ngIf="!yamlMode">
        <label class="mt-4 block text-sm">Name<input class="mt-1 w-full rounded-md border border-slate-300 px-3 py-2" formControlName="name" /></label>
        <label class="mt-3 block text-sm">Project<input class="mt-1 w-full rounded-md border border-slate-300 px-3 py-2" formControlName="projectRef" /></label>
        <div class="mt-4">
          <div class="flex items-center justify-between">
            <h3 class="text-sm font-semibold">Steps</h3>
            <button type="button" class="rounded-md border border-slate-300 px-3 py-1 text-xs" (click)="addStep()">Add step</button>
          </div>
          <div formArrayName="steps" class="mt-3 space-y-3">
            <div *ngFor="let step of steps.controls; let i = index" [formGroupName]="i" class="rounded-md border border-slate-200 p-3">
						<div class="mb-2 flex justify-end gap-1"><button type="button" class="rounded border px-2 py-1 text-xs" [disabled]="i === 0" (click)="moveStep(i, -1)">↑</button><button type="button" class="rounded border px-2 py-1 text-xs" [disabled]="i === steps.length - 1" (click)="moveStep(i, 1)">↓</button><button type="button" class="rounded border border-rose-200 px-2 py-1 text-xs text-rose-700" (click)="removeStep(i)">Remove</button></div>
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
				<p *ngIf="form.invalid" class="mt-3 text-xs text-rose-700">Name and every step name, image and command are required.</p>
				<div class="mt-4 flex gap-2"><button class="rounded-md bg-emerald-700 px-4 py-2 text-sm font-medium text-white disabled:opacity-50" [disabled]="form.invalid">Create</button><button type="button" class="rounded-md border border-slate-300 px-4 py-2 text-sm text-slate-500" disabled title="Dry-run API is not available">Dry run (not available)</button></div>
				</ng-container>
      </form>
    </section>
  `
})
export class PipelineTemplatesPageComponent {
  private readonly api = inject(ApiClient);
  private readonly fb = inject(FormBuilder);
  private readonly refresh$ = new Subject<void>();
  error: ApiError | null = null;
	yamlMode = false;
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

	removeStep(index: number): void {
		this.steps.removeAt(index);
	}

	moveStep(index: number, direction: -1 | 1): void {
		const target = index + direction;
		if (target < 0 || target >= this.steps.length) return;
		const control = this.steps.at(index);
		this.steps.removeAt(index);
		this.steps.insert(target, control);
	}

	get yamlPreview(): string {
		const value = this.form.getRawValue();
		const lines = ['apiVersion: cicd.cloudivision.io/v1alpha1', 'kind: PipelineTemplate', 'metadata:', `  name: ${value.name || 'template-name'}`, 'spec:', '  steps:'];
		for (const step of value.steps) {
			lines.push(`    - name: ${step.name || 'step'}`, `      image: ${step.image || 'image'}`, `      command: [${step.command.split(' ').filter(Boolean).map((part) => JSON.stringify(part)).join(', ')}]`);
		}
		lines.push('  build:', `    enabled: ${value.buildEnabled}`, `    builder: ${value.builder}`, `    dockerfile: ${value.dockerfile}`, `    contextDir: ${value.contextDir}`);
		return lines.join('\n');
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
