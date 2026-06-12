import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';
import { of } from 'rxjs';

import { ApiClient } from '../../api/client';
import { DashboardPageComponent } from './dashboard-page.component';

class FakeApiClient {
  projects() {
    return of([]);
  }

  buildRuns() {
    return of([]);
  }

  releases() {
    return of([]);
  }
}

describe('DashboardPageComponent', () => {
  let fixture: ComponentFixture<DashboardPageComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [DashboardPageComponent],
      providers: [
        provideRouter([]),
        { provide: ApiClient, useClass: FakeApiClient }
      ]
    }).compileComponents();
    fixture = TestBed.createComponent(DashboardPageComponent);
  });

  it('renders dashboard cards', () => {
    fixture.detectChanges();

    expect(fixture.nativeElement.textContent).toContain('Dashboard');
    expect(fixture.nativeElement.textContent).toContain('Projects');
    expect(fixture.nativeElement.textContent).toContain('Failed Builds');
  });
});
