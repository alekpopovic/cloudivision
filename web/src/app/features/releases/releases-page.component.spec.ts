import { ComponentFixture, TestBed, fakeAsync, tick } from '@angular/core/testing';
import { of } from 'rxjs';

import { ApiClient } from '../../api/client';
import { Release } from '../../api/models';
import { ReleasesPageComponent } from './releases-page.component';

class FakeApiClient {
  approveCalls = 0;
  rejectCalls = 0;
  releasesResponse: Release[] = [
    {
      name: 'release-1',
      namespace: 'ci',
      spec: {
        projectRef: 'project',
        environmentRef: 'prod',
        buildRunRef: 'build-1',
        image: { repository: 'ghcr.io/cloudivision/app', tag: 'main' },
        approval: { required: true },
        strategy: 'gitops'
      },
      status: { phase: 'AwaitingApproval' }
    }
  ];

  releases() {
    return of(this.releasesResponse);
  }

  approveRelease() {
    this.approveCalls++;
    return of(this.releasesResponse[0]);
  }

  rejectRelease() {
    this.rejectCalls++;
    return of(this.releasesResponse[0]);
  }
}

describe('ReleasesPageComponent', () => {
  let fixture: ComponentFixture<ReleasesPageComponent>;
  let api: FakeApiClient;

  beforeEach(async () => {
    api = new FakeApiClient();
    await TestBed.configureTestingModule({
      imports: [ReleasesPageComponent],
      providers: [{ provide: ApiClient, useValue: api }]
    }).compileComponents();
    fixture = TestBed.createComponent(ReleasesPageComponent);
  });

  it('shows approval actions for releases awaiting approval', fakeAsync(() => {
    fixture.detectChanges();
    tick(0);
    fixture.detectChanges();

    expect(fixture.nativeElement.textContent).toContain('Approval required');
    expect(fixture.nativeElement.textContent).toContain('Approve');
    expect(fixture.nativeElement.textContent).toContain('Reject');
  }));

  it('calls API when approving a release', fakeAsync(() => {
    fixture.detectChanges();
    tick(0);
    const component = fixture.componentInstance;

    component.approve(api.releasesResponse[0]);
    tick(0);

    expect(api.approveCalls).toBe(1);
    expect(api.rejectCalls).toBe(0);
  }));
});
