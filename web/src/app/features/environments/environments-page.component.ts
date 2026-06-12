import { CommonModule } from '@angular/common';
import { Component, inject } from '@angular/core';
import { catchError, of } from 'rxjs';

import { ApiClient } from '../../api/client';
import { ApiError } from '../../api/models';
import { EmptyStateComponent } from '../../shared/empty-state.component';
import { ErrorMessageComponent } from '../../shared/error-message.component';
import { PageHeaderComponent } from '../../shared/page-header.component';
import { StatusBadgeComponent } from '../../shared/status-badge.component';

@Component({
  selector: 'app-environments-page',
  standalone: true,
  imports: [CommonModule, PageHeaderComponent, StatusBadgeComponent, EmptyStateComponent, ErrorMessageComponent],
  template: `
    <app-page-header title="Environments" description="GitOps-managed deployment targets and health signals." />
    <app-error-message [error]="error" />
    <div class="rounded-md border border-slate-200 bg-white" *ngIf="environments$ | async as environments">
      <div class="border-b border-slate-200 px-4 py-3 font-medium">Environment List</div>
      <div *ngIf="environments.length; else empty" class="divide-y divide-slate-100">
        <div *ngFor="let environment of environments" class="grid gap-3 px-4 py-3 md:grid-cols-4">
          <div>
            <p class="text-sm font-medium">{{ environment.spec.displayName || environment.name }}</p>
            <p class="text-xs text-slate-500">{{ environment.spec.projectRef }} / {{ environment.spec.type }}</p>
          </div>
          <app-status-badge [status]="environment.status?.phase || 'Pending'" />
          <app-status-badge [status]="environment.status?.syncStatus || 'Unknown'" />
          <app-status-badge [status]="environment.status?.healthStatus || 'Unknown'" />
        </div>
      </div>
      <ng-template #empty><app-empty-state title="No environments" message="Environment CRs will appear here." /></ng-template>
    </div>
  `
})
export class EnvironmentsPageComponent {
  private readonly api = inject(ApiClient);
  error: ApiError | null = null;
  readonly environments$ = this.api.environments().pipe(catchError((error: ApiError) => { this.error = error; return of([]); }));
}
