import { CommonModule } from '@angular/common';
import { Component, inject } from '@angular/core';
import { FormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { Subject, catchError, of, startWith, switchMap } from 'rxjs';

import { ApiClient } from '../../api/client';
import { ApiError, Release } from '../../api/models';
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
          <div class="mt-3 rounded-md border border-amber-200 bg-amber-50 px-3 py-2 text-xs text-amber-900" *ngIf="canActOnApproval(release)">
            <p class="font-medium">Approval required</p>
            <p>Development mode uses the actor entered below until production auth is configured.</p>
          </div>
          <form [formGroup]="approvalForm" class="mt-3 grid gap-2 md:grid-cols-[180px_1fr_auto_auto]" *ngIf="canActOnApproval(release)">
            <input class="rounded-md border border-slate-300 px-3 py-2 text-sm" formControlName="actor" placeholder="dev-mode actor" />
            <input class="rounded-md border border-slate-300 px-3 py-2 text-sm" formControlName="comment" placeholder="comment" />
            <button type="button" class="rounded-md bg-emerald-700 px-3 py-2 text-sm font-medium text-white disabled:opacity-50" [disabled]="approvalForm.invalid || actionInFlight === release.name" (click)="approve(release)">Approve</button>
            <button type="button" class="rounded-md border border-red-300 px-3 py-2 text-sm font-medium text-red-700 disabled:opacity-50" [disabled]="approvalForm.invalid || actionInFlight === release.name" (click)="reject(release)">Reject</button>
          </form>
          <p class="mt-2 text-xs text-slate-500" *ngIf="release.spec.approval?.approvedBy">Approved by {{ release.spec.approval?.approvedBy }}</p>
          <p class="mt-2 text-xs text-red-700" *ngIf="release.spec.approval?.rejectedBy">Rejected by {{ release.spec.approval?.rejectedBy }}</p>
        </div>
      </div>
      <ng-template #empty><app-empty-state title="No releases" message="Successful GitOps-enabled builds create Release CRs." /></ng-template>
    </div>
  `
})
export class ReleasesPageComponent {
  private readonly api = inject(ApiClient);
  private readonly fb = inject(FormBuilder);
  private readonly refresh$ = new Subject<void>();
  error: ApiError | null = null;
  actionInFlight: string | null = null;
  approvalForm = this.fb.nonNullable.group({
    actor: ['developer', Validators.required],
    comment: ['']
  });
  readonly releases$ = this.refresh$.pipe(
    startWith(undefined),
    switchMap(() => this.api.releases().pipe(catchError((error: ApiError) => { this.error = error; return of([]); })))
  );

  canActOnApproval(release: Release): boolean {
    return release.status?.phase === 'AwaitingApproval' && !release.spec.approval?.approvedBy && !release.spec.approval?.rejectedBy;
  }

  approve(release: Release): void {
    this.submitApproval(release, 'approve');
  }

  reject(release: Release): void {
    this.submitApproval(release, 'reject');
  }

  private submitApproval(release: Release, action: 'approve' | 'reject'): void {
    if (this.approvalForm.invalid) {
      return;
    }
    this.error = null;
    this.actionInFlight = release.name;
    const body = this.approvalForm.getRawValue();
    const request = action === 'approve'
      ? this.api.approveRelease(release.namespace, release.name, body)
      : this.api.rejectRelease(release.namespace, release.name, body);
    request.subscribe({
      next: () => {
        this.actionInFlight = null;
        this.refresh$.next();
      },
      error: (error: ApiError) => {
        this.actionInFlight = null;
        this.error = error;
      }
    });
  }
}
