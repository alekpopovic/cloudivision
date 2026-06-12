import { Component, Input } from '@angular/core';
import { CommonModule } from '@angular/common';

@Component({
  selector: 'app-page-header',
  standalone: true,
  template: `
    <header class="mb-6 flex flex-col gap-3 border-b border-slate-200 pb-5 md:flex-row md:items-end md:justify-between">
      <div>
        <p *ngIf="eyebrow" class="text-xs font-semibold uppercase text-emerald-700">{{ eyebrow }}</p>
        <h1 class="mt-1 text-2xl font-semibold text-slate-950">{{ title }}</h1>
        <p *ngIf="description" class="mt-2 max-w-3xl text-sm leading-6 text-slate-600">{{ description }}</p>
      </div>
      <ng-content />
    </header>
  `,
  imports: [CommonModule]
})
export class PageHeaderComponent {
  @Input({ required: true }) title = '';
  @Input() eyebrow = '';
  @Input() description = '';
}
