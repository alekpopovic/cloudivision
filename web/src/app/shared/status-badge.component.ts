import { Component, Input } from '@angular/core';

@Component({
  selector: 'app-status-badge',
  standalone: true,
  template: `<span class="inline-flex items-center rounded-full px-2.5 py-1 text-xs font-medium" [class]="badgeClass">{{ label }}</span>`
})
export class StatusBadgeComponent {
  @Input() status = 'Pending';

  get label(): string {
    return this.status || 'Pending';
  }

  get badgeClass(): string {
    const normalized = this.label.toLowerCase();
    if (['succeeded', 'ready', 'deployed', 'synced', 'healthy'].includes(normalized)) {
      return 'bg-emerald-50 text-emerald-700 ring-1 ring-emerald-200';
    }
    if (['failed', 'error', 'cancelled', 'rolledback'].includes(normalized)) {
      return 'bg-rose-50 text-rose-700 ring-1 ring-rose-200';
    }
    if (['running', 'queued', 'deploying', 'awaitingapproval', 'pending'].includes(normalized)) {
      return 'bg-amber-50 text-amber-700 ring-1 ring-amber-200';
    }
    return 'bg-slate-100 text-slate-700 ring-1 ring-slate-200';
  }
}
