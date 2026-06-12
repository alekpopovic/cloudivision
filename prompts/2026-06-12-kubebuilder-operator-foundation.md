# Kubebuilder operator foundation

User prompt:

```text
Continue working on the cloudivision project.

Task:
Set up the Kubebuilder-compatible operator foundation for cloudivision.

Context:
The project needs Kubernetes CRDs and controllers. If the kubebuilder CLI is available in the environment, use it. If it is not available, manually create a compatible layout and document which kubebuilder commands would have been used.

Desired API group:
- group: cicd
- domain: cloudivision.io
- version: v1alpha1

Required Kinds:
- Project
- Repository
- PipelineTemplate
- BuildRun
- Environment
- Release

Steps:
1. Add Kubernetes API types in /api/v1alpha1.
2. Add GroupVersion and SchemeBuilder.
3. Add minimal controllers in /internal/controller for:
   - BuildRun
   - Release
   - Project
4. Update /cmd/controller/main.go with manager setup:
   - scheme registration
   - metrics bind address
   - healthz
   - readyz
   - optional leader election controlled through config/env flag
5. Add RBAC marker comments where needed.
6. Add /config skeleton:
   - CRD generation target
   - RBAC manifests
   - manager deployment
   - kustomization files
7. Update Makefile targets:
   - generate
   - manifests
   - install
   - uninstall
   - run-controller-local

CRDs can have minimal Spec/Status for now, but they must be valid Go types with kubebuilder markers.

Acceptance criteria:
- go test ./... passes.
- controller main builds.
- API types are registered in the scheme.
- If controller-gen is available, make manifests generates CRD YAML.
- If controller-gen is not available, document the exact commands that should be run.
```
