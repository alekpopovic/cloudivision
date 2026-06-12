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
    </div>
  `
})
export class ErrorMessageComponent {
  @Input() error: ApiError | null = null;
}
