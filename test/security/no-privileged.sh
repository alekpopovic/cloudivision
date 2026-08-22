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

if grep -En '^[[:space:]]*privileged:[[:space:]]*true([[:space:]]*#.*)?$' "$manifest"; then
  echo "SECURITY VIOLATION: privileged containers are forbidden" >&2
  exit 1
fi
echo "PASS: no privileged containers"
