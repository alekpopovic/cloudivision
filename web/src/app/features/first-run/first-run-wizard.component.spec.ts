import { TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';

import { ApiClient } from '../../api/client';
import { FirstRunWizardComponent } from './first-run-wizard.component';

describe('FirstRunWizardComponent', () => {
  it('renders the first-run workflow', async () => {
    await TestBed.configureTestingModule({ imports: [FirstRunWizardComponent], providers: [provideRouter([]), { provide: ApiClient, useValue: {} }] }).compileComponents();
    const fixture = TestBed.createComponent(FirstRunWizardComponent);
    fixture.detectChanges();
    expect(fixture.nativeElement.textContent).toContain('First-run wizard');
    expect(fixture.nativeElement.textContent).toContain('Create Project');
    expect(fixture.nativeElement.textContent).toContain('View logs');
  });
});
