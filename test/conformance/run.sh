#!/usr/bin/env bash
set -Eeuo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib.sh
source "${SCRIPT_DIR}/lib.sh"

for scenario_file in "${SCRIPT_DIR}"/scenarios/*.sh; do
  # shellcheck disable=SC1090
  source "${scenario_file}"
done

require_command kubectl
kubectl cluster-info >/dev/null
kubectl get crd buildruns.cicd.cloudivision.io >/dev/null

trap 'on_error $?' ERR
trap cleanup_namespaces EXIT

run_scenario "manual BuildRun succeeds" scenario_manual_success
run_scenario "manual BuildRun fails with diagnostics" scenario_manual_failure
run_scenario "repeated reconcile creates no duplicate Job" scenario_no_duplicate_job
run_scenario "GitHub webhook delivery is idempotent" scenario_webhook_idempotency
run_scenario "successful GitOps BuildRun creates Release" scenario_gitops_release

printf '\nConformance summary: %d passed, %d failed, %d skipped checks/scenarios\n' \
  "${PASSED_SCENARIOS}" "${FAILED_SCENARIOS}" "${SKIPPED_CHECKS}"

if (( FAILED_SCENARIOS > 0 )); then
  exit 1
fi
