# Uninstall cloudivision

Create a backup first, then remove the Helm-managed workloads:

```sh
helm uninstall cloudivision --namespace cloudivision
```

CRDs placed in the chart `crds/` directory are preserved by Helm uninstall. Consequently, Project, Repository, PipelineTemplate, BuildRun, Environment, and Release objects remain stored in the cluster and can reconcile again after reinstall. Runner Jobs owned by BuildRuns and namespaces previously created by the Project controller are not part of the Helm release and may also remain.

Inspect before cleanup:

```sh
kubectl get projects,buildruns,releases -A
kubectl get namespaces -l app.kubernetes.io/managed-by=cloudivision
```

Delete project namespaces only after confirming they do not contain application workloads or secrets that must be retained. Namespace deletion removes every namespaced object in it.

To permanently remove all cloudivision data, explicitly delete custom resources first and CRDs last:

```sh
kubectl delete releases,environments,buildruns,repositories,pipelinetemplates,projects --all -A
kubectl delete crd projects.cicd.cloudivision.io repositories.cicd.cloudivision.io pipelinetemplates.cicd.cloudivision.io buildruns.cicd.cloudivision.io environments.cicd.cloudivision.io releases.cicd.cloudivision.io
```

Deleting a CRD irreversibly deletes every custom resource of that kind from Kubernetes storage. A later CRD reinstall does not restore them; only a tested backup can.
