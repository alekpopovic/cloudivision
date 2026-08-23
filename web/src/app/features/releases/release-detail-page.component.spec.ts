import { TestBed } from '@angular/core/testing';
import { ActivatedRoute, convertToParamMap, provideRouter } from '@angular/router';
import { of } from 'rxjs';

import { ApiClient } from '../../api/client';
import { ReleaseDetailPageComponent } from './release-detail-page.component';

class FakeApiClient {
  releases() { return of([{ name: 'release-1', namespace: 'ci', spec: { projectRef: 'p', environmentRef: 'prod', buildRunRef: 'build-1', image: { repository: 'example/app', tag: 'v1' }, strategy: 'gitops', promotionMode: 'pull-request', promotedFrom: 'staging-release', rollbackOf: 'failed-release', rollbackTo: 'previous-release' }, status: { phase: 'GitOpsChangeCommitted', gitCommit: 'abc123', deployment: { provider: 'argocd', syncStatus: 'OutOfSync', healthStatus: 'Progressing', observedRevision: 'main@abc123', observedAt: '2026-08-23T00:00:00Z' }, pullRequest: { url: 'https://github.com/acme/gitops/pull/4' }, conditions: [] } }]); }
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
    expect(fixture.nativeElement.textContent).toContain('OutOfSync');
    expect(fixture.nativeElement.textContent).toContain('Progressing');
    expect(fixture.nativeElement.textContent).toContain('main@abc123');
    expect(fixture.nativeElement.textContent).toContain('staging-release');
    expect(fixture.nativeElement.textContent).toContain('previous-release');
  });
});
