import { ComponentFixture, TestBed } from '@angular/core/testing';

import { ErrorMessageComponent } from './error-message.component';

describe('ErrorMessageComponent', () => {
  let fixture: ComponentFixture<ErrorMessageComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({ imports: [ErrorMessageComponent] }).compileComponents();
    fixture = TestBed.createComponent(ErrorMessageComponent);
  });

  it('displays backend code and message', () => {
    fixture.componentRef.setInput('error', { code: 'unauthorized', message: 'not allowed' });
    fixture.detectChanges();
    expect(fixture.nativeElement.textContent).toContain('unauthorized');
    expect(fixture.nativeElement.textContent).toContain('not allowed');
  });
});
