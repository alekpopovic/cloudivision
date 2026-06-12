import { TestBed } from '@angular/core/testing';

import { AppComponent } from './app.component';

describe('AppComponent', () => {
  it('creates the app', async () => {
    await TestBed.configureTestingModule({
      imports: [AppComponent]
    }).compileComponents();

    const fixture = TestBed.createComponent(AppComponent);
    expect(fixture.componentInstance).toBeTruthy();
  });

  it('renders the product name', async () => {
    await TestBed.configureTestingModule({
      imports: [AppComponent]
    }).compileComponents();

    const fixture = TestBed.createComponent(AppComponent);
    fixture.detectChanges();

    expect(fixture.nativeElement.textContent).toContain('cloudivision');
  });
});

