import { TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';
import { of } from 'rxjs';

import { ApiClient } from './api/client';
import { AppComponent } from './app.component';

class FakeApiClient {
  currentUser() {
    return of({ subject: 'dev-user', displayName: 'Development User', roles: ['admin'], devMode: true });
  }
}

describe('AppComponent', () => {
  it('creates the app', async () => {
    await TestBed.configureTestingModule({
      imports: [AppComponent],
      providers: [provideRouter([]), { provide: ApiClient, useClass: FakeApiClient }]
    }).compileComponents();

    const fixture = TestBed.createComponent(AppComponent);
    expect(fixture.componentInstance).toBeTruthy();
  });

  it('renders the product name', async () => {
    await TestBed.configureTestingModule({
      imports: [AppComponent],
      providers: [provideRouter([]), { provide: ApiClient, useClass: FakeApiClient }]
    }).compileComponents();

    const fixture = TestBed.createComponent(AppComponent);
    fixture.detectChanges();

    expect(fixture.nativeElement.textContent).toContain('cloudivision');
  });
});
