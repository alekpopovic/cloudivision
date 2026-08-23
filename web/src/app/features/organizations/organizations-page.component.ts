import { CommonModule } from '@angular/common';
import { Component, inject } from '@angular/core';
import { combineLatest, forkJoin, map, of, startWith, switchMap } from 'rxjs';

import { ApiClient } from '../../api/client';
import { OrganizationContext } from '../../core/organization-context.service';
import { PageHeaderComponent } from '../../shared/page-header.component';

@Component({
  selector: 'app-organizations-page',
  standalone: true,
  imports: [CommonModule, PageHeaderComponent],
  template: `
    <app-page-header title="Organization & teams" description="Memberships and project-level team access." />
    <ng-container *ngIf="view$ | async as view">
      <div *ngIf="view.organization; else empty" class="space-y-6">
        <section class="rounded-md border border-slate-200 bg-white p-5"><p class="text-xs uppercase tracking-wide text-slate-500">Organization</p><h2 class="mt-1 text-xl font-semibold">{{ view.organization.name }}</h2></section>
        <section class="grid gap-5 lg:grid-cols-2">
          <div class="rounded-md border border-slate-200 bg-white p-5"><h2 class="font-semibold">Teams</h2><ul class="mt-3 divide-y"><li *ngFor="let team of view.teams" class="py-2 text-sm">{{ team.name }} <span class="text-slate-400">{{ team.id }}</span></li><li *ngIf="!view.teams.length" class="py-3 text-sm text-slate-500">No teams.</li></ul></div>
          <div class="rounded-md border border-slate-200 bg-white p-5"><h2 class="font-semibold">Members</h2><ul class="mt-3 divide-y"><li *ngFor="let member of view.members" class="flex justify-between py-2 text-sm"><span>{{ member.userSubject }}</span><span class="text-slate-500">{{ member.role }}{{ member.teamId ? ' · ' + member.teamId : '' }}</span></li><li *ngIf="!view.members.length" class="py-3 text-sm text-slate-500">No members.</li></ul></div>
        </section>
        <section class="rounded-md border border-slate-200 bg-white p-5"><h2 class="font-semibold">Project access</h2><div class="mt-3 overflow-x-auto"><table class="w-full text-left text-sm"><thead class="text-slate-500"><tr><th class="py-2">Project</th><th>Team</th><th>Role</th></tr></thead><tbody><tr *ngFor="let access of view.access" class="border-t"><td class="py-2">{{ access.project }}</td><td>{{ access.teamId }}</td><td>{{ access.role }}</td></tr></tbody></table></div><p *ngIf="!view.access.length" class="mt-3 text-sm text-slate-500">No project grants.</p></section>
      </div>
      <ng-template #empty><p class="rounded-md border border-dashed border-slate-300 p-8 text-center text-slate-500">No organization membership is configured.</p></ng-template>
    </ng-container>
  `
})
export class OrganizationsPageComponent {
  private readonly api = inject(ApiClient);
  private readonly context = inject(OrganizationContext);
  readonly view$ = combineLatest([this.api.organizations(), this.context.selected$.pipe(startWith(''))]).pipe(
    map(([organizations, selected]) => ({ organizations, organization: organizations.find((item) => item.id === selected) ?? organizations[0] })),
    switchMap(({ organization }) => organization ? forkJoin({ organization: of(organization), teams: this.api.organizationTeams(organization.id), members: this.api.organizationMembers(organization.id), access: this.api.organizationProjectAccess(organization.id) }) : of({ organization: null, teams: [], members: [], access: [] }))
  );
}
