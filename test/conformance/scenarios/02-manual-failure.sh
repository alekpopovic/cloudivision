#!/usr/bin/env bash

scenario_manual_failure() {
  prepare_namespace failure
  apply_fixture buildrun-failure.yaml
  wait_for_labeled_count jobs "cloudivision.io/buildrun=conformance-failure" 1
  wait_for_jsonpath buildrun/conformance-failure '{.status.phase}' Failed
  assert_non_empty_jsonpath buildrun/conformance-failure '{.status.failure.reason}'
  assert_non_empty_jsonpath buildrun/conformance-failure '{.status.failure.message}'
}
