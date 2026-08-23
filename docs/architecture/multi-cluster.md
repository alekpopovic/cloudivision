# Multi-cluster architecture

`ClusterTarget` is an experimental, namespaced inventory and health resource.
Its spec identifies a local or external Kubernetes API, intended use
(`build`, `deploy`, or `both`), context, default namespace, and discovery labels.
Status records reachability, Kubernetes version, observed generation, and a
standard `Reachable` condition.

The local control-plane cluster remains the implicit default and appears as
`local` in `GET /api/v1/cluster-targets`. A `ClusterTarget` without a kubeconfig
reference also checks the controller's local REST configuration. External
targets load a namespaced Secret key (default `kubeconfig`) and may select a
kubeconfig context.

The controller performs discovery health checks only. BuildRun and Release
placement is unchanged: Kubernetes Job execution remains local, while CD still
uses existing GitOps repository/provider flows. External scheduling requires a
future trust, tenancy, credential and workload identity design.
