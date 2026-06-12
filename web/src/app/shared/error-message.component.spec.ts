import { ComponentFixture, TestBed } from '@angular/core/testing';

import { ErrorMessageComponent } from './error-message.component';

describe('ErrorMessageComponent', () => {
  let fixture: ComponentFixture<ErrorMessageComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({ imports: [ErrorMessageComponent] }).compileComponents();
    fixture = TestBed.createComponent(ErrorMessageComponent);
  });

  it('displays backend code, message and request ID', () => {
    fixture.componentRef.setInput('error', { code: 'unauthorized', message: 'not allowed', requestId: 'req-1' });
    fixture.detectChanges();
    expect(fixture.nativeElement.textContent).toContain('unauthorized');
    expect(fixture.nativeElement.textContent).toContain('not allowed');
    expect(fixture.nativeElement.textContent).toContain('req-1');
  });
});
