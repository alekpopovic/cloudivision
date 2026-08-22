# Backup and restore

## Backup

Record the application/chart versions and export schemas separately from instances:

```sh
kubectl get crd -o name | grep 'cicd.cloudivision.io$' | xargs kubectl get -o yaml > cloudivision-crds.yaml
kubectl get projects,repositories,pipelinetemplates,buildruns,environments,releases -A -o yaml > cloudivision-resources.yaml
kubectl -n cloudivision get configmap -o yaml > cloudivision-configmaps.yaml
```

Back up referenced Kubernetes Secrets with an encrypted, access-controlled tool such as Velero plus encryption, or rely on the external secret manager as the source of truth. Plaintext Secret YAML must not be committed or copied into logs.

If PostgreSQL audit storage is enabled, use a database-consistent snapshot or `pg_dump --format=custom`. Record the server version and verify a restore with `pg_restore --list`. GitOps repositories and container registries are external systems and need independent retention, availability, and recovery procedures.

## Restore into a new cluster

1. Provision compatible Kubernetes, ingress, storage, registry access, and any Argo CD/Flux dependencies.
2. Install or apply the backed-up CRDs and wait for `Established`.
3. Restore external secrets and PostgreSQL before starting the API.
4. Restore custom resources in dependency order: Projects, PipelineTemplates, Repositories, Environments, BuildRuns, then Releases.
5. Install the matching cloudivision chart version and verify controller/API health.
6. Confirm existing terminal runs remain unchanged and inspect non-terminal runs before allowing reconciliation.
7. Reconnect webhooks, validate GitOps credentials, run a non-production build/release, and compare audit continuity.

Resource `status` may be omitted for a desired-state-only recovery. Preserving status requires a backup tool that can restore status subresources; controllers will otherwise reconstruct the observable state where possible.

## Troubleshooting

If restored CRs are rejected, confirm CRDs were restored first and versions match. If terminal status is missing, keep external automation paused while controllers reconstruct observable state. Validate PostgreSQL ownership/extensions and Secret names/keys before enabling the API or webhooks.
