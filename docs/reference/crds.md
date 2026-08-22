# CRD reference

cloudivision installs six namespaced `cicd.cloudivision.io/v1alpha1` resources.
The generated OpenAPI schemas in `charts/cloudivision/crds/` are authoritative.

| Kind | Purpose | Important references/status |
| --- | --- | --- |
| `Project` | Tenant boundary and namespace policy | namespace, registry, runner service account |
| `Repository` | Git source and webhook settings | project and pipeline template |
| `PipelineTemplate` | Reusable steps, build and security defaults | parameters and executor |
| `BuildRun` | One immutable build request | source, template, phase, artifact, release |
| `Environment` | GitOps target and promotion policy | provider, repository path, approval |
| `Release` | One GitOps promotion | artifact, environment, approval, sync state |

The common flow is:

```text
Project <- Repository -> PipelineTemplate
                    \-> BuildRun -> Release -> Environment
```

Example discovery commands:

```sh
kubectl api-resources --api-group=cicd.cloudivision.io
kubectl explain buildrun.spec
kubectl -n cloudivision get projects,repositories,pipelinetemplates,buildruns,environments,releases
```

All operational state belongs in `status`; clients should write desired state only
to `spec`. Controllers own conditions and observed generation. See the individual
[concept pages](../index.md#concepts) and the
[compatibility policy](../api/compatibility-policy.md) before changing stored resources.
