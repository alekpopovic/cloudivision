# CRD generation

`controller-gen` is not currently available in this environment, so CRD YAML has not been generated yet.

When `controller-gen` is available, run:

```sh
controller-gen rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases
```

The equivalent Kubebuilder setup commands for this API would be:

```sh
kubebuilder init --domain cloudivision.io --repo github.com/cloudivision/cloudivision
kubebuilder create api --group cicd --version v1alpha1 --kind Project --resource --controller
kubebuilder create api --group cicd --version v1alpha1 --kind Repository --resource=false --controller=false
kubebuilder create api --group cicd --version v1alpha1 --kind PipelineTemplate --resource=false --controller=false
kubebuilder create api --group cicd --version v1alpha1 --kind BuildRun --resource --controller
kubebuilder create api --group cicd --version v1alpha1 --kind Environment --resource=false --controller=false
kubebuilder create api --group cicd --version v1alpha1 --kind Release --resource --controller
```
