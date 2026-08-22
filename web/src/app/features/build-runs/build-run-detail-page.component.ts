import { CommonModule } from '@angular/common';
import { Component, inject } from '@angular/core';
import { ActivatedRoute, RouterLink } from '@angular/router';
import { EMPTY, catchError, combineLatest, map, of, switchMap, timer } from 'rxjs';

import { ApiClient } from '../../api/client';
import { ApiError, BuildRun, PipelineTemplate, Release } from '../../api/models';
import { ConditionsTimelineComponent } from '../../shared/conditions-timeline.component';
import { ErrorMessageComponent } from '../../shared/error-message.component';
import { KeyValueListComponent } from '../../shared/key-value-list.component';
import { LogsViewerComponent } from '../../shared/logs-viewer.component';
import { PageHeaderComponent } from '../../shared/page-header.component';
import { PolicyDecisionComponent } from '../../shared/policy-decision.component';
import { StatusBadgeComponent } from '../../shared/status-badge.component';

@Component({
  selector: 'app-build-run-detail-page',
  standalone: true,
  imports: [CommonModule, RouterLink, PageHeaderComponent, StatusBadgeComponent, KeyValueListComponent, ConditionsTimelineComponent, LogsViewerComponent, ErrorMessageComponent, PolicyDecisionComponent],
  template: `
    <app-error-message [error]="error" />
    <ng-container *ngIf="vm$ | async as vm">
      <app-page-header [title]="vm.run.name" description="BuildRun timeline, failure evidence, artifact metadata and logs." />
      <div class="mb-4 flex flex-wrap items-center gap-3">
        <app-status-badge [status]="vm.run.status?.phase || 'Pending'" />
        <span class="text-xs text-slate-500">Total duration: {{ duration(vm.run) }}</span>
        <button type="button" class="rounded-md border border-rose-300 px-3 py-1.5 text-xs font-medium text-rose-700 disabled:opacity-40" [disabled]="vm.run.status?.phase !== 'Failed' || actionInFlight" (click)="rerun(vm.run, 'retry')">Retry failed build</button>
        <button type="button" class="rounded-md border border-slate-300 px-3 py-1.5 text-xs font-medium disabled:opacity-40" [disabled]="actionInFlight" (click)="rerun(vm.run, 'rerun')">Rerun same params</button>
      </div>

      <section *ngIf="vm.run.status?.failure as failure" class="mb-5 rounded-md border border-rose-200 bg-rose-50 p-4 text-sm text-rose-900">
        <p class="font-semibold">{{ failure.reason || 'Build failed' }}</p>
        <p class="mt-1 whitespace-pre-wrap">{{ failure.message || 'No failure message was reported.' }}</p>
      </section>
      <app-policy-decision *ngIf="vm.run.status?.policy" class="mb-5 block" [decision]="vm.run.status?.policy" />

      <nav class="mb-4 flex gap-2 border-b border-slate-200" aria-label="BuildRun detail tabs">
        <button class="border-b-2 px-3 py-2 text-sm font-medium" [class.border-blue-600]="activeTab === 'details'" [class.text-blue-700]="activeTab === 'details'" (click)="activeTab = 'details'">Debug details</button>
        <button class="border-b-2 px-3 py-2 text-sm font-medium" [class.border-blue-600]="activeTab === 'supply-chain'" [class.text-blue-700]="activeTab === 'supply-chain'" (click)="activeTab = 'supply-chain'">Supply Chain</button>
      </nav>

      <ng-container *ngIf="activeTab === 'details'">
        <div class="grid gap-5 lg:grid-cols-2">
          <section class="rounded-md border border-slate-200 bg-white p-4">
            <h2 class="mb-3 text-sm font-semibold">Commit and artifact</h2>
            <app-key-value-list [items]="[
              { key: 'Project', value: vm.run.spec.projectRef },
              { key: 'Repository', value: vm.run.spec.repositoryRef },
              { key: 'Pipeline Template', value: vm.run.spec.pipelineTemplateRef },
              { key: 'Revision', value: vm.run.spec.revision },
              { key: 'Commit SHA', value: vm.run.spec.commitSHA || 'Not reported' },
              { key: 'Branch', value: vm.run.spec.branch || 'Not reported' },
              { key: 'Triggered by', value: (vm.run.spec.triggeredBy.actor || '-') + ' via ' + vm.run.spec.triggeredBy.type },
              { key: 'Image', value: image(vm.run) },
              { key: 'Image digest', value: vm.run.status?.image?.digest || vm.run.spec.image.digest || 'Not recorded' }
            ]" />
          </section>
          <section class="rounded-md border border-slate-200 bg-white p-4">
            <h2 class="mb-3 text-sm font-semibold">Related Releases</h2>
            <div *ngIf="vm.releases.length; else noReleases" class="space-y-2">
              <a *ngFor="let release of vm.releases" [routerLink]="['/releases', release.namespace, release.name]" class="flex items-center justify-between rounded border border-slate-200 px-3 py-2 hover:bg-slate-50">
                <span class="text-sm font-medium">{{ release.name }}</span><app-status-badge [status]="release.status?.phase || 'Pending'" />
              </a>
            </div>
            <ng-template #noReleases><p class="text-sm text-slate-500">No Release references this BuildRun.</p></ng-template>
          </section>
        </div>

        <section class="mt-5 rounded-md border border-slate-200 bg-white p-4">
          <h2 class="mb-3 text-sm font-semibold">Step timeline</h2>
          <div *ngIf="vm.template?.spec?.steps?.length; else noSteps" class="space-y-2">
            <div *ngFor="let step of vm.template?.spec?.steps; let index = index" class="grid gap-1 rounded border border-slate-200 px-3 py-2 md:grid-cols-[2rem_1fr_auto]">
              <span class="text-xs font-semibold text-slate-400">{{ index + 1 }}</span>
              <span><span class="block text-sm font-medium">{{ step.name }}</span><span class="text-xs text-slate-500">{{ step.image }}</span></span>
              <span class="text-xs text-slate-500">Duration: not reported<span *ngIf="step.timeoutSeconds"> · limit {{ step.timeoutSeconds }}s</span></span>
            </div>
          </div>
          <ng-template #noSteps><p class="text-sm text-slate-500">Step metadata is unavailable for this PipelineTemplate.</p></ng-template>
        </section>

        <div class="mt-5 grid gap-6 lg:grid-cols-2">
          <section class="rounded-md border border-slate-200 bg-white p-4">
            <h2 class="mb-3 text-sm font-semibold">Condition timeline</h2>
            <app-conditions-timeline [conditions]="vm.run.status?.conditions || []" />
          </section>
          <section>
            <h2 class="mb-3 text-sm font-semibold">Logs</h2>
            <app-logs-viewer [lines]="vm.logs" [loading]="logsLoading" [error]="logsError" [fileName]="vm.run.name + '.log'" />
          </section>
        </div>
      </ng-container>

      <section *ngIf="activeTab === 'supply-chain'" class="rounded-md border border-slate-200 bg-white p-4">
        <h2 class="mb-3 text-sm font-semibold">Artifact evidence</h2>
        <app-key-value-list [items]="[
          { key: 'Image digest', value: vm.run.status?.image?.digest || vm.run.spec.image.digest || 'Not recorded' },
          { key: 'SBOM path', value: vm.run.status?.supplyChain?.sbomPath || 'Not generated' },
          { key: 'SBOM digest', value: vm.run.status?.supplyChain?.sbomDigest || 'Not recorded' },
          { key: 'Scan results', value: vm.run.status?.supplyChain?.scannerResultsRef || 'Not scanned' },
          { key: 'Critical / High', value: (vm.run.status?.supplyChain?.criticalVulnerabilities || 0) + ' / ' + (vm.run.status?.supplyChain?.highVulnerabilities || 0) },
          { key: 'Signature', value: vm.run.status?.supplyChain?.signatureRef || 'Not signed' },
          { key: 'Provenance', value: vm.run.status?.supplyChain?.provenanceRef || 'Not written' },
          { key: 'Policy evidence', value: vm.run.status?.policy?.allowed ? 'Allowed' : (vm.run.status?.policy?.reason || 'Not evaluated') }
        ]" />
      </section>
    </ng-container>
  `
})
export class BuildRunDetailPageComponent {
  private readonly api = inject(ApiClient);
  private readonly route = inject(ActivatedRoute);
  activeTab: 'details' | 'supply-chain' = 'details';
  actionInFlight = false;
  error: ApiError | null = null;
  logsLoading = true;
  logsError = '';
  readonly params$ = this.route.paramMap.pipe(map((params) => ({ namespace: params.get('namespace') || '', name: params.get('name') || '' })));
  private readonly run$ = timer(0, 2000).pipe(
    switchMap(() => this.params$),
    switchMap((params) => this.api.buildRun(params.namespace, params.name)),
    catchError((error: ApiError) => { this.error = error; return EMPTY; })
  );
  private readonly logs$ = timer(0, 2000).pipe(
    switchMap(() => this.params$),
    switchMap((params) => this.api.buildRunLogs(params.namespace, params.name)),
    map((response) => { this.logsLoading = false; this.logsError = ''; return response.lines; }),
    catchError((error: ApiError) => { this.logsLoading = false; this.logsError = `${error.code}: ${error.message}`; return of([] as string[]); })
  );
  readonly vm$ = combineLatest([this.run$, this.logs$, this.api.pipelineTemplates(), this.api.releases()]).pipe(
    map(([run, logs, templates, releases]) => ({
      run,
      logs,
      template: templates.find((template) => template.name === run.spec.pipelineTemplateRef) as PipelineTemplate | undefined,
      releases: releases.filter((release) => release.spec.buildRunRef === run.name) as Release[]
    }))
  );

  image(run: BuildRun): string {
    const image = run.status?.image?.repository ? run.status.image : run.spec.image;
    return `${image.repository || ''}${image.tag ? ':' + image.tag : ''}${image.digest ? '@' + image.digest : ''}`;
  }

  duration(run: BuildRun): string {
    if (!run.status?.startedAt) return 'not started';
    const end = run.status.completedAt ? new Date(run.status.completedAt).getTime() : Date.now();
    const seconds = Math.max(0, Math.round((end - new Date(run.status.startedAt).getTime()) / 1000));
    return `${seconds}s`;
  }

  rerun(run: BuildRun, action: 'retry' | 'rerun'): void {
    this.actionInFlight = true;
    this.error = null;
    const suffix = Date.now().toString(36);
    const name = `${run.name}-${action}-${suffix}`.slice(0, 63).replace(/-$/, '');
    this.api.createBuildRun({ name, namespace: run.namespace, spec: { ...run.spec, triggeredBy: { type: 'manual', actor: 'web-rerun' } } }).subscribe({
      next: () => { this.actionInFlight = false; },
      error: (error: ApiError) => { this.actionInFlight = false; this.error = error; }
    });
  }

}
