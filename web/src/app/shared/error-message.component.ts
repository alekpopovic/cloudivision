import { Component, Input } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ApiError } from '../api/models';

@Component({
  selector: 'app-error-message',
  standalone: true,
  imports: [CommonModule],
  template: `
    <div *ngIf="error" class="rounded-md border border-rose-200 bg-rose-50 px-4 py-3 text-sm text-rose-800">
      <span class="font-semibold">{{ error.code }}</span>
      <span class="ml-2">{{ error.message }}</span>
      <div *ngIf="error.requestId" class="mt-1 font-mono text-xs text-rose-700">request ID: {{ error.requestId }}</div>
      <ul *ngIf="error.violations?.length" class="mt-2 list-disc space-y-1 pl-5">
        <li *ngFor="let violation of error.violations"><strong>{{ violation.policy }}</strong>: {{ violation.message }}<code *ngIf="violation.fieldPath" class="ml-1 text-xs">({{ violation.fieldPath }})</code></li>
      </ul>
    </div>
  `
})
export class ErrorMessageComponent {
  @Input() error: ApiError | null = null;
}
