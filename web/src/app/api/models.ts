export interface ApiError {
  code: string;
  message: string;
  requestId?: string;
}

export interface Condition {
  type: string;
  status: string;
  reason?: string;
  message?: string;
  lastTransitionTime?: string;
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
    webhook?: { enabled: boolean; events?: string[] };
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
    steps?: PipelineStep[];
    build?: {
      enabled: boolean;
      contextDir?: string;
      dockerfile?: string;
      builder?: 'buildkit' | 'buildah' | 'none';
      image?: string;
      push?: boolean;
    };
    resources?: Record<string, string | number>;
    security?: Record<string, boolean>;
    supplyChain?: {
      generateSBOM?: boolean;
      scanImage?: boolean;
      signImage?: boolean;
      requireSignedBaseImages?: boolean;
    };
  };
  status?: { phase?: string; conditions?: Condition[] };
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
}

export interface BuildRun {
  name: string;
  namespace: string;
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
    gitOps?: Record<string, string | boolean>;
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
    };
    failure?: { reason?: string; message?: string };
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
    gitOps?: { provider?: 'argocd' | 'flux' | 'generic'; applicationName?: string; namespace?: string };
    policy?: {
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
    approval?: { required: boolean; approvedBy?: string; approvedAt?: string };
    strategy: 'gitops';
  };
  status?: {
    phase?: string;
    gitCommit?: string;
    deployment?: { provider?: string; applicationName?: string; syncStatus?: string; healthStatus?: string };
    conditions?: Condition[];
  };
}

export interface LogsResponse {
  namespace: string;
  buildRun: string;
  podName?: string;
  lines: string[];
}
