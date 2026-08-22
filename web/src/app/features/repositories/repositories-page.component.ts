import { CommonModule } from '@angular/common';
import { Component, inject } from '@angular/core';
import { FormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { RouterLink } from '@angular/router';
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
  imports: [CommonModule, ReactiveFormsModule, RouterLink, PageHeaderComponent, StatusBadgeComponent, EmptyStateComponent, ErrorMessageComponent],
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
							<p class="mt-1 text-xs text-slate-500">Webhook: {{ repository.status?.lastWebhookAt ? 'verified at ' + repository.status?.lastWebhookAt : 'not verified yet' }}</p>
							<p class="mt-1 text-xs text-slate-500">Filters: {{ filterSummary(repository) }}</p>
            </div>
          </div>
          <ng-template #empty><app-empty-state title="No repositories" message="Create a repository to receive Git events." /></ng-template>
        </div>
      </div>
			<div class="rounded-md border border-slate-200 bg-white p-4">
			<div class="mb-4 flex items-center gap-2 text-xs"><span class="rounded-full bg-blue-700 px-2 py-1 text-white">{{ wizardStep }}</span><span>Repository onboarding</span></div>
      <form *ngIf="wizardStep === 1" [formGroup]="form" (ngSubmit)="create()">
        <h2 class="text-sm font-semibold">Connect source repository</h2>
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
				<label class="mt-3 block text-sm">Included branches<input class="mt-1 w-full rounded-md border border-slate-300 px-3 py-2" formControlName="branchInclude" placeholder="main, release/*" /><span class="mt-1 block text-xs text-slate-500">Comma-separated names or glob patterns.</span></label>
				<label class="mt-3 block text-sm">Excluded branches<input class="mt-1 w-full rounded-md border border-slate-300 px-3 py-2" formControlName="branchExclude" placeholder="wip/*, dependabot/*" /></label>
				<label class="mt-3 block text-sm">Pipeline template
					<select class="mt-1 w-full rounded-md border border-slate-300 px-3 py-2" formControlName="pipelineTemplateRef">
						<option value="">Select a template</option><option *ngFor="let template of templates$ | async" [value]="template.name">{{ template.name }}</option>
					</select>
				</label>
				<a routerLink="/pipeline-templates" class="mt-2 inline-block text-xs font-medium text-blue-700 hover:underline">Create a PipelineTemplate first</a>
        <button class="mt-4 block rounded-md bg-emerald-700 px-4 py-2 text-sm font-medium text-white disabled:opacity-50" [disabled]="form.invalid">Continue</button>
      </form>
			<section *ngIf="wizardStep === 2 && createdRepository" class="space-y-3 text-sm">
				<h2 class="font-semibold">Configure webhook secret</h2>
				<p>Add this URL to the source provider's webhook settings:</p>
				<code class="block break-all rounded bg-slate-100 p-3 text-xs">{{ webhookUrl(createdRepository) }}</code>
				<p>Create a random webhook secret, store it in the Kubernetes Secret referenced by the Repository, and configure the same value at the provider. Never paste the secret into build parameters.</p>
				<p class="rounded border border-amber-200 bg-amber-50 p-3 text-xs text-amber-900">Verification becomes available after the first signed webhook. Current status: {{ createdRepository.status?.lastWebhookAt ? 'verified' : 'not verified' }}.</p>
				<button type="button" class="rounded border border-slate-300 px-3 py-2" (click)="resetWizard()">Add another repository</button>
			</section>
			</div>
    </section>
  `
})
export class RepositoriesPageComponent {
  private readonly api = inject(ApiClient);
  private readonly fb = inject(FormBuilder);
  private readonly refresh$ = new Subject<void>();
  error: ApiError | null = null;
	wizardStep = 1;
	createdRepository: Repository | null = null;
  apiBase = '';
  readonly repositories$ = combineLatest([
    this.refresh$.pipe(startWith(undefined), switchMap(() => this.api.repositories().pipe(catchError((error: ApiError) => { this.error = error; return of([]); })))),
    this.api.webhookUrl('github', '')
  ]).pipe(map(([repositories, base]) => { this.apiBase = base.replace(/\/github\/$/, ''); return repositories; }));
	readonly templates$ = this.api.pipelineTemplates().pipe(catchError(() => of([])));
  readonly form = this.fb.nonNullable.group({
    name: ['', Validators.required],
    projectRef: ['', Validators.required],
    provider: ['github' as Repository['spec']['provider'], Validators.required],
    url: ['', Validators.required],
    defaultBranch: ['main', Validators.required],
    branchInclude: ['main'],
    branchExclude: [''],
    pipelineTemplateRef: ['', Validators.required]
  });

  webhookUrl(repository: Repository): string {
    return `${this.apiBase}/${repository.spec.provider}/${repository.name}`;
  }

	filterSummary(repository: Repository): string {
		const webhook = repository.spec.webhook;
		const include = webhook?.branchFilters?.include?.join(', ') || repository.spec.defaultBranch;
		const exclude = webhook?.branchFilters?.exclude?.join(', ') || 'none';
		const tags = webhook?.tagFilters?.include?.join(', ') || 'disabled';
		const pullRequests = webhook?.pullRequest?.enabled ? `enabled (${webhook.pullRequest.events?.join(', ') || 'opened, reopened, synchronize'})` : 'disabled';
		return `branches ${include}; excludes ${exclude}; tags ${tags}; PRs ${pullRequests}`;
	}

	private patterns(value: string): string[] {
		return value.split(',').map((item) => item.trim()).filter(Boolean);
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
				webhook: {
					enabled: true,
					events: ['push'],
					branchFilters: { include: this.patterns(value.branchInclude), exclude: this.patterns(value.branchExclude) },
					pullRequest: { enabled: false, buildForks: false, requireTrustedActor: false }
				}
      }
		}).subscribe({ next: (repository) => { this.createdRepository = repository; this.wizardStep = 2; this.refresh$.next(); }, error: (error: ApiError) => (this.error = error) });
  }

	resetWizard(): void {
		this.createdRepository = null;
		this.wizardStep = 1;
		this.form.reset({ provider: 'github', defaultBranch: 'main', branchInclude: 'main', branchExclude: '' });
	}
}
