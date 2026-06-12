import { CommonModule } from '@angular/common';
import { Component, inject } from '@angular/core';
import { ActivatedRoute } from '@angular/router';
import { switchMap } from 'rxjs';

import { ApiClient } from '../../api/client';
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
      <app-key-value-list [items]="[
        { key: 'Name', value: project.name },
        { key: 'Control namespace', value: project.namespace },
        { key: 'Runner namespace', value: project.spec.namespace },
        { key: 'Owner team', value: project.spec.ownerTeam },
        { key: 'Default registry', value: project.spec.defaultRegistry },
        { key: 'ServiceAccount', value: project.spec.serviceAccountName || 'cloudivision-runner' }
      ]" />
      <h2 class="mt-6 mb-3 text-sm font-semibold">Conditions</h2>
      <app-conditions-timeline [conditions]="project.status?.conditions || []" />
    </ng-container>
  `
})
export class ProjectDetailPageComponent {
  private readonly api = inject(ApiClient);
  private readonly route = inject(ActivatedRoute);
  readonly project$ = this.route.paramMap.pipe(
    switchMap((params) => this.api.project(params.get('name') || '', params.get('namespace') || undefined))
  );
}
