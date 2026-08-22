import { ComponentFixture, TestBed, discardPeriodicTasks, fakeAsync, tick } from '@angular/core/testing';
import { of } from 'rxjs';

import { ApiClient } from '../../api/client';
import { ProvidersPageComponent } from './providers-page.component';

describe('ProvidersPageComponent', () => {
  let fixture: ComponentFixture<ProvidersPageComponent>;

  beforeEach(async () => {
		await TestBed.configureTestingModule({
			imports: [ProvidersPageComponent],
			providers: [{
				provide: ApiClient,
				useValue: {
					providerHealth: () => of([{
						name: 'generic', type: 'git',
						capabilities: [{ name: 'clone', description: 'Clone' }],
						health: { healthy: true, message: 'ready', checkedAt: '2026-08-22T00:00:00Z' }
					}])
				}
			}]
		}).compileComponents();
		fixture = TestBed.createComponent(ProvidersPageComponent);
	});

	it('shows provider health and capabilities', fakeAsync(() => {
    fixture.detectChanges(); tick(0); fixture.detectChanges();
    expect(fixture.nativeElement.textContent).toContain('generic');
    expect(fixture.nativeElement.textContent).toContain('Healthy');
    expect(fixture.nativeElement.textContent).toContain('clone');
    discardPeriodicTasks();
	}));
});
