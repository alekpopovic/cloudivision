export interface ApiError {
  code: string;
  message: string;
  requestId?: string;
  violations?: Array<{ policy: string; severity: string; message: string; fieldPath?: string }>;
}

export interface Page<T> {
  items: T[];
  nextPageToken?: string;
  totalCount: number;
  limit: number;
}

export interface ApprovalActionRequest {
  actor: string;
  comment?: string;
}

export interface ReleasePromoteRequest { targetEnvironmentRef: string; actor?: string; }
export interface ReleaseRollbackRequest { targetReleaseRef: string; actor?: string; reason?: string; }

export type Role = 'admin' | 'project-admin' | 'developer' | 'viewer';

export interface Principal {
  subject: string;
  email?: string;
  groups?: string[];
  displayName?: string;
  roles?: Role[];
  devMode?: boolean;
}

export interface Condition {
  type: string;
  status: string;
  reason?: string;
  message?: string;
  lastTransitionTime?: string;
}

export interface PolicyDecision {
  allowed: boolean;
  reason?: string;
  message?: string;
  evaluatedAt?: string;
  violations?: Array<{ policy: string; severity: string; message: string; fieldPath?: string }>;
}

export interface Project {
  name: string;
  namespace: string;
  spec: {
    displayName: string;
    description?: string;
    ownerTeam: string;
    namespace: string;
    defaultRegistry: string;
    defaultBranch?: string;
    serviceAccountName?: string;
    registry?: {
      provider?: 'generic' | 'ghcr' | 'gitlab' | 'harbor' | 'ecr' | 'gcr' | 'acr';
      imagePrefix?: string;
      credentialSecretRef?: { name: string; key?: string };
    };
    imageTagPolicy?: { defaultTagTemplate?: string };
    quotas?: {
      maxConcurrentBuildRuns?: number;
      maxQueuedBuildRuns?: number;
      maxCPU?: string;
      maxMemory?: string;
      maxBuildDurationSeconds?: number;
      maxArtifactsSize?: string;
      maxLogSize?: string;
    };
    notifications?: {
      enabled: boolean;
      provider: 'webhook' | 'slack' | 'teams' | 'email';
      secretRef?: { name: string; key?: string };
      events?: string[];
      filters?: { project?: string; repository?: string; environment?: string; phase?: string };
    };
    isolation?: {
      createNamespace: boolean;
      podSecurityLevel: 'baseline' | 'restricted';
      networkPolicyMode: 'disabled' | 'defaultDeny' | 'egressAllowList';
    };
  };
  status?: { phase?: string; namespaceReady?: boolean; conditions?: Condition[] };
}

export interface Repository {
  name: string;
  namespace: string;
  spec: {
    projectRef: string;
    provider: 'github' | 'gitlab' | 'gitea' | 'generic';
    url: string;
    defaultBranch: string;
    pipelineTemplateRef: string;
    webhook?: {
      enabled: boolean;
      events?: string[];
      branchFilters?: { include?: string[]; exclude?: string[] };
      tagFilters?: { include?: string[]; exclude?: string[] };
      pullRequest?: { enabled?: boolean; events?: string[]; buildForks?: boolean; requireTrustedActor?: boolean };
    };
  };
  status?: { phase?: string; lastWebhookAt?: string; conditions?: Condition[] };
}

export interface PipelineTemplate {
  name: string;
  namespace: string;
  spec: {
    projectRef?: string;
    description?: string;
    params?: Array<{ name: string; description?: string; default?: string; required: boolean }>;
    cache?: {
      enabled?: boolean;
      mode?: 'pvc' | 'registry' | 'object-storage';
      key?: string;
      paths?: string[];
      restoreKeys?: string[];
      ttlSeconds?: number;
    };
    steps?: PipelineStep[];
    build?: {
      enabled: boolean;
      contextDir?: string;
      dockerfile?: string;
      builder?: 'buildkit' | 'buildah' | 'none';
      image?: string;
      push?: boolean;
      buildArgs?: Record<string, string>;
      target?: string;
      platforms?: string[];
      labels?: Record<string, string>;
      cache?: { enabled?: boolean; mode?: 'inline' | 'registry' | 'local'; ref?: string };
    };
    resources?: Record<string, string | number>;
    security?: Record<string, boolean>;
    supplyChain?: {
      generateSBOM?: boolean;
      scanImage?: boolean;
      signImage?: boolean;
      requireSignedBaseImages?: boolean;
			sbomAdapter?: 'noop' | 'syft';
			scannerAdapter?: 'noop' | 'grype';
			signerAdapter?: 'noop' | 'cosign';
			provenanceAdapter?: 'noop' | 'json';
			cosignKeyless?: boolean;
			signingKeySecretRef?: { name: string; key: string };
    };
  };
  status?: { phase?: string; conditions?: Condition[] };
}

export interface CatalogPipelineTemplate {
  name: string;
  version: string;
  description: string;
  parameters?: Array<{ name: string; description?: string; default?: string; required: boolean }>;
  spec: PipelineTemplate['spec'];
}

export interface PipelineStep {
  name: string;
  image: string;
  command?: string[];
  args?: string[];
  workingDir?: string;
  env?: Array<{ name: string; value?: string }>;
  timeoutSeconds?: number;
  continueOnError?: boolean;
  artifacts?: { paths: string[]; optional?: boolean; retentionDays?: number };
}

export interface BuildRun {
  name: string;
  namespace: string;
  annotations?: Record<string, string>;
  spec: {
    projectRef: string;
    repositoryRef: string;
    pipelineTemplateRef: string;
    revision: string;
    branch?: string;
    commitSHA?: string;
    triggeredBy: { type: 'webhook' | 'manual' | 'schedule' | 'api'; actor?: string; eventID?: string };
    image: { repository: string; tag?: string; digest?: string };
    params?: Record<string, string>;
    executor?: 'job' | 'tekton';
    gitOps?: {
      enabled?: boolean;
      repoURL?: string;
      branch?: string;
      path?: string;
      strategy?: 'helm-values' | 'kustomize-image' | 'raw-yaml';
      environmentRef?: string;
      valuesFile?: string;
      imageRepositoryField?: string;
      imageTagField?: string;
      imageDigestField?: string;
      kustomize?: { kustomizationFile?: string; imageName?: string };
      rawYaml?: { files?: string[]; workloadKind?: 'Deployment' | 'StatefulSet' | 'DaemonSet' | 'CronJob'; workloadName?: string; containerName?: string };
    };
  };
  status?: {
    phase?: string;
    conditions?: Condition[];
    startedAt?: string;
    completedAt?: string;
    image?: { repository?: string; tag?: string; digest?: string };
    supplyChain?: {
      sbomPath?: string;
      sbomDigest?: string;
      signatureRef?: string;
      provenanceRef?: string;
      scannerResultsRef?: string;
			criticalVulnerabilities?: number;
			highVulnerabilities?: number;
			mediumVulnerabilities?: number;
			lowVulnerabilities?: number;
    };
    policy?: PolicyDecision;
    failure?: { reason?: string; message?: string };
    artifacts?: Array<{ name: string; path: string; type?: string; size: number; digest: string; ref: string; createdAt?: string }>;
  };
}

export interface Environment {
  name: string;
  namespace: string;
  spec: {
    projectRef: string;
    displayName: string;
    namespace: string;
    type: 'dev' | 'staging' | 'production' | 'custom';
    requiresApproval: boolean;
    gitOps?: { provider?: 'argocd' | 'flux' | 'generic'; applicationName?: string; namespace?: string; resourceKind?: 'Kustomization' | 'HelmRelease' };
    policy?: {
      requireImageDigest?: boolean;
      allowLatest?: boolean;
      requireSignedImages?: boolean;
      requireSBOM?: boolean;
      blockCriticalVulnerabilities?: boolean;
    };
  };
  status?: { phase?: string; syncStatus?: string; healthStatus?: string; conditions?: Condition[] };
}

export interface Release {
  name: string;
  namespace: string;
  spec: {
    projectRef: string;
    environmentRef?: string;
    buildRunRef: string;
    image: { repository: string; tag?: string; digest?: string };
    approval?: {
      required: boolean;
      approvedBy?: string;
      approvedAt?: string;
      rejectedBy?: string;
      rejectedAt?: string;
      comment?: string;
    };
    strategy: 'gitops';
		promotionMode?: 'direct-commit' | 'pull-request';
		pullRequest?: {
			titleTemplate?: string;
			bodyTemplate?: string;
			targetBranch?: string;
			reviewers?: string[];
			labels?: string[];
		};
		promotedFrom?: string;
		rollbackOf?: string;
		rollbackTo?: string;
  };
  status?: {
    phase?: string;
    gitCommit?: string;
		failure?: { reason?: string; message?: string };
    deployment?: { provider?: string; applicationName?: string; syncStatus?: string; healthStatus?: string; operationPhase?: string; observedRevision?: string; observedAt?: string };
		pullRequest?: {
			provider?: string;
			url?: string;
			reference?: string;
			headBranch?: string;
			targetBranch?: string;
			mergeStatus?: string;
		};
    policy?: PolicyDecision;
    conditions?: Condition[];
  };
}

export interface LogsResponse {
  namespace: string;
  buildRun: string;
  podName?: string;
  backend?: string;
  ref?: string;
  lines: string[];
}

export interface ProviderCapability {
  name: string;
  description: string;
}

export interface ProviderSummary {
  name: string;
  type: string;
  capabilities: ProviderCapability[];
}

export interface ProviderHealthResult extends ProviderSummary {
  health: { healthy: boolean; message: string; checkedAt: string };
}
