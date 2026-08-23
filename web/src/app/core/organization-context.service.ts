import { Injectable } from '@angular/core';
import { BehaviorSubject } from 'rxjs';

@Injectable({ providedIn: 'root' })
export class OrganizationContext {
  private readonly selected = new BehaviorSubject<string>('');
  readonly selected$ = this.selected.asObservable();
  get value(): string { return this.selected.value; }
  select(id: string): void { this.selected.next(id); }
}
