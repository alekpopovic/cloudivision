# Kubebuilder operator foundation

This repository uses a Kubebuilder-compatible layout for the cloudivision controller, API types, RBAC markers, and config manifests.

The `kubebuilder` and `controller-gen` CLIs were not available when this foundation was created, so the initial layout was written manually.

When the tools are available, the intended setup commands are:

```sh
kubebuilder init --domain cloudivision.io --repo github.com/cloudivision/cloudivision
kubebuilder create api --group cicd --version v1alpha1 --kind Project --resource --controller
kubebuilder create api --group cicd --version v1alpha1 --kind Repository --resource=false --controller=false
kubebuilder create api --group cicd --version v1alpha1 --kind PipelineTemplate --resource=false --controller=false
kubebuilder create api --group cicd --version v1alpha1 --kind BuildRun --resource --controller
kubebuilder create api --group cicd --version v1alpha1 --kind Environment --resource=false --controller=false
kubebuilder create api --group cicd --version v1alpha1 --kind Release --resource --controller
```

Generate CRDs and RBAC from Go markers with:

```sh
controller-gen rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases
```

Generate deepcopy code with:

```sh
controller-gen object:headerFile="hack/boilerplate.go.txt" paths="./..."
```
