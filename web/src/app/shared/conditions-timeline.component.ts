import { Component, Input } from '@angular/core';
import { CommonModule } from '@angular/common';
import { Condition } from '../api/models';
import { StatusBadgeComponent } from './status-badge.component';

@Component({
  selector: 'app-conditions-timeline',
  standalone: true,
  imports: [CommonModule, StatusBadgeComponent],
  template: `
    <ol class="space-y-3">
      <li *ngFor="let condition of conditions" class="rounded-md border border-slate-200 bg-white px-4 py-3">
        <div class="flex flex-wrap items-center gap-2">
          <span class="font-medium text-slate-950">{{ condition.type }}</span>
          <app-status-badge [status]="condition.status" />
          <span class="text-xs text-slate-500">{{ condition.lastTransitionTime || '' }}</span>
        </div>
        <p class="mt-2 text-sm text-slate-600">{{ condition.reason || '' }} {{ condition.message || '' }}</p>
      </li>
    </ol>
  `
})
export class ConditionsTimelineComponent {
  @Input() conditions: Condition[] = [];
}
