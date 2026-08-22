#!/usr/bin/env bash
set -euo pipefail

manifest="${1:?usage: no-invalid-pod-security-context.sh RENDERED_MANIFEST}"

if grep -Eq '^[[:space:]]+podSecurityContext:' "${manifest}"; then
  echo "FAIL: rendered workload uses unknown podSecurityContext field; use spec.securityContext" >&2
  exit 1
fi

echo "PASS: rendered workloads use valid Pod securityContext fields"
