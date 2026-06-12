import { CommonModule } from '@angular/common';
import { Component, inject } from '@angular/core';
import { ActivatedRoute } from '@angular/router';
import { combineLatest, map, switchMap, timer } from 'rxjs';

import { ApiClient } from '../../api/client';
import { ConditionsTimelineComponent } from '../../shared/conditions-timeline.component';
import { KeyValueListComponent } from '../../shared/key-value-list.component';
import { LogsViewerComponent } from '../../shared/logs-viewer.component';
import { PageHeaderComponent } from '../../shared/page-header.component';
import { StatusBadgeComponent } from '../../shared/status-badge.component';

@Component({
  selector: 'app-build-run-detail-page',
  standalone: true,
  imports: [CommonModule, PageHeaderComponent, StatusBadgeComponent, KeyValueListComponent, ConditionsTimelineComponent, LogsViewerComponent],
  template: `
    <ng-container *ngIf="vm$ | async as vm">
      <app-page-header [title]="vm.run.name" description="BuildRun metadata, conditions and logs." />
      <div class="mb-4"><app-status-badge [status]="vm.run.status?.phase || 'Pending'" /></div>
      <app-key-value-list [items]="[
        { key: 'Project', value: vm.run.spec.projectRef },
        { key: 'Repository', value: vm.run.spec.repositoryRef },
        { key: 'Pipeline Template', value: vm.run.spec.pipelineTemplateRef },
        { key: 'Revision', value: vm.run.spec.revision },
        { key: 'Image', value: image(vm.run) },
        { key: 'Failure', value: vm.run.status?.failure?.message || '' }
      ]" />
      <div class="mt-6 grid gap-6 lg:grid-cols-2">
        <section>
          <h2 class="mb-3 text-sm font-semibold">Conditions</h2>
          <app-conditions-timeline [conditions]="vm.run.status?.conditions || []" />
        </section>
        <section>
          <h2 class="mb-3 text-sm font-semibold">Logs</h2>
          <app-logs-viewer [lines]="vm.logs" />
        </section>
      </div>
    </ng-container>
  `
})
export class BuildRunDetailPageComponent {
  private readonly api = inject(ApiClient);
  private readonly route = inject(ActivatedRoute);
  readonly params$ = this.route.paramMap.pipe(map((params) => ({ namespace: params.get('namespace') || '', name: params.get('name') || '' })));
  readonly vm$ = combineLatest([
    timer(0, 2000).pipe(switchMap(() => this.params$), switchMap((params) => this.api.buildRun(params.namespace, params.name))),
    timer(0, 2000).pipe(switchMap(() => this.params$), switchMap((params) => this.api.buildRunLogs(params.namespace, params.name)))
  ]).pipe(map(([run, logs]) => ({ run, logs: logs.lines })));

  image(run: { status?: { image?: { repository?: string; tag?: string; digest?: string } }; spec: { image: { repository: string; tag?: string; digest?: string } } }): string {
    const image = run.status?.image?.repository ? run.status.image : run.spec.image;
    return `${image.repository || ''}${image.tag ? ':' + image.tag : ''}${image.digest ? '@' + image.digest : ''}`;
  }
}
