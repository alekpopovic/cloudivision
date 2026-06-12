import { ComponentFixture, TestBed, fakeAsync, tick } from '@angular/core/testing';
import { provideRouter } from '@angular/router';
import { of } from 'rxjs';

import { ApiClient } from '../../api/client';
import { BuildRunsPageComponent } from './build-runs-page.component';

class FakeApiClient {
  buildRunsCalls = 0;

  buildRuns() {
    this.buildRunsCalls++;
    return of([]);
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
});
