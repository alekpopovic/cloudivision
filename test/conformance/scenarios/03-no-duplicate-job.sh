#!/usr/bin/env bash

scenario_no_duplicate_job() {
  local iteration count
  prepare_namespace duplicate
  apply_fixture buildrun-duplicate.yaml
  wait_for_labeled_count jobs "cloudivision.io/buildrun=conformance-duplicate" 1

  for iteration in 1 2 3; do
    kubectl -n "${CURRENT_NAMESPACE}" annotate buildrun conformance-duplicate \
      "conformance.cloudivision.io/reconcile=${iteration}" --overwrite >/dev/null
  done
  wait_for_jsonpath buildrun/conformance-duplicate '{.status.phase}' Succeeded
  count="$(kubectl -n "${CURRENT_NAMESPACE}" get jobs \
    -l cloudivision.io/buildrun=conformance-duplicate -o name | sed '/^$/d' | wc -l | tr -d ' ')"
  [[ "${count}" == "1" ]] || {
    printf 'expected one Job after repeated reconciles, got %s\n' "${count}" >&2
    return 1
  }
}
