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

  it('renders the brand mark and one icon for every primary navigation item', async () => {
    await TestBed.configureTestingModule({
      imports: [AppComponent],
      providers: [provideRouter([]), { provide: ApiClient, useClass: FakeApiClient }]
    }).compileComponents();

    const fixture = TestBed.createComponent(AppComponent);
    fixture.detectChanges();

    const host: HTMLElement = fixture.nativeElement;
    const logo = host.querySelector<HTMLImageElement>('img[alt="cloudivision logo"]');
    const navigationIcons = host.querySelectorAll('nav[aria-label="Primary navigation"] img');

    expect(logo?.getAttribute('src')).toBe('/assets/brand/cloudivision-mark.png');
    expect(navigationIcons.length).toBe(fixture.componentInstance.nav.length);
  });
});
