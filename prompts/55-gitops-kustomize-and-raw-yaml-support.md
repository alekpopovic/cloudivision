# 55. GitOps Kustomize and Raw YAML Support

```text
Continue working on the cloudivision project.

Task:
Add Kustomize and raw YAML GitOps update strategies.

Goal:
Support common deployment repository layouts beyond Helm values.

Strategies:
1. kustomize-image
- Update images section in kustomization.yaml.
- Support image name, newName, newTag, digest where practical.

2. raw-yaml
- Update image fields in Deployment/StatefulSet/DaemonSet/CronJob manifests.
- Match by container name if configured.
- Avoid changing unrelated images.

CRD/API:
Extend gitOps strategy config if needed:
- kustomize:
  - kustomizationFile
  - imageName
- rawYaml:
  - files []string
  - workloadKind optional
  - workloadName optional
  - containerName optional

Tests:
- kustomize image update
- raw Deployment update
- container name match
- no matching image
- invalid YAML
- repeated reconcile no duplicate change

Docs:
- docs/gitops/kustomize.md
- docs/gitops/raw-yaml.md

Acceptance criteria:
- go test ./... passes.
- Kustomize strategy works on sample repo.
- Raw YAML strategy works on sample Deployment.
- Existing helm-values strategy remains working.
```
