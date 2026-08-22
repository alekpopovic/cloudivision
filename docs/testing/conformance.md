# Conformance testing

The conformance suite proves core cloudivision behavior against the Kubernetes cluster in the current kubectl context. It creates isolated namespaces, fails fast on real errors and removes its namespaces on exit.

## Prerequisites

- `kubectl` with permission to create/delete namespaces and cloudivision resources
- installed cloudivision CRDs, controller and runner image
- cluster egress to the configured public source repository
- `curl` and `openssl` when webhook/API checks are enabled

For a local cluster:

```sh
./hack/kind-create.sh
./hack/kind-load-images.sh
./hack/install-dev.sh
make conformance
```

The suite runs these scenarios:

1. successful manual BuildRun, timestamps and optional API logs;
2. failed manual BuildRun with reason/message;
3. repeated reconcile with exactly one runner Job;
4. duplicate signed GitHub webhook delivery;
5. successful GitOps-enabled BuildRun creating one Release, with an optional real commit assertion.

## Configuration

| Variable | Default | Purpose |
|---|---|---|
| `CONFORMANCE_TIMEOUT` | `300` | Maximum seconds for each explicit wait. |
| `CONFORMANCE_NAMESPACE_PREFIX` | `cloudivision-conformance` | Prefix for disposable scenario namespaces. |
| `CONFORMANCE_REPOSITORY_URL` | Docker's public getting-started todo app | Repository cloned by runner Jobs; override with another credential-free fixture. |
| `CONFORMANCE_REVISION` | `main` | Branch, tag or commit checked out by runner Jobs. |
| `CONFORMANCE_API_BASE_URL` | empty | Reachable API URL; enables API log and webhook assertions. |
| `CONFORMANCE_KEEP_NAMESPACES` | `false` | Keep scenario namespaces for manual inspection. |
| `CONFORMANCE_CONTROLLER_NAMESPACE` | `cloudivision` | Namespace used to collect controller failure logs. |
| `CONFORMANCE_CONTROLLER_SELECTOR` | `app.kubernetes.io/component=controller` | Controller pod label selector. |
| `CONFORMANCE_GITOPS_REPOSITORY_URL` | empty | Git repository reachable and writable from the controller; enables commit assertion. |
| `CONFORMANCE_GITOPS_BRANCH` | `main` | GitOps branch to update. |
| `CONFORMANCE_GITOPS_PATH` | `values.yaml` | Helm values file updated in the GitOps repository. |

To include API/webhook checks with a local port-forward:

```sh
kubectl -n cloudivision port-forward svc/cloudivision-cloudivision-api 8080:8080
CONFORMANCE_API_BASE_URL=http://127.0.0.1:8080 make conformance
```

An empty optional variable never silently passes: the suite prints `SKIP` and includes it in the summary. Success, failure and duplicate-Job scenarios are never optional.

## Failures and cleanup

On a real failure the suite prints all BuildRuns, describes scenario BuildRuns, lists Jobs/Pods, prints pod logs and attempts to print controller logs. It then exits immediately with a non-zero code.

Keep failed resources for deeper investigation:

```sh
CONFORMANCE_KEEP_NAMESPACES=true make conformance
```

Delete them later with explicit names, for example:

```sh
kubectl delete namespace cloudivision-conformance-success
```

If a run times out before a Job starts, verify Project-created ServiceAccounts/RBAC, controller watches and runner image availability. If a runner fails during clone, verify cluster DNS/egress and `CONFORMANCE_REPOSITORY_URL`. If the API checks fail across project namespaces, verify the API ServiceAccount has intentionally scoped `buildruns`, `pods` and `pods/log` access there.
