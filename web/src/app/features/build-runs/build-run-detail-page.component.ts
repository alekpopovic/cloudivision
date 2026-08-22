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
			<nav class="mb-4 flex gap-2 border-b border-slate-200" aria-label="BuildRun detail tabs">
				<button class="border-b-2 px-3 py-2 text-sm font-medium" [class.border-blue-600]="activeTab === 'details'" [class.text-blue-700]="activeTab === 'details'" (click)="activeTab = 'details'">Details</button>
				<button class="border-b-2 px-3 py-2 text-sm font-medium" [class.border-blue-600]="activeTab === 'supply-chain'" [class.text-blue-700]="activeTab === 'supply-chain'" (click)="activeTab = 'supply-chain'">Supply Chain</button>
			</nav>
			<ng-container *ngIf="activeTab === 'details'">
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
			<section *ngIf="activeTab === 'supply-chain'" class="rounded-md border border-slate-200 bg-white p-4">
				<h2 class="mb-3 text-sm font-semibold">Artifact evidence</h2>
				<app-key-value-list [items]="[
					{ key: 'Image digest', value: vm.run.status?.image?.digest || vm.run.spec.image.digest || 'Not recorded' },
					{ key: 'SBOM path', value: vm.run.status?.supplyChain?.sbomPath || 'Not generated' },
					{ key: 'SBOM digest', value: vm.run.status?.supplyChain?.sbomDigest || 'Not recorded' },
					{ key: 'Scan results', value: vm.run.status?.supplyChain?.scannerResultsRef || 'Not scanned' },
					{ key: 'Critical / High', value: (vm.run.status?.supplyChain?.criticalVulnerabilities || 0) + ' / ' + (vm.run.status?.supplyChain?.highVulnerabilities || 0) },
					{ key: 'Signature', value: vm.run.status?.supplyChain?.signatureRef || 'Not signed' },
					{ key: 'Provenance', value: vm.run.status?.supplyChain?.provenanceRef || 'Not written' },
					{ key: 'Policy evidence', value: policyDecision(vm.run) }
				]" />
			</section>
    </ng-container>
  `
})
export class BuildRunDetailPageComponent {
  private readonly api = inject(ApiClient);
  private readonly route = inject(ActivatedRoute);
	activeTab: 'details' | 'supply-chain' = 'details';
  readonly params$ = this.route.paramMap.pipe(map((params) => ({ namespace: params.get('namespace') || '', name: params.get('name') || '' })));
  readonly vm$ = combineLatest([
    timer(0, 2000).pipe(switchMap(() => this.params$), switchMap((params) => this.api.buildRun(params.namespace, params.name))),
    timer(0, 2000).pipe(switchMap(() => this.params$), switchMap((params) => this.api.buildRunLogs(params.namespace, params.name)))
  ]).pipe(map(([run, logs]) => ({ run, logs: logs.lines })));

  image(run: { status?: { image?: { repository?: string; tag?: string; digest?: string } }; spec: { image: { repository: string; tag?: string; digest?: string } } }): string {
    const image = run.status?.image?.repository ? run.status.image : run.spec.image;
    return `${image.repository || ''}${image.tag ? ':' + image.tag : ''}${image.digest ? '@' + image.digest : ''}`;
  }

	policyDecision(run: { status?: { supplyChain?: { criticalVulnerabilities?: number; sbomDigest?: string; signatureRef?: string } } }): string {
		const evidence = run.status?.supplyChain;
		if ((evidence?.criticalVulnerabilities || 0) > 0) {
			return 'Blocked when target policy forbids critical vulnerabilities';
		}
		return evidence?.sbomDigest || evidence?.signatureRef ? 'Evidence available for Release policy evaluation' : 'No SBOM or signature evidence recorded';
	}
}
