#!/usr/bin/env bash

CONFORMANCE_TIMEOUT="${CONFORMANCE_TIMEOUT:-300}"
CONFORMANCE_NAMESPACE_PREFIX="${CONFORMANCE_NAMESPACE_PREFIX:-cloudivision-conformance}"
CONFORMANCE_REPOSITORY_URL="${CONFORMANCE_REPOSITORY_URL:-https://github.com/docker/getting-started-todo-app.git}"
CONFORMANCE_REVISION="${CONFORMANCE_REVISION:-main}"
CONFORMANCE_API_BASE_URL="${CONFORMANCE_API_BASE_URL:-}"
CONFORMANCE_KEEP_NAMESPACES="${CONFORMANCE_KEEP_NAMESPACES:-false}"
CONFORMANCE_CONTROLLER_NAMESPACE="${CONFORMANCE_CONTROLLER_NAMESPACE:-cloudivision}"
CONFORMANCE_CONTROLLER_SELECTOR="${CONFORMANCE_CONTROLLER_SELECTOR:-app.kubernetes.io/component=controller}"
CONFORMANCE_GITOPS_REPOSITORY_URL="${CONFORMANCE_GITOPS_REPOSITORY_URL:-}"
CONFORMANCE_GITOPS_BRANCH="${CONFORMANCE_GITOPS_BRANCH:-main}"
CONFORMANCE_GITOPS_PATH="${CONFORMANCE_GITOPS_PATH:-values.yaml}"

PASSED_SCENARIOS=0
FAILED_SCENARIOS=0
SKIPPED_CHECKS=0
CURRENT_NAMESPACE=""
CREATED_NAMESPACES=()

log() {
  printf '[conformance] %s\n' "$*"
}

require_command() {
  if ! command -v "$1" >/dev/null 2>&1; then
    printf 'required command not found: %s\n' "$1" >&2
    exit 2
  fi
}

record_skip() {
  SKIPPED_CHECKS=$((SKIPPED_CHECKS + 1))
  log "SKIP: $*"
}

run_scenario() {
  local name="$1"
  local function_name="$2"
  log "START: ${name}"
  "${function_name}"
  PASSED_SCENARIOS=$((PASSED_SCENARIOS + 1))
  log "PASS: ${name}"
}

on_error() {
  local rc="$1"
  trap - ERR
  FAILED_SCENARIOS=$((FAILED_SCENARIOS + 1))
  log "FAIL: conformance stopped during ${CURRENT_NAMESPACE:-preflight} (exit ${rc})"
  diagnose_namespace "${CURRENT_NAMESPACE}"
  printf '\nConformance summary: %d passed, %d failed, %d skipped checks/scenarios\n' \
    "${PASSED_SCENARIOS}" "${FAILED_SCENARIOS}" "${SKIPPED_CHECKS}"
  exit "${rc}"
}

scenario_namespace() {
  printf '%s-%s' "${CONFORMANCE_NAMESPACE_PREFIX}" "$1"
}

prepare_namespace() {
  local suffix="$1"
  CURRENT_NAMESPACE="$(scenario_namespace "${suffix}")"
  if kubectl get namespace "${CURRENT_NAMESPACE}" >/dev/null 2>&1; then
    log "deleting stale namespace ${CURRENT_NAMESPACE}"
    kubectl delete namespace "${CURRENT_NAMESPACE}" --wait=true --timeout="${CONFORMANCE_TIMEOUT}s"
  fi
  kubectl create namespace "${CURRENT_NAMESPACE}" >/dev/null
  CREATED_NAMESPACES+=("${CURRENT_NAMESPACE}")
  apply_fixture base.yaml
  wait_for_jsonpath project/conformance '{.status.phase}' Ready
}

cleanup_namespaces() {
  local namespace
  if [[ "${CONFORMANCE_KEEP_NAMESPACES}" == "true" ]]; then
    log "keeping conformance namespaces: ${CREATED_NAMESPACES[*]:-none}"
    return
  fi
  for namespace in "${CREATED_NAMESPACES[@]}"; do
    kubectl delete namespace "${namespace}" --wait=false --ignore-not-found >/dev/null 2>&1 || true
  done
}

escaped_sed_replacement() {
  printf '%s' "$1" | sed -e 's/[\\&|]/\\&/g'
}

render_fixture() {
  local fixture="$1"
  local namespace repository_url revision gitops_url gitops_branch gitops_path
  namespace="$(escaped_sed_replacement "${CURRENT_NAMESPACE}")"
  repository_url="$(escaped_sed_replacement "${CONFORMANCE_REPOSITORY_URL}")"
  revision="$(escaped_sed_replacement "${CONFORMANCE_REVISION}")"
  gitops_url="$(escaped_sed_replacement "${CONFORMANCE_GITOPS_REPOSITORY_URL:-https://127.0.0.1:1/gitops.git}")"
  gitops_branch="$(escaped_sed_replacement "${CONFORMANCE_GITOPS_BRANCH}")"
  gitops_path="$(escaped_sed_replacement "${CONFORMANCE_GITOPS_PATH}")"
  sed \
    -e "s|__NAMESPACE__|${namespace}|g" \
    -e "s|__REPOSITORY_URL__|${repository_url}|g" \
    -e "s|__REVISION__|${revision}|g" \
    -e "s|__GITOPS_REPOSITORY_URL__|${gitops_url}|g" \
    -e "s|__GITOPS_BRANCH__|${gitops_branch}|g" \
    -e "s|__GITOPS_PATH__|${gitops_path}|g" \
    "${SCRIPT_DIR}/fixtures/${fixture}"
}

apply_fixture() {
  local fixture="$1"
  log "applying ${fixture} to ${CURRENT_NAMESPACE}"
  render_fixture "${fixture}" | kubectl apply -f - >/dev/null
}

wait_for_jsonpath() {
  local resource="$1"
  local jsonpath="$2"
  local expected="$3"
  local timeout="${4:-${CONFORMANCE_TIMEOUT}}"
  local started value
  started="${SECONDS}"
  while (( SECONDS - started < timeout )); do
    value="$(kubectl -n "${CURRENT_NAMESPACE}" get "${resource}" -o "jsonpath=${jsonpath}" 2>/dev/null || true)"
    if [[ "${value}" == "${expected}" ]]; then
      return 0
    fi
    sleep 2
  done
  printf 'timeout after %ss waiting for %s jsonpath %s to equal %q (last value %q)\n' \
    "${timeout}" "${resource}" "${jsonpath}" "${expected}" "${value:-}" >&2
  return 1
}

wait_for_labeled_count() {
  local resource="$1"
  local selector="$2"
  local expected="$3"
  local timeout="${4:-${CONFORMANCE_TIMEOUT}}"
  local started count
  started="${SECONDS}"
  while (( SECONDS - started < timeout )); do
    count="$(kubectl -n "${CURRENT_NAMESPACE}" get "${resource}" -l "${selector}" -o name 2>/dev/null | sed '/^$/d' | wc -l | tr -d ' ')"
    if [[ "${count}" == "${expected}" ]]; then
      return 0
    fi
    sleep 2
  done
  printf 'timeout after %ss waiting for %s with selector %s count=%s (last count %s)\n' \
    "${timeout}" "${resource}" "${selector}" "${expected}" "${count:-0}" >&2
  return 1
}

wait_for_non_empty_jsonpath() {
  local resource="$1"
  local jsonpath="$2"
  local timeout="${3:-${CONFORMANCE_TIMEOUT}}"
  local started value
  started="${SECONDS}"
  while (( SECONDS - started < timeout )); do
    value="$(kubectl -n "${CURRENT_NAMESPACE}" get "${resource}" -o "jsonpath=${jsonpath}" 2>/dev/null || true)"
    if [[ -n "${value}" ]]; then
      return 0
    fi
    sleep 2
  done
  printf 'timeout after %ss waiting for %s jsonpath %s to become non-empty\n' \
    "${timeout}" "${resource}" "${jsonpath}" >&2
  return 1
}

assert_non_empty_jsonpath() {
  local resource="$1"
  local jsonpath="$2"
  local value
  value="$(kubectl -n "${CURRENT_NAMESPACE}" get "${resource}" -o "jsonpath=${jsonpath}")"
  if [[ -z "${value}" ]]; then
    printf '%s %s is empty\n' "${resource}" "${jsonpath}" >&2
    return 1
  fi
}

diagnose_namespace() {
  local namespace="$1"
  if [[ -z "${namespace}" ]]; then
    return
  fi
  printf '\n===== conformance diagnostics: %s =====\n' "${namespace}" >&2
  kubectl get buildruns -A >&2 || true
  kubectl -n "${namespace}" get buildruns -o name 2>/dev/null | while read -r buildrun; do
    kubectl -n "${namespace}" describe "${buildrun}" >&2 || true
  done
  kubectl -n "${namespace}" get jobs,pods -o wide >&2 || true
  kubectl -n "${namespace}" get pods -o name 2>/dev/null | while read -r pod; do
    printf '\n--- logs: %s ---\n' "${pod}" >&2
    kubectl -n "${namespace}" logs "${pod}" --all-containers --tail=200 >&2 || true
  done
  kubectl -n "${CONFORMANCE_CONTROLLER_NAMESPACE}" logs \
    -l "${CONFORMANCE_CONTROLLER_SELECTOR}" --all-containers --tail=200 >&2 || true
}
