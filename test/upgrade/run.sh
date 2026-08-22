#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
CHART="${ROOT_DIR}/charts/cloudivision"
FIXTURE="${ROOT_DIR}/test/upgrade/fixtures/lifecycle.yaml"
LIVE="${UPGRADE_TEST_LIVE:-false}"
RELEASE="${UPGRADE_TEST_RELEASE:-cloudivision-upgrade}"
NAMESPACE="${UPGRADE_TEST_NAMESPACE:-cloudivision-upgrade-test}"
CONTEXT="${UPGRADE_TEST_CONTEXT:-}"
REPOSITORY_URL="${UPGRADE_TEST_REPOSITORY_URL:-https://github.com/cloudivision/cloudivision.git}"
REVISION="${UPGRADE_TEST_REVISION:-main}"
IMAGE_REGISTRY="${UPGRADE_TEST_IMAGE_REGISTRY:-ghcr.io/cloudivision}"
BASE_TAG="${UPGRADE_TEST_BASE_TAG:-dev}"
TARGET_TAG="${UPGRADE_TEST_TARGET_TAG:-${BASE_TAG}}"

require() {
  command -v "$1" >/dev/null 2>&1 || { echo "upgrade-test prerequisite missing: $1" >&2; exit 2; }
}

render_fixture() {
  sed -e "s|__NAMESPACE__|${NAMESPACE}|g" -e "s|__REPOSITORY_URL__|${REPOSITORY_URL}|g" -e "s|__REVISION__|${REVISION}|g" "${FIXTURE}"
}

wait_for_phase() {
  local name="$1"
  local deadline=$((SECONDS + 600))
  local phase
  while (( SECONDS < deadline )); do
    phase="$(kubectl --context "${CONTEXT}" -n "${NAMESPACE}" get buildrun "${name}" -o jsonpath='{.status.phase}' 2>/dev/null || true)"
    case "${phase}" in
      Succeeded) return 0 ;;
      Failed|Cancelled) kubectl --context "${CONTEXT}" -n "${NAMESPACE}" get buildrun "${name}" -o yaml; return 1 ;;
    esac
    sleep 5
  done
  echo "timed out waiting for BuildRun ${name}" >&2
  return 1
}

require helm
test -s "${FIXTURE}" || { echo "upgrade fixture missing: ${FIXTURE}" >&2; exit 1; }

first_render="$(mktemp)"
second_render="$(mktemp)"
fixture_render="$(mktemp)"
trap 'rm -f "${first_render}" "${second_render}" "${fixture_render}"' EXIT

helm template "${RELEASE}" "${CHART}" --include-crds --set controller.image.tag="${BASE_TAG}" >"${first_render}"
helm template "${RELEASE}" "${CHART}" --include-crds --set controller.image.tag="${TARGET_TAG}" >"${second_render}"
render_fixture >"${fixture_render}"

grep -q 'kind: CustomResourceDefinition' "${first_render}"
grep -q 'kind: Project' "${fixture_render}"
grep -q 'kind: BuildRun' "${fixture_render}"
if grep -Eq 'helm\.sh/hook:.*(pre-delete|post-delete)' "${CHART}/crds/cicd.cloudivision.io_crds.yaml"; then
  echo "CRDs must not contain Helm delete hooks" >&2
  exit 1
fi
echo "PASS: chart renders for base and target tags; lifecycle fixture and retained CRD layout are present"

if [[ "${LIVE}" != "true" ]]; then
  echo "SKIP: live install/BuildRun/upgrade/uninstall assertions (set UPGRADE_TEST_LIVE=true on a disposable kind cluster)"
  exit 0
fi

require kubectl
[[ -n "${CONTEXT}" ]] || { echo "UPGRADE_TEST_CONTEXT is required in live mode" >&2; exit 2; }
if [[ "${CONTEXT}" != kind-* && "${UPGRADE_TEST_ALLOW_NON_KIND:-false}" != "true" ]]; then
  echo "refusing live lifecycle test outside a kind context; set UPGRADE_TEST_ALLOW_NON_KIND=true only for another disposable cluster" >&2
  exit 2
fi
kubectl config get-contexts "${CONTEXT}" >/dev/null

cleanup() {
  helm --kube-context "${CONTEXT}" uninstall "${RELEASE}" -n "${NAMESPACE}" >/dev/null 2>&1 || true
  kubectl --context "${CONTEXT}" delete namespace "${NAMESPACE}" --ignore-not-found >/dev/null 2>&1 || true
}
trap 'cleanup; rm -f "${first_render}" "${second_render}" "${fixture_render}"' EXIT

helm --kube-context "${CONTEXT}" upgrade --install "${RELEASE}" "${CHART}" \
  --namespace "${NAMESPACE}" --create-namespace \
  --set global.imageRegistry="${IMAGE_REGISTRY}" \
  --set controller.image.tag="${BASE_TAG}" --set api.image.tag="${BASE_TAG}" --set web.image.tag="${BASE_TAG}" --set runner.image.tag="${BASE_TAG}"
kubectl --context "${CONTEXT}" -n "${NAMESPACE}" rollout status deployment --all --timeout=5m
kubectl --context "${CONTEXT}" apply -f "${fixture_render}"
wait_for_phase upgrade-before

project_uid="$(kubectl --context "${CONTEXT}" -n "${NAMESPACE}" get project upgrade-test -o jsonpath='{.metadata.uid}')"
build_uid="$(kubectl --context "${CONTEXT}" -n "${NAMESPACE}" get buildrun upgrade-before -o jsonpath='{.metadata.uid}')"

# Helm does not upgrade files in crds/, so apply target schemas first.
kubectl --context "${CONTEXT}" apply -f "${CHART}/crds/cicd.cloudivision.io_crds.yaml"
helm --kube-context "${CONTEXT}" upgrade "${RELEASE}" "${CHART}" \
  --namespace "${NAMESPACE}" \
  --set global.imageRegistry="${IMAGE_REGISTRY}" \
  --set controller.image.tag="${TARGET_TAG}" --set api.image.tag="${TARGET_TAG}" --set web.image.tag="${TARGET_TAG}" --set runner.image.tag="${TARGET_TAG}"
kubectl --context "${CONTEXT}" -n "${NAMESPACE}" rollout status deployment --all --timeout=5m

test "$(kubectl --context "${CONTEXT}" -n "${NAMESPACE}" get project upgrade-test -o jsonpath='{.metadata.uid}')" = "${project_uid}"
test "$(kubectl --context "${CONTEXT}" -n "${NAMESPACE}" get buildrun upgrade-before -o jsonpath='{.metadata.uid}')" = "${build_uid}"
sed 's/name: upgrade-before/name: upgrade-after/' "${fixture_render}" | awk 'BEGIN{emit=0} /^kind: BuildRun$/{emit=1; print "apiVersion: cicd.cloudivision.io/v1alpha1"} emit{print}' | kubectl --context "${CONTEXT}" apply -f -
wait_for_phase upgrade-after

helm --kube-context "${CONTEXT}" uninstall "${RELEASE}" -n "${NAMESPACE}"
kubectl --context "${CONTEXT}" get crd buildruns.cicd.cloudivision.io >/dev/null
kubectl --context "${CONTEXT}" -n "${NAMESPACE}" get project upgrade-test >/dev/null
echo "PASS: resources survived upgrade, a new BuildRun reconciled, and Helm uninstall retained CRDs/custom resources"
