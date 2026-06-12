import { CommonModule } from '@angular/common';
import { Component, inject } from '@angular/core';
import { FormBuilder, ReactiveFormsModule } from '@angular/forms';
import { catchError, of } from 'rxjs';

import { ApiClient } from '../../api/client';
import { ApiError } from '../../api/models';
import { EmptyStateComponent } from '../../shared/empty-state.component';
import { ErrorMessageComponent } from '../../shared/error-message.component';
import { PageHeaderComponent } from '../../shared/page-header.component';
import { StatusBadgeComponent } from '../../shared/status-badge.component';

@Component({
  selector: 'app-releases-page',
  standalone: true,
  imports: [CommonModule, ReactiveFormsModule, PageHeaderComponent, StatusBadgeComponent, EmptyStateComponent, ErrorMessageComponent],
  template: `
    <app-page-header title="Releases" description="GitOps release requests, approvals and deployment status." />
    <app-error-message [error]="error" />
    <div class="rounded-md border border-slate-200 bg-white" *ngIf="releases$ | async as releases">
      <div class="border-b border-slate-200 px-4 py-3 font-medium">Release List</div>
      <div *ngIf="releases.length; else empty" class="divide-y divide-slate-100">
        <div *ngFor="let release of releases" class="px-4 py-3">
          <div class="flex flex-wrap items-center justify-between gap-3">
            <div>
              <p class="text-sm font-medium">{{ release.name }}</p>
              <p class="text-xs text-slate-500">{{ release.spec.projectRef }} / {{ release.spec.environmentRef || '-' }} / {{ release.spec.buildRunRef }}</p>
            </div>
            <app-status-badge [status]="release.status?.phase || 'Pending'" />
          </div>
          <div class="mt-3 grid gap-2 text-xs text-slate-600 md:grid-cols-3">
            <span>Image: {{ release.spec.image.repository }}:{{ release.spec.image.tag || '-' }}</span>
            <span>Sync: {{ release.status?.deployment?.syncStatus || '-' }}</span>
            <span>Health: {{ release.status?.deployment?.healthStatus || '-' }}</span>
          </div>
          <form [formGroup]="approvalForm" class="mt-3 flex gap-2" *ngIf="release.spec.approval?.required && !release.spec.approval?.approvedBy">
            <input class="rounded-md border border-slate-300 px-3 py-2 text-sm" formControlName="approvedBy" placeholder="approver" />
            <button type="button" class="rounded-md border border-slate-300 px-3 py-2 text-sm text-slate-600">Approve</button>
          </form>
        </div>
      </div>
      <ng-template #empty><app-empty-state title="No releases" message="Successful GitOps-enabled builds create Release CRs." /></ng-template>
    </div>
  `
})
export class ReleasesPageComponent {
  private readonly api = inject(ApiClient);
  private readonly fb = inject(FormBuilder);
  error: ApiError | null = null;
  approvalForm = this.fb.nonNullable.group({ approvedBy: [''] });
  readonly releases$ = this.api.releases().pipe(catchError((error: ApiError) => { this.error = error; return of([]); }));
}
