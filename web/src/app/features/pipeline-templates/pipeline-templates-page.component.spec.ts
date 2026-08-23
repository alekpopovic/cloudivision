import { TestBed } from '@angular/core/testing';
import { ReactiveFormsModule } from '@angular/forms';
import { of } from 'rxjs';

import { ApiClient } from '../../api/client';
import { PipelineTemplatesPageComponent } from './pipeline-templates-page.component';

class FakeApiClient {
  pipelineTemplates() { return of([]); }
  catalogPipelineTemplates() { return of([{ name: 'go', version: '1.0.0', description: 'Go CI', spec: { steps: [{ name: 'test', image: 'golang:1.26', command: ['go', 'test', './...'] }], build: { enabled: true, builder: 'buildkit', push: true } } }]); }
  installCatalogPipelineTemplate() { return of({}); }
  createPipelineTemplate() { return of({}); }
}

describe('PipelineTemplatesPageComponent catalog', () => {
  it('renders and imports a catalog template', async () => {
    await TestBed.configureTestingModule({ imports: [ReactiveFormsModule, PipelineTemplatesPageComponent], providers: [{ provide: ApiClient, useClass: FakeApiClient }] }).compileComponents();
    const fixture = TestBed.createComponent(PipelineTemplatesPageComponent);
    fixture.detectChanges();
    await fixture.whenStable();
    expect(fixture.nativeElement.textContent).toContain('Go CI');
    fixture.componentInstance.importCatalog({ name: 'go', version: '1.0.0', description: 'Go CI', spec: { steps: [{ name: 'test', image: 'golang:1.26', command: ['go', 'test', './...'] }], build: { enabled: true, builder: 'buildkit', push: true } } });
    expect(fixture.componentInstance.form.controls.name.value).toBe('go');
    expect(fixture.componentInstance.steps.length).toBe(1);
  });
});
