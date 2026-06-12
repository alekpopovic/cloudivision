# Kubebuilder operator foundation

This repository uses a Kubebuilder-compatible layout for the cloudivision controller, API types, RBAC markers, and config manifests.

The initial layout was written manually, then CRDs, RBAC, and deepcopy code were generated with `controller-gen`.

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
make manifests
```

Generate deepcopy code with:

```sh
make generate
```

The Makefile intentionally scopes generation to `./api/...` and `./internal/controller/...` instead of `./...`. This avoids accidentally scanning local caches such as `.cache/go-mod` while still covering API schemas and controller RBAC markers.
