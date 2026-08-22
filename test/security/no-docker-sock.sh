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

if grep -En 'docker\.sock' "$manifest"; then
  echo "SECURITY VIOLATION: mounting or referencing docker.sock is forbidden" >&2
  exit 1
fi
echo "PASS: no docker.sock references"
