import { CommonModule } from '@angular/common';
import { Component, inject } from '@angular/core';
import { FormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { RouterLink } from '@angular/router';
import { catchError, of, switchMap, tap } from 'rxjs';

import { ApiClient } from '../../api/client';
import { ApiError, CatalogPipelineTemplate, Repository } from '../../api/models';
import { ErrorMessageComponent } from '../../shared/error-message.component';
import { PageHeaderComponent } from '../../shared/page-header.component';

@Component({
  selector: 'app-first-run-wizard',
  standalone: true,
  imports: [CommonModule, ReactiveFormsModule, RouterLink, PageHeaderComponent, ErrorMessageComponent],
  template: `
    <app-page-header title="First-run wizard" description="Create a minimal Project, Repository, PipelineTemplate and BuildRun." />
    <app-error-message [error]="error" />
    <p *ngIf="emptyInstallation$ | async" class="mb-4 rounded border border-blue-200 bg-blue-50 p-3 text-sm text-blue-900">This installation is empty. Complete the guided setup to trigger its first BuildRun.</p>
    <ol class="mb-6 grid gap-2 text-xs sm:grid-cols-4 lg:grid-cols-8">
      <li *ngFor="let label of steps; let i = index" class="rounded border p-2" [class.border-blue-400]="i <= currentStage" [class.bg-blue-50]="i <= currentStage"><span class="mr-1 font-bold text-blue-700">{{ i + 1 }}</span>{{ label }}</li>
    </ol>
    <section *ngIf="completedBuildName; else setupForm" class="rounded-md border border-emerald-200 bg-emerald-50 p-5">
      <h2 class="font-semibold text-emerald-900">Starter resources created</h2>
      <p class="mt-2 text-sm text-emerald-800">BuildRun {{ completedBuildName }} was submitted. Follow its conditions and logs next.</p>
      <a [routerLink]="['/build-runs', form.controls.namespace.value, completedBuildName]" class="mt-4 inline-block rounded bg-emerald-700 px-4 py-2 text-sm font-medium text-white">View logs</a>
      <div class="mt-4 text-sm text-emerald-900"><a href="/docs/getting-started/first-release" class="underline">Next: image builds and GitOps releases</a></div>
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
          <label class="mt-3 block text-sm">Pipeline catalog<select class="mt-1 w-full rounded border border-slate-300 px-3 py-2" formControlName="catalogTemplate"><option *ngFor="let item of catalog$ | async" [value]="item.name">{{ item.name }} — {{ item.description }}</option></select></label>
          <label class="mt-3 flex items-center gap-2 text-sm"><input type="checkbox" formControlName="webhookEnabled" /> Configure webhook now (optional)</label>
          <div *ngIf="form.controls.webhookEnabled.value" class="mt-2 grid grid-cols-2 gap-2"><input class="rounded border px-2 py-1 text-sm" placeholder="Secret name" formControlName="webhookSecretName" /><input class="rounded border px-2 py-1 text-sm" placeholder="Secret key" formControlName="webhookSecretKey" /></div>
          <label class="mt-3 flex items-center gap-2 text-sm"><input type="checkbox" formControlName="configureGitOps" /> Show GitOps as a next step</label>
        </div>
        <div class="lg:col-span-2">
          <p class="text-xs text-slate-500">The selected built-in template is installed with safe defaults. Advanced image, webhook and GitOps settings can be skipped.</p>
          <pre class="mt-3 overflow-x-auto rounded bg-slate-950 p-3 text-xs text-slate-100">{{ copyableCommand }}</pre>
          <button class="mt-3 rounded bg-blue-700 px-4 py-2 text-sm font-medium text-white disabled:opacity-50" [disabled]="!canSubmit || submitting">{{ submitting ? 'Creating…' : 'Create and trigger BuildRun' }}</button>
        </div>
      </form>
    </ng-template>
  `
})
export class FirstRunWizardComponent {
  private readonly api = inject(ApiClient);
  private readonly fb = inject(FormBuilder);
  readonly steps = ['Welcome', 'Project', 'Repository', 'Template', 'Webhook', 'BuildRun', 'Logs', 'Next steps'];
  error: ApiError | null = null;
  submitting = false;
  completedBuildName = '';
  currentStage = 0;
  readonly emptyInstallation$ = this.api.projects().pipe(switchMap((projects) => of(projects.length === 0)), catchError(() => of(false)));
  readonly catalog$ = this.api.catalogPipelineTemplates().pipe(catchError((error: ApiError) => { this.error = error; return of([] as CatalogPipelineTemplate[]); }));
  readonly form = this.fb.nonNullable.group({
    projectName: ['demo', Validators.required], ownerTeam: ['platform', Validators.required], namespace: ['demo-ci', Validators.required],
    registry: ['ghcr.io/example', Validators.required], provider: ['github' as Repository['spec']['provider'], Validators.required],
    repositoryUrl: ['', [Validators.required, Validators.pattern(/^https?:\/\/.+/)]], branch: ['main', Validators.required],
    imageRepository: ['ghcr.io/example/demo', Validators.required], catalogTemplate: ['go', Validators.required], webhookEnabled: [false], webhookSecretName: [''], webhookSecretKey: ['secret'], configureGitOps: [false]
  });

  get canSubmit(): boolean { const value = this.form.getRawValue(); return this.form.valid && (!value.webhookEnabled || (!!value.webhookSecretName && !!value.webhookSecretKey)); }
  get copyableCommand(): string { return this.commandPreview.replace('\n+', '\n'); }
  get commandPreview(): string { const v = this.form.getRawValue(); return `curl -X POST "$API_URL/api/v1/catalog/pipeline-templates/${v.catalogTemplate}/install" \\\n+  -H 'Content-Type: application/json' -d '{"namespace":"${v.namespace}","projectRef":"${v.projectName}","name":"${v.projectName}-starter"}'`; }

  createStarter(): void {
    if (!this.canSubmit) return;
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
      tap(() => this.currentStage = 2),
      switchMap(() => this.api.installCatalogPipelineTemplate(value.catalogTemplate, { name: templateName, namespace: value.namespace, projectRef: value.projectName })),
      tap(() => this.currentStage = 3),
      switchMap(() => this.api.createRepository({ name: repositoryName, namespace: value.namespace, spec: {
        projectRef: value.projectName, provider: value.provider, url: value.repositoryUrl, defaultBranch: value.branch, pipelineTemplateRef: templateName,
        webhook: value.webhookEnabled ? { enabled: true, events: ['push'], secretRef: { name: value.webhookSecretName, key: value.webhookSecretKey } } : { enabled: false, events: [] }
      } })),
      tap(() => this.currentStage = 5),
      switchMap(() => this.api.createBuildRun({ name: buildName, namespace: value.namespace, spec: {
        projectRef: value.projectName, repositoryRef: repositoryName, pipelineTemplateRef: templateName, revision: value.branch, branch: value.branch,
        triggeredBy: { type: 'manual', actor: 'first-run-wizard' }, image: { repository: value.imageRepository, tag: 'first-run' }, executor: 'job',
        gitOps: { enabled: false }
      } }))
    ).subscribe({
      next: () => { this.submitting = false; this.completedBuildName = buildName; this.currentStage = 7; },
      error: (error: ApiError) => { this.submitting = false; this.error = error; }
    });
  }
}
