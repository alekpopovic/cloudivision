import { Component, Input } from '@angular/core';

@Component({
  selector: 'app-loading-state',
  standalone: true,
  template: `<div class="rounded-md border border-slate-200 bg-white px-4 py-3 text-sm text-slate-600">{{ label }}</div>`
})
export class LoadingStateComponent {
  @Input() label = 'Loading...';
}
