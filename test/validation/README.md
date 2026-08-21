# CRD admission validation cases

The files in `invalid/` are expected to be rejected by a Kubernetes API server with the generated CRDs installed. They cover cross-field rules that cannot be represented by simple required/enum markers.

Run against a disposable cluster:

```sh
kubectl apply -k config/crd
for fixture in test/validation/invalid/*.yaml; do
  if kubectl apply --server-side --dry-run=server -f "${fixture}"; then
    echo "unexpectedly accepted: ${fixture}" >&2
    exit 1
  fi
done
```

Valid resources are exercised by `test/conformance/fixtures`. The generated-schema unit test guards marker drift without requiring an API server; server-side admission remains the authoritative rejection check.
