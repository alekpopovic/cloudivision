import { CommonModule } from '@angular/common';
import { Component, inject } from '@angular/core';
import { FormArray, FormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { catchError, of, startWith, Subject, switchMap } from 'rxjs';

import { ApiClient } from '../../api/client';
import { ApiError, CatalogPipelineTemplate } from '../../api/models';
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
    <section class="mb-6 rounded-md border border-slate-200 bg-white p-4">
      <h2 class="text-sm font-semibold">Built-in catalog</h2>
      <div class="mt-3 grid gap-3 md:grid-cols-2 xl:grid-cols-4" *ngIf="catalog$ | async as catalog">
        <article *ngFor="let item of catalog" class="rounded border border-slate-200 p-3">
          <div class="flex justify-between gap-2"><strong class="text-sm">{{ item.name }}</strong><span class="text-xs text-slate-500">v{{ item.version }}</span></div>
          <p class="mt-1 text-xs text-slate-600">{{ item.description }}</p>
          <div class="mt-3 flex gap-2"><button type="button" class="rounded border border-slate-300 px-2 py-1 text-xs" (click)="importCatalog(item)">Edit</button><button type="button" class="rounded bg-blue-700 px-2 py-1 text-xs text-white" (click)="installCatalog(item)">Install</button></div>
        </article>
      </div>
    </section>
    <section class="grid gap-6 xl:grid-cols-[1fr_30rem]">
      <div class="rounded-md border border-slate-200 bg-white">
        <div class="border-b border-slate-200 px-4 py-3 font-medium">Templates</div>
        <div *ngIf="templates$ | async as templates">
          <div *ngIf="templates.length; else empty" class="divide-y divide-slate-100">
            <div *ngFor="let template of templates" class="px-4 py-3">
              <div class="flex items-center justify-between">
                <p class="text-sm font-medium">{{ template.name }}</p>
                <div class="flex items-center gap-2"><button type="button" class="rounded border px-2 py-1 text-xs" (click)="editTemplate(template)">Edit</button><app-status-badge [status]="template.status?.phase || 'Ready'" /></div>
              </div>
              <p class="mt-1 text-xs text-slate-500">{{ template.spec.steps?.length || 0 }} steps / {{ template.spec.build?.builder || 'none' }}</p>
              <p class="mt-1 text-xs" [class.text-emerald-700]="template.spec.cache?.enabled" [class.text-slate-500]="!template.spec.cache?.enabled">Cache: {{ template.spec.cache?.enabled ? (template.spec.cache?.mode || 'configured') : 'disabled' }}</p>
            </div>
          </div>
          <ng-template #empty><app-empty-state title="No templates" message="Create a template with steps and build settings." /></ng-template>
        </div>
      </div>
      <form [formGroup]="form" (ngSubmit)="save()" class="rounded-md border border-slate-200 bg-white p-4">
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
              <input class="mt-2 w-full rounded-md border border-slate-300 px-3 py-2 text-sm" placeholder="args" formControlName="args" />
              <input class="mt-2 w-full rounded-md border border-slate-300 px-3 py-2 text-sm" placeholder="working directory" formControlName="workingDir" />
              <textarea class="mt-2 w-full rounded-md border border-slate-300 px-3 py-2 text-sm" placeholder="Environment, one KEY=value per line" formControlName="env"></textarea>
              <div class="mt-2 grid grid-cols-2 gap-2"><label class="text-xs">Timeout (seconds)<input type="number" min="1" class="mt-1 w-full rounded border px-2 py-1" formControlName="timeoutSeconds" /></label><label class="flex items-end gap-2 pb-1 text-xs"><input type="checkbox" formControlName="continueOnError" /> Continue on error</label></div>
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
        <label class="mt-3 block text-sm">Image repository<input class="mt-1 w-full rounded-md border border-slate-300 px-3 py-2" formControlName="buildImage" /></label>
        <label class="mt-3 flex items-center gap-2 text-sm"><input type="checkbox" formControlName="push" /> Push image</label>
        <textarea class="mt-3 w-full rounded-md border border-slate-300 px-3 py-2 text-sm" placeholder="Build args, one KEY=value per line" formControlName="buildArgs"></textarea>
        <label class="mt-3 flex items-center gap-2 text-sm"><input type="checkbox" formControlName="allowPrivileged" /> Allow privileged workload</label>
        <p *ngIf="form.controls.allowPrivileged.value" class="mt-2 rounded bg-amber-50 p-2 text-xs text-amber-800">Privileged execution is unsafe and is normally denied by backend policy.</p>
				<p *ngIf="form.invalid || !hasExecutableWork" class="mt-3 text-xs text-rose-700">Name, executable work, and every step name, image and command are required.</p>
        <p *ngIf="duplicateStepNames.length" class="mt-2 text-xs text-rose-700">Duplicate step names: {{ duplicateStepNames.join(', ') }}</p>
				<div class="mt-4 flex gap-2"><button class="rounded-md bg-emerald-700 px-4 py-2 text-sm font-medium text-white disabled:opacity-50" [disabled]="!canSave">{{ editingName ? 'Save changes' : 'Create' }}</button><button *ngIf="editingName" type="button" class="rounded-md border border-slate-300 px-4 py-2 text-sm" (click)="cancelEdit()">Cancel</button></div>
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
  editingName = '';
  editingNamespace = '';
  readonly templates$ = this.refresh$.pipe(
    startWith(undefined),
    switchMap(() => this.api.pipelineTemplates().pipe(catchError((error: ApiError) => { this.error = error; return of([]); })))
  );
  readonly catalog$ = this.api.catalogPipelineTemplates().pipe(catchError((error: ApiError) => { this.error = error; return of([]); }));
  readonly form = this.fb.nonNullable.group({
    name: ['', Validators.required],
    projectRef: [''],
    steps: this.fb.array([this.stepGroup()]),
    buildEnabled: [true],
    builder: ['buildkit' as 'buildkit' | 'buildah' | 'none'],
    dockerfile: ['Dockerfile'],
    contextDir: ['.'],
    buildImage: [''],
    push: [true],
    buildArgs: [''],
    allowPrivileged: [false]
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
		lines.push('  build:', `    enabled: ${value.buildEnabled}`, `    builder: ${value.builder}`, `    dockerfile: ${value.dockerfile}`, `    contextDir: ${value.contextDir}`, `    image: ${value.buildImage}`, `    push: ${value.push}`);
		return lines.join('\n');
	}

  get duplicateStepNames(): string[] {
    const names = this.steps.controls.map((control) => String(control.get('name')?.value || '')).filter(Boolean);
    return [...new Set(names.filter((name, index) => names.indexOf(name) !== index))];
  }

  get hasExecutableWork(): boolean { return this.steps.length > 0 || this.form.controls.buildEnabled.value; }
  get canSave(): boolean { return this.form.valid && this.hasExecutableWork && this.duplicateStepNames.length === 0; }

  save(): void {
	  if (!this.canSave) return;
    const value = this.form.getRawValue();
	  const spec = {
		projectRef: value.projectRef || undefined,
		steps: value.steps.map((step) => ({ name: step.name, image: step.image, command: this.words(step.command), args: this.words(step.args), workingDir: step.workingDir || undefined, env: this.keyValues(step.env).map(([name, value]) => ({ name, value })), timeoutSeconds: step.timeoutSeconds || undefined, continueOnError: step.continueOnError })),
		build: { enabled: value.buildEnabled, builder: value.builder, dockerfile: value.dockerfile, contextDir: value.contextDir, image: value.buildImage || undefined, push: value.push, buildArgs: Object.fromEntries(this.keyValues(value.buildArgs)) },
		security: { runAsNonRoot: true, allowPrivileged: value.allowPrivileged, readOnlyRootFilesystem: false }
	  };
    const request$ = this.editingName ? this.api.updatePipelineTemplate(this.editingNamespace, this.editingName, spec) : this.api.createPipelineTemplate({
      name: value.name,
		spec
	  });
    request$.subscribe({ next: () => { this.cancelEdit(); this.refresh$.next(); }, error: (error: ApiError) => (this.error = error) });
  }

  importCatalog(item: CatalogPipelineTemplate): void {
    while (this.steps.length) this.steps.removeAt(0);
    for (const step of item.spec.steps || []) this.steps.push(this.stepGroup(step));
    this.form.patchValue({ name: item.name, projectRef: item.spec.projectRef || '', buildEnabled: item.spec.build?.enabled ?? false, builder: item.spec.build?.builder || 'none', dockerfile: item.spec.build?.dockerfile || 'Dockerfile', contextDir: item.spec.build?.contextDir || '.', buildImage: item.spec.build?.image || '', push: item.spec.build?.push ?? true, buildArgs: this.objectLines(item.spec.build?.buildArgs), allowPrivileged: item.spec.security?.['allowPrivileged'] === true });
  }

  installCatalog(item: CatalogPipelineTemplate): void {
    this.error = null;
    const projectRef = this.form.controls.projectRef.value || undefined;
    this.api.installCatalogPipelineTemplate(item.name, { projectRef }).subscribe({ next: () => this.refresh$.next(), error: (error: ApiError) => this.error = error });
  }

  editTemplate(template: import('../../api/models').PipelineTemplate): void {
    this.editingName = template.name; this.editingNamespace = template.namespace;
    while (this.steps.length) this.steps.removeAt(0);
    for (const step of template.spec.steps || []) this.steps.push(this.stepGroup(step));
    this.form.patchValue({ name: template.name, projectRef: template.spec.projectRef || '', buildEnabled: template.spec.build?.enabled ?? false, builder: template.spec.build?.builder || 'none', dockerfile: template.spec.build?.dockerfile || 'Dockerfile', contextDir: template.spec.build?.contextDir || '.', buildImage: template.spec.build?.image || '', push: template.spec.build?.push ?? true, buildArgs: this.objectLines(template.spec.build?.buildArgs), allowPrivileged: template.spec.security?.['allowPrivileged'] === true });
  }

  cancelEdit(): void {
    this.editingName = ''; this.editingNamespace = '';
    while (this.steps.length) this.steps.removeAt(0); this.steps.push(this.stepGroup());
    this.form.reset({ name: '', projectRef: '', buildEnabled: true, builder: 'buildkit', dockerfile: 'Dockerfile', contextDir: '.', buildImage: '', push: true, buildArgs: '', allowPrivileged: false });
  }

  private stepGroup(step?: { name: string; image: string; command?: string[]; args?: string[]; workingDir?: string; env?: Array<{name: string; value?: string}>; timeoutSeconds?: number; continueOnError?: boolean }) {
    return this.fb.nonNullable.group({
	  name: [step?.name || '', Validators.required],
	  image: [step?.image || '', Validators.required],
	  command: [(step?.command || []).join(' '), Validators.required],
      args: [(step?.args || []).join(' ')], workingDir: [step?.workingDir || ''], env: [(step?.env || []).map((entry) => `${entry.name}=${entry.value || ''}`).join('\n')], timeoutSeconds: [step?.timeoutSeconds || 600, [Validators.required, Validators.min(1)]], continueOnError: [step?.continueOnError || false]
    });
  }

  private words(value: string): string[] { return value.trim().split(/\s+/).filter(Boolean); }
  private keyValues(value: string): Array<[string, string]> { return value.split('\n').map((line) => line.trim()).filter(Boolean).map((line) => { const at = line.indexOf('='); return at > 0 ? [line.slice(0, at), line.slice(at + 1)] : [line, '']; }); }
  private objectLines(value: Record<string, string> | undefined): string { return Object.entries(value || {}).map(([key, item]) => `${key}=${item}`).join('\n'); }
}
