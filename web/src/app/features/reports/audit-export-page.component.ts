import { CommonModule } from '@angular/common';
import { Component, inject } from '@angular/core';
import { FormBuilder, ReactiveFormsModule } from '@angular/forms';

import { ApiClient } from '../../api/client';
import { PageHeaderComponent } from '../../shared/page-header.component';

@Component({selector:'app-audit-export-page',standalone:true,imports:[CommonModule,ReactiveFormsModule,PageHeaderComponent],template:`
  <app-page-header title="Audit export" description="Export filtered, redacted audit history as JSON or CSV." />
  <form [formGroup]="form" class="grid gap-3 rounded-md border border-slate-200 bg-white p-5 md:grid-cols-3">
    <label *ngFor="let field of fields" class="text-sm font-medium text-slate-700">{{ field.label }}<input [type]="field.type || 'text'" [formControlName]="field.name" class="mt-1 w-full rounded-md border border-slate-300 px-3 py-2 font-normal"></label>
    <div class="flex items-end gap-2 md:col-span-3"><button type="button" (click)="export('json')" class="rounded-md bg-brand-700 px-4 py-2 text-sm font-semibold text-white">Export JSON</button><button type="button" (click)="export('csv')" class="rounded-md border border-slate-300 px-4 py-2 text-sm font-semibold">Export CSV</button><span class="text-sm text-rose-700" *ngIf="error">{{ error }}</span></div>
  </form>
`})
export class AuditExportPageComponent {
  private readonly api=inject(ApiClient);private readonly fb=inject(FormBuilder);error='';
  readonly fields=[{name:'organization',label:'Organization',type:'text'},{name:'project',label:'Project',type:'text'},{name:'repository',label:'Repository',type:'text'},{name:'buildRun',label:'BuildRun',type:'text'},{name:'release',label:'Release',type:'text'},{name:'actor',label:'Actor',type:'text'},{name:'eventType',label:'Event type',type:'text'},{name:'from',label:'From',type:'datetime-local'},{name:'to',label:'To',type:'datetime-local'}] as const;
  readonly form=this.fb.group({organization:'',project:'',repository:'',buildRun:'',release:'',actor:'',eventType:'',from:'',to:''});
  export(format:'json'|'csv'):void{this.error='';this.api.auditExport(format,this.filters()).subscribe({next:(blob)=>downloadBlob(blob,`audit-events.${format}`),error:(error)=>this.error=error.message});}
  private filters():Record<string,string>{return Object.fromEntries(Object.entries(this.form.getRawValue()).filter(([,value])=>!!value).map(([key,value])=>[key,key==='from'||key==='to'?new Date(value!).toISOString():value!]))}
}

export function downloadBlob(blob:Blob,name:string):void{const url=URL.createObjectURL(blob);const anchor=document.createElement('a');anchor.href=url;anchor.download=name;anchor.click();URL.revokeObjectURL(url);}
