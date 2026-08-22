# Upgrade cloudivision

Back up CRDs, custom resources, secrets, and the optional audit database before every upgrade. Review release notes for schema or configuration changes and test the exact source and target versions in a disposable cluster.

## Helm and CRDs

Helm installs files from `charts/cloudivision/crds` only on the first install; it does not upgrade or roll them back. Apply compatible CRD schemas before upgrading controllers:

```sh
kubectl apply -f charts/cloudivision/crds/cicd.cloudivision.io_crds.yaml
kubectl wait --for=condition=Established --timeout=2m crd --all
helm upgrade cloudivision charts/cloudivision --namespace cloudivision --reuse-values
kubectl -n cloudivision rollout status deployment --all --timeout=5m
```

Applying a CRD updates its schema without deleting existing custom resources. Verify objects remain readable and note `status.storedVersions` before removing a served API version.

## Health checks

Check controller/API pods, rollout state, warning Events, and reconciliation after the upgrade:

```sh
kubectl -n cloudivision get deploy,pod
kubectl -n cloudivision get events --sort-by=.lastTimestamp
kubectl get projects,repositories,pipelinetemplates,buildruns,environments,releases -A
kubectl -n cloudivision port-forward service/cloudivision-api 8080:8080
curl --fail http://127.0.0.1:8080/readyz
```

Create a non-production smoke BuildRun and confirm it reaches a terminal state without duplicate Jobs. Run `make upgrade-test` for offline validation or its documented live mode on a disposable cluster.

## Rollback

Capture `helm history cloudivision -n cloudivision` before upgrading. If workloads fail but the old binary understands the newly applied CRD schema, run `helm rollback cloudivision REVISION -n cloudivision` and verify reconciliation. Helm does not roll back CRDs. If the old binary is not schema-compatible, restore the previous cluster/CRD backup into a separate cluster rather than replacing a CRD in place.

`v1alpha1` does not yet promise storage or conversion compatibility between all releases. There is one served/storage version and no conversion webhook, so destructive field changes require an explicit migration. Never remove a value from `status.storedVersions` until all stored objects have been migrated and verified.
