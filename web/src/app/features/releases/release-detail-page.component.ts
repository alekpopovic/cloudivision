import { CommonModule } from '@angular/common';
import { Component, inject } from '@angular/core';
import { ActivatedRoute, RouterLink } from '@angular/router';
import { combineLatest, map } from 'rxjs';

import { ApiClient } from '../../api/client';
import { BuildRun, Release } from '../../api/models';
import { ConditionsTimelineComponent } from '../../shared/conditions-timeline.component';
import { KeyValueListComponent } from '../../shared/key-value-list.component';
import { PageHeaderComponent } from '../../shared/page-header.component';
import { PolicyDecisionComponent } from '../../shared/policy-decision.component';
import { StatusBadgeComponent } from '../../shared/status-badge.component';

@Component({
  selector: 'app-release-detail-page',
  standalone: true,
  imports: [CommonModule, RouterLink, PageHeaderComponent, StatusBadgeComponent, KeyValueListComponent, ConditionsTimelineComponent, PolicyDecisionComponent],
  template: `
    <ng-container *ngIf="vm$ | async as vm">
      <app-page-header [title]="vm.release.name" description="Promotion, approval and deployment debugging." />
      <div class="mb-5 flex items-center gap-3"><app-status-badge [status]="vm.release.status?.phase || 'Pending'" /><span class="text-xs text-slate-500">{{ vm.release.spec.promotionMode || 'direct-commit' }}</span></div>
      <section *ngIf="vm.release.status?.failure as failure" class="mb-5 rounded-md border border-rose-200 bg-rose-50 p-4 text-sm text-rose-900">
        <p class="font-semibold">{{ failure.reason }}</p><p class="mt-1">{{ failure.message }}</p>
      </section>
      <app-policy-decision *ngIf="vm.release.status?.policy" class="mb-5 block" [decision]="vm.release.status?.policy" />
      <p *ngIf="actionError" class="mb-5 rounded-md border border-rose-200 bg-rose-50 p-3 text-sm text-rose-800">{{ actionError }}</p>
      <div class="grid gap-5 lg:grid-cols-2">
        <section class="rounded-md border border-slate-200 bg-white p-4">
          <h2 class="mb-3 text-sm font-semibold">GitOps deployment</h2>
          <app-key-value-list [items]="[
            { key: 'Environment', value: vm.release.spec.environmentRef || '-' },
            { key: 'Image repository', value: vm.release.spec.image.repository },
            { key: 'Image tag', value: vm.release.spec.image.tag || 'Not recorded' },
            { key: 'Image digest', value: vm.release.spec.image.digest || 'Not recorded' },
            { key: 'Git commit', value: vm.release.status?.gitCommit || 'Not committed' },
            { key: 'Provider', value: vm.release.status?.deployment?.provider || '-' },
            { key: 'Sync', value: vm.release.status?.deployment?.syncStatus || '-' },
            { key: 'Health', value: vm.release.status?.deployment?.healthStatus || '-' },
            { key: 'Observed revision', value: vm.release.status?.deployment?.observedRevision || '-' },
            { key: 'Last observed', value: vm.release.status?.deployment?.observedAt || '-' }
          ]" />
          <div class="mt-3 flex flex-wrap gap-3 text-sm">
            <a *ngIf="commitUrl(vm.buildRun, vm.release) as url" [href]="url" target="_blank" rel="noopener noreferrer" class="font-medium text-blue-700 hover:underline">Open GitOps commit</a>
            <a *ngIf="vm.release.status?.pullRequest?.url" [href]="vm.release.status?.pullRequest?.url" target="_blank" rel="noopener noreferrer" class="font-medium text-blue-700 hover:underline">Open pull request</a>
            <a [routerLink]="['/build-runs', vm.release.namespace, vm.release.spec.buildRunRef]" class="font-medium text-blue-700 hover:underline">Open BuildRun</a>
          </div>
        </section>
        <section class="rounded-md border border-slate-200 bg-white p-4">
          <h2 class="mb-3 text-sm font-semibold">Approval history</h2>
          <p class="text-sm" *ngIf="vm.release.spec.approval?.approvedBy">Approved by <strong>{{ vm.release.spec.approval?.approvedBy }}</strong> at {{ vm.release.spec.approval?.approvedAt || 'unknown time' }}</p>
          <p class="text-sm text-rose-700" *ngIf="vm.release.spec.approval?.rejectedBy">Rejected by <strong>{{ vm.release.spec.approval?.rejectedBy }}</strong> at {{ vm.release.spec.approval?.rejectedAt || 'unknown time' }}</p>
          <p class="text-sm text-slate-500" *ngIf="!vm.release.spec.approval?.approvedBy && !vm.release.spec.approval?.rejectedBy">No approval action recorded.</p>
          <p class="mt-2 text-xs text-slate-500" *ngIf="vm.release.spec.approval?.comment">{{ vm.release.spec.approval?.comment }}</p>
          <div class="mt-4 flex gap-2">
            <button type="button" class="rounded-md bg-blue-700 px-3 py-2 text-sm text-white disabled:bg-slate-300" [disabled]="vm.release.status?.phase !== 'Deployed'" (click)="promote(vm.release)">Promote</button>
            <button type="button" class="rounded-md border border-slate-300 px-3 py-2 text-sm" (click)="rollback(vm.release)">Rollback</button>
          </div>
          <app-key-value-list class="mt-4 block" [items]="[
            { key: 'Promoted from', value: vm.release.spec.promotedFrom || '-' },
            { key: 'Rollback of', value: vm.release.spec.rollbackOf || '-' },
            { key: 'Rollback to', value: vm.release.spec.rollbackTo || '-' }
          ]" />
        </section>
      </div>
      <section class="mt-5 rounded-md border border-slate-200 bg-white p-4">
        <h2 class="mb-3 text-sm font-semibold">Deployment timeline</h2>
        <app-conditions-timeline [conditions]="vm.release.status?.conditions || []" />
      </section>
    </ng-container>
  `
})
export class ReleaseDetailPageComponent {
  private readonly api = inject(ApiClient);
  private readonly route = inject(ActivatedRoute);
  actionError = '';
  readonly vm$ = combineLatest([this.api.releases(), this.api.buildRuns(), this.route.paramMap]).pipe(
    map(([releases, runs, params]) => {
      const release = releases.find((item) => item.namespace === params.get('namespace') && item.name === params.get('name')) as Release;
      return { release, buildRun: runs.find((run) => run.name === release.spec.buildRunRef) };
    })
  );

  commitUrl(buildRun: BuildRun | undefined, release: Release): string {
    const repository = String(buildRun?.spec.gitOps?.['repoURL'] || '').replace(/\.git$/, '');
    const commit = release.status?.gitCommit;
    return repository && commit && /github\.com|gitlab\.com/.test(repository) ? `${repository}/commit/${commit}` : '';
  }

  promote(release: Release): void {
    const targetEnvironmentRef = window.prompt('Target environment name');
    if (!targetEnvironmentRef) return;
    this.actionError = '';
    this.api.promoteRelease(release.namespace, release.name, { targetEnvironmentRef }).subscribe({ next: (created) => window.location.assign(`/releases/${created.namespace}/${created.name}`), error: (error) => this.actionError = error.message });
  }

  rollback(release: Release): void {
    const targetReleaseRef = window.prompt('Previous successful release name');
    if (!targetReleaseRef) return;
    const reason = window.prompt('Rollback reason') || '';
    this.actionError = '';
    this.api.rollbackRelease(release.namespace, release.name, { targetReleaseRef, reason }).subscribe({ next: (created) => window.location.assign(`/releases/${created.namespace}/${created.name}`), error: (error) => this.actionError = error.message });
  }
}
