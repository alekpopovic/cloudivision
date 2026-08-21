# Prompt 03 — Detailed CRD Models

Phase: Kubernetes API

```text
Continue working on the cloudivision project.

Task:
Implement detailed v1alpha1 CRD models for cloudivision.

Kind: Project
Spec:
- displayName string
- description string optional
- ownerTeam string
- namespace string
- defaultRegistry string
- defaultBranch string default "main"
- serviceAccountName string optional
- isolation:
  - createNamespace bool
  - podSecurityLevel enum: baseline, restricted
  - networkPolicyMode enum: disabled, defaultDeny, egressAllowList
Status:
- phase enum: Pending, Ready, Error
- namespaceReady bool
- conditions []metav1.Condition
- observedGeneration int64

Kind: Repository
Spec:
- projectRef string
- provider enum: github, gitlab, gitea, generic
- url string
- defaultBranch string
- credentialSecretRef: name string, key string optional
- webhook: enabled bool, secretRef name/key, events []string
- pipelineTemplateRef string
Status:
- phase enum: Pending, Ready, Error
- lastWebhookAt metav1.Time optional
- conditions []metav1.Condition
- observedGeneration int64

Kind: PipelineTemplate
Spec:
- projectRef string optional
- description string optional
- params []ParamSpec
- steps []PipelineStep
- build:
  - enabled bool
  - contextDir string default "."
  - dockerfile string default "Dockerfile"
  - builder enum: buildkit, buildah, none
  - image string optional
  - push bool default true
- resources:
  - cpuRequest string
  - cpuLimit string
  - memoryRequest string
  - memoryLimit string
  - timeoutSeconds int
- security:
  - allowPrivileged bool default false
  - runAsNonRoot bool default true
  - readOnlyRootFilesystem bool default false
Status:
- phase enum: Ready, Error
- conditions []metav1.Condition
- observedGeneration int64

ParamSpec:
- name string
- description string optional
- default string optional
- required bool

PipelineStep:
- name string
- image string
- command []string
- args []string
- workingDir string optional
- env []corev1.EnvVar
- timeoutSeconds int optional
- continueOnError bool default false

Kind: BuildRun
Spec:
- projectRef string
- repositoryRef string
- pipelineTemplateRef string
- revision string
- branch string optional
- commitSHA string optional
- triggeredBy: type enum webhook/manual/schedule/api, actor optional, eventID optional
- image: repository, tag optional, digest optional
- params map[string]string
- executor enum: job, tekton default job
- gitOps: enabled bool, repoURL optional, branch optional, path optional, strategy enum helm-values/kustomize-image/raw-yaml, environmentRef optional
Status:
- phase enum: Pending, Queued, Running, Succeeded, Failed, Cancelled
- conditions []metav1.Condition
- observedGeneration int64
- startedAt/completedAt optional
- jobRef name/namespace
- pipelineRunRef name/namespace
- image repository/tag/digest
- failure reason/message
- log podName/containerName/lastLines max 20 optional

Kind: Environment
Spec:
- projectRef string
- displayName string
- namespace string
- type enum: dev, staging, production, custom
- requiresApproval bool
- gitOps: provider enum argocd/flux/generic, applicationName optional, namespace optional
Status:
- phase enum: Pending, Ready, Error
- syncStatus optional
- healthStatus optional
- conditions []metav1.Condition
- observedGeneration int64

Kind: Release
Spec:
- projectRef string
- environmentRef string
- buildRunRef string
- image repository/tag/digest
- approval required bool, approvedBy optional, approvedAt optional
- strategy enum: gitops
Status:
- phase enum: Pending, AwaitingApproval, Deploying, Deployed, Failed, RolledBack
- conditions []metav1.Condition
- observedGeneration int64
- startedAt/completedAt optional
- gitCommit optional
- deployment provider/applicationName/syncStatus/healthStatus

Implementation steps:
1. Implement Go types with kubebuilder markers.
2. Add enum validation, required/optional markers and printcolumn markers.
3. Add helper functions in /internal/domain for setting conditions, computing phase, marking BuildRun started/succeeded/failed and validating phase transitions.
4. Add unit tests for helper functions.
5. Update deepcopy and CRD YAML if tooling exists.
6. Add sample YAML files in /deploy/examples for every Kind.

Acceptance criteria:
- go test ./... passes.
- API types are idiomatic and avoid map[string]interface{} unless truly justified.
- Status conditions use metav1.Condition.
- CRD status does not depend on a database.
- Sample YAML exists for Project, Repository, PipelineTemplate, BuildRun, Environment and Release.
```
