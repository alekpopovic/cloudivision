import { ComponentFixture, TestBed, fakeAsync, tick } from '@angular/core/testing';
import { provideRouter } from '@angular/router';
import { of } from 'rxjs';

import { ApiClient } from '../../api/client';
import { BuildRunsPageComponent } from './build-runs-page.component';

class FakeApiClient {
  buildRunsCalls = 0;
  createBuildRunCalls = 0;

  buildRuns() {
    this.buildRunsCalls++;
    return of([]);
  }

  createBuildRun() {
    this.createBuildRunCalls++;
    return of({});
  }
}

describe('BuildRunsPageComponent', () => {
  let fixture: ComponentFixture<BuildRunsPageComponent>;
  let api: FakeApiClient;

  beforeEach(async () => {
    api = new FakeApiClient();
    await TestBed.configureTestingModule({
      imports: [BuildRunsPageComponent],
      providers: [
        provideRouter([]),
        { provide: ApiClient, useValue: api }
      ]
    }).compileComponents();
    fixture = TestBed.createComponent(BuildRunsPageComponent);
  });

  it('calls API service for BuildRuns', fakeAsync(() => {
    fixture.detectChanges();
    tick(0);
    expect(api.buildRunsCalls).toBeGreaterThan(0);
  }));

  it('keeps manual trigger form invalid until required fields are filled', () => {
    fixture.detectChanges();
    const component = fixture.componentInstance;

    component.trigger();

    expect(component.triggerForm.invalid).toBeTrue();
    expect(api.createBuildRunCalls).toBe(0);

    component.triggerForm.patchValue({
      name: 'build-1',
      namespace: 'ci',
      projectRef: 'project',
      repositoryRef: 'repo',
      pipelineTemplateRef: 'template',
      revision: 'main',
      imageRepository: 'ghcr.io/cloudivision/app'
    });

    expect(component.triggerForm.valid).toBeTrue();
  });
});
