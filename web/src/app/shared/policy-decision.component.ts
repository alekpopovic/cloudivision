import { CommonModule } from '@angular/common';
import { Component, Input } from '@angular/core';

import { PolicyDecision } from '../api/models';

@Component({
  selector: 'app-policy-decision',
  standalone: true,
  imports: [CommonModule],
  template: `
    <section *ngIf="decision" class="rounded-md border p-4" [class.border-emerald-200]="decision.allowed" [class.bg-emerald-50]="decision.allowed" [class.border-rose-200]="!decision.allowed" [class.bg-rose-50]="!decision.allowed">
      <div class="flex items-center justify-between gap-3">
        <h2 class="text-sm font-semibold">Policy decision: {{ decision.allowed ? 'Allowed' : 'Denied' }}</h2>
        <span class="text-xs text-slate-500">{{ decision.reason || '-' }}</span>
      </div>
      <p class="mt-1 text-sm text-slate-700">{{ decision.message }}</p>
      <ul *ngIf="decision.violations?.length" class="mt-3 space-y-2">
        <li *ngFor="let violation of decision.violations" class="rounded border border-rose-200 bg-white px-3 py-2 text-sm text-rose-900">
          <span class="font-semibold">{{ violation.policy }}</span>
          <span class="ml-2 text-xs uppercase">{{ violation.severity }}</span>
          <p>{{ violation.message }}</p>
          <code *ngIf="violation.fieldPath" class="text-xs text-slate-600">{{ violation.fieldPath }}</code>
        </li>
      </ul>
    </section>
  `
})
export class PolicyDecisionComponent {
  @Input() decision?: PolicyDecision;
}
