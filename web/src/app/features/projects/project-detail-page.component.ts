import { CommonModule } from '@angular/common';
import { Component, inject } from '@angular/core';
import { ActivatedRoute } from '@angular/router';
import { switchMap } from 'rxjs';

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
    <ng-container *ngIf="project$ | async as project">
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
        { key: 'ServiceAccount', value: project.spec.serviceAccountName || 'cloudivision-runner' },
        { key: 'Concurrent builds', value: quotaNumber(project.spec.quotas?.maxConcurrentBuildRuns, 'Unlimited') },
        { key: 'Queued builds', value: quotaNumber(project.spec.quotas?.maxQueuedBuildRuns, 'Unlimited') },
        { key: 'Maximum CPU / memory', value: (project.spec.quotas?.maxCPU || 'Unbounded') + ' / ' + (project.spec.quotas?.maxMemory || 'Unbounded') },
        { key: 'Maximum duration', value: quotaNumber(project.spec.quotas?.maxBuildDurationSeconds, 'Unbounded', 's') },
        { key: 'Artifact / log size', value: (project.spec.quotas?.maxArtifactsSize || 'Unbounded') + ' / ' + (project.spec.quotas?.maxLogSize || 'Unbounded') }
      ]" />
      <h2 class="mt-6 mb-3 text-sm font-semibold">Conditions</h2>
      <app-conditions-timeline [conditions]="project.status?.conditions || []" />
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
