import { CommonModule } from '@angular/common';
import { AfterViewChecked, Component, ElementRef, Input, ViewChild } from '@angular/core';
import { FormsModule } from '@angular/forms';

@Component({
  selector: 'app-logs-viewer',
  standalone: true,
  imports: [CommonModule, FormsModule],
  template: `
    <div class="rounded-md border border-slate-800 bg-slate-950 text-slate-100">
      <div class="flex flex-wrap items-center gap-2 border-b border-slate-800 p-2 text-xs">
        <input class="min-w-40 flex-1 rounded border border-slate-700 bg-slate-900 px-2 py-1" [(ngModel)]="search" placeholder="Search logs" aria-label="Search logs" />
        <select *ngIf="steps.length" class="rounded border border-slate-700 bg-slate-900 px-2 py-1" [(ngModel)]="stepFilter" aria-label="Filter by step">
          <option value="">All steps</option>
          <option *ngFor="let step of steps" [value]="step">{{ step }}</option>
        </select>
        <label class="flex items-center gap-1"><input type="checkbox" [(ngModel)]="autoScroll" /> Auto-scroll</label>
        <button class="rounded border border-slate-700 px-2 py-1" type="button" (click)="togglePause()">{{ paused ? 'Resume' : 'Pause' }}</button>
        <button class="rounded border border-slate-700 px-2 py-1" type="button" (click)="copyLogs()">Copy</button>
        <button class="rounded border border-slate-700 px-2 py-1" type="button" (click)="downloadLogs()">Download</button>
      </div>
      <p *ngIf="loading" class="p-4 text-xs text-slate-400">Loading logs…</p>
      <p *ngIf="error" class="p-4 text-xs text-rose-300">{{ error }}</p>
      <pre #viewport *ngIf="!loading && !error" class="max-h-[34rem] overflow-auto p-4 text-xs leading-5"><ng-container *ngFor="let line of filteredLines">{{ cleanLine(line) }}
</ng-container><ng-container *ngIf="!filteredLines.length">No logs available.</ng-container></pre>
    </div>
  `
})
export class LogsViewerComponent implements AfterViewChecked {
  @Input() lines: string[] = [];
  @Input() loading = false;
  @Input() error = '';
  @Input() fileName = 'build.log';
  @ViewChild('viewport') viewport?: ElementRef<HTMLElement>;

  search = '';
  stepFilter = '';
  autoScroll = true;
  paused = false;
  private frozenLines: string[] = [];

  get visibleLines(): string[] {
    return this.paused ? this.frozenLines : this.lines;
  }

  get steps(): string[] {
    const names = this.lines.map((line) => /^\[step:([^\]]+)\]/.exec(line)?.[1]).filter((name): name is string => !!name);
    return [...new Set(names)];
  }

  get filteredLines(): string[] {
    const query = this.search.toLowerCase();
    return this.visibleLines.filter((line) => {
      const clean = this.cleanLine(line);
      return (!query || clean.toLowerCase().includes(query)) && (!this.stepFilter || clean.startsWith(`[step:${this.stepFilter}]`));
    });
  }

  togglePause(): void {
    if (!this.paused) {
      this.frozenLines = [...this.lines];
    }
    this.paused = !this.paused;
  }

  cleanLine(line: string): string {
    return line.replace(/\u001b\[[0-9;]*m/g, '');
  }

  copyLogs(): void {
    if (typeof navigator !== 'undefined' && navigator.clipboard) {
      void navigator.clipboard.writeText(this.filteredLines.map((line) => this.cleanLine(line)).join('\n'));
    }
  }

  downloadLogs(): void {
    if (typeof document === 'undefined' || typeof URL === 'undefined') return;
    const url = URL.createObjectURL(new Blob([this.filteredLines.map((line) => this.cleanLine(line)).join('\n')], { type: 'text/plain' }));
    const anchor = document.createElement('a');
    anchor.href = url;
    anchor.download = this.fileName;
    anchor.click();
    URL.revokeObjectURL(url);
  }

  ngAfterViewChecked(): void {
    if (this.autoScroll && !this.paused && this.viewport) {
      this.viewport.nativeElement.scrollTop = this.viewport.nativeElement.scrollHeight;
    }
  }
}
