import { CommonModule } from '@angular/common';
import { Component, inject } from '@angular/core';
import { FormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { RouterLink } from '@angular/router';
import { switchMap } from 'rxjs';

import { ApiClient } from '../../api/client';
import { ApiError, Repository } from '../../api/models';
import { ErrorMessageComponent } from '../../shared/error-message.component';
import { PageHeaderComponent } from '../../shared/page-header.component';

@Component({
  selector: 'app-first-run-wizard',
  standalone: true,
  imports: [CommonModule, ReactiveFormsModule, RouterLink, PageHeaderComponent, ErrorMessageComponent],
  template: `
    <app-page-header title="First-run wizard" description="Create a minimal Project, Repository, PipelineTemplate and BuildRun." />
    <app-error-message [error]="error" />
    <ol class="mb-6 grid gap-2 text-xs sm:grid-cols-6">
      <li *ngFor="let label of steps; let i = index" class="rounded border border-slate-200 bg-white p-2"><span class="mr-1 font-bold text-blue-700">{{ i + 1 }}</span>{{ label }}</li>
    </ol>
    <section *ngIf="completedBuildName; else setupForm" class="rounded-md border border-emerald-200 bg-emerald-50 p-5">
      <h2 class="font-semibold text-emerald-900">Starter resources created</h2>
      <p class="mt-2 text-sm text-emerald-800">BuildRun {{ completedBuildName }} was submitted. Follow its conditions and logs next.</p>
      <a [routerLink]="['/build-runs', form.controls.namespace.value, completedBuildName]" class="mt-4 inline-block rounded bg-emerald-700 px-4 py-2 text-sm font-medium text-white">View logs</a>
    </section>
    <ng-template #setupForm>
      <form [formGroup]="form" (ngSubmit)="createStarter()" class="grid gap-5 rounded-md border border-slate-200 bg-white p-5 lg:grid-cols-2">
        <div>
          <h2 class="text-sm font-semibold">Project</h2>
          <label class="mt-3 block text-sm">Project name<input class="mt-1 w-full rounded border border-slate-300 px-3 py-2" formControlName="projectName" /></label>
          <label class="mt-3 block text-sm">Owner team<input class="mt-1 w-full rounded border border-slate-300 px-3 py-2" formControlName="ownerTeam" /></label>
          <label class="mt-3 block text-sm">Namespace<input class="mt-1 w-full rounded border border-slate-300 px-3 py-2" formControlName="namespace" /></label>
          <label class="mt-3 block text-sm">Default registry<input class="mt-1 w-full rounded border border-slate-300 px-3 py-2" formControlName="registry" /></label>
        </div>
        <div>
          <h2 class="text-sm font-semibold">Source and first build</h2>
          <label class="mt-3 block text-sm">Provider<select class="mt-1 w-full rounded border border-slate-300 px-3 py-2" formControlName="provider"><option value="github">GitHub</option><option value="gitlab">GitLab</option><option value="gitea">Gitea</option><option value="generic">Generic Git</option></select></label>
          <label class="mt-3 block text-sm">Repository URL<input class="mt-1 w-full rounded border border-slate-300 px-3 py-2" formControlName="repositoryUrl" /></label>
          <label class="mt-3 block text-sm">Default branch<input class="mt-1 w-full rounded border border-slate-300 px-3 py-2" formControlName="branch" /></label>
          <label class="mt-3 block text-sm">Image repository<input class="mt-1 w-full rounded border border-slate-300 px-3 py-2" formControlName="imageRepository" /></label>
          <label class="mt-3 flex items-center gap-2 text-sm"><input type="checkbox" formControlName="configureGitOps" /> Configure GitOps later (optional)</label>
        </div>
        <div class="lg:col-span-2">
          <p class="text-xs text-slate-500">The starter template runs a safe Alpine smoke step. You can edit build settings and configure GitOps after the first successful run.</p>
          <button class="mt-3 rounded bg-blue-700 px-4 py-2 text-sm font-medium text-white disabled:opacity-50" [disabled]="form.invalid || submitting">{{ submitting ? 'Creating…' : 'Create and trigger BuildRun' }}</button>
        </div>
      </form>
    </ng-template>
  `
})
export class FirstRunWizardComponent {
  private readonly api = inject(ApiClient);
  private readonly fb = inject(FormBuilder);
  readonly steps = ['Create Project', 'Add Repository', 'Add PipelineTemplate', 'Trigger BuildRun', 'View logs', 'Optional GitOps'];
  error: ApiError | null = null;
  submitting = false;
  completedBuildName = '';
  readonly form = this.fb.nonNullable.group({
    projectName: ['demo', Validators.required], ownerTeam: ['platform', Validators.required], namespace: ['demo-ci', Validators.required],
    registry: ['ghcr.io/example', Validators.required], provider: ['github' as Repository['spec']['provider'], Validators.required],
    repositoryUrl: ['', [Validators.required, Validators.pattern(/^https?:\/\/.+/)]], branch: ['main', Validators.required],
    imageRepository: ['ghcr.io/example/demo', Validators.required], configureGitOps: [false]
  });

  createStarter(): void {
    if (this.form.invalid) return;
    this.submitting = true;
    this.error = null;
    const value = this.form.getRawValue();
    const templateName = `${value.projectName}-starter`;
    const repositoryName = `${value.projectName}-repo`;
    const buildName = `${value.projectName}-first-build`;
    this.api.createProject({ name: value.projectName, namespace: value.namespace, spec: {
      displayName: value.projectName, ownerTeam: value.ownerTeam, namespace: value.namespace, defaultRegistry: value.registry, defaultBranch: value.branch,
      isolation: { createNamespace: true, podSecurityLevel: 'restricted', networkPolicyMode: 'defaultDeny' }
    } }).pipe(
      switchMap(() => this.api.createPipelineTemplate({ name: templateName, namespace: value.namespace, spec: {
        projectRef: value.projectName, steps: [{ name: 'smoke', image: 'alpine:3.22', command: ['sh', '-ec'], args: ['echo cloudivision starter pipeline'] }],
        build: { enabled: false, builder: 'none', push: false }, security: { runAsNonRoot: true, allowPrivileged: false }
      } })),
      switchMap(() => this.api.createRepository({ name: repositoryName, namespace: value.namespace, spec: {
        projectRef: value.projectName, provider: value.provider, url: value.repositoryUrl, defaultBranch: value.branch, pipelineTemplateRef: templateName,
        webhook: { enabled: true, events: ['push'] }
      } })),
      switchMap(() => this.api.createBuildRun({ name: buildName, namespace: value.namespace, spec: {
        projectRef: value.projectName, repositoryRef: repositoryName, pipelineTemplateRef: templateName, revision: value.branch, branch: value.branch,
        triggeredBy: { type: 'manual', actor: 'first-run-wizard' }, image: { repository: value.imageRepository, tag: 'first-run' }, executor: 'job',
        gitOps: { enabled: false }
      } }))
    ).subscribe({
      next: () => { this.submitting = false; this.completedBuildName = buildName; },
      error: (error: ApiError) => { this.submitting = false; this.error = error; }
    });
  }
}
