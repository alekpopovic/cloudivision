import { HttpClient, HttpErrorResponse, HttpParams } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';
import { Observable, catchError, map, of, shareReplay, switchMap, throwError } from 'rxjs';

import { environment } from '../../environments/environment';
import { ApiError, ApprovalActionRequest, BuildRun, Environment, LogsResponse, Page, PipelineTemplate, Principal, Project, ProviderHealthResult, ProviderSummary, Release, ReleasePromoteRequest, ReleaseRollbackRequest, Repository } from './models';

interface RuntimeConfig {
  apiBaseUrl?: string;
}

@Injectable({ providedIn: 'root' })
export class ApiClient {
  private readonly http = inject(HttpClient);
  // TODO(observability): add an Angular HttpInterceptor for trace context propagation when backend tracing is enabled.
  private readonly config$ = this.http.get<RuntimeConfig>('/assets/config.json').pipe(
    catchError(() => of({} as RuntimeConfig)),
    map((config) => ({
      apiBaseUrl: config.apiBaseUrl ?? environment.apiBaseUrl ?? ''
    })),
    shareReplay({ bufferSize: 1, refCount: true })
  );

  projects(): Observable<Project[]> {
    return this.get<Project[]>('/api/v1/projects');
  }

  currentUser(): Observable<Principal> {
    return this.get<Principal>('/api/v1/auth/me');
  }

  createProject(body: { name: string; namespace?: string; spec: Project['spec'] }): Observable<Project> {
    return this.post<Project>('/api/v1/projects', body);
  }

  project(name: string, namespace?: string): Observable<Project> {
    return this.get<Project>(`/api/v1/projects/${name}`, namespace ? { namespace } : undefined);
  }

  purgeProjectCache(name: string, namespace?: string, repository?: string): Observable<{ project: string; repository?: string; purged: boolean }> {
    return this.post<{ project: string; repository?: string; purged: boolean }>(
      `/api/v1/projects/${name}/cache/purge${this.queryString({ namespace, repository })}`,
      undefined
    );
  }

  repositories(): Observable<Repository[]> {
    return this.get<Repository[]>('/api/v1/repositories');
  }

  createRepository(body: { name: string; namespace?: string; spec: Repository['spec'] }): Observable<Repository> {
    return this.post<Repository>('/api/v1/repositories', body);
  }

  pipelineTemplates(): Observable<PipelineTemplate[]> {
    return this.get<PipelineTemplate[]>('/api/v1/pipeline-templates');
  }

  createPipelineTemplate(body: { name: string; namespace?: string; spec: PipelineTemplate['spec'] }): Observable<PipelineTemplate> {
    return this.post<PipelineTemplate>('/api/v1/pipeline-templates', body);
  }

  buildRuns(params?: Record<string, string>): Observable<BuildRun[]> {
    return this.buildRunPage(params).pipe(map((page) => page.items));
  }

  buildRunPage(params?: Record<string, string>): Observable<Page<BuildRun>> {
    return this.get<Page<BuildRun>>('/api/v1/build-runs', { limit: '50', ...(params ?? {}) });
  }

  createBuildRun(body: { name: string; namespace?: string; spec: BuildRun['spec'] }): Observable<BuildRun> {
    return this.post<BuildRun>('/api/v1/build-runs', body);
  }

  buildRun(namespace: string, name: string): Observable<BuildRun> {
    return this.get<BuildRun>(`/api/v1/build-runs/${namespace}/${name}`);
  }

  buildRunLogs(namespace: string, name: string, tailLines = 200): Observable<LogsResponse> {
    return this.get<LogsResponse>(`/api/v1/build-runs/${namespace}/${name}/logs`, { tailLines: String(tailLines) });
  }

  buildRunAction(namespace: string, name: string, action: 'cancel' | 'retry' | 'rerun'): Observable<BuildRun> {
    return this.post<BuildRun>(`/api/v1/build-runs/${namespace}/${name}/${action}`, undefined);
  }

  downloadArtifact(namespace: string, buildRun: string, artifactName: string): Observable<Blob> {
    return this.config$.pipe(
      switchMap((config) => this.http.get(`${config.apiBaseUrl}/api/v1/build-runs/${namespace}/${buildRun}/artifacts/${encodeURIComponent(artifactName)}`, { responseType: 'blob' })),
      catchError((error) => throwError(() => this.toApiError(error)))
    );
  }

  environments(): Observable<Environment[]> {
    return this.get<Environment[]>('/api/v1/environments');
  }

  releases(): Observable<Release[]> {
    return this.releasePage().pipe(map((page) => page.items));
  }

  releasePage(params?: Record<string, string>): Observable<Page<Release>> {
    return this.get<Page<Release>>('/api/v1/releases', { limit: '50', ...(params ?? {}) });
  }

	providers(): Observable<ProviderSummary[]> {
		return this.get<ProviderSummary[]>('/api/v1/providers');
	}

	providerHealth(): Observable<ProviderHealthResult[]> {
		return this.get<ProviderHealthResult[]>('/api/v1/providers/health');
	}

  approveRelease(namespace: string, name: string, body: ApprovalActionRequest): Observable<Release> {
    return this.post<Release>(`/api/v1/releases/${namespace}/${name}/approve`, body);
  }

  rejectRelease(namespace: string, name: string, body: ApprovalActionRequest): Observable<Release> {
    return this.post<Release>(`/api/v1/releases/${namespace}/${name}/reject`, body);
  }

  promoteRelease(namespace: string, name: string, body: ReleasePromoteRequest): Observable<Release> {
    return this.post<Release>(`/api/v1/releases/${namespace}/${name}/promote`, body);
  }

  rollbackRelease(namespace: string, name: string, body: ReleaseRollbackRequest): Observable<Release> {
    return this.post<Release>(`/api/v1/releases/${namespace}/${name}/rollback`, body);
  }

  webhookUrl(provider: Repository['spec']['provider'], repositoryName: string): Observable<string> {
    return this.config$.pipe(map((config) => `${config.apiBaseUrl}/api/v1/webhooks/${provider}/${repositoryName}`));
  }

  private get<T>(path: string, params?: Record<string, string>): Observable<T> {
    return this.config$.pipe(
      switchMap((config) => this.http.get<T>(`${config.apiBaseUrl}${path}`, { params: new HttpParams({ fromObject: params ?? {} }) })),
      catchError((error) => throwError(() => this.toApiError(error)))
    );
  }

  private post<T>(path: string, body: unknown): Observable<T> {
    return this.config$.pipe(
      switchMap((config) => this.http.post<T>(`${config.apiBaseUrl}${path}`, body)),
      catchError((error) => throwError(() => this.toApiError(error)))
    );
  }

  private queryString(values: Record<string, string | undefined>): string {
    const params = new URLSearchParams();
    for (const [key, value] of Object.entries(values)) {
      if (value) params.set(key, value);
    }
    const encoded = params.toString();
    return encoded ? `?${encoded}` : '';
  }

  private toApiError(error: unknown): ApiError {
    if (error instanceof HttpErrorResponse && error.error && typeof error.error === 'object') {
      const body = error.error as Partial<ApiError>;
      return {
        code: body.code ?? `http_${error.status}`,
        message: body.message ?? error.message,
        requestId: body.requestId ?? error.headers.get('X-Request-ID') ?? undefined
      };
    }
    return { code: 'request_failed', message: error instanceof Error ? error.message : 'Request failed' };
  }
}
