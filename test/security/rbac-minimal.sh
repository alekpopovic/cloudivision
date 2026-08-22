#!/usr/bin/env sh
set -eu

manifest=${1:-}
temporary=
if [ -z "$manifest" ]; then
  manifest=$(mktemp)
  temporary=$manifest
  trap 'rm -f "$temporary"' EXIT
  helm template cloudivision charts/cloudivision --include-crds > "$manifest"
fi

if grep -Ein 'cluster-admin' "$manifest"; then
  echo "SECURITY VIOLATION: cluster-admin must not be granted to any cloudivision workload" >&2
  exit 1
fi

if awk '
  function check_rule() {
    if (has_secrets && has_broad_verb) violation = 1
  }
  /^  - apiGroups:/ {
    check_rule()
    in_rule = 1
    has_secrets = 0
    has_broad_verb = 0
    next
  }
  /^---[[:space:]]*$/ {
    check_rule()
    in_rule = 0
    has_secrets = 0
    has_broad_verb = 0
    next
  }
  in_rule && /resources:[[:space:]]*\[[^]]*"secrets"[^]]*\]/ { has_secrets = 1 }
  in_rule && /verbs:[[:space:]]*\[[^]]*("list"|"watch"|"\*")[^]]*\]/ { has_broad_verb = 1 }
  END { check_rule(); exit violation }
' "$manifest"; then
  :
else
  echo "SECURITY VIOLATION: a Role or ClusterRole can enumerate all Secrets" >&2
  exit 1
fi

echo "PASS: no cluster-admin binding or broad Secret enumeration"
