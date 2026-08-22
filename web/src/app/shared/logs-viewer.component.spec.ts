import { ComponentFixture, TestBed } from '@angular/core/testing';

import { LogsViewerComponent } from './logs-viewer.component';

describe('LogsViewerComponent', () => {
  let fixture: ComponentFixture<LogsViewerComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({ imports: [LogsViewerComponent] }).compileComponents();
    fixture = TestBed.createComponent(LogsViewerComponent);
  });

  it('renders loading, error and success states', () => {
    fixture.componentInstance.loading = true;
    fixture.detectChanges();
    expect(fixture.nativeElement.textContent).toContain('Loading logs');

    fixture.componentInstance.loading = false;
    fixture.componentInstance.error = 'logs unavailable';
    fixture.detectChanges();
    expect(fixture.nativeElement.textContent).toContain('logs unavailable');

    fixture.componentInstance.error = '';
    fixture.componentInstance.lines = ['[step:test] success'];
    fixture.detectChanges();
    expect(fixture.nativeElement.textContent).toContain('success');
  });

  it('filters lines by search text and step', () => {
    const component = fixture.componentInstance;
    component.lines = ['[step:test] alpha', '[step:build] beta'];
    component.search = 'beta';
    expect(component.filteredLines).toEqual(['[step:build] beta']);
    component.search = '';
    component.stepFilter = 'test';
    expect(component.filteredLines).toEqual(['[step:test] alpha']);
  });
});
