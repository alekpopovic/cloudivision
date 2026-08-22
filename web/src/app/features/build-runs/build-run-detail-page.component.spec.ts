import { ComponentFixture, TestBed, discardPeriodicTasks, fakeAsync, tick } from '@angular/core/testing';
import { ActivatedRoute, convertToParamMap, provideRouter } from '@angular/router';
import { of } from 'rxjs';

import { ApiClient } from '../../api/client';
import { BuildRunDetailPageComponent } from './build-run-detail-page.component';

class FakeApiClient {
  buildRun() {
    return of({
      name: 'failed-build', namespace: 'ci',
      spec: { projectRef: 'p', repositoryRef: 'r', pipelineTemplateRef: 't', revision: 'abc', triggeredBy: { type: 'manual' }, image: { repository: 'example/app' } },
      status: { phase: 'Failed', failure: { reason: 'StepFailed', message: 'unit tests failed' }, conditions: [] }
    });
  }
  buildRunLogs() { return of({ namespace: 'ci', buildRun: 'failed-build', lines: ['failure'] }); }
  pipelineTemplates() { return of([]); }
  releases() { return of([]); }
  createBuildRun() { return this.buildRun(); }
}

describe('BuildRunDetailPageComponent', () => {
  let fixture: ComponentFixture<BuildRunDetailPageComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [BuildRunDetailPageComponent],
      providers: [
        provideRouter([]),
        { provide: ApiClient, useClass: FakeApiClient },
        { provide: ActivatedRoute, useValue: { paramMap: of(convertToParamMap({ namespace: 'ci', name: 'failed-build' })) } }
      ]
    }).compileComponents();
    fixture = TestBed.createComponent(BuildRunDetailPageComponent);
  });

  it('renders the backend failure reason and message', fakeAsync(() => {
    fixture.detectChanges();
		tick(0);
		fixture.detectChanges();
    expect(fixture.nativeElement.textContent).toContain('StepFailed');
    expect(fixture.nativeElement.textContent).toContain('unit tests failed');
		discardPeriodicTasks();
	}));
});
