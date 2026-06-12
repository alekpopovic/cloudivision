import { Component, Input } from '@angular/core';

@Component({
  selector: 'app-empty-state',
  standalone: true,
  template: `
    <div class="rounded-md border border-dashed border-slate-300 bg-white px-6 py-10 text-center">
      <h2 class="text-sm font-semibold text-slate-900">{{ title }}</h2>
      <p class="mt-2 text-sm text-slate-600">{{ message }}</p>
    </div>
  `
})
export class EmptyStateComponent {
  @Input() title = 'No data';
  @Input() message = 'Nothing to show yet.';
}
