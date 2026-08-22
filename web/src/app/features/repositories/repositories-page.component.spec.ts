import { TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';
import { of } from 'rxjs';

import { ApiClient } from '../../api/client';
import { RepositoriesPageComponent } from './repositories-page.component';

class FakeApiClient {
  repositories() { return of([]); }
  pipelineTemplates() { return of([]); }
  webhookUrl() { return of('/api/v1/webhooks/github/'); }
  createRepository(body: unknown) { return of(body); }
}

describe('RepositoriesPageComponent', () => {
  it('validates all repository wizard required fields', async () => {
    await TestBed.configureTestingModule({ imports: [RepositoriesPageComponent], providers: [provideRouter([]), { provide: ApiClient, useClass: FakeApiClient }] }).compileComponents();
    const fixture = TestBed.createComponent(RepositoriesPageComponent);
    const component = fixture.componentInstance;
    expect(component.form.invalid).toBeTrue();
    component.form.patchValue({ name: 'repo', projectRef: 'project', url: 'https://github.com/acme/repo.git', pipelineTemplateRef: 'build' });
    expect(component.form.valid).toBeTrue();
  });
});
