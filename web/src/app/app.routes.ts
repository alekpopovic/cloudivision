import { Routes } from '@angular/router';
import { DashboardPageComponent } from './features/dashboard/dashboard-page.component';
import { BuildRunDetailPageComponent } from './features/build-runs/build-run-detail-page.component';
import { BuildRunsPageComponent } from './features/build-runs/build-runs-page.component';
import { EnvironmentsPageComponent } from './features/environments/environments-page.component';
import { PipelineTemplatesPageComponent } from './features/pipeline-templates/pipeline-templates-page.component';
import { ProjectDetailPageComponent } from './features/projects/project-detail-page.component';
import { ProjectsPageComponent } from './features/projects/projects-page.component';
import { ReleasesPageComponent } from './features/releases/releases-page.component';
import { ReleaseDetailPageComponent } from './features/releases/release-detail-page.component';
import { RepositoriesPageComponent } from './features/repositories/repositories-page.component';
import { FirstRunWizardComponent } from './features/first-run/first-run-wizard.component';
import { ProvidersPageComponent } from './features/providers/providers-page.component';
import { OrganizationsPageComponent } from './features/organizations/organizations-page.component';
import { AuditExportPageComponent } from './features/reports/audit-export-page.component';
import { ReportsPageComponent } from './features/reports/reports-page.component';
import { PluginsPageComponent } from './features/plugins/plugins-page.component';

export const routes: Routes = [
  { path: '', pathMatch: 'full', redirectTo: 'dashboard' },
  { path: 'dashboard', component: DashboardPageComponent },
  { path: 'first-run', component: FirstRunWizardComponent },
  { path: 'projects', component: ProjectsPageComponent },
  { path: 'organization', component: OrganizationsPageComponent },
  { path: 'audit', component: AuditExportPageComponent },
  { path: 'reports', component: ReportsPageComponent },
  { path: 'projects/:namespace/:name', component: ProjectDetailPageComponent },
  { path: 'repositories', component: RepositoriesPageComponent },
  { path: 'pipeline-templates', component: PipelineTemplatesPageComponent },
	{ path: 'providers', component: ProvidersPageComponent },
  { path: 'plugins', component: PluginsPageComponent },
  { path: 'build-runs', component: BuildRunsPageComponent },
  { path: 'build-runs/:namespace/:name', component: BuildRunDetailPageComponent },
  { path: 'environments', component: EnvironmentsPageComponent },
  { path: 'releases', component: ReleasesPageComponent },
  { path: 'releases/:namespace/:name', component: ReleaseDetailPageComponent },
  { path: '**', redirectTo: 'dashboard' }
];
