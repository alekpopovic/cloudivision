import { Component, Input } from '@angular/core';
import { CommonModule } from '@angular/common';

@Component({
  selector: 'app-key-value-list',
  standalone: true,
  imports: [CommonModule],
  template: `
    <dl class="grid gap-3 sm:grid-cols-2">
      <div *ngFor="let item of items" class="rounded-md border border-slate-200 bg-white px-4 py-3">
        <dt class="text-xs font-medium uppercase text-slate-500">{{ item.key }}</dt>
        <dd class="mt-1 break-words text-sm text-slate-950">{{ item.value || '-' }}</dd>
      </div>
    </dl>
  `
})
export class KeyValueListComponent {
  @Input() items: Array<{ key: string; value?: string }> = [];
}
