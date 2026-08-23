# cicd.cloudivision.io/v1alpha1 API

`v1alpha1` is the experimental cloudivision Kubernetes API. All resources are namespaced and expose a status subresource. References in this version are string names and resolve within the referring object's namespace unless stated otherwise.

## Shared conventions

- `status.observedGeneration` identifies the spec generation reflected by status.
- `status.conditions` uses Kubernetes `metav1.Condition`. Condition consumers should select by `type`, not array position.
- Terminal BuildRun phases are `Succeeded`, `Failed` and `Cancelled`.
- Timestamps are RFC 3339 Kubernetes timestamps.
- Secret references contain Secret names/keys, never secret values.

Defaults currently generated into the CRDs include `Project.spec.defaultBranch: main`, `PipelineTemplate.spec.build.contextDir: .`, `dockerfile: Dockerfile`, `push: true`, `security.runAsNonRoot: true`, and `BuildRun.spec.executor: job`. Kubernetes applies schema defaults on admission; Go clients constructing objects in memory must not assume defaults have run.

## Project

Project defines the team boundary and desired namespace isolation.

Important spec fields:

| Field | Meaning |
|---|---|
| `displayName`, `description`, `ownerTeam` | Human ownership metadata. |
| `namespace` | Namespace where runner RBAC/workloads are managed. |
| `defaultRegistry`, `defaultBranch` | Artifact/source defaults. |
| `registry.provider`, `registry.imagePrefix` | Registry adapter and resolved image prefix. |
| `registry.credentialSecretRef` | Dedicated namespaced registry Secret name and optional Docker config key. |
| `serviceAccountName` | Optional runner ServiceAccount override. |
| `isolation.createNamespace` | Whether the controller creates the target namespace. |
| `isolation.podSecurityLevel` | `baseline` or `restricted`. |
| `isolation.networkPolicyMode` | `disabled`, `defaultDeny` or `egressAllowList`. |

Status phases are `Pending`, `Ready` and `Error`. `namespaceReady` and the Ready/Failed condition explain whether namespace, runner RBAC and network policy reconciliation succeeded.

Implemented registry providers and Secret formats are documented in the
[registry provider](../registry/providers.md) and
[credential](../registry/credentials.md) guides.

```yaml
apiVersion: cicd.cloudivision.io/v1alpha1
kind: Project
metadata: {name: demo, namespace: demo-ci}
spec:
  displayName: Demo
  ownerTeam: platform
  namespace: demo-ci
  defaultRegistry: registry.example.com/demo
  isolation:
    createNamespace: false
    podSecurityLevel: restricted
    networkPolicyMode: disabled
```

## Repository

Repository connects source control to a Project and default PipelineTemplate.

Required identity fields are `projectRef`, `provider`, `url`, `defaultBranch` and `pipelineTemplateRef`. Provider is `github`, `gitlab`, `gitea` or `generic`. `credentialSecretRef` is optional. `webhook.enabled` controls event ingestion; enabled webhooks require a `secretRef` and can list accepted events.

Status phases are `Pending`, `Ready` and `Error`. `lastWebhookAt` records the latest accepted delivery when updated, while conditions and `observedGeneration` describe readiness.

```yaml
apiVersion: cicd.cloudivision.io/v1alpha1
kind: Repository
metadata: {name: demo, namespace: demo-ci}
spec:
  projectRef: demo
  provider: github
  url: https://github.com/example/demo.git
  defaultBranch: main
  pipelineTemplateRef: demo
  webhook:
    enabled: true
    secretRef: {name: demo-webhook, key: secret}
    events: [push]
```

## PipelineTemplate

PipelineTemplate is reusable execution policy. `params` declares inputs. Ordered `steps` contain `name`, execution `image`, `command`, `args`, relative/absolute `workingDir`, literal environment values, timeout and continue-on-error behavior.

`build` configures optional image creation with builder `buildkit`, `buildah` or `none`, context, Dockerfile, image, push behavior, build arguments, target stage, platforms, labels and optional inline/registry/local cache. Registry and local cache modes require a cache reference. `resources` supplies runner CPU/memory and total timeout. `security` controls privilege request, non-root and read-only-root-filesystem intent. `supplyChain` requests SBOM, scanning, signing and signed-base-image policy hooks. BuildKit details and status conditions are documented in the [BuildKit guide](../build/buildkit.md).

Status is `Ready` or `Error` with conditions and observed generation. The Job executor in v0.1 executes commands inside the runner image; per-step image isolation is not yet implemented and is documented as a dogfood limitation.

```yaml
apiVersion: cicd.cloudivision.io/v1alpha1
kind: PipelineTemplate
metadata: {name: demo, namespace: demo-ci}
spec:
  projectRef: demo
  steps:
    - name: test
      image: golang:1.26-alpine
      command: [go]
      args: [test, ./...]
  build:
    enabled: false
    builder: none
    push: false
  security:
    allowPrivileged: false
    runAsNonRoot: true
```

## BuildRun

BuildRun is one immutable-intent CI execution. It references a Project, Repository and PipelineTemplate; identifies `revision`, optional branch/commit SHA and trigger; selects `job` or `tekton`; and provides output image and parameters. `gitOps.enabled` requests Release creation after success and carries repository, branch, path, strategy (`helm-values`, `kustomize-image`, `raw-yaml`) and Environment reference. Helm-values releases may additionally select `valuesFile` and dotted image field paths. Kustomize releases may select `kustomizationFile`/`imageName`; raw-YAML releases may select files, workload kind/name, and container name.

Lifecycle: empty/Pending → Queued → Running → Succeeded, Failed or Cancelled. Status includes start/completion timestamps, Job/PipelineRun reference, produced image, supply-chain references, structured failure and recent log metadata. Controllers must not create a new child for a terminal run.

```yaml
apiVersion: cicd.cloudivision.io/v1alpha1
kind: BuildRun
metadata: {name: demo-1, namespace: demo-ci}
spec:
  projectRef: demo
  repositoryRef: demo
  pipelineTemplateRef: demo
  revision: main
  triggeredBy: {type: manual, actor: developer}
  image: {repository: registry.example.com/demo/app, tag: main}
  executor: job
  gitOps: {enabled: false}
```

## Environment

Environment models a GitOps deployment target. It references a Project, has display/target namespace, type (`dev`, `staging`, `production`, `custom`), approval requirement, GitOps provider (`argocd`, `flux`, `generic`) and optional application/namespace. Policy can require signed images, SBOMs and vulnerability scan evidence.

Status phases are `Pending`, `Ready` and `Error`; sync/health strings carry provider state. Environment reconciliation is currently limited, so callers must use conditions rather than infer readiness from existence.

```yaml
apiVersion: cicd.cloudivision.io/v1alpha1
kind: Environment
metadata: {name: dev, namespace: demo-ci}
spec:
  projectRef: demo
  displayName: Development
  namespace: demo-dev
  type: dev
  requiresApproval: false
  gitOps: {provider: generic}
```

## Release

Release connects a successful BuildRun artifact to an Environment using strategy `gitops`. `projectRef`, `environmentRef`, `buildRunRef` and image are required. The image must contain a non-empty tag or digest. Approval records required/approved/rejected state and actor timestamps.

Lifecycle: Pending → AwaitingApproval (when required) → Deploying → Deployed, or Failed/RolledBack. Status records timestamps, Git commit and provider deployment sync/health. Approval changes are made through the API so actor and audit metadata are preserved.

```yaml
apiVersion: cicd.cloudivision.io/v1alpha1
kind: Release
metadata: {name: demo-dev-1, namespace: demo-ci}
spec:
  projectRef: demo
  environmentRef: dev
  buildRunRef: demo-1
  image: {repository: registry.example.com/demo/app, tag: main}
  approval: {required: false}
  strategy: gitops
```

## Validation notes

Current enum/min-length/default markers are reflected in checked-in CRDs. Admission also enforces these cross-field rules:

- enabled Repository webhooks require `secretRef`;
- PipelineTemplates require at least one step or an enabled image build, and step names are unique;
- enabled BuildRun GitOps requires a strategy, Environment reference and repository URL;
- production Environments require approval;
- Releases require an Environment and an image tag or digest.

Controllers still validate external references, Secret existence, artifact policy evidence and runtime capabilities because admission cannot prove cross-resource or external conditions. Environment supply-chain policy is therefore enforced by the Release controller against BuildRun status rather than by CEL.
