# Detailed v1alpha1 CRD models

User prompt:

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
- credentialSecretRef:
  - name string
  - key string optional
- webhook:
  - enabled bool
  - secretRef:
    - name string
    - key string
  - events []string
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
- triggeredBy:
  - type enum: webhook, manual, schedule, api
  - actor string optional
  - eventID string optional
- image:
  - repository string
  - tag string optional
  - digest string optional
- params map[string]string
- executor enum: job, tekton default job
- gitOps:
  - enabled bool
  - repoURL string optional
  - branch string optional
  - path string optional
  - strategy enum: helm-values, kustomize-image, raw-yaml
  - environmentRef string optional

Status:
- phase enum: Pending, Queued, Running, Succeeded, Failed, Cancelled
- conditions []metav1.Condition
- observedGeneration int64
- startedAt metav1.Time optional
- completedAt metav1.Time optional
- jobRef:
  - name string
  - namespace string
- pipelineRunRef:
  - name string
  - namespace string
- image:
  - repository string
  - tag string
  - digest string
- failure:
  - reason string
  - message string
- log:
  - podName string
  - containerName string
  - lastLines []string optional, max 20

Kind: Environment

Spec:
- projectRef string
- displayName string
- namespace string
- type enum: dev, staging, production, custom
- requiresApproval bool
- gitOps:
  - provider enum: argocd, flux, generic
  - applicationName string optional
  - namespace string optional

Status:
- phase enum: Pending, Ready, Error
- syncStatus string optional
- healthStatus string optional
- conditions []metav1.Condition
- observedGeneration int64

Kind: Release

Spec:
- projectRef string
- environmentRef string
- buildRunRef string
- image:
  - repository string
  - tag string
  - digest string
- approval:
  - required bool
  - approvedBy string optional
  - approvedAt metav1.Time optional
- strategy enum: gitops

Status:
- phase enum: Pending, AwaitingApproval, Deploying, Deployed, Failed, RolledBack
- conditions []metav1.Condition
- observedGeneration int64
- startedAt metav1.Time optional
- completedAt metav1.Time optional
- gitCommit string optional
- deployment:
  - provider string
  - applicationName string
  - syncStatus string
  - healthStatus string

Implementation steps:
1. Implement Go types with kubebuilder markers.
2. Add enum validation where appropriate.
3. Add required/optional markers.
4. Add printcolumn markers for important status fields.
5. Add helper functions in /internal/domain for:
   - setting conditions
   - computing phase from conditions
   - marking BuildRun started/succeeded/failed
   - validating phase transitions
6. Add unit tests for helper functions.
7. Update generated deepcopy files if the tooling exists.
8. Update CRD YAML if controller-gen exists.
9. Add sample YAML files in /deploy/examples for every Kind.

Acceptance criteria:
- go test ./... passes.
- API types are idiomatic and avoid map[string]interface{} unless truly justified.
- Status conditions use metav1.Condition.
- CRD status does not depend on a database.
- Sample YAML exists for Project, Repository, PipelineTemplate, BuildRun, Environment and Release.
```
