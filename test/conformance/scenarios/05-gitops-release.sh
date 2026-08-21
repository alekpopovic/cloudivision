#!/usr/bin/env bash

scenario_gitops_release() {
  prepare_namespace gitops
  apply_fixture buildrun-gitops.yaml
  wait_for_jsonpath buildrun/conformance-gitops '{.status.phase}' Succeeded
  wait_for_labeled_count releases "cloudivision.io/buildrun=conformance-gitops" 1

  if [[ -n "${CONFORMANCE_GITOPS_REPOSITORY_URL}" ]]; then
    wait_for_non_empty_jsonpath release/conformance-gitops-conformance-dev '{.status.gitCommit}'
  else
    record_skip "GitOps commit assertion (set an accessible CONFORMANCE_GITOPS_REPOSITORY_URL)"
  fi
}
