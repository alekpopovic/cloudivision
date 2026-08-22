import { TestBed } from '@angular/core/testing';
import { ActivatedRoute, convertToParamMap, provideRouter } from '@angular/router';
import { of } from 'rxjs';

import { ApiClient } from '../../api/client';
import { ReleaseDetailPageComponent } from './release-detail-page.component';

class FakeApiClient {
  releases() { return of([{ name: 'release-1', namespace: 'ci', spec: { projectRef: 'p', environmentRef: 'prod', buildRunRef: 'build-1', image: { repository: 'example/app', tag: 'v1' }, strategy: 'gitops', promotionMode: 'pull-request' }, status: { phase: 'GitOpsChangeCommitted', gitCommit: 'abc123', pullRequest: { url: 'https://github.com/acme/gitops/pull/4' }, conditions: [] } }]); }
  buildRuns() { return of([{ name: 'build-1', namespace: 'ci', spec: { gitOps: { repoURL: 'https://github.com/acme/gitops.git' } } }]); }
}

describe('ReleaseDetailPageComponent', () => {
  it('shows GitOps commit and PR links', async () => {
    await TestBed.configureTestingModule({
      imports: [ReleaseDetailPageComponent],
      providers: [provideRouter([]), { provide: ApiClient, useClass: FakeApiClient }, { provide: ActivatedRoute, useValue: { paramMap: of(convertToParamMap({ namespace: 'ci', name: 'release-1' })) } }]
    }).compileComponents();
    const fixture = TestBed.createComponent(ReleaseDetailPageComponent);
    fixture.detectChanges();
    expect(fixture.nativeElement.textContent).toContain('Open GitOps commit');
    expect(fixture.nativeElement.textContent).toContain('Open pull request');
  });
});
