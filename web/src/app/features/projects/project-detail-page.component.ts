import { CommonModule } from '@angular/common';
import { Component, inject } from '@angular/core';
import { ActivatedRoute } from '@angular/router';
import { combineLatest, map, switchMap } from 'rxjs';

import { ApiClient } from '../../api/client';
import { ApiError } from '../../api/models';
import { ConditionsTimelineComponent } from '../../shared/conditions-timeline.component';
import { KeyValueListComponent } from '../../shared/key-value-list.component';
import { PageHeaderComponent } from '../../shared/page-header.component';
import { StatusBadgeComponent } from '../../shared/status-badge.component';

@Component({
  selector: 'app-project-detail-page',
  standalone: true,
  imports: [CommonModule, PageHeaderComponent, StatusBadgeComponent, KeyValueListComponent, ConditionsTimelineComponent],
  template: `
    <ng-container *ngIf="vm$ | async as vm">
      <ng-container *ngIf="vm.project as project">
      <app-page-header [title]="project.spec.displayName || project.name" [description]="project.spec.description || 'Project detail'" />
      <div class="mb-4"><app-status-badge [status]="project.status?.phase || 'Pending'" /></div>
      <div class="mb-4 flex items-center gap-3">
        <button type="button" class="rounded border border-rose-300 px-3 py-2 text-sm font-medium text-rose-700 disabled:opacity-40" [disabled]="purgingCache" (click)="purgeCache(project)">{{ purgingCache ? 'Purging…' : 'Purge dependency cache' }}</button>
        <span class="text-xs text-emerald-700" *ngIf="cacheMessage">{{ cacheMessage }}</span>
      </div>
      <app-key-value-list [items]="[
        { key: 'Name', value: project.name },
        { key: 'Control namespace', value: project.namespace },
        { key: 'Runner namespace', value: project.spec.namespace },
        { key: 'Owner team', value: project.spec.ownerTeam },
        { key: 'Default registry', value: project.spec.defaultRegistry },
        { key: 'Registry credentials', value: project.spec.registry?.credentialSecretRef?.name ? 'Configured' : 'Not configured' },
        { key: 'ServiceAccount', value: project.spec.serviceAccountName || 'cloudivision-runner' },
        { key: 'Concurrent builds', value: quotaNumber(project.spec.quotas?.maxConcurrentBuildRuns, 'Unlimited') },
        { key: 'Queued builds', value: quotaNumber(project.spec.quotas?.maxQueuedBuildRuns, 'Unlimited') },
        { key: 'Maximum CPU / memory', value: (project.spec.quotas?.maxCPU || 'Unbounded') + ' / ' + (project.spec.quotas?.maxMemory || 'Unbounded') },
        { key: 'Maximum duration', value: quotaNumber(project.spec.quotas?.maxBuildDurationSeconds, 'Unbounded', 's') },
        { key: 'Artifact / log size', value: (project.spec.quotas?.maxArtifactsSize || 'Unbounded') + ' / ' + (project.spec.quotas?.maxLogSize || 'Unbounded') }
      ]" />
      <h2 class="mt-6 mb-3 text-sm font-semibold">Conditions</h2>
      <app-conditions-timeline [conditions]="project.status?.conditions || []" />
      <section class="mt-6 rounded-md border border-slate-200 bg-white p-4">
        <h2 class="text-sm font-semibold">Notification settings</h2>
        <app-key-value-list class="mt-3 block" [items]="[
          { key: 'Enabled', value: project.spec.notifications?.enabled ? 'Yes' : 'No' },
          { key: 'Provider', value: project.spec.notifications?.provider || 'Not configured' },
          { key: 'Events', value: project.spec.notifications?.events?.join(', ') || 'All events' },
          { key: 'Secret', value: project.spec.notifications?.secretRef?.name ? 'Configured' : 'Not configured' },
          { key: 'Provider health', value: vm.notificationHealth }
        ]" />
        <p class="mt-2 text-xs text-slate-500">Endpoint URLs and tokens are never returned by the API.</p>
      </section>
      </ng-container>
    </ng-container>
  `
})
export class ProjectDetailPageComponent {
  private readonly api = inject(ApiClient);
  private readonly route = inject(ActivatedRoute);
  purgingCache = false;
  cacheMessage = '';
  readonly project$ = this.route.paramMap.pipe(
    switchMap((params) => this.api.project(params.get('name') || '', params.get('namespace') || undefined))
  );
  readonly vm$ = combineLatest([this.project$, this.api.providerHealth()]).pipe(map(([project, health]) => ({
    project,
    notificationHealth: health.find((item) => item.type === 'notifications' && item.name === project.spec.notifications?.provider)?.health.message || 'Not configured'
  })));

  quotaNumber(value: number | undefined, fallback: string, suffix = ''): string {
    return value ? `${value}${suffix}` : fallback;
  }

  purgeCache(project: { name: string; namespace: string }): void {
    this.purgingCache = true;
    this.cacheMessage = '';
    this.api.purgeProjectCache(project.name, project.namespace).subscribe({
      next: () => { this.purgingCache = false; this.cacheMessage = 'Cache purged.'; },
      error: (error: ApiError) => { this.purgingCache = false; this.cacheMessage = `${error.code}: ${error.message}`; }
    });
  }
}
