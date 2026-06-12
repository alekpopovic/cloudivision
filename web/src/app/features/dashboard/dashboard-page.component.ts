import { CommonModule } from '@angular/common';
import { Component, inject } from '@angular/core';
import { RouterLink } from '@angular/router';
import { combineLatest, map, startWith } from 'rxjs';

import { ApiClient } from '../../api/client';
import { EmptyStateComponent } from '../../shared/empty-state.component';
import { ErrorMessageComponent } from '../../shared/error-message.component';
import { PageHeaderComponent } from '../../shared/page-header.component';
import { StatusBadgeComponent } from '../../shared/status-badge.component';

@Component({
  selector: 'app-dashboard-page',
  standalone: true,
  imports: [CommonModule, RouterLink, PageHeaderComponent, StatusBadgeComponent, EmptyStateComponent, ErrorMessageComponent],
  template: `
    <app-page-header title="Dashboard" description="Current CI/CD activity across projects, builds and releases." />
    <ng-container *ngIf="vm$ | async as vm">
      <app-error-message [error]="vm.error" />
      <section class="grid gap-4 md:grid-cols-4">
        <div class="rounded-md border border-slate-200 bg-white p-4" *ngFor="let card of vm.cards">
          <p class="text-xs font-medium uppercase text-slate-500">{{ card.label }}</p>
          <p class="mt-2 text-3xl font-semibold">{{ card.value }}</p>
        </div>
      </section>
      <section class="mt-6 grid gap-6 lg:grid-cols-2">
        <div class="rounded-md border border-slate-200 bg-white">
          <div class="border-b border-slate-200 px-4 py-3 font-medium">Latest BuildRuns</div>
          <div class="divide-y divide-slate-100" *ngIf="vm.buildRuns.length; else noBuilds">
            <a *ngFor="let run of vm.buildRuns" [routerLink]="['/build-runs', run.namespace, run.name]" class="flex items-center justify-between px-4 py-3 hover:bg-slate-50">
              <span>
                <span class="block text-sm font-medium">{{ run.name }}</span>
                <span class="text-xs text-slate-500">{{ run.spec.projectRef }} / {{ run.spec.repositoryRef }}</span>
              </span>
              <app-status-badge [status]="run.status?.phase || 'Pending'" />
            </a>
          </div>
          <ng-template #noBuilds><app-empty-state title="No builds" message="BuildRuns will appear here." /></ng-template>
        </div>
        <div class="rounded-md border border-slate-200 bg-white">
          <div class="border-b border-slate-200 px-4 py-3 font-medium">Releases In Progress</div>
          <div class="divide-y divide-slate-100" *ngIf="vm.releases.length; else noReleases">
            <div *ngFor="let release of vm.releases" class="flex items-center justify-between px-4 py-3">
              <span>
                <span class="block text-sm font-medium">{{ release.name }}</span>
                <span class="text-xs text-slate-500">{{ release.spec.environmentRef || '-' }}</span>
              </span>
              <app-status-badge [status]="release.status?.phase || 'Pending'" />
            </div>
          </div>
          <ng-template #noReleases><app-empty-state title="No active releases" message="Deploying releases will appear here." /></ng-template>
        </div>
      </section>
    </ng-container>
  `
})
export class DashboardPageComponent {
  private readonly api = inject(ApiClient);
  readonly vm$ = combineLatest({
    projects: this.api.projects().pipe(startWith([])),
    buildRuns: this.api.buildRuns().pipe(startWith([])),
    releases: this.api.releases().pipe(startWith([]))
  }).pipe(
    map(({ projects, buildRuns, releases }) => ({
      error: null,
      cards: [
        { label: 'Projects', value: projects.length },
        { label: 'Latest BuildRuns', value: buildRuns.length },
        { label: 'Failed Builds', value: buildRuns.filter((run) => run.status?.phase === 'Failed').length },
        { label: 'Releases In Progress', value: releases.filter((release) => release.status?.phase === 'Deploying').length }
      ],
      buildRuns: buildRuns.slice(0, 6),
      releases: releases.filter((release) => release.status?.phase === 'Deploying').slice(0, 6)
    }))
  );
}
