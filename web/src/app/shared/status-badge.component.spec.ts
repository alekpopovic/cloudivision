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

  it('uses success classes for succeeded status', () => {
    fixture.componentRef.setInput('status', 'Succeeded');
    fixture.detectChanges();
    const badge: HTMLElement = fixture.nativeElement.querySelector('span');
    expect(badge.className).toContain('bg-emerald-50');
    expect(badge.className).toContain('text-emerald-700');
  });

	it('uses failure classes for specific Release failures', () => {
		fixture.componentRef.setInput('status', 'FailedGitPush');
		fixture.detectChanges();
		expect((fixture.nativeElement.querySelector('span') as HTMLElement).className).toContain('bg-rose-50');
	});

	it('uses in-progress classes while waiting for sync', () => {
		fixture.componentRef.setInput('status', 'WaitingForSync');
		fixture.detectChanges();
		expect((fixture.nativeElement.querySelector('span') as HTMLElement).className).toContain('bg-amber-50');
	});
});
