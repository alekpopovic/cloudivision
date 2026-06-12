import { ComponentFixture, TestBed } from '@angular/core/testing';

import { StatusBadgeComponent } from './status-badge.component';

describe('StatusBadgeComponent', () => {
  let fixture: ComponentFixture<StatusBadgeComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({ imports: [StatusBadgeComponent] }).compileComponents();
    fixture = TestBed.createComponent(StatusBadgeComponent);
  });

  it('renders the correct label', () => {
    fixture.componentRef.setInput('status', 'Succeeded');
    fixture.detectChanges();
    expect(fixture.nativeElement.textContent).toContain('Succeeded');
  });
});
