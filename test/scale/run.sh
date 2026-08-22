#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
SCENARIO="${SCALE_SCENARIO:-limited}"
LIVE="${SCALE_TEST_LIVE:-false}"
CONTEXT="${SCALE_TEST_CONTEXT:-}"
NAMESPACE="${SCALE_TEST_NAMESPACE:-cloudivision-scale}"
RUN_ID="$(printf '%s' "${SCALE_TEST_RUN_ID:-$(date +%s)}" | tr '[:upper:]' '[:lower:]' | sed -E 's/[^a-z0-9-]+/-/g; s/^-+//; s/-+$//' | cut -c1-20)"
RUN_ID="${RUN_ID:-run}"
OUTPUT="$(mktemp)"
export GOCACHE="${GOCACHE:-${ROOT_DIR}/.cache/go-build}"
mkdir -p "${GOCACHE}"
trap 'rm -f "${OUTPUT}"' EXIT

count=10
projects=1
gitops=false
case "${SCENARIO}" in
  limited) count=10 ;;
  100-buildruns) count=100 ;;
  1000-buildruns) count=1000; projects=20 ;;
  50-runner-jobs) count=50 ;;
  10-releases) count=10; gitops=true ;;
  100-webhooks)
    if [[ "${LIVE}" != "true" || -z "${SCALE_WEBHOOK_COMMAND:-}" ]]; then
      echo "SKIP: webhook burst requires SCALE_TEST_LIVE=true and a signature-aware SCALE_WEBHOOK_COMMAND"
      exit 0
    fi
    start="$(date +%s)"
    for i in $(seq 1 100); do SCALE_EVENT_NUMBER="${i}" bash -c "${SCALE_WEBHOOK_COMMAND}" & done
    wait
    elapsed=$(( $(date +%s) - start ))
    echo "webhook burst completed in ${elapsed}s (target <=10s)"
    exit 0
    ;;
  large-logs)
    echo "SKIP: 100MB logs are opt-in; use a dedicated log-producing PipelineTemplate and isolated storage budget"
    exit 0
    ;;
  api-list) count=1000; projects=20 ;;
  *) echo "unknown SCALE_SCENARIO=${SCENARIO}" >&2; exit 2 ;;
esac

go run "${ROOT_DIR}/test/scale/generate-buildruns.go" --count "${count}" --projects "${projects}" --namespace "${NAMESPACE}" --run-id "${RUN_ID}" --gitops="${gitops}" >"${OUTPUT}"
generated="$(grep -o '"kind":"BuildRun"' "${OUTPUT}" | wc -l)"
test "${generated}" -eq "${count}"
echo "PASS: generated ${generated} BuildRuns across ${projects} project(s) for scenario ${SCENARIO}"

if [[ "${LIVE}" != "true" ]]; then
  echo "SKIP: cluster apply and measurements (set SCALE_TEST_LIVE=true with SCALE_TEST_CONTEXT on a disposable cluster)"
  exit 0
fi

command -v kubectl >/dev/null || { echo "kubectl is required" >&2; exit 2; }
[[ -n "${CONTEXT}" ]] || { echo "SCALE_TEST_CONTEXT is required in live mode" >&2; exit 2; }
if [[ "${CONTEXT}" != kind-* && "${SCALE_TEST_ALLOW_NON_KIND:-false}" != "true" ]]; then
  echo "refusing scale load outside kind; explicitly set SCALE_TEST_ALLOW_NON_KIND=true for another isolated cluster" >&2
  exit 2
fi
kubectl --context "${CONTEXT}" get namespace "${NAMESPACE}" >/dev/null 2>&1 || kubectl --context "${CONTEXT}" create namespace "${NAMESPACE}"

cleanup() {
  kubectl --context "${CONTEXT}" -n "${NAMESPACE}" delete buildruns,environments,pipelinetemplates,repositories,projects -l "cloudivision.io/scale-run=${RUN_ID}" --ignore-not-found --wait=false >/dev/null 2>&1 || true
}
trap 'cleanup; rm -f "${OUTPUT}"' EXIT

started="$(date +%s)"
kubectl --context "${CONTEXT}" apply -f "${OUTPUT}"
echo "Applied at $(date -u +%FT%TZ); sampling controller/API resources"
kubectl --context "${CONTEXT}" top pods -A -l app.kubernetes.io/name=cloudivision 2>/dev/null || echo "SKIP: metrics-server pod CPU/memory sample unavailable"

deadline=$((SECONDS + ${SCALE_WAIT_TIMEOUT_SECONDS:-900}))
while (( SECONDS < deadline )); do
  terminal="$(kubectl --context "${CONTEXT}" -n "${NAMESPACE}" get buildruns -l "cloudivision.io/scale-run=${RUN_ID}" -o jsonpath='{range .items[*]}{.status.phase}{"\n"}{end}' | grep -Ec '^(Succeeded|Failed|Cancelled)$' || true)"
  [[ "${terminal}" -eq "${count}" ]] && break
  sleep 5
done
elapsed=$(( $(date +%s) - started ))
terminal="$(kubectl --context "${CONTEXT}" -n "${NAMESPACE}" get buildruns -l "cloudivision.io/scale-run=${RUN_ID}" -o jsonpath='{range .items[*]}{.status.phase}{"\n"}{end}' | grep -Ec '^(Succeeded|Failed|Cancelled)$' || true)"
echo "RESULT scenario=${SCENARIO} objects=${count} terminal=${terminal} elapsed_seconds=${elapsed}"
kubectl --context "${CONTEXT}" top pods -A -l app.kubernetes.io/name=cloudivision 2>/dev/null || true

if [[ -n "${SCALE_API_BASE_URL:-}" ]]; then
  curl --fail --silent --show-error -o /dev/null -w 'API list latency: %{time_total}s\n' "${SCALE_API_BASE_URL%/}/api/v1/build-runs?namespace=${NAMESPACE}&limit=100"
  first="$(kubectl --context "${CONTEXT}" -n "${NAMESPACE}" get buildrun -l "cloudivision.io/scale-run=${RUN_ID}" -o jsonpath='{.items[0].metadata.name}')"
  curl --fail --silent --show-error -o /dev/null -w 'Log endpoint latency: %{time_total}s\n' "${SCALE_API_BASE_URL%/}/api/v1/build-runs/${NAMESPACE}/${first}/logs?tailLines=200" || true
else
  echo "SKIP: API list/log latency (set SCALE_API_BASE_URL)"
fi
[[ "${terminal}" -eq "${count}" ]] || { echo "not all BuildRuns reached a terminal phase before timeout" >&2; exit 1; }
