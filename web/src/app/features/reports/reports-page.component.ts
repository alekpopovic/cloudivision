import { CommonModule } from '@angular/common';
import { Component, inject } from '@angular/core';
import { RouterLink } from '@angular/router';
import { catchError, forkJoin, of } from 'rxjs';

import { ApiClient } from '../../api/client';
import { PageHeaderComponent } from '../../shared/page-header.component';
import { downloadBlob } from './audit-export-page.component';

@Component({selector:'app-reports-page',standalone:true,imports:[CommonModule,RouterLink,PageHeaderComponent],template:`
  <app-page-header title="Compliance reports" description="Build, release and security summaries from Kubernetes status and audit history." />
  <p *ngIf="error" class="mb-4 rounded-md bg-rose-50 p-3 text-sm text-rose-700">{{ error }}</p>
  <div *ngIf="reports$ | async as reports" class="grid gap-5 lg:grid-cols-3">
    <article class="rounded-md border border-slate-200 bg-white p-5"><h2 class="font-semibold">Builds</h2><p class="mt-4 text-3xl font-bold">{{ reports.builds.total }}</p><p class="text-sm text-slate-500">{{ reports.builds.succeeded }} succeeded · {{ reports.builds.failed }} failed</p><p class="mt-2 text-sm">Average {{ reports.builds.averageDurationSeconds | number:'1.0-1' }} seconds</p><button (click)="export('builds')" class="mt-4 text-sm font-semibold text-brand-700">Download CSV</button></article>
    <article class="rounded-md border border-slate-200 bg-white p-5"><h2 class="font-semibold">Releases</h2><p class="mt-4 text-3xl font-bold">{{ reports.releases.total }}</p><p class="text-sm text-slate-500">{{ reports.releases.approvals }} approvals · {{ reports.releases.rollbacks }} rollbacks</p><p class="mt-2 text-sm">{{ reports.releases.deploymentFailures }} deployment failures</p><button (click)="export('releases')" class="mt-4 text-sm font-semibold text-brand-700">Download CSV</button></article>
    <article class="rounded-md border border-slate-200 bg-white p-5"><h2 class="font-semibold">Security</h2><p class="mt-4 text-3xl font-bold">{{ reports.security.policyDenials }}</p><p class="text-sm text-slate-500">policy denials</p><p class="mt-2 text-sm">{{ reports.security.webhookRejections }} webhook rejections · {{ reports.security.criticalVulnerabilityBlocks }} critical blocks</p><button (click)="export('security')" class="mt-4 text-sm font-semibold text-brand-700">Download CSV</button></article>
  </div>
  <a routerLink="/audit" class="mt-6 inline-block text-sm font-semibold text-brand-700">Open detailed audit export →</a>
`})
export class ReportsPageComponent {
  private readonly api=inject(ApiClient);error='';
  readonly reports$=forkJoin({builds:this.api.buildReport(),releases:this.api.releaseReport(),security:this.api.securityReport()}).pipe(catchError((error)=>{this.error=error.message;return of({builds:{total:0,succeeded:0,failed:0,successRate:0,failureRate:0,averageDurationSeconds:0,failedByReason:{}},releases:{total:0,byEnvironment:{},approvals:0,rejections:0,rollbacks:0,deploymentFailures:0},security:{policyDenials:0,unsignedReleasesBlocked:0,webhookRejections:0,criticalVulnerabilityBlocks:0}})}));
  export(report:'builds'|'releases'|'security'):void{this.api.reportExport(report).subscribe({next:(blob)=>downloadBlob(blob,`${report}-report.csv`),error:(error)=>this.error=error.message});}
}
