import { TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';
import { of, throwError } from 'rxjs';

import { ApiClient } from '../../api/client';
import { FirstRunWizardComponent } from './first-run-wizard.component';

describe('FirstRunWizardComponent', () => {
  class FakeApiClient {
    fail = false;
    buildCalls = 0;
    projects() { return of([]); }
    catalogPipelineTemplates() { return of([{ name: 'go', version: '1.0.0', description: 'Go', spec: {} }]); }
    createProject() { return this.fail ? throwError(() => ({ code: 'create_failed', message: 'boom' })) : of({}); }
    installCatalogPipelineTemplate() { return of({}); }
    createRepository() { return of({}); }
    createBuildRun() { this.buildCalls++; return of({}); }
  }

  it('renders the first-run workflow', async () => {
    await TestBed.configureTestingModule({ imports: [FirstRunWizardComponent], providers: [provideRouter([]), { provide: ApiClient, useClass: FakeApiClient }] }).compileComponents();
    const fixture = TestBed.createComponent(FirstRunWizardComponent);
    fixture.detectChanges();
    expect(fixture.nativeElement.textContent).toContain('First-run wizard');
    expect(fixture.nativeElement.textContent).toContain('Welcome');
    expect(fixture.nativeElement.textContent).toContain('Logs');
    expect(fixture.nativeElement.textContent).toContain('installation is empty');
  });

  it('validates forms, selects a catalog template and triggers a BuildRun', async () => {
    await TestBed.configureTestingModule({ imports: [FirstRunWizardComponent], providers: [provideRouter([]), { provide: ApiClient, useClass: FakeApiClient }] }).compileComponents();
    const fixture = TestBed.createComponent(FirstRunWizardComponent); const component = fixture.componentInstance;
    expect(component.form.controls.repositoryUrl.invalid).toBeTrue();
    component.form.patchValue({ repositoryUrl: 'https://github.com/acme/demo.git', catalogTemplate: 'go' });
    expect(component.canSubmit).toBeTrue();
    component.createStarter();
    expect((TestBed.inject(ApiClient) as unknown as FakeApiClient).buildCalls).toBe(1);
    expect(component.completedBuildName).toContain('first-build');
  });

  it('shows API errors', async () => {
    await TestBed.configureTestingModule({ imports: [FirstRunWizardComponent], providers: [provideRouter([]), { provide: ApiClient, useClass: FakeApiClient }] }).compileComponents();
    const component = TestBed.createComponent(FirstRunWizardComponent).componentInstance;
    component.form.controls.repositoryUrl.setValue('https://github.com/acme/demo.git');
    (TestBed.inject(ApiClient) as unknown as FakeApiClient).fail = true;
    component.createStarter();
    expect(component.error?.code).toBe('create_failed');
  });
});
