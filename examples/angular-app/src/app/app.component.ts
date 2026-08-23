import { Component } from '@angular/core';

@Component({
  selector: 'app-root',
  standalone: true,
  template: `<main><p>cloudivision example</p><h1>Delivery dashboard</h1><span class="status">Pipeline ready</span></main>`,
  styles: [`main{max-width:48rem;margin:5rem auto;padding:2rem;background:white;border-radius:.75rem}p{color:#0369a1}.status{color:#047857}`]
})
export class AppComponent {}
