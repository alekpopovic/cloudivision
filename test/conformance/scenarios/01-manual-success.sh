#!/usr/bin/env bash

scenario_manual_success() {
  prepare_namespace success
  apply_fixture buildrun-success.yaml
  wait_for_labeled_count jobs "cloudivision.io/buildrun=conformance-success" 1
  wait_for_jsonpath buildrun/conformance-success '{.status.phase}' Succeeded
  assert_non_empty_jsonpath buildrun/conformance-success '{.status.startedAt}'
  assert_non_empty_jsonpath buildrun/conformance-success '{.status.completedAt}'

  if [[ -n "${CONFORMANCE_API_BASE_URL}" ]]; then
    require_command curl
    curl --fail --silent --show-error \
      "${CONFORMANCE_API_BASE_URL%/}/api/v1/build-runs/${CURRENT_NAMESPACE}/conformance-success/logs" >/dev/null
  else
    record_skip "API log assertion (set CONFORMANCE_API_BASE_URL when the API is reachable)"
  fi
}
