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

# A future exception must name the exact path here and document its owner and risk.
allowlisted_hostpaths_regex='a^'
violations=$(grep -En '^[[:space:]]*(-[[:space:]]*)?hostPath:[[:space:]]*($|#)' "$manifest" || true)
if [ -n "$violations" ] && ! printf '%s\n' "$violations" | grep -Eq "$allowlisted_hostpaths_regex"; then
  printf '%s\n' "$violations" >&2
  echo "SECURITY VIOLATION: hostPath is forbidden unless explicitly allowlisted" >&2
  exit 1
fi
echo "PASS: no non-allowlisted hostPath volumes"
