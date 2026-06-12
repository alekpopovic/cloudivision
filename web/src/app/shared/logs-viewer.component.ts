import { Component, Input } from '@angular/core';

@Component({
  selector: 'app-logs-viewer',
  standalone: true,
  template: `
    <pre class="max-h-[34rem] overflow-auto rounded-md bg-slate-950 p-4 text-xs leading-5 text-slate-100">{{ lines.join('\n') || 'No logs available.' }}</pre>
  `
})
export class LogsViewerComponent {
  @Input() lines: string[] = [];
}
