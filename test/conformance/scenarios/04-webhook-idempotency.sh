#!/usr/bin/env bash

scenario_webhook_idempotency() {
  local secret payload signature endpoint response_file first_code second_code count
  if [[ -z "${CONFORMANCE_API_BASE_URL}" ]]; then
    record_skip "webhook idempotency scenario (set CONFORMANCE_API_BASE_URL)"
    return 0
  fi
  require_command curl
  require_command openssl

  prepare_namespace webhook
  secret='cloudivision-conformance-secret'
  kubectl -n "${CURRENT_NAMESPACE}" create secret generic conformance-webhook \
    --from-literal="secret=${secret}" >/dev/null
  apply_fixture webhook-repository.yaml

  payload="${SCRIPT_DIR}/fixtures/github-push.json"
  signature="sha256=$(openssl dgst -sha256 -hmac "${secret}" -hex "${payload}" | sed 's/^.* //')"
  endpoint="${CONFORMANCE_API_BASE_URL%/}/api/v1/webhooks/github/conformance-webhook?namespace=${CURRENT_NAMESPACE}"
  response_file="$(mktemp)"

  first_code="$(curl --silent --show-error --output "${response_file}" --write-out '%{http_code}' \
    -X POST "${endpoint}" -H 'Content-Type: application/json' -H 'X-GitHub-Event: push' \
    -H 'X-GitHub-Delivery: conformance-delivery-1' -H "X-Hub-Signature-256: ${signature}" \
    --data-binary "@${payload}")"
  [[ "${first_code}" == "201" ]] || {
    printf 'first webhook returned HTTP %s: ' "${first_code}" >&2
    sed -n '1,20p' "${response_file}" >&2
    rm -f "${response_file}"
    return 1
  }

  second_code="$(curl --silent --show-error --output "${response_file}" --write-out '%{http_code}' \
    -X POST "${endpoint}" -H 'Content-Type: application/json' -H 'X-GitHub-Event: push' \
    -H 'X-GitHub-Delivery: conformance-delivery-1' -H "X-Hub-Signature-256: ${signature}" \
    --data-binary "@${payload}")"
  rm -f "${response_file}"
  [[ "${second_code}" == "200" ]] || {
    printf 'duplicate webhook returned HTTP %s, want 200\n' "${second_code}" >&2
    return 1
  }

  count="$(kubectl -n "${CURRENT_NAMESPACE}" get buildruns -o name | sed '/^$/d' | wc -l | tr -d ' ')"
  [[ "${count}" == "1" ]] || {
    printf 'expected one webhook BuildRun, got %s\n' "${count}" >&2
    return 1
  }
}
