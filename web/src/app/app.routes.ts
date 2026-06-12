import { Routes } from '@angular/router';
import { DashboardPageComponent } from './features/dashboard/dashboard-page.component';
import { BuildRunDetailPageComponent } from './features/build-runs/build-run-detail-page.component';
import { BuildRunsPageComponent } from './features/build-runs/build-runs-page.component';
import { EnvironmentsPageComponent } from './features/environments/environments-page.component';
import { PipelineTemplatesPageComponent } from './features/pipeline-templates/pipeline-templates-page.component';
import { ProjectDetailPageComponent } from './features/projects/project-detail-page.component';
import { ProjectsPageComponent } from './features/projects/projects-page.component';
import { ReleasesPageComponent } from './features/releases/releases-page.component';
import { RepositoriesPageComponent } from './features/repositories/repositories-page.component';

export const routes: Routes = [
  { path: '', pathMatch: 'full', redirectTo: 'dashboard' },
  { path: 'dashboard', component: DashboardPageComponent },
  { path: 'projects', component: ProjectsPageComponent },
  { path: 'projects/:namespace/:name', component: ProjectDetailPageComponent },
  { path: 'repositories', component: RepositoriesPageComponent },
  { path: 'pipeline-templates', component: PipelineTemplatesPageComponent },
  { path: 'build-runs', component: BuildRunsPageComponent },
  { path: 'build-runs/:namespace/:name', component: BuildRunDetailPageComponent },
  { path: 'environments', component: EnvironmentsPageComponent },
  { path: 'releases', component: ReleasesPageComponent },
  { path: '**', redirectTo: 'dashboard' }
];
