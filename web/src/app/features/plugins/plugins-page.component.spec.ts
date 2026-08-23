import { ComponentFixture, TestBed, discardPeriodicTasks, fakeAsync, tick } from '@angular/core/testing';
import { of } from 'rxjs';

import { ApiClient } from '../../api/client';
import { PluginsPageComponent } from './plugins-page.component';

describe('PluginsPageComponent', () => {
  let fixture: ComponentFixture<PluginsPageComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [PluginsPageComponent],
      providers: [{ provide: ApiClient, useValue: { pluginHealth: () => of([{
        metadata: { name: 'default', type: 'policy', version: '1.0.0', capabilities: [{ name: 'evaluate', description: 'Evaluate' }], configSchema: {}, configured: true, configurationStatus: 'ready' },
        health: { healthy: true, message: 'ready', checkedAt: '2026-08-23T00:00:00Z' }
      }]) } }]
    }).compileComponents();
    fixture = TestBed.createComponent(PluginsPageComponent);
  });

  it('shows plugin metadata, health and capabilities', fakeAsync(() => {
    fixture.detectChanges(); tick(0); fixture.detectChanges();
    expect(fixture.nativeElement.textContent).toContain('default');
    expect(fixture.nativeElement.textContent).toContain('Healthy');
    expect(fixture.nativeElement.textContent).toContain('evaluate');
    discardPeriodicTasks();
  }));
});
