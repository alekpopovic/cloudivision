#!/usr/bin/env bash
set -Eeuo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
CHART="${ROOT_DIR}/charts/cloudivision"
OUTPUT_DIR="$(mktemp -d)"
trap 'rm -rf -- "${OUTPUT_DIR}"' EXIT

command -v helm >/dev/null 2>&1 || { echo "helm-test: helm is required" >&2; exit 1; }

helm lint "${CHART}" >/dev/null
helm template cloudivision "${CHART}" --include-crds >"${OUTPUT_DIR}/default.yaml"
helm template cloudivision "${CHART}" --set ingress.enabled=true \
  --set 'ingress.tls[0].secretName=cloudivision-tls' \
  --set 'ingress.tls[0].hosts[0]=cloudivision.example.com' >"${OUTPUT_DIR}/ingress.yaml"
helm template cloudivision "${CHART}" --set networkPolicy.enabled=true >"${OUTPUT_DIR}/network-policy.yaml"
helm template cloudivision "${CHART}" --set database.enabled=true --set audit.backend=postgres >"${OUTPUT_DIR}/database.yaml"
helm template cloudivision "${CHART}" -f "${ROOT_DIR}/test/helm/production-values.yaml" >"${OUTPUT_DIR}/production.yaml"

grep -q 'readOnlyRootFilesystem: true' "${OUTPUT_DIR}/default.yaml"
grep -q 'kind: Ingress' "${OUTPUT_DIR}/ingress.yaml"
grep -q 'secretName: cloudivision-tls' "${OUTPUT_DIR}/ingress.yaml"
grep -q 'kind: NetworkPolicy' "${OUTPUT_DIR}/network-policy.yaml"
grep -q 'CLOU_DIVISION_DATABASE_URL' "${OUTPUT_DIR}/database.yaml"
grep -q 'kind: HorizontalPodAutoscaler' "${OUTPUT_DIR}/production.yaml"
grep -q 'kind: PodDisruptionBudget' "${OUTPUT_DIR}/production.yaml"
grep -q 'kind: ServiceMonitor' "${OUTPUT_DIR}/production.yaml"
grep -q 'priorityClassName: "platform-medium"' "${OUTPUT_DIR}/production.yaml"

echo "helm-test: default, ingress/TLS, NetworkPolicy, database, and production renders passed"
